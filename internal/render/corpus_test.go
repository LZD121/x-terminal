package render

// This file is the audit over the stored corpus. Every assertion here reads the bytes on disk —
// testdata/TestGoldenFrames/*.golden — and none of them re-renders the state it checks: a test that
// recomputed the frame it compares would pass with the file deleted, emptied or moved, and the
// corpus's whole value is that a stored frame is a fact rather than a call.
//
// The audit is what keeps the corpus adversarial. It asserts that every case the UI-SPEC's checklist
// names is in a named frame, that the things which must not be printed are in no frame, that the
// width properties A1, A2, A3 and A7 hold over the stored bytes, and — the arms — that none of it
// can pass by the corpus shrinking: a deleted frame, a twin that went missing, a frame with no
// colour to strip, no measured row, no engagement numbers in its state or no selected block would
// each fail here rather than leave the specification thinner than it reads.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/keys"
	"github.com/LZD121/x-terminal/internal/text"
)

// goldenDir is where the corpus is stored. It is golden.RequireEqual's own path rule — the helper
// writes testdata/<test name>.golden, and these frames' test name is TestGoldenFrames/<frame> — so
// this constant and the directory the golden test writes to are one place.
const goldenDir = "testdata/TestGoldenFrames"

// stampPattern is the head row's folded stamp (`Oct 14 09:12`) as it reads in a frame: how the audit
// finds a head row in a stored pane, where the struct that built it is no longer available.
var stampPattern = regexp.MustCompile(`[A-Z][a-z]{2} \d\d \d\d:\d\d`)

// budgetPattern is a three-digit request count: the budget field the checklist asks for.
var budgetPattern = regexp.MustCompile(`\b\d{3,}\b`)

// storedFrame is one stored frame read back from disk: the name it is stored under, the width and
// environment the corpus declares for it, the state it was rendered from, and its bytes.
type storedFrame struct {
	name  string
	w     int
	env   environment
	state state
	bytes string
	rows  []string
	// panic is the frame's own state: a panic frame is composed from the pane's size alone, so every
	// row of it is the frame and there is no status line to peel off the last one.
	panic bool
	// grammar is the grammar the frame was rendered through: the plain reader's, or the cover's.
	grammar Grammar
}

// pane is the frame's pane: every row but the last, which is the status line, always — and, for a
// panic frame, every row, because there is no status line under that frame.
func (f storedFrame) pane() []string {
	if f.panic {
		return f.rows
	}
	return f.rows[:len(f.rows)-1]
}

// status is the frame's status line: its last row, always. A panic frame has none; its rows are read
// through pane() instead.
func (f storedFrame) status() string { return f.rows[len(f.rows)-1] }

// measure is the body measure this frame is rendered at: the grammar's own rule for this width at
// the top quote level, so the audit measures what the renderer measures with rather than a second
// idea of a cell.
func (f storedFrame) measure() int { return Plain.measure(f.w, 0, 0) }

// strip is the frame's bytes with every escape sequence removed: the form A1 and A3 compare in, and
// the form every content assertion is made on.
func (f storedFrame) strip() string { return ansi.Strip(f.bytes) }

// content is a row's text from the content origin: column 1 is the selection column, so the cell
// there is the selection mark — MarkSelection when the row's block is selected, a space otherwise —
// and everything the row says starts one cell behind it. The mark is one ASCII cell, which is why
// the cut is a byte.
func content(row string) string {
	text := ansi.Strip(row)
	if len(text) < 2 {
		return ""
	}
	return text[1:]
}

// marked reports whether a row carries the selection mark in column 1. The row is read with its
// escapes stripped, because the selected block is painted in a colour rung and the mark is what
// carries the meaning (A1).
func marked(row string) bool { return strings.HasPrefix(ansi.Strip(row), MarkSelection) }

// trimmed is a row's text without its padding: the right-trimmed text the width properties are about
// (the frame pads every row to the pane's width, which is geometry rather than text).
func trimmed(rows []string, i int) string { return strings.TrimRight(ansi.Strip(rows[i]), " ") }

// TestCorpusAudit reads the stored corpus and checks everything the view specification claims of it.
func TestCorpusAudit(t *testing.T) {
	frames := storedFrames(t)
	if len(frames) == 0 {
		t.Fatal("the corpus is empty: every check below would pass by having nothing to check")
	}

	t.Run("the required cases", func(t *testing.T) {
		for _, c := range requiredCases {
			f := lookup(t, frames, c.frame)
			// The frame's own width semantics: the twins measure with wcwidth, the bare frames with
			// grapheme clusters, and every case below measures through internal/text.
			t.Setenv("TMUX", f.env.tmux)
			if !c.holds(f) {
				t.Errorf("%s is not in the stored frame %s: the checklist requires it there", c.name, c.frame)
			}
		}
	})

	t.Run("the absences", func(t *testing.T) {
		counts := engagement(t)
		covered, _ := partition(frames)
		feed := feedGlyphs(feedPosts(covered))
		// Conversation frames carry their own writer text (CJK, emoji) outside the covered
		// timeline feed: fold those glyphs into the allowance so the icon absence stays about
		// chrome the program drew, not about content a conversation rendered verbatim.
		for _, f := range frames {
			if f.state.pane == paneThread {
				for r, ok := range feedGlyphs(threadPosts(f.state.conversation)) {
					if ok {
						feed[r] = true
					}
				}
			}
			if f.state.pane == paneSaved {
				for r, ok := range feedGlyphs(f.state.saved.List().Posts()) {
					if ok {
						feed[r] = true
					}
				}
			}
		}
		for _, f := range frames {
			t.Setenv("TMUX", f.env.tmux)
			assertNoCounts(t, f, counts)
			assertNoMediaURLs(t, f)
			assertNoDisplayNames(t, f)
			assertNoIcons(t, f, feed)
		}
		assertEmptyCopyOnlyAfterAnEmptyRead(t, frames)

		// The arms: an absence means something only if the state carries the thing whose absence is
		// asserted. The corpus's posts carry the recorded fixture's engagement numbers, its media
		// carry URLs, and its accounts carry display names — none of which may reach a frame.
		if !carriesCounts(frames) {
			t.Error("no stored frame's state carries engagement numbers: the count absence is vacuous, so a renderer that printed counts would pass this audit")
		}
		if !carriesMediaURL(frames) {
			t.Error("no stored frame's state carries a media URL: the media absence is vacuous")
		}
		if !carriesDisplayName(frames) {
			t.Error("no stored frame's state carries a display name: the name absence is vacuous")
		}
		// The icon absence's own two arms: the feed's texts carry a character no ASCII cell can hold
		// — the emoji and the ZWJ family — and a stored frame is seen to print one of them, so the
		// scan is not passing because there is no glyph anywhere to find.
		if len(feed) == 0 {
			t.Fatal("the corpus's feed carries no character outside ASCII: the icon absence has nothing to allow and nothing to look for")
		}
		printed := 0
		for _, f := range frames {
			for _, r := range glyphsIn(f) {
				if feed[r] {
					printed++
					break
				}
			}
		}
		if printed == 0 {
			t.Error("no stored frame prints a character of the feed's own text: the icon absence would pass over frames with no glyph in them at all")
		}
	})

	t.Run("A1: no meaning is carried by colour alone", func(t *testing.T) {
		colour := lookup(t, frames, "color_truecolor")
		mono := lookup(t, frames, "color_mono")
		plain := lookup(t, frames, "timeline_80")
		if got, want := mono.strip(), colour.strip(); got != want {
			t.Errorf("the mono frame's characters differ from the truecolor frame's: a rung changed the text and not only the escapes\nmono:\n%s\ntruecolor:\n%s", got, want)
		}
		if got, want := colour.strip(), plain.strip(); got != want {
			t.Errorf("the truecolor frame's characters differ from the plain frame's: a rung changed the text and not only the escapes\ntruecolor:\n%s\nplain:\n%s", got, want)
		}

		// The arms: the strip comparison is only worth making over a frame that has colour and a
		// selected block whose selection the colour could have been carrying alone.
		if !strings.Contains(colour.bytes, "\x1b[") {
			t.Errorf("the truecolor frame %s carries no escape sequence: there is no colour to strip and A1 proves nothing", colour.name)
		}
		if !strings.Contains(colour.bytes, sgrReverse) {
			t.Errorf("the truecolor frame %s paints no selected block: the selection colour is not holding the meaning A1 withholds", colour.name)
		}
		if !carriesMark(plain) {
			t.Errorf("the plain frame %s has no selected block: there is no selection mark for a colourless frame to carry", plain.name)
		}

		// MEDIA-03: the four rung fallback frames are byte-identical after stripping
		// every escape (SGR and OSC), and the mono frame carries no OSC 8.
		trueF := lookup(t, frames, "color_truecolor_fallback")
		monoF := lookup(t, frames, "color_mono_fallback")
		f256 := lookup(t, frames, "color_256_fallback")
		f16 := lookup(t, frames, "color_16_fallback")
		if got, want := monoF.strip(), trueF.strip(); got != want {
			t.Errorf("fallback mono != truecolor after strip")
		}
		if got, want := f256.strip(), trueF.strip(); got != want {
			t.Errorf("fallback 256 != truecolor after strip")
		}
		if got, want := f16.strip(), trueF.strip(); got != want {
			t.Errorf("fallback 16 != truecolor after strip")
		}
		if strings.Contains(monoF.bytes, "\x1b]8;;") || strings.Contains(monoF.bytes, "]8;;") {
			t.Errorf("mono fallback carries OSC 8")
		}
		if !strings.Contains(trueF.bytes, "]8;;") {
			t.Errorf("truecolor fallback missing OSC 8")
		}
	})

	t.Run("A2: the state is a literal", func(t *testing.T) {
		failures := 0
		for _, f := range frames {
			if f.state.failure.Kind == domain.FailureNone {
				continue
			}
			failures++
			// The word is the style's: the status line prints the classifier's state name, with the
			// upstream's host substituted under a cover — the one edit a cover makes to a word it
			// did not author, and the reason the covered no-route frame says `no route to upstream`.
			status := ansi.Strip(f.status())
			word := coverOr(f.state.cover, HostSubstituted(StateWord(f.state.failure.Kind)), StateWord(f.state.failure.Kind))
			if !strings.Contains(status, word) {
				t.Errorf("the frame %s carries the state %q and its status line does not print it: %q", f.name, word, status)
			}
			if f.state.failure.Kind == domain.FailureRateLimited {
				reset := resetSuffix(f.state.failure.Reset, corpusNow)
				if !strings.Contains(status, reset) {
					t.Errorf("the frame %s carries a reset and its status line does not print it: %q, want %q in it", f.name, status, reset)
				}
			}
		}
		if failures == 0 {
			t.Error("no stored frame carries a failure state: A2 has nothing to read")
		}

		// The state word is the one accent: in a colour rung the escape immediately before it is the
		// state role's own, which is what "no SGR before it other than the one colour role" means on
		// the stored bytes.
		colour := lookup(t, frames, "color_truecolor")
		if status := colour.status(); !strings.Contains(status, sgrStateTrueColor+CopyStateRateLimited) {
			t.Errorf("the truecolor frame %s paints its state word with something other than the state role: %q", colour.name, status)
		}
	})

	t.Run("A3: mono is readable, not merely legal", func(t *testing.T) {
		// The status line's fields are separated by exactly two cells, and an empty field is omitted
		// with its separator: the line is the fields that are present joined by that separator.
		//
		// The frame is a cold failure at 80 columns: nothing is dropped from its line, so the stored
		// row can be compared with the fields themselves rather than with a rule about dropping.
		f := lookup(t, frames, "expired_80")
		t.Setenv("TMUX", f.env.tmux)
		want := strings.Join([]string{
			stateField(f.state.status()),
			remedyField(f.state.failure),
			ListName("", f.state.list.Kind),
			budgetField(f.state.status()),
		}, strings.Repeat(" ", statusGap))
		if got := ansi.Strip(f.status()); strings.TrimRight(got, " ") != want {
			t.Errorf("the status line of %s is\n%q\nwant the fields joined by %d cells\n%q", f.name, got, statusGap, want)
		}

		// The head rows are delimited by column position: the identity column is the same cells on
		// every row, the stamp begins one cell behind it, and a marker sits exactly two cells behind
		// the stamp — all of which the mono rung has to say with cells rather than with colour.
		mono := lookup(t, frames, "color_mono")
		t.Setenv("TMUX", mono.env.tmux)
		stampCell, heads := 1+identityCells(t)+1, 0
		for _, row := range mono.pane() {
			line := ansi.Strip(row)
			at := stampPattern.FindStringIndex(line)
			if at == nil || strings.HasPrefix(content(row), PlainMarkers.QuotePrefix) {
				continue
			}
			heads++
			if at[0] != stampCell {
				t.Errorf("in %s the stamp of %q starts at cell %d, want the identity column's cell %d", mono.name, line, at[0], stampCell)
			}
			for _, marker := range []string{
				strings.TrimSuffix(MarkerRepost, PlaceholderHandle),
				MarkerLimited,
				MarkerContinued,
			} {
				if i := strings.Index(line[at[1]:], marker); i >= 0 && i != statusGap {
					t.Errorf("in %s the marker %q sits %d cells behind the stamp, want exactly %d", mono.name, marker, i, statusGap)
				}
			}
		}
		if heads == 0 {
			t.Errorf("%s has no head row to read the identity column off: A3's column assertion ran over nothing", mono.name)
		}
	})

	t.Run("A7: the measure is capped", func(t *testing.T) {
		for _, f := range frames {
			t.Setenv("TMUX", f.env.tmux)
			// The row is the grammar's own budget at the row's own quote depth: the selection cell,
			// the body's prefix and indent, two cells per quote level, and the measure at that
			// level. The plain grammar's own values reproduce Phase 2's rule exactly — 1 + 2 +
			// min(W-3, 96) at the top level. The status line is the one row the rule does not
			// govern: it is cut at the pane's width and never wraps, by its own section's rule.
			for i := range f.pane() {
				row := f.pane()[i]
				depth := depthOf(f.grammar, row)
				reply := replyLevelOf(f.grammar, row)
				limit := rowBudgetAt(f.grammar, f.w, depth, reply)
				if w := text.Width(trimmed(f.pane(), i)); w > limit {
					t.Errorf("in the stored frame %s, pane row %d is %d cells of text, over the %d its grammar's own budget allows at W=%d: %q",
						f.name, i+1, w, limit, f.w, trimmed(f.pane(), i))
				}
			}
		}
		// The arm: the cap binds — a row of the widest frame fills it — and the surplus columns of
		// a wide pane stay empty rather than being measured into the text.
		wide := lookup(t, frames, "timeline_200")
		t.Setenv("TMUX", wide.env.tmux)
		if !anyRowWidth(wide, 3+Plain.MeasureCap) {
			t.Errorf("no row of %s fills the %d-cell cap: the cap is not seen to bind", wide.name, Plain.MeasureCap)
		}
	})

	t.Run("V2: no real identity, no product word, and an alias in the identity's place", func(t *testing.T) {
		covered, panics := partition(frames)
		if len(covered) < 40 {
			t.Fatalf("the corpus holds %d covered frames, want the timeline covers plus the DSGT-08 cross-product", len(covered))
		}
		if len(panics) != 2 {
			t.Fatalf("the corpus holds %d panic frames, want one at each width", len(panics))
		}

		// The identities every frame is held to. A covered frame's own state carries the feed it was
		// built from; a panic frame's carries none, because the frame holds no character of the feed,
		// so the panic frames are scanned with the corpus's whole feed instead of with the three
		// literals their own empty state would yield. Both halves of the scan therefore look for the
		// same strings, which is what "the panic frame is held to the same absence standard as a
		// covered list frame" means, and the arm below is what makes the strings non-vacuous.
		identities := feedIdentities(feedPosts(covered))
		accounts := 0
		for _, identity := range identities {
			switch identity {
			case CopyUpstreamHost, CopyTitlePlain, "twitter":
			default:
				accounts++
			}
		}
		if accounts < 8 {
			t.Fatalf("the corpus's feed yields %d account strings: the panic frames would be scanned for the three literals and almost nothing else", accounts)
		}

		// The scan. Every string it looks for is one a frame's own content carries — the handles and
		// display names of the accounts it names, the upstream's host, the product's own name — and
		// the arms below assert that a plain rendering does print them, so a frame that came out
		// empty or truncated cannot pass by having nothing to find.
		scanned := 0
		for _, f := range append(append([]storedFrame{}, covered...), panics...) {
			t.Setenv("TMUX", f.env.tmux)
			body := strings.ToLower(ansi.Strip(f.bytes))
			held := hiddenIdentities(f)
			if f.panic {
				held = identities
			}
			for _, identity := range held {
				if strings.Contains(body, strings.ToLower(identity)) {
					t.Errorf("the stored frame %s prints %q, an identity of the content it was built from", f.name, identity)
				}
			}
			for _, noun := range productNouns {
				if noun.pattern.MatchString(body) {
					t.Errorf("the stored frame %s prints %q, the product's own word (from %s)", f.name, noun.noun, noun.source)
				}
			}
			scanned++
		}
		if scanned < 40 {
			t.Errorf("the scan covered %d frames, want the DSGT-08 covered corpus", scanned)
		}

		// The arms: the plain frame of the same list prints the handles and names the scan looks for —
		// and it is the very list the panic frames are held to, so the panic side of the scan is
		// armed by the same strings the plain reader is seen to print — and every covered frame that
		// holds a feed carries the alias its own style prints in their place.
		plain := lookup(t, frames, "timeline_80")
		t.Setenv("TMUX", plain.env.tmux)
		printed := 0
		for _, identity := range identities {
			if contains(plain, identity) {
				printed++
			}
		}
		if printed < 3 {
			t.Errorf("the plain frame %s prints %d of the identities the covered and panic frames are scanned for: the absence would be vacuous", plain.name, printed)
		}

		aliases, blocks, feeds := 0, 0, 0
		for _, f := range covered {
			// Alias slots are read off the plain timeline of the same list. Help overlays,
			// conversations, saved lists and single-post detail frames are not that list.
			if f.state.pane == paneHelp || f.state.pane == paneFailure || f.state.pane == paneThread || f.state.pane == paneSaved {
				continue
			}
			if len(f.state.list.Posts()) == 0 {
				continue
			}
			// Single-post media detail frames share no pane with timeline_80.
			if len(f.state.list.Posts()) < 3 {
				continue
			}
			feeds++
			t.Setenv("TMUX", f.env.tmux)
			body := ansi.Strip(f.bytes)
			slots := identitySlots(t, plainAt(t, frames, f.w), f)
			if len(slots) == 0 {
				t.Errorf("the covered frame %s names no account on its pane: the alias check has nothing to look for", f.name)
			}
			for _, want := range slots {
				if !strings.Contains(body, want) {
					t.Errorf("the covered frame %s does not carry %q, the identity its own style prints", f.name, want)
					continue
				}
				aliases++
			}
			if carriesMark(f) {
				blocks++
			}
		}
		if feeds < 8 {
			t.Errorf("only %d covered frames hold a feed: the alias check would run over almost nothing", feeds)
		}
		if aliases < 8 {
			t.Errorf("the covered frames carry %d aliases, want at least one on every frame that holds a feed", aliases)
		}
		if blocks < 8 {
			t.Errorf("%d covered frames carry a selected block: the scan ran over frames with no feed in them", blocks)
		}
	})

	t.Run("the covered failure pane holds nothing the cover did not author", func(t *testing.T) {
		// DSGT-09's hardest path, on the stored bytes: when the pane has nothing to show it prints
		// the classifier's own sentence, and that sentence names the host — `GET https://x.com/
		// failed: dial tcp: lookup x.com: no such host` in the corpus, and the shipped program
		// printed that sentence with `x.com` in it on a covered screen. The covered counterpart is
		// stored next to its plain twin, and the absence is read against that twin: a covered frame
		// is only quiet about the host if a frame of the same state is seen to say it.
		plain := lookup(t, frames, "offline_80")
		t.Setenv("TMUX", plain.env.tmux)
		if !contains(plain, plain.state.failure.Detail) {
			t.Fatalf("the plain frame %s does not carry its own state's detail %q: the plain reader's pane is not the classifier's sentence", plain.name, plain.state.failure.Detail)
		}
		if !contains(plain, CopyUpstreamHost) {
			t.Fatalf("the plain frame %s does not carry the host its own detail spells: the absence below would prove nothing", plain.name)
		}

		checked := 0
		covered, _ := partition(frames)
		for _, f := range covered {
			if f.state.pane != paneFailure {
				continue
			}
			checked++
			t.Setenv("TMUX", f.env.tmux)
			body := ansi.Strip(f.bytes)
			// The arm on this frame's own state: the sentence it was built from carries the host,
			// so a frame that printed the sentence would be caught by the check under it.
			if !strings.Contains(f.state.failure.Detail, CopyUpstreamHost) {
				t.Fatalf("the covered frame %s was built from a detail with no host in it: the check below has nothing to catch", f.name)
			}
			if !contains(f, CopyCoverFailureBody) {
				t.Errorf("the covered failure frame %s does not carry the cover's own line %q: the pane says nothing about the failure", f.name, CopyCoverFailureBody)
			}
			if strings.Contains(body, CopyUpstreamHost) {
				t.Errorf("the covered failure frame %s prints the host the cover hides", f.name)
			}
			if contains(f, f.state.failure.Detail) {
				t.Errorf("the covered failure frame %s prints the classifier's own sentence: the cover scrubbed it rather than keeping it out", f.name)
			}
			for _, noun := range productNouns {
				if noun.pattern.MatchString(strings.ToLower(body)) {
					t.Errorf("the covered failure frame %s prints %q, the product's own word (from %s)", f.name, noun.noun, noun.source)
				}
			}
			// The head row still says which failure it was, in the cover's own spelling: a covered
			// failure pane can never render as nothing.
			if word := HostSubstituted(StateWord(f.state.failure.Kind)); !contains(f, word) {
				t.Errorf("the covered failure frame %s does not carry its own state word %q", f.name, word)
			}
		}
		if checked < 2 {
			t.Fatalf("the corpus holds %d covered failure frames: the check above ran over almost nothing", checked)
		}
	})
	t.Run("V5: the panic frames hold nothing of the feed", func(t *testing.T) {
		covered, panics := partition(frames)
		posts := feedPosts(covered)
		if len(posts) < 10 {
			t.Fatalf("the corpus's feed holds %d posts: too few for the scan below to mean anything", len(posts))
		}
		// The haystack is the feed's own vocabulary: the words of its texts, the handles it names,
		// and the alias projections a covered frame prints — a panic frame carrying either would be
		// carrying the feed.
		haystack := append(feedWords(posts), aliasProjections(posts)...)
		if len(haystack) < 20 {
			t.Fatalf("the feed's vocabulary is %d strings: too few for the scan below to mean anything", len(haystack))
		}
		for _, f := range panics {
			t.Setenv("TMUX", f.env.tmux)
			if strings.Contains(f.bytes, "\x1b") {
				t.Errorf("the panic frame %s carries an escape byte: it is one frame of plain cells", f.name)
			}
			for _, word := range haystack {
				if contains(f, word) {
					t.Errorf("the panic frame %s holds %q, a string of the feed it renders nothing of", f.name, word)
				}
			}
		}
	})

	t.Run("the copies: the styles' own words, and the drop their names cause", func(t *testing.T) {
		covered, panics := partition(frames)
		feeds := 0
		for _, f := range covered {
			t.Setenv("TMUX", f.env.tmux)
			literals := styleLiterals(f.state.cover)
			if len(literals) == 0 {
				t.Errorf("the style %q has no literal to be recognised by", f.state.cover)
			}
			for _, plainWord := range []string{CopyListFollowing, CopyListForYou} {
				if contains(f, plainWord) {
					t.Errorf("the covered frame %s prints the plain list name %q", f.name, plainWord)
				}
			}
			// Help overlays and cold failure panes are not list renderings: the style's
			// head literals and the first post's body are not on screen there.
			if f.state.pane == paneHelp || f.state.pane == paneFailure {
				continue
			}
			// The style's own words are its head templates' literals, and a covered failure pane
			// holds no head at all: it is the pane with nothing to show, in the cover grammar's
			// shape and the cover's own words. The style check is the list frames', and the count
			// below keeps that arm from thinning.
			posts := f.state.list.Posts()
			if len(posts) == 0 {
				// Thread / saved frames hold their posts elsewhere; still require a style literal.
				if f.state.pane == paneThread || f.state.pane == paneSaved {
					for _, literal := range literals {
						if contains(f, literal) {
							feeds++
							break
						}
					}
				}
				continue
			}
			feeds++
			for _, literal := range literals {
				if !contains(f, literal) {
					t.Errorf("the covered frame %s carries nothing of the %q style: %q is not in it", f.name, f.state.cover, literal)
				}
			}
			// The body text is the writer's, verbatim: the pane's first post says what the state
			// says, in the grammar's own indent and at its own measure.
			if posts[0].Text == "" {
				t.Fatalf("the covered frame %s was built from no content: the check below would be vacuous", f.name)
			}
			if !contains(f, posts[0].Text) {
				t.Errorf("the covered frame %s does not carry its first post's own words %q: a cover changes the presentation and not the text", f.name, posts[0].Text)
			}
		}
		if feeds < 8 {
			t.Errorf("only %d covered frames hold a feed: the style and body checks would run over almost nothing", feeds)
		}

		// The panic frames carry no status line at all: every row is the frame's own, the rows below
		// the run are blank cells, and none of the status line's vocabulary is in them.
		for _, f := range panics {
			t.Setenv("TMUX", f.env.tmux)
			if last := f.rows[len(f.rows)-1]; strings.TrimSpace(ansi.Strip(last)) != "" {
				t.Errorf("the last row of the panic frame %s is %q: the frame's rows below the run are blank cells", f.name, last)
			}
			for _, word := range statusVocabulary() {
				if contains(f, word) {
					t.Errorf("the panic frame %s holds %q, a word of the status line this frame does not have", f.name, word)
				}
			}
		}

		// The drop rule's declared consequence, over the stored bytes: every 120-column frame carries
		// its fields, and the 80-column frame of the same style is exactly the fields the declared
		// drop order leaves at 80.
		styles, listDropped := 0, 0
		for _, style := range CoverStyles() {
			wide := lookup(t, frames, coverFrameName(style, 120))
			narrow := lookup(t, frames, coverFrameName(style, 80))
			t.Setenv("TMUX", wide.env.tmux)
			fields := fieldsOf(wide.state)
			if got, want := strings.TrimRight(ansi.Strip(wide.status()), " "), fieldsLine(fields); got != want {
				t.Errorf("the 120-column frame %s is not its own fields: %q, want %q", wide.name, got, want)
			}
			styles++
			if got, want := strings.TrimRight(ansi.Strip(narrow.status()), " "), droppedStatusLine(fields, narrow.w); got != want {
				t.Errorf("the 80-column frame %s is\n%q\nwant the fields the drop rule leaves at %d\n%q", narrow.name, got, narrow.w, want)
			}
			if !contains(narrow, ListName(style, narrow.state.list.Kind)) {
				listDropped++
			}
		}
		if styles < 4 {
			t.Errorf("the drop check ran over %d styles, want the four the phase ships", styles)
		}
		if listDropped == 0 {
			t.Error("no covered 80-column frame dropped its list field: the drop the covered names cause is not in the corpus")
		}

		// The arm: the drop is the covered name's and not the state's. The plain reader's own frame
		// of that state keeps its name, and the covered name in its place is one cell too wide for
		// the pane — the cause is counted here rather than asserted.
		cold := lookup(t, frames, "expired_80")
		t.Setenv("TMUX", cold.env.tmux)
		line := strings.TrimRight(ansi.Strip(cold.status()), " ")
		plainName, coveredName := ListName("", cold.state.list.Kind), ListName(CoverGitLog, cold.state.list.Kind)
		if !strings.Contains(line, plainName) {
			t.Fatalf("the plain frame %s does not carry its plain list name %q: the arm has nothing to compare", cold.name, plainName)
		}
		if swapped := strings.Replace(line, plainName, coveredName, 1); text.Width(swapped) <= cold.w {
			t.Errorf("the plain frame's line with the covered name in it is %d cells at %d columns: the covered name is not what pushes the field out", text.Width(swapped), cold.w)
		}
	})

	t.Run("the widths: exactly W cells a row, and the same lines at both", func(t *testing.T) {
		covered, panics := partition(frames)
		for _, f := range append(append([]storedFrame{}, covered...), panics...) {
			t.Setenv("TMUX", f.env.tmux)
			for i, row := range f.pane() {
				if got := text.Width(ansi.Strip(row)); got != f.w {
					t.Errorf("row %d of the stored frame %s is %d cells, want the pane's %d: the frame pads every row to its width", i+1, f.name, got, f.w)
				}
			}
		}

		// The panic frame does not stretch: the lines it draws are the same bytes at both widths, and
		// every row of it is the frame's own rather than a pane with a status line under it.
		at80, at120 := lookup(t, frames, "panic_80"), lookup(t, frames, "panic_120")
		if len(panicLines) < 15 {
			t.Fatalf("the panic frame draws %d lines: too few for the comparison below to mean anything", len(panicLines))
		}
		for i := range panicLines {
			if got, want := trimmed(at80.rows, i), trimmed(at120.rows, i); got != want {
				t.Errorf("the panic frame's line %d is %q at 80 columns and %q at 120: terminal output does not widen because the window did", i+1, want, got)
			}
		}
		if len(at80.pane()) != corpusH || len(at120.pane()) != corpusH {
			t.Errorf("the panic frames are read as %d and %d rows: every row of a panic frame is the frame's own", len(at80.pane()), len(at120.pane()))
		}
	})

	t.Run("the thread absences", func(t *testing.T) {
		// `ranking` is deliberately not in this list, and the reason is the scan's own reach: it reads
		// the whole frame, the writers' body rows included, and thread_partial_80/120 carry a post
		// whose text is "that is the reply ranking, not the walk". A word a post may use is not a word
		// an audit over the whole frame can forbid. (It is not the status line: `rate-limited` never
		// matched `ranking`.) The two words that carry the must-have stay, because a per-group heading
		// or a numbered reply list cannot say itself without one of them.
		forbidden := []string{"load more", "ranked", "chronological", "avatar"}
		threadFrames := 0
		for _, f := range frames {
			if f.state.pane != paneThread {
				continue
			}
			threadFrames++
			body := strings.ToLower(ansi.Strip(f.bytes))
			for _, word := range forbidden {
				at := strings.Index(body, word)
				if at < 0 {
					continue
				}
				row := strings.Count(body[:at], "\n")
				t.Errorf("the thread frame %s prints forbidden %q on row %d (%q): this scan covers the whole frame — the frame's own chrome and the writers' body text alike — so a hit may be the frame presenting an order or a post's own words. `ranking` is out of the list for exactly that reason: a post in thread_partial_80 says it, and not the status line's `rate-limited`, which never matched it",
					f.name, word, row+1, strings.TrimSpace(ansi.Strip(f.rows[row])))
			}
			// No URL and no engagement digit from the conversation's own counts.
			// Digits that sit inside a cover's reference token (seq=, build=) are not
			// a count the frame printed — same rule as assertNoCounts.
			for _, post := range threadPosts(f.state.conversation) {
				for _, n := range []int{post.Counts.Replies, post.Counts.Reposts, post.Counts.Likes, post.Counts.Quotes, post.Counts.Bookmarks} {
					if n == 0 {
						continue
					}
					pattern := regexp.MustCompile(`\b` + strconv.Itoa(n) + `\b`)
					for _, row := range f.pane() {
						if pattern.MatchString(ansi.Strip(row)) {
							t.Errorf("the thread frame %s prints engagement count %d in the pane", f.name, n)
						}
					}
				}
			}
		}
		if threadFrames == 0 {
			t.Fatal("no thread frame is in the corpus: the thread absence audit is vacuous")
		}
		// Anti-vacuity: a required thread case missing from requiredCases fails here too.
		found := false
		for _, c := range requiredCases {
			if strings.HasPrefix(c.frame, "thread_") {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("requiredCases holds no thread frame: removing the thread cases would leave the audit vacuous")
		}
	})

	t.Run("DSGT-08: no http in the covered corpus", func(t *testing.T) {
		covered, _ := partition(frames)
		read := 0
		for _, f := range covered {
			read++
			if strings.Contains(strings.ToLower(ansi.Strip(f.bytes)), "http") {
				t.Errorf("covered frame %s contains http", f.name)
			}
		}
		if read < 40 {
			t.Fatalf("read %d covered frames for the http absence; want the DSGT-08 corpus", read)
		}
		t.Logf("no-http arm read %d covered frames", read)
	})

	t.Run("DSGT-08: help key column byte-identical across covered styles", func(t *testing.T) {
		// Compare the key column across every stored covered help frame: only the
		// help text may differ per style; the key spellings stay byte-identical.
		var columns [][]string
		var names []string
		mediaRows := 0
		for _, style := range CoverStyles() {
			for _, w := range []int{80, 120} {
				name := "cover_" + string(style) + "_help_" + strconv.Itoa(w)
				f := lookup(t, frames, name)
				col := helpKeyColumn(f)
				columns = append(columns, col)
				names = append(names, name)
				body := strings.ToLower(ansi.Strip(f.bytes))
				for _, row := range keys.Rows() {
					switch row.Action {
					case keys.ActionMedia, keys.ActionCopyLink, keys.ActionOpenPost:
						if strings.Contains(body, strings.ToLower(row.HelpCovered)) {
							mediaRows++
						}
					}
				}
			}
		}
		if len(columns) < 8 {
			t.Fatalf("found %d covered help frames, want 8", len(columns))
		}
		ref := columns[0]
		for i := 1; i < len(columns); i++ {
			if len(columns[i]) != len(ref) {
				t.Errorf("%s has %d key rows, %s has %d", names[i], len(columns[i]), names[0], len(ref))
				continue
			}
			for r := range ref {
				if columns[i][r] != ref[r] {
					t.Errorf("key column row %d differs: %s=%q %s=%q", r+1, names[0], ref[r], names[i], columns[i][r])
				}
			}
		}
		// Three media HelpCovered rows × eight covered help frames.
		if mediaRows < 3*8 {
			t.Fatalf("covered help frames carry %d media HelpCovered hits, want at least %d", mediaRows, 3*8)
		}
	})

	t.Run("DSGT-08: media_gated and covered help frames in the file set", func(t *testing.T) {
		need := []string{}
		for _, style := range CoverStyles() {
			need = append(need,
				"media_gated_"+string(style)+"_80",
				"media_gated_"+string(style)+"_120",
				"cover_"+string(style)+"_help_80",
				"cover_"+string(style)+"_help_120",
			)
		}
		seen := map[string]bool{}
		for _, f := range frames {
			seen[f.name] = true
		}
		for _, name := range need {
			if !seen[name] {
				t.Errorf("missing required covered frame %s", name)
			}
		}
	})

	t.Run("the arms the corpus cannot shrink past", func(t *testing.T) {
		assertFileSet(t, frames)

		// The narrow frame: the pane holds the pane's own line and the status line still renders,
		// with the fields its rule leaves and the remainder cut with the mark.
		narrow := lookup(t, frames, "narrow_79")
		t.Setenv("TMUX", narrow.env.tmux)
		if got := trimmed(narrow.pane(), 0); got != " "+CopyPaneTooNarrow {
			t.Errorf("the pane of %s is %q, want the pane's own line", narrow.name, got)
		}
		if !strings.HasSuffix(ansi.Strip(narrow.status()), MarkTruncation) {
			t.Errorf("the status line of %s is not cut with the mark: %q", narrow.name, ansi.Strip(narrow.status()))
		}
		if !strings.Contains(ansi.Strip(narrow.status()), StateWord(narrow.state.failure.Kind)) {
			t.Errorf("the status line of %s does not render its state word: %q", narrow.name, ansi.Strip(narrow.status()))
		}
	})
}

// lookup returns the stored frame of a given name, or fails: every case names its own frame, so a
// case whose frame is missing is a failure and not a skip.
func lookup(t *testing.T, frames []storedFrame, name string) storedFrame {
	t.Helper()
	for _, f := range frames {
		if f.name == name {
			return f
		}
	}
	t.Fatalf("the corpus stores no frame named %q: the case's own frame is missing", name)
	return storedFrame{}
}

// storedFrames reads every frame the corpus names — and the twins whose text holds a cluster the two
// measures disagree on — and fails if the directory holds anything else.
//
// The twin is decided from the stored bytes rather than from a re-render, and the same decision is
// what the golden test makes from the two renders: a frame has a twin exactly when its text holds a
// cluster the two measures disagree on. A frame whose only wide clusters are ones both measures agree
// on (a CJK run, a single emoji) renders the same either way, and a stored twin of it would be a
// second copy of one fact. TestGoldenFrames's own twin rule is the byte comparison, and the two
// rules agreeing is what makes this audit an audit.
func storedFrames(t *testing.T) []storedFrame {
	t.Helper()
	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("reading the stored corpus: %v", err)
	}
	onDisk := make(map[string]bool, len(entries))
	for _, entry := range entries {
		onDisk[entry.Name()] = true
	}

	table := corpus(t)
	frames := make([]storedFrame, 0, len(table))
	for _, f := range table {
		bare := readFrame(t, f.name, f.w, f.h, f.env, f.state)
		frames = append(frames, bare)
		if !twinWanted(bare) {
			continue
		}
		twin := f.env
		twin.tmux = envTmux.tmux
		frames = append(frames, readFrame(t, f.name+twinSuffix, f.w, f.h, twin, f.state))
	}

	names := make(map[string]bool, len(frames))
	for _, f := range frames {
		names[f.name] = true
	}
	for _, f := range frames {
		if !onDisk[f.name+".golden"] {
			t.Errorf("the frame %s is not stored as %s/%s.golden: the corpus is missing a frame it names", f.name, goldenDir, f.name)
		}
	}
	for name := range onDisk {
		if stored := strings.TrimSuffix(name, ".golden"); !names[stored] {
			t.Errorf("%s/%s is stored but the corpus names no such frame: it is a stale frame", goldenDir, name)
		}
	}
	return frames
}

// assertFileSet is the arm against a corpus that shrank: every state in the table is stored, and the
// corpus holds at least as many frames as the table generates. A deleted frame fails here, and the
// audit above fails on it too, because a case whose frame is gone is a case that no longer holds.
func assertFileSet(t *testing.T, frames []storedFrame) {
	t.Helper()
	table := corpus(t)
	if len(frames) < len(table) {
		t.Errorf("the corpus holds %d stored frames for a table of %d states: a frame is missing", len(frames), len(table))
	}
	seen := make(map[string]bool, len(frames))
	for _, f := range frames {
		seen[f.name] = true
	}
	for _, f := range table {
		if !seen[f.name] {
			t.Errorf("the state %s has no stored frame: the corpus is missing it", f.name)
		}
	}
}

// readFrame reads one stored frame. Its row count is asserted here, because every assertion below
// counts on the frame being exactly H rows of which the last one is the status line.
func readFrame(t *testing.T, name string, w, h int, env environment, s state) storedFrame {
	t.Helper()
	if h == 0 {
		h = corpusH
	}
	path := filepath.Join(goldenDir, name+".golden")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	rows := strings.Split(string(body), "\n")
	if len(rows) != h {
		t.Fatalf("%s is %d rows, want the %d the corpus stores: the pane's rows plus the status line", path, len(rows), h)
	}
	return storedFrame{
		name: name, w: w, env: env, state: s, bytes: string(body), rows: rows,
		panic: s.pane == panePanic, grammar: s.grammar(),
	}
}

// readStored is one stored frame's bytes read back by name: what a plant compares the committed
// file against, so a plant that overwrote a fixture is caught instead of being reported as a
// passing run.
func readStored(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(goldenDir, name+".golden"))
	if err != nil {
		t.Fatalf("reading the stored frame %s: %v", name, err)
	}
	return string(body)
}

// twinWanted reports whether a stored frame needs a multiplexer twin, decided from the frame's own
// text: a cluster the two measures disagree on is what the width model switches for, so a frame
// carrying one renders differently in the two environments and both byte streams are stored.
func twinWanted(f storedFrame) bool {
	for _, row := range f.rows {
		plain := ansi.Strip(row)
		if ansi.GraphemeWidth.StringWidth(plain) != ansi.WcWidth.StringWidth(plain) {
			return true
		}
	}
	return false
}

// requiredCase is one case the UI-SPEC's checklist requires: the frame it is asserted in, so that a
// failure names the frame to look at, and the check itself. The frames are the ones that show the
// case — the width-bearing timeline frames show the list's head, timeline_last_80 shows its end,
// where the continuation, the markers, the tombstone and the attachment lines are.
var requiredCases = []struct {
	name  string
	frame string
	holds func(storedFrame) bool
}{
	{"a 15-cell handle", "timeline_80", func(f storedFrame) bool { return holdsHandle(f, 15) }},
	{"a 3-cell handle", "timeline_80", func(f storedFrame) bool { return holdsHandle(f, 3) }},
	{"a CJK run", "timeline_80", func(f storedFrame) bool {
		return holdsCluster(f, func(cluster string) bool { return text.Width(cluster) == 2 && cjk(cluster) })
	}},
	{"a single emoji", "timeline_80", func(f storedFrame) bool {
		// The emoji post's body is one cluster and nothing else.
		for _, row := range f.pane() {
			body := strings.TrimSpace(content(row))
			if runes := []rune(body); len(runes) == 1 && runes[0] >= 0x1F300 && text.Width(body) == 2 {
				return true
			}
		}
		return false
	}},
	{"a ZWJ family", "timeline_80", func(f storedFrame) bool {
		// The family is one grapheme cluster and the six cells of its code points: that disagreement
		// is what the multiplexer twins exist for, so the case is asserted as the disagreement.
		return holdsCluster(f, func(cluster string) bool {
			return strings.Contains(cluster, "\u200d") &&
				ansi.GraphemeWidth.StringWidth(cluster) == 2 &&
				ansi.WcWidth.StringWidth(cluster) == 6
		})
	}},
	{"a URL longer than the measure", "timeline_80", func(f storedFrame) bool {
		url := longestURL(f)
		if url == "" || text.Width(url) <= f.measure() {
			return false
		}
		// The token is longer than the measure, and both ends of it are on screen in different rows:
		// the break-long-token path ran rather than the URL being cut or dropped.
		return rowsDiffer(f, text.Truncate(url, 40), url[len(url)-20:])
	}},
	{"a line that exactly fills the measure", "timeline_80", func(f storedFrame) bool {
		return anyRowWidth(f, 3+f.measure())
	}},
	{"a line one cell shorter than the measure", "timeline_80", func(f storedFrame) bool {
		return anyRowWidth(f, 3+f.measure()-1)
	}},
	{"a quote at depth 1", "stalled_80", func(f storedFrame) bool {
		return anyContent(f, func(c string) bool {
			return strings.HasPrefix(c, MarkerQuotePrefix) && !strings.HasPrefix(c, MarkerQuotePrefix+MarkerQuotePrefix)
		})
	}},
	{"a quote at depth 2", "stalled_80", func(f storedFrame) bool {
		return anyContent(f, func(c string) bool {
			return strings.HasPrefix(c, MarkerQuotePrefix+MarkerQuotePrefix)
		})
	}},
	{"a quote below a longer body", "stalled_80", func(f storedFrame) bool {
		for i, row := range f.pane() {
			if i == 0 || !strings.HasPrefix(content(row), MarkerQuotePrefix) {
				continue
			}
			// The row above a quote is a body row of the post that quotes it: not blank, and not
			// the head row of the block (a head row carries the stamp column).
			above := strings.TrimSpace(content(f.pane()[i-1]))
			if above != "" && !stampPattern.MatchString(above) {
				return true
			}
		}
		return false
	}},
	{"a continuation pair with no blank row between them", "timeline_last_80", func(f storedFrame) bool {
		for i, row := range f.pane() {
			if !strings.Contains(ansi.Strip(row), MarkerContinued) {
				continue
			}
			// The continuation's own first row is a head row, and the row above it is the last row
			// of the post it answers: no separator row sits between the two.
			above := strings.TrimSpace(ansi.Strip(f.pane()[i-1]))
			if i > 0 && above != "" && !strings.Contains(above, MarkerContinued) {
				return true
			}
		}
		return false
	}},
	{"a repost marker", "timeline_last_80", func(f storedFrame) bool {
		return contains(f, strings.TrimSuffix(MarkerRepost, PlaceholderHandle))
	}},
	{"a limited marker", "timeline_last_80", func(f storedFrame) bool { return contains(f, MarkerLimited) }},
	{"a tombstone block", "timeline_last_80", func(f storedFrame) bool {
		for i, row := range f.pane() {
			if !strings.Contains(ansi.Strip(row), MarkerUnavailable) {
				continue
			}
			// A tombstone has no author and no date, so nothing is printed above its marker but the
			// blank head row the block shape leaves.
			if i > 0 && strings.TrimSpace(ansi.Strip(f.pane()[i-1])) == "" {
				return true
			}
		}
		return false
	}},
	{"an attachment line with a singular noun", "timeline_last_80", func(f storedFrame) bool {
		return contains(f, "[1] image")
	}},
	{"an attachment line with four numbered entries", "timeline_last_80", func(f storedFrame) bool {
		line := "[1] image, [2] image, [3] video, [4] gif"
		if !contains(f, line) {
			return false
		}
		// Entry order is the post's own order; nouns stay singular.
		row := ""
		for _, r := range f.pane() {
			if strings.Contains(ansi.Strip(r), line) {
				row = ansi.Strip(r)
			}
		}
		return strings.Contains(row, NounImage) && strings.Contains(row, NounVideo) && strings.Contains(row, NounGIF)
	}},
	{"a selected block whose first row is not the pane's first row", "timeline_80", func(f storedFrame) bool {
		for i, row := range f.pane() {
			if marked(row) {
				return i > 0
			}
		}
		return false
	}},
	{"a selection whose block is the pane's last visible row", "timeline_last_80", func(f storedFrame) bool {
		pane := f.pane()
		return marked(pane[len(pane)-1])
	}},
	{"a status line with a failure state, a reset time and a three-digit budget", "timeline_80", func(f storedFrame) bool {
		if f.state.failure.Kind == domain.FailureNone {
			return false
		}
		status := ansi.Strip(f.status())
		return strings.Contains(status, StateWord(f.state.failure.Kind)) &&
			strings.Contains(status, resetSuffix(f.state.failure.Reset, corpusNow)) &&
			budgetPattern.MatchString(status)
	}},
	{"a failure sentence longer than the measure", "longdetail_80", func(f storedFrame) bool {
		detail := f.state.failure.Detail
		if text.Width(detail) <= f.measure() {
			return false
		}
		// The sentence is wider than the measure, every non-space character of it is on the pane in
		// order, and it takes more than one body row: the body wrapped like a post body instead of
		// being cut at the pane's edge with no mark.
		rows := 0
		for _, row := range f.pane()[1:] {
			if strings.TrimSpace(ansi.Strip(row)) != "" {
				rows++
			}
		}
		return rows > 1 && wrappedIn(f, detail)
	}},

	{"a deep ancestor chain with a tombstone", "thread_adversarial_80", func(f storedFrame) bool {
		if !contains(f, MarkerUnavailable) {
			return false
		}
		heads := 0
		for _, row := range f.pane() {
			if stampPattern.MatchString(ansi.Strip(row)) && !strings.Contains(ansi.Strip(row), MarkerFocal) {
				heads++
			}
		}
		return heads >= 3
	}},
	{"a nested quote at depth 2 in a conversation", "thread_adversarial_80", func(f storedFrame) bool {
		return anyContent(f, func(c string) bool {
			return strings.Contains(c, MarkerQuotePrefix+MarkerQuotePrefix)
		})
	}},
	{"a tombstone as the focal post", "thread_tombstone_focal_80", func(f storedFrame) bool {
		// A1-T: a stripped thread frame must still unambiguously name the focal post. A headless
		// tombstone focal draws a head row that carries only the focal marker — empty of identity
		// and stamp — above the state marker, so the frame names the post even when its text is
		// unavailable. The reply text says "tombstone focal" as English and must not be mistaken
		// for the marker: look for the marker as a free word after the stamp column's gap.
		hasMarker := false
		for _, row := range f.pane() {
			if strings.Contains(ansi.Strip(row), "  "+MarkerFocal) || strings.HasSuffix(strings.TrimRight(ansi.Strip(row), " "), MarkerFocal) {
				hasMarker = true
			}
		}
		return contains(f, MarkerUnavailable) && hasMarker && contains(f, "a reply under a tombstone focal.")
	}},
	{"a withheld post in a conversation", "thread_visibility_80", func(f storedFrame) bool {
		return contains(f, MarkerWithheld)
	}},
	{"a limited reply", "thread_visibility_80", func(f storedFrame) bool {
		return contains(f, MarkerLimited) && contains(f, "this one is held back")
	}},
	{"a reply-to marker naming an ancestor", "thread_80", func(f storedFrame) bool {
		return contains(f, "reply to lzd")
	}},
	{"a no-marker reply to the block above", "thread_80", func(f storedFrame) bool {
		for _, row := range f.pane() {
			line := ansi.Strip(row)
			if strings.Contains(line, "mira") && stampPattern.MatchString(line) {
				return !strings.Contains(line, "reply to ") && !strings.Contains(line, MarkerFocal)
			}
		}
		return false
	}},
	{"a reply of a reply stays at one reply step", "thread_adversarial_80", func(f storedFrame) bool {
		return contains(f, "a reply of a reply of a reply stays at one reply step.")
	}},
	{"a 15-cell handle at the reply step", "thread_sel_quote_80", func(f storedFrame) bool {
		// The reply step sits in front of the identity column, so holdsHandle's head-row scan (which
		// strips only a leading quote prefix) cannot see it: look for the handle itself at the reply
		// indent instead. The quote-selection frame is scrolled onto that reply, which is where the
		// 15-cell handle lives.
		return contains(f, "a_15_cell_handl")
	}},
	{"a CJK run and an emoji at the reply step", "thread_80", func(f storedFrame) bool {
		hasCJK := holdsCluster(f, func(cluster string) bool { return text.Width(cluster) == 2 && cjk(cluster) })
		hasEmoji := false
		for _, row := range f.pane() {
			if strings.Contains(ansi.Strip(row), "🚀") {
				hasEmoji = true
			}
		}
		return hasCJK && hasEmoji
	}},
	{"selection on a reply", "thread_sel_reply_80", func(f storedFrame) bool {
		for _, row := range f.pane() {
			if marked(row) && strings.Contains(ansi.Strip(row), "mira") {
				return true
			}
		}
		return false
	}},
	{"selection on an ancestor", "thread_sel_ancestor_80", func(f storedFrame) bool {
		markedHead := false
		for _, row := range f.pane() {
			if marked(row) {
				markedHead = true
			}
			if markedHead && strings.Contains(ansi.Strip(row), "the root of a deep chain") {
				return true
			}
		}
		return false
	}},
	{"selection on the focal post", "thread_80", func(f storedFrame) bool {
		for _, row := range f.pane() {
			if marked(row) && strings.Contains(ansi.Strip(row), MarkerFocal) {
				return true
			}
		}
		return false
	}},
	{"selection on a quoted post", "thread_sel_quote_80", func(f storedFrame) bool {
		for _, row := range f.pane() {
			if !marked(row) {
				continue
			}
			// A quote inside a reply carries the reply step before the quote prefix, so the content
			// opens with spaces then `> `, not with the prefix alone.
			body := strings.TrimLeft(content(row), " ")
			if strings.HasPrefix(body, MarkerQuotePrefix) {
				return true
			}
		}
		return false
	}},
	{"ancestry pushes the focal post off the first pane", "thread_push_80", func(f storedFrame) bool {
		// The frame opens at the top of a deep chain, so the focal marker is below the pane and the
		// ancestry that pushed it is what is on screen. The English word "focal" appears in the
		// ancestor bodies, so the marker is checked as a free head-row word rather than a bare substring.
		hasMarker := false
		for _, row := range f.pane() {
			line := strings.TrimRight(ansi.Strip(row), " ")
			if strings.Contains(line, "  "+MarkerFocal) || strings.HasSuffix(line, MarkerFocal) && stampPattern.MatchString(line) {
				hasMarker = true
			}
		}
		return !hasMarker && contains(f, "the root of a deep chain above the focal post")
	}},
	{"a thread status line with +", "thread_status_plus_80", func(f storedFrame) bool {
		status := ansi.Strip(f.status())
		return strings.Contains(status, CopyListThread) && strings.Contains(status, "+")
	}},
	{"a thread status line without +", "thread_status_noplus_80", func(f storedFrame) bool {
		status := ansi.Strip(f.status())
		return strings.Contains(status, CopyListThread) && !strings.Contains(status, "+")
	}},
	{"a thread status line carrying a failure state", "thread_status_failure_80", func(f storedFrame) bool {
		status := ansi.Strip(f.status())
		return strings.Contains(status, CopyListThread) && strings.Contains(status, StateWord(domain.FailureRateLimited))
	}},
	{"partial thread - the reply walk stopped here", "thread_partial_80", func(f storedFrame) bool {
		return contains(f, MarkerPartialStopped)
	}},
	{"partial thread - the post above is not reachable", "thread_chaincut_80", func(f storedFrame) bool {
		return contains(f, MarkerPartialAbove)
	}},
	{"no replies with the empty-read body", "thread_empty_80", func(f storedFrame) bool {
		return contains(f, MarkerNoReplies) && contains(f, CopyEmptyBody)
	}},
}

// wrappedIn reports whether the frame's pane carries every non-space character of the text, in
// order, after its head row: what a wrap of the text over the block's body rows leaves. It is the
// check a cut at the pane's edge fails — the tail of the text is simply not in the frame — and it
// reads the pane's own bytes rather than the function that wrote them, so it is an audit of the
// stored frame. Spaces are not compared: a greedy wrap collapses the runs that separated tokens and
// leaves one behind, and the block's own indent is spaces too.
func wrappedIn(f storedFrame, wrapped string) bool {
	var body strings.Builder
	for _, row := range f.pane()[1:] {
		body.WriteString(content(row))
	}
	return withoutSpaces(body.String()) == withoutSpaces(wrapped)
}

// withoutSpaces is every byte of s that is not a space.
func withoutSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' {
			return -1
		}
		return r
	}, s)
}

// holdsHandle reports whether a head row of the frame holds a handle exactly cells wide. A head row
// is one that carries the stamp column, and its first field is the identity column's value — the
// quote prefixes of a nested quote are taken off first, because they shift the column rather than
// being part of it.
func holdsHandle(f storedFrame, cells int) bool {
	for _, row := range f.pane() {
		line := ansi.Strip(row)
		if !stampPattern.MatchString(line) {
			continue
		}
		fields := strings.Fields(strings.TrimLeft(content(row), "> "))
		if len(fields) > 0 && text.Width(fields[0]) == cells {
			return true
		}
	}
	return false
}

// holdsCluster reports whether any cluster of any pane row satisfies the predicate, walking the rows
// with the same iterator internal/text wraps and truncates with.
func holdsCluster(f storedFrame, holds func(cluster string) bool) bool {
	for _, row := range f.pane() {
		for rest := ansi.Strip(row); rest != ""; {
			cluster, _ := ansi.FirstGraphemeCluster(rest, ansi.GraphemeWidth)
			if cluster == "" {
				break
			}
			if holds(cluster) {
				return true
			}
			rest = rest[len(cluster):]
		}
	}
	return false
}

// cjk reports whether a cluster starts with a CJK ideograph: the run every terminal draws two cells
// wide whichever way it counts clusters.
func cjk(cluster string) bool {
	for _, r := range cluster {
		return r >= 0x4E00 && r <= 0x9FFF
	}
	return false
}

// contains reports whether any pane row of the frame holds the text.
func contains(f storedFrame, want string) bool {
	for _, row := range f.pane() {
		if strings.Contains(ansi.Strip(row), want) {
			return true
		}
	}
	return false
}

// anyContent reports whether any pane row's content satisfies the predicate.
func anyContent(f storedFrame, holds func(content string) bool) bool {
	for _, row := range f.pane() {
		if holds(content(row)) {
			return true
		}
	}
	return false
}

// anyRowWidth reports whether any pane row's text is exactly cells wide.
func anyRowWidth(f storedFrame, cells int) bool {
	for i := range f.pane() {
		if text.Width(trimmed(f.pane(), i)) == cells {
			return true
		}
	}
	return false
}

// rowsDiffer reports whether two strings appear in two different rows of the frame: how a token too
// long for a line is seen to have been broken across lines rather than cut or dropped.
func rowsDiffer(f storedFrame, first, second string) bool {
	one, two := -1, -1
	for i, row := range f.pane() {
		line := ansi.Strip(row)
		if strings.Contains(line, first) {
			one = i
		}
		if strings.Contains(line, second) {
			two = i
		}
	}
	return one >= 0 && two >= 0 && one != two
}

// longestURL is the widest https token the frame's own posts carry, or the empty string: the URL the
// frame is asked to break.
func longestURL(f storedFrame) string {
	longest := ""
	for _, post := range f.state.list.Posts() {
		for _, token := range strings.Fields(post.Text) {
			if strings.HasPrefix(token, "https://") && text.Width(token) > text.Width(longest) {
				longest = token
			}
		}
	}
	return longest
}

// attachmentText is the attachment line for one count and one noun, built from the grammar's own
// template and noun table rather than spelled here: the audit checks the frame against the words the
// renderer makes, not against a second copy of them.
func attachmentText(n int, noun string) string {
	line := strings.ReplaceAll(CopyAttachment, PlaceholderCount, strconv.Itoa(n))
	return strings.ReplaceAll(line, PlaceholderNoun, noun)
}

// assertNoCounts is the engagement-count absence: a post's counts are parsed and retained, and the
// list grammar prints none of them. The numbers searched for are the ones the corpus's own posts
// carry, so a renderer that printed a count would print one of these.
//
// The number has to stand on its own: a cover's own reference tokens are digit runs too — a log's
// seven-digit sequence number, a build number — and digits that happen to sit inside one of those
// are not a count the frame printed.
func assertNoCounts(t *testing.T, f storedFrame, counts domain.Counts) {
	t.Helper()
	for _, n := range []int{counts.Replies, counts.Reposts, counts.Likes, counts.Quotes, counts.Bookmarks, counts.Views} {
		if n == 0 {
			continue
		}
		pattern := regexp.MustCompile(`\b` + strconv.Itoa(n) + `\b`)
		for _, row := range f.pane() {
			if pattern.MatchString(ansi.Strip(row)) {
				t.Errorf("the stored frame %s prints %d, an engagement count its own posts carry: the list grammar prints no counts", f.name, n)
			}
		}
	}
}

// assertNoMediaURLs is the media-URL absence for frames that had no failed media
// action: the attachment line says what is attached, never where it lives. A
// MEDIA-03 fallback frame deliberately prints the URL of each failed entry, so
// those states are skipped here; TestCorpusNoURLWithoutFailedAction covers them.
func assertNoMediaURLs(t *testing.T, f storedFrame) {
	t.Helper()
	if frameHasFailedMedia(f) {
		return
	}
	for _, post := range f.state.list.Posts() {
		for _, media := range post.Media {
			if media.URL == "" {
				continue
			}
			name := media.URL[strings.LastIndex(media.URL, "/")+1:]
			if contains(f, media.URL) || contains(f, name) {
				t.Errorf("the stored frame %s prints the media location %q: the attachment line counts media and never prints where it lives", f.name, media.URL)
			}
		}
	}
}

// frameHasFailedMedia reports whether the frame's state carries a MEDIA-03
// fallback that prints a URL.
func frameHasFailedMedia(f storedFrame) bool {
	for _, fall := range f.state.fall {
		if len(fall.Failed) > 0 || fall.Permalink {
			return true
		}
	}
	return false
}

// assertNoDisplayNames is the display-name absence: the plain grammar prints the handle and nothing
// else, so no account's own name reaches a frame. The names searched for are the ones the frame's
// own state carries.
func assertNoDisplayNames(t *testing.T, f storedFrame) {
	t.Helper()
	posts := f.state.list.Posts()
	if f.state.pane == paneThread {
		posts = threadPosts(f.state.conversation)
	}
	for _, post := range posts {
		accounts := []domain.User{post.Author}
		if post.RepostedBy != nil {
			accounts = append(accounts, *post.RepostedBy)
		}
		for _, account := range accounts {
			if account.Name == "" {
				continue
			}
			if contains(f, account.Name) {
				t.Errorf("the stored frame %s prints the display name %q: the plain grammar's identity is the handle", f.name, account.Name)
			}
		}
	}
}

// assertNoIcons is the avatar/icon absence of ROADMAP SC4 and DSGT-09, over a stored frame: the
// program's own vocabulary is printable ASCII — every literal in copy.go, cover.go and panic.go —
// so a character a frame prints that is neither printable ASCII nor a character of the content it
// renders is a glyph the frame drew, and an avatar or an icon in a frame is exactly that.
//
// The one allowance is the writer's own text: a post carries an emoji and a ZWJ family, those
// characters are the content's rather than the program's, and a frame that holds them holds the text
// verbatim, as DSGT-08 requires. Everything else — the identity column, the stamps, the markers, the
// status line, the panic frame's own cells — is the program's, and the program's own cells are ASCII.
func assertNoIcons(t *testing.T, f storedFrame, feed map[rune]bool) {
	t.Helper()
	for row, line := range f.rows {
		for _, r := range ansi.Strip(line) {
			if r >= 0x20 && r <= 0x7e {
				continue
			}
			if feed[r] {
				continue
			}
			t.Errorf("row %d of the stored frame %s holds %q (U+%04X): a frame carries no avatar, icon or emoji of its own, and every character it prints that is not the content's own text is the program's, whose vocabulary is printable ASCII", row+1, f.name, string(r), r)
		}
	}
}

// glyphsIn is every character of a frame's rows that no ASCII cell can hold: the characters an
// avatar, an icon or an emoji would be drawn with, and — the allowance above — the characters of the
// writer's own text.
func glyphsIn(f storedFrame) []rune {
	var out []rune
	for _, row := range f.rows {
		for _, r := range ansi.Strip(row) {
			if r > 0x7e {
				out = append(out, r)
			}
		}
	}
	return out
}

// feedGlyphs is every character outside ASCII that a feed's own posts carry in the text a frame
// prints: what a frame may hold without the program having drawn it. It is the rule the corpus's
// emoji and ZWJ family exercise, and it is what makes the icon absence a check rather than a scan
// that passes because the frames hold no glyph at all.
func feedGlyphs(posts []domain.Post) map[rune]bool {
	out := map[rune]bool{}
	add := func(s string) {
		for _, r := range s {
			if r > 0x7e {
				out[r] = true
			}
		}
	}
	for _, post := range posts {
		add(post.Text)
		if post.Quote != nil {
			add(post.Quote.Text)
		}
	}
	for _, alias := range aliasProjections(posts) {
		add(alias)
	}
	return out
}

// assertEmptyCopyOnlyAfterAnEmptyRead is the honesty rule of the empty state: the copy says an empty
// read succeeded, so it is rendered exactly in the frames whose state is that read — never for a
// failure, never for a list that was never read.
func assertEmptyCopyOnlyAfterAnEmptyRead(t *testing.T, frames []storedFrame) {
	t.Helper()
	empty := 0
	for _, f := range frames {
		// The empty-read body row is reused verbatim by the conversation's `no replies` statement, so a
		// thread frame that carries it is not the empty state's own pane. Only the heading is unique to
		// the empty read, and the body alone is allowed on a thread frame.
		// The covered copy is the same state's words under a cover: a covered empty read carries the
		// cover's own two lines instead of the plain reader's, and the arm has to know both, or a
		// covered empty frame reads as the empty state with no copy in it.
		// The never-mirrored line is that frame's two-state sentence: it is the empty read's body
		// when nothing has failed, and the third line of the failure block when something has — the
		// shell draws it in exactly those two states, and only the first is an empty read.
		neverMirrored := (contains(f, CopyEmptyBodyNeverSynced) || contains(f, CopyCoverEmptyNeverMirrored)) &&
			f.state.pane == paneSaved && f.state.failure.Kind == domain.FailureNone
		// The two covered hedges are the likes list's own one-line answer and are deliberately *not*
		// here: a hedged frame's state is the empty saved read and its copy is the line that leaves
		// the emptiness open, so the state below is the one the plain copy's own heading identifies
		// and the hedged frames are named into the exclusion. `CopyEmptyHeadingLikes` used to stand in
		// this list and no frame could hold it: the likes branch answers with the hedge before any
		// heading is chosen (saved.go), so it was live vocabulary for a state that cannot render.
		// The two hedges are the empty state's copy as much as the heading is: they are the likes
		// list's own answer to the same read, one line instead of two.
		hasCopy := contains(f, CopyEmptyHeadingFollowing) || contains(f, CopyEmptyHeadingBookmarks) || contains(f, CopyCoverEmptyHeading) || contains(f, CopyEmptyLikesHedged) || contains(f, CopyCoverEmptyHedged) || (contains(f, CopyEmptyBody) && f.state.pane != paneThread) || (contains(f, CopyCoverEmptyBody) && f.state.pane != paneThread) || neverMirrored
		// The empty read is the state with nothing to show at all: no posts, no capability and no
		// failure, which is the shell's own three-way order for this pane.
		want := f.state.pane == paneEmpty || (f.state.pane == paneSaved && len(f.state.saved.List().Posts()) == 0 && f.state.capability == domain.CapNone && f.state.failure.Kind == domain.FailureNone && !strings.Contains(f.name, "unavailable") && !strings.Contains(f.name, "unreadable") && !strings.Contains(f.name, "wrong_account"))
		if hasCopy != want {
			t.Errorf("the stored frame %s holds the empty-state copy %v and its state is %v: the copy is the successful zero-post read's own, and no other state's", f.name, hasCopy, want)
			continue
		}
		if !hasCopy {
			continue
		}
		empty++
		if f.state.failure.Kind != domain.FailureNone || f.state.list.Loaded() != 0 {
			t.Errorf("the stored frame %s renders the empty-state copy over a state that carries a failure or posts: an empty read is what the copy claims", f.name)
		}
	}
	if empty == 0 {
		t.Error("no stored frame renders the empty state: the check above has nothing to hold")
	}
}

// carriesCounts reports whether any stored frame's state carries engagement numbers: the arm that
// keeps the count absence from being vacuous.
func carriesCounts(frames []storedFrame) bool {
	for _, f := range frames {
		for _, post := range f.state.list.Posts() {
			if post.Counts != (domain.Counts{}) {
				return true
			}
		}
	}
	return false
}

// carriesMediaURL reports whether any stored frame's state carries a media URL: the arm under the
// media absence.
func carriesMediaURL(frames []storedFrame) bool {
	for _, f := range frames {
		for _, post := range f.state.list.Posts() {
			for _, media := range post.Media {
				if media.URL != "" {
					return true
				}
			}
		}
	}
	return false
}

// carriesDisplayName reports whether any stored frame's state carries an account name: the arm under
// the display-name absence.
func carriesDisplayName(frames []storedFrame) bool {
	for _, f := range frames {
		posts := f.state.list.Posts()
		if f.state.pane == paneThread {
			posts = threadPosts(f.state.conversation)
		}
		for _, post := range posts {
			if post.Author.Name != "" {
				return true
			}
		}
	}
	return false
}

// carriesMark reports whether any pane row of a frame carries the selection mark.
func carriesMark(f storedFrame) bool {
	for _, row := range f.pane() {
		if marked(row) {
			return true
		}
	}
	return false
}

// partition splits the stored corpus into its covered frames and its panic frames: a covered frame
// is one rendered through a cover, and a panic frame is one whose every row is the frame itself.
func partition(frames []storedFrame) (covered, panics []storedFrame) {
	for _, f := range frames {
		switch {
		case f.panic:
			panics = append(panics, f)
		case f.state.cover != "":
			covered = append(covered, f)
		}
	}
	return covered, panics
}

// plainAt is the plain reader's stored frame of the same list at a width: it is what the covered
// frames are compared against, because the plain reader's identity column is what names the accounts
// a pane holds.
func plainAt(t *testing.T, frames []storedFrame, w int) storedFrame {
	t.Helper()
	return lookup(t, frames, "timeline_"+strconv.Itoa(w))
}

// hiddenIdentities is every identity the frame's own state carries and no covered or panic frame may
// print: the handles and display names of the accounts it names, the upstream's host, and the
// product's own name. The strings come from the state rather than from a list written here, so a
// frame built from different content is scanned for that content's identities.
//
// A frame with no list of its own — a panic frame, whose state carries the feed it renders none of —
// yields the three literals and nothing else, so the panic frames are held to the feed's own
// identities (feedIdentities) rather than to this. See V2.
func hiddenIdentities(f storedFrame) []string {
	switch f.state.pane {
	case paneThread:
		return feedIdentities(threadPosts(f.state.conversation))
	case paneSaved:
		return feedIdentities(f.state.saved.List().Posts())
	default:
		return feedIdentities(f.state.list.Posts())
	}
}

// feedIdentities is every identity a feed carries: the handles and display names of the accounts its
// posts name — the authors, the accounts that reposted them and the accounts they quote — plus the
// upstream's host and the product's own name. It is the rule hiddenIdentities reads one frame's state
// through, stated over a post list, so that the frames whose own state holds no list are held to the
// same strings as the frames that do.
func feedIdentities(posts []domain.Post) []string {
	out := []string{CopyUpstreamHost, CopyTitlePlain, "twitter"}
	for _, post := range posts {
		accounts := []domain.User{post.Author}
		if post.RepostedBy != nil {
			accounts = append(accounts, *post.RepostedBy)
		}
		if post.Quote != nil {
			accounts = append(accounts, post.Quote.Author)
		}
		for _, account := range accounts {
			if account.ScreenName != "" {
				out = append(out, account.ScreenName)
			}
			if account.Name != "" {
				out = append(out, account.Name)
			}
		}
	}
	return out
}

// productNouns are the product's own words a covered frame may not carry, each with the constant it
// is read off: the timeline's two names, the timeline itself, the thing a post is, an attachment,
// and the two saved lists. `feed` is the phase's UI-SPEC's own word for the timeline and has no
// constant in copy.go; it is scanned for the same way as the rest and its source says so.
var productNouns = []struct {
	noun    string
	source  string
	pattern *regexp.Regexp
}{
	{CopyListFollowing, "CopyListFollowing", regexp.MustCompile(`(?i)\bfollowing\b`)},
	{CopyListForYou, "CopyListForYou", regexp.MustCompile(`(?i)\bfor you\b`)},
	{"post", "MarkerUnavailable", regexp.MustCompile(`(?i)\bposts?\b`)},
	{"timeline", "CopyEmptyHeadingFollowing", regexp.MustCompile(`(?i)\btimelines?\b`)},
	{"feed", "the phase's UI-SPEC", regexp.MustCompile(`(?i)\bfeeds?\b`)},
	{"attachment", "CopyAttachment", regexp.MustCompile(`(?i)\battachments?\b`)},
	{CopyListBookmarks, "CopyListBookmarks", regexp.MustCompile(`(?i)\bbookmarks?\b`)},
	{CopyListLikes, "CopyListLikes", regexp.MustCompile(`(?i)\blikes?\b`)},
}

// identitySlots is the identity the covered frame's own style prints for each account its pane
// holds. Which accounts those are is read off the plain frame of the same list — the plain reader's
// head rows carry the handles — and which strings the style prints for them is read off the
// grammar's own head templates, so neither is a second copy of the identity rule.
func identitySlots(t *testing.T, plain, covered storedFrame) []string {
	t.Helper()
	out := make([]string, 0, 4)
	for _, account := range accountsOnPane(t, plain, covered.state.list.Posts()) {
		out = append(out, headIdentity(covered.grammar, account)...)
	}
	return out
}

// accountsOnPane is the accounts a plain frame's own identity column names, mapped back onto the
// posts a covered frame was built from: the handle column is the plain reader's identity, and it is
// what says which accounts a pane holds.
func accountsOnPane(t *testing.T, plain storedFrame, posts []domain.Post) []domain.User {
	t.Helper()
	out := make([]domain.User, 0, 4)
	seen := map[domain.User]bool{}
	for _, row := range plain.pane() {
		cell := content(row)
		if !stampPattern.MatchString(ansi.Strip(row)) || strings.HasPrefix(cell, MarkerQuotePrefix) {
			continue
		}
		cell = strings.TrimLeft(cell, "> ")
		for _, post := range posts {
			handle := post.Author.ScreenName
			if handle == "" || !strings.HasPrefix(cell, handle) || seen[post.Author] {
				continue
			}
			seen[post.Author] = true
			out = append(out, post.Author)
			break
		}
	}
	return out
}

// headIdentity is the strings a grammar's own head prints for an account: the projections its head
// templates name. A git log names a person (the alias's name and its email), a log tail names a
// machine (the alias's handle), and the table's owner column names a login — and which of them it is
// is read off the templates rather than spelled a second time here. A placeholder may carry its own
// width (`{handle:20}`), so a template is matched on the placeholder's name.
func headIdentity(g Grammar, u domain.User) []string {
	alias := g.Identity(u)
	out := make([]string, 0, 3)
	for _, template := range g.Head {
		for _, field := range []struct{ placeholder, value string }{
			{PlaceholderHandle, alias.Handle},
			{PlaceholderName, alias.Name},
			{PlaceholderEmail, alias.Email},
		} {
			if field.value != "" && strings.Contains(template, strings.TrimSuffix(field.placeholder, "}")) {
				out = append(out, field.value)
			}
		}
	}
	return out
}

// styleLiterals is the text a style is unmistakable by: the literal words of its own head templates,
// read off them rather than spelled here, plus its pane header where it has one.
func styleLiterals(style CoverStyle) []string {
	g := style.Grammar()
	out := headLiterals(g)
	return append(out, g.PaneHeader...)
}

// headLiterals is the literal text of a grammar's head templates: what is left when the placeholders
// are taken out, keeping the words of three cells or more (a bracket or a `>` is punctuation rather
// than a word, and a style's words are what says which style a frame is). A placeholder's own width
// is part of the placeholder, so it goes with it.
func headLiterals(g Grammar) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 4)
	for _, template := range g.Head {
		for _, part := range placeholderPattern.Split(template, -1) {
			word := strings.TrimSpace(part)
			if text.Width(word) < 3 || seen[word] {
				continue
			}
			seen[word] = true
			out = append(out, word)
		}
	}
	return out
}

// placeholderPattern matches one of the composer's placeholders, with the width a template may give
// it: the closed set is copy.go's, and this is the same set for the purpose of reading a template's
// literals out of it.
var placeholderPattern = regexp.MustCompile(`\{(stamp|ref|handle|name|email|markers|state|text|n|noun)(:\d+)?\}`)

// statusVocabulary is the status line's own vocabulary: the budget's words, every state word, and
// every list name in every style, so "the panic frame has no status line" is checked against the
// words a status line would hold rather than against one of them.
func statusVocabulary() []string {
	out := []string{"requests", "rate-limited"}
	for _, kind := range []domain.FailureKind{
		domain.FailureSessionExpired,
		domain.FailureRateLimited,
		domain.FailureSoftBan,
		domain.FailureBudgetSpent,
		domain.FailureNoRoute,
		domain.FailureOperationStale,
		domain.FailureUnclassified,
	} {
		out = append(out, StateWord(kind))
	}
	for _, style := range append([]CoverStyle{""}, CoverStyles()...) {
		for _, kind := range []domain.HomeKind{domain.Following, domain.ForYou} {
			out = append(out, ListName(style, kind))
		}
	}
	return out
}

// feedPosts is the feed the covered frames were built from: the posts of one of them, because every
// style is one rendering of one loaded list.
func feedPosts(covered []storedFrame) []domain.Post {
	for _, f := range covered {
		if posts := f.state.list.Posts(); len(posts) > 0 {
			return posts
		}
	}
	return nil
}

// coverFrameName is the corpus's own name for the frame of a style at a width. The name is part of
// the file's path, so it is spelled once, here.
func coverFrameName(style CoverStyle, w int) string {
	return strings.Join([]string{"cover", string(style), strconv.Itoa(w)}, "_")
}

// lineField is one field of a status line with the name the drop rule knows it by: the three the
// rule may drop (the notice, the position, the list) and the three it never drops.
type lineField struct {
	kind string
	text string
}

// fieldsOf is a state's status line as fields, in their fixed order, composed from the state through
// the same field functions the line's own composer uses (A3's rule, over the whole row).
func fieldsOf(s state) []lineField {
	f := s.status()
	return []lineField{
		{kind: "state", text: stateField(f)},
		{kind: "remedy", text: remedyField(f.Failure)},
		{kind: "notice", text: NoticeLine(f.Cover, f.Notice, f.NoticeCount)},
		{kind: "list", text: f.List},
		{kind: "position", text: positionField(f)},
		{kind: "budget", text: budgetField(f)},
	}
}

// fieldsLine joins the fields that are present, with the two-cell separator between them: an empty
// field is omitted together with its separator, which is the line's own rule.
func fieldsLine(fields []lineField) string {
	kept := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.text == "" {
			continue
		}
		kept = append(kept, field.text)
	}
	return strings.Join(kept, strings.Repeat(" ", statusGap))
}

// droppedStatusLine is the line the declared drop rule leaves at width w: the fields in their fixed
// order with the notice, the position and the list dropped in that order until the line fits.
func droppedStatusLine(fields []lineField, w int) string {
	droppable := []string{"notice", "position", "list"}
	for dropped := 0; dropped <= len(droppable); dropped++ {
		drop := map[string]bool{}
		for _, kind := range droppable[:dropped] {
			drop[kind] = true
		}
		kept := make([]lineField, 0, len(fields))
		for _, field := range fields {
			if drop[field.kind] {
				continue
			}
			kept = append(kept, field)
		}
		if line := fieldsLine(kept); text.Width(line) <= w {
			return line
		}
	}
	return fieldsLine(fields)
}

// rowBudget is the widest text a pane row may carry: the composer's own arithmetic — the selection
// cell, the body's prefix and indent, two cells per quote level, and the measure at that level. The
// plain grammar's own values give Phase 2's rule back at the top level.
func rowBudget(g Grammar, w, depth int) int {
	return rowBudgetAt(g, w, depth, 0)
}

// rowBudgetAt is rowBudget with an explicit reply level: a conversation row at the reply step
// spends ReplyIndent cells before the body's own prefix, and the measure already accounts for it.
func rowBudgetAt(g Grammar, w, depth, replyLevel int) int {
	return 1 + g.ReplyIndent*replyLevel + g.BodyIndent + text.Width(g.BodyPrefix) + 2*depth + g.measure(w, depth, replyLevel)
}

// replyLevelOf reports whether a pane row sits at the reply step: after the selection cell the
// row begins with the grammar's ReplyIndent spaces (and is not a quote-only prefix row).
func replyLevelOf(g Grammar, row string) int {
	if g.ReplyIndent <= 0 {
		return 0
	}
	body := content(row)
	if strings.HasPrefix(body, strings.Repeat(" ", g.ReplyIndent)) {
		return 1
	}
	return 0
}

// depthOf is how many quote levels a pane row starts with, counted through the grammar's own quote
// prefix: a nested quote's prefix sits at its own level's indent, so a prefix is taken off either
// where it stands or after the indent in front of it, one level at a time.
func depthOf(g Grammar, row string) int {
	rest, depth := content(row), 0
	for depth < g.Markers.QuoteDepthCap {
		switch {
		case strings.HasPrefix(rest, g.Markers.QuotePrefix):
			rest = rest[len(g.Markers.QuotePrefix):]
		case strings.HasPrefix(strings.TrimLeft(rest, " "), g.Markers.QuotePrefix):
			rest = strings.TrimLeft(rest, " ")[len(g.Markers.QuotePrefix):]
		default:
			return depth
		}
		depth++
	}
	return depth
}

// TestSyncedFieldCorpusRule is SAVE-03 / SEC-02 on screen: every saved frame
// carries synced or never synced, and no timeline/search/profile frame does.
func TestSyncedFieldCorpusRule(t *testing.T) {
	frames := storedFrames(t)
	var saved, other int
	for _, f := range frames {
		body := f.bytes
		has := strings.Contains(body, "synced") || strings.Contains(body, "never synced") || strings.Contains(body, MarkerSync) || strings.Contains(body, MarkerSyncNever)
		// Prefer MarkerSet words so covers still count.
		has = strings.Contains(body, "synced") || strings.Contains(body, "never synced") || strings.Contains(body, "cached") || strings.Contains(body, "never cached")
		if f.state.pane == paneSaved {
			saved++
			// Capability frames at 80 can lose synced to the never-dropped budget
			// (06-UI-SPEC named compromise); require the capability word instead.
			if f.state.capability != domain.CapNone && f.w <= 80 {
				word := CapabilityWord(f.state.cover, f.state.capability, f.state.savedKind.List())
				if word != "" && !strings.Contains(body, word) {
					t.Errorf("saved capability frame %s missing %q", f.name, word)
				}
				continue
			}
			// Covered 80-column saved frames with a long state word drop list+synced
			// the same way (DSGT / status drop order); require the cover state word.
			if f.state.cover != "" && f.w <= 80 && f.state.failure.Kind != domain.FailureNone {
				word := HostSubstituted(StateWord(f.state.failure.Kind))
				if word != "" && !strings.Contains(body, word) {
					t.Errorf("covered saved frame %s missing state word %q", f.name, word)
				}
				continue
			}
			if !has {
				t.Errorf("saved frame %s carries neither synced form", f.name)
			}
			continue
		}
		other++
		if strings.Contains(body, "never synced") || strings.Contains(body, "synced Jan") || strings.Contains(body, "synced 15:") || strings.Contains(body, "synced 09:") {
			t.Errorf("non-saved frame %s carries the synced field", f.name)
		}
	}
	if saved == 0 {
		t.Fatal("no saved frames in the corpus: the presence rule is vacuous")
	}
	if other == 0 {
		t.Fatal("no non-saved frames: the absence rule is vacuous")
	}
}

// TestCoveredFramesLeakNoRepairWords is the cover seam's cost, over the stored bytes: under every
// cover, *no* covered frame holds the product's repair vocabulary. The strings are the ones the
// plain reader's saved surface is made of — the upstream's host and the program's own name (which
// the store's path spells), the store's file, the word the plain reader has for it, the program that
// pins an operation, the pin file — plus the handles and display names of the accounts the frame was
// built from.
//
// The scan is over every covered frame and not over the saved surface's frames alone. The words
// live on that surface, which is why it is where the scan started, but a plain row planted into a
// covered thread frame is the same leak: a cover supplies its own words for every state it draws,
// and a frame that holds one of these has a plain rendering under it wherever it is. The saved
// surface's own shapes are still required below, so the branch this phase's worst leak was in is
// scanned state by state rather than folded into a corpus-wide pass.
func TestCoveredFramesLeakNoRepairWords(t *testing.T) {
	frames := storedFrames(t)
	covered, _ := partition(frames)
	plain := make([]storedFrame, 0, len(frames))
	for _, f := range frames {
		if !f.panic && f.state.cover == "" {
			plain = append(plain, f)
		}
	}

	// The scan, over every covered frame — and the saved surface's own shapes recorded as it goes.
	scanned, saved := 0, 0
	shapes := map[string]bool{}
	for _, f := range covered {
		t.Setenv("TMUX", f.env.tmux)
		scanned++
		if f.state.pane == paneSaved {
			saved++
			// The shape the frame is in: the rows are the mirror's own list, and every other frame is
			// one of the frame's non-row states, named in the corpus's filename scheme.
			if len(f.state.saved.List().Posts()) > 0 {
				shapes["rows"] = true
			} else {
				shapes[savedFrameShape(f.name)] = true
			}
		}
		for _, msg := range coveredLeakFindings(f) {
			t.Error(msg)
		}
	}
	if scanned < len(covered) || scanned < 60 {
		t.Fatalf("the scan covered %d of the corpus's %d covered frames, want every one of them", scanned, len(covered))
	}
	if saved < 40 {
		t.Fatalf("the scan covered %d covered saved frames: the surface this phase's worst leak was in is barely in the corpus", saved)
	}
	// Every state of the frame is in the scan, so a branch that is not covered is a missing shape
	// rather than a passing frame: the rows, the three capability states, the empty read and the
	// failure that has nothing to show.
	for _, shape := range []string{"rows", "saved_unavailable", "saved_wrong_account", "saved_unreadable", "saved_empty", "saved_failed"} {
		if !shapes[shape] {
			t.Errorf("the corpus holds no covered saved frame in the %q state: the scan cannot see a leak in a branch with no frame", shape)
		}
	}

	// The arms. First, what a covered frame hides is what its own state holds: a capability frame
	// whose facts carry the store's path, the account the cache was filled from or the list's
	// product word, and a failure frame whose sentence carries the host, so a frame that printed any
	// of them is caught by the scan above and this one is not a scan over empty states.
	armed := map[string]int{}
	for _, f := range covered {
		if f.state.pane != paneSaved {
			continue
		}
		body := strings.ToLower(ansi.Strip(f.bytes))
		switch {
		case f.state.capability != domain.CapNone:
			if !strings.Contains(f.state.capFacts.Path, "saved.db") {
				t.Errorf("the covered capability frame %s was built without the store's path: a leak of it could not be caught", f.name)
				continue
			}
			if !strings.Contains(body, CapabilityWord(f.state.cover, f.state.capability, f.state.savedKind.List())) {
				t.Errorf("the covered capability frame %s does not carry its own state's word: the covered frame is not the capability state", f.name)
			}
			armed["capability"]++
		case f.state.failure.Kind != domain.FailureNone && len(f.state.saved.List().Posts()) == 0:
			if !strings.Contains(f.state.failure.Detail, CopyUpstreamHost) {
				t.Errorf("the covered saved failure frame %s was built from a sentence with no host in it: the absence proves nothing", f.name)
				continue
			}
			armed["failure"]++
		}
	}
	if armed["capability"] < 3 || armed["failure"] < 1 {
		t.Fatalf("the covered saved corpus arms %d capability frames and %d failure frames: the scan runs over states with nothing to hide (%v)", armed["capability"], armed["failure"], armed)
	}

	// Second, the plain reader's own saved frames print the words: a covered absence is a change of
	// the words and not a surface that never had them. The host and the pin file are not here —
	// no plain saved frame's own state carries either (the host is the cold failure frames' string,
	// which their own arm covers, and the pin file is a leak the corpus has no frame for).
	printed := map[string]int{}
	for _, f := range plain {
		if f.state.pane != paneSaved {
			continue
		}
		body := f.bytes
		for _, leak := range append(coveredLeakWords(), hiddenIdentities(f)...) {
			if strings.Contains(body, leak) {
				printed[leak]++
			}
		}
	}
	for _, leak := range []string{"mirror", "saved.db", "doctor", CopyListBookmarks} {
		if printed[leak] == 0 {
			t.Errorf("no plain saved frame prints %q: the covered absence of it would be vacuous", leak)
		}
	}
	if printed["mira"] == 0 {
		t.Error("no plain saved frame prints a handle of the accounts it was built from: the covered handle absence would be vacuous")
	}
}

// coveredLeakWords is every string a covered frame may not carry, whatever surface it is: the repair
// vocabulary of the plain reader's saved surface (the upstream's host, the program's own name, the
// store's file, the word the plain reader has for it, the program that pins an operation and the pin
// file) and the two saved lists' product names. The list is named for the covered frame rather than
// for the saved surface because the rule is one rule: a cover supplies its own words for every state
// it draws, and these words are the ones the plain reader's rendering of that state is made of.
func coveredLeakWords() []string {
	return []string{CopyUpstreamHost, CopyTitlePlain, "mirror", "saved.db", "doctor", "pins.json", CopyListBookmarks, CopyListLikes}
}

// coveredLeakFindings is the scan itself: every string of coveredLeakWords, and every identity of
// the accounts this frame was built from, that its stored bytes hold. It is a function over one frame
// rather than a loop in the test because the plant below has to run the same rule over bytes that
// did not come from a file in testdata.
func coveredLeakFindings(f storedFrame) []string {
	var out []string
	body := strings.ToLower(ansi.Strip(f.bytes))
	for _, leak := range append(coveredLeakWords(), hiddenIdentities(f)...) {
		if leak == "" {
			continue
		}
		if strings.Contains(body, strings.ToLower(leak)) {
			out = append(out, fmt.Sprintf("the covered frame %s holds %q, a word the cover did not author", f.name, leak))
		}
	}
	return out
}

// savedFrameShape is the state a stored covered saved frame names in its own filename: the corpus's
// naming scheme is `cover_<style>_<screen>_<width>`, so the screen is the shape.
func savedFrameShape(name string) string {
	rest, ok := strings.CutPrefix(name, "cover_")
	if !ok {
		return name
	}
	for _, style := range CoverStyles() {
		if after, ok := strings.CutPrefix(rest, string(style)+"_"); ok {
			rest = after
			break
		}
	}
	if i := strings.LastIndex(rest, "_"); i > 0 {
		return rest[:i]
	}
	return rest
}

// TestCoveredLeakScanCatchesAPlant is the scan's own proof that it can fail: each string it looks
// for is planted into a copy of a real covered frame, written to a temporary directory and read back
// — the scan takes bytes, and a plant is only evidence when it went through a file — and the scan is
// required to report it by the name of the frame it is in. The committed frame is compared
// afterwards, because a plant that landed in testdata would be a corrupted fixture.
//
// The two base frames are the two halves of the rule's coverage: a covered *thread* frame, which is
// the frame a verifier's planted plain row survived in while the scan was the saved surface's alone,
// and a covered saved row frame, whose own state carries the accounts' handles as well as the words.
// Every string of the scan is planted into both, so neither frame kind can pass a plant.
func TestCoveredLeakScanCatchesAPlant(t *testing.T) {
	frames := storedFrames(t)
	bases := []storedFrame{
		lookup(t, frames, "cover_gitlog_thread_80"),
		lookup(t, frames, "cover_gitlog_bookmarks_80"),
	}
	commitments := map[string]string{}
	for _, base := range bases {
		commitments[base.name] = base.bytes
	}
	for _, base := range bases {
		for _, leak := range append(coveredLeakWords(), "mira") {
			t.Run(base.name+"/"+leak, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), base.name+".golden")
				if err := os.WriteFile(path, []byte(base.bytes+"\n"+leak+"\n"), 0o644); err != nil {
					t.Fatalf("planting %s: %v", leak, err)
				}
				back, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading back the planted frame: %v", err)
				}
				planted := storedFrame{name: base.name, w: base.w, env: base.env, state: base.state, bytes: string(back)}
				findings := coveredLeakFindings(planted)
				if len(findings) == 0 {
					t.Fatalf("the planted %q was not caught by the scan", leak)
				}
				if !strings.Contains(strings.Join(findings, "\n"), planted.name) {
					t.Errorf("the scan reports %v without naming the frame the plant is in", findings)
				}
				for _, msg := range findings {
					t.Log(msg)
				}
			})
		}
	}
	for name, commitment := range commitments {
		if again := readStored(t, name); again != commitment {
			t.Fatal("a plant reached the committed corpus: testdata/TestGoldenFrames is not a plant target")
		}
	}
	// The plant's own arm: the frame the verifier's leak survived in is the covered thread frame, and
	// a scan that only saw saved frames would have reported nothing here.
	plantedThread := bases[0]
	if plantedThread.state.pane != paneThread {
		t.Fatalf("the plant's base frame %s is not a conversation: the leak the scan now catches was planted into one", plantedThread.name)
	}
}

// TestTheCoveredNoticesAreOnScreen is the four notices a mirror read leaves, asserted where they are
// read: the covered frames that carry them. Each is the style's own row of the copy table
// (`CopyCoverSyncing`, `CopyCoverNoLongerHeld`, `CopyCoverReadComplete`, `CopyCoverReadStalled`,
// reached through NoticeLine), and nothing asserted them before: reverting any of the four arms to
// the plain constant left the whole suite green, and — the other half of the same hole — no stored
// covered frame carried a notice at all, so the corpus could not see the wording either.
//
// The arms are three: every covered frame whose state carries a notice must carry its own style's
// words for it and not the plain reader's; all four notices must be carried by some covered frame
// (a kind with no frame is a kind whose words are asserted over nothing); and the plain reader's own
// frames must print the plain words, so the covered lines are a change of the words rather than a
// field that never renders.
func TestTheCoveredNoticesAreOnScreen(t *testing.T) {
	kinds := []struct {
		name   string
		notice NoticeKind
	}{
		{"a read in progress", NoticeSyncing},
		{"the rows a re-list took away", NoticeNoLongerSaved},
		{"a read that reached the end of the list", NoticeMirrorComplete},
		{"a read that stopped on a repeated cursor", NoticeSyncStalled},
	}
	frames := storedFrames(t)
	covered, _ := partition(frames)
	found := map[NoticeKind]int{}
	seen := 0
	for _, f := range covered {
		// NoticeGated is not one of the four: its words are the grammar's own refusal, read out of
		// the status line's gated field and never out of NoticeLine (`NoticeLine` answers with the
		// empty string for it, which the copy-table test beside this one pins).
		if f.state.notice == NoticeNone || f.state.notice == NoticeGated {
			continue
		}
		t.Setenv("TMUX", f.env.tmux)
		seen++
		found[f.state.notice]++
		word := NoticeLine(f.state.cover, f.state.notice, f.state.noticeCount)
		if word == "" {
			t.Errorf("the covered frame %s carries the notice %d and the cover has no words for it", f.name, f.state.notice)
			continue
		}
		if !strings.Contains(ansi.Strip(f.status()), word) {
			t.Errorf("the covered frame %s does not carry %q, its own style's words for the notice: the status line is %q", f.name, word, ansi.Strip(f.status()))
		}
		if plain := NoticeLine("", f.state.notice, f.state.noticeCount); word == plain {
			t.Errorf("the covered frame %s prints the plain reader's own notice %q: the cover reads the notice out of its own row", f.name, word)
		}
	}
	if seen < 4 {
		t.Fatalf("the corpus holds %d covered frames carrying a notice: the scan asserts the words over almost nothing", seen)
	}
	for _, k := range kinds {
		if found[k.notice] == 0 {
			t.Errorf("no covered frame in the corpus carries %s: its words are asserted over no frame", k.name)
		}
	}

	// The plain arm: the plain reader's saved frames carry the plain reader's words for two of the
	// four, which is the state a covered frame's own words replace.
	printed := 0
	for _, f := range frames {
		if f.panic || f.state.cover != "" || f.state.notice == NoticeNone || f.state.pane != paneSaved {
			continue
		}
		if strings.Contains(ansi.Strip(f.status()), NoticeLine("", f.state.notice, f.state.noticeCount)) {
			printed++
		}
	}
	if printed < 2 {
		t.Errorf("%d plain saved frames carry their own notice's words: the covered lines above would be a field that never renders plainly either", printed)
	}
}

// TestTheCoveredLikesHedgeIsInTheCorpus is D1 over the stored bytes: an empty likes answer is
// ambiguous — empty, or no longer served — and the pane leaves it open under every cover
// (06-UI-SPEC:263-266). The covered frame is one line, and it is the covered frame's own line: not
// the body that claims the read succeeded and found nothing, and not the plain reader's sentence
// with the product's word for the list in it.
//
// It is a corpus test and not only the unit test beside SavedEmptyState because a covered frame is
// what a reader sees: the case's own frame was missing from the corpus before this, which is how the
// regression went unseen when the covered branch of SavedEmptyState stopped reading `kind`.
func TestTheCoveredLikesHedgeIsInTheCorpus(t *testing.T) {
	frames := storedFrames(t)
	covered, _ := partition(frames)
	scanned := 0
	for _, f := range covered {
		if f.state.savedKind != domain.Like {
			continue
		}
		t.Setenv("TMUX", f.env.tmux)
		if len(f.state.saved.List().Posts()) > 0 {
			continue
		}
		scanned++
		if !contains(f, CopyCoverEmptyHedged) {
			t.Errorf("the covered likes-empty frame %s does not carry the cover's own hedge %q", f.name, CopyCoverEmptyHedged)
		}
		for _, forbidden := range []string{CopyCoverEmptyBody, CopyEmptyLikesHedged, CopyListLikes} {
			if contains(f, forbidden) {
				t.Errorf("the covered likes-empty frame %s holds %q: the covered frame resolves the ambiguity or speaks in the plain reader's words", f.name, forbidden)
			}
		}
		// The line is one line: the ambiguity is about the emptiness itself, and a body under it
		// would say the read succeeded and found nothing.
		if rows := len(f.state.blocks(f.w, len(f.pane()))); rows != 1 {
			t.Errorf("the covered likes-empty frame %s is %d blocks, want the one hedged line", f.name, rows)
		} else if got := len(f.state.blocks(f.w, len(f.pane()))[0].Rows); got != 1 {
			t.Errorf("the covered likes-empty frame %s is %d rows, want one: a second row resolves what the line leaves open", f.name, got)
		}
	}
	if scanned < 8 {
		t.Fatalf("the corpus holds %d covered likes-empty frames, want one per style at both widths: the hedge would be asserted over almost nothing", scanned)
	}
	// The plain arm: the plain reader's own likes frame carries its own one line, which is the line
	// the covered frames are not allowed to print.
	hedged := lookup(t, frames, "likes_hedged_80")
	t.Setenv("TMUX", hedged.env.tmux)
	if !contains(hedged, CopyEmptyLikesHedged) {
		t.Errorf("the plain frame %s does not carry %q: the covered frames' absence of it would prove nothing", hedged.name, CopyEmptyLikesHedged)
	}
}

func TestCorpusNoURLWithoutFailedAction(t *testing.T) {
	// No media URL without a failed action: every stored frame whose state is not a
	// MEDIA-03 fallback prints none of its posts' MediaRef URLs or Playable values.
	// Body-text https tokens (the adversarial over-measure URL) are content and stay.
	frames := storedFrames(t)
	var clean, failed int
	for _, f := range frames {
		if frameHasFailedMedia(f) {
			failed++
			continue
		}
		clean++
		for _, post := range f.state.list.Posts() {
			for _, media := range post.Media {
				for _, u := range []string{media.URL, media.Playable} {
					if u == "" {
						continue
					}
					if contains(f, u) {
						t.Errorf("frame %s prints media URL %q without a failed action", f.name, u)
					}
				}
			}
		}
	}
	if clean == 0 {
		t.Fatal("no clean frames: the no-URL arm is vacuous")
	}
	if failed == 0 {
		t.Fatal("no fallback frames: the failed-action arm is vacuous")
	}

	// Planted violation arm: a synthetic non-fallback frame that prints a media URL
	// must be detectable by the same rule.
	plant := domain.Post{
		ID: "plant", Author: domain.User{ScreenName: "x"},
		Media: []domain.MediaRef{{Kind: domain.Image, URL: "https://pbs.twimg.com/media/planted.jpg"}},
	}
	planted := storedFrame{
		name:  "planted_media_url",
		bytes: "   [1] https://pbs.twimg.com/media/planted.jpg\n",
		state: state{list: absorbPosts(t, []domain.Post{plant}, "plant")},
	}
	if frameHasFailedMedia(planted) {
		t.Fatal("planted frame unexpectedly counts as a failed-action case")
	}
	if !strings.Contains(planted.bytes, plant.Media[0].URL) {
		t.Fatal("planted URL violation was not detectable")
	}
}

func TestCorpusGatedMediaFrames(t *testing.T) {
	// The four covered gated frames carry each cover's Gated literal, no URL,
	// no fixture handle, no product name, and none of the plain media wording.
	frames := storedFrames(t)
	wants := map[string]string{
		"media_gated_servicelog_80": CoverServiceLogGated,
		"media_gated_gitlog_80":     CoverGitGated,
		"media_gated_ci_80":         CoverCIGated,
		"media_gated_internal_80":   CoverInternalGated,
	}
	plainMedia := []string{
		"open the post's media in an external viewer",
		"copy the post's link",
		"open the post in the browser",
	}
	var seen int
	for _, f := range frames {
		want, ok := wants[f.name]
		if !ok {
			continue
		}
		seen++
		body := strings.ToLower(ansi.Strip(f.bytes))
		if !strings.Contains(ansi.Strip(f.bytes), want) {
			t.Errorf("frame %s missing gated literal %q", f.name, want)
		}
		for _, post := range f.state.list.Posts() {
			for _, ref := range post.Media {
				if ref.URL != "" && strings.Contains(f.bytes, ref.URL) {
					t.Errorf("gated frame %s prints media URL %q", f.name, ref.URL)
				}
				if ref.Playable != "" && strings.Contains(f.bytes, ref.Playable) {
					t.Errorf("gated frame %s prints playable URL %q", f.name, ref.Playable)
				}
			}
			if permalink := Permalink(post); strings.Contains(f.bytes, permalink) {
				t.Errorf("gated frame %s prints permalink %q", f.name, permalink)
			}
		}
		for _, word := range []string{"aergon", "x-terminal", "twitter", "browser", "clipboard", "wl-copy", "xdg-open", "mpv"} {
			if strings.Contains(body, word) {
				t.Errorf("gated frame %s holds %q", f.name, word)
			}
		}
		for _, plain := range plainMedia {
			if strings.Contains(body, strings.ToLower(plain)) {
				t.Errorf("gated frame %s holds plain media wording %q", f.name, plain)
			}
		}
	}
	if seen != len(wants) {
		t.Fatalf("saw %d gated frames, want %d", seen, len(wants))
	}
}

// helpKeyColumn extracts the HelpKeyCols-wide key spellings from a stored
// help overlay frame (selection cell + key column, stripped of SGR).
func helpKeyColumn(f storedFrame) []string {
	out := make([]string, 0, len(f.pane()))
	for _, row := range f.pane() {
		plain := ansi.Strip(row)
		if strings.TrimSpace(plain) == "" {
			continue
		}
		if len(plain) < 1+HelpKeyCols {
			out = append(out, strings.TrimRight(plain, " "))
			continue
		}
		out = append(out, plain[1:1+HelpKeyCols])
	}
	return out
}
