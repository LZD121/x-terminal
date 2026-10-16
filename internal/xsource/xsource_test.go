package xsource

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// fixturePath is a real UserTweets response, recorded with the browser's own session on
// 2026-10-14 and copied here unchanged. Everything this file asserts about the parser is
// asserted against it: a parser driven by a fixture written by the same hand as the parser
// agrees with itself and with nothing else.
const fixturePath = "testdata/fixture-usertweets.json"

// minFixtureBytes is that response's size, near enough. A fixture that has been quietly
// emptied, or replaced by a hand-written stub, fails every test in this file instead of
// passing all of them.
const minFixtureBytes = 100_000

// recordedDate is the format and value created_at carries in the recording.
const recordedDate = "Tue Oct 13 22:16:43 +0000 2026"

// The synthetic bodies below are built from these three pieces rather than written out whole:
// the envelope is the same in every response, and a test about a tweet shape should not be able
// to fail because of a brace in the response around it.
const timelineEnvelopePrefix = `{"data":{"user":{"result":{"timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[`
const timelineEnvelopeSuffix = `]}]}}}}}}`

func fixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading the recorded response: %v", err)
	}
	return data
}

// timeline wraps entries in the envelope a timeline response has.
func timeline(entries ...string) []byte {
	return []byte(timelineEnvelopePrefix + strings.Join(entries, ",") + timelineEnvelopeSuffix)
}

// tweetItem is one timeline item carrying a tweet, the shape the recording has for a standalone
// post.
func tweetItem(entryID, result string) string {
	return `{"entryId":"` + entryID + `","content":{"__typename":"TimelineTimelineItem","itemContent":{"__typename":"TimelineTweet","tweet_results":{"result":` + result + `}}}}`
}

// tweetResult is a tweet node with a body, a date and an author. extra is spliced in as
// additional fields of the tweet node: a note_tweet, a quoted tweet, a reposted one.
func tweetResult(id, text, handle, extra string) string {
	return `{"__typename":"Tweet","rest_id":"` + id + `","legacy":{"full_text":"` + text + `","created_at":"` + recordedDate + `"},"core":{"user_results":{"result":{"__typename":"User","rest_id":"1","core":{"screen_name":"` + handle + `"}}}}` + extra + `}`
}

// tweetResultWithLegacy is tweetResult with extra fields spliced into the tweet's legacy section
// instead of into the tweet itself. That is where media, conversations, reply-to ids and the
// counters live, and a test that puts them at the tweet's root would be asserting against a shape
// the upstream does not send.
func tweetResultWithLegacy(id, text, handle, legacyExtra string) string {
	return `{"__typename":"Tweet","rest_id":"` + id + `","legacy":{"full_text":"` + text + `","created_at":"` + recordedDate + `"` + legacyExtra + `},"core":{"user_results":{"result":{"__typename":"User","rest_id":"1","core":{"screen_name":"` + handle + `"}}}}` + `}`
}

func TestTheFixtureIsTheWholeRecordedResponse(t *testing.T) {
	info, err := os.Stat(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < minFixtureBytes {
		t.Fatalf("%s is %d bytes, want the recorded response's %d or so: a fixture this small cannot be the one this parser was written against, and every assertion below would pass for the wrong reason", fixturePath, info.Size(), minFixtureBytes)
	}
}

func TestParseTimelineReadsTheRecordedResponse(t *testing.T) {
	page, err := ParseTimeline(fixture(t))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}

	// The plan's floor: the recorded answer held 20 posts, and a parser that quietly found
	// none — or only the ones outside the conversation modules — fails here.
	if len(page.Posts) < 20 {
		t.Fatalf("the recorded response yielded %d posts, want at least 20", len(page.Posts))
	}
	if page.TopCursor == "" || page.BottomCursor == "" {
		t.Errorf("the cursors came out as top=%q bottom=%q, want both: a timeline nobody can page is half a parse", page.TopCursor, page.BottomCursor)
	}

	// The first post, pinned. A zero-value parse cannot produce these four values, which is
	// what makes this the file's anti-vacuity assertion.
	first := page.Posts[0]
	switch {
	case first.ID != "2110132653248672225":
		t.Errorf("the first post has id %q, want the id the recording carries", first.ID)
	case first.Text != "big feelings, small text box":
		t.Errorf("the first post's text is %q, want the text the recording carries", first.Text)
	case first.Author.ScreenName != "X":
		t.Errorf("the first post is by %q, want the handle the recording carries", first.Author.ScreenName)
	case !first.CreatedAt.Equal(time.Date(2026, 10, 13, 22, 16, 43, 0, time.UTC)):
		t.Errorf("the first post is dated %s, want 2026-10-13T22:16:43Z", first.CreatedAt)
	}

	// Every post, screened for the four things a screen cannot render around, and counted for
	// the relationships and fields this phase added.
	var quoted, withMedia, replies, described int
	for i, post := range page.Posts {
		switch {
		case post.ID == "":
			t.Errorf("post %d has no id", i)
		case post.Author.ScreenName == "":
			t.Errorf("post %d has no author", i)
		case post.Text == "":
			t.Errorf("post %d has no text", i)
		case post.CreatedAt.IsZero():
			t.Errorf("post %d has no date", i)
		}
		if post.Quote != nil {
			quoted++
			if post.Quote.Text == "" || post.Quote.Author.ScreenName == "" || post.Quote.CreatedAt.IsZero() {
				t.Errorf("post %d quotes %q, whose body, author or date is empty: the quoted post is read out of the same response as a whole post, and half of one is worse than none", i, post.Quote.ID)
			}
		}
		if len(post.Media) > 0 {
			withMedia++
			for _, media := range post.Media {
				if media.URL == "" {
					t.Errorf("post %d carries a %s attachment with no URL", i, media.Kind)
				}
			}
		}
		if post.ReplyTo != "" {
			replies++
		}
		if post.Lang != "" && post.ConversationID != "" && post.Counts != (domain.Counts{}) {
			described++
		}
	}
	switch {
	case quoted == 0:
		t.Error("no post came out with a quote, but the recording holds four posts that quote another")
	case withMedia == 0:
		t.Error("no post came out with a media ref, but the recording holds two posts with attachments")
	case replies == 0:
		t.Error("no post came out with a reply-to id, but the recording holds two posts that answer another")
	case described == 0:
		t.Error("no post came out with a language, a conversation and counts, but every recorded post carries all three")
	}

	// The answer's cost is the number of tweet entries it carried: 20 in the recording, counted
	// before this parser collapsed its two reposts and before anything above it de-duplicates.
	if page.Requested != 20 {
		t.Errorf("the recorded response reports Requested = %d, want the 20 tweet entries it carries", page.Requested)
	}

	// The quote, the media ref, the reply-to and the counts, each pinned to the post the
	// recording carries them on: a parser that read the right fields off the wrong nesting, or
	// off the wrong post, fails here rather than passing on a count.
	pinned := map[domain.PostID]domain.Post{}
	for _, post := range page.Posts {
		pinned[post.ID] = post
	}
	quoting := pinned["2109787535509799385"]
	const quotedBody = "We've unified plans across Grok, Cursor, and X Premium.\n\nIntroducing The X Subscription. https://t.co/SvkrVvBa0p"
	if quoting.Quote == nil || quoting.Quote.ID != "2109783335296008414" || quoting.Quote.Text != quotedBody {
		t.Errorf("the post that quotes 2109783335296008414 came out quoting %+v, want the nested post's whole body: the recording nests it under quoted_status_result.result, and a parser that kept only an id would leave the quote empty", quoting.Quote)
	} else if quoting.Quote.Author.ScreenName != "SpaceXSI" {
		t.Errorf("the quoted post is attributed to %q, want the handle the nested response carries", quoting.Quote.Author.ScreenName)
	}
	video := pinned["2102147636702634195"]
	const videoURL = "https://pbs.twimg.com/amplify_video_thumb/2102147574534672385/img/8u36LZa_mB2Kh5Ic.jpg"
	if len(video.Media) != 1 || video.Media[0].Kind != domain.Video || video.Media[0].URL != videoURL {
		t.Errorf("the post with a video came out with %+v, want one video ref whose URL is the recording's media_url_https", video.Media)
	}
	reply := pinned["2108304810823074061"]
	if reply.ReplyTo != "2108299253655257139" || reply.ConversationID != "2108299253655257139" {
		t.Errorf("the self-thread's second post answers %q in conversation %q, want the ids the recording carries", reply.ReplyTo, reply.ConversationID)
	}
	if got := pinned["2110132653248672225"].Counts; got != (domain.Counts{Replies: 596, Reposts: 449, Likes: 3813, Quotes: 128, Bookmarks: 139, Views: 1075958}) {
		t.Errorf("the first post's counts came out %+v, want the numbers the recording carries (views, quoted as a string, included)", got)
	}
}

func TestParseTimelineRefusesATruncatedResponse(t *testing.T) {
	truncated := fixture(t)[:512]

	page, err := ParseTimeline(truncated)
	if err == nil {
		t.Fatalf("a truncated response parsed into %d posts and no error", len(page.Posts))
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("the failure is %T (%v), want a typed parse error naming the path", err, err)
	}
	if len(page.Posts) != 0 {
		t.Errorf("a truncated response still produced %d posts", len(page.Posts))
	}
}

func TestParseTimelineRefusesAnAnswerWithNoEntries(t *testing.T) {
	// A well-formed body whose instructions carry nothing. Saying "0 posts" here is exactly how
	// a shape change becomes an empty timeline on screen.
	empty := []byte(`{"data":{"user":{"result":{"__typename":"User","timeline":{"timeline":{"instructions":[{"type":"TimelineClearCache"}]}}}}}}`)
	page, err := ParseTimeline(empty)
	if err == nil {
		t.Fatalf("a response with no entries parsed into %d posts and no error", len(page.Posts))
	}
	if !strings.Contains(err.Error(), "instructions") {
		t.Errorf("the error does not name the path: %v", err)
	}
}

func TestParseTimelineSkipsWhatIsNotAPost(t *testing.T) {
	// The recording's own shape: an item that is not a tweet (a user card) beside a tweet. The
	// card must not become a post, and it must not fail the page either.
	card := `{"entryId":"user-1","content":{"__typename":"TimelineTimelineItem","itemContent":{"__typename":"TimelineUser","user_results":{"result":{"rest_id":"99"}}}}}`
	page, err := ParseTimeline(timeline(card, tweetItem("tweet-1", tweetResult("5", "hello", "someone", ""))))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	if len(page.Posts) != 1 {
		t.Fatalf("the answer yielded %d posts, want the one tweet in it", len(page.Posts))
	}
	if page.Posts[0].Text != "hello" {
		t.Errorf("the post is %q, want the tweet's text", page.Posts[0].Text)
	}
}

func TestParseTimelineUnwrapsAConversationModule(t *testing.T) {
	// A self-thread: two tweets inside one module entry, which the upstream sends beside the
	// standalone items. Dropping them would drop posts from the middle of a timeline.
	module := `{"entryId":"profile-conversation-1","content":{"__typename":"TimelineTimelineModule","items":[` +
		`{"entryId":"profile-conversation-1-tweet-1","item":{"itemContent":{"__typename":"TimelineTweet","tweet_results":{"result":` + tweetResult("11", "first", "threader", "") + `}}}},` +
		`{"entryId":"profile-conversation-1-tweet-2","item":{"itemContent":{"__typename":"TimelineTweet","tweet_results":{"result":` + tweetResult("12", "second", "threader", "") + `}}}}` +
		`]}}`

	page, err := ParseTimeline(timeline(module))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	if len(page.Posts) != 2 || page.Posts[0].Text != "first" || page.Posts[1].Text != "second" {
		t.Fatalf("the module yielded %d posts (%v)", len(page.Posts), texts(page.Posts))
	}
}

func TestParseTimelineCollapsesARepost(t *testing.T) {
	// The response recorded from the live site on 2026-10-14 nests the original tweet under legacy
	// (18 of one account's 20 entries); the research-era shape puts it at the tweet's root. Both are
	// exercised here: reading neither renders a repost as a post whose text begins "RT @…".
	original := tweetResult("800", "original text", "original", "")
	inner := `,"retweeted_status_result":{"result":` + original + `}`

	for _, tc := range []struct{ name, result string }{
		{name: "at the tweet's root", result: tweetResult("900", "RT @original: original text", "reposter", inner)},
		{name: "under legacy, the shape the live run sent",
			result: `{"__typename":"Tweet","rest_id":"900","core":{"user_results":{"result":{"__typename":"User","rest_id":"2","core":{"screen_name":"reposter"}}}},"legacy":{"full_text":"RT @original: original text","created_at":"` + recordedDate + `"` + inner + `}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, err := ParseTimeline(timeline(tweetItem("tweet-900", tc.result)))
			if err != nil {
				t.Fatalf("ParseTimeline: %v", err)
			}
			if len(page.Posts) != 1 {
				t.Fatalf("the answer yielded %d posts, want the one repost", len(page.Posts))
			}
			post := page.Posts[0]
			switch {
			case post.ID != "800":
				t.Errorf("the repost's id is %q, want the original post's", post.ID)
			case post.Text != "original text":
				t.Errorf("the repost's text is %q, want the original's", post.Text)
			case post.Author.ScreenName != "original":
				t.Errorf("the repost is attributed to %q, want the original author", post.Author.ScreenName)
			case post.RepostedBy == nil:
				t.Fatal("the repost carries no reposted-by marker, so a screen cannot tell it from an original post")
			case post.RepostedBy.ScreenName != "reposter":
				t.Errorf("the repost is marked as by %q, want the account that reposted it", post.RepostedBy.ScreenName)
			}
		})
	}
}

func TestParseTimelinePrefersTheLongFormText(t *testing.T) {
	long := tweetResult("7", "the truncated body", "longwinded", `,"note_tweet":{"text":"the whole long post"}`)

	page, err := ParseTimeline(timeline(tweetItem("tweet-7", long)))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	if got := page.Posts[0].Text; got != "the whole long post" {
		t.Errorf("the post's text is %q, want note_tweet's, which is the whole body", got)
	}
}

func TestParseTimelineRefusesATweetItCannotRead(t *testing.T) {
	for _, tc := range []struct{ name, result, wantPath string }{
		{
			name:     "a result type this build does not know",
			result:   `{"__typename":"TweetSomethingThisBuildHasNeverSeen"}`,
			wantPath: "__typename",
		},
		{
			name:     "a tweet with no body in either place",
			result:   `{"__typename":"Tweet","rest_id":"7","legacy":{"created_at":"` + recordedDate + `"},"core":{"user_results":{"result":{"rest_id":"1","core":{"screen_name":"someone"}}}}}`,
			wantPath: "legacy.full_text",
		},
		{
			name:     "a tweet whose date is not the one this build reads",
			result:   `{"__typename":"Tweet","rest_id":"7","legacy":{"full_text":"hello","created_at":"2026-10-13T22:16:43Z"},"core":{"user_results":{"result":{"rest_id":"1","core":{"screen_name":"someone"}}}}}`,
			wantPath: "legacy.created_at",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, err := ParseTimeline(timeline(tweetItem("tweet-7", tc.result)))
			if err == nil {
				t.Fatalf("the page parsed into %d posts and no error", len(page.Posts))
			}
			if !strings.Contains(err.Error(), tc.wantPath) {
				t.Errorf("the error does not name %s: %v", tc.wantPath, err)
			}
		})
	}
}

func TestParseUserReadsTheNewSchema(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{
			name: "Viewer's shape",
			body: `{"data":{"viewer":{"user_results":{"result":{"__typename":"User","rest_id":"783214","core":{"screen_name":"X","name":"X"},"avatar":{"image_url":"https://pbs.twimg.com/profile_images/1/a_normal.jpg"},"banner":{"image_url":"https://pbs.twimg.com/profile_banners/1/b.jpg"},"has_graduated_access":true}}}}}`,
		},
		{
			name: "UserByScreenName's shape",
			body: `{"data":{"user":{"result":{"__typename":"User","rest_id":"783214","core":{"screen_name":"X","name":"X"},"avatar":{"image_url":"https://pbs.twimg.com/profile_images/1/a_normal.jpg"},"banner":{"image_url":"https://pbs.twimg.com/profile_banners/1/b.jpg"},"has_graduated_access":true}}}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user, err := ParseUser([]byte(tc.body))
			if err != nil {
				t.Fatalf("ParseUser: %v", err)
			}
			switch {
			case user.ID != "783214":
				t.Errorf("the user's id is %q, want the rest_id the response carries", user.ID)
			case user.ScreenName != "X":
				t.Errorf("the handle is %q, want the one the response carries", user.ScreenName)
			case user.Name != "X":
				t.Errorf("the name is %q, want the one the response carries", user.Name)
			case user.AvatarURL == "":
				t.Error("the avatar came out empty, but the response carries one")
			case user.BannerURL == "":
				t.Error("the banner came out empty, but the response carries one")
			case !user.HasGraduatedAccess:
				t.Error("has_graduated_access came out false, but the response says true")
			}
		})
	}
}

func TestParseUserDoesNotFallBackToTheRetiredSchema(t *testing.T) {
	// The research-era shape: everything a user had was under legacy, including the
	// followers_count the current schema does not have at all. A parser that read legacy would
	// answer this body with a user — and would answer every *current* body with an empty one,
	// which is the failure mode this whole package is written against.
	retired := []byte(`{"data":{"user":{"result":{"__typename":"User","legacy":{"screen_name":"oldname","followers_count":1234,"created_at":"` + recordedDate + `"}}}}}`)

	user, err := ParseUser(retired)
	if err == nil {
		t.Fatalf("the retired shape parsed into %+v, want an error", user)
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("the failure is %T (%v), want a typed parse error", err, err)
	}
	if !strings.Contains(err.Error(), "core.screen_name") && !strings.Contains(err.Error(), "rest_id") {
		t.Errorf("the error does not name what was missing: %v", err)
	}

	// The same body with only the handle moved under core: the parser must read core and never
	// fall back to the legacy field sitting right beside it.
	half := []byte(`{"data":{"user":{"result":{"__typename":"User","rest_id":"7","core":{"name":"Someone"},"legacy":{"screen_name":"oldname"}}}}}`)
	if user, err := ParseUser(half); err == nil || user.ScreenName != "" {
		t.Fatalf("a body whose handle exists only under legacy parsed into %+v (%v), want an error naming core.screen_name", user, err)
	}
}

func TestParseUserRefusesAUserWithoutAnIdentity(t *testing.T) {
	for _, tc := range []struct{ name, body, wantPath string }{
		{
			name:     "no id",
			body:     `{"data":{"user":{"result":{"core":{"screen_name":"someone"}}}}}`,
			wantPath: "data.user.result.rest_id",
		},
		{
			name:     "no handle",
			body:     `{"data":{"user":{"result":{"rest_id":"7","core":{"name":"Someone"}}}}}`,
			wantPath: "data.user.result.core.screen_name",
		},
		{
			name:     "no user at all",
			body:     `{"data":{}}`,
			wantPath: "data.user.result",
		},
		{
			name:     "not JSON",
			body:     `<!DOCTYPE html><html>`,
			wantPath: "the response body",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseUser([]byte(tc.body))
			if err == nil {
				t.Fatal("the body parsed into a user")
			}
			if !strings.Contains(err.Error(), tc.wantPath) {
				t.Errorf("the error does not name %s: %v", tc.wantPath, err)
			}
		})
	}
}

// texts lists the text of each post, for a failure message that says what was parsed.
func texts(posts []domain.Post) []string {
	out := make([]string, 0, len(posts))
	for _, post := range posts {
		out = append(out, post.Text)
	}
	return out
}

// TestParseTimelineReadsATombstoneAsAPost pins the shape a tombstone arrives in: no rest_id, no
// legacy, no core. It is not an error — a tombstone in a timeline is a post that used to exist,
// and a parser that refused it would turn one unreadable post into an unreadable timeline.
func TestParseTimelineReadsATombstoneAsAPost(t *testing.T) {
	body := `{"__typename":"TweetTombstone","tombstone":{"text":{"text":"This Post is unavailable.","entities":[]}}}`

	page, err := ParseTimeline(timeline(tweetItem("tweet-1234567890", body)))
	if err != nil {
		t.Fatalf("a timeline carrying a tombstone failed: %v", err)
	}
	if len(page.Posts) != 1 {
		t.Fatalf("the answer yielded %d posts, want the one tombstone", len(page.Posts))
	}

	post := page.Posts[0]
	switch {
	case post.ID != "1234567890":
		t.Errorf("the tombstone's id is %q, want the one its entry id carries: everything above this package de-duplicates on the id, and a tombstone carries none of its own in this shape", post.ID)
	case post.Visibility != domain.Unavailable:
		t.Errorf("the tombstone's visibility is %s, want unavailable", post.Visibility)
	case post.Text != "" || post.Author.ScreenName != "" || !post.CreatedAt.IsZero():
		t.Errorf("the tombstone came out as %+v, want nothing but its id and its visibility: it has no body, author or date to show", post)
	case page.Requested != 1:
		t.Errorf("Requested is %d, want 1: the answer carried one tweet entry", page.Requested)
	}

	// When the response does carry a rest_id, that is the identity — the entry id is a fallback,
	// not a preference.
	withID := `{"__typename":"TweetTombstone","rest_id":"987","tombstone":{"text":{"text":"gone"}}}`
	page, err = ParseTimeline(timeline(tweetItem("tweet-1234567890", withID)))
	if err != nil {
		t.Fatalf("a tombstone with an id failed: %v", err)
	}
	if page.Posts[0].ID != "987" {
		t.Errorf("the tombstone's id is %q, want the rest_id it carries", page.Posts[0].ID)
	}
}

// TestParseTimelineMarksALimitedPost covers the wrapper the fixture has no example of: the
// upstream still serves the body, and the wrapper is the whole evidence that the post is held
// back. Reading it as an ordinary tweet would render a limited post as a full one.
func TestParseTimelineMarksALimitedPost(t *testing.T) {
	inner := tweetResult("7", "the body is still here", "limited", "")
	wrapped := `{"__typename":"TweetWithVisibilityResults","tweet":` + inner + `}`

	page, err := ParseTimeline(timeline(tweetItem("tweet-7", wrapped)))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	post := page.Posts[0]
	switch {
	case post.ID != "7":
		t.Errorf("the limited post's id is %q, want the inner tweet's", post.ID)
	case post.Text != "the body is still here":
		t.Errorf("the limited post's body is %q, want the one the wrapper carried", post.Text)
	case post.Visibility != domain.Limited:
		t.Errorf("the wrapped post's visibility is %s, want limited", post.Visibility)
	}

	// The same shape without its inner tweet is an error, not a post with nothing in it.
	if _, err := ParseTimeline(timeline(tweetItem("tweet-7", `{"__typename":"TweetWithVisibilityResults"}`))); err == nil {
		t.Error("a visibility wrapper with no inner tweet parsed into a post")
	}
}

// TestParseTimelineBoundsTheQuoteChain shows what the bound is for and what it costs: a chain of
// six nested quotes comes back five posts deep, the deepest of them whole (id, author, body) and
// carrying no quote of its own. The chain is cut, the post is not.
func TestParseTimelineBoundsTheQuoteChain(t *testing.T) {
	node := tweetResult("1", "body 1", "author", "")
	for i := 2; i <= 6; i++ {
		quote := `,"quoted_status_result":{"result":` + node + `}`
		node = tweetResult(strconv.Itoa(i), "body "+strconv.Itoa(i), "author", quote)
	}

	page, err := ParseTimeline(timeline(tweetItem("tweet-6", node)))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	if len(page.Posts) != 1 {
		t.Fatalf("the answer yielded %d posts, want the one post the entry carried", len(page.Posts))
	}

	var depth int
	for post := &page.Posts[0]; post != nil; post = post.Quote {
		depth++
		if depth > 10 {
			t.Fatal("the quote chain does not end: the nesting is written by whoever wrote the post, so an unbounded parser is a stack overflow anyone can author")
		}
	}
	if depth != 5 {
		t.Errorf("the chain is %d posts deep, want 5: the bound is four levels of quotes, and the post past it is still read with its id, author and body", depth)
	}

	deepest := page.Posts[0]
	for deepest.Quote != nil {
		deepest = *deepest.Quote
	}
	switch {
	case deepest.ID != "2":
		t.Errorf("the deepest post is %q, want the one at the bound", deepest.ID)
	case deepest.Text != "body 2" || deepest.Author.ScreenName != "author":
		t.Errorf("the post at the bound came out as %+v, want its body and author: past the bound the post is still a post", deepest)
	}
}

// TestParseTimelineReadsMediaRefs covers the media shapes the recording has no example of: the
// entities fallback, the animated_gif kind, alt text, and the ways a media list can be unreadable.
func TestParseTimelineReadsMediaRefs(t *testing.T) {
	entities := `,"entities":{"media":[` +
		`{"type":"photo","media_url_https":"https://pbs.twimg.com/media/a.jpg","ext_alt_text":"a screenshot of the pane"},` +
		`{"type":"animated_gif","media_url_https":"https://pbs.twimg.com/tweet_video_thumb/b.jpg"}]}`

	page, err := ParseTimeline(timeline(tweetItem("tweet-7", tweetResultWithLegacy("7", "with attachments", "someone", entities))))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	want := []domain.MediaRef{
		{Kind: domain.Image, URL: "https://pbs.twimg.com/media/a.jpg", AltText: "a screenshot of the pane"},
		{Kind: domain.GIF, URL: "https://pbs.twimg.com/tweet_video_thumb/b.jpg"},
	}
	got := page.Posts[0].Media
	if len(got) != len(want) {
		t.Fatalf("the post came out with %d media refs, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("media ref %d is %+v, want %+v", i, got[i], want[i])
		}
	}

	// extended_entities wins when a post carries both: it is the list the upstream maintains, and
	// entities can still hold media that was replaced.
	both := `,"extended_entities":{"media":[{"type":"video","media_url_https":"https://pbs.twimg.com/media/new.jpg"}]},` +
		`"entities":{"media":[{"type":"photo","media_url_https":"https://pbs.twimg.com/media/old.jpg"}]}`
	page, err = ParseTimeline(timeline(tweetItem("tweet-8", tweetResultWithLegacy("8", "replaced media", "someone", both))))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	if got := page.Posts[0].Media; len(got) != 1 || got[0].Kind != domain.Video || got[0].URL != "https://pbs.twimg.com/media/new.jpg" {
		t.Errorf("a post with both entity lists came out with %+v, want extended_entities' one video", got)
	}

	// A media type this build does not know, a media entry with no URL and a view count that is
	// not a number are all errors naming the path: an attachment silently skipped is an
	// attachment line that lies about how many there are, and a ref with no URL is a ref nothing
	// can open.
	for _, tc := range []struct {
		name     string
		extra    string
		wantPath string
		inLegacy bool
	}{
		{
			name:     "a type this build does not know",
			extra:    `,"extended_entities":{"media":[{"type":"audio","media_url_https":"https://pbs.twimg.com/a.mp3"}]}`,
			wantPath: "legacy.extended_entities.media[0].type",
			inLegacy: true,
		},
		{
			name:     "a ref with no URL",
			extra:    `,"extended_entities":{"media":[{"type":"photo"}]}`,
			wantPath: "legacy.extended_entities.media[0].media_url_https",
			inLegacy: true,
		},
		{
			name:     "a view count that is not a number",
			extra:    `,"views":{"count":"many"}`,
			wantPath: "views.count",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := tweetResult("9", "body", "someone", tc.extra)
			if tc.inLegacy {
				result = tweetResultWithLegacy("9", "body", "someone", tc.extra)
			}
			page, err := ParseTimeline(timeline(tweetItem("tweet-9", result)))
			if err == nil {
				t.Fatalf("the answer parsed into %d posts, want an error naming %s", len(page.Posts), tc.wantPath)
			}
			if !strings.Contains(err.Error(), tc.wantPath) {
				t.Errorf("the error does not name %s: %v", tc.wantPath, err)
			}
		})
	}

	// An unquoted view count is a number the same way a quoted one is.
	page, err = ParseTimeline(timeline(tweetItem("tweet-10", tweetResult("10", "body", "someone", `,"views":{"count":42}`))))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	if got := page.Posts[0].Counts.Views; got != 42 {
		t.Errorf("an unquoted view count came out as %d, want 42", got)
	}
}

func TestParseMediaPlayableFromFixture(t *testing.T) {
	page, err := ParseTimeline(fixture(t))
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	wantPoster := "https://pbs.twimg.com/amplify_video_thumb/2102147574534672385/img/8u36LZa_mB2Kh5Ic.jpg"
	wantPlayable := "https://video.twimg.com/amplify_video/2102147574534672385/vid/avc1/1920x1080/TT9XobnynUUlzndh.mp4?tag=29"
	var found *domain.MediaRef
	for i := range page.Posts {
		for j := range page.Posts[i].Media {
			ref := &page.Posts[i].Media[j]
			if ref.Kind == domain.Video && ref.URL == wantPoster {
				found = ref
				break
			}
		}
		if found != nil {
			break
		}
	}
	if found == nil {
		t.Fatal("fixture carries no video with the recorded poster")
	}
	if found.Playable != wantPlayable {
		t.Errorf("Playable = %q, want highest-bitrate mp4 %q", found.Playable, wantPlayable)
	}
	if found.Playable == "" {
		t.Fatal("Playable is empty")
	}
}

func TestVariantSelection(t *testing.T) {

	hlsOnly := `,"extended_entities":{"media":[{"type":"video","media_url_https":"https://pbs.twimg.com/poster.jpg","video_info":{"variants":[{"content_type":"application/x-mpegURL","url":"https://video.twimg.com/a.m3u8"}]}}]}`
	page, err := ParseTimeline(timeline(tweetItem("tweet-hls", tweetResultWithLegacy("hls", "body", "someone", hlsOnly))))
	if err != nil {
		t.Fatalf("hls-only: %v", err)
	}
	if got := page.Posts[0].Media[0].Playable; got != "https://video.twimg.com/a.m3u8" {
		t.Errorf("hls-only Playable = %q", got)
	}

	noInfo := `,"extended_entities":{"media":[{"type":"video","media_url_https":"https://pbs.twimg.com/poster.jpg"}]}`
	page, err = ParseTimeline(timeline(tweetItem("tweet-nov", tweetResultWithLegacy("nov", "body", "someone", noInfo))))
	if err != nil {
		t.Fatalf("no video_info: %v", err)
	}
	if got := page.Posts[0].Media[0]; got.Playable != "" || got.URL != "https://pbs.twimg.com/poster.jpg" {
		t.Errorf("no video_info ref = %+v", got)
	}

	bad := `,"extended_entities":{"media":[{"type":"video","media_url_https":"https://pbs.twimg.com/poster.jpg","video_info":{"variants":[{"content_type":"video/mp4","bitrate":1000}]}}]}`
	_, err = ParseTimeline(timeline(tweetItem("tweet-bad", tweetResultWithLegacy("bad", "body", "someone", bad))))
	if err == nil {
		t.Fatal("missing variant url must be a parse error")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("error is %T %v, want ParseError", err, err)
	}
	if !strings.Contains(pe.Path, "variants[0].url") {
		t.Errorf("path = %q, want variants[0].url", pe.Path)
	}

	gif := `,"extended_entities":{"media":[{"type":"animated_gif","media_url_https":"https://pbs.twimg.com/g.jpg","video_info":{"variants":[{"bitrate":0,"content_type":"video/mp4","url":"https://video.twimg.com/g.mp4"}]}}]}`
	page, err = ParseTimeline(timeline(tweetItem("tweet-gif", tweetResultWithLegacy("gif", "body", "someone", gif))))
	if err != nil {
		t.Fatalf("gif: %v", err)
	}
	got := page.Posts[0].Media[0]
	if got.Kind != domain.GIF || got.Playable != "https://video.twimg.com/g.mp4" {
		t.Errorf("gif ref = %+v", got)
	}
}

// stubClient stands in for the transport: it answers with the body it was given and remembers
// the operation and the variables it was asked with. No socket, no session, no registry.
type stubClient struct {
	body  []byte
	err   error
	op    string
	vars  map[string]any
	calls int
}

func (s *stubClient) GraphQL(_ context.Context, opName string, variables map[string]any) ([]byte, error) {
	s.calls++
	s.op, s.vars = opName, variables
	return s.body, s.err
}

// TestAdapterAsksForTheRegisteredOperations pins the wire half of the port: which operation each
// list name reads, and the variables the request carries. An operation the registry does not have
// never reaches here — the client refuses it by name — so this is the only place the mapping from
// a list to an operation exists.
func TestAdapterAsksForTheRegisteredOperations(t *testing.T) {
	frozen := time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)
	restore := now
	now = func() time.Time { return frozen }
	defer func() { now = restore }()

	for _, tc := range []struct {
		name   string
		kind   domain.HomeKind
		cursor domain.Cursor
		wantOp string
	}{
		{"following, first page", domain.Following, "", "HomeLatestTimeline"},
		{"for you, paging", domain.ForYou, "cursor-bottom-from-the-last-page", "HomeTimeline"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &stubClient{body: fixture(t)}
			page, err := New(client).Home(context.Background(), tc.kind, tc.cursor)
			if err != nil {
				t.Fatalf("Home: %v", err)
			}
			switch {
			case client.op != tc.wantOp:
				t.Errorf("Home asked for the operation %q, want %q", client.op, tc.wantOp)
			case client.vars["count"] != 20:
				t.Errorf("Home sent count = %v, want 20", client.vars["count"])
			case client.vars["cursor"] != string(tc.cursor):
				t.Errorf("Home sent cursor = %v, want %q", client.vars["cursor"], tc.cursor)
			// includePromotedContent is pinned with the value the measurement settled on, so a
			// later edit that quietly drops a required variable fails here rather than on a
			// live read (measured 2026-10-15: 422 naming it, then 200 with it present).
			case client.vars["includePromotedContent"] != false:
				t.Errorf("Home sent includePromotedContent = %v, want false: the upstream named it as required (2026-10-15)", client.vars["includePromotedContent"])
			}
			if seen, ok := client.vars["seenTweetIds"].([]string); !ok || len(seen) != 0 {
				t.Errorf("Home sent seenTweetIds = %v, want an empty array", client.vars["seenTweetIds"])
			}
			if got := len(client.vars); got != 4 {
				t.Errorf("Home sent %d variables (%v), want exactly the 4 measured ones", got, client.vars)
			}
			switch {
			case len(page.Posts) == 0:
				t.Error("the page came back with no posts, want the recorded response's")
			case page.Next == "":
				t.Error("the page came back with no next cursor, want the recorded response's bottom cursor")
			case page.Requested != 20:
				t.Errorf("the page reports Requested = %d, want the 20 tweet entries the recording carries", page.Requested)
			case !page.FetchedAt.Equal(frozen):
				t.Errorf("the page is stamped %s, want the frozen clock's %s", page.FetchedAt, frozen)
			}
		})
	}
}

func TestAdapterPassesAFailedAnswerThrough(t *testing.T) {
	sentinel := errors.New("the transport's own classified failure")
	client := &stubClient{err: sentinel}

	page, err := New(client).Home(context.Background(), domain.Following, "")
	if !errors.Is(err, sentinel) {
		t.Errorf("Home reported %v, want the transport's error unchanged: the caller names the state from it", err)
	}
	if len(page.Posts) != 0 || page.Next != "" || page.Requested != 0 {
		t.Errorf("a failed page carried %+v, want nothing", page)
	}

	if _, err := New(client).User(context.Background(), "x"); !errors.Is(err, sentinel) {
		t.Errorf("User reported %v, want the transport's error unchanged", err)
	}
}

func TestAdapterUserAsksByScreenName(t *testing.T) {
	client := &stubClient{body: []byte(`{"data":{"user":{"result":{"__typename":"User","rest_id":"783214","core":{"screen_name":"X","name":"X"}}}}}`)}

	user, err := New(client).User(context.Background(), "X")
	if err != nil {
		t.Fatalf("User: %v", err)
	}
	switch {
	case client.op != "UserByScreenName":
		t.Errorf("User asked for the operation %q, want UserByScreenName", client.op)
	case client.vars["screen_name"] != "X":
		t.Errorf("User sent screen_name = %v, want the handle it was asked for", client.vars["screen_name"])
	case user.ScreenName != "X" || user.ID != "783214":
		t.Errorf("User returned %+v, want the profile the response carries", user)
	}
}
