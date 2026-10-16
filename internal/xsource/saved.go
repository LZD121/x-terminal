package xsource

import (
	"context"
	"fmt"

	"github.com/LZD121/x-terminal/internal/domain"
)

// The saved-list operations, by the names the registry holds. They are names rather than
// query ids on purpose: an id is only valid until x.com ships a new bundle, and a literal one
// here would be a stale id waiting to be compiled in (XAPI-02). An operation the registry does
// not have is a typed error from the client's lookup, never a fallback id.
const (
	opBookmarks = "Bookmarks"
	opLikes     = "Likes"
)

// Saved returns one page of a saved list — bookmarks or likes — newest first.
//
// Bookmarks and Likes are read by operation name; which id each name resolves to is the
// registry's business, including any pin the user wrote. The request variables are the
// measured ones: a count of 20 and the cursor as a string (empty for page one). Likes also
// carries the viewer's userId and includePromotedContent=false, which the live endpoint
// answered under on 2026-10-15.
//
// The body is read with the same timeline entry walk Home uses — only the operation names
// and the variables differ. A failed request is passed through unchanged: the transport has
// already classified it into a typed *xweb.APIError that names its state.
func (a *Adapter) Saved(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	op, vars, err := a.savedRequest(kind, c)
	if err != nil {
		return domain.PostPage{}, err
	}

	raw, err := a.client.GraphQL(ctx, op, vars)
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

// savedRequest picks the operation name and the measured variables for one saved kind.
func (a *Adapter) savedRequest(kind domain.SavedKind, c domain.Cursor) (string, map[string]any, error) {
	switch kind {
	case domain.Bookmark:
		return opBookmarks, map[string]any{
			"count":  pageCount,
			"cursor": string(c),
		}, nil
	case domain.Like:
		if a == nil || a.viewerID == "" {
			return "", nil, fmt.Errorf("xsource: Likes needs the viewer's account id; call WithViewerID before Saved")
		}
		return opLikes, map[string]any{
			"userId":                 a.viewerID,
			"count":                  pageCount,
			"cursor":                 string(c),
			"includePromotedContent": false,
		}, nil
	default:
		return "", nil, fmt.Errorf("xsource: saved kind %v is not one this build knows", kind)
	}
}
