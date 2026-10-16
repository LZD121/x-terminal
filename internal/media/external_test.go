package media

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

type call struct {
	name string
	args []string
}

type recorder struct {
	mu    sync.Mutex
	calls []call
	block chan struct{} // when set, Run waits until closed
	err   error
}

func (r *recorder) Run(ctx context.Context, name string, args []string) (<-chan error, error) {
	r.mu.Lock()
	r.calls = append(r.calls, call{name: name, args: append([]string(nil), args...)})
	block := r.block
	err := r.err
	r.mu.Unlock()
	if block != nil {
		<-block
	}
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *recorder) seen() []call {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]call, len(r.calls))
	copy(out, r.calls)
	return out
}

func TestOpenDispatchTable(t *testing.T) {
	rec := &recorder{}
	e := External{
		Opener:   "xdg-open",
		LookPath: func(string) (string, error) { return "/bin/true", nil },
		Run:      rec.Run,
	}
	ctx := context.Background()

	t.Run("image", func(t *testing.T) {
		rec.mu.Lock()
		rec.calls = nil
		rec.mu.Unlock()
		res, err := e.Open(ctx, domain.MediaRef{Kind: domain.Image, URL: "https://pbs.twimg.com/a.jpg"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Outcome != OpenedImage {
			t.Fatalf("outcome = %v, want OpenedImage", res.Outcome)
		}
		got := rec.seen()
		if len(got) != 1 || got[0].name != "xdg-open" || !reflect.DeepEqual(got[0].args, []string{"https://pbs.twimg.com/a.jpg"}) {
			t.Fatalf("calls = %#v", got)
		}
	})

	t.Run("video playable", func(t *testing.T) {
		rec.mu.Lock()
		rec.calls = nil
		rec.mu.Unlock()
		res, err := e.Open(ctx, domain.MediaRef{
			Kind:     domain.Video,
			URL:      "https://pbs.twimg.com/poster.jpg",
			Playable: "https://video.twimg.com/v.mp4",
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.Outcome != OpenedVideo {
			t.Fatalf("outcome = %v, want OpenedVideo", res.Outcome)
		}
		wantArgs := []string{"--no-terminal", "https://video.twimg.com/v.mp4"}
		got := rec.seen()
		if len(got) != 1 || got[0].name != "mpv" || !reflect.DeepEqual(got[0].args, wantArgs) {
			t.Fatalf("calls = %#v, want mpv %v", got, wantArgs)
		}
	})

	t.Run("gif playable", func(t *testing.T) {
		rec.mu.Lock()
		rec.calls = nil
		rec.mu.Unlock()
		res, err := e.Open(ctx, domain.MediaRef{
			Kind:     domain.GIF,
			URL:      "https://pbs.twimg.com/g.jpg",
			Playable: "https://video.twimg.com/g.mp4",
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.Outcome != OpenedGIF {
			t.Fatalf("outcome = %v, want OpenedGIF", res.Outcome)
		}
		wantArgs := []string{"--no-terminal", "https://video.twimg.com/g.mp4"}
		got := rec.seen()
		if len(got) != 1 || got[0].name != "mpv" || !reflect.DeepEqual(got[0].args, wantArgs) {
			t.Fatalf("calls = %#v", got)
		}
	})

	t.Run("video poster", func(t *testing.T) {
		rec.mu.Lock()
		rec.calls = nil
		rec.mu.Unlock()
		res, err := e.Open(ctx, domain.MediaRef{
			Kind: domain.Video,
			URL:  "https://pbs.twimg.com/poster.jpg",
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.Outcome != OpenedPoster {
			t.Fatalf("outcome = %v, want OpenedPoster", res.Outcome)
		}
		wantArgs := []string{"--no-terminal", "https://pbs.twimg.com/poster.jpg"}
		got := rec.seen()
		if len(got) != 1 || got[0].name != "mpv" || !reflect.DeepEqual(got[0].args, wantArgs) {
			t.Fatalf("calls = %#v", got)
		}
	})
}

func TestOpenMissingProgram(t *testing.T) {
	started := false
	e := External{
		Opener: "xdg-open",
		LookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		Run: func(context.Context, string, []string) (<-chan error, error) {
			started = true
			return nil, nil
		},
	}
	res, err := e.Open(context.Background(), domain.MediaRef{Kind: domain.Image, URL: "https://x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != NoProgram {
		t.Fatalf("outcome = %v, want NoProgram", res.Outcome)
	}
	if started {
		t.Fatal("Run was called for a missing program")
	}
}

func TestOpenNonZeroExit(t *testing.T) {
	e := External{
		Opener:   "xdg-open",
		LookPath: func(string) (string, error) { return "/bin/true", nil },
		Run:      func(context.Context, string, []string) (<-chan error, error) { return nil, errors.New("exit 1") },
	}
	res, err := e.Open(context.Background(), domain.MediaRef{Kind: domain.Image, URL: "https://x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != SpawnFailed {
		t.Fatalf("outcome = %v, want SpawnFailed", res.Outcome)
	}
}

func TestOpenNoShellArgv(t *testing.T) {
	// The URL carries every shell metacharacter that would matter if a command line were joined.
	url := `https://evil.example/a b;$(touch /tmp/x)` + "`id`" + `"quoted"`
	rec := &recorder{}
	e := External{
		Opener:   "xdg-open",
		LookPath: func(string) (string, error) { return "/bin/true", nil },
		Run:      rec.Run,
	}
	_, err := e.Open(context.Background(), domain.MediaRef{Kind: domain.Image, URL: url})
	if err != nil {
		t.Fatal(err)
	}
	got := rec.seen()
	if len(got) != 1 {
		t.Fatalf("calls = %d", len(got))
	}
	if len(got[0].args) != 1 || got[0].args[0] != url {
		t.Fatalf("argv = %#v, want exactly one element equal to the URL", got[0].args)
	}
}

func TestOpenDoesNotWaitOnChild(t *testing.T) {
	// Run is Start-shaped (StartAndReap): it returns after spawn while a reaper
	// waits on the child. A two-second sleep must not delay Open beyond the spawn.
	ctx := context.Background()
	e := External{
		Opener:   "xdg-open",
		LookPath: func(string) (string, error) { return "/bin/true", nil },
		Run:      StartAndReap,
	}
	// Point LookPath at sleep so Open dispatches a real long-lived child via StartAndReap.
	e.LookPath = func(name string) (string, error) {
		if name == "xdg-open" {
			return "/usr/bin/sleep", nil
		}
		return DefaultLookPath(name)
	}
	e.Opener = "sleep"
	e.Run = func(ctx context.Context, name string, args []string) (<-chan error, error) {
		// Ignore the URL arg; sleep for two seconds so a Wait-shaped call would block.
		return StartAndReap(ctx, "sleep", []string{"2"})
	}
	started := time.Now()
	res, err := e.Open(ctx, domain.MediaRef{Kind: domain.Image, URL: "https://example.invalid/x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != OpenedImage {
		t.Fatalf("outcome = %v, want OpenedImage", res.Outcome)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("Open blocked for %s; StartAndReap must return after spawn", elapsed)
	}
}

func TestOpenStartAndReapNonZeroExitSurfacesOnDone(t *testing.T) {
	// Production Run is StartAndReap: Open must not wait, but Wait's non-zero
	// exit must still become SpawnFailed via Result.Done (MEDIA-03).
	ctx := context.Background()
	e := External{
		Opener:   "false",
		LookPath: DefaultLookPath,
		Run:      StartAndReap,
	}
	started := time.Now()
	res, err := e.Open(ctx, domain.MediaRef{Kind: domain.Image, URL: "https://example.invalid/x"})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("Open blocked for %s", elapsed)
	}
	if res.Outcome != OpenedImage {
		t.Fatalf("immediate outcome = %v, want OpenedImage (start succeeded)", res.Outcome)
	}
	if res.Done == nil {
		t.Fatal("Done is nil; StartAndReap must expose Wait")
	}
	select {
	case waitErr := <-res.Done:
		if waitErr == nil {
			t.Fatal("Wait error is nil; false must exit non-zero")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Done")
	}
}
func TestTargetPrefersPlayable(t *testing.T) {
	if got := target(domain.MediaRef{URL: "poster", Playable: "play"}); got != "play" {
		t.Fatalf("target = %q, want play", got)
	}
	if got := target(domain.MediaRef{URL: "poster"}); got != "poster" {
		t.Fatalf("target = %q, want poster", got)
	}
}

func TestOpenURLOneArg(t *testing.T) {
	const url = "https://x.com/aergon/status/1812041"
	rec := &recorder{}
	e := External{
		Opener:   "xdg-open",
		LookPath: func(string) (string, error) { return "/usr/bin/xdg-open", nil },
		Run:      rec.Run,
	}
	res, err := e.OpenURL(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != OpenedURL {
		t.Fatalf("outcome = %v, want OpenedURL", res.Outcome)
	}
	if res.Rows != nil {
		t.Fatalf("Rows = %v, want nil", res.Rows)
	}
	got := rec.seen()
	if len(got) != 1 || got[0].name != "xdg-open" || !reflect.DeepEqual(got[0].args, []string{url}) {
		t.Fatalf("calls = %#v", got)
	}
}

func TestOpenURLMissingOpener(t *testing.T) {
	started := false
	e := External{
		Opener:   "",
		LookPath: func(string) (string, error) { return "/bin/true", nil },
		Run: func(context.Context, string, []string) (<-chan error, error) {
			started = true
			return nil, nil
		},
	}
	res, err := e.OpenURL(context.Background(), "https://x.com/i/status/1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != NoOpener {
		t.Fatalf("outcome = %v, want NoOpener", res.Outcome)
	}
	if started {
		t.Fatal("Run was called with empty Opener")
	}
}

func TestOpenURLMissingProgram(t *testing.T) {
	started := false
	e := External{
		Opener: "xdg-open",
		LookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		Run: func(context.Context, string, []string) (<-chan error, error) {
			started = true
			return nil, nil
		},
	}
	res, err := e.OpenURL(context.Background(), "https://x.com/aergon/status/1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != NoProgram {
		t.Fatalf("outcome = %v, want NoProgram", res.Outcome)
	}
	if started {
		t.Fatal("Run was called for a missing program")
	}
}

func TestOpenURLNonZeroExit(t *testing.T) {
	e := External{
		Opener:   "xdg-open",
		LookPath: func(string) (string, error) { return "/bin/true", nil },
		Run:      func(context.Context, string, []string) (<-chan error, error) { return nil, errors.New("exit 1") },
	}
	res, err := e.OpenURL(context.Background(), "https://x.com/aergon/status/1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != SpawnFailed {
		t.Fatalf("outcome = %v, want SpawnFailed", res.Outcome)
	}
}
