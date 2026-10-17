package render

import (
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// TestLinksMenuRows: the heading, one numbered row per kind, the scheme dropped from a URL, and every
// row measured to the pane.
func TestLinksMenuRows(t *testing.T) {
	links := []domain.Link{
		{Kind: domain.LinkURL, Target: "https://example.invalid/a/very/long/path/that/will/not/fit/in/a/narrow/pane"},
		{Kind: domain.LinkHashtag, Target: "go"},
		{Kind: domain.LinkMention, Target: "someone"},
	}
	for _, w := range []int{40, 80, 200} {
		rows := Links(Plain, w, links)
		if len(rows) != 4 {
			t.Fatalf("width %d: %d rows, want a heading and three links", w, len(rows))
		}
		if !strings.Contains(rows[0], MarkerLinksHeading) {
			t.Fatalf("the heading is %q", rows[0])
		}
		if !strings.HasPrefix(rows[1], " 1  example.invalid/") {
			t.Fatalf("the URL row is %q: the scheme should not be printed", rows[1])
		}
		if !strings.HasPrefix(rows[2], " 2  #go") {
			t.Fatalf("the tag row is %q", rows[2])
		}
		if !strings.HasPrefix(rows[3], " 3  @someone") {
			t.Fatalf("the mention row is %q", rows[3])
		}
		for i, row := range rows {
			if got := text.Width(row); got > w {
				t.Fatalf("width %d row %d is %d cells", w, i, got)
			}
		}
	}
}

// TestLinksMenuStopsAtNine: a digit reaches nine, so a tenth link is a sentence rather than a row
// the reader cannot choose.
func TestLinksMenuStopsAtNine(t *testing.T) {
	links := make([]domain.Link, 12)
	for i := range links {
		links[i] = domain.Link{Kind: domain.LinkHashtag, Target: "t"}
	}
	rows := Links(Plain, 80, links)
	if len(rows) != 1+9+1 {
		t.Fatalf("%d rows, want a heading, nine links and the sentence", len(rows))
	}
	if !strings.Contains(rows[len(rows)-1], "more links") {
		t.Fatalf("the last row is %q", rows[len(rows)-1])
	}
}
