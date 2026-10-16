package cli

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixture has the shape of the real cookie — auth_token 40 bytes, ct0 160 — and is not a
// credential. It is the needle the "did anything print the cookie" assertions search for.
var (
	fixtureToken = "0123456789abcdef0123456789abcdef01234567"
	fixtureCTO   = strings.Repeat("b1c2d3e4", 20)
)

// hermeticConfig points this package's verbs at a temp directory: no verb under test reads
// the developer's own configuration, sees the developer's own session file, an ambient proxy
// setting, or the developer's own pins file and first-run marker under the data directory.
func hermeticConfig(t *testing.T, sessionPath string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("X_TERMINAL_SESSION", sessionPath)
	for _, name := range []string{"X_TERMINAL_PROXY", "HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY", "NO_PROXY"} {
		t.Setenv(name, "")
	}
}

// run calls a verb with captured streams, the way Main would.
func run(t *testing.T, handler Handler, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errs bytes.Buffer
	code = handler(&Context{Stdout: &out, Stderr: &errs}, args)
	return code, out.String(), errs.String()
}

// writeSession writes a 0600 session file and returns its path.
func writeSession(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "session")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing the session file: %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod 0600: %v", err)
	}
	return path
}

func TestLogoutDeletesTheSessionFileAndNothingElse(t *testing.T) {
	dir := t.TempDir()
	path := writeSession(t, dir, "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n")
	bystander := filepath.Join(dir, "keep-me")
	if err := os.WriteFile(bystander, []byte("not a credential"), 0o600); err != nil {
		t.Fatalf("writing the bystander file: %v", err)
	}
	hermeticConfig(t, path)

	code, stdout, stderr := run(t, Logout)
	if code != ExitOK {
		t.Fatalf("logout exited %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the session file is still there (or could not be checked): %v", err)
	}
	if _, err := os.Lstat(bystander); err != nil {
		t.Errorf("logout removed a file that is not the session file: %v", err)
	}
	if _, err := os.Lstat(dir); err != nil {
		t.Errorf("logout removed the directory that held the session file: %v", err)
	}

	// The path, and the state of what was deleted — lengths only. Saying what is about to go
	// before it goes is the whole difference between this verb and `rm`.
	for _, want := range []string{"about to delete " + path, "session: present", "auth_token 40 bytes", "deleted " + path} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the output does not say %q:\n%s", want, stdout)
		}
	}
	for _, secret := range []string{fixtureToken, fixtureCTO} {
		if strings.Contains(stdout, secret) || strings.Contains(stderr, secret) {
			t.Error("logout printed the cookie")
		}
	}
}

func TestLogoutRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	target := writeSession(t, dir, "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n")
	link := filepath.Join(dir, "session-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlinking %s to %s: %v", link, target, err)
	}
	hermeticConfig(t, link)

	code, stdout, stderr := run(t, Logout)
	if code == ExitOK {
		t.Fatalf("logout followed a symlink: stdout %q", stdout)
	}
	if !strings.Contains(stderr, "symlink") || !strings.Contains(stderr, link) {
		t.Errorf("the refusal does not name the symlink and the path: %q", stderr)
	}
	if strings.Contains(stdout, "about to delete") {
		t.Errorf("logout announced a deletion it then refused: %q", stdout)
	}
	for _, path := range []string{link, target} {
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("logout removed %s despite refusing to: %v", path, err)
		}
	}
}

func TestLogoutOnAMissingFileIsNotAFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	hermeticConfig(t, path)

	code, stdout, stderr := run(t, Logout)
	if code != ExitOK {
		t.Fatalf("logout exited %d for a file that is already gone, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if !strings.Contains(stdout, "no session file at "+path) {
		t.Errorf("the output does not say there was nothing to delete:\n%s", stdout)
	}
}

func TestLogoutRefusesADirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("making a directory at the session path: %v", err)
	}
	hermeticConfig(t, path)

	code, _, stderr := run(t, Logout)
	if code == ExitOK {
		t.Fatal("logout deleted something that is not a regular file")
	}
	if !strings.Contains(stderr, "not a regular file") {
		t.Errorf("the refusal does not say what it found: %q", stderr)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Errorf("logout removed the directory: %v", err)
	}
}

func TestLogoutTakesNoArguments(t *testing.T) {
	hermeticConfig(t, filepath.Join(t.TempDir(), "session"))

	code, _, stderr := run(t, Logout, "--force")
	if code != ExitUsage {
		t.Fatalf("logout with an argument exited %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "takes no arguments") {
		t.Errorf("the refusal does not say what was wrong: %q", stderr)
	}
}
