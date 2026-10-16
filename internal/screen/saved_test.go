package screen

import (
	"errors"
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

func TestNoticeForTable(t *testing.T) {
	cases := []struct {
		name string
		p    domain.SyncProgress
		want SyncNotice
	}{
		{"absent wins", domain.SyncProgress{Absent: 3, Added: 2, Stop: domain.StopEndOfList}, SyncNoticeNoLongerSaved},
		{"stalled", domain.SyncProgress{Stop: domain.StopStalled}, SyncNoticeStalled},
		{"complete", domain.SyncProgress{Stop: domain.StopEndOfList}, SyncNoticeComplete},
		{"new posts", domain.SyncProgress{Added: 4, Stop: domain.StopNoUnknownIDs}, SyncNoticeNewPosts},
		{"no new posts", domain.SyncProgress{Stop: domain.StopNoUnknownIDs}, SyncNoticeNoNewPosts},
	}
	for _, tc := range cases {
		if got := NoticeFor(tc.p); got != tc.want {
			t.Errorf("%s: NoticeFor = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestSavedAbsorbsMirrorPage(t *testing.T) {
	s := NewSaved(domain.ListBookmarks, domain.CapNone)
	page := domain.PostPage{
		Posts: []domain.Post{{
			ID: "p1", Author: domain.User{ID: "u1", ScreenName: "alice"},
			Text: "hello", CreatedAt: time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC),
		}},
		Next: "c1",
	}
	s = s.Update(MirrorPage{Page: page})
	if got := len(s.List().Posts()); got != 1 {
		t.Fatalf("posts = %d, want 1", got)
	}
	if !s.List().HasMore() {
		t.Fatal("want HasMore after a page with a next cursor")
	}
}

func TestSavedMirrorFailedSetsCapability(t *testing.T) {
	s := NewSaved(domain.ListBookmarks, domain.CapNone)
	page := domain.PostPage{Posts: []domain.Post{{ID: "p1", Author: domain.User{ID: "u1", ScreenName: "a"}, Text: "x"}}}
	s = s.Update(MirrorPage{Page: page})
	s = s.Update(MirrorFailed{Err: errors.New("boom"), Cap: domain.CapMirrorUnreadable})
	if s.Capability() != domain.CapMirrorUnreadable {
		t.Fatalf("capability = %v, want CapMirrorUnreadable", s.Capability())
	}
	if got := len(s.List().Posts()); got != 1 {
		t.Fatalf("rows changed on failure: %d", got)
	}
}

// TestSavedCapabilityGoesWhenTheFramesOwnActionSucceeds is the UI-SPEC's stickiness rule
// implemented (06-UI-SPEC.md:421-423): a capability state stays until an action of its own frame
// succeeds, and nothing else clears it. Each action releases what it proves:
//
//   - a page read that delivered rows proves the file could be read, so it releases the
//     file-unreadable state;
//   - a sync that reached its own end read and wrote the file and passed the owner check, so it
//     releases whatever the frame wore;
//   - a failed action releases nothing (the state outlives a failure, which is what stickiness
//     means for a remedy that has not happened yet);
//   - a page read does not release the account state: a page comes off the disk and says nothing
//     about which account the session is, and its remedy is a sync that succeeds after the account
//     is re-pointed;
//   - a run the store recorded as a failure (StopFailure) is not a sync that succeeded.
func TestSavedCapabilityGoesWhenTheFramesOwnActionSucceeds(t *testing.T) {
	page := domain.PostPage{Posts: []domain.Post{{ID: "p1", Author: domain.User{ID: "u1", ScreenName: "a"}, Text: "x"}}}
	failed := func(c domain.Capability) Saved {
		s := NewSaved(domain.ListBookmarks, domain.CapNone)
		return s.Update(MirrorFailed{Err: errors.New("boom"), Cap: c})
	}

	t.Run("a read that delivered its page clears the file state", func(t *testing.T) {
		s := failed(domain.CapMirrorUnreadable).Update(MirrorPage{Page: page})
		if got := s.Capability(); got != domain.CapNone {
			t.Errorf("capability after a page arrived = %v, want CapNone: the read succeeded, so the file state it refutes is over", got)
		}
	})

	t.Run("a failed read leaves it", func(t *testing.T) {
		s := failed(domain.CapMirrorUnreadable).Update(MirrorFailed{Err: errors.New("boom"), Cap: domain.CapNone})
		if got := s.Capability(); got != domain.CapMirrorUnreadable {
			t.Errorf("capability after a second failed read = %v, want it to stay CapMirrorUnreadable: nothing succeeded", got)
		}
	})

	t.Run("a capability the frame was handed is cleared the same way", func(t *testing.T) {
		s := NewSaved(domain.ListBookmarks, domain.CapNone).
			Update(CapabilitySet{Cap: domain.CapMirrorUnreadable}).
			Update(MirrorPage{Page: page})
		if got := s.Capability(); got != domain.CapNone {
			t.Errorf("capability after a page arrived = %v, want CapNone: how the state was set does not change what a successful read of the list proves", got)
		}
	})

	t.Run("a page read does not clear the account state", func(t *testing.T) {
		s := failed(domain.CapWrongAccount).Update(MirrorPage{Page: page})
		if got := s.Capability(); got != domain.CapWrongAccount {
			t.Errorf("capability after a page arrived = %v, want CapWrongAccount: a page off the disk says nothing about which account the session is", got)
		}
	})

	t.Run("a sync that finished clears every state", func(t *testing.T) {
		for _, c := range []domain.Capability{domain.CapOperationMissing, domain.CapWrongAccount, domain.CapMirrorUnreadable} {
			s := failed(c).Update(SyncFinished{Progress: domain.SyncProgress{
				Added: 1, Full: true, Done: true, Stop: domain.StopEndOfList,
			}})
			if got := s.Capability(); got != domain.CapNone {
				t.Errorf("capability %v after a sync that finished = %v, want CapNone", c, got)
			}
		}
	})

	t.Run("a run recorded as a failure clears nothing", func(t *testing.T) {
		s := failed(domain.CapMirrorUnreadable).Update(SyncFinished{Progress: domain.SyncProgress{
			Done: true, Stop: domain.StopFailure,
		}})
		if got := s.Capability(); got != domain.CapMirrorUnreadable {
			t.Errorf("capability after a walk that recorded a failure = %v, want it to stay CapMirrorUnreadable", got)
		}
	})
}

func TestSavedSyncSteppedAndFinished(t *testing.T) {
	s := NewSaved(domain.ListBookmarks, domain.CapNone)
	s = s.Update(SyncStarted{Full: true})
	if !s.Sync().Running || !s.Sync().Full {
		t.Fatalf("after SyncStarted: %+v", s.Sync())
	}
	s = s.Update(SyncStepped{Progress: domain.SyncProgress{Added: 5, Full: true}})
	if s.Sync().Added != 5 || !s.Sync().Running {
		t.Fatalf("after SyncStepped: %+v", s.Sync())
	}
	s = s.Update(SyncFinished{Progress: domain.SyncProgress{
		Added: 5, Absent: 2, Full: true, Done: true, Stop: domain.StopEndOfList,
	}})
	st := s.Sync()
	if st.Running || !st.Done || st.Absent != 2 || st.Stop != domain.StopEndOfList {
		t.Fatalf("after SyncFinished: %+v", st)
	}
}

func TestSavedStatusRead(t *testing.T) {
	s := NewSaved(domain.ListLikes, domain.CapNone)
	at := time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)
	s = s.Update(StatusRead{Owner: "mira", LastSuccess: at})
	if s.Owner() != "mira" || !s.Synced().Equal(at) {
		t.Fatalf("status = %q %v", s.Owner(), s.Synced())
	}
}
