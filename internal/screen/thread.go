package screen

import (
	"slices"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Thread is one conversation on screen: the post the frame was opened on, the chain it answers, the
// post's own continuations, and one page of the replies under it.
//
// It rides the same walk as the home list — one Walk of posts, with this frame's own flags beside it —
// so the pagination trigger, the stall rule, the single-flight latch, the request gate and the refresh
// diff are that one implementation rather than a second one for conversations.
//
// Opening costs no request: the frame starts from the copy of the post the caller already holds (a row
// of the list, or a post's nested quote), which is why nothing here can fail until the first page
// arrives, and why a conversation opened on a post the reader is looking at is on screen in the frame
// the key was pressed in.
type Thread struct {
	// focal is the post the conversation is of. It opens as the copy the caller held, and page one's own
	// copy replaces it: a later page that answered without it keeps the one the frame holds, because an
	// empty Focal.ID means "this page did not carry it" and never "the post is gone".
	focal domain.Post
	// ancestors is the chain above the focal post, root first, in the answer's own order, and it may be
	// empty. It is not a walk: these posts arrived with page one and nothing pages them.
	ancestors []domain.Post
	// continued is the focal post's own continuations, in the answer's own group order.
	continued []domain.Post
	// walk is the reply stream: its rows are the replies and nothing else, so the last loaded reply is the
	// end of the only walk this frame has, and a row above it is never a trigger.
	walk Walk[domain.Post, domain.ThreadPage]
	// read records that an answer has come back at all: the difference between the frame that holds only
	// the copy it was opened with and one that has read the conversation. It is what the empty-reply
	// statement and the third partial-thread condition both turn on.
	read bool
}

// NewThread opens a conversation on the copy of the post the caller already holds. It costs no request
// and cannot fail, and the selection starts on the focal post — so the rows the reader opened it for are
// on screen at once, with the first page asked for from the frame's own trigger.
func NewThread(focal domain.Post) Thread {
	walk := NewWalk[domain.Post, domain.ThreadPage](postKey)
	// The focal post is a row of this frame and not a row of the reply walk, so it is placed rather than
	// selected through the walk's own rows: the frame's rows are the four lists below, and every one of
	// them is selectable.
	walk.selected = Target{ID: string(focal.ID)}
	return Thread{focal: focal, walk: walk}
}

// ThreadRequested asks for one page of the conversation: the first page while the frame holds only the
// copy it was opened with, and then the walk's next page of replies.
type ThreadRequested struct{}

// ThreadRefreshRequested asks for page one of the conversation again: `r` in a thread frame.
type ThreadRefreshRequested struct{}

// ThreadFetched is one page of a conversation arriving: page one, or a later page of the reply walk.
type ThreadFetched struct{ Page domain.ThreadPage }

// ThreadFailed is a conversation's request that did not deliver data. It is the failure the status line
// prints, already named by the classifier in internal/cli.
type ThreadFailed struct{ Failure domain.Failure }

// ThreadRefreshed is the answer to `r` in a conversation: the reply list re-read.
type ThreadRefreshed struct{ Page domain.ThreadPage }

func (ThreadRequested) event()        {}
func (ThreadRefreshRequested) event() {}
func (ThreadFetched) event()          {}
func (ThreadFailed) event()           {}
func (ThreadRefreshed) event()        {}

// Update applies one event and returns the next state with the single effect it asks for.
//
// It is the home frame's shape exactly: now is the caller's clock and only the request looks at it, an
// event this frame does not know (including a nil Event, and including the home list's own events) is a
// no-op, and the only effects this frame can produce are a page and a refresh of the conversation.
func (t Thread) Update(ev Event, now time.Time) (Thread, Effect) {
	t.walk = t.walk.keys(postKey)
	switch e := ev.(type) {
	case ThreadRequested:
		return t.requested(now)
	case ThreadRefreshRequested:
		return t.refreshRequested(now)
	case ThreadFetched:
		return t.fetched(e.Page), Effect{}
	case ThreadFailed:
		return t.failed(e.Failure), Effect{}
	case ThreadRefreshed:
		return t.refreshed(e.Page), Effect{}
	}
	return t, Effect{}
}

// requested is the reply walk's trigger.
//
// The first page is asked for while nothing has been read: the frame holds a copy of the post and no
// reply of it, and page one is what turns the copy into the conversation. After that, a page is asked for
// only when the selection has reached the last loaded reply — reaching a row above the walk (an ancestor,
// the focal post, one of its continuations) is not a trigger at all, because those posts came with page
// one and are not a walk, and a frame that asked from them would be spending a request the reader did not
// ask for.
//
// Everything else the walk itself refuses: a request already out, a walk that has ended on a cursor the
// upstream would not advance, and a failure state that forbids a request at this time.
func (t Thread) requested(now time.Time) (Thread, Effect) {
	if t.read && !t.walk.AtEnd(t.walk.Selected()) {
		return t, Effect{}
	}
	next, effect := t.walk.Page(ThreadRequest{Focal: t.focal.ID, Cursor: t.walk.Next()}, now)
	t.walk = next
	return t, effect
}

// refreshRequested is `r` in a conversation: page one again, through the walk's own refresh latch.
func (t Thread) refreshRequested(now time.Time) (Thread, Effect) {
	next, effect := t.walk.Refresh(ThreadRequest{Focal: t.focal.ID}, now)
	t.walk = next
	return t, effect
}

// fetched folds one page of the conversation into the frame.
//
// The focal post is replaced when the answer carried one, the chain above is replaced when the answer
// carried ancestors and kept when it carried none — a later page is not evidence that the chain was
// erased — and the answer's self-thread and reply groups become the continuations and the reply walk in
// the answer's own order. The reply walk absorbs the page's rows under the walk's own rules, so a reply
// whose row the frame already holds is not added twice and a page that adds nothing unseen ends the walk.
func (t Thread) fetched(page domain.ThreadPage) Thread {
	held := t.walk.Selected()
	t = t.fold(page)
	t.walk = t.walk.absorb(page, replyPosts(page), page.Next)
	t.read = true
	// Page one may replace the focal copy the frame opened with. Hold the selection by target
	// across the fold, then fall back to the focal post so the cursor cannot stay on an id the
	// frame no longer holds (the timeline copy's id).
	if next, ok := t.walk.SelectAt(t.Targets(), held); ok {
		t.walk = next
	} else if next, ok := t.walk.SelectAt(t.Targets(), Target{ID: string(t.focal.ID)}); ok {
		t.walk = next
	}
	return t
}

// failed records a failure and changes nothing else: every row stays exactly as it was — including the
// copy of the focal post the frame was opened with — and the failure is what the status line prints. A
// conversation whose first page failed is this same state, not an empty conversation.
func (t Thread) failed(f domain.Failure) Thread {
	t.walk = t.walk.pageFailed(f)
	return t
}

// refreshed re-reads the conversation.
//
// The answer's replies replace the frame's: this list is a ranking, and re-reading it is the only honest
// way to make it current — a reply the client cannot place in that ranking is a reply it cannot merge
// into it. The ancestors and the focal post follow the fold above, the count of reply ids the frame did
// not hold before is what the notice line reports, and the selection is held by target and then by id and
// otherwise falls back to the focal post: the one row this frame always has.
func (t Thread) refreshed(page domain.ThreadPage) Thread {
	held := t.walk.Selected()
	t = t.fold(page)
	t.walk = t.walk.refresh(replyPosts(page), page.Next, RefreshReplace)
	t.walk = t.walk.SelectNear(held)
	if !t.holds(t.walk.Selected()) {
		if next, ok := t.walk.SelectAt(t.Targets(), Target{ID: string(t.focal.ID)}); ok {
			t.walk = next
		}
	}
	t.read = true
	return t
}

// fold is the part of an answer that a page and a refresh share: the focal post, the chain above it and
// the focal post's own continuations, each replaced only by an answer that carried it.
func (t Thread) fold(page domain.ThreadPage) Thread {
	if page.Focal.ID != "" {
		t.focal = page.Focal
	}
	if len(page.Ancestors) > 0 {
		t.ancestors = slices.Clone(page.Ancestors)
	}
	if continued, ok := continuations(page); ok {
		t.continued = continued
	}
	return t
}

// continuations is the posts of an answer's self-thread groups, in the answer's own group order, and
// whether the answer carried any such group at all: an answer that carried none is not evidence that the
// focal post stopped continuing itself.
func continuations(page domain.ThreadPage) ([]domain.Post, bool) {
	var out []domain.Post
	found := false
	for _, group := range page.Groups {
		if group.Kind != domain.SelfThread {
			continue
		}
		found = true
		out = append(out, group.Posts...)
	}
	return out, found
}

// replyPosts is the posts of an answer's reply groups, in the answer's own order: the rows of the reply
// walk, and the only rows a page of a conversation contributes to it.
func replyPosts(page domain.ThreadPage) []domain.Post {
	var out []domain.Post
	for _, group := range page.Groups {
		if group.Kind != domain.Replies {
			continue
		}
		out = append(out, group.Posts...)
	}
	return out
}

// Focal is the post the conversation is of: the copy the frame holds, which is the caller's copy until
// page one replaces it.
func (t Thread) Focal() domain.Post { return t.focal }

// Ancestors is the chain above the focal post, root first, in the answer's own order.
func (t Thread) Ancestors() []domain.Post { return t.ancestors }

// Continued is the focal post's own continuations, in the answer's own group order.
func (t Thread) Continued() []domain.Post { return t.continued }

// Replies is the replies the frame holds, in the answer's own order. The slice is the frame's own: a
// caller reads it and does not modify it.
func (t Thread) Replies() []domain.Post { return t.walk.Rows() }

// Targets is the frame's row order, which is the order the answer carried: the chain above root first,
// then the focal post, then the focal post's own continuations, then the replies — each block followed by
// its quote chain, so a quoted post is a row the cursor can land on and open. Nothing is re-sorted and no
// tree is derived: a reply whose parent is not in the frame keeps its place.
func (t Thread) Targets() []Target {
	out := make([]Target, 0, t.walk.Loaded()+len(t.ancestors)+1+len(t.continued))
	out = append(out, postTargets(t.ancestors)...)
	out = append(out, postTargets([]domain.Post{t.focal})...)
	out = append(out, postTargets(t.continued)...)
	return append(out, postTargets(t.walk.Rows())...)
}

// blocks is the frame's top-level posts in row order: what the status line counts, and what a target's
// Top names.
func (t Thread) blocks() []domain.Post {
	out := make([]domain.Post, 0, t.walk.Loaded()+len(t.ancestors)+1+len(t.continued))
	out = append(out, t.ancestors...)
	out = append(out, t.focal)
	out = append(out, t.continued...)
	return append(out, t.walk.Rows()...)
}

// holds reports whether a target is one of this frame's rows.
func (t Thread) holds(tgt Target) bool { return indexOf(t.Targets(), tgt) >= 0 }

// Post is the copy of the post the frame holds for a target, quote chain included — the chain above, the
// focal post, its continuations and the replies are all searched. It is what the shell opens a further
// frame from, which is why opening a quoted post costs no request: the copy is already here.
func (t Thread) Post(tgt Target) (domain.Post, bool) {
	lists := [4][]domain.Post{t.ancestors, {t.focal}, t.continued, t.walk.Rows()}
	for _, posts := range lists {
		if post, ok := postFor(posts, tgt); ok {
			return post, true
		}
	}
	return domain.Post{}, false
}

// Selected is the row the cursor is on, empty when nothing is selected.
func (t Thread) Selected() Target { return t.walk.Selected() }

// Select moves the selection onto one of the frame's own rows, and a target that is not one of them moves
// nothing.
func (t Thread) Select(tgt Target) Thread {
	next, ok := t.walk.SelectAt(t.Targets(), tgt)
	if !ok {
		return t
	}
	t.walk = next
	return t
}

// Move walks the frame's own row order by a step, clamped at both ends: from the focal post into the
// quote it carries is one step, and from the last reply there is nowhere further to go.
func (t Thread) Move(step int) Thread {
	t.walk = t.walk.Move(t.Targets(), step)
	return t
}

// Anchor is the selected block's head row's distance from the pane's first row.
func (t Thread) Anchor() int { return t.walk.Anchor() }

// WithAnchor returns the frame with the anchor the shell worked out.
func (t Thread) WithAnchor(anchor int) Thread {
	t.walk = t.walk.WithAnchor(anchor)
	return t
}

// AtEnd reports whether the selection is on the last loaded reply: the pagination trigger.
func (t Thread) AtEnd() bool { return t.walk.AtEnd(t.walk.Selected()) }

// Read reports whether an answer has come back at all. A frame that has not read holds only the copy it
// was opened with, which is what the pane's loading line is for.
func (t Thread) Read() bool { return t.read }

// Loading reports whether the first page is out and no answer has arrived yet: the state the pane says
// `loading thread...` in.
func (t Thread) Loading() bool { return !t.read && t.walk.InFlight() }

// LoadingOlder reports whether the request in flight is a page of the reply walk, which is the state the
// status line says `loading replies` in.
func (t Thread) LoadingOlder() bool { return t.walk.LoadingOlder() }

// HasMore reports whether the reply walk may continue: a cursor is stored and the walk has not ended.
func (t Thread) HasMore() bool { return t.walk.HasMore() }

// Stalled reports whether the reply walk ended because the upstream would not advance the cursor: the
// first of the three partial-thread conditions.
func (t Thread) Stalled() bool { return t.walk.Stalled() }

// NewReplies is how many replies the last refresh brought that the frame did not already hold.
func (t Thread) NewReplies() int { return t.walk.NewRows() }

// Failure is the last failure, FailureNone when there is none. It is sticky, exactly as the home list's
// is: a conversation has no vocabulary for failure of its own.
func (t Thread) Failure() domain.Failure { return t.walk.Failure() }

// Position is where the reader is in the conversation, one-based: selected counts the block the selection
// is inside and total counts the posts the frame holds — the chain above, the focal post, its
// continuations and the replies read so far. Stepping into a quoted row does not change the number: the
// number answers "which post am I in", not "which row".
//
// selected is 0 when the selection is not one of the frame's posts, which is the state the pane has no
// position for.
func (t Thread) Position() (selected, total int) {
	blocks := t.blocks()
	top := t.walk.Selected().Top().ID
	for i, post := range blocks {
		if string(post.ID) == top {
			return i + 1, len(blocks)
		}
	}
	return 0, len(blocks)
}

// PartialStopped is the first partial-thread condition: the reply walk ended on a cursor the upstream
// would not advance, so the conversation stops here. It is a question and not a statement: what the pane
// prints for it is 04-03's, and nothing here builds a sentence.
func (t Thread) PartialStopped() bool { return t.walk.Stalled() }

// PartialAbove is the second partial-thread condition: the focal post answers a post the frame does not
// hold, so the conversation above it is not reachable — the chain the frame carries has a hole at its
// root.
func (t Thread) PartialAbove() bool {
	parent := t.focal.ReplyTo
	if parent == "" {
		return false
	}
	if parent == t.focal.ID {
		return false
	}
	return !slices.ContainsFunc(t.ancestors, func(p domain.Post) bool { return p.ID == parent })
}

// PartialHidden is the third partial-thread condition: the answer carried no replies and no cursor, while
// the focal post's own reply count says there are replies. The count is read here as evidence and is
// never printed by anybody (REND-01) — this frame exposes the question, not the number.
func (t Thread) PartialHidden() bool {
	return t.read &&
		t.walk.Loaded() == 0 &&
		t.walk.Next() == "" &&
		t.focal.Counts.Replies > 0
}
