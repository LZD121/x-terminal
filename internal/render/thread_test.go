package render

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// This file is the conversation's row-level tests: the UI-SPEC's 80- and 120-column frames
// transcribed, the width property at both reply levels and both quote levels, the marker chain's
// truncation, the eight new markers' totality, the plain measure's Phase 2 equality, and the
// mutual exclusion of continued and reply-to.

// threadUsers are the accounts the UI-SPEC's conversation names.
func threadUsers() (lzd, aergon, mira, kestrel, long domain.User) {
	lzd = domain.User{ID: "u-lzd", ScreenName: "lzd"}
	aergon = domain.User{ID: "u-aergon", ScreenName: "aergon"}
	mira = domain.User{ID: "u-mira", ScreenName: "mira"}
	kestrel = domain.User{ID: "u-kestrel", ScreenName: "kestrel"}
	long = domain.User{ID: "u-long", ScreenName: "longhandle_name"}
	return
}

// threadSpecConversation is the UI-SPEC's worked conversation: lzd as the chain above, aergon as
// the focal post quoting lzd, aergon's continuation, then mira / kestrel / longhandle_name as the
// replies — one with CJK and an emoji, one answering the ancestor (`reply to lzd`), one quoting
// the focal post (a quote inside a reply).
func threadSpecConversation() Conversation {
	lzd, aergon, mira, kestrel, long := threadUsers()
	ancestor := domain.Post{
		ID: "a1", Author: lzd, CreatedAt: time.Date(2026, 10, 14, 8, 5, 0, 0, time.UTC),
		ConversationID: "c", Text: "the pane never lies about what it is showing - that is the only rule i care about here, and it is the rule the rest of this program is written against.",
		Counts: domain.Counts{Replies: 42424, Reposts: 53535, Likes: 64646, Quotes: 75757, Bookmarks: 86868},
	}
	focal := domain.Post{
		ID: "f1", Author: aergon, CreatedAt: time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC),
		ConversationID: "c", ReplyTo: "a1",
		Text:   "cursor in, cursor out: the render layer never does arithmetic on a cursor.",
		Counts: domain.Counts{Replies: 31313, Reposts: 41414, Likes: 51515, Quotes: 61616, Bookmarks: 71717},
		Quote: &domain.Post{
			ID: "q1", Author: lzd, CreatedAt: time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC),
			ConversationID: "c", Text: "the page cache is keyed by the cursor, not by an index.",
		},
	}
	continued := domain.Post{
		ID: "c1", Author: aergon, CreatedAt: time.Date(2026, 10, 14, 9, 13, 0, 0, time.UTC),
		ConversationID: "c", ReplyTo: "f1",
		Text: "a late page never moves what you are reading.",
	}
	return Conversation{
		Ancestors: []domain.Post{ancestor},
		Focal:     focal,
		Continued: []domain.Post{continued},
		Replies: []domain.Post{
			{
				ID: "r1", Author: mira, CreatedAt: time.Date(2026, 10, 14, 9, 20, 0, 0, time.UTC),
				ConversationID: "c", ReplyTo: "f1",
				Text: "does that hold when the walk re-runs? 日本語のテキストは2セル幅です。 🚀",
			},
			{
				ID: "r2", Author: kestrel, CreatedAt: time.Date(2026, 10, 14, 9, 24, 0, 0, time.UTC),
				ConversationID: "c", ReplyTo: "a1",
				Text: "then the page cache is doing more work than the pane admits.",
			},
			{
				ID: "r3", Author: long, CreatedAt: time.Date(2026, 10, 14, 9, 31, 0, 0, time.UTC),
				ConversationID: "c", ReplyTo: "r2",
				Text: "that is the part i keep getting wrong, so:",
				Quote: &domain.Post{
					ID: "f1", Author: aergon, CreatedAt: time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC),
					ConversationID: "c",
					Text:           "cursor in, cursor out: the render layer never does arithmetic on a cursor.",
				},
			},
		},
		Read: true,
	}
}

// TestThreadBlocksTheUISpecFrameAt80Columns is the UI-SPEC's primary conversation frame at 80
// columns: the ancestor, the focal block with its nested quote carrying both the selection cell and
// the quote prefix, the continuation with no separator above it, and the three replies.
func TestThreadBlocksTheUISpecFrameAt80Columns(t *testing.T) {
	plainEnv(t)
	c := threadSpecConversation()
	assertRows(t, "the 80-column conversation", rowsOf(ThreadBlocks(c, Plain, 80, Selection{ID: c.Focal.ID}, specNow, nil)), []string{
		" lzd              Oct 14 08:05",
		"   the pane never lies about what it is showing - that is the only rule i care",
		"   about here, and it is the rule the rest of this program is written against.",
		"",
		">aergon           Oct 14 09:12  focal",
		">  cursor in, cursor out: the render layer never does arithmetic on a cursor.",
		">> lzd              Oct 14 09:00",
		">>   the page cache is keyed by the cursor, not by an index.",
		" aergon           Oct 14 09:13  continued",
		"   a late page never moves what you are reading.",
		"",
		"   mira             Oct 14 09:20",
		"     does that hold when the walk re-runs? 日本語のテキストは2セル幅です。 🚀",
		"",
		"   kestrel          Oct 14 09:24  reply to lzd",
		"     then the page cache is doing more work than the pane admits.",
		"",
		"   longhandle_name  Oct 14 09:31",
		"     that is the part i keep getting wrong, so:",
		"   > aergon           Oct 14 09:12",
		"   >   cursor in, cursor out: the render layer never does arithmetic on a",
		"   >   cursor.",
	})
}

// TestThreadBlocksTheUISpecFrameAt120Columns is the same conversation at 120 columns. The
// measure is capped at 96 cells, so the ancestor's line wraps at the cap and the reply's quoted
// line — which needed two rows at 80 — fits on one.
func TestThreadBlocksTheUISpecFrameAt120Columns(t *testing.T) {
	plainEnv(t)
	c := threadSpecConversation()
	assertRows(t, "the 120-column conversation", rowsOf(ThreadBlocks(c, Plain, 120, Selection{ID: c.Focal.ID}, specNow, nil)), []string{
		" lzd              Oct 14 08:05",
		"   the pane never lies about what it is showing - that is the only rule i care about here, and it",
		"   is the rule the rest of this program is written against.",
		"",
		">aergon           Oct 14 09:12  focal",
		">  cursor in, cursor out: the render layer never does arithmetic on a cursor.",
		">> lzd              Oct 14 09:00",
		">>   the page cache is keyed by the cursor, not by an index.",
		" aergon           Oct 14 09:13  continued",
		"   a late page never moves what you are reading.",
		"",
		"   mira             Oct 14 09:20",
		"     does that hold when the walk re-runs? 日本語のテキストは2セル幅です。 🚀",
		"",
		"   kestrel          Oct 14 09:24  reply to lzd",
		"     then the page cache is doing more work than the pane admits.",
		"",
		"   longhandle_name  Oct 14 09:31",
		"     that is the part i keep getting wrong, so:",
		"   > aergon           Oct 14 09:12",
		"   >   cursor in, cursor out: the render layer never does arithmetic on a cursor.",
	})
}

// TestThePlainMeasureIsPhaseTwos pins that the plain grammar at replyLevel 0 still measures
// Phase 2's min(W - 3 - 2·depth, 96), so the ancestor and focal rows cannot move.
func TestThePlainMeasureIsPhaseTwos(t *testing.T) {
	for _, w := range []int{1, 2, 3, 5, 8, 55, 60, 77, 80, 96, 120, 200} {
		for depth := 0; depth <= Plain.Markers.QuoteDepthCap+1; depth++ {
			got := Plain.measure(w, depth, 0)
			d := depth
			if d > Plain.Markers.QuoteDepthCap {
				d = Plain.Markers.QuoteDepthCap
			}
			want := w - 3 - 2*d
			if want > Plain.MeasureCap {
				want = Plain.MeasureCap
			}
			if got != want {
				t.Errorf("plain measure at w=%d depth=%d replyLevel=0 is %d, want Phase 2's %d",
					w, depth, got, want)
			}
		}
	}
}

// TestTheMarkerSetsAreTotal is the conversation markers' totality: every shipped grammar's eight
// new slots carry a literal, so a pane statement or a relationship marker cannot render as a blank
// row under any style.
func TestTheMarkerSetsAreTotal(t *testing.T) {
	grammars := []Grammar{Plain, ServiceLog, GitLog, CI, Internal}
	for _, g := range grammars {
		for name, slot := range map[string]string{
			"Focal":          g.Markers.Focal,
			"ReplyTo":        g.Markers.ReplyTo,
			"NoReplies":      g.Markers.NoReplies,
			"LoadingThread":  g.Markers.LoadingThread,
			"PartialStopped": g.Markers.PartialStopped,
			"PartialAbove":   g.Markers.PartialAbove,
			"PartialHidden":  g.Markers.PartialHidden,
			"Withheld":       g.Markers.Withheld,
		} {
			if slot == "" {
				t.Errorf("the %s grammar's %s marker is empty: a shipped grammar must spell every conversation marker",
					g.Name, name)
			}
		}
	}
}

// TestContinuedAndReplyToAreMutuallyExclusive: a reply block carries one of continued and reply-to
// and never both. Continuations live outside the reply section; replies that answer elsewhere carry
// reply-to and never continued.
func TestContinuedAndReplyToAreMutuallyExclusive(t *testing.T) {
	plainEnv(t)
	c := threadSpecConversation()
	blocks := ThreadBlocks(c, Plain, 80, Selection{ID: c.Focal.ID}, specNow, nil)
	for _, block := range blocks {
		head := ""
		for _, row := range block.Rows {
			if strings.TrimSpace(row) != "" {
				head = row
				break
			}
		}
		hasContinued := strings.Contains(head, MarkerContinued)
		hasReplyTo := strings.Contains(head, "reply to ")
		if hasContinued && hasReplyTo {
			t.Fatalf("block %q carries both continued and reply-to: %q", block.PostID, head)
		}
	}
	// The arm: the fixture really produces both markers, on different blocks.
	joined := strings.Join(rowsOf(blocks), "\n")
	if !strings.Contains(joined, MarkerContinued) {
		t.Fatal("the conversation carries no continued marker: the exclusion would be vacuous")
	}
	if !strings.Contains(joined, "reply to lzd") {
		t.Fatal("the conversation carries no reply-to marker: the exclusion would be vacuous")
	}
}

// TestThreadRowsNeverExceedThePane is A4 for the conversation: at both reply levels and both
// quote levels, for every w in the UI-SPEC's table, no row exceeds W and the deepest legal row
// (reply + quote + quote) stays inside W.
func TestThreadRowsNeverExceedThePane(t *testing.T) {
	plainEnv(t)
	c := threadSpecConversation()
	// A depth-2 quote inside a reply is the deepest legal row.
	lzd, aergon, _, _, _ := threadUsers()
	deep := c
	deep.Replies = append([]domain.Post{}, c.Replies...)
	deep.Replies[2].Quote = &domain.Post{
		ID: "q-outer", Author: aergon, CreatedAt: specNow, ConversationID: "c",
		Text: "outer quote body that is long enough to wrap under a narrow measure at every width in the table",
		Quote: &domain.Post{
			ID: "q-inner", Author: lzd, CreatedAt: specNow.Add(-time.Minute), ConversationID: "c",
			Text: "inner quote body that is also long enough to wrap under a narrow measure at every width",
		},
	}
	for _, w := range []int{1, 2, 3, 5, 8, 55, 77, 96} {
		for replyLevel := 0; replyLevel <= 1; replyLevel++ {
			for depth := 0; depth <= Plain.Markers.QuoteDepthCap; depth++ {
				cells := bodyCells(Plain, w, depth, replyLevel)
				if cells != 1+Plain.ReplyIndent*replyLevel+Plain.BodyIndent+2*min(depth, Plain.Markers.QuoteDepthCap)+Plain.measure(w, depth, replyLevel) {
					t.Errorf("bodyCells at w=%d depth=%d reply=%d is %d, not the composer's own arithmetic",
						w, depth, replyLevel, cells)
				}
			}
		}
		// The deepest legal row is a quote inside a reply. When the measure at that depth is
		// non-positive the pane cannot hold the row's own prefixes, so the property is the
		// measure itself rather than a render that would have nowhere to put the text.
		if Plain.measure(w, Plain.Markers.QuoteDepthCap, 1) < 1 {
			continue
		}
		for _, row := range rowsOf(ThreadBlocks(deep, Plain, w, Selection{ID: deep.Focal.ID}, specNow, nil)) {
			if got := text.Width(row); got > w {
				t.Errorf("at %d columns, row %q is %d cells wide", w, row, got)
			}
		}
	}
}

// TestThreadMarkerChainTruncates: a marker chain that would push the head past W is truncated with
// the truncation mark in its last cell rather than pushing the row past the pane.
func TestThreadMarkerChainTruncates(t *testing.T) {
	plainEnv(t)
	_, aergon, _, _, _ := threadUsers()
	reposter := domain.User{ID: "u-rep", ScreenName: strings.Repeat("r", 40)}
	parent := domain.Post{
		ID: "p-parent", Author: domain.User{ID: "u-parent", ScreenName: strings.Repeat("p", 40)},
		CreatedAt: specNow.Add(-time.Hour),
	}
	focal := domain.Post{
		ID: "f-wide", Author: aergon, CreatedAt: specNow, ConversationID: "c",
		RepostedBy: &reposter, Visibility: domain.Limited,
		Text: "focal",
	}
	reply := domain.Post{
		ID: "r-wide", Author: domain.User{ID: "u-reply", ScreenName: strings.Repeat("q", 40)},
		CreatedAt: specNow.Add(time.Minute), ConversationID: "c", ReplyTo: parent.ID,
		RepostedBy: &reposter, Visibility: domain.Limited, Text: "reply",
	}
	c := Conversation{
		Ancestors: []domain.Post{parent},
		Focal:     focal,
		Replies:   []domain.Post{reply},
		Read:      true,
	}
	for _, w := range []int{MinPaneCols, 80} {
		for _, row := range rowsOf(ThreadBlocks(c, Plain, w, Selection{ID: focal.ID}, specNow, nil)) {
			if got := text.Width(row); got > w {
				t.Errorf("at %d columns a marker head exceeded the pane: %q (%d cells)", w, row, got)
			}
		}
		joined := strings.Join(rowsOf(ThreadBlocks(c, Plain, w, Selection{ID: focal.ID}, specNow, nil)), "\n")
		if !strings.Contains(joined, MarkTruncation) {
			t.Errorf("at %d columns the over-long marker chain carried no truncation mark", w)
		}
	}
}

// TestThreadPrintsNoCountDigit: a conversation's Counts are consulted as evidence and never printed.
// The fixture's own five-digit numbers are searched for, so a renderer that printed a count would
// print one of these.
func TestThreadPrintsNoCountDigit(t *testing.T) {
	plainEnv(t)
	c := threadSpecConversation()
	joined := strings.Join(rowsOf(ThreadBlocks(c, Plain, 120, Selection{ID: c.Focal.ID}, specNow, nil)), "\n")
	seen := 0
	for _, counts := range []domain.Counts{c.Ancestors[0].Counts, c.Focal.Counts} {
		for _, n := range []int{counts.Replies, counts.Reposts, counts.Likes, counts.Quotes, counts.Bookmarks} {
			if n == 0 {
				t.Fatal("a conversation count is zero: the absence would be vacuous")
			}
			seen++
			digit := strconv.Itoa(n)
			if strings.Contains(joined, digit) {
				t.Errorf("a rendered thread row contains the count digit %q", digit)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no count was searched for")
	}
	if Plain.ShowCounts {
		t.Fatal("the plain grammar has ShowCounts on: a conversation would print engagement numbers")
	}
}
