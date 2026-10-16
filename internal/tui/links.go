package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/render"
)

// copyLinkMsg is the answer to one y keypress: the selected post's id, the
// permalink that was handed to the clipboard chain, and which links ran.
type copyLinkMsg struct {
	PostID  domain.PostID
	URL     string
	Outcome media.ClipboardOutcome
}

// openPostMsg is the answer to one o keypress: the selected post's id, the
// permalink handed to OpenURL, and the seam's result.
type openPostMsg struct {
	PostID domain.PostID
	URL    string
	Result media.Result
}

// copyLink builds the selected post's permalink and copies it through Deps.Clip.
// A missing selection is a no-op. The body never tests display state: the
// dispatcher gate refuses under a cover before this arm runs.
func (m Model) copyLink() (Model, tea.Cmd) {
	f := m.top()
	sel := f.selected()
	post, ok := f.post(sel)
	if !ok || post.ID == "" {
		return m, nil
	}
	url := render.Permalink(post)
	id := post.ID
	clip := m.deps.Clip
	return m, func() tea.Msg {
		outcome := media.NoClipboard
		if clip != nil {
			outcome = clip.Copy(context.Background(), url)
		}
		return copyLinkMsg{PostID: id, URL: url, Outcome: outcome}
	}
}

// copyLinkAnswered absorbs a copyLinkMsg into a notice and, on NoClipboard, the
// post's permalink fallback row. Neither marks read, moves, nor requests.
func (m Model) copyLinkAnswered(msg copyLinkMsg) Model {
	switch msg.Outcome {
	case media.Copied:
		return m.setNotice(render.NoticeLinkCopied, 0)
	case media.CopiedWithWLCopy:
		return m.setNotice(render.NoticeLinkCopiedWL, 0)
	default:
		m = m.setPermalinkFall(msg.PostID, msg.URL)
		return m.setNotice(render.NoticeNoClipboard, 0)
	}
}

// openPost builds the selected post's permalink and opens it through Deps.Media.
// A missing selection is a no-op. The body never tests display state.
func (m Model) openPost() (Model, tea.Cmd) {
	f := m.top()
	sel := f.selected()
	post, ok := f.post(sel)
	if !ok || post.ID == "" {
		return m, nil
	}
	url := render.Permalink(post)
	id := post.ID
	renderer := m.deps.Media
	return m, func() tea.Msg {
		res, _ := renderer.OpenURL(context.Background(), url)
		return openPostMsg{PostID: id, URL: url, Result: res}
	}
}

// openPostAnswered absorbs an openPostMsg into a notice and, on opener failure,
// the post's permalink fallback row. A successful OpenURL with Done set is
// watched so a child that starts then exits non-zero still prints the URL.
func (m Model) openPostAnswered(msg openPostMsg) (Model, tea.Cmd) {
	switch msg.Result.Outcome {
	case media.OpenedURL:
		m = m.setNotice(render.NoticeOpenedBrowser, 0)
		if msg.Result.Done == nil {
			return m, nil
		}
		id := msg.PostID
		url := msg.URL
		done := msg.Result.Done
		return m, func() tea.Msg {
			err := <-done
			if err == nil {
				return nil
			}
			return openPostMsg{
				PostID: id,
				URL:    url,
				Result: media.Result{Outcome: media.SpawnFailed},
			}
		}
	default:
		m = m.setPermalinkFall(msg.PostID, msg.URL)
		return m.setNotice(render.NoticeNoOpener, 0), nil
	}
}

// setPermalinkFall records a permalink url-row for postID without clearing any
// attachment failure indexes already on that post.
func (m Model) setPermalinkFall(id domain.PostID, url string) Model {
	f := m.top()
	if f.Fall == nil {
		f.Fall = render.Fallbacks{}
	} else {
		next := make(render.Fallbacks, len(f.Fall))
		for k, v := range f.Fall {
			next[k] = v
		}
		f.Fall = next
	}
	cur := f.Fall[id]
	cur.Permalink = true
	cur.Link = url
	f.Fall[id] = cur
	return m.setTop(f)
}
