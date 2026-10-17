package domain

import "context"

// UserListKind is which account list a page was asked for. The three are separate operations on
// the wire and separate questions to a reader — "who does this person follow", "who follows them",
// and "which of their followers X has verified" are not one list with a filter — so they are three
// values rather than one with a flag.
//
// The zero value is Following because that is the list a profile screen offers first, in the same
// way Latest is the zero search product.
type UserListKind uint8

const (
	// FollowingList is the accounts the subject follows.
	FollowingList UserListKind = iota
	// FollowersList is the accounts that follow the subject.
	FollowersList
	// VerifiedFollowersList is the verified accounts among the subject's followers: X's own
	// subset, not a filter this program applies to FollowersList.
	VerifiedFollowersList
)

// String names the list in the words a status line prints.
func (k UserListKind) String() string {
	switch k {
	case FollowersList:
		return "followers"
	case VerifiedFollowersList:
		return "verified followers"
	}
	return "following"
}

// Directory is the account-list and media reads: the four questions a profile screen can ask about
// one account beyond its header and its recent posts.
//
// It is an interface of its own rather than four more methods on Source because its presence is a
// fact the build can be without — a registry that has lost an operation is a list this build cannot
// reach, and the screen says so instead of a Source that would have to answer with a lie. The
// Save/Translate precedent: a nil Directory is a shell with no lists, not a wiring mistake.
//
// Every method is addressed by the numeric account id, because that is what the operations take:
// the profile header is what turns a handle into an id, and a second lookup here would be a second
// request for a fact already on screen.
type Directory interface {
	// Following returns one page of the accounts the subject follows.
	Following(ctx context.Context, userID string, c Cursor) (UserPage, error)
	// Followers returns one page of the accounts that follow the subject.
	Followers(ctx context.Context, userID string, c Cursor) (UserPage, error)
	// VerifiedFollowers returns one page of the verified accounts among the subject's followers.
	VerifiedFollowers(ctx context.Context, userID string, c Cursor) (UserPage, error)
	// Media returns one page of the subject's posts that carry an attachment.
	Media(ctx context.Context, userID string, c Cursor) (PostPage, error)
}
