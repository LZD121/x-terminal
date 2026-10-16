package render

// This file is the covers' acceptance: the UI-SPEC's eight worked renderings, byte for byte, plus the
// claim that makes them more than eight strings — that the four styles are four Grammar values handed
// to the one composer, and that the identity on screen is an alias and never an account.
//
// The eight renderings are transcribed from `## The Four Covers` of the phase's UI-SPEC, right-trimmed
// the way the document shows them (the frame pads every row to the pane's width, which is geometry
// and not text) and with the two plain renderings of the same section beside them, so that five
// grammars are compared against one another on one page.

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// coverNow is the instant the fixture's posts carry: the UI-SPEC's worked renderings are stamped with
// 09:12:00Z and 09:00:00Z. The clock is injected, so the bytes below are a fact about the styles and
// not about the day the test is run.
var coverNow = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

// The fixture's two accounts, with the ids the UI-SPEC's alias table uses. Each carries a display name
// as well, because the absence that matters is the absence of a name as well as of a handle.
var (
	coverAuthor = domain.User{ID: "44196397", ScreenName: "aergon", Name: "Ergon Rios"}
	coverQuoted = domain.User{ID: "14706153", ScreenName: "lzd", Name: "Lea Zdanov"}
)

// coverFixture is the UI-SPEC's fixture pair as content: the post that quotes, and the post it quotes.
// One domain.Post, two records — every difference between the renderings below is a presentation field
// of this same value.
func coverFixture() []domain.Post {
	return []domain.Post{{
		ID: domain.PostID(coverAuthor.ID), Author: coverAuthor, CreatedAt: coverNow,
		ConversationID: coverAuthor.ID,
		Text: "cursor in, cursor out: the render layer never does arithmetic on a cursor. if the " +
			"upstream hands back one you have seen, that is the end of the walk.",
		Quote: &domain.Post{
			ID: domain.PostID(coverQuoted.ID), Author: coverQuoted,
			CreatedAt: coverNow.Add(-12 * time.Minute), ConversationID: coverAuthor.ID,
			Text: "the page cache is keyed by the cursor, not by an index.",
		},
	}}
}

// coverEnv pins the environment a cover's literal comparison runs under: the plain reader's rung, so
// that the comparison is on characters and not on escapes, and the zone the fixture's injected clock is
// spelled in.
func coverEnv(t *testing.T) {
	t.Helper()
	plainEnv(t)
	t.Setenv("TZ", "UTC")
}

// coverRows is one grammar's rendering of a list, as the UI-SPEC's own blocks show it: every row
// right-trimmed, unselected (the mark is the empty cell the document draws), and with the escapes
// stripped, because the environment under test paints none.
func coverRows(grammar Grammar, posts []domain.Post, w int) []string {
	rows := rowsOf(Blocks(posts, grammar, w, Selection{ID: ""}, coverNow, nil))
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = strings.TrimRight(ansi.Strip(row), " ")
	}
	return out
}

// TestCoverRenderings is the phase's acceptance at the unit level: the four styles' renderings of the
// fixture at 80 and 120 columns, and the plain reader's own two, transcribed byte for byte.
//
// Every row is the document's own. A style that moved a column, a marker, an indent, a stamp or an
// alias fails here with the row and its width, and the width is measured by internal/text, which is
// the only measure the renderer has.
func TestCoverRenderings(t *testing.T) {
	coverEnv(t)
	tests := []struct {
		name    string
		grammar Grammar
		w       int
		want    []string
	}{
		{"plain at 80", Plain, 80, []string{
			" aergon           Oct 14 09:12",
			"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if",
			"   the upstream hands back one you have seen, that is the end of the walk.",
			" > lzd              Oct 14 09:00",
			" >   the page cache is keyed by the cursor, not by an index.",
		}},
		{"plain at 120", Plain, 120, []string{
			" aergon           Oct 14 09:12",
			"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands",
			"   back one you have seen, that is the end of the walk.",
			" > lzd              Oct 14 09:00",
			" >   the page cache is keyed by the cursor, not by an index.",
		}},
		{"servicelog at 80", ServiceLog, 80, []string{
			" 2026-10-14T09:12:00Z INFO  svc=agent-rollback-70 seq=2805966",
			"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if",
			"   the upstream hands back one you have seen, that is the end of the walk.",
			" 2026-10-14T09:00:00Z INFO  svc=guard-migrate-48 seq=2148784 parent=2805966",
			"   the page cache is keyed by the cursor, not by an index.",
		}},
		{"servicelog at 120", ServiceLog, 120, []string{
			" 2026-10-14T09:12:00Z INFO  svc=agent-rollback-70 seq=2805966",
			"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands",
			"   back one you have seen, that is the end of the walk.",
			" 2026-10-14T09:00:00Z INFO  svc=guard-migrate-48 seq=2148784 parent=2805966",
			"   the page cache is keyed by the cursor, not by an index.",
		}},
		{"gitlog at 80", GitLog, 80, []string{
			" commit 51140f214ccc",
			" Author: Anya Karlsson <anya.karlsson70@corp.internal>",
			" Date:   Wed Oct 14 09:12:00 2026 +0000",
			"",
			"     cursor in, cursor out: the render layer never does arithmetic on a cursor.",
			"     if the upstream hands back one you have seen, that is the end of the walk.",
			"     > Clara Sorensen  Wed Oct 14 09:00:00 2026 +0000",
			"     >   the page cache is keyed by the cursor, not by an index.",
		}},
		{"gitlog at 120", GitLog, 120, []string{
			" commit 51140f214ccc",
			" Author: Anya Karlsson <anya.karlsson70@corp.internal>",
			" Date:   Wed Oct 14 09:12:00 2026 +0000",
			"",
			"     cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands",
			"     back one you have seen, that is the end of the walk.",
			"     > Clara Sorensen  Wed Oct 14 09:00:00 2026 +0000",
			"     >   the page cache is keyed by the cursor, not by an index.",
		}},
		{"ci at 80", CI, 80, []string{
			" [09:12:00] build=4655 job=agent-rollback-70",
			" | cursor in, cursor out: the render layer never does arithmetic on a cursor. if",
			" | the upstream hands back one you have seen, that is the end of the walk.",
			" | > [09:00:00] build=9457 job=guard-migrate-48",
			" | >   the page cache is keyed by the cursor, not by an index.",
		}},
		{"ci at 120", CI, 120, []string{
			" [09:12:00] build=4655 job=agent-rollback-70",
			" | cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands",
			" | back one you have seen, that is the end of the walk.",
			" | > [09:00:00] build=9457 job=guard-migrate-48",
			" | >   the page cache is keyed by the cursor, not by an index.",
		}},
		{"internal at 80", Internal, 80, []string{
			" id        state    owner                 updated",
			" REQ-4655  open     anya.karlsson         Oct 14 09:12",
			"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if",
			"   the upstream hands back one you have seen, that is the end of the walk.",
			" > REQ-9457  open     clara.sorensen        Oct 14 09:00",
			" >   the page cache is keyed by the cursor, not by an index.",
		}},
		{"internal at 120", Internal, 120, []string{
			" id        state    owner                 updated",
			" REQ-4655  open     anya.karlsson         Oct 14 09:12",
			"   cursor in, cursor out: the render layer never does arithmetic on a cursor. if the upstream hands",
			"   back one you have seen, that is the end of the walk.",
			" > REQ-9457  open     clara.sorensen        Oct 14 09:00",
			" >   the page cache is keyed by the cursor, not by an index.",
		}},
	}
	for _, tc := range tests {
		got := coverRows(tc.grammar, coverFixture(), tc.w)
		if len(got) != len(tc.want) {
			t.Errorf("%s: %d rows, want %d\n got: %q\nwant: %q", tc.name, len(got), len(tc.want), got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: row %d differs\n got (%d cells): %q\nwant (%d cells): %q",
					tc.name, i+1, text.Width(got[i]), got[i], text.Width(tc.want[i]), tc.want[i])
			}
		}
	}
}

// TestCoverGrammarsAreValues is the phase's central structural claim, asserted rather than described:
// a style is a Grammar value and the composer is one function over it.
//
// Four arms. The rendered bytes do not depend on the style's Name, so nothing in the composer reads a
// name — a style is its fields. Every style's rendering differs from the plain reader's, so a cover
// that changed nothing would be caught. No two styles render the same bytes, so four styles really are
// four styles. And the composer asks the grammar for exactly the accounts the content carries — each
// post's own author and nothing else — so an identity function is never handed an account the frame
// has no business knowing.
func TestCoverGrammarsAreValues(t *testing.T) {
	coverEnv(t)
	plain := strings.Join(coverRows(Plain, coverFixture(), 80), "\n")
	seen := map[string]bool{}
	for _, style := range CoverStyles() {
		grammar := style.Grammar()
		rendering := strings.Join(coverRows(grammar, coverFixture(), 80), "\n")

		renamed := grammar
		renamed.Name = "not-" + string(style)
		if got := strings.Join(coverRows(renamed, coverFixture(), 80), "\n"); got != rendering {
			t.Errorf("the %s style renders differently under another name: the composer reads a name, and a style is its fields", style)
		}
		if rendering == plain {
			t.Errorf("the %s style renders the plain reader's bytes", style)
		}
		if seen[rendering] {
			t.Errorf("the %s style renders the same bytes as another style: the four styles are not four", style)
		}
		seen[rendering] = true

		accounts := distinctAccounts(identityAccounts(t, grammar))
		want := []domain.User{coverAuthor, coverQuoted}
		if len(accounts) != len(want) {
			t.Fatalf("the %s style's identity function was asked about %d accounts (%v), want the %d the content carries",
				style, len(accounts), accounts, len(want))
		}
		for i, account := range accounts {
			if account != want[i] {
				t.Errorf("the %s style's identity function was asked about %v, want the post's own author %v", style, account, want[i])
			}
		}
	}

	// The text, the order and the quote relationship are the same in every rendering: every word of
	// each post's body is on screen, in order, in every grammar — a wrap, a gutter, an indent and a
	// marker may sit between two words and no word may be lost, repeated into another post's place or
	// moved — and the quoted post's text stands behind the text of the post that quotes it.
	outer, quoted := coverFixture()[0], coverFixture()[0].Quote
	for _, grammar := range append([]Grammar{Plain}, coverGrammars()...) {
		rows := coverRows(grammar, coverFixture(), 80)
		outerRows := wordRows(t, grammar, rows, outer.Text, 0)
		// The quoted post's text is walked starting behind the quoting post's last word: if the grammar
		// had put the quoted record first, its words would not be found there at all.
		wordRows(t, grammar, rows, quoted.Text, outerRows[len(outerRows)-1]+1)
	}
}

// wordRows walks a text's words through a rendering's rows and returns the row each word was found
// in, in order: the words must appear in the order the post has them, each at or after the row the
// word before it was found in, every one of them must be there, and the walk starts at row from. It is
// the layout-agnostic half of "the text and the order are untouched": a wrap point, a gutter, an
// indent and a marker all sit between two words, and none of them may swallow one or move it into
// another record.
func wordRows(t *testing.T, grammar Grammar, rows []string, text string, from int) []int {
	t.Helper()
	found := make([]int, 0, len(strings.Fields(text)))
	at := from
	for _, word := range strings.Fields(text) {
		row := -1
		for i := at; i < len(rows); i++ {
			if strings.Contains(rows[i], word) {
				row = i
				break
			}
		}
		if row < 0 {
			t.Fatalf("the %s grammar does not carry the word %q of a post's text at or after row %d\n%q",
				grammar.Name, word, at+1, rows)
		}
		found = append(found, row)
		at = row
	}
	if len(found) == 0 {
		t.Fatalf("the text under test (%q) has no words, so the walk above proves nothing", text)
	}
	return found
}

// distinctAccounts is the accounts identityAccounts saw, in the order they were first asked about: a
// head that spells an identity in two fields of one line — git's `{name} <{email}>` — asks about one
// account twice, and what a test cares about is which accounts the composer knows and in what order.
func distinctAccounts(seen []domain.User) []domain.User {
	out := make([]domain.User, 0, len(seen))
	for _, account := range seen {
		known := false
		for _, have := range out {
			if have == account {
				known = true
				break
			}
		}
		if !known {
			out = append(out, account)
		}
	}
	return out
}

// identityAccounts runs one rendering with the grammar's identity function wrapped, and returns the
// accounts the composer asked it about, in the order it asked.
func identityAccounts(t *testing.T, grammar Grammar) []domain.User {
	t.Helper()
	var seen []domain.User
	inner := grammar.Identity
	grammar.Identity = func(u domain.User) CoverAlias {
		seen = append(seen, u)
		return inner(u)
	}
	if rows := rowsOf(Blocks(coverFixture(), grammar, 80, Selection{ID: ""}, coverNow, nil)); len(rows) == 0 {
		t.Fatal("the wrapped grammar rendered nothing: the call count below would be vacuous")
	}
	return seen
}

// coverGrammars is the four styles' grammars, in the frozen order CoverStyles returns them.
func coverGrammars() []Grammar {
	grammars := make([]Grammar, 0, len(CoverStyles()))
	for _, style := range CoverStyles() {
		grammars = append(grammars, style.Grammar())
	}
	return grammars
}

// TestCoveredRenderingsHideTheIdentity is DSGT-06 at the unit level, and it scans the rendered bytes
// rather than trusting the mapping: for each of the eight renderings, no handle and no display name the
// content was built from appears in it, and the alias the UI-SPEC's worktable names does. The second
// half is the arm — an empty or truncated rendering cannot pass by having nothing in it.
func TestCoveredRenderingsHideTheIdentity(t *testing.T) {
	coverEnv(t)
	aliases := map[CoverStyle][]string{
		CoverServiceLog: {"agent-rollback-70", "guard-migrate-48"},
		CoverGitLog: {
			"Anya Karlsson", "anya.karlsson70@corp.internal",
			"Clara Sorensen",
		},
		CoverCI:       {"agent-rollback-70", "guard-migrate-48"},
		CoverInternal: {"anya.karlsson", "clara.sorensen"},
	}
	secrets := []string{coverAuthor.ScreenName, coverAuthor.Name, coverQuoted.ScreenName, coverQuoted.Name}
	for _, secret := range secrets {
		if secret == "" {
			t.Fatal("the fixture stopped carrying a handle or a display name: the absence below would be vacuous")
		}
	}
	for _, style := range CoverStyles() {
		for _, w := range []int{80, 120} {
			rendering := strings.Join(coverRows(style.Grammar(), coverFixture(), w), "\n")
			for _, secret := range secrets {
				if strings.Contains(rendering, secret) {
					t.Errorf("the %s rendering at %d columns holds the account's own %q:\n%s", style, w, secret, rendering)
				}
			}
			for _, alias := range aliases[style] {
				if !strings.Contains(rendering, alias) {
					t.Errorf("the %s rendering at %d columns does not hold the alias %q:\n%s", style, w, alias, rendering)
				}
			}
		}
	}

	// The same scan over the plain reader, as the arm that says what the covers must differ from: the
	// plain grammar prints the handle, and it is the rendering the covers are the absence of.
	if plain := strings.Join(coverRows(Plain, coverFixture(), 80), "\n"); !strings.Contains(plain, coverAuthor.ScreenName) {
		t.Fatalf("the plain reader no longer prints the handle, so the absence above proves nothing:\n%s", plain)
	}
}

// TestNoTabByteUnderAnyGrammar is the grid's "no TAB byte is ever emitted", over every grammar this
// phase ships and over text that carries one. A cover's aligned columns are spaces, because a tab's
// width is the terminal's state and not the frame's text: a tab that reached a row would move every
// column behind it by an amount the frame cannot compute, which is the same defect as a second width
// model.
//
// The post whose text carries a tab is what keeps the assertion from being vacuous: text.Wrap collapses
// whitespace, so no row of any grammar holds the byte. The fixture's own posts are rendered beside it,
// so a cover's literals are scanned too.
func TestNoTabByteUnderAnyGrammar(t *testing.T) {
	coverEnv(t)
	tabbed := domain.Post{
		ID: "tab", Author: domain.User{ID: "9", ScreenName: "tabber"},
		CreatedAt: coverNow, Text: "one\ttwo\tthree, and a body that wraps somewhere",
	}
	quoted := coverFixture()
	quoted[0].Quote.Text = "quoted\ttext"
	posts := append(quoted, tabbed)
	for _, grammar := range append([]Grammar{Plain}, coverGrammars()...) {
		for _, w := range []int{60, 80, 120, 200} {
			for _, row := range rowsOf(Blocks(posts, grammar, w, Selection{ID: ""}, coverNow, nil)) {
				if strings.Contains(row, "\t") {
					t.Errorf("the %s grammar emitted a tab byte at %d columns: %q", grammar.Name, w, row)
				}
			}
		}
	}
	// The stored corpus as well: the plain reader's frames are the ones a terminal has already been
	// asked to draw, and a tab byte in one of them is the defect this rule exists for, whether it
	// arrived from a template or from the content.
	frames := storedFrames(t)
	for _, frame := range frames {
		if strings.Contains(frame.bytes, "\t") {
			t.Errorf("the stored frame %s carries a tab byte", frame.name)
		}
	}
	if len(frames) == 0 {
		t.Fatal("the stored corpus is empty, so the scan above has nothing to read")
	}

	// The arm: the byte under test is really in the content, or the scan above has nothing to find.
	if !strings.Contains(tabbed.Text, "\t") || !strings.Contains(quoted[0].Quote.Text, "\t") {
		t.Fatal("the fixture carries no tab byte, so the absence above is vacuous")
	}
}

// TestTheGitDateDivergesFromTheUISpecsWeekday holds the one place where the UI-SPEC's worked gitlog
// rendering cannot be the output of the format the same document specifies, with the numbers: the
// block is stamped `Mon Oct 14 09:12:00 2026 +0000`, and 2026-10-14 is a Wednesday. `Mon Jan 2
// 15:04:05 2006 -0700` is git's own default date format and the format is what the style is pinned to;
// the weekday in the fenced block is the format's example text and not the instant's.
//
// The engine's own rendering of that instant is asserted in TestCoverRenderings as `Wed`, which is what
// makes this a recorded divergence and not an unnoticed one: the format string here and the rendered
// weekday there are the two halves of the same claim.
func TestTheGitDateDivergesFromTheUISpecsWeekday(t *testing.T) {
	coverEnv(t)
	if got := coverGrammars()[1].Stamp.Clock; got != "Mon Jan 2 15:04:05 2006 -0700" {
		t.Fatalf("the gitlog stamp format is %q, want git's own default", got)
	}
	if weekday := coverNow.Weekday().String(); weekday != "Wednesday" {
		t.Fatalf("the fixture's instant falls on a %s, so the divergence this test records is stale", weekday)
	}
	fenced := " Date:   Mon Oct 14 09:12:00 2026 +0000"
	rendered := coverRows(GitLog, coverFixture(), 80)[2]
	if rendered == fenced {
		t.Fatal("the gitlog date now reads the UI-SPEC's own weekday: this test's premise is stale")
	}
	if !strings.HasSuffix(fenced, "14 09:12:00 2026 +0000") || !strings.HasSuffix(rendered, "14 09:12:00 2026 +0000") {
		t.Fatalf("the two date lines differ by more than the weekday:\n%s\n%s", fenced, rendered)
	}
	if !strings.HasPrefix(rendered, " Date:   Wed") {
		t.Fatalf("the gitlog date renders %q, want the instant's own weekday behind git's own format", rendered)
	}
}

// TestTheStyleVocabulary is the four styles as a vocabulary rather than as four grammar values: their
// ids are the ones `display.cover` carries and the cycle key steps through, CoverStyles returns them
// in the frozen order, CoverStyleOf resolves each one and refuses everything else — including the
// empty string, which is the plain reader and not a cover, and an id that differs only in case — and
// every state names what is on screen: the style's own title under a cover, `x-terminal` for the
// plain reader and `make` for the panic frame, so no title ever lies about its frame.
func TestTheStyleVocabulary(t *testing.T) {
	coverEnv(t)
	frozen := []struct {
		style CoverStyle
		id    string
		title string
	}{
		{CoverServiceLog, "servicelog", "svc-log"},
		{CoverGitLog, "gitlog", "git"},
		{CoverCI, "ci", "pipeline"},
		{CoverInternal, "internal", "ops-console"},
	}

	styles := CoverStyles()
	if len(styles) != len(frozen) {
		t.Fatalf("CoverStyles returns %d styles, want %d", len(styles), len(frozen))
	}
	for i, want := range frozen {
		if styles[i] != want.style {
			t.Errorf("CoverStyles()[%d] is %q, want %q: the order is the cycle key's", i, styles[i], want.style)
		}
		if string(want.style) != want.id {
			t.Errorf("the style %q carries the id %q, want %q", want.title, want.style, want.id)
		}
		if got, ok := CoverStyleOf(want.id); !ok || got != want.style {
			t.Errorf("CoverStyleOf(%q) is (%q, %v), want (%q, true)", want.id, got, ok, want.style)
		}
		if got := want.style.Title(); got != want.title {
			t.Errorf("the %q style's title is %q, want %q", want.style, got, want.title)
		}
		if got := want.style.Grammar().Name; got != want.id {
			t.Errorf("the %q style's grammar is named %q, want the id", want.style, got)
		}
	}

	// An id that is not a style is "not a style": the caller keeps the default it had rather than
	// guessing at a cover it has never heard of.
	for _, id := range []string{"", "plain", "svc-log", "servicelogs", "SERVICELOG", "SerViceLog", "x-terminal"} {
		if style, ok := CoverStyleOf(id); ok {
			t.Errorf("CoverStyleOf(%q) resolved to the %q style: an unknown id is not a style", id, style)
		}
	}

	// The two states that are not a cover name their own thing, and no covered title names the
	// product.
	if CoverStyle("").Title() != CopyTitlePlain || CopyTitlePlain != "x-terminal" {
		t.Errorf("the plain reader's title is %q / %q, want x-terminal", CoverStyle("").Title(), CopyTitlePlain)
	}
	if CopyTitlePanic != "make" {
		t.Errorf("the panic frame's title is %q, want the name of the command its output comes from", CopyTitlePanic)
	}
	if plain := coverRows(CoverStyle("").Grammar(), coverFixture(), 80); plain[0] != coverRows(Plain, coverFixture(), 80)[0] {
		t.Errorf("the zero style renders %q where the plain grammar renders %q: the zero value is the plain reader", plain[0], coverRows(Plain, coverFixture(), 80)[0])
	}
	for _, want := range frozen {
		if title := want.style.Title(); strings.Contains(title, "x-terminal") || title == CopyTitlePlain {
			t.Errorf("the %q style's title names the product: %q", want.style, title)
		}
		if want.style.Grammar().Name != string(want.style) {
			t.Errorf("the %q style's grammar is not the style it is worn as", want.style)
		}
	}
}

// TestGatedMarkersNonEmpty is REND-07 / DSGT-10 at the grammar: every cover
// grammar carries a non-empty refusal notice, the plain grammar's stays empty
// (unreachable in normal), and no literal names the product, a purpose, a
// handle, a program, a URL or the way out of the cover.
func TestGatedMarkersNonEmpty(t *testing.T) {
	if Plain.Markers.Gated != "" || PlainMarkers.Gated != "" {
		t.Fatalf("plain Gated is %q, want empty: the notice is unreachable in normal", Plain.Markers.Gated)
	}
	wants := map[string]string{
		ServiceLog.Name: CoverServiceLogGated,
		GitLog.Name:     CoverGitGated,
		CI.Name:         CoverCIGated,
		Internal.Name:   CoverInternalGated,
	}
	for _, g := range []Grammar{ServiceLog, GitLog, CI, Internal} {
		want, ok := wants[g.Name]
		if !ok {
			t.Fatalf("no expected Gated for grammar %q", g.Name)
		}
		if g.Markers.Gated == "" {
			t.Errorf("grammar %q has an empty Gated literal", g.Name)
		}
		if g.Markers.Gated != want {
			t.Errorf("grammar %q Gated is %q, want %q", g.Name, g.Markers.Gated, want)
		}
		lower := strings.ToLower(g.Markers.Gated)
		for _, word := range []string{
			"x.com", "twitter", "x-terminal", "browser", "attachment", "link", "media",
			"clipboard", "wl-copy", "xdg-open", "mpv", "handle", "cover", "immersive",
			"panic", "leave", "exit", "normal", "press m", "http", "post", "feed", "timeline",
		} {
			if strings.Contains(lower, word) {
				t.Errorf("grammar %q Gated %q holds forbidden %q", g.Name, g.Markers.Gated, word)
			}
		}
	}
}
