package tui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// The phase requirement's import boundary, and the packages that would make the shell a second
// composition root. The five the requirement names are the point — net/http, os/exec, database/sql
// and the two lower layers a timeline could leak into — and the rest are the ones that would let
// the shell reach the wire, the credential, the check or the CLI table without naming those five.
//
// internal/netcheck is deliberately not on the list: Deps.Failure takes the reachability check's
// results as an argument, which is a type in a signature rather than a call, and
// TestTheShellOnlyNamesNetcheckAsAType is what keeps it that way.
var forbiddenImports = []string{
	"net/http",
	"os/exec",
	"database/sql",
	"github.com/LZD121/x-terminal/internal/xsource",
	"github.com/LZD121/x-terminal/internal/store",
	"github.com/LZD121/x-terminal/internal/xweb",
	"github.com/LZD121/x-terminal/internal/session",
	"github.com/LZD121/x-terminal/internal/cli",
	"github.com/LZD121/x-terminal/internal/fake",
}

// forbidden reports which root of the list an import path reaches, or the empty string for a path
// that reaches none. A package is forbidden when it is the root itself or any package beneath it: a
// package's subpackage is a package too, and the subpackages are where the reachable things live —
// `internal/xweb/volatile` reads the operation table off disk, and an exact-match scan lets it in
// behind its parent's name. The comparison is on path segments, so a package whose name merely begins
// with a forbidden one (`internal/xwebx`) is not reported.
func forbidden(path string) string {
	for _, banned := range forbiddenImports {
		if path == banned || strings.HasPrefix(path, banned+"/") {
			return banned
		}
	}
	return ""
}

// parsedFile is one .go file of a package: its name (for the error messages) and its syntax tree.
type parsedFile struct {
	name string
	file *ast.File
}

// parsePackage parses every non-test .go file of a package directory.
//
// Test files are skipped on purpose rather than by accident: a package's tests are allowed to import
// anything they need to drive it, so a scanner that walked them would report a violation for a test
// import and the boundary would stop meaning anything.
func parsePackage(t *testing.T, dir string) []parsedFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("listing %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	files := make([]parsedFile, 0, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		files = append(files, parsedFile{name: name, file: file})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files
}

// parseSource parses one synthetic source. It is how the scanners themselves are exercised: a
// scanner that reports nothing anywhere is indistinguishable from a scanner that reports nothing.
func parseSource(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "planted.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("the planted source does not parse: %v\n%s", err, src)
	}
	return file
}

// importPaths is every import path of one file, in source order.
func importPaths(f *ast.File) []string {
	paths := make([]string, 0, len(f.Imports))
	for _, spec := range f.Imports {
		paths = append(paths, strings.Trim(spec.Path.Value, `"`))
	}
	return paths
}

// forbiddenIn is the shell's scan as one function: every import of every parsed file that reaches a
// forbidden root, as sentences. It is one function rather than a loop inside one test so that the
// planted-violation arm below is held to exactly the rule the clean run is.
func forbiddenIn(files []parsedFile) []string {
	var violations []string
	for _, parsed := range files {
		for _, path := range importPaths(parsed.file) {
			if banned := forbidden(path); banned != "" {
				violations = append(violations, parsed.name+" imports "+banned)
			}
		}
	}
	return violations
}

// plantImport copies one real file of this package into a temporary directory with one import added
// to its import block, and returns that directory. It is the arm the boundary needs: a scan that
// finds nothing anywhere passes every negative assertion ever written, so the violation has to be
// planted in a file the scan would otherwise read — view.go's own source, not a synthetic string.
func plantImport(t *testing.T, name, path string) string {
	t.Helper()
	body, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("reading %s to plant an import in: %v", name, err)
	}
	at := strings.Index(string(body), "import (")
	if at < 0 {
		t.Fatalf("%s carries no import block to plant an import in", name)
	}
	at += len("import (")
	planted := string(body)[:at] + "\n\t_ \"" + path + "\"\n" + string(body)[at:]

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, name), planted)
	return dir
}

// TestTheShellCannotReachIO is the import boundary: the shell is pure Update/View over a port it
// was handed, and this is the test that keeps it that way rather than the comment that says so.
//
// The boundary did not move when the disguise landed: the shell may call internal/term through the
// effect/Cmd boundary — one method per state change, a value in and nothing read back — and it may
// not import a process-spawning package itself, so the list below is exactly what it was. The shell
// reaches a spawned program through internal/media's seam and still cannot import a process-spawning
// package itself.
//
// Its arms are what make the silence meaningful. The scan has to have found files and imports at
// all — an empty scan passes every "no forbidden import" assertion ever written — the scanner is
// exercised on a source that does contain a forbidden path, the shell has to import the UI toolkit,
// which proves the scan is looking at the shell's directory and not at an empty one, and a planted
// violation in a real package file — view.go's own source with one import added — has to make the
// same scan report it.
func TestTheShellCannotReachIO(t *testing.T) {
	files := parsePackage(t, ".")
	if len(files) < 3 {
		t.Fatalf("the scan parsed %d files, want the shell's own sources: an empty scan fails", len(files))
	}

	imports := 0
	imported := map[string]bool{}
	for _, parsed := range files {
		for _, path := range importPaths(parsed.file) {
			imports++
			imported[path] = true
		}
	}
	for _, violation := range forbiddenIn(files) {
		t.Errorf("internal/tui/%s, which the shell may not reach", violation)
	}
	if imports < 8 {
		t.Fatalf("the scan collected %d imports, far fewer than the shell has: the scan is not looking at the shell", imports)
	}
	if !imported["charm.land/bubbletea/v2"] {
		t.Fatalf("the scan did not find the UI toolkit in internal/tui's imports: it is not reading the shell")
	}

	// The arm the disguise's boundary asks for: the same scan over the same directory with one real
	// package file carrying a planted import. view.go's own source gets the one added line, so what
	// the scanner is handed is a file of this package and not a synthetic string.
	planted := plantImport(t, "view.go", "os/exec")
	if !slices.Contains(forbiddenIn(parsePackage(t, planted)), "view.go imports os/exec") {
		t.Fatalf("a planted os/exec in a real package file was not reported: the scan above proves nothing")
	}

	// The scanner's own arms. First the planted imports the scan reads: two forbidden paths are
	// reported.
	synthetic := parseSource(t, "package p\n\nimport (\n\t_ \"net/http\"\n\t_ \"os/exec\"\n)\n")
	if got := importPaths(synthetic); len(got) != 2 || got[0] != "net/http" {
		t.Fatalf("the scanner read %q out of a source importing net/http and os/exec", got)
	}
	for _, path := range importPaths(synthetic) {
		if banned := forbidden(path); banned != path {
			t.Fatalf("the scanner did not report the planted import %s (it called it %q)", path, banned)
		}
	}

	// And then the shape an exact-match scan lets through: a forbidden package reached by a
	// subpackage of it. `internal/xweb/volatile` is the demonstrated one — it reads the operation
	// table off disk — and a package whose name merely begins with a forbidden one is not a
	// subpackage of it.
	for _, planted := range []struct {
		path string
		want string
	}{
		{"net/http", "net/http"},
		{"net/http/httptest", "net/http"},
		{"github.com/LZD121/x-terminal/internal/xweb/volatile", "github.com/LZD121/x-terminal/internal/xweb"},
		{"github.com/LZD121/x-terminal/internal/session/whatever", "github.com/LZD121/x-terminal/internal/session"},
		{"github.com/LZD121/x-terminal/internal/xwebx", ""},
		{"golang.org/x/net/http2", ""},
	} {
		if got := forbidden(planted.path); got != planted.want {
			t.Errorf("the import scan calls %q forbidden by %q, want %q", planted.path, got, planted.want)
		}
	}
}

// TestProcessSpawnsAreOnlyInTermAndMedia is the other half of the boundary, and it is
// module-wide on purpose: os/exec may appear in exactly two packages of the module —
// internal/term (the panic entry, ED 3, the title and the tmux subprocesses) and
// internal/media (this phase's spawns) — and the shell stays a pure Update/View over
// ports. Formerly TestOsExecIsOnlyInInternalTerm (Phase 3), which allowed only term.
//
// Its arms are the point again. A scan that found no files, or no imports at all, would
// report no violation whatever the tree held; both allowlisted packages must appear among
// the files the scan found, so the test cannot pass while either is missing; the scanner
// is exercised on a synthetic source that carries os/exec; and a planted import in a real
// package outside the two named ones makes the scan fail.
func TestProcessSpawnsAreOnlyInTermAndMedia(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving the module root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("internal/tui is not two levels inside the module: %v", err)
	}

	dirs := modulePackageDirs(t, root)
	files, importers := 0, []string{}
	for _, dir := range dirs {
		parsed := parsePackage(t, dir)
		files += len(parsed)
		for _, name := range osExecImporters(parsed) {
			rel, err := filepath.Rel(root, filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("naming %s: %v", filepath.Join(dir, name), err)
			}
			importers = append(importers, filepath.ToSlash(rel))
		}
	}

	if len(dirs) < 10 {
		t.Fatalf("the walk found %d package directories, fewer than the module has: it is not reading the tree", len(dirs))
	}
	if files < len(dirs) || importsOf(t, dirs) == 0 {
		t.Fatalf("the scan parsed %d files across %d packages and found no imports: an empty scan fails", files, len(dirs))
	}
	allowed := func(path string) bool {
		return strings.HasPrefix(path, "internal/term/") || strings.HasPrefix(path, "internal/media/")
	}
	for _, path := range importers {
		if !allowed(path) {
			t.Errorf("%s imports os/exec, and only internal/term and internal/media may", path)
		}
	}
	var haveTerm, haveMedia bool
	for _, path := range importers {
		if strings.HasPrefix(path, "internal/term/") {
			haveTerm = true
		}
		if strings.HasPrefix(path, "internal/media/") {
			haveMedia = true
		}
	}
	if !haveTerm || !haveMedia {
		t.Fatalf("the scan found os/exec in %q: both internal/term and internal/media must be importing it", importers)
	}

	// The scanner's own arm: the same reader, on a source that does carry the import.
	synthetic := parsePackage(t, plantSource(t, "package p\n\nimport (\n\t_ \"os/exec\"\n\t_ \"strings\"\n)\n"))
	if got := osExecImporters(synthetic); len(got) != 1 || got[0] != "planted.go" {
		t.Fatalf("the scanner read %q out of a source importing os/exec and strings", got)
	}

	// Anti-vacuity: an empty directory set must fail the "both packages present" check.
	if empty := osExecImporters(nil); len(empty) != 0 {
		t.Fatalf("osExecImporters(nil) = %q", empty)
	}
	var emptyImporters []string
	var emptyHaveTerm, emptyHaveMedia bool
	for _, path := range emptyImporters {
		if strings.HasPrefix(path, "internal/term/") {
			emptyHaveTerm = true
		}
		if strings.HasPrefix(path, "internal/media/") {
			emptyHaveMedia = true
		}
	}
	if emptyHaveTerm || emptyHaveMedia {
		t.Fatal("an empty importer set reported an allowlisted package")
	}
	if emptyHaveTerm && emptyHaveMedia {
		t.Fatal("anti-vacuity arm would pass on an empty scan")
	}
}

// osExecImporters is the module scan's decision in one place: the file names of one package that
// import os/exec.
func osExecImporters(files []parsedFile) []string {
	var out []string
	for _, parsed := range files {
		if slices.Contains(importPaths(parsed.file), "os/exec") {
			out = append(out, parsed.name)
		}
	}
	return out
}

// modulePackageDirs is every package directory of the module: a directory holding at least one
// non-test .go file. Dot-directories (the repository's own .planning and .git among them) and
// testdata directories are skipped, because neither is a package of the module.
func modulePackageDirs(t *testing.T, root string) []string {
	t.Helper()
	var dirs []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	sort.Strings(dirs)
	return slices.Compact(dirs)
}

// importsOf is how many import statements the scan collected across the module, which is the
// second shape of an empty scan: files read and nothing inside them.
func importsOf(t *testing.T, dirs []string) int {
	t.Helper()
	total := 0
	for _, dir := range dirs {
		for _, parsed := range parsePackage(t, dir) {
			total += len(parsed.file.Imports)
		}
	}
	return total
}

// plantSource parses one synthetic source in a directory of its own, so that a scanner which works
// on directories can be exercised on bytes it has never seen.
func plantSource(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "planted.go"), src)
	return dir
}

// TestTheShellOnlyNamesNetcheckAsAType is the one exception to the boundary, held to one file and to
// one use. Deps.Failure takes []netcheck.Result because the composition root already has a check's
// report in hand; the shell may name that type and may not run a check, so an import anywhere else
// and a call anywhere at all are both violations.
func TestTheShellOnlyNamesNetcheckAsAType(t *testing.T) {
	const netcheck = "github.com/LZD121/x-terminal/internal/netcheck"
	calls := 0

	for _, parsed := range parsePackage(t, ".") {
		names := importPaths(parsed.file)
		for _, path := range names {
			if path == netcheck && parsed.name != "model.go" {
				t.Errorf("internal/tui/%s imports %s: the reachability report is named in Deps and nowhere else", parsed.name, netcheck)
			}
		}
		ast.Inspect(parsed.file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
				if pkg, ok := selector.X.(*ast.Ident); ok && pkg.Name == "netcheck" {
					calls++
					t.Errorf("internal/tui/%s calls netcheck.%s: the shell never runs a check", parsed.name, selector.Sel.Name)
				}
			}
			return true
		})
	}

	// The arm: the call scanner does report a call, so the assertion above is not vacuous.
	planted := parseSource(t, "package p\n\nimport \"x/internal/netcheck\"\n\nfunc f() {\n\t_, _ = netcheck.NewTarget(nil)\n\t_ = netcheck.ProxyAddress(\"\")\n\t_ = netcheck.Check(nil, nil, nil)\n}\n")
	plantedCalls := 0
	ast.Inspect(planted, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
				if pkg, ok := selector.X.(*ast.Ident); ok && pkg.Name == "netcheck" {
					plantedCalls++
				}
			}
		}
		return true
	})
	if plantedCalls != 3 {
		t.Fatalf("the call scanner found %d calls to netcheck in a source with three", plantedCalls)
	}
}

// TestNoSecondMeasurement is A6: internal/text is the only package allowed to measure a string, and
// the four packages that compose the screen may not carry a second way to do it. The three shapes
// checked are the three a second measure appears as: a byte count (unicode/utf8), a width taken by
// fmt (`%*s`, `%-*s`, `%.*s`, any `%`-verb with a `*`), and len() applied to a string.
func TestNoSecondMeasurement(t *testing.T) {
	for _, dir := range []string{".", "../screen", "../render", "../keys"} {
		files := parsePackage(t, dir)
		// The arm: a package that was not found would report no violation at all.
		if len(files) == 0 {
			t.Fatalf("%s parsed no files: a missing package must fail", dir)
		}
		for _, parsed := range files {
			for _, violation := range measurements(parsed.file) {
				t.Errorf("internal/%s/%s: %s", filepath.Base(dir), parsed.name, violation)
			}
		}
	}

	// internal/text measures, and does not format: fmt would be a second way to build a field.
	for _, parsed := range parsePackage(t, "../text") {
		for _, path := range importPaths(parsed.file) {
			if path == "fmt" {
				t.Errorf("internal/text/%s imports fmt: the one package that measures does not format", parsed.name)
			}
		}
	}

	// The scanner's arms, each one a planted source: a violation of each shape is reported, and the
	// shapes that are not violations are not.
	for _, planted := range []struct {
		src  string
		want string
	}{
		{"package p\n\nimport \"unicode/utf8\"\n\nfunc f() { _ = utf8.RuneCountInString(\"x\") }\n", "unicode/utf8"},
		{"package p\n\nimport \"fmt\"\n\nfunc f(s string) { _ = fmt.Sprintf(\"%*s\", 4, s) }\n", "%*s"},
		{"package p\n\nimport \"fmt\"\n\nfunc f(s string) { _ = fmt.Sprintf(\"%.*s\", 4, s) }\n", "%.*s"},
		{"package p\n\nimport \"fmt\"\n\nfunc f(s string) { _ = fmt.Sprintf(\"%.2s\", s) }\n", ""},
		{"package p\n\nfunc f(s string) string { return fmt_placeholder(s) }\nfunc fmt_placeholder(s string) string { return s }\n", ""},
		{"package p\n\ntype post struct{ Text string }\n\nfunc f(p post) int { return len(p.Text) }\n", "the string p.Text with len()"},
	} {
		violations := measurements(parseSource(t, planted.src))
		if planted.want == "" {
			if len(violations) != 0 {
				t.Errorf("the scan reported %q for a source with no violation", violations)
			}
			continue
		}
		if !containsSubstring(violations, planted.want) {
			t.Errorf("the scan reported %q for a source with %s in it", violations, planted.want)
		}
	}

	// The slice arm: len() on something that is not a string is not a measurement of text.
	if violations := measurements(parseSource(t, "package p\n\nfunc f(rows []string) int { return len(rows) }\n")); len(violations) != 0 {
		t.Errorf("the scan flagged len(rows) where rows is a []string: %q", violations)
	}
}

// TestTestFilesAreNotScanned pins the skip rule the boundary depends on: a test may import whatever
// it drives the package with, and a scan that walked the test files would report those imports.
func TestTestFilesAreNotScanned(t *testing.T) {
	for _, parsed := range parsePackage(t, ".") {
		if strings.HasSuffix(parsed.name, "_test.go") {
			t.Fatalf("the scan parsed the test file %s", parsed.name)
		}
	}
	// The arm: the directory does hold test files, so the rule above is not vacuous.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading internal/tui: %v", err)
	}
	tests := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "_test.go") {
			tests++
		}
	}
	if tests == 0 {
		t.Fatalf("internal/tui holds no test files: the skip rule above proves nothing here")
	}
	// And the planted arm: the filter is the name, so a planted test file is skipped while a planted
	// source file is not.
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "p.go"), "package p\n\nimport _ \"net/http\"\n")
	writeFile(t, filepath.Join(dir, "p_test.go"), "package p\n\nimport _ \"os/exec\"\n")
	parsed := parsePackage(t, dir)
	if len(parsed) != 1 || parsed[0].name != "p.go" {
		t.Fatalf("the scan parsed %v, want only p.go", names(parsed))
	}
	if got := importPaths(parsed[0].file); len(got) != 1 || got[0] != "net/http" {
		t.Fatalf("the scan read %q out of p.go", got)
	}
}

// measurements is the A6 scanner: every second way to measure a string that one file carries, as
// sentences. It collects the file's plain string declarations first — struct fields, vars, consts
// and function results — and then reads the len() calls against them, because len() of a slice is
// not a measurement of text while len() of a string is the width bug this phase exists to prevent.
//
// Named string types are deliberately not collected: domain.PostID is a string underneath and a
// length of one is a fact about an id rather than a display decision.
func measurements(f *ast.File) []string {
	var violations []string

	for _, path := range importPaths(f) {
		if path == "unicode/utf8" {
			violations = append(violations, "imports unicode/utf8: a second measurement of a string")
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if verb := widthVerb(lit.Value); verb != "" {
				violations = append(violations, "carries the literal "+lit.Value+" with the fmt width verb "+verb+": fmt is not the program's measure")
			}
		}
		return true
	})

	names := stringNames(f)
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if fun, ok := call.Fun.(*ast.Ident); ok && fun.Name == "len" && len(call.Args) == 1 {
			declared, written := stringRef(call.Args[0])
			if names[declared] {
				violations = append(violations, "measures the string "+written+" with len(): internal/text is the only measure")
			}
		}
		return true
	})

	return violations
}

// stringNames is every name a file declares as a plain string: struct fields, vars and consts (at any
// depth), and functions or methods whose result is a string. The value of a declaration is read as
// well as its type, so `var x = "text"` and `x := "text"` count.
func stringNames(f *ast.File) map[string]bool {
	names := map[string]bool{}
	add := func(list []*ast.Ident) {
		for _, ident := range list {
			names[ident.Name] = true
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.StructType:
			for _, field := range node.Fields.List {
				if isString(field.Type) {
					add(field.Names)
				}
			}
		case *ast.ValueSpec:
			if isString(node.Type) || (node.Type == nil && declaresString(node.Values)) {
				add(node.Names)
			}
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE && declaresString(node.Rhs) {
				for _, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						names[ident.Name] = true
					}
				}
			}
		case *ast.FuncDecl:
			if results := node.Type.Results; results != nil && len(results.List) == 1 && isString(results.List[0].Type) {
				names[node.Name.Name] = true
			}
		}
		return true
	})
	return names
}

// isString reports whether a type expression is the plain string type, not a named string type and
// not a slice of strings.
func isString(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "string"
}

// declaresString reports whether the first value of a declaration is a string literal.
func declaresString(values []ast.Expr) bool {
	if len(values) == 0 {
		return false
	}
	lit, ok := values[0].(*ast.BasicLit)
	return ok && lit.Kind == token.STRING
}

// stringRef is what a len() argument names: the declaration name the scan looks up, and the
// expression as it is written, for the message. `len(p.Text)` names Text and was written p.Text;
// `len(rows())` names rows and was written rows().
func stringRef(expr ast.Expr) (declared, written string) {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name, e.Name
	case *ast.SelectorExpr:
		_, prefix := stringRef(e.X)
		return e.Sel.Name, prefix + "." + e.Sel.Name
	case *ast.CallExpr:
		declared, prefix := stringRef(e.Fun)
		return declared, prefix + "()"
	case *ast.ParenExpr:
		return stringRef(e.X)
	}
	return "", ""
}

// widthVerb returns the fmt verb a string literal carries whose width or precision is taken from an
// argument, or "". The shape is the one the rule names: a `%`, an optional argument index, the flags,
// then a width and a precision, either of which may be a `*` — a `*` in either position is a
// measurement performed by fmt rather than by internal/text. `%%` is not a verb, and a literal whose
// `%`-verbs name no width is not a width.
func widthVerb(literal string) string {
	for rest := literal; len(rest) > 0; {
		percent := strings.IndexByte(rest, '%')
		if percent < 0 {
			return ""
		}
		rest = rest[percent+1:]
		if strings.HasPrefix(rest, "%") {
			// An escaped percent is one character of text, not the start of a verb.
			rest = rest[1:]
			continue
		}
		if strings.HasPrefix(rest, "[") {
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return ""
			}
			rest = rest[end+1:]
		}
		flags := rest[:len(rest)-len(strings.TrimLeft(rest, "-+ #0"))]
		verb := "%" + flags
		rest = rest[len(flags):]
		// The width, and then the precision: digits are a fixed field and a `*` is one taken from an
		// argument, in either position — `%.*s` measures with fmt's own count exactly as `%*s` does.
		star := false
		digits := func() string {
			n := 0
			for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
				n++
			}
			out := rest[:n]
			rest = rest[n:]
			return out
		}
		if strings.HasPrefix(rest, "*") {
			star, verb, rest = true, verb+"*", rest[1:]
		} else {
			verb += digits()
		}
		if strings.HasPrefix(rest, ".") {
			verb, rest = verb+".", rest[1:]
			if strings.HasPrefix(rest, "*") {
				star, verb, rest = true, verb+"*", rest[1:]
			} else {
				verb += digits()
			}
		}
		if rest == "" {
			return ""
		}
		verb, rest = verb+rest[:1], rest[1:]
		if star {
			return verb
		}
	}
	return ""
}

// names is the file names of a parse, for an error message.
func names(files []parsedFile) []string {
	out := make([]string, 0, len(files))
	for _, parsed := range files {
		out = append(out, parsed.name)
	}
	return out
}

// containsSubstring reports whether any of the sentences carries the substring.
func containsSubstring(sentences []string, want string) bool {
	for _, sentence := range sentences {
		if strings.Contains(sentence, want) {
			return true
		}
	}
	return false
}

// writeFile plants a source file for the arms above.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

}
