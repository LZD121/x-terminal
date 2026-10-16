package fake

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// Package comment reminder: this file is a test double. No clock, no I/O, no package
// outside the standard library, internal/domain, and the fixture loader fake.go already has
// — plus volatile.ErrUnknownOperation so Unavailable can name the capability state without a
// network and without a pins file.

const (
	savedBookmarkCursorPrefix = "saved-bookmark-"
	savedLikeCursorPrefix     = "saved-like-"
)

//go:embed testdata/pages-saved.json
var recordedSavedJSON []byte

// savedDoc is the on-disk shape: one array of pages (each an array of posts) per saved kind.
type savedDoc struct {
	Bookmark [][]domain.Post `json:"bookmark"`
	Like     [][]domain.Post `json:"like"`
}

// The saved-list knobs live on Source so a screen test can drive FailSaved / Unavailable /
// StallSavedWith without a second type. They are declared here rather than in fake.go so that
// file stays untouched by this plan.
var (
	savedOnce = sync.OnceValues(func() (savedDoc, error) {
		var doc savedDoc
		if err := json.Unmarshal(recordedSavedJSON, &doc); err != nil {
			return savedDoc{}, fmt.Errorf("fake: testdata/pages-saved.json is not a saved-pages document: %w", err)
		}
		if len(doc.Bookmark) == 0 {
			return savedDoc{}, fmt.Errorf("fake: testdata/pages-saved.json holds no bookmark pages")
		}
		return doc, nil
	})
)

func initSavedArms(s *Source) {
	if s.savedPages != nil {
		return
	}
	doc, err := savedOnce()
	if err != nil {
		s.savedUnreadable = err
		s.savedPages = map[domain.SavedKind][]domain.PostPage{}
		return
	}
	s.savedPages = map[domain.SavedKind][]domain.PostPage{
		domain.Bookmark: cursorPages(doc.Bookmark, savedBookmarkCursorPrefix),
		domain.Like:     cursorPages(doc.Like, savedLikeCursorPrefix),
	}
}

// Saved serves one page of a saved list from the recorded fixture's parse.
//
// Every call counts in Calls() the same way Home does, so a screen test can assert a saved
// frame issued zero requests. The cursors are synthetic and opaque (saved-bookmark-1, …).
func (s *Source) Saved(ctx context.Context, kind domain.SavedKind, c domain.Cursor) (domain.PostPage, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	initSavedArms(s)

	if err, ok := s.failSaved[kind]; ok {
		delete(s.failSaved, kind)
		return domain.PostPage{}, err
	}
	if s.unavailable != nil && s.unavailable[kind] {
		return domain.PostPage{}, fmt.Errorf("%w: %q is not in the snapshot and no pin names it", volatile.ErrUnknownOperation, kind.List())
	}
	if s.savedUnreadable != nil {
		return domain.PostPage{}, s.savedUnreadable
	}
	pages := s.savedPages[kind]
	if len(pages) == 0 {
		return domain.PostPage{}, fmt.Errorf("fake: no saved pages for %s", kind.List())
	}
	page, ok := savedPageFor(pages, c)
	if !ok {
		return domain.PostPage{}, fmt.Errorf("fake: no saved page follows the cursor %q for %s", c, kind.List())
	}
	page.FetchedAt = frozenAt
	page.Requested = len(page.Posts)
	if s.stallSaved != nil {
		if stall, ok := s.stallSaved[kind]; ok {
			page.Next = stall
		}
	}
	return page, nil
}

func savedPageFor(pages []domain.PostPage, c domain.Cursor) (domain.PostPage, bool) {
	if c == "" {
		if len(pages) == 0 {
			return domain.PostPage{}, false
		}
		return pages[0], true
	}
	for i := 0; i+1 < len(pages); i++ {
		if pages[i].Next == c {
			return pages[i+1], true
		}
	}
	return domain.PostPage{}, false
}

// FailSaved makes the next Saved call for kind fail with err, and only that call.
func (s *Source) FailSaved(kind domain.SavedKind, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failSaved == nil {
		s.failSaved = map[domain.SavedKind]error{}
	}
	s.failSaved[kind] = err
}

// Unavailable makes Saved for kind return the typed error the registry produces for an
// operation it does not hold, so the capability state is drivable with no network and no pins file.
func (s *Source) Unavailable(kind domain.SavedKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unavailable == nil {
		s.unavailable = map[domain.SavedKind]bool{}
	}
	s.unavailable[kind] = true
}

// StallSavedWith makes every Saved page for kind carry c as its next cursor.
func (s *Source) StallSavedWith(kind domain.SavedKind, c domain.Cursor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stallSaved == nil {
		s.stallSaved = map[domain.SavedKind]domain.Cursor{}
	}
	s.stallSaved[kind] = c
}
