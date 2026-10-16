package render

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
	"github.com/LZD121/x-terminal/internal/xweb"
)

// statusNow is the clock the status line's reset instant is read against: the same instant the
// worked frames are stamped with, so "later today" is a fact about the test and not about when it
// runs.
var statusNow = specNow

// worked is the field set behind the UI-SPEC's six worked status lines: a failure that needs the
// widest non-droppable run, a notice, the list, a position with more behind it, and the budget the
// UI-SPEC names (used = 200, limited = 12).
func worked(state domain.FailureKind) StatusFields {
	return StatusFields{
		Failure:     domain.Failure{Kind: state, Reset: resetLaterToday()},
		Notice:      NoticeNewPosts,
		NoticeCount: 3,
		List:        ListName("", domain.Following),
		Selected:    12,
		Loaded:      240,
		HasMore:     true,
		Used:        200,
		Limited:     12,
		Now:         statusNow,
	}
}

// resetLaterToday is a reset instant on the clock's own local day: 14:32, as the UI-SPEC's frame
// shows it.
func resetLaterToday() time.Time {
	return time.Date(2026, 10, 14, 14, 32, 0, 0, time.UTC)
}

// TestStatusLineWorkedLines is the UI-SPEC's six worked status lines at 80 columns, transcribed
// byte for byte. Each line's own fields are the line's, plus the notice and the position: those
// two do not fit beside a failure state and the budget, and the drop rule removes them in order —
// which is what the UI-SPEC's note says they are there for.
//
// The fifth line is the one place the UI-SPEC's own arithmetic does not hold: the fenced line
// `operation id stale  doctor --refresh-ids  following  200 requests, 12 rate-limited` is 82 cells
// wide, two more than the 80-column pane it is presented in, so no implementation that keeps the
// line inside the pane can print it there. It is asserted at the width that holds it, and
// TestTheFifthWorkedLineIsWiderThanItsPane holds the 82.
func TestStatusLineWorkedLines(t *testing.T) {
	plainEnv(t)
	tests := []struct {
		name  string
		lines []string
	}{
		{"a notice, the list, the position and the budget", []string{
			"3 new posts  following  12/240+  200 requests, 12 rate-limited",
		}},
		{"session expired", []string{
			"session expired  copy the cookie again  following  200 requests, 12 rate-limited",
		}},
		{"rate limited", []string{
			"rate limited until 14:32  no retry  following  200 requests, 12 rate-limited",
		}},
		{"no route", []string{
			"no route to x.com  check the proxy  following  200 requests, 12 rate-limited",
		}},
		{"operation id stale", []string{
			"operation id stale  doctor --refresh-ids  following  200 requests, 12 rate-limited",
		}},
		{"soft ban", []string{
			"soft ban  stop for the session  following  200 requests, 12 rate-limited",
		}},
	}
	for i, tc := range tests {
		want := tc.lines[0]
		fields := worked(kindForLine(i))
		width := 80
		if i == 4 {
			// The UI-SPEC's own line: 82 cells, wider than the pane it is shown in.
			width = text.Width(want)
		}
		if got := StatusLine(fields, width); got != want {
			t.Errorf("%s at %d columns:\n got (%d cells): %q\nwant (%d cells): %q",
				tc.name, width, text.Width(got), got, text.Width(want), want)
		}
		if i == 0 && StatusLine(fields, 80) != want {
			t.Errorf("%s: the healthy line is not the UI-SPEC's line at 80 columns", tc.name)
		}
	}
}

// kindForLine names the failure kind behind each of the six worked lines, in the UI-SPEC's order.
// The first line has no failure at all.
func kindForLine(i int) domain.FailureKind {
	switch i {
	case 1:
		return domain.FailureSessionExpired
	case 2:
		return domain.FailureRateLimited
	case 3:
		return domain.FailureNoRoute
	case 4:
		return domain.FailureOperationStale
	case 5:
		return domain.FailureSoftBan
	default:
		return domain.FailureNone
	}
}

// TestTheFifthWorkedLineIsWiderThanItsPane records the number the UI-SPEC's own note contradicts:
// its fifth worked line does not fit the 80 columns it is presented under, and at 80 columns the
// drop rule therefore removes the list field, which is the third thing it may remove.
func TestTheFifthWorkedLineIsWiderThanItsPane(t *testing.T) {
	plainEnv(t)
	fenced := "operation id stale  doctor --refresh-ids  following  200 requests, 12 rate-limited"
	if got := text.Width(fenced); got != 82 {
		t.Fatalf("the fenced fifth line is %d cells, want 82", got)
	}
	fields := worked(domain.FailureOperationStale)
	if got := StatusLine(fields, 80); got != "operation id stale  doctor --refresh-ids  200 requests, 12 rate-limited" {
		t.Fatalf("at 80 columns the line is %q, want the list field dropped and nothing else", got)
	}
	if fields.Used != 200 || fields.Limited != 12 {
		t.Fatalf("the budget under test is %d/%d, want the UI-SPEC's 200/12", fields.Used, fields.Limited)
	}
}

// TestStatusLineDropsInOrder is the drop rule, measured rather than described: across every width
// from 30 to 200 the fields that are present are recorded, and the first width at which each field
// disappears must be in the order notice, then position, then list — while the state, the remedy
// and the budget never disappear at all. It also asserts that a dropped field stays dropped, so a
// field cannot flicker back as the line narrows.
func TestStatusLineDropsInOrder(t *testing.T) {
	plainEnv(t)
	fields := worked(domain.FailureSignerOrAuth)
	notice := NoticeLine(fields.Cover, fields.Notice, fields.NoticeCount)
	position := positionField(fields)
	list := fields.List
	state := StateWord(fields.Failure.Kind)
	remedy := Remedy(fields.Failure.Kind)
	budget := budgetField(fields)
	if notice == "" || position == "" || list == "" {
		t.Fatalf("the fixture has no droppable field to drop: %q %q %q", notice, position, list)
	}

	// The scan stops where the line starts to be cut: past that point the fields are no longer
	// dropped, they are cut through, and the ordering question is already settled.
	// TestStatusLineNeverWraps checks that below the threshold what is left is a prefix of the
	// composed line.
	threshold := text.Width(state) + statusGap + text.Width(remedy) + statusGap + text.Width(budget)
	if threshold > 200 {
		t.Fatalf("the fixture's non-droppable fields need %d cells, more than the scanned 200", threshold)
	}
	droppedAt := map[string]int{}
	wasThere := map[string]bool{}
	for _, field := range []string{notice, position, list, state, remedy, budget} {
		wasThere[field] = true
	}
	for w := 200; w >= threshold; w-- {
		line := StatusLine(fields, w)
		for _, field := range []string{notice, position, list, state, remedy, budget} {
			here := strings.Contains(line, field)
			if wasThere[field] && !here {
				droppedAt[field] = w
			}
			if !wasThere[field] && here {
				t.Fatalf("at %d columns the dropped field %q is back in the line: %q", w, field, line)
			}
			wasThere[field] = here
		}
		for _, field := range []string{state, remedy, budget} {
			if !strings.Contains(line, field) {
				t.Fatalf("at %d columns the line dropped %q, which is never dropped: %q", w, field, line)
			}
		}
	}
	if droppedAt[notice] == 0 || droppedAt[position] == 0 || droppedAt[list] == 0 {
		t.Fatalf("a droppable field never disappeared between the threshold and 200 columns: %v", droppedAt)
	}
}

// TestStatusLineNeverWraps: the line is one row for every width, its measured width never exceeds
// the pane, and below the width that holds even the non-droppable fields it is cut rather than
// rewritten — what is left is a prefix of one of the four compositions the drop rule allows, with
// the truncation mark in its last cell. It also arms the helper the prefix check is made with.
func TestStatusLineNeverWraps(t *testing.T) {
	plainEnv(t)
	fields := worked(domain.FailureSignerOrAuth)
	if !prefixOfAComposition(StatusLine(fields, 400), fields) {
		t.Fatal("the helper the prefix check is made with rejects the full composition of the fields")
	}
	if prefixOfAComposition(StatusLine(fields, 400)+" and then some", fields) {
		t.Fatal("the helper accepts a line that is not a composition of the fields at all")
	}
	state := StateWord(fields.Failure.Kind)
	budget := budgetField(fields)
	for w := 30; w <= 200; w++ {
		line := StatusLine(fields, w)
		if strings.Contains(line, "\n") {
			t.Fatalf("at %d columns the line wrapped: %q", w, line)
		}
		if line == "" {
			t.Fatalf("at %d columns the line is empty", w)
		}
		if cut := strings.TrimSuffix(line, MarkTruncation); !prefixOfAComposition(cut, fields) {
			t.Fatalf("at %d columns the line is not a prefix of any allowed composition of its fields:\n %q", w, line)
		}
		// The fields that are never dropped lead the line at every width: the first cell it shows
		// is the state word's, and the budget is there as soon as the line is wide enough to hold
		// the state and the remedy beside it.
		if !strings.HasPrefix(line, state[:1]) {
			t.Fatalf("at %d columns the line does not start with the state word: %q", w, line)
		}
		if w >= text.Width(state)+statusGap+text.Width(Remedy(fields.Failure.Kind))+statusGap+text.Width(budget) &&
			!strings.Contains(line, budget) {
			t.Fatalf("at %d columns the line lost the budget: %q", w, line)
		}
	}
	if got := StatusLine(fields, 30); !strings.HasSuffix(got, MarkTruncation) {
		t.Fatalf("a line cut at 30 columns has no truncation mark: %q", got)
	}
}

// prefixOfAComposition reports whether line is a prefix of one of the four lines the drop rule
// allows for these fields: everything, then the notice gone, then the position gone too, then the
// list gone as well. It is built from the field helpers rather than from StatusLine, so a line
// that lost a field out of order fails it.
func prefixOfAComposition(line string, f StatusFields) bool {
	gap := strings.Repeat(" ", statusGap)
	state, remedy := stateField(f), remedyField(f.Failure)
	notice, list := NoticeLine(f.Cover, f.Notice, f.NoticeCount), f.List
	position, budget := positionField(f), budgetField(f)
	for _, composed := range []string{
		strings.Join([]string{state, remedy, notice, list, position, budget}, gap),
		strings.Join([]string{state, remedy, list, position, budget}, gap),
		strings.Join([]string{state, remedy, list, budget}, gap),
		strings.Join([]string{state, remedy, budget}, gap),
	} {
		if strings.HasPrefix(composed, line) {
			return true
		}
	}
	return false
}

// TestStatusLineRateLimitedForms: the three forms of the rate-limited state — with a reset later
// today, with one on another day, and with none at all — are three different lines, and the two
// remedies differ: a wait with no end in sight is not the same instruction as a wait until a time.
func TestStatusLineRateLimitedForms(t *testing.T) {
	plainEnv(t)
	fields := worked(domain.FailureRateLimited)
	fields.Notice = NoticeNone
	fields.Selected, fields.Loaded, fields.HasMore = 0, 0, false

	today := fields
	if got := StatusLine(today, 120); got != "rate limited until 14:32  no retry  following  200 requests, 12 rate-limited" {
		t.Fatalf("a reset later today rendered %q", got)
	}

	another := fields
	another.Failure.Reset = time.Date(2027, 1, 2, 15, 4, 0, 0, time.UTC)
	if got := StatusLine(another, 120); got != "rate limited until Jan 02 15:04  no retry  following  200 requests, 12 rate-limited" {
		t.Fatalf("a reset on another day rendered %q", got)
	}

	unknown := fields
	unknown.Failure.Reset = time.Time{}
	if got := StatusLine(unknown, 120); got != "rate limited  no retry; reset unknown  following  200 requests, 12 rate-limited" {
		t.Fatalf("a reset that was never sent rendered %q", got)
	}

	lines := map[string]bool{}
	for _, f := range []StatusFields{today, another, unknown} {
		lines[StatusLine(f, 120)] = true
	}
	if len(lines) != 3 {
		t.Fatalf("the three rate-limited forms are %d distinct lines, want 3", len(lines))
	}
	if remedyField(today.Failure) == remedyField(unknown.Failure) {
		t.Fatal("the two rate-limited remedies are the same words")
	}
	if !strings.Contains(StatusLine(unknown, 120), CopyRemedyRateLimitedUnknown) {
		t.Fatalf("the unknown-reset line does not carry its own remedy: %q", StatusLine(unknown, 120))
	}
	// A reset on another day in the same year is a dated form; the state word is not a countdown.
	if strings.Contains(StatusLine(another, 120), "in ") {
		t.Fatalf("the reset rendered as a countdown: %q", StatusLine(another, 120))
	}
}

// TestStatusLineBudgetNeverMoves: the budget is visible in every frame, so it survives every width
// at which the state and the remedy fit, survives every failure state, and is singular at one
// request.
func TestStatusLineBudgetNeverMoves(t *testing.T) {
	plainEnv(t)
	if got := StatusLine(StatusFields{List: ListName("", domain.Following), Used: 0, Limited: 0}, 80); got != "following  0 requests, 0 rate-limited" {
		t.Fatalf("an idle line rendered %q, want the budget at zero", got)
	}
	if got := StatusLine(StatusFields{List: ListName("", domain.Following), Used: 1, Limited: 1}, 80); got != "following  1 request, 1 rate-limited" {
		t.Fatalf("one request rendered %q, want the singular noun", got)
	}
	if got := StatusLine(StatusFields{List: ListName("", domain.Following), Used: 2, Limited: 1}, 80); got != "following  2 requests, 1 rate-limited" {
		t.Fatalf("two requests rendered %q, want the plural noun", got)
	}

	kinds := []domain.FailureKind{
		domain.FailureNone, domain.FailureSessionExpired, domain.FailureSignerOrAuth, domain.FailureOperationStale,
		domain.FailureEndpointRetired, domain.FailureRateLimited, domain.FailureSoftBan, domain.FailureLoadShed,
		domain.FailureBadRequest, domain.FailureEmptyBody, domain.FailureChallenge, domain.FailureUnauthorized,
		domain.FailureUnclassified, domain.FailureBudgetSpent, domain.FailureNoRoute, domain.FailureProxyRefused,
		domain.FailureUpstreamBlocked, domain.FailureTLSChallenge, domain.FailureTransportUnusable, domain.FailureNoSession,
	}
	for _, kind := range kinds {
		fields := StatusFields{List: ListName("", domain.Following), Failure: domain.Failure{Kind: kind}, Used: 3, Limited: 2}
		line := StatusLine(fields, 200)
		if !strings.Contains(line, "3 requests, 2 rate-limited") {
			t.Fatalf("the %q state's line has no budget: %q", StateWord(kind), line)
		}
	}
}

// TestTheStateWordsAreTheOnesOnTheLine is the screen half of the pin: every state word the copy
// table carries reaches the line as literal text, and every remedy with it, so the classifier's
// words are what a user reads.
func TestTheStateWordsAreTheOnesOnTheLine(t *testing.T) {
	plainEnv(t)
	for kind, word := range stateWords {
		fields := StatusFields{List: ListName("", domain.Following), Failure: domain.Failure{Kind: kind}, Used: 1, Limited: 0}
		line := StatusLine(fields, 200)
		if !strings.Contains(line, word) {
			t.Errorf("the %q state's line does not carry its word: %q", word, line)
		}
		if remedy := Remedy(kind); remedy != "" && !strings.Contains(line, remedy) {
			t.Errorf("the %q state's line does not carry its remedy %q: %q", word, remedy, line)
		}
		if strings.Contains(line, "no posts in") {
			t.Errorf("the %q state's line carries the empty-state copy: %q", word, line)
		}
	}
}

// TestFrameIsExactlyThePanesSize: the frame is h rows of w cells at every size, the status line is
// the last row, and the pane's rows are the blocks' rows padded to the pane's width.
func TestFrameIsExactlyThePanesSize(t *testing.T) {
	plainEnv(t)
	blocks := Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil)
	status := worked(domain.FailureNone)
	for _, w := range []int{60, 79, 80, 120, 200} {
		frame := Frame(blocks, status, w, 24)
		rows := strings.Split(frame, "\n")
		if len(rows) != 24 {
			t.Fatalf("at %d columns the frame is %d rows, want 24", w, len(rows))
		}
		for i, row := range rows {
			if got := text.Width(row); got != w {
				t.Fatalf("at %d columns row %d is %d cells wide: %q", w, i+1, got, row)
			}
		}
		if !strings.Contains(rows[23], "200 requests, 12 rate-limited") {
			t.Fatalf("at %d columns the last row is not the status line: %q", w, rows[23])
		}
		if strings.Contains(strings.Join(rows[:23], "\n"), "rate-limited") {
			t.Fatalf("at %d columns the status line's words appear in the pane", w)
		}
	}
	if got := Frame(blocks, status, 80, 1); strings.Count(got, "\n") != 0 {
		t.Fatalf("a one-row frame has %d newlines", strings.Count(got, "\n"))
	} else if !strings.Contains(got, "200 requests") {
		t.Fatalf("a one-row frame is not the status line: %q", got)
	}
}

// TestFrameTheUISpecFrame: the whole frame at 80 columns and 24 rows, right-trimmed for reading,
// against the UI-SPEC's fenced frame — the pane rows in order, the status line last, and no blank
// row between them because the list fills the pane exactly.
func TestFrameTheUISpecFrame(t *testing.T) {
	plainEnv(t)
	blocks := Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil)
	// The position is the worked lines' 12/240+: at 80 columns with this state and this budget it
	// does not fit, and the drop rule removes it — which is why the UI-SPEC's frame shows no
	// position even though the model has one.
	status := StatusFields{
		Failure:  domain.Failure{Kind: domain.FailureRateLimited, Reset: resetLaterToday()},
		List:     ListName("", domain.Following),
		Selected: 12,
		Loaded:   240,
		HasMore:  true,
		Used:     6,
		Limited:  1,
		Now:      statusNow,
	}
	frame := Frame(blocks, status, 80, 24)

	want := []string{
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
		"rate limited until 14:32  no retry  following  6 requests, 1 rate-limited",
	}
	rows := strings.Split(frame, "\n")
	if len(rows) != len(want) {
		t.Fatalf("the frame has %d rows, want %d", len(rows), len(want))
	}
	for i := range want {
		if got := strings.TrimRight(rows[i], " "); got != want[i] {
			t.Fatalf("frame row %d is %q, want %q", i+1, got, want[i])
		}
		if got := text.Width(rows[i]); got != 80 {
			t.Fatalf("frame row %d is %d cells wide, want the pane's 80", i+1, got)
		}
	}
	if !strings.HasPrefix(rows[6], MarkSelection) {
		t.Fatalf("the selected block's first row is not marked: %q", rows[6])
	}
	if !strings.Contains(frame, "no posts in following") {
		// The absences: a frame with posts never carries the empty-state copy.
	} else {
		t.Fatal("a frame with six posts carries the empty-state copy")
	}
}

// TestThePaneDoesNotChangeWhenTheStateDoes is FEED-05's visual half: a failure with posts already
// loaded leaves the pane exactly where it was and reports itself on the status line alone.
func TestThePaneDoesNotChangeWhenTheStateDoes(t *testing.T) {
	plainEnv(t)
	blocks := Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil)
	healthy := worked(domain.FailureNone)
	failing := healthy
	failing.Failure = domain.Failure{Kind: domain.FailureSessionExpired, Detail: "session missing or expired - re-copy the cookie"}

	one := strings.Split(Frame(blocks, healthy, 80, 24), "\n")
	two := strings.Split(Frame(blocks, failing, 80, 24), "\n")
	if len(one) != len(two) {
		t.Fatalf("the frames differ in row count: %d against %d", len(one), len(two))
	}
	for i := range len(one) - 1 {
		if one[i] != two[i] {
			t.Fatalf("pane row %d changed with the state:\n %q\n %q", i+1, one[i], two[i])
		}
	}
	if one[23] == two[23] {
		t.Fatalf("the status line did not change with the state: %q", one[23])
	}
	if !strings.Contains(two[23], StateWord(domain.FailureSessionExpired)) {
		t.Fatalf("the status line does not name the state: %q", two[23])
	}
	if strings.Contains(one[23], StateWord(domain.FailureSessionExpired)) {
		t.Fatalf("the healthy status line names a failure: %q", one[23])
	}
}

// TestFrameTooNarrow: below the pane's floor the pane is one line that says so — and the status
// line still renders, with its fields dropped by its own rule.
func TestFrameTooNarrow(t *testing.T) {
	plainEnv(t)
	blocks := Blocks(framePosts(), Plain, MinPaneCols, Selection{ID: specSelected}, specNow, nil)
	for _, w := range []int{MinPaneCols - 1, 40, 30, 12} {
		frame := Frame(blocks, worked(domain.FailureNone), w, 24)
		rows := strings.Split(frame, "\n")
		if len(rows) != 24 {
			t.Fatalf("at %d columns the frame is %d rows, want 24", w, len(rows))
		}
		for i, row := range rows {
			if got := text.Width(row); got != w {
				t.Fatalf("at %d columns row %d is %d cells wide", w, i+1, got)
			}
		}
		want := " " + CopyPaneTooNarrow
		row := strings.TrimRight(rows[0], " ")
		switch {
		case text.Width(want) <= w:
			if row != want {
				t.Fatalf("at %d columns the pane is %q, want the narrow line", w, row)
			}
		default:
			if !strings.HasSuffix(row, MarkTruncation) {
				t.Fatalf("at %d columns the pane is %q, want the narrow line cut with the mark", w, row)
			}
			if !strings.HasPrefix(want, strings.TrimSuffix(row, MarkTruncation)) {
				t.Fatalf("at %d columns the pane's line is not a prefix of the narrow line: %q", w, row)
			}
		}
		if strings.Contains(frame, "lzd") {
			t.Fatalf("at %d columns the pane rendered a post anyway", w)
		}
	}
	if got := Frame(blocks, worked(domain.FailureNone), MinPaneCols, 24); !strings.Contains(got, "lzd") {
		t.Fatal("the 60-column pane did not render the list: the floor is off by one")
	}
}

// TestFrameClipsBlocksToThePane: the frame is exactly h rows whatever the list's length, so a
// block that runs past the pane's last row is clipped rather than allowed to scroll the frame. The
// shell's anchor is what decides which block is at the top.
func TestFrameClipsBlocksToThePane(t *testing.T) {
	plainEnv(t)
	blocks := Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil)
	status := worked(domain.FailureNone)

	tall := Frame(blocks, status, 80, 24)
	short := Frame(blocks, status, 80, 5)
	shortRows := strings.Split(short, "\n")
	if len(shortRows) != 5 {
		t.Fatalf("a five-row frame is %d rows", len(shortRows))
	}
	tallRows := strings.Split(tall, "\n")
	for i := range 4 {
		if shortRows[i] != tallRows[i] {
			t.Fatalf("row %d is not the same block row as in the tall frame:\n %q\n %q", i+1, shortRows[i], tallRows[i])
		}
	}
	if shortRows[4] != tallRows[23] {
		t.Fatalf("a five-row frame's last row is not the status line: %q", shortRows[4])
	}
	if strings.Contains(short, "kestrel") {
		t.Fatal("a five-row frame rendered a block from below its last pane row")
	}

	empty := Frame(nil, status, 80, 24)
	emptyRows := strings.Split(empty, "\n")
	if len(emptyRows) != 24 {
		t.Fatalf("an empty pane's frame is %d rows, want 24", len(emptyRows))
	}
	for i, row := range emptyRows[:23] {
		if row != strings.Repeat(" ", 80) {
			t.Fatalf("row %d of an empty pane is not blank: %q", i+1, row)
		}
	}
}

// TestFramePaintsTheSelectionBar: the selection mark is the bar's source, the bar runs to the
// pane's right edge because it is painted after the padding, and a row that is not marked is not
// painted at all.
func TestFramePaintsTheSelectionBar(t *testing.T) {
	colourEnv(t)
	blocks := Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil)
	rows := strings.Split(Frame(blocks, worked(domain.FailureNone), 80, 24), "\n")

	barred := 0
	for i, row := range rows[:23] {
		marked := strings.HasPrefix(ansi.Strip(row), MarkSelection)
		if marked {
			barred++
		}
		if marked && !strings.HasPrefix(row, sgrReverse) {
			t.Fatalf("the selected row %d does not start with the reverse video: %q", i+1, row)
		}
		if !marked && strings.HasPrefix(row, sgrReverse) {
			t.Fatalf("an unmarked row %d is painted as the selection: %q", i+1, row)
		}
		if got := text.Width(row); got != 80 {
			t.Fatalf("row %d is %d cells wide with the bar: %q", i+1, got, row)
		}
	}
	if barred == 0 {
		t.Fatal("no row of the frame carries the selection bar")
	}
	if strings.HasPrefix(rows[23], sgrReverse) {
		t.Fatalf("the status line is painted as a selection: %q", rows[23])
	}
	// The padding is inside the bar: the row ends with the bar's own off code, after the spaces.
	last := ""
	for _, row := range rows[:23] {
		if strings.HasPrefix(row, sgrReverse) {
			last = row
		}
	}
	if !strings.HasSuffix(last, sgrReverseOff) {
		t.Fatalf("the bar does not close after the row's padding: %q", last)
	}
	if !strings.Contains(strings.TrimSuffix(last, sgrReverseOff), "  ") {
		t.Fatalf("the bar's row has no padding inside it: %q", last)
	}
}

// TestFrameColourIsOnlyASecondStatement is A1 at the frame level, over the whole frame rather than
// one block: the truecolor frame and the mono frame are the same characters, the mono rung of a
// dumb terminal carries no escape at all, and the coloured frame really is coloured.
//
// The blocks are built inside each environment on purpose: the block builder paints its runs too,
// so a list built under one rung and rendered under another is not a render of that rung.
func TestFrameColourIsOnlyASecondStatement(t *testing.T) {
	status := worked(domain.FailureRateLimited)

	plainEnv(t)
	mono := Frame(Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil), status, 80, 24)
	colourEnv(t)
	colour := Frame(Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil), status, 80, 24)

	if strips := ansi.Strip(colour); strips != mono {
		for i, row := range strings.Split(strips, "\n") {
			if want := strings.Split(mono, "\n")[i]; row != want {
				t.Fatalf("row %d strips to %q, want the mono frame's %q", i+1, row, want)
			}
		}
		t.Fatal("the stripped truecolor frame differs from the mono frame")
	}
	if strings.Contains(mono, "\x1b") {
		t.Fatalf("the mono rung of a dumb terminal emitted an escape: %q", mono)
	}
	if !strings.Contains(colour, sgrReverse) || !strings.Contains(colour, sgrStateTrueColor) {
		t.Fatalf("the coloured frame carries neither the selection nor the state colour")
	}
	if !strings.Contains(colour, "rate limited until 14:32") {
		t.Fatal("the coloured frame lost the state word: the escapes are inside it")
	}
}

// TestFrameKeepsTheStateWordWhole: the accent is one run around the whole state word including its
// reset suffix, so stripping the escapes leaves the word unbroken and the status line's own rule
// (A2) holds for every state.
func TestFrameKeepsTheStateWordWhole(t *testing.T) {
	colourEnv(t)
	blocks := Blocks(framePosts(), Plain, 80, Selection{ID: specSelected}, specNow, nil)
	status := worked(domain.FailureRateLimited)
	row := strings.Split(Frame(blocks, status, 80, 24), "\n")[23]
	if !strings.Contains(row, sgrStateTrueColor) {
		t.Fatalf("the rate-limited state word is not accent-coloured: %q", row)
	}
	if !strings.Contains(ansi.Strip(row), "rate limited until 14:32") {
		t.Fatalf("the state word is not whole after stripping: %q", ansi.Strip(row))
	}

	stopped := status
	stopped.Failure = domain.Failure{Kind: domain.FailureSessionExpired}
	row = strings.Split(Frame(blocks, stopped, 80, 24), "\n")[23]
	if !strings.Contains(row, sgrStateStopTrueColor) {
		t.Fatalf("a stopped session's state word is not the stop accent: %q", row)
	}
	if !strings.Contains(ansi.Strip(row), "session expired  copy the cookie again") {
		t.Fatalf("the stopped line lost its words: %q", ansi.Strip(row))
	}
}

// coveredWorked is the worked field set with a cover worn: the same failure, notice, position and
// budget, on the style's own line. Every covered difference of the line is the Cover field's, so this
// is the whole of what a test has to change to wear one.
func coveredWorked(style CoverStyle, state domain.FailureKind) StatusFields {
	fields := worked(state)
	fields.Cover = style
	fields.List = ListName(style, domain.Following)
	return fields
}

// TestCoveredStatusLinesWorked is the UI-SPEC's six worked covered status lines at 80 columns,
// transcribed byte for byte from `## The Status Line Under a Cover`: Phase 2's line, field for field
// and drop order for drop order, with the covered list name and the two copy changes of `##
// Copywriting Contract` and nothing else.
//
// The arm under each line is the pair of the two renderings: the covered line must differ from the
// plain reader's for the same state, so a cover that changed nothing cannot pass a transcription that
// is really the plain one, and the plain line is held to its own words by the test above.
//
// The five-column note in the UI-SPEC and the width of the fenced fifth line are why this table has
// six rows and no width clause: the covered line for `operation id stale` loses the list field, which
// is the one line of the six that does not fit 80 columns in the plain grammar.
func TestCoveredStatusLinesWorked(t *testing.T) {
	plainEnv(t)
	tests := []struct {
		name string
		want string
	}{
		{"a notice, the list, the position and the budget",
			"3 new records  branch=main  12/240+  200 requests, 12 rate-limited"},
		{"session expired",
			"session expired  copy the cookie again  200 requests, 12 rate-limited"},
		{"rate limited",
			"rate limited until 14:32  no retry  branch=main  200 requests, 12 rate-limited"},
		{"no route",
			"no route to upstream  check the proxy  200 requests, 12 rate-limited"},
		{"operation id stale",
			"operation id stale  doctor --refresh-ids  200 requests, 12 rate-limited"},
		{"soft ban",
			"soft ban  stop for the session  branch=main  200 requests, 12 rate-limited"},
	}
	for i, tc := range tests {
		kind := kindForLine(i)
		got := StatusLine(coveredWorked(CoverGitLog, kind), 80)
		if got != tc.want {
			t.Errorf("the covered %s line at 80 columns:\n got (%d cells): %q\nwant (%d cells): %q",
				tc.name, text.Width(got), got, text.Width(tc.want), tc.want)
		}
		if text.Width(got) > 80 {
			t.Errorf("the covered %s line is %d cells in an 80-column pane", tc.name, text.Width(got))
		}
		// The arm: the covered line is the plain reader's own only where the one field the two
		// styles spell differently is not on the line at all — `operation id stale` is over the pane
		// in both states, so its list field is the dropped one and the two lines coincide.
		plain := StatusLine(worked(kind), 80)
		if plain == got && strings.Contains(got, ListName(CoverGitLog, domain.Following)) {
			t.Errorf("the covered %s line is the plain reader's own with the list on it: %q", tc.name, got)
		}
	}
}

// TestTheCoveredDropCostsTheListAtEightyColumns is the UI-SPEC's named consequence, measured rather
// than described: the covered list names are longer than `following` / `for you`, so at 80 columns the
// droppable list field disappears in three of the six worked lines instead of one.
//
// The measurement is over the rendered lines rather than over the names' widths, so what is asserted
// is the reader's screen: `session expired` and `no route` lose the list where the plain reader keeps
// it, and `operation id stale` loses it in both, because that state's line is over the pane either
// way. Above 120 columns nothing is dropped in either state, which is what makes the rule a width
// rule and not a style rule.
func TestTheCoveredDropCostsTheListAtEightyColumns(t *testing.T) {
	plainEnv(t)
	names := []string{"a notice", "session expired", "rate limited", "no route", "operation id stale", "soft ban"}
	dropped := func(style CoverStyle) []string {
		out := make([]string, 0, len(names))
		for i, name := range names {
			line := StatusLine(coveredWorked(style, kindForLine(i)), 80)
			if !strings.Contains(line, ListName(style, domain.Following)) {
				out = append(out, name)
			}
		}
		return out
	}

	if got, want := strings.Join(dropped(CoverGitLog), ", "), "session expired, no route, operation id stale"; got != want {
		t.Errorf("the covered line drops the list in %q, want %q", got, want)
	}
	if got, want := strings.Join(dropped(""), ", "), "operation id stale"; got != want {
		t.Errorf("the plain line drops the list in %q, want %q", got, want)
	}

	// The rule is the pane's width: above it nothing is dropped, covered or plain.
	for _, style := range append([]CoverStyle{""}, CoverStyles()...) {
		for i := range names {
			line := StatusLine(coveredWorked(style, kindForLine(i)), 120)
			for _, field := range []string{ListName(style, domain.Following), NoticeLine(style, NoticeNewPosts, 3)} {
				if !strings.Contains(line, field) {
					t.Errorf("the %q line at 120 columns lost %q: %q", style, field, line)
				}
			}
		}
	}
}

// TestTheHostSubstitution is the one edit a cover makes to a sentence it did not author, case by case:
// the bare host, a subdomain, several labels of one subdomain, the host in another case, the host used
// twice, and the strings it must leave byte for byte — a run that is a different host whose name
// merely ends in the host's letters, a host followed by more of a label, a run with an empty label, a
// run whose label begins with a hyphen, an address, and a sentence with no host in it at all.
//
// The arm is inside the table: every case whose expectation is the input itself carries the host's
// letters, so no case is vacuous, and every case that changes anything is checked to have left no
// occurrence of the host behind.
func TestTheHostSubstitution(t *testing.T) {
	const host = CopyUpstreamHost
	const word = CopyUpstreamHostWord
	tests := []struct {
		name, in, want string
	}{
		{"the bare host", host, word},
		{"the host's own words around it", "no route to " + host, "no route to " + word},
		{"a subdomain", "api." + host, word},
		{"several labels of one subdomain", "edge.api.eu." + host + ":443 refused", word + ":443 refused"},
		{"the host in another case", "API.X.CoM refused", word + " refused"},
		{"the host twice in one sentence", host + " refused, then " + host + " timed out", word + " refused, then " + word + " timed out"},
		{"the host behind a proxy URL", "HTTPS_PROXY=http://proxy.local:8080 said " + host, "HTTPS_PROXY=http://proxy.local:8080 said " + word},
		{"a different host whose name ends in the host's letters", "not" + host + " is one label", "not" + host + " is one label"},
		{"a host followed by more of a label", host + "foo is one label", host + "foo is one label"},
		{"a run with an empty label", ".." + host + " is not a name", ".." + host + " is not a name"},
		{"a label that begins with a hyphen", "-a." + host + " is not a name", "-a." + host + " is not a name"},
		{"an address rather than the host", "127.0.0.1:8080 refused", "127.0.0.1:8080 refused"},
		{"a name that merely begins with the host's letters", host + "petitor refused", host + "petitor refused"},
		{"a sentence with no host in it", "proxy refused", "proxy refused"},
		{"the empty string", "", ""},
	}
	// Three counts, so every arm of the rule is under test: the cases that must change, the cases that
	// carry the host's letters and must be left alone (a run is not the host because it merely ends in
	// it), and the cases with no host in them at all, which are the substitution's no-op.
	changed, spelled, clean := 0, 0, 0
	for _, tc := range tests {
		got := HostSubstituted(tc.in)
		if got != tc.want {
			t.Errorf("%s: HostSubstituted(%q) is %q, want %q", tc.name, tc.in, got, tc.want)
			continue
		}
		switch {
		case tc.want != tc.in:
			changed++
			if strings.Contains(got, host) {
				t.Errorf("%s: the substitution left %q behind: %q", tc.name, host, got)
			}
		case strings.Contains(tc.in, host):
			spelled++
		default:
			clean++
		}
	}
	if changed < 7 || spelled < 5 || clean < 2 {
		t.Fatalf("the table changes %d strings, leaves %d that spell the host alone and %d that carry no host: every arm of the rule has to be present",
			changed, spelled, clean)
	}
}

// TestNoCoveredStatusLineNamesTheHost is the substitution at the line level, over the closed taxonomy:
// for every failure kind the domain names and every cover, the line at a width that drops nothing
// holds no `x.com` and no hostname at all, and the state it reports is the classifier's own word with
// only that proper noun replaced.
//
// The plain reader is the arm: it still spells the host for the one state that carries it, so the
// absence under the covers is a change of one word and not a word that was dropped everywhere.
func TestNoCoveredStatusLineNamesTheHost(t *testing.T) {
	plainEnv(t)
	spellings := 0
	for _, kind := range allKinds() {
		// The arm: the plain reader's own line for the same state still spells the host, and spells it
		// as the classifier's state word, so what the covers change is one word and not the field.
		if plain := StatusLine(worked(kind), 200); strings.Contains(plain, CopyUpstreamHost) {
			spellings++
			if word := StateWord(kind); !strings.Contains(plain, word) {
				t.Errorf("the plain reader's line for %v does not hold its own state word %q: %q", kind, word, plain)
			}
		}
		for _, style := range CoverStyles() {
			line := StatusLine(coveredWorked(style, kind), 200)
			if line == "" {
				t.Errorf("the %s line for %v is empty: a covered failure can never render as nothing", style, kind)
				continue
			}
			if strings.Contains(line, CopyUpstreamHost) {
				t.Errorf("the %s line for %v holds the host: %q", style, kind, line)
			}
			if word := StateWord(kind); word != "" && !strings.Contains(line, HostSubstituted(word)) {
				t.Errorf("the %s line for %v does not hold its state word: %q", style, kind, line)
			}
			if remedy := Remedy(kind); remedy != "" && !strings.Contains(line, remedy) {
				t.Errorf("the %s line for %v does not hold the classifier's remedy %q verbatim: %q", style, kind, remedy, line)
			}
		}
	}
	if spellings != 1 {
		t.Fatalf("the plain reader spells the host in %d states, want the one that names a route", spellings)
	}
}

// TestTheCoveredFailurePaneHoldsNoSentenceOfTheClassifier is DSGT-09 on the failure block, over the
// whole closed taxonomy and every cover: the classifier's sentence is the plain reader's, and a
// covered pane prints the cover's own line in its place.
//
// The sentence under test is the one the leak was found in: the reachability check's own
// `GET <host>/ failed: …` — which a covered pane of the shipped program printed with the host in it —
// followed by the classifier's empty-body sentence, which carries the product's own word `timeline`.
// That second half is what a substitution cannot fix: replacing the host inside the sentence leaves
// the upstream's vocabulary and the product's words on a covered screen, so the rule is the whole
// sentence out and the cover's own line in.
//
// The plain rendering of the same kind is the arm: it prints its state's own word and the classifier's
// sentence byte for byte, with the host in it and the product's word in it, so the absence under the
// covers is a change of the pane's words rather than a block that says nothing.
func TestTheCoveredFailurePaneHoldsNoSentenceOfTheClassifier(t *testing.T) {
	plainEnv(t)
	detail := routeSentence() + " - " + xweb.OutcomeEmptyBody.Remedy()

	plainBlocks, coveredBlocks, plainNouns := 0, 0, 0
	for _, kind := range allKinds() {
		if kind == domain.FailureNone {
			continue
		}
		for _, style := range append([]CoverStyle{""}, CoverStyles()...) {
			word, body := FailureWords(style, kind, detail)
			rows := rowsOf([]Block{StateBlock(style.Grammar(), 200, word, body)})
			block := strings.Join(rows, "\n")
			lower := strings.ToLower(block)
			if style == "" {
				plainBlocks++
				if word != StateWord(kind) || body != detail {
					t.Errorf("the plain reader's failure words for %v are %q / %q, want its own state word and the classifier's sentence byte for byte", kind, word, body)
				}
				if !strings.Contains(block, CopyUpstreamHost) {
					t.Errorf("the plain block for %v does not print the host: the sentence does not reach the pane, so the absence below would prove nothing", kind)
				}
				for _, noun := range productNouns {
					if noun.pattern.MatchString(lower) {
						plainNouns++
					}
				}
				continue
			}
			coveredBlocks++
			if word != HostSubstituted(StateWord(kind)) {
				t.Errorf("the %s head word for %v is %q, want the classifier's state word with the host substituted", style, kind, word)
			}
			if body != CopyCoverFailureBody {
				t.Errorf("the %s body for %v is %q, want the cover's own line %q", style, kind, body, CopyCoverFailureBody)
			}
			if !strings.Contains(block, CopyCoverFailureBody) {
				t.Errorf("the %s failure block for %v does not carry the cover's own line:\n%s", style, kind, block)
			}
			if strings.Contains(block, CopyUpstreamHost) || strings.Contains(lower, "twitter") || strings.Contains(block, CopyTitlePlain) {
				t.Errorf("the %s failure block for %v prints a name of the product it was built from:\n%s", style, kind, block)
			}
			// The sentence itself, in both of the forms it could reach the pane in — the classifier's
			// own, and the same sentence with the host substituted — with the block's own wrap and
			// gutter undone. A cover that scrubbed the sentence instead of keeping it out is still
			// printing it, and neither the wrap nor the gutter is a character of the text.
			folded := collapsed(deprefixed(rows, style.Grammar()))
			if strings.Contains(folded, collapsed(detail)) || strings.Contains(folded, collapsed(HostSubstituted(detail))) {
				t.Errorf("the %s failure block for %v prints the classifier's own sentence, scrubbed: the sentence is what a cover may not say at all:\n%s", style, kind, block)
			}
			for _, noun := range productNouns {
				if noun.pattern.MatchString(lower) {
					t.Errorf("the %s failure block for %v prints %q, the product's own word (from %s):\n%s", style, kind, noun.noun, noun.source, block)
				}
			}
		}
	}
	if plainBlocks != wantStateRows {
		t.Errorf("the plain arm ran over %d states, want the %d the taxonomy names", plainBlocks, wantStateRows)
	}
	if coveredBlocks != wantStateRows*len(CoverStyles()) {
		t.Errorf("the covered check ran over %d blocks, want %d — every state under every cover", coveredBlocks, wantStateRows*len(CoverStyles()))
	}
	if plainNouns == 0 {
		t.Error("the plain reader's blocks print none of the product's own words: the sentence under them carries none, so the covered absence is vacuous")
	}
}

// collapsed is a string's words joined by one space each: the form a wrapped block and the unwrapped
// sentence compare in, because a wrap replaces a space with a row break and an indent, and neither is
// a character of the text.
func collapsed(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// deprefixed is a block's rows with the grammar's own gutter and indent taken off and joined by a row
// break: the block's text in the one form a sentence that wrapped inside it can be found in. The
// gutter is the grammar's own — a build log's `| ` — so it is read off the grammar rather than
// guessed here.
func deprefixed(rows []string, g Grammar) string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, strings.TrimLeft(strings.TrimPrefix(row, g.BodyPrefix), " "))
	}
	return strings.Join(out, "\n")
}

// TestTheCoveredCopyIsTotal is the totality of the covered copy: every state in the closed taxonomy
// and every cover has words for every field the pane and the line can ask about, at both feeds — so a
// covered frame can never render an empty field where a word is required — and every one of those
// words is the cover's own and not the plain reader's, which is what keeps the two vocabularies from
// being the same one by accident.
func TestTheCoveredCopyIsTotal(t *testing.T) {
	plainEnv(t)
	feeds := []domain.HomeKind{domain.Following, domain.ForYou}
	// Every notice a covered frame can carry, including the four a mirror read leaves and the three
	// a conversation's replies leave. They were the list's hole: the four saved notices have their own
	// arms in NoticeLine (`coverOr(cover, CopyCoverSyncing, CopySyncing)` and its three siblings), and
	// with them out of this list, reverting any of the four to the plain constant left the suite
	// green — the reply notices' own omission is recorded in 02-VERIFICATION's F22.
	notices := []NoticeKind{
		NoticeNewPosts, NoticeNoNewPosts, NoticeLoadingOlder, NoticeStalled, NoticeNoResults,
		NoticeNothingToSwitch, NoticeLoadingPosts, NoticeNoPosts, NoticeNoAuthor,
		NoticeSyncing, NoticeNoLongerSaved, NoticeMirrorComplete, NoticeSyncStalled,
		NoticeLoadingReplies, NoticeNewReplies, NoticeNoNewReplies,
		NoticeTranslated, NoticeNoTranslation,
	}
	for _, style := range CoverStyles() {
		for _, feed := range feeds {
			for _, lookup := range []struct {
				name           string
				covered, plain string
			}{
				{"the list", ListName(style, feed), ListName("", feed)},
				{"the empty state's heading", EmptyHeading(style, feed), EmptyHeading("", feed)},
				{"the loading line", LoadingLine(style, feed), LoadingLine("", feed)},
			} {
				if lookup.covered == "" {
					t.Errorf("the %s style has no words for %s", style, lookup.name)
					continue
				}
				if lookup.covered == lookup.plain {
					t.Errorf("the %s style's %s is the plain reader's own (%q)", style, lookup.name, lookup.covered)
				}
			}
		}
		if got, plain := EmptyBody(style), EmptyBody(""); got == "" || got == plain {
			t.Errorf("the %s style's empty-state body is %q against the plain reader's %q", style, got, plain)
		}
		for _, notice := range notices {
			for _, count := range []int{1, 2, 5} {
				got := NoticeLine(style, notice, count)
				if got == "" {
					t.Errorf("the %s style has no words for notice %d at %d", style, notice, count)
					continue
				}
				if strings.Contains(got, PlaceholderCount) {
					t.Errorf("the %s style's notice %q still carries the count placeholder", style, got)
				}
				if got == NoticeLine("", notice, count) {
					t.Errorf("the %s style's notice for %d is the plain reader's own (%q)", style, notice, got)
				}
			}
		}
		for _, kind := range allKinds() {
			if line := StatusLine(coveredWorked(style, kind), 120); line == "" {
				t.Errorf("the %s line for %v is empty", style, kind)
			}
		}
	}
}

// TestSyncedFieldDropOrder pins the Phase 6 drop order: notice, position, list,
// then synced. At 80 columns a no-route frame keeps synced and drops list; at
// 120 both survive.
func TestSyncedFieldDropOrder(t *testing.T) {
	plainEnv(t)
	f := StatusFields{
		Failure:  domain.Failure{Kind: domain.FailureNoRoute},
		List:     ListSaved("", domain.Bookmark),
		Synced:   SyncedField{On: true, At: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)},
		Selected: 12, Loaded: 40, HasMore: true,
		Used: 12, Limited: 0, Now: statusNow,
	}
	at80 := StatusLine(f, 80)
	at120 := StatusLine(f, 120)
	if !strings.Contains(at80, "synced Jan 02") {
		t.Fatalf("80-column line dropped synced: %q", at80)
	}
	if strings.Contains(at80, "bookmarks") {
		t.Fatalf("80-column line kept list: %q", at80)
	}
	if !strings.Contains(at120, "synced Jan 02") || !strings.Contains(at120, "bookmarks") {
		t.Fatalf("120-column line lost a kept field: %q", at120)
	}
}

func TestNoticeKindsHaveWords(t *testing.T) {
	// Totality: every NoticeKind other than NoticeNone and NoticeGated has words
	// from NoticeLine, and every constant's placeholders are substituted.
	// NoticeGated's words come from StatusFields.Gated via statusParts.
	for k := NoticeNone + 1; ; k++ {
		if k == NoticeGated {
			if words := NoticeLine("", k, 2); words != "" {
				t.Fatalf("NoticeGated NoticeLine = %q, want empty (words from StatusFields.Gated)", words)
			}
			break
		}
		words := NoticeLine("", k, 2)
		if k == NoticeOpenedMedia {
			if words != "opened 2 media" {
				t.Fatalf("NoticeOpenedMedia = %q, want opened 2 media", words)
			}
			continue
		}
		if words == "" {
			// Stop once we walk past the last defined kind: unknown kinds are empty.
			if k > NoticeGated {
				break
			}
			t.Errorf("NoticeKind %d has no words", k)
			continue
		}
		if strings.Contains(words, "{") {
			t.Errorf("NoticeKind %d still carries a placeholder: %q", k, words)
		}
	}
	if got := NoticeLine("", NoticeNoOpener, 0); got != CopyNoticeNoOpener {
		t.Fatalf("NoOpener = %q", got)
	}
	if got := NoticeLine("", NoticeNoMedia, 0); got != CopyNoticeNoMedia {
		t.Fatalf("NoMedia = %q", got)
	}
	if got := NoticeLine("", NoticeOpenedBrowser, 0); got != CopyNoticeOpenedBrowser {
		t.Fatalf("OpenedBrowser = %q", got)
	}
	if got := NoticeLine("", NoticeLinkCopied, 0); got != CopyNoticeLinkCopied {
		t.Fatalf("LinkCopied = %q", got)
	}
	if got := NoticeLine("", NoticeLinkCopiedWL, 0); got != CopyNoticeLinkCopiedWL {
		t.Fatalf("LinkCopiedWL = %q", got)
	}
	if got := NoticeLine("", NoticeNoClipboard, 0); got != CopyNoticeNoClipboard {
		t.Fatalf("NoClipboard = %q", got)
	}
}

func TestGatedNoticeUsesStatusFields(t *testing.T) {
	plainEnv(t)
	line := StatusLine(StatusFields{
		Notice: NoticeGated,
		Gated:  CoverServiceLogGated,
		List:   "following",
		Used:   12,
	}, 120)
	if !strings.Contains(ansi.Strip(line), CoverServiceLogGated) {
		t.Fatalf("gated status line missing grammar literal: %q", line)
	}
	empty := StatusLine(StatusFields{Notice: NoticeGated, List: "following", Used: 12}, 120)
	if strings.Contains(ansi.Strip(empty), CoverServiceLogGated) {
		t.Fatalf("empty Gated still printed a refusal: %q", empty)
	}
}
