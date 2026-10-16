package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// TestThePromptRowNamesTheProduct: the prefix is the grammar's marker with the product named,
// before any request is paid for, and the typed text follows it after two spaces.
func TestThePromptRowNamesTheProduct(t *testing.T) {
	plainEnv(t)
	row, caret := PromptRow(Plain, domain.Latest, "page cache", 80)
	if want := " search: latest  page cache"; row != want {
		t.Errorf("the prompt row is %q, want %q", row, want)
	}
	if want := text.Width(" search: latest  page cache") + 1; caret != want {
		t.Errorf("the caret column is %d, want %d (after the last typed cell)", caret, want)
	}
}

// TestThePromptCaretStaysInTheRow is the caret's bound: within 1..W for every prefix and query,
// with the row itself never exceeding W — the tail of a long query is what shows, so the caret
// stays on screen. The queries are the width model's own cases: empty, CJK, emoji, and a query far
// longer than the row.
func TestThePromptCaretStaysInTheRow(t *testing.T) {
	plainEnv(t)
	queries := map[string]string{
		"empty":     "",
		"plain":     "page cache",
		"cjk":       "日本語の検索",
		"emoji":     "go 🚀 fast",
		"combining": "café au lait",
		"long":      strings.Repeat("x", 200),
	}
	for _, w := range []int{1, 2, 3, 5, 8, 17, 55, 77, 96} {
		for name, query := range queries {
			row, caret := PromptRow(Plain, domain.Latest, query, w)
			if caret < 1 || caret > w {
				t.Errorf("at w=%d with the %s query the caret is %d, outside 1..%d", w, name, caret, w)
			}
			if got := text.Width(row); got > w {
				t.Errorf("at w=%d with the %s query the row is %d cells wide: %q", w, name, got, row)
			}
		}
	}
}

// TestThePromptShowsTheTail: a query longer than the row shows its end, cut at a cluster boundary,
// and the caret lands on the row's last cell.
func TestThePromptShowsTheTail(t *testing.T) {
	plainEnv(t)
	query := "the quick brown fox jumps over the lazy dog"
	row, caret := PromptRow(Plain, domain.Latest, query, 30)
	shown := strings.TrimPrefix(ansi.Strip(row), " search: latest  ")
	if !strings.HasSuffix(query, shown) {
		t.Errorf("the shown text %q is not the query's tail", shown)
	}
	if caret != 30 {
		t.Errorf("a full row's caret is %d, want the last cell (30)", caret)
	}
}

// TestTheListLabelEchoesTheQueryCapped: the status field's echo is at most 32 cells with the
// truncation mark in its last cell, cut at a cluster boundary — CJK and emoji included.
func TestTheListLabelEchoesTheQueryCapped(t *testing.T) {
	plainEnv(t)
	for _, tc := range []struct {
		name  string
		query string
		want  string
	}{
		{"short", "page cache", `search: latest "page cache"`},
		{"exactly the cap", strings.Repeat("x", 32), `search: latest "` + strings.Repeat("x", 32) + `"`},
		{"over the cap", strings.Repeat("x", 40), `search: latest "` + strings.Repeat("x", 31) + `~"`},
		{"cjk over the cap", strings.Repeat("検", 20), `search: latest "` + strings.Repeat("検", 15) + `~"`},
		{"emoji over the cap", strings.Repeat("🚀", 20), `search: latest "` + strings.Repeat("🚀", 15) + `~"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ListLabelSearch(Plain, domain.Latest, tc.query)
			if got != tc.want {
				t.Errorf("the list field is %q, want %q", got, tc.want)
			}
		})
	}

	// The product word is the field's own.
	if got := ListLabelSearch(Plain, domain.Top, "page cache"); !strings.Contains(got, `"page cache"`) || !strings.Contains(got, "top") {
		t.Errorf("the top frame's list field is %q, want the product and the query named", got)
	}
}

// TestTheListLabelFitsAtEveryWidth is A4's arm for the query echo: at every width the property
// holds — the echo never exceeds its cap, and a capped echo ends with the mark.
func TestTheListLabelFitsAtEveryWidth(t *testing.T) {
	plainEnv(t)
	queries := []string{"page cache", "日本語の検索結果 🚀", strings.Repeat("x", 200)}
	for _, w := range []int{1, 2, 3, 5, 8, 55, 77, 96} {
		_ = w // the echo's cap does not depend on the pane's width
		for _, query := range queries {
			label := ListLabelSearch(Plain, domain.Latest, query)
			echo := label[strings.Index(label, `"`)+1 : strings.LastIndex(label, `"`)]
			if got := text.Width(echo); got > ListLabelQueryCells {
				t.Errorf("the echo of %q is %d cells, over the %d-cell cap", query, got, ListLabelQueryCells)
			}
			if text.Width(query) > ListLabelQueryCells && !strings.HasSuffix(echo, MarkTruncation) {
				t.Errorf("the capped echo of %q is %q, without the truncation mark in its last cell", query, echo)
			}
		}
	}
}

// TestEveryGrammarHasThePromptsWords is the totality arm for the new marker fields: a grammar
// without them renders a blank prompt or a blank loading statement, which is a state without
// words.
func TestEveryGrammarHasThePromptsWords(t *testing.T) {
	plainEnv(t)
	grammars := append([]Grammar{Plain}, func() []Grammar {
		out := make([]Grammar, 0, 4)
		for _, style := range CoverStyles() {
			out = append(out, style.Grammar())
		}
		return out
	}()...)
	for _, g := range grammars {
		if !strings.Contains(g.Markers.PromptSearch, PlaceholderProduct) {
			t.Errorf("the %s grammar's prompt prefix is %q, without the {product} slot", g.Name, g.Markers.PromptSearch)
		}
		if g.Markers.LoadingResults == "" {
			t.Errorf("the %s grammar has no loading statement for a results frame", g.Name)
		}
	}
}
