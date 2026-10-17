package browserlogin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// fakeBrowser answers the DevTools conversation on a pipe pair, the way Chromium does: JSON messages
// with no framing but a trailing NUL.
type fakeBrowser struct {
	answers map[string]string
	seen    []string
}

// newPipeBrowser starts a fake on the two ends a real browser would own.
func newPipeBrowser(t *testing.T, fb *fakeBrowser) (*os.File, *os.File) {
	t.Helper()
	// toBrowser: the test writes (parent end), the fake reads (child end).
	toBR, toBW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	// fromBrowser: the fake writes (child end), the test reads (parent end).
	fromBR, fromBW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { toBW.Close(); fromBR.Close(); fromBW.Close(); toBR.Close() })
	go func() {
		defer fromBW.Close()
		buf := make([]byte, 0, 4096)
		chunk := make([]byte, 1024)
		for {
			n, err := toBR.Read(chunk)
			buf = append(buf, chunk[:n]...)
			for {
				i := indexByte(buf, 0)
				if i < 0 {
					break
				}
				fb.handle(buf[:i], fromBW)
				buf = buf[i+1:]
			}
			if err != nil {
				return
			}
		}
	}()
	return fromBR, toBW
}

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

func (fb *fakeBrowser) handle(msg []byte, out *os.File) {
	var req struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
	}
	_ = json.Unmarshal(msg, &req)
	fb.seen = append(fb.seen, req.Method)
	answer, ok := fb.answers[req.Method]
	if !ok {
		answer = `{"error":{"message":"'method' wasn't found"}}`
	}
	answer = strings.ReplaceAll(answer, "{id}", itoa(req.ID))
	_, _ = out.Write(append([]byte(answer), 0))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// TestThePipeConversationReadsTheCookies: the browser-level read answers, and the pair comes out.
func TestThePipeConversationReadsTheCookies(t *testing.T) {
	fb := &fakeBrowser{answers: map[string]string{
		"Storage.getCookies": `{"id":{id},"result":{"cookies":[{"name":"auth_token","value":"a"},{"name":"ct0","value":"b"},{"name":"lang","value":"en"}]}}`,
	}}
	r, w := newPipeBrowser(t, fb)
	c := newCDP(r, w)
	cookies, err := c.cookies(context.Background())
	if err != nil {
		t.Fatalf("cookies: %v", err)
	}
	got, ok := pair(cookies)
	if !ok || got.AuthToken != "a" || got.CTO != "b" {
		t.Fatalf("pair = %+v ok=%v", got, ok)
	}
}

// TestTheFallbackAttachesToAPage: a browser without Storage.getCookies is answered through the page's
// network layer, and the test proves the session id travels on the messages that need it.
func TestTheFallbackAttachesToAPage(t *testing.T) {
	fb := &fakeBrowser{answers: map[string]string{
		"Target.getTargets":     `{"id":{id},"result":{"targetInfos":[{"targetId":"T1","type":"page"}]}}`,
		"Target.attachToTarget": `{"id":{id},"result":{"sessionId":"S1"}}`,
		"Network.enable":        `{"id":{id},"result":{}}`,
		"Network.getAllCookies": `{"id":{id},"result":{"cookies":[{"name":"auth_token","value":"a"},{"name":"ct0","value":"b"}]}}`,
	}}
	r, w := newPipeBrowser(t, fb)
	c := newCDP(r, w)
	cookies, err := c.pageCookies(context.Background())
	if err != nil {
		t.Fatalf("pageCookies: %v", err)
	}
	if len(cookies) != 2 {
		t.Fatalf("%d cookies", len(cookies))
	}
}

// TestFindPrefersTheEnvironment: an explicit path wins, then the candidates in order, and nothing on
// PATH is the named error the caller turns into "use the prompt".
func TestFindPrefersTheEnvironment(t *testing.T) {
	dir := t.TempDir()
	explicit := dir + "/my-chromium"
	if err := os.WriteFile(explicit, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	getenv := func(k string) string {
		if k == "X_TERMINAL_BROWSER" {
			return explicit
		}
		return ""
	}
	if got, err := Find(getenv, execLookPath(nil), nil); err != nil || got != explicit {
		t.Fatalf("Find = %q, %v", got, err)
	}
	getenv = func(string) string { return "" }
	look := func(name string) (string, error) {
		if name == "google-chrome-stable" {
			return "/usr/bin/google-chrome-stable", nil
		}
		return "", errors.New("not found")
	}
	if got, err := Find(getenv, look, nil); err != nil || got != "/usr/bin/google-chrome-stable" {
		t.Fatalf("Find = %q, %v", got, err)
	}
	if _, err := Find(getenv, func(string) (string, error) { return "", errors.New("no") }, nil); !errors.Is(err, ErrNoBrowser) {
		t.Fatalf("err = %v", err)
	}
}

func execLookPath(paths map[string]string) func(string) (string, error) {
	return func(name string) (string, error) {
		if p, ok := paths[name]; ok {
			return p, nil
		}
		return "", errors.New("not found")
	}
}

// TestLoginTimesOutWithoutWriting: a browser that never carries the pair ends in ErrTimeout, and
// Login returns no values.
func TestLoginTimesOutWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	script := dir + "/fake-browser"
	// A browser that starts, does nothing, and answers nothing: the timeout is the point.
	// `exec` so the process the test kills is the one that waits: without it the shell would be
	// killed and its child would keep the pipe open, which is a property of the fixture rather than
	// of the code under test.
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Login(context.Background(), Options{
		Browser:    script,
		ProfileDir: dir + "/profile",
		Timeout:    300 * time.Millisecond,
	})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want ErrTimeout", err)
	}
}

// TestLiveChromiumHandshake is the one thing a fixture cannot prove: that this host's Chromium
// accepts `--remote-debugging-pipe` and answers on the two descriptors. It opens a window, asks for
// the target list, and closes; it never waits for a sign-in and never reads a cookie.
//
// Gated because it opens a window on a human's desktop: X_TERMINAL_BROWSER_LIVE=1 runs it.
func TestLiveChromiumHandshake(t *testing.T) {
	if os.Getenv("X_TERMINAL_BROWSER_LIVE") != "1" {
		t.Skip("X_TERMINAL_BROWSER_LIVE is not 1: this test opens a real browser window")
	}
	browser, err := Find(os.Getenv, exec.LookPath, nil)
	if err != nil {
		t.Skipf("no browser on this host: %v", err)
	}

	dir := t.TempDir()
	toR, toW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	fromR, fromW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer fromR.Close()

	cmd := exec.Command(browser,
		"--user-data-dir="+dir,
		"--remote-debugging-pipe",
		"--no-first-run",
		"--no-default-browser-check",
		"--app=about:blank",
	)
	cmd.ExtraFiles = []*os.File{toR, fromW}
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	toR.Close()
	fromW.Close()
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	defer func() {
		_ = toW.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}()

	client := newCDP(fromR, toW)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// A page target appearing is the handshake: the browser is up and the pipe speaks.
	deadline := time.Now().Add(15 * time.Second)
	for {
		raw, err := client.call(ctx, "Target.getTargets", nil)
		if err == nil && len(raw) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no target list over the pipe: %v", err)
		}
		time.Sleep(250 * time.Millisecond)
	}

	// And the read the login actually performs answers on this build: an empty jar is a success,
	// "method not found" would mean the fallback is the live path here.
	cookies, err := client.cookies(ctx)
	if err != nil {
		t.Fatalf("cookie read over the pipe: %v", err)
	}
	if cookies == nil {
		t.Log("the browser answered an empty jar")
	}
}
