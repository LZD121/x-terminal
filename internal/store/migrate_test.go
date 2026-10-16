package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

func TestMigrateFreshAndIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	db := rawOpen(t, path)
	ctx := context.Background()

	applied, err := migrate(ctx, db)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if applied != 1 {
		t.Fatalf("first Migrate applied %d, want 1", applied)
	}
	ver, err := schemaVersion(ctx, db)
	if err != nil || ver != 1 {
		t.Fatalf("version after first = (%d, %v), want (1, nil)", ver, err)
	}

	applied, err = migrate(ctx, db)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if applied != 0 {
		t.Fatalf("second Migrate applied %d, want 0", applied)
	}
	ver, err = schemaVersion(ctx, db)
	if err != nil || ver != 1 {
		t.Fatalf("version after second = (%d, %v), want (1, nil)", ver, err)
	}
}

func TestMigrateRefusesNewerSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	db := rawOpen(t, path)
	ctx := context.Background()
	if _, err := migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO saved_items(kind, post_id, sort_index, first_seen, last_seen_at, present, json)
		VALUES('bookmark','keep','z',1,1,1,'{}')`); err != nil {
		t.Fatalf("seed row: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE meta SET value = '99' WHERE key = 'schema_version'`); err != nil {
		t.Fatalf("set version 99: %v", err)
	}

	applied, err := migrate(ctx, db)
	if !errors.Is(err, ErrSchemaNewer) {
		t.Fatalf("Migrate against newer = (%d, %v), want ErrSchemaNewer", applied, err)
	}
	if applied != 0 {
		t.Errorf("applied %d, want 0", applied)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM saved_items`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("rows after refuse = %d (%v), want 1 untouched", n, err)
	}
}

func TestMigrateTornUpgradeRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	db := rawOpen(t, path)
	ctx := context.Background()
	if _, err := migrate(ctx, db); err != nil {
		t.Fatalf("Migrate v1: %v", err)
	}
	seen := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	s := &Store{path: path, w: db, r: db}
	if _, err := s.Upsert(ctx, []domain.SavedItem{{
		Kind: domain.Bookmark, Post: samplePost("keep", "aergon", "intact"),
		SortIndex: "z", FirstSeen: seen, LastSeen: seen, Present: true,
	}}); err != nil {
		t.Fatalf("seed Upsert: %v", err)
	}

	old := migrations
	migrations = append(append([]migration{}, old...), migration{
		version: 2,
		ddl: `
CREATE TABLE torn_probe (id INTEGER PRIMARY KEY);
CREATE TABLE torn_probe (id INTEGER PRIMARY KEY);
`,
	})
	defer func() { migrations = old }()

	applied, err := migrate(ctx, db)
	if err == nil {
		t.Fatal("Migrate with a failing migration succeeded")
	}
	if applied != 0 {
		t.Errorf("applied %d before the failure, want 0 committed", applied)
	}
	ver, err := schemaVersion(ctx, db)
	if err != nil || ver != 1 {
		t.Fatalf("version after torn = (%d, %v), want (1, nil)", ver, err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM saved_items WHERE post_id = 'keep'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("seed row after torn = %d (%v), want 1", n, err)
	}
	var probe string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE name = 'torn_probe'`).Scan(&probe)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("torn_probe after failed migration = %q (%v), want absent", probe, err)
	}
}

func TestMigrateKeepsRowsAcrossUpgrade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.db")
	db := rawOpen(t, path)
	ctx := context.Background()
	if _, err := migrate(ctx, db); err != nil {
		t.Fatalf("Migrate v1: %v", err)
	}
	seen := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	s := &Store{path: path, w: db, r: db}
	if _, err := s.Upsert(ctx, []domain.SavedItem{{
		Kind: domain.Like, Post: samplePost("42", "mira", "survives"),
		SortIndex: "y", FirstSeen: seen, LastSeen: seen, Present: true,
	}}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	old := migrations
	migrations = append(append([]migration{}, old...), migration{
		version: 2,
		ddl:     `CREATE TABLE upgrade_marker (id INTEGER PRIMARY KEY);`,
	})
	defer func() { migrations = old }()

	applied, err := migrate(ctx, db)
	if err != nil {
		t.Fatalf("upgrade Migrate: %v", err)
	}
	if applied != 1 {
		t.Fatalf("upgrade applied %d, want 1", applied)
	}
	ver, err := schemaVersion(ctx, db)
	if err != nil || ver != 2 {
		t.Fatalf("version = (%d, %v), want 2", ver, err)
	}
	var text string
	if err := db.QueryRow(`SELECT json FROM saved_items WHERE post_id = '42'`).Scan(&text); err != nil {
		t.Fatalf("row missing after upgrade: %v", err)
	}
	if !strings.Contains(text, "survives") {
		t.Fatalf("row json after upgrade = %q", text)
	}
}
