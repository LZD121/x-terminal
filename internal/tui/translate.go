package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
)

// translatedMsg is the answer to one translation request.
//
// ID is the post the request was about, carried so the answer can be cached under it whatever frame
// is on top when it arrives, and so the notice can be withheld when the reader has moved on. Err is
// the port's own error, classified before it got here — the shell prints the classifier's state word
// and never invents one of its own.
type translatedMsg struct {
	ID     domain.PostID
	Result domain.Translation
	Err    error
}

// translate is the `T` arm: show the selected post's body in the configured language.
//
// The rules are in order, and each one is a rule about spending nothing:
//
//   - No post under the cursor (a people row, an empty frame) is `nothing to open`, the same honest
//     line `enter` prints, and no request.
//   - No port or no configured language is a wiring fact, not a failure: the notice field says there
//     is no translation available and nothing is spent. A build without the endpoint must not look
//     like a request that failed.
//   - A post whose answer is already remembered spends nothing: re-reaching an already-translated
//     post re-announces what is known. This is the one-press-per-post rule made structural — the map
//     is read before any request is built, rather than the press being remembered.
//   - Otherwise the process's own request discipline applies before anything goes out, exactly as
//     `page` and `refresh` apply it: nothing is issued while a request is out or while the state
//     forbids one.
//
// The command it returns is one port call. The context is Background for the reason fetchPage's is:
// the shell has no cancellation to offer, and one request is bounded by the transport's own timeout
// and by the limiter every other request passes through. pending is incremented so the in-flight
// latch, the debounce and the pane's loading bookkeeping count this request, and the answer's
// handler releases it.
func (m Model) translate() (Model, tea.Cmd) {
	f := m.top()
	p, ok := f.post(f.selected())
	if !ok {
		return m.setNotice(render.NoticeNothingToOpen, 0), nil
	}
	if m.deps.Translate == nil || m.deps.TranslateLang == "" {
		return m.setNotice(render.NoticeNoTranslation, 0), nil
	}
	if tr, ok := m.translations[p.ID]; ok {
		if tr.Available {
			return m.setNotice(render.NoticeTranslated, 0), nil
		}
		return m.setNotice(render.NoticeNoTranslation, 0), nil
	}
	if m.outstanding() > 0 || !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	tr, lang, id := m.deps.Translate, m.deps.TranslateLang, p.ID
	m.pending++
	return m, func() tea.Msg {
		out, err := tr.Translate(context.Background(), id, lang)
		return translatedMsg{ID: id, Result: out, Err: err}
	}
}

// translateAnswered absorbs one translation answer.
//
// The answer is about a post, not about a frame, so it is remembered whether or not that post is
// still on screen: the reader who comes back to it finds the translation already there and spends
// nothing. The notice is the part that depends on the frame, and it is set only when the post that
// answered is still the top frame's selected post — "showing a translation" would be a lie about a
// frame that is showing a different post.
//
// A refusal names the state through the process's own mapper and sets no notice, which is the
// saved-walk failure path's precedent: the classifier's word belongs on the state line, and a notice
// beside it would be a second, weaker account of the same thing.
func (m Model) translateAnswered(msg translatedMsg) Model {
	m = m.release()
	if msg.Err != nil {
		m.failure = m.failureOf(msg.Err)
		return m.setNotice(render.NoticeNone, 0)
	}
	m = m.withTranslation(msg.ID, msg.Result)
	if !m.selectedPostIs(msg.ID) {
		return m.setNotice(render.NoticeNone, 0)
	}
	if msg.Result.Available {
		return m.setNotice(render.NoticeTranslated, 0)
	}
	return m.setNotice(render.NoticeNoTranslation, 0)
}

// selectedPostIs reports whether the top frame's selected row is still the post id names.
func (m Model) selectedPostIs(id domain.PostID) bool {
	f := m.top()
	p, ok := f.post(f.selected())
	return ok && p.ID == id
}

// translatedConversation is the translation swap for a thread frame: the four slices and the focal
// post, in one call, so the thread arm has one place to pass its whole conversation through.
func (m Model) translatedConversation(c render.Conversation, sel render.Selection) render.Conversation {
	if _, ok := m.translations[sel.ID]; !ok {
		return c
	}
	c.Ancestors = m.translated(c.Ancestors, sel)
	c.Focal = m.translatedPost(c.Focal, sel)
	c.Continued = m.translated(c.Continued, sel)
	c.Replies = m.translated(c.Replies, sel)
	return c
}

// translated is one list's posts with the selected row's remembered translation in place.
//
// With nothing remembered for the selection's own id — the ordinary path, and the reason every
// stored frame stays byte-identical — it returns the slice it was handed rather than a copy of it.
// With an answer, it copies the slice and swaps the one post the selection names, at whatever depth
// it names it: the selection is a row, and a row can be a post quoted inside another, which is what
// Selection.Path exists for. The copy happens only when the slice actually holds that post.
//
// Nothing else about the post moves. Media, Counts, Quote, Visibility and every other field are the
// original's: this is the body in another language, never a second post.
func (m Model) translated(posts []domain.Post, sel render.Selection) []domain.Post {
	tr, ok := m.translations[sel.ID]
	if !ok || !tr.Available {
		// Nothing is remembered, or what is remembered is "the service carries none for this post":
		// either way the rows are the ones the frame already had, and the notice field is what says
		// the second thing.
		return posts
	}
	var out []domain.Post
	for i := range posts {
		if !holdsPost(posts[i], sel.ID) {
			continue
		}
		if out == nil {
			out = make([]domain.Post, len(posts))
			copy(out, posts)
		}
		out[i] = swapPostText(posts[i], sel.ID, tr)
	}
	if out == nil {
		return posts
	}
	return out
}

// translatedPost is the single-post arm of the same swap, for the thread frame's focal post.
func (m Model) translatedPost(p domain.Post, sel render.Selection) domain.Post {
	tr, ok := m.translations[sel.ID]
	if !ok || !tr.Available {
		return p
	}
	if !holdsPost(p, sel.ID) {
		return p
	}
	return swapPostText(p, sel.ID, tr)
}

// holdsPost reports whether this post, or anything it quotes, is the id asked for. The quote chain
// is walked rather than indexed: the response nests them (domain.Post.Quote), and a quoted post is a
// selectable row.
func holdsPost(p domain.Post, id domain.PostID) bool {
	if p.ID == id {
		return true
	}
	return p.Quote != nil && holdsPost(*p.Quote, id)
}

// swapPostText returns p with the post id names carrying the translation's text.
//
// Only the body moves: the id, the author, the date and the quote are the original's. A copy of the
// enclosing post is made only when the swap happens inside its quoted chain — a post that shares no
// part of the chain is handed back as the value it was.
func swapPostText(p domain.Post, id domain.PostID, tr translation) domain.Post {
	if p.ID == id {
		p.Text = tr.Text
		return p
	}
	if p.Quote == nil {
		return p
	}
	quoted := swapPostText(*p.Quote, id, tr)
	if quoted.Text == p.Quote.Text {
		return p
	}
	p.Quote = &quoted
	return p
}
