// Package screen holds the list state machine: the pure reducers that decide what a pane shows,
// what a key means, and whether a request may be issued at all.
//
// Nothing here performs anything, and nothing here is able to: there is no tea, no
// context.Context, no clock, no file, no terminal and no call to a domain.Source. An update takes
// the state and one event and returns the next state and an Effect, and the shell
// (internal/tui) is what turns an Effect into a request and feeds the answer back as an event.
//
// That split is what makes this phase's honesty rules cheap to prove: no duplicate post, no
// cursor loop, no silent loss when a page fails, no request at all after a soft ban — each one is
// a property of a function that takes values and returns values, testable with no socket, no
// sleep and no terminal.
package screen

import "github.com/LZD121/x-terminal/internal/domain"

// EffectKind is what an update asks the shell to do.
//
// The set is closed and has a zero value that means "nothing to do", because most events are
// purely local (moving the selection, absorbing an answer, a key that arrived while a request was
// already out) and the shell has to be able to tell that apart from a request.
type EffectKind uint8

const (
	// EffectNone is no request at all.
	EffectNone EffectKind = iota
	// EffectPage fetches one more page of the list, at Cursor.
	EffectPage
	// EffectRefresh fetches page one again, carrying Head — the ids of the rows already loaded —
	// so that the answer can be diffed against what the walk holds.
	EffectRefresh
)

// Effect is one request the shell has to make, as a value.
//
// An Effect is a description, not an act: this package cannot perform one. The shell turns
// EffectPage and EffectRefresh into the commands its UI toolkit understands, and the answers come
// back as the frame's own events.
//
// Req is the whole of what the shell needs in order to route the read. A frame's effect does not
// name a feed the shell has to interpret: the request says which read is wanted and with what
// arguments, so a frame kind added later adds a request case (a ThreadRequest, a UserRequest)
// rather than a field every shell has to learn.
type Effect struct {
	// Kind is which request this is, or EffectNone for no request.
	Kind EffectKind
	// Req is the read this effect asks for: a FeedRequest for a home timeline, a ThreadRequest for a
	// conversation. It is nil when Kind is EffectNone.
	Req Request
	// Cursor is the position to fetch: the next page's cursor for EffectPage, and empty — page
	// one — for EffectRefresh.
	Cursor domain.Cursor
	// Head is the ids of the rows the walk already holds, for EffectRefresh only, which is what
	// the refresh is asking the upstream to diff against. It is nil for a page request.
	//
	// The ids are the strings the walk keys its rows by — the same vocabulary Target.ID speaks —
	// and not a post id type: the walk is element-blind, and the phase after this one asks the
	// same question about accounts.
	//
	// The slice is the walk's own copy and the caller must not modify it.
	Head []string
}
