package xsource

import (
	"context"
	"encoding/json"

	"github.com/LZD121/x-terminal/internal/domain"
)

// The four operations a profile screen can reach beyond its header and its recent posts. They are
// named here rather than spelled at each call so that the registry is the only place an id lives
// (XAPI-02) and a rename is one edit.
const (
	opFollowing       = "Following"
	opFollowers       = "Followers"
	opVerifiedFollows = "BlueVerifiedFollowers"
	opUserMedia       = "UserMedia"
)

// userListPath is where all three account lists answer: the same user-timeline container UserTweets
// uses, carrying TimelineUser entries instead of tweets. Measured on this host on 2026-10-17
// through the operator's own session and proxy — Following, Followers and BlueVerifiedFollowers each
// answered 200 with their entries under `data.user.result.timeline.timeline.instructions`.
const userListPath = "data.user.result.timeline.timeline.instructions"

// ParseUserPage reads one page of accounts out of an account-list response body: Following,
// Followers or BlueVerifiedFollowers.
//
// It is the same instruction walk the timelines and the People product use — one entry loop, one
// cursor reader, one TimelineUser arm (peoplePage) — so a shape change has exactly one home. A body
// with neither accounts nor cursors is a typed error naming the path, never an empty page.
func ParseUserPage(raw []byte) (domain.UserPage, error) {
	var envelope timelineEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.UserPage{}, unreadable("the response body", err)
	}
	timeline, path := timelineFrom(envelope)
	if timeline == nil {
		return domain.UserPage{}, missing(userListPath)
	}

	var page peoplePage
	if err := walkInstructions(path, timeline, &page); err != nil {
		return domain.UserPage{}, err
	}
	return domain.UserPage{
		Rows:      page.Users,
		Next:      domain.Cursor(page.BottomCursor),
		Requested: page.Requested,
	}, nil
}

// userList returns one page of one account list. The three lists differ only in the operation they
// name; the variables are the measured set the web client sends (userId, count, cursor, and the
// promoted-content flag the upstream's own validation asked for on UserTweets).
//
// Followers is answerable only over POST — measured on this host on 2026-10-17: GET answered 404
// with a zero-byte body, POST answered 200 with 44 TimelineUser entries under the same id,
// variables, features, field toggles and headers. That row lives in internal/xweb's post table,
// where every wire method is decided, and this function does not know about it.
func (a *Adapter) userList(ctx context.Context, op, userID string, c domain.Cursor) (domain.UserPage, error) {
	raw, err := a.client.GraphQL(ctx, op, map[string]any{
		"userId":                 userID,
		"count":                  pageCount,
		"cursor":                 string(c),
		"includePromotedContent": false,
	})
	if err != nil {
		return domain.UserPage{}, err
	}
	page, err := ParseUserPage(raw)
	if err != nil {
		return domain.UserPage{}, err
	}
	page.FetchedAt = now()
	return page, nil
}

// Following returns one page of the accounts the subject follows.
func (a *Adapter) Following(ctx context.Context, userID string, c domain.Cursor) (domain.UserPage, error) {
	return a.userList(ctx, opFollowing, userID, c)
}

// Followers returns one page of the accounts that follow the subject.
func (a *Adapter) Followers(ctx context.Context, userID string, c domain.Cursor) (domain.UserPage, error) {
	return a.userList(ctx, opFollowers, userID, c)
}

// VerifiedFollowers returns one page of the verified accounts among the subject's followers.
func (a *Adapter) VerifiedFollowers(ctx context.Context, userID string, c domain.Cursor) (domain.UserPage, error) {
	return a.userList(ctx, opVerifiedFollows, userID, c)
}

// Media returns one page of the subject's posts that carry an attachment.
//
// The body is read by the same ParseTimeline the profile's posts use: UserMedia answers under the
// user-timeline container with TimelineTweet entries, so the entries and cursors are the walk's own
// grammar. The variables are the web client's own five extras, measured on this host on 2026-10-17:
// the four this operation's answer was read with, and withV2Timeline, which the page's shape names.
func (a *Adapter) Media(ctx context.Context, userID string, c domain.Cursor) (domain.PostPage, error) {
	raw, err := a.client.GraphQL(ctx, opUserMedia, map[string]any{
		"userId":                 userID,
		"count":                  pageCount,
		"cursor":                 string(c),
		"includePromotedContent": true,
		"withClientEventToken":   false,
		"withBirdwatchNotes":     false,
		"withVoice":              true,
		"withV2Timeline":         true,
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
