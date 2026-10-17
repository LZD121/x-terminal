package render

import (
	"strconv"
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Links is the menu a post's body opens: a heading, then one numbered row per thing the body names.
//
// It is a full-pane block, like the help overlay and for the same reason: the reader is choosing
// between rows, and a menu squeezed beside the list would be a second thing to read. The rows are
// numbered because the number is the whole interaction — there is no cursor in this menu.
//
// A URL loses its scheme in the row: the scheme is the same for every address a post carries, and
// the cells it costs are cells the path needs. The full address is what is opened.
func Links(g Grammar, w int, links []domain.Link) []string {
	rows := make([]string, 0, len(links)+1)
	rows = append(rows, " "+MarkerLinksHeading)
	for i, link := range links {
		if i >= maxLinkRows {
			rows = append(rows, "     more links than a digit can reach")
			break
		}
		row := " " + strconv.Itoa(i+1) + "  " + linkLabel(link)
		rows = append(rows, cutField(row, w, MarkTruncation))
	}
	return rows
}

// maxLinkRows is how many rows the menu can act on: a digit key reaches nine.
const maxLinkRows = 9

// MarkerLinksHeading is the menu's heading. It is in the marker table rather than a literal in this
// function so a cover's vocabulary could carry its own word for it later without a second edit.
const MarkerLinksHeading = "links"

// linkLabel is one link as the menu prints it.
func linkLabel(link domain.Link) string {
	switch link.Kind {
	case domain.LinkHashtag:
		return "#" + link.Target
	case domain.LinkMention:
		return "@" + link.Target
	}
	return strings.TrimPrefix(strings.TrimPrefix(link.Target, "https://"), "http://")
}
