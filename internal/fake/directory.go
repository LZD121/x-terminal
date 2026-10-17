package fake

import (
	"context"
	"sync"

	"github.com/LZD121/x-terminal/internal/domain"
)

// DirectoryCall is one recorded account-list or media request: which read, whose, and the cursor.
// The kind is recorded rather than inferred from the method because the three account methods are
// one read with three names, and a double that could not tell `f` from `e` could not prove the keys
// reach different operations.
type DirectoryCall struct {
	Kind   domain.UserListKind
	UserID string
	Cursor domain.Cursor
	Media  bool
}

// Directory is the recording domain.Directory the shell's tests are driven by.
//
// It answers with the same scripted page for every call and records each one, so a test can assert
// what a keypress asked for and how many times — without a socket, and without the shell being able
// to see a wire shape through it. Users is what the three account lists answer with; Posts is what
// the media tab answerers with.
type Directory struct {
	mu sync.Mutex
	// seen is every call in order.
	seen []DirectoryCall
	// Users is the account page every account-list call returns.
	Users domain.UserPage
	// Posts is the posts page the media call returns.
	Posts domain.PostPage
	// TrendList is what the Explore read answers with. It is not `Trends`: that name is the method.
	TrendList []domain.Trend
	// Err, when non-nil, is returned by every call in place of a page.
	Err error
}

// record appends one call and returns the scripted answer.
func (d *Directory) record(c DirectoryCall) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen = append(d.seen, c)
	return d.Err
}

// Calls is every call the double saw, in order.
func (d *Directory) Calls() []DirectoryCall {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]DirectoryCall(nil), d.seen...)
}

// Following answers the following list.
func (d *Directory) Following(_ context.Context, userID string, c domain.Cursor) (domain.UserPage, error) {
	if err := d.record(DirectoryCall{Kind: domain.FollowingList, UserID: userID, Cursor: c}); err != nil {
		return domain.UserPage{}, err
	}
	return d.Users, nil
}

// Followers answers the followers list.
func (d *Directory) Followers(_ context.Context, userID string, c domain.Cursor) (domain.UserPage, error) {
	if err := d.record(DirectoryCall{Kind: domain.FollowersList, UserID: userID, Cursor: c}); err != nil {
		return domain.UserPage{}, err
	}
	return d.Users, nil
}

// VerifiedFollowers answers the verified-followers list.
func (d *Directory) VerifiedFollowers(_ context.Context, userID string, c domain.Cursor) (domain.UserPage, error) {
	if err := d.record(DirectoryCall{Kind: domain.VerifiedFollowersList, UserID: userID, Cursor: c}); err != nil {
		return domain.UserPage{}, err
	}
	return d.Users, nil
}

// Trends answers Explore's list.
func (d *Directory) Trends(_ context.Context) ([]domain.Trend, error) {
	if err := d.record(DirectoryCall{}); err != nil {
		return nil, err
	}
	return d.TrendList, nil
}

// Media answers the media tab.
func (d *Directory) Media(_ context.Context, userID string, c domain.Cursor) (domain.PostPage, error) {
	if err := d.record(DirectoryCall{UserID: userID, Cursor: c, Media: true}); err != nil {
		return domain.PostPage{}, err
	}
	return d.Posts, nil
}
