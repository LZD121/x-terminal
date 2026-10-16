package media

import (
	"context"
	"encoding/base64"
	"io"
)

// ClipboardOutcome reports which half of the copy chain actually ran.
// The zero value is not a success.
type ClipboardOutcome uint8

const (
	// ClipboardNone is the zero value: not an answer anyone produced.
	ClipboardNone ClipboardOutcome = iota
	// Copied means a successful copy ran with only one link: OSC 52 was
	// written and wl-copy did not also succeed, or wl-copy alone succeeded
	// because TERM was empty/dumb so the escape was skipped.
	Copied
	// CopiedWithWLCopy means both the OSC 52 escape and a successful wl-copy ran.
	CopiedWithWLCopy
	// NoClipboard means neither link could run: no escape was written and wl-copy
	// did not copy.
	NoClipboard
)

// String names each value in lowercase words for tests and logs. The sentence a
// user reads is never this method.
func (o ClipboardOutcome) String() string {
	switch o {
	case ClipboardNone:
		return "none"
	case Copied:
		return "copied"
	case CopiedWithWLCopy:
		return "copied with wl-copy"
	case NoClipboard:
		return "no clipboard"
	}
	return "clipboard outcome"
}

// Clipboard is the REND-07 chain that copies a post's permalink.
//
// Only a permalink ever reaches the sink: no post body, no handle list, no
// session material. A clipboard manager may keep what this writes; the "no
// trace" promise covers the timeline, not the user's own clipboard.
//
// Out, Env, LookPath and Run are injected so a test writes nothing and spawns
// nothing. Out is the program's own output file (the same writer internal/term's
// sink uses).
type Clipboard struct {
	Out      io.Writer
	Env      func(string) string
	LookPath func(string) (string, error)
	Run      func(ctx context.Context, name string, args []string, stdin string) error
}

// Copy hands url to the chain and reports which links ran. It returns no
// error: there is nothing a caller could do with one, and NoClipboard is the
// honest report when neither link could.
//
//  1. OSC 52 is written unless $TERM is empty or "dumb" — the same condition
//     under which this program emits no other escape (and deliberately not
//     NO_COLOR). The escape is one Write of
//     "\x1b]52;c;{base64 of the URL}\x1b\\". A terminal that ignores it is
//     indistinguishable from one that honours it, so no detection is attempted.
//  2. wl-copy additionally runs when $WAYLAND_DISPLAY is set and LookPath
//     resolves it, with the URL on stdin. A non-zero exit means it did not copy.
//  3. Nothing else, ever — no xclip, no xsel, no clipboard command from config.
//
// Outcome: CopiedWithWLCopy when both ran; Copied when OSC 52 alone wrote, or
// when wl-copy alone succeeded under empty/dumb TERM; NoClipboard when neither.
func (c Clipboard) Copy(ctx context.Context, url string) ClipboardOutcome {
	wrote := c.writeOSC52(url)
	wl := c.runWLCopy(ctx, url)
	switch {
	case wrote && wl:
		return CopiedWithWLCopy
	case wrote:
		return Copied
	case wl:
		return Copied
	default:
		return NoClipboard
	}
}

func (c Clipboard) writeOSC52(url string) bool {
	if c.Out == nil || c.Env == nil {
		return false
	}
	term := c.Env("TERM")
	if term == "" || term == "dumb" {
		return false
	}
	esc := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(url)) + "\x1b\\"
	_, err := io.WriteString(c.Out, esc)
	return err == nil
}

func (c Clipboard) runWLCopy(ctx context.Context, url string) bool {
	if c.Env == nil || c.LookPath == nil || c.Run == nil {
		return false
	}
	if c.Env("WAYLAND_DISPLAY") == "" {
		return false
	}
	if _, err := c.LookPath("wl-copy"); err != nil {
		return false
	}
	return c.Run(ctx, "wl-copy", nil, url) == nil
}
