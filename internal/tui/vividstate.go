package tui

import (
	"github.com/LZD121/x-terminal/internal/domain"
)

// vividState is the vivid presentation's own bookkeeping, and it exists only while that
// presentation is on screen. Nothing here is a fact about the feed: the frames hold those, and a
// copy of them here would be a second source of truth about what is loaded.
//
// It is session memory and it is written nowhere — the audit's persistence check enumerates its
// destinations and this value adds none — which is the same rule the translation map follows.
type vividState struct {
	// nav is the destination the presentation is showing, as an index into vivid.NavRows. It is
	// remembered rather than derived from the top frame because a nav destination and a frame kind
	// are not one-to-one: two destinations may show the same kind of frame (bookmarks and likes are
	// both saved lists), and the column must keep saying which one was chosen.
	nav int
	// marked is the last refresh's own news: the ids the walk reported for this refresh. It is a
	// set because a refresh answers with a range and the list holds ids, and it is replaced whole
	// rather than merged so that "new" lasts exactly as long as the next refresh.
	marked map[domain.PostID]bool
	// opened is the posts the reader has opened in this presentation. It is what dims a row after
	// the reader has read it, and it is cleared with the rest when the presentation is left.
	opened map[domain.PostID]bool
	// images is the images this presentation has placed: the id the terminal knows each by, which
	// post it belongs to, and how many rows it occupies. Panic deletes every id here, which is the
	// one thing that must survive being forgotten by the model.
	images []placedImage
}

// placedImage is one image the program has asked the terminal to keep, and it is deliberately the
// smallest set of facts that makes the image removable: the terminal identifies an image by its id,
// the pane needs the post to know when it may still be on screen, and the rows are what the
// reading pane reserved for it.
type placedImage struct {
	id   int
	post domain.PostID
	rows int
}

// news reports whether a post is one the last refresh brought.
func (v vividState) news(id domain.PostID) bool { return v.marked[id] }

// read reports whether the reader has opened a post in this presentation.
func (v vividState) read(id domain.PostID) bool { return v.opened[id] }

// withNews is the value with one refresh's new ids in it, replacing whatever the previous refresh
// brought. A nil ids is the honest "this refresh brought nothing new", and it is not the same as
// "do not touch the marks": an empty set clears them.
func (v vividState) withNews(ids []domain.PostID) vividState {
	marked := make(map[domain.PostID]bool, len(ids))
	for _, id := range ids {
		marked[id] = true
	}
	v.marked = marked
	return v
}

// withOpened is the value with one post marked read.
func (v vividState) withOpened(id domain.PostID) vividState {
	opened := make(map[domain.PostID]bool, len(v.opened)+1)
	for known := range v.opened {
		opened[known] = true
	}
	opened[id] = true
	v.opened = opened
	return v
}

// withImage is the value with one placed image. The slice is copied rather than appended in place
// for the same reason every other map here is: a model value held before the answer must still
// render the presentation it was taken from.
func (v vividState) withImage(img placedImage) vividState {
	images := make([]placedImage, len(v.images), len(v.images)+1)
	copy(images, v.images)
	v.images = append(images, img)
	return v
}
