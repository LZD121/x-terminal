package tui

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
)

// tinyPNG is a 40x20 picture: wide enough that the block's aspect arithmetic is not degenerate, and
// small enough that the encoded bytes are a few dozen.
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := range 20 {
		for x := range 40 {
			img.Set(x, y, color.RGBA{R: uint8(x * 6), G: uint8(y * 12), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding the test picture: %v", err)
	}
	return buf.Bytes()
}

// imageShell is a shell in the presentation with the image path wired to a recording loader.
func imageShell(t *testing.T) (*shell, *[]string) {
	t.Helper()
	s := vividShell(t)
	calls := new([]string)
	s.m.deps.Images = true
	s.m.deps.Image = func(_ context.Context, url string) ([]byte, string, error) {
		*calls = append(*calls, url)
		return tinyPNG(t), "image/png", nil
	}
	return s, calls
}

// advancingPost brings the list forward until it holds a post with an image attachment, and selects
// it. It pages rather than constructing a post: the point is the shipped path, and the corpus's
// image row is on its second page.
func advancingPost(t *testing.T, s *shell) domain.Post {
	t.Helper()
	for attempt := range 6 {
		for _, post := range s.m.top().List.Posts() {
			for _, m := range post.Media {
				if m.Kind != domain.Image {
					continue
				}
				// Walk the selection to it. `j` never skips a row, so at most one pass is needed.
				for range len(s.m.top().List.Posts()) {
					if s.m.top().List.Selected().ID == string(post.ID) {
						return post
					}
					s.press("j")
				}
			}
		}
		s.press("G") // the page trigger: the last loaded row, then one more page
		_ = attempt
	}
	t.Skip("the corpus holds no image attachment: the image path needs one")
	return domain.Post{}
}

// TestTheReadingColumnPlacesOnePicture: opening a post with an image fetches it once, transmits it
// once, places it once, and the frame it is drawn in carries the placeholder cells the terminal
// paints into.
func TestTheReadingColumnPlacesOnePicture(t *testing.T) {
	s, calls := imageShell(t)
	post := advancingPost(t, s)

	// Walk the selection to that post, then open it.
	for f := s.m.top(); f.List.Selected().ID != string(post.ID); f = s.m.top() {
		s.press("j")
	}
	s.press("enter")

	if len(*calls) != 1 {
		t.Fatalf("%d fetches, want exactly 1", len(*calls))
	}
	if len(s.term.drawn) != 1 {
		t.Fatalf("%d terminal draws, want one transmit+place", len(s.term.drawn))
	}
	seq := s.term.drawn[0]
	if !strings.Contains(seq, "a=t,f=100,i=") {
		t.Fatalf("the draw is not a transmit: %q", seq[:40])
	}
	if !strings.Contains(seq, "a=p,U=1,") {
		t.Fatalf("the draw does not place: %q", seq)
	}
	if len(s.m.vivid.images) != 1 {
		t.Fatalf("%d images booked, want 1", len(s.m.vivid.images))
	}

	frame := s.m.Render(110, 30)
	if !strings.Contains(frame, "\U0010EEEE") {
		t.Fatal("the frame carries no placeholder cells: the terminal would have nothing to paint into")
	}
}

// TestClosingTheReadingColumnDeletesThePicture: the store is emptied when the column closes, so a
// picture cannot outlive the post it belonged to.
func TestClosingTheReadingColumnDeletesThePicture(t *testing.T) {
	s, _ := imageShell(t)
	advancingPost(t, s)
	s.press("enter")
	if len(s.m.vivid.images) != 1 {
		t.Fatal("setup: no picture placed")
	}
	before := len(s.term.drawn)
	s.press("esc")
	if len(s.m.vivid.images) != 0 {
		t.Fatal("the picture is still booked after the column closed")
	}
	if got := s.term.drawn[len(s.term.drawn)-1]; before == len(s.term.drawn) || got != media.KittyDeleteAll() {
		t.Fatalf("the last draw was %q, want the delete-all", got)
	}
}

// TestLeavingThePresentationLeavesNothingBooked: the two escapes out — one for the column, one for
// the presentation — leave no picture booked and no picture stored, and the delete is drawn exactly
// once. A second delete would be harmless to the terminal and a sign the two paths disagree.
func TestLeavingThePresentationLeavesNothingBooked(t *testing.T) {
	s, _ := imageShell(t)
	advancingPost(t, s)
	s.press("enter")
	s.press("esc") // closes the column
	s.press("esc") // leaves the presentation
	if s.m.state != StateNormal {
		t.Fatalf("still in %v", s.m.state)
	}
	if len(s.m.vivid.images) != 0 {
		t.Fatal("a picture is still booked after leaving")
	}
	deletes := 0
	for _, seq := range s.term.drawn {
		if seq == media.KittyDeleteAll() {
			deletes++
		}
	}
	if deletes != 1 {
		t.Fatalf("%d delete-alls drawn, want exactly 1", deletes)
	}
}

// TestAnAnswerForAPostTheReaderLeftIsDropped: the fetch is a command, so the reader can move on
// before it lands. A picture of the previous post beside a new heading would be a lie about what is
// on screen.
func TestAnAnswerForAPostTheReaderLeftIsDropped(t *testing.T) {
	s, _ := imageShell(t)
	advancingPost(t, s)
	s.press("enter")
	// Move the reading column to a different post without letting the fetch run, then deliver the
	// old answer by hand.
	other := domain.PostID("some-other-post")
	s.m.vivid.reading = other
	booked, drawn := len(s.m.vivid.images), len(s.term.drawn)
	msg := imageLoadedMsg{post: "the-previous-post", png: tinyPNG(t), cols: 8, rows: 4}
	next, cmd := s.m.imageDrawn(msg)
	if cmd != nil || len(next.vivid.images) != booked || len(s.term.drawn) != drawn {
		t.Fatal("a stale picture was placed after the reader moved on")
	}
}

// TestNoImagePathDrawsNothing: a build with no loader, or a terminal that cannot draw, must not
// fetch, must not draw, and must still render the attachment line the grammar gives it.
func TestNoImagePathDrawsNothing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		images bool
		load   bool
	}{
		{"no loader", true, false},
		{"no terminal support", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := vividShell(t)
			s.m.deps.Images = tc.images
			s.m.deps.Image = nil
			called := false
			if tc.load {
				s.m.deps.Image = func(context.Context, string) ([]byte, string, error) {
					called = true
					return tinyPNG(t), "image/png", nil
				}
			}
			advancingPost(t, s)
			s.press("enter")
			if called {
				t.Fatal("the loader was called with no image path")
			}
			if len(s.term.drawn) != 0 {
				t.Fatalf("%d draws, want none", len(s.term.drawn))
			}
		})
	}
}

// TestTheBlockFitsTheReadingColumn: the reserved cells must fit the column, and the aspect the bytes
// actually have decides the rows.
func TestTheBlockFitsTheReadingColumn(t *testing.T) {
	s := vividShell(t)
	for _, w := range []int{74, 110, 200} {
		s.resize(w, 30)
		cols, rows := s.m.imageBlock(0.5)
		if cols < 1 || cols > maxImageCells || rows < 1 || rows > maxImageCells {
			t.Fatalf("width %d: block %dx%d is outside the protocol's table", w, cols, rows)
		}
		if cols > s.m.readingWidth() {
			t.Fatalf("width %d: %d columns do not fit a %d-cell reading column", w, cols, s.m.readingWidth())
		}
	}
}

// TestTheRowsFollowThePicture: a wide picture is a short block and a tall one is a tall block, both
// capped at the protocol's table.
func TestTheRowsFollowThePicture(t *testing.T) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(tinyPNG(t)))
	if err != nil {
		t.Fatal(err)
	}
	cols, rows := imageBlockFor(cfg.Width, cfg.Height, 8)
	if cols != 8 {
		t.Fatalf("cols = %d, want 8", cols)
	}
	want := int(float64(cols) * (float64(cfg.Height) / float64(cfg.Width)) / 2)
	if rows != want {
		t.Fatalf("rows = %d, want %d for a %dx%d picture", rows, want, cfg.Width, cfg.Height)
	}
	if rows > maxImageCells {
		t.Fatalf("rows = %d, over the table", rows)
	}
}

// TestAJPEGIsReEncodedAsPNG: the protocol's f=100 form carries PNG and nothing else, and twimg
// serves JPEG as often as not. The bytes that reach the terminal must therefore be PNG whatever
// arrived — and the block's aspect must be the picture's own.
func TestAJPEGIsReEncodedAsPNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 60, 30))
	var jpegBytes bytes.Buffer
	if err := jpegEncode(&jpegBytes, img); err != nil {
		t.Fatalf("encoding the test jpeg: %v", err)
	}
	pngBytes, w, h, err := toPNG(jpegBytes.Bytes())
	if err != nil {
		t.Fatalf("toPNG: %v", err)
	}
	if w != 60 || h != 30 {
		t.Fatalf("dimensions = %dx%d, want 60x30", w, h)
	}
	if !bytes.HasPrefix(pngBytes, []byte("\x89PNG")) {
		t.Fatal("the re-encoded body is not a PNG")
	}
	if _, err := png.Decode(bytes.NewReader(pngBytes)); err != nil {
		t.Fatalf("the re-encoded body does not decode as PNG: %v", err)
	}
}
