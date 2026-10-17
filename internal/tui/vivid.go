package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/vivid"
)

// The vivid presentation's shell side: which blocks each screen puts in each column, how the nav
// column's destination is read off the frame that is on top, and the two marks.
//
// It is a translation and not a second model. Every fact the columns show comes from the frame the
// plain reader is already holding — the same posts, the same order, the same selection — and the
// only things added are the marks, which belong to this presentation and are cleared with it.

// vividGrammar is the composition grammar: the vivid one, never a cover's. A cover is a disguise and
// this presentation is not one.
func (m Model) vividGrammar() render.Grammar { return render.Vivid }

// destOf maps the top frame onto the nav column's destination. The frame kind and the destination
// are not one-to-one — bookmarks and likes are both saved lists — so the saved frame's own kind
// decides, and a thread inherits the destination it was opened from.
func (m Model) destOf() vivid.Destination {
	for i := len(m.frames) - 1; i >= 0; i-- {
		f := m.frames[i]
		switch f.Kind {
		case frameSearch:
			return vivid.DestSearch
		case frameProfile:
			return vivid.DestProfile
		case frameSaved:
			if f.Saved.Kind() == domain.ListLikes {
				return vivid.DestLikes
			}
			return vivid.DestBookmarks
		case frameHome:
			return vivid.DestHome
		}
	}
	return vivid.DestHome
}

// vividView assembles one frame of the presentation. The list column is the frame's own blocks in the
// vivid grammar; the reading column is the opened conversation's, and it is empty until the reader
// opens one — the thread frame being the case where the two are the same conversation and the reading
// column is therefore the half that is not in the list.
func (m Model) vividView() vivid.View {
	f := m.top()
	w, h := m.w, m.h
	g := vivid.GeometryOf(w)
	// The list column loses one cell to the mark column and one to the scroll bar the plain reader
	// does not have. Both are the presentation's, so the blocks are built at the column's own
	// measure rather than at the surface's.
	measure := g.List
	if measure < 1 {
		measure = 1
	}
	v := vivid.View{
		Dest:     m.destOf(),
		ListName: m.listName(f),
		Width:    w,
		Height:   h,
		Status:   m.status(),
	}
	v.Marks = vivid.Marks{New: m.vivid.marked, Opened: m.vivid.opened}

	if f.Kind == frameThread {
		// A conversation has no list beside it: the list column is the thread's own ancestors, and
		// the reading column holds the focal post and its replies.
		v.List = m.scrolledBlocks(m.threadAncestors(f, measure))
		v.Reading = m.scrolledBlocks(m.threadReading(f, g.Reading))
		if title := m.readingTitle(f); title != "" {
			v.ReadingTitle = title
		}
		return v
	}

	v.List = m.scrolledBlocks(m.blocksAt(f, measure-1))
	v.Notice = "open a post to read it here"
	if m.vivid.reading != "" {
		if post, ok := m.framePostByID(m.vivid.reading); ok {
			v.Reading = m.scrolledBlocks(m.threadReadingOf(post, g.Reading))
			v.ReadingTitle = m.postHeading(post)
		}
	}
	return v
}

// scrolledBlocks drops the rows above the pane's first row — the plain reader's own skip, applied to
// a column. The columns scroll together on the frame's one anchor: two independent scroll positions
// would be two places for the reader to lose, and the list is the column that is being navigated.
func (m Model) scrolledBlocks(blocks []render.Block) []render.Block {
	return m.scrolled(blocks, m.top())
}

// threadAncestors is the conversation above the focal post, in the vivid grammar.
func (m Model) threadAncestors(f frame, w int) []render.Block {
	posts := f.Thread.Ancestors()
	if len(posts) == 0 {
		return nil
	}
	return render.Blocks(posts, m.vividGrammar(), w, render.Selection{}, m.deps.Now(), nil)
}

// threadReading is the conversation's own half: the focal post, its continuations and its replies.
func (m Model) threadReading(f frame, w int) []render.Block {
	posts := append([]domain.Post{f.Thread.Focal()}, f.Thread.Continued()...)
	posts = append(posts, f.Thread.Replies()...)
	if len(posts) == 0 {
		return nil
	}
	return render.Blocks(posts, m.vividGrammar(), w, render.Selection{}, m.deps.Now(), nil)
}

// threadReadingOf is a conversation opened beside a list: the post's own copy, because the thread's
// page has not been requested yet in this presentation — the reader is looking at the head and the
// replies arrive behind it.
func (m Model) threadReadingOf(post domain.Post, w int) []render.Block {
	return render.Blocks([]domain.Post{post}, m.vividGrammar(), w, render.Selection{}, m.deps.Now(), nil)
}

// listName is the list column's heading: the plain reader's own label, which already names the feed
// or the query.
func (m Model) listName(f frame) string {
	st := m.status()
	return st.List
}

// readingTitle is the reading column's heading for a conversation: the author and the post id.
func (m Model) readingTitle(f frame) string {
	post := f.Thread.Focal()
	if post.ID == "" {
		return ""
	}
	return m.postHeading(post)
}

// postHeading is one post named in the heading's own form: who and which. It is one function so the
// two callers cannot disagree about the shape.
func (m Model) postHeading(post domain.Post) string {
	name := post.Author.Name
	if name == "" {
		name = post.Author.ScreenName
	}
	return strings.TrimSpace(name + " / " + string(post.ID))
}

// framePostByID finds a post in the top frame's own data. It is what the reading column shows before
// the conversation has been fetched: the copy the list already holds.
func (m Model) framePostByID(id domain.PostID) (domain.Post, bool) {
	f := m.top()
	for _, tgt := range f.targets() {
		post, ok := f.post(tgt)
		if ok && post.ID == id {
			return post, true
		}
	}
	return domain.Post{}, false
}

// openIntoReading is the vivid mode's `enter`: it fills the reading column instead of pushing a
// thread frame, which is the whole point of the second column — the list stays where it was.
func (m Model) openIntoReading() (Model, tea.Cmd) {
	f := m.top()
	sel := f.selected()
	if sel.ID == "" {
		return m.setNotice(render.NoticeNothingToOpen, 0), nil
	}
	post, ok := f.post(sel)
	if !ok {
		return m.setNotice(render.NoticeNothingToOpen, 0), nil
	}
	m.vivid.reading = post.ID
	m.vivid = m.vivid.withOpened(post.ID)
	return m, nil
}
