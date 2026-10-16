package screen

import (
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Profile is one account on screen: the header the first request returned, and that account's
// recent posts on the same Walk the timeline and the search frame use. The frame carries the
// handle, whether the header arrived, the header itself, the walk, and which of the two refresh
// requests is out. Everything else — the page cache, the dedupe set, the cursor, the stall flag,
// the latch, the failure, the selection, the anchor — is the walk's and exists once, in walk.go.
type Profile struct {
	handle     string
	header     domain.Profile
	haveHeader bool
	walk       Walk[domain.Post, domain.PostPage]
	// refreshing is true while an `r` pair is in flight: the header request and then page one.
	// A second `r` while it is true spends nothing.
	refreshing bool
	// refreshAwaitingPosts is true after a refresh's header arrived and before its posts answer:
	// the posts answer then folds through RefreshPrepend rather than absorb.
	refreshAwaitingPosts bool
	// headerFailure is a failed header's own failure. It is distinct from the walk's failure so a
	// posts failure after a successful header cannot erase that the header is on screen, and so a
	// failed header spends one request and issues no posts request.
	headerFailure domain.Failure
}

// HeaderFetched is the header answer arriving.
type HeaderFetched struct{ Header domain.Profile }

// HeaderFailed is a header request that did not deliver data.
type HeaderFailed struct{ Failure domain.Failure }

// PostsFetched is one page of the account's posts arriving.
type PostsFetched struct{ Page domain.PostPage }

// PostsFailed is a posts request that did not deliver data.
type PostsFailed struct{ Failure domain.Failure }

// PostsRequested is the profile's pagination trigger: the selection reached the last loaded post
// and the walk has somewhere to go.
type PostsRequested struct{}

// ProfileRefreshRequested is `r` in a profile frame: the header again, then page one.
type ProfileRefreshRequested struct{}

// PostsRefreshed is the posts answer to a refresh: page one again, prepended.
type PostsRefreshed struct{ Page domain.PostPage }

func (HeaderFetched) event()           {}
func (HeaderFailed) event()            {}
func (PostsFetched) event()            {}
func (PostsFailed) event()             {}
func (PostsRequested) event()          {}
func (ProfileRefreshRequested) event() {}
func (PostsRefreshed) event()          {}

// Open returns the frame and the FIRST effect only: the header request. The posts request is
// issued by Update when the header arrives, so the two requests are ordered by the frame's state
// and not by the shell's convenience.
func Open(handle string) (Profile, Effect) {
	p := Profile{handle: handle}
	p.walk = p.walk.keys(postKey)
	return p, Effect{Kind: EffectPage, Req: ProfileRequest{Handle: handle}}
}

// Handle is the account the frame was opened on.
func (p Profile) Handle() string { return p.handle }

// Header returns the header and whether it has arrived.
func (p Profile) Header() (domain.Profile, bool) { return p.header, p.haveHeader }

// Walk is the frame's post list state.
func (p Profile) Walk() Walk[domain.Post, domain.PostPage] { return p.walk }

// Refreshing reports whether an `r` pair is out.
func (p Profile) Refreshing() bool { return p.refreshing || p.refreshAwaitingPosts }

// HeaderFailure is a failed header's failure, FailureNone when the header succeeded or has not
// been asked for yet.
func (p Profile) HeaderFailure() domain.Failure { return p.headerFailure }

// Move walks a row sequence by a step, clamped at both ends.
func (p Profile) Move(seq []Target, step int) Profile {
	p.walk = p.walk.Move(seq, step)
	return p
}

// WithAnchor returns the frame with the anchor the shell worked out.
func (p Profile) WithAnchor(anchor int) Profile {
	p.walk = p.walk.WithAnchor(anchor)
	return p
}

// AtEnd reports whether the selection is on the last loaded post and the walk has somewhere to go.
func (p Profile) AtEnd() bool { return p.walk.AtEnd(p.walk.Selected()) }

// Update applies one event and returns the next state with the single effect it asks for.
func (p Profile) Update(ev Event, now time.Time) (Profile, Effect) {
	p.walk = p.walk.keys(postKey)
	switch e := ev.(type) {
	case HeaderFetched:
		p.header = e.Header
		p.haveHeader = true
		p.headerFailure = domain.Failure{}
		req := UserPostsRequest{ID: e.Header.User.ID}
		if p.refreshing {
			// The `r` pair's own second half. The walk was latched by Refresh when the pair went out
			// and the header request is the first half, so the posts request is issued without
			// touching the latch: the pair is one outstanding read either way.
			p.refreshing = false
			p.refreshAwaitingPosts = true
			return p, Effect{Kind: EffectPage, Req: req}
		}
		// The frame's first post page. It is asked for through the walk's own gate — the same gate
		// every other page request goes through — which is what latches the walk while the request is
		// out: the shell reads that latch to say `loading posts` rather than claiming an empty read
		// succeeded (05-VERIFICATION 02.2), and the walk releases it in absorb.
		next, effect := p.walk.Page(req, now)
		p.walk = next
		return p, effect
	case HeaderFailed:
		p.headerFailure = e.Failure
		p.refreshing = false
		p.refreshAwaitingPosts = false
		return p, Effect{}
	case PostsFetched:
		if p.refreshAwaitingPosts {
			p.refreshAwaitingPosts = false
			p.walk = p.walk.refresh(e.Page.Posts, e.Page.Next, RefreshPrepend)
			return p, Effect{}
		}
		p.walk = p.walk.absorb(e.Page, e.Page.Posts, e.Page.Next)
		return p, Effect{}
	case PostsRefreshed:
		p.refreshAwaitingPosts = false
		p.walk = p.walk.refresh(e.Page.Posts, e.Page.Next, RefreshPrepend)
		return p, Effect{}
	case PostsFailed:
		p.refreshAwaitingPosts = false
		p.walk = p.walk.pageFailed(e.Failure)
		return p, Effect{}
	case PostsRequested:
		if !p.haveHeader {
			return p, Effect{}
		}
		next, effect := p.walk.Page(UserPostsRequest{ID: p.header.User.ID}.At(p.walk.Next()), now)
		p.walk = next
		return p, effect
	case ProfileRefreshRequested:
		if p.refreshing || p.refreshAwaitingPosts || p.walk.InFlight() {
			return p, Effect{}
		}
		// Gate like a refresh: refuse when the last failure forbids a request.
		if effect := (Effect{}); p.walk.InFlight() {
			return p, effect
		}
		next, effect := p.walk.Refresh(ProfileRequest{Handle: p.handle}, now)
		if effect.Kind == EffectNone {
			return p, Effect{}
		}
		p.walk = next
		p.refreshing = true
		// Re-issue as the header request: Refresh above latched the walk with ProfileRequest.
		return p, Effect{Kind: EffectRefresh, Req: ProfileRequest{Handle: p.handle}, Head: effect.Head}
	}
	return p, Effect{}
}
