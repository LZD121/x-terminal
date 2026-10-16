package cli

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/release"
	"github.com/LZD121/x-terminal/internal/session"
)

// fakeReleaseWire is releaseWire with no upstream at all: a test hands it the Release
// the verb should see and the bytes each URL answers with, and it counts what was
// asked for. Those counts are how the arms below assert the plan's shape — --check
// spends exactly one API request and no download, an unwritable directory spends the
// API request and nothing else — without a socket leaving the machine.
type fakeReleaseWire struct {
	rel         release.Release
	latestErr   error
	bodies      map[string][]byte
	latestCalls int
	fetches     []string
}

func (f *fakeReleaseWire) Latest(context.Context) (release.Release, error) {
	f.latestCalls++
	if f.latestErr != nil {
		return release.Release{}, f.latestErr
	}
	return f.rel, nil
}

func (f *fakeReleaseWire) Fetch(_ context.Context, url string, _ int64) ([]byte, error) {
	f.fetches = append(f.fetches, url)
	body, ok := f.bodies[url]
	if !ok {
		return nil, fmt.Errorf("fake: nothing is served at %s", url)
	}
	return body, nil
}

// releaseWith builds a Release carrying the assets this platform's upgrade would take:
// the workflow's own names for the tag, and the per-asset digest beside each one.
func releaseWith(tag string) release.Release {
	name := release.AssetName(tag, runtime.GOOS, runtime.GOARCH)
	base := "https://github.com/LZD121/x-terminal/releases/download/" + tag + "/"
	return release.Release{
		Tag: tag,
		Assets: []release.Asset{
			{Name: "SHA256SUMS", Size: 1, URL: base + "SHA256SUMS"},
			{Name: name, Size: 1, URL: base + name},
			{Name: name + ".sha256", Size: 1, URL: base + name + ".sha256"},
		},
	}
}

// installablePayload is a payload with only the shape the verb's checks want: this
// platform's object-file magic, the tag it would have been published under, and the
// length the test asks for. Nothing here is a real binary, and nothing lands in the
// tree.
func installablePayload(tag string, size int) []byte {
	payload := make([]byte, size)
	magic := []byte{0x7f, 'E', 'L', 'F'}
	if runtime.GOOS == "darwin" {
		magic = []byte{0xcf, 0xfa, 0xed, 0xfe}
	}
	copy(payload, magic)
	copy(payload[8:], tag)
	for i := 8 + len(tag); i < size; i++ {
		payload[i] = 'x'
	}
	return payload
}

// served is a Release whose asset and digest are served from memory, with the digest
// written the way the release workflow writes it (`sha256sum "${name}" > "${name}.sha256"`).
// It returns the asset too, so an arm can name the URLs it expects to see fetched.
func served(tag string, payload []byte) (*fakeReleaseWire, release.Asset) {
	name := release.AssetName(tag, runtime.GOOS, runtime.GOARCH)
	base := "https://github.com/LZD121/x-terminal/releases/download/" + tag + "/"
	asset := release.Asset{Name: name, Size: int64(len(payload)), URL: base + name}
	digest := release.Asset{Name: name + ".sha256", Size: 80, URL: base + name + ".sha256"}
	wire := &fakeReleaseWire{
		rel: release.Release{Tag: tag, Assets: []release.Asset{
			{Name: "SHA256SUMS", Size: 1, URL: base + "SHA256SUMS"},
			asset,
			digest,
		}},
		bodies: map[string][]byte{
			asset.URL:  payload,
			digest.URL: []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(payload), name)),
		},
	}
	return wire, asset
}

// withReleaseWire replaces the client seam: the verb's tests reach no network, and
// production's release.New is the only thing that ever builds a real client.
func withReleaseWire(t *testing.T, wire *fakeReleaseWire) *fakeReleaseWire {
	t.Helper()
	previous := updateClient
	updateClient = func(config.Config, session.Secret) (releaseWire, error) { return wire, nil }
	t.Cleanup(func() { updateClient = previous })
	return wire
}

// withVersion stamps the build the way a release build is stamped, which is the only way
// a test can ask the comparison questions: the default is an unstamped working-tree build.
func withVersion(t *testing.T, v string) {
	t.Helper()
	previous := version
	version = v
	t.Cleanup(func() { version = previous })
}

// withUpdateEnv is this package's environment seam, replaced the way internal/config's
// and internal/session's are: no test depends on whether the developer's shell exports
// a token.
func withUpdateEnv(t *testing.T, values map[string]string) {
	t.Helper()
	previous := updateEnv
	updateEnv = func(name string) string { return values[name] }
	t.Cleanup(func() { updateEnv = previous })
}

// withExecutable points the verb at a file in a temporary directory instead of the
// running test binary, and returns its path and the bytes it held.
func withExecutable(t *testing.T, body string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "x-terminal")
	if err := os.WriteFile(target, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	previous := updateExecutable
	updateExecutable = func() (string, error) { return target, nil }
	t.Cleanup(func() { updateExecutable = previous })
	return target, dir
}

// updateTestConfig points config.Load at a temp directory and clears the ambient proxy,
// so no arm reads the developer's own configuration.
func updateTestConfig(t *testing.T) {
	t.Helper()
	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	withUpdateEnv(t, nil)
}

// TestUpdateCheckOnTheLatestReleaseAnswersWithoutDownloading is acceptance 1's answer
// arm: the build is the latest release, the verb says so, exits OK, spends exactly one
// API request and touches nothing.
func TestUpdateCheckOnTheLatestReleaseAnswersWithoutDownloading(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v1.1.0")
	wire := withReleaseWire(t, &fakeReleaseWire{rel: releaseWith("v1.1.0")})

	code, stdout, stderr := run(t, Update, "--check")
	if code != ExitOK {
		t.Fatalf("update --check exited %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if wire.latestCalls != 1 {
		t.Errorf("update --check made %d API requests, want exactly one", wire.latestCalls)
	}
	if len(wire.fetches) != 0 {
		t.Errorf("update --check downloaded %v", wire.fetches)
	}
	wantRelease := fmt.Sprintf("release v1.1.0, %s/%s asset %s\n", runtime.GOOS, runtime.GOARCH, release.AssetName("v1.1.0", runtime.GOOS, runtime.GOARCH))
	if !strings.Contains(stdout, wantRelease) {
		t.Errorf("update --check did not print the resolved release line %q:\n%s", wantRelease, stdout)
	}
	if want := "this build (v1.1.0) is the latest release\n"; !strings.Contains(stdout, want) {
		t.Errorf("update --check did not print %q:\n%s", want, stdout)
	}
}

// TestUpdateCheckTwoTagsBehindPrintsTheAvailableLine: the upgrade exists, and --check
// stops at saying so — the resolved tag and the asset it would take, in that order.
func TestUpdateCheckTwoTagsBehindPrintsTheAvailableLine(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v1.0.0")
	wire := withReleaseWire(t, &fakeReleaseWire{rel: releaseWith("v1.2.0")})

	code, stdout, stderr := run(t, Update, "--check")
	if code != ExitOK {
		t.Fatalf("update --check exited %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if wire.latestCalls != 1 {
		t.Errorf("update --check made %d API requests, want exactly one", wire.latestCalls)
	}
	if len(wire.fetches) != 0 {
		t.Errorf("update --check downloaded %v", wire.fetches)
	}
	if want := "x-terminal v1.2.0 is available (this build is v1.0.0)\n"; !strings.Contains(stdout, want) {
		t.Errorf("update --check did not print %q:\n%s", want, stdout)
	}
	if !strings.Contains(stdout, release.AssetName("v1.2.0", runtime.GOOS, runtime.GOARCH)) {
		t.Errorf("update --check did not name the asset it would take:\n%s", stdout)
	}
}

// TestUpdateCheckAheadOfTheLatestRefusesTheDowngrade: the operator's build is newer than
// the latest release. That is an answer, not a failure to answer, so it exits OK — and it
// says there is nothing to install.
func TestUpdateCheckAheadOfTheLatestRefusesTheDowngrade(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v2.0.0")
	withReleaseWire(t, &fakeReleaseWire{rel: releaseWith("v1.1.0")})

	code, stdout, stderr := run(t, Update, "--check")
	if code != ExitOK {
		t.Fatalf("a downgrade refusal exited %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if want := "the latest release (v1.1.0) is older than this build (v2.0.0): nothing to install\n"; !strings.Contains(stdout, want) {
		t.Errorf("update --check did not print %q:\n%s", want, stdout)
	}
}

// TestUpdateOnAnUnstampedBuildNamesTheState: a build with no version stamp has no ordering
// against any release, so the verb refuses rather than inventing one — and the state
// carries the runbook pointer and the fix.
func TestUpdateOnAnUnstampedBuildNamesTheState(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "")
	withReleaseWire(t, &fakeReleaseWire{rel: releaseWith("v1.1.0")})

	code, _, stderr := run(t, Update, "--check")
	if code != ExitError {
		t.Fatalf("an unstamped build exited %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr, stateNotARelease) {
		t.Errorf("the state line does not name %q: %s", stateNotARelease, stderr)
	}
	if !strings.Contains(stderr, "RUNBOOK.md#"+updateRunbookAnchor) {
		t.Errorf("the state line carries no runbook pointer: %s", stderr)
	}
}

// TestUpdateCheckNamesAnAnonymousNotFound is the state a private repository produced:
// GitHub answers 404 both for a repository it cannot see and for a release that does not
// exist, so the detail says both and names the token to export — with a different sentence
// when one was already set.
func TestUpdateCheckNamesAnAnonymousNotFound(t *testing.T) {
	const apiURL = "https://api.github.com/repos/LZD121/x-terminal/releases/latest"

	for _, tc := range []struct {
		name     string
		env      map[string]string
		wantWord string
	}{
		{name: "anonymous", env: nil, wantWord: "export GH_TOKEN or GITHUB_TOKEN"},
		{name: "with a token", env: map[string]string{"GH_TOKEN": "ghp_fixture"}, wantWord: "even with GH_TOKEN set"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updateTestConfig(t)
			withVersion(t, "v1.1.0")
			withUpdateEnv(t, tc.env)
			withReleaseWire(t, &fakeReleaseWire{
				latestErr: &release.HTTPError{Status: 404, URL: apiURL, Body: `{"message":"Not Found"}`},
			})

			code, stdout, stderr := run(t, Update, "--check")
			if code != ExitError {
				t.Fatalf("a 404 exited %d, want %d", code, ExitError)
			}
			if !strings.Contains(stderr, stateNotReachable) {
				t.Errorf("the state line does not name %q: %s", stateNotReachable, stderr)
			}
			if !strings.Contains(stderr, apiURL) {
				t.Errorf("the state line does not name the URL that answered: %s", stderr)
			}
			if !strings.Contains(stderr, tc.wantWord) {
				t.Errorf("the state line does not say %q: %s", tc.wantWord, stderr)
			}
			for _, token := range tc.env {
				if strings.Contains(stdout+stderr, token) {
					t.Errorf("the token reached the output: %s%s", stdout, stderr)
				}
			}
		})
	}
}

// TestUpdateCheckNamesAServerError: anything that is not a 404 is the status and the URL,
// which is what tells GitHub's own fault from a repository that is not visible.
func TestUpdateCheckNamesAServerError(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v1.1.0")
	withReleaseWire(t, &fakeReleaseWire{
		latestErr: &release.HTTPError{Status: 503, URL: "https://api.github.com/repos/LZD121/x-terminal/releases/latest", Body: "Service Unavailable"},
	})

	code, _, stderr := run(t, Update, "--check")
	if code != ExitError {
		t.Fatalf("a 503 exited %d, want %d", code, ExitError)
	}
	for _, want := range []string{stateNotReachable, "503", "Service Unavailable"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the state line does not carry %q: %s", want, stderr)
		}
	}
}

// TestUpdateCheckReportsAPlatformWithNoAsset: the asset list comes from the Release
// itself, so a platform it does not build for is a named state naming the asset that was
// wanted — never a 404 on a download URL derived from the tag.
func TestUpdateCheckReportsAPlatformWithNoAsset(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v1.1.0")
	wire := withReleaseWire(t, &fakeReleaseWire{rel: release.Release{
		Tag:    "v1.1.0",
		Assets: []release.Asset{{Name: "SHA256SUMS", Size: 1, URL: "https://github.com/x/SHA256SUMS"}},
	}})

	code, _, stderr := run(t, Update, "--check")
	if code != ExitError {
		t.Fatalf("a release with no asset for this platform exited %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr, stateNoAsset) {
		t.Errorf("the state line does not name %q: %s", stateNoAsset, stderr)
	}
	if want := release.AssetName("v1.1.0", runtime.GOOS, runtime.GOARCH); !strings.Contains(stderr, want) {
		t.Errorf("the state line does not name the asset it wanted (%s): %s", want, stderr)
	}
	if len(wire.fetches) != 0 {
		t.Errorf("a release with no asset for this platform downloaded %v", wire.fetches)
	}
}

// TestUpdateRejectsAnUnknownArgument: the verb's own line and the usage to stderr, exactly
// as sync does for a flag nobody knows.
func TestUpdateRejectsAnUnknownArgument(t *testing.T) {
	updateTestConfig(t)
	code, _, stderr := run(t, Update, "--dry-run")
	if code != ExitUsage {
		t.Fatalf("an unknown argument exited %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "update: unknown argument --dry-run") {
		t.Errorf("the refusal does not name the argument: %s", stderr)
	}
	if !strings.Contains(stderr, "usage: x-terminal") {
		t.Errorf("the refusal did not print the usage: %s", stderr)
	}
}

// TestUpdateReadsTheTokenInTheDocumentedOrder: GH_TOKEN then GITHUB_TOKEN, first non-empty
// wins, and an absent pair is the anonymous path — which is the primary one, since the
// repository is public.
func TestUpdateReadsTheTokenInTheDocumentedOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		want session.Secret
	}{
		{name: "GH_TOKEN wins", env: map[string]string{"GH_TOKEN": "from_gh", "GITHUB_TOKEN": "from_github"}, want: "from_gh"},
		{name: "GITHUB_TOKEN alone", env: map[string]string{"GITHUB_TOKEN": "from_github"}, want: "from_github"},
		{name: "anonymous", env: nil, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updateTestConfig(t)
			withVersion(t, "v1.1.0")
			withUpdateEnv(t, tc.env)
			var got session.Secret
			previous := updateClient
			updateClient = func(_ config.Config, token session.Secret) (releaseWire, error) {
				got = token
				return &fakeReleaseWire{rel: releaseWith("v1.1.0")}, nil
			}
			t.Cleanup(func() { updateClient = previous })

			if code, _, stderr := run(t, Update, "--check"); code != ExitOK {
				t.Fatalf("update --check exited %d: %s", code, stderr)
			}
			if got != tc.want {
				t.Errorf("the verb carried %q, want %q", got.Reveal(), tc.want.Reveal())
			}
		})
	}
}

// TestUpdateInstallsTheVerifiedAsset is the happy path end to end: the target the process
// was started from becomes the served payload, at the mode an installed binary needs, and
// the output says which path took the bytes, which digest verified them, and when the
// change takes effect.
func TestUpdateInstallsTheVerifiedAsset(t *testing.T) {
	updateTestConfig(t)
	target, _ := withExecutable(t, "the old binary")
	withVersion(t, "v1.1.0")
	payload := installablePayload("v1.2.0", 64)
	wire, _ := served("v1.2.0", payload)
	withReleaseWire(t, wire)

	before, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := run(t, Update)
	if code != ExitOK {
		t.Fatalf("update exited %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Errorf("the target holds %q, want the served payload", got)
	}
	after, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Error("the target is the same file it was: the payload was written in place rather than renamed over it")
	}
	if perm := after.Mode().Perm(); perm != 0o755 {
		t.Errorf("the installed file is mode %04o, want 0755", perm)
	}
	for _, want := range []string{
		"replacing " + target + " (this build v1.1.0) with v1.2.0",
		"x-terminal v1.2.0 is available (this build is v1.1.0)",
		"verified against " + release.AssetName("v1.2.0", runtime.GOOS, runtime.GOARCH) + ".sha256",
		"replaced " + target + " with v1.2.0",
		"the process that ran this is still the old image: the next launch reports v1.2.0",
		"v1.1.0 -> v1.2.0",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the output does not carry %q:\n%s", want, stdout)
		}
	}
}

// TestUpdateAlreadyLatestChangesNothing is acceptance 1 at the verb: the build is the
// latest release, so nothing is downloaded and the installed file's bytes are identical.
func TestUpdateAlreadyLatestChangesNothing(t *testing.T) {
	updateTestConfig(t)
	target, _ := withExecutable(t, "the installed bytes")
	withVersion(t, "v1.1.0")
	wire, _ := served("v1.1.0", installablePayload("v1.1.0", 64))
	withReleaseWire(t, wire)

	code, stdout, stderr := run(t, Update)
	if code != ExitOK {
		t.Fatalf("update at the latest release exited %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if want := "this build (v1.1.0) is the latest release\n"; !strings.Contains(stdout, want) {
		t.Errorf("the output does not carry %q:\n%s", want, stdout)
	}
	if len(wire.fetches) != 0 {
		t.Errorf("an up-to-date build downloaded %v", wire.fetches)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "the installed bytes" {
		t.Errorf("an up-to-date run changed the installed file to %q", got)
	}
}

// TestUpdateRefusesADigestThatDoesNotMatch: the bytes that arrived are not the bytes the
// Release publishes, so nothing is installed and the target is untouched.
func TestUpdateRefusesADigestThatDoesNotMatch(t *testing.T) {
	updateTestConfig(t)
	target, _ := withExecutable(t, "the installed bytes")
	withVersion(t, "v1.1.0")
	wire, asset := served("v1.2.0", installablePayload("v1.2.0", 64))
	// The digest published for a different payload: the shape is perfect and the bytes
	// are not, which is exactly the corruption the digest is there to catch.
	wire.bodies[asset.URL+".sha256"] = []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256([]byte("other bytes")), asset.Name))
	withReleaseWire(t, wire)

	code, _, stderr := run(t, Update)
	if code != ExitError {
		t.Fatalf("a digest mismatch exited %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr, stateNotVerified) || !strings.Contains(stderr, "RUNBOOK.md#"+updateRunbookAnchor) {
		t.Errorf("the state line does not name %q with its anchor: %s", stateNotVerified, stderr)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "the installed bytes" {
		t.Errorf("a failed verification left %q in place, want the installed bytes", got)
	}
}

// TestUpdateRefusesAPayloadThatIsNotThisBuild: a payload with a matching digest is still
// refused when its shape is not an object file carrying the tag it was published under,
// and the same state covers a digest file that names another platform's artifact.
func TestUpdateRefusesAPayloadThatIsNotThisBuild(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload []byte
		digest  func(asset release.Asset, payload []byte) []byte
	}{
		{
			name:    "a text body with a matching digest",
			payload: []byte("#!/bin/sh\necho v1.2.0\n"),
			digest: func(asset release.Asset, payload []byte) []byte {
				return []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(payload), asset.Name))
			},
		},
		{
			name:    "a digest for another platform's asset",
			payload: installablePayload("v1.2.0", 64),
			digest: func(_ release.Asset, payload []byte) []byte {
				return []byte(fmt.Sprintf("%x  x-terminal_v1.2.0_darwin_amd64\n", sha256.Sum256(payload)))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updateTestConfig(t)
			target, _ := withExecutable(t, "the installed bytes")
			withVersion(t, "v1.1.0")
			wire, asset := served("v1.2.0", tc.payload)
			wire.bodies[asset.URL+".sha256"] = tc.digest(asset, tc.payload)
			withReleaseWire(t, wire)

			code, _, stderr := run(t, Update)
			if code != ExitError {
				t.Fatalf("an unshaped payload exited %d, want %d", code, ExitError)
			}
			if !strings.Contains(stderr, stateNotVerified) {
				t.Errorf("the state line does not name %q: %s", stateNotVerified, stderr)
			}
			if got, err := os.ReadFile(target); err != nil || string(got) != "the installed bytes" {
				t.Errorf("a refused payload left %q in place (%v)", got, err)
			}
		})
	}
}

// TestUpdateReportsAReleaseWithNoDigest: a Release that publishes no per-asset digest has
// published nothing this verb can verify, and an unverifiable payload is not a payload
// that is probably fine. The download is not even attempted.
func TestUpdateReportsAReleaseWithNoDigest(t *testing.T) {
	updateTestConfig(t)
	target, _ := withExecutable(t, "the installed bytes")
	withVersion(t, "v1.1.0")
	wire, asset := served("v1.2.0", installablePayload("v1.2.0", 64))
	wire.rel.Assets = []release.Asset{asset}
	withReleaseWire(t, wire)

	code, _, stderr := run(t, Update)
	if code != ExitError {
		t.Fatalf("a release with no digest exited %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr, stateNotVerified) {
		t.Errorf("the state line does not name %q: %s", stateNotVerified, stderr)
	}
	if len(wire.fetches) != 0 {
		t.Errorf("a release with no digest still downloaded %v", wire.fetches)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "the installed bytes" {
		t.Errorf("a refused release left %q in place (%v)", got, err)
	}
}

// TestUpdateReportsAnUnwritableInstallPath: the free local question is asked before the
// two downloads, so a machine that cannot take the update is never made to fetch bytes —
// the request count is the point of that order. Root ignores the mode, so this arm skips
// under root rather than lying.
func TestUpdateReportsAnUnwritableInstallPath(t *testing.T) {
	updateTestConfig(t)
	if os.Geteuid() == 0 {
		t.Skip("running as root: mode 0500 does not refuse root, so this arm cannot fail here")
	}
	target, dir := withExecutable(t, "the installed bytes")
	withVersion(t, "v1.1.0")
	wire, _ := served("v1.2.0", installablePayload("v1.2.0", 64))
	withReleaseWire(t, wire)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) }) //nolint:errcheck

	code, _, stderr := run(t, Update)
	if code != ExitError {
		t.Fatalf("an unwritable directory exited %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr, stateNotWritable) || !strings.Contains(stderr, "RUNBOOK.md#"+updateRunbookAnchor) {
		t.Errorf("the state line does not name %q with its anchor: %s", stateNotWritable, stderr)
	}
	if wire.latestCalls != 1 {
		t.Errorf("the verb made %d API requests, want the one the comparison needs", wire.latestCalls)
	}
	if len(wire.fetches) != 0 {
		t.Errorf("an unwritable install path still downloaded %v", wire.fetches)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "the installed bytes" {
		t.Errorf("a refused install left %q in place (%v)", got, err)
	}
}

// TestUpdateForceInstallsTheSameTag: --force is the operator choosing which release to
// take, so the resolved release is installed even when the running build is already on it.
func TestUpdateForceInstallsTheSameTag(t *testing.T) {
	updateTestConfig(t)
	target, _ := withExecutable(t, "the installed bytes")
	withVersion(t, "v1.1.0")
	payload := installablePayload("v1.1.0", 64)
	wire, _ := served("v1.1.0", payload)
	withReleaseWire(t, wire)

	code, stdout, stderr := run(t, Update, "--force")
	if code != ExitOK {
		t.Fatalf("update --force exited %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != string(payload) {
		t.Errorf("update --force left %q in place (%v), want the served payload", got, err)
	}
	if !strings.Contains(stdout, "v1.1.0 -> v1.1.0") {
		t.Errorf("the output does not name the old and new tags:\n%s", stdout)
	}
}

// TestUpdateForceDoesNotReachPastTheChecks: --force chooses which release to take, never
// whether the bytes verified. With the digest gone from the asset list the install fails
// exactly as it does without the flag, and the target is untouched.
func TestUpdateForceDoesNotReachPastTheChecks(t *testing.T) {
	updateTestConfig(t)
	target, _ := withExecutable(t, "the installed bytes")
	withVersion(t, "v1.1.0")
	wire, asset := served("v1.1.0", installablePayload("v1.1.0", 64))
	wire.rel.Assets = []release.Asset{asset}
	withReleaseWire(t, wire)

	code, _, stderr := run(t, Update, "--force")
	if code != ExitError {
		t.Fatalf("update --force with no digest exited %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr, stateNotVerified) {
		t.Errorf("the state line does not name %q: %s", stateNotVerified, stderr)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "the installed bytes" {
		t.Errorf("a forced install without a digest left %q in place (%v)", got, err)
	}
}

// TestUpdateNeverPrintsTheToken sweeps the two streams of a successful run for the
// credential the operator exported: the verb's output is the thing a user pastes, so no
// line of it may carry the token.
func TestUpdateNeverPrintsTheToken(t *testing.T) {
	updateTestConfig(t)
	_, _ = withExecutable(t, "the installed bytes")
	withVersion(t, "v1.1.0")
	withUpdateEnv(t, map[string]string{"GH_TOKEN": "ghp_a_token_nobody_may_print"})
	wire, _ := served("v1.2.0", installablePayload("v1.2.0", 64))
	withReleaseWire(t, wire)

	code, stdout, stderr := run(t, Update)
	if code != ExitOK {
		t.Fatalf("update exited %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if strings.Contains(stdout+stderr, "ghp_a_token_nobody_may_print") {
		t.Errorf("the token reached the output: %s%s", stdout, stderr)
	}
	if !strings.Contains(stdout, "replaced ") {
		t.Errorf("the run did not install anything, so the sweep proves nothing:\n%s", stdout)
	}
}

// TestUpdateIsDispatchableFromTheFrontDoor is the arm that proves the table row and the
// body are joined: Main is the path the binary itself takes.
func TestUpdateIsDispatchableFromTheFrontDoor(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v1.1.0")
	withReleaseWire(t, &fakeReleaseWire{rel: releaseWith("v1.2.0")})

	if code := Main([]string{"update", "--check"}); code != ExitOK {
		t.Fatalf("x-terminal update --check exited %d, want %d", code, ExitOK)
	}
}
