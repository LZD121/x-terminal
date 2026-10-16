package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// PageSize is the mirror's page size — the network page size — so position means
// one thing on every list.
const PageSize = 20

// now is the caller's instant for FetchedAt; tests freeze it.
var now = time.Now

// FileError marks a failure of the store's own operation on the mirror file: saved.db could not be
// opened, migrated or read.
//
// It exists because the walk's failures are not all the file's. A page the walk fetched and could
// not read (internal/xsource answers a 200 whose shape drifted with its own *ParseError), a
// classified answer, a spent request cap, a missing session and a transport that never left this
// machine are all facts about a request, and they travel back through this package untouched:
// nothing here wraps them, so nothing here claims them. This type is the one thing that does claim
// the file — errors.As(err, &FileError{}) is true exactly when the store's own file operation is
// what failed — and it is what a caller maps onto "the mirror cannot be read" states.
//
// Error reports the wrapped error's own message, so every surface that prints a store error is
// unchanged; Unwrap keeps the store's own sentinels and driver errors reachable through errors.Is
// and errors.As.
type FileError struct{ Err error }

func (e *FileError) Error() string { return e.Err.Error() }

func (e *FileError) Unwrap() error { return e.Err }

// Page reads present = 1 rows of kind, ordered sort_index DESC, keyed by the
// cursor (sort_index < ? when given), limited to PageSize. Next is the last
// row's sort_index when the page is full and older rows exist, empty otherwise.
func (s *Store) Page(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	if !kind.Valid() {
		return domain.PostPage{}, fmt.Errorf("store: page: invalid kind %d", kind)
	}
	page, err := s.page(ctx, kind, c)
	if err != nil {
		// Every failure of the read is a failure of the file, and this is the one place that says
		// so. A kind this build does not have is the caller's mistake rather than the file's, which
		// is why it is refused above instead of here.
		return domain.PostPage{}, &FileError{Err: err}
	}
	return page, nil
}

// page is Page's own body: the read of the file, whose every failure is the file's.
func (s *Store) page(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if c == "" {
		rows, err = s.r.QueryContext(ctx, `
			SELECT sort_index, json FROM saved_items
			WHERE kind = ? AND present = 1
			ORDER BY sort_index DESC
			LIMIT ?`, kind.String(), PageSize+1)
	} else {
		rows, err = s.r.QueryContext(ctx, `
			SELECT sort_index, json FROM saved_items
			WHERE kind = ? AND present = 1 AND sort_index < ?
			ORDER BY sort_index DESC
			LIMIT ?`, kind.String(), string(c), PageSize+1)
	}
	if err != nil {
		return domain.PostPage{}, fmt.Errorf("store: page: %w", err)
	}
	defer rows.Close()

	type row struct {
		sortIndex string
		body      string
	}
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.sortIndex, &r.body); err != nil {
			return domain.PostPage{}, fmt.Errorf("store: page scan: %w", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		return domain.PostPage{}, fmt.Errorf("store: page rows: %w", err)
	}

	hasMore := len(got) > PageSize
	if hasMore {
		got = got[:PageSize]
	}
	page := domain.PostPage{
		Posts:     make([]domain.Post, 0, len(got)),
		Requested: len(got),
		FetchedAt: now().UTC(),
	}
	for _, r := range got {
		var snap snapshot
		if err := json.Unmarshal([]byte(r.body), &snap); err != nil {
			return domain.PostPage{}, fmt.Errorf("store: page decode: %w", err)
		}
		page.Posts = append(page.Posts, snap.post())
	}
	if hasMore && len(got) > 0 {
		page.Next = domain.Cursor(got[len(got)-1].sortIndex)
	}
	return page, nil
}

// Status returns the owner and that kind's last success from meta, plus present
// and absent counts. Reads go through the reader pool; nothing in this file writes.
func (s *Store) Status(ctx context.Context, kind domain.SavedKind) (domain.MirrorStatus, error) {
	if !kind.Valid() {
		return domain.MirrorStatus{}, fmt.Errorf("store: status: invalid kind %d", kind)
	}
	st, err := s.status(ctx, kind)
	if err != nil {
		// Every failure of the read is a failure of the file, said in one place; an invalid kind is
		// refused above rather than marked here.
		return domain.MirrorStatus{}, &FileError{Err: err}
	}
	return st, nil
}

// status is Status's own body: the read of the file, whose every failure is the file's.
func (s *Store) status(ctx context.Context, kind domain.SavedKind) (domain.MirrorStatus, error) {
	var st domain.MirrorStatus
	err := s.r.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'owner'`).Scan(&st.Owner)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		st.Owner = ""
	case err != nil:
		return domain.MirrorStatus{}, fmt.Errorf("store: status owner: %w", err)
	}

	var stamp string
	err = s.r.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, lastSuccessKey(kind)).Scan(&stamp)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// zero LastSuccess
	case err != nil:
		return domain.MirrorStatus{}, fmt.Errorf("store: status last success: %w", err)
	default:
		var unix int64
		if _, scanErr := fmt.Sscanf(stamp, "%d", &unix); scanErr == nil && unix > 0 {
			st.LastSuccess = time.Unix(unix, 0).UTC()
		}
	}

	if err := s.r.QueryRowContext(ctx,
		`SELECT count(*) FROM saved_items WHERE kind = ? AND present = 1`, kind.String(),
	).Scan(&st.Present); err != nil {
		return domain.MirrorStatus{}, fmt.Errorf("store: status present: %w", err)
	}
	if err := s.r.QueryRowContext(ctx,
		`SELECT count(*) FROM saved_items WHERE kind = ? AND present = 0`, kind.String(),
	).Scan(&st.Absent); err != nil {
		return domain.MirrorStatus{}, fmt.Errorf("store: status absent: %w", err)
	}
	return st, nil
}

// ReadFacts opens path read-only and reports the closed set of facts the
// doctor's mirror line may print. It opens one connection with mode=ro plus
// the query_only pragma — no writer pool, no migration, no journal_mode
// read-back — because a diagnostic that repairs by accident is a diagnostic
// nobody can trust.
//
// A missing file is Exists=false with a nil error: a presence fact, not a
// failure. A file whose schema_version is newer than this build knows returns
// Newer=true with the version and no error. Anything else unreadable is a
// named error carrying the path and no row contents.
func ReadFacts(path string) (domain.MirrorFacts, error) {
	facts := domain.MirrorFacts{Path: path}
	if path == "" {
		return facts, nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return facts, nil
		}
		return facts, fmt.Errorf("store: read facts %s: %w", path, err)
	}
	facts.Exists = true

	db, err := sql.Open("sqlite", factsDSN(path))
	if err != nil {
		return facts, fmt.Errorf("store: read facts %s: %w", path, err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	version, err := schemaVersion(ctx, db)
	if err != nil {
		return facts, fmt.Errorf("store: read facts %s: %w", path, err)
	}
	if version < 0 {
		return facts, fmt.Errorf("store: read facts %s: no schema_version", path)
	}
	facts.SchemaVersion = version
	newest := migrations[len(migrations)-1].version
	if version > newest {
		facts.Newer = true
		return facts, nil
	}

	var owner string
	err = db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'owner'`).Scan(&owner)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		facts.Owner = ""
	case err != nil:
		return facts, fmt.Errorf("store: read facts %s: owner: %w", path, err)
	default:
		facts.Owner = owner
	}

	kinds := make([]domain.MirrorKindCount, 0, 2)
	for _, kind := range []domain.SavedKind{domain.Bookmark, domain.Like} {
		var present, absent int
		if err := db.QueryRowContext(ctx,
			`SELECT count(*) FROM saved_items WHERE kind = ? AND present = 1`, kind.String(),
		).Scan(&present); err != nil {
			return facts, fmt.Errorf("store: read facts %s: present %s: %w", path, kind, err)
		}
		if err := db.QueryRowContext(ctx,
			`SELECT count(*) FROM saved_items WHERE kind = ? AND present = 0`, kind.String(),
		).Scan(&absent); err != nil {
			return facts, fmt.Errorf("store: read facts %s: absent %s: %w", path, kind, err)
		}
		kinds = append(kinds, domain.MirrorKindCount{Kind: kind, Present: present, Absent: absent})
	}
	facts.Kinds = kinds

	var finished sql.NullInt64
	var stop string
	err = db.QueryRowContext(ctx,
		`SELECT finished_at, stop_reason FROM sync_runs ORDER BY id DESC LIMIT 1`,
	).Scan(&finished, &stop)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// never ran
	case err != nil:
		return facts, fmt.Errorf("store: read facts %s: last run: %w", path, err)
	default:
		if finished.Valid && finished.Int64 > 0 {
			facts.LastRun = time.Unix(finished.Int64, 0).UTC()
		}
		facts.Stop = parseStopReason(stop)
	}
	return facts, nil
}

// factsDSN is the read-only DSN ReadFacts alone uses: mode=ro and query_only,
// with none of Open's write-side pragmas. It must not migrate and must not be
// able to write.
func factsDSN(path string) string {
	return "file:" + (&url.URL{Path: path}).EscapedPath() +
		"?mode=ro" +
		"&_pragma=query_only(1)" +
		"&_pragma=busy_timeout(5000)"
}

func parseStopReason(s string) domain.StopReason {
	for _, r := range []domain.StopReason{
		domain.StopNoUnknownIDs,
		domain.StopEndOfList,
		domain.StopPageCap,
		domain.StopFailure,
		domain.StopAbandoned,
		domain.StopStalled,
	} {
		if r.String() == s {
			return r
		}
	}
	return domain.StopNone
}
