package render

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
)

// TestPhase5MarkersFillEveryGrammar: every shipped grammar fills the new row templates
// and statements — no empty template where a statement is expected.
func TestPhase5MarkersFillEveryGrammar(t *testing.T) {
	grammars := append([]Grammar{Plain}, func() []Grammar {
		out := make([]Grammar, 0, len(CoverStyles()))
		for _, style := range CoverStyles() {
			out = append(out, style.Grammar())
		}
		return out
	}()...)

	need := []struct {
		name string
		get  func(Grammar) string
	}{
		{"LoadingResults", func(g Grammar) string { return g.Markers.LoadingResults }},
		{"LoadingProfile", func(g Grammar) string { return g.Markers.LoadingProfile }},
		{"NoAuthor", func(g Grammar) string { return g.Markers.NoAuthor }},
		{"NoPosts", func(g Grammar) string { return g.Markers.NoPosts }},
		{"PromptSearch", func(g Grammar) string { return g.Markers.PromptSearch }},
		{"PromptPeople", func(g Grammar) string { return g.Markers.PromptPeople }},
	}
	for _, g := range grammars {
		for _, n := range need {
			if n.get(g) == "" {
				t.Errorf("grammar %q leaves Markers.%s empty", g.Name, n.name)
			}
		}
		if len(userHeadTemplates(g)) == 0 {
			t.Errorf("grammar %q resolves no user-head template", g.Name)
		}
		if g.Name == Plain.Name && len(g.AccountFacts) == 0 {
			t.Errorf("Plain AccountFacts is empty")
		}
		// Notice vocabulary used by the new surfaces must resolve under every cover.
		if NoticeLine(coverOf(g), NoticeNoResults, 0) == "" {
			t.Errorf("grammar %q resolves NoticeNoResults to empty", g.Name)
		}
		if NoticeLine(coverOf(g), NoticeNothingToSwitch, 0) == "" {
			t.Errorf("grammar %q resolves NoticeNothingToSwitch to empty", g.Name)
		}
	}
}

func coverOf(g Grammar) CoverStyle {
	if s, ok := CoverOfGrammar(g); ok {
		return s
	}
	return ""
}

// TestCoveredFramesOmitPlainPeopleLiterals: a covered people frame contains none of the
// plain loading/empty literals (Phase 3 frame-scan rule, extended).
func TestCoveredFramesOmitPlainPeopleLiterals(t *testing.T) {
	plainEnv(t)
	users := []domain.User{
		{ID: "1", ScreenName: "alice", Name: "Alice"},
		{ID: "2", ScreenName: "bob", Name: "Bob"},
	}
	for _, style := range CoverStyles() {
		g := style.Grammar()
		frame := strings.Join(rowsOf(UserRows(users, g, 80, "1")), "\n")
		for _, lit := range []string{MarkerLoadingResults, CopyNoticeNoResults} {
			if strings.Contains(frame, lit) {
				t.Errorf("covered grammar %q people frame holds plain literal %q", g.Name, lit)
			}
		}
	}
}
