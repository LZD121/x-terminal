// Package browserlogin gets a session cookie out of a browser the user has just signed in to.
//
// It is the automated half of `x-terminal login`. The manual half is a prompt: the reader copies two
// values out of a browser that is already logged in. This half opens one instead — a throwaway
// Chromium profile, so the user's own profile is never read, touched or locked — lets them sign in,
// and reads the two cookies from the browser's own DevTools interface.
//
// Three decisions are worth stating, because each closes a way this could go wrong.
//
//   - The profile is temporary and its directory is 0700. It is deleted on every path out, success
//     or failure: the session cookies live in it until then, and a leftover profile would be a
//     second copy of the credential on disk.
//   - The DevTools channel is a pipe, not a port. `--remote-debugging-pipe` gives this process two
//     file descriptors; `--remote-debugging-port` would give every process on the machine a control
//     channel to a browser that is, at that moment, logged in as the user.
//   - The browser is never asked to log in by itself. The user types their password into Chromium,
//     where it belongs; this package only watches for the cookies to appear.
package browserlogin

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Result is the pair the browser ended up holding. It is two strings rather than a domain value
// because this package sits below the CLI and above nothing: the caller turns it into a session.
type Result struct {
	AuthToken string
	CTO       string
}

// Options is one attempt.
type Options struct {
	// Browser is the executable. Empty means Find's answer.
	Browser string
	// URL is the page the window opens on. Empty means the login page.
	URL string
	// Timeout bounds the wait for the user to sign in. Zero means five minutes.
	Timeout time.Duration
	// Progress receives one line per step, for a human watching the terminal. It never receives a
	// cookie value: this package has none to give until the last moment, and it does not print them.
	Progress io.Writer
	// ProfileDir is the throwaway profile. Empty means a temporary directory, which is what the
	// shipped verb uses; a test supplies its own.
	ProfileDir string
	// BrowserCandidates, LookPath and Run are the environment, for tests.
	BrowserCandidates []string
	LookPath          func(string) (string, error)
}

// ErrNoBrowser is a machine without a browser this package knows how to drive. The caller's remedy
// is the manual prompt, not a retry.
var ErrNoBrowser = errors.New("browserlogin: no Chromium-family browser found")

// ErrTimeout is the user not signing in within the budget. Nothing is written on this path.
var ErrTimeout = errors.New("browserlogin: the sign-in did not finish in time")

// DefaultCandidates are the Chromium-family binaries this knows how to drive, in order. The engine
// is the same in all of them and the two flags this uses (`--remote-debugging-pipe`, `--app`) are
// supported by every Chromium build.
var DefaultCandidates = []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome", "microsoft-edge-stable"}

// Find is the browser to launch: X_TERMINAL_BROWSER when it is set and executable, otherwise the
// first candidate on PATH.
func Find(env func(string) string, lookPath func(string) (string, error), candidates []string) (string, error) {
	if env != nil {
		if explicit := strings.TrimSpace(env("X_TERMINAL_BROWSER")); explicit != "" {
			if _, err := os.Stat(explicit); err == nil {
				return explicit, nil
			}
			return "", fmt.Errorf("browserlogin: X_TERMINAL_BROWSER names %s, which is not an executable file", explicit)
		}
	}
	if len(candidates) == 0 {
		candidates = DefaultCandidates
	}
	for _, name := range candidates {
		if path, err := lookPath(name); err == nil {
			return path, nil
		}
	}
	return "", ErrNoBrowser
}

// Login launches the browser, waits until it holds auth_token and ct0, and returns them.
//
// The window is opened with `--app=<url>`: a browser window without tabs or an address bar, which is
// the smallest surface that can host a login page.
func Login(ctx context.Context, opts Options) (Result, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}
	if opts.URL == "" {
		opts.URL = "https://x.com/login"
	}
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
	}

	browser := opts.Browser
	if browser == "" {
		var err error
		browser, err = Find(os.Getenv, opts.LookPath, opts.BrowserCandidates)
		if err != nil {
			return Result{}, err
		}
	}

	dir := opts.ProfileDir
	if dir == "" {
		tmp, err := os.MkdirTemp("", "x-terminal-login-*")
		if err != nil {
			return Result{}, fmt.Errorf("browserlogin: creating a profile directory: %w", err)
		}
		dir = tmp
		// The profile holds the cookies until the window closes; nothing about this package's
		// failures is worth leaving them on disk for.
		defer os.RemoveAll(tmp)
	} else if err := os.MkdirAll(dir, 0o700); err != nil {
		return Result{}, fmt.Errorf("browserlogin: preparing the profile directory: %w", err)
	}

	// Two pipes: fd 3 into the browser, fd 4 out of it. The browser speaks DevTools over them and
	// nothing else on the machine can reach them.
	toBrowserR, toBrowserW, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("browserlogin: pipe: %w", err)
	}
	fromBrowserR, fromBrowserW, err := os.Pipe()
	if err != nil {
		toBrowserR.Close()
		toBrowserW.Close()
		return Result{}, fmt.Errorf("browserlogin: pipe: %w", err)
	}
	defer fromBrowserR.Close()

	cmd := exec.Command(browser,
		"--user-data-dir="+dir,
		"--remote-debugging-pipe",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-features=Translate",
		"--app="+opts.URL,
	)
	// The child's fds 3 and 4; the parent keeps the other ends.
	cmd.ExtraFiles = []*os.File{toBrowserR, fromBrowserW}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		toBrowserR.Close()
		toBrowserW.Close()
		fromBrowserW.Close()
		return Result{}, fmt.Errorf("browserlogin: starting %s: %w", filepath.Base(browser), err)
	}
	// The parent's copies of the child's ends are closed, so the child sees EOF rather than a pipe
	// held open by its own parent.
	toBrowserR.Close()
	fromBrowserW.Close()

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	client := newCDP(fromBrowserR, toBrowserW)
	defer func() {
		_ = client.close()
		// Closing the pipe is what asks a Chromium to exit; a browser that ignores it is killed.
		select {
		case <-exited:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-exited
		}
	}()

	if opts.Progress != nil {
		fmt.Fprintf(opts.Progress, "opened %s; sign in there; waiting for the cookies (up to %s)\n", filepath.Base(browser), opts.Timeout)
	}

	// The polling runs off this goroutine because a DevTools read blocks on a pipe with no deadline:
	// a browser that answers nothing must not hold the verb, and the caller's own timer is what ends
	// the wait. Closing the pipe on the way out is what unblocks the reader.
	results := make(chan pollResult, 1)
	pollCtx, stopPolling := context.WithCancel(ctx)
	defer stopPolling()
	go poll(pollCtx, client, results)

	timer := time.NewTimer(opts.Timeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-timer.C:
			return Result{}, ErrTimeout
		case err := <-exited:
			exited <- err // put it back for the deferred wait
			if err != nil {
				return Result{}, fmt.Errorf("browserlogin: the browser exited before sign-in finished: %w", err)
			}
			return Result{}, errors.New("browserlogin: the browser was closed before the cookies appeared")
		case res := <-results:
			if res.err != nil {
				continue
			}
			if result, ok := pair(res.cookies); ok {
				return result, nil
			}
		}
	}
}

// pollResult is one cookie read: the jar, or why the browser did not answer.
type pollResult struct {
	cookies []cookie
	err     error
}

// poll asks for the jar every so often until the context ends. One buffered slot is enough: the
// caller only ever wants the newest answer.
func poll(ctx context.Context, client *cdp, out chan<- pollResult) {
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		cookies, err := client.cookies(ctx)
		select {
		case out <- pollResult{cookies: cookies, err: err}:
		case <-ctx.Done():
			return
		}
	}
}

// pair finds the two cookies a session needs. Anything else in the jar — the language, the consent,
// the guest token — is not this package's business.
func pair(cookies []cookie) (Result, bool) {
	var r Result
	for _, c := range cookies {
		switch c.Name {
		case "auth_token":
			r.AuthToken = c.Value
		case "ct0":
			r.CTO = c.Value
		}
	}
	return r, r.AuthToken != "" && r.CTO != ""
}

// cookie is one entry of the browser's jar.
type cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// cdp is the DevTools conversation over the two pipe descriptors.
//
// The wire format is JSON in both directions, each message terminated by a NUL byte. It is a small
// enough protocol to speak directly, and doing so is what keeps this package free of a WebSocket
// dependency and of a listening socket.
type cdp struct {
	r    *bufio.Reader
	w    *os.File
	next int
}

func newCDP(r io.Reader, w *os.File) *cdp { return &cdp{r: bufio.NewReader(r), w: w} }

// close closes the write end. Chromium treats that as "the client is gone" and exits.
func (c *cdp) close() error { return c.w.Close() }

// call sends one command and reads until its answer arrives, skipping events (they carry no id).
func (c *cdp) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return c.send(ctx, method, params, "")
}

// callSession is the same conversation on a target's session: the flattened form puts the session id
// on every message instead of opening a second channel.
func (c *cdp) callSession(ctx context.Context, method string, params any, sessionID string) (json.RawMessage, error) {
	return c.send(ctx, method, params, sessionID)
}

func (c *cdp) send(ctx context.Context, method string, params any, sessionID string) (json.RawMessage, error) {
	c.next++
	id := c.next
	msg := map[string]any{"id": id, "method": method}
	if sessionID != "" {
		msg["sessionId"] = sessionID
	}
	if params != nil {
		msg["params"] = params
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if _, err := c.w.Write(append(body, 0)); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		line, err := c.r.ReadBytes(0)
		if err != nil {
			return nil, err
		}
		var answer struct {
			ID    int `json:"id"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(line[:len(line)-1], &answer); err != nil {
			continue // an event this client did not ask for
		}
		if answer.ID != id {
			continue
		}
		if answer.Error != nil {
			return nil, errors.New(answer.Error.Message)
		}
		return answer.Result, nil
	}
}

// cookies asks the browser for its whole jar.
//
// `Storage.getCookies` is the browser-level read and needs no target; an older build answers
// "method not found", and the fallback attaches to the page and asks its network layer instead. Both
// shapes end in the same cookie list.
func (c *cdp) cookies(ctx context.Context) ([]cookie, error) {
	raw, err := c.call(ctx, "Storage.getCookies", map[string]any{})
	if err == nil {
		var answer struct {
			Cookies []cookie `json:"cookies"`
		}
		if err := json.Unmarshal(raw, &answer); err == nil {
			return answer.Cookies, nil
		}
	}
	return c.pageCookies(ctx)
}

// pageCookies is the fallback: attach to the first page target, enable its Network domain, and read.
func (c *cdp) pageCookies(ctx context.Context) ([]cookie, error) {
	raw, err := c.call(ctx, "Target.getTargets", nil)
	if err != nil {
		return nil, err
	}
	var targets struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(raw, &targets); err != nil {
		return nil, err
	}
	for _, t := range targets.TargetInfos {
		if t.Type != "page" {
			continue
		}
		attached, err := c.call(ctx, "Target.attachToTarget", map[string]any{"targetId": t.TargetID, "flatten": true})
		if err != nil {
			continue
		}
		var session struct {
			SessionID string `json:"sessionId"`
		}
		if err := json.Unmarshal(attached, &session); err != nil || session.SessionID == "" {
			continue
		}
		if _, err := c.callSession(ctx, "Network.enable", nil, session.SessionID); err != nil {
			continue
		}
		body, err := c.callSession(ctx, "Network.getAllCookies", nil, session.SessionID)
		if err != nil {
			continue
		}
		var answer struct {
			Cookies []cookie `json:"cookies"`
		}
		if err := json.Unmarshal(body, &answer); err == nil {
			return answer.Cookies, nil
		}
	}
	return nil, errors.New("browserlogin: the browser answered no cookie list")
}
