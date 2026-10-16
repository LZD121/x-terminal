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
//
// Five minutes, because this run makes up to three requests in sequence — the Release, the
// digest, the asset — and each may spend up to internal/release's own handshake budget before
// it transfers anything. Measured 2026-10-16 through this proxy, GitHub handshakes run several
// to tens of seconds; a two-minute cap over three of them is a cap that decides the verb failed
// while a working network is still answering. The wire's own budgets are the real limits; this
// is the outer bound that keeps a wedged run from hanging forever.
const updateTimeout = 5 * time.Minute

// digestLimit bounds the per-asset digest file this verb reads. A sha256sum line is
// about eighty bytes, so this is already generous, and it is bounded because the body
// arrives from the network: the file is a line, and reading it as anything larger
// would be reading something that is not one.
const digestLimit = 4 << 10

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

// updateExecutable is the file this process was started from — the only path this
// verb may touch. It is os.Executable rather than anything the operator believes is
// installed (the two can differ, and the difference is exactly what a bug report gets
// wrong), and it is a seam because a test has to point it at a file in a temporary
// directory rather than replace the running test binary.
var updateExecutable = os.Executable

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
	Fetch(ctx context.Context, url string, limit int64) ([]byte, error)
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
	asset, err := release.AssetFor(rel.Assets, rel.Tag, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNoAsset, err.Error()))
		return ExitError
	}
	if check {
		// --check does not touch the machine, so its one fact is printed before either
		// verdict: which asset this platform's Release carries, which is what the
		// answer rests on. Presence facts only — a tag, a platform, a name.
		fmt.Fprintf(ctx.Stdout, "release %s, %s/%s asset %s\n", rel.Tag, runtime.GOOS, runtime.GOARCH, asset.Name)
	}

	running := Version()
	order := 0
	if running != "" {
		order, err = release.CompareTags(running, rel.Tag)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "update: %v\n", err)
			return ExitError
		}
		updateVerdict(ctx, running, rel.Tag, order)
	} else if !force {
		// A build with no version stamp has no ordering against the release, and
		// inventing one would be the verb guessing which release the operator is on.
		// --force is the operator saying they do not care about the ordering; it never
		// reaches past a check that verifies bytes.
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotARelease,
			"install a published asset, or build with the release workflow's -ldflags stamps"))
		return ExitError
	}

	if check {
		return ExitOK
	}
	if !force && running != "" && order >= 0 {
		// The comparison answered the question: "this build is the latest release" and a
		// downgrade refusal are answers rather than failures to answer, so both leave
		// nothing to install — and the machine is left exactly as it was.
		return ExitOK
	}
	return updateInstall(ctx, client, rel, asset, running, runCtx)
}

// updateInstall is the install path, written in the order it runs. Each step is
// there because it is cheaper than the one after it: the release was resolved and
// compared above (the free question — is there anything to install at all), the
// install directory is asked whether it accepts a temporary file next (the free
// local question, so a machine that cannot take the update is never made to fetch
// bytes), and only then are the digest and the payload downloaded.
//
// The download URL is never assembled by hand from the tag: it comes from the
// asset's own browser_download_url, and the digest beside it is resolved from the
// Release's asset list rather than by appending a suffix to a URL.
//
// Every arm that fails prints its state through stateLine and returns ExitError; a
// payload that cannot be verified is never installed, and the target file is left
// byte-identical on every one of them.
func updateInstall(ctx *Context, client releaseWire, rel release.Release, asset release.Asset, running string, runCtx context.Context) int {
	exe, err := updateExecutable()
	if err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotWritable, err.Error()))
		return ExitError
	}
	target, err := release.ResolveExecutable(exe)
	if err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotWritable, err.Error()))
		return ExitError
	}
	// Before a byte is downloaded: the rename needs write permission on the target's
	// directory, and a machine that cannot take the update should not be made to
	// fetch it.
	if err := release.CheckReplaceable(target); err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotWritable, err.Error()))
		return ExitError
	}

	fmt.Fprintf(ctx.Stdout, "replacing %s (this build %s) with %s\n", target, buildName(running), rel.Tag)

	// The digest is the per-asset file the Release itself names, and a Release that
	// publishes none for this platform has published nothing this verb can verify: an
	// unverifiable payload is not a payload that is probably fine.
	digestAsset, err := release.AssetNamed(rel.Assets, asset.Name+".sha256")
	if err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotVerified, err.Error()))
		return ExitError
	}
	digestBody, err := client.Fetch(runCtx, digestAsset.URL, digestLimit)
	if err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotVerified, err.Error()))
		return ExitError
	}
	want, err := release.ParseDigest(digestBody, asset.Name)
	if err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotVerified, err.Error()))
		return ExitError
	}

	// The asset is fetched with the size the Release published as its bound, and the
	// shape check below compares the same number: a body over the bound is refused
	// before it is read into memory.
	payload, err := client.Fetch(runCtx, asset.URL, asset.Size)
	if err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotVerified, err.Error()))
		return ExitError
	}
	if err := release.Verify(payload, want); err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotVerified, err.Error()))
		return ExitError
	}
	if err := release.LooksLikeThisBuild(payload, rel.Tag, runtime.GOOS, asset.Size); err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotVerified, err.Error()))
		return ExitError
	}
	fmt.Fprintf(ctx.Stdout, "verified against %s\n", digestAsset.Name)

	if err := release.Replace(target, payload); err != nil {
		fmt.Fprintln(ctx.Stderr, stateLine(stateNotWritable, err.Error()))
		return ExitError
	}
	fmt.Fprintf(ctx.Stdout, "replaced %s with %s\n", target, rel.Tag)
	// The process that ran this keeps the image it was started with, so the operator
	// is told when the change takes effect rather than being left to believe it is live.
	fmt.Fprintf(ctx.Stdout, "the process that ran this is still the old image: the next launch reports %s\n", rel.Tag)
	fmt.Fprintf(ctx.Stdout, "%s -> %s\n", buildName(running), rel.Tag)
	return ExitOK
}

// buildName is what the output calls the running build: its version when it has one,
// and "working tree" when it does not — the same fact the User-Agent carries, and
// always the truth about the process that ran this.
func buildName(running string) string {
	if running == "" {
		return "working tree"
	}
	return running
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
