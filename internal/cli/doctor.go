package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/opsgen"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/store"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// doctorTimeout bounds a whole run. Each stage has its own timeout — a dial at ten seconds, a
// request at thirty — and this is the backstop for a path that hangs where none of them
// covers. A self-check that never returns is one of the ways this tool could stop telling the
// truth about what it is doing.
const doctorTimeout = 90 * time.Second

// jsonLineLimit bounds one message inside the JSON report: the report is the thing a user
// pastes into a bug report, and a diagnostic line has no business being longer than a
// sentence.
const jsonLineLimit = 200

// doctorLabels are the first-column words the report may print. Presence-fact tests assert
// new lines stay inside this closed set.
var doctorLabels = []string{
	"config", "registry", "mirror", "signer", "session",
	"route", "proxy", "upstream", "requests", "probe",
}

// doctorRunCheck is the reachability check the verb spends its requests on. Tests replace
// it with a stub that returns fixed outcomes so the report builders can be exercised without
// dialling; production uses netcheck against the configured path.
var doctorRunCheck = runDoctorCheck

func runDoctorCheck(runCtx context.Context, cfg config.Config, sess session.Session) (results []netcheck.Result, budget func() (int, int, time.Time), hop string, client *xweb.Client) {
	hop = cfg.Redacted().Proxy
	budget = func() (int, int, time.Time) { return 0, 0, time.Now() }
	target, err := netcheck.NewTarget(cfg)
	if err != nil {
		return netcheck.Check(runCtx, cfg, sess), budget, hop, nil
	}
	results = netcheck.CheckWith(runCtx, target, sess)
	budget = func() (int, int, time.Time) { return target.Client().Limiter.Budget() }
	if line := target.Client().ProxyLine(); line != "direct" {
		hop = line
	} else {
		hop = ""
	}
	return results, budget, hop, target.Client()
}

// Doctor is the `doctor` verb: run the connectivity self-check, and print one line per link
// of the path, the session's own summary, and the request budget. Every line is a presence
// fact — a name, a length, a path, a status — and no line carries a credential.
//
// It exits non-zero when a link failed. A stage that was skipped for want of a session is not
// a failure: the path can be perfect while there is nothing to authenticate with, and the
// line says which of the two it is.
func Doctor(ctx *Context, args []string) int {
	var (
		asJSON     bool
		showNotice bool
		refreshIDs bool
		pin        string
		pinID      string
		probeOp    string
		hasProbe   bool
	)
	// A classic index loop rather than `range len(args)`: --pin and --probe consume the
	// arguments after them, so this loop moves the index itself.
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			asJSON = true
		case "--show-notice":
			showNotice = true
		case "--refresh-ids":
			refreshIDs = true
		case "--pin":
			if i+2 >= len(args) {
				fmt.Fprintln(ctx.Stderr, "doctor --pin needs an operation name and a query id")
				usage(ctx.Stderr)
				return ExitUsage
			}
			pin, pinID = args[i+1], args[i+2]
			i += 2
		case "--probe":
			if i+1 >= len(args) {
				fmt.Fprintln(ctx.Stderr, "doctor --probe needs an operation name")
				usage(ctx.Stderr)
				return ExitUsage
			}
			probeOp = args[i+1]
			hasProbe = true
			i++
		default:
			fmt.Fprintf(ctx.Stderr, "doctor: unknown argument %s\n", args[i])
			usage(ctx.Stderr)
			return ExitUsage
		}
	}

	if showNotice && asJSON {
		// The statement would be printed into the JSON object a caller is parsing. Saying so is
		// the only answer that is not a lie about either flag.
		fmt.Fprintln(ctx.Stderr, "doctor: --show-notice and --json cannot be combined: the statement would land inside the JSON report")
		usage(ctx.Stderr)
		return ExitUsage
	}

	if showNotice && (refreshIDs || pin != "" || hasProbe) {
		// Same reason as the --json combination above: the statement and the rest of the verb do
		// not fit in one invocation. Running the flag that was asked for beside it would spend
		// requests on a run whose output the user did not ask for, and dropping it silently would
		// be a lie about what ran.
		fmt.Fprintln(ctx.Stderr, "doctor: --show-notice prints the account-risk statement and exits, so it cannot be combined with --refresh-ids, --pin or --probe")
		usage(ctx.Stderr)
		return ExitUsage
	}

	if hasProbe && (refreshIDs || pin != "") {
		fmt.Fprintln(ctx.Stderr, "doctor: --probe spends exactly one request on one named operation and cannot be combined with --refresh-ids or --pin")
		usage(ctx.Stderr)
		return ExitUsage
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
		return ExitError
	}
	if showNotice {
		// Asked for by name, the statement is the whole run: no session is read, no request is
		// sent and no check runs. The check spends live requests against the account's budget,
		// and a user asking to read the statement has not asked to spend any — the notice is a
		// paragraph of prose, and it is printed here rather than after a check for exactly that
		// reason.
		ShowNotice(ctx)
		return ExitOK
	}
	switch {
	case pin != "":
		// A pin is not a check: it writes one line into the user's own pins file and exits,
		// without spending a request or reading the network at all.
		if err := volatile.Pin(cfg.PinsPath(), pin, pinID); err != nil {
			fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
			return ExitError
		}
		fmt.Fprintf(ctx.Stdout, "pinned %s to %s in %s\n", pin, pinID, cfg.PinsPath())
		fmt.Fprintf(ctx.Stdout, "a rebuild is not needed: the pin is read from that file at every start\n")
		return ExitOK
	case refreshIDs:
		return refreshRegistry(ctx, cfg)
	case hasProbe:
		return runProbe(ctx, cfg, probeOp, asJSON)
	}

	// The session is read before the check runs, so that the session stage can be skipped
	// rather than aimed at nothing. A missing file is a state; a file that is there and
	// unusable — a symlink, a mode anyone on the machine can read — is a failure, and it gets
	// its own exit code.
	sess, sessionErr := session.Load(cfg.SessionPath)

	runCtx, cancel := context.WithTimeout(context.Background(), doctorTimeout)
	defer cancel()
	results, budget, hop, client := doctorRunCheck(runCtx, cfg, sess)

	ok := sessionErr == nil || errors.Is(sessionErr, session.ErrNoSession)
	for _, r := range results {
		if r.Outcome.Failed() {
			ok = false
		}
	}

	// The operation table, through the same path the client reads it through. Its own warning is a
	// pins file the user wrote and this run could not read: that has to be visible, because the
	// alternative is a pin that silently does nothing.
	registry, err := volatile.Load(cfg.PinsPath())
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "registry: %v\n", err)
	}
	for _, warning := range registry.Warnings() {
		fmt.Fprintf(ctx.Stderr, "registry warning: %s\n", warning)
	}

	mirror, mirrorErr := store.ReadFacts(cfg.SavedDBPath())
	if mirrorFails(mirror, mirrorErr) {
		ok = false
	}
	signerName := "none"
	if client != nil {
		signerName = client.SignerName()
	}
	signerBreak := signerBreakFromResults(results)
	if signerBreak {
		ok = false
	}

	if ok {
		// The first successful run of any verb is where the account-risk statement belongs: the
		// user has just watched the tool work, which is when it is worth reading.
		ShowNoticeOnce(ctx, cfg.NoticePath())
	}
	if asJSON {
		return reportJSON(ctx, cfg, sess, sessionErr, results, budget, hop, registryLine(registry), mirror, mirrorErr, signerName, signerBreak, ok)
	}
	return reportText(ctx, cfg, sess, sessionErr, results, budget, hop, registryLine(registry), mirror, mirrorErr, signerName, signerBreak, ok)
}

// runProbe is doctor --probe: one request for one named operation through the same client
// path the check uses. A repair is one request; this flag never sweeps.
func runProbe(ctx *Context, cfg config.Config, opName string, asJSON bool) int {
	vars, ok := probeVariables(opName)
	if !ok {
		fmt.Fprintf(ctx.Stderr, "doctor: --probe does not know %q; known operations: %s\n", opName, strings.Join(probeOperations(), ", "))
		usage(ctx.Stderr)
		return ExitUsage
	}
	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: --probe needs the session cookie: %v\n", err)
		return ExitError
	}
	client, err := newAPIClient(cfg, sess)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
		return ExitError
	}
	runCtx, cancel := context.WithTimeout(context.Background(), doctorTimeout)
	defer cancel()
	probe, err := client.Probe(runCtx, opName, vars)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
		return ExitError
	}
	remaining := "unknown"
	if probe.RemainingKnown {
		remaining = fmt.Sprintf("%d", probe.Remaining)
	}
	reset := "unknown"
	if !probe.Reset.IsZero() {
		reset = probe.Reset.UTC().Format(time.RFC3339)
	}
	success := probe.Outcome.Success()
	if asJSON {
		report := map[string]any{
			"probe": map[string]any{
				"operation":       opName,
				"outcome":         probe.Outcome.Error(),
				"remaining":       probe.Remaining,
				"remaining_known": probe.RemainingKnown,
				"reset":           reset,
				"bytes":           probe.Bytes,
				"detail":          probe.Detail,
			},
			"ok": success,
		}
		if !probe.RemainingKnown {
			report["probe"].(map[string]any)["remaining"] = nil
		}
		encoded, encErr := json.MarshalIndent(report, "", "  ")
		if encErr != nil {
			fmt.Fprintf(ctx.Stderr, "doctor: %v\n", encErr)
			return ExitError
		}
		fmt.Fprintf(ctx.Stdout, "%s\n", encoded)
	} else {
		line := fmt.Sprintf("%-10s %s  %s  remaining %s  reset %s",
			"probe", opName, probe.Outcome.Error(), remaining, reset)
		// The upstream's own sentence goes on the same line, when it said anything: it is
		// what names the variable a rejected request shape is missing, which is the whole
		// reason the probe reports it. A successful probe is unchanged byte for byte.
		if probe.Detail != "" {
			line += "  " + probe.Detail
		}
		fmt.Fprintf(ctx.Stdout, "%s\n", line)
	}
	if !success {
		return ExitError
	}
	return ExitOK
}

// probeRequest is one row of the probe's request-shape table: an operation this build reads, the
// variables the probe sends for it, and the measurement the row rests on.
type probeRequest struct {
	Operation string
	Variables map[string]any
	// Note says what a reader needs to know about this row's values: which of them are probe
	// placeholders rather than values a real read sends, and which variable the upstream was
	// measured asking for.
	Note string
}

// probeRequests is the closed table of request shapes --probe can build: every operation this
// build reads, with the variables a measurement supports. An operation outside it is a usage
// error, and the list of known operations in that message is derived from this table rather than
// typed out beside it, so the two cannot disagree.
//
// The rule every row keeps is the one the read path keeps: the variables are the measured
// request's, and nothing else is sent. A key here is either one the request already carried
// under a measured 200, or one the upstream's own 422 named — an invented key is a claim about
// the upstream's contract that nothing measured. Where a value is a placeholder for the read,
// the row's Note says so; the probe's answer is discarded either way, so the shape is what it
// tests.
func probeRequests() []probeRequest {
	return []probeRequest{
		{
			Operation: "Viewer",
			Variables: map[string]any{},
			Note:      "the session's own account: no variables",
		},
		{
			Operation: "HomeLatestTimeline",
			Variables: map[string]any{
				"count":                  20,
				"cursor":                 "",
				"seenTweetIds":           []string{},
				"includePromotedContent": false,
			},
			Note: "includePromotedContent was measured required on 2026-10-15 (422 naming it under path:[\"variable\",…]); both true and false answered 200, and false is what the read path sends",
		},
		{
			Operation: "HomeTimeline",
			Variables: map[string]any{
				"count":                  20,
				"cursor":                 "",
				"seenTweetIds":           []string{},
				"includePromotedContent": false,
			},
			Note: "the For-you feed: same measured shape as HomeLatestTimeline",
		},
		{
			Operation: "Bookmarks",
			Variables: map[string]any{
				"count":  20,
				"cursor": "",
			},
			Note: "unchanged: measured answering 200 on 2026-10-15, the control row",
		},
		{
			Operation: "Likes",
			Variables: map[string]any{
				"count":                  20,
				"cursor":                 "",
				"userId":                 "0",
				"includePromotedContent": false,
			},
			Note: "userId \"0\" is a probe placeholder — the probe does not carry the session account's id",
		},
		{
			Operation: "UserTweets",
			Variables: map[string]any{
				"userId":                 "0",
				"count":                  20,
				"cursor":                 "",
				"includePromotedContent": false,
				"withVoice":              false,
			},
			Note: "userId \"0\" is a probe placeholder; includePromotedContent and withVoice are the names the upstream's own 422 asked for on 2026-10-15, and the values are the measured-required ones rather than values the upstream was shown to demand",
		},
		{
			Operation: "TweetDetail",
			Variables: map[string]any{
				// The focal id the recorded body in
				// internal/xsource/testdata/fixture-tweetdetail.json is about: the id its
				// provenance row carries and TestTheAdapterAsksForOnePage reads through
				// fixtureFocalID. A real id on purpose — a made-up placeholder can be refused
				// for a reason other than the variable being measured, which would make the
				// probe's name for a missing variable worthless.
				"focalTweetId":           "2106019976381689871",
				"cursor":                 "",
				"includePromotedContent": false,
				"withBirdwatchNotes":     false,
				"withVoice":              false,
			},
			Note: "the focal id is the recorded fixture's own; the three names after cursor are the ones this operation's own 422 asked for, in the order it asked, and the request answered 200 once all three were present (measured 2026-10-15)",
		},
		{
			Operation: "SearchTimeline",
			Variables: map[string]any{
				"rawQuery": "sea otter",
				"count":    20,
				"product":  "Latest",
				"cursor":   "",
			},
			Note: "rawQuery is a probe placeholder query; this operation answers only over POST (measured 2026-10-15), and the method is the request builder's decision rather than this table's",
		},
	}
}

// probeVariables resolves one operation's request shape through the table above.
func probeVariables(op string) (map[string]any, bool) {
	for _, row := range probeRequests() {
		if row.Operation == op {
			return row.Variables, true
		}
	}
	return nil, false
}

// probeOperations lists the table's operations in the table's own order, for the usage error
// that names them.
func probeOperations() []string {
	rows := probeRequests()
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Operation)
	}
	return names
}

// registryLine describes the operation table by presence: how many operations it holds, when the
// snapshot was mined, how many entries are the user's own pins, and whether it has been refreshed
// since. It never prints an id: this line travels into bug reports, and an id in one would be a
// stale id waiting to be copied into code.
func registryLine(registry *volatile.Registry) string {
	if registry == nil {
		return "none: the operation table could not be loaded"
	}
	provenance := registry.Provenance()
	line := fmt.Sprintf("%d operations, mined %s", registry.Len(), provenance.MinedAt)
	pinned := 0
	for _, name := range registry.Names() {
		if op, err := registry.Lookup(name); err == nil && op.Source == volatile.SourcePin {
			pinned++
		}
	}
	if pinned > 0 {
		line += fmt.Sprintf(", %d pinned", pinned)
	}
	if provenance.RefreshedAt != "" {
		line += fmt.Sprintf(", refreshed %s", provenance.RefreshedAt)
	}
	return line
}

// refreshRegistry is `doctor --refresh-ids`: what cmd/x-opsgen --live --merge does, behind the flag
// the doctor already advertises. It reads the authenticated shell and the bundle it references
// through the configured transport, merges the operations that bundle declares into the registry
// file, and prints the diff. It adds and updates only, it never deletes, and a refresh that
// changed nothing does not touch the file.
//
// It needs the session cookie, because the operation map only ships to an authenticated client,
// and the two requests it spends are counted in no budget line: this flag is a deliberate,
// occasional act, not part of the check.
func refreshRegistry(ctx *Context, cfg config.Config) int {
	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: refreshing the operation ids needs the session cookie: %v\n", err)
		return ExitError
	}
	client, err := xweb.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
		return ExitError
	}

	registry, err := volatile.ReadFile(opsgen.RegistryPath)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
		fmt.Fprintf(ctx.Stderr, "doctor: --refresh-ids rewrites the registry the binary embeds, so it has to be run from the repository root (or the file moved to %s)\n", opsgen.RegistryPath)
		return ExitError
	}

	runCtx, cancel := context.WithTimeout(context.Background(), doctorTimeout)
	defer cancel()
	refresh, err := opsgen.Fetch(runCtx, client, sess, opsgen.ShellURL)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
		return ExitError
	}

	result := registry.Merge(refresh.Operations())
	fmt.Fprintf(ctx.Stdout, "bundle: %s (%d bytes, sha256 %s)\n", refresh.BundleURL, refresh.Bytes, refresh.SHA256)
	fmt.Fprintf(ctx.Stdout, "registry: %s (%d operations in %s)\n", result, registry.Len(), opsgen.RegistryPath)
	if !result.Changed() {
		fmt.Fprintf(ctx.Stdout, "the snapshot is current: nothing written\n")
		return ExitOK
	}
	registry.RecordRefresh(refresh.At, refresh.BundleURL)
	if err := registry.Write(opsgen.RegistryPath); err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
		return ExitError
	}
	fmt.Fprintf(ctx.Stdout, "wrote %s — rebuild for the refreshed ids to take effect\n", opsgen.RegistryPath)
	return ExitOK
}

// reportText is the human report: the configuration the run used, the session it found, the
// operation table it reads through, one line per check, and the line from the request budget.
func reportText(ctx *Context, cfg config.Config, sess session.Session, sessionErr error, results []netcheck.Result, budget func() (int, int, time.Time), hop, registry string, mirror domain.MirrorFacts, mirrorErr error, signerName string, signerBreak, ok bool) int {
	fmt.Fprintf(ctx.Stdout, "%-10s %s\n", "config", configLine(cfg, hop))
	for _, warning := range cfg.Warnings {
		fmt.Fprintf(ctx.Stderr, "config warning: %s\n", warning)
	}
	fmt.Fprintf(ctx.Stdout, "%-10s %s\n", "registry", registry)
	fmt.Fprintf(ctx.Stdout, "%-10s %s\n", "mirror", mirrorLine(mirror, mirrorErr))
	fmt.Fprintf(ctx.Stdout, "%-10s %s\n", "signer", signerLine(signerName, signerBreak))
	fmt.Fprintf(ctx.Stdout, "%-10s %s\n", "session", sess.Summary())
	for _, warning := range sess.Warnings {
		fmt.Fprintf(ctx.Stderr, "session warning: %s\n", warning)
	}
	if sessionErr != nil {
		// The message names the path and, for a missing session, what to put in it. It carries no
		// cookie: session.Load never puts one in an error.
		fmt.Fprintf(ctx.Stderr, "session: %v\n", sessionErr)
	}

	for _, r := range results {
		detail := r.Detail
		if r.Outcome.Failed() {
			if ptr := RunbookAnchor(checkState(r)); ptr != "" {
				detail = appendRunbook(detail, ptr)
			}
		}
		fmt.Fprintf(ctx.Stdout, "%-10s %-36s %8s  %s\n",
			stageName(r.Outcome), r.Outcome, r.Elapsed.Round(time.Millisecond), detail)
	}

	used, limited, _ := budget()
	fmt.Fprintf(ctx.Stdout, "requests: %d used, %d limited, cap %d\n", used, limited, cfg.SessionRequestCap)

	if !ok {
		return ExitError
	}
	return ExitOK
}

// The JSON report carries the same data as the text one. The strings are the paths and
// settings the run used, the session summary, and one bounded sentence per check; nothing in
// it is a credential, and nothing in it is unbounded.
type doctorReport struct {
	Config   doctorConfig  `json:"config"`
	Registry string        `json:"registry"`
	Mirror   doctorMirror  `json:"mirror"`
	Signer   doctorSigner  `json:"signer"`
	Session  doctorSession `json:"session"`
	Checks   []doctorCheck `json:"checks"`
	Budget   doctorBudget  `json:"budget"`
	OK       bool          `json:"ok"`
}

type doctorConfig struct {
	Path     string   `json:"path"`
	Proxy    string   `json:"proxy"` // empty means a direct dial
	Media    string   `json:"media"`
	Warnings []string `json:"warnings,omitempty"`
}

type doctorMirror struct {
	Path          string             `json:"path"`
	Exists        bool               `json:"exists"`
	SchemaVersion int                `json:"schema_version"`
	Newer         bool               `json:"newer"`
	Owner         string             `json:"owner"`
	Kinds         []doctorMirrorKind `json:"kinds"`
	LastRun       string             `json:"last_run,omitempty"`
	Stop          string             `json:"stop,omitempty"`
	Error         string             `json:"error,omitempty"`
	Runbook       string             `json:"runbook,omitempty"`
}

type doctorMirrorKind struct {
	Kind    string `json:"kind"`
	Present int    `json:"present"`
	Absent  int    `json:"absent"`
}

type doctorSigner struct {
	Name    string `json:"name"`
	Runbook string `json:"runbook,omitempty"`
}

type doctorSession struct {
	Summary  string   `json:"summary"`
	Path     string   `json:"path"`
	Warnings []string `json:"warnings,omitempty"`
	Error    string   `json:"error,omitempty"`
}

type doctorCheck struct {
	Stage     string `json:"stage"`
	Outcome   string `json:"outcome"`
	Detail    string `json:"detail"`
	ElapsedMS int64  `json:"elapsed_ms"`
	Runbook   string `json:"runbook,omitempty"`
}

type doctorBudget struct {
	Used    int    `json:"used"`
	Limited int    `json:"limited"`
	Cap     int    `json:"cap"`
	ResetAt string `json:"reset_at"`
}

// reportJSON prints the report as one JSON object on stdout, and nothing else on stdout: a
// caller that pipes this into a parser must not find a warning line spliced into it.
func reportJSON(ctx *Context, cfg config.Config, sess session.Session, sessionErr error, results []netcheck.Result, budget func() (int, int, time.Time), hop, registry string, mirror domain.MirrorFacts, mirrorErr error, signerName string, signerBreak, ok bool) int {
	used, limited, reset := budget()
	report := doctorReport{
		Config: doctorConfig{
			Path:     filepath.Join(cfg.ConfigDir, config.ConfigFileName),
			Proxy:    hop,
			Media:    mediaFacts(cfg),
			Warnings: clippedAll(cfg.Warnings),
		},
		Registry: clipped(registry),
		Mirror:   mirrorJSON(mirror, mirrorErr),
		Signer: doctorSigner{
			Name: signerName,
		},
		Session: doctorSession{
			Summary:  sess.Summary(),
			Path:     cfg.SessionPath,
			Warnings: clippedAll(sess.Warnings),
		},
		Checks: make([]doctorCheck, 0, len(results)),
		Budget: doctorBudget{Used: used, Limited: limited, Cap: cfg.SessionRequestCap, ResetAt: reset.UTC().Format(time.RFC3339Nano)},
		OK:     ok,
	}
	if signerBreak {
		report.Signer.Runbook = RunbookAnchor(string(xweb.OutcomeSignerOrAuth))
	}
	if sessionErr != nil {
		report.Session.Error = clipped(sessionErr.Error())
	}
	for _, r := range results {
		check := doctorCheck{
			Stage:     stageName(r.Outcome),
			Outcome:   string(r.Outcome),
			Detail:    clipped(r.Detail),
			ElapsedMS: r.Elapsed.Milliseconds(),
		}
		if r.Outcome.Failed() {
			check.Runbook = RunbookAnchor(checkState(r))
		}
		report.Checks = append(report.Checks, check)
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "doctor: %v\n", err)
		return ExitError
	}
	fmt.Fprintf(ctx.Stdout, "%s\n", encoded)

	if !ok {
		return ExitError
	}
	return ExitOK
}

// stageName is the link of the path one outcome belongs to: the report's first column, and
// the answer to "which part is broken".
func stageName(outcome netcheck.Outcome) string {
	switch outcome {
	case netcheck.OutcomeConfigFailed:
		return "config"
	case netcheck.OutcomeRouteOK, netcheck.OutcomeNoRoute:
		return "route"
	case netcheck.OutcomeProxyOK, netcheck.OutcomeProxyRefused:
		return "proxy"
	case netcheck.OutcomeUpstreamOK, netcheck.OutcomeUpstreamBlocked, netcheck.OutcomeChallenge:
		return "upstream"
	default:
		return "session"
	}
}

// configLine names the config file this run resolved and the first hop its requests took. The
// hop is the redacted form — a proxy URL may carry a password — and it comes from the client
// that will make the requests, not from the string that was configured.
func configLine(cfg config.Config, hop string) string {
	line := filepath.Join(cfg.ConfigDir, config.ConfigFileName)
	if hop != "" {
		line += "   proxy " + hop
	} else {
		line += "   no proxy (direct dial)"
	}
	return line + "   media " + mediaFacts(cfg) + "   translate " + translateFacts(cfg)
}

// mediaFacts reports the opener dispatch as presence facts: the configured program for
// images and links, and the fixed mpv name for video and gif. The video program name is
// hardcoded rather than imported from internal/media so this package cannot grow a cycle.
func mediaFacts(cfg config.Config) string {
	opener := cfg.MediaOpener
	if opener == "" {
		opener = config.DefaultMediaOpener
	}
	return fmt.Sprintf("%s (images, links), mpv (video, gif)", opener)
}

// translateFacts reports the translation setting as one more presence fact: the language the resolved
// configuration would ask x.com's translator for, which the `T` key reads off the same value.
//
// It is a fact and not a probe, for the reason the rest of this line is: naming whether api.x.com
// answers this endpoint would spend one of the account's requests — the one thing a doctor run must
// not do outside the reachability check the run already reports — and the endpoint is not in the
// operation registry, so there is no id here to pin either. The answer for this endpoint belongs to
// the key's own refusal (which lands on the status line with RUNBOOK.md's pointer), and the endpoint's
// exact shape is one gated test away: the runbook's request-shape section names it.
func translateFacts(cfg config.Config) string {
	language := cfg.TranslateLanguage
	if language == "" {
		language = config.DefaultTranslateLanguage
	}
	return language
}

func mirrorFails(facts domain.MirrorFacts, err error) bool {
	return err != nil || facts.Newer
}

func mirrorLine(facts domain.MirrorFacts, err error) string {
	if err != nil {
		path := facts.Path
		if path == "" {
			path = "saved.db"
		}
		return fmt.Sprintf("%s  unreadable: %s  see %s", path, clipped(err.Error()), RunbookAnchor("mirror unreadable"))
	}
	if !facts.Exists {
		return fmt.Sprintf("no mirror yet (%s)", facts.Path)
	}
	if facts.Newer {
		return fmt.Sprintf("%s  schema %d (newer than this build knows - see %s)",
			facts.Path, facts.SchemaVersion, RunbookAnchor("mirror newer"))
	}
	owner := facts.Owner
	if owner != "" && !strings.HasPrefix(owner, "@") {
		owner = "@" + owner
	}
	if owner == "" {
		owner = "@"
	}
	var bookmarks, likes domain.MirrorKindCount
	for _, k := range facts.Kinds {
		switch k.Kind {
		case domain.Bookmark:
			bookmarks = k
		case domain.Like:
			likes = k
		}
	}
	line := fmt.Sprintf("%s  schema %d  owner %s  bookmarks %d present %d absent  likes %d present %d absent",
		facts.Path, facts.SchemaVersion, owner,
		bookmarks.Present, bookmarks.Absent,
		likes.Present, likes.Absent)
	if !facts.LastRun.IsZero() {
		line += fmt.Sprintf("  last sync %s (%s)",
			facts.LastRun.UTC().Format("2006-01-02 15:04"), facts.Stop.String())
	}
	return line
}

func mirrorJSON(facts domain.MirrorFacts, err error) doctorMirror {
	out := doctorMirror{
		Path:          facts.Path,
		Exists:        facts.Exists,
		SchemaVersion: facts.SchemaVersion,
		Newer:         facts.Newer,
		Owner:         facts.Owner,
		Kinds:         make([]doctorMirrorKind, 0, len(facts.Kinds)),
	}
	for _, k := range facts.Kinds {
		out.Kinds = append(out.Kinds, doctorMirrorKind{
			Kind:    k.Kind.List(),
			Present: k.Present,
			Absent:  k.Absent,
		})
	}
	if !facts.LastRun.IsZero() {
		out.LastRun = facts.LastRun.UTC().Format(time.RFC3339)
		out.Stop = facts.Stop.String()
	}
	if err != nil {
		out.Error = clipped(err.Error())
		out.Runbook = RunbookAnchor("mirror unreadable")
	} else if facts.Newer {
		out.Runbook = RunbookAnchor("mirror newer")
	}
	return out
}

func signerLine(name string, signerBreak bool) string {
	if name == "" {
		name = "none"
	}
	line := name + " (no transaction-id header is sent; the read below is the measurement)"
	if name != "none" {
		line = name + " (the read below is the measurement)"
	}
	if signerBreak {
		line = appendRunbook(line, RunbookAnchor(string(xweb.OutcomeSignerOrAuth)))
	}
	return line
}

func signerBreakFromResults(results []netcheck.Result) bool {
	for _, r := range results {
		if strings.Contains(r.Detail, string(xweb.OutcomeSignerOrAuth)) {
			return true
		}
	}
	return false
}

func appendRunbook(detail, ptr string) string {
	if ptr == "" {
		return detail
	}
	if detail == "" {
		return "see " + ptr
	}
	if strings.Contains(detail, ptr) {
		return detail
	}
	return detail + "  see " + ptr
}

// checkState picks the runbook table key for one check result: path failures use the
// netcheck outcome word, API-rejected lines prefer a classifier word carried in Detail.
func checkState(r netcheck.Result) string {
	switch r.Outcome {
	case netcheck.OutcomeAPIRejected:
		for _, o := range failingXwebOutcomes() {
			if strings.Contains(r.Detail, string(o)) {
				return string(o)
			}
		}
		return string(xweb.OutcomeUnclassified)
	case netcheck.OutcomeSessionRejected:
		for _, o := range []xweb.Outcome{xweb.OutcomeAuthRejected, xweb.OutcomeSignerOrAuth} {
			if strings.Contains(r.Detail, string(o)) {
				return string(o)
			}
		}
		return string(netcheck.OutcomeSessionRejected)
	default:
		return string(r.Outcome)
	}
}

// runbookAnchors maps every failing state the report can print onto a RUNBOOK.md section.
// Missing mirror is present with an empty pointer: it is not a failure. Plan 03 reads this
// table through RunbookAnchor / RunbookAnchors.
var runbookAnchors = map[string]string{
	string(netcheck.OutcomeConfigFailed):    "RUNBOOK.md#no-path-to-x.com",
	string(netcheck.OutcomeNoRoute):         "RUNBOOK.md#no-path-to-x.com",
	string(netcheck.OutcomeProxyRefused):    "RUNBOOK.md#no-path-to-x.com",
	string(netcheck.OutcomeUpstreamBlocked): "RUNBOOK.md#no-path-to-x.com",
	string(netcheck.OutcomeChallenge):       "RUNBOOK.md#no-path-to-x.com",
	string(netcheck.OutcomeSessionRejected): "RUNBOOK.md#session-expiry",
	string(netcheck.OutcomeAPIRejected):     "RUNBOOK.md#unclassified",

	string(xweb.OutcomeAuthRejected):     "RUNBOOK.md#session-expiry",
	string(xweb.OutcomeSignerOrAuth):     "RUNBOOK.md#signer-break",
	string(xweb.OutcomeUnknownOperation): "RUNBOOK.md#operation-id-rotation",
	string(xweb.OutcomeEndpointRetired):  "RUNBOOK.md#endpoint-retired",
	string(xweb.OutcomeRateLimited):      "RUNBOOK.md#429",
	string(xweb.OutcomeSoftBan):          "RUNBOOK.md#soft-ban",
	string(xweb.OutcomeLoadShed):         "RUNBOOK.md#load-shed",
	string(xweb.OutcomeBadRequest):       "RUNBOOK.md#request-shape-drift",
	string(xweb.OutcomeEmptyBody):        "RUNBOOK.md#empty-response-body",
	string(xweb.OutcomeChallenge):        "RUNBOOK.md#no-path-to-x.com",
	string(xweb.OutcomeUnauthorized):     "RUNBOOK.md#unclassified",
	string(xweb.OutcomeUnclassified):     "RUNBOOK.md#unclassified",

	"mirror newer":      "RUNBOOK.md#the-mirror",
	"mirror unreadable": "RUNBOOK.md#the-mirror",
	"mirror missing":    "",
}

// RunbookAnchor returns the RUNBOOK.md#anchor pointer for a failing state, or "" when the
// state is not a failure (notably the missing-mirror presence fact).
func RunbookAnchor(state string) string {
	return runbookAnchors[state]
}

// RunbookAnchors returns a copy of the doctor's failing-state → runbook pointer table so
// plan 03 can assert every anchor exists in RUNBOOK.md without re-deriving the set.
func RunbookAnchors() map[string]string {
	out := make(map[string]string, len(runbookAnchors))
	for k, v := range runbookAnchors {
		out[k] = v
	}
	return out
}

func failingXwebOutcomes() []xweb.Outcome {
	return []xweb.Outcome{
		xweb.OutcomeAuthRejected,
		xweb.OutcomeSignerOrAuth,
		xweb.OutcomeUnknownOperation,
		xweb.OutcomeEndpointRetired,
		xweb.OutcomeRateLimited,
		xweb.OutcomeSoftBan,
		xweb.OutcomeLoadShed,
		xweb.OutcomeBadRequest,
		xweb.OutcomeEmptyBody,
		xweb.OutcomeChallenge,
		xweb.OutcomeUnauthorized,
		xweb.OutcomeUnclassified,
	}
}

func failingNetcheckOutcomes() []netcheck.Outcome {
	return []netcheck.Outcome{
		netcheck.OutcomeConfigFailed,
		netcheck.OutcomeNoRoute,
		netcheck.OutcomeProxyRefused,
		netcheck.OutcomeUpstreamBlocked,
		netcheck.OutcomeChallenge,
		netcheck.OutcomeSessionRejected,
		netcheck.OutcomeAPIRejected,
	}
}

// clipped shortens one message to jsonLineLimit bytes on a rune boundary, and says that it
// cut something.
func clipped(line string) string {
	if len(line) <= jsonLineLimit {
		return line
	}
	for cut := jsonLineLimit; cut > 0; cut-- {
		if utf8.RuneStart(line[cut]) {
			return line[:cut] + "..."
		}
	}
	return line[:jsonLineLimit]
}

// clippedAll is clipped over a list of messages.
func clippedAll(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, clipped(line))
	}
	return out
}
