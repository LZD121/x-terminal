// Package store is the project's only SQL: saved.db under the data directory,
// its three tables, ordered migrations with the version bump in the same
// transaction, a query_only reader, and a write API whose type is the whole of
// the timeline-never-reaches-disk promise.
//
// Nothing outside this package imports database/sql for the mirror. The
// exported surface is an allowlist — Open, Close, Path, Page, Status, Upsert,
// MarkAbsent, RecordRun, SetOwner, ReadFacts, plus the FileError type that
// marks a failure of the file itself — and every mutating entry takes
// []domain.SavedItem, a domain.SavedKind with an instant, a domain.SyncResult
// or an owner string. There is no exported field or function that hands out a
// database handle, runs a statement, or writes an arbitrary key and value.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	_ "modernc.org/sqlite"
)

// Store is one open saved.db: a writer pool that may write and a reader pool
// that may not. Neither handle is exported — a caller that could reach one
// could skip a pragma, open its own transaction or leak a read transaction
// that blocks WAL checkpointing.
type Store struct {
	path string
	w    *sql.DB
	r    *sql.DB
}

// Open opens path, migrates the file, and refuses a filesystem that cannot give
// WAL. On any error both pools are closed and no half-open handle is returned.
func Open(path string) (*Store, error) {
	s, err := open(path)
	if err != nil {
		// Every failure of opening is a failure of the file — it could not be opened, migrated or
		// made WAL-capable — and this is the one place that says so.
		return nil, &FileError{Err: err}
	}
	return s, nil
}

// open is Open's own body: the opening itself, whose every failure is the file's.
func open(path string) (*Store, error) {
	w, err := sql.Open("sqlite", dsn(path, ""))
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	w.SetMaxOpenConns(1)

	r, err := sql.Open("sqlite", dsn(path, "&_pragma=query_only(1)"))
	if err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}

	s := &Store{path: path, w: w, r: r}
	ctx := context.Background()
	if _, err := migrate(ctx, s.w); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}

	// journal_mode=WAL only sticks if the filesystem can provide it — on one
	// that cannot, SQLite silently keeps "delete" and every write becomes a
	// chance for "database is locked". Read the mode back instead of trusting
	// the DSN string.
	var mode string
	if err := s.w.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("store: open %s: read journal_mode: %w", path, err)
	}
	if !strings.EqualFold(mode, "wal") {
		_ = s.Close()
		return nil, fmt.Errorf("store: open %s: journal_mode is %q, want wal: the file cannot live on a filesystem that does not support WAL", path, mode)
	}
	return s, nil
}

// Close releases both pools. The file itself is left alone.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.w != nil {
		errs = append(errs, s.w.Close())
	}
	if s.r != nil {
		errs = append(errs, s.r.Close())
	}
	return errors.Join(errs...)
}

// Path returns the file this store opened.
func (s *Store) Path() string { return s.path }

// dsn is the one DSN both pools are opened with. The pragmas travel inside it
// because pragmas are per connection, not per file. _txlock=immediate is why
// every BeginTx in this package is a write-locked transaction.
func dsn(path, extra string) string {
	return "file:" + (&url.URL{Path: path}).EscapedPath() +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_txlock=immediate" +
		extra
}

// schemaV1 is the DDL of exactly three tables and no fourth. There is no
// accounts table: one account per mirror is the decision, and the owner is a
// meta row.
const schemaV1 = `
CREATE TABLE meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
CREATE TABLE saved_items (
	kind         TEXT    NOT NULL CHECK (kind IN ('bookmark','like')),
	post_id      TEXT    NOT NULL,
	sort_index   TEXT    NOT NULL,
	first_seen   INTEGER NOT NULL,
	last_seen_at INTEGER NOT NULL,
	present      INTEGER NOT NULL DEFAULT 1,
	json         TEXT    NOT NULL,
	PRIMARY KEY (kind, post_id)
);
CREATE INDEX saved_items_order ON saved_items(kind, present, sort_index DESC);
CREATE TABLE sync_runs (
	id          INTEGER PRIMARY KEY,
	kind        TEXT    NOT NULL,
	full        INTEGER NOT NULL,
	started_at  INTEGER NOT NULL,
	finished_at INTEGER,
	pages       INTEGER NOT NULL DEFAULT 0,
	items       INTEGER NOT NULL DEFAULT 0,
	absent      INTEGER NOT NULL DEFAULT 0,
	stop_reason TEXT    NOT NULL DEFAULT ''
);
`

// declaredTables names each table and its columns in declaration order. The
// schema allowlist test asserts the file against these values, so a column
// added later without a migration fails a test rather than a review.
var declaredTables = map[string][]string{
	"meta": {"key", "value"},
	"saved_items": {
		"kind", "post_id", "sort_index", "first_seen", "last_seen_at", "present", "json",
	},
	"sync_runs": {
		"id", "kind", "full", "started_at", "finished_at", "pages", "items", "absent", "stop_reason",
	},
}

// declaredMetaKeys is the closed set of meta keys this build writes.
var declaredMetaKeys = []string{
	"schema_version",
	"owner",
	"last_success_bookmark",
	"last_success_like",
}
