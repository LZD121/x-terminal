// Package fake is the domain.Source the tests are driven by: the recorded fixture's posts, or the
// hand-authored set the golden frames are rendered from, served with no network, no session, no
// clock and no file access.
//
// It is a test double and nothing else, and the doc comment says so because the rule it breaks is
// deliberate: the real adapter (internal/xsource) is the only production implementation of
// domain.Source, internal/cli is the only package that names both, and the shell may not reach
// this one at all (internal/tui's boundary test forbids it). What the fake buys is that every
// screen in this phase can be driven to any state the state machine defines — an empty page, a
// page that adds nothing, a failed page, a cursor that repeats, a soft ban — with no socket, no
// sleep and no terminal.
//
// Nothing here invents a page to make a test pass: Recorded serves the recorded response's own
// posts (and internal/fake/parity_test.go proves it), and Adversarial is the documented content
// set the golden frames need.
package fake

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// The two page sets this package serves, in the order PostPage.Next hands them out: the first
// page is fetched with an empty cursor and every later one with the cursor the page before it
// carried. The cursors are synthetic and opaque — nothing above the port may read one, and a test
// that pokes at one is testing the fake rather than the list.
const (
	recordedCursorPrefix    = "recorded-"
	adversarialCursorPrefix = "adversarial-"
	// recordedSearchCursorPrefix is the recorded search page set's cursor prefix: the recorded
	// cursor prefix, namespaced so a test cannot confuse the two sets' cursors.
	recordedSearchCursorPrefix = recordedCursorPrefix + "search-"
	// recordedPeopleCursorPrefix is the recorded people page set's cursor prefix.
	recordedPeopleCursorPrefix = recordedCursorPrefix + "people-"
	// adversarialPeopleCursorPrefix is the adversarial people page set's cursor prefix.
	adversarialPeopleCursorPrefix = adversarialCursorPrefix + "people-"
)

// recordedPageSize is how many of the recorded fixture's posts one served page carries. Eight is
// small enough that the recorded twenty posts come back as three pages, which is what makes
// pagination drivable at all, and it is a constant so that the split cannot drift between the
// source and the test that proves it equal to the fixture.
const recordedPageSize = 8

// frozenAt is the instant every served page is stamped with, and the stamp the adversarial set's
// posts carry. The fake has no clock of its own: a test that wants a different instant asks for
// this one, and nothing in this phase renders FetchedAt.
var frozenAt = time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC)

//go:embed testdata/pages.json
var recordedJSON []byte

// Source is a domain.Source served from memory.
type Source struct {
	mu sync.Mutex

	// pages is the page set this source serves, in list order. The cursors between them are
	// already computed: a page's Next is the cursor that fetches the page after it.
	pages []domain.PostPage
	// unreadable is why a page set could not be read at all. It is what every call reports, rather
	// than an empty timeline: a fake that answered "no posts" here would let every test above it
	// pass for the wrong reason.
	unreadable error

	calls    int
	lastKind domain.HomeKind
	haveKind bool
	// lastQuery and lastProduct are the most recent Search call's arguments, remembered so a test
	// can prove the request that went out is the one the frame was opened with. emptySearch is the
	// EmptySearch knob: a successful read that carries no rows.
	lastQuery   string
	lastProduct domain.SearchProduct
	haveSearch  bool
	emptySearch bool
	// peoplePages is the account page set SearchPeople serves. When nil, SearchPeople fails closed
	// naming the missing set rather than inventing an empty page.
	peoplePages     []domain.UserPage
	lastPeopleQuery string
	havePeople      bool
	// Profile / UserPosts bookkeeping and knobs. profile is the header Profile returns when set;
	// profilePages is the posts page set UserPosts serves when set, otherwise pages. failPostsAfterHeader
	// is the goldens' posts-failure-with-header-kept knob.
	profile              *domain.Profile
	profilePages         []domain.PostPage
	lastProfileHandle    string
	haveProfile          bool
	lastUserPostsID      string
	haveUserPosts        bool
	failPostsAfterHeader bool
	failNext             error
	stalling             bool
	stall                domain.Cursor
	// Saved-list state (see saved.go). Declared here so Saved can live in its own file
	// without rewriting the Source type twice.
	savedPages      map[domain.SavedKind][]domain.PostPage
	savedUnreadable error
	failSaved       map[domain.SavedKind]error
	unavailable     map[domain.SavedKind]bool
	stallSaved      map[domain.SavedKind]domain.Cursor
}

// Recorded serves the recorded fixture's posts, split into pages of recordedPageSize.
//
// The posts come from the checked-in testdata/pages.json, which is written from
// xsource.ParseTimeline over internal/xsource/testdata/fixture-usertweets.json — the same body the
// live adapter reads — so a screen driven by this source is driven by the real response and not by
// a hand-written stand-in.
func Recorded() *Source {
	pages, err := recorded()
	return &Source{pages: pages, unreadable: err}
}

// recorded parses the embedded page document once per process.
var recorded = sync.OnceValues(func() ([]domain.PostPage, error) {
	var pages [][]domain.Post
	if err := json.Unmarshal(recordedJSON, &pages); err != nil {
		return nil, fmt.Errorf("fake: testdata/pages.json is not a document of pages: %w", err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("fake: testdata/pages.json holds no pages: an empty document would serve an empty timeline that no test could tell from a broken one")
	}
	for i, page := range pages {
		if len(page) == 0 {
			return nil, fmt.Errorf("fake: testdata/pages.json: page %d is empty", i+1)
		}
	}
	return cursorPages(pages, recordedCursorPrefix), nil
})

// Adversarial serves the hand-authored page set the golden frames are rendered from: the cases the
// UI-SPEC's width model and markers exist for, and the ones the recorded response has no example
// of (a tombstone, a limited post, an alt-texted image, an animated gif, a 15-cell handle).
func Adversarial() *Source {
	return &Source{pages: cursorPages(adversarialPages(), adversarialCursorPrefix)}
}

// splitRecorded cuts the recorded fixture's posts into the pages Recorded serves. It is here
// rather than in the test so that the writer of testdata/pages.json and the reader of it cannot
// disagree about the split.
func splitRecorded(posts []domain.Post) [][]domain.Post {
	pages := make([][]domain.Post, 0, (len(posts)+recordedPageSize-1)/recordedPageSize)
	for start := 0; start < len(posts); start += recordedPageSize {
		end := min(start+recordedPageSize, len(posts))
		pages = append(pages, posts[start:end])
	}
	return pages
}

// cursorPages gives each page the cursor that fetches the page after it, and the empty cursor to the
// last one. A cursor names the position it stands for, not the page it fetches: `recorded-2` is the
// position after the second page of the recorded set, and fetching it returns the third. Nothing
// above the port may read one, which is why the walk is driven by handing back whatever the last
// answer carried.
func cursorPages(pages [][]domain.Post, prefix string) []domain.PostPage {
	served := make([]domain.PostPage, 0, len(pages))
	for i, page := range pages {
		next := domain.Cursor("")
		if i+1 < len(pages) {
			next = domain.Cursor(prefix + strconv.Itoa(i+1))
		}
		served = append(served, domain.PostPage{Posts: page, Next: next})
	}
	return served
}

// Home serves one page of the walk.
//
// The empty cursor fetches page one; every other cursor is one a page handed back. A cursor that
// names no page is an error naming it, never an empty page: a fake that answered nothing here would
// hide a state machine that invented a cursor.
func (s *Source) Home(ctx context.Context, kind domain.HomeKind, c domain.Cursor) (domain.PostPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	s.lastKind, s.haveKind = kind, true

	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return domain.PostPage{}, err
	}
	if s.unreadable != nil {
		return domain.PostPage{}, s.unreadable
	}

	page, ok := s.pageFor(c)
	if !ok {
		return domain.PostPage{}, fmt.Errorf("fake: no page follows the cursor %q; this source serves the %s page set, whose first page is fetched with an empty cursor", c, s.kindName())
	}
	page.FetchedAt = frozenAt
	page.Requested = len(page.Posts)
	if s.stalling {
		page.Next = s.stall
	}
	return page, nil
}

// User answers with a profile for the handle. The fake has no account and no profile to show: the
// handle is echoed as the identity, which is enough for two handles to be told apart, and no
// screen in this phase renders a profile.
func (s *Source) User(ctx context.Context, handle string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return domain.User{}, err
	}
	if s.unreadable != nil {
		return domain.User{}, s.unreadable
	}
	return domain.User{ID: handle, ScreenName: handle, Name: handle}, nil
}

// Calls is how many Home and User calls this source has served. A test asserts it does not move to
// prove that no request was issued — after a stall, after a soft ban, and while a page is in
// flight.
func (s *Source) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// LastKind is the feed kind of the most recent Home call, and false before the first one.
func (s *Source) LastKind() (domain.HomeKind, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastKind, s.haveKind
}

// FailNext makes the next Home or User call fail with err, and only that call: one failure is what
// a partial-failure test needs, and a fake that kept failing would make the second half of such a
// test unreachable.
func (s *Source) FailNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

// StallWith makes every page this source hands out carry c as its next cursor, instead of the one
// the page set carries. It is how a test drives the cursor that repeats — the upstream answering
// "older" with a cursor the walk has already used — and the walk should end there rather than
// loop. The override stays until the source is discarded.
func (s *Source) StallWith(c domain.Cursor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stalling, s.stall = true, c
}

// pageFor resolves a cursor to the page it fetches.
func (s *Source) pageFor(c domain.Cursor) (domain.PostPage, bool) {
	if c == "" {
		if len(s.pages) == 0 {
			return domain.PostPage{}, false
		}
		return s.pages[0], true
	}
	for i := 0; i+1 < len(s.pages); i++ {
		if s.pages[i].Next == c {
			return s.pages[i+1], true
		}
	}
	return domain.PostPage{}, false
}

// kindName says which page set is being served, for an error message. It is read off the cursors
// the set hands out rather than from a field, because the page set is the only thing that knows
// which one it is.
func (s *Source) kindName() string {
	for _, page := range s.pages {
		if strings.HasPrefix(string(page.Next), adversarialCursorPrefix) {
			return "adversarial"
		}
	}
	return "recorded"
}

// adversarialPages is the hand-authored page set the golden frames are rendered from.
//
// Every case the UI-SPEC's width model, markers and placeholders name has a post here, in the
// three pages a walk of it produces: the first two pages end with a stored cursor and the third
// ends the list, so `HasMore`, the position's `+` and end-of-list are all reachable. The cases
// that need a shape the recorded response has no example of — a tombstone, a limited post, an
// animated gif, alt text, a 15-cell handle — are here and nowhere else.
func adversarialPages() [][]domain.Post {
	// The continuation pair: one conversation, and the second post answering the first. A
	// renderer prints the second with the continued marker and no blank row between them.
	const conversation = "adv-conversation-1"
	first := adversarialPost(8, "lzd", "the first half of a longer thought")
	second := adversarialPost(9, "lzd", "and the second half of it")
	second.ConversationID = conversation
	second.ReplyTo = first.ID

	// The 15-cell handle: one cell short of what the plain grammar's 16-cell handle column holds,
	// which is the widest a handle may be and still be printed whole.
	widest := adversarialPost(0, "a_15_cell_handl", "a fifteen-cell handle, the widest the column holds")

	// The two runs the width model exists for: a CJK run, a single emoji and a ZWJ family, none of
	// which is one cell per rune.
	//
	// The family is written as escapes because the sequence is three wide code points joined by two
	// zero-width joiners, and a file that loses the joiners carries four separate emoji instead:
	// four clusters that measure eight cells in either environment, where the family is one two-cell
	// cluster in a bare terminal and the six cells of its code points under a multiplexer. The
	// corpus's frames are where that difference has to be visible (internal/render's tmux twins), so
	// the joiners are written in a form no editor can strip.
	cjk := adversarialPost(1, "lzd", "日本語のタイムラインを端末で読む。カーソルが戻ってきたら、そこが終わり。")
	emoji := adversarialPost(2, "lzd", "🚀")
	family := adversarialPost(3, "lzd", "\U0001F468\u200D\U0001F469\u200D\U0001F467")

	// A URL longer than the widest measure, and the four lines that make an off-by-one in the wrap
	// visible: one that fills the 120-column measure (96 cells) exactly and one a cell shorter, and
	// the same pair for the 80-column measure (77). Their widths are built, not counted by hand.
	url := adversarialPost(4, "lzd", "the link that runs past the measure: https://example.invalid/2026/10/14/a-path-that-cannot-fit-on-one-line/a-fragment#and-a-query?after=that")
	// The body's own links, one of each kind: the corpus's case for the links menu, and the one
	// post in the set whose body names things a reader can follow.
	url.Links = []domain.Link{
		{Kind: domain.LinkURL, Target: "https://example.invalid/2026/10/14/a-path-that-cannot-fit-on-one-line/a-fragment#and-a-query?after=that"},
		{Kind: domain.LinkHashtag, Target: "example"},
		{Kind: domain.LinkMention, Target: "someone"},
	}
	widths := adversarialPost(5, "lzd", strings.Join([]string{
		measure(96, "exactly the measure at 120 columns"),
		measure(95, "one cell shorter than that"),
		measure(77, "exactly the measure at 80 columns"),
		measure(76, "one cell shorter than that"),
	}, "\n"))

	// The two quote depths, and a body followed by a quote.
	quoted := adversarialPost(100, "lzd", "the quoted body, one level down")
	quoted.CreatedAt = frozenAt.Add(-time.Hour)
	depthOne := adversarialPost(6, "lzd", "a body long enough that the quote below it is the second thing a reader sees, which is where the quote prefix has to hold its column.")
	depthOne.Quote = &quoted

	// The depth-2 case is the same quoted post with a post of its own under it, so the two frames
	// differ in one thing: one more quote level.
	quotedTwice := quoted
	quotedTwice.Quote = &domain.Post{
		ID: "adv-quote-2", Author: domain.User{ID: "u-lzd", ScreenName: "lzd", Name: "name of lzd"},
		Text: "and the post that post quotes, two levels down", CreatedAt: frozenAt.Add(-2 * time.Hour),
	}
	depthTwo := adversarialPost(7, "lzd", "a body with a quote under it")
	depthTwo.Quote = &quotedTwice

	// The repost: the original's author, with the account that reposted it named separately.
	repost := adversarialPost(10, "someone else", "a post that was reposted")
	repost.RepostedBy = &domain.User{ID: "u-lzd", ScreenName: "lzd", Name: "name of lzd"}

	// A limited post: the body is served, and the wrapper says the upstream is holding it back.
	limited := adversarialPost(11, "lzd", "a body that is still served under a limited post")
	limited.Visibility = domain.Limited

	// The tombstone: no author, no body and no date, which is what the parser reads out of a
	// TweetTombstone. It is a post, and it is the only post in the set with nothing to render.
	tombstone := domain.Post{ID: "adv-tombstone", Visibility: domain.Unavailable}

	// Attachments: one noun in the singular, and a plural line that has to keep the grammar's fixed
	// noun order (image, video, gif).
	single := adversarialPost(13, "lzd", "one attachment")
	single.Media = []domain.MediaRef{{Kind: domain.Image, URL: "https://example.invalid/one.jpg", AltText: "a screenshot of the pane"}}
	several := adversarialPost(14, "lzd", "four attachments")
	several.Media = []domain.MediaRef{
		{Kind: domain.Image, URL: "https://example.invalid/a.jpg"},
		{Kind: domain.Image, URL: "https://example.invalid/b.jpg"},
		{Kind: domain.Video, URL: "https://example.invalid/c.jpg"},
		{Kind: domain.GIF, URL: "https://example.invalid/d.jpg"},
	}

	return [][]domain.Post{
		{widest, cjk, emoji, family, url, widths, depthOne, depthTwo, first},
		{second, repost, limited, tombstone, single},
		{several, adversarialPost(15, "lzd", "the last post of the set, so that the end of the walk has something in it")},
	}
}

// adversarialPost builds one post of the adversarial set. index stamps it — one minute older each
// time, so the list's order is the answer's order — and doubles as its id.
func adversarialPost(index int, handle, body string) domain.Post {
	id := "adv-" + strconv.Itoa(index)
	return domain.Post{
		ID:             domain.PostID(id),
		Author:         domain.User{ID: "u-" + handle, ScreenName: handle, Name: "name of " + handle},
		Text:           body,
		CreatedAt:      frozenAt.Add(-time.Duration(index) * time.Minute),
		Lang:           "en",
		ConversationID: id,
	}
}

// measure is one line of text that is exactly width cells wide: the words, a space, and then
// ASCII dashes to the measure. It exists so the two lines the wrap boundary is tested with are
// exactly as wide as they claim, without a hand-counted run of characters that rots the first
// time somebody edits a word. It counts cells as bytes, which is the same for ASCII.
func measure(width int, words string) string {
	if width <= len(words) {
		return words
	}
	return words + " " + strings.Repeat("-", width-len(words)-1)
}
