package render

import (
	"strconv"

	"github.com/LZD121/x-terminal/internal/domain"
)

// TrendRows is Explore's trend list as pane rows: one row per trend, the selected one marked.
//
// It is one row per trend rather than a block per trend because a trend has no body: the selection
// mark is the row's whole presentation, and the row is the name.
func TrendRows(g Grammar, w int, trends []domain.Trend, selected string) []string {
	rows := make([]string, 0, len(trends))
	for i, trend := range trends {
		mark := " "
		if strconv.Itoa(i) == selected {
			mark = MarkSelection
		}
		rows = append(rows, cutField(mark+" "+trend.Name, w, MarkTruncation))
	}
	return rows
}
