package audit

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"regexp"
	"slices"
	"strings"
)

// opClass classifies every GraphQL operation name this build may spell. A name
// the table does not classify is a finding ("unclassified"), a name classified
// write is a finding, and an empty collected set is a finding (anti-vacuity).
// Every name in it is one the registry holds, which is what checkOpNames audits:
// an entry for an operation this build cannot resolve is a promise about a
// request that could never be sent, so the two files are not allowed to drift
// apart in silence.
//
// This check scans names the code *spells* as string literals. A name assembled
// at run time from the registry is not visible to it — which is exactly why the
// transport check exists beside it. The registry JSON is data and carries the
// whole bundle's map, writes included; it is exempt from the write-name rule
// while the code that could name one is not.
var opClass = map[string]string{
	// Reads this build may name.
	"Viewer":                "read",
	"HomeTimeline":          "read",
	"HomeLatestTimeline":    "read",
	"UserByScreenName":      "read",
	"UserTweets":            "read",
	"TweetDetail":           "read",
	"SearchTimeline":        "read",
	"Bookmarks":             "read",
	"Likes":                 "read",
	"Following":             "read",
	"Followers":             "read",
	"BlueVerifiedFollowers": "read",
	"UserMedia":             "read",
	"ExplorePage":           "read",

	// Writes: the post, reply, like, repost, follow and bookmark write operations.
	"CreateTweet":       "write",
	"CreateNoteTweet":   "write",
	"CreateRetweet":     "write",
	"DeleteTweet":       "write",
	"DeleteRetweet":     "write",
	"FavoriteTweet":     "write",
	"UnfavoriteTweet":   "write",
	"CreateBookmark":    "write",
	"DeleteBookmark":    "write",
	"CreateHighlight":   "write",
	"DeleteHighlight":   "write",
	"PinTweet":          "write",
	"UnpinTweet":        "write",
	"ModerateTweet":     "write",
	"UnmoderateTweet":   "write",
	"DownvoteTweet":     "write",
	"UndoDownvoteTweet": "write",
	"RemoveFollower":    "write",
	"dmBlockUser":       "write",
	"dmUnblockUser":     "write",
}

var (
	opNameRe    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	methodLitRe = regexp.MustCompile(`^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS|CONNECT|TRACE)$`)
)

// The two methods this tree may build a request with, as plain strings. They are not the
// http.Method* constants on purpose: this file is itself scanned by the check below, whose
// constant arm reports every such spelling outside the one exemption file — and a check that
// reports itself is a check nobody can read. (Its tests, being test files, are not scanned.)
const (
	methodGet  = "GET"
	methodPost = "POST"
)

type opHit struct {
	Name string
	Path string
	Line int
}

// readonlySources is the Go source the readonly check reads: every non-test Go
// file outside this package, which spells the operation names it looks for. A
// file that does not parse is reported once, by name, by the source check, and
// is not counted as inspected here.
func readonlySources(t Tree) []goSource {
	var files Tree
	for _, f := range t.GoFiles() {
		if strings.HasPrefix(f.Path, "internal/audit/") {
			continue
		}
		files = append(files, f)
	}
	return parseGoSources(files, 0)
}

// operationNames collects every string literal that looks like a GraphQL
// operation name: any literal that either appears in the registry's own
// operation-name set or appears in this package's write-name table.
func operationNames(sources []goSource, registry map[string]bool) []opHit {
	var hits []opHit
	for _, s := range sources {
		ast.Inspect(s.AST, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val := strings.Trim(lit.Value, `"`)
			if !opNameRe.MatchString(val) {
				return true
			}
			_, inTable := opClass[val]
			_, inReg := registry[val]
			if !inTable && !inReg {
				return true
			}
			pos := s.FSet.Position(lit.Pos())
			hits = append(hits, opHit{Name: val, Path: s.Path, Line: pos.Line})
			return true
		})
	}
	return hits
}

// registryJSONFile is the checked-in registry, and registryNames is the
// operation-name set it carries. It is data this package reads rather than
// source it audits, so a decode failure is a finding at the check that reads it:
// returning an empty set in silence would narrow the search and say nothing.
const registryJSONFile = "internal/xweb/volatile/registry.json"

func registryNames(t Tree) (map[string]bool, error) {
	out := map[string]bool{}
	for _, f := range t {
		if f.Path != registryJSONFile {
			continue
		}
		var file struct {
			Operations map[string]json.RawMessage `json:"operations"`
		}
		if err := json.Unmarshal(f.Src, &file); err != nil {
			return out, err
		}
		for name := range file.Operations {
			out[name] = true
		}
	}
	return out, nil
}

// registrySources is the snapshot the classification table is checked against: the registry JSON
// the tree holds, as the data it is rather than as source this package reads.
func registrySources(t Tree) Tree {
	var out Tree
	for _, f := range t {
		if f.Path == registryJSONFile {
			out = append(out, f)
		}
	}
	return out
}

// checkOpNames is the classification table's own promise, audited on its own: every operation name
// the table knows is one the registry can resolve.
//
// The table is what decides whether a name spelled in source is a read or a write, so an entry the
// registry does not hold is a promise about an operation this build cannot ask for — a read that
// would have no cap and no remedy, or a write name the write rule guards against something nobody
// can send. The subset is checked rather than left to review because the two files are edited
// independently: the snapshot is regenerated by cmd/x-opsgen from what x.com ships, and a name
// leaving the snapshot is exactly the silent drift this check exists to catch. It was written for
// the stale ListLatestTweetsTimeline entry — an operation volatile never held.
func checkOpNames(t Tree) Report {
	const name = "opnames"
	sources := registrySources(t)
	r := Report{Check: name, Promise: "every operation name the classification table knows is one the registry resolves", Files: len(sources)}
	if len(sources) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, fmt.Sprintf("read no %s: a table cannot be checked against a snapshot that was not read", registryJSONFile)))
		return r
	}
	if len(opClass) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "the classification table is empty: every name in it resolves vacuously"))
		return r
	}
	registry, err := registryNames(t)
	if err != nil {
		r.Findings = append(r.Findings, fileFinding(name, registryJSONFile, fmt.Sprintf("does not decode: %v — the names it holds cannot be read, so the table is unchecked", err), 0))
		return r
	}
	names := make([]string, 0, len(opClass))
	for n := range opClass {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		if !registry[n] {
			r.Findings = append(r.Findings, fileFinding(name, "internal/audit/readonly.go",
				fmt.Sprintf("the table classifies %q as a %s and the registry does not hold it: an entry for an operation this build cannot resolve", n, opClass[n]), 0))
		}
	}
	return r
}

func checkReadonly(t Tree) Report {
	const name = "readonly"
	sources := readonlySources(t)
	r := Report{Check: name, Promise: "no write operation is nameable in non-test source", Files: len(sources)}
	if len(sources) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no Go files: a scan of nothing proves nothing"))
		return r
	}
	registry, err := registryNames(t)
	if err != nil {
		r.Findings = append(r.Findings, fileFinding(name, registryJSONFile, fmt.Sprintf("does not decode: %v — the collected name set is this package's table alone", err), 0))
	}
	hits := operationNames(sources, registry)
	if len(hits) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, fmt.Sprintf("collected 0 operation-name literals across %d Go files: anti-vacuity", len(sources))))
		return r
	}
	for _, h := range hits {
		class, ok := opClass[h.Name]
		switch {
		case !ok:
			r.Findings = append(r.Findings, fileFinding(name, h.Path, fmt.Sprintf("unclassified operation name %q: is this a read or a write?", h.Name), h.Line))
		case class == "write":
			r.Findings = append(r.Findings, fileFinding(name, h.Path, fmt.Sprintf("write operation name %q is spelled in source", h.Name), h.Line))
		}
	}
	return r
}

// methodExemptionFile is the one file allowed to build a POST: internal/xweb/graphql.go, which
// owns the per-operation wire method decision (its `postOperations` table). The exemption is
// named here so it is visible and printed rather than silent, exactly the way the timer check
// names the limiter's floor.
//
// The promise is nevertheless carried by more than the file name, and the four things that carry
// it are worth stating where the check is read: no write operation name is spellable anywhere in
// non-test source (checkReadonly, beside this one), only GET and POST exist anywhere in the tree
// (the constant arm below), POST exists in exactly this one declared file (both arms below), and
// the client refuses a non-`query` operation before it builds a request at all.
const methodExemptionFile = "internal/xweb/graphql.go"

func checkTransport(t Tree) Report {
	const name = "transport"
	sources := parseGoSources(t.GoFiles(), 0)
	r := Report{Check: name, Promise: "no mutation is expressible over the wire", Files: len(sources)}
	if len(sources) == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, "inspected no Go files: a scan of nothing proves nothing"))
		return r
	}
	var (
		builders        int
		exemptionInTree bool
		postLine        int
	)
	for _, s := range sources {
		if filepathToSlash(s.Path) == methodExemptionFile {
			exemptionInTree = true
		}
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
			if !ok || xid.Name != "http" {
				return true
			}
			switch sel.Sel.Name {
			case "NewRequest", "NewRequestWithContext":
				builders++
				pos := s.FSet.Position(call.Pos())
				if !requestBuilderOK(s.Path) {
					r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("http.%s builds an *http.Request outside the GET request-builder packages", sel.Sel.Name), pos.Line))
				}
				// The two methods are compared as literals rather than as http.Method*
				// selectors, because this file is scanned by this check: spelling one here
				// would make the constant arm below report the checker itself.
				method, line, known := requestMethod(s.FSet, call)
				switch {
				case !known || method == methodGet:
					// A method this arm cannot read as a literal is not a pass: the constant
					// arm below catches every http.Method* spelling, and a method assembled
					// at run time in an unexempted file is caught there too.
				case method == methodPost:
					if filepathToSlash(s.Path) == methodExemptionFile {
						postLine = line
					} else {
						r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("request method POST is built outside the declared exemption %s", methodExemptionFile), line))
					}
				default:
					r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("request method %s is neither GET nor POST", method), line))
				}
			}
			return true
		})
		// The constant arm: every http.Method* spelling anywhere, except MethodGet.
		ast.Inspect(s.AST, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			xid, ok := sel.X.(*ast.Ident)
			if !ok || xid.Name != "http" {
				return true
			}
			if !strings.HasPrefix(sel.Sel.Name, "Method") || sel.Sel.Name == "MethodGet" {
				return true
			}
			pos := s.FSet.Position(sel.Pos())
			if sel.Sel.Name == "MethodPost" {
				if filepathToSlash(s.Path) == methodExemptionFile {
					if postLine == 0 {
						postLine = pos.Line
					}
					return true
				}
				r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("http.MethodPost is the POST method constant, and it exists outside the declared exemption %s", methodExemptionFile), pos.Line))
				return true
			}
			r.Findings = append(r.Findings, fileFinding(name, s.Path, fmt.Sprintf("http.%s is a method constant that is neither GET nor POST", sel.Sel.Name), pos.Line))
			return true
		})
	}
	if builders == 0 {
		r.Findings = append(r.Findings, emptyFinding(name, fmt.Sprintf("found 0 http.NewRequest* call sites across %d Go files: anti-vacuity", len(sources))))
	}
	if !exemptionInTree {
		r.Findings = append(r.Findings, emptyFinding(name, fmt.Sprintf("the declared POST exemption %s is not in the tree that was read: an exemption for a file that is not there cannot have inspected anything", methodExemptionFile)))
	} else {
		// Print the exemption so it is counted, not silent.
		r.Promise = fmt.Sprintf("%s (POST exemption %s:%d)", r.Promise, methodExemptionFile, postLine)
	}
	return r
}

// requestBuilderOK reports whether path may call http.NewRequest*. The plan
// names internal/xweb; the real tree also builds GET-only requests in
// internal/netcheck (doctor's path probe), internal/opsgen (bundle refresh) and
// internal/release (the update verb's release wire — two GETs to a host that is
// not x.com, carrying no x.com session and spending no account budget). All four
// stay GET-only via the method arm of this check: nothing is exempted here, no
// method arm is widened, and no method constant other than the one GET spelling
// is added anywhere.
func requestBuilderOK(p string) bool {
	p = path.Clean(filepathToSlash(p))
	for _, root := range []string{"internal/xweb", "internal/netcheck", "internal/opsgen", "internal/release"} {
		if p == root || strings.HasPrefix(p, root+"/") {
			return true
		}
	}
	return false
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

func requestMethod(fset *token.FileSet, call *ast.CallExpr) (string, int, bool) {
	// NewRequest(method, ...) or NewRequestWithContext(ctx, method, ...)
	idx := 0
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "NewRequestWithContext" {
		idx = 1
	}
	if idx >= len(call.Args) {
		return "", 0, false
	}
	arg := call.Args[idx]
	switch v := arg.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", 0, false
		}
		m := strings.Trim(v.Value, `"`)
		if !methodLitRe.MatchString(m) {
			return "", 0, false
		}
		return m, fset.Position(v.Pos()).Line, true
	case *ast.SelectorExpr:
		if xid, ok := v.X.(*ast.Ident); ok && xid.Name == "http" && strings.HasPrefix(v.Sel.Name, "Method") {
			m := strings.TrimPrefix(v.Sel.Name, "Method")
			return strings.ToUpper(m), fset.Position(v.Pos()).Line, true
		}
	case *ast.Ident:
		// method variable — not a literal; transport still catches Method* constants separately
	}
	return "", 0, false
}
