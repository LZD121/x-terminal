package render

import (
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Permalink builds the presentation URL for a post at the call site from the
// post's ID and author ScreenName. It is never stored on a domain value.
//
// When the author's handle is known (after stripping a single leading "@"),
// the form is https://x.com/{handle}/status/{id}. A post with no handle — a
// tombstone or withheld case — builds the handle-free form
// https://x.com/i/status/{id} rather than refusing; that form still resolves.
// Both forms are what y (copy) and o (open) act on.
func Permalink(p domain.Post) string {
	handle := strings.TrimPrefix(p.Author.ScreenName, "@")
	id := string(p.ID)
	if handle != "" {
		return "https://x.com/" + handle + "/status/" + id
	}
	return "https://x.com/i/status/" + id
}
