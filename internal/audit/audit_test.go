package audit

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// tests run with cwd = package dir
	root := filepath.Clean(filepath.Join(wd, "../.."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repository root not found from %s: %v", wd, err)
	}
	return root
}

func TestLoad(t *testing.T) {
	tree, err := Load(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) == 0 {
		t.Fatal("Load returned no files")
	}
	var goFiles int
	var sawRegistry, sawTest bool
	for _, f := range tree {
		if strings.HasSuffix(f.Path, ".go") {
			goFiles++
			if strings.HasSuffix(f.Path, "_test.go") {
				sawTest = true
			}
		}
		if f.Path == "internal/xweb/volatile/registry.json" {
			sawRegistry = true
		}
	}
	if goFiles == 0 {
		t.Fatal("Load found no Go files")
	}
	if !sawRegistry {
		t.Fatal("Load did not include the registry JSON")
	}
	if sawTest {
		t.Fatal("Load included a _test.go file; tests are excluded so they may name what they drive")
	}
	t.Logf("Load: %d files, %d Go sources", len(tree), goFiles)
}

func TestPackageImportsNeitherStoreNorXweb(t *testing.T) {
	dir := "."
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, imp := range file.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				if strings.Contains(p, "/internal/store") || strings.HasSuffix(p, "/store") {
					t.Errorf("%s imports internal/store", fset.Position(imp.Pos()))
				}
				if strings.Contains(p, "/internal/xweb") || strings.HasSuffix(p, "/xweb") {
					t.Errorf("%s imports internal/xweb", fset.Position(imp.Pos()))
				}
				if strings.Contains(p, "/internal/cli") {
					t.Errorf("%s imports the shell (internal/cli)", fset.Position(imp.Pos()))
				}
			}
		}
	}
}

// auditCheckNames is the audit's check set, written out by hand: the contract this
// file, the CLI's report and the reader of the promise line are all written
// against. A check deleted from Checks(), renamed, reordered or added changes this
// list or fails the build, so a promise cannot stop being audited quietly.
//
// The order is part of the contract as well: Checks() documents fixed report
// order, and both the text and the JSON report are read in it.
var auditCheckNames = []string{
	"source",
	"readonly",
	// opnames is the classification table's own promise, audited on its own: the
	// table and the snapshot are edited independently, so the subset is a check
	// rather than a review habit.
	"opnames",
	"transport",
	"timers",
	"persistence",
	"distribution",
	"account",
	"boundaries",
	"leak",
	// hidden is the second display state's own promise: the design says the key sequence is the only
	// way in, and a start-up path is the one thing that would quietly make that false.
	"hidden",
}

func checkNames(checks []Check) []string {
	names := make([]string, 0, len(checks))
	for _, c := range checks {
		names = append(names, c.Name)
	}
	return names
}

// TestCheckSetIsPinned: the set of promises this build makes is a written-down
// list, not whatever Checks() happens to return. Deleting a check from Checks() is
// not a way to stop auditing what it promised — it is this test failing by name.
func TestCheckSetIsPinned(t *testing.T) {
	checks := Checks()
	if len(checks) != len(auditCheckNames) {
		t.Fatalf("the audit runs %d checks %v, want the pinned %d %v: a check added or deleted changes the audited promise set",
			len(checks), checkNames(checks), len(auditCheckNames), auditCheckNames)
	}
	for i, want := range auditCheckNames {
		got := checks[i]
		if got.Name != want {
			t.Errorf("check %d is %q, want %q — the names and the report order are part of the contract\n got: %v\nwant: %v",
				i, got.Name, want, checkNames(checks), auditCheckNames)
			continue
		}
		if got.Run == nil {
			t.Errorf("check %q has no Run function: it would be reported as a finding, not as a check", want)
		}
		if got.Promise == "" {
			t.Errorf("check %q states no promise: a check with nothing to promise proves nothing to the reader", want)
		}
	}
}

func TestRunRealTree(t *testing.T) {
	reports, err := Run(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != len(auditCheckNames) {
		t.Fatalf("got %d reports, want the pinned %d (%v)", len(reports), len(auditCheckNames), auditCheckNames)
	}
	for i, r := range reports {
		if r.Check != auditCheckNames[i] {
			t.Errorf("report %d is %q, want %q: the report order is the pinned order", i, r.Check, auditCheckNames[i])
		}
		t.Logf("%s: %d files, %d findings — %s", r.Check, r.Files, len(r.Findings), r.Promise)
		for _, f := range r.Findings {
			t.Errorf("%s: %s:%d: %s", r.Check, f.Path, f.Line, f.Detail)
		}
	}
}

func TestEmptyTreeIsAFinding(t *testing.T) {
	for _, c := range Checks() {
		r := c.Run(nil)
		if len(r.Findings) == 0 {
			t.Errorf("%s: empty tree produced no finding", c.Name)
		}
	}
}

func TestPanicBecomesFinding(t *testing.T) {
	r := runCheck(Check{Name: "boom", Promise: "x", Run: func(Tree) Report {
		panic("planted")
	}}, nil)
	if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Detail, "panicked") {
		t.Fatalf("panic was not recovered as a finding: %+v", r)
	}
}

// brokenSource is a Go file that does not parse: the string stops mid-call, and
// both a timer and a write sit behind the point where the parser stopped. A scan
// that reads syntax used to skip a file like this in silence, so a tree holding
// one reported no findings and exited 0.
const brokenSource = `package p

import (
	"os"
	"time"
)

func f() {
	time.Sleep(time.Second)
	_ = os.WriteFile("/tmp/hidden.log", nil, 0o600)
`

// TestSourceFailsOnAFileThatDoesNotParse: the finding names the file, the line
// the parser stopped at and the parser's own message — pinned here, wording and
// all, because that message is what the reader gets instead of a silent pass.
// The file is still counted among the files this check inspected: it read the
// file and reported what it found, which is not the same as skipping it.
func TestSourceFailsOnAFileThatDoesNotParse(t *testing.T) {
	r := checkSource(treeOf(
		File{Path: "internal/cli/fine.go", Src: []byte("package cli\n\nfunc g() {}\n")},
		File{Path: "internal/cli/broken.go", Src: []byte(brokenSource)},
	))
	if len(r.Findings) != 1 {
		t.Fatalf("want exactly one finding, got %+v", r.Findings)
	}
	got := r.Findings[0]
	if got.Check != "source" || got.Path != "internal/cli/broken.go" {
		t.Fatalf("the finding does not name the file it could not parse: %+v", got)
	}
	if want := "cannot parse: expected '}', found 'EOF'"; got.Detail != want {
		t.Fatalf("detail is %q, want the parser's message %q", got.Detail, want)
	}
	if got.Line != 10 {
		t.Fatalf("the finding is not at the line the parser stopped at: %+v", got)
	}
	if r.Files != 2 {
		t.Fatalf("Files=%d, want 2: a file it could not parse is still a file it read", r.Files)
	}
	clean := checkSource(treeOf(File{Path: "internal/cli/fine.go", Src: []byte("package cli\n\nfunc g() {}\n")}))
	if len(clean.Findings) != 0 {
		t.Fatalf("a file that parses is a finding: %+v", clean.Findings)
	}
	// One finding per file that does not parse, not one for the tree: a second
	// broken file is a second finding at its own path.
	two := checkSource(treeOf(
		File{Path: "internal/cli/broken.go", Src: []byte(brokenSource)},
		File{Path: "internal/xweb/broken.go", Src: []byte("package xweb\n\nfunc g( {\n")},
	))
	if len(two.Findings) != 2 {
		t.Fatalf("want one finding per file, got %+v", two.Findings)
	}
	for _, f := range two.Findings {
		if !strings.HasPrefix(f.Detail, "cannot parse: ") {
			t.Fatalf("finding does not carry the parse failure: %+v", f)
		}
	}
}

// TestRunFailsOnATreeHoldingAFileThatDoesNotParse is the end-to-end arm: a tree
// on disk whose only Go file is unparseable, with a timer and a write behind the
// syntax error. The scans that read that syntax find nothing in it, and the run
// still fails and names the file, because the verdict is the report's rather
// than any one scan's.
func TestRunFailsOnATreeHoldingAFileThatDoesNotParse(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.go"), []byte(brokenSource), 0o600); err != nil {
		t.Fatal(err)
	}
	reports, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	var named bool
	for _, r := range reports {
		for _, f := range r.Findings {
			if r.Check == "source" && f.Path == "broken.go" {
				named = true
			}
		}
		// A scan that reads Go syntax reports this file as one it did not
		// inspect, not as one it inspected and found nothing in.
		if r.Check == "timers" && r.Files != 0 {
			t.Errorf("timers counted %d files while reading none of them: %+v", r.Files, r.Findings)
		}
	}
	if !named {
		t.Fatalf("the run did not name the file it could not parse: %+v", reports)
	}
	if FindingCount(reports) == 0 {
		t.Fatal("a tree whose only Go file does not parse reported no findings")
	}
}

// TestScansDoNotCountAFileTheyCannotParse pins the two halves of the fix: the
// scans read the files that parse and count those, and the one file they could
// not read is named by the source check. A tree in which nothing parses trips the
// scan's own vacuity arm rather than passing over it.
func TestScansDoNotCountAFileTheyCannotParse(t *testing.T) {
	exempt := File{Path: limiterFloorFile, Src: []byte("package xweb\n\nimport \"time\"\n\nfunc sleepContext() { t := time.NewTimer(0); _ = t }\n")}
	broken := File{Path: "internal/cli/broken.go", Src: []byte(brokenSource)}

	r := checkTimers(treeOf(exempt, broken))
	if r.Files != 1 {
		t.Fatalf("timers counted Files=%d, want 1: the file it could not parse is not one it inspected", r.Files)
	}
	if len(r.Findings) != 0 {
		t.Fatalf("timers reported %+v: the timer behind the syntax error is the source check's finding", r.Findings)
	}
	if !hasDetail(checkSource(treeOf(exempt, broken)), "cannot parse") {
		t.Fatal("the source check does not name the file the timers scan could not parse")
	}
	if only := checkTimers(treeOf(broken)); only.Files != 0 || !hasDetail(only, "inspected no Go files") {
		t.Fatalf("Files=%d findings=%+v, want the vacuity finding", only.Files, only.Findings)
	}
}
