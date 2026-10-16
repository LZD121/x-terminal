package text

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// a4Widths is the width set the agreement properties run over: every corpus entry is measured,
// filled, wrapped and cut at each of these widths, in both environments.
var a4Widths = []int{1, 2, 3, 5, 8, 57, 77, 96}

// corpusEntry is one named adversarial string. The name is what a failure prints, so a failure
// says which string broke the agreement.
type corpusEntry struct {
	name string
	s    string
}

// wantCorpusLen is the size of the corpus written as a literal, so that deleting an entry fails
// the suite instead of quietly weakening it: the UI-SPEC's adversarial list — a 15-cell handle, a
// 3-cell handle, a CJK run, a single emoji, the ZWJ family, a URL past 200 cells, a 10 000-cell
// body, the empty string, a run of spaces, a 57-cell line, a 56-cell line — plus one entry per
// width in a4Widths for a wide cluster sitting exactly on that boundary, plus the four the width
// model's own edges turn up: a string ending in a Prepend character, a token that measures nothing
// at all, and escape bytes at and after a cut (terminated and not): 11 + 8 + 4.
const wantCorpusLen = 23

// corpusMaxCluster is the widest cluster the corpus holds: the ZWJ family's six code-point cells
// under tmux. At a measure at least this wide no cluster has to be left out, which is where the
// no-loss property can be asked for.
const corpusMaxCluster = 6

// corpus returns the adversarial set described by wantCorpusLen.
func corpus() []corpusEntry {
	entries := []corpusEntry{
		{name: "handle of 15 cells", s: "longhandle_name"},
		{name: "handle of 3 cells", s: "lzd"},
		{name: "CJK run", s: "日本語のテキスト"},
		{name: "single emoji", s: "\U0001F600"},
		{name: "ZWJ family", s: zwjFamily},
		{name: "URL past 200 cells", s: "https://example.com/" + strings.Repeat("x", 200)},
		{name: "body of 10 000 cells", s: strings.Repeat("the pane never lies about what it is showing; the terminal does. ", 200)},
		{name: "empty string", s: ""},
		{name: "run of spaces", s: "     "},
		{name: "line of exactly 57 cells", s: "the pane never lies about what it is showing, not one bit"},
		{name: "line of 56 cells", s: "the pane never lies about what it is showing, never once"},
		// The edges the width model's own properties turn up. A UAX#29 Prepend character absorbs the
		// space that follows it, so a field padded after one is a cell short unless the fill is
		// measured; a token of zero-width characters is content a wrap still has to carry; and escape
		// bytes at or after a cut are a cut that is not a prefix, or a field no fill can close.
		{name: "a Prepend character at the end", s: "a\u06DD"},
		{name: "a zero-width token", s: "\u0301"},
		{name: "an escape sequence after the cut", s: "abcdef\x1b[31m"},
		{name: "an unterminated escape sequence", s: "ab\x1b["},
	}
	for _, w := range a4Widths {
		entries = append(entries, corpusEntry{
			name: "wide cluster on the " + strconv.Itoa(w) + "-cell boundary",
			s:    strings.Repeat("a", w-1) + "日" + "bb",
		})
	}
	return entries
}

// TestCorpusIsWhatItClaims holds the corpus to the widths and shapes its names claim. A corpus
// entry that is not what its name says is an agreement suite about nothing.
func TestCorpusIsWhatItClaims(t *testing.T) {
	byName := make(map[string]string, wantCorpusLen)
	for _, c := range corpus() {
		byName[c.name] = c.s
	}
	cells := map[string]int{
		"handle of 15 cells":       15,
		"handle of 3 cells":        3,
		"CJK run":                  16,
		"single emoji":             2,
		"run of spaces":            5,
		"empty string":             0,
		"line of exactly 57 cells": 57,
		"line of 56 cells":         56,
	}
	for name, want := range cells {
		s, ok := byName[name]
		if !ok {
			t.Errorf("the corpus has no entry named %q", name)
			continue
		}
		if got := Width(s); got != want {
			t.Errorf("Width(%q) = %d, want %d: %s", s, got, want, name)
		}
	}
	if s := byName["URL past 200 cells"]; Width(s) <= 200 || strings.Contains(s, " ") {
		t.Errorf("%q is %d cells and holds a space: want more than 200 cells and no space", s, Width(s))
	}
	if s := byName["body of 10 000 cells"]; Width(s) < 10000 {
		t.Errorf("the body is %d cells, want at least 10 000", Width(s))
	}
	if got := Width(byName["ZWJ family"]); got != 2 && got != 6 {
		t.Errorf("Width(%q) = %d, want 2 in a bare terminal and 6 under tmux", zwjFamily, got)
	}
	for _, w := range a4Widths {
		name := "wide cluster on the " + strconv.Itoa(w) + "-cell boundary"
		s, ok := byName[name]
		if !ok {
			t.Errorf("the corpus has no entry named %q", name)
			continue
		}
		ascii := strings.Repeat("a", w-1)
		if !strings.HasPrefix(s, ascii) || !strings.HasSuffix(s, "日bb") {
			t.Errorf("%q is not the boundary shape for w = %d", s, w)
			continue
		}
		if got := Width(ascii); got != w-1 {
			t.Errorf("the prefix of %q is %d cells, want %d: the wide cluster begins at cell %d",
				s, got, w-1, w)
		}
	}
	// The four edge entries are the shapes their names claim: a trailing Prepend character, a token
	// that measures nothing, and escape bytes at and after a cut (one terminated, one not). A
	// replacement that dropped any of those properties would weaken the corpus silently.
	if s := byName["a Prepend character at the end"]; !strings.HasSuffix(s, "\u06DD") || Width(s) == 0 {
		t.Errorf("%q does not end in a Prepend character that measures: the padding edge is gone", s)
	}
	if s := byName["a zero-width token"]; s == "" || Width(s) != 0 || strings.ContainsAny(s, " ") {
		t.Errorf("%q is not a single token of zero width: the no-loss edge is gone", s)
	}
	if s := byName["an escape sequence after the cut"]; !strings.Contains(s, "\x1b[") || Width(s) != 6 {
		t.Errorf("%q is not six visible cells followed by an escape sequence: the cut edge is gone", s)
	}
	if s := byName["an unterminated escape sequence"]; !strings.HasSuffix(s, "\x1b[") || Width(s) != 2 {
		t.Errorf("%q does not end in an unterminated escape introducer: the unclosable field edge is gone", s)
	}
}

// TestA4Agreement is A4: the four functions agree with each other and with the environment. For
// every corpus entry and every width in a4Widths it asserts the properties the UI-SPEC fixes —
// Width(Pad(s, w)) == w; Width(Truncate(s, w)) <= w; every line of Wrap(s, w) has Width <= w;
// Truncate is a prefix of s cut at a cluster boundary; no line of Wrap ends with a space — and
// runs the whole table twice, once with $TMUX unset and once with it set, requiring both to
// hold. The properties are environment-independent; only the wrap points move.
func TestA4Agreement(t *testing.T) {
	entries := corpus()
	if len(entries) != wantCorpusLen {
		t.Fatalf("the corpus holds %d entries, want %d: a deleted case would weaken the suite silently",
			len(entries), wantCorpusLen)
	}
	names := make(map[string]bool, wantCorpusLen)
	for _, c := range entries {
		if names[c.name] {
			t.Errorf("the corpus repeats the case %q: two entries under one name hide one of them", c.name)
		}
		names[c.name] = true
	}
	for _, env := range envArms() {
		t.Run(env.name, func(t *testing.T) {
			t.Setenv("TMUX", env.tmux)
			exercised := make(map[string]bool, wantCorpusLen)
			for _, c := range entries {
				exercised[c.name] = true
				starts := clusterStarts(c.s)
				t.Run(c.name, func(t *testing.T) {
					for _, w := range a4Widths {
						assertAgreement(t, c.s, w, starts)
					}
				})
			}
			if len(exercised) != wantCorpusLen {
				t.Errorf("%d of %d corpus entries were exercised", len(exercised), wantCorpusLen)
			}
		})
	}
}

// assertAgreement is A4 for one string and one width. starts is the set of byte offsets in s
// where a cluster begins under the active method.
func assertAgreement(t *testing.T, s string, w int, starts map[int]bool) {
	t.Helper()
	cut := Truncate(s, w)
	if got := Width(cut); got > w {
		t.Errorf("Width(Truncate(s, %d)) = %d, wider than the measure (s = %q, Truncate = %q)", w, got, s, cut)
	}
	if !strings.HasPrefix(s, cut) {
		t.Errorf("Truncate(s, %d) = %q, which is not a prefix of s = %q", w, cut, s)
	}
	if !starts[len(cut)] {
		t.Errorf("Truncate(s, %d) = %q cuts s at byte %d, which is not a cluster boundary (s = %q)",
			w, cut, len(cut), s)
	}
	for _, line := range Wrap(s, w) {
		if got := Width(line); got > w {
			t.Errorf("a line of Wrap(s, %d) = %q is %d cells wide, wider than the measure (s = %q)",
				w, line, got, s)
		}
		if strings.HasSuffix(line, " ") {
			t.Errorf("a line of Wrap(s, %d) = %q ends with a space (s = %q)", w, line, s)
		}
	}
	// No loss, no duplication: where the measure can hold every cluster in the corpus, the
	// lines carry exactly the non-space characters of s, in order, and nothing else.
	if w >= corpusMaxCluster {
		if got, want := nonSpace(Wrap(s, w)), nonSpace([]string{s}); got != want {
			t.Errorf("Wrap(s, %d) does not carry the non-space characters of s in order: got %q, want %q (s = %q)",
				w, got, want, s)
		}
	}
	// Pad last: its exactness rests on Truncate's bound, and a Pad that panics on a broken
	// Truncate would hide the Truncate failure above rather than report it.
	padded := Pad(s, w)
	if got := Width(padded); got != w {
		t.Errorf("Width(Pad(s, %d)) = %d, want exactly %d: the field must be its column wide whatever s is (s = %q, Pad = %q)",
			w, got, w, s, padded)
	}
}

// clusterStarts returns the byte offsets in s where a cluster begins under the active method,
// plus len(s) for the whole string. A prefix of s is cut at a cluster boundary exactly when its
// byte length is one of these offsets: that is the check that catches a cut inside a cluster.
func clusterStarts(s string) map[int]bool {
	starts := make(map[int]bool, len(s)+1)
	m := method()
	for i := 0; i < len(s); {
		starts[i] = true
		cluster, _ := ansi.FirstGraphemeCluster(s[i:], m)
		i += len(cluster)
	}
	starts[len(s)] = true
	return starts
}

// nonSpace returns every byte of the lines that is not a space, joined: the greedy wrap collapses
// the whitespace that separated tokens and may not lose or duplicate anything else.
func nonSpace(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		for i := range line {
			if line[i] != ' ' {
				b.WriteByte(line[i])
			}
		}
	}
	return b.String()
}

// TestEnvironmentMovesTheMeasure is the arm that proves the dispatch is live rather than a
// comment: the same string measures differently, and wraps differently, in the two environments.
func TestEnvironmentMovesTheMeasure(t *testing.T) {
	t.Setenv("TMUX", "")
	bareCells := Width(zwjFamily)
	bareLines := Wrap(zwjFamily, 4)
	bareFirstCells := 0
	if len(bareLines) == 1 {
		bareFirstCells = Width(bareLines[0])
	}

	t.Setenv("TMUX", tmuxValue)
	tmuxCells := Width(zwjFamily)
	tmuxLines := Wrap(zwjFamily, 4)
	tmuxEights := Wrap(zwjFamily, 8)

	if bareCells != 2 || tmuxCells != 6 {
		t.Errorf("Width(%q) = %d with TMUX unset and %d with it set, want 2 and 6", zwjFamily, bareCells, tmuxCells)
	}
	if bareCells == tmuxCells {
		t.Errorf("Width(%q) = %d in both environments: the dispatch is not live", zwjFamily, bareCells)
	}
	if len(bareLines) != 1 || len(tmuxLines) != 0 {
		t.Errorf("Wrap(%q, 4) = %q with TMUX unset and %q with it set, want one line and no line: the family is two cells by grapheme and six by wcwidth, and six cells do not fit in four",
			zwjFamily, bareLines, tmuxLines)
	}
	if len(bareLines) == len(tmuxLines) {
		t.Errorf("Wrap(%q, 4) returns %d lines in both environments: the wrap points did not move", zwjFamily, len(bareLines))
	}
	if bareFirstCells != 2 {
		t.Errorf("with TMUX unset the family occupies %d cells of the first line, want 2", bareFirstCells)
	}
	// Where the measure is wide enough to hold the family, the tmux arm draws one line and that
	// line is six cells wide: the code points of the family, not the cluster.
	if len(tmuxEights) != 1 || Width(tmuxEights[0]) != 6 {
		t.Errorf("Wrap(%q, 8) with TMUX set = %q, want one line of 6 cells", zwjFamily, tmuxEights)
	}
}

// TestA4ArmsAreNotVacuous is the suite's proof that it would fail against a naive package: a
// byte- or rune-counting measure, a wrap that never breaks a long token and a grapheme-hardcoded
// truncate are each shown to be within reach of an assertion here, and rejected by it.
func TestA4ArmsAreNotVacuous(t *testing.T) {
	byName := make(map[string]string, wantCorpusLen)
	for _, c := range corpus() {
		byName[c.name] = c.s
	}
	url := byName["URL past 200 cells"]
	for _, env := range envArms() {
		t.Run(env.name, func(t *testing.T) {
			t.Setenv("TMUX", env.tmux)
			// A measure that counted bytes or runes cannot pass: it disagrees with Width on
			// both strings, in both environments.
			for _, s := range []string{"日本語のテキスト", zwjFamily} {
				if got := len(s); got == Width(s) {
					t.Errorf("len(%q) = %d and Width = %d: this case cannot tell a byte count from a measure", s, got, got)
				}
			}
			// A wrap that never breaks an over-long token cannot pass: the naive wrap is the
			// token itself, which is wider than every width in a4Widths, while Wrap keeps every
			// line within them.
			for _, w := range a4Widths {
				if linesWithin([]string{url}, w) {
					t.Errorf("the URL is %d cells and w = %d, so the whole token would pass the measure check: this arm proves nothing",
						Width(url), w)
				}
				for _, line := range Wrap(url, w) {
					if Width(line) > w {
						t.Errorf("a line of Wrap(url, %d) = %q is %d cells wide: the token was not broken", w, line, Width(line))
					}
				}
			}
			// A Truncate that handed its input back unchanged cannot pass: the long URL is a
			// strict prefix shorter than s, and Width says so.
			if got := Truncate(url, 8); got == url {
				t.Errorf("Truncate(%q, 8) returned its input whole", url)
			} else if len(got) >= len(url) {
				t.Errorf("Truncate(url, 8) = %q is not a strict prefix of %q", got, url)
			}
			// The corpus is a named table of wantCorpusLen entries and every one of them was
			// exercised: TestA4Agreement asserts the count and the exercise, and this arm fails
			// if the table it reads has shrunk.
			if len(byName) != wantCorpusLen {
				t.Errorf("the corpus holds %d named entries, want %d", len(byName), wantCorpusLen)
			}
		})
	}
}

// linesWithin reports whether every line fits in w cells.
func linesWithin(lines []string, w int) bool {
	for _, line := range lines {
		if Width(line) > w {
			return false
		}
	}
	return true
}
