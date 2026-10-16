package fake

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/xsource"
)

// The three TweetDetail bodies this package's thread fixtures are written from, and the three
// documents this test owns. Focal ids come from provenance.json — the one place that says which
// post a body is about — rather than from a literal that would keep passing after the row moved.
const (
	threadRecordedFixture  = "../xsource/testdata/fixture-tweetdetail.json"
	threadShapesFixture    = "../xsource/testdata/fixture-tweetdetail-shapes.json"
	threadTombstoneFixture = "../xsource/testdata/fixture-tweetdetail-tombstone.json"
	threadProvenancePath   = "../xsource/testdata/provenance.json"

	threadRecordedPath  = "testdata/thread.json"
	threadShapesPath    = "testdata/thread-shapes.json"
	threadTombstonePath = "testdata/thread-tombstone.json"
)

// TestThreadParity is the parity proof for the conversation fixtures: the pages ThreadRecorded,
// ThreadShapes and ThreadTombstone serve are exactly what xsource.ParseThread reads out of the
// three TweetDetail bodies, with the focal ids provenance names.
//
// With -update it rewrites the three documents; without it, it compares the checked-in fixtures
// against a fresh parse so a decoder change that moves a page fails here rather than silently
// moving every golden frame.
func TestThreadParity(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		out    string
		focal  string
		newSrc func() *Thread
	}{
		{"recorded", threadRecordedFixture, threadRecordedPath, focalIDFromProvenance(t, "fixture-tweetdetail.json"), ThreadRecorded},
		{"shapes", threadShapesFixture, threadShapesPath, focalIDFromProvenance(t, "fixture-tweetdetail-shapes.json"), ThreadShapes},
		{"tombstone", threadTombstoneFixture, threadTombstonePath, focalIDFromProvenance(t, "fixture-tweetdetail-tombstone.json"), ThreadTombstone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := canonicalizeThreadPages([]domain.ThreadPage{parseThreadFixture(t, tc.body, domain.PostID(tc.focal))})

			if *update {
				encoded, err := json.MarshalIndent(want, "", "  ")
				if err != nil {
					t.Fatalf("encoding the thread pages: %v", err)
				}
				if err := os.WriteFile(tc.out, append(encoded, '\n'), 0o644); err != nil {
					t.Fatalf("writing %s: %v", tc.out, err)
				}
				t.Logf("wrote %s: 1 page, focal %s, next %q", tc.out, want[0].Focal.ID, want[0].Next)
				return
			}

			gotDoc := readThreadDoc(t, tc.out)
			if !reflect.DeepEqual(gotDoc, want) {
				t.Fatalf("%s does not match ParseThread over %s: rewrite with -update", tc.out, filepath.Base(tc.body))
			}

			source := tc.newSrc()
			if source.unreadable != nil {
				t.Fatalf("the fake's %s page set is unreadable: %v", tc.name, source.unreadable)
			}
			got, err := source.Thread(context.Background(), domain.PostID(tc.focal), "")
			if err != nil {
				t.Fatalf("serving page one: %v", err)
			}
			got.FetchedAt = time.Time{}
			served := canonicalizeThreadPages([]domain.ThreadPage{got})
			if !reflect.DeepEqual(served, want) {
				t.Fatalf("the fake's page one for %s does not match the decoder's page", tc.name)
			}
		})
	}
}

// TestThreadSourceKnobs covers the knobs a screen test drives with: the call counter, a one-shot
// failure, a stall that echoes the asked cursor, and an unknown cursor ending the walk with an
// empty page rather than an error.
func TestThreadSourceKnobs(t *testing.T) {
	src := ThreadTombstone()
	if src.Calls() != 0 {
		t.Fatalf("a fresh source reports %d calls", src.Calls())
	}

	page, err := src.Thread(context.Background(), "9000000000000000300", "")
	if err != nil {
		t.Fatalf("page one: %v", err)
	}
	if page.Focal.ID == "" || page.Next != "" {
		t.Fatalf("tombstone page one: focal %q next %q, want a focal tombstone and no cursor", page.Focal.ID, page.Next)
	}
	if src.Calls() != 1 {
		t.Fatalf("after one call the counter is %d", src.Calls())
	}

	unknown, err := src.Thread(context.Background(), "9000000000000000300", "no-such-cursor")
	if err != nil {
		t.Fatalf("an unknown cursor must end the walk, not fail: %v", err)
	}
	if unknown.Next != "" || len(unknown.Groups) != 0 || unknown.Focal.ID != "" {
		t.Fatalf("unknown cursor answered %+v, want an empty page", unknown)
	}

	src.FailNext(errors.New("injected"))
	if _, err := src.Thread(context.Background(), "x", ""); err == nil || err.Error() != "injected" {
		t.Fatalf("FailNext did not fire: %v", err)
	}
	if _, err := src.Thread(context.Background(), "9000000000000000300", ""); err != nil {
		t.Fatalf("FailNext must be one-shot: second call failed: %v", err)
	}

	stall := ThreadRecorded()
	first, err := stall.Thread(context.Background(), "2106019976381689871", "")
	if err != nil {
		t.Fatalf("recorded page one: %v", err)
	}
	if first.Next == "" {
		t.Fatal("recorded page one carries no cursor: it is not distinguishable from the tombstone page")
	}
	stall.StallWith(first.Next)
	stalled, err := stall.Thread(context.Background(), "2106019976381689871", "")
	if err != nil {
		t.Fatalf("stalled page one: %v", err)
	}
	if stalled.Next != first.Next {
		t.Fatalf("StallWith set Next to %q, want the asked stall cursor %q", stalled.Next, first.Next)
	}
}

// TestThreadShapesCarriesRequiredCases is the arm under the shapes fixture: a tombstone, a
// limited post, a withheld post and a depth-2 quote must all be present, or a golden frame that
// needs one would fail as a rendering bug rather than as a missing post.
func TestThreadShapesCarriesRequiredCases(t *testing.T) {
	page, err := ThreadShapes().Thread(context.Background(), "9000000000000000100", "")
	if err != nil {
		t.Fatalf("shapes page one: %v", err)
	}

	var (
		tombstone bool
		limited   bool
		withheld  bool
		depth2    bool
	)
	walk := func(p domain.Post) {
		switch p.Visibility {
		case domain.Unavailable:
			tombstone = true
		case domain.Limited:
			limited = true
		case domain.Withheld:
			withheld = true
		}
		if p.Quote != nil && p.Quote.Quote != nil {
			depth2 = true
		}
	}
	walk(page.Focal)
	for _, a := range page.Ancestors {
		walk(a)
	}
	for _, g := range page.Groups {
		for _, p := range g.Posts {
			walk(p)
		}
	}
	switch {
	case !tombstone:
		t.Error("shapes fixture carries no tombstone")
	case !limited:
		t.Error("shapes fixture carries no limited post")
	case !withheld:
		t.Error("shapes fixture carries no withheld post")
	case !depth2:
		t.Error("shapes fixture carries no depth-2 quote")
	}
}

func parseThreadFixture(t *testing.T, path string, focal domain.PostID) domain.ThreadPage {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	page, err := xsource.ParseThread(raw, focal)
	if err != nil {
		t.Fatalf("ParseThread(%s): %v", filepath.Base(path), err)
	}
	if page.Focal.ID == "" && focal != "" {
		// A tombstone focal still carries its id; an empty focal on a first page is a broken body.
		if filepath.Base(path) != "fixture-tweetdetail-tombstone.json" {
			t.Fatalf("ParseThread(%s) carried no focal post for %s", filepath.Base(path), focal)
		}
	}
	return page
}

func readThreadDoc(t *testing.T, path string) []domain.ThreadPage {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var pages []domain.ThreadPage
	if err := json.Unmarshal(raw, &pages); err != nil {
		t.Fatalf("%s is not a document of thread pages: %v", path, err)
	}
	return pages
}

// canonicalizeThreadPages round-trips pages through JSON so a DeepEqual against a fixture written
// the same way is not tripped by time.Location differences (UTC vs FixedZone +0000) that encoding/json
// introduces on unmarshal. The bytes on disk are the comparison's ground truth.
func canonicalizeThreadPages(pages []domain.ThreadPage) []domain.ThreadPage {
	raw, err := json.Marshal(pages)
	if err != nil {
		panic("canonicalizeThreadPages: " + err.Error())
	}
	var out []domain.ThreadPage
	if err := json.Unmarshal(raw, &out); err != nil {
		panic("canonicalizeThreadPages: " + err.Error())
	}
	return out
}

func focalIDFromProvenance(t *testing.T, file string) string {
	t.Helper()
	raw, err := os.ReadFile(threadProvenancePath)
	if err != nil {
		t.Fatalf("reading provenance: %v", err)
	}
	var doc struct {
		Files []struct {
			File    string `json:"file"`
			FocalID string `json:"focalId"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("provenance is not readable JSON: %v", err)
	}
	for _, row := range doc.Files {
		if row.File == file {
			if row.FocalID == "" {
				t.Fatalf("provenance records no focalId for %s", file)
			}
			return row.FocalID
		}
	}
	t.Fatalf("provenance has no row for %s", file)
	return ""
}
