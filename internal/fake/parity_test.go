package fake

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"strconv"
	"testing"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/xsource"
)

// The two files this test exists to keep in step, and the test that owns the second one.
//
// The import of internal/xsource is deliberate and allowed here: what the shell's boundary scan
// watches is the non-test files, and this is the one place where the fake's recorded pages and the
// real parser are compared. A fake that drifted from the fixture — a field renamed, a post
// dropped, a page split changed — must fail a build rather than quietly make a screen test pass.
const (
	// fixturePath is the recorded response itself, reached the way a test reaches another
	// package's fixture: by path from this package's directory. It is the same body the live
	// adapter is tested against, which is the point — the fake's pages have to be that response's
	// parse, not a second copy of it.
	fixturePath = "../xsource/testdata/fixture-usertweets.json"
	// pagesPath is the document this test owns: an array of pages, each an array of domain posts.
	pagesPath = "testdata/pages.json"
)

// update rewrites testdata/pages.json from the fixture. It is a flag rather than a checked-in
// generator so that the file on disk is always what the parser makes of the response checked in
// beside it.
var update = flag.Bool("update", false, "rewrite testdata/pages.json from the recorded fixture")

var updateProfile = flag.Bool("update-profile", false, "rewrite testdata/pages-profile.json from the profile posts fixture")
var updateSearchPeople = flag.Bool("update-search-people", false, "rewrite testdata/pages-search-people.json from the people fixture")
var updateSearch = flag.Bool("update-search", false, "rewrite testdata/pages-search.json from the search fixture")
var updateSaved = flag.Bool("update-saved", false, "rewrite testdata/pages-saved.json from the bookmarks fixture")

// TestFakePagesMatchTheFixture is the parity proof: the pages Recorded serves are the posts
// xsource.ParseTimeline reads out of the recorded response, in the same order, split the same way,
// carrying the same values.
//
// With -update it writes testdata/pages.json; without it, it reads what is on disk and compares it
// against a fresh parse, field by field, so that the file cannot silently become the fixture.
func TestFakePagesMatchTheFixture(t *testing.T) {
	parsed := parseFixture(t)
	want := splitRecorded(parsed.Posts)
	if len(want) < 2 {
		t.Fatalf("the recorded response splits into %d page(s): a page set of one cannot be paged, and the fake exists to make pagination drivable", len(want))
	}

	if *update {
		encoded, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			t.Fatalf("encoding the pages: %v", err)
		}
		if err := os.WriteFile(pagesPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("writing %s: %v", pagesPath, err)
		}
		t.Logf("wrote %s: %d pages, %d posts", pagesPath, len(want), len(parsed.Posts))
		return
	}

	// The fake's own reader and its own cursor walk, not the file: what is compared below is what a
	// screen would be served.
	source := Recorded()
	if len(source.pages) == 0 {
		t.Fatalf("the fake's recorded page set is empty: %v", source.unreadable)
	}
	got, err := servedPages(source)
	if err != nil {
		t.Fatalf("walking the fake's pages: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("the fake serves %d pages and the fixture splits into %d: testdata/pages.json is not the split splitRecorded makes", len(got), len(want))
	}
	var compared int
	for i := range want {
		if len(got[i].Posts) != len(want[i]) {
			t.Fatalf("page %d carries %d posts and the fixture's page carries %d", i+1, len(got[i].Posts), len(want[i]))
		}
		for j := range want[i] {
			compared++
			diff(t, i+1, j, got[i].Posts[j], want[i][j])
		}
	}

	// The arms. The walk above only proves something if it compared every recorded post, and the
	// page set is only drivable if the pages between them carry the cursors that lead on.
	if compared != len(parsed.Posts) {
		t.Errorf("the comparison covered %d posts and the fixture holds %d", compared, len(parsed.Posts))
	}
	if last := got[len(got)-1]; last.Next != "" {
		t.Errorf("the last page carries the next cursor %q, want the empty one that ends the walk", last.Next)
	}
	for i, page := range got[:len(got)-1] {
		if page.Next == "" {
			t.Errorf("page %d carries no next cursor, so nothing past it can be fetched", i+1)
		}
	}
}

const (
	savedFixturePath = "../xsource/testdata/fixture-bookmarks.json"
	savedPagesPath   = "testdata/pages-saved.json"
)

// TestFakeSavedPagesMatchTheFixture is the saved-list parity proof: the pages Saved serves for
// bookmarks are the posts xsource.ParseTimeline reads out of the recorded Bookmarks body.
func TestFakeSavedPagesMatchTheFixture(t *testing.T) {
	raw, err := os.ReadFile(savedFixturePath)
	if err != nil {
		t.Fatalf("reading the bookmarks fixture: %v", err)
	}
	parsed, err := xsource.ParseTimeline(raw)
	if err != nil {
		t.Fatalf("ParseTimeline(bookmarks): %v", err)
	}
	if len(parsed.Posts) == 0 {
		t.Fatal("the bookmarks fixture parsed into no posts")
	}
	want := splitRecorded(parsed.Posts)

	if *updateSaved {
		doc := map[string]any{
			"bookmark": want,
			"like":     likeGoldenPages(),
		}
		encoded, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			t.Fatalf("encoding: %v", err)
		}
		if err := os.WriteFile(savedPagesPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("writing %s: %v", savedPagesPath, err)
		}
		t.Logf("wrote %s: %d bookmark pages, %d posts", savedPagesPath, len(want), len(parsed.Posts))
		return
	}

	source := Recorded()
	got, err := servedSavedPages(source, domain.Bookmark)
	if err != nil {
		t.Fatalf("walking saved bookmark pages: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("the fake serves %d bookmark pages and the fixture splits into %d", len(got), len(want))
	}
	var compared int
	for i := range want {
		if len(got[i].Posts) != len(want[i]) {
			t.Fatalf("bookmark page %d carries %d posts, want %d", i+1, len(got[i].Posts), len(want[i]))
		}
		for j := range want[i] {
			compared++
			diff(t, i+1, j, got[i].Posts[j], want[i][j])
		}
	}
	if compared != len(parsed.Posts) {
		t.Errorf("compared %d posts, fixture holds %d", compared, len(parsed.Posts))
	}
	if last := got[len(got)-1]; last.Next != "" {
		t.Errorf("last bookmark page carries next %q, want empty", last.Next)
	}
	for i, page := range got[:len(got)-1] {
		if page.Next == "" {
			t.Errorf("bookmark page %d carries no next cursor", i+1)
		}
	}
}

func servedSavedPages(source *Source, kind domain.SavedKind) ([]domain.PostPage, error) {
	var pages []domain.PostPage
	cursor := domain.Cursor("")
	for {
		page, err := source.Saved(context.Background(), kind, cursor)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
		if page.Next == "" {
			return pages, nil
		}
		cursor = page.Next
		if len(pages) > 64 {
			return nil, errors.New("saved walk does not end")
		}
	}
}

// likeGoldenPages is the likes page set the goldens need: a 15-cell handle, a tombstone, a CJK
// run and a ZWJ family, an attachment, and a page that ends with a stored next cursor.
func likeGoldenPages() [][]domain.Post {
	return [][]domain.Post{
		{
			adversarialPost(0, "a_15_cell_handl", "a fifteen-cell handle, the widest the column holds"),
			adversarialPost(1, "lzd", "日本語のタイムラインを端末で読む。"),
			adversarialPost(2, "lzd", "\U0001F468\u200D\U0001F469\u200D\U0001F467"),
		},
		{
			func() domain.Post {
				p := adversarialPost(3, "lzd", "one attachment")
				p.Media = []domain.MediaRef{{Kind: domain.Image, URL: "https://example.invalid/one.jpg", AltText: "a screenshot"}}
				return p
			}(),
			domain.Post{ID: "saved-tombstone", Visibility: domain.Unavailable},
			adversarialPost(4, "lzd", "the last like on the page that ends with a next cursor"),
		},
		{
			adversarialPost(5, "lzd", "the last like of the set"),
		},
	}
}

// parseFixture reads and parses the recorded response the fake's pages were written from.
func parseFixture(t *testing.T) xsource.Page {
	t.Helper()
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading the recorded response: %v", err)
	}
	parsed, err := xsource.ParseTimeline(raw)
	if err != nil {
		t.Fatalf("parsing the recorded response: %v", err)
	}
	if len(parsed.Posts) == 0 {
		t.Fatal("the recorded response parsed into no posts: every comparison below would pass for the wrong reason")
	}
	return parsed
}

// diff compares one field at a time and stops at the first disagreement, so that a failure names
// the page, the post and the value rather than dumping two documents.
func diff(t *testing.T, page, index int, got, want domain.Post) {
	t.Helper()
	at := "page " + strconv.Itoa(page) + ", post " + strconv.Itoa(index+1)

	switch {
	case got.ID != want.ID:
		t.Fatalf("%s: id is %q, want %q", at, got.ID, want.ID)
	case got.Author.ScreenName != want.Author.ScreenName:
		t.Fatalf("%s: byline is %q, want %q", at, got.Author.ScreenName, want.Author.ScreenName)
	case got.Author.ID != want.Author.ID:
		t.Fatalf("%s: author id is %q, want %q", at, got.Author.ID, want.Author.ID)
	case got.Text != want.Text:
		t.Fatalf("%s: body is %q, want %q", at, got.Text, want.Text)
	case !got.CreatedAt.Equal(want.CreatedAt):
		t.Fatalf("%s: instant is %s, want %s", at, got.CreatedAt, want.CreatedAt)
	case got.Lang != want.Lang:
		t.Fatalf("%s: language is %q, want %q", at, got.Lang, want.Lang)
	case got.ConversationID != want.ConversationID:
		t.Fatalf("%s: conversation is %q, want %q", at, got.ConversationID, want.ConversationID)
	case got.ReplyTo != want.ReplyTo:
		t.Fatalf("%s: reply-to is %q, want %q", at, got.ReplyTo, want.ReplyTo)
	case got.Visibility != want.Visibility:
		t.Fatalf("%s: visibility is %s, want %s", at, got.Visibility, want.Visibility)
	case got.Counts != want.Counts:
		t.Fatalf("%s: counts are %+v, want %+v", at, got.Counts, want.Counts)
	case len(got.Media) != len(want.Media):
		t.Fatalf("%s: carries %d media refs, want %d", at, len(got.Media), len(want.Media))
	case (got.Quote == nil) != (want.Quote == nil):
		t.Fatalf("%s: quote presence is %v, want %v", at, got.Quote != nil, want.Quote != nil)
	case (got.RepostedBy == nil) != (want.RepostedBy == nil):
		t.Fatalf("%s: repost presence is %v, want %v", at, got.RepostedBy != nil, want.RepostedBy != nil)
	}
	for k := range want.Media {
		if got.Media[k] != want.Media[k] {
			t.Fatalf("%s: media ref %d is %+v, want %+v", at, k, got.Media[k], want.Media[k])
		}
	}
	if want.Quote != nil && got.Quote.ID != want.Quote.ID {
		t.Fatalf("%s: quotes %q, want %q", at, got.Quote.ID, want.Quote.ID)
	}
	if want.RepostedBy != nil && got.RepostedBy.ScreenName != want.RepostedBy.ScreenName {
		t.Fatalf("%s: reposted by %q, want %q", at, got.RepostedBy.ScreenName, want.RepostedBy.ScreenName)
	}
}

// servedPages walks the fake the way the list does: page one with an empty cursor, then whatever
// the page before it handed back. It is the only way to see the fake's pages, and it is what makes
// this test a proof about the source rather than about a struct in a file.
func servedPages(source *Source) ([]domain.PostPage, error) {
	var pages []domain.PostPage
	cursor := domain.Cursor("")
	for {
		page, err := source.Home(context.Background(), domain.Following, cursor)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
		if page.Next == "" {
			return pages, nil
		}
		cursor = page.Next
		if len(pages) > 64 {
			return nil, errors.New("the walk does not end: the fake hands back a cursor it has already served")
		}
	}
}

// TestAdversarialCarriesEveryRequiredCase is the arm under internal/render's corpus audit: the
// golden frames can only show a case the set actually contains, and a case dropped here would
// otherwise be noticed as a rendering bug rather than as a missing post. It is the documented
// content list of adversarialPages, made checkable.
func TestAdversarialCarriesEveryRequiredCase(t *testing.T) {
	pages, err := servedPages(Adversarial())
	if err != nil {
		t.Fatalf("walking the adversarial pages: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("the adversarial set is %d page(s): with one page there is no page that ends with a stored next cursor", len(pages))
	}

	var (
		posts        []domain.Post
		handles      = map[string]int{}
		mediaCounts  = map[int]int{}
		quoteDepths  = map[int]int{}
		visibilities = map[domain.Visibility]int{}
		reposted     int
		continuation int
	)
	for _, page := range pages {
		posts = append(posts, page.Posts...)
	}
	for _, post := range posts {
		handles[post.Author.ScreenName]++
		mediaCounts[len(post.Media)]++
		depth := 0
		for q := post.Quote; q != nil; q = q.Quote {
			depth++
		}
		quoteDepths[depth]++
		visibilities[post.Visibility]++
		if post.RepostedBy != nil {
			reposted++
		}
		if post.ReplyTo != "" {
			continuation++
		}
	}

	switch {
	case len(posts) < 15:
		t.Errorf("the set holds %d posts, want the full adversarial set", len(posts))
	case handles["a_15_cell_handl"] == 0:
		t.Error("the set holds no post with a 15-cell handle")
	case handles["lzd"] == 0:
		t.Error("the set holds no post with a 3-cell handle")
	case quoteDepths[1] == 0 || quoteDepths[2] == 0:
		t.Errorf("the set's quote depths are %v, want a depth-1 and a depth-2 quote", quoteDepths)
	case mediaCounts[1] == 0 || mediaCounts[4] == 0:
		t.Errorf("the set's attachment counts are %v, want a singular and a plural example", mediaCounts)
	case reposted == 0:
		t.Error("the set holds no repost")
	case visibilities[domain.Limited] == 0:
		t.Error("the set holds no limited post")
	case visibilities[domain.Unavailable] == 0:
		t.Error("the set holds no tombstone")
	case continuation == 0:
		t.Error("the set holds no continuation pair")
	}
}

// TestTheFakeCountsAndRefusesWhatItCannotServe covers the knobs a screen test drives with: the call
// counter, the feed kind of the last request, a one-shot failure, and a cursor that names no page.
//
// The last one matters most: a fake that answered an unknown cursor with an empty page would let a
// state machine that invented a cursor pass its own test.
func TestTheFakeCountsAndRefusesWhatItCannotServe(t *testing.T) {
	source := Recorded()
	if got := source.Calls(); got != 0 {
		t.Errorf("a fresh source reports %d calls, want 0", got)
	}
	if kind, ok := source.LastKind(); ok {
		t.Errorf("a source that has served nothing reports the feed kind %v, want none", kind)
	}

	if _, err := source.Home(context.Background(), domain.Following, ""); err != nil {
		t.Fatalf("serving page one: %v", err)
	}
	if _, err := source.Home(context.Background(), domain.ForYou, "recorded-1"); err != nil {
		t.Fatalf("serving the page after recorded-1: %v", err)
	}
	if _, err := source.User(context.Background(), "lzd"); err != nil {
		t.Fatalf("serving a user: %v", err)
	}
	if got := source.Calls(); got != 3 {
		t.Errorf("after two pages and a user the source reports %d calls, want 3", got)
	}
	if kind, ok := source.LastKind(); !ok || kind != domain.ForYou {
		t.Errorf("the last feed asked for is %v (%v), want the for-you timeline", kind, ok)
	}

	// FailNext fails one call and only that one: the page after it is served normally, which is what
	// makes a partial-failure test possible at all.
	sentinel := errors.New("the transport's own classified failure")
	source.FailNext(sentinel)
	if _, err := source.Home(context.Background(), domain.Following, ""); !errors.Is(err, sentinel) {
		t.Errorf("the call after FailNext reported %v, want the error it was given", err)
	}
	if _, err := source.Home(context.Background(), domain.Following, ""); err != nil {
		t.Errorf("the call after the failing one reported %v, want a page: FailNext is one shot", err)
	}

	if _, err := source.Home(context.Background(), domain.Following, "no-such-cursor"); err == nil {
		t.Error("a cursor that names no page was answered with a page")
	}
}

func TestFakeProfilePagesMatchTheFixture(t *testing.T) {
	raw, err := os.ReadFile("../xsource/testdata/fixture-profile-posts.json")
	if err != nil {
		t.Fatalf("reading profile posts fixture: %v", err)
	}
	parsed, err := xsource.ParseTimeline(raw)
	if err != nil {
		t.Fatalf("parsing profile posts fixture: %v", err)
	}
	want := splitRecorded(parsed.Posts)
	if len(want) < 2 {
		t.Fatalf("profile posts split into %d page(s)", len(want))
	}
	if *updateProfile {
		body, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(profilePagesPath, append(body, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s: %d pages, %d posts", profilePagesPath, len(want), len(parsed.Posts))
		return
	}
	source := RecordedProfile()
	if source.unreadable != nil {
		t.Fatalf("RecordedProfile: %v", source.unreadable)
	}
	got, err := servedProfilePages(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("fake serves %d pages, fixture splits into %d", len(got), len(want))
	}
	for i := range want {
		if len(got[i].Posts) != len(want[i]) {
			t.Fatalf("page %d posts %d, want %d", i+1, len(got[i].Posts), len(want[i]))
		}
		for j := range want[i] {
			diff(t, i+1, j, got[i].Posts[j], want[i][j])
		}
	}
	header, err := source.Profile(context.Background(), "X")
	if err != nil {
		t.Fatal(err)
	}
	if header.Bio == "" {
		t.Fatal("recorded profile header bio empty")
	}
}

func servedProfilePages(source *Source) ([]domain.PostPage, error) {
	var pages []domain.PostPage
	cursor := domain.Cursor("")
	for {
		page, err := source.UserPosts(context.Background(), "783214", cursor)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
		if page.Next == "" {
			return pages, nil
		}
		cursor = page.Next
		if len(pages) > 64 {
			return nil, errors.New("profile walk does not end")
		}
	}
}

const (
	peopleFixturePath     = "../xsource/testdata/fixture-search-people.json"
	peoplePagesParityPath = "testdata/pages-search-people.json"
)

// TestFakePeoplePagesMatchTheFixture is the people parity proof: the pages RecordedPeople
// serves are xsource.ParseSearchPeople's output over the people fixture.
func TestFakePeoplePagesMatchTheFixture(t *testing.T) {
	raw, err := os.ReadFile(peopleFixturePath)
	if err != nil {
		t.Fatalf("reading the people fixture: %v", err)
	}
	parsed, err := xsource.ParseSearchPeople(raw)
	if err != nil {
		t.Fatalf("ParseSearchPeople: %v", err)
	}
	if len(parsed.Rows) == 0 {
		t.Fatal("the people fixture parsed into no accounts")
	}
	want := [][]domain.User{parsed.Rows}

	if *updateSearchPeople {
		encoded, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			t.Fatalf("encoding the people pages: %v", err)
		}
		if err := os.WriteFile(peoplePagesParityPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("writing %s: %v", peoplePagesParityPath, err)
		}
		t.Logf("wrote %s: %d pages, %d accounts", peoplePagesParityPath, len(want), len(parsed.Rows))
		return
	}

	source := RecordedPeople()
	if len(source.peoplePages) == 0 {
		t.Fatalf("the fake's recorded people page set is empty: %v", source.unreadable)
	}
	got, err := servedPeoplePages(source)
	if err != nil {
		t.Fatalf("walking the fake's people pages: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("the fake serves %d people pages and the fixture splits into %d", len(got), len(want))
	}
	for i := range want {
		if len(got[i].Rows) != len(want[i]) {
			t.Fatalf("people page %d carries %d accounts and the fixture's page carries %d", i+1, len(got[i].Rows), len(want[i]))
		}
		for j := range want[i] {
			g, w := got[i].Rows[j], want[i][j]
			if g.ID != w.ID || g.ScreenName != w.ScreenName || g.Name != w.Name {
				t.Fatalf("people page %d account %d is %+v, want %+v", i+1, j+1, g, w)
			}
		}
	}
	if last := got[len(got)-1]; last.Next != "" {
		t.Errorf("the last people page carries next cursor %q, want empty", last.Next)
	}
}

func servedPeoplePages(source *Source) ([]domain.UserPage, error) {
	var pages []domain.UserPage
	cursor := domain.Cursor("")
	for {
		page, err := source.SearchPeople(context.Background(), "people", cursor)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
		if page.Next == "" {
			return pages, nil
		}
		cursor = page.Next
		if len(pages) > 64 {
			return nil, errors.New("people walk does not end")
		}
	}
}

const (
	searchFixturePath     = "../xsource/testdata/fixture-search.json"
	searchPagesParityPath = "testdata/pages-search.json"
)

// TestFakeSearchPagesMatchTheFixture is the posts-search parity proof: the pages RecordedSearch
// serves are xsource.ParseSearch's output over the search fixture.
func TestFakeSearchPagesMatchTheFixture(t *testing.T) {
	raw, err := os.ReadFile(searchFixturePath)
	if err != nil {
		t.Fatalf("reading the search fixture: %v", err)
	}
	parsed, err := xsource.ParseSearch(raw)
	if err != nil {
		t.Fatalf("ParseSearch: %v", err)
	}
	if len(parsed.Posts) == 0 {
		t.Fatal("the search fixture parsed into no posts")
	}
	want := [][]domain.Post{parsed.Posts}

	if *updateSearch {
		encoded, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			t.Fatalf("encoding the search pages: %v", err)
		}
		if err := os.WriteFile(searchPagesParityPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("writing %s: %v", searchPagesParityPath, err)
		}
		t.Logf("wrote %s: %d pages, %d posts", searchPagesParityPath, len(want), len(parsed.Posts))
		return
	}

	source := RecordedSearch()
	if len(source.pages) == 0 {
		t.Fatalf("the fake's recorded search page set is empty: %v", source.unreadable)
	}
	got, err := servedSearchPages(source)
	if err != nil {
		t.Fatalf("walking the fake's search pages: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("the fake serves %d search pages and the fixture splits into %d", len(got), len(want))
	}
	for i := range want {
		if len(got[i].Posts) != len(want[i]) {
			t.Fatalf("search page %d carries %d posts and the fixture's page carries %d", i+1, len(got[i].Posts), len(want[i]))
		}
		for j := range want[i] {
			g, w := got[i].Posts[j], want[i][j]
			if g.ID != w.ID || g.Text != w.Text || g.Author.ScreenName != w.Author.ScreenName {
				t.Fatalf("search page %d post %d is %+v, want %+v", i+1, j+1, g, w)
			}
		}
	}
	if last := got[len(got)-1]; last.Next != "" {
		t.Errorf("the last search page carries next cursor %q, want empty", last.Next)
	}
}

func servedSearchPages(source *Source) ([]domain.PostPage, error) {
	var pages []domain.PostPage
	cursor := domain.Cursor("")
	for {
		page, err := source.Search(context.Background(), "timeline", domain.Latest, cursor)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
		if page.Next == "" {
			return pages, nil
		}
		cursor = page.Next
		if len(pages) > 64 {
			return nil, errors.New("search walk does not end")
		}
	}
}

// TestAdversarialPeopleCarriesRequiredCases pins the documented adversarial people content.
func TestAdversarialPeopleCarriesRequiredCases(t *testing.T) {
	src := AdversarialPeople()
	pages, err := servedPeoplePages(src)
	if err != nil {
		t.Fatalf("walking adversarial people: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("adversarial people is %d page(s): need a page boundary", len(pages))
	}
	var (
		cutName    bool
		emptyName  bool
		longHandle bool
	)
	for _, page := range pages {
		for _, u := range page.Rows {
			if len([]rune(u.Name)) > 40 {
				cutName = true
			}
			if u.Name == "" {
				emptyName = true
			}
			if len(u.ScreenName) >= 15 {
				longHandle = true
			}
		}
	}
	if !cutName || !emptyName || !longHandle {
		t.Errorf("adversarial people missing cases: cut=%v empty=%v longHandle=%v", cutName, emptyName, longHandle)
	}
}
