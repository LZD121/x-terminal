package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/render"
)

// mediaMsg is the answer to one media keypress: the selected post's id and one
// Result per attachment, in the post's own order. It updates the notice and the
// post's fallback rows. Results that carry a Done channel are watched by a
// follow-up Cmd so a child that starts then exits non-zero still becomes a
// SpawnFailed URL fallback without blocking Open.
type mediaMsg struct {
	PostID  domain.PostID
	Results []media.Result
}

// mediaExitMsg is Wait's answer for one attachment after Open returned success.
// A non-nil Err demotes that attachment to SpawnFailed and prints its URL.
type mediaExitMsg struct {
	PostID domain.PostID
	Index  int // 0-based attachment index
	Err    error
}

// openMedia opens every attachment of the selected post through Deps.Media.
// A post with no attachments gets the honest "does not apply" notice and no
// command. The arm is per post, not per attachment: the pane reports one
// outcome for the action the user took.
func (m Model) openMedia() (Model, tea.Cmd) {
	f := m.top()
	sel := f.selected()
	post, ok := f.post(sel)
	if !ok || len(post.Media) == 0 {
		return m.setNotice(render.NoticeNoMedia, 0), nil
	}
	refs := append([]domain.MediaRef(nil), post.Media...)
	id := post.ID
	renderer := m.deps.Media
	return m, func() tea.Msg {
		results := make([]media.Result, 0, len(refs))
		for _, ref := range refs {
			res, _ := renderer.Open(context.Background(), ref)
			results = append(results, res)
		}
		return mediaMsg{PostID: id, Results: results}
	}
}

// mediaAnswered absorbs a mediaMsg into the top frame's Fallbacks and one notice,
// then returns Cmds that watch each successful Result.Done for a late non-zero exit.
//
// Notice rule: the first failed attachment of a post decides between
// "no opener - URL shown" and "opened …". When every attachment failed (or the
// first result is a failure and none succeeded), the notice is NoticeNoOpener;
// when any succeeded, the notice reports those successes (one noun, or
// "opened {n} media" for a set) and only the failed indexes stay in Fall.Failed.
// A mixed set therefore reports the successes and prints the failed entries'
// URLs, because both facts are on screen.
//
// Nothing removes a fallback but the next successful action on that post: a
// selection move, a page, a failed refresh and a cover round trip leave it.
func (m Model) mediaAnswered(msg mediaMsg) (Model, tea.Cmd) {
	if len(msg.Results) == 0 {
		return m.setNotice(render.NoticeNoMedia, 0), nil
	}
	m = m.applyMediaResults(msg.PostID, msg.Results)
	var cmds []tea.Cmd
	for i, res := range msg.Results {
		if res.Done == nil {
			continue
		}
		switch res.Outcome {
		case media.OpenedImage, media.OpenedVideo, media.OpenedGIF, media.OpenedPoster:
		default:
			continue
		}
		id := msg.PostID
		idx := i
		done := res.Done
		cmds = append(cmds, func() tea.Msg {
			err := <-done
			if err == nil {
				return nil
			}
			return mediaExitMsg{PostID: id, Index: idx, Err: err}
		})
	}
	return m, tea.Batch(cmds...)
}

// mediaExitAnswered demotes one attachment whose child exited non-zero after
// Open returned success. The pane must not have blocked on Wait; this is the
// second message that makes MEDIA-03's "exited non-zero" arm true.
func (m Model) mediaExitAnswered(msg mediaExitMsg) Model {
	if msg.Err == nil || msg.Index < 0 {
		return m
	}
	f := m.top()
	post, ok := m.postByID(msg.PostID)
	if !ok || msg.Index >= len(post.Media) {
		return m
	}
	fall, has := f.Fall.For(msg.PostID)
	idx1 := msg.Index + 1
	for _, n := range fall.Failed {
		if n == idx1 {
			return m // already recorded
		}
	}
	failed := append(append([]int(nil), fall.Failed...), idx1)
	if f.Fall == nil {
		f.Fall = render.Fallbacks{}
	} else {
		next := make(render.Fallbacks, len(f.Fall))
		for k, v := range f.Fall {
			next[k] = v
		}
		f.Fall = next
	}
	entry := fall
	if !has {
		entry = render.MediaFallback{}
	}
	entry.Failed = failed
	f.Fall[msg.PostID] = entry
	m = m.setTop(f)
	okCount := len(post.Media) - len(failed)
	if okCount <= 0 {
		return m.setNotice(render.NoticeNoOpener, 0)
	}
	if okCount == 1 {
		// Keep the existing success notice when one attachment remains open;
		// the URL row is the honest half for the failure.
		return m
	}
	return m.setNotice(render.NoticeOpenedMedia, okCount)
}

// applyMediaResults writes Fallbacks and the notice for one media answer.
func (m Model) applyMediaResults(id domain.PostID, results []media.Result) Model {
	var failed []int
	var okOutcomes []media.Outcome
	for i, res := range results {
		switch res.Outcome {
		case media.OpenedImage, media.OpenedVideo, media.OpenedGIF, media.OpenedPoster:
			okOutcomes = append(okOutcomes, res.Outcome)
		case media.NoOpener, media.NoProgram, media.SpawnFailed:
			failed = append(failed, i+1)
		default:
			failed = append(failed, i+1)
		}
	}
	f := m.top()
	if f.Fall == nil {
		f.Fall = render.Fallbacks{}
	} else {
		// Copy-on-write so a previous frame value is not mutated under a shared map.
		next := make(render.Fallbacks, len(f.Fall))
		for k, v := range f.Fall {
			next[k] = v
		}
		f.Fall = next
	}
	if len(failed) == 0 {
		// Successful action on this post clears its fallback entry.
		delete(f.Fall, id)
	} else {
		f.Fall[id] = render.MediaFallback{Failed: append([]int(nil), failed...)}
	}
	m = m.setTop(f)
	if len(okOutcomes) == 0 {
		return m.setNotice(render.NoticeNoOpener, 0)
	}
	if len(okOutcomes) == 1 {
		return m.setNotice(noticeOfOutcome(okOutcomes[0]), 0)
	}
	return m.setNotice(render.NoticeOpenedMedia, len(okOutcomes))
}

// postByID finds a post on the top frame by id (any target).
func (m Model) postByID(id domain.PostID) (domain.Post, bool) {
	f := m.top()
	for _, t := range f.targets() {
		p, ok := f.post(t)
		if ok && p.ID == id {
			return p, true
		}
	}
	return domain.Post{}, false
}

// noticeOfOutcome is the closed media.Outcome → render.NoticeKind table. Every
// Outcome value has a row; every row's kind has words in status.go / copy.go.
func noticeOfOutcome(o media.Outcome) render.NoticeKind {
	switch o {
	case media.OpenedImage:
		return render.NoticeOpenedImage
	case media.OpenedVideo:
		return render.NoticeOpenedVideo
	case media.OpenedGIF:
		return render.NoticeOpenedGIF
	case media.OpenedPoster:
		return render.NoticeOpenedPoster
	case media.OpenedURL:
		return render.NoticeOpenedBrowser
	case media.NoOpener, media.NoProgram, media.SpawnFailed:
		return render.NoticeNoOpener
	default:
		return render.NoticeNoOpener
	}
}
