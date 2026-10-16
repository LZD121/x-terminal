package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/release"
	"github.com/LZD121/x-terminal/internal/session"
)

// updateTimeout bounds one whole verb run, downloads included. A verb that never
// returns is one of the ways this tool could stop telling the truth about what it
// is doing.
const updateTimeout = 2 * time.Minute

// updateRunbookAnchor is the RUNBOOK.md section every state this verb can print
// points at. The anchor and the states are declared here, once, because three
// things read them: the verb prints them, the runbook join maps through them
// (internal/cli/runbook_test.go), and that test's classifier vocabulary is built
// from updateStates rather than typed a second time.
const updateRunbookAnchor = "the-update-path"

// The five states this verb can fail in. Each one names a fact the operator can
// act on, and each one carries the anchor above in its printed line.
const (
	stateNotReachable = "the release is not reachable"
	stateNoAsset      = "no release asset for this platform"
	stateNotVerified  = "the download did not verify"
	stateNotWritable  = "the install path is not writable"
	stateNotARelease  = "this build is not a release"
)

// updateStates is every state the verb can print, in the order it declares them.
var updateStates = []string{stateNotReachable, stateNoAsset, stateNotVerified, stateNotWritable, stateNotARelease}

// stateLine is one state as the operator reads it: the state, the detail that says
// which release, path or platform it is about, and the RUNBOOK.md section that
// repairs it. It goes to stderr — it is a diagnostic, not the answer --check was
// asked for.
func stateLine(state, detail string) string {
	return fmt.Sprintf("update: %s: %s (RUNBOOK.md#%s)", state, detail, updateRunbookAnchor)
}

// updateEnv is this package's view of the environment, in internal/config's and
// internal/session's words and for their reason: the tests replace it, so no test
// depends on whether the developer's shell exports a token. A GitHub token is a
// credential rather than a setting — internal/config's own header says secrets
// are not its business — so it is read here and carried as a session.Secret from
// the moment it is read.
var updateEnv = os.Getenv

// releaseWire is the release wire the verb speaks through: resolve the latest
// Release, and fetch one URL's bytes.
//
// It is an interface rather than *release.Client because this package's tests
// have to drive the verb without a socket, and release's client keeps its origin
// and its token unexported on purpose — a test outside that package has no way to
// point one at a fake upstream, and giving the production type an exported way to
// be redirected is exactly what the plan refused. The wire's own request shape is
// asserted inside internal/release, where the client is built.
type releaseWire interface {
	Latest(ctx context.Context) (release.Release, error)
}

// updateClient is the seam this verb's tests replace. Production builds the real
// release client over the transport the resolved configuration names; a test
// replaces it with one that answers from memory and reaches no network at all.
var updateClient = func(cfg config.Config, token session.Secret) (releaseWire, error) {
	return release.New(cfg, token)
}

// Update is the `update` verb: replace the file this process was started from with
// the asset this platform's latest GitHub Release carries, after verifying the
// digest the same Release publishes.
//
// Exit codes follow the verb's own single rule: an answer the verb can print — up
// to date, an upgrade available, a downgrade refused, a build with no release
// identity — is ExitOK, and ExitError is reserved for the states where the verb
// could not do the thing it was asked (the release unreachable, no asset for this
// platform, the download unverified, the install path unwritable). A mistaken
// invocation is ExitUsage, as everywhere.
//
// The token is read from GH_TOKEN, then GITHUB_TOKEN, first non-empty wins, and
// an absent token means the anonymous path — which is the primary one, since the
// repository is public. Nothing writes it: no file, no keyring, no config key, and
// no line of this verb prints it or its length.
func Update(ctx *Context, args []string) int {
	var (
		check bool
		force bool
	)
	for _, arg := range args {
		switch arg {
		case "--check":
			check = true
		case "--force":
			force = true
		default:
			fmt.Fprintf(ctx.Stderr, "update: unknown argument %s\n", arg)
			usage(ctx.Stderr)
			return ExitUsage
		}
	}

	if !check {
		// The install is the other half of this verb and it is not in this change:
		// it lands beside the same-directory rename it makes, with the audit's
		// seventh persistence destination. Until it is here the verb refuses to
		// look like it did something it did not do, and it names the invocation
		// that answers the question.
		fmt.Fprintln(ctx.Stderr, "update: this build answers the question and cannot install yet: run `x-terminal update --check`")
		return ExitError
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "update: %v\n", err)
		return ExitError
	}

	token, tokenSet := updateToken()
	// The build's own identity goes into the User-Agent: this package owns the
	// version string and internal/release cannot import it, so it is handed over
	// rather than duplicated there.
	release.SetIdentity(Version())
	client, err := updateClient(cfg, token)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "update: %v\n", err)
		return ExitError
	}

	runCtx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()

	rel, err := client.Latest(runCtx)
	if err != nil {
		state, detail := updateUnreachable(err, tokenSet)
		fmt.Fprintln(ctx.Stderr, stateLine(state, detail))
		return ExitError
	}

	// The asset list comes from the Release itself and this is the only place it is
	// read: an asset name derived from the tag would answer a platform this Release
	// does not build for with a 404 on a download URL instead of a state the verb
	// can print. It is resolved before either verdict because it is the fact the
	// answer rests on, and only presence facts are printed.
	asset, assetErr := release.AssetFor(rel.Assets, rel.Tag, runtime.GOOS, runtime.GOARCH)
	if assetErr != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNoAsset, assetErr.Error()))
		return ExitError
	}
	fmt.Fprintf(ctx.Stdout, "release %s, %s/%s asset %s\n", rel.Tag, runtime.GOOS, runtime.GOARCH, asset.Name)

	running := Version()
	// A build with no version stamp has no ordering against the release, and
	// inventing one would be the verb guessing which release the operator is on.
	// --force is the operator saying they do not care about the ordering; it never
	// reaches past a check that verifies bytes.
	if running == "" {
		if !force {
			fmt.Fprintln(ctx.Stderr, stateLine(stateNotARelease,
				"install a published asset, or build with the release workflow's -ldflags stamps"))
			return ExitError
		}
		return ExitOK
	}

	order, err := release.CompareTags(running, rel.Tag)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "update: %v\n", err)
		return ExitError
	}
	updateVerdict(ctx, running, rel.Tag, order)
	return ExitOK
}

// updateVerdict prints what the comparison says: the release is newer, this build
// is the release, or the release is older. All three are answers the verb can
// print, which is why all three leave ExitOK — a downgrade refusal is an answer
// and not a failure to answer.
func updateVerdict(ctx *Context, running, tag string, order int) {
	switch {
	case order < 0:
		fmt.Fprintf(ctx.Stdout, "x-terminal %s is available (this build is %s)\n", tag, running)
	case order == 0:
		fmt.Fprintf(ctx.Stdout, "this build (%s) is the latest release\n", running)
	default:
		fmt.Fprintf(ctx.Stdout, "the latest release (%s) is older than this build (%s): nothing to install\n", tag, running)
	}
}

// updateToken reads the operator's GitHub token, in the documented order: GH_TOKEN
// then GITHUB_TOKEN, first non-empty wins. The second return says whether one was
// set at all, because an anonymous 404 and a token-bearing 404 are two different
// facts and the operator needs to know which one they have.
func updateToken() (session.Secret, bool) {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if value := updateEnv(name); value != "" {
			return session.Secret(value), true
		}
	}
	return "", false
}

// updateUnreachable names what a failed read of the release means.
//
// The 404 arm is spelled out because GitHub answers 404 both for a repository a
// token cannot see and for a release that does not exist: on a repository that may
// be private the two are indistinguishable from here, so the detail names the
// token to export and, when one was already set, says that too — "even with
// GH_TOKEN set" is a different fact from "no token was set".
func updateUnreachable(err error, tokenSet bool) (string, string) {
	var httpErr *release.HTTPError
	if !errors.As(err, &httpErr) {
		return stateNotReachable, err.Error()
	}
	if httpErr.Status == 404 {
		if tokenSet {
			return stateNotReachable, fmt.Sprintf(
				"%s answered 404 even with GH_TOKEN set: GitHub answers 404 both for a repository a token cannot see and for a release that does not exist, so the tag or the repository name is the fact to check",
				httpErr.URL)
		}
		return stateNotReachable, fmt.Sprintf(
			"%s answered 404: GitHub answers 404 both for a repository it cannot see and for a release that does not exist; export GH_TOKEN or GITHUB_TOKEN if the repository is private",
			httpErr.URL)
	}
	return stateNotReachable, fmt.Sprintf("%s answered %d: %s", httpErr.URL, httpErr.Status, httpErr.Body)
}
