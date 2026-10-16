// Package fake's Mirror is a test double for domain.Mirror: scripted pages,
// status, sync runs and call counters. No clock, no I/O, no package outside the
// standard library and internal/domain.
package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Mirror is an in-memory domain.Mirror. A screen or shell test scripts its
// pages, status and sync run, then asserts the call counters stayed at zero for
// a network Source across open, scroll, page and pop.
type Mirror struct {
	mu sync.Mutex

	pages     map[domain.SavedKind][]domain.PostPage
	status    map[domain.SavedKind]domain.MirrorStatus
	script    map[domain.SavedKind][]domain.SyncProgress
	failPage  map[domain.SavedKind]error
	failBegin map[domain.SavedKind]error

	pageCalls    int
	beginCalls   int
	stepCalls    int
	abandonCalls int
}

// NewMirror returns an empty fake mirror ready for scripting.
func NewMirror() *Mirror {
	return &Mirror{
		pages:     make(map[domain.SavedKind][]domain.PostPage),
		status:    make(map[domain.SavedKind]domain.MirrorStatus),
		script:    make(map[domain.SavedKind][]domain.SyncProgress),
		failPage:  make(map[domain.SavedKind]error),
		failBegin: make(map[domain.SavedKind]error),
	}
}

var _ domain.Mirror = (*Mirror)(nil)

// Pages scripts the pages Page serves for kind, in order. The first page is
// fetched with an empty cursor; later ones with the previous page's Next.
func (m *Mirror) Pages(kind domain.SavedKind, pages ...domain.PostPage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pages[kind] = append([]domain.PostPage(nil), pages...)
}

// StatusOf scripts the status Status returns for kind.
func (m *Mirror) StatusOf(kind domain.SavedKind, st domain.MirrorStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status[kind] = st
}

// RunScript scripts the progress values Begin's SyncRun hands out one Step at a
// time. The last value should carry Done; Abandon is recorded separately.
func (m *Mirror) RunScript(kind domain.SavedKind, steps []domain.SyncProgress) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.script[kind] = append([]domain.SyncProgress(nil), steps...)
}

// FailPage makes the next Page call for kind fail with err, and only that call.
func (m *Mirror) FailPage(kind domain.SavedKind, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failPage[kind] = err
}

// FailBegin makes Begin for kind fail with err.
func (m *Mirror) FailBegin(kind domain.SavedKind, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failBegin[kind] = err
}

// PageCalls is how many times Page has been called.
func (m *Mirror) PageCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pageCalls
}

// BeginCalls is how many times Begin has been called.
func (m *Mirror) BeginCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.beginCalls
}

// StepCalls is how many times Step has been called across every run.
func (m *Mirror) StepCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stepCalls
}

// AbandonCalls is how many times Abandon has been called across every run.
func (m *Mirror) AbandonCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.abandonCalls
}

// Page serves one scripted page for kind.
func (m *Mirror) Page(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pageCalls++
	if err, ok := m.failPage[kind]; ok {
		delete(m.failPage, kind)
		return domain.PostPage{}, err
	}
	pages := m.pages[kind]
	if len(pages) == 0 {
		return domain.PostPage{}, nil
	}
	if c == "" {
		return pages[0], nil
	}
	for i := range pages {
		if i > 0 && pages[i-1].Next == c {
			return pages[i], nil
		}
	}
	return domain.PostPage{}, fmt.Errorf("fake: mirror: unknown cursor %q for %s", c, kind)
}

// Status returns the scripted status for kind.
func (m *Mirror) Status(ctx context.Context, kind domain.SavedKind) (domain.MirrorStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status[kind], nil
}

// Begin starts a scripted sync run for kind.
func (m *Mirror) Begin(ctx context.Context, kind domain.SavedKind, full bool) (domain.SyncRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.beginCalls++
	if err, ok := m.failBegin[kind]; ok {
		delete(m.failBegin, kind)
		return nil, err
	}
	steps := append([]domain.SyncProgress(nil), m.script[kind]...)
	for i := range steps {
		steps[i].Kind = kind
		steps[i].Full = full || steps[i].Full
	}
	return &syncRun{parent: m, steps: steps}, nil
}

type syncRun struct {
	parent *Mirror
	steps  []domain.SyncProgress
	i      int
}

func (r *syncRun) Step(ctx context.Context) (domain.SyncProgress, error) {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()
	r.parent.stepCalls++
	if r.i >= len(r.steps) {
		return domain.SyncProgress{Done: true, Stop: domain.StopEndOfList}, nil
	}
	p := r.steps[r.i]
	r.i++
	return p, nil
}

func (r *syncRun) Abandon(ctx context.Context, stop domain.StopReason) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()
	r.parent.abandonCalls++
	return nil
}
