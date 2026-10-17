package tui

import (
	"context"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
)

// Explore's trend list: `x` opens it, `j`/`k` move, `enter` searches the trend.
//
// It is a frame and not a menu because it is a place the reader is: `q` returns to the feed exactly
// as from a search, and a trend's search is a frame of its own on top. The list is one screen with no
// pagination, so the selection is an index rather than a walk.

// trendsMsg is Explore's answer.
type trendsMsg struct {
	Frame  string
	Trends []domain.Trend
}

// openTrends is `x`: push the frame and issue its one request.
func (m Model) openTrends() (Model, tea.Cmd) {
	if m.directory() == nil {
		return m.setNotice(render.NoticeGated, 0), nil
	}
	m.seq++
	id := "trends#" + strconv.FormatUint(m.seq, 10)
	m.frames = append(m.frames, frame{ID: id, Kind: frameTrends})
	if f := m.top(); f.ID != id {
		return m, nil
	}
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	m = m.setNotice(render.NoticeNone, 0)
	return m, m.fetchTrends(id)
}

// fetchTrends issues Explore's one request.
func (m Model) fetchTrends(id string) tea.Cmd {
	dir := m.directory()
	return func() tea.Msg {
		trends, err := dir.Trends(context.Background())
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return trendsMsg{Frame: id, Trends: trends}
	}
}

// trendsAnswered absorbs Explore's list.
func (m Model) trendsAnswered(msg trendsMsg) Model {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m
	}
	m.frames[i].Read = true
	m.frames[i].Trends = msg.Trends
	m.failure = domain.Failure{}
	return m
}

// followTrend is `enter` on Explore: the selected trend's own query as a search.
func (m Model) followTrend() (Model, tea.Cmd) {
	f := m.top()
	i, ok := trendIndex(f.trendTarget())
	if !ok || i >= len(f.Trends) {
		return m.setNotice(render.NoticeNothingToOpen, 0), nil
	}
	if f.Trends[i].Query == "" {
		return m.setNotice(render.NoticeNothingToOpen, 0), nil
	}
	return m.pushPostSearch(f.Trends[i].Query)
}
