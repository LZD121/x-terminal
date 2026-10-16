package render

import (
	"strconv"
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// EntCol is the attachment entry's prefix width in cells: `[n] ` (and `url `).
const EntCol = 4

// UrlCol is where a wrapped URL continues: BodyIndent (sm) + EntCol.
const UrlCol = 6

// UrlPrefix is the permalink fallback row's four-cell marker.
const UrlPrefix = "url "

// MediaFallback is which attachments of one post could not be opened, and whether
// that post also prints a permalink row. Failed attachment URLs are still read
// from the post's own refs at render time. Link is the permalink printed when
// Permalink is true (a failed o/y); it is presentation, never stored on domain.
//
// Failed holds 1-based attachment indexes.
type MediaFallback struct {
	Failed    []int
	Permalink bool
	Link      string
}

// Fallbacks is the per-post fallback map a frame carries. The zero value (a nil
// map) means "nothing failed", which keeps every existing caller's frame
// byte-identical aside from the numbered attachment form itself.
type Fallbacks map[domain.PostID]MediaFallback

// For returns the fallback raised for id, or the zero value when none was.
func (f Fallbacks) For(id domain.PostID) (MediaFallback, bool) {
	if f == nil {
		return MediaFallback{}, false
	}
	v, ok := f[id]
	return v, ok
}

// targetURL is the URL the media path would hand over — Playable when it is
// non-empty, else URL. internal/media's dispatch uses the same rule, so the
// printed URL is the URL.
func targetURL(ref domain.MediaRef) string {
	if ref.Playable != "" {
		return ref.Playable
	}
	return ref.URL
}

// PerEntryMarker reports the attachment marker template and whether it carries
// both `{n}` and `{noun}` — the shape a numbered entry needs. The attachment
// marker is rendered once per attachment, not once as a prefix plus per-entry
// tails: Phase 2's "words before `{n}` printed once" cannot express `[1] image,
// [2] video`.
func PerEntryMarker(template string) (string, bool) {
	hasN := strings.Contains(template, PlaceholderCount)
	hasNoun := strings.Contains(template, PlaceholderNoun)
	return template, hasN && hasNoun
}

// singularNoun is the noun one attachment entry takes: every numbered entry is
// exactly one attachment, so the plural nouns never occur on this line.
func singularNoun(kind domain.MediaKind) string {
	switch kind {
	case domain.Video:
		return NounVideo
	case domain.GIF:
		return NounGIF
	default:
		return NounImage
	}
}

// entryMarker expands the per-entry template for one 1-based index and noun.
func entryMarker(template string, n int, noun string) string {
	line := strings.ReplaceAll(template, PlaceholderCount, strconv.Itoa(n))
	return strings.ReplaceAll(line, PlaceholderNoun, noun)
}

// failedSet reports whether a 1-based attachment index is in the fallback.
func failedSet(failed []int) map[int]bool {
	out := make(map[int]bool, len(failed))
	for _, n := range failed {
		out[n] = true
	}
	return out
}

// anyURLEntry reports whether any attachment of the post prints as a URL.
func anyURLEntry(media []domain.MediaRef, fall MediaFallback) bool {
	if len(fall.Failed) == 0 {
		return false
	}
	set := failedSet(fall.Failed)
	for i := range media {
		if set[i+1] {
			return true
		}
	}
	return false
}

// attachmentEntries is the unsplit attachment line: one numbered entry per
// attachment in the post's own order, joined by CopyAttachmentSeparator. It is
// used when nothing failed. An empty media slice yields the empty string.
func attachmentEntries(template string, media []domain.MediaRef) string {
	if len(media) == 0 {
		return ""
	}
	parts := make([]string, 0, len(media))
	for i, ref := range media {
		parts = append(parts, entryMarker(template, i+1, singularNoun(ref.Kind)))
	}
	return strings.Join(parts, CopyAttachmentSeparator)
}

// osc8 wraps text in an OSC 8 hyperlink to url. It is applied when the row is
// emitted and never before measurement: escapes are zero cells wide, and A6
// forbids measuring an escaped string. It adds no SGR — the URL stays body
// colour — so Phase 2's "exactly one text attribute" still holds.
//
// This is the one case in the program where a click is the only remaining route
// to the thing, which is why the escapes are worth spending here and nowhere
// else. The mono rung emits none; colour rungs emit exactly one pair.
func osc8(pal Palette, url, text string) string {
	if url == "" || text == "" {
		return text
	}
	if pal.rung == RungMono {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// wrapURL breaks url into lines of at most col cells with no truncation mark.
// A URL whose length is an exact multiple of col emits no trailing empty row;
// every character is kept. internal/text.Wrap is the one measurement.
func wrapURL(url string, col int) []string {
	if col < 1 {
		col = 1
	}
	return text.Wrap(url, col)
}
