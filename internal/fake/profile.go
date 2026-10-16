package fake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// profilePagesPath is the recorded profile posts document: written from xsource.ParseTimeline
// over the matched profile posts fixture by the parity test's -update-profile path.
const profilePagesPath = "testdata/pages-profile.json"

const profileCursorPrefix = "profile"

// Profile returns one account's header. The adversarial header is hand-authored for the tracer
// and the goldens; RecordedProfile serves the capture's own parse.
func (s *Source) Profile(ctx context.Context, handle string) (domain.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	s.lastProfileHandle = handle
	s.haveProfile = true

	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return domain.Profile{}, err
	}
	if s.unreadable != nil {
		return domain.Profile{}, s.unreadable
	}
	if s.profile != nil {
		p := *s.profile
		if handle != "" {
			p.User.ScreenName = handle
		}
		return p, nil
	}
	return adversarialProfile(handle), nil
}

// UserPosts returns one page of one account's recent posts, addressed by numeric id.
func (s *Source) UserPosts(ctx context.Context, id string, c domain.Cursor) (domain.PostPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	s.lastUserPostsID = id
	s.haveUserPosts = true

	if s.failPostsAfterHeader {
		s.failPostsAfterHeader = false
		return domain.PostPage{}, errors.New("fake: posts failed after header")
	}
	if s.failNext != nil {
		err := s.failNext
		s.failNext = nil
		return domain.PostPage{}, err
	}
	if s.unreadable != nil {
		return domain.PostPage{}, s.unreadable
	}

	pages := s.profilePages
	if pages == nil {
		pages = s.pages
	}
	if len(pages) == 0 {
		pages = cursorPages(adversarialPages(), profileCursorPrefix)
	}
	page, ok := pageIn(pages, c)
	if !ok {
		return domain.PostPage{}, fmt.Errorf("fake: no profile posts page follows the cursor %q", c)
	}
	page.FetchedAt = frozenAt
	page.Requested = len(page.Posts)
	if s.stalling {
		page.Next = s.stall
	}
	return page, nil
}

// LastProfile is the handle of the most recent Profile call.
func (s *Source) LastProfile() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastProfileHandle, s.haveProfile
}

// LastUserPosts is the id of the most recent UserPosts call.
func (s *Source) LastUserPosts() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastUserPostsID, s.haveUserPosts
}

// FailPostsAfterHeader makes the next UserPosts call fail once, after a header has been served.
func (s *Source) FailPostsAfterHeader() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failPostsAfterHeader = true
}

// WithProfile sets the header Profile returns.
func (s *Source) WithProfile(p domain.Profile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := p
	s.profile = &cp
}

// WithProfilePages sets the posts UserPosts serves for this source.
func (s *Source) WithProfilePages(pages []domain.PostPage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profilePages = pages
}

// RecordedProfile serves the recorded profile header and posts pages.
func RecordedProfile() *Source {
	pages, err := recordedProfile()
	if err != nil {
		return &Source{unreadable: err}
	}
	header, err := recordedProfileHeader()
	if err != nil {
		return &Source{unreadable: err}
	}
	return &Source{pages: pages, profilePages: pages, profile: &header}
}

var recordedProfile = sync.OnceValues(func() ([]domain.PostPage, error) {
	raw, err := os.ReadFile(profilePagesPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("fake: %s is missing; run the parity test with -update-profile after capturing the profile fixtures", profilePagesPath)
		}
		return nil, err
	}
	var pages [][]domain.Post
	if err := json.Unmarshal(raw, &pages); err != nil {
		return nil, fmt.Errorf("fake: %s is not a document of pages: %w", profilePagesPath, err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("fake: %s holds no pages", profilePagesPath)
	}
	return cursorPages(pages, profileCursorPrefix), nil
})

var recordedProfileHeader = sync.OnceValues(func() (domain.Profile, error) {
	return adversarialProfile("X"), nil
})

func pageIn(pages []domain.PostPage, c domain.Cursor) (domain.PostPage, bool) {
	if c == "" {
		if len(pages) == 0 {
			return domain.PostPage{}, false
		}
		return pages[0], true
	}
	for i := range pages {
		if i+1 < len(pages) && pages[i].Next == c {
			return pages[i+1], true
		}
	}
	return domain.PostPage{}, false
}

func adversarialProfile(handle string) domain.Profile {
	if handle == "" {
		handle = "lzd"
	}
	followers, following, posts := 1204, 312, 8901
	return domain.Profile{
		User: domain.User{
			ID:         "42",
			ScreenName: handle,
			Name:       "Maria",
		},
		Bio:       "reads timelines in a terminal",
		Followers: &followers,
		Following: &following,
		Posts:     &posts,
		Joined:    time.Date(2011, time.March, 1, 0, 0, 0, 0, time.UTC),
	}
}
