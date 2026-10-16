package fake

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/LZD121/x-terminal/internal/domain"
)

// The three conversation fixtures this package serves, each an array of domain.ThreadPage
// values written by thread_parity_test.go from xsource.ParseThread over the corresponding
// TweetDetail body. A source is one of the three; the shared Source struct is not grown.
//
//go:embed testdata/thread.json
var threadRecordedJSON []byte

//go:embed testdata/thread-shapes.json
var threadShapesJSON []byte

//go:embed testdata/thread-tombstone.json
var threadTombstoneJSON []byte

// Thread is a domain.Threads served from memory: one fixture's pages, with the same call
// counting and failure injection Source exposes for the home timeline.
type Thread struct {
	mu sync.Mutex

	// pages is the page set this source serves, in walk order. The empty cursor fetches the
	// first; a page's Next is the cursor that fetches the page after it. An unknown cursor is
	// the end of the walk — an empty Next and no groups — never an invented page.
	pages []domain.ThreadPage
	// unreadable is why a page set could not be read at all. Every call reports it rather than
	// answering with an empty conversation that no test could tell from a broken one.
	unreadable error

	calls    int
	failNext error
	stalling bool
	stall    domain.Cursor
}

// ThreadRecorded serves the recorded TweetDetail body's page (and any later page the fixture
// holds). The body's own Next distinguishes it from the tombstone body's empty-cursor page.
func ThreadRecorded() *Thread {
	pages, err := threadRecorded()
	return &Thread{pages: pages, unreadable: err}
}

// ThreadShapes serves the synthetic shapes body: ancestors, a self-thread, a module cursor, a
// tombstone, a limited post, a withheld post, a depth-2 quote, a rail entry and a promoted item.
func ThreadShapes() *Thread {
	pages, err := threadShapes()
	return &Thread{pages: pages, unreadable: err}
}

// ThreadTombstone serves the synthetic tombstone body: a focal post that is a tombstone with two
// replies, and no next cursor on its first page.
func ThreadTombstone() *Thread {
	pages, err := threadTombstone()
	return &Thread{pages: pages, unreadable: err}
}

var threadRecorded = sync.OnceValues(func() ([]domain.ThreadPage, error) {
	return loadThreadPages(threadRecordedJSON, "thread.json")
})

var threadShapes = sync.OnceValues(func() ([]domain.ThreadPage, error) {
	return loadThreadPages(threadShapesJSON, "thread-shapes.json")
})

var threadTombstone = sync.OnceValues(func() ([]domain.ThreadPage, error) {
	return loadThreadPages(threadTombstoneJSON, "thread-tombstone.json")
})

func loadThreadPages(raw []byte, name string) ([]domain.ThreadPage, error) {
	var pages []domain.ThreadPage
	if err := json.Unmarshal(raw, &pages); err != nil {
		return nil, fmt.Errorf("fake: testdata/%s is not a document of thread pages: %w", name, err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("fake: testdata/%s holds no pages: an empty document would serve an empty conversation that no test could tell from a broken one", name)
	}
	return pages, nil
}

// Thread serves one page of the conversation walk.
//
// The empty cursor fetches page one; every other cursor is one a page handed back. A cursor the
// fixture does not hold is the end of the walk: an empty Next and no groups, never an invented
// page and never an error that would look like a soft ban.
func (s *Thread) Thread(ctx context.Context, id domain.PostID, c domain.Cursor) (domain.ThreadPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++

	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return domain.ThreadPage{}, err
	}
	if s.unreadable != nil {
		return domain.ThreadPage{}, s.unreadable
	}

	page, ok := s.pageFor(c)
	if !ok {
		page = domain.ThreadPage{}
		if s.stalling {
			page.Next = s.stall
		}
		return page, nil
	}
	page.FetchedAt = frozenAt
	if s.stalling {
		page.Next = s.stall
	}
	return page, nil
}

// Calls is how many Thread calls this source has served. A test asserts it does not move to prove
// that no request was issued — after a stall, while a page is in flight, and when popping a frame.
func (s *Thread) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// FailNext makes the next Thread call fail with err, and only that call: one failure is what a
// cold-start failure and a failed reply page are tested with.
func (s *Thread) FailNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

// StallWith makes every page this source hands out carry c as its next cursor, instead of the one
// the page set carries. It is how a test drives the cursor that repeats — the upstream answering
// "more" with a cursor the walk has already used — and the walk should end there rather than loop.
func (s *Thread) StallWith(c domain.Cursor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stalling, s.stall = true, c
}

// pageFor resolves a cursor to the page it fetches.
func (s *Thread) pageFor(c domain.Cursor) (domain.ThreadPage, bool) {
	if c == "" {
		if len(s.pages) == 0 {
			return domain.ThreadPage{}, false
		}
		return s.pages[0], true
	}
	for i := 0; i+1 < len(s.pages); i++ {
		if s.pages[i].Next == c {
			return s.pages[i+1], true
		}
	}
	return domain.ThreadPage{}, false
}
