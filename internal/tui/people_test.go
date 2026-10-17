package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/fake"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/text"
)

// peopleShell is a shell driven by the adversarial people page set so account rows
// and page boundaries are reachable without inventing a second source.
func peopleShell(t *testing.T, w, h int) *shell {
	t.Helper()
	src := fake.AdversarialPeople()
	th := fake.ThreadShapes()
	s := &shell{t: t, src: src, th: th, now: frozen, w: w, h: h, term: &fakeTerm{}}
	s.m = New(Deps{
		Source:  bothSource{Source: src, threads: th},
		Threads: bothSource{Source: src, threads: th},
		Budget:  func() (int, int, time.Time) { return s.used, s.limited, s.now },
		Failure: testFailure,
		Now:     func() time.Time { return s.now },
		Term:    s.term,
		Display: Display{State: StateNormal, Cover: render.CoverServiceLog},
		Persist: func(state, cover string) { s.persisted = append(s.persisted, state+"/"+cover) },
		Media:   &fake.Renderer{Answer: media.Result{Outcome: media.OpenedImage}},
		Clip:    &fake.Clipboard{},
		Directory: &fake.Directory{Users: domain.UserPage{Rows: []domain.User{
			{ID: "u-1", ScreenName: "pump-sign-57", Name: "Pump Sign"},
			{ID: "u-2", ScreenName: "batch-restore-09", Name: "Batch Restore"},
		}}},
	})
	s.send(tea.WindowSizeMsg{Width: w, Height: h})
	s.run(s.m.Init())
	return s
}

// TestPeopleSearchRendersOneRowPerAccount: `u`, one request, one row per account, no bio/counts/stamp.
func TestPeopleSearchRendersOneRowPerAccount(t *testing.T) {
	s := peopleShell(t, 80, 24)
	before := s.src.Calls()

	s.press("u")
	if s.m.prompt == nil {
		t.Fatal("`u` did not open the prompt")
	}
	if s.src.Calls() != before {
		t.Fatalf("opening the people prompt issued a request")
	}
	row := strings.Split(s.pane(), "\n")[0]
	if !strings.Contains(ansi.Strip(row), "people:") {
		t.Errorf("the prompt's row is %q, and it does not name the people product", ansi.Strip(row))
	}

	for _, r := range "people" {
		s.press(string(r))
	}
	s.press("enter")
	if s.m.prompt != nil {
		t.Fatal("`enter` left the people prompt open")
	}
	if s.src.Calls() != before+1 {
		t.Fatalf("`enter` issued %d requests past %d, want exactly one SearchPeople", s.src.Calls()-before, before)
	}
	query, product, ok := s.src.LastSearch()
	if !ok || query != "people" || product != domain.People {
		t.Fatalf("the search that went out is (%q, %v, %v), want (people, People, true)", query, product, ok)
	}

	users := s.m.top().People.Walk().Rows()
	if len(users) == 0 {
		t.Fatal("the people frame holds no accounts after a successful SearchPeople")
	}
	pane := ansi.Strip(s.pane())
	nonBlank := 0
	for _, line := range strings.Split(pane, "\n") {
		if strings.TrimSpace(line) != "" {
			nonBlank++
		}
	}
	if nonBlank != len(users) {
		t.Errorf("the pane has %d non-blank rows for %d accounts", nonBlank, len(users))
	}
	for _, line := range strings.Split(pane, "\n") {
		low := strings.ToLower(line)
		if strings.Contains(low, "follower") || strings.Contains(low, "bio") {
			t.Errorf("a people row carries forbidden content: %q", line)
		}
	}
	if !strings.Contains(s.status(), `people: "people"`) {
		t.Errorf("the status line is %q, without the people list label", s.status())
	}

	// A display name longer than the row is cut with the truncation mark.
	var cut domain.User
	for _, u := range users {
		if text.Width(u.Name) > 40 {
			cut = u
			break
		}
	}
	if cut.ID == "" {
		t.Fatal("the adversarial people set carries no cut display name")
	}
	block := render.UserRow(cut, render.Plain, 80, cut.ID)
	if len(block.Rows) == 0 || !strings.Contains(block.Rows[0], render.MarkTruncation) {
		t.Errorf("the cut display name's row is %q, want the truncation mark", block.Rows)
	}
}

// TestAccountRowOpensTheProfile: enter on an account row costs exactly 2 more requests.
func TestAccountRowOpensTheProfile(t *testing.T) {
	s := peopleShell(t, 80, 24)
	before := s.src.Calls()
	s.press("u")
	for _, r := range "people" {
		s.press(string(r))
	}
	s.press("enter")
	if s.src.Calls() != before+1 {
		t.Fatalf("people search issued %d, want 1", s.src.Calls()-before)
	}
	afterSearch := s.src.Calls()
	s.press("enter")
	if s.m.top().Kind != frameProfile {
		t.Fatalf("`enter` on an account row left kind %v, want profile", s.m.top().Kind)
	}
	if s.src.Calls() != afterSearch+2 {
		t.Fatalf("opening the profile issued %d requests past the people search, want exactly 2", s.src.Calls()-afterSearch)
	}
}

// TestPeopleOrderToggle: `s` on a people frame prints nothing to switch and spends 0.
func TestPeopleOrderToggle(t *testing.T) {
	s := peopleShell(t, 80, 24)
	s.press("u")
	for _, r := range "people" {
		s.press(string(r))
	}
	s.press("enter")
	before := s.src.Calls()
	s.press("s")
	if s.src.Calls() != before {
		t.Fatalf("`s` on a people frame issued a request")
	}
	if !strings.Contains(s.status(), render.CopyNoticeNothingToSwitch) {
		t.Errorf("the status line is %q, without %q", s.status(), render.CopyNoticeNothingToSwitch)
	}
}

// TestPeopleEmptyIsNoResultsInTheNotice: a successful people read with zero rows.
func TestPeopleEmptyIsNoResultsInTheNotice(t *testing.T) {
	s := peopleShell(t, 80, 24)
	s.src.EmptySearch()
	s.press("u")
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
		t.Errorf("the empty people search's pane is %q, want an empty pane", pane)
	}
	if !strings.Contains(s.status(), render.CopyNoticeNoResults) {
		t.Errorf("the status line is %q, without %q", s.status(), render.CopyNoticeNoResults)
	}
}
