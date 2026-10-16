package screen

import "github.com/LZD121/x-terminal/internal/domain"

// postKey is the home list's row key: a post's own id, as the string the walk keys its rows by.
func postKey(p domain.Post) string { return string(p.ID) }

// List is one timeline on screen: the posts in list order, where the walk has got to, which row is
// selected, and the state of the requests around it.
//
// The list machinery is the shared walk (walk.go) and nothing here repeats any of it: this file is
// the frame's own policy — which timeline it shows, which events it accepts, and what each of them
// means here. The de-duplication, the stall rule, the single-flight latch, the request gate and the
// refresh diff are implemented once, there.
//
// The zero value is a usable empty Following list — no posts, no cursor, nothing selected, no
// request in flight, no failure — which is what the shell starts with before its first fetch.
//
// Every field is unexported, and the accessors below are read-only, so the shell cannot put a list
// into a state the reducers do not describe.
type List struct {
	// Kind is which home timeline this list shows. It is exported because the shell chooses it when
	// it builds a frame, and it travels in every request the frame builds.
	Kind domain.HomeKind

	// walk is the list's state: the rows, the cursor, the latches, the failure, the selection and the
	// anchor. It is the same walk the thread frame rides.
	walk Walk[domain.Post, domain.PostPage]
}

// w is this frame's walk with the frame's own row key in place. A frame's zero value is a usable
// state, so the key function is filled in at the seam between the frame and the walk rather than
// required of whoever built the frame.
func (l List) w() Walk[domain.Post, domain.PostPage] { return l.walk.keys(postKey) }

// Posts returns the list's posts in list order. The slice is the list's own: a caller reads it and
// does not modify it. (No copy: the shell asks for this on every frame, and copying the list per
// frame would be an allocation in the one path that must not have one.)
func (l List) Posts() []domain.Post { return l.w().Rows() }

// Targets is the frame's row order, which is what the cursor moves through: each post in list order
// and then that post's quote chain, so a quoted post is one step from the post that quotes it and is
// a row the shell can open like any other.
func (l List) Targets() []Target { return postTargets(l.Posts()) }

// Selected is the row the cursor is on, empty when nothing is selected.
func (l List) Selected() Target { return l.w().Selected() }

// Post is the copy of the post the frame holds for a target, quote chain included. It is what the
// shell opens a frame from, which is why opening a post — or a post quoted inside one — costs no
// request: the copy is already here.
func (l List) Post(t Target) (domain.Post, bool) { return postFor(l.Posts(), t) }

// Move walks the frame's own row order by a step: one row for `j`/`k`, whatever the shell computes for
// a half or a whole pane, clamped at both ends so the selection never leaves the sequence.
func (l List) Move(step int) List {
	l.walk = l.w().Move(l.Targets(), step)
	return l
}

// Anchor is the selected block's head row's distance from the pane's first row. The arithmetic that
// computes it is the shell's, because the pane geometry is.
func (l List) Anchor() int { return l.w().Anchor() }

// WithAnchor returns the frame with the anchor the shell worked out.
func (l List) WithAnchor(anchor int) List {
	l.walk = l.w().WithAnchor(anchor)
	return l
}

// Next is the cursor the walk continues from, empty at end-of-list. It is carried in the status
// line's position field as the `+` that says there is more.
func (l List) Next() domain.Cursor { return l.w().Next() }

// Inflight reports whether a request is out. No update issues another while it is true.
func (l List) Inflight() bool { return l.w().InFlight() }

// Failure is the last failure, FailureNone when there is none. It is sticky: only a successful
// fetch clears it, and nothing here runs on a timer.
func (l List) Failure() domain.Failure { return l.w().Failure() }

// NewPosts is how many posts the last refresh brought that the list did not already hold. Zero is
// the documented "no new posts" notice, which is a different thing from no refresh having
// happened at all.
func (l List) NewPosts() int { return l.w().NewRows() }

// LoadingOlder reports whether the request in flight is a page of the walk.
func (l List) LoadingOlder() bool { return l.w().LoadingOlder() }

// Stalled reports whether the walk has ended without reaching the end of the timeline: a cursor
// came back a second time, or a page contributed nothing. It is a flag and not a countdown: the
// list never retries the same cursor, and nothing here waits for a timer to try.
func (l List) Stalled() bool { return l.w().Stalled() }

// Loaded is how many posts the list holds.
func (l List) Loaded() int { return l.w().Loaded() }

// HasMore reports whether the walk may continue: a cursor is stored and the walk has not stalled.
func (l List) HasMore() bool { return l.w().HasMore() }

// Index is where a post sits in the list, or -1 when the list does not hold it. It is a post of the
// list that is looked for, not a row of it: a quoted post's own row is found through Target.Top.
func (l List) Index(id string) int { return l.w().Index(id) }

// AtEnd reports whether the selection's block is the last loaded post and the walk has somewhere to
// go: the state in which the shell asks for the next page. It is a question about the list, not about
// the pane, which is why no pixel geometry is involved.
func (l List) AtEnd() bool { return l.w().AtEnd(l.w().Selected()) }
