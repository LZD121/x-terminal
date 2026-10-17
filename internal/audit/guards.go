package audit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"strings"
)

// The only permitted wait is the limiter's own inter-request floor. The
// exemption is named here so it is visible and counted, not silent.
const (
	limiterFloorFile = "internal/xweb/limiter.go"
	limiterFloorFunc = "sleepContext"
)

// timerExemption is one declared place a timer may live. Each entry names the file, the function
// and why the wait is not the thing the promise is about — a request floor or a local sign-in wait,
// never a poll of the upstream. Every entry is required to be reached: an exemption nothing uses is
// a hole nobody is watching.
type timerExemption struct {
	File string
	Func string
	Note string
}

var timerExemptions = []timerExemption{
	{limiterFloorFile, limiterFloorFunc, "the request floor"},
	// login --browser waits for a human to sign in and for a local browser to hand its cookies
	// over. Nothing here is requested from x.com, and the wait ends when the verb does.
	{"internal/browserlogin/browser.go", "Login", "the sign-in deadline"},
	{"internal/browserlogin/browser.go", "poll", "the sign-in polling interval"},
}

// timerExempt is the declared exemption covering one call, or nil.
func timerExempt(filePath string, file *ast.File, pos token.Pos) *timerExemption {
	for i := range timerExemptions {
		e := &timerExemptions[i]
		if filePath == e.File && inFunc(file, pos, e.Func) {
			return e
		}
	}
	return nil
}

var bannedTimerSels = map[string]bool{
	"NewTicker": true,
	"NewTimer":  true,
	"AfterFunc": true,
	"Tick":      true,
	"Sleep":     true,
}

var writeCallNames = map[string]bool{
	"Create":     true,
	"OpenFile":   true,
	"WriteFile":  true,
	"CreateTemp": true,
}

// distributionNameRes and distributionPathRes name the artifacts this project promises not to ship.
// That list changed twice, each time deliberately, when the repository's distribution changed.
//
// The promise as first written was "no distribution artifact and no hosted mode" and it banned a
// release workflow outright. That was true of a tree nobody else could build, and false the moment
// the project was pushed to a private repository with a release job: the check would have gone on
// printing a promise the repository no longer kept, which is the failure this package exists to
// catch. It was narrowed rather than deleted, for the reason the project always had: internal-endpoint
// usage is against X's terms, so what must not exist is a *distributable* form of this program — a
// container image, a hosted instance, an installer, a package recipe, some artifact a stranger can
// pick up and run. A release of source plus a Go binary, out of a *private* repository, for the
// operator's own machines, was not that: it was the build this project already was, with a place to
// put it.
//
// On 2026-10-16 the repository was made **public**, and that reasoning's premise is gone: the
// Release's source and its three platform binaries are now downloadable by anyone, with no
// credential, so a stranger can in fact pick this program up and run it. The fact is recorded here,
// in the artifact that would otherwise go on printing the private-repository version of the promise,
// and `PROJECT.md`'s Out of Scope entry and `README.md`'s "What this is not" say the same thing in
// the same words. What this check *enforces* is deliberately unchanged — the author accepted the
// release and the exposure it carries; what is not accepted is a second, easier distribution path
// appearing without that acceptance being repeated.
//
// So the name arm keeps the images, the package recipes, the installers and the release scripts, and
// the path arm keeps the installer/release script directories. The hosted-mode arm below (listeners,
// servers, bind addresses) is untouched and is still the strongest guarantee in the set. A workflow
// named `release.yml` does not trip this check, and that is the promise told the truth rather than
// the check weakened: what it would have caught — a published image, a package, an installer — is
// what it still catches, and the plant tests below prove it can still fail.
var distributionNameRes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^Dockerfile(\..+)?$`),
	regexp.MustCompile(`(?i)^Containerfile(\..+)?$`),
	regexp.MustCompile(`(?i)^PKGBUILD$`),
	regexp.MustCompile(`(?i)^\.goreleaser(\..+)?$`),
	regexp.MustCompile(`(?i)^.*\.goreleaser\.(yml|yaml)$`),
	// The installer family: `install`, `install.sh`, and the variants someone reaches for when the
	// first one is taken (`install-local.sh`, `install_latest`). The one exact path allowed is
	// distributionAllowed's; everything else in the family is a finding.
	regexp.MustCompile(`(?i)^install([-_a-z0-9]+)?(\.sh)?$`),
	regexp.MustCompile(`(?i)^release(\.sh)?$`),
}

var distributionPathRes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^scripts/install`),
	regexp.MustCompile(`(?i)^scripts/release`),
	regexp.MustCompile(`(?i)^\.github/workflows/.*publish-image`),
	regexp.MustCompile(`(?i)^\.github/workflows/.*container`),
	regexp.MustCompile(`(?i)^\.github/workflows/.*package`),
}

// distributionAllowed names the one artifact the name rule below permits, by exact path. The promise
// it belongs to is about a *distributable* form — an image, a package, a hosted instance — and this
// is none of those: it downloads the release binary, verifies the digest the release publishes, and
// installs it under the operator's home. It is named here, printed in the promise, and required to be
// present, so it cannot become a silent hole for the next installer-shaped file.
var distributionAllowed = map[string]string{
	"install.sh": "the install helper: it fetches this repository's own release binary, verifies the published digest and writes it under $HOME; no package, no image, no hosted service",
}

var hostedConstructs = map[string]bool{
	"Listen":            true,
	"ListenPacket":      true,
	"ListenUnix":        true,
	"ListenAndServe":    true,
	"ListenAndServeTLS": true,
	"Serve":             true,
	"ServeTLS":          true,
}

var accountFlagLits = []string{
	"--" + "account",
	"--" + "user",
}

var accountConfigKeys = []string{
	"account_path",
	"accounts",
}

var volatileVocab = []string{
	"query" + "Id",
}

var tuiForbidden = []string{
	"net/http",
	"os/exec",
	"database/sql",
	"github.com/LZD121/x-terminal/internal/store",
	"github.com/LZD121/x-terminal/internal/xsource",
	"github.com/LZD121/x-terminal/internal/xweb",
	"github.com/LZD121/x-terminal/internal/session",
	"github.com/LZD121/x-terminal/internal/cli",
	"github.com/LZD121/x-terminal/internal/fake",
}

func checkTimers(t Tree) Report {
	const name = "timers"
	sources := parseGoSources(t.GoFiles(), 0)
	r := Report{Check: name, Promise: "no timer or sleep outside the limiter's floor", Files: len(sources)}
	if len(sources) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no Go files: a scan of nothing proves nothing"))
		return r
	}
	reached := map[string]int{}
	for _, s := range sources {
		ast.Inspect(s.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			xid, ok := sel.X.(*ast.Ident)
			if !ok || xid.Name != "time" {
				return true
			}
			if !bannedTimerSels[sel.Sel.Name] {
				return true
			}
			pos := s.FSet.Position(call.Pos())
			if exempt := timerExempt(s.Path, s.AST, call.Pos()); exempt != nil {
				key := exempt.File + "." + exempt.Func
				if _, seen := reached[key]; !seen {
					reached[key] = pos.Line
				}
				return true
			}
			r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("banned timer call time.%s", sel.Sel.Name), pos.Line))
			return true
		})
	}
	// Every declared exemption must be reached when its file is in the tree under test, and each
	// is printed so it is counted rather than silent. A tree that does not hold the file at all is
	// a partial scan (a fixture), not a stale declaration.
	present := map[string]bool{}
	for _, src := range sources {
		present[src.Path] = true
	}
	for _, e := range timerExemptions {
		key := e.File + "." + e.Func
		line, ok := reached[key]
		if !ok {
			// The limiter's floor is required of every tree: a tree with no limiter is a tree this
			// check proved nothing about. The others are required only when their file is here, so
			// a fixture that deliberately holds one package does not fail the check.
			if e.File == limiterFloorFile || present[e.File] {
				r.Findings = append(r.Findings, emptyFinding(name, fmt.Sprintf("declared timer exemption %s absent from %s: a file that no longer needs it is a declaration to remove", key, e.File)))
			}
			continue
		}
		r.Promise = fmt.Sprintf("%s (exemption %s:%d %s: %s)", r.Promise, e.File, line, e.Func, e.Note)
	}
	return r
}

func inFunc(file *ast.File, pos token.Pos, name string) bool {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != name || fn.Body == nil {
			continue
		}
		if pos >= fn.Body.Pos() && pos <= fn.Body.End() {
			return true
		}
	}
	return false
}

type destHit struct {
	Dest string
	Path string
	Line int
	Call string
}

// configWriterFile is the one file the running tool writes its own settings
// through; registryWriterFile is the file whose write call sites are the
// checked-in operation-id snapshot; spikeRecorderFile/spikeRecorderFunc name the
// one function in the development spike that records a fetched body; and
// updaterWriterFile is the file that installs the running tool's own binary.
// Each is named in writeDestinations below: this check refuses a skip that is
// not printed, and it anchors these four to the file (and, for the spike, to
// the function) that writes them, so a new write beside them — another file of
// the same package included — does not inherit them.
const (
	configWriterFile   = "internal/config/config.go"
	registryWriterFile = "internal/xweb/volatile/registry.go"
	spikeRecorderFile  = "cmd/spike/main.go"
	spikeRecorderFunc  = "recorder"
	updaterWriterFile  = "internal/release/install.go"
	// sessionWriterFile is the one place a credential is written. It is anchored to the file for
	// the same reason the others are: a write beside it must not inherit the destination.
	sessionWriterFile = "internal/session/save.go"
	// browserProfileDir is the package that creates a throwaway browser profile for `login
	// --browser`: a directory the browser then writes into, and this program deletes on the way
	// out. Anchored to the package, because every file of it is about that one directory.
	browserProfileDir = "internal/browserlogin/"
)

// Destination kinds: the running tool's own state, and the repository's
// development tools. The promise names both groups, so the printed line says
// which writers are the tool's and which are the checkout's.
const (
	runningTool     = "the running tool"
	developmentTool = "a development tool"
)

// writeDestination is one place this tree is allowed to write. Every entry is
// declared here, printed in the check's line with the call site that reached
// it, and required to be reached: a declared destination nothing writes to is a
// finding, and a write that matches no entry is a finding with its file:line.
type writeDestination struct {
	Name string
	Kind string
	Note string
}

var writeDestinations = []writeDestination{
	{"config", runningTool, "the config file the display keys live in: SetDisplay rewrites those two lines and nothing else"},
	{"pins", runningTool, "pins.json"},
	{"notice", runningTool, "the first-run marker"},
	{"saved", runningTool, "saved.db and its wal/shm"},
	{"update", runningTool, "the installed binary the update verb replaces: a temporary file in the target's own directory, renamed over it, removed on every failure path"},
	{"session", runningTool, "the credential file `login` writes: 0600 from creation, a temporary file in the same directory, fsynced, renamed over the target, removed on every failure path"},
	{"browser-profile", runningTool, "the throwaway browser profile `login --browser` creates for one sign-in: a 0700 directory, removed on every path out, never the user's own profile"},
	{"registry", developmentTool, "cmd/x-opsgen writes the checked-in operation-id snapshot through internal/xweb/volatile"},
	{"spike-recording", developmentTool, "cmd/spike --record writes fetched response bodies into the operator's own directory"},
}

// destinationSummary groups the declared names by kind for the printed line.
func destinationSummary() string {
	var groups []string
	for _, kind := range []string{runningTool, developmentTool} {
		var names []string
		for _, d := range writeDestinations {
			if d.Kind == kind {
				names = append(names, d.Name)
			}
		}
		if len(names) > 0 {
			groups = append(groups, fmt.Sprintf("%s (%s)", strings.Join(names, ", "), kind))
		}
	}
	return strings.Join(groups, "; ")
}

// persistencePromise is the line this check prints and Checks() carries. The
// count and the names come from the table, so the printed promise cannot drift
// from what the check accepts.
func persistencePromise() string {
	return fmt.Sprintf("every write resolves to one of the %d named destinations: %s", len(writeDestinations), destinationSummary())
}

func checkPersistence(t Tree) Report {
	const name = "persistence"
	sources := parseGoSources(t.GoFiles(), 0)
	r := Report{Check: name, Promise: persistencePromise(), Files: len(sources)}
	if len(sources) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no Go files: a scan of nothing proves nothing"))
		return r
	}

	reached := map[string]bool{}
	var hits, unknown []destHit

	// Every non-test Go file is scanned. There is no path skip: the two
	// development writers are declared destinations, not exemptions.
	for _, s := range sources {
		hints := destinationHints(s.AST)
		record := func(dest string, line int, call string) {
			h := destHit{Dest: dest, Path: s.Path, Line: line, Call: call}
			hits = append(hits, h)
			if dest == "other" {
				unknown = append(unknown, h)
				return
			}
			reached[dest] = true
		}
		ast.Inspect(s.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn, ok := callName(call)
			if !ok {
				return true
			}
			line := s.FSet.Position(call.Pos()).Line
			switch fn {
			case "os.Create", "os.OpenFile", "os.WriteFile", "os.CreateTemp", "os.MkdirTemp", "writeAtomically", "writeFile":
				record(classifyWrite(s.Path, s.AST, call.Pos(), hints), line, fn)
			case "sql.Open":
				dest := "saved"
				if !strings.Contains(s.Path, "internal/store/") {
					dest = "other"
				}
				record(dest, line, fn)
			}
			return true
		})
	}

	// Also credit destinations named by path helpers even when the write is
	// indirect: the tool's own files are reached through config accessors as
	// often as through a literal. The config destination is not credited here: its
	// declaration is a write call site in one file, and a destination the promise
	// names that only a constant reaches is a place the promise claims to have
	// audited and never did.
	for _, f := range t.GoFiles() {
		src := string(f.Src)
		switch {
		case strings.Contains(src, `"pins.json"`) || strings.Contains(src, "PinsPath"):
			if strings.Contains(f.Path, "internal/config/") || strings.Contains(f.Path, "internal/xweb/volatile/pins.go") {
				reached["pins"] = true
			}
		}
		if strings.Contains(src, `"notice-shown"`) || strings.Contains(src, "NoticePath") {
			if strings.Contains(f.Path, "internal/config/") || strings.Contains(f.Path, "internal/cli/notice.go") {
				reached["notice"] = true
			}
		}
		if strings.Contains(src, `"saved.db"`) || strings.Contains(src, "SavedDBPath") {
			if strings.Contains(f.Path, "internal/config/") || strings.Contains(f.Path, "internal/store/") {
				reached["saved"] = true
			}
		}
	}

	// The print: every write that resolved, with its file:line, followed by
	// every declared destination that nothing reached.
	var resolved []string
	for _, h := range hits {
		if h.Dest == "other" {
			continue
		}
		resolved = append(resolved, fmt.Sprintf("%s@%s:%d", h.Dest, h.Path, h.Line))
	}
	if len(resolved) > 0 {
		r.Promise = fmt.Sprintf("%s [%s]", r.Promise, strings.Join(unique(resolved), ", "))
	}
	for _, d := range writeDestinations {
		if !reached[d.Name] {
			r.Findings = append(r.Findings, emptyFinding(name, fmt.Sprintf("declared destination %q (%s) reached by no write call site: a destination nothing writes to proves nothing about a write that does", d.Name, d.Note)))
		}
	}
	for _, h := range unknown {
		r.Findings = append(r.Findings, fileFinding(name, h.Path, fmt.Sprintf("write-capable call %s resolves to no declared destination: a write path nobody declared is a persistence file nobody named", h.Call), h.Line))
	}
	// A scan that saw no write call at all has audited no write: the arm fires on
	// the count alone, so a tree that parses to zero writes is reported as vacuous
	// and not only as a set of unreached destinations. Both are true of such a
	// tree, and both are printed.
	if len(hits) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "found 0 write-capable call sites: anti-vacuity"))
	}
	return r
}

func destinationHints(file *ast.File) map[string]bool {
	hints := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		v := strings.Trim(lit.Value, `"`)
		switch {
		case v == "pins.json" || strings.Contains(v, "pins.json"):
			hints["pins"] = true
		case v == "notice-shown" || strings.Contains(v, "notice-shown"):
			hints["notice"] = true
		case v == "saved.db" || strings.Contains(v, "saved.db") || strings.Contains(v, "-wal") || strings.Contains(v, "-shm"):
			hints["saved"] = true
		}
		return true
	})
	return hints
}

// classifyWrite decides which declared destination one write call resolves to.
// Every destination that the running tool owns is anchored to the file that
// writes it (SetDisplay's two lines into the config file beside the session
// file, pins.json, the first-run marker, saved.db), and the two development
// destinations are anchored to theirs — and, for the spike, to the function.
// The session file is among them only through the one function that writes it
// (session.Save, reached by `login`): it is 0600 from creation and renamed into
// place. A write that names it anywhere else is a new destination and is reported. The
// hint fallback is confined to internal/: a write in cmd/ is a new destination
// even when it names a file the running tool writes.
func classifyWrite(filePath string, file *ast.File, pos token.Pos, hints map[string]bool) string {
	switch {
	case filePath == configWriterFile:
		// SetDisplay's two keys line by line, and the temporary file writeFile
		// renames over it.
		return "config"
	case filePath == registryWriterFile:
		// The snapshot and its temporary file in the same directory.
		return "registry"
	case filePath == spikeRecorderFile && inFunc(file, pos, spikeRecorderFunc):
		return "spike-recording"
	case filePath == updaterWriterFile:
		// The updater's temporary file in the target's own directory, renamed over
		// the target it replaces.
		return "update"
	case filePath == sessionWriterFile:
		// The credential's temporary file, renamed over the session path: the one
		// write of a cookie this tree performs.
		return "session"
	case strings.HasPrefix(filePath, browserProfileDir):
		// The throwaway profile the sign-in browser is given, and its removal.
		return "browser-profile"
	case strings.Contains(filePath, "internal/cli/notice.go"):
		return "notice"
	case strings.Contains(filePath, "internal/xweb/volatile/pins.go"):
		return "pins"
	case strings.Contains(filePath, "internal/store/"):
		return "saved"
	}
	if strings.HasPrefix(filePath, "internal/") {
		for _, d := range []string{"pins", "notice", "saved"} {
			if hints[d] {
				return d
			}
		}
	}
	return "other"
}

func callName(call *ast.CallExpr) (string, bool) {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		if id, ok := fun.X.(*ast.Ident); ok {
			return id.Name + "." + fun.Sel.Name, true
		}
	case *ast.Ident:
		return fun.Name, true
	}
	return "", false
}

func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func checkDistribution(t Tree) Report {
	const name = "distribution"
	r := Report{Check: name, Promise: "no distributable form of this program (image, package, installer) and no hosted mode, except the one declared install helper", Files: len(t)}
	if len(t) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no files: a scan of nothing proves nothing"))
		return r
	}
	allowed := map[string]bool{}
	for _, f := range t {
		base := path.Base(f.Path)
		if note, ok := distributionAllowed[f.Path]; ok {
			allowed[note] = true
			continue
		}
		for _, re := range distributionNameRes {
			if re.MatchString(base) {
				r.Findings = append(r.Findings, fileFinding(name, f.Path, fmt.Sprintf("distributable artifact %q exists (image, package or installer)", base), 0))
			}
		}
		for _, re := range distributionPathRes {
			if re.MatchString(f.Path) {
				r.Findings = append(r.Findings, fileFinding(name, f.Path, "a workflow that publishes a distributable form (image, package or installer) exists", 0))
			}
		}
	}
	// The declared helper is printed with its reason, and is required when its file is in the tree:
	// an allowance whose file is gone is a declaration to remove, not a permission to keep.
	present := map[string]bool{}
	for _, f := range t {
		present[f.Path] = true
	}
	for path, note := range distributionAllowed {
		if !present[path] {
			continue
		}
		if allowed[note] {
			r.Promise = fmt.Sprintf("%s (%s: %s)", r.Promise, path, note)
		}
	}

	// The file-name and manifest arm above reads every file it was handed. This
	// arm reads the syntax of the Go files that parse: one that does not is
	// reported once, by name, by the source check, and is not read silently here.
	for _, s := range parseGoSources(t.GoFiles(), 0) {
		ast.Inspect(s.AST, func(n ast.Node) bool {
			// net.Listen*, http.Server type, ListenAndServe*
			switch v := n.(type) {
			case *ast.SelectorExpr:
				xid, ok := v.X.(*ast.Ident)
				if !ok {
					return true
				}
				if (xid.Name == "net" || xid.Name == "http") && hostedConstructs[v.Sel.Name] {
					pos := s.FSet.Position(v.Pos())
					r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("hosted-mode construct %s.%s", xid.Name, v.Sel.Name), pos.Line))
				}
				if xid.Name == "http" && v.Sel.Name == "Server" {
					pos := s.FSet.Position(v.Pos())
					r.Findings = append(r.Findings, fileFinding(name, s.Path, "http.Server is a hosted-mode construct", pos.Line))
				}
			case *ast.BasicLit:
				if v.Kind != token.STRING {
					return true
				}
				if strings.HasPrefix(s.Path, "internal/audit/") {
					return true
				}
				lit := strings.Trim(v.Value, `"`)
				if bindAddr(lit) {
					pos := s.FSet.Position(v.Pos())
					r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("bind address literal %q", lit), pos.Line))
				}
			}
			return true
		})
	}
	return r
}

func bindAddr(s string) bool {
	if s == ":http" || s == ":https" || s == ":0" {
		return true
	}
	if strings.HasPrefix(s, ":") {
		rest := s[1:]
		if rest == "" {
			return false
		}
		for _, r := range rest {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	if strings.HasPrefix(s, "0.0.0.0:") || strings.HasPrefix(s, "127.0.0.1:") || strings.HasPrefix(s, "[::]:") {
		return true
	}
	return false
}

func checkAccount(t Tree) Report {
	const name = "account"
	sources := parseGoSources(t.GoFiles(), parser.ParseComments)
	r := Report{Check: name, Promise: "one session store: one file per account name, each 0600, and one credential pair per file", Files: len(sources)}
	if len(sources) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no Go files: a scan of nothing proves nothing"))
		return r
	}
	var sessionPathHits []Finding
	var credentialPairHits []Finding
	for _, s := range sources {
		// Session-path resolution: Config.SessionPath default assignment and session.Load.
		if s.Path == "internal/config/config.go" {
			ast.Inspect(s.AST, func(n ast.Node) bool {
				kv, ok := n.(*ast.KeyValueExpr)
				if !ok {
					return true
				}
				id, ok := kv.Key.(*ast.Ident)
				if !ok {
					return true
				}
				// Two resolutions are declared: the legacy single file and the directory the named
				// accounts live in. Each is one place, and the check counts them separately so a
				// second store cannot appear beside them.
				pos := s.FSet.Position(kv.Pos())
				switch id.Name {
				case "SessionPath":
					sessionPathHits = append(sessionPathHits, fileFinding(name, s.Path, "session-path resolution (Config.SessionPath)", pos.Line))
				case "SessionsDir":
					sessionPathHits = append(sessionPathHits, fileFinding(name, s.Path, "session-store resolution (Config.SessionsDir)", pos.Line))
				}
				return true
			})
		}
		if s.Path == "internal/session/session.go" {
			for _, decl := range s.AST.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || fn.Name.Name != "Load" {
					continue
				}
				pos := s.FSet.Position(fn.Pos())
				sessionPathHits = append(sessionPathHits, fileFinding(name, s.Path, "session loader (session.Load)", pos.Line))
			}
			// Credential pair: the two cookie keys.
			ast.Inspect(s.AST, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				v := strings.Trim(lit.Value, `"`)
				if v == "auth_token" || v == "ct0" {
					pos := s.FSet.Position(lit.Pos())
					credentialPairHits = append(credentialPairHits, fileFinding(name, s.Path, fmt.Sprintf("credential key %q", v), pos.Line))
				}
				return true
			})
		}
		if strings.HasPrefix(s.Path, "internal/audit/") {
			continue
		}
		ast.Inspect(s.AST, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			v := strings.Trim(lit.Value, `"`)
			for _, flag := range accountFlagLits {
				if v == flag {
					pos := s.FSet.Position(lit.Pos())
					r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("per-account flag %q", v), pos.Line))
				}
			}
			for _, key := range accountConfigKeys {
				if v == key {
					pos := s.FSet.Position(lit.Pos())
					r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("per-account config key %q", v), pos.Line))
				}
			}
			return true
		})
	}
	if len(sessionPathHits) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "found 0 session-path resolutions: a build with no session path is not a build with one account"))
	}
	// Three resolutions are declared: the legacy file's path and the sessions directory, both in
	// internal/config/config.go, and the loader in internal/session/session.go. Any other hit, and
	// a second hit at any site, is a second credential surface — reported at its own file:line
	// rather than by its position in the walk.
	var configDefaults, configStores, loaders int
	for _, h := range sessionPathHits {
		switch h.Detail {
		case "session-path resolution (Config.SessionPath)":
			configDefaults++
			if configDefaults > 1 {
				r.Findings = append(r.Findings, secondSessionPath(h))
			}
		case "session-store resolution (Config.SessionsDir)":
			configStores++
			if configStores > 1 {
				r.Findings = append(r.Findings, secondSessionPath(h))
			}
		case "session loader (session.Load)":
			loaders++
			if loaders > 1 {
				r.Findings = append(r.Findings, secondSessionPath(h))
			}
		default:
			r.Findings = append(r.Findings, secondSessionPath(h))
		}
	}
	if len(credentialPairHits) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "found 0 credential-pair keys (auth_token/ct0): anti-vacuity"))
	}
	return r
}

// secondSessionPath marks a resolution that is not one of the two declared
// sites, keeping the finding at the file and line that resolved it.
func secondSessionPath(h Finding) Finding {
	h.Detail = "second session-path resolution: " + h.Detail
	return h
}

func checkBoundaries(t Tree) Report {
	const name = "boundaries"
	// One parse per file, imports only: the tui import arm and the os/exec
	// confirmation arm read the same import block. A file that does not parse is
	// reported once, by name, by the source check, and is not counted here.
	sources := parseGoSources(t.GoFiles(), parser.ImportsOnly)
	r := Report{Check: name, Promise: "import boundaries and volatile vocabulary still hold", Files: len(sources)}
	if len(sources) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no Go files: a scan of nothing proves nothing"))
		return r
	}
	var tuiFiles, sawOsExecTerm, sawOsExecMedia, sawOsExecBrowser int
	for _, s := range sources {
		if strings.HasPrefix(s.Path, "internal/tui/") {
			tuiFiles++
			for _, imp := range s.AST.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				for _, bad := range tuiForbidden {
					if boundaryHit(p, bad) {
						pos := s.FSet.Position(imp.Pos())
						r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("internal/tui imports %s", p), pos.Line))
					}
				}
			}
		}
		if strings.Contains(string(s.Src), `"os/exec"`) {
			switch {
			case strings.HasPrefix(s.Path, "internal/term/"):
				sawOsExecTerm++
			case strings.HasPrefix(s.Path, "internal/media/"):
				sawOsExecMedia++
			case strings.HasPrefix(s.Path, browserProfileDir):
				// The third declared spawner: `login --browser` starts a throwaway Chromium.
				sawOsExecBrowser++
			default:
				// Confirm it is an import, not a comment string.
				for _, imp := range s.AST.Imports {
					if strings.Trim(imp.Path.Value, `"`) == "os/exec" {
						pos := s.FSet.Position(imp.Pos())
						r.Findings = append(r.Findings, fileFinding(name, s.Path, "os/exec import outside internal/term, internal/media and internal/browserlogin", pos.Line))
					}
				}
			}
		}
		// Volatile vocabulary outside internal/xweb/volatile (and outside this
		// package, which names the tokens it looks for).
		if strings.HasPrefix(s.Path, "internal/xweb/volatile/") || strings.HasPrefix(s.Path, "internal/opsgen/") || strings.HasPrefix(s.Path, "internal/audit/") {
			continue
		}
		for _, vocab := range volatileVocab {
			if idx := strings.Index(string(s.Src), vocab); idx >= 0 {
				line := 1 + strings.Count(string(s.Src[:idx]), "\n")
				r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("volatile vocabulary %q outside internal/xweb/volatile", vocab), line))
			}
		}
	}
	if tuiFiles == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "found 0 internal/tui Go files: anti-vacuity"))
	}
	if sawOsExecTerm == 0 || sawOsExecMedia == 0 || sawOsExecBrowser == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, fmt.Sprintf("os/exec allowlist incomplete: term=%d media=%d browserlogin=%d (all three must appear)", sawOsExecTerm, sawOsExecMedia, sawOsExecBrowser)))
	}
	return r
}

func boundaryHit(imp, forbidden string) bool {
	if imp == forbidden {
		return true
	}
	if strings.HasPrefix(imp, forbidden+"/") {
		return true
	}
	// Module path forms: github.com/.../internal/store
	if i := strings.Index(imp, "/"+forbidden); i >= 0 {
		rest := imp[i+1:]
		if rest == forbidden || strings.HasPrefix(rest, forbidden+"/") {
			return true
		}
	}
	return false
}
