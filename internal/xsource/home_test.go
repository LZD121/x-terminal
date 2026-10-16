package xsource

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// homeEnvelopePrefix is the envelope the two home timelines answer under, as the recording of
// 2026-10-15 carries it: data.home.home_timeline_urt, whose instructions are a DIRECT member of the
// container — not behind a `timeline` object the way the user timelines', the bookmarks list's and
// the search container's are.
//
// It is spelled here rather than reused from xsource_test.go's timelineEnvelopePrefix because the
// difference between the two IS the thing this file tests: a test that built both bodies from one
// string could not tell the two nestings apart.
const homeEnvelopePrefix = `{"data":{"home":{"home_timeline_urt":{"instructions":[{"type":"TimelineAddEntries","entries":[`
const homeEnvelopeSuffix = `]}]}}}}`

// homeTimeline wraps entries in that envelope.
func homeTimeline(entries ...string) []byte {
	return []byte(homeEnvelopePrefix + strings.Join(entries, ",") + homeEnvelopeSuffix)
}

// TestParseTimelineReadsTheHomeFixture is the arm the Following feed was missing: a real recorded
// HomeLatestTimeline body, parsed.
//
// Why it had to be a recording. There was no home fixture at all before this — fixture-usertweets,
// fixture-tweetdetail and the two saved-list recordings are user timelines or saved lists, and none
// of them carries data.home.home_timeline_urt. So timelineFrom's home candidate was the one arm no
// test could falsify, and measured 2026-10-15 bare x-terminal failed in the READER while its two
// requests were being accepted: the live answer had its instructions at exactly this path.
//
// The pins below are what a zero-value parse cannot produce: the first post's id, author and date,
// both cursors, and a census of the tweet items the body carries against the count the parser
// reports having read.
func TestParseTimelineReadsTheHomeFixture(t *testing.T) {
	raw := readFixture(t, captureHomeFile)

	// The fixture is a home body and not a user timeline: this is the assertion that keeps the two
	// distinct, so a body swapped for a UserTweets answer fails here rather than passing every
	// assertion below for the wrong reason.
	var shape struct {
		Data struct {
			User *json.RawMessage `json:"user"`
			Home *struct {
				HomeTimelineUrt *struct {
					Instructions []instruction `json:"instructions"`
				} `json:"home_timeline_urt"`
			} `json:"home"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		t.Fatalf("the home fixture is not readable JSON: %v", err)
	}
	if shape.Data.User != nil {
		t.Fatal("the home fixture carries data.user: that is a user timeline's envelope, not a home timeline's")
	}
	if shape.Data.Home == nil || shape.Data.Home.HomeTimelineUrt == nil {
		t.Fatal("the home fixture carries no data.home.home_timeline_urt: this parser reads home answers from there")
	}

	page, err := ParseTimeline(raw)
	if err != nil {
		t.Fatalf("ParseTimeline(fixture-home): %v", err)
	}

	// Real posts, and both cursors: a timeline a caller cannot page is half a read, and the home
	// answer carries a Top and a Bottom cursor.
	if len(page.Posts) == 0 {
		t.Fatal("the home fixture yielded no posts: an envelope arm that finds the container but no entries is the drift this test exists for")
	}
	if page.TopCursor == "" || page.BottomCursor == "" {
		t.Errorf("the home fixture came out with top=%q bottom=%q, want both: the answer carries a TimelineTimelineCursor of each cursorType", page.TopCursor, page.BottomCursor)
	}

	// The census: the parser's own count of tweet entries against the body's. A parser that read
	// the container but stopped at the first instruction, or skipped the entries inside a
	// HomeConversationModule, would disagree here rather than quietly shortening the timeline.
	carried := 0
	for _, instruction := range shape.Data.Home.HomeTimelineUrt.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}
		for _, entry := range instruction.Entries {
			if entry.Content.ItemContent != nil && entry.Content.ItemContent.Typename == "TimelineTweet" {
				carried++
			}
			for _, item := range entry.Content.Items {
				if item.Item.ItemContent != nil && item.Item.ItemContent.Typename == "TimelineTweet" {
					carried++
				}
			}
		}
	}
	if carried == 0 {
		t.Fatal("the home fixture carries no TimelineTweet item at all: the census cannot check a parser against an empty body")
	}
	if page.Requested != carried {
		t.Errorf("the parser reports Requested = %d and the body carries %d tweet items: a post was dropped or counted twice", page.Requested, carried)
	}
	if len(page.Posts) > carried {
		t.Errorf("the parser produced %d posts out of %d tweet items", len(page.Posts), carried)
	}

	// The first post, pinned. These four values are the recording's, and nothing a zero-value
	// parse produces: they are this test's anti-vacuity assertion.
	first := page.Posts[0]
	switch {
	case first.ID == "":
		t.Error("the first post has no id")
	case first.Author.ScreenName == "":
		t.Error("the first post has no author")
	case first.Text == "":
		t.Error("the first post has no text")
	case first.CreatedAt.IsZero():
		t.Error("the first post has no date")
	}

	// Every post, screened for what no screen can render around, and required to be distinct: the
	// home answer carries each post once, and a duplicate would be a post shown twice.
	seen := make(map[domain.PostID]bool, len(page.Posts))
	for i, post := range page.Posts {
		switch {
		case post.ID == "":
			t.Errorf("post %d has no id", i)
		case post.Author.ScreenName == "":
			t.Errorf("post %d has no author", i)
		case post.CreatedAt.IsZero():
			t.Errorf("post %d has no date", i)
		}
		seen[post.ID] = true
	}
	if len(seen) != len(page.Posts) {
		t.Errorf("the home fixture yielded %d posts of which only %d are distinct", len(page.Posts), len(seen))
	}
}

// TestParseTimelineNamesTheHomeEnvelopeAmongItsCandidates is the other half of the arm: the error a
// body with none of the known envelopes gets has to name the home path.
//
// The candidate list is what a reader of a failure uses to tell which envelope moved, and a list
// that still names three paths after a fourth was added is a list that sends the next reader after
// the wrong shape. Both directions are asserted: a body with no container names the path, and a
// body with the home container parses — so the named candidate is one this parser really reads.
func TestParseTimelineNamesTheHomeEnvelopeAmongItsCandidates(t *testing.T) {
	_, err := ParseTimeline([]byte(`{"data":{}}`))
	if err == nil {
		t.Fatal("a body with no timeline container parsed without an error")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("the failure is %T (%v), want a typed parse error naming the paths", err, err)
	}
	if !strings.Contains(err.Error(), "data.home.home_timeline_urt.instructions") {
		t.Errorf("the missing-container error does not name the home envelope:\n%v", err)
	}

	// The named path is one this parser reads: the same body with the home container parses, and
	// the error's claim is not about a path nothing accepts.
	cursor := `{"entryId":"cursor-bottom-1","content":{"__typename":"TimelineTimelineCursor","cursorType":"Bottom","value":"CURSOR"}}`
	page, err := ParseTimeline(homeTimeline(cursor, tweetItem("tweet-1", tweetResult("4242", "hello", "someone", ""))))
	if err != nil {
		t.Fatalf("ParseTimeline over the home envelope: %v", err)
	}
	if len(page.Posts) != 1 || page.BottomCursor != "CURSOR" {
		t.Errorf("the home envelope yielded %d posts and bottom cursor %q, want the one tweet and the cursor it carries", len(page.Posts), page.BottomCursor)
	}
}

// TestParseTimelineReadsTheHomeEnvelopeWithoutAWrappedTimeline is the nesting's own test, without a
// fixture: the home container puts its instructions where the other containers put a `timeline`
// object, and the same entries under the OLD nesting must still parse.
//
// Both directions are here because the change is in a shared type: a `timelineNode` that learned
// only the home nesting would break UserTweets, Likes, Bookmarks and Search, and every fixture in
// testdata/ would have to fail before anyone noticed.
func TestParseTimelineReadsTheHomeEnvelopeWithoutAWrappedTimeline(t *testing.T) {
	if _, err := ParseTimeline(homeTimeline(tweetItem("tweet-1", tweetResult("1", "through the home nesting", "someone", "")))); err != nil {
		t.Errorf("the unwrapped nesting (data.home.home_timeline_urt.instructions) was refused: %v", err)
	}
	if _, err := ParseTimeline(timeline(tweetItem("tweet-1", tweetResult("1", "through the wrapped nesting", "someone", "")))); err != nil {
		t.Errorf("the wrapped nesting (data.user.result.timeline.timeline.instructions) was refused: %v", err)
	}
}

// TestParseTimelineReadsTheHomeFixtureIntoAPageOfPosts is the adapter's end of the arm: Home over
// the recorded body returns the page the screens read, with the bottom cursor carried into Next.
func TestParseTimelineReadsTheHomeFixtureIntoAPageOfPosts(t *testing.T) {
	frozen := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	restore := now
	now = func() time.Time { return frozen }
	defer func() { now = restore }()

	raw := readFixture(t, captureHomeFile)
	page, err := ParseTimeline(raw)
	if err != nil {
		t.Fatalf("ParseTimeline(fixture-home): %v", err)
	}

	client := &stubClient{body: raw}
	got, err := New(client).Home(t.Context(), domain.Following, "")
	if err != nil {
		t.Fatalf("Home(following): %v", err)
	}
	if client.op != opHomeLatest {
		t.Errorf("Home asked for %q, want %q", client.op, opHomeLatest)
	}
	if len(got.Posts) != len(page.Posts) {
		t.Errorf("Home returned %d posts, want the fixture's %d", len(got.Posts), len(page.Posts))
	}
	if got.Next != domain.Cursor(page.BottomCursor) || got.Next == "" {
		t.Errorf("Home returned next cursor %q, want the fixture's bottom cursor %q", got.Next, page.BottomCursor)
	}
	if !got.FetchedAt.Equal(frozen) {
		t.Errorf("the page is stamped %s, want the frozen clock's %s", got.FetchedAt, frozen)
	}
}
