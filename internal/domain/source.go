package domain

import "context"

// Source is the read side of x.com, phrased in this package's terms: a page of one of the two
// home timelines, one account by handle, one page of a keyword search, one page of matching
// accounts, one profile header, one page of that account's recent posts, and one page of a
// saved list (bookmarks or likes).
//
// It is declared here, by the code that consumes it, rather than beside either of its
// implementations: internal/xsource implements it in production (over the GraphQL client and
// the volatile registry) and internal/fake implements it in tests (from the recorded
// fixture), and internal/cli is the only package that names both.
//
// Neither method can say anything about the wire. There is no HTTP status, no response body, no
// operation name, no map, no retry count and no error type from the transport layer here:
// everything about how a page was obtained lives below the port, because the one thing the
// screens above it must never do is guess at the difference between "no posts" and "no answer".
// A failure crosses the seam as an error, and internal/cli turns it into the Failure the status
// line prints (internal/domain/failure.go).
type Source interface {
	// Home returns one page of the given home timeline. An empty cursor asks for page one;
	// the page's own Next asks for the page after it; an empty Next in the answer means the
	// timeline has nothing older.
	Home(ctx context.Context, kind HomeKind, c Cursor) (PostPage, error)
	// User returns one account by handle (without the @).
	User(ctx context.Context, handle string) (User, error)
	// Search returns one page of a keyword search. An empty cursor asks for page one; the
	// page's own Next continues; the product decides the ranking and the implementation does
	// not sort.
	Search(ctx context.Context, query string, p SearchProduct, c Cursor) (PostPage, error)
	// SearchPeople returns one page of accounts matching the query. An empty cursor asks for
	// page one; the page's own Next continues. It is a method of its own because a page of
	// accounts is not a PostPage: the walk is shared, the page value is not.
	SearchPeople(ctx context.Context, query string, c Cursor) (UserPage, error)
	// Profile returns one account's header by handle (without the @): bio, the three
	// counts and the join instant. It is one read and carries no posts; UserPosts is
	// the second read, addressed by the numeric id this answer returned.
	Profile(ctx context.Context, handle string) (Profile, error)
	// UserPosts returns one page of one account's recent posts, addressed by the
	// numeric id the header returned. An empty cursor asks for page one; the page's
	// own Next continues.
	UserPosts(ctx context.Context, id string, c Cursor) (PostPage, error)
	// Saved returns one page of a saved list, newest first; an empty cursor asks for page one
	// and an empty Next in the answer means there is nothing older. The pane never calls this
	// on open — it reads the mirror — and the sync walk (Phase 6) is what fetches through it.
	Saved(ctx context.Context, kind SavedKind, c Cursor) (PostPage, error)
}
