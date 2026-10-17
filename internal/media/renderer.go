// Package media is the I/O seam that opens a post's attachments.
//
// This package holds no words. An Outcome is a value; every user-visible sentence
// for it is a constant in internal/render/copy.go, exactly as domain.FailureKind
// is content and its words are presentation.
//
// It never fetches a media byte — the URL handed over is the one the payload
// carried, and the whole media path is one spawn — and it never retries: one
// keypress is one attempt, and a repair is a second keypress by a human.
//
// v1 ships one implementation (External). The v1.x addition is kitty.go plus one
// config value plus one wiring line in internal/cli/feed.go, with no list or
// detail code touched — that is the honest reading of MEDIA-02's "a one-file
// addition", and it is why Open takes a ref rather than a URL. The Kitty path is
// shaped by the measurements in .planning/research/TERMINAL-MEDIA-PROBE.md: a bare
// Ghostty renders an APC graphics sequence, tmux consumes a bare one, and a
// DCS-wrapped sequence renders with text drawn across the image because tmux does
// not model image cells. A scrolling pane in that configuration smears, so v1
// opens media outside the process and the seam keeps a rows half for the renderer
// that comes later.
package media

import (
	"context"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Outcome is one answer this phase can produce. The zero value is not a success.
type Outcome uint8

const (
	// OutcomeNone is the zero value: not a success, and not an answer anyone produced.
	OutcomeNone Outcome = iota
	// OpenedImage is an image handed to the configured opener.
	OpenedImage
	// OpenedVideo is a video handed a playable variant URL to mpv.
	OpenedVideo
	// OpenedGIF is an animated gif handed a playable variant URL to mpv.
	OpenedGIF
	// OpenedPoster is a video or gif whose response carried no variant, so the poster
	// frame is what was handed over and the words say so.
	OpenedPoster
	// OpenedURL is a non-attachment URL (today a permalink) handed to the opener.
	OpenedURL
	// NoOpener is a missing program name (empty opener for an image).
	NoOpener
	// NoProgram is a named program missing from PATH. A missing program is a state,
	// not a failure — MEDIA-03's fallback exists for it.
	NoProgram
	// SpawnFailed is a spawn that did not start, or a program that exited
	// non-zero. When Open returns Opened* with Done set, a later non-zero Wait
	// is reported as SpawnFailed by the pane without blocking Open.
	SpawnFailed
)

// String names each value in lowercase words for tests and logs. The sentence a
// user reads is never this method.
func (o Outcome) String() string {
	switch o {
	case OutcomeNone:
		return "none"
	case OpenedImage:
		return "opened image"
	case OpenedVideo:
		return "opened video"
	case OpenedGIF:
		return "opened gif"
	case OpenedPoster:
		return "opened poster"
	case OpenedURL:
		return "opened url"
	case NoOpener:
		return "no opener"
	case NoProgram:
		return "no program"
	case SpawnFailed:
		return "spawn failed"
	}
	return "outcome"
}

// Result carries an outcome value and optional pane rows. A renderer that draws
// inside the terminal (the v1.x Kitty implementation, IMG-01) supplies rows for
// the pane to print in a detail frame and spawns nothing; a renderer that spawns
// a program returns a notice and no rows. Rows are printed only in a detail
// frame, never while a scrollable list is on screen, because the probe measured
// text being drawn across a scrolled image under tmux.
//
// Done, when non-nil, delivers the child's Wait error exactly once after Open
// returned a success. The pane observes it asynchronously so a start-then-fail
// opener still becomes SpawnFailed / URL fallback without blocking the frame.
type Result struct {
	Outcome Outcome
	Rows    []string
	Done    <-chan error
}

// Renderer is the one-file seam: Open is one call per attachment; OpenURL is the
// second entry point for a URL that is not an attachment (today a permalink).
// v1 ships one implementation; the v1.x addition is kitty.go plus one config
// value plus one wiring line in internal/cli/feed.go, with no list or detail
// code touched.
//
// The v1.x in-terminal renderer answers OpenURL with its own honest outcome: a
// renderer that draws in the terminal cannot open a browser, so the pane prints
// the URL — the same behaviour a missing opener has.
type Renderer interface {
	Open(ctx context.Context, ref domain.MediaRef) (Result, error)
	OpenURL(ctx context.Context, url string) (Result, error)
}
