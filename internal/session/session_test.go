package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures have the shape the real cookies have — auth_token is 40
// characters of base64url, ct0 is 160 — but neither is a credential, because a
// test that used a real one would itself be the leak it is looking for.
var (
	fixtureToken = "0123456789abcdef0123456789abcdef01234567"
	fixtureCTO   = strings.Repeat("b1c2d3e4", 20)
	fixtureFile  = "auth_token=" + fixtureToken + "\nct0=" + fixtureCTO + "\n"
)

// fakeEnv installs vars as this package's whole environment. A key that is
// absent reads as "" — unset — so no test here depends on whether the
// developer's shell happens to export a cookie.
func fakeEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	previous := env
	env = func(key string) string { return vars[key] }
	t.Cleanup(func() { env = previous })
}

// writeSession writes a session file with an exact mode and returns its path.
// The chmod after the write is deliberate: os.WriteFile applies the mode it was
// given masked by the umask, and a permission test must not be decided by the
// umask of whoever runs the suite.
func writeSession(t *testing.T, mode os.FileMode, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session")
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatalf("writing the session file: %v", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %04o: %v", mode, err)
	}
	return path
}

// refuses asserts that Load refused path with a message mentioning every want,
// and that the refusal itself quotes no credential. Call fakeEnv first: nothing
// here may be decided by the ambient environment.
func refuses(t *testing.T, path string, want ...string) {
	t.Helper()
	if _, err := Load(path); err == nil {
		t.Fatalf("Load accepted %s", path)
	} else {
		for _, w := range want {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("the refusal does not mention %q: %v", w, err)
			}
		}
		if strings.Contains(err.Error(), fixtureToken) || strings.Contains(err.Error(), fixtureCTO) {
			t.Errorf("the refusal quotes the credential: %v", err)
		}
	}
}

func TestLoadAcceptsMode0600(t *testing.T) {
	fakeEnv(t, nil)
	path := writeSession(t, 0o600, fixtureFile)

	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	if !sess.Valid() {
		t.Fatalf("Valid() = false for a complete session: %s", sess.Summary())
	}
	// Comparing two Secrets compares the bytes underneath: this asserts exactly
	// what the loader kept and cannot print it if the assertion fails.
	if sess.AuthToken != Secret(fixtureToken) {
		t.Errorf("AuthToken is not the 40-byte value the file holds")
	}
	if sess.CTO != Secret(fixtureCTO) {
		t.Errorf("CTO is not the 160-byte value the file holds")
	}
	if sess.Source != path {
		t.Errorf("Source = %q, want the path it was read from, %q", sess.Source, path)
	}
	if len(sess.Warnings) != 0 {
		t.Errorf("a clean session file warned: %q", sess.Warnings)
	}
}

func TestLoadRefusesLooseMode(t *testing.T) {
	fakeEnv(t, nil)
	path := writeSession(t, 0o644, fixtureFile)

	refuses(t, path, "chmod 600", path, "0644")
}

func TestLoadRefusesSymlink(t *testing.T) {
	fakeEnv(t, nil)
	target := writeSession(t, 0o600, fixtureFile)
	link := filepath.Join(t.TempDir(), "session")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlinking %s to %s: %v", link, target, err)
	}

	refuses(t, link, "symlink", link)
}

func TestEnvironmentOverridesWin(t *testing.T) {
	path := writeSession(t, 0o600, fixtureFile)
	fakeEnv(t, map[string]string{
		"X_TERMINAL_AUTH_TOKEN": "environment-token",
		"X_TERMINAL_CT0":        "environment-ct0",
	})

	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if sess.AuthToken != Secret("environment-token") {
		t.Errorf("AuthToken is not the override's value")
	}
	if sess.CTO != Secret("environment-ct0") {
		t.Errorf("CTO is not the override's value")
	}
	if sess.Source != "environment" {
		t.Errorf("Source = %q, want %q — the values came from the overrides", sess.Source, "environment")
	}
	// The override is a different string from the file's value, so "the override
	// won" is a claim about precedence rather than a coincidence.
	if sess.AuthToken == Secret(fixtureToken) {
		t.Fatal("the override and the file hold the same string — this test cannot tell which one won")
	}
}

func TestBothOverridesWithoutAFileAreASession(t *testing.T) {
	fakeEnv(t, map[string]string{
		"X_TERMINAL_AUTH_TOKEN": "environment-token",
		"X_TERMINAL_CT0":        "environment-ct0",
	})
	path := filepath.Join(t.TempDir(), "session")

	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load with both overrides and no file: %v", err)
	}
	if !sess.Valid() {
		t.Errorf("Valid() = false: %s", sess.Summary())
	}
	if sess.Source != "environment" {
		t.Errorf("Source = %q, want %q", sess.Source, "environment")
	}
}

func TestMissingFileWithoutOverridesIsErrNoSession(t *testing.T) {
	fakeEnv(t, nil)
	path := filepath.Join(t.TempDir(), "session")

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load invented a session out of a missing file and an empty environment")
	}
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("err = %v, want it to be ErrNoSession", err)
	}
	for _, want := range []string{path, "auth_token", "ct0", "session cookies"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message does not mention %q: %v", want, err)
		}
	}
}

func TestAFileThatHoldsNoKeySaysSo(t *testing.T) {
	fakeEnv(t, nil)
	path := writeSession(t, 0o600, "# my session\n\nthis line is neither key nor comment\n")

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load accepted a file with no cookie in it")
	}
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("err = %v, want it to be ErrNoSession", err)
	}
	// "No session" about a file that is right there and wrong would be the kind
	// of half-truth this project exists to avoid.
	if !strings.Contains(err.Error(), "holds no auth_token or ct0 line") {
		t.Errorf("the message does not tell an empty file from a missing one: %v", err)
	}
}

func TestHalfASessionIsIncomplete(t *testing.T) {
	fakeEnv(t, nil)
	path := writeSession(t, 0o600, "auth_token="+fixtureToken+"\n")

	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v (a file with one half is a state, not a refusal)", err)
	}
	if sess.Valid() {
		t.Fatal("Valid() = true for an auth_token with no ct0")
	}
	if !strings.Contains(strings.Join(sess.Warnings, "\n"), "ct0") {
		t.Errorf("no warning names the missing half: %q", sess.Warnings)
	}
	if !strings.Contains(sess.Summary(), "incomplete") {
		t.Errorf("Summary does not say the session is incomplete: %s", sess.Summary())
	}
	if sess.AuthToken != Secret(fixtureToken) {
		t.Errorf("the half that was there did not load")
	}
}

func TestUnknownKeyWarnsAndTheKnownKeysStillLoad(t *testing.T) {
	fakeEnv(t, nil)
	path := writeSession(t, 0o600, fixtureFile+"twid=1234567890\n# a comment\n")

	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(sess.Warnings) != 1 {
		t.Fatalf("Warnings = %q, want one for twid", sess.Warnings)
	}
	if !strings.Contains(sess.Warnings[0], "twid") {
		t.Errorf("the warning does not name the key: %q", sess.Warnings[0])
	}
	if sess.AuthToken != Secret(fixtureToken) || sess.CTO != Secret(fixtureCTO) {
		t.Errorf("the known keys did not load alongside the unknown one")
	}
}

func TestAPastedValueOnTheKeySideIsNotEchoed(t *testing.T) {
	fakeEnv(t, nil)
	path := writeSession(t, 0o600, fixtureToken+"=oops\n"+fixtureFile)

	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(sess.Warnings) == 0 {
		t.Fatal("no warning for a line whose key is a pasted cookie")
	}
	if joined := strings.Join(sess.Warnings, "\n"); strings.Contains(joined, fixtureToken) {
		t.Errorf("a warning echoed a value pasted on the key side: %q", joined)
	}
	if sess.AuthToken != Secret(fixtureToken) {
		t.Errorf("the well-formed lines did not load")
	}
}

func TestQuotedValuesAndCommentsLoad(t *testing.T) {
	fakeEnv(t, nil)
	path := writeSession(t, 0o600,
		"# the cookie, copied out of the browser\n"+
			"AUTH_TOKEN=\""+fixtureToken+"\" # from zen\n"+
			"CT0="+fixtureCTO+"\n")

	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if sess.AuthToken != Secret(fixtureToken) || sess.CTO != Secret(fixtureCTO) {
		t.Errorf("the quoted, commented and upper-case lines did not load")
	}
	if len(sess.Warnings) != 0 {
		t.Errorf("a well-formed file warned: %q", sess.Warnings)
	}
}

func TestSummaryPrintsLengthsAndNeverValues(t *testing.T) {
	fakeEnv(t, nil)
	path := writeSession(t, 0o600, fixtureFile)

	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	summary := sess.Summary()
	want := fmt.Sprintf("session: present (auth_token %d bytes, ct0 %d bytes, source %s)",
		len(fixtureToken), len(fixtureCTO), path)
	if summary != want {
		t.Errorf("Summary() = %q, want %q", summary, want)
	}
	for _, value := range []string{fixtureToken, fixtureCTO} {
		if strings.Contains(summary, value) {
			t.Errorf("Summary printed a credential: %s", summary)
		}
	}
}
