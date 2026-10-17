package xsource

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
)

// minimalUserListBody is an account-list body under the user-timeline envelope: n TimelineUser
// entries and, optionally, a bottom cursor. It is synthetic — the shape is the operation's contract,
// and the live answer was measured separately (see the note on userListPath).
func minimalUserListBody(t *testing.T, n int, bottom string) []byte {
	t.Helper()
	entries := make([]any, 0, n+1)
	for i := range n {
		id := "u" + string(rune('a'+i))
		entries = append(entries, map[string]any{
			"entryId": "user-" + id,
			"content": map[string]any{
				"__typename": "TimelineTimelineItem",
				"itemContent": map[string]any{
					"__typename": "TimelineUser",
					"user_results": map[string]any{
						"result": map[string]any{
							"__typename": "User",
							"rest_id":    id,
							"core": map[string]any{
								"screen_name": "handle_" + id,
								"name":        "Name " + id,
							},
						},
					},
				},
			},
		})
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
		t.Fatalf("encoding the account-list body: %v", err)
	}
	return raw
}

// TestParseUserPageReadsAccountsAndCursor: the three lists share one parser, and it must read the
// account rows and the cursor out of the same container UserTweets uses.
func TestParseUserPageReadsAccountsAndCursor(t *testing.T) {
	page, err := ParseUserPage(minimalUserListBody(t, 3, "cursor-bottom-1"))
	if err != nil {
		t.Fatalf("ParseUserPage: %v", err)
	}
	if len(page.Rows) != 3 {
		t.Fatalf("%d rows, want 3", len(page.Rows))
	}
	if page.Rows[0].ScreenName != "handle_ua" {
		t.Fatalf("first row is %q", page.Rows[0].ScreenName)
	}
	if page.Next != "cursor-bottom-1" {
		t.Fatalf("next cursor = %q", page.Next)
	}
	if page.Requested != 3 {
		t.Fatalf("requested = %d, want 3", page.Requested)
	}
}

// TestParseUserPageRefusesABodyWithNoList: a body with neither accounts nor cursors is a typed
// error naming the path, never an empty page that reads as "nobody follows them".
func TestParseUserPageRefusesABodyWithNoList(t *testing.T) {
	raw := []byte(`{"data":{"user":{"result":{}}}}`)
	_, err := ParseUserPage(raw)
	if err == nil {
		t.Fatal("a body with no timeline was accepted")
	}
	if got := err.Error(); !contains(got, "data.user.result.timeline.timeline.instructions") {
		t.Fatalf("the error does not name the path: %v", err)
	}
}

// TestAdapterAsksEachListByItsOwnName: four questions, four operations, and the variables are the
// measured set — no invented key, and the count the pager asks for.
func TestAdapterAsksEachListByItsOwnName(t *testing.T) {
	cases := []struct {
		name string
		call func(*Adapter, context.Context, string, domain.Cursor) error
		op   string
	}{
		{"Following", func(a *Adapter, c context.Context, id string, cur domain.Cursor) error {
			_, err := a.Following(c, id, cur)
			return err
		}, opFollowing},
		{"Followers", func(a *Adapter, c context.Context, id string, cur domain.Cursor) error {
			_, err := a.Followers(c, id, cur)
			return err
		}, opFollowers},
		{"VerifiedFollowers", func(a *Adapter, c context.Context, id string, cur domain.Cursor) error {
			_, err := a.VerifiedFollowers(c, id, cur)
			return err
		}, opVerifiedFollows},
	}
	for _, tc := range cases {
		client := &recordingClient{body: minimalUserListBody(t, 1, "")}
		if err := tc.call(New(client), context.Background(), "42", "c1"); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if client.op != tc.op {
			t.Fatalf("%s asked %q, want %q", tc.name, client.op, tc.op)
		}
		want := map[string]any{
			"userId":                 "42",
			"count":                  pageCount,
			"cursor":                 "c1",
			"includePromotedContent": false,
		}
		if !sameVariables(client.vars, want) {
			t.Fatalf("%s sent %v, want %v", tc.name, client.vars, want)
		}
	}
}

// TestAdapterMediaAsksForTheMediaOperation: the media tab is a posts page under its own operation,
// with the variables the page's own shape asked for.
func TestAdapterMediaAsksForTheMediaOperation(t *testing.T) {
	client := &recordingClient{body: minimalTimelineBody(t, 2, "")}
	page, err := New(client).Media(context.Background(), "42", "")
	if err != nil {
		t.Fatalf("Media: %v", err)
	}
	if client.op != opUserMedia {
		t.Fatalf("asked %q, want %q", client.op, opUserMedia)
	}
	if len(page.Posts) != 2 {
		t.Fatalf("%d posts, want 2", len(page.Posts))
	}
	want := map[string]any{
		"userId":                 "42",
		"count":                  pageCount,
		"cursor":                 "",
		"includePromotedContent": true,
		"withClientEventToken":   false,
		"withBirdwatchNotes":     false,
		"withVoice":              true,
		"withV2Timeline":         true,
	}
	if !sameVariables(client.vars, want) {
		t.Fatalf("sent %v, want %v", client.vars, want)
	}
}

// sameVariables compares two variable maps value by value: reflect.DeepEqual on maps is available,
// but a readable failure needs the two maps printed, which is what this is for.
func sameVariables(got, want map[string]any) bool {
	if len(got) != len(want) {
		return false
	}
	for k, w := range want {
		if g, ok := got[k]; !ok || g != w {
			return false
		}
	}
	return true
}

// contains is strings.Contains with the argument order the tests above read best.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
