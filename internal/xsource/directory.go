package xsource

import (
	"context"
	"encoding/json"
	"net/url"

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

// opExplore is Explore's own operation: one request, one screen.
const opExplore = "ExplorePage"

// exploreEnvelope is where ExplorePage puts its trends: the page's body, its initial timeline, and
// that timeline's instructions. Measured on this host on 2026-10-17 through the operator's own
// session and proxy — 96 kB, nine TimelineTrend entries among the segmented timelines.
type exploreEnvelope struct {
	Data struct {
		ExplorePage *struct {
			Body *struct {
				// InitialTimeline is one nesting deeper than a user timeline: the container under
				// it is the timelineNode the shared walk reads.
				InitialTimeline *struct {
					Timeline *timelineNode `json:"timeline"`
				} `json:"initialTimeline"`
			} `json:"body"`
		} `json:"explore_page"`
	} `json:"data"`
}

// trendNode is one Explore trend as the response carries it: the name it prints, and the link the
// web client follows, whose `q` parameter is the search the trend stands for.
type trendNode struct {
	Typename   string          `json:"__typename"`
	Name       string          `json:"name"`
	TrendURL   json.RawMessage `json:"trend_url"`
	SocialText string          `json:"social_context"`
}

// ParseTrends reads Explore's trend list out of an ExplorePage body, in the answer's order.
//
// The walk is the page's own rather than the shared timeline walk: a trend lives inside a
// TimelineTimelineModule's items, which the entry loop does not descend into, and it carries no
// cursor a walk would use. A body with no explore page is a typed error naming the path.
func ParseTrends(raw []byte) ([]domain.Trend, error) {
	var envelope exploreEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, unreadable("the response body", err)
	}
	if envelope.Data.ExplorePage == nil || envelope.Data.ExplorePage.Body == nil ||
		envelope.Data.ExplorePage.Body.InitialTimeline == nil ||
		envelope.Data.ExplorePage.Body.InitialTimeline.Timeline == nil {
		return nil, missing("data.explore_page.body.initialTimeline.timeline.timeline.instructions")
	}
	out := make([]domain.Trend, 0, 16)
	seen := map[string]bool{}
	for _, instruction := range envelope.Data.ExplorePage.Body.InitialTimeline.Timeline.instructionList() {
		for _, entry := range instruction.Entries {
			for _, content := range trendContents(entry) {
				if content.Typename != "TimelineTrend" || content.Name == "" {
					continue
				}
				query := trendQuery(content)
				if query == "" || seen[query] {
					continue
				}
				seen[query] = true
				out = append(out, domain.Trend{Name: content.Name, Query: query})
			}
		}
	}
	if len(out) == 0 {
		return nil, missing("data.explore_page: no TimelineTrend entry")
	}
	return out, nil
}

// trendContents is every trend candidate one entry carries: the entry's own itemContent, and every
// item of its module — an entry is either one thing or a list of them.
func trendContents(entry entryNode) []trendNode {
	out := make([]trendNode, 0, 4)
	if entry.Content.ItemContent != nil {
		if t, ok := itemTrend(*entry.Content.ItemContent); ok {
			out = append(out, t)
		}
	}
	for _, item := range entry.Content.Items {
		if item.Item.ItemContent != nil {
			if t, ok := itemTrend(*item.Item.ItemContent); ok {
				out = append(out, t)
			}
		}
	}
	return out
}

// itemTrend decodes one itemContent as a trend when it declares itself one.
func itemTrend(content itemContent) (trendNode, bool) {
	if content.Typename != "TimelineTrend" {
		return trendNode{}, false
	}
	return trendNode{Typename: content.Typename, Name: content.Name, TrendURL: content.TrendURL}, content.Name != ""
}

// trendQuery is the search a trend stands for.
//
// It is the `q` parameter of the trend's own link when that link is a search URL, and the name
// otherwise. Measured on this host on 2026-10-17: the live answer carries
// `{"url": "twitter://trending/<id>", "urlType": "DeepLink"}` — a deep link with no query in it —
// so the name is what a trend is searched by, which is also what the web client does when a trend
// row is clicked.
func trendQuery(node trendNode) string {
	if raw := trendURLString(node.TrendURL); raw != "" {
		if u, err := url.Parse(raw); err == nil {
			if q := u.Query().Get("q"); q != "" {
				return q
			}
		}
	}
	return node.Name
}

// trendURLString reads a trend_url in either spelling: a bare string, or an object with a `url`.
func trendURLString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.URL
	}
	return ""
}

// Trends returns Explore's trend list.
func (a *Adapter) Trends(ctx context.Context) ([]domain.Trend, error) {
	raw, err := a.client.GraphQL(ctx, opExplore, map[string]any{})
	if err != nil {
		return nil, err
	}
	return ParseTrends(raw)
}
