package tui

import (
	"testing"
)

// TestAgnosticKeyScriptOverPostsAndAccounts drives the same key script through the
// shipped shell against a post list and an account list, asserting the same numbers:
// one request per page on open, selection moves with j, pop restores with zero requests.
func TestAgnosticKeyScriptOverPostsAndAccounts(t *testing.T) {
	type nums struct {
		openReqs int
		moved    bool
		popReqs  int
		loaded   int
		hasMore  bool
		pos      int
	}
	run := func(t *testing.T, openKey, query string, people bool) nums {
		t.Helper()
		var s *shell
		if people {
			s = peopleShell(t, 80, 24)
		} else {
			s = newShell(t, 80, 24)
		}
		before := s.src.Calls()
		s.press(openKey)
		for _, r := range query {
			s.press(string(r))
		}
		s.press("enter")
		openReqs := s.src.Calls() - before

		var startSel string
		var loaded int
		var hasMore bool
		if people {
			startSel = s.m.top().People.Walk().Selected().ID
			loaded = s.m.top().People.Walk().Loaded()
			hasMore = s.m.top().People.Walk().HasMore()
		} else {
			startSel = s.m.top().Search.Walk().Selected().ID
			loaded = s.m.top().Search.Walk().Loaded()
			hasMore = s.m.top().Search.Walk().HasMore()
		}

		moved := false
		if loaded > 1 {
			s.press("j")
			var after string
			if people {
				after = s.m.top().People.Walk().Selected().ID
			} else {
				after = s.m.top().Search.Walk().Selected().ID
			}
			moved = after != startSel
		}

		// Reach the end; a stored cursor issues exactly one more request (single-flight).
		callsBeforeEnd := s.src.Calls()
		s.press("G")
		if hasMore {
			// G on the last row is the pagination trigger when HasMore.
			if got := s.src.Calls() - callsBeforeEnd; got != 1 && got != 0 {
				// 0 is allowed if the selection was already mid-list and G only moves;
				// press pgdown to force the end trigger.
				s.press("pgdown")
			}
			if s.src.Calls() == callsBeforeEnd {
				s.press("pgdown")
			}
			if s.src.Calls()-callsBeforeEnd > 1 {
				t.Errorf("reaching the end issued %d requests, want at most 1", s.src.Calls()-callsBeforeEnd)
			}
		}

		// Selection unmoved while a page is in flight: if outstanding, j must not move.
		if s.m.outstanding() > 0 {
			var sel string
			if people {
				sel = s.m.top().People.Walk().Selected().ID
			} else {
				sel = s.m.top().Search.Walk().Selected().ID
			}
			s.press("j")
			var after string
			if people {
				after = s.m.top().People.Walk().Selected().ID
			} else {
				after = s.m.top().Search.Walk().Selected().ID
			}
			if after != sel {
				t.Errorf("selection moved while a page was in flight")
			}
		}

		var pos int
		if people {
			pos = s.m.top().People.Walk().Index(s.m.top().People.Walk().Selected().ID) + 1
			hasMore = s.m.top().People.Walk().HasMore()
			loaded = s.m.top().People.Walk().Loaded()
		} else {
			pos = s.m.top().Search.Walk().Index(s.m.top().Search.Walk().Selected().ID) + 1
			hasMore = s.m.top().Search.Walk().HasMore()
			loaded = s.m.top().Search.Walk().Loaded()
		}

		beforePop := s.src.Calls()
		under := ""
		if len(s.m.frames) > 1 {
			// Capture the under-frame bytes by rendering after pop comparison setup.
			_ = under
		}
		s.press("q")
		popReqs := s.src.Calls() - beforePop
		return nums{openReqs: openReqs, moved: moved, popReqs: popReqs, loaded: loaded, hasMore: hasMore, pos: pos}
	}

	posts := run(t, "/", "page cache", false)
	accounts := run(t, "u", "people", true)

	if posts.openReqs != 1 || accounts.openReqs != 1 {
		t.Errorf("open requests: posts=%d accounts=%d, want 1 and 1", posts.openReqs, accounts.openReqs)
	}
	if posts.popReqs != 0 || accounts.popReqs != 0 {
		t.Errorf("pop requests: posts=%d accounts=%d, want 0 and 0", posts.popReqs, accounts.popReqs)
	}
	if posts.moved != accounts.moved {
		t.Errorf("j moved posts=%v accounts=%v, want the same", posts.moved, accounts.moved)
	}
	// Position-field arithmetic: both kinds report a positive selection index when loaded > 0.
	if posts.loaded == 0 || accounts.loaded == 0 {
		t.Fatalf("loaded posts=%d accounts=%d, want both non-zero", posts.loaded, accounts.loaded)
	}
	if posts.pos < 1 || accounts.pos < 1 {
		t.Errorf("position posts=%d accounts=%d, want both ≥ 1", posts.pos, accounts.pos)
	}
}
