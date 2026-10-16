// The filesystem half of internal/release: everything that touches the disk, in
// one file on purpose. internal/audit's persistence check anchors this build's one
// new write destination to this file (`updaterWriterFile`), so a write in another
// file of the same package is a new destination rather than an inheritance.
//
// Two facts it is built around:
//
//   - The write happens in the target's own directory, so the rename that
//     publishes it is same-filesystem and therefore atomic. The target path is
//     never a partial binary.
//   - The temporary file is created before anything is installed, so a directory
//     that refuses one is answered before a byte is downloaded.
//
// What it does not do, so that the promise is not wider than the code: no
// directory fsync, and no chmod of anything but its own temporary file. The
// promise is that the target path is never a partial binary, not durability
// across a power cut.
package release

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// updateTempPattern is the temporary file every write here goes through, in the
// target's own directory. It is a named constant because two places create one
// (the writability check and the replacement) and because the audit's persistence
// plant test reaches the same call site with the same pattern.
const updateTempPattern = ".x-terminal-update-*"

// ResolveExecutable resolves the path this process was started from to the file a
// replacement has to touch: absolute, symlinks followed, and required to be a
// regular file.
//
// The stance on symlinks is deliberate and it is not internal/session's.
// session.Load refuses a symlink because the file that was mode-checked is then
// not the file that is read — a security property of a credential. A symlinked
// binary has no such property: a `~/.local/bin` entry pointing into a versioned
// directory is an ordinary Go-install shape, the file being replaced is identified
// by its resolved path, and the payload is identified by its digest. So the
// resolved target is what gets written, and it is what the verb prints.
func ResolveExecutable(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("release: resolving %s: %w", path, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// A dangling link fails here, which is the right place: its target is not
		// a file anything can be installed over.
		return "", fmt.Errorf("release: resolving %s: %w", abs, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("release: resolving %s: %w", resolved, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("release: %s is not a regular file: the update verb replaces the file this process was started from", resolved)
	}
	return resolved, nil
}

// CheckReplaceable is the free, local question the verb asks before it downloads
// anything: can this machine take the update at all?
//
// It creates and removes a temporary file in the target's own directory, which is
// the honest test of writability: a same-directory rename needs write permission
// on the *directory*, not on the file being replaced, so inspecting the file's own
// mode would answer a different question and get it wrong in both directions.
func CheckReplaceable(target string) error {
	dir := filepath.Dir(target)
	f, err := os.CreateTemp(dir, updateTempPattern)
	if err != nil {
		return fmt.Errorf("release: %s does not accept a temporary file, so the rename that publishes the new binary cannot happen there: %w", dir, err)
	}
	tmp := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return fmt.Errorf("release: closing %s: %w", tmp, err)
	}
	if err := os.Remove(tmp); err != nil {
		return fmt.Errorf("release: removing %s: %w", tmp, err)
	}
	return nil
}

// digestPattern is the sha256sum form the release workflow writes:
// `sha256sum "${name}" > "${name}.sha256"`, which is 64 lower-case hex digits, two
// spaces and the artifact's own name.
var digestPattern = regexp.MustCompile(`^([0-9a-f]{64})  (.+)$`)

// ParseDigest reads one per-asset digest file and asserts the name it read against
// the name it asked for.
//
// It is the per-asset file and not SHA256SUMS, and this parser is why: the release
// workflow builds SHA256SUMS as `cat ./*.sha256 | sort -k2`, so the two files agree
// line for line — but the per-asset file names the artifact in its own line, which
// means verification has no row-selection step that could pick a different
// platform's digest, and the name it read can be asserted rather than trusted. A
// body holding more than one line is refused rather than searched: choosing a line
// would be the guess this file exists to remove.
func ParseDigest(body []byte, wantName string) ([32]byte, error) {
	var sum [32]byte
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if len(lines) != 1 {
		return sum, fmt.Errorf("release: the checksum file holds %d lines: a per-asset digest file names exactly one artifact, and a body with more than one cannot be read without choosing a line", len(lines))
	}
	fields := digestPattern.FindStringSubmatch(lines[0])
	if fields == nil {
		return sum, fmt.Errorf("release: %q is not a sha256sum line: expected 64 lower-case hex digits, two spaces and the artifact's name", lines[0])
	}
	if fields[2] != wantName {
		return sum, fmt.Errorf("release: the checksum file names %s and this asset is %s: a digest for another artifact cannot verify this one", fields[2], wantName)
	}
	decoded, err := hex.DecodeString(fields[1])
	if err != nil {
		return sum, fmt.Errorf("release: %q is not a sha256 digest: %w", fields[1], err)
	}
	copy(sum[:], decoded)
	return sum, nil
}

// Verify compares the payload's digest with the one the Release published.
//
// What this proves, stated where it is used: the digest and the payload come from
// the same origin, so this defeats corruption, truncation and a mismatched asset —
// and it is not a provenance or authenticity proof. A publisher who could replace
// the asset could replace the digest beside it, which is what SHA256SUMS would have
// proved too.
func Verify(payload []byte, want [32]byte) error {
	got := sha256.Sum256(payload)
	if !bytes.Equal(got[:], want[:]) {
		return fmt.Errorf("release: the payload's sha256 is %x and the Release published %x: the download was truncated, corrupted, or is not the asset this Release names (this is an integrity check and not a provenance proof — both values come from the same Release)", got, want)
	}
	return nil
}

// LooksLikeThisBuild is the three shape checks that run after the digest: the
// payload's length must equal the size the API published for that asset, its first
// bytes must be this platform's object-file magic, and it must contain the tag it
// was published under.
//
// The tag arm rests on a measured fact: the release workflow stamps every asset
// with `-X github.com/LZD121/x-terminal/internal/cli.version=<tag>`, so the tag is a
// string in the image of every asset this repository publishes.
//
// What these do not prove, so that the check is not oversold: they are shape checks
// over bytes that arrived from the same origin as the digest, so a payload crafted
// for this release would pass all three. What they catch is the ordinary failure —
// a truncated or corrupted download, an asset from another platform, and a binary
// that cannot be the one this tag was built for.
func LooksLikeThisBuild(payload []byte, tag, goos string, size int64) error {
	if int64(len(payload)) != size {
		return fmt.Errorf("release: the payload is %d bytes and the Release published %d for this asset: a truncated download is refused rather than installed", len(payload), size)
	}
	if !hasExecutableMagic(payload, goos) {
		return fmt.Errorf("release: the payload does not begin with a %s object-file header (% x): it is not a binary this platform can run", goos, head(payload, 4))
	}
	if tag != "" && !bytes.Contains(payload, []byte(tag)) {
		return fmt.Errorf("release: the payload does not carry %s, the tag it was published under: the release workflow stamps every asset with -X github.com/LZD121/x-terminal/internal/cli.version=<tag>, so a binary without its own tag was not built for this Release", tag)
	}
	return nil
}

// hasExecutableMagic reports whether payload starts with an object-file header for
// goos: \x7fELF for linux, and the Mach-O magics for darwin (32- and 64-bit, each
// byte order, plus the universal headers the toolchain can emit).
//
// A platform this release does not publish for has no magic this build knows, and
// the arm then refuses the payload rather than passing vacuously — fail closed, on
// the same reasoning as the asset picker: a platform with no asset is answered by
// the release's own list, before anything is downloaded.
func hasExecutableMagic(payload []byte, goos string) bool {
	switch goos {
	case "linux":
		return bytes.HasPrefix(payload, []byte{0x7f, 'E', 'L', 'F'})
	case "darwin":
		for _, magic := range [][]byte{
			{0xfe, 0xed, 0xfa, 0xce}, // Mach-O 32-bit, big endian
			{0xce, 0xfa, 0xed, 0xfe}, // Mach-O 32-bit, little endian
			{0xfe, 0xed, 0xfa, 0xcf}, // Mach-O 64-bit, big endian
			{0xcf, 0xfa, 0xed, 0xfe}, // Mach-O 64-bit, little endian
			{0xca, 0xfe, 0xba, 0xbe}, // universal, big endian
			{0xbe, 0xba, 0xfe, 0xca}, // universal, little endian
		} {
			if bytes.HasPrefix(payload, magic) {
				return true
			}
		}
	}
	return false
}

// head returns the first n bytes of b, or b itself when it is shorter.
func head(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}

// Replace writes payload over target through a temporary file in the target's own
// directory.
//
// The order is the promise: create the temporary file beside the target, write the
// payload, Sync so the bytes are on disk before the rename publishes them, Chmod
// 0755 on the temporary file — the mode an installed binary needs, set before it is
// the target rather than after — Close, and only then rename over the target. Every
// failure path removes what it created, so a failed update leaves the directory
// byte-identical to what it was.
//
// A rename over a running executable is allowed and the running process keeps the
// image it was started with, so this takes effect at the next launch — which is
// what the verb prints.
func Replace(target string, payload []byte) error {
	tmp, err := writeTemp(filepath.Dir(target), payload)
	if err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return fmt.Errorf("release: renaming %s over %s: %w", tmp, target, err)
	}
	return nil
}

// writeTemp writes payload to a fresh temporary file in dir with the mode an
// installed binary needs, and returns its path. It is where this package creates a
// file, and it removes whatever it created on every failure path.
func writeTemp(dir string, payload []byte) (string, error) {
	f, err := os.CreateTemp(dir, updateTempPattern)
	if err != nil {
		return "", fmt.Errorf("release: creating a temporary file in %s: %w", dir, err)
	}
	tmp := f.Name()
	discard := func() {
		f.Close()      //nolint:errcheck
		os.Remove(tmp) //nolint:errcheck
	}
	if _, err := f.Write(payload); err != nil {
		discard()
		return "", fmt.Errorf("release: writing %s: %w", tmp, err)
	}
	if err := f.Sync(); err != nil {
		discard()
		return "", fmt.Errorf("release: flushing %s: %w", tmp, err)
	}
	if err := f.Chmod(0o755); err != nil {
		discard()
		return "", fmt.Errorf("release: setting the mode of %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return "", fmt.Errorf("release: closing %s: %w", tmp, err)
	}
	return tmp, nil
}
