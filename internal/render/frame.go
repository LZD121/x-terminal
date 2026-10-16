package render

import (
	"strings"

	"github.com/LZD121/x-terminal/internal/text"
)

// MinPaneCols is the pane's floor, in cells. Below it the pane renders one line — CopyPaneTooNarrow
// — and the status line still renders, with its fields dropped by its own rule: a pane too narrow
// to read a post is not a pane with no status.
const MinPaneCols = 60

// Frame composes blocks and a status line into exactly h rows of exactly w cells, joined by
// newlines with no trailing one, so the whole frame can be written into a pane of exactly h rows
// without scrolling.
//
// The pane occupies rows 1..h-1 and the status line is row h. Column 1 of every pane row is the
// selection column: the row's own first cell, which the block builder wrote. Blocks are laid out
// top to bottom with the first block at row 1 and no leading separator; rows below the last block
// are blank but still emitted, because a full-height repaint leaves no stale region. Blocks that
// do not fit in the pane are not drawn: the shell's anchor is what decides which of them are at
// the top, and this function does not scroll.
//
// Frame pads and paints, and never truncates a content row: a block row wider than the pane is a
// bug in the block builder. The status line is the one row truncation is legal on, and it does
// its own cutting (status.go) before it arrives here.
func Frame(blocks []Block, status StatusFields, w, h int) string {
	if h < 1 {
		return ""
	}
	pal := For(RungFromEnv())
	rows := make([]string, 0, h)
	pane := h - 1
	if w < MinPaneCols {
		if pane > 0 {
			// A pane too narrow to lay a post out in says so instead of drawing a mutilated one.
			line := cutField(" "+CopyPaneTooNarrow, w, MarkTruncation)
			rows = append(rows, text.Pad(line, w))
		}
	} else {
		for _, block := range blocks {
			if len(rows) >= pane {
				break
			}
			for _, row := range block.Rows {
				if len(rows) >= pane {
					break
				}
				rows = append(rows, paneRow(row, w, pal))
			}
		}
	}
	for len(rows) < pane {
		rows = append(rows, blankRow(w))
	}
	return strings.Join(append(rows, text.Pad(statusLine(status, w, pal), w)), "\n")
}

// paneRow pads one block row to the pane's width and, when the row's first cell is the selection
// mark, paints the padded row as reverse video. The mark is what carries the selection, so the
// bar is painted from it rather than from anything the row carries beside it — and it is painted
// after the padding, so the bar runs to the pane's right edge instead of stopping where the text
// happens to end. A row of a block whose post is not selected starts with a space and is not
// painted at all.
func paneRow(row string, w int, pal Palette) string {
	padded := text.Pad(row, w)
	if strings.HasPrefix(row, MarkSelection) {
		return pal.paint(RoleSelection, padded)
	}
	return padded
}

// blankRow is one entirely blank pane row: w cells of nothing, and no escapes — the separator
// between two blocks, and the fill below the last one, are the frame's own background.
func blankRow(w int) string {
	if w < 1 {
		return ""
	}
	return strings.Repeat(" ", w)
}
