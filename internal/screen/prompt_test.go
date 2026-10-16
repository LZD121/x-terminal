package screen

import (
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
)

// TestThePromptEditsByCluster is the backspace rule: one keypress drops exactly one grapheme
// cluster — an `e` with its combining mark, a ZWJ family and an emoji each go in one step, and no
// half cluster is ever left on the row.
func TestThePromptEditsByCluster(t *testing.T) {
	for _, tc := range []struct {
		name  string
		typed string
		want  string
	}{
		{"an ASCII letter", "page cache", "page cach"},
		{"e with its combining mark", "café", "caf"},
		{"a ZWJ family", "a👨‍👩‍👧‍👦", "a"},
		{"an emoji", "go 🚀", "go "},
		{"an empty field", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPrompt(domain.Latest, tc.typed)
			if got := p.Backspace().Text(); got != tc.want {
				t.Errorf("Backspace of %q is %q, want %q", tc.typed, got, tc.want)
			}
		})
	}
}

// TestThePromptDropsWordsAndClears is ctrl+w and ctrl+u: the last word goes, trailing spaces with
// it, and the row clears whole.
func TestThePromptDropsWordsAndClears(t *testing.T) {
	for _, tc := range []struct {
		name  string
		typed string
		want  string
	}{
		{"the last word", "page cache", "page "},
		{"trailing spaces first", "page cache  ", "page "},
		{"the only word", "cache", ""},
		{"an empty field", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPrompt(domain.Latest, tc.typed)
			if got := p.DeleteWord().Text(); got != tc.want {
				t.Errorf("DeleteWord of %q is %q, want %q", tc.typed, got, tc.want)
			}
		})
	}
	if got := NewPrompt(domain.Latest, "page cache").Clear().Text(); got != "" {
		t.Errorf("Clear leaves %q, want the empty row", got)
	}
}

// TestThePromptIsOneRow is the paste rule: a newline in a pasted run arrives as a space, because
// the field is one row and a paste is not a way to smuggle a second one in.
func TestThePromptIsOneRow(t *testing.T) {
	p := NewPrompt(domain.Latest, "page")
	p = p.Insert(" cache\r\nwith\nnewlines\rover")
	if got, want := p.Text(), "page cache with newlines over"; got != want {
		t.Errorf("a pasted run arrives as %q, want %q", got, want)
	}
}

// TestThePromptSubmitsNothingEmpty is the no-op arm: an empty or all-space query submits nothing,
// and a real one submits trimmed.
func TestThePromptSubmitsNothingEmpty(t *testing.T) {
	for _, tc := range []struct {
		typed string
		want  string
		ok    bool
	}{
		{"", "", false},
		{"   ", "", false},
		{"  page cache  ", "page cache", true},
	} {
		got, ok := NewPrompt(domain.Latest, tc.typed).Submitted()
		if ok != tc.ok || got != tc.want {
			t.Errorf("Submitted of %q is (%q, %v), want (%q, %v)", tc.typed, got, ok, tc.want, tc.ok)
		}
	}
}

// TestThePromptCarriesItsProductFromTheMomentItOpens: the product is a value of the field, so the
// row can name it before the request is paid for, and the seed is the text a frame re-opens with.
func TestThePromptCarriesItsProductFromTheMomentItOpens(t *testing.T) {
	p := NewPrompt(domain.Top, "page cache")
	if p.Product() != domain.Top {
		t.Errorf("the prompt's product is %v, want top", p.Product())
	}
	if p.Text() != "page cache" {
		t.Errorf("the prompt's seed is %q, want %q", p.Text(), "page cache")
	}
}
