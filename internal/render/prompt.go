package render

import (
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// PromptRow is the one row of the query prompt: the blank selection cell (nothing is selected
// while the prompt is open), the prefix — which names the product before the request is paid for —
// two spaces, and as much of the typed text as fits. It returns the row and the caret's column,
// 1-based, always within 1..w.
//
// The prefix is Markers.PromptSearch for posts and Markers.PromptPeople for people, so a cover
// renames the prompt without a code change. For posts the product fills the {product} slot; for
// people the prefix is the whole statement (`people:`). The row carries no cursor byte: the caret
// is the terminal's own cursor, and the returned column is the shell's to place (05-UI-SPEC
// records this as the one place the program positions a cursor — a deviation, with its cost
// stated there).
//
// A query longer than the row shows its TAIL, so the caret stays on screen, and the caret column
// is computed from what is shown. The row never exceeds w cells; internal/text measures every
// string here, so a CJK or emoji query measures correctly.
func PromptRow(g Grammar, p domain.SearchProduct, typed string, w int) (string, int) {
	pal := For(RungFromEnv())
	var prefix string
	if p == domain.People {
		prefix = g.Markers.PromptPeople
		if prefix == "" {
			prefix = MarkerPromptPeople
		}
	} else {
		prefix = g.Markers.PromptSearch
		if prefix == "" {
			prefix = MarkerPromptSearch
		}
		prefix = strings.ReplaceAll(prefix, PlaceholderProduct, p.String())
	}
	fixed := " " + prefix + "  "
	fixedWidth := text.Width(fixed)

	avail := w - fixedWidth
	if avail < 0 {
		avail = 0
	}
	// The tail, with the space the cut may have landed in front of trimmed away: a tail that
	// begins with a space wastes a cell and reads as a second separator.
	shown := strings.TrimLeft(text.Tail(typed, avail), " ")

	row := pal.paint(RoleMuted, fixed) + shown
	if text.Width(row) > w {
		// A pane narrower than the prefix alone: the row is cut like any field that does not fit,
		// rather than handed to the terminal wider than the pane — which would let the terminal,
		// not the frame, decide where the line goes next.
		row = cutField(row, w, MarkTruncation)
	}

	// The caret sits after the last shown cell, clamped into the row. When the pane is so narrow
	// that the prefix itself was cut, the caret lands on the row's last cell — there is nowhere
	// else for it to go, and the caret's own rule (always within 1..W) holds either way.
	caret := text.Width(row) + 1
	if caret > w {
		caret = w
	}
	if caret < 1 {
		caret = 1
	}
	return row, caret

}
