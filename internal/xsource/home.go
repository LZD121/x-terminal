package xsource

import (
	"context"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// GraphQLer is the one call this adapter makes of the transport: one GraphQL operation by name,
// answered with a response body or with an error that already names its state.
//
// It is an interface here rather than a *xweb.Client so that the adapter can be driven by a
// recorded body in a test with no socket, and so that this package never depends on the HTTP
// layer's concrete type. *xweb.Client satisfies it as it stands, so the composition root passes
// one without an adapter type of its own.
type GraphQLer interface {
	GraphQL(ctx context.Context, opName string, variables map[string]any) ([]byte, error)
}

// Adapter is the production domain.Source: it asks the transport for a page of a timeline or for
// one account, and reads the answer with this package's parsers. Viewer names the session's
// account so the mirror can record whose list it holds; the sync path is the only caller.
//
// It carries no state, no cache and no retry policy. Everything about the request that could go
// stale (the query id, the feature switches, the field toggles) is resolved from the volatile
// registry inside the transport, and everything about the answer's shape is resolved by the
// parsers: this type is the seam between the two, and nothing else.
type Adapter struct {
	client GraphQLer
	// viewerID is the account id Likes asks for. Bookmarks does not need it; Likes refuses
	// without it rather than inventing one. Set with WithViewerID at the composition root
	// once the session's own account is known.
	viewerID string
	// translator is the REST translation endpoint, if this build has one. Nil means Translate
	// answers ErrNoTranslator: a composition root that did not wire one is a wiring fact, and the
	// absence is reportable rather than a panic in a keypress. Set with WithTranslator.
	translator Translator
}

// The operations this adapter reads, by the names the registry holds. They are names rather than
// query ids on purpose: an id is only valid until x.com ships a new bundle, and a literal one here
// would be a stale id waiting to be compiled in (XAPI-02). An operation the registry does not
// have is a typed error from the client's lookup, never a fallback id.
const (
	opHomeLatest = "HomeLatestTimeline"
	opHome       = "HomeTimeline"
	opUser       = "UserByScreenName"
)

// pageCount is how many entries one page asks for. It is the number the operations were measured
// answering under, and the one constant every paged read shares; nothing in this program depends
// on the upstream honoring it.
const pageCount = 20

// now is the clock a page is stamped with. It is a package variable so that a test can freeze it:
// FetchedAt is carried into the list and from there into nothing this phase renders, but a page
// that quietly read the wall clock would make a recorded frame unmovable.
var now = time.Now

// New builds the adapter over one GraphQL client.
func New(client GraphQLer) *Adapter { return &Adapter{client: client} }

// WithViewerID returns a copy of the adapter that carries the account id Likes asks for.
// It is a copy rather than a mutation so a composition root can build the adapter before it
// knows the id (Viewer is a separate read) and then hand the same client to both.
func (a *Adapter) WithViewerID(id string) *Adapter {
	if a == nil {
		return &Adapter{viewerID: id}
	}
	copied := *a
	copied.viewerID = id
	return &copied
}

// Adapter implements the port the screens are written against. The assertion is here rather than
// in the tests because a signature change in one of the two packages has to fail a build rather
// than a screen run.
var _ domain.Source = (*Adapter)(nil)

// Home returns one page of a home timeline.
//
// Following reads HomeLatestTimeline and ForYou reads HomeTimeline: the registry names both, and
// which id each name resolves to is the registry's business, not this function's. The variables
// are the measured request — a count of 20, the cursor as a string (empty for the first page), an
// empty seenTweetIds array, and includePromotedContent.
//
// includePromotedContent is sent as false, and that is a measured value rather than a guess:
// measured 2026-10-15, the upstream answered this request's shape with 422
// GRAPHQL_VALIDATION_FAILED, naming that variable under path:["variable","includePromotedContent"],
// and answered 200 once it was present — with both true and false.
//
// false is what this client sends because its renderer marks no advertisement anywhere in a
// timeline, so asking for promoted content would put posts in the pane that are indistinguishable
// from the ones the reader asked for. It is not a switch that keeps advertisements out, and the
// recording says so: the following answer captured into testdata/fixture-home.json on 2026-10-15
// was requested with false and carried 12 `promoted-tweet-…` entries, and the shared entry walk
// reads them as posts like any other (the thread path drops promotions inside a conversation
// module — detail.go's promotedItemMark — and a timeline has no equivalent). false is still what is
// sent: true would ask for more of what this renderer cannot mark.
//
// A failed request is passed through unchanged. The transport has already classified it into a
// typed *xweb.APIError that names its state, and re-wrapping it here would only put this package
// between the caller and the sentence the user needs.
func (a *Adapter) Home(ctx context.Context, kind domain.HomeKind, c domain.Cursor) (domain.PostPage, error) {
	op := opHomeLatest
	if kind == domain.ForYou {
		op = opHome
	}

	raw, err := a.client.GraphQL(ctx, op, map[string]any{
		"count":                  pageCount,
		"cursor":                 string(c),
		"seenTweetIds":           []string{},
		"includePromotedContent": false,
	})
	if err != nil {
		return domain.PostPage{}, err
	}

	page, err := ParseTimeline(raw)
	if err != nil {
		return domain.PostPage{}, err
	}
	return domain.PostPage{
		Posts:     page.Posts,
		Next:      domain.Cursor(page.BottomCursor),
		Requested: page.Requested,
		FetchedAt: now(),
	}, nil
}

// User returns one account by handle (without the @).
func (a *Adapter) User(ctx context.Context, handle string) (domain.User, error) {
	raw, err := a.client.GraphQL(ctx, opUser, map[string]any{"screen_name": handle})
	if err != nil {
		return domain.User{}, err
	}
	return ParseUser(raw)
}
