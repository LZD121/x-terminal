package render

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// specNow is the clock the UI-SPEC's worked frames are stamped with. It is injected rather than
// read, so the frames below are a function of these bytes and not of the day they are run on.
var specNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

// specSelected is the post the UI-SPEC's frames have selected: the repost by longhandle_name,
// whose block is not the pane's first row.
const specSelected = domain.PostID("2")

// plainEnv pins the environment a literal comparison runs under: TERM=dumb and NO_COLOR=1 paint
// nothing at all, so a row's characters can be read in the expectation. This host's
// non-interactive shells export both, and a test that inherited them would be testing a different
// rung than the one it names.
func plainEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TERM", "dumb")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("COLORTERM", "")
	// $TMUX moves the measure (internal/text.Width dispatches on it), so a literal expectation
	// pins it too: the frames below are the bare terminal's, and TestTheMeasureFollowsTheEnvironment
	// is where the multiplexer's own measure is asserted.
	t.Setenv("TMUX", "")
}

// colourEnv pins a colour-capable environment, the other half of every colour assertion.
func colourEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("NO_COLOR", "")
	t.Setenv("TMUX", "")
}

// framePosts is the content of the UI-SPEC's two worked frames: the six-block list with a quote, a
// repost with two images, a continuation pair, a wide-cluster line with an over-long URL and a
// tombstone. It is hand-authored here because those frames are their own fixture — the recorded
// response and the adversarial page set are other lists, and plan 05 stores those.
func framePosts() []domain.Post {
	lzd := domain.User{ID: "u-lzd", ScreenName: "lzd"}
	aergon := domain.User{ID: "u-aergon", ScreenName: "aergon"}
	mira := domain.User{ID: "u-mira", ScreenName: "mira"}
	kestrel := domain.User{ID: "u-kestrel", ScreenName: "kestrel"}
	wide := domain.User{ID: "u-longhandle_name", ScreenName: "longhandle_name"}

	return []domain.Post{
		{
			ID: "1", Author: lzd, CreatedAt: specNow, ConversationID: "1",
			Text: "cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands back one you have seen, that is the end of the walk.",
			Quote: &domain.Post{
				ID: "0", Author: aergon, CreatedAt: specNow.Add(-12 * time.Minute), ConversationID: "1",
				Text: "the page cache is keyed by the cursor, not by an index.",
			},
		},
		{
			ID: "2", Author: wide, CreatedAt: specNow.Add(-31 * time.Minute), ConversationID: "2",
			RepostedBy: &aergon,
			Text:       "keyset pagination, no offsets: scrolling back up costs zero requests because the pages are already in memory.",
			Media: []domain.MediaRef{
				{Kind: domain.Image, URL: "https://example.invalid/a.jpg"},
				{Kind: domain.Image, URL: "https://example.invalid/b.jpg"},
			},
		},
		{
			ID: "3", Author: mira, CreatedAt: specNow.Add(-52 * time.Minute), ConversationID: "3",
			Text: "the pane never lies about what it is showing - that is the only rule i care about here.",
		},
		{
			ID: "4", Author: mira, CreatedAt: specNow.Add(-51 * time.Minute), ConversationID: "3", ReplyTo: "3",
			Text: "and a stalled cursor is reported, never retried.",
		},
		{
			ID: "5", Author: aergon, CreatedAt: specNow.Add(-67 * time.Minute), ConversationID: "5",
			Text: "日本語のテキストは2セル幅です。 🚀 and 👨‍👩‍👧 measure as one cluster, and this link breaks: https://example.com/a/very/long/path/that/does/not/fit",
		},
		{
			ID: "6", Author: kestrel, CreatedAt: specNow.Add(-82 * time.Minute), ConversationID: "6",
			Visibility: domain.Unavailable,
		},
	}
}

// rowsOf is every row of every block of one render, in order: what the pane will lay out.
func rowsOf(blocks []Block) []string {
	rows := make([]string, 0, 8*len(blocks))
	for _, block := range blocks {
		rows = append(rows, block.Rows...)
	}
	return rows
}

// assertRows compares a render with an expected list of rows, and reports the first row that
// differs with its width, so a wrap-point failure names the cell count that moved.
func assertRows(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d rows, want %d\n got: %q\nwant: %q", name, len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: row %d differs\n got (%d cells): %q\nwant (%d cells): %q",
				name, i+1, text.Width(got[i]), got[i], text.Width(want[i]), want[i])
		}
	}
}

// identityCells is the plain grammar's identity column width, read off its own head template: the
// column is not a struct field any more — the field that used to hold it is deleted, and the
// template's `{handle:16}` is the one mechanism that places it — so a test that spelled 16 again
// would be a second copy of the layout, and one that read it with the composer's own parser is
// reading the grammar's declaration.
func identityCells(t *testing.T) int {
	t.Helper()
	fields := parsePlaceholders(Plain.Head[0])
	if len(fields) == 0 || fields[0].name != PlaceholderHandle || fields[0].cells < 1 {
		t.Fatalf("the plain grammar's head template %q declares no identity column", Plain.Head[0])
	}
	return fields[0].cells
}

// TestBlocksTheUISpecFrameAt80Columns is the UI-SPEC's primary worked frame, transcribed row for
// row, with the leading selection cell included in every expectation.
//
// One row differs from the fenced example, and it is asserted as the engine produces it rather
// than as the UI-SPEC shows it: for the first post the example breaks after "cursor." (a 74-cell
// line) although the next token, "if", fits in the 77-cell measure exactly. Every other row of the
// frame — the head rows, the quote's two rows, the repost head with its provenance, the
// continuation with no separator row above it, the attachment line, the tombstone and the
// wide-cluster wrap — is byte-identical, and
// TestTheFrameDivergencesFromTheUISpecAreTheMeasuredOnes holds both numbers.
func TestBlocksTheUISpecFrameAt80Columns(t *testing.T) {
	plainEnv(t)
	assertRows(t, "the 80-column frame", rowsOf(Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil)), []string{
		" lzd              Oct 14 09:12",
		"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if",
		"   the upstream hands back one you have seen, that is the end of the walk.",
		" > aergon           Oct 14 09:00",
		" >   the page cache is keyed by the cursor, not by an index.",

		"",
		">longhandle_name  Oct 14 08:41  reposted by aergon",
		">  keyset pagination, no offsets: scrolling back up costs zero requests because",
		">  the pages are already in memory.",
		">  [1] image, [2] image",

		"",
		" mira             Oct 14 08:20",
		"   the pane never lies about what it is showing - that is the only rule i care",
		"   about here.",
		" mira             Oct 14 08:21  continued",
		"   and a stalled cursor is reported, never retried.",

		"",
		" aergon           Oct 14 08:05",
		"   日本語のテキストは2セル幅です。 🚀 and 👨‍👩‍👧 measure as one cluster, and this",
		"   link breaks: https://example.com/a/very/long/path/that/does/not/fit",

		"",
		" kestrel          Oct 14 07:50",
		"   post unavailable",
	})
}

// TestBlocksTheUISpecFrameAt120Columns is the same list, the same selection and the same state at
// 120 columns. The measure is capped at 96 cells, so the body does not stretch to 120 — which is
// the point of the frame — and the rows below are the ones the cap produces. The UI-SPEC's fenced
// 120-column frame wraps two of these posts wider than the cap (114 and 109 cells of text), which
// TestTheFrameDivergencesFromTheUISpecAreTheMeasuredOnes records.
func TestBlocksTheUISpecFrameAt120Columns(t *testing.T) {
	plainEnv(t)
	assertRows(t, "the 120-column frame", rowsOf(Blocks(framePosts(), Plain, 120, Selection{ID: specSelected}, specNow, nil)), []string{
		" lzd              Oct 14 09:12",
		"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands",
		"   back one you have seen, that is the end of the walk.",
		" > aergon           Oct 14 09:00",
		" >   the page cache is keyed by the cursor, not by an index.",

		"",
		">longhandle_name  Oct 14 08:41  reposted by aergon",
		">  keyset pagination, no offsets: scrolling back up costs zero requests because the pages are",
		">  already in memory.",
		">  [1] image, [2] image",

		"",
		" mira             Oct 14 08:20",
		"   the pane never lies about what it is showing - that is the only rule i care about here.",
		" mira             Oct 14 08:21  continued",
		"   and a stalled cursor is reported, never retried.",

		"",
		" aergon           Oct 14 08:05",
		"   日本語のテキストは2セル幅です。 🚀 and 👨‍👩‍👧 measure as one cluster, and this link breaks:",
		"   https://example.com/a/very/long/path/that/does/not/fit",

		"",
		" kestrel          Oct 14 07:50",
		"   post unavailable",
	})
}

// TestTheTwoGrammarsExample is the UI-SPEC's proof that the content/presentation split is real:
// the same post, rendered by the same composer, with the author outside and the quoted author
// inside it.
func TestTheTwoGrammarsExample(t *testing.T) {
	plainEnv(t)
	aergon := domain.User{ID: "u-aergon", ScreenName: "aergon"}
	lzd := domain.User{ID: "u-lzd", ScreenName: "lzd"}
	post := domain.Post{
		ID: "1", Author: aergon, CreatedAt: specNow, ConversationID: "1",
		Text: "cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands back one you have seen, that is the end of the walk.",
		Quote: &domain.Post{
			ID: "0", Author: lzd, CreatedAt: specNow.Add(-12 * time.Minute), ConversationID: "1",
			Text: "the page cache is keyed by the cursor, not by an index.",
		},
	}
	assertRows(t, "the two-grammars example", rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{ID: ""}, specNow, nil)), []string{
		" aergon           Oct 14 09:12",
		"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if",
		"   the upstream hands back one you have seen, that is the end of the walk.",
		" > lzd              Oct 14 09:00",
		" >   the page cache is keyed by the cursor, not by an index.",
	})
}

// TestTheFrameDivergencesFromTheUISpecAreTheMeasuredOnes holds the two places where the UI-SPEC's
// fenced frames cannot be the output of the width model the same document specifies, with the
// numbers:
//
//   - at 80 columns the example breaks the first body line after "cursor." — a 74-cell line —
//     although the next token fits in the measure exactly (74 + 1 + 2 == 77 == min(80-3, 96)). An
//     engine that wrapped there would be wrapping at 76, not at the measure.
//   - at 120 columns the example's first body line carries 114 cells of text and its keyset line
//     109, both wider than the document's own 96-cell cap, and wider than the rows A7 audits
//     ("no rendered row of any golden frame exceeds min(W-3, 96) cells of text, at any width"). A
//     renderer that reproduced those rows would be ignoring the cap at exactly the width the cap
//     exists for.
//
// The plan is the spec for the layout and its measure is `min(W - 3 - 2*depth, 96)`; these are the
// numbers that say which of the two a frame can be.
func TestTheFrameDivergencesFromTheUISpecAreTheMeasuredOnes(t *testing.T) {
	plainEnv(t)
	posts := framePosts()

	// The UI-SPEC's 80-column first body line, as fenced, and the token that fits after it.
	fenced := "   cursor in, cursor out: the render layer never does arithmetic on a cursor."
	if got := text.Width(fenced) - 3; got != 74 {
		t.Fatalf("the fenced first body line is %d cells of text, want 74", got)
	}
	// The three cells of selection column and indent are not part of the measure, so the text is
	// what is compared: 74 cells, then the separating space and the two-cell token.
	if got := text.Width(fenced) - 3 + 1 + text.Width("if"); got != 77 {
		t.Fatalf("the fenced line plus the next token is %d cells of text, want the 77-cell measure", got)
	}
	rows := rowsOf(Blocks(posts, Plain, 80, Selection{ID: specSelected}, specNow, nil))
	if got := text.Width(rows[1]) - 3; got != 77 {
		t.Fatalf("the engine's first body line is %d cells of text, want the 77-cell measure filled", got)
	}
	if !strings.HasSuffix(rows[1], " if") {
		t.Fatalf("the engine's first body line does not carry the token that fits: %q", rows[1])
	}

	// The UI-SPEC's 120-column rows, as fenced: 114 and 109 cells of text against a 96-cell cap.
	for _, fenced := range []struct {
		row   string
		cells int
	}{
		{"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands back one you have", 114},
		{">  keyset pagination, no offsets: scrolling back up costs zero requests because the pages are already in memory.", 109},
	} {
		if got := text.Width(fenced.row); got != fenced.cells+3 {
			t.Fatalf("the fenced 120-column row is %d cells wide, want %d", got, fenced.cells+3)
		}
		if fenced.cells <= Plain.MeasureCap {
			t.Fatalf("the fenced row is %d cells of text, which the %d-cell cap would hold: this test's premise is stale",
				fenced.cells, Plain.MeasureCap)
		}
	}

	// What the cap produces instead: every body row of the 120-column frame is at or under the
	// measure, and at least one fills it exactly, so the cap is visibly the limit it claims to be
	// rather than a number nothing reaches.
	rows = rowsOf(Blocks(posts, Plain, 120, Selection{ID: specSelected}, specNow, nil))
	filled := 0
	for _, row := range rows {
		// A depth-0 body row is the selection cell, the two-cell indent and then the text; a
		// quoted body row starts one cell further in, and is not what the cap is measured on here.
		if !strings.HasPrefix(row, "   ") && !strings.HasPrefix(row, ">  ") {
			continue
		}
		width := text.Width(row) - 3
		if width > Plain.MeasureCap {
			t.Fatalf("body row %q is %d cells of text, wider than the %d-cell cap", row, width, Plain.MeasureCap)
		}
		if width == Plain.MeasureCap {
			filled++
		}
	}
	if filled == 0 {
		t.Fatal("no body row at 120 columns fills the 96-cell measure: the cap cannot be seen to bind")
	}
}

// TestTheMeasureIsTheCapNotThePane is A7 at the unit level: at any width — including one wider
// than the cap by a hundred cells — no row carries more text than the measure allows, and the
// surplus columns stay empty.
func TestTheMeasureIsTheCapNotThePane(t *testing.T) {
	plainEnv(t)
	for _, w := range []int{60, 80, 120, 200} {
		measure := min(w-3, Plain.MeasureCap)
		for _, row := range rowsOf(Blocks(framePosts(), Plain, w, Selection{ID: specSelected}, specNow, nil)) {
			if width := text.Width(row); width > 3+measure {
				t.Errorf("at %d columns, row %q is %d cells wide, over 3+%d", w, row, width, measure)
			}
		}
	}
}

// TestTheHandleColumnIsFixed covers the identity column: a legal 15-cell handle is printed whole
// in a 16-cell field, a wider one is truncated at a cluster boundary with the truncation mark in
// its last cell, and the stamp behind the column starts at the same cell either way.
func TestTheHandleColumnIsFixed(t *testing.T) {
	plainEnv(t)
	handle := func(name string) domain.Post {
		return domain.Post{
			ID: "1", Author: domain.User{ID: "u-" + name, ScreenName: name}, CreatedAt: specNow,
			Text: "a body",
		}
	}

	head := Blocks([]domain.Post{handle("longhandle_name")}, Plain, 80, Selection{ID: ""}, specNow, nil)[0].Rows[0]
	if head != " longhandle_name  Oct 14 09:12" {
		t.Fatalf("a 15-cell handle rendered as %q, want the whole handle and the stamp behind it", head)
	}
	if got := text.Width("longhandle_name"); got != 15 {
		t.Fatalf("the fixture's handle is %d cells, want 15", got)
	}
	// A 19-cell identity in a 16-cell column: the 15 cells that fit, then the truncation mark.
	// A 20-cell identity in a 16-cell column: the 15 cells that fit, then the truncation mark.
	over := "a_handle_of_17_cells"
	if got := text.Width(over); got != 20 {
		t.Fatalf("the fixture's over-long handle is %d cells, want 20", got)
	}
	cut := Blocks([]domain.Post{handle(over)}, Plain, 80, Selection{ID: ""}, specNow, nil)[0].Rows[0]
	if !strings.HasPrefix(cut, " a_handle_of_17_~ Oct 14 09:12") {
		t.Fatalf("an over-long handle rendered as %q, want 15 cells cut and marked before the stamp", cut)
	}
	if got, want := strings.Index(cut, "Oct 14"), strings.Index(head, "Oct 14"); got != want {
		t.Fatalf("the stamp moved with the identity: column %d against %d", got, want)
	}
	cells := identityCells(t)
	if got := text.Width(cut[1 : 1+cells]); got != cells {
		t.Fatalf("the identity column is %d cells on the truncated row, want %d", got, cells)
	}
}

// TestAContinuationCarriesNoSeparator: a reply whose parent is the block above it, by the same
// author, drops the blank separator row and marks its own head row. The rule is directional — the
// parent, rendered under its reply, gets no marker — and it needs both accounts to be named: two
// posts with no author at all are not two posts by the same one.
func TestAContinuationCarriesNoSeparator(t *testing.T) {
	plainEnv(t)
	mira := domain.User{ID: "u-mira", ScreenName: "mira"}
	parent := domain.Post{ID: "3", Author: mira, CreatedAt: specNow, ConversationID: "3", Text: "the parent"}
	reply := domain.Post{ID: "4", Author: mira, CreatedAt: specNow.Add(-time.Minute), ConversationID: "3", ReplyTo: "3", Text: "the reply"}
	stranger := domain.Post{ID: "5", Author: domain.User{ID: "u-other", ScreenName: "other"}, CreatedAt: specNow.Add(-2 * time.Minute), ReplyTo: "4", Text: "another account's reply"}
	anonymous := domain.Post{ID: "6", CreatedAt: specNow.Add(-3 * time.Minute), ReplyTo: "5", Text: "no author at all"}
	anonymousParent := domain.Post{ID: "7", CreatedAt: specNow.Add(-4 * time.Minute), Text: "no author either"}

	blocks := Blocks([]domain.Post{parent, reply}, Plain, 80, Selection{ID: ""}, specNow, nil)
	if blocks[1].Rows[0] == "" {
		t.Fatalf("a continuation kept the separator row: %q", blocks[1].Rows)
	}
	if got := blocks[1].Rows[0]; got != " mira             Oct 14 09:11  continued" {
		t.Fatalf("the continuation's head row is %q, want the marker", got)
	}
	if got := blocks[0].Rows[0]; strings.Contains(got, "continued") {
		t.Fatalf("the parent above the reply carries the marker: %q", got)
	}

	if blocks := Blocks([]domain.Post{reply, parent}, Plain, 80, Selection{ID: ""}, specNow, nil); strings.Contains(blocks[1].Rows[0], "continued") {
		t.Fatalf("a parent below its reply was marked as the continuation: %q", blocks[1].Rows[0])
	}
	if blocks := Blocks([]domain.Post{reply, stranger}, Plain, 80, Selection{ID: ""}, specNow, nil); strings.Contains(blocks[1].Rows[0], "continued") {
		t.Fatalf("a reply by another account was marked as the continuation: %q", blocks[1].Rows[0])
	}
	if blocks := Blocks([]domain.Post{anonymousParent, anonymous}, Plain, 80, Selection{ID: ""}, specNow, nil); strings.Contains(blocks[1].Rows[0], "continued") {
		t.Fatalf("two posts with no author at all were marked as one author's thread: %q", blocks[1].Rows[0])
	}
}

// TestSelectionIsTheMarkAndOnlyTheMark: every row of the selected block starts with the selection
// mark, every row of every other block does not, and the mark follows the block's post id rather
// than a row index. It is asserted under the rung that emits no reverse video at all, so nothing
// but the character carries the selection.
func TestSelectionIsTheMarkAndOnlyTheMark(t *testing.T) {
	plainEnv(t)
	blocks := Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil)
	selected := 0
	for _, block := range blocks {
		if len(block.Rows) == 0 {
			t.Fatalf("block %q has no rows", block.PostID)
		}
		want := block.PostID == specSelected
		if want {
			selected++
		}
		for _, row := range block.Rows {
			if row == "" {
				continue // the separator row belongs to the block but is not one of its rows
			}
			if got := strings.HasPrefix(row, MarkSelection); got != want {
				t.Fatalf("block %q: row %q has mark %v, want %v", block.PostID, row, got, want)
			}
		}
	}
	if selected != 1 {
		t.Fatalf("%d blocks are selected, want exactly the one post id", selected)
	}
	for _, row := range rowsOf(Blocks(framePosts(), Plain, 80, Selection{ID: "nobody"}, specNow, nil)) {
		if strings.HasPrefix(row, MarkSelection) {
			t.Fatalf("a post id that is in no block still marked a row: %q", row)
		}
	}
}

// TestTheListGrammarPrintsNoCountAndNoURL is the absence test: the post carries engagement counts
// and media URLs, and not one of the counts and not one byte of a URL reaches a row. The counts
// are five digits each so that no digit of a stamp or of a count in the attachment line can be
// mistaken for one, and the test first asserts the fixture really carries them.
func TestTheListGrammarPrintsNoCountAndNoURL(t *testing.T) {
	plainEnv(t)
	post := domain.Post{
		ID: "1", Author: domain.User{ID: "u-lzd", ScreenName: "lzd"}, CreatedAt: specNow,
		Text:   "a post with no number in its body",
		Counts: domain.Counts{Replies: 11111, Reposts: 22222, Likes: 33333, Quotes: 44444, Bookmarks: 55555, Views: 66666},
		Media: []domain.MediaRef{
			{Kind: domain.Image, URL: "https://example.invalid/secret-path.jpg", AltText: "an alt text the list does not print"},
			{Kind: domain.Video, URL: "https://example.invalid/another-secret.mp4"},
		},
	}
	if post.Counts.Replies == 0 || post.Media[0].URL == "" || post.Media[0].AltText == "" {
		t.Fatal("the fixture stopped carrying the counts, the URL or the alt text this test is about")
	}
	rows := rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{ID: ""}, specNow, nil))
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "Oct 14 09:12") {
		t.Fatalf("the rows under test do not carry the stamp, so they are not a rendered post:\n%s", joined)
	}
	for _, count := range []string{"11111", "22222", "33333", "44444", "55555", "66666"} {
		if strings.Contains(joined, count) {
			t.Fatalf("the engagement count %s reached the list grammar:\n%s", count, joined)
		}
	}
	for _, leak := range []string{"https://", "example.invalid", "secret", "alt text"} {
		if strings.Contains(joined, leak) {
			t.Fatalf("the media URL or alt text (%s) reached the list grammar:\n%s", leak, joined)
		}
	}
	if !strings.Contains(joined, "[1] image, [2] video") {
		t.Fatalf("the attachment line does not describe the two kinds present:\n%s", joined)
	}
	if Plain.ShowCounts {
		t.Fatal("the plain grammar has ShowCounts on: the list prints engagement numbers")
	}
}

// TestMarkerPlaceholders covers the body a post renders when it has nothing else: a tombstone, a
// post the upstream withheld, a post with no text at all, one behind a limited wrapper, and one
// whose only content is an attachment. A tombstone has no author and no date either, and its head
// row says nothing rather than inventing a date for it.
func TestMarkerPlaceholders(t *testing.T) {
	plainEnv(t)
	lzd := domain.User{ID: "u-lzd", ScreenName: "lzd"}
	tombstone := domain.Post{ID: "1", Visibility: domain.Unavailable, ConversationID: "1"}
	withheld := domain.Post{ID: "2", Author: lzd, CreatedAt: specNow, Text: "a body nobody may read", Visibility: domain.Withheld}
	empty := domain.Post{ID: "3", Author: lzd, CreatedAt: specNow, ConversationID: "3"}
	limited := domain.Post{ID: "4", Author: lzd, CreatedAt: specNow, Text: "a body the upstream is holding back", Visibility: domain.Limited}
	mediaOnly := domain.Post{
		ID: "5", Author: lzd, CreatedAt: specNow,
		Media: []domain.MediaRef{{Kind: domain.GIF, URL: "https://example.invalid/c.gif"}},
	}

	blocks := Blocks([]domain.Post{tombstone, withheld, empty, limited, mediaOnly}, Plain, 80, Selection{ID: ""}, specNow, nil)
	assertRows(t, "the placeholders", rowsOf(blocks), []string{
		strings.Repeat(" ", 1+identityCells(t)+1+StampCells), // no author and no date: nothing at all
		"   post unavailable",

		"",
		" lzd              Oct 14 09:12",
		"   post unavailable",

		"",
		" lzd              Oct 14 09:12",
		"   post text missing",

		"",
		" lzd              Oct 14 09:12  limited",
		"   a body the upstream is holding back",

		"",
		" lzd              Oct 14 09:12",
		"   [1] gif",
	})
	head := blocks[0].Rows[0]
	if strings.TrimSpace(head) != "" {
		t.Fatalf("the tombstone's head row says %q, want nothing at all", head)
	}
	if got := text.Width(head); got != 1+identityCells(t)+1+StampCells {
		t.Fatalf("the tombstone's head row is %d cells, want the head row's columns", got)
	}
	if strings.Contains(rowsOf(blocks)[1], "Jan 01") {
		t.Fatal("the tombstone's head row carries a fabricated date")
	}
	if len(StateBlock(Plain, 80, "head", "").Rows) != 1 {
		t.Fatal("a state block with no body rendered more than its head line")
	}
}

// TestTheAttachmentLineCountsEveryKind: one counted noun per kind present, in the fixed order
// image, video, gif, singular at one and plural otherwise, and the kinds that are absent are not
// mentioned at all.
func TestTheAttachmentLineCountsEveryKind(t *testing.T) {
	plainEnv(t)
	media := func(kinds ...domain.MediaKind) []domain.MediaRef {
		refs := make([]domain.MediaRef, 0, len(kinds))
		for i, kind := range kinds {
			refs = append(refs, domain.MediaRef{Kind: kind, URL: "https://example.invalid/" + string(rune('a'+i))})
		}
		return refs
	}
	tests := []struct {
		name  string
		media []domain.MediaRef
		want  string
	}{
		{"one image", media(domain.Image), "[1] image"},
		{"two images", media(domain.Image, domain.Image), "[1] image, [2] image"},
		{"the fixed order", media(domain.GIF, domain.Video, domain.Image, domain.Image), "[1] gif, [2] video, [3] image, [4] image"},
		{"one of each", media(domain.GIF, domain.Video, domain.Image), "[1] gif, [2] video, [3] image"},
		{"only a video", media(domain.Video), "[1] video"},
	}
	for _, tc := range tests {
		post := domain.Post{ID: "1", Author: domain.User{ID: "u-lzd", ScreenName: "lzd"}, CreatedAt: specNow, Media: tc.media}
		rows := rowsOf(Blocks([]domain.Post{post}, Plain, 80, Selection{ID: ""}, specNow, nil))
		if len(rows) != 2 || strings.TrimSpace(rows[1]) != tc.want {
			t.Errorf("%s rendered %q, want the body row %q", tc.name, rows, tc.want)
		}
	}
}

// TestTheQuotePrefixStopsGrowingAtTheCap: a quote renders its own rows at its prefix's origin, a
// quote inside a quote renders at two levels, the measure shrinks by two cells per level, and a
// level deeper than the cap keeps the two-level prefix and its text rather than collapsing.
func TestTheQuotePrefixStopsGrowingAtTheCap(t *testing.T) {
	plainEnv(t)
	lzd := domain.User{ID: "u-lzd", ScreenName: "lzd"}
	deepest := domain.Post{ID: "3", Author: lzd, CreatedAt: specNow.Add(-2 * time.Minute), Text: "the innermost body"}
	middle := domain.Post{ID: "2", Author: lzd, CreatedAt: specNow.Add(-time.Minute), Text: "the middle body", Quote: &deepest}
	outer := domain.Post{ID: "1", Author: lzd, CreatedAt: specNow, Text: "the outer body", Quote: &middle}

	assertRows(t, "a depth-2 quote", rowsOf(Blocks([]domain.Post{outer}, Plain, 80, Selection{ID: ""}, specNow, nil)), []string{
		" lzd              Oct 14 09:12",
		"   the outer body",
		" > lzd              Oct 14 09:11",
		" >   the middle body",
		" > > lzd              Oct 14 09:10",
		" > >   the innermost body",
	})
	if Plain.Markers.QuoteDepthCap != 2 {
		t.Fatalf("the plain grammar's quote depth cap is %d, want 2", Plain.Markers.QuoteDepthCap)
	}

	// A level deeper than the cap keeps the two-level prefix and its text: nothing is omitted.
	deepest.Quote = &domain.Post{ID: "4", Author: lzd, CreatedAt: specNow.Add(-3 * time.Minute), Text: "a fourth level"}
	deeper := strings.Join(rowsOf(Blocks([]domain.Post{outer}, Plain, 80, Selection{ID: ""}, specNow, nil)), "\n")
	if !strings.Contains(deeper, "a fourth level") {
		t.Fatalf("a quote deeper than the cap lost its text:\n%s", deeper)
	}
	if strings.Contains(deeper, "> > > ") {
		t.Fatalf("the quote prefix grew past the cap:\n%s", deeper)
	}

	// The measure shrinks by two cells per level: at the 60-column floor the depth-1 measure is
	// 55 cells and the depth-2 measure is 53, so a 55-cell token fits on one line one level down
	// and wraps two levels down.
	token := strings.Repeat("x", 55)
	quoted := domain.Post{ID: "6", Author: lzd, CreatedAt: specNow.Add(-5 * time.Minute), Text: token}
	oneLevel := domain.Post{ID: "5", Author: lzd, CreatedAt: specNow.Add(-4 * time.Minute), Text: "a short body", Quote: &quoted}
	twoLevels := domain.Post{ID: "7", Author: lzd, CreatedAt: specNow.Add(-6 * time.Minute), Text: "a short body", Quote: &quoted}

	rows := rowsOf(Blocks([]domain.Post{oneLevel}, Plain, MinPaneCols, Selection{ID: ""}, specNow, nil))
	if len(rows) != 4 {
		t.Fatalf("a 55-cell token one level down rendered %d rows, want 4 (head, body, quoted head, quoted body):\n%q", len(rows), rows)
	}
	if got := text.Width(rows[3]); got != 3+2+55 {
		t.Fatalf("the quoted body row is %d cells, want the whole 55-cell token behind a depth-1 prefix: %q", got, rows[3])
	}
	rows = rowsOf(Blocks([]domain.Post{domain.Post{
		ID: "8", Author: lzd, CreatedAt: specNow.Add(-7 * time.Minute), Text: "a short body", Quote: &twoLevels,
	}}, Plain, MinPaneCols, Selection{ID: ""}, specNow, nil))
	if len(rows) != 7 {
		t.Fatalf("the same token two levels down rendered %d rows, want 7: the measure did not shrink\n%q", len(rows), rows)
	}
	// The token is wrapped and not lost: the two rows it broke into carry it whole, and the first
	// of them fills the depth-2 measure (53 cells at the 60-column floor) exactly.
	wrapped := rows[len(rows)-2:]
	line := func(row string) string { return row[1+2*Plain.Markers.QuoteDepthCap+QuoteBodyIndent:] }
	if got := line(wrapped[0]) + line(wrapped[1]); got != token {
		t.Fatalf("the two levels down body rows carry %q, want the whole token", got)
	}
	if got := text.Width(line(wrapped[0])); got != 53 {
		t.Fatalf("the depth-2 body line is %d cells of text, want the 53-cell measure: %q", got, wrapped[0])
	}
}

// TestABlockRowNeverExceedsThePane: the frame truncates no content row, so the builder must not
// produce one. A head whose markers no longer fit behind the identity column and the stamp wraps
// with text.Wrap — the same wrap the body uses — and nothing is lost at the break: the marker that
// does not fit on the first row is on the next one, and the stamp, which is the first thing behind
// the identity column, is never what gets cut.
func TestABlockRowNeverExceedsThePane(t *testing.T) {
	plainEnv(t)
	post := domain.Post{
		ID: "1", Author: domain.User{ID: "u-a", ScreenName: "a_15_cell_handl"}, CreatedAt: specNow,
		RepostedBy: &domain.User{ID: "u-b", ScreenName: "a_fifteen_cell"},
		Text:       "a body",
		Visibility: domain.Limited,
	}
	for _, w := range []int{MinPaneCols, 79, 80, 120, 200} {
		for _, row := range rowsOf(Blocks([]domain.Post{post}, Plain, w, Selection{ID: ""}, specNow, nil)) {
			if text.Width(row) > w {
				t.Errorf("at %d columns, row %q is %d cells wide", w, row, text.Width(row))
			}
		}
	}

	// The head needs 1 + 16 + 1 + 12 + 2 + 26 + 2 + 7 = 67 cells and the pane's floor is 60, so at
	// the floor it wraps into two rows. The whole row's text is still on screen, once, in order —
	// which is what "the frame truncates no content row" has to mean.
	head := rowsOf(Blocks([]domain.Post{post}, Plain, MinPaneCols, Selection{ID: ""}, specNow, nil))
	if len(head) != 3 {
		t.Fatalf("the head at the narrow floor rendered %d rows, want 3 (two head rows and the body): %q", len(head), head)
	}
	whole := strings.Join([]string{head[0], head[1]}, " ")
	if got := text.Width(head[0]); got > MinPaneCols {
		t.Fatalf("the first head row is %d cells, over the %d-cell pane: %q", got, MinPaneCols, head[0])
	}
	for _, want := range []string{"a_15_cell_handl", "Oct 14 09:12", "reposted by a_fifteen_cell", "limited"} {
		if !strings.Contains(whole, want) {
			t.Fatalf("the wrapped head lost %q: %q", want, whole)
		}
	}
	if strings.Contains(head[0], "limited") {
		t.Fatalf("the first head row carries the marker that did not fit: %q", head[0])
	}
	if !strings.Contains(head[0], "Oct 14 09:12") {
		t.Fatalf("the wrap moved the stamp off the first row: %q", head[0])
	}
}

// serviceGrammar is a second display style for the seam test: the same content, a different
// identity function, a different stamp and a different provenance marker. It is not one of the
// phase's four covers — it is a Grammar built by hand out of Phase 2's own fields, and it is here so
// that a change of grammar can be shown to change the identity and the chrome and nothing else.
func serviceGrammar() Grammar {
	g := Plain
	g.Name = "test-second-style"
	g.Identity = func(u domain.User) CoverAlias {
		return CoverAlias{Handle: "id-" + u.ID, Name: "name-" + u.ID}
	}
	g.Stamp = StampFormat{Clock: "2006-01-02T15:04:05Z", Date: "2006-01-02T15:04:05Z"}
	// The head declares no width for the stamp, because this style's stamp is wider than the plain
	// reader's 12-cell column: a template that declared {stamp:12} would cut it, and the point of the
	// field's width is that a format is as wide as it says it is.
	g.Head = []string{"{handle:16} {stamp}{markers}"}
	g.QuoteHead = g.Head
	g.Markers.Repost = "surfaced by {handle}"
	return g
}

// TestTheGrammarIsTheOnlyThingThatChanges is the seam, asserted the way it can fail: the same
// posts rendered under two grammars keep their order, their text and their row count, and the
// cover's rows carry the cover's identity and instant instead of the plain reader's. A renderer
// that reached for the handle or the time itself would print the plain handle under the cover.
func TestTheGrammarIsTheOnlyThingThatChanges(t *testing.T) {
	plainEnv(t)
	posts := framePosts()
	plainBlocks := Blocks(posts, Plain, 80, Selection{ID: specSelected}, specNow, nil)
	coverBlocks := Blocks(posts, serviceGrammar(), 80, Selection{ID: specSelected}, specNow, nil)

	if len(plainBlocks) != len(coverBlocks) {
		t.Fatalf("%d blocks under the cover, want %d: the grammar changed the list", len(coverBlocks), len(plainBlocks))
	}
	for i := range plainBlocks {
		if plainBlocks[i].PostID != coverBlocks[i].PostID {
			t.Fatalf("block %d is %q under the cover, want %q", i, coverBlocks[i].PostID, plainBlocks[i].PostID)
		}
		if len(plainBlocks[i].Rows) != len(coverBlocks[i].Rows) {
			t.Fatalf("block %q has %d rows under the cover, want %d: the grammar changed the layout",
				plainBlocks[i].PostID, len(coverBlocks[i].Rows), len(plainBlocks[i].Rows))
		}
	}

	plainRows := strings.Join(rowsOf(plainBlocks), "\n")
	coverRows := strings.Join(rowsOf(coverBlocks), "\n")
	if !strings.Contains(coverRows, "id-u-aergon") || !strings.Contains(coverRows, "2026-10-14T09:12:00Z") {
		t.Fatalf("the cover grammar did not decide the identity and the stamp:\n%s", coverRows)
	}
	if strings.Contains(coverRows, " aergon ") {
		t.Fatalf("the cover printed the plain reader's handle: the identity leaked out of the grammar:\n%s", coverRows)
	}
	if !strings.Contains(plainRows, " aergon ") {
		t.Fatalf("the plain grammar stopped printing the handle:\n%s", plainRows)
	}
	if !strings.Contains(coverRows, "surfaced by id-u-aergon") {
		t.Fatalf("the cover's provenance marker was not used:\n%s", coverRows)
	}
	for _, want := range []string{
		"the page cache is keyed by the cursor, not by an index.",
		"the pane never lies about what it is showing",
		"日本語のテキストは2セル幅です。",
	} {
		if !strings.Contains(coverRows, want) {
			t.Fatalf("the cover lost the content %q", want)
		}
	}
}

// TestColourNeverCarriesMeaningAlone is A1 at the block level: with a truecolor-capable
// environment the rows do carry escapes — so the comparison below is not between two plain
// strings — and stripping them leaves exactly the mono rung's characters.
func TestColourNeverCarriesMeaningAlone(t *testing.T) {
	plainEnv(t)
	plainRows := rowsOf(Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil))

	colourEnv(t)
	colourRows := rowsOf(Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil))

	coloured, selected, stamp := 0, 0, 0
	for i, row := range colourRows {
		if row != ansi.Strip(row) {
			coloured++
		}
		if got, want := ansi.Strip(row), plainRows[i]; got != want {
			t.Fatalf("row %d strips to %q, want the mono row %q", i+1, got, want)
		}
		if strings.HasPrefix(plainRows[i], MarkSelection) && row != plainRows[i] {
			selected++
		}
		if strings.Contains(plainRows[i], "Oct 14 08:41") && row != plainRows[i] {
			stamp++
		}
	}
	if coloured == 0 {
		t.Fatal("no row of the coloured render carries an escape: the comparison above is between two plain renders")
	}
	if selected == 0 {
		t.Error("no row of the selected block is painted")
	}
	if stamp == 0 {
		t.Error("no head row's stamp is painted muted")
	}
}

// TestTheStampColumnNeverMoves: the two forms the plain grammar uses are exactly StampCells wide
// whatever the instant and whatever the clock's zone, the zero instant carries no date at all, and
// a grammar whose stamp is wider than the column keeps its own width instead of being cut.
func TestTheStampColumnNeverMoves(t *testing.T) {
	plainEnv(t)
	tokyo := time.FixedZone("UTC+9", 9*60*60)
	tests := []struct {
		name          string
		clock, layout string
		at            time.Time
		want          string
	}{
		{"this year", "Jan 02 15:04", "Jan 02 15:04", specNow, "Oct 14 09:12"},
		{"another year", "Jan 02 15:04", "Jan 02  2006", time.Date(2025, 1, 2, 15, 4, 0, 0, time.UTC), "Jan 02  2025"},
		{"local time", "Jan 02 15:04", "Jan 02 15:04", time.Date(2026, 10, 14, 23, 30, 0, 0, time.UTC), "Oct 15 08:30"},
		{"the zero instant", "Jan 02 15:04", "Jan 02 15:04", time.Time{}, strings.Repeat(" ", StampCells)},
		{"a cover's wider stamp", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05Z", specNow, "2026-10-14T09:12:00Z"},
	}
	for _, tc := range tests {
		clock := specNow
		if tc.name == "local time" {
			clock = specNow.In(tokyo)
		}
		format := StampFormat{Clock: tc.clock, Date: tc.layout, At: clock}
		got := format.Stamp(tc.at)
		if got != tc.want {
			t.Errorf("%s: Stamp = %q, want %q", tc.name, got, tc.want)
		}
		if width := text.Width(got); width < StampCells {
			t.Errorf("%s: Stamp is %d cells wide, want at least %d", tc.name, width, StampCells)
		}
		if tc.name != "a cover's wider stamp" && text.Width(got) != StampCells {
			t.Errorf("%s: Stamp is %d cells wide, want exactly the %d-cell column", tc.name, text.Width(got), StampCells)
		}
	}
}

// TestStateBlockIsTheSameShapeAsAPost: the loading line, the empty state and a cold failure are
// rendered in the block shape, so the pane never looks like a different kind of screen.
func TestStateBlockIsTheSameShapeAsAPost(t *testing.T) {
	plainEnv(t)
	assertRows(t, "the loading block", StateBlock(Plain, 80, CopyLoadingFollowing, "").Rows, []string{
		" loading following...",
	})
	assertRows(t, "the empty state", StateBlock(Plain, 80, CopyEmptyHeadingFollowing, CopyEmptyBody).Rows, []string{
		" no posts in following",
		"   this is an empty read that succeeded - press r to refresh",
	})
	block := StateBlock(Plain, 80, StateWord(domain.FailureNoRoute), "the failing sentence")
	assertRows(t, "a cold failure", block.Rows, []string{
		" no route to x.com",
		"   the failing sentence",
	})
	if block.PostID != "" {
		t.Fatalf("a state block claims to be the post %q", block.PostID)
	}
}

// TestTheMeasureFollowsTheEnvironment is the renderer's half of the width model's environment
// switch, and the arm that makes "no second measurement" falsifiable here: the same list at the
// same width wraps differently under a multiplexer, because the ZWJ family is one two-cell cluster
// to a terminal and the six cells its code points occupy to a multiplexer's grid. A renderer that
// measured with len(), with a rune count, or with a table of its own would produce the same rows in
// both environments and fail here.
func TestTheMeasureFollowsTheEnvironment(t *testing.T) {
	family := "👨‍👩‍👧"
	plainEnv(t)
	if got := text.Width(family); got != 2 {
		t.Fatalf("the family measures %d cells with no multiplexer, want the 2-cell cluster", got)
	}
	// Each arm's rows are measured while its own environment is set: a row rendered for a
	// terminal and then measured as a multiplexer would measure it is a different number, which is
	// exactly the switch this test is about.
	bare := rowsOf(Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil))
	bareWidths := widths(bare)

	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	if got := text.Width(family); got != 6 {
		t.Fatalf("the family measures %d cells under a multiplexer, want the 6 its code points take", got)
	}
	tmux := rowsOf(Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil))
	tmuxWidths := widths(tmux)

	moved := 0
	for i := range min(len(bare), len(tmux)) {
		if bare[i] != tmux[i] {
			moved++
		}
	}
	if moved == 0 {
		t.Fatal("every row is identical in both environments: nothing was measured through internal/text")
	}
	// The wide-cluster post is the row that must move: it is the one whose measure differs between
	// the two environments at all.
	wide := func(rows []string) string {
		for _, row := range rows {
			if strings.Contains(row, family) {
				return row
			}
		}
		return ""
	}
	if wide(bare) == "" || wide(tmux) == "" {
		t.Fatal("the wide cluster is not rendered in one of the two environments")
	}
	if wide(bare) == wide(tmux) {
		t.Fatalf("the wide-cluster row is the same in both environments: %q", wide(bare))
	}
	for i, width := range append(bareWidths, tmuxWidths...) {
		if width > 80 {
			t.Fatalf("row %d of the frame is %d cells wide in an 80-column pane", i+1, width)
		}
	}
}

// widths measures every row of a render with the environment in force when it is called.
func widths(rows []string) []int {
	out := make([]int, 0, len(rows))
	for _, row := range rows {
		out = append(out, text.Width(row))
	}
	return out
}

// placeholderGrammar is a grammar with the plain reader's fields and a head of the caller's choosing:
// the expander's own fixture, so that a rule below is shown to hold of the expander and not of one
// style's templates.
func placeholderGrammar(head ...string) Grammar {
	g := Plain
	g.Name = "placeholder"
	g.Head = head
	g.QuoteHead = head
	return g
}

// TestThePlaceholderExpander is the template language's whole surface, one rule per case: the closed
// set of fields, the two row rules (an empty template line is a blank row, a line whose only field is
// empty is dropped), the `{x:N}` column with its truncation mark, and the wrap of a line longer than
// the pane.
//
// The arm under each case is the fixture's own text: a case that found no row would pass for a
// renderer that drew nothing at all, so every expectation names the body row beside the head row it is
// about.
func TestThePlaceholderExpander(t *testing.T) {
	plainEnv(t)
	post := func(text string) domain.Post {
		return domain.Post{
			ID: "1", Author: domain.User{ID: "u-lzd", ScreenName: "lzd"}, CreatedAt: specNow,
			Text: text,
		}
	}
	tests := []struct {
		name  string
		head  []string
		post  domain.Post
		want  []string
		width int
	}{
		{
			name:  "a template line whose only field is empty is dropped",
			head:  []string{"{ref}"},
			post:  post("a body"),
			want:  []string{"   a body"},
			width: 80,
		},
		{
			name:  "an empty template line is a blank row",
			head:  []string{"", "{handle}"},
			post:  post("a body"),
			want:  []string{" ", " lzd", "   a body"},
			width: 80,
		},
		{
			name:  "a field is padded to its column",
			head:  []string{"[{handle:8}]"},
			post:  post("a body"),
			want:  []string{" [lzd     ]", "   a body"},
			width: 80,
		},
		{
			name: "a field that does not fit is cut at its column with the mark",
			head: []string{"{handle:8}"},
			post: domain.Post{
				ID: "1", Author: domain.User{ID: "u-a", ScreenName: "a_handle_of_17_cells"}, CreatedAt: specNow,
				Text: "a body",
			},
			want:  []string{" a_handl~", "   a body"},
			width: 80,
		},
		{
			name: "the post's own text can be a head field",
			head: []string{"{text}"},
			post: post("a body"),
			// The text is a field of the head here and the body renders underneath it: a grammar that
			// puts the post's own words in its head renders them there, and the layout below is still
			// the layout. No style this phase ships does it, which is why the case is a template of its
			// own rather than a style.
			want:  []string{" a body", "   a body"},
			width: 80,
		},
		{
			name:  "a placeholder outside the closed set names nothing",
			head:  []string{"{nonsense}"},
			post:  post("a body"),
			want:  []string{"   a body"},
			width: 80,
		},
		{
			name: "a head line wider than the pane wraps at the pane, and the continuation starts at the head's own indent",
			head: []string{"{text}"},
			post: post("the pane never lies about what it is showing - that is the only rule i care about here."),
			// The head's two wrapped rows are what the case is about, and the post's own body is
			// still the block's body underneath them: a head renders a field at the pane's width and
			// a body renders the same text at the measure, so the same sentence is on screen twice
			// here and each of the two is wrapped by the rule that owns it.
			want: []string{
				" the pane never lies about what it is showing - that is the",
				" only rule i care about here.",
				"   the pane never lies about what it is showing - that is",
				"   the only rule i care about here.",
			},
			width: 60,
		},
	}
	for _, tc := range tests {
		got := rowsOf(Blocks([]domain.Post{tc.post}, placeholderGrammar(tc.head...), tc.width, Selection{ID: ""}, specNow, nil))
		assertRows(t, tc.name, got, tc.want)
		for _, row := range got {
			if width := text.Width(row); width > tc.width {
				t.Errorf("%s: row %q is %d cells wide in a %d-column pane", tc.name, row, width, tc.width)
			}
		}
	}

	// The plain grammar's own two templates are one template: a quoted post's head is a head.
	if !slices.Equal(Plain.QuoteHead, Plain.Head) {
		t.Errorf("the plain grammar's quote head is %q and its head is %q, want the same templates", Plain.QuoteHead, Plain.Head)
	}
}

// TestTheMeasureIsTheGrammarsOwn is the measure table: the five indent and prefix combinations this
// phase ships, at three pane widths and three quote depths, with the numbers spelled out. The plain
// grammar's row is Phase 2's min(W - 3 - 2·depth, 96) — which is the regression the stored frames also
// check — and the covers' rows are the same rule with the body's own prefix and indent in it.
//
// The arm is the layout itself: at 120 columns a body of one long token fills every grammar's measure
// exactly, so the numbers below are the ones the rows are laid out at and not a second opinion about
// them.
func TestTheMeasureIsTheGrammarsOwn(t *testing.T) {
	plainEnv(t)
	long := strings.Repeat("x", 400)
	tests := []struct {
		name    string
		grammar Grammar
		measure map[int][3]int // pane width -> the measure at depth 0, 1 and 2
	}{
		{
			"plain",
			Plain,
			map[int][3]int{60: {57, 55, 53}, 80: {77, 75, 73}, 120: {96, 96, 96}, 200: {96, 96, 96}},
		},
		{
			"servicelog",
			ServiceLog,
			map[int][3]int{60: {57, 55, 53}, 80: {77, 75, 73}, 120: {96, 96, 96}, 200: {96, 96, 96}},
		},
		{
			"gitlog",
			GitLog,
			map[int][3]int{60: {55, 53, 51}, 80: {75, 73, 71}, 120: {96, 96, 96}, 200: {96, 96, 96}},
		},
		{
			"ci",
			CI,
			map[int][3]int{60: {57, 55, 53}, 80: {77, 75, 73}, 120: {96, 96, 96}, 200: {96, 96, 96}},
		},
		{
			"internal",
			Internal,
			map[int][3]int{60: {57, 55, 53}, 80: {77, 75, 73}, 120: {96, 96, 96}, 200: {96, 96, 96}},
		},
	}
	for _, tc := range tests {
		for w, want := range tc.measure {
			for depth, cells := range want {
				if got := tc.grammar.measure(w, depth, 0); got != cells {
					t.Errorf("the %s measure at %d columns and depth %d is %d, want %d", tc.name, w, depth, got, cells)
				}
			}
		}

		// The layout's own arm: a 400-cell token wraps at the measure, and the widest row is the
		// selection cell, the body's prefix and indent, and the measure.
		rows := rowsOf(Blocks([]domain.Post{{
			ID: "1", Author: domain.User{ID: "u-lzd", ScreenName: "lzd"}, CreatedAt: specNow, Text: long,
		}}, tc.grammar, 120, Selection{ID: ""}, specNow, nil))
		filled := 0
		for _, row := range rows[1:] {
			if got := text.Width(row); got > 1+text.Width(tc.grammar.BodyPrefix)+tc.grammar.BodyIndent+tc.grammar.measure(120, 0, 0) {
				t.Errorf("the %s body row %q is %d cells wide, over the measure's own row", tc.name, row, got)
			}
			if text.Width(row) == 1+text.Width(tc.grammar.BodyPrefix)+tc.grammar.BodyIndent+tc.grammar.measure(120, 0, 0) {
				filled++
			}
		}
		if filled == 0 {
			t.Errorf("no body row of the %s grammar fills its measure at 120 columns: the measure is not seen to bind\n%q", tc.name, rows)
		}
	}
}

// TestTheMarkerPlaceIsWhereTheMarkersLand is the placement table: the same post, whose record carries a
// repost and a limited state and whose block is a continuation, drawn by three grammars that differ only
// in MarkerPlace. An inline grammar appends the run to the head's last line; a head-lines grammar gives
// each marker a line of its own at the head's indent; a body-lines grammar puts them where a body row
// sits, under the row they belong to.
func TestTheMarkerPlaceIsWhereTheMarkersLand(t *testing.T) {
	plainEnv(t)
	reposter := domain.User{ID: "u-b", ScreenName: "other"}
	mira := domain.User{ID: "u-mira", ScreenName: "mira"}
	posts := []domain.Post{
		{ID: "1", Author: mira, CreatedAt: specNow, ConversationID: "1", Text: "the parent"},
		{
			ID: "2", Author: mira, CreatedAt: specNow.Add(-time.Minute), ConversationID: "1", ReplyTo: "1",
			RepostedBy: &reposter, Visibility: domain.Limited, Text: "the reply",
		},
	}

	inline := placeholderGrammar("{handle:16} {stamp:12}{markers}")
	assertRows(t, "inline markers", rowsOf(Blocks(posts, inline, 80, Selection{ID: ""}, specNow, nil)), []string{
		" mira             Oct 14 09:12",
		"   the parent",
		" mira             Oct 14 09:11  reposted by other  limited  continued",
		"   the reply",
	})

	headLines := inline
	headLines.MarkerPlace = MarkersHeadLines
	assertRows(t, "head-line markers", rowsOf(Blocks(posts, headLines, 80, Selection{ID: ""}, specNow, nil)), []string{
		" mira             Oct 14 09:12",
		"   the parent",
		" mira             Oct 14 09:11",
		" reposted by other",
		" limited",
		" continued",
		"   the reply",
	})

	bodyLines := inline
	bodyLines.MarkerPlace = MarkersBodyLines
	bodyLines.BodyIndent = 2
	bodyLines.BodyPrefix = "| "
	assertRows(t, "body-line markers", rowsOf(Blocks(posts, bodyLines, 80, Selection{ID: ""}, specNow, nil)), []string{
		" mira             Oct 14 09:12",
		" |   the parent",
		" mira             Oct 14 09:11",
		" |   reposted by other",
		" |   limited",
		" |   continued",
		" |   the reply",
	})

	// The arm: the fixture really carries the three markers the three placements place, or a grammar
	// that dropped them all would pass every case above.
	if posts[1].RepostedBy == nil || posts[1].RepostedBy.ScreenName != "other" ||
		posts[1].Visibility != domain.Limited ||
		posts[1].ReplyTo != posts[0].ID || posts[1].Author != posts[0].Author {
		t.Fatalf("the fixture does not carry a repost by another account, a limited state and a same-author continuation: %+v", posts[1])
	}
}

// TestTheQuoteIsARecordOrANestedBlock is QuoteAsRecord's two shapes: with it, the quoted post is a
// record of its own at the same origin, carrying the back-reference to the record that quotes it; without
// it, a nested block or row behind the quote prefix, which is Phase 2's shape and the plain reader's.
//
// Either way the quoted post's text is on screen and in the same place in the list: the two renderings
// differ in the head, the indent and the marker, and in nothing that is true about the posts.
func TestTheQuoteIsARecordOrANestedBlock(t *testing.T) {
	plainEnv(t)
	post := func() domain.Post {
		return domain.Post{
			ID: "1", Author: domain.User{ID: "u-lzd", ScreenName: "lzd"}, CreatedAt: specNow,
			Text: "the outer body",
			Quote: &domain.Post{
				ID: "2", Author: domain.User{ID: "u-mira", ScreenName: "mira"},
				CreatedAt: specNow.Add(-time.Minute), Text: "the quoted body",
			},
		}
	}

	nested := placeholderGrammar("{handle:16} {stamp:12}{markers}")
	nested.QuoteHead = []string{"{handle:16} {stamp:12}"}
	assertRows(t, "a nested quote", rowsOf(Blocks([]domain.Post{post()}, nested, 80, Selection{ID: ""}, specNow, nil)), []string{
		" lzd              Oct 14 09:12",
		"   the outer body",
		" > mira             Oct 14 09:11",
		" >   the quoted body",
	})

	record := nested
	record.QuoteAsRecord = true
	record.Head = []string{"{handle:16} {stamp:12}{markers}"}
	record.Markers.Parent = " parent={ref}"
	record.Ref = func(id domain.PostID) string { return "ref-" + string(id) }
	assertRows(t, "a quoted record", rowsOf(Blocks([]domain.Post{post()}, record, 80, Selection{ID: ""}, specNow, nil)), []string{
		" lzd              Oct 14 09:12",
		"   the outer body",
		" mira             Oct 14 09:11 parent=ref-1",
		"   the quoted body",
	})

	// The arm: the quoted post's text is in both renderings and neither rendering reordered anything.
	for _, rows := range [][]string{
		rowsOf(Blocks([]domain.Post{post()}, nested, 80, Selection{ID: ""}, specNow, nil)),
		rowsOf(Blocks([]domain.Post{post()}, record, 80, Selection{ID: ""}, specNow, nil)),
	} {
		joined := strings.Join(rows, "\n")
		if !strings.Contains(joined, "the outer body") || !strings.Contains(joined, "the quoted body") {
			t.Fatalf("a quote rendering lost a post's text:\n%s", joined)
		}
	}
}

// TestThePaneHeaderIsAContentRow: a grammar's pane header lines are drawn once as the pane's first
// content rows, they scroll with the content — they are a block of the list and not a row the frame
// paints over it — and they never carry the selection mark, however the selection is set.
func TestThePaneHeaderIsAContentRow(t *testing.T) {
	plainEnv(t)
	post := domain.Post{
		ID: "1", Author: domain.User{ID: "u-lzd", ScreenName: "lzd"}, CreatedAt: specNow, Text: "a body",
	}
	blocks := Blocks([]domain.Post{post}, Internal, 80, Selection{ID: post.ID}, specNow, nil)
	if len(blocks) < 2 {
		t.Fatalf("the internal grammar rendered %d blocks, want a pane header and a post", len(blocks))
	}
	if got, want := blocks[0].Rows[0], " "+CoverInternalPaneHeader; got != want {
		t.Fatalf("the pane header row is %q, want %q", got, want)
	}
	if blocks[0].PostID != "" {
		t.Fatalf("the pane header claims to be the post %q", blocks[0].PostID)
	}
	for _, row := range blocks[0].Rows {
		if strings.HasPrefix(row, MarkSelection) {
			t.Fatalf("the pane header carries the selection mark: %q", row)
		}
	}
	if !strings.HasPrefix(blocks[1].Rows[0], MarkSelection) {
		t.Fatalf("the selected post's head is not marked: %q", blocks[1].Rows[0])
	}
	// It scrolls: it is the first block, so it is what the shell's window drops first, and the grammar
	// with no pane header renders nothing above its first post.
	if got := Blocks([]domain.Post{post}, Plain, 80, Selection{ID: post.ID}, specNow, nil)[0].Rows[0]; strings.Contains(got, "state") {
		t.Fatalf("the plain grammar rendered a pane header: %q", got)
	}
}
