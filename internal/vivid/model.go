// Package vivid is the second presentation: the same read-only data laid out as a navigation
// column, a list and a reading pane, with a status bar under them.
//
// It is pure. It borrows the block grammar from internal/render — the rows a post becomes are the
// renderer's, not a second layout of the same post — and it composes those rows into columns. It
// imports no tea, no I/O, no clock: a frame is a function of the value it is given, which is what
// makes every screen here testable as bytes.
//
// It is not a cover and it is not the plain reader. A cover is a disguise of the plain reader's
// single list; this is a different surface, worn where no disguise is needed, and its own rules —
// density, columns, images — are the presentation's, not a style's.
package vivid

import (
	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/render"
)

// Destination is one row of the navigation column. The order of the consts below is the column's
// order, and the value is what a status bar and a test name.
type Destination uint8

const (
	// DestHome is the timeline the plain reader opens on.
	DestHome Destination = iota
	// DestSearch is the query prompt's results.
	DestSearch
	// DestBookmarks is the saved stream, bookmarks half.
	DestBookmarks
	// DestLikes is the saved stream, likes half.
	DestLikes
	// DestProfile is one account's surface.
	DestProfile
)

// Name is the destination's own word, lowercase and without ceremony: the column prints it and a
// test failure reads it.
func (d Destination) Name() string {
	switch d {
	case DestSearch:
		return "search"
	case DestBookmarks:
		return "bookmarks"
	case DestLikes:
		return "likes"
	case DestProfile:
		return "profile"
	}
	return "home"
}

// Destinations is every destination, in the column's order. It is a function rather than a slice a
// caller could reorder, for the reason render.CoverStyles is: the order is a fact about the program.
func Destinations() []Destination {
	return []Destination{DestHome, DestSearch, DestBookmarks, DestLikes, DestProfile}
}

// Marks are the two facts the presentation draws on a row and no other presentation keeps: what the
// last refresh brought, and what the reader has opened. A nil map is the empty set; a post that is in
// neither is an ordinary row.
type Marks struct {
	// New is the last refresh's own news.
	New map[domain.PostID]bool
	// Opened is what the reader has read in this presentation.
	Opened map[domain.PostID]bool
}

// mark is what the presentation says about one post, as facts rather than as cells: whether it is
// news, and whether it is dimmed because it has been read. A post that is both is news — the reader
// has not seen the new one yet — and the mark is drawn by the renderer, which is the only writer of
// a row's cells.
func (m Marks) mark(id domain.PostID) (isNew, dim bool) {
	return m.New[id], !m.New[id] && m.Opened[id]
}

// View is one vivid frame: everything the renderer needs and nothing it could derive twice.
type View struct {
	// Dest is the destination the column marks.
	Dest Destination
	// List is the left column: the surface's own blocks, already in the presentation's grammar.
	List []render.Block
	// Reading is the right column: the opened post's conversation. Empty is the ordinary state
	// before anything has been opened, and the column says so rather than showing a blank half.
	Reading []render.Block
	// ReadingTitle is the reading column's own heading: who or what it is showing, or the empty
	// string before anything is.
	ReadingTitle string
	// Marks is the two row facts above.
	Marks Marks
	// Notice is the reading column's one line when it has nothing to show, in the words of the
	// thing the column is for. It is a value rather than a derived string so that the shell, which
	// owns the wording, keeps owning it.
	Notice string
	// ListName is the list column's heading — "home", "search", a query — and Status is the bar's
	// own fields.
	ListName string
	Status   render.StatusFields
	// Width and Height are the surface's whole size, including the status bar.
	Width, Height int
	// Theme is the colour the frame is drawn in. The zero value paints nothing, so a caller that
	// never sets it gets the mono shape rather than a guess about the terminal.
	Theme Theme
}
