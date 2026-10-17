package vivid

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
)

// update rewrites the golden files instead of comparing. It is the same flag name the render
// package's goldens use, so one invocation refreshes both corpora.
var update = flag.Bool("update", false, "rewrite the golden frames")

// TestGoldenFrames pins the three shapes the presentation has: the two-column surface, the narrow
// one-column surface, and the reading pane with a conversation open. They are stored as bytes for
// the reason the render package's are — the frame is the spec, and a diff of it is the review.
func TestGoldenFrames(t *testing.T) {
	posts := []render.Block{
		{PostID: "101", Rows: []string{"> mira  9:14", "  Shipped the parser.", "  ~ a body line that is long enough to be cut in a narrow column"}},
		{PostID: "102", Rows: []string{"  dev  9:20", "  Fixed the widths."}},
	}
	thread := []render.Block{
		{PostID: "101", Rows: []string{"  mira  9:14", "  Shipped the parser."}},
		{PostID: "103", Rows: []string{"  ana  9:21", "  Does it handle wide glyphs?"}},
	}
	base := View{
		Dest:         DestHome,
		ListName:     "home",
		List:         posts,
		Reading:      thread,
		ReadingTitle: "mira / 101",
		Marks:        Marks{New: map[domain.PostID]bool{"101": true}, Opened: map[domain.PostID]bool{"102": true}},
		Status:       render.StatusFields{List: "home", Selected: 1, Loaded: 2, Used: 12},
	}
	cases := []struct {
		name string
		view View
	}{
		{"two-column", withSize(base, 100, 12)},
		{"narrow", withSize(View{Dest: DestHome, ListName: "home", List: posts}, 50, 8)},
		{"reading-empty", withSize(View{Dest: DestSearch, ListName: "search: go", List: posts, Notice: "open a post to read it here"}, 100, 10)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Render(c.view)
			path := filepath.Join("testdata", c.name+".golden")
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v (run with -update to record it)", path, err)
			}
			if string(want) != got {
				t.Fatalf("%s differs from the recorded frame:\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
			}
		})
	}
}

// withSize is the view at a size. It exists so the table above reads as shapes rather than as two
// trailing literals on every line.
func withSize(v View, w, h int) View {
	v.Width, v.Height = w, h
	return v
}
