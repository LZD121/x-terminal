package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSaveWritesAMode0600FileThatLoadsBack: the round trip is the contract — what login writes, the
// loader reads, with nothing else on disk and nothing looser than 0600.
func TestSaveWritesAMode0600FileThatLoadsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x-terminal", "session")
	if err := Save(path, Secret("token-value"), Secret("ct0-value")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("mode is %04o, want 0600", perm)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatal(err)
	} else if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("directory mode is %04o, want 0700", perm)
	}
	sess, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if sess.AuthToken != Secret("token-value") || sess.CTO != Secret("ct0-value") {
		t.Fatal("the values did not round-trip")
	}
	if len(sess.Warnings) != 0 {
		t.Fatalf("the written file warns: %v", sess.Warnings)
	}
}

// TestSaveRefusesHalfAPair: one value is not a session.
func TestSaveRefusesHalfAPair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session")
	if err := Save(path, Secret("only-a-token"), ""); err == nil {
		t.Fatal("half a pair was written")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a refused save left a file behind")
	}
}

// TestSaveRefusesASymlink: the loader refuses one, so the writer must too — otherwise login would
// write the credential wherever the link points.
func TestSaveRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "session")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := Save(link, Secret("a"), Secret("b")); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("err = %v", err)
	}
	body, _ := os.ReadFile(target)
	if string(body) != "x" {
		t.Fatal("the symlink's target was written through")
	}
}

// TestSaveLeavesNoTemporaryFileOnFailure is the one failure a user cannot see: a temp file holding
// the cookie, left behind by a failed rename. The test makes the rename fail by replacing the
// directory with a file after the temporary file exists — the simplest way to reach that arm.
func TestSaveLeavesNoTemporaryFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	// A directory at the target path makes the rename fail on Linux (a directory cannot be
	// replaced by a file), which is the arm this test needs.
	target := filepath.Join(dir, "session")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Save(target, Secret("a"), Secret("b")); err == nil {
		t.Fatal("a directory at the target path was accepted")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".session-") {
			t.Fatalf("a temporary file survived the failure: %s", e.Name())
		}
	}
	// And the directory is still a directory: a refused save did not remove someone's path.
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatal("the target path was replaced")
	}
}

// TestCleanInputTrimsWhatAPasteBrings: whitespace, quotes, and a whole `key=` line.
func TestCleanInputTrimsWhatAPasteBrings(t *testing.T) {
	cases := []struct{ key, in, want string }{
		{"auth_token", "  abc  ", "abc"},
		{"auth_token", `"abc"`, "abc"},
		{"auth_token", "auth_token=abc", "abc"},
		{"auth_token", "auth_token = abc # copied from devtools", "abc"},
		{"ct0", "ct0=deadbeef", "deadbeef"},
	}
	for _, c := range cases {
		if got := string(CleanInput(c.key, c.in)); got != c.want {
			t.Fatalf("CleanInput(%q, %q) = %q, want %q", c.key, c.in, got, c.want)
		}
	}
}

// TestSaveLeavesAPreexistingDirectoryAlone: the path a reader chooses may be a shared directory or
// their own home. Only a directory this call creates is tightened to 0700; an existing one keeps its
// mode, and the file inside it is still 0600.
func TestSaveLeavesAPreexistingDirectoryAlone(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared")
	if err := os.Mkdir(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(shared, "session")
	if err := Save(path, Secret("a"), Secret("b")); err != nil {
		t.Fatalf("Save into a 0755 directory: %v", err)
	}
	info, err := os.Stat(shared)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Fatalf("the existing directory's mode changed to %04o", perm)
	}
	if fileInfo, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Fatalf("the file's mode is %04o", perm)
	}
}
