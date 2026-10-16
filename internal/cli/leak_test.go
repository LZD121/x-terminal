package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/audit"
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/netcheck"
	"github.com/LZD121/x-terminal/internal/render"
	"github.com/LZD121/x-terminal/internal/store"
)

// This file is the phase's cookie sweep over the surfaces the CLI's own code
// produces. Every surface below is the real thing: the doctor verb's text and
// JSON report with a fixture session, the sync verb's report over a fake
// fetcher, a media notice through the renderer's own notice line, a store error
// from a read that actually fails, the audit's own report, and a panic recovered
// in this process. Nothing here is a hand-written stand-in for output: the
// search is audit.Sweep, the same function the audit's leak check runs, so the
// surfaces and the assertion cannot drift apart.

// TestLeakSweepFindsNoCookieInRealOutput is the leak arm: if any of these
// producers ever prints the session's credential — a debug print in sync, an
// error that carries the token — this fails and names it.
func TestLeakSweepFindsNoCookieInRealOutput(t *testing.T) {
	surfaces := realLeakSurfaces(t)

	// Anti-vacuity: each surface has something in it to search.
	for _, s := range surfaces {
		if s.Body == "" {
			t.Fatalf("surface %q is empty: the sweep would pass over it whatever it held", s.Name)
		}
	}
	findings := audit.Sweep("leak", surfaces)
	for _, f := range findings {
		t.Errorf("%s", f.Detail)
	}
	// The needle is the value the fixture session holds, so "absent" below means
	// redaction rather than a needle that was never in the haystack.
	if audit.FixtureAuthToken != fixtureToken {
		t.Fatalf("the sweep's needle is not this fixture's token: %q vs %q", audit.FixtureAuthToken, fixtureToken)
	}
	for _, s := range surfaces {
		if at := strings.Index(s.Body, fixtureToken); at >= 0 {
			t.Fatalf("surface %q carries the auth token at offset %d", s.Name, at)
		}
	}
	t.Logf("searched %d real surfaces for two credentials; both absent", len(surfaces))

	// Planted arm, permanent: the same search must fail on these very surfaces
	// when one of them carries the cookie. This is what makes the arm above
	// evidence rather than a search that cannot fail.
	planted := make([]audit.Surface, len(surfaces))
	copy(planted, surfaces)
	for i := range planted {
		if planted[i].Name == "sync-report" {
			planted[i].Body += "\nmirror " + fixtureToken + "\n"
		}
	}
	if got := audit.Sweep("leak", planted); len(got) == 0 {
		t.Fatal("a cookie planted into the sync surface was not caught: the sweep above proves nothing")
	}
}

// realLeakSurfaces runs the producers and captures what they wrote.
func realLeakSurfaces(t *testing.T) []audit.Surface {
	t.Helper()
	sessionPath := writeSession(t, t.TempDir(), "auth_token="+fixtureToken+"\nct0="+fixtureCTO+"\n")
	hermeticConfig(t, sessionPath)
	withStubDoctorCheck(t, []netcheck.Result{{
		Outcome: netcheck.OutcomeSessionSkipped,
		Detail:  "no session configured",
	}})

	var surfaces []audit.Surface
	for _, c := range []struct {
		name string
		args []string
	}{
		{"doctor-report", nil},
		{"doctor-json", []string{"--json"}},
	} {
		code, stdout, stderr := run(t, Doctor, c.args...)
		if code != ExitOK {
			t.Fatalf("doctor %v exited %d: %s", c.args, code, stderr)
		}
		surfaces = append(surfaces, audit.Surface{Name: c.name, Body: stdout + stderr})
	}

	// The sync verb, over the same fake fetcher its own tests use: a real run of
	// the real report builder.
	withOpenMirror(t, syncFetcherForLeak(), &syncFakeViewer{user: domain.User{ScreenName: "aergon", ID: "1"}})
	code, stdout, stderr := run(t, Sync, "--kind", "bookmarks")
	if code != ExitOK {
		t.Fatalf("sync exited %d: %s", code, stderr)
	}
	surfaces = append(surfaces, audit.Surface{Name: "sync-report", Body: stdout + stderr})

	// The media notices the pane prints, through the renderer's own line.
	var notices []string
	for _, cover := range []render.CoverStyle{"", render.CoverServiceLog, render.CoverCI} {
		for _, kind := range []render.NoticeKind{
			render.NoticeOpenedImage,
			render.NoticeOpenedVideo,
			render.NoticeOpenedGIF,
			render.NoticeOpenedPoster,
			render.NoticeOpenedMedia,
			render.NoticeNoOpener,
			render.NoticeNoMedia,
			render.NoticeOpenedBrowser,
		} {
			notices = append(notices, render.NoticeLine(cover, kind, 3))
		}
	}
	surfaces = append(surfaces, audit.Surface{Name: "media-notices", Body: strings.Join(notices, "\n")})

	// A store error from a read that really fails: the file is there and is not
	// a database.
	notADB := filepath.Join(t.TempDir(), "saved.db")
	if err := os.WriteFile(notADB, []byte("not a sqlite file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadFacts(notADB); err == nil {
		t.Fatalf("reading %s did not fail: the store error surface would be empty", notADB)
	} else {
		surfaces = append(surfaces, audit.Surface{Name: "store-error", Body: err.Error()})
	}

	// The audit's own report, over this tree.
	reports, err := audit.Run(moduleRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	surfaces = append(surfaces, audit.Surface{Name: "audit-report", Body: audit.FormatReports(reports)})

	// A recovered panic's output, produced here.
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				surfaces = append(surfaces, audit.Surface{Name: "recovered-panic", Body: fmt.Sprintf("panic: %v", rec)})
			}
		}()
		panic("cli leak surface")
	}()

	if len(surfaces) != 7 {
		t.Fatalf("built %d surfaces, want 7 (doctor x2, sync, media, store, audit, panic)", len(surfaces))
	}
	return surfaces
}

// syncFetcherForLeak is a one-kind fake: an empty bookmark page and an empty end
// of list, so the sync report is built without a network call.
func syncFetcherForLeak() *syncFakeFetcher {
	fetch := newSyncFake()
	fetch.pages[domain.Bookmark] = []domain.PostPage{{}}
	fetch.pages[domain.Like] = []domain.PostPage{{}}
	return fetch
}
