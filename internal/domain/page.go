package domain

import (
	"strconv"
	"time"
)

// Cursor is the upstream's own position marker, opaque to everything above the port: a value
// is carried, compared and handed back, and nothing here reads it, splits it, orders by it or
// builds one. The only two things the program knows about it are the two this type can say —
// a cursor is either stored or empty, and the empty one means the request was for page one
// when it goes out and end-of-list when it comes back.
type Cursor string

// HomeKind is which of the two home timelines a page came from. Following is the zero value
// because it is the surface this phase opens on.
type HomeKind uint8

const (
	// Following is the timeline of the accounts the user follows (the upstream's
	// HomeLatestTimeline).
	Following HomeKind = iota
	// ForYou is the recommended timeline (the upstream's HomeTimeline).
	ForYou
)

// String names the list in the words the status line's `list` field prints.
func (k HomeKind) String() string {
	switch k {
	case Following:
		return "following"
	case ForYou:
		return "for you"
	}
	return "home kind " + strconv.Itoa(int(k))
}

// PostPage is one page of a timeline: the posts it carried, where the walk continues, and what
// the answer cost.
//
// It is the port's whole read result, and it deliberately cannot express anything about the
// wire: no status, no body, no operation name, no headers.
type PostPage struct {
	// Posts are the posts the answer carried, in the answer's order. That order is the list's
	// order and no caller may re-sort it.
	Posts []Post
	// Next is where the walk continues, empty at end-of-list. An empty Next is a fact about
	// the timeline (there is nothing older), never a failure to read a cursor.
	Next Cursor
	// Requested is the count of entries the page carried, as the reader counted them: internal/xsource
	// counts the answer's tweet entries (the only entries that can become posts) and internal/store
	// counts the rows one mirror page carried. It is what the page cost, not what it contributed: a
	// page whose Requested is 20 and which adds zero unseen posts is a different fact from an empty
	// answer, and a caller that compared the page against its own growth could not tell the two
	// apart.
	//
	// Nothing in this build's read path reads it, and it is carried with that said out loud
	// rather than left to look read: the walk's stall rule is on the rows it appended
	// (internal/screen/walk.go) and the mirror walk's is on the ids its upsert added
	// (internal/store/sync.go), and both need the rows themselves rather than their count. What
	// does read it is the adapter's own tests, against the recorded responses
	// (internal/xsource), where the claim is that the parser counted every entry the answer
	// carried — a measurement only the adapter that read the answer can make, which is why the
	// field is on the port instead of being recomputed by each caller from the rows it kept.
	Requested int
	// FetchedAt is when the answer arrived, from the caller's clock (the adapter takes a clock
	// as an argument; nothing in this package reads one).
	FetchedAt time.Time
}
