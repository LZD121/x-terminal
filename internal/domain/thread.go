package domain

import (
	"context"
	"strconv"
	"time"
)

// GroupKind is what kind of authored block a group of posts is. It is content, not presentation:
// the two kinds mean different things — one account continuing its own thought, versus the
// conversation under a post — and the renderer prints them differently without deriving anything
// from the posts themselves.
type GroupKind uint8

const (
	// SelfThread is the focal post and its own immediate replies: one authored block, continued.
	// It is the zero value because it is the narrower claim of the two.
	SelfThread GroupKind = iota
	// Replies is the conversation under a post: posts by anyone, in the answer's own ranked order.
	Replies
)

// String names the kind in one lowercase word, for logs and for tests.
func (k GroupKind) String() string {
	switch k {
	case SelfThread:
		return "self-thread"
	case Replies:
		return "replies"
	}
	return "group kind " + strconv.Itoa(int(k))
}

// Group is one authored block inside a conversation: the posts the answer carried for it, and the
// handle that asks for more of them.
type Group struct {
	// Kind is whether this block is the focal post continuing itself or the conversation under
	// it. Nothing above the port re-derives it from the posts.
	Kind GroupKind
	// Posts are the posts the answer carried for this block, in the answer's order. That order is
	// the pane's order and no caller may re-sort it.
	Posts []Post
	// More is this group's own unspent handle, empty when the block is complete. It is opaque for
	// the same reason a cursor is: nothing here reads, splits, orders by or builds one.
	More Cursor
}

// ThreadPage is one page of a conversation: the chain above the focal post, the post itself and
// one page of its replies, plus what the answer cost.
//
// One request is one page. This value deliberately cannot express anything about the wire: no
// entry, module, instruction, cursor type or ordering index of the upstream's ever reaches it.
type ThreadPage struct {
	// Focal is the post the page was asked for. A later page may carry no focal post — the
	// screen keeps the one it read from page one — so an empty Focal.ID means "this page did not
	// carry it", never "the post is gone".
	Focal Post
	// Ancestors is the chain above the focal post, root first, in the answer's own order, and it
	// may be empty. It is not a walk: these posts arrived with page one.
	Ancestors []Post
	// Groups is the reply page, in the answer's own order — the focal post's own continuations
	// and the conversation under it, in the order the answer ranked them.
	Groups []Group
	// Next is where the reply walk continues, empty at the end of the walk. An empty Next is a
	// fact about the conversation (there is nothing more), never a failure to read a cursor.
	Next Cursor
	// Requested is the count of tweet-bearing items the answer carried — the focal post, the
	// ancestors and the groups' own items, before anything was dropped. It is what the answer
	// cost, not what it contributed: a page whose Requested is 42 and which adds nothing unseen
	// is the stall case, and a caller comparing it against the pane's growth could not tell that
	// apart from an answer that carried nothing.
	Requested int
	// Skipped is the count of entries the walker classified as not content: cursors,
	// advertisements, a recommendation rail, a composer slot and whatever decoration the entry
	// stream ships next. It is what makes an entry class this build does not know visible instead
	// of invisible — a dropped decoration that left no trace would be indistinguishable from a
	// decoder that lost a reply.
	Skipped int
	// FetchedAt is when the answer arrived, from the caller's clock: the adapter stamps it from
	// the package's own clock (a test freezes that one), and nothing in this package reads a clock.
	FetchedAt time.Time
}

// Threads is Source's thread read: the conversation containing a post — the chain above it, the
// post itself and one page of its replies.
//
// An empty cursor asks for the first page and the answer's Next asks for the page after it. It is
// declared here, beside Source rather than inside it, for the reason the phase's own note records:
// four phases add four reads to this port, and an interface every phase edits is an interface
// every phase collides in. The adapter satisfies both, and a caller asks for the one it needs.
type Threads interface {
	Thread(ctx context.Context, id PostID, c Cursor) (ThreadPage, error)
}
