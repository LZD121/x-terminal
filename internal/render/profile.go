package render

import (
	"strconv"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// ProfileHeader is the account's header block: identity, every bio line, then the facts row.
// The block is not selectable — PostID stays empty — because there is nothing to open inside an
// account's own profile. Absent values are omitted, never zeroed: a nil count pointer drops its
// clause, a zero Joined drops the joined clause, and an empty bio prints no bio rows.
func ProfileHeader(p domain.Profile, g Grammar, w int) Block {
	b := &builder{grammar: g.withClock(time.Time{}), width: w, pal: For(RungFromEnv())}
	rows := make([]string, 0, 8)

	head := g.UserHead
	if len(head) == 0 && g.Markers.UserRow != "" {
		head = []string{g.Markers.UserRow}
	}
	if len(head) == 0 {
		head = Plain.UserHead
	}
	for _, line := range head {
		runs, ok := b.expand(line, template{account: p.User})
		if !ok {
			continue
		}
		rows = append(rows, b.lead(" ")+paintRuns(runs, b.pal))
	}

	if p.Bio != "" {
		measure := g.measure(w, 0, 0)
		for _, line := range text.Wrap(p.Bio, measure) {
			rows = append(rows, b.bodyRow(" ", "", line))
		}
	}

	if facts := profileFacts(p, g, w, b); facts != "" {
		rows = append(rows, b.bodyRow(" ", "", facts))
	}
	return Block{Rows: rows}
}

// Profile is the profile frame's one block list: the header, then the account's posts as ordinary
// Blocks. The blank separator between header and first post is prepended onto the first post
// block (Phase 2's separator rule), so the shell's blockIndex stays walk.Index+1.
func Profile(p domain.Profile, posts []domain.Post, g Grammar, w int, sel Selection, now time.Time) []Block {
	header := ProfileHeader(p, g, w)
	out := make([]Block, 0, 1+len(posts))
	out = append(out, header)
	if len(posts) == 0 {
		return out
	}
	postBlocks := Blocks(posts, g, w, sel, now, nil)
	if len(g.PaneHeader) > 0 && len(postBlocks) > 0 && postBlocks[0].PostID == "" {
		postBlocks = postBlocks[1:]
	}
	if g.Separator && len(postBlocks) > 0 {
		if len(postBlocks[0].Rows) == 0 || postBlocks[0].Rows[0] != "" {
			postBlocks[0].Rows = append([]string{""}, postBlocks[0].Rows...)
		}
	}
	return append(out, postBlocks...)
}

// profileFacts is the header's facts row: the grammar's own clauses, each omitted when the value it
// names is absent, joined by two spaces and cut to the pane's measure like any other field.
//
// A grammar that carries no clauses of its own renders no facts row. The plain reader's clauses are
// not a fallback for a cover: they are the plain reader's words for an account's numbers
// (`followers:`, `following:`, `posts:`, `joined:`), and a covered frame that printed them would be
// speaking in the product's voice about an account it is not naming. Every shipped cover writes its
// own clauses (cover.go); this branch is what holds the rule for a grammar that does not.
func profileFacts(p domain.Profile, g Grammar, w int, b *builder) string {
	clauses := g.AccountFacts
	if clauses == nil {
		if _, isCover := CoverOfGrammar(g); isCover {
			return ""
		}
		clauses = Plain.AccountFacts
	}
	if len(clauses) == 0 {
		return ""
	}
	parts := make([]string, 0, len(clauses))
	for _, tmpl := range clauses {
		clause, ok := expandFact(tmpl, p, g)
		if !ok {
			continue
		}
		parts = append(parts, clause)
	}
	if len(parts) == 0 {
		return ""
	}
	joined := strings.Join(parts, "  ")
	measure := g.measure(w, 0, 0)
	if measure < 1 {
		return ""
	}
	if text.Width(joined) <= measure {
		return joined
	}
	return cutField(joined, measure, g.Markers.Truncation)
}

func expandFact(tmpl string, p domain.Profile, g Grammar) (string, bool) {
	switch {
	case strings.Contains(tmpl, "{followers}"):
		if p.Followers == nil {
			return "", false
		}
		return strings.ReplaceAll(tmpl, "{followers}", strconv.Itoa(*p.Followers)), true
	case strings.Contains(tmpl, "{following}"):
		if p.Following == nil {
			return "", false
		}
		return strings.ReplaceAll(tmpl, "{following}", strconv.Itoa(*p.Following)), true
	case strings.Contains(tmpl, "{posts}"):
		if p.Posts == nil {
			return "", false
		}
		return strings.ReplaceAll(tmpl, "{posts}", strconv.Itoa(*p.Posts)), true
	case strings.Contains(tmpl, "{joined}") || strings.Contains(tmpl, PlaceholderJoined):
		if p.Joined.IsZero() || g.Stamp.MonthYear == "" {
			return "", false
		}
		joined := p.Joined.In(time.UTC).Format(g.Stamp.MonthYear)
		out := strings.ReplaceAll(tmpl, "{joined}", joined)
		out = strings.ReplaceAll(out, PlaceholderJoined, joined)
		return out, true
	}
	return "", false
}

func paintRuns(runs []headRun, pal Palette) string {
	var out strings.Builder
	for _, run := range runs {
		out.WriteString(pal.paint(run.role, run.text))
	}
	return out.String()
}
