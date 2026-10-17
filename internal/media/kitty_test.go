package media

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
)

// TestKittyOpenTransmitsAndAnswersWithTheID: a renderer with a writer and a draw function uploads the
// bytes once and tells the pane which id to print.
func TestKittyOpenTransmitsAndAnswersWithTheID(t *testing.T) {
	var out bytes.Buffer
	next := 0
	k := &Kitty{
		Out:    &out,
		NextID: func() int { next++; return next },
		Draw:   func(string) ([]byte, error) { return []byte("png"), nil },
	}
	res, err := k.Open(context.Background(), domain.MediaRef{URL: "https://pbs.twimg.com/a.png"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if res.Outcome != OpenedImage || res.ImageID != 1 {
		t.Fatalf("outcome = %v id = %d", res.Outcome, res.ImageID)
	}
	if len(res.Rows) != 0 {
		t.Fatal("the renderer must not supply rows: the pane prints the placeholder cells")
	}
	if !bytes.Contains(out.Bytes(), []byte("a=t,f=100,i=1")) {
		t.Fatalf("the upload is not in the output: %q", out.String())
	}
}

// TestKittyRefusesAnOversizeImage: the cap is enforced before a byte reaches the terminal.
func TestKittyRefusesAnOversizeImage(t *testing.T) {
	var out bytes.Buffer
	k := &Kitty{
		Out:    &out,
		Cap:    4,
		NextID: func() int { return 1 },
		Draw:   func(string) ([]byte, error) { return []byte("much more than four"), nil },
	}
	res, _ := k.Open(context.Background(), domain.MediaRef{URL: "https://pbs.twimg.com/a.png"})
	if res.Outcome != SpawnFailed {
		t.Fatalf("outcome = %v, want SpawnFailed", res.Outcome)
	}
	if out.Len() != 0 {
		t.Fatal("an image over the cap must not be written")
	}
}

// TestKittyWithNoDrawPathIsAnHonestNoOpener: a build with no graphics path says so, and the pane prints
// the URL rather than a blank cell.
func TestKittyWithNoDrawPathIsAnHonestNoOpener(t *testing.T) {
	k := &Kitty{Out: &bytes.Buffer{}, NextID: func() int { return 1 }}
	res, _ := k.Open(context.Background(), domain.MediaRef{URL: "https://pbs.twimg.com/a.png"})
	if res.Outcome != NoOpener {
		t.Fatalf("outcome = %v, want NoOpener", res.Outcome)
	}
	res, _ = k.OpenURL(context.Background(), "https://x.com/someone/status/1")
	if res.Outcome != NoOpener {
		t.Fatalf("OpenURL outcome = %v, want NoOpener", res.Outcome)
	}
}

// TestKittyDeleteAllWrapsUnderTmux: the panic path and the renderer must agree about the wrapping, or
// one of them is a payload tmux eats.
func TestKittyDeleteAllWrapsUnderTmux(t *testing.T) {
	var out bytes.Buffer
	k := &Kitty{Out: &out, Tmux: true}
	if err := k.DeleteAll(); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !bytes.HasPrefix(out.Bytes(), []byte("\x1bPtmux;")) {
		t.Fatalf("the delete was not wrapped: %q", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("\x1b\x1b_G")) {
		t.Fatalf("the inner ESC was not doubled: %q", out.String())
	}
}

// TestKittyDrawFailureIsAnOutcome: a fetch error becomes the same state a missing opener has, because
// the reader's remedy is the same.
func TestKittyDrawFailureIsAnOutcome(t *testing.T) {
	k := &Kitty{
		Out:    &bytes.Buffer{},
		NextID: func() int { return 1 },
		Draw:   func(string) ([]byte, error) { return nil, errors.New("no route") },
	}
	res, _ := k.Open(context.Background(), domain.MediaRef{URL: "https://pbs.twimg.com/a.png"})
	if res.Outcome != SpawnFailed {
		t.Fatalf("outcome = %v, want SpawnFailed", res.Outcome)
	}
}
