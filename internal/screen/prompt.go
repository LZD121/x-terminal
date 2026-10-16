package screen

import (
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// Prompt is the query field the shell opens on `/`: the product the search will run under and the
// text typed so far. It is where the product is visible before the request is paid for — the pane
// names it from the moment the prompt opens — and it performs nothing: no request, no clock, no
// terminal. The shell opens it, applies the keys, and reads Submitted on `enter`.
//
// The field is one row: a pasted newline arrives as a space, and the editing is deliberately
// minimal — insert, backspace one cluster, drop the last word, clear. The walk it submits into
// does not exist until `enter`, which is why an open prompt is free.
type Prompt struct {
	product domain.SearchProduct
	text    string
}

// NewPrompt opens the field for a product, seeded with text — the shell re-seeds it with the
// frame's own query when `/` is pressed over a results frame, which is the recovery path for a
// query the status field cut short.
func NewPrompt(p domain.SearchProduct, seed string) Prompt {
	return Prompt{product: p, text: seed}
}

// Product is the product the search will run under: decided when the prompt opens, named on its
// row the whole time it is open.
func (p Prompt) Product() domain.SearchProduct { return p.product }

// Text is the typed text as typed, spaces included.
func (p Prompt) Text() string { return p.text }

// Insert appends one key's — or one paste's — text. A newline in the run becomes a space: the
// field is one row, and a paste is not a way to smuggle a second one in.
func (p Prompt) Insert(s string) Prompt {
	p.text += strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
	return p
}

// Backspace drops exactly one grapheme cluster, by internal/text's own iterator: an `e` with its
// combining mark, a ZWJ family and an emoji each go in one keypress, and no half cluster is ever
// left on screen.
func (p Prompt) Backspace() Prompt {
	p.text = text.DropLastCluster(p.text)
	return p
}

// DeleteWord drops the last word: the trailing spaces first, then everything back to the space
// before them. A space byte is never part of a multi-byte encoding, so the cut cannot split a
// cluster.
func (p Prompt) DeleteWord() Prompt {
	trimmed := strings.TrimRight(p.text, " ")
	if i := strings.LastIndexByte(trimmed, ' '); i >= 0 {
		p.text = trimmed[:i+1]
		return p
	}
	p.text = ""
	return p
}

// Clear empties the row.
func (p Prompt) Clear() Prompt {
	p.text = ""
	return p
}

// Submitted is the query `enter` asks for, and whether there is one at all: an empty or all-space
// field is the documented no-op — the prompt stays open and nothing is spent.
func (p Prompt) Submitted() (string, bool) {
	query := strings.TrimSpace(p.text)
	return query, query != ""
}
