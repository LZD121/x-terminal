package render

import (
	"testing"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// This file is the conversation's stored frames: the UI-SPEC's thread states at four widths,
// the partial / visibility / empty / chain-cut / narrow frames, and the rung matrix. They join
// Phase 2's corpus through corpus() so TestGoldenFrames and corpus_test.go audit them the same
// way they audit the timeline frames.

// paneThread is the conversation's pane: ThreadBlocks over the state's own Conversation, scrolled
// the way a list pane is scrolled.
const paneThread paneKind = 100

// threadStatus is the conversation's status line: list word thread, the position over the posts
// the frame holds, and the budget. HasMore is the reply walk's cursor; Notice is the conversation's
// own when the state carries one.
func threadStatus(s state) StatusFields {
	posts := threadPosts(s.conversation)
	selected := threadSelectedIndex(posts, s.threadSel)
	notice := s.threadNotice
	if notice == NoticeNone && s.conversation.Stalled {
		notice = NoticeStalled
	}
	return StatusFields{
		Failure:  s.failure,
		Notice:   notice,
		List:     ThreadListName(s.cover),
		Selected: selected,
		Loaded:   len(posts),
		HasMore:  s.threadHasMore,
		Used:     s.used,
		Limited:  s.limited,
		Now:      corpusNow,
		Cover:    s.cover,
		Gated:    s.grammar().Markers.Gated,
	}
}

// threadSelectedIndex is the 1-based index of the block the selection is inside: a quoted post
// does not move the number — the number answers which post the reader is in.
func threadSelectedIndex(posts []domain.Post, sel Selection) int {
	if sel.ID == "" {
		return 1
	}
	if len(sel.Path) > 0 {
		outer := sel.Path[0]
		for i, p := range posts {
			if p.ID == outer {
				return i + 1
			}
		}
	}
	for i, p := range posts {
		if p.ID == sel.ID {
			return i + 1
		}
		if p.Quote != nil && (p.Quote.ID == sel.ID || quoteHolds(p.Quote, sel.ID)) {
			return i + 1
		}
	}
	return 1
}

func quoteHolds(q *domain.Post, id domain.PostID) bool {
	if q == nil {
		return false
	}
	if q.ID == id {
		return true
	}
	return quoteHolds(q.Quote, id)
}

// threadPosts is the conversation's posts in frame order: ancestors, focal, continuations, replies.
func threadPosts(c Conversation) []domain.Post {
	out := make([]domain.Post, 0, len(c.Ancestors)+1+len(c.Continued)+len(c.Replies))
	out = append(out, c.Ancestors...)
	out = append(out, c.Focal)
	out = append(out, c.Continued...)
	out = append(out, c.Replies...)
	return out
}

// threadFrames is every stored conversation frame the UI-SPEC names.
func threadFrames(t *testing.T) []frame {
	t.Helper()

	base := threadSpecConversation()
	// Engagement numbers on every post so the count absence is a check rather than a tautology.
	counts := engagement(t)
	base.Ancestors[0].Counts = counts
	base.Focal.Counts = counts
	for i := range base.Continued {
		base.Continued[i].Counts = counts
	}
	for i := range base.Replies {
		base.Replies[i].Counts = counts
	}

	spec := state{
		pane:          paneThread,
		conversation:  base,
		threadSel:     Selection{ID: base.Focal.ID},
		threadHasMore: true,
		used:          12,
		limited:       0,
	}

	partialConv := Conversation{
		Focal: domain.Post{
			ID: "f1", Author: domain.User{ID: "u-aergon", ScreenName: "aergon"},
			CreatedAt: time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC), ConversationID: "c",
			Text:   "cursor in, cursor out: the render layer never does arithmetic on a cursor.",
			Counts: counts,
		},
		Replies: []domain.Post{
			{
				ID: "r1", Author: domain.User{ID: "u-mira", ScreenName: "mira"},
				CreatedAt: time.Date(2026, 10, 14, 9, 20, 0, 0, time.UTC), ConversationID: "c", ReplyTo: "f1",
				Text:   "the walk stopped for me too, two pages in.",
				Counts: counts,
			},
			{
				ID: "r2", Author: domain.User{ID: "u-kestrel", ScreenName: "kestrel"},
				CreatedAt: time.Date(2026, 10, 14, 9, 24, 0, 0, time.UTC), ConversationID: "c", ReplyTo: "f1",
				Text:   "that is the reply ranking, not the walk.",
				Counts: counts,
			},
		},
		Read:    true,
		Stalled: true,
	}
	partial := state{
		pane:          paneThread,
		conversation:  partialConv,
		threadSel:     Selection{ID: "r1"},
		threadHasMore: false,
		used:          12,
		limited:       1,
	}

	visConv := Conversation{
		Focal: domain.Post{
			ID: "f1", Author: domain.User{ID: "u-aergon", ScreenName: "aergon"},
			CreatedAt: time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC), ConversationID: "c",
			Text: "cursor in, cursor out: the render layer never does arithmetic on a cursor.", Counts: counts,
		},
		Replies: []domain.Post{
			{
				ID: "r1", Author: domain.User{ID: "u-mira", ScreenName: "mira"},
				CreatedAt: time.Date(2026, 10, 14, 9, 20, 0, 0, time.UTC), ConversationID: "c", ReplyTo: "f1",
				Visibility: domain.Limited,
				Text:       "this one is held back, and the upstream still hands me the text.",
				Counts:     counts,
			},
			{ID: "r2", Visibility: domain.Unavailable, Counts: counts},
			{ID: "r3", Visibility: domain.Withheld, Text: "a body nobody may read", Counts: counts},
		},
		Read: true,
	}
	visibility := state{
		pane: paneThread, conversation: visConv, threadSel: Selection{ID: "r1"},
		used: 12, limited: 0,
	}

	emptyConv := Conversation{
		Focal: domain.Post{
			ID: "f1", Author: domain.User{ID: "u-aergon", ScreenName: "aergon"},
			CreatedAt: time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC), ConversationID: "c",
			Text: "cursor in, cursor out: the render layer never does arithmetic on a cursor.", Counts: counts,
		},
		Read: true, NoReplies: true,
	}
	empty := state{
		pane: paneThread, conversation: emptyConv, threadSel: Selection{ID: "f1"},
		used: 12, limited: 0,
	}

	chainConv := Conversation{
		Focal: domain.Post{
			ID: "f1", Author: domain.User{ID: "u-aergon", ScreenName: "aergon"},
			CreatedAt: time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC), ConversationID: "c", ReplyTo: "missing",
			Text: "cursor in, cursor out: the render layer never does arithmetic on a cursor.", Counts: counts,
		},
		Replies: []domain.Post{{
			ID: "r1", Author: domain.User{ID: "u-mira", ScreenName: "mira"},
			CreatedAt: time.Date(2026, 10, 14, 9, 20, 0, 0, time.UTC), ConversationID: "c", ReplyTo: "f1",
			Text: "a reply under a cut chain.", Counts: counts,
		}},
		Read: true, PartialAbove: true,
	}
	chaincut := state{
		pane: paneThread, conversation: chainConv, threadSel: Selection{ID: "f1"},
		used: 12, limited: 0,
	}

	// Adversarial conversation covering the checklist the corpus audit reads: deep ancestors with a
	// tombstone, a depth-2 quote, a tombstone focal, withheld, limited reply, reply-to, no-marker
	// reply, reply-of-reply indent flat, 15-cell handle + CJK + emoji at the reply step.
	adversarial := threadAdversarialConversation(counts)
	adv := state{
		pane: paneThread, conversation: adversarial,
		threadSel: Selection{ID: adversarial.Focal.ID}, threadHasMore: true,
		used: 12, limited: 0, failure: domain.Failure{
			Kind: domain.FailureRateLimited, Reset: corpusNow.Add(5*time.Hour + 20*time.Minute),
		},
	}
	// Scroll each selection frame so its selected block is on screen: the adversarial conversation
	// is taller than the pane, and a frame that opens at the top would only show the chain above.
	adv.threadAt = adversarial.Focal.ID
	ancestorSel := adv
	ancestorSel.threadSel = Selection{ID: adversarial.Ancestors[0].ID}
	ancestorSel.threadAt = adversarial.Ancestors[0].ID
	replySel := adv
	replySel.threadSel = Selection{ID: adversarial.Replies[0].ID}
	replySel.threadAt = adversarial.Replies[0].ID
	quoteSel := adv
	quotePost := adversarial.Replies[len(adversarial.Replies)-1]
	quoteSel.threadSel = Selection{
		ID:   quotePost.Quote.ID,
		Path: []domain.PostID{quotePost.ID},
	}
	quoteSel.threadAt = quotePost.ID
	// pushed opens at the top of a conversation whose ancestry alone fills the first pane, so the
	// focal post is off screen until the reader scrolls.
	pushed := adv
	pushed.threadAt = ""
	pushed.scroll = 0

	narrow := spec
	narrow.failure = domain.Failure{
		Kind: domain.FailureRateLimited, Reset: corpusNow.Add(5*time.Hour + 20*time.Minute),
	}
	narrow.used, narrow.limited = 200, 12

	plain := []frame{
		{name: "thread_60", w: 60, env: envPlain, state: spec},
		{name: "thread_80", w: 80, env: envPlain, state: spec},
		{name: "thread_120", w: 120, env: envPlain, state: spec},
		{name: "thread_200", w: 200, env: envPlain, state: spec},
		{name: "thread_partial_80", w: 80, env: envPlain, state: partial},
		{name: "thread_partial_120", w: 120, env: envPlain, state: partial},
		{name: "thread_visibility_80", w: 80, env: envPlain, state: visibility},
		{name: "thread_empty_80", w: 80, env: envPlain, state: empty},
		{name: "thread_chaincut_80", w: 80, env: envPlain, state: chaincut},
		{name: "thread_narrow_79", w: MinPaneCols - 1, env: envPlain, state: narrow},
		{name: "thread_color_truecolor", w: 80, env: envTrueColor, state: spec},
		{name: "thread_color_256", w: 80, env: env256, state: spec},
		{name: "thread_color_16", w: 80, env: env16, state: spec},
		{name: "thread_color_mono", w: 80, env: envMono, state: spec},
		// Adversarial content frames the corpus audit names.
		{name: "thread_adversarial_80", w: 80, env: envPlain, state: adv},
		{name: "thread_sel_ancestor_80", w: 80, env: envPlain, state: ancestorSel},
		{name: "thread_sel_reply_80", w: 80, env: envPlain, state: replySel},
		{name: "thread_sel_quote_80", w: 80, env: envPlain, state: quoteSel},
		{name: "thread_push_80", w: 80, env: envPlain, state: pushed},
		{name: "thread_tombstone_focal_80", w: 80, env: envPlain, state: tombstoneFocalState(counts)},
		{name: "thread_status_plus_80", w: 80, env: envPlain, state: spec},
		{name: "thread_status_noplus_80", w: 80, env: envPlain, state: func() state {
			s := spec
			s.threadHasMore = false
			return s
		}()},
		{name: "thread_status_failure_80", w: 80, env: envPlain, state: func() state {
			s := spec
			s.failure = domain.Failure{
				Kind: domain.FailureRateLimited, Reset: corpusNow.Add(5*time.Hour + 20*time.Minute),
			}
			s.used, s.limited = 200, 12
			return s
		}()},
	}
	return plain
}

func tombstoneFocalState(counts domain.Counts) state {
	c := Conversation{
		Focal: domain.Post{ID: "tomb", Visibility: domain.Unavailable, Counts: counts},
		Replies: []domain.Post{{
			ID: "r1", Author: domain.User{ID: "u-mira", ScreenName: "mira"},
			CreatedAt: time.Date(2026, 10, 14, 9, 20, 0, 0, time.UTC), ConversationID: "c", ReplyTo: "tomb",
			Text: "a reply under a tombstone focal.", Counts: counts,
		}},
		Read: true,
	}
	return state{pane: paneThread, conversation: c, threadSel: Selection{ID: "tomb"}, used: 12}
}

func threadAdversarialConversation(counts domain.Counts) Conversation {
	lzd, aergon, mira, kestrel, _ := threadUsers()
	wide := domain.User{ID: "u-wide", ScreenName: "a_15_cell_handl", Name: "Fifteen Cell"}
	root := domain.Post{
		ID: "anc-root", Author: lzd, CreatedAt: time.Date(2026, 10, 14, 7, 0, 0, 0, time.UTC),
		ConversationID: "c", Text: "the root of a deep chain above the focal post. the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - ", Counts: counts,
	}
	mid := domain.Post{
		ID: "anc-mid", Author: aergon, CreatedAt: time.Date(2026, 10, 14, 7, 30, 0, 0, time.UTC),
		ConversationID: "c", ReplyTo: "anc-root", Text: "the middle of the chain. the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - ", Counts: counts,
	}
	mid2 := domain.Post{
		ID: "anc-mid2", Author: kestrel, CreatedAt: time.Date(2026, 10, 14, 7, 45, 0, 0, time.UTC),
		ConversationID: "c", ReplyTo: "anc-mid", Text: "another ancestor in the deep chain. the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - ", Counts: counts,
	}
	tomb := domain.Post{ID: "anc-tomb", Visibility: domain.Unavailable, Counts: counts}
	parent := domain.Post{
		ID: "anc-parent", Author: mira, CreatedAt: time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC),
		ConversationID: "c", ReplyTo: "anc-tomb", Text: "the parent just above the focal post. the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - the ancestry above the focal post is long enough that the first pane cannot hold the focal block when the frame opens at the top of the conversation - ", Counts: counts,
	}
	focal := domain.Post{
		ID: "adv-focal", Author: aergon, CreatedAt: time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC),
		ConversationID: "c", ReplyTo: "anc-parent",
		Text: "the adversarial focal post with a depth-2 quote.", Counts: counts,
		Quote: &domain.Post{
			ID: "q1", Author: lzd, CreatedAt: time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC),
			ConversationID: "c", Text: "depth-1 quote body.",
			Quote: &domain.Post{
				ID: "q2", Author: mira, CreatedAt: time.Date(2026, 10, 14, 8, 50, 0, 0, time.UTC),
				ConversationID: "c", Text: "depth-2 quote body.",
			},
		},
	}
	replies := []domain.Post{
		{ // no-marker reply to focal
			ID: "r-mira", Author: mira, CreatedAt: time.Date(2026, 10, 14, 9, 20, 0, 0, time.UTC),
			ConversationID: "c", ReplyTo: "adv-focal",
			Text: "does that hold when the walk re-runs? 日本語のテキストは2セル幅です。 🚀", Counts: counts,
		},
		{ // reply-to ancestor
			ID: "r-kestrel", Author: kestrel, CreatedAt: time.Date(2026, 10, 14, 9, 24, 0, 0, time.UTC),
			ConversationID: "c", ReplyTo: "anc-root",
			Text: "then the page cache is doing more work than the pane admits.", Counts: counts,
		},
		{ // reply of reply of reply — flat indent
			ID: "r-flat", Author: lzd, CreatedAt: time.Date(2026, 10, 14, 9, 25, 0, 0, time.UTC),
			ConversationID: "c", ReplyTo: "r-kestrel",
			Text: "a reply of a reply of a reply stays at one reply step.", Counts: counts,
		},
		{ // limited
			ID: "r-limited", Author: mira, CreatedAt: time.Date(2026, 10, 14, 9, 26, 0, 0, time.UTC),
			ConversationID: "c", ReplyTo: "adv-focal", Visibility: domain.Limited,
			Text: "this one is held back, and the upstream still hands me the text.", Counts: counts,
		},
		{ID: "r-tomb", Visibility: domain.Unavailable, Counts: counts},
		{ID: "r-withheld", Visibility: domain.Withheld, Text: "secret", Counts: counts},
		{ // 15-cell handle + quote inside reply
			ID: "r-wide", Author: wide, CreatedAt: time.Date(2026, 10, 14, 9, 31, 0, 0, time.UTC),
			ConversationID: "c", ReplyTo: "r-flat",
			Text: "that is the part i keep getting wrong, so:", Counts: counts,
			Quote: &domain.Post{
				ID: "adv-focal", Author: aergon, CreatedAt: time.Date(2026, 10, 14, 9, 12, 0, 0, time.UTC),
				ConversationID: "c", Text: "the adversarial focal post with a depth-2 quote.",
			},
		},
	}
	return Conversation{
		Ancestors: []domain.Post{root, mid, mid2, tomb, parent},
		Focal:     focal,
		Replies:   replies,
		Read:      true,
	}
}
