package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

// ErrSchemaNewer refuses to migrate a database written by a newer build: the
// file may hold tables and columns this build knows nothing about, and running
// old DDL against it is a downgrade, not an upgrade.
var ErrSchemaNewer = errors.New("store: database schema is newer than this build")

// migration is one version of the schema, applied whole or not at all.
type migration struct {
	version int
	ddl     string
}

// migrations is the whole history, oldest first; the last entry is the newest
// version this build knows. Appended to (never edited) when a later build adds one.
var migrations = []migration{
	{version: 1, ddl: schemaV1},
}

// migrate applies every migration newer than the version recorded in
// meta.schema_version and returns how many it applied — 1 on a fresh file, 0 on
// an already-migrated one, which is what makes idempotence assertable rather
// than inferred from the absence of an error.
//
// Unexported on purpose: Open is the only exported entry that opens a writable
// file, and the package's exported surface is the ten-name allowlist (Open,
// Close, Path, Page, Status, Upsert, MarkAbsent, RecordRun, SetOwner,
// ReadFacts) plus the FileError type.
func migrate(ctx context.Context, db *sql.DB) (int, error) {
	current, err := schemaVersion(ctx, db)
	if err != nil {
		return 0, err
	}
	newest := migrations[len(migrations)-1].version
	if current > newest {
		return 0, fmt.Errorf("%w: file is at version %d, this build knows %d", ErrSchemaNewer, current, newest)
	}
	applied := 0
	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return applied, err
		}
		applied++
	}
	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, m migration) error {
	// BEGIN IMMEDIATE comes from _txlock=immediate on the DSN — this is not a
	// second locking mechanism.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: migrate %d: begin: %w", m.version, err)
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, m.ddl); err != nil {
		return fmt.Errorf("store: migrate %d: %w", m.version, err)
	}
	// Recorded in the same transaction as the DDL: one version per commit.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO meta(key, value) VALUES('schema_version', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		strconv.Itoa(m.version)); err != nil {
		return fmt.Errorf("store: migrate %d: record version: %w", m.version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: migrate %d: commit: %w", m.version, err)
	}
	return nil
}

// schemaVersion reports the version recorded in meta, or -1 when there is no
// meta table or no schema_version row — which is what a fresh file looks like.
// sqlite_master is consulted first because reading meta on a file that has never
// been migrated is a "no such table" error, not an empty result.
func schemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'meta'`).Scan(&name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return -1, nil
	case err != nil:
		return 0, fmt.Errorf("store: read schema version: %w", err)
	}
	var value string
	err = db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&value)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return -1, nil
	case err != nil:
		return 0, fmt.Errorf("store: read schema version: %w", err)
	}
	version, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("store: schema_version %q: %w", value, err)
	}
	return version, nil
}
