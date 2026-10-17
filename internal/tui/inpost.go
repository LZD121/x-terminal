package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/screen"
)

// The body-links menu: `L` on a post opens the numbered list of what its body names, and a digit
// follows one.
//
// The three kinds land three ways. A URL leaves the program through the same opener the permalink
// uses. A hashtag is a search inside the program — it is a read, and the reader already has a
// results frame to put it in. A mention is a profile, pushed the way `a` pushes one.
//
// The menu is model state, not a frame: opening it spends nothing, closing it restores the pane byte
// for byte, and it is deliberately unavailable under a cover, where a row of URLs is exactly the
// leak the disguise exists to prevent.

// linkMenu is one post's links while the reader is choosing.
type linkMenu struct {
	post  domain.PostID
	links []domain.Link
}

// linksMsg is the answer to one body URL handed to the opener.
type linksMsg struct {
	URL    string
	Result media.Result
}

// openLinks is `L`: build the menu from the selected post's body links.
func (m Model) openLinks() (Model, tea.Cmd) {
	if m.covered() != "" {
		// A menu of addresses under a cover is the one thing the cover exists to keep off screen.
		return m.setNotice(render.NoticeGated, 0), nil
	}
	f := m.top()
	post, ok := f.post(f.selected())
	if !ok || len(post.Links) == 0 {
		return m.setNotice(render.NoticeNoLinks, 0), nil
	}
	m.links = &linkMenu{post: post.ID, links: post.Links}
	return m, nil
}

// linksKey is the menu's own key handling: a digit follows one link, escape closes, and every other
// key is swallowed — a menu that let `j` through would move a list the reader cannot see.
func (m Model) linksKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape, msg.Code == tea.KeyEnter, msg.String() == "q":
		m.links = nil
		return m, nil
	}
	if len(msg.Text) == 1 {
		if d := int(msg.Text[0] - '1'); d >= 0 && d < len(m.links.links) {
			link := m.links.links[d]
			m.links = nil
			return m.followLink(link)
		}
	}
	return m, nil
}

// followLink is one chosen link, by kind.
func (m Model) followLink(link domain.Link) (Model, tea.Cmd) {
	switch link.Kind {
	case domain.LinkHashtag:
		return m.pushPostSearch("#" + link.Target)
	case domain.LinkMention:
		return m.pushProfile(link.Target)
	}
	url := link.Target
	renderer := m.deps.Media
	return m, func() tea.Msg {
		res, _ := renderer.OpenURL(context.Background(), url)
		return linksMsg{URL: url, Result: res}
	}
}

// linksAnswered is one body URL's opener answer, as a notice. There is no fallback row: the row that
// would carry the address is a post's, and a body link's address is already one digit away in the
// menu that was just closed.
func (m Model) linksAnswered(msg linksMsg) Model {
	switch msg.Result.Outcome {
	case media.OpenedURL:
		return m.setNotice(render.NoticeOpenedBrowser, 0)
	}
	return m.setNotice(render.NoticeNoOpener, 0)
}

// pushPostSearch is one query's results as a frame: the hashtag case, and the path submitSearch
// already takes for a typed query.
func (m Model) pushPostSearch(query string) (Model, tea.Cmd) {
	results, effect := screen.NewPostResults(query, domain.Latest)
	m.seq++
	m.frames = append(m.frames, frame{ID: searchID(query, domain.Latest, m.seq), Kind: frameSearch, Search: results})
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(m.top(), effect, false)
}
