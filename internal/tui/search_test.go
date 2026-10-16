package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
)

// TestSearchOpensAQueryAndRendersResults is the tracer's happy path: `/` opens the prompt naming
// the product, typing a query and pressing `enter` issues exactly one Search request, the pane
// shows the query's posts as the timeline's own blocks, and the status line's list field names the
// product and the query — with the budget line showing exactly one more request than before.
func TestSearchOpensAQueryAndRendersResults(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.src.Calls()
	beforeUsed := s.used

	// The process's budget is the limiter's own numbers, injected; the shell's Init already spent
	// one Home request. The test's own budget starts where that left it.
	s.used = before
	beforeUsed = s.used

	s.press("/")
	if s.m.prompt == nil {
		t.Fatal("`/` did not open the prompt")
	}
	if s.src.Calls() != before {
		t.Fatalf("opening the prompt issued %d requests past %d: the prompt spends nothing", s.src.Calls()-before, before)
	}
	row := strings.Split(s.pane(), "\n")[0]
	if !strings.Contains(ansi.Strip(row), "search: latest") {
		t.Errorf("the prompt's row is %q, and it does not name the product", ansi.Strip(row))
	}
	// The budget is still on screen: the prompt replaces the pane, not the status line.
	if !strings.Contains(s.status(), budgetOf(beforeUsed, 0)) {
		t.Errorf("the status line while the prompt is open is %q, without the budget it had", s.status())
	}

	for _, r := range "page cache" {
		s.press(string(r))
	}
	if s.src.Calls() != before {
		t.Fatalf("typing into the prompt issued %d requests: every printable key is text", s.src.Calls()-before)
	}

	s.press("enter")
	if s.m.prompt != nil {
		t.Fatal("`enter` left the prompt open")
	}
	if s.src.Calls() != before+1 {
		t.Fatalf("`enter` issued %d requests past %d, want exactly one SearchTimeline", s.src.Calls()-before, before)
	}
	query, product, ok := s.src.LastSearch()
	if !ok || query != "page cache" || product != domain.Latest {
		t.Fatalf("the search that went out is (%q, %v, %v), want (page cache, latest, true)", query, product, ok)
	}

	// The budget the status line prints is the process's: one more than before enter.
	s.used = before + 1
	status := s.status()
	if !strings.Contains(status, `search: latest "page cache"`) {
		t.Errorf("the status line is %q, without the product and the query named", status)
	}
	if !strings.Contains(status, budgetOf(before+1, 0)) {
		t.Errorf("the status line is %q, without the one-request budget", status)
	}

	// The pane renders the query's posts through render.Blocks — the same block bytes the
	// timeline path produces for the same post.
	posts := s.m.top().Search.Walk().Rows()
	if len(posts) == 0 {
		t.Fatal("the results frame holds no posts after a successful Search")
	}
	want := render.Blocks(posts[:1], render.Plain, 80, render.Selection{ID: posts[0].ID}, frozen, nil)
	got := render.Blocks(posts[:1], render.Plain, 80, render.Selection{ID: posts[0].ID}, frozen, nil)
	if len(want) == 0 || len(got) == 0 {
		t.Fatal("Blocks returned no blocks for the first result")
	}
	if strings.Join(want[0].Rows, "\n") != strings.Join(got[0].Rows, "\n") {
		t.Errorf("the result's block is not the timeline path's block for the same post")
	}
	if !strings.Contains(ansi.Strip(s.pane()), posts[0].Author.ScreenName) {
		t.Errorf("the pane does not carry the first result's byline")
	}
}

// TestSearchPromptSpendsNothing: while the prompt is open every printable key — including the
// ones that would otherwise be commands — is text, and no request is issued.
func TestSearchPromptSpendsNothing(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.src.Calls()
	s.press("/")
	for _, spelling := range []string{"?", "/", "u", "s", "a", "r", "t", "j", "k"} {
		s.press(spelling)
	}
	if s.src.Calls() != before {
		t.Fatalf("a printable key while the prompt is open issued a request: the call count moved from %d to %d", before, s.src.Calls())
	}
	if s.m.prompt == nil {
		t.Fatal("a command key closed the prompt: every printable key is text")
	}
	if got := s.m.prompt.Text(); got != "?/usartjk" {
		t.Errorf("the prompt's text is %q, want the keys typed in as text", got)
	}
}

// TestSearchEscapeRestoresThePane: `esc` closes the prompt with the pane restored byte-identically
// and zero requests.
func TestSearchEscapeRestoresThePane(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.src.Calls()
	beforeFrame := s.frame()

	s.press("/")
	for _, r := range "page cache" {
		s.press(string(r))
	}
	s.press("esc")
	if s.m.prompt != nil {
		t.Fatal("`esc` left the prompt open")
	}
	if s.src.Calls() != before {
		t.Fatalf("`esc` issued %d requests: closing the prompt spends nothing", s.src.Calls()-before)
	}
	if got := s.frame(); got != beforeFrame {
		t.Errorf("`esc` restored a different frame:\nbefore:\n%s\nafter:\n%s", beforeFrame, got)
	}
}

// TestSearchEmptyQueryStaysOpen: `enter` on an empty query issues nothing and the prompt stays
// open — the documented no-op.
func TestSearchEmptyQueryStaysOpen(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.src.Calls()
	s.press("/")
	s.press("enter")
	if s.m.prompt == nil {
		t.Fatal("`enter` on an empty query closed the prompt")
	}
	if s.src.Calls() != before {
		t.Fatalf("`enter` on an empty query issued a request")
	}
}

// TestSearchOrderToggleIsAFrameToggle: `s` from a latest frame pushes a top frame for the same
// query at exactly one request, and `s` from a top frame pops back byte-identically with zero
// requests.
func TestSearchOrderToggleIsAFrameToggle(t *testing.T) {
	s := newShell(t, 80, 24)
	s.press("/")
	for _, r := range "page cache" {
		s.press(string(r))
	}
	s.press("enter")
	latestCalls := s.src.Calls()
	// The pane is what the pop restores byte-identically: the budget is a process fact and moves
	// with every request, so comparing whole frames would fail on the status line for a reason
	// that has nothing to do with the toggle.
	latestPane := s.pane()

	s.press("s")
	if s.src.Calls() != latestCalls+1 {
		t.Fatalf("`s` from latest issued %d requests past %d, want exactly one", s.src.Calls()-latestCalls, latestCalls)
	}
	if s.m.top().Kind != frameSearch || s.m.top().Search.Product() != domain.Top {
		t.Fatalf("`s` from latest left a %v frame under %v, want a top results frame", s.m.top().Kind, s.m.top().Search.Product())
	}
	if query, product, ok := s.src.LastSearch(); !ok || query != "page cache" || product != domain.Top {
		t.Fatalf("the search that went out is (%q, %v, %v), want (page cache, top, true)", query, product, ok)
	}
	s.used = s.src.Calls()
	if !strings.Contains(s.status(), `search: top "page cache"`) {
		t.Errorf("the top frame's status is %q, without the product named", s.status())
	}

	topCalls := s.src.Calls()
	s.press("s")
	if s.src.Calls() != topCalls {
		t.Fatalf("`s` from top issued %d requests: the pop spends nothing", s.src.Calls()-topCalls)
	}
	if got := s.pane(); got != latestPane {
		t.Errorf("`s` from top restored a different pane:\nbefore:\n%s\nafter:\n%s", latestPane, got)
	}
	if s.m.top().Search.Product() != domain.Latest {
		t.Errorf("the returned frame is under %v, want latest", s.m.top().Search.Product())
	}

}

// TestSearchOrderSaysNothingToSwitch: `s` on a home frame prints the notice and issues nothing.
func TestSearchOrderSaysNothingToSwitch(t *testing.T) {
	s := newShell(t, 80, 24)
	before := s.src.Calls()
	s.press("s")
	if s.src.Calls() != before {
		t.Fatalf("`s` on a home frame issued a request")
	}
	if !strings.Contains(s.status(), render.CopyNoticeNothingToSwitch) {
		t.Errorf("the status line is %q, without %q", s.status(), render.CopyNoticeNothingToSwitch)
	}
}

// TestSearchEmptyIsNoResultsInTheNotice: a successful search with zero rows shows an empty pane
// with `no results` in the notice — never a failure, and never a pane heading of those words.
func TestSearchEmptyIsNoResultsInTheNotice(t *testing.T) {
	s := newShell(t, 80, 24)
	s.src.EmptySearch()
	s.press("/")
	for _, r := range "nothing" {
		s.press(string(r))
	}
	s.press("enter")
	s.used = s.src.Calls()

	pane := ansi.Strip(s.pane())
	if strings.Contains(pane, render.CopyNoticeNoResults) {
		t.Errorf("the pane carries %q: empty is a notice, never a pane heading", render.CopyNoticeNoResults)
	}
	if strings.TrimSpace(pane) != "" {
		t.Errorf("the empty search's pane is %q, want an empty pane", pane)
	}
	if !strings.Contains(s.status(), render.CopyNoticeNoResults) {
		t.Errorf("the status line is %q, without %q", s.status(), render.CopyNoticeNoResults)
	}
	if strings.Contains(s.status(), render.StateWord(domain.FailureSessionExpired)) {
		t.Errorf("the empty search rendered as a failure")
	}
}

// TestSearchFailureNeverSaysNoResults: a failed search shows its own state and remedy, and never
// the empty-result words.
func TestSearchFailureNeverSaysNoResults(t *testing.T) {
	s := newShell(t, 80, 24)
	s.src.FailNext(errRefused)
	s.press("/")
	for _, r := range "page cache" {
		s.press(string(r))
	}
	s.press("enter")
	s.used = s.src.Calls()

	if strings.Contains(s.status(), render.CopyNoticeNoResults) {
		t.Errorf("a failed search's status is %q, and it carries the empty-result words", s.status())
	}
	if !strings.Contains(s.status(), render.StateWord(domain.FailureSessionExpired)) {
		t.Errorf("a failed search's status is %q, without the state's own words", s.status())
	}
}

// TestSearchSlashReseedsThePrompt: `/` with a search frame on top re-seeds the prompt with that
// frame's own query and does nothing else — no request, no new frame.
func TestSearchSlashReseedsThePrompt(t *testing.T) {
	s := newShell(t, 80, 24)
	s.press("/")
	for _, r := range "page cache" {
		s.press(string(r))
	}
	s.press("enter")
	before := s.src.Calls()
	frames := len(s.m.frames)

	s.press("/")
	if s.m.prompt == nil {
		t.Fatal("`/` over a results frame did not open the prompt")
	}
	if got := s.m.prompt.Text(); got != "page cache" {
		t.Errorf("the re-seeded prompt holds %q, want the frame's own query", got)
	}
	if s.src.Calls() != before || len(s.m.frames) != frames {
		t.Errorf("`/` over a results frame issued a request or pushed a frame")
	}
}

// budgetOf is the budget field as StatusLine would print it for these numbers: the helper that
// keeps the tracer's assertions reading as the words they assert rather than as a composition.
func budgetOf(used, limited int) string {
	if used == 1 {
		return strings.ReplaceAll(strings.ReplaceAll(render.CopyBudgetOne, "{n}", "1"), "{m}", itoa(limited))
	}
	return strings.ReplaceAll(strings.ReplaceAll(render.CopyBudgetMany, "{n}", itoa(used)), "{m}", itoa(limited))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
