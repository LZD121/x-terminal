package screen

import (
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Results is one query's results on screen: the posts or accounts a search returned, in the
// answer's order, with the product they were asked under. It is ONE implementation for both
// entity kinds and both products — the account instantiation is Results[domain.User,
// domain.UserPage], and it is this type, not a second reducer.
//
// The frame carries exactly four things: the query, the product, the walk and the request it
// re-issues. Everything else — the page cache, the dedupe set, the cursor, the stall flag, the
// latch, the failure, the selection, the anchor — is the walk's and exists once, in walk.go. The
// list's order is the answer's order: nothing here sorts, ranks or numbers a row, because a `top`
// page is the upstream's ranking and re-ordering it would be a lie about what was asked.
type Results[T any, P any] struct {
	query   string
	product domain.SearchProduct
	walk    Walk[T, P]
	req     Request
}

// ResultsRequested is the results frame's pagination trigger: the selection reached the last
// loaded row and the walk has somewhere to go.
type ResultsRequested struct{}

// ResultsRefreshRequested is `r` in a results frame: page one again, same query, same product.
type ResultsRefreshRequested struct{}

// ResultsFailed is a results frame's request that did not deliver data — the failure the status
// line prints, already named by the classifier in internal/cli.
type ResultsFailed struct{ Failure domain.Failure }

func (ResultsRequested) event()        {}
func (ResultsRefreshRequested) event() {}
func (ResultsFailed) event()           {}

// NewResults opens the frame and returns the ONE effect that opens it: page one, empty cursor.
// The walk's latch is not set here, exactly as the home list's is not for the shell's first
// request — the process's own outstanding count is the gate until the first answer arrives, and
// the walk's gates take over from there.
func NewResults[T any, P any](query string, p domain.SearchProduct, req Request) (Results[T, P], Effect) {
	r := Results[T, P]{query: query, product: p, req: req}
	return r, Effect{Kind: EffectPage, Req: req}
}

// NewPostResults opens a results frame over posts: the walk keyed by the post id the element
// supplies, which is the one fact about the row the walk cannot derive for itself.
func NewPostResults(query string, p domain.SearchProduct) (Results[domain.Post, domain.PostPage], Effect) {
	r, effect := NewResults[domain.Post, domain.PostPage](query, p, SearchRequest{Query: query, Product: p})
	r.walk = r.walk.keys(postKey)
	return r, effect
}

// NewPeopleResults opens a results frame over accounts: the SAME reducer and the SAME walk,
// keyed by the account id each element supplies. No second cursor, dedupe set or stall rule.
func NewPeopleResults(query string) (Results[domain.User, domain.UserPage], Effect) {
	r, effect := NewResults[domain.User, domain.UserPage](query, domain.People, PeopleRequest{Query: query})
	r.walk = r.walk.keys(userKey)
	return r, effect
}

// NewUserResults opens an account-list frame over the same reducer and the same walk the People
// product uses, keyed by the account id. The label is what the status line prints — the owner's
// handle — and it is carried in the query field because that is the frame's one display string:
// Results has no second place to put a title, and a second place would be a second thing to keep
// in step.
func NewUserResults(kind domain.UserListKind, userID, label string) (Results[domain.User, domain.UserPage], Effect) {
	r, effect := NewResults[domain.User, domain.UserPage](label, domain.People, UsersRequest{Kind: kind, UserID: userID})
	r.walk = r.walk.keys(userKey)
	return r, effect
}

// NewMediaResults opens a media-tab frame: posts, the same walk and the same absorb rule every
// list of posts uses, addressed by the account id the media read takes.
func NewMediaResults(userID, label string) (Results[domain.Post, domain.PostPage], Effect) {
	r, effect := NewResults[domain.Post, domain.PostPage](label, domain.Latest, MediaRequest{UserID: userID})
	r.walk = r.walk.keys(postKey)
	return r, effect
}

// userKey is the people list's row key// userKey is the people list's row key: an account's own id, as the string the walk stores.
func userKey(u domain.User) string { return u.ID }

// Rows is the frame's rows in list order. It is a convenience for the shell's people dispatch;
// the walk's own Rows is the same slice.
func (r Results[T, P]) Rows() []T { return r.walk.Rows() }

// Query is the text the search was submitted with. It is carried so the status line can name it
// and `/` can re-seed the prompt with it; it is never written anywhere else.
func (r Results[T, P]) Query() string { return r.query }

// Product is the ranking the search was asked under.
func (r Results[T, P]) Product() domain.SearchProduct { return r.product }

// Walk is the frame's list state: the rows, the cursor, the latch, the failure, the selection and
// the anchor, read-only from here. The shell renders and pages through it.
func (r Results[T, P]) Walk() Walk[T, P] { return r.walk }

// Move walks a row sequence by a step, clamped at both ends. The sequence is the caller's — the
// frame is generic over the row, so the shell builds the order its rows render in and the walk
// moves within it.
func (r Results[T, P]) Move(seq []Target, step int) Results[T, P] {
	r.walk = r.walk.Move(seq, step)
	return r
}

// WithAnchor returns the frame with the anchor the shell worked out.
func (r Results[T, P]) WithAnchor(anchor int) Results[T, P] {
	r.walk = r.walk.WithAnchor(anchor)
	return r
}

// AtEnd reports whether the selection is on the last loaded row and the walk has somewhere to go:
// the state in which the shell asks for the next page.
func (r Results[T, P]) AtEnd() bool { return r.walk.AtEnd(r.walk.Selected()) }

// Update applies one event and returns the next state with the single effect it asks for. It is
// translation only: the frame's own events fold into the walk's Page, Refresh, absorb, refresh
// and pageFailed, and an event this frame does not know — including a nil one and the other
// frames' own — is a no-op. The request is re-issued through the union's At, so this file never
// names a cursor.
func (r Results[T, P]) Update(ev Event, now time.Time) (Results[T, P], Effect) {
	switch e := ev.(type) {
	case ResultsRequested:
		next, effect := r.walk.Page(r.req.At(r.walk.Next()), now)
		r.walk = next
		return r, effect
	case ResultsRefreshRequested:
		next, effect := r.walk.Refresh(r.req.At(""), now)
		r.walk = next
		return r, effect
	case ResultsFetched[T, P]:
		r.walk = r.walk.absorb(e.Page, e.Rows, e.Next)
		return r, Effect{}
	case ResultsRefreshed[T, P]:
		// The home rule: what the answer carries that the walk does not is prepended, the count
		// is the notice, and the selection, the pages and the walk's cursor stay where they were.
		r.walk = r.walk.refresh(e.Rows, e.Next, RefreshPrepend)
		return r, Effect{}
	case ResultsFailed:
		r.walk = r.walk.pageFailed(e.Failure)
		return r, Effect{}
	}
	return r, Effect{}
}
