package xsource

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// opUserTweets is the account-timeline operation, by the name the registry holds.
const opUserTweets = "UserTweets"

// ParseProfile reads one account's header out of a UserByScreenName response body.
//
// Identity (rest_id, core.screen_name) is required exactly as ParseUser requires it. Bio, the
// three counts and the join instant are optional: an absent field leaves the matching Profile
// field absent (a nil count pointer, an empty bio, a zero Joined), and an explicit zero count
// sets a pointer to zero — which is the whole of the screen's "omit, never zero" rule at the
// model level. created_at is parsed here into time.Time; nothing above the port sees a string.
func ParseProfile(raw []byte) (domain.Profile, error) {
	var envelope userEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.Profile{}, unreadable("the response body", err)
	}

	var firstErr error
	for _, candidate := range []struct {
		path string
		node *userNode
	}{
		{"data.user.result", envelope.Data.User.Result},
		{"data.viewer.user_results.result", envelope.Data.Viewer.UserResults.Result},
	} {
		if candidate.node == nil {
			continue
		}
		p, err := profileFrom(candidate.path, candidate.node)
		if err == nil {
			return p, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return domain.Profile{}, firstErr
	}
	return domain.Profile{}, missing("data.user.result (and data.viewer.user_results.result)")
}

func profileFrom(path string, node *userNode) (domain.Profile, error) {
	user, err := userFrom(path, node)
	if err != nil {
		return domain.Profile{}, err
	}
	p := domain.Profile{User: user}
	if node.ProfileBio != nil {
		p.Bio = node.ProfileBio.Description
	}
	if node.RelationshipCounts != nil {
		p.Followers = node.RelationshipCounts.Followers
		p.Following = node.RelationshipCounts.Following
	}
	if node.TweetCounts != nil {
		p.Posts = node.TweetCounts.Tweets
	}
	if node.Core.CreatedAt != "" {
		joined, err := time.Parse(timelineTime, node.Core.CreatedAt)
		if err != nil {
			return domain.Profile{}, &ParseError{Path: path + ".core.created_at", Detail: err.Error()}
		}
		p.Joined = joined.UTC()
	}
	return p, nil
}

// Profile returns one account's header by handle.
func (a *Adapter) Profile(ctx context.Context, handle string) (domain.Profile, error) {
	raw, err := a.client.GraphQL(ctx, opUser, map[string]any{"screen_name": handle})
	if err != nil {
		return domain.Profile{}, err
	}
	return ParseProfile(raw)
}

// UserPosts returns one page of one account's recent posts, addressed by the numeric id the
// header returned. The answer is read with the same ParseTimeline the home timelines use.
//
// includePromotedContent and withVoice are the names this operation's own shape asked for:
// measured 2026-10-15, UserTweets with userId/count/cursor answered 422
// GRAPHQL_VALIDATION_FAILED naming includePromotedContent under path:["variable",…], then 422
// naming withVoice, and 200 once both were present. Both are sent as false — the value the
// upstream was measured to require rather than one it was shown to demand, and the one that keeps
// promoted posts out of a list this renderer marks no advertisement in.
func (a *Adapter) UserPosts(ctx context.Context, id string, c domain.Cursor) (domain.PostPage, error) {
	raw, err := a.client.GraphQL(ctx, opUserTweets, map[string]any{
		"userId":                 id,
		"count":                  pageCount,
		"cursor":                 string(c),
		"includePromotedContent": false,
		"withVoice":              false,
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
