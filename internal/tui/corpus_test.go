package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
)

const tuiGoldenDir = "testdata/TestGoldenFrames"

var sgrSeq = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func readGolden(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(tuiGoldenDir, name+".golden"))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(body)
}

// plantFrame writes body into a temp copy of the named frame and reads the copy
// back: the scanners take bytes, and a plant is only evidence when it went
// through a file, but it must never be the committed frame. Writing into
// testdata would bump a committed golden's mtime on every run and a crash
// between plant and restore would leave the fixture corrupted.
func plantFrame(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+".golden")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("planting %s: %v", name, err)
	}
	back, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back planted %s: %v", name, err)
	}
	return string(back)
}

func stripSGR(s string) string { return sgrSeq.ReplaceAllString(s, "") }

// storedTUIFrames reads every stored frame of this corpus by name: testdata/TestGoldenFrames is
// where golden.RequireEqual writes them, and this is the one place they are read.
func storedTUIFrames(t *testing.T) map[string]string {
	t.Helper()
	onDisk, err := os.ReadDir(tuiGoldenDir)
	if err != nil {
		t.Fatalf("reading golden dir: %v", err)
	}
	stored := map[string]string{}
	for _, e := range onDisk {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".golden") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".golden")
		stored[name] = readGolden(t, name)
	}
	return stored
}

// coveredTUIFrame is one stored frame of this corpus that a cover rendered, with the style read off
// its own name: the corpus's naming scheme is `<screen>_<style>_<width>`, so the style is the name's
// segment that is one of the four style ids. Nothing else about the name is a style id — `color` and
// the screens' own words are not — so the segment is what says a frame is covered.
type coveredTUIFrame struct {
	name  string
	style render.CoverStyle
	body  string
}

// coveredTUIFrames is every stored covered frame of this corpus, in the sorted order of the names,
// which is the order the failures below read in.
func coveredTUIFrames(stored map[string]string) []coveredTUIFrame {
	out := make([]coveredTUIFrame, 0, len(stored))
	for name, body := range stored {
		for _, style := range render.CoverStyles() {
			if !slices.Contains(strings.Split(name, "_"), string(style)) {
				continue
			}
			out = append(out, coveredTUIFrame{name: name, style: style, body: body})
			break
		}
	}
	slices.SortFunc(out, func(a, b coveredTUIFrame) int { return strings.Compare(a.name, b.name) })
	return out
}

// TestCorpusAudit asserts adversarial content, absences and properties over the STORED bytes.
func TestCorpusAudit(t *testing.T) {
	frames := goldenFrames(t)
	stored := storedTUIFrames(t)
	// Arms that keep the corpus from shrinking.
	if len(stored) < len(frames) {
		t.Fatalf("stored corpus has %d frames, table names %d", len(stored), len(frames))
	}
	for _, f := range frames {
		if _, ok := stored[f.name]; !ok {
			t.Errorf("missing stored frame %s", f.name)
		}
	}

	// DSGT-08: covered search/people/profile cross-product must be stored.
	for _, style := range render.CoverStyles() {
		for _, w := range []int{80, 120} {
			for _, prefix := range []string{"search_", "search_people_", "profile_"} {
				name := prefix + string(style) + "_" + strconv.Itoa(w)
				if _, ok := stored[name]; !ok {
					t.Errorf("missing covered frame %s", name)
				}
			}
		}
	}

	foundIn := func(needle, hint string, names ...string) {
		t.Helper()
		for _, name := range names {
			body, ok := stored[name]
			if !ok {
				continue
			}
			if strings.Contains(ansi.Strip(body), needle) || strings.Contains(body, needle) {
				return
			}
		}
		t.Errorf("adversarial case %q (%s) not found in %v", needle, hint, names)
	}

	// Adversarial content.
	foundIn("日本語", "CJK query", "search_80", "search_120", "search_200", "search_60")
	foundIn("🔍", "emoji query", "search_80", "search_120", "search_200", "search_60")
	foundIn(strings.Repeat("q", 20), "long query", "prompt_120")
	foundIn(render.MarkTruncation, "cut display name", "search_people_80", "search_people_120")
	foundIn("reads timelines", "bio", "profile_80", "profile_120", "profile_empty_80")
	foundIn("Bare", "bare account / no facts", "profile_bare_80")
	foundIn(render.MarkSelection, "selection on a row", "search_80", "search_people_80", "profile_80")
	foundIn("+", "has-more marker", "search_80", "search_people_80", "profile_80")

	// Stalled frame: notice, no `+` in the position field ideally — at least the frame exists.
	if _, ok := stored["search_stalled_80"]; !ok {
		t.Error("missing search_stalled_80")
	}
	if _, ok := stored["profile_scrolled_80"]; !ok {
		t.Error("missing profile_scrolled_80")
	}
	if _, ok := stored["search_failure_80"]; !ok {
		t.Error("missing search_failure_80")
	}

	// The four profile cases 05-03's must-have names, each asserted over the stored bytes of a frame
	// named in the failure message. They are the cases whose absence a byte-for-byte comparison cannot
	// see: a three-line bio that keeps only two lines, a repost without its marker, a body without its
	// quote prefix and a tombstone without its statement are all still perfectly good frames.
	for _, msg := range scanBioHeader("profile_bio_80", ansi.Strip(stored["profile_bio_80"])) {
		t.Error(msg)
	}
	for _, msg := range scanContent("profile_repost_80", ansi.Strip(stored["profile_repost_80"]),
		[]string{strings.Replace(render.MarkerRepost, "{handle}", repostHandle, 1)}) {
		t.Error(msg)
	}
	for _, msg := range scanContent("profile_quote_80", ansi.Strip(stored["profile_quote_80"]),
		[]string{render.MarkerQuotePrefix + "  " + quotedBody}) {
		t.Error(msg)
	}
	for _, msg := range scanContent("profile_tombstone_80", ansi.Strip(stored["profile_tombstone_80"]),
		[]string{render.MarkerUnavailable}) {
		t.Error(msg)
	}

	// Homogeneity, both directions. One direction was asserted from the start: a people frame carries
	// no post block (its rows are UserHead's, so a stamp or a body continuation under a row is a leak).
	// The other direction is the mirror, and it is the one this arm was missing: a posts frame carries
	// no user row. The shape test is the same one both ways — a user row is a handle in the identity
	// column and then a display name, with no stamp, which no post's identity row is — and it is
	// exercised first against the people frame that legitimately carries rows of that shape, so the
	// absence below is a check and not a scanner that matches nothing.
	control := userRows(ansi.Strip(stored["search_people_80"]))
	if len(control) < 4 {
		t.Errorf("the user-row shape test finds %d rows in search_people_80, which holds eight accounts: the homogeneity arm would be vacuous", len(control))
	}
	found := false
	for _, row := range control {
		if strings.Contains(row.text, "alice") {
			found = true
		}
	}
	if !found {
		t.Errorf("the user-row shape test misses search_people_80's first account row: %+v", control)
	}
	for name, body := range stored {
		if !postsFrame(name) {
			continue
		}
		for _, msg := range scanUserRowsInPostsFrame(name, ansi.Strip(body)) {
			t.Error(msg)
		}
	}

	// Absences over every stored frame.
	for name, body := range stored {
		plain := ansi.Strip(body)
		for _, msg := range scanForbiddenListAbsences(name, plain) {
			t.Error(msg)
		}
		// Homogeneity: a people frame must not look like a post block (byline stamp pattern),
		// and a search posts frame must not be only handle+name rows without body — soft check
		// via frame name prefixes.
		if strings.HasPrefix(name, "search_people") {
			if strings.Contains(plain, "Oct 14") && strings.Contains(plain, "  ") {
				// People rows have no stamps; a stamp means a post block leaked in.
				for _, line := range strings.Split(plain, "\n") {
					if strings.Contains(line, "Oct 14") {
						t.Errorf("%s people frame carries a post stamp line: %q", name, line)
					}
				}
			}
			assertNoBioInPeopleRows(t, name, plain)
		}
		if strings.HasPrefix(name, "search_") && !strings.HasPrefix(name, "search_people") {
			assertNoMatchTermAttribute(t, name, body)
		}
	}

	// A1-S: strip SGR from truecolor and mono of the same state → identical.
	for _, base := range []string{"search_color", "profile_color"} {
		tc := stripSGR(stored[base+"_truecolor"])
		mono := stripSGR(stored[base+"_mono"])
		if tc != mono {
			t.Errorf("A1-S failed for %s: truecolor and mono differ after SGR strip", base)
		}
	}

	// A2-S: colour frames keep the product/query echo in the status field.
	for _, name := range []string{"search_color_truecolor", "search_color_256", "search_color_16", "search_color_mono"} {
		body := stored[name]
		plain := ansi.Strip(body)
		rows := strings.Split(plain, "\n")
		if len(rows) == 0 {
			continue
		}
		status := rows[len(rows)-1]
		if !strings.Contains(status, "search:") {
			t.Errorf("A2-S: %s status does not name the search: %q", name, status)
		}
	}

	// A4 extended: wrap/pad/truncate properties for fixture strings at several widths.
	fixtures := []string{
		"Alice",
		"A display name so long it must be cut with the truncation mark in the last cell the row may occupy for the goldens",
		"reads timelines in a terminal",
		"日本語 🔍 timeline",
	}
	for _, s := range fixtures {
		for _, w := range []int{1, 2, 3, 5, 8, 55, 77, 96} {
			cut := text.Truncate(s, w)
			if text.Width(cut) > w {
				t.Errorf("A4: Truncate(%q, %d) width %d", s, w, text.Width(cut))
			}
		}
	}

	// Prompt caret column stays within 1..W — PromptRow returns the column.
	for _, q := range []string{"", "page cache", strings.Repeat("q", 40), "日本語"} {
		for _, w := range []int{40, 80, 120} {
			_, col := render.PromptRow(render.Plain, 0 /* Latest */, q, w)
			if col < 1 || col > w {
				t.Errorf("prompt caret column %d out of 1..%d for %q", col, w, q)
			}
		}
	}
}

// TestCorpusAuditFailsOnPlantedViolation: break once, watch, revert — the audit can fail.
// It plants known violations into a temp copy of a frame and runs the same forbidden
// scanners TestCorpusAudit uses, expecting each plant to fail with a named frame/row.
func TestCorpusAuditFailsOnPlantedViolation(t *testing.T) {
	committed := readGolden(t, "search_people_80")

	t.Run("likes in people frame", func(t *testing.T) {
		orig := readGolden(t, "search_people_80")
		planted := plantFrame(t, "search_people_80", orig+"\nlikes: 999 rank #1\n")

		errors := scanForbiddenListAbsences("search_people_80", ansi.Strip(planted))
		if len(errors) == 0 {
			t.Fatal("planted likes/rank violation was not caught by the absence scanner")
		}
		for _, msg := range errors {
			t.Log(msg)
		}
	})

	t.Run("bio in people row", func(t *testing.T) {
		orig := readGolden(t, "search_people_80")
		lines := strings.Split(orig, "\n")
		if len(lines) < 3 {
			t.Fatal("search_people_80 too short to plant a bio")
		}
		// Insert a bio-shaped continuation under the first account row.
		plant := append([]string{}, lines[:1]...)
		plant = append(plant, "  reads timelines in a terminal")
		plant = append(plant, lines[1:]...)
		body := plantFrame(t, "search_people_80", strings.Join(plant, "\n"))

		errors := scanPeopleBioAbsences("search_people_80", ansi.Strip(body))
		if len(errors) == 0 {
			t.Fatal("planted bio in people row was not caught by the bio absence scanner")
		}
		for _, msg := range errors {
			t.Log(msg)
		}
	})

	t.Run("SGR match attribute in search frame", func(t *testing.T) {
		plain := readGolden(t, "search_80")
		idx := strings.Index(plain, "timeline")
		if idx < 0 {
			t.Fatal("search_80 has no timeline match term to wrap")
		}
		planted := plantFrame(t, "search_80", plain[:idx]+"\x1b[1m"+"timeline"+"\x1b[0m"+plain[idx+len("timeline"):])

		errors := scanMatchTermAttributes("search_80", planted)
		if len(errors) == 0 {
			t.Fatal("planted SGR match attribute was not caught by the match-attribute scanner")
		}
		for _, msg := range errors {
			t.Log(msg)
		}
	})

	// The four restored profile cases, each planted by removing the case from a temp copy of the frame
	// the audit looks for it in: the audit must name the frame and the content it lost.
	t.Run("bio line lost", func(t *testing.T) {
		orig := readGolden(t, "profile_bio_80")
		// Blank the second bio line (row 2): the header still has three bio rows, and the sentence they
		// reassemble into no longer matches.
		lines := strings.Split(orig, "\n")
		if len(lines) < 4 {
			t.Fatal("profile_bio_80 is too short to plant a lost bio line")
		}
		lines[2] = strings.Repeat(" ", len(lines[2]))
		body := plantFrame(t, "profile_bio_80", strings.Join(lines, "\n"))
		errors := scanBioHeader("profile_bio_80", ansi.Strip(body))
		if len(errors) == 0 {
			t.Fatal("a blanked bio line was not caught by the header scanner")
		}
		for _, msg := range errors {
			t.Log(msg)
		}
	})
	t.Run("repost marker lost", func(t *testing.T) {
		marker := strings.Replace(render.MarkerRepost, "{handle}", repostHandle, 1)
		body := plantFrame(t, "profile_repost_80", blankRows(t, "profile_repost_80", readGolden(t, "profile_repost_80"), marker))
		errors := scanContent("profile_repost_80", ansi.Strip(body), []string{marker})
		if len(errors) == 0 {
			t.Fatal("a blanked repost marker was not caught by the content scanner")
		}
		for _, msg := range errors {
			t.Log(msg)
		}
	})
	t.Run("quote row lost", func(t *testing.T) {
		needle := render.MarkerQuotePrefix + "  " + quotedBody
		body := plantFrame(t, "profile_quote_80", blankRows(t, "profile_quote_80", readGolden(t, "profile_quote_80"), needle))
		errors := scanContent("profile_quote_80", ansi.Strip(body), []string{needle})
		if len(errors) == 0 {
			t.Fatal("a blanked quote row was not caught by the content scanner")
		}
		for _, msg := range errors {
			t.Log(msg)
		}
	})
	t.Run("tombstone lost", func(t *testing.T) {
		body := plantFrame(t, "profile_tombstone_80", blankRows(t, "profile_tombstone_80", readGolden(t, "profile_tombstone_80"), render.MarkerUnavailable))
		errors := scanContent("profile_tombstone_80", ansi.Strip(body), []string{render.MarkerUnavailable})
		if len(errors) == 0 {
			t.Fatal("a blanked tombstone statement was not caught by the content scanner")
		}
		for _, msg := range errors {
			t.Log(msg)
		}
	})
	t.Run("user row in a posts frame", func(t *testing.T) {
		// An account row planted where a posts frame's own rows are: the reverse-homogeneity direction.
		orig := readGolden(t, "search_80")
		lines := strings.Split(orig, "\n")
		if len(lines) < 3 {
			t.Fatal("search_80 is too short to plant a user row")
		}
		plant := append([]string{}, lines[:2]...)
		plant = append(plant, " alice            Alice"+strings.Repeat(" ", 40))
		plant = append(plant, lines[2:]...)
		body := plantFrame(t, "search_80", strings.Join(plant, "\n"))
		errors := scanUserRowsInPostsFrame("search_80", ansi.Strip(body))
		if len(errors) == 0 {
			t.Fatal("a user row planted into search_80 was not caught by the homogeneity scanner")
		}
		for _, msg := range errors {
			t.Log(msg)
		}
	})

	// The committed frame is what this suite compares against: every plant above
	// landed in a temp directory, so its bytes are exactly what they were.
	if got := readGolden(t, "search_people_80"); got != committed {
		t.Fatal("a plant reached the committed frame: internal/tui/testdata is not a plant target")
	}
}

// blankRows replaces the first row of body that carries needle with spaces of the same width: a plant
// that removes the content the audit looks for without changing the frame's row count or width.
func blankRows(t *testing.T, name, body, needle string) string {
	t.Helper()
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.Contains(ansi.Strip(line), needle) {
			lines[i] = strings.Repeat(" ", len(line))
			return strings.Join(lines, "\n")
		}
	}
	t.Fatalf("%s carries no row with %q to blank", name, needle)
	return ""
}

// scanBioHeader asserts the three-line bio case over a stored frame: the header's rows between its
// identity row and its facts row are the bio's, one row per wrapped line, and they reassemble into the
// sentence the frame was generated from — so a dropped, duplicated or re-ordered line fails by name
// rather than passing as "three rows with some words in them".
func scanBioHeader(name, plain string) []string {
	var out []string
	rows := strings.Split(plain, "\n")
	facts := -1
	for i, row := range rows {
		if strings.Contains(row, "followers:") {
			facts = i
			break
		}
	}
	if facts < 1 {
		return []string{fmt.Sprintf("%s: the pane's first rows carry no facts row, so the header's bio cannot be delimited", name)}
	}
	lines := make([]string, 0, facts-1)
	for _, row := range rows[1:facts] {
		lines = append(lines, strings.TrimSpace(row))
	}
	if got := strings.Join(lines, " "); got != threeLineBio {
		out = append(out, fmt.Sprintf("%s: the header's bio rows reassemble into %q, want %q", name, got, threeLineBio))
	}
	if len(lines) != 3 {
		out = append(out, fmt.Sprintf("%s: the header's bio occupies %d rows, and the case is a bio that wraps to three", name, len(lines)))
	}
	return out
}

// scanContent asserts that every needle is in a stored frame's bytes, naming the frame: the case is
// the content, and the frame is where the audit says it is.
func scanContent(name, plain string, needles []string) []string {
	var out []string
	for _, needle := range needles {
		if !strings.Contains(plain, needle) {
			out = append(out, fmt.Sprintf("%s does not carry %q", name, needle))
		}
	}
	return out
}

// postsFrame reports whether a stored frame is a posts surface: a results frame that is not the people
// product, or a profile frame. The people product is not it — its rows are account rows, which is the
// other half of the homogeneity rule — and neither are the saved surfaces, whose rows are the mirror's.
func postsFrame(name string) bool {
	switch {
	case strings.HasPrefix(name, "search_people"), strings.HasPrefix(name, "bookmarks_"), strings.HasPrefix(name, "likes_"):
		return false
	case strings.HasPrefix(name, "search_"), strings.HasPrefix(name, "profile_"):
		return true
	}
	return false
}

// userRow is one account-shaped pane row: the row's index in the pane and the row itself.
type userRow struct {
	line int
	text string
}

// userRows is the homogeneity rule's shape test, over one frame's rows: the rows that look like a
// people row. Such a row is UserHead's — the selection column, a handle in the identity column, two or
// more spaces, and then a display name — and its tail carries no stamp: every grammar this build ships
// prints a post's identity row with digits (a date, a build id, a sequence number) in the tail, and no
// people row carries one. The handle is word-shaped, which is what keeps the pane's own punctuation out
// of the test: a quote row's prefix (`> `, repeated once per level) is not a handle.
//
// The test is deliberately one shape for both directions: the audit asserts it finds people rows in the
// frame that legitimately carries them, so a scanner that matched nothing could not pass.
func userRows(plain string) []userRow {
	var out []userRow
	for i, line := range strings.Split(plain, "\n") {
		row := strings.TrimRight(line, " ")
		if len(row) < 2 || (row[0] != ' ' && row[0] != render.MarkSelection[0]) {
			continue
		}
		fields := strings.SplitN(row[1:], "  ", 2)
		if len(fields) != 2 {
			continue
		}
		handle, tail := fields[0], strings.TrimSpace(fields[1])
		if handle == "" || len(handle) > 15 || strings.ContainsAny(handle, " \t><|") {
			continue
		}
		if tail == "" || strings.ContainsAny(tail, "0123456789") {
			continue
		}
		out = append(out, userRow{line: i, text: row})
	}
	return out
}

// scanUserRowsInPostsFrame is the homogeneity arm's other direction: a posts frame that carries a user
// row. A posts frame composes two things above its first post — a profile's header and the internal
// grammar's column header — and both are the pane's first row (a scroll drops whole blocks, so such a
// row is either at the top or gone); any account-shaped row below the first row of a posts frame is a
// people row that leaked into a frame whose rows are posts.
func scanUserRowsInPostsFrame(name, plain string) []string {
	var out []string
	for _, row := range userRows(plain) {
		if row.line == 0 {
			continue
		}
		out = append(out, fmt.Sprintf("%s posts frame carries a user row: %q", name, strings.TrimSpace(row.text)))
	}
	return out
}

func scanForbiddenListAbsences(name, plain string) []string {
	var out []string
	low := strings.ToLower(plain)
	for _, bad := range []string{"likes", "reposts", "views", "rank #", "results:", "matches:"} {
		if strings.Contains(low, bad) {
			out = append(out, fmt.Sprintf("%s contains forbidden %q", name, bad))
		}
	}
	return out
}

// assertNoBioInPeopleRows fails when a people golden carries a bio-shaped continuation under an account head.
// People rows are exactly UserHead (handle+name); a profile bio line under a row is forbidden.
func assertNoBioInPeopleRows(t *testing.T, name, plain string) {
	t.Helper()
	for _, msg := range scanPeopleBioAbsences(name, plain) {
		t.Error(msg)
	}
}

func scanPeopleBioAbsences(name, plain string) []string {
	var out []string
	lines := strings.Split(plain, "\n")
	for i, line := range lines {
		trim := strings.TrimRight(line, " ")
		if trim == "" || strings.HasPrefix(strings.TrimLeft(trim, " "), "people:") {
			continue
		}
		// Skip blank continuation rows; look at the immediate next non-empty row after a head.
		if i+1 >= len(lines) {
			continue
		}
		nxt := lines[i+1]
		nxtTrim := strings.TrimRight(nxt, " ")
		if nxtTrim == "" {
			continue
		}
		// A bio/body continuation is indented by two or more spaces and is not an account head.
		if !strings.HasPrefix(nxt, "  ") {
			continue
		}
		inner := strings.TrimLeft(nxt, " ")
		if strings.HasPrefix(inner, render.MarkSelection) {
			continue
		}
		if looksLikePeopleAccountHead(inner) {
			continue
		}
		out = append(out, fmt.Sprintf("%s people frame row %d carries a bio/body continuation under an account row: %q", name, i+1, nxtTrim))
	}
	return out
}

func looksLikePeopleAccountHead(s string) bool {
	// UserHead: handle field then display name. Handles have no spaces.
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return false
	}
	handle := strings.TrimSuffix(fields[0], render.MarkTruncation)
	for _, r := range handle {
		if r == ' ' {
			return false
		}
	}
	// A bio sentence usually has many words; an account head is handle + optional name tokens
	// starting in the handle column. Treat "handle then short name" as a head only when the
	// first field looks handle-shaped (no spaces already) AND the line does not start mid-sentence
	// punctuation. For our goldens, account heads never start with two spaces, so this helper
	// only sees the trimmed inner of an indented line — which should never be a head.
	return false
}

// assertNoMatchTermAttribute fails when a search posts golden wraps a query match term in SGR.
func assertNoMatchTermAttribute(t *testing.T, name, body string) {
	t.Helper()
	for _, msg := range scanMatchTermAttributes(name, body) {
		t.Error(msg)
	}
}

func scanMatchTermAttributes(name, body string) []string {
	var out []string
	terms := []string{"timeline", "日本語", "page cache"}
	for _, term := range terms {
		idx := 0
		for {
			i := strings.Index(body[idx:], term)
			if i < 0 {
				break
			}
			abs := idx + i
			start := abs - 16
			if start < 0 {
				start = 0
			}
			prefix := body[start:abs]
			locs := sgrSeq.FindAllStringIndex(prefix, -1)
			if len(locs) > 0 {
				last := locs[len(locs)-1]
				between := prefix[last[1]:]
				if between == "" || strings.TrimSpace(between) == "" {
					out = append(out, fmt.Sprintf("%s search frame wraps match term %q in an SGR attribute near offset %d", name, term, abs))
				}
			}
			idx = abs + len(term)
		}
	}
	return out
}

// tuiLeakWords is every string a covered frame of this corpus may not carry: the repair vocabulary
// of the plain reader's saved surface (the upstream's host, the program's own name, the store's
// file, the word the plain reader has for it, the program that pins an operation and the pin file)
// and the two saved lists' product words. It is internal/render's covered-frame rule as this half of
// the corpus sees it (`corpus_test.go`'s coveredLeakWords there), spelled from the same exported
// constants because that list is a test's own and cannot be read from here.
func tuiLeakWords() []string {
	return []string{
		render.CopyUpstreamHost, render.CopyTitlePlain, "mirror", "saved.db", "doctor", "pins.json",
		render.CopyListBookmarks, render.CopyListLikes,
	}
}

// corpusAccounts is the accounts this corpus's frames were built from: the accounts the corpus's own
// source answers with — the home feed's authors, the people page's rows and the header of the profile
// the walk opens — as the identity strings a covered frame may not carry (their handles and display
// names) and as the values themselves, which is what the aliases the covered frames *do* carry are
// computed from. Both are read from a driven shell rather than written down here, so a fixture that
// changes changes the scan with it.
func corpusAccounts(t *testing.T) (identities []string, accounts []domain.User) {
	t.Helper()
	s := goldenShell(t, 80, goldenH)
	s.press("u")
	typeQuery(s, "people")
	s.press("enter")

	seenID := map[string]bool{}
	seenString := map[string]bool{}
	add := func(given ...domain.User) {
		for _, account := range given {
			if account.ScreenName == "" && account.Name == "" {
				continue
			}
			if key := account.ID + "\x00" + account.ScreenName; !seenID[key] {
				seenID[key] = true
				accounts = append(accounts, account)
			}
			for _, value := range []string{account.ScreenName, account.Name} {
				if value == "" || seenString[value] {
					continue
				}
				seenString[value] = true
				identities = append(identities, value)
			}
		}
	}
	writers := strings.Builder{}
	for _, f := range s.m.frames {
		for _, post := range f.List.Posts() {
			add(post.Author)
			if post.RepostedBy != nil {
				add(*post.RepostedBy)
			}
			for _, text := range []string{post.Text} {
				writers.WriteString(strings.ToLower(text))
				writers.WriteString("\n")
			}
			if post.Quote != nil {
				add(post.Quote.Author)
				writers.WriteString(strings.ToLower(post.Quote.Text))
				writers.WriteString("\n")
			}
		}
		if f.Kind == frameSearch && f.isPeople() {
			add(f.People.Walk().Rows()...)
		}
		if header, ok := f.Profile.Header(); ok {
			add(header.User)
			writers.WriteString(strings.ToLower(header.Bio))
			writers.WriteString("\n")
		}
	}
	// A covered frame may carry the *writers'* words: the feed's own text is content, and a covered
	// frame of the same screen prints it verbatim. An identity string that the corpus's own writer
	// text already holds as a word is therefore not a string a covered frame "leaked" — the fixture's
	// long-display-name account is called `Fifteen`, and one of its posts says "a fifteen-cell handle"
	// — so those strings are dropped from the scan's targets rather than reported as findings. The
	// rule stays the handles and the names of the accounts the frames were built from, and the
	// exclusion is stated here because it is a fact about this fixture's content.
	text := writers.String()
	kept := make([]string, 0, len(identities))
	for _, identity := range identities {
		if strings.Contains(text, strings.ToLower(identity)) {
			t.Logf("the identity %q is dropped from the scan's targets: the corpus's own writer text holds it as a word, and a covered frame may print the writers' words", identity)
			continue
		}
		kept = append(kept, identity)
	}
	return kept, accounts
}

// corpusAliasSlots is what a covered frame of one style prints for every account of the corpus: the
// four projections of the alias, each of which is a string only a cover prints. It is the presence
// half of the scan below — a frame that carried none of them would be a frame that was not rendered
// through the cover at all.
func corpusAliasSlots(accounts []domain.User) map[render.CoverStyle][]string {
	out := map[render.CoverStyle][]string{}
	for _, account := range accounts {
		for _, style := range render.CoverStyles() {
			alias := render.CoverAliasOf(style, account)
			for _, slot := range []string{alias.Handle, alias.Name, alias.Login, alias.Email} {
				if slot != "" {
					out[style] = append(out[style], slot)
				}
			}
		}
	}
	return out
}

// tuileakFindings is the scan over one covered frame's stored bytes: every string of tuiLeakWords and
// every identity of the corpus that the frame holds. It is a function over one frame's bytes rather
// than a loop in the test because the plant test below runs the same rule over bytes that came from a
// temporary file.
//
// The frame's own aliases are taken out of the body first. They are the words the cover authored, so
// they are the one thing a covered frame is *supposed* to hold — and one of them is a machine name
// whose first word is the plain reader's own word for the store (`mirror-merge-65`, from the frozen
// word lists), so a bare substring scan would report the cover's own vocabulary as a leak. What is
// scanned is therefore the frame with its style's identities masked: everything else the frame says.
func tuileakFindings(f coveredTUIFrame, identities, aliases []string) []string {
	var out []string
	body := strings.ToLower(ansi.Strip(f.body))
	for _, alias := range aliases {
		body = strings.ReplaceAll(body, strings.ToLower(alias), " ")
	}
	for _, leak := range append(tuiLeakWords(), identities...) {
		if leak == "" {
			continue
		}
		if strings.Contains(body, strings.ToLower(leak)) {
			out = append(out, fmt.Sprintf("the covered tui frame %s holds %q, a string the cover of %s did not author", f.name, leak, f.style))
		}
	}
	return out
}

// TestTheCoveredTUIFramesLeakNoPlainVocabulary is the covered-frame rule over this corpus's half of
// the stored frames — the half internal/render's audit cannot read. Until this test existed the only
// thing between a covered tui frame and a planted plain row was the golden byte comparison, which
// catches a plant only when the frame has been re-blessed with it: an independent verifier planted
// `saved.db doctor --pin mirror` into a re-blessed covered thread frame in the other corpus and both
// suites passed, and this half had no scanner at all.
//
// The arms are three: the corpus holds the DSGT-08 cross-product (a scan over one frame proves
// nothing); the plain siblings of the covered screens carry the accounts the covered ones may not; and
// every covered frame carries its own style's alias for some account, so the scan is not passing over
// frames that were rendered plainly and happen to hold no account at all.
func TestTheCoveredTUIFramesLeakNoPlainVocabulary(t *testing.T) {
	stored := storedTUIFrames(t)
	covered := coveredTUIFrames(stored)
	if len(covered) < 24 {
		t.Fatalf("the corpus holds %d covered frames, want the four styles' search, people and profile screens at both widths: the scan would run over a sample", len(covered))
	}
	identities, accounts := corpusAccounts(t)
	if len(identities) < 8 || len(accounts) < 4 {
		t.Fatalf("the corpus's source names %d accounts and %d identity strings: the absence of either would be vacuous", len(accounts), len(identities))
	}
	slots := corpusAliasSlots(accounts)

	// The arm: the plain screens of the same corpus carry the accounts the covered ones may not, so
	// the covered absence is a change of the words and not a corpus that never had them.
	plainPrinted := 0
	for _, name := range []string{"search_80", "search_people_80", "profile_80"} {
		body := stored[name]
		for _, account := range identities {
			if strings.Contains(body, account) {
				plainPrinted++
				break
			}
		}
	}
	if plainPrinted < 3 {
		t.Fatalf("%d of the corpus's plain screens print an account the covered screens are scanned for", plainPrinted)
	}

	scanCoveredTUIFrames(t, covered, identities, slots)
}

// TestTheCoveredTUIFrameLeakScanCatchesAPlant is the scan's own proof that it can fail: each string
// it looks for is planted into a copy of a real covered frame of this corpus, written to a temporary
// directory and read back — the scan takes bytes, and a plant is only evidence when it went through a
// file — and the scan is required to report it by the name of the frame it is in. The committed frame
// is compared afterwards, because a plant that landed in testdata would be a corrupted fixture.
//
// The base frame is a covered *people* frame: a plain row is exactly the shape of the leak this half
// of the corpus had no scanner for (an independent verifier re-blessed a covered frame with one and
// both suites passed).
func TestTheCoveredTUIFrameLeakScanCatchesAPlant(t *testing.T) {
	const name = "search_people_gitlog_80"
	committed := readGolden(t, name)
	identities, accounts := corpusAccounts(t)
	slots := corpusAliasSlots(accounts)[kindCheckStyle]
	base := coveredTUIFrame{name: name, style: kindCheckStyle, body: committed}

	for _, leak := range append(tuiLeakWords(), "alice") {
		t.Run(leak, func(t *testing.T) {
			planted := plantFrame(t, name, committed+"\n"+leak+"\n")
			findings := tuileakFindings(coveredTUIFrame{name: name, style: kindCheckStyle, body: planted}, identities, slots)
			if len(findings) == 0 {
				t.Fatalf("the planted %q was not caught by the scan", leak)
			}
			if !strings.Contains(strings.Join(findings, "\n"), name) {
				t.Errorf("the scan reports %v without naming the frame the plant is in", findings)
			}
			for _, msg := range findings {
				t.Log(msg)
			}
		})
	}

	// The scan's other half, on the same base: the frame is one the scan reads at all, and its own
	// aliases are what the mask takes out (a scan run over an alias-masked frame with no aliases to
	// mask would still pass and would mean less).
	if got := tuileakFindings(base, identities, slots); len(got) != 0 {
		t.Errorf("the committed frame %s reports findings of its own: %v", name, got)
	}
	if len(slots) == 0 {
		t.Fatal("the base frame's style has no alias to mask: the mask above would be a no-op")
	}
	if again := readGolden(t, name); again != committed {
		t.Fatal("a plant reached the committed corpus: testdata/TestGoldenFrames is not a plant target")
	}
}

// scanCoveredTUIFrames is the scan loop of TestTheCoveredTUIFramesLeakNoPlainVocabulary: every covered
// frame's bytes are scanned for the rule, and every one of them must also be seen to carry an alias of
// its own style, so the scan cannot pass over frames that were never rendered through a cover.
func scanCoveredTUIFrames(t *testing.T, covered []coveredTUIFrame, identities []string, slots map[render.CoverStyle][]string) {
	t.Helper()
	scanned, aliased := 0, 0
	for _, f := range covered {
		scanned++
		for _, msg := range tuileakFindings(f, identities, slots[f.style]) {
			t.Error(msg)
		}
		body := ansi.Strip(f.body)
		for _, slot := range slots[f.style] {
			if strings.Contains(body, slot) {
				aliased++
				break
			}
		}
	}
	if scanned != len(covered) {
		t.Errorf("the scan covered %d of the corpus's %d covered frames", scanned, len(covered))
	}
	if aliased != len(covered) {
		t.Errorf("%d of the corpus's %d covered frames print no alias of their own style: a frame rendered through the plain grammar would hold no alias and no leak either", aliased, len(covered))
	}
}
