package render

import (
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

func TestCapabilityBlocksAt80(t *testing.T) {
	plainEnv(t)
	facts := CapabilityFacts{Path: "/tmp/saved.db", Owner: "mira", List: "likes"}
	for _, c := range []domain.Capability{
		domain.CapOperationMissing, domain.CapWrongAccount, domain.CapMirrorUnreadable,
	} {
		blocks := CapabilityBlock(Plain, c, facts, 80)
		if len(blocks) != 1 || len(blocks[0].Rows) < 2 {
			t.Fatalf("%v: want a multi-row block, got %#v", c, blocks)
		}
		head := strings.TrimSpace(blocks[0].Rows[0])
		if want := CapabilityWord("", c, facts.List); head != want {
			t.Errorf("%v head = %q, want %q", c, head, want)
		}
		body := strings.Join(blocks[0].Rows[1:], "\n")
		if !strings.Contains(body, "  ") {
			t.Errorf("%v body is not indented: %q", c, body)
		}
	}
}

func TestSavedEmptyStatesDiffer(t *testing.T) {
	plainEnv(t)
	never := SavedEmptyState(Plain, domain.Bookmark, false, 80)
	synced := SavedEmptyState(Plain, domain.Bookmark, true, 80)
	if strings.Join(rowsOf(never), "\n") == strings.Join(rowsOf(synced), "\n") {
		t.Fatal("never-synced and synced-empty frames are identical")
	}
	if !strings.Contains(strings.Join(rowsOf(never), "\n"), "nothing is mirrored") {
		t.Fatal("never-synced body missing")
	}
	if !strings.Contains(strings.Join(rowsOf(synced), "\n"), CopyEmptyBody) {
		t.Fatal("synced-empty body missing")
	}
}

func TestLikesHedgedIsOneLine(t *testing.T) {
	plainEnv(t)
	blocks := SavedEmptyState(Plain, domain.Like, true, 80)
	if len(blocks) != 1 || len(blocks[0].Rows) != 1 {
		t.Fatalf("likes hedge = %#v, want one line", blocks)
	}
	if !strings.Contains(blocks[0].Rows[0], CopyEmptyLikesHedged) {
		t.Fatalf("hedge line = %q", blocks[0].Rows[0])
	}
}

// TestTheCoveredLikesHedgeIsOneLine is the ambiguity rule under every cover: an empty likes answer
// is either empty or no longer served, and the pane does not resolve it (06-UI-SPEC:263-266) — so a
// covered likes frame is the cover's own one-line hedge and no body, in every style, at every
// synced state, and never the covered "the read succeeded and found nothing".
//
// The two arms are what make the absence of a body and of the other line mean something: the same
// grammar's bookmarks empty read IS the heading over the body (so the hedge is the kind's and not a
// state the grammar can never render), and the plain reader's likes frame is the plain hedge over
// the same division of lines (so the covered line is the cover's own words and not the plain one's).
func TestTheCoveredLikesHedgeIsOneLine(t *testing.T) {
	plainEnv(t)
	for _, style := range CoverStyles() {
		g := style.Grammar()
		for _, synced := range []bool{true, false} {
			blocks := SavedEmptyState(g, domain.Like, synced, 80)
			if len(blocks) != 1 || len(blocks[0].Rows) != 1 {
				t.Errorf("the %s style's likes-empty frame (synced=%v) is %#v: the ambiguity is one line and no body", style, synced, blocks)
				continue
			}
			line := blocks[0].Rows[0]
			if !strings.Contains(line, CopyCoverEmptyHedged) {
				t.Errorf("the %s style's likes line is %q, want the cover's own hedge %q", style, line, CopyCoverEmptyHedged)
			}
			// The heading is deliberately not in this list: the hedge is the heading's own words with
			// the other half of the question on them (`no records in this view, or ...`), and it is
			// the *body* that claims a read succeeded and found nothing.
			for _, forbidden := range []string{CopyCoverEmptyBody, CopyEmptyLikesHedged, CopyListLikes} {
				if strings.Contains(line, forbidden) {
					t.Errorf("the %s style's likes line %q holds %q: the covered line resolves the ambiguity or speaks in the plain reader's words", style, line, forbidden)
				}
			}
		}
		bookmarks := SavedEmptyState(g, domain.Bookmark, true, 80)
		joined := strings.Join(rowsOf(bookmarks), "\n")
		if !strings.Contains(joined, CopyCoverEmptyHeading) || !strings.Contains(joined, CopyCoverEmptyBody) {
			t.Errorf("the %s style's bookmarks empty read is %q, want the covered heading over the covered body: the hedge would then be the state's rather than the kind's", style, joined)
		}
	}
	plainBlocks := SavedEmptyState(Plain, domain.Like, true, 80)
	if len(plainBlocks) != 1 || len(plainBlocks[0].Rows) != 1 || !strings.Contains(plainBlocks[0].Rows[0], CopyEmptyLikesHedged) {
		t.Fatalf("the plain reader's likes frame is %#v, want its own one hedged line: the covered line above would have no counterpart", plainBlocks)
	}
	if CopyCoverEmptyHedged == CopyEmptyLikesHedged {
		t.Fatal("the covered hedge is the plain reader's own sentence: a covered frame would print the product's word for the list")
	}
}

func TestStateBlockWithMirrorThirdLine(t *testing.T) {
	plainEnv(t)
	f := domain.Failure{Kind: domain.FailureNoRoute, Detail: "check the proxy"}
	with := StateBlockWithMirror(Plain, f, domain.Bookmark, false, 80)
	without := StateBlockWithMirror(Plain, f, domain.Bookmark, true, 80)
	if len(with[0].Rows) <= len(without[0].Rows) {
		t.Fatalf("empty-mirror failure should grow a third line: %d vs %d", len(with[0].Rows), len(without[0].Rows))
	}
	joined := strings.Join(with[0].Rows, "\n")
	if !strings.Contains(joined, "nothing is mirrored for bookmarks") {
		t.Fatalf("third line missing: %q", joined)
	}
}

func TestSyncedStampForms(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)
	cases := []struct {
		at    time.Time
		want  string
		cells int
	}{
		{now.Add(2 * time.Hour), "11:12", 5},
		{time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC), "Jan 02", 6},
		{time.Date(2006, 1, 2, 15, 4, 0, 0, time.UTC), "Jan  2006", 9},
	}
	for _, tc := range cases {
		got := SyncedStamp(tc.at, now)
		if got != tc.want {
			t.Errorf("SyncedStamp(%v) = %q, want %q", tc.at, got, tc.want)
		}
		if text.Width(got) != tc.cells {
			t.Errorf("SyncedStamp(%v) width = %d, want %d", tc.at, text.Width(got), tc.cells)
		}
		full := MarkerSync + " " + got
		wantFull := map[string]int{"11:12": 12, "Jan 02": 13, "Jan  2006": 16}[got]
		if text.Width(full) != wantFull {
			t.Errorf("synced field %q width = %d, want %d", full, text.Width(full), wantFull)
		}
	}
}

func TestListSaved(t *testing.T) {
	if ListSaved("", domain.Bookmark) != "bookmarks" || ListSaved("", domain.Like) != "likes" {
		t.Fatal(ListSaved("", domain.Bookmark), ListSaved("", domain.Like))
	}
	if got := ListSaved(CoverServiceLog, domain.Bookmark); got != CoverServiceLogListBookmarks {
		t.Fatalf("covered bookmarks = %q", got)
	}
	if got := ListSaved(CoverGitLog, domain.Like); got != CoverGitListLikes {
		t.Fatalf("covered likes = %q", got)
	}
}

func TestMarkerSyncNonEmptyEveryGrammar(t *testing.T) {
	for _, g := range []Grammar{Plain, ServiceLog, GitLog, CI, Internal} {
		if g.Markers.Sync == "" || g.Markers.SyncNever == "" {
			t.Errorf("%s: Sync=%q SyncNever=%q", g.Name, g.Markers.Sync, g.Markers.SyncNever)
		}
	}
}
