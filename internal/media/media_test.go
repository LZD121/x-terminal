package media

import (
	"context"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
)

func TestOutcomeTableIsTotalAndDistinct(t *testing.T) {
	outcomes := []Outcome{
		OutcomeNone,
		OpenedImage,
		OpenedVideo,
		OpenedGIF,
		OpenedPoster,
		OpenedURL,
		NoOpener,
		NoProgram,
		SpawnFailed,
	}
	if len(outcomes) != int(SpawnFailed)+1 {
		t.Fatalf("table holds %d outcomes and last is %d", len(outcomes), SpawnFailed)
	}
	seen := map[Outcome]bool{}
	for i, o := range outcomes {
		if o != Outcome(i) {
			t.Errorf("row %d holds %d: table must list every value in order", i, o)
		}
		if seen[o] {
			t.Errorf("outcome %d appears twice", o)
		}
		seen[o] = true
		if o == OutcomeNone && o != 0 {
			t.Errorf("OutcomeNone is %d, want zero", o)
		}
		if o != OutcomeNone && o == 0 {
			t.Errorf("row %d is the zero value and only OutcomeNone may be", i)
		}
	}
	if OutcomeNone != 0 {
		t.Fatal("zero value must not be a success")
	}
}

func TestOutcomeWords(t *testing.T) {
	for _, tc := range []struct {
		o    Outcome
		want string
	}{
		{OutcomeNone, "none"},
		{OpenedImage, "opened image"},
		{OpenedVideo, "opened video"},
		{OpenedGIF, "opened gif"},
		{OpenedPoster, "opened poster"},
		{OpenedURL, "opened url"},
		{NoOpener, "no opener"},
		{NoProgram, "no program"},
		{SpawnFailed, "spawn failed"},
		{Outcome(99), "outcome"},
	} {
		if got := tc.o.String(); got != tc.want {
			t.Errorf("%v.String() = %q, want %q", tc.o, got, tc.want)
		}
	}
}

func TestExternalResultsCarryNoRows(t *testing.T) {
	// v1 prints no rows; the rows half is for the Kitty renderer.
	e := External{
		Opener:   "xdg-open",
		LookPath: func(string) (string, error) { return "/bin/true", nil },
		Run:      func(context.Context, string, []string) (<-chan error, error) { return nil, nil },
	}
	refs := []domain.MediaRef{
		{Kind: domain.Image, URL: "https://example.invalid/a.jpg"},
		{Kind: domain.Video, URL: "https://example.invalid/poster.jpg", Playable: "https://example.invalid/v.mp4"},
		{Kind: domain.GIF, URL: "https://example.invalid/g.jpg", Playable: "https://example.invalid/g.mp4"},
		{Kind: domain.Video, URL: "https://example.invalid/poster.jpg"},
	}
	for _, ref := range refs {
		res, err := e.Open(context.Background(), ref)
		if err != nil {
			t.Fatalf("Open(%v): %v", ref.Kind, err)
		}
		if res.Rows != nil {
			t.Errorf("Open(%v) Rows = %v, want nil", ref.Kind, res.Rows)
		}
	}
}
