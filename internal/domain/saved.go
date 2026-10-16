package domain

import (
	"context"
	"strconv"
	"time"
)

// SavedKind is which of the two saved lists a row belongs to. Bookmark is the
// zero value because it is the list this phase can always reach.
type SavedKind uint8

const (
	// Bookmark is a post the account has bookmarked.
	Bookmark SavedKind = iota
	// Like is a post the account has liked.
	Like
)

// String returns the singular word the file stores and the tests print.
func (k SavedKind) String() string {
	switch k {
	case Bookmark:
		return "bookmark"
	case Like:
		return "like"
	}
	return "saved kind " + strconv.Itoa(int(k))
}

// List returns the status line's list field word for this kind.
func (k SavedKind) List() string {
	switch k {
	case Bookmark:
		return "bookmarks"
	case Like:
		return "likes"
	}
	return k.String()
}

// Valid reports whether k is one of the two closed values.
func (k SavedKind) Valid() bool {
	return k == Bookmark || k == Like
}

// ListKind is the identity of any list a frame can show: the two home timelines
// and the two saved lists. A frame carries one ListKind while HomeKind and
// SavedKind keep their own types at the ports that only speak one pair.
type ListKind uint8

const (
	// ListFollowing is the following home timeline, and the zero value.
	ListFollowing ListKind = iota
	// ListForYou is the for-you home timeline.
	ListForYou
	// ListBookmarks is the bookmarks mirror.
	ListBookmarks
	// ListLikes is the likes mirror.
	ListLikes
)

// String names each list in the status line's words.
func (k ListKind) String() string {
	switch k {
	case ListFollowing:
		return "following"
	case ListForYou:
		return "for you"
	case ListBookmarks:
		return "bookmarks"
	case ListLikes:
		return "likes"
	}
	return "list kind " + strconv.Itoa(int(k))
}

// HomeList maps a home timeline onto the shared list identity.
func HomeList(k HomeKind) ListKind {
	switch k {
	case ForYou:
		return ListForYou
	default:
		return ListFollowing
	}
}

// SavedList maps a saved kind onto the shared list identity.
func SavedList(k SavedKind) ListKind {
	switch k {
	case Like:
		return ListLikes
	default:
		return ListBookmarks
	}
}

// Home reports the home timeline this list is, or false when it is a saved list.
func (k ListKind) Home() (HomeKind, bool) {
	switch k {
	case ListFollowing:
		return Following, true
	case ListForYou:
		return ForYou, true
	default:
		return 0, false
	}
}

// Saved reports the saved kind this list is, or false when it is a home timeline.
func (k ListKind) Saved() (SavedKind, bool) {
	switch k {
	case ListBookmarks:
		return Bookmark, true
	case ListLikes:
		return Like, true
	default:
		return 0, false
	}
}

// SavedItem is one row of a saved list as the write API accepts it. The store
// builds the on-disk JSON from Post; callers never produce the stored shape.
type SavedItem struct {
	Kind      SavedKind
	Post      Post
	SortIndex string
	FirstSeen time.Time
	LastSeen  time.Time
	Present   bool
}

// RunID is the primary key of one sync_runs row.
type RunID int64

// StopReason is why a sync walk ended. StopNone is the zero value so an unset
// progress cannot be read as a named stop.
type StopReason uint8

const (
	// StopNone is "no stop yet": the zero value on a progress that is still walking.
	StopNone StopReason = iota
	// StopNoUnknownIDs is the incremental walk's stop: a page carried no id the
	// mirror did not already hold for that kind.
	StopNoUnknownIDs
	// StopEndOfList is the full re-list's stop: the upstream handed back an empty Next.
	StopEndOfList
	// StopPageCap is the walk hitting the page budget before the end of the list.
	StopPageCap
	// StopFailure is a page that did not deliver data; the run row is written before
	// the error returns.
	StopFailure
	// StopAbandoned is the caller's early stop (a popped frame, a spent budget).
	StopAbandoned
	// StopStalled is a cursor that repeated or a page that contributed nothing.
	StopStalled
)

// String returns the words the run record and the sync verb print.
func (r StopReason) String() string {
	switch r {
	case StopNone:
		return ""
	case StopNoUnknownIDs:
		return "no unknown ids"
	case StopEndOfList:
		return "end of list"
	case StopPageCap:
		return "page cap"
	case StopFailure:
		return "failure"
	case StopAbandoned:
		return "abandoned"
	case StopStalled:
		return "cursor stalled"
	}
	return "stop reason " + strconv.Itoa(int(r))
}

// SyncProgress is what one step of a walk reports: the running pages and items
// while it walks, and on the final progress (Done) the run's own outcome.
type SyncProgress struct {
	Kind   SavedKind
	Pages  int
	Added  int
	Absent int
	Full   bool
	Done   bool
	Stop   StopReason
}

// SyncResult is the run's record: what the store writes into sync_runs and what
// the sync verb prints.
type SyncResult struct {
	Kind     SavedKind
	Full     bool
	Pages    int
	Added    int
	Absent   int
	Stop     StopReason
	Started  time.Time
	Finished time.Time
}

// MirrorStatus is what the pane's status line reads about one saved list.
type MirrorStatus struct {
	// Owner is the account the mirror was synced from, empty when it never was.
	Owner string
	// LastSuccess is the zero time when there has never been a successful run.
	LastSuccess time.Time
	// Present is the count of rows with present = 1.
	Present int
	// Absent is the count of rows marked present = 0 (the "{n} no longer saved" count).
	Absent int
}

// MirrorKindCount is one saved kind's present and absent row counts for the
// doctor's mirror line. It carries no row contents and no per-row instant.
type MirrorKindCount struct {
	Kind    SavedKind
	Present int
	Absent  int
}

// MirrorFacts is the closed set of facts the doctor's mirror line may carry.
// There is no field for a post body, a row's contents or a per-row instant — a
// mirror line is a report about a file, and this type is where that stays true.
//
// Newer means the file records a schema version this build does not know
// (ErrSchemaNewer). LastRun and Stop are the newest sync_runs row's instant and
// stop reason; both are zero for a mirror that never ran.
type MirrorFacts struct {
	Path          string
	Exists        bool
	SchemaVersion int
	Newer         bool
	Owner         string
	Kinds         []MirrorKindCount
	LastRun       time.Time
	Stop          StopReason
}

// Capability is a fact about this build and this machine, deliberately not a
// FailureKind: a "we never dialled" state must not pose as a "we dialled and this
// happened" one.
type Capability uint8

const (
	// CapNone is "no capability problem": the zero value.
	CapNone Capability = iota
	// CapOperationMissing is an operation id this build does not hold.
	CapOperationMissing
	// CapWrongAccount is a session whose account is not the mirror's owner.
	CapWrongAccount
	// CapMirrorUnreadable is a saved.db that exists and cannot be opened or read.
	CapMirrorUnreadable
)

// String names each capability in one lowercase phrase.
func (c Capability) String() string {
	switch c {
	case CapNone:
		return ""
	case CapOperationMissing:
		return "operation missing"
	case CapWrongAccount:
		return "wrong account"
	case CapMirrorUnreadable:
		return "mirror unreadable"
	}
	return "capability " + strconv.Itoa(int(c))
}

// Mirror is the port the pane reads saved pages through. The pane never fetches
// on open: Page serves from disk, Status fills the synced field, and Begin starts
// the one walk both the pane's r and the sync verb drive.
type Mirror interface {
	Page(ctx context.Context, kind SavedKind, c Cursor) (PostPage, error)
	Status(ctx context.Context, kind SavedKind) (MirrorStatus, error)
	Begin(ctx context.Context, kind SavedKind, full bool) (SyncRun, error)
}

// SyncRun is one walk advanced one page at a time so the caller owns progress
// and the stop. Step records the run's own failure row before returning an
// error; Abandon records the caller's early stop without marking anything absent.
type SyncRun interface {
	Step(ctx context.Context) (SyncProgress, error)
	Abandon(ctx context.Context, stop StopReason) error
}
