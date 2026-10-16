package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// ErrOwnerConflict refuses to overwrite a different non-empty owner. The mirror
// is one account's; re-pointing it is deleting the file.
var ErrOwnerConflict = errors.New("store: mirror owner conflict")

// Upsert writes items in one transaction. first_seen is written only on insert;
// a conflict keeps it, moves last_seen_at and sort_index, restores present = 1,
// and rewrites the JSON. The returned count is how many ids the mirror did not
// already hold for that kind — the number the incremental walk stops on.
func (s *Store) Upsert(ctx context.Context, items []domain.SavedItem) (added int, err error) {
	if len(items) == 0 {
		return 0, nil
	}
	tx, err := s.w.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("store: upsert begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO saved_items(kind, post_id, sort_index, first_seen, last_seen_at, present, json)
		VALUES(?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT(kind, post_id) DO UPDATE SET
			last_seen_at = excluded.last_seen_at,
			sort_index   = excluded.sort_index,
			present      = 1,
			json         = excluded.json`)
	if err != nil {
		return 0, fmt.Errorf("store: upsert prepare: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		if !item.Kind.Valid() {
			return 0, fmt.Errorf("store: upsert: invalid kind %d", item.Kind)
		}
		var exists int
		err := tx.QueryRowContext(ctx,
			`SELECT 1 FROM saved_items WHERE kind = ? AND post_id = ?`,
			item.Kind.String(), string(item.Post.ID)).Scan(&exists)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			added++
		case err != nil:
			return 0, fmt.Errorf("store: upsert lookup: %w", err)
		}

		body, err := json.Marshal(snapshotOf(item.Post))
		if err != nil {
			return 0, fmt.Errorf("store: upsert marshal: %w", err)
		}
		if _, err := stmt.ExecContext(ctx,
			item.Kind.String(),
			string(item.Post.ID),
			item.SortIndex,
			item.FirstSeen.Unix(),
			item.LastSeen.Unix(),
			string(body),
		); err != nil {
			return 0, fmt.Errorf("store: upsert exec: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("store: upsert commit: %w", err)
	}
	return added, nil
}

// MarkAbsent marks present = 1 rows of kind whose last_seen_at is before the
// run's start. A row is never deleted: an un-bookmark is "gone from x.com", not
// history quietly disappearing.
func (s *Store) MarkAbsent(ctx context.Context, kind domain.SavedKind, before time.Time) (int64, error) {
	if !kind.Valid() {
		return 0, fmt.Errorf("store: mark absent: invalid kind %d", kind)
	}
	res, err := s.w.ExecContext(ctx,
		`UPDATE saved_items SET present = 0 WHERE kind = ? AND present = 1 AND last_seen_at < ?`,
		kind.String(), before.Unix())
	if err != nil {
		return 0, fmt.Errorf("store: mark absent: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: mark absent rows: %w", err)
	}
	return n, nil
}

// RecordRun inserts one sync_runs row and — in the same transaction, only when
// the stop is a successful one — writes that kind's last-success meta row.
func (s *Store) RecordRun(ctx context.Context, run domain.SyncResult) error {
	if !run.Kind.Valid() {
		return fmt.Errorf("store: record run: invalid kind %d", run.Kind)
	}
	tx, err := s.w.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: record run begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var full int
	if run.Full {
		full = 1
	}
	var finished any
	if !run.Finished.IsZero() {
		finished = run.Finished.Unix()
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sync_runs(kind, full, started_at, finished_at, pages, items, absent, stop_reason)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		run.Kind.String(), full, run.Started.Unix(), finished,
		run.Pages, run.Added, run.Absent, run.Stop.String(),
	); err != nil {
		return fmt.Errorf("store: record run insert: %w", err)
	}

	if isSuccessfulStop(run.Stop) {
		key := lastSuccessKey(run.Kind)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO meta(key, value) VALUES(?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			key, fmt.Sprintf("%d", run.Finished.Unix()),
		); err != nil {
			return fmt.Errorf("store: record run last success: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: record run commit: %w", err)
	}
	return nil
}

func isSuccessfulStop(stop domain.StopReason) bool {
	switch stop {
	case domain.StopNoUnknownIDs, domain.StopEndOfList, domain.StopPageCap:
		return true
	default:
		return false
	}
}

func lastSuccessKey(kind domain.SavedKind) string {
	switch kind {
	case domain.Like:
		return "last_success_like"
	default:
		return "last_success_bookmark"
	}
}

// SetOwner writes the meta row owner, refusing to overwrite a different
// non-empty owner and returning an error that carries both handles when it refuses.
func (s *Store) SetOwner(ctx context.Context, owner string) error {
	var current string
	err := s.w.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'owner'`).Scan(&current)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// first write
	case err != nil:
		return fmt.Errorf("store: set owner read: %w", err)
	default:
		if current != "" && current != owner {
			return fmt.Errorf("%w: mirror belongs to %q, session is %q", ErrOwnerConflict, current, owner)
		}
		if current == owner {
			return nil
		}
	}
	if _, err := s.w.ExecContext(ctx, `
		INSERT INTO meta(key, value) VALUES('owner', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, owner); err != nil {
		return fmt.Errorf("store: set owner: %w", err)
	}
	return nil
}

// snapshot is the render-ready post shape stored in saved_items.json. Engagement
// counts are deliberately not stored; the author's account id is deliberately
// stored so an offline read under a cover can alias by id.
type snapshot struct {
	ID             string      `json:"id"`
	Author         snapUser    `json:"author"`
	Text           string      `json:"text"`
	CreatedAt      int64       `json:"created_at"`
	RepostedBy     *snapUser   `json:"reposted_by,omitempty"`
	Lang           string      `json:"lang,omitempty"`
	ConversationID string      `json:"conversation_id,omitempty"`
	ReplyTo        string      `json:"reply_to,omitempty"`
	Quote          *snapshot   `json:"quote,omitempty"`
	Media          []snapMedia `json:"media,omitempty"`
	Visibility     string      `json:"visibility"`
}

type snapUser struct {
	ID         string `json:"id"`
	ScreenName string `json:"screen_name"`
	Name       string `json:"name"`
}

type snapMedia struct {
	Kind    string `json:"kind"`
	URL     string `json:"url"`
	AltText string `json:"alt_text,omitempty"`
}

func snapshotOf(p domain.Post) snapshot {
	s := snapshot{
		ID:             string(p.ID),
		Author:         snapUserOf(p.Author),
		Text:           p.Text,
		CreatedAt:      p.CreatedAt.Unix(),
		Lang:           p.Lang,
		ConversationID: p.ConversationID,
		ReplyTo:        string(p.ReplyTo),
		Visibility:     p.Visibility.String(),
	}
	if p.RepostedBy != nil {
		u := snapUserOf(*p.RepostedBy)
		s.RepostedBy = &u
	}
	if p.Quote != nil {
		q := snapshotOf(*p.Quote)
		s.Quote = &q
	}
	if len(p.Media) > 0 {
		s.Media = make([]snapMedia, len(p.Media))
		for i, m := range p.Media {
			s.Media[i] = snapMedia{Kind: m.Kind.String(), URL: m.URL, AltText: m.AltText}
		}
	}
	return s
}

func snapUserOf(u domain.User) snapUser {
	return snapUser{ID: u.ID, ScreenName: u.ScreenName, Name: u.Name}
}

func (s snapshot) post() domain.Post {
	p := domain.Post{
		ID:             domain.PostID(s.ID),
		Author:         s.Author.user(),
		Text:           s.Text,
		CreatedAt:      time.Unix(s.CreatedAt, 0).UTC(),
		Lang:           s.Lang,
		ConversationID: s.ConversationID,
		ReplyTo:        domain.PostID(s.ReplyTo),
		Visibility:     parseVisibility(s.Visibility),
	}
	if s.RepostedBy != nil {
		u := s.RepostedBy.user()
		p.RepostedBy = &u
	}
	if s.Quote != nil {
		q := s.Quote.post()
		p.Quote = &q
	}
	if len(s.Media) > 0 {
		p.Media = make([]domain.MediaRef, len(s.Media))
		for i, m := range s.Media {
			p.Media[i] = domain.MediaRef{Kind: parseMediaKind(m.Kind), URL: m.URL, AltText: m.AltText}
		}
	}
	return p
}

func (u snapUser) user() domain.User {
	return domain.User{ID: u.ID, ScreenName: u.ScreenName, Name: u.Name}
}

func parseVisibility(s string) domain.Visibility {
	switch s {
	case "limited":
		return domain.Limited
	case "unavailable":
		return domain.Unavailable
	case "withheld":
		return domain.Withheld
	default:
		return domain.Visible
	}
}

func parseMediaKind(s string) domain.MediaKind {
	switch s {
	case "video":
		return domain.Video
	case "gif":
		return domain.GIF
	default:
		return domain.Image
	}
}

// ErrWrongOwner refuses to walk a mirror that belongs to a different account.
// The offline read keeps working; only the sync is refused. Re-pointing the
// mirror is deleting saved.db.
var ErrWrongOwner = errors.New("store: wrong account")

// SavedFetcher is the one page fetch the walk makes. Declared here, by the code
// that calls it, so the walk never names the adapter or the transport: a fake
// fetcher is what makes the whole walk testable with no socket.
// *xsource.Adapter satisfies it as it stands.
type SavedFetcher interface {
	Saved(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error)
}

// Viewer is the one read the walk makes of the session's identity. The adapter
// satisfies it; the store declares the shape it needs and nothing else.
type Viewer interface {
	Viewer(ctx context.Context) (domain.User, error)
}

// pageCap is a cap, not a target: a full re-list that never reaches the end of
// the list stops after this many pages rather than walking forever. The walk is
// otherwise bounded by the end of the list, by the session budget at the
// transport, and by the caller's Abandon.
const pageCap = 40

// The ordering key. sort_index is the whole of the mirror's order: Page scans
// it descending and keys its pages with `sort_index < cursor`, so the key alone
// decides what is newest — and if two rows ever hold the same key, whichever of
// them falls below a page boundary is never returned at all. The rule is that a
// run's keys are unique against every other run's and always above them:
//
//   - A run's band is the sortBand keys immediately above every key already in
//     the file. The band's top is fixed when the run opens, so every key a later
//     run writes is above every key an earlier one wrote, and the rows a walk
//     re-fetches (the newest page of the list) move above the rows it did not
//     touch. The old rule restarted its counter at zero every run, which is what
//     let two runs write the same top band and tie.
//   - Inside the band the keys descend with the post's position in the walk —
//     newest page first, index 0 highest — because the page order is the sort
//     order.
//   - A band is exactly as wide as the most rows one run can write — pageCap
//     pages of PageSize rows — so two bands can never touch.
//
// The key is a ten-digit zero-padded decimal because that is the width of the
// keys already in the file: the column is TEXT, so a key of a different width
// would compare as a string and order wrongly against them. Introducing this
// rule rewrites nothing: an existing file's rows keep their keys and sit below
// every band written after them, and a file that already holds a tie from the
// old rule stops gaining new ones.
//
// Nothing is persisted to hand out bands: the floor is the file's own keys, which
// is what lets a file written by the old rule adopt this one with no migration
// and with no row rewritten. Two runs of one kind that overlapped would each take
// the band they read; two different posts then share a key only where the
// upstream list moved between their two fetches.
const (
	// sortBand is the width of one run's band: a run writes at most pageCap
	// pages, and a page carries at most PageSize rows (the adapter's measured
	// page count), so this is the most keys a run can need and the least that
	// keeps two bands apart.
	sortBand = pageCap * PageSize

	// sortKeyBase is where a file with no rows yet starts: the old rule's own
	// base, so a fresh mirror's keys have the shape and the magnitude of the keys
	// the old rule left behind.
	sortKeyBase = 1_000_000_000

	// sortKeyMax is the largest ten-digit key. A band that no longer fits under
	// it is refused rather than written at eleven digits, where Page would order
	// it as a string and quietly lose rows. At sortBand keys per run that is some
	// eleven million runs away; renumbering the file is a migration's job, not
	// something this walk may do in passing.
	sortKeyMax = 9_999_999_999
)

// errSortSpaceFull refuses a run whose band would leave the ten-digit key space.
var errSortSpaceFull = errors.New("store: sort key space exhausted")

// sortFloor is the key a new band must start above: the greatest key spelled in
// the file, and never below sortKeyBase. CAST reads a key the mirror did not
// write (a test's "z") as 0 and every decimal key as the number it spells, which
// is the comparison Page makes between keys of equal width.
func (s *Store) sortFloor(ctx context.Context) (int64, error) {
	var floor int64
	if err := s.r.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(CAST(sort_index AS INTEGER)), 0) FROM saved_items`).Scan(&floor); err != nil {
		return 0, fmt.Errorf("store: sort floor: %w", err)
	}
	if floor < sortKeyBase {
		floor = sortKeyBase
	}
	return floor, nil
}

// Mirror is the store's domain.Mirror: Page and Status read the file, Begin
// starts the one walk both the pane's r and the sync verb drive.
type Mirror struct {
	st     *Store
	fetch  SavedFetcher
	viewer Viewer
}

// NewMirror builds the port over an open store, a page fetcher and a viewer.
func NewMirror(st *Store, fetch SavedFetcher, viewer Viewer) *Mirror {
	return &Mirror{st: st, fetch: fetch, viewer: viewer}
}

var _ domain.Mirror = (*Mirror)(nil)

// Page delegates to the store's reader: present = 1 rows, newest first.
func (m *Mirror) Page(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	return m.st.Page(ctx, kind, c)
}

// Status delegates to the store's reader: owner, last success, present/absent.
func (m *Mirror) Status(ctx context.Context, kind domain.SavedKind) (domain.MirrorStatus, error) {
	return m.st.Status(ctx, kind)
}

// Path returns the mirror file's path so the sync verb can print it once.
func (m *Mirror) Path() string {
	if m == nil || m.st == nil {
		return ""
	}
	return m.st.Path()
}

// Begin opens one walk. It makes exactly one Viewer read (visible on the budget
// line like every other request), records the owner on a fresh mirror, and
// refuses a foreign session before any page is fetched.
func (m *Mirror) Begin(ctx context.Context, kind domain.SavedKind, full bool) (domain.SyncRun, error) {
	if m == nil || m.st == nil {
		return nil, fmt.Errorf("store: begin: nil mirror")
	}
	if !kind.Valid() {
		return nil, fmt.Errorf("store: begin: invalid kind %d", kind)
	}
	if m.viewer == nil {
		return nil, fmt.Errorf("store: begin: nil viewer")
	}
	if m.fetch == nil {
		return nil, fmt.Errorf("store: begin: nil fetcher")
	}

	user, err := m.viewer.Viewer(ctx)
	if err != nil {
		// A viewer error is already a named state; no run started, so no run row.
		return nil, err
	}
	handle := user.ScreenName

	st, err := m.st.Status(ctx, kind)
	if err != nil {
		return nil, err
	}
	switch {
	case st.Owner == "":
		if err := m.st.SetOwner(ctx, handle); err != nil {
			return nil, err
		}
	case st.Owner == handle:
		// same account — continue
	default:
		return nil, fmt.Errorf("%w: this mirror was synced from @%s, session is @%s", ErrWrongOwner, st.Owner, handle)
	}
	// The run's key band, fixed here so that every key it writes is above every
	// key in the file and below the next run's band. See the sortBand rule.
	floor, err := m.st.sortFloor(ctx)
	if err != nil {
		return nil, err
	}
	if floor+sortBand > sortKeyMax {
		return nil, fmt.Errorf("%w: the file's keys reach %d and a run needs %d more", errSortSpaceFull, floor, sortBand)
	}

	started := now()
	return &run{
		st:      m.st,
		fetch:   m.fetch,
		kind:    kind,
		full:    full,
		started: started,
		bandTop: floor + sortBand,
		seen:    make(map[domain.PostID]bool),
		used:    make(map[domain.Cursor]bool),
	}, nil
}

// run is one walk advanced one page at a time. The caller owns progress and the
// stop; the store owns the transaction and the run row.
type run struct {
	st       *Store
	fetch    SavedFetcher
	kind     domain.SavedKind
	full     bool
	started  time.Time
	cursor   domain.Cursor
	pages    int
	added    int
	absent   int
	written  int   // posts written so far; the position the next post takes in the band
	bandTop  int64 // the band's first key; the walk descends from it
	seen     map[domain.PostID]bool
	used     map[domain.Cursor]bool
	done     bool
	last     domain.SyncProgress
	finished bool // Abandon/Step already recorded the run row
}

var _ domain.SyncRun = (*run)(nil)

// Step fetches one page, upserts it, and decides whether the walk is done.
func (r *run) Step(ctx context.Context) (domain.SyncProgress, error) {
	if r.done {
		return r.last, nil
	}

	if r.used[r.cursor] {
		// Defensive: a caller that somehow re-steps the same cursor without Done.
		return r.finish(ctx, domain.StopStalled)
	}
	r.used[r.cursor] = true

	page, err := r.fetch.Saved(ctx, r.kind, r.cursor)
	if err != nil {
		prog, recErr := r.recordStop(ctx, domain.StopFailure, 0)
		if recErr != nil {
			return prog, fmt.Errorf("store: record failure run: %v (fetch: %w)", recErr, err)
		}
		return prog, err
	}
	// The band is what one run's keys come from: a page is written from it in walk
	// order, and a page that no longer fits ends the walk rather than pushing a key
	// at or below the floor, where another run's rows live — two rows sharing a key
	// is the one thing Page cannot survive. No honest fetcher gets here: a run
	// writes at most pageCap pages of PageSize rows.
	room := sortBand - r.written
	posts := page.Posts
	overflow := len(posts) > room
	if overflow {
		posts = posts[:room]
	}

	items := make([]domain.SavedItem, 0, len(posts))
	for i, p := range posts {
		// The page's order is the sort order: the key descends with the post's
		// position in the walk (newest page first, index 0 highest), and the whole
		// band sits above every key already in the file, so the rows this run
		// re-fetched lead the mirror. See the sortBand rule.
		idx := fmt.Sprintf("%010d", r.bandTop-int64(r.written+i))
		items = append(items, domain.SavedItem{
			Kind:      r.kind,
			Post:      p,
			SortIndex: idx,
			FirstSeen: r.started,
			LastSeen:  r.started,
			Present:   true,
		})
		r.seen[p.ID] = true
	}
	r.written += len(posts)

	pageAdded, err := r.st.Upsert(ctx, items)
	if err != nil {
		prog, recErr := r.recordStop(ctx, domain.StopFailure, 0)
		if recErr != nil {
			return prog, fmt.Errorf("store: record failure run: %v (upsert: %w)", recErr, err)
		}
		return prog, err
	}
	r.added += pageAdded
	r.pages++
	r.cursor = page.Next

	prog := domain.SyncProgress{
		Kind:  r.kind,
		Pages: r.pages,
		Added: r.added,
		Full:  r.full,
	}

	switch {
	case overflow:
		// A page that no longer fits the band ends the walk at the bound the band
		// is — the same stop, and the same marking, the page cap gives — and it is
		// checked before the page's own end: a page the band cut short is not the
		// end of the list.
		return r.finish(ctx, domain.StopPageCap)
	case page.Next == "":
		return r.finish(ctx, domain.StopEndOfList)
	case pageAdded == 0 && !r.full:
		// Incremental rule: the first page that adds nothing new ends the walk.
		return r.finish(ctx, domain.StopNoUnknownIDs)
	case r.pages >= pageCap:
		// StopPageCap counts as success for the last-success row: the walk reached
		// its bound and (for a full walk) marked the rows it did not see.
		return r.finish(ctx, domain.StopPageCap)
	case r.used[page.Next]:
		return r.finish(ctx, domain.StopStalled)
	default:
		r.last = prog
		return prog, nil
	}
}

// Abandon is the caller's own early stop. It records the run row with that
// reason and marks nothing absent: an abandoned full walk is not a completed
// one, and marking rows on it would report as "gone from x.com" what was merely
// not walked.
func (r *run) Abandon(ctx context.Context, stop domain.StopReason) error {
	if r.done || r.finished {
		return nil
	}
	if stop == domain.StopNone {
		stop = domain.StopAbandoned
	}
	_, err := r.recordStop(ctx, stop, 0)
	return err
}

func (r *run) finish(ctx context.Context, stop domain.StopReason) (domain.SyncProgress, error) {
	absent := 0
	if r.full && (stop == domain.StopEndOfList || stop == domain.StopPageCap) {
		// Mark-absent instant is the run's own start, so rows the walk saw (LastSeen
		// = started) are immune and only the rows it did not are marked. StopStalled
		// is not a completed full walk: marking on it would report as gone what was
		// merely not reached.
		n, err := r.st.MarkAbsent(ctx, r.kind, r.started)
		if err != nil {
			return r.recordStop(ctx, domain.StopFailure, 0)
		}
		absent = int(n)
	}
	return r.recordStop(ctx, stop, absent)
}

func (r *run) recordStop(ctx context.Context, stop domain.StopReason, absent int) (domain.SyncProgress, error) {
	r.absent = absent
	finished := now()
	res := domain.SyncResult{
		Kind:     r.kind,
		Full:     r.full,
		Pages:    r.pages,
		Added:    r.added,
		Absent:   absent,
		Stop:     stop,
		Started:  r.started,
		Finished: finished,
	}
	if err := r.st.RecordRun(ctx, res); err != nil {
		r.done = true
		r.finished = true
		r.last = domain.SyncProgress{
			Kind: r.kind, Pages: r.pages, Added: r.added, Absent: absent,
			Full: r.full, Done: true, Stop: stop,
		}
		return r.last, err
	}
	r.done = true
	r.finished = true
	r.last = domain.SyncProgress{
		Kind: r.kind, Pages: r.pages, Added: r.added, Absent: absent,
		Full: r.full, Done: true, Stop: stop,
	}
	return r.last, nil
}
