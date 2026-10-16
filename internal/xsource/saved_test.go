package xsource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

func TestSavedDecodesTheBookmarksFixture(t *testing.T) {
	raw := readFixture(t, captureBookmarksFile)
	page, err := ParseTimeline(raw)
	if err != nil {
		t.Fatalf("ParseTimeline(bookmarks): %v", err)
	}
	if len(page.Posts) == 0 {
		t.Fatal("the bookmarks fixture decoded to no posts")
	}
	if page.BottomCursor == "" {
		t.Fatal("the bookmarks fixture carries no bottom cursor")
	}
	for i, post := range page.Posts {
		if post.ID == "" || post.Author.ScreenName == "" || post.Text == "" {
			t.Fatalf("post %d is missing id/handle/text: %+v", i, post)
		}
	}

	frozen := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	restore := now
	now = func() time.Time { return frozen }
	defer func() { now = restore }()

	client := &recordingClient{body: raw}
	got, err := New(client).Saved(context.Background(), domain.Bookmark, "")
	if err != nil {
		t.Fatalf("Saved(bookmarks): %v", err)
	}
	if client.op != opBookmarks {
		t.Errorf("Saved asked for %q, want %q by name", client.op, opBookmarks)
	}
	if client.vars["count"] != pageCount {
		t.Errorf("count = %v, want %d", client.vars["count"], pageCount)
	}
	if client.vars["cursor"] != "" {
		t.Errorf("page-one cursor = %v, want empty", client.vars["cursor"])
	}
	if len(got.Posts) != len(page.Posts) {
		t.Fatalf("Saved returned %d posts, want the fixture's %d", len(got.Posts), len(page.Posts))
	}
	if got.Next != domain.Cursor(page.BottomCursor) {
		t.Errorf("Next = %q, want the fixture's bottom cursor %q", got.Next, page.BottomCursor)
	}
	if !got.FetchedAt.Equal(frozen) {
		t.Errorf("FetchedAt = %s, want frozen %s", got.FetchedAt, frozen)
	}

	client2 := &recordingClient{body: raw}
	if _, err := New(client2).Saved(context.Background(), domain.Bookmark, got.Next); err != nil {
		t.Fatalf("Saved(bookmarks, next): %v", err)
	}
	if client2.vars["cursor"] != string(got.Next) {
		t.Errorf("paging cursor = %v, want %q", client2.vars["cursor"], got.Next)
	}
}

func TestSavedDecodesTheLikesFixture(t *testing.T) {
	raw := readFixture(t, captureLikesFile)
	page, err := ParseTimeline(raw)
	if err != nil {
		t.Fatalf("ParseTimeline(likes): %v", err)
	}
	if len(page.Posts) == 0 {
		t.Fatal("the likes fixture decoded to no posts")
	}
	client := &recordingClient{body: raw}
	got, err := New(client).WithViewerID(redactedAccountID).Saved(context.Background(), domain.Like, "")
	if err != nil {
		t.Fatalf("Saved(likes): %v", err)
	}
	if client.op != opLikes {
		t.Errorf("Saved asked for %q, want %q", client.op, opLikes)
	}
	if client.vars["userId"] != redactedAccountID {
		t.Errorf("userId = %v, want the redacted placeholder", client.vars["userId"])
	}
	if client.vars["includePromotedContent"] != false {
		t.Errorf("includePromotedContent = %v, want false", client.vars["includePromotedContent"])
	}
	if len(got.Posts) != len(page.Posts) {
		t.Fatalf("Saved returned %d posts, want %d", len(got.Posts), len(page.Posts))
	}
}

func TestSavedSkipsNonPostEntries(t *testing.T) {
	card := `{"entryId":"promoted-1","content":{"__typename":"TimelineTimelineItem","itemContent":{"__typename":"TimelineMessagePrompt"}}}`
	cursor := `{"entryId":"cursor-bottom-0","content":{"__typename":"TimelineTimelineCursor","cursorType":"Bottom","value":"next"}}`
	body := []byte(`{"data":{"bookmark_timeline_v2":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[` + cursor + `,` + card + `,` + string(tweetItem("tweet-1", tweetResult("1", "hello", "someone", ""))) + `]}]}}}}`)
	page, err := ParseTimeline(body)
	if err != nil {
		t.Fatalf("ParseTimeline: %v", err)
	}
	if len(page.Posts) != 1 || page.Posts[0].Text != "hello" {
		t.Fatalf("got %d posts (%v), want the one tweet", len(page.Posts), texts(page.Posts))
	}
	if page.BottomCursor != "next" {
		t.Errorf("bottom cursor = %q, want next", page.BottomCursor)
	}
	if page.Requested != 1 {
		t.Errorf("Requested = %d, want 1 tweet entry counted", page.Requested)
	}
}

func TestSavedPassesAFailedAnswerThrough(t *testing.T) {
	sentinel := errors.New("the transport's own classified failure")
	client := &recordingClient{err: sentinel}
	page, err := New(client).Saved(context.Background(), domain.Bookmark, "")
	if !errors.Is(err, sentinel) {
		t.Errorf("Saved reported %v, want the transport's error unchanged", err)
	}
	if len(page.Posts) != 0 || page.Next != "" {
		t.Errorf("a failed page carried %+v, want nothing", page)
	}
}

func TestSavedRefusesAnUnknownKind(t *testing.T) {
	client := &recordingClient{body: []byte(`{}`)}
	_, err := New(client).Saved(context.Background(), domain.SavedKind(99), "")
	if err == nil {
		t.Fatal("Saved accepted an unknown kind")
	}
	if client.op != "" {
		t.Errorf("an unknown kind still issued a request for %q", client.op)
	}
}

// pinClient resolves the operation through a registry (pins included) before dialling the
// inner GraphQLer. It is how this package's tests exercise the pin path without reaching
// into xweb.Client's unexported registry field: Lookup is the same seam the live client uses.
type pinClient struct {
	reg   *volatile.Registry
	body  []byte
	err   error
	calls int
	op    string
	vars  map[string]any
}

func (c *pinClient) GraphQL(_ context.Context, opName string, variables map[string]any) ([]byte, error) {
	if _, err := c.reg.Lookup(opName); err != nil {
		return nil, err
	}
	c.calls++
	c.op, c.vars = opName, variables
	return c.body, c.err
}

// TestPinPathIsBothHalvesOfXAPI04 is the requirement's acceptance: a deliberately wrong pin
// fails as the named stale-operation state (never an empty page), the measured pin succeeds,
// and an operation the registry does not hold is a request-free typed error. No code change
// between the wrong-pin and right-pin arms — only the pins file.
func TestPinPathIsBothHalvesOfXAPI04(t *testing.T) {
	fixture := readFixture(t, captureBookmarksFile)
	row := rowFor(t, provenance(t), captureBookmarksFile)
	if row.QueryID == "" {
		t.Fatal("the bookmarks provenance row records no queryId")
	}
	wrongID := "AAAAAAAAAAAAAAAAAAAAAA"
	if wrongID == row.QueryID {
		t.Fatal("the deliberately wrong id equals the measured one")
	}

	pins := filepath.Join(t.TempDir(), volatile.PinsFileName)
	stale := &xweb.APIError{
		Operation: opBookmarks,
		Outcome:   xweb.OutcomeUnknownOperation,
		Status:    404,
	}

	// Wrong pin fails loudly.
	if err := volatile.Pin(pins, opBookmarks, wrongID); err != nil {
		t.Fatalf("Pin(wrong): %v", err)
	}
	reg, err := volatile.Load(pins)
	if err != nil {
		t.Fatalf("Load(wrong): %v", err)
	}
	op, err := reg.Lookup(opBookmarks)
	if err != nil {
		t.Fatalf("Lookup after wrong pin: %v", err)
	}
	if op.QueryID != wrongID {
		t.Fatalf("after wrong pin Lookup returned id %q, want %q: the pin file was ignored", op.QueryID, wrongID)
	}
	client := &pinClient{reg: reg, err: stale}
	page, err := New(client).Saved(context.Background(), domain.Bookmark, "")
	if err == nil {
		t.Fatal("a wrong pin returned no error: an empty success page is the failure mode XAPI-04 exists to prevent")
	}
	if len(page.Posts) != 0 {
		t.Errorf("a wrong-pin failure carried %d posts, want an empty page with a non-nil error", len(page.Posts))
	}
	var apiErr *xweb.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("wrong-pin error is %T (%v), want *xweb.APIError", err, err)
	}
	if apiErr.Outcome != xweb.OutcomeUnknownOperation {
		t.Fatalf("wrong-pin outcome is %q, want %q", apiErr.Outcome, xweb.OutcomeUnknownOperation)
	}
	if client.calls != 1 {
		t.Errorf("wrong-pin issued %d requests, want 1", client.calls)
	}

	// Right pin works — same binary, same adapter type, only the pins file changed.
	if err := volatile.Pin(pins, opBookmarks, row.QueryID); err != nil {
		t.Fatalf("Pin(right): %v", err)
	}
	reg, err = volatile.Load(pins)
	if err != nil {
		t.Fatalf("Load(right): %v", err)
	}
	op, err = reg.Lookup(opBookmarks)
	if err != nil {
		t.Fatalf("Lookup after right pin: %v", err)
	}
	if op.QueryID != row.QueryID {
		t.Fatalf("after right pin Lookup returned id %q, want the measured %q", op.QueryID, row.QueryID)
	}
	client = &pinClient{reg: reg, body: fixture}
	page, err = New(client).Saved(context.Background(), domain.Bookmark, "")
	if err != nil {
		t.Fatalf("right pin failed: %v", err)
	}
	if len(page.Posts) == 0 {
		t.Fatal("right pin returned an empty page")
	}
	if client.calls != 1 {
		t.Errorf("right-pin issued %d requests, want 1", client.calls)
	}

	// An operation the registry does not hold is a different, request-free state.
	client = &pinClient{reg: &volatile.Registry{}}
	_, err = New(client).Saved(context.Background(), domain.Bookmark, "")
	if !errors.Is(err, volatile.ErrUnknownOperation) {
		t.Fatalf("missing operation error is %v, want ErrUnknownOperation", err)
	}
	if client.calls != 0 {
		t.Fatalf("a missing operation issued %d requests, want 0: that is the capability state's fact", client.calls)
	}
}

func TestSavedFixturesCarryNoCapturedBodies(t *testing.T) {
	for _, name := range []string{captureBookmarksFile, captureLikesFile} {
		data := readFixture(t, name)
		page, err := ParseTimeline(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, post := range page.Posts {
			if post.Text == "" {
				continue
			}
			if strings.Contains(post.Text, " ") && !allX(post.Text) {
				t.Errorf("%s still carries a natural-language body %q: the capture must replace post text", name, post.Text)
			}
		}
	}
}

func allX(s string) bool {
	for _, r := range s {
		if r != 'x' && r != 'X' {
			return false
		}
	}
	return len(s) > 0
}

func TestCaptureBookmarksSkipsWithoutGate(t *testing.T) {
	if os.Getenv(captureGate) == "1" {
		t.Skip("gate is set; the skip arm cannot run")
	}
	// The gate itself is what TestCaptureBookmarks checks; this arm documents the contract.
	if !strings.Contains(fmt.Sprintf("%s", captureGate), "CAPTURE") {
		t.Fatal("capture gate renamed")
	}
}
