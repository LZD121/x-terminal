package fake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/LZD121/x-terminal/internal/domain"
)

// searchPagesPath is the recorded search pages document: written from xsource.ParseSearch over
// internal/xsource/testdata/fixture-search.json by the parity test's -update-search path, the same
// pattern as pages.json with its own flag, so Phase 2's pages cannot be rewritten by mistake.
//
// It is read lazily rather than embedded because the fixture it is written from only exists once
// the gated capture has run: a build that embedded a file nobody captured would not compile, and
// a fake that cannot load its pages must fail closed instead.
const searchPagesPath = "testdata/pages-search.json"

// Search serves one page of a keyword search.
//
// The pages are the source's own page set — the adversarial set or the recorded one — because the
// fake exists to drive lists, and the list a search drives is the same list. The query and the
// product are remembered rather than consulted: the page set does not depend on them, and a test
// asserts them to prove the request that went out is the one the frame was opened with. All the
// knobs Home documents apply: the call counter, the one-shot failure, the stall override, and the
// refusal of a cursor that names no page.
func (s *Source) Search(ctx context.Context, query string, p domain.SearchProduct, c domain.Cursor) (domain.PostPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	s.lastQuery, s.lastProduct, s.haveSearch = query, p, true

	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return domain.PostPage{}, err
	}
	if s.unreadable != nil {
		return domain.PostPage{}, s.unreadable
	}
	if s.emptySearch {
		// A successful read that carried nothing: the `no results` state, which is a success and
		// not a failure — an error here would drive the wrong arm of every screen above.
		return domain.PostPage{FetchedAt: frozenAt}, nil
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

// LastSearch is the query and product of the most recent Search call, and false before the first
// one: what a test asserts when it needs to know which search went out.
func (s *Source) LastSearch() (string, domain.SearchProduct, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastQuery, s.lastProduct, s.haveSearch
}

// EmptySearch makes every Search call answer a successful empty page. It is how a test drives the
// `no results` state: a read that succeeded and carried no rows, which is not a failure and is
// never served as one.
func (s *Source) EmptySearch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emptySearch = true
}

// RecordedSearch serves the recorded search pages, written from the captured SearchTimeline
// fixture. It fails closed when the document is not there: the capture is gated, and a fake that
// silently served nothing here would let every test above it pass for the wrong reason.
func RecordedSearch() *Source {
	pages, err := recordedSearch()
	return &Source{pages: pages, unreadable: err}
}

// recordedSearch parses the recorded search pages document once per process, if it is there.
var recordedSearch = sync.OnceValues(func() ([]domain.PostPage, error) {
	data, err := os.ReadFile(searchPagesPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("fake: %s is not there: the gated capture (X_TERMINAL_CAPTURE=1, TestCaptureSearch) writes the fixture it is parsed from, and the parity test's -update-search path writes this document", searchPagesPath)
	}
	if err != nil {
		return nil, fmt.Errorf("fake: reading %s: %w", searchPagesPath, err)
	}
	var pages [][]domain.Post
	if err := json.Unmarshal(data, &pages); err != nil {
		return nil, fmt.Errorf("fake: %s is not a document of pages: %w", searchPagesPath, err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("fake: %s holds no pages", searchPagesPath)
	}
	return cursorPages(pages, recordedSearchCursorPrefix), nil
})

// peoplePagesPath is the recorded people pages document: written from xsource.ParseSearchPeople
// over internal/xsource/testdata/fixture-search-people.json by the parity test's
// -update-search-people path.
const peoplePagesPath = "testdata/pages-search-people.json"

// SearchPeople returns one page of accounts matching the query.
//
// The pages are the source's own people page set — the adversarial set or the recorded one —
// with the same counter and knobs Search documents. The query is remembered rather than
// consulted: the page set does not depend on it, and a test asserts it to prove the request
// that went out is the one the frame was opened with.
func (s *Source) SearchPeople(ctx context.Context, query string, c domain.Cursor) (domain.UserPage, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	s.lastPeopleQuery, s.havePeople = query, true
	// A people search is also a search: LastSearch reports the People product so tests that
	// already assert LastSearch keep working when the frame was opened by u.
	s.lastQuery, s.lastProduct, s.haveSearch = query, domain.People, true

	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return domain.UserPage{}, err
	}
	if s.unreadable != nil {
		return domain.UserPage{}, s.unreadable
	}
	if s.emptySearch {
		return domain.UserPage{FetchedAt: frozenAt}, nil
	}

	page, ok := s.peoplePageFor(c)
	if !ok {
		return domain.UserPage{}, fmt.Errorf("fake: no people page follows the cursor %q; this source serves the %s people page set, whose first page is fetched with an empty cursor", c, s.peopleKindName())
	}
	page.FetchedAt = frozenAt
	page.Requested = len(page.Rows)
	if s.stalling {
		page.Next = s.stall
	}
	return page, nil
}

// LastPeopleSearch is the query of the most recent SearchPeople call, and false before the first.
func (s *Source) LastPeopleSearch() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastPeopleQuery, s.havePeople
}

// AdversarialPeople serves the hand-authored people page set the goldens need: a display name
// that must be cut, an account with no display name, a long handle, and enough accounts for a
// page boundary. The post page set stays the adversarial posts so open-into-profile still works.
func AdversarialPeople() *Source {
	return &Source{
		pages:       cursorPages(adversarialPages(), adversarialCursorPrefix),
		peoplePages: cursorUserPages(adversarialPeoplePages(), adversarialPeopleCursorPrefix),
	}
}

// RecordedPeople serves the recorded people pages written from the captured SearchTimeline
// People fixture. It fails closed when the document is not there.
func RecordedPeople() *Source {
	pages, err := recordedPeople()
	return &Source{peoplePages: pages, unreadable: err}
}

var recordedPeople = sync.OnceValues(func() ([]domain.UserPage, error) {
	data, err := os.ReadFile(peoplePagesPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("fake: %s is not there: the gated capture (X_TERMINAL_CAPTURE=1, TestCaptureSearchPeople) writes the fixture it is parsed from, and the parity test's -update-search-people path writes this document", peoplePagesPath)
	}
	if err != nil {
		return nil, fmt.Errorf("fake: reading %s: %w", peoplePagesPath, err)
	}
	var pages [][]domain.User
	if err := json.Unmarshal(data, &pages); err != nil {
		return nil, fmt.Errorf("fake: %s is not a document of pages: %w", peoplePagesPath, err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("fake: %s holds no pages", peoplePagesPath)
	}
	return cursorUserPages(pages, recordedPeopleCursorPrefix), nil
})

// peoplePageFor resolves a cursor to the people page it fetches.
func (s *Source) peoplePageFor(c domain.Cursor) (domain.UserPage, bool) {
	if c == "" {
		if len(s.peoplePages) == 0 {
			return domain.UserPage{}, false
		}
		return s.peoplePages[0], true
	}
	for i := 0; i+1 < len(s.peoplePages); i++ {
		if s.peoplePages[i].Next == c {
			return s.peoplePages[i+1], true
		}
	}
	return domain.UserPage{}, false
}

func (s *Source) peopleKindName() string {
	for _, page := range s.peoplePages {
		if strings.HasPrefix(string(page.Next), adversarialPeopleCursorPrefix) {
			return "adversarial"
		}
	}
	return "recorded"
}

// cursorUserPages gives each people page the cursor that fetches the page after it.
func cursorUserPages(pages [][]domain.User, prefix string) []domain.UserPage {
	served := make([]domain.UserPage, 0, len(pages))
	for i, page := range pages {
		next := domain.Cursor("")
		if i+1 < len(pages) {
			next = domain.Cursor(prefix + strconv.Itoa(i+1))
		}
		served = append(served, domain.UserPage{Rows: page, Next: next})
	}
	return served
}

// adversarialPeoplePages is the hand-authored people set the goldens and the agnostic tests
// drive: a cut display name, an empty display name, a long handle, and enough rows for a page
// boundary (two pages of eight).
func adversarialPeoplePages() [][]domain.User {
	const pageSize = 8
	cutName := "A display name so long it must be cut with the truncation mark in the last cell the row may occupy for the goldens"
	users := []domain.User{
		{ID: "p1", ScreenName: "alice", Name: "Alice"},
		{ID: "p2", ScreenName: "bob", Name: cutName},
		{ID: "p3", ScreenName: "noname", Name: ""},
		{ID: "p4", ScreenName: "longhandlefifteen", Name: "Fifteen"},
		{ID: "p5", ScreenName: "carol", Name: "Carol"},
		{ID: "p6", ScreenName: "dave", Name: "Dave"},
		{ID: "p7", ScreenName: "erin", Name: "Erin"},
		{ID: "p8", ScreenName: "frank", Name: "Frank"},
		{ID: "p9", ScreenName: "grace", Name: "Grace"},
		{ID: "p10", ScreenName: "heidi", Name: "Heidi"},
		{ID: "p11", ScreenName: "ivan", Name: "Ivan"},
		{ID: "p12", ScreenName: "judy", Name: "Judy"},
	}
	pages := make([][]domain.User, 0, (len(users)+pageSize-1)/pageSize)
	for start := 0; start < len(users); start += pageSize {
		end := start + pageSize
		if end > len(users) {
			end = len(users)
		}
		pages = append(pages, users[start:end])
	}
	return pages
}
