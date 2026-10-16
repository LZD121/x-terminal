package store

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
	"github.com/LZD121/x-terminal/internal/screen"
)

const storeImport = "github.com/LZD121/x-terminal/internal/store"

// TestBoundaryImportScan proves no home/thread/search surface imports
// internal/store. The arm is a glob over the files that exist when it runs.
func TestBoundaryImportScan(t *testing.T) {
	root := moduleRoot(t)
	files := timelineSurfaces(t, root)
	if len(files) == 0 {
		t.Fatal("timeline surface scan found no files — the scan is vacuous")
	}

	var importCount int
	var violations []string
	for _, path := range files {
		f := parseFile(t, path)
		for _, imp := range importPaths(f) {
			importCount++
			if imp == storeImport || strings.HasPrefix(imp, storeImport+"/") {
				violations = append(violations, rel(root, path)+" imports "+imp)
			}
		}
	}
	if importCount == 0 {
		t.Fatalf("collected no imports from %d files — the scan is vacuous", len(files))
	}
	if len(violations) > 0 {
		t.Fatalf("timeline surfaces import internal/store:\n  %s", strings.Join(violations, "\n  "))
	}

	// Positive control: internal/store itself must import database/sql.
	storeFiles := nonTestGoFiles(t, filepath.Join(root, "internal", "store"))
	var sawSQL bool
	for _, path := range storeFiles {
		for _, imp := range importPaths(parseFile(t, path)) {
			if imp == "database/sql" {
				sawSQL = true
			}
		}
	}
	if !sawSQL {
		t.Fatal("internal/store does not import database/sql — the scan is not looking at real imports")
	}

	// Positive control: internal/tui still imports bubbletea.
	tuiFiles := nonTestGoFiles(t, filepath.Join(root, "internal", "tui"))
	var sawTea bool
	for _, path := range tuiFiles {
		for _, imp := range importPaths(parseFile(t, path)) {
			if imp == "charm.land/bubbletea/v2" {
				sawTea = true
			}
		}
	}
	if !sawTea {
		t.Fatal("internal/tui does not import charm.land/bubbletea/v2 — the scan is not looking at the package")
	}

	// Planted violation: a synthetic source that imports internal/store must be reported.
	planted := parseSource(t, `package planted
import (
	_ "github.com/LZD121/x-terminal/internal/store"
)
`)
	if !fileImportsStore(planted) {
		t.Fatal("planted source imports internal/store but the scanner did not report it")
	}

	// Plant into a real timeline fixture file and assert the same scan reports it.
	fixture := filepath.Join(root, "internal", "xsource", "home.go")
	dir := plantStoreImport(t, fixture)
	plantedFiles, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(plantedFiles) == 0 {
		t.Fatalf("planted fixture missing: %v", err)
	}
	var found bool
	for _, path := range plantedFiles {
		if fileImportsStore(parseFile(t, path)) {
			found = true
		}
	}
	if !found {
		t.Fatal("planted home.go fixture imports internal/store but the scanner did not report it")
	}

	t.Logf("scanned %d timeline files, %d imports; store files=%d; planted violation detected", len(files), importCount, len(storeFiles))
}

// TestBoundaryDataDirAfterTimelineSession drives a full fake timeline session and
// asserts the data directory holds only what was there before plus saved.db and
// its WAL siblings, with saved_items empty.
func TestBoundaryDataDirAfterTimelineSession(t *testing.T) {
	dir := t.TempDir()
	// Pre-existing files the timeline must not disturb.
	if err := os.WriteFile(filepath.Join(dir, "pins.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatalf("pins: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notice"), []byte("shown"), 0o600); err != nil {
		t.Fatalf("notice: %v", err)
	}
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir before: %v", err)
	}
	beforeNames := map[string]bool{}
	for _, e := range before {
		beforeNames[e.Name()] = true
	}

	// Open the store beside the pre-existing files (what cli will do).
	dbPath := filepath.Join(dir, "saved.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	// Drive a full timeline session through fake + screen reducers — no write
	// path from the timeline into the store exists, which is the property.
	src := fake.Adversarial()
	list := screen.List{Kind: domain.Following}
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	var cursor domain.Cursor
	for i := range 8 {
		page, err := src.Home(context.Background(), domain.Following, cursor)
		if err != nil {
			t.Fatalf("Home page %d: %v", i, err)
		}
		var effect screen.Effect
		list, effect = list.Update(screen.PageFetched{Page: page}, now)
		_ = effect
		if page.Next == "" {
			break
		}
		cursor = page.Next
		list, _ = list.Update(screen.PageRequested{}, now)
	}
	if list.Loaded() == 0 {
		t.Fatal("timeline session loaded no posts — the inspection would be vacuous")
	}

	// Refresh once and absorb a failure too, so both reducers ran.
	list, _ = list.Update(screen.RefreshRequested{}, now)
	page, err := src.Home(context.Background(), domain.Following, "")
	if err != nil {
		t.Fatalf("refresh Home: %v", err)
	}
	list, _ = list.Update(screen.Refreshed{Page: page}, now)
	list, _ = list.Update(screen.PageFailed{Failure: domain.Failure{Kind: domain.FailureNoRoute, Detail: "no route"}}, now)
	_ = list

	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir after: %v", err)
	}
	for _, e := range after {
		name := e.Name()
		if beforeNames[name] {
			continue
		}
		switch name {
		case "saved.db", "saved.db-wal", "saved.db-shm":
			// allowed siblings of the mirror
		default:
			t.Errorf("data dir gained unexpected entry %q after a timeline session", name)
		}
	}
	for name := range beforeNames {
		found := false
		for _, e := range after {
			if e.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pre-existing %q disappeared after the session", name)
		}
	}

	var n int
	if err := s.r.QueryRow(`SELECT count(*) FROM saved_items`).Scan(&n); err != nil {
		t.Fatalf("count saved_items: %v", err)
	}
	if n != 0 {
		t.Fatalf("saved_items holds %d rows after a timeline session; want 0 — the timeline reached disk", n)
	}
}

func timelineSurfaces(t *testing.T, root string) []string {
	t.Helper()
	var out []string

	// internal/xsource: home/tweets/user/detail/search (+ profile is also a
	// network surface that must stay out of the store).
	xsourceDir := filepath.Join(root, "internal", "xsource")
	for _, path := range nonTestGoFiles(t, xsourceDir) {
		base := filepath.Base(path)
		switch {
		case strings.HasPrefix(base, "home"),
			strings.HasPrefix(base, "tweets"),
			strings.HasPrefix(base, "user"),
			strings.HasPrefix(base, "detail"),
			strings.HasPrefix(base, "search"),
			strings.HasPrefix(base, "profile"):
			out = append(out, path)
		}
	}

	// internal/screen: list/absorb/effect/thread/search (+ walk, profile, prompt).
	screenDir := filepath.Join(root, "internal", "screen")
	for _, path := range nonTestGoFiles(t, screenDir) {
		base := filepath.Base(path)
		switch {
		case strings.HasPrefix(base, "list"),
			strings.HasPrefix(base, "absorb"),
			strings.HasPrefix(base, "effect"),
			strings.HasPrefix(base, "thread"),
			strings.HasPrefix(base, "search"),
			strings.HasPrefix(base, "walk"),
			strings.HasPrefix(base, "profile"),
			strings.HasPrefix(base, "prompt"):
			out = append(out, path)
		}
	}

	// Every non-test file of internal/render and internal/tui.
	for _, pkg := range []string{"render", "tui"} {
		out = append(out, nonTestGoFiles(t, filepath.Join(root, "internal", pkg))...)
	}
	return out
}

func nonTestGoFiles(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	var out []string
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		out = append(out, path)
	}
	return out
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getcwd: %v", err)
	}
	// Tests run with the package directory as cwd.
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above package dir")
		}
		dir = parent
	}
}

func parseFile(t *testing.T, path string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return f
}

func parseSource(t *testing.T, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "planted.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("planted source does not parse: %v", err)
	}
	return f
}

func importPaths(f *ast.File) []string {
	out := make([]string, 0, len(f.Imports))
	for _, spec := range f.Imports {
		out = append(out, strings.Trim(spec.Path.Value, `"`))
	}
	return out
}

func fileImportsStore(f *ast.File) bool {
	for _, imp := range importPaths(f) {
		if imp == storeImport || strings.HasPrefix(imp, storeImport+"/") {
			return true
		}
	}
	return false
}

func plantStoreImport(t *testing.T, srcPath string) string {
	t.Helper()
	body, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read %s: %v", srcPath, err)
	}
	at := strings.Index(string(body), "import (")
	if at < 0 {
		t.Fatalf("%s has no import block", srcPath)
	}
	at += len("import (")
	planted := string(body)[:at] + "\n\t_ \"" + storeImport + "\"\n" + string(body)[at:]
	dir := t.TempDir()
	dst := filepath.Join(dir, filepath.Base(srcPath))
	if err := os.WriteFile(dst, []byte(planted), 0o600); err != nil {
		t.Fatalf("write planted: %v", err)
	}
	return dir
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return r
}
