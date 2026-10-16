package render

// This file is the panic frame's acceptance: one constant frame of exactly H rows of exactly W
// cells that asks the model for nothing but its size, renders no character of the feed, and cannot
// be the product speaking — no prompt, no clock, no hostname, no identity, no escape byte.
//
// The frame's own columns are read from panic.go and every line is held to them, so the constants
// the frame is built with and the lines it draws cannot drift apart, and the absence assertions are
// armed with the strings they scan for: a word the fixture does not carry would make them vacuous.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// panicSizes is the frame's size table: the widths the phase names at the pane's own height, the two
// heights either side of the run of lines, and the widths either side of the pane's floor.
var panicSizes = []struct{ w, h int }{
	{80, 24}, {120, 24}, {60, 24}, {79, 24}, {200, 24},
	{80, 12}, {80, 40}, {120, 12}, {120, 40},
	{MinPaneCols, 24}, {MinPaneCols - 1, 24}, {MinPaneCols - 1, 12}, {30, 40},
}

// panicClockPattern is what a wall clock looks like in a terminal: `09:12` or `09:12:00`. The frame
// carries durations and nothing else that is a number, so a clock in it would be a frame that moved.
var panicClockPattern = regexp.MustCompile(`\d\d:\d\d`)

// TestPanicFrameIsExactlyHRowsOfWCells is the frame's whole contract: H rows of W cells, space
// padded, at every size — and not one escape byte in any of them.
func TestPanicFrameIsExactlyHRowsOfWCells(t *testing.T) {
	plainEnv(t)
	for _, size := range panicSizes {
		frame := PanicFrame(size.w, size.h)
		rows := strings.Split(frame, "\n")
		if len(rows) != size.h {
			t.Errorf("PanicFrame(%d, %d) is %d rows, want %d", size.w, size.h, len(rows), size.h)
			continue
		}
		if strings.HasSuffix(frame, "\n") {
			t.Errorf("PanicFrame(%d, %d) ends in a newline: the frame is the rows themselves", size.w, size.h)
		}
		for i, row := range rows {
			if got := text.Width(row); got != size.w {
				t.Errorf("PanicFrame(%d, %d): row %d is %d cells, want %d: %q", size.w, size.h, i+1, got, size.w, row)
			}
			if strings.Contains(row, "\x1b") {
				t.Errorf("PanicFrame(%d, %d): row %d carries an escape byte: %q", size.w, size.h, i+1, row)
			}
			// No avatar, no icon, no emoji and no other glyph: the frame's own characters are the
			// program's, every literal in copy.go, cover.go and panic.go is printable ASCII, and this
			// frame renders no character of the feed — so a character outside printable ASCII in it
			// is a glyph the frame drew. It is the one frame with no allowance at all: the covered
			// frames may hold the writer's own emoji because they render the writer's text, and this
			// frame renders no text of anybody's.
			for _, r := range row {
				if r >= ' ' && r <= '~' {
					continue
				}
				t.Errorf("PanicFrame(%d, %d): row %d holds %q (U+%04X): the frame carries no avatar, icon or emoji, and its own cells are printable ASCII", size.w, size.h, i+1, string(r), r)
			}
		}
	}

	// A frame of no rows is no frame, and a frame of one row is the run's last row: the H rows are
	// the frame's own and the run is what fills them.
	if got := PanicFrame(80, 0); got != "" {
		t.Errorf("PanicFrame(80, 0) is %q, want the empty frame", got)
	}
	last := text.Pad(panicLines[len(panicLines)-1], 80)
	if got := PanicFrame(80, 1); got != last {
		t.Errorf("PanicFrame(80, 1) is %q, want the run's last line %q", got, last)
	}

	// The frame is a constant: the same call is the same bytes, so nothing in it is read from a clock,
	// a counter or anything the model holds.
	if PanicFrame(80, 24) != PanicFrame(80, 24) {
		t.Error("two calls to PanicFrame(80, 24) differ: the frame is not a constant")
	}
	if panicClockPattern.MatchString(PanicFrame(80, 24)) {
		t.Error("the panic frame carries a wall clock: the frame must not move")
	}
}

// TestPanicFrameIsConstantAboveTheLongestLine is the frame's most striking property, asserted rather
// than described: at any width that holds the run the frame is the same lines, byte for byte, padded
// to the pane. Terminal output does not widen because the window did, and a frame that filled 120
// cells of text would read as a UI rather than as output.
//
// The lines are held to the frame's own three columns here, which is what panic.go's constants are
// for: the `ok` at the status column, the package inside the package field, and the duration at the
// duration column, spaced with spaces only.
func TestPanicFrameIsConstantAboveTheLongestLine(t *testing.T) {
	plainEnv(t)
	narrow := strings.Split(PanicFrame(80, 24), "\n")
	wide := strings.Split(PanicFrame(120, 24), "\n")
	for i := range panicLines {
		if got, want := strings.TrimRight(narrow[i], " "), strings.TrimRight(wide[i], " "); got != want {
			t.Errorf("row %d is %q at 80 columns and %q at 120: the frame stretches with the pane", i+1, got, want)
		}
		if got, want := strings.TrimRight(narrow[i], " "), panicLines[i]; got != want {
			t.Errorf("row %d is %q, want the frame's own line %q", i+1, got, want)
		}
	}
	for i := len(panicLines); i < 24; i++ {
		if got := strings.TrimRight(narrow[i], " "); got != "" {
			t.Errorf("row %d is %q, want a blank row below the run", i+1, got)
		}
	}

	// Every line at the frame's own columns, and the fields behind them.
	for i, line := range panicLines {
		if strings.Contains(line, "\t") {
			t.Errorf("line %d carries a tab byte: %q", i+1, line)
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			t.Errorf("line %d has %d fields, want the status, the package and the duration: %q", i+1, len(fields), line)
			continue
		}
		status, pkg, duration := fields[0], fields[1], fields[2]
		if status != "ok" {
			t.Errorf("line %d's status word is %q, want the run's own", i+1, status)
		}
		if got := line[panicStatusCol : panicStatusCol+text.Width(status)]; got != status {
			t.Errorf("line %d has %q at its status column, want %q", i+1, got, status)
		}
		if got := strings.TrimRight(line[panicPackageCol:panicDurationCol], " "); got != pkg {
			t.Errorf("line %d has %q in its package field, want %q", i+1, got, pkg)
		}
		if text.Width(pkg) > panicPackageCells {
			t.Errorf("line %d's package %q is %d cells, over the frame's %d-cell package field",
				i+1, pkg, text.Width(pkg), panicPackageCells)
		}
		if got := line[panicDurationCol:]; got != duration {
			t.Errorf("line %d has %q at its duration column, want %q", i+1, got, duration)
		}
		if got, want := text.Width(line), panicDurationCol+text.Width(duration); got != want {
			t.Errorf("line %d is %d cells, want %d: the duration ends the line", i+1, got, want)
		}
	}
}

// TestPanicFrameShowsItsLastRowsWhenTheContentIsTallerThanThePane is the run taller than the pane: the
// frame shows its last H rows and adds no truncation mark, because the terminal scrolled and nothing
// was cut.
//
// The lines the frame ships all fit any width above the pane's floor (the longest is 46 cells), so
// the frame is never in its wrapping branch with them; the branch is reached here with a line of the
// test's own — the frame's lines are a package variable and it is put back when the test ends — so
// the rule is exercised rather than left unread.
func TestPanicFrameShowsItsLastRowsWhenTheContentIsTallerThanThePane(t *testing.T) {
	plainEnv(t)
	rows := strings.Split(PanicFrame(80, 12), "\n")
	if len(rows) != 12 {
		t.Fatalf("PanicFrame(80, 12) is %d rows, want 12", len(rows))
	}
	for i, row := range rows {
		want := panicLines[len(panicLines)-12+i]
		if got := strings.TrimRight(row, " "); got != want {
			t.Errorf("row %d is %q, want the run's own line %q: the pane shows the run's last rows", i+1, got, want)
		}
	}
	if strings.Contains(PanicFrame(80, 12), MarkTruncation) {
		t.Error("the scrolled frame carries the truncation mark: the terminal scrolled and nothing was cut")
	}

	restore := panicLines[0]
	panicLines[0] = strings.Repeat("corp/very-long-package-name ", 12)
	t.Cleanup(func() { panicLines[0] = restore })

	longest := 0
	for _, line := range panicLines {
		if w := text.Width(line); w > longest {
			longest = w
		}
	}
	if longest <= MinPaneCols {
		t.Fatalf("the line under test is %d cells and the pane's floor is %d: the wrap branch is not reached", longest, MinPaneCols)
	}

	// The wrapped run: the line under test wrapped the way a terminal wraps it, and every other line
	// whole, because every one of them fits the pane.
	for _, line := range panicLines[1:] {
		if text.Width(line) > MinPaneCols {
			t.Fatalf("the frame's own line %q is %d cells, over the %d-cell pane: the run behind the wrapped line is not the run under test",
				line, text.Width(line), MinPaneCols)
		}
	}
	wrapped := append(text.Wrap(panicLines[0], MinPaneCols), panicLines[1:]...)
	if len(wrapped) <= 24 {
		t.Fatalf("the wrapped run is %d rows, want more than 24 for the pane below to have scrolled", len(wrapped))
	}
	// A pane that holds the wrapped run shows all of it, and a pane shorter than it shows the run's
	// last H rows: the terminal scrolled, so the rows it kept are the rows at the bottom.
	for _, h := range []int{24, 12} {
		rows = strings.Split(PanicFrame(MinPaneCols, h), "\n")
		if len(rows) != h {
			t.Fatalf("PanicFrame(%d, %d) is %d rows, want %d", MinPaneCols, h, len(rows), h)
		}
		for i, row := range rows {
			line := wrapped[len(wrapped)-h+i]
			if got := strings.TrimRight(row, " "); got != strings.TrimRight(line, " ") {
				t.Errorf("PanicFrame(%d, %d) row %d is %q, want the wrapped run's row %q: a terminal wraps a long line rather than cutting it, and a scrolled pane keeps its last rows",
					MinPaneCols, h, i+1, got, line)
			}
		}
		if strings.Contains(strings.Join(rows, "\n"), MarkTruncation) {
			t.Errorf("PanicFrame(%d, %d) carries the truncation mark: the terminal scrolled and nothing was cut", MinPaneCols, h)
		}
	}
}

// TestPanicFrameIsThePlainReadersFloorLineBelowThePaneFloor is the one width the frame refuses to lay
// out: below the pane's floor it is the same one-line floor message the plain reader uses, padded to
// W, and the row count is still H.
func TestPanicFrameIsThePlainReadersFloorLineBelowThePaneFloor(t *testing.T) {
	plainEnv(t)
	for _, w := range []int{MinPaneCols - 1, 40, 20} {
		rows := strings.Split(PanicFrame(w, 24), "\n")
		floor := strings.Split(Frame(nil, StatusFields{}, w, 24), "\n")
		if rows[0] != floor[0] {
			t.Errorf("at %d columns the panic frame's first row is %q and the plain reader's own floor row is %q",
				w, rows[0], floor[0])
		}
		for i, row := range rows[1:] {
			if strings.TrimSpace(row) != "" {
				t.Errorf("at %d columns row %d is %q, want a blank row under the floor line", w, i+2, row)
			}
		}
		if strings.Contains(strings.Join(rows, "\n"), panicLines[0]) {
			t.Errorf("at %d columns the frame draws the run's own lines: the floor line is all there is room for", w)
		}
	}
	// The floor is the pane's and not the frame's: at exactly the floor the run is back.
	if got := strings.TrimRight(strings.Split(PanicFrame(MinPaneCols, 24), "\n")[0], " "); got != panicLines[0] {
		t.Errorf("at %d columns the frame's first row is %q, want the run's first line %q", MinPaneCols, got, panicLines[0])
	}
}

// TestPanicFrameHoldsNoCharacterOfTheFeed is DSGT-09's absence claim at the unit level, and it scans
// the frame itself rather than trusting the frame's construction: no handle, no display name, no word
// of any body, no alias, no hostname, no product noun, no prompt and no path from a machine.
//
// Every string it looks for is armed twice. The words the feed's frames carry are asserted to be in a
// rendering first, so a frame the scan could not see one in would fail this test rather than pass it;
// the strings no frame may carry are asserted to be in the fixture's own content; and every product
// noun is read off the constant that spells it, so none of them is invented here.
func TestPanicFrameHoldsNoCharacterOfTheFeed(t *testing.T) {
	plainEnv(t)
	posts := append(framePosts(), coverFixture()...)

	feed := strings.Join(rowsOf(Blocks(posts, Plain, 80, Selection{ID: ""}, corpusNow, nil)), "\n")
	for _, grammar := range coverGrammars() {
		feed += "\n" + strings.Join(rowsOf(Blocks(posts, grammar, 80, Selection{ID: ""}, corpusNow, nil)), "\n")
	}

	// The words a feed rendering carries: each handle the content names, each word of four cells or
	// more of its texts, and each projection of each account's alias.
	showed := 0
	for _, word := range append(feedWords(posts), aliasProjections(posts)...) {
		if !strings.Contains(feed, word) {
			t.Fatalf("the fixture's own renderings do not carry %q: the scan below would pass by finding nothing", word)
		}
		showed++
	}
	if showed < 40 {
		t.Fatalf("the scan is armed by %d strings, want the fixture's whole vocabulary", showed)
	}

	// The strings only the content carries: the display names and the ids long enough to scan for.
	content := strings.Join(contentStrings(posts), "\n")
	hidden := 0
	for _, secret := range contentStrings(posts) {
		if !strings.Contains(content, secret) {
			t.Fatalf("the fixture content does not carry %q: the absence below would be vacuous", secret)
		}
		hidden++
	}
	if hidden == 0 {
		t.Fatal("the fixture carries no display name and no id: the absence below proves nothing")
	}

	// The product nouns, each read off the constant that spells it.
	product := []struct{ noun, source string }{
		{CopyTitlePlain, CopyTitlePlain},
		{CopyUpstreamHost, CopyUpstreamHost},
		{CopyListFollowing, CopyListFollowing},
		{CopyListForYou, CopyListForYou},
		{"posts", CopyEmptyHeadingFollowing},
		{"records", CopyCoverEmptyHeading},
		{"reposted", MarkerRepost},
		{"{noun}", CopyAttachment},
	}
	for _, entry := range product {
		if !strings.Contains(entry.source, entry.noun) {
			t.Fatalf("the noun %q is not in %q: the list below is not the program's own words", entry.noun, entry.source)
		}
	}

	// The widths that hold the run: below the pane's floor the frame is the plain reader's own floor
	// line — the one line this program does print that says `pane` and cuts itself with `~` — and that
	// line is not the panic frame's content. The floor branch has its own test above.
	for _, size := range panicSizes {
		if size.w < MinPaneCols {
			continue
		}
		frame := PanicFrame(size.w, size.h)
		for _, word := range append(append(feedWords(posts), aliasProjections(posts)...), contentStrings(posts)...) {
			if strings.Contains(frame, word) {
				t.Errorf("PanicFrame(%d, %d) holds %q, a string of the feed it renders nothing of", size.w, size.h, word)
			}
		}
		for _, entry := range product {
			if strings.Contains(frame, entry.noun) {
				t.Errorf("PanicFrame(%d, %d) holds the product's own word %q", size.w, size.h, entry.noun)
			}
		}
		// A prompt, a path from the user's machine, an account and a cut mark: four things the frame
		// makes no claim about.
		for _, shape := range []string{"$", "@", "https://", "http://", "~/", "/home", "~"} {
			if strings.Contains(frame, shape) {
				t.Errorf("PanicFrame(%d, %d) holds %q: the frame claims nothing about the machine it is on", size.w, size.h, shape)
			}
		}
	}
}

// feedWords is the words a feed rendering carries: every handle the content names, and every word of
// four cells or more of its texts. The short words are left out on purpose — the frame's own package
// names carry `not`, `in` and `ok` — and every word kept is asserted to be in a rendering before the
// frame is scanned for it.
func feedWords(posts []domain.Post) []string {
	words := make([]string, 0, 2*len(posts))
	seen := map[string]bool{}
	add := func(word string) {
		if word == "" || seen[word] {
			return
		}
		seen[word] = true
		words = append(words, word)
	}
	account := func(u domain.User) { add(u.ScreenName) }
	for _, post := range posts {
		account(post.Author)
		if post.RepostedBy != nil {
			account(*post.RepostedBy)
		}
		for _, word := range strings.Fields(post.Text) {
			if len(word) >= 4 {
				add(word)
			}
		}
		if post.Quote != nil {
			account(post.Quote.Author)
			for _, word := range strings.Fields(post.Quote.Text) {
				if len(word) >= 4 {
					add(word)
				}
			}
		}
	}
	return words
}

// aliasProjections is every field of every account's alias, for every style: an alias is what a
// covered frame prints where the feed's frames print a handle, so a panic frame that carried one
// would be carrying the feed's own disguise.
func aliasProjections(posts []domain.Post) []string {
	projections := make([]string, 0, 8)
	seen := map[string]bool{}
	add := func(alias CoverAlias) {
		for _, field := range []string{alias.Handle, alias.Name, alias.Login, alias.Email} {
			if field == "" || seen[field] {
				continue
			}
			seen[field] = true
			projections = append(projections, field)
		}
	}
	for _, style := range CoverStyles() {
		for _, post := range posts {
			add(CoverAliasOf(style, post.Author))
		}
	}
	return projections
}

// contentStrings is the strings only the content carries and no frame may show: the display names of
// the accounts it names, and the ids long enough to be scanned for. A one-character id is left out
// because the frame's own durations carry digits — scanning for `1` would prove nothing about
// content — and the ids that are kept are the fixture's own.
func contentStrings(posts []domain.Post) []string {
	out := make([]string, 0, len(posts))
	seen := map[string]bool{}
	add := func(s string) {
		if len(s) < 4 || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	account := func(u domain.User) {
		add(u.Name)
		add(u.ID)
	}
	for _, post := range posts {
		account(post.Author)
		if post.RepostedBy != nil {
			account(*post.RepostedBy)
		}
		add(string(post.ID))
		if post.Quote != nil {
			account(post.Quote.Author)
			add(string(post.Quote.ID))
		}
	}
	return out
}
