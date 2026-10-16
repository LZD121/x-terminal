package cli

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/release"
	"github.com/LZD121/x-terminal/internal/session"
)

// fakeReleaseWire is releaseWire with no upstream at all: a test hands it the Release the verb
// should see, and it counts what was asked for. That count is how the arms below assert the shape
// the plan requires — --check spends exactly one API request and no download — without a socket
// leaving the machine.
type fakeReleaseWire struct {
	rel         release.Release
	latestErr   error
	latestCalls int
}

func (f *fakeReleaseWire) Latest(context.Context) (release.Release, error) {
	f.latestCalls++
	if f.latestErr != nil {
		return release.Release{}, f.latestErr
	}
	return f.rel, nil
}

// releaseWith builds a Release carrying the assets this platform's upgrade would take: the
// workflow's own names for the tag, and the per-asset digest beside each one.
func releaseWith(tag string) release.Release {
	name := release.AssetName(tag, runtime.GOOS, runtime.GOARCH)
	return release.Release{
		Tag: tag,
		Assets: []release.Asset{
			{Name: "SHA256SUMS", Size: 1, URL: "https://github.com/LZD121/x-terminal/releases/download/" + tag + "/SHA256SUMS"},
			{Name: name, Size: 1, URL: "https://github.com/LZD121/x-terminal/releases/download/" + tag + "/" + name},
			{Name: name + ".sha256", Size: 1, URL: "https://github.com/LZD121/x-terminal/releases/download/" + tag + "/" + name + ".sha256"},
		},
	}
}

// withReleaseWire replaces the client seam: the verb's tests reach no network, and production's
// release.New is the only thing that ever builds a real client.
func withReleaseWire(t *testing.T, wire *fakeReleaseWire) *fakeReleaseWire {
	t.Helper()
	previous := updateClient
	updateClient = func(config.Config, session.Secret) (releaseWire, error) { return wire, nil }
	t.Cleanup(func() { updateClient = previous })
	return wire
}

// withVersion stamps the build the way a release build is stamped, which is the only way a test can
// ask the comparison questions: the default is an unstamped working-tree build.
func withVersion(t *testing.T, v string) {
	t.Helper()
	previous := version
	version = v
	t.Cleanup(func() { version = previous })
}

// withUpdateEnv is this package's environment seam, replaced the way internal/config's and
// internal/session's are: no test depends on whether the developer's shell exports a token.
func withUpdateEnv(t *testing.T, values map[string]string) {
	t.Helper()
	previous := updateEnv
	updateEnv = func(name string) string { return values[name] }
	t.Cleanup(func() { updateEnv = previous })
}

// updateTestConfig points config.Load at a temp directory and clears the ambient proxy, so no arm
// reads the developer's own configuration.
func updateTestConfig(t *testing.T) {
	t.Helper()
	hermeticConfig(t, writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n"))
	withUpdateEnv(t, nil)
}

// TestUpdateCheckOnTheLatestReleaseAnswersWithoutDownloading is acceptance 1's answer arm: the
// build is the latest release, the verb says so, exits OK, spends exactly one API request and
// touches nothing.
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
	wantRelease := fmt.Sprintf("release v1.1.0, %s/%s asset %s\n", runtime.GOOS, runtime.GOARCH, release.AssetName("v1.1.0", runtime.GOOS, runtime.GOARCH))
	if !strings.Contains(stdout, wantRelease) {
		t.Errorf("update --check did not print the resolved release line %q:\n%s", wantRelease, stdout)
	}
	if want := "this build (v1.1.0) is the latest release\n"; !strings.Contains(stdout, want) {
		t.Errorf("update --check did not print %q:\n%s", want, stdout)
	}
}

// TestUpdateCheckTwoTagsBehindPrintsTheAvailableLine: the upgrade exists, and --check stops at
// saying so — the resolved tag and the asset it would take, in that order.
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
	if want := "x-terminal v1.2.0 is available (this build is v1.0.0)\n"; !strings.Contains(stdout, want) {
		t.Errorf("update --check did not print %q:\n%s", want, stdout)
	}
	if !strings.Contains(stdout, release.AssetName("v1.2.0", runtime.GOOS, runtime.GOARCH)) {
		t.Errorf("update --check did not name the asset it would take:\n%s", stdout)
	}
}

// TestUpdateCheckAheadOfTheLatestRefusesTheDowngrade: the operator's build is newer than the latest
// release. That is an answer, not a failure to answer, so it exits OK — and it says there is
// nothing to install.
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

// TestUpdateOnAnUnstampedBuildNamesTheState: a build with no version stamp has no ordering against
// any release, so the verb refuses rather than inventing one — and the state carries the runbook
// pointer and the fix.
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

// TestUpdateCheckNamesAnAnonymousNotFound is the state a private repository produced: GitHub answers
// 404 both for a repository it cannot see and for a release that does not exist, so the detail says
// both and names the token to export — with a different sentence when one was already set.
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

// TestUpdateCheckNamesAServerError: anything that is not a 404 is the status and the URL, which is
// what tells GitHub's own fault from a repository that is not visible.
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

// TestUpdateCheckReportsAPlatformWithNoAsset: the asset list comes from the Release itself, so a
// platform it does not build for is a named state naming the asset that was wanted — never a 404
// on a download URL derived from the tag.
func TestUpdateCheckReportsAPlatformWithNoAsset(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v1.1.0")
	withReleaseWire(t, &fakeReleaseWire{rel: release.Release{
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
}

// TestUpdateRejectsAnUnknownArgument: the verb's own line and the usage to stderr, exactly as sync
// does for a flag nobody knows.
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

// TestUpdateReadsTheTokenInTheDocumentedOrder: GH_TOKEN then GITHUB_TOKEN, first non-empty wins, and
// an absent pair is the anonymous path — which is the primary one, since the repository is public.
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

// TestUpdateWithoutCheckDoesNotInstallYet: this build's verb answers the question, and the install
// half lands beside the same-directory rename it makes. The arm exists so that the interim shape is
// tested rather than silent; the install's own arms replace it.
func TestUpdateWithoutCheckDoesNotInstallYet(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v1.1.0")
	wire := withReleaseWire(t, &fakeReleaseWire{rel: releaseWith("v1.2.0")})

	code, _, stderr := run(t, Update)
	if code != ExitError {
		t.Fatalf("a bare update exited %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr, "cannot install yet") {
		t.Errorf("a bare update did not say what it can and cannot do: %s", stderr)
	}
	if wire.latestCalls != 0 {
		t.Errorf("a bare update spent %d requests on work it did not do", wire.latestCalls)
	}
}

// TestUpdateIsDispatchableFromTheFrontDoor is the arm that proves the table row and the body are
// joined: Main is the path the binary itself takes.
func TestUpdateIsDispatchableFromTheFrontDoor(t *testing.T) {
	updateTestConfig(t)
	withVersion(t, "v1.1.0")
	withReleaseWire(t, &fakeReleaseWire{rel: releaseWith("v1.2.0")})

	if code := Main([]string{"update", "--check"}); code != ExitOK {
		t.Fatalf("x-terminal update --check exited %d, want %d", code, ExitOK)
	}
}
