package domain

import (
	"strings"
	"testing"
	"time"
)

// TestFailureKindsAreTheClosedTable is the totality test over the state taxonomy.
//
// It asserts three things at once, by construction:
//
//   - the table lists every constant, in order, so its index equals its value: a kind inserted
//     in the middle (which silently renumbers every kind after it) fails here;
//   - no two rows are the same constant, which is what makes the table a claim about the
//     taxonomy rather than a list of numbers;
//   - every kind except FailureNone is non-zero, so an unset Failure cannot be read as one of
//     the named states.
func TestFailureKindsAreTheClosedTable(t *testing.T) {
	kinds := []FailureKind{
		FailureNone,
		FailureSessionExpired,
		FailureSignerOrAuth,
		FailureOperationStale,
		FailureEndpointRetired,
		FailureRateLimited,
		FailureSoftBan,
		FailureLoadShed,
		FailureBadRequest,
		FailureEmptyBody,
		FailureChallenge,
		FailureUnauthorized,
		FailureUnclassified,
		FailureBudgetSpent,
		FailureNoRoute,
		FailureProxyRefused,
		FailureUpstreamBlocked,
		FailureTLSChallenge,
		FailureTransportUnusable,
		FailureNoSession,
	}

	if len(kinds) != int(FailureNoSession)+1 {
		t.Fatalf("the table holds %d kinds and the last constant is %d: a kind was added or removed without a row here, and the taxonomy is only closed if every state has one", len(kinds), FailureNoSession)
	}

	for i, kind := range kinds {
		if kind != FailureKind(i) {
			t.Errorf("row %d holds %d: the table must list every kind in order, or a kind inserted in the middle renumbers the rest unnoticed", i, kind)
		}
		switch {
		case kind == FailureNone && kind != 0:
			t.Errorf("FailureNone is %d, want the zero value: an unset Failure must read as none", kind)
		case kind != FailureNone && kind == 0:
			t.Errorf("the kind at row %d is the zero value, and only FailureNone may be", i)
		}
	}
}

func TestHomeKindZeroIsFollowing(t *testing.T) {
	if got := HomeKind(0).String(); got != "following" {
		t.Errorf("HomeKind(0).String() = %q, want %q: Following is the surface this phase opens on, so a list nobody set is the following timeline", got, "following")
	}
	if got := ForYou.String(); got != "for you" {
		t.Errorf("ForYou.String() = %q, want %q: it is the status line's `list` field wording", got, "for you")
	}
	if got := HomeKind(7).String(); !strings.Contains(got, "7") {
		t.Errorf("a kind outside the closed set prints %q, want the number it is: defaulting an unknown list onto one of the two names is a silent lie about which timeline is on screen", got)
	}
}

func TestTheZeroFailureIsNoneAndNotRetryable(t *testing.T) {
	var f Failure
	if f.Kind != FailureNone {
		t.Errorf("the zero Failure has Kind %d, want FailureNone", f.Kind)
	}
	if f.Retry {
		t.Error("the zero Failure has Retry true: an unset failure would silently enable a request, which is the one thing this field exists to prevent")
	}
	if f.Stopped {
		t.Error("the zero Failure has Stopped true, so a failure nobody recorded would stop the session")
	}
	if !f.Reset.IsZero() {
		t.Error("the zero Failure carries a reset instant, so a rate limit nobody recorded would look like it reopens")
	}
}

// TestValueTypeWords covers the String methods of the content enums. They exist for logs and
// for tests, and the unknown-value branch is part of the test: an unmapped value must name
// itself rather than pass as one of the mapped ones.
func TestValueTypeWords(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  string
		want string
	}{
		{"Visible", Visible.String(), "visible"},
		{"Limited", Limited.String(), "limited"},
		{"Unavailable", Unavailable.String(), "unavailable"},
		{"Withheld", Withheld.String(), "withheld"},
		{"Image", Image.String(), "image"},
		{"Video", Video.String(), "video"},
		{"GIF", GIF.String(), "gif"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s prints %q, want %q", tc.name, tc.got, tc.want)
		}
	}

	if got := Visibility(9).String(); !strings.Contains(got, "9") {
		t.Errorf("an unmapped Visibility prints %q, want the number it is", got)
	}
	if got := MediaKind(9).String(); !strings.Contains(got, "9") {
		t.Errorf("an unmapped MediaKind prints %q, want the number it is", got)
	}
}

// TestPostCarriesTheContentTheSplitOwns is the anti-vacuity arm for the content model: a post
// built by hand carries every field the UI-SPEC's content table names, and a zero post is a
// post whose zero values are the documented ones (visible, no media, no quote, no counts).
func TestPostCarriesTheContentTheSplitOwns(t *testing.T) {
	quoted := Post{ID: "2", Text: "quoted body", Visibility: Limited}
	post := Post{
		ID:             "1",
		Author:         User{ID: "7", ScreenName: "aergon", Name: "Aergon", AvatarURL: "a", BannerURL: "b"},
		Text:           "body",
		CreatedAt:      time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC),
		RepostedBy:     &User{ID: "8", ScreenName: "lzd"},
		Lang:           "en",
		ConversationID: "100",
		ReplyTo:        "99",
		Quote:          &quoted,
		Media:          []MediaRef{{Kind: Video, URL: "https://example.invalid/x", AltText: "alt"}},
		Visibility:     Visible,
		Counts:         Counts{Replies: 1, Reposts: 2, Likes: 3, Quotes: 4, Bookmarks: 5, Views: 6},
	}

	switch {
	case post.ID != PostID("1"):
		t.Errorf("the post's id is %q, want the value it was built with", post.ID)
	case post.Quote == nil || post.Quote.ID != "2":
		t.Error("the quote is not reachable through Quote.ID, which is where the split says the quoted post's identity lives")
	case post.ReplyTo != PostID("99"):
		t.Errorf("ReplyTo is %q, want the id it was built with", post.ReplyTo)
	case len(post.Media) != 1 || post.Media[0].Kind != Video:
		t.Error("the media ref did not survive the round trip")
	}

	var zero Post
	switch {
	case zero.Visibility != Visible:
		t.Error("a zero post is not Visible: the parser never setting visibility must mean it read the whole post")
	case zero.Quote != nil || zero.Media != nil:
		t.Error("a zero post carries a quote or media")
	case zero.Counts != (Counts{}):
		t.Error("a zero post carries engagement counts")
	}
}

func TestZeroMediaRefHasEmptyPlayable(t *testing.T) {
	var ref MediaRef
	if ref.Playable != "" {
		t.Errorf("zero MediaRef.Playable = %q, want empty", ref.Playable)
	}
	if ref.URL != "" || ref.AltText != "" || ref.Kind != Image {
		t.Errorf("zero MediaRef = %+v, want empty image", ref)
	}
}
