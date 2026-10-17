package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"

	"github.com/LZD121/x-terminal/internal/browserlogin"
	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xsource"
)

// Login is the `login` verb: it reads the two session cookies, checks them against x.com with one
// request, and writes them to the session file.
//
// It exists because the file is a credential and the manual path was error-prone in exactly the ways
// that look like bugs later: a value copied with its key, a pair pasted with one half missing, a file
// created world-readable, a cookie that was already dead when it was written. Each of those is a
// check here, and each failure names itself without ever printing a value.
//
// It never opens a browser and never imports one's cookie store. The two values come from the reader,
// typed or pasted at a prompt with no echo, which is the one path this project will put a credential
// through.
func Login(ctx *Context, args []string) int {
	replace, verify, browser := false, true, false
	for _, arg := range args {
		switch arg {
		case "--replace":
			replace = true
		case "--no-verify":
			verify = false
		case "--browser":
			browser = true
		case "-h", "--help":
			fmt.Fprintln(ctx.Stdout, "login reads auth_token and ct0, checks them with one request, and writes the session file at 0600.\n\n  --browser     open a throwaway Chromium window on the login page and read the cookies from it\n  --replace     overwrite a session that is already there\n  --no-verify   write without the check request (the cookie is unproven; doctor will tell you later)")
			return ExitOK
		default:
			fmt.Fprintf(ctx.Stderr, "login: unknown argument %q\n", arg)
			return ExitUsage
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "login: %v\n", err)
		return ExitError
	}
	for _, warning := range cfg.Warnings {
		fmt.Fprintf(ctx.Stderr, "config warning: %s\n", warning)
	}

	// An existing session is not replaced by accident: a login that overwrote a working pair with a
	// typo would lock the reader out of their own feed until they found the browser again.
	// The check is on the file, not on the merged session: with X_TERMINAL_* set, those values are
	// this verb's input and would otherwise look like a session already on disk.
	if current, err := session.FileSession(cfg.SessionPath); err == nil && current.Valid() && !replace {
		fmt.Fprintf(ctx.Stdout, "a session is already here: %s\n", current.Summary())
		fmt.Fprintf(ctx.Stdout, "pass --replace to write a new pair over it\n")
		return ExitOK
	}

	var authToken, cto session.Secret
	if browser {
		authToken, cto, err = readFromBrowser(ctx)
		if errors.Is(err, browserlogin.ErrNoBrowser) {
			// The machine has no browser this knows how to drive: the manual path is right there,
			// and saying so is the whole remedy.
			fmt.Fprintf(ctx.Stderr, "login: %v; run the same command without --browser and paste the two values instead\n", err)
			return ExitError
		}
	} else {
		guide(ctx.Stdout)
		authToken, cto, err = readFromTerminal(ctx)
	}
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "login: %v\n", err)
		return ExitError
	}
	if authToken == "" || cto == "" {
		fmt.Fprintln(ctx.Stderr, "login: both values are required: an auth_token without a ct0 is answered 401, which reads exactly like an expired cookie")
		return ExitError
	}

	candidate := session.Session{AuthToken: authToken, CTO: cto, Source: "login"}

	if verify {
		account, err := loginVerify(cfg, candidate)
		if err != nil {
			// The classifier's own sentence: refused credentials, a missing proxy and a rate limit
			// are three different next moves. Nothing is written on this path.
			fmt.Fprintf(ctx.Stderr, "login: the cookies were not accepted, nothing was written: %v\n", err)
			return ExitError
		}
		fmt.Fprintf(ctx.Stdout, "signed in as @%s — %s\n", account.ScreenName, account.Name)
	}

	if err := session.Save(cfg.SessionPath, authToken, cto); err != nil {
		fmt.Fprintf(ctx.Stderr, "login: %v\n", err)
		return ExitError
	}
	fmt.Fprintf(ctx.Stdout, "wrote %s (mode 0600, owner only)\n", cfg.SessionPath)
	if !verify {
		fmt.Fprintln(ctx.Stdout, "the pair was not checked: run `x-terminal doctor` before relying on it")
	}
	fmt.Fprintln(ctx.Stdout, "next: `x-terminal doctor` checks the session against x.com; `x-terminal logout` removes this copy")
	return ExitOK
}

// readFromTerminal is the manual half: the guide, then two lines with no echo.
//
// One reader for both lines: a bufio.Reader buffers past the first newline, and a second one built
// over the same input would lose the bytes the first had already read. (That is not a hypothetical:
// the piped form of this verb failed exactly that way.)
func readFromTerminal(ctx *Context) (session.Secret, session.Secret, error) {
	input := bufio.NewReader(ctx.Stdin)
	authToken, err := readSecret(ctx, input, "auth_token", "X_TERMINAL_AUTH_TOKEN")
	if err != nil {
		return "", "", err
	}
	cto, err := readSecret(ctx, input, "ct0", "X_TERMINAL_CT0")
	if err != nil {
		return "", "", err
	}
	return authToken, cto, nil
}

// readFromBrowser is the automated half: a throwaway Chromium window on the login page, and the two
// cookies read out of that window's own jar once they appear.
func readFromBrowser(ctx *Context) (session.Secret, session.Secret, error) {
	result, err := browserlogin.Login(context.Background(), browserlogin.Options{Progress: ctx.Stdout})
	if err != nil {
		return "", "", err
	}
	fmt.Fprintln(ctx.Stdout, "the browser handed over both cookies; it is closing")
	return session.CleanInput("auth_token", result.AuthToken), session.CleanInput("ct0", result.CTO), nil
}

// loginVerify is the one check request: the Viewer read, through the same client and classifier
// every other verb uses. It is a variable so a test can answer without a socket.
var loginVerify = func(cfg config.Config, sess session.Session) (xsource.ViewerAccount, error) {
	client, err := newAPIClient(cfg, sess)
	if err != nil {
		return xsource.ViewerAccount{}, err
	}
	runCtx, cancel := context.WithTimeout(context.Background(), whoamiTimeout)
	defer cancel()
	raw, err := client.GraphQL(runCtx, "Viewer", nil)
	if err != nil {
		return xsource.ViewerAccount{}, err
	}
	user, err := xsource.ParseUser(raw)
	if err != nil {
		return xsource.ViewerAccount{}, err
	}
	return xsource.ViewerAccount{ScreenName: user.ScreenName, Name: user.Name}, nil
}

// guide is the three sentences a reader needs. It names the page and the two cookie names because
// those are what the browser shows; it never suggests a shell command that would put a value in a
// history file.
func guide(w io.Writer) {
	fmt.Fprintln(w, "Sign in with the two session cookies of a browser that is already logged in to x.com:")
	fmt.Fprintln(w, "  1. open https://x.com/home in that browser")
	fmt.Fprintln(w, "  2. open its developer tools, then Application -> Storage -> Cookies -> https://x.com")
	fmt.Fprintln(w, "  3. copy the value of auth_token, then the value of ct0")
	fmt.Fprintln(w, "They are session cookies: they stop working when that browser session ends.")
	fmt.Fprintln(w, "Type or paste each one below; nothing is echoed, and neither value is ever printed again.")
}

// readSecret reads one value: the environment variable when it is set (the scriptable path, which
// keeps the value out of the terminal), and otherwise a line from stdin — with echo off when stdin is
// the terminal itself.
func readSecret(ctx *Context, input *bufio.Reader, key, envVar string) (session.Secret, error) {
	if value := os.Getenv(envVar); value != "" {
		fmt.Fprintf(ctx.Stdout, "%s: from %s\n", key, envVar)
		return session.CleanInput(key, value), nil
	}

	fmt.Fprintf(ctx.Stdout, "%s: ", key)
	value, err := readLine(ctx.Stdin, input)
	fmt.Fprintln(ctx.Stdout)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", key, err)
	}
	return session.CleanInput(key, value), nil
}

// readLine reads one line, with echo off when the input is the process's own terminal. The check is
// on identity rather than on IsTerminal alone: a test hands a reader, and a reader has no fd to put
// in raw mode.
func readLine(orig io.Reader, input *bufio.Reader) (string, error) {
	// The terminal itself is the one input that must not echo; a pipe or a test's reader has no
	// echo to switch off and no fd to put in raw mode.
	if f, ok := orig.(*os.File); ok && term.IsTerminal(f.Fd()) {
		raw, err := term.ReadPassword(f.Fd())
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
	line, err := input.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	// EOF with nothing left is an empty answer, which the verb reports as a missing value; it is
	// not an I/O failure, and a user who pressed Ctrl-D deserves the sentence about the pair.
	return strings.TrimRight(line, "\r\n"), nil
}
