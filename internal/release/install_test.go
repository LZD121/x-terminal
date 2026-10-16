package release

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sha256Sum is the digest a test asks Verify for, so no digest in these arms is
// typed by hand.
func sha256Sum(b []byte) [32]byte { return sha256.Sum256(b) }

// fakePayload is a byte string with only the shape the checks look for: the
// platform's object-file magic, a tag somewhere inside it, and the length the test
// asks for. Nothing here is a real binary, and nothing here lands in the tree.
func fakePayload(tag string, size int) []byte {
	payload := make([]byte, size)
	copy(payload, []byte{0x7f, 'E', 'L', 'F'})
	copy(payload[8:], tag)
	for i := 8 + len(tag); i < size; i++ {
		payload[i] = 'x'
	}
	return payload
}

// entries counts a directory's entries, which is how the arms below say that a
// failed write left no temporary file behind.
func entries(t *testing.T, dir string) int {
	t.Helper()
	list, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	return len(list)
}

// TestResolveExecutableFollowsASymlinkToItsTarget: a `~/.local/bin` entry pointing
// into a versioned directory is an ordinary Go-install shape, so the resolved target
// is what gets replaced and what the verb prints.
func TestResolveExecutableFollowsASymlinkToItsTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "x-terminal")
	if err := os.WriteFile(target, []byte("old bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ResolveExecutable(link)
	if err != nil {
		t.Fatalf("ResolveExecutable(%s): %v", link, err)
	}
	if got != want {
		t.Errorf("ResolveExecutable(%s) = %q, want the resolved target %q", link, got, want)
	}
	// The path of a file that is not a symlink resolves to itself, absolute.
	if got, err := ResolveExecutable(target); err != nil || got != want {
		t.Errorf("ResolveExecutable(%s) = %q, %v", target, got, err)
	}
	if got, err := ResolveExecutable(filepath.Join(dir, ".", "x-terminal")); err != nil || got != want {
		t.Errorf("an unnormalised path was not resolved: %q, %v", got, err)
	}
}

// TestResolveExecutableRefusesADirectoryAndADanglingLink: the target has to be a
// regular file, because that is what a replacement can be published over.
func TestResolveExecutableRefusesADirectoryAndADanglingLink(t *testing.T) {
	dir := t.TempDir()
	if _, err := ResolveExecutable(dir); err == nil {
		t.Error("ResolveExecutable accepted a directory")
	}
	link := filepath.Join(dir, "dangling")
	if err := os.Symlink(filepath.Join(dir, "nowhere"), link); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveExecutable(link); err == nil {
		t.Error("ResolveExecutable accepted a dangling link")
	}
	if _, err := ResolveExecutable(filepath.Join(dir, "missing")); err == nil {
		t.Error("ResolveExecutable accepted a path that does not exist")
	}
}

// TestCheckReplaceableAcceptsAWritableDirectoryAndRefusesAReadOnlyOne: the check is
// about the *directory*, because that is what a same-directory rename needs. The
// read-only arm skips under root rather than lying — root ignores the mode, and a
// passing assertion there would prove nothing.
func TestCheckReplaceableAcceptsAWritableDirectoryAndRefusesAReadOnlyOne(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "x-terminal")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CheckReplaceable(target); err != nil {
		t.Fatalf("a writable directory was refused: %v", err)
	}
	if got := entries(t, dir); got != 1 {
		t.Errorf("the check left %d entries in a directory that held one", got)
	}

	if os.Geteuid() == 0 {
		t.Skip("running as root: mode 0500 does not refuse root, so this arm cannot fail here")
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) }) //nolint:errcheck
	if err := CheckReplaceable(target); err == nil {
		t.Error("a read-only directory was accepted")
	}
}

// TestReplaceWritesThePayloadAndPublishesItByRename is the atomicity arm: the target
// gets the payload, the mode an installed binary needs, and a different inode — which
// is what says the bytes arrived through a rename of a temporary file rather than an
// in-place write that a reader could have caught half-done.
func TestReplaceWritesThePayloadAndPublishesItByRename(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "x-terminal")
	if err := os.WriteFile(target, []byte("old bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("new bytes, in full")

	if err := Replace(target, payload); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	after, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Error("the target is the same file it was: the payload was written in place rather than renamed over it")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Errorf("the target holds %q, want %q", got, payload)
	}
	if perm := after.Mode().Perm(); perm != 0o755 {
		t.Errorf("the installed file is mode %04o, want 0755", perm)
	}
	if n := entries(t, dir); n != 1 {
		t.Errorf("the directory holds %d entries after a successful replace, want 1", n)
	}
}

// TestReplaceLeavesNothingBehindWhenTheRenameFails: every failure path removes its
// temporary file, so a failed update leaves the directory byte-identical to what it
// was. A directory target is how the rename is made to fail on every platform: a
// directory cannot be replaced by a rename of a file.
func TestReplaceLeavesNothingBehindWhenTheRenameFails(t *testing.T) {
	dir := t.TempDir()
	obstacle := filepath.Join(dir, "x-terminal")
	if err := os.Mkdir(obstacle, 0o755); err != nil {
		t.Fatal(err)
	}
	before := entries(t, dir)

	if err := Replace(obstacle, []byte("some payload")); err == nil {
		t.Fatal("Replace renamed a file over a directory")
	}
	if after := entries(t, dir); after != before {
		t.Errorf("a failed replace left %d entries, want %d: the temporary file was not removed", after, before)
	}
	if info, err := os.Stat(obstacle); err != nil || !info.IsDir() {
		t.Errorf("the failed replace changed the target: %v", err)
	}
}

// TestParseDigestReadsTheWorkflowsOneLineForm: 64 lower-case hex digits, two spaces
// and the artifact's own name, which is what `sha256sum "${name}" > "${name}.sha256"`
// writes.
func TestParseDigestReadsTheWorkflowsOneLineForm(t *testing.T) {
	sum := strings.Repeat("ab", 32)
	const name = "x-terminal_v1.1.0_linux_amd64"
	got, err := ParseDigest([]byte(sum+"  "+name+"\n"), name)
	if err != nil {
		t.Fatalf("ParseDigest: %v", err)
	}
	var want [32]byte
	for i := range want {
		want[i] = 0xab
	}
	if got != want {
		t.Errorf("ParseDigest read %x, want %x", got, want)
	}
	// A body with no trailing newline is the same line.
	if _, err := ParseDigest([]byte(sum+"  "+name), name); err != nil {
		t.Errorf("a body without a trailing newline was refused: %v", err)
	}
}

// TestParseDigestRefusesWhatItCannotVerify: a multi-line body is refused rather than
// searched (that is the whole reason this reads the per-asset file and not
// SHA256SUMS), a digest for another artifact is refused by name, and anything that is
// not the sha256sum shape is refused with what it found.
func TestParseDigestRefusesWhatItCannotVerify(t *testing.T) {
	const name = "x-terminal_v1.1.0_linux_amd64"
	sum := strings.Repeat("ab", 32)
	other := strings.Repeat("cd", 32)

	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "two lines",
			body: sum + "  " + name + "\n" + other + "  x-terminal_v1.1.0_darwin_arm64\n",
			want: "2 lines",
		},
		{
			name: "another artifact's digest",
			body: other + "  x-terminal_v1.1.0_darwin_arm64\n",
			want: "x-terminal_v1.1.0_darwin_arm64",
		},
		{
			name: "not a sha256sum line",
			body: "not a digest at all\n",
			want: "not a sha256sum line",
		},
		{
			name: "upper-case hex",
			body: strings.Repeat("AB", 32) + "  " + name + "\n",
			want: "not a sha256sum line",
		},
		{
			name: "too few digits",
			body: strings.Repeat("ab", 31) + "  " + name + "\n",
			want: "not a sha256sum line",
		},
		{
			name: "empty body",
			body: "",
			want: "not a sha256sum line",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseDigest([]byte(tc.body), name); err == nil {
				t.Fatal("ParseDigest accepted a body it cannot verify")
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal does not name %q: %v", tc.want, err)
			}
		})
	}
}

// TestVerifyAcceptsAMatchAndRefusesEverythingElse: what it catches is corruption,
// truncation and a mismatched asset, and the error says both digests and what the
// check is worth.
func TestVerifyAcceptsAMatchAndRefusesEverythingElse(t *testing.T) {
	payload := []byte("the bytes that arrived")
	var want [32]byte
	if err := Verify(payload, sha256Sum(payload)); err != nil {
		t.Fatalf("Verify refused a matching digest: %v", err)
	}
	err := Verify(payload, want)
	if err == nil {
		t.Fatal("Verify accepted a digest that is not the payload's")
	}
	if !strings.Contains(err.Error(), "integrity") || !strings.Contains(err.Error(), "provenance") {
		t.Errorf("the refusal does not say what the check is worth: %v", err)
	}
	// A truncated payload against the full payload's digest is the ordinary failure.
	full := []byte("the bytes that arrived in full")
	if err := Verify(full[:10], sha256Sum(full)); err == nil {
		t.Error("Verify accepted a truncated payload")
	}
}

// TestLooksLikeThisBuildRefusesEachShapeItCanSee: the published size, the platform's
// magic and the tag the payload carries, each with its own refusal.
func TestLooksLikeThisBuildRefusesEachShapeItCanSee(t *testing.T) {
	const tag = "v1.1.0"
	payload := fakePayload(tag, 64)

	if err := LooksLikeThisBuild(payload, tag, "linux", int64(len(payload))); err != nil {
		t.Fatalf("a payload with the magic, the tag and the published size was refused: %v", err)
	}

	for _, tc := range []struct {
		name    string
		payload []byte
		tag     string
		goos    string
		size    int64
		want    string
	}{
		{name: "short payload", payload: payload[:32], tag: tag, goos: "linux", size: 64, want: "truncated"},
		{name: "a size the release did not publish", payload: payload, tag: tag, goos: "linux", size: 65, want: "the Release published 65"},
		{name: "no object-file magic", payload: []byte("#!/bin/sh\necho " + tag + "\n"), tag: tag, goos: "linux", size: int64(len("#!/bin/sh\necho " + tag + "\n")), want: "object-file header"},
		{name: "carries no tag", payload: fakePayload("v9.9.9", 64), tag: tag, goos: "linux", size: 64, want: "does not carry v1.1.0"},
		{name: "a platform with no magic here", payload: payload, tag: tag, goos: "windows", size: 64, want: "object-file header"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := LooksLikeThisBuild(tc.payload, tc.tag, tc.goos, tc.size); err == nil {
				t.Fatal("LooksLikeThisBuild accepted a payload that is not this build")
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal does not say %q: %v", tc.want, err)
			}
		})
	}

	// A Mach-O payload is accepted for darwin, so the magic arm is not a linux-only rule.
	macho := append([]byte{0xcf, 0xfa, 0xed, 0xfe}, []byte(tag)...)
	if err := LooksLikeThisBuild(macho, tag, "darwin", int64(len(macho))); err != nil {
		t.Errorf("a Mach-O payload was refused for darwin: %v", err)
	}
}
