package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xsource"
)

// loginEnv points config.Load at a temporary config home and returns the session path it resolves.
func loginEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("X_TERMINAL_AUTH_TOKEN", "")
	t.Setenv("X_TERMINAL_CT0", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return cfg.SessionPath
}

// loginRun runs the verb with the given stdin and a verification answer.
func loginRun(t *testing.T, stdin string, verify func(config.Config, session.Session) (xsource.ViewerAccount, error), args ...string) (int, string) {
	t.Helper()
	previous := loginVerify
	if verify != nil {
		loginVerify = verify
	}
	t.Cleanup(func() { loginVerify = previous })
	var out, errs bytes.Buffer
	code := Login(&Context{Stdout: &out, Stderr: &errs, Stdin: strings.NewReader(stdin)}, args)
	return code, out.String() + errs.String()
}

// TestLoginWritesThePairAt0600 is the whole point: the two values land in the session file, at 0600,
// and the output never carries either one.
func TestLoginWritesThePairAt0600(t *testing.T) {
	path := loginEnv(t)
	token := "0123456789abcdef0123456789abcdef01234567"
	cto := strings.Repeat("b1c2d3e4", 20)
	code, out := loginRun(t, token+"\n"+cto+"\n", func(config.Config, session.Session) (xsource.ViewerAccount, error) {
		return xsource.ViewerAccount{ScreenName: "someone", Name: "Some One"}, nil
	})
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "@someone") {
		t.Fatalf("the account was not named: %s", out)
	}
	if strings.Contains(out, token) || strings.Contains(out, cto) {
		t.Fatal("the output carries a cookie value")
	}
	sess, err := session.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if sess.AuthToken != session.Secret(token) || sess.CTO != session.Secret(cto) {
		t.Fatal("the values did not round-trip through the file")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("mode is %04o", perm)
	}
}

// TestLoginWritesNothingWhenTheCheckFails: a rejected pair must not reach the file, or the reader
// would run doctor against a cookie the verb already knew was dead.
func TestLoginWritesNothingWhenTheCheckFails(t *testing.T) {
	path := loginEnv(t)
	code, out := loginRun(t, "a\nb\n", func(config.Config, session.Session) (xsource.ViewerAccount, error) {
		return xsource.ViewerAccount{}, os.ErrPermission
	})
	if code != ExitError {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "nothing was written") {
		t.Fatalf("the failure does not say what it did: %s", out)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a rejected login wrote a session file")
	}
}

// TestLoginLeavesAnExistingSessionAlone: replacing a working pair needs the flag.
func TestLoginLeavesAnExistingSessionAlone(t *testing.T) {
	path := loginEnv(t)
	if err := session.Save(path, "old-token", "old-cto"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	code, out := loginRun(t, "new-token\nnew-cto\n", nil)
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "--replace") {
		t.Fatalf("the refusal does not name the flag: %s", out)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("the existing session was overwritten without --replace")
	}
}

// TestLoginReplaceOverwrites: with the flag it does, and the check still runs.
func TestLoginReplaceOverwrites(t *testing.T) {
	path := loginEnv(t)
	if err := session.Save(path, "old-token", "old-cto"); err != nil {
		t.Fatal(err)
	}
	checked := false
	code, out := loginRun(t, "new-token\nnew-cto\n", func(_ config.Config, s session.Session) (xsource.ViewerAccount, error) {
		checked = true
		if s.AuthToken != "new-token" {
			t.Fatalf("the check saw %v", s.AuthToken)
		}
		return xsource.ViewerAccount{ScreenName: "someone"}, nil
	}, "--replace")
	if code != ExitOK || !checked {
		t.Fatalf("exit %d checked=%v: %s", code, checked, out)
	}
	sess, err := session.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if sess.AuthToken != "new-token" {
		t.Fatal("the file kept the old pair")
	}
}

// TestLoginNoVerifySkipsTheCheck: the escape hatch writes and says the pair is unproven.
func TestLoginNoVerifySkipsTheCheck(t *testing.T) {
	path := loginEnv(t)
	called := false
	code, out := loginRun(t, "a\nb\n", func(config.Config, session.Session) (xsource.ViewerAccount, error) {
		called = true
		return xsource.ViewerAccount{}, nil
	}, "--no-verify")
	if code != ExitOK || called {
		t.Fatalf("exit %d called=%v: %s", code, called, out)
	}
	if !strings.Contains(out, "not checked") {
		t.Fatalf("the output does not warn: %s", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the file was not written: %v", err)
	}
}

// TestLoginReadsTheEnvironmentWhenSet: the scriptable path, which keeps the value out of the
// terminal's scrollback.
func TestLoginReadsTheEnvironmentWhenSet(t *testing.T) {
	path := loginEnv(t)
	t.Setenv("X_TERMINAL_AUTH_TOKEN", "env-token")
	t.Setenv("X_TERMINAL_CT0", "env-cto")
	code, out := loginRun(t, "", func(_ config.Config, s session.Session) (xsource.ViewerAccount, error) {
		if s.AuthToken != "env-token" || s.CTO != "env-cto" {
			t.Fatalf("the check saw %v / %v", s.AuthToken, s.CTO)
		}
		return xsource.ViewerAccount{ScreenName: "someone"}, nil
	})
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, out)
	}
	if strings.Contains(out, "env-token") || strings.Contains(out, "env-cto") {
		t.Fatal("the output carries a value")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the file was not written: %v\noutput:\n%s", err, out)
	}
}

// TestLoginRefusesHalfAPair: two lines are required, and the second empty one is not a ct0.
func TestLoginRefusesHalfAPair(t *testing.T) {
	path := loginEnv(t)
	code, out := loginRun(t, "only-a-token\n\n", nil, "--no-verify")
	if code != ExitError {
		t.Fatalf("exit %d: %s", code, out)
	}
	if _, err := os.Stat(filepath.Clean(path)); !os.IsNotExist(err) {
		t.Fatal("half a pair was written")
	}
}

// TestLoginAsWritesANamedAccount: `--as` puts the pair in the sessions directory, and the legacy
// file is not touched.
func TestLoginAsWritesANamedAccount(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("X_TERMINAL_AUTH_TOKEN", "")
	t.Setenv("X_TERMINAL_CT0", "")
	code, out := loginRun(t, "a\nb\n", nil, "--no-verify", "--as", "work")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, out)
	}
	store := session.Store{Dir: filepath.Join(home, "x-terminal", "sessions"), Legacy: filepath.Join(home, "x-terminal", "session")}
	sess, err := store.Load("work")
	if err != nil {
		t.Fatalf("named account: %v", err)
	}
	if sess.AuthToken != "a" {
		t.Fatal("the named account did not hold the pair")
	}
	if _, err := os.Stat(store.Legacy); !os.IsNotExist(err) {
		t.Fatal("the legacy file was written by --as")
	}
}

// TestLoginRefusesAnAccountNameThatIsAPath: the name is a file name, not a path.
func TestLoginRefusesAnAccountNameThatIsAPath(t *testing.T) {
	loginEnv(t)
	for _, bad := range []string{"../escape", "a/b", ".hidden"} {
		code, out := loginRun(t, "a\nb\n", nil, "--no-verify", "--as", bad)
		if code != ExitUsage {
			t.Fatalf("%q: exit %d (%s)", bad, code, out)
		}
	}
}
