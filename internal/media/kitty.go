package media

import (
	"context"
	"io"
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Kitty is the in-terminal renderer: it draws a post's image where the reading pane reserved the
// cells, instead of handing a URL to another program.
//
// It answers Open with a Result that carries no rows, because the placement is not a row of text: the
// pane prints KittyCells where the image goes and this renderer only talks to the terminal. That
// split is deliberate — a renderer that also decided layout would be a second layout engine, and the
// one the pane has is measured with the grammar everything else is measured with.
type Kitty struct {
	// Out is the terminal's own writer.
	Out io.Writer
	// Tmux is whether the output is behind a multiplexer, in which case every sequence is DCS-wrapped.
	Tmux bool
	// NextID allocates image ids. It is a function rather than a counter so the caller owns the
	// bookkeeping the panic path has to be able to delete.
	NextID func() int
	// Cap is the largest image this renderer will hold, in bytes.
	Cap int64
	// Draw is how the bytes are obtained: the pane hands a URL and gets PNG back. A nil Draw is a
	// renderer with no fetch path, and it answers with NoOpener so the pane prints the URL — the
	// honest answer for a build with no graphics support.
	Draw func(url string) ([]byte, error)
}

// Open transmits one attachment and places it. The rows it occupies are the caller's: the pane knows
// its own geometry and calls KittyCells with the size it reserved.
func (k *Kitty) Open(_ context.Context, ref domain.MediaRef) (Result, error) {
	if k.Out == nil || k.Draw == nil || k.NextID == nil {
		return Result{Outcome: NoOpener}, nil
	}
	png, err := k.Draw(ref.URL)
	if err != nil {
		return Result{Outcome: SpawnFailed}, nil
	}
	if k.Cap > 0 && int64(len(png)) > k.Cap {
		return Result{Outcome: SpawnFailed}, nil
	}
	id := k.NextID()
	_, err = io.WriteString(k.Out, k.wrap(KittyTransmit(id, png)))
	if err != nil {
		return Result{Outcome: SpawnFailed}, nil
	}
	return Result{Outcome: OpenedImage, ImageID: id}, nil
}

// OpenURL is the renderer's second entry point. A renderer that draws in the terminal cannot open a
// browser, so it answers with the same outcome a missing opener does and the pane prints the URL.
func (k *Kitty) OpenURL(context.Context, string) (Result, error) {
	return Result{Outcome: NoOpener}, nil
}

// DeleteAll removes every image the terminal is holding. It is the panic path's extra step.
func (k *Kitty) DeleteAll() error {
	if k.Out == nil {
		return nil
	}
	_, err := io.WriteString(k.Out, k.wrap(KittyDeleteAll()))
	return err
}

// wrap is one sequence, DCS-wrapped when there is a multiplexer between this process and the terminal.
func (k *Kitty) wrap(seq string) string {
	if k.Tmux {
		return TmuxWrap(seq)
	}
	return seq
}

// KittySupported reports whether this terminal speaks the protocol. It reads TERM and TERM_PROGRAM and
// never KITTY_WINDOW_ID, which Ghostty does not export — the probe measured that.
func KittySupported(env func(string) string) bool {
	if env == nil {
		return false
	}
	switch {
	case strings.Contains(env("TERM"), "ghostty"):
		return true
	case strings.Contains(env("TERM_PROGRAM"), "ghostty"):
		return true
	case strings.Contains(env("TERM"), "kitty"):
		return true
	case env("KITTY_WINDOW_ID") != "":
		// Another kitty-protocol terminal may export it; Ghostty does not, so this is a bonus
		// signal rather than the one the detection rests on.
		return true
	}
	return false
}
