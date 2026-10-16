package store

import (
	"database/sql"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestSchemaAllowlist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	got := readSchema(t, s.r)
	if err := compareSchema(got, declaredTables); err != nil {
		t.Fatalf("clean file: %v", err)
	}

	// Anti-vacuity: a scan that found nothing must not pass.
	if len(got.tables) < 3 {
		t.Fatalf("allowlist found %d tables, want at least 3", len(got.tables))
	}
	cols := 0
	for _, c := range got.columns {
		cols += len(c)
	}
	if cols < 10 {
		t.Fatalf("allowlist found %d columns, want at least 10", cols)
	}

	keys := readMetaKeys(t, s.r)
	for _, k := range keys {
		if !slices.Contains(declaredMetaKeys, k) {
			t.Errorf("meta key %q is not in declaredMetaKeys", k)
		}
	}
	if !slices.Contains(keys, "schema_version") {
		t.Fatal("fresh file must hold schema_version")
	}

	for _, cols := range got.columns {
		for _, name := range cols {
			switch strings.ToLower(name) {
			case "blob", "payload", "extra", "data":
				t.Errorf("generic container column %q is forbidden", name)
			}
		}
	}
}

// TestExportedSurfaceAllowlist keeps the store's public names a list rather
// than a habit. ReadFacts sits beside the mutators; shrinking the surface to
// one function must still fail the anti-vacuity arm. FileError is the one
// exported type: it carries no behaviour, it marks a failure of the file the
// package owns.
func TestExportedSurfaceAllowlist(t *testing.T) {
	must := []string{
		"Open", "Close", "Path", "Page", "Status",
		"Upsert", "MarkAbsent", "RecordRun", "SetOwner", "ReadFacts",
		"FileError",
	}
	got := exportedStoreNames(t)
	for _, name := range must {
		if !slices.Contains(got, name) {
			t.Errorf("exported surface missing %q (have %v)", name, got)
		}
	}
	mutators := 0
	for _, name := range got {
		switch name {
		case "Upsert", "MarkAbsent", "RecordRun", "SetOwner":
			mutators++
		}
	}
	if mutators < 4 {
		t.Fatalf("anti-vacuity: found %d mutators, want the four write APIs", mutators)
	}
	if !slices.Contains(got, "ReadFacts") {
		t.Fatal("ReadFacts missing from the exported-surface allowlist")
	}
}

func exportedStoreNames(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		name := info.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse store: %v", err)
	}
	pkg := pkgs["store"]
	if pkg == nil {
		t.Fatal("package store not found")
	}
	var names []string
	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.IsExported() {
					names = append(names, d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.IsExported() {
						names = append(names, ts.Name.Name)
					}
				}
			}
		}
	}
	return names
}

func TestSchemaAllowlistFailsOnExtraTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if _, err := s.w.Exec(`CREATE TABLE sneak (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("plant extra table: %v", err)
	}
	got := readSchema(t, s.r)
	if err := compareSchema(got, declaredTables); err == nil {
		t.Fatal("compareSchema accepted a file with an extra table; the comparator is not exercised")
	}
}

func TestSchemaAllowlistFailsOnExtraColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if _, err := s.w.Exec(`ALTER TABLE meta ADD COLUMN payload TEXT`); err != nil {
		t.Fatalf("plant extra column: %v", err)
	}
	got := readSchema(t, s.r)
	if err := compareSchema(got, declaredTables); err == nil {
		t.Fatal("compareSchema accepted a file with an extra column; the comparator is not exercised")
	}
}

type fileSchema struct {
	tables  []string
	columns map[string][]string
	notnull map[string][]bool
}

func readSchema(t *testing.T, db *sql.DB) fileSchema {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type IN ('table','index') AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatalf("sqlite_master: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	out := fileSchema{columns: map[string][]string{}, notnull: map[string][]bool{}}
	for _, name := range names {
		var typ string
		if err := db.QueryRow(`SELECT type FROM sqlite_master WHERE name = ?`, name).Scan(&typ); err != nil {
			t.Fatalf("type of %s: %v", name, err)
		}
		if typ != "table" {
			continue
		}
		out.tables = append(out.tables, name)
		info, err := db.Query(`PRAGMA table_info(` + name + `)`)
		if err != nil {
			t.Fatalf("table_info(%s): %v", name, err)
		}
		for info.Next() {
			var (
				cid     int
				cname   string
				ctype   string
				notnull int
				dflt    sql.NullString
				pk      int
			)
			if err := info.Scan(&cid, &cname, &ctype, &notnull, &dflt, &pk); err != nil {
				t.Fatalf("table_info scan: %v", err)
			}
			out.columns[name] = append(out.columns[name], cname)
			out.notnull[name] = append(out.notnull[name], notnull != 0)
		}
		_ = info.Close()
	}
	sort.Strings(out.tables)
	return out
}

func compareSchema(got fileSchema, want map[string][]string) error {
	wantTables := make([]string, 0, len(want))
	for name := range want {
		wantTables = append(wantTables, name)
	}
	sort.Strings(wantTables)
	if !slices.Equal(got.tables, wantTables) {
		return fmt.Errorf("tables = %v, want %v", got.tables, wantTables)
	}
	for _, table := range wantTables {
		if !slices.Equal(got.columns[table], want[table]) {
			return fmt.Errorf("columns of %s = %v, want %v", table, got.columns[table], want[table])
		}
	}
	return nil
}

func readMetaKeys(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT key FROM meta ORDER BY key`)
	if err != nil {
		t.Fatalf("meta keys: %v", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			t.Fatalf("scan: %v", err)
		}
		keys = append(keys, k)
	}
	return keys
}
