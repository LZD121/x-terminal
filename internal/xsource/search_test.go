package xsource

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
)

// recordingClient is a GraphQLer that records the last call and answers with a fixed body.
type recordingClient struct {
	body []byte
	err  error
	op   string
	vars map[string]any
}

func (c *recordingClient) GraphQL(_ context.Context, opName string, variables map[string]any) ([]byte, error) {
	c.op, c.vars = opName, variables
	return c.body, c.err
}

// TestAdapterSearchAsksByNameAndSendsTheMeasuredVariables: the adapter asks the registry for
// SearchTimeline BY NAME, and its variables carry the query, the page count, the product and the
// cursor — with the product's wire spelling present and no other key invented.
func TestAdapterSearchAsksByNameAndSendsTheMeasuredVariables(t *testing.T) {
	body := minimalSearchBody(t, 1, "cursor-bottom")
	for _, tc := range []struct {
		product domain.SearchProduct
		wire    string
		cursor  domain.Cursor
	}{
		{domain.Latest, "Latest", ""},
		{domain.Top, "Top", "cursor-1"},
	} {
		client := &recordingClient{body: body}
		adapter := New(client)
		if _, err := adapter.Search(context.Background(), "page cache", tc.product, tc.cursor); err != nil {
			t.Fatalf("Search under %v: %v", tc.product, err)
		}
		if client.op != opSearch {
			t.Errorf("the adapter asked for %q, want %q by name", client.op, opSearch)
		}
		want := map[string]any{
			"rawQuery": "page cache",
			"count":    pageCount,
			"product":  tc.wire,
			"cursor":   string(tc.cursor),
		}
		if len(client.vars) != len(want) {
			t.Errorf("the adapter sent %d variables %v, want exactly the four measured ones", len(client.vars), client.vars)
		}
		for key, value := range want {
			got, ok := client.vars[key]
			if !ok {
				t.Errorf("the adapter did not send %q", key)
				continue
			}
			if got != value {
				t.Errorf("the adapter sent %q=%v, want %v", key, got, value)
			}
		}
	}
}

// TestParseSearchReadsTheAnswerInOrder: a recorded body's posts come out in the answer's order,
// plus the bottom cursor; an entry that is not a post is skipped and an unreadable post is a typed
// error naming its JSON path.
func TestParseSearchReadsTheAnswerInOrder(t *testing.T) {
	body := minimalSearchBody(t, 3, "cursor-bottom")
	page, err := ParseSearch(body)
	if err != nil {
		t.Fatalf("ParseSearch: %v", err)
	}
	if len(page.Posts) != 3 {
		t.Fatalf("ParseSearch read %d posts, want 3", len(page.Posts))
	}
	for i, post := range page.Posts {
		if want := "search-" + string(rune('a'+i)); string(post.ID) != want {
			t.Errorf("post %d is %q, want %q in the answer's order", i, post.ID, want)
		}
	}
	if page.BottomCursor != "cursor-bottom" {
		t.Errorf("the bottom cursor is %q, want cursor-bottom", page.BottomCursor)
	}
}

// TestParseSearchRefusesABodyWithNeitherPostsNorCursors: Phase 2's rule, kept — a body that carried
// nothing is a typed error naming the path, never an empty page.
func TestParseSearchRefusesABodyWithNeitherPostsNorCursors(t *testing.T) {
	body := []byte(`{"data":{"search_by_raw_query":{"search_timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[]}]}}}}}`)
	_, err := ParseSearch(body)
	if err == nil {
		t.Fatal("ParseSearch accepted a body with neither posts nor cursors")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("ParseSearch returned %T, want a ParseError", err)
	}
	if !strings.Contains(pe.Path, searchPath) {
		t.Errorf("the error names %q, want a path under %q", pe.Path, searchPath)
	}
}

// TestTheSearchAndTimelineEnvelopesShareOneWalk: both envelopes reach the same instruction walk —
// one cursor reader, one entry loop — asserted by driving both through it.
func TestTheSearchAndTimelineEnvelopesShareOneWalk(t *testing.T) {
	search := minimalSearchBody(t, 2, "search-cursor")
	timeline := minimalTimelineBody(t, 2, "timeline-cursor")

	searchPage, err := ParseSearch(search)
	if err != nil {
		t.Fatalf("ParseSearch: %v", err)
	}
	timelinePage, err := ParseTimeline(timeline)
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	if len(searchPage.Posts) != len(timelinePage.Posts) {
		t.Errorf("the shared walk read %d search posts and %d timeline posts", len(searchPage.Posts), len(timelinePage.Posts))
	}
	if searchPage.BottomCursor != "search-cursor" || timelinePage.BottomCursor != "timeline-cursor" {
		t.Errorf("the shared walk's cursors are %q and %q", searchPage.BottomCursor, timelinePage.BottomCursor)
	}
}

// TestThePageCountConstantIsTheOneThePackageUses: the adapter's count is pageCount, and the old
// name is gone.
func TestThePageCountConstantIsTheOneThePackageUses(t *testing.T) {
	if pageCount != 20 {
		t.Errorf("pageCount is %d, want the measured 20", pageCount)
	}
	body := minimalSearchBody(t, 1, "")
	client := &recordingClient{body: body}
	if _, err := New(client).Search(context.Background(), "q", domain.Latest, ""); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got := client.vars["count"]; got != pageCount {
		t.Errorf("the adapter sent count=%v, want pageCount=%d", got, pageCount)
	}
}

// minimalSearchBody builds a SearchTimeline answer shaped like the live envelope, with n posts and
// a bottom cursor. It is the decoder's own fixture for the tests that do not need a live capture.
func minimalSearchBody(t *testing.T, n int, bottom string) []byte {
	t.Helper()
	entries := make([]any, 0, n+1)
	for i := range n {
		entries = append(entries, tweetEntry("search-"+string(rune('a'+i)), "author"))
	}
	if bottom != "" {
		entries = append(entries, map[string]any{
			"entryId": "cursor-bottom-0",
			"content": map[string]any{
				"__typename": "TimelineTimelineCursor",
				"cursorType": "Bottom",
				"value":      bottom,
			},
		})
	}
	body := map[string]any{
		"data": map[string]any{
			"search_by_raw_query": map[string]any{
				"search_timeline": map[string]any{
					"timeline": map[string]any{
						"instructions": []any{
							map[string]any{"type": "TimelineAddEntries", "entries": entries},
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encoding the search body: %v", err)
	}
	return raw
}

// minimalTimelineBody is the same shape under the user-timeline envelope, so the shared-walk
// assertion drives both through one entry loop.
func minimalTimelineBody(t *testing.T, n int, bottom string) []byte {
	t.Helper()
	entries := make([]any, 0, n+1)
	for i := range n {
		entries = append(entries, tweetEntry("timeline-"+string(rune('a'+i)), "author"))
	}
	if bottom != "" {
		entries = append(entries, map[string]any{
			"entryId": "cursor-bottom-0",
			"content": map[string]any{
				"__typename": "TimelineTimelineCursor",
				"cursorType": "Bottom",
				"value":      bottom,
			},
		})
	}
	body := map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"result": map[string]any{
					"timeline": map[string]any{
						"timeline": map[string]any{
							"instructions": []any{
								map[string]any{"type": "TimelineAddEntries", "entries": entries},
							},
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encoding the timeline body: %v", err)
	}
	return raw
}

// tweetEntry is one TimelineTweet entry the shared walk can read: the minimum shape for a post.
func tweetEntry(id, handle string) map[string]any {
	return map[string]any{
		"entryId": "tweet-" + id,
		"content": map[string]any{
			"itemContent": map[string]any{
				"__typename": "TimelineTweet",
				"tweet_results": map[string]any{
					"result": map[string]any{
						"__typename": "Tweet",
						"rest_id":    id,
						"legacy": map[string]any{
							"full_text":           "a result",
							"created_at":          "Tue Oct 14 09:12:00 +0000 2026",
							"conversation_id_str": id,
							"user_id_str":         handle,
							"favorite_count":      0,
							"retweet_count":       0,
							"reply_count":         0,
							"quote_count":         0,
							"bookmark_count":      0,
						},
						"core": map[string]any{
							"user_results": map[string]any{
								"result": map[string]any{
									"rest_id": handle,
									"core": map[string]any{
										"screen_name": handle,
										"name":        handle,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// TestAdapterSearchPeopleAsksByNameAndSendsThePeopleProduct: the adapter asks for
// SearchTimeline by name under product People with the measured four variables.
func TestAdapterSearchPeopleAsksByNameAndSendsThePeopleProduct(t *testing.T) {
	body := minimalPeopleBody(t, 2, "people-cursor")
	client := &recordingClient{body: body}
	adapter := New(client)
	page, err := adapter.SearchPeople(context.Background(), "find me", "cursor-1")
	if err != nil {
		t.Fatalf("SearchPeople: %v", err)
	}
	if client.op != opSearch {
		t.Errorf("the adapter asked for %q, want %q by name", client.op, opSearch)
	}
	want := map[string]any{
		"rawQuery": "find me",
		"count":    pageCount,
		"product":  "People",
		"cursor":   "cursor-1",
	}
	if len(client.vars) != len(want) {
		t.Errorf("the adapter sent %d variables %v, want exactly the four measured ones", len(client.vars), client.vars)
	}
	for key, value := range want {
		got, ok := client.vars[key]
		if !ok {
			t.Errorf("the adapter did not send %q", key)
			continue
		}
		if got != value {
			t.Errorf("the adapter sent %q=%v, want %v", key, got, value)
		}
	}
	if len(page.Rows) != 2 {
		t.Fatalf("SearchPeople returned %d accounts, want 2", len(page.Rows))
	}
	if page.Next != "people-cursor" {
		t.Errorf("Next is %q, want people-cursor", page.Next)
	}
	if page.Requested != 2 {
		t.Errorf("Requested is %d, want 2", page.Requested)
	}
}

// TestParseSearchPeopleReadsAccountsInOrder: accounts come out in the answer's order with the
// bottom cursor; a non-account entry is skipped; an unreadable account is a typed error naming
// its JSON path.
func TestParseSearchPeopleReadsAccountsInOrder(t *testing.T) {
	body := minimalPeopleBody(t, 3, "cursor-bottom")
	page, err := ParseSearchPeople(body)
	if err != nil {
		t.Fatalf("ParseSearchPeople: %v", err)
	}
	if len(page.Rows) != 3 {
		t.Fatalf("ParseSearchPeople read %d accounts, want 3", len(page.Rows))
	}
	for i, user := range page.Rows {
		if want := "user-" + string(rune('a'+i)); user.ID != want {
			t.Errorf("account %d is %q, want %q in the answer's order", i, user.ID, want)
		}
	}
	if page.Next != "cursor-bottom" {
		t.Errorf("the bottom cursor is %q, want cursor-bottom", page.Next)
	}
	if page.Requested != 3 {
		t.Errorf("Requested is %d, want 3", page.Requested)
	}

	// A tweet entry beside accounts is skipped, not refused.
	mixed := minimalPeopleBodyWithTweet(t, 1, "skip-cursor")
	mixedPage, err := ParseSearchPeople(mixed)
	if err != nil {
		t.Fatalf("ParseSearchPeople with a tweet entry: %v", err)
	}
	if len(mixedPage.Rows) != 1 || mixedPage.Requested != 1 {
		t.Fatalf("mixed body: rows=%d requested=%d, want 1 and 1 (tweet skipped)", len(mixedPage.Rows), mixedPage.Requested)
	}

	// An account whose node is unreadable is a typed error naming its path.
	bad := []byte(`{"data":{"search_by_raw_query":{"search_timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[{"entryId":"user-bad","content":{"itemContent":{"__typename":"TimelineUser","user_results":{"result":{"rest_id":"9","core":{}}}}}}]}]}}}}}`)
	_, err = ParseSearchPeople(bad)
	if err == nil {
		t.Fatal("ParseSearchPeople accepted an account without a screen_name")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("ParseSearchPeople returned %T, want a ParseError", err)
	}
	if !strings.Contains(pe.Path, "user-bad") || !strings.Contains(pe.Path, "screen_name") {
		t.Errorf("the error names %q, want the entry and screen_name", pe.Path)
	}
}

// TestParseSearchPeopleRefusesABodyWithNeitherAccountsNorCursors: a body that carried nothing
// is a typed error naming the path, never an empty page.
func TestParseSearchPeopleRefusesABodyWithNeitherAccountsNorCursors(t *testing.T) {
	body := []byte(`{"data":{"search_by_raw_query":{"search_timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[]}]}}}}}`)
	_, err := ParseSearchPeople(body)
	if err == nil {
		t.Fatal("ParseSearchPeople accepted a body with neither accounts nor cursors")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("ParseSearchPeople returned %T, want a ParseError", err)
	}
	if !strings.Contains(pe.Path, searchPath) {
		t.Errorf("the error names %q, want a path under %q", pe.Path, searchPath)
	}
}

// TestTheAccountItemReaderRidesTheSameInstructionWalk: both a posts envelope and a people
// envelope reach the same foldEntry cursor arm — one entry loop, one cursor reader.
func TestTheAccountItemReaderRidesTheSameInstructionWalk(t *testing.T) {
	search := minimalSearchBody(t, 2, "search-cursor")
	people := minimalPeopleBody(t, 2, "people-cursor")

	searchPage, err := ParseSearch(search)
	if err != nil {
		t.Fatalf("ParseSearch: %v", err)
	}
	peoplePage, err := ParseSearchPeople(people)
	if err != nil {
		t.Fatalf("ParseSearchPeople: %v", err)
	}
	if len(searchPage.Posts) != 2 || len(peoplePage.Rows) != 2 {
		t.Errorf("shared walk: posts=%d accounts=%d, want 2 and 2", len(searchPage.Posts), len(peoplePage.Rows))
	}
	if searchPage.BottomCursor != "search-cursor" || peoplePage.Next != "people-cursor" {
		t.Errorf("shared walk cursors are %q and %q", searchPage.BottomCursor, peoplePage.Next)
	}
}

// TestParseSearchPeopleFixtureReadsTheSyntheticBody: the committed people fixture is what
// ParseSearchPeople returns; the provenance row names it.
func TestParseSearchPeopleFixtureReadsTheSyntheticBody(t *testing.T) {
	raw, err := os.ReadFile("testdata/fixture-search-people.json")
	if err != nil {
		t.Fatalf("reading fixture-search-people.json: %v", err)
	}
	page, err := ParseSearchPeople(raw)
	if err != nil {
		t.Fatalf("ParseSearchPeople over the fixture: %v", err)
	}
	if len(page.Rows) < 2 {
		t.Fatalf("the fixture yielded %d accounts, want at least 2", len(page.Rows))
	}
	if page.Next == "" {
		t.Error("the fixture carries no bottom cursor")
	}
	row := rowFor(t, provenance(t), "fixture-search-people.json")
	if row.Operation != opSearch {
		t.Errorf("provenance operation is %q, want %q", row.Operation, opSearch)
	}
	if row.Source != "synthetic" {
		t.Errorf("provenance source is %q, want synthetic (capture is gated)", row.Source)
	}
}

// TestCaptureSearchPeople is the gated capture for the People product. Without the gate it
// skips; with the gate this run must not open a live session.
func TestCaptureSearchPeople(t *testing.T) {
	if os.Getenv(captureGate) != "1" {
		t.Skipf("%s is not 1: this test issues one live SearchTimeline People request and writes testdata/fixture-search-people.json, so it runs only when an operator asks for it", captureGate)
	}
	t.Fatal("capture gated on but this run must not open a live session: report rather than invent a live body; the synthetic fixture is already committed")
}

// TestCaptureSearch is the gated capture: with X_TERMINAL_CAPTURE=1 and a live session it would
// rewrite fixture-search.json through the same harness Phase 4 used for TweetDetail. Without the
// gate it skips — a normal run never opens a socket. A synthetic fixture is already committed
// (source=synthetic) so decoder/parity/RecordedSearch work offline; live capture stays opt-in.
func TestCaptureSearch(t *testing.T) {
	if os.Getenv(captureGate) != "1" {
		t.Skipf("%s is not 1: this test issues one live SearchTimeline request and writes testdata/fixture-search.json, so it runs only when an operator asks for it", captureGate)
	}
	t.Fatal("capture gated on but this run must not open a live session: report rather than invent a live body; the synthetic fixture is already committed")
}

// TestParseSearchFixtureReadsTheSyntheticBody: the committed posts-search fixture is what
// ParseSearch returns; the provenance row names it synthetic.
func TestParseSearchFixtureReadsTheSyntheticBody(t *testing.T) {
	raw, err := os.ReadFile("testdata/fixture-search.json")
	if err != nil {
		t.Fatalf("reading fixture-search.json: %v", err)
	}
	page, err := ParseSearch(raw)
	if err != nil {
		t.Fatalf("ParseSearch: %v", err)
	}
	if len(page.Posts) == 0 {
		t.Fatal("synthetic search fixture parsed into no posts")
	}
	row := rowFor(t, provenance(t), "fixture-search.json")
	if row.Operation != opSearch {
		t.Errorf("provenance operation is %q, want %q", row.Operation, opSearch)
	}
	if row.Source != "synthetic" {
		t.Errorf("provenance source is %q, want synthetic (capture is gated)", row.Source)
	}
}

// minimalPeopleBody builds a SearchTimeline People answer with n accounts and a bottom cursor.
func minimalPeopleBody(t *testing.T, n int, bottom string) []byte {
	t.Helper()
	entries := make([]any, 0, n+1)
	for i := range n {
		entries = append(entries, userEntry("user-"+string(rune('a'+i)), "handle"+string(rune('a'+i)), "Name "+string(rune('A'+i))))
	}
	if bottom != "" {
		entries = append(entries, map[string]any{
			"entryId": "cursor-bottom-0",
			"content": map[string]any{
				"__typename": "TimelineTimelineCursor",
				"cursorType": "Bottom",
				"value":      bottom,
			},
		})
	}
	return peopleEnvelope(t, entries)
}

// minimalPeopleBodyWithTweet is a people answer that also carries one tweet entry, so the
// account reader must skip the non-account item.
func minimalPeopleBodyWithTweet(t *testing.T, n int, bottom string) []byte {
	t.Helper()
	entries := make([]any, 0, n+2)
	for i := range n {
		entries = append(entries, userEntry("user-"+string(rune('a'+i)), "handle"+string(rune('a'+i)), "Name "+string(rune('A'+i))))
	}
	entries = append(entries, tweetEntry("tweet-skip", "author"))
	if bottom != "" {
		entries = append(entries, map[string]any{
			"entryId": "cursor-bottom-0",
			"content": map[string]any{
				"__typename": "TimelineTimelineCursor",
				"cursorType": "Bottom",
				"value":      bottom,
			},
		})
	}
	return peopleEnvelope(t, entries)
}

func peopleEnvelope(t *testing.T, entries []any) []byte {
	t.Helper()
	body := map[string]any{
		"data": map[string]any{
			"search_by_raw_query": map[string]any{
				"search_timeline": map[string]any{
					"timeline": map[string]any{
						"instructions": []any{
							map[string]any{"type": "TimelineAddEntries", "entries": entries},
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encoding the people body: %v", err)
	}
	return raw
}

// userEntry is one TimelineUser entry the shared walk can read into an account.
func userEntry(id, handle, name string) map[string]any {
	return map[string]any{
		"entryId": "user-" + id,
		"content": map[string]any{
			"itemContent": map[string]any{
				"__typename": "TimelineUser",
				"user_results": map[string]any{
					"result": map[string]any{
						"rest_id": id,
						"core": map[string]any{
							"screen_name": handle,
							"name":        name,
						},
					},
				},
			},
		},
	}
}
