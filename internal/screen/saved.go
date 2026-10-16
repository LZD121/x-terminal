package screen

import (
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Saved is one saved-list frame: the mirror walk, the sync state, and the list's
// capability. Its rows come from the mirror as values; the shell never puts a
// fetched page into this list directly.
type Saved struct {
	kind       domain.ListKind
	list       List
	capability domain.Capability
	owner      string
	synced     time.Time
	sync       SyncState
}

// SyncState is one sync walk's progress as the frame keeps it.
type SyncState struct {
	Running bool
	Added   int
	Stop    domain.StopReason
	Absent  int
	Done    bool
	Full    bool
}

// Kind is which saved list this frame shows.
func (s Saved) Kind() domain.ListKind { return s.kind }

// List is the walk's rows, selection, cursor, failure and position facts.
func (s Saved) List() List { return s.list }

// Capability is the list's own capability state, CapNone when there is none.
func (s Saved) Capability() domain.Capability { return s.capability }

// Owner is the account the mirror was synced from, empty when it never was.
func (s Saved) Owner() string { return s.owner }

// Synced is the last successful run's instant, zero for never.
func (s Saved) Synced() time.Time { return s.synced }

// Sync is the walk's current progress.
func (s Saved) Sync() SyncState { return s.sync }

// NewSaved starts an empty saved frame for kind, carrying the capability the
// composition root already knows about this build.
func NewSaved(kind domain.ListKind, cap domain.Capability) Saved {
	return Saved{kind: kind, capability: cap}
}

// MirrorPage is one page of the mirror arriving. The walk absorbs it; the
// capability is left alone.
type MirrorPage struct{ Page domain.PostPage }

// MirrorFailed is a mirror read that did not deliver data. Cap is the capability
// the composition root mapped from the error; CapNone leaves the frame's
// capability untouched so the shell can fall through to Deps.Failure.
type MirrorFailed struct {
	Err error
	Cap domain.Capability
}

// SyncStarted begins a sync walk. Full is whether the walk is a full re-list.
type SyncStarted struct{ Full bool }

// SyncStepped is one page of the sync walk. Progress.Done ends the run.
type SyncStepped struct{ Progress domain.SyncProgress }

// SyncFinished carries the run's final progress — its Absent, Full and Stop —
// which is what the notice is computed from and what the frame's state keeps.
type SyncFinished struct{ Progress domain.SyncProgress }

// CapabilitySet sets the list's capability. CapNone clears it.
type CapabilitySet struct{ Cap domain.Capability }

// StatusRead is the mirror's status for this list: owner, last success, and an
// optional capability the status itself named.
type StatusRead struct {
	Owner       string
	LastSuccess time.Time
	Cap         domain.Capability
}

func (MirrorPage) event()    {}
func (MirrorFailed) event()  {}
func (SyncStarted) event()   {}
func (SyncStepped) event()   {}
func (SyncFinished) event()  {}
func (CapabilitySet) event() {}
func (StatusRead) event()    {}

// Update applies one event to the saved frame. A mirror page is absorbed by the
// walk's own page reducer; a failed mirror read sets the capability and leaves
// every row alone; SyncStepped / SyncFinished move the sync state; StatusRead
// fills the synced field.
//
// The capability a frame wears is sticky in the spec's sense (06-UI-SPEC.md,
// "Stickiness"): it stays until an action of this frame succeeds, and nothing
// else — no timer, no other frame's failure — clears it. Two actions can
// succeed here, and each releases exactly what it proves:
//
//   - a read of the list that delivered its page (MirrorPage) releases the
//     file state, which is what that read disproves — the file was read;
//   - a sync that reached its own end (SyncFinished, and not a run the store
//     recorded as a failure) releases whatever the frame wore: the walk read and
//     wrote the file through the list's own operation, and its owner check
//     passed before the first page was asked for.
//
// Two states survive a read, and they are not exceptions but the same rule: a
// page comes off the disk, so it says nothing about which account the session is
// (only a sync's owner check refutes the account state) and nothing about the
// operation set the frame's own requests go through (the missing-operation state
// is not even reachable beside a read: a frame wearing it issues no read and
// refuses a sync, and its remedy — the restart that loads the pin — builds a
// frame with the capability the composition root decides then).
func (s Saved) Update(ev Event) Saved {
	switch e := ev.(type) {
	case MirrorPage:
		next, _ := s.list.Update(PageFetched{Page: e.Page}, time.Time{})
		s.list = next
		if s.capability == domain.CapMirrorUnreadable {
			s.capability = domain.CapNone
		}
	case MirrorFailed:
		if e.Cap != domain.CapNone {
			s.capability = e.Cap
		}
	case SyncStarted:
		s.sync = SyncState{Running: true, Full: e.Full}
	case SyncStepped:
		s.sync.Running = !e.Progress.Done
		s.sync.Added = e.Progress.Added
		s.sync.Absent = e.Progress.Absent
		s.sync.Full = e.Progress.Full
		if e.Progress.Done {
			s.sync.Done = true
			s.sync.Stop = e.Progress.Stop
			s.sync.Running = false
		}
	case SyncFinished:
		s.sync = SyncState{
			Added:   e.Progress.Added,
			Absent:  e.Progress.Absent,
			Stop:    e.Progress.Stop,
			Done:    true,
			Full:    e.Progress.Full,
			Running: false,
		}
		// StopFailure is the one final progress a run records without the caller
		// seeing an error (the walk's MarkAbsent failed and RecordRun succeeded):
		// a walk that ended that way is not a sync that succeeded, so its state
		// stays.
		if e.Progress.Stop != domain.StopFailure {
			s.capability = domain.CapNone
		}
	case CapabilitySet:
		s.capability = e.Cap
	case StatusRead:
		s.owner = e.Owner
		s.synced = e.LastSuccess
		if e.Cap != domain.CapNone {
			s.capability = e.Cap
		}
	}
	return s
}

// WithList replaces the walk's list (used after a sync re-reads the mirror from
// page one so the screen holds exactly what is on disk).
func (s Saved) WithList(l List) Saved {
	s.list = l
	return s
}

// SyncNotice is which sync outcome the pane announces. It is shaped like
// render.NoticeKind so the shell maps one table onto the status line.
type SyncNotice uint8

const (
	// SyncNoticeNone is no sync notice.
	SyncNoticeNone SyncNotice = iota
	// SyncNoticeNoLongerSaved is "{n} no longer saved".
	SyncNoticeNoLongerSaved
	// SyncNoticeStalled is "sync stopped (cursor stalled)".
	SyncNoticeStalled
	// SyncNoticeComplete is "mirror complete".
	SyncNoticeComplete
	// SyncNoticeNewPosts is "{n} new posts".
	SyncNoticeNewPosts
	// SyncNoticeNoNewPosts is "no new posts".
	SyncNoticeNoNewPosts
)

// NoticeFor picks the sync outcome notice from the final progress.
//
//	Absent > 0              -> SyncNoticeNoLongerSaved
//	Stop == StopStalled     -> SyncNoticeStalled
//	Stop == StopEndOfList   -> SyncNoticeComplete
//	Added > 0               -> SyncNoticeNewPosts
//	otherwise               -> SyncNoticeNoNewPosts
func NoticeFor(p domain.SyncProgress) SyncNotice {
	switch {
	case p.Absent > 0:
		return SyncNoticeNoLongerSaved
	case p.Stop == domain.StopStalled:
		return SyncNoticeStalled
	case p.Stop == domain.StopEndOfList:
		return SyncNoticeComplete
	case p.Added > 0:
		return SyncNoticeNewPosts
	default:
		return SyncNoticeNoNewPosts
	}
}
