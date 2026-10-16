package render

import (
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// UserRow is one account as a directory entry: exactly one row — the UserHead template
// (`{handle:16} {name}` under Plain, overridden by Markers.UserRow when a grammar wants
// their shapes to differ) — and nothing else. No bio, no counts, no verification glyph,
// no avatar, no location, no website, no join date, no stamp.
//
// The handle field is padded to the template's declared width; the display name is truncated
// so the row's content (handle field, one space, name) is at most the measure, with the
// truncation mark in the last cell it may occupy (`name ≤ measure − handleCells − 1`). An
// account with no display name prints nothing where the name would be: no placeholder, no word.
//
// PostID carries the account's id so the selection mark and the shell's geometry look the
// row up the same way they look up a post block — the walk keys on a string, and this is
// that string at the render seam.
func UserRow(u domain.User, g Grammar, w int, selected string) Block {
	b := &builder{grammar: g.withClock(time.Time{}), width: w, pal: For(RungFromEnv())}
	head := userHeadTemplates(g)
	measure := g.measure(w, 0, 0)
	handleCells := userHandleCells(head)

	// Truncate the display name so handle+space+name fit the measure. The handle field and
	// its trailing separator are fixed; the name is what remains to cut.
	if measure > 0 && u.Name != "" {
		nameBudget := measure - handleCells - 1
		if nameBudget < 1 {
			u.Name = ""
		} else if text.Width(u.Name) > nameBudget {
			u.Name = cutField(u.Name, nameBudget, g.Markers.Truncation)
		}
	}

	mark := " "
	if selected != "" && u.ID == selected {
		mark = MarkSelection
	}

	rows := make([]string, 0, len(head))
	for _, line := range head {
		runs, ok := b.expand(line, template{account: u})
		if !ok {
			continue
		}
		rows = append(rows, b.lead(mark)+paintRuns(runs, b.pal))
	}
	if len(rows) == 0 {
		rows = append(rows, b.lead(mark))
	}
	return Block{PostID: domain.PostID(u.ID), Rows: rows}
}

// UserRows is the people list's blocks in list order: one UserRow per account, with the same
// separator rule as Blocks (one blank row between blocks, none before the first). The selected
// account's rows carry the selection cell like any other target.
func UserRows(users []domain.User, g Grammar, w int, selected string) []Block {
	out := make([]Block, 0, len(users))
	for i, u := range users {
		block := UserRow(u, g, w, selected)
		if i > 0 && g.Separator {
			if len(block.Rows) == 0 || block.Rows[0] != "" {
				block.Rows = append([]string{""}, block.Rows...)
			}
		}
		out = append(out, block)
	}
	return out
}

// userHeadTemplates is the identity row's templates: Markers.UserRow when a grammar wants the
// user-row shape to differ from UserHead, otherwise UserHead, otherwise Plain's UserHead. One
// path for the directory row and the profile header's identity row, so the two cannot drift.
func userHeadTemplates(g Grammar) []string {
	if g.Markers.UserRow != "" {
		return []string{g.Markers.UserRow}
	}
	if len(g.UserHead) > 0 {
		return g.UserHead
	}
	return Plain.UserHead
}

// userHandleCells is the identity field's declared width from the head template (`{handle:16}`
// → 16, `{handle:20}` → 20). Covers that widen the field still measure the name against the
// remaining cells of the row. Falls back to 16 when the template declares no width.
func userHandleCells(head []string) int {
	for _, line := range head {
		for _, field := range parsePlaceholders(line) {
			if field.name == PlaceholderHandle {
				if field.cells > 0 {
					return field.cells
				}
				return 16
			}
		}
	}
	return 16
}
