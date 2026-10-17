package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/screen"
)

// The four reads a profile screen can reach beyond its header and its posts: the three account
// lists and the media tab. Each is one frame pushed from the profile, one request, one answer.
//
// They are separate frames rather than tabs inside the profile frame because the profile's own
// reducer is about one timeline: a tab would be a second walk and a second failure state inside a
// type that already has both. A pushed frame keeps the rule the rest of this shell follows — one
// frame, one read, one place the reader was — and `q` returns to the profile exactly as it was.

// usersMsg is one answer to an account-list request.
type usersMsg struct {
	Frame   string
	Kind    domain.UserListKind
	Page    domain.UserPage
	Cursor  domain.Cursor
	Refresh bool
}

// mediaTabMsg is one answer to a media-tab request.
type mediaTabMsg struct {
	Frame   string
	Page    domain.PostPage
	Cursor  domain.Cursor
	Refresh bool
}

// directory is the port these four reads go through, or nil when this build has none. A nil port
// is a presence fact, not a wiring mistake: the frame says the list is unavailable and spends
// nothing, the same way a build without a translation endpoint answers `T`.
func (m Model) directory() domain.Directory { return m.deps.Directory }

// pushAccounts opens one account list of the account the top frame is about.
func (m Model) pushAccounts(kind domain.UserListKind) (Model, tea.Cmd) {
	owner, ok := m.frameOwner()
	if !ok || owner.ID == "" {
		// The list is addressed by the numeric id the header carries. Before the header arrives
		// there is nothing to ask with, and inventing a lookup from the handle would be a second
		// request for a fact the profile frame is already fetching.
		return m.setNotice(render.NoticeGated, 0), nil
	}
	if m.directory() == nil {
		return m.setNotice(render.NoticeGated, 0), nil
	}
	results, effect := screen.NewUserResults(kind, owner.ID, owner.ScreenName)
	m.seq++
	id := accountsID(kind, owner.ID, m.seq)
	m.frames = append(m.frames, frame{
		ID:           id,
		Kind:         frameAccounts,
		Accounts:     results,
		AccountsKind: kind,
		Owner:        owner,
	})
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(m.frames[len(m.frames)-1], effect, false)
}

// pushMedia opens the account's media tab under the same rules.
func (m Model) pushMedia() (Model, tea.Cmd) {
	owner, ok := m.frameOwner()
	if !ok || owner.ID == "" {
		// The list is addressed by the numeric id the header carries. Before the header arrives
		// there is nothing to ask with, and inventing a lookup from the handle would be a second
		// request for a fact the profile frame is already fetching.
		return m.setNotice(render.NoticeGated, 0), nil
	}
	if m.directory() == nil {
		return m.setNotice(render.NoticeGated, 0), nil
	}
	results, effect := screen.NewMediaResults(owner.ID, owner.ScreenName)
	m.seq++
	id := mediaID(owner.ID, m.seq)
	m.frames = append(m.frames, frame{ID: id, Kind: frameMedia, Media: results, Owner: owner})
	if !m.requestAllowed(m.deps.Now()) {
		return m, nil
	}
	m = m.setNotice(render.NoticeNone, 0)
	return m.request(m.frames[len(m.frames)-1], effect, false)
}

// frameOwner is the account the top frame is about: a profile's header, or the owner an account
// list or media frame already carries. It is what turns "this account's following" into a request:
// the header the profile read is the only lookup, and a list opened from a list reuses the same id.
func (m Model) frameOwner() (domain.User, bool) {
	f := m.top()
	switch f.Kind {
	case frameProfile:
		return m.profileAccount(f)
	case frameAccounts, frameMedia:
		if f.Owner.ID != "" {
			return f.Owner, true
		}
	}
	return domain.User{}, false
}

// fetchAccounts issues one account-list request, choosing the method by the list the request names.
// One command for the three lists because they differ in one call and nothing else: a second and a
// third near-copy would be three places for the page count or the failure mapping to drift.
func (m Model) fetchAccounts(id string, req screen.UsersRequest, refresh bool) tea.Cmd {
	dir := m.directory()
	return func() tea.Msg {
		var (
			page domain.UserPage
			err  error
		)
		switch req.Kind {
		case domain.FollowersList:
			page, err = dir.Followers(context.Background(), req.UserID, req.Cursor)
		case domain.VerifiedFollowersList:
			page, err = dir.VerifiedFollowers(context.Background(), req.UserID, req.Cursor)
		default:
			page, err = dir.Following(context.Background(), req.UserID, req.Cursor)
		}
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return usersMsg{Frame: id, Kind: req.Kind, Page: page, Cursor: req.Cursor, Refresh: refresh}
	}
}

// fetchMedia issues one media-tab request.
func (m Model) fetchMedia(id string, req screen.MediaRequest, refresh bool) tea.Cmd {
	dir := m.directory()
	return func() tea.Msg {
		page, err := dir.Media(context.Background(), req.UserID, req.Cursor)
		if err != nil {
			return failureMsg{Frame: id, Failure: m.failureOf(err)}
		}
		return mediaTabMsg{Frame: id, Page: page, Cursor: req.Cursor, Refresh: refresh}
	}
}

// accountsAnswered absorbs one account-list page.
func (m Model) accountsAnswered(msg usersMsg) Model {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m
	}
	m.frames[i].Read = true
	event := screen.Event(screen.ResultsFetched[domain.User, domain.UserPage]{
		Page: msg.Page, Rows: msg.Page.Rows, Next: msg.Page.Next,
	})
	if msg.Refresh {
		event = screen.ResultsRefreshed[domain.User, domain.UserPage]{
			Page: msg.Page, Rows: msg.Page.Rows, Next: msg.Page.Next,
		}
	}
	next, _ := m.frames[i].Accounts.Update(event, m.deps.Now())
	m.frames[i].Accounts = next
	m.failure = domain.Failure{}
	if msg.Refresh {
		if n := next.Walk().NewRows(); n > 0 {
			m = m.setNotice(render.NoticeNewPosts, n)
		} else {
			m = m.setNotice(render.NoticeNoNewPosts, 0)
		}
	}
	return m
}

// mediaTabAnswered absorbs one media-tab page.
func (m Model) mediaTabAnswered(msg mediaTabMsg) Model {
	m = m.release()
	i := m.frameIndexID(msg.Frame)
	if i < 0 {
		return m
	}
	m.frames[i].Read = true
	event := screen.Event(screen.ResultsFetched[domain.Post, domain.PostPage]{
		Page: msg.Page, Rows: msg.Page.Posts, Next: msg.Page.Next,
	})
	if msg.Refresh {
		event = screen.ResultsRefreshed[domain.Post, domain.PostPage]{
			Page: msg.Page, Rows: msg.Page.Posts, Next: msg.Page.Next,
		}
	}
	next, _ := m.frames[i].Media.Update(event, m.deps.Now())
	m.frames[i].Media = next
	m.failure = domain.Failure{}
	if msg.Refresh {
		if n := next.Walk().NewRows(); n > 0 {
			m = m.setNotice(render.NoticeNewPosts, n)
		} else {
			m = m.setNotice(render.NoticeNoNewPosts, 0)
		}
	}
	return m
}
