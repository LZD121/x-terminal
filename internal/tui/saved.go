package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/keys"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/screen"
)

// savedPageMsg is one mirror page arriving for a saved frame.
type savedPageMsg struct {
	Frame  string
	Page   domain.PostPage
	Cursor domain.Cursor
}

// savedFailureMsg is a mirror read or sync step that did not deliver data.
type savedFailureMsg struct {
	Frame string
	Err   error
}

// syncStartedMsg carries the run Begin returned.
type syncStartedMsg struct {
	Frame string
	Run   domain.SyncRun
	Full  bool
}

// syncStepMsg is one Step's progress.
type syncStepMsg struct {
	Frame    string
	Progress domain.SyncProgress
}

// savedStatusMsg is the mirror's Status for a saved frame.
type savedStatusMsg struct {
	Frame  string
	Status domain.MirrorStatus
}

// pushSaved appends a saved frame carrying SavedCapability(kind) and returns the
// mirror Page command. Opening a saved frame issues no Source request.
func (m Model) pushSaved(kind domain.SavedKind) (Model, tea.Cmd) {
	if m.deps.Saved == nil {
		return m, nil
	}
	cap := domain.CapNone
	if m.deps.SavedCapability != nil {
		cap = m.deps.SavedCapability(kind)
	}
	m.seq++
	id := savedID(domain.SavedList(kind), m.seq)
	f := frame{
		ID:    id,
		Kind:  frameSaved,
		Saved: screen.NewSaved(domain.SavedList(kind), cap),
	}
	m.frames = append(m.frames, f)
	m = m.setNotice(render.NoticeNone, 0)
	if cap == domain.CapOperationMissing {
		// The capability state is already on the frame; no request can be made.
		return m, nil
	}
	return m, tea.Batch(m.fetchSavedPage(id, kind, ""), m.fetchSavedStatus(id, kind))
}

func (m Model) fetchSavedPage(id string, kind domain.SavedKind, c domain.Cursor) tea.Cmd {
	return func() tea.Msg {
		page, err := m.deps.Saved.Page(context.Background(), kind, c)
		if err != nil {
			return savedFailureMsg{Frame: id, Err: err}
		}
		return savedPageMsg{Frame: id, Page: page, Cursor: c}
	}
}

func (m Model) fetchSavedStatus(id string, kind domain.SavedKind) tea.Cmd {
	return func() tea.Msg {
		st, err := m.deps.Saved.Status(context.Background(), kind)
		if err != nil {
			return savedFailureMsg{Frame: id, Err: err}
		}
		return savedStatusMsg{Frame: id, Status: st}
	}
}

func (m Model) savedPageAnswered(msg savedPageMsg) Model {
	i := m.frameIndexID(msg.Frame)
	if i < 0 || m.frames[i].Kind != frameSaved {
		return m
	}
	m.frames[i].Read = true
	m.frames[i].Saved = m.frames[i].Saved.Update(screen.MirrorPage{Page: msg.Page})
	return m
}

func (m Model) savedStatusAnswered(msg savedStatusMsg) Model {
	i := m.frameIndexID(msg.Frame)
	if i < 0 || m.frames[i].Kind != frameSaved {
		return m
	}
	m.frames[i].Saved = m.frames[i].Saved.Update(screen.StatusRead{
		Owner:       msg.Status.Owner,
		LastSuccess: msg.Status.LastSuccess,
	})
	return m
}

func (m Model) savedFailed(msg savedFailureMsg) (Model, tea.Cmd) {
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m, nil
	}
	cap := domain.CapNone
	if m.deps.SavedError != nil {
		cap = m.deps.SavedError(msg.Err)
	}
	if cap != domain.CapNone {
		m.frames[i].Saved = m.frames[i].Saved.Update(screen.MirrorFailed{Err: msg.Err, Cap: cap})
		m.frames[i].Read = true
		return m, nil
	}
	// Fall through to the process-wide failure mapper.
	m.failure = m.failureOf(msg.Err)
	m.frames[i].Read = true
	return m, nil
}

// refreshSaved is r on a saved frame: the same Begin/Step/Abandon the verb drives.
func (m Model) refreshSaved() (Model, tea.Cmd) {
	f := m.top()
	if f.Kind != frameSaved {
		return m, nil
	}
	if f.Saved.Sync().Running || f.syncRun != nil || m.outstanding() > 0 {
		return m, nil
	}
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	if f.Saved.Capability() == domain.CapOperationMissing {
		return m, nil
	}
	kind, ok := f.Saved.Kind().Saved()
	if !ok {
		return m, nil
	}
	id := f.ID
	return m, func() tea.Msg {
		run, err := m.deps.Saved.Begin(context.Background(), kind, false)
		if err != nil {
			return savedFailureMsg{Frame: id, Err: err}
		}
		return syncStartedMsg{Frame: id, Run: run, Full: false}
	}
}

func (m Model) syncStarted(msg syncStartedMsg) (Model, tea.Cmd) {
	i := m.frameIndexID(msg.Frame)
	if i < 0 || m.frames[i].Kind != frameSaved {
		return m, nil
	}
	m.frames[i].syncRun = msg.Run
	m.frames[i].Saved = m.frames[i].Saved.Update(screen.SyncStarted{Full: msg.Full})
	m = m.setNotice(render.NoticeSyncing, 0)
	return m, m.stepSync(msg.Frame, msg.Run)
}

func (m Model) stepSync(id string, run domain.SyncRun) tea.Cmd {
	return func() tea.Msg {
		p, err := run.Step(context.Background())
		if err != nil {
			return savedFailureMsg{Frame: id, Err: err}
		}
		return syncStepMsg{Frame: id, Progress: p}
	}
}

func (m Model) syncStepped(msg syncStepMsg) (Model, tea.Cmd) {
	i := m.frameIndexID(msg.Frame)
	if i < 0 || m.frames[i].Kind != frameSaved {
		return m, nil
	}
	m.frames[i].Saved = m.frames[i].Saved.Update(screen.SyncStepped{Progress: msg.Progress})
	if !msg.Progress.Done {
		m = m.setNotice(render.NoticeSyncing, msg.Progress.Added)
		return m, m.stepSync(msg.Frame, m.frames[i].syncRun)
	}
	// Finished: record the outcome, clear the run, re-read the mirror from page one.
	m.frames[i].Saved = m.frames[i].Saved.Update(screen.SyncFinished{Progress: msg.Progress})
	m.frames[i].syncRun = nil
	m = m.setNotice(noticeForSync(msg.Progress), noticeCountForSync(msg.Progress))
	kind, ok := m.frames[i].Saved.Kind().Saved()
	if !ok {
		return m, nil
	}
	// Replace rows with a fresh walk so the screen is exactly what is on disk.
	m.frames[i].Saved = m.frames[i].Saved.WithList(screen.List{})
	m.frames[i].Read = false
	return m, tea.Batch(
		m.fetchSavedPage(msg.Frame, kind, ""),
		m.fetchSavedStatus(msg.Frame, kind),
	)
}

func noticeForSync(p domain.SyncProgress) render.NoticeKind {
	switch screen.NoticeFor(p) {
	case screen.SyncNoticeNoLongerSaved:
		return render.NoticeNoLongerSaved
	case screen.SyncNoticeStalled:
		return render.NoticeSyncStalled
	case screen.SyncNoticeComplete:
		return render.NoticeMirrorComplete
	case screen.SyncNoticeNewPosts:
		return render.NoticeNewPosts
	default:
		return render.NoticeNoNewPosts
	}
}

func noticeCountForSync(p domain.SyncProgress) int {
	if p.Absent > 0 {
		return p.Absent
	}
	return p.Added
}

// abandonSavedSync abandons an in-flight sync when the saved frame is popped.
func (m Model) abandonSavedSync(f frame) tea.Cmd {
	run := f.syncRun
	if run == nil {
		return nil
	}
	return func() tea.Msg {
		_ = run.Abandon(context.Background(), domain.StopAbandoned)
		return nil
	}
}

// savedAction routes bookmarks/likes keys.
func (m Model) savedAction(action keys.Action) (Model, tea.Cmd) {
	switch action {
	case keys.ActionBookmarks:
		return m.pushSaved(domain.Bookmark)
	case keys.ActionLikes:
		return m.pushSaved(domain.Like)
	}
	return m, nil
}

// pageSaved is the pagination trigger on a saved frame: one more mirror page.
func (m Model) pageSaved() (Model, tea.Cmd) {
	f := m.top()
	if f.Kind != frameSaved {
		return m, nil
	}
	l := f.Saved.List()
	if !l.AtEnd() || l.Inflight() {
		return m, nil
	}
	next, effect := l.Update(screen.PageRequested{}, m.deps.Now())
	if effect.Kind != screen.EffectPage {
		return m, nil
	}
	f.Saved = f.Saved.WithList(next)
	m = m.setTop(f)
	kind, ok := f.Saved.Kind().Saved()
	if !ok {
		return m, nil
	}
	return m, m.fetchSavedPage(f.ID, kind, effect.Cursor)
}
