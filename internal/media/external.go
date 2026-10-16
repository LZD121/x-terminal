package media

import (
	"context"

	"github.com/LZD121/x-terminal/internal/domain"
)

// videoProgram is the fixed player for video and animated media. It is fixed
// rather than a second config key: the configuration surface stays at the one
// opener the project already ships (config.MediaOpener), and --no-terminal is
// fixed because a player that seizes the pane would fight Bubble Tea for the
// terminal.
const videoProgram = "mpv"

// External is v1's only Renderer: it hands each attachment to a program and
// returns a notice, never rows.
//
// Opener is config.MediaOpener (default xdg-open). Env and LookPath are fields
// rather than direct calls so the dispatch's decisions are testable on a machine
// with no xdg-open. Run starts the child and must return without waiting on it:
// the production wiring starts the process, returns Start's error, and delivers
// Wait on the returned channel so the pane never blocks and no zombie accumulates.
// A program's stdout is not this program's data — nothing here reads it.
type External struct {
	Opener   string
	Env      func(string) string
	LookPath func(string) (string, error)
	Run      func(ctx context.Context, name string, args []string) (<-chan error, error)
}

// target is the URL handed to a program and (plan 02) the URL a failed action
// prints: Playable when the response carried a variant, else the poster/URL.
// One function so the two can never disagree.
func target(ref domain.MediaRef) string {
	if ref.Playable != "" {
		return ref.Playable
	}
	return ref.URL
}

// Open resolves the program, starts it, and returns. A missing program is
// NoProgram with a nil error — a state, not a failure. A spawn that fails to
// start is SpawnFailed. A program that starts successfully returns the opened
// outcome with Done set to Wait's channel; a later non-zero exit is observed
// by the pane as SpawnFailed without blocking Open.
func (e External) Open(ctx context.Context, ref domain.MediaRef) (Result, error) {
	name, args, out, ok := e.dispatch(ref)
	if !ok {
		return Result{Outcome: out}, nil
	}
	if e.LookPath != nil {
		if _, err := e.LookPath(name); err != nil {
			return Result{Outcome: NoProgram}, nil
		}
	}
	run := e.Run
	if run == nil {
		return Result{Outcome: SpawnFailed}, nil
	}
	done, err := run(ctx, name, args)
	if err != nil {
		return Result{Outcome: SpawnFailed}, nil
	}
	return Result{Outcome: out, Done: done}, nil
}

// OpenURL hands a non-attachment URL (today a permalink) to the same configured
// opener with the same one-argument, no-shell rule as an image. Empty Opener is
// NoOpener; a missing program is NoProgram; a spawn that fails to start is
// SpawnFailed. Success is OpenedURL with Done set for a later Wait — distinct
// from OpenedImage so the pane can say "opened in browser". The v1.x
// in-terminal renderer answers this method with its own honest outcome (a
// renderer that draws in the terminal cannot open a browser, so the pane prints
// the URL — the same behaviour a missing opener has).
func (e External) OpenURL(ctx context.Context, url string) (Result, error) {
	if e.Opener == "" {
		return Result{Outcome: NoOpener}, nil
	}
	if e.LookPath != nil {
		if _, err := e.LookPath(e.Opener); err != nil {
			return Result{Outcome: NoProgram}, nil
		}
	}
	run := e.Run
	if run == nil {
		return Result{Outcome: SpawnFailed}, nil
	}
	done, err := run(ctx, e.Opener, []string{url})
	if err != nil {
		return Result{Outcome: SpawnFailed}, nil
	}
	return Result{Outcome: OpenedURL, Done: done}, nil
}

// dispatch picks the program and argv by kind. The URL is exactly one argv
// element — no shell, no string-joined command line, no strings.Fields of
// anything a post carried.
func (e External) dispatch(ref domain.MediaRef) (name string, args []string, out Outcome, ok bool) {
	url := target(ref)
	switch ref.Kind {
	case domain.Image:
		if e.Opener == "" {
			return "", nil, NoOpener, false
		}
		return e.Opener, []string{url}, OpenedImage, true
	case domain.Video, domain.GIF:
		out = OpenedVideo
		if ref.Kind == domain.GIF {
			out = OpenedGIF
		}
		if ref.Playable == "" {
			out = OpenedPoster
		}
		return videoProgram, []string{"--no-terminal", url}, out, true
	default:
		return "", nil, SpawnFailed, false
	}
}
