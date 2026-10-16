package render

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// Block is one contiguous group of pane rows: a post's head, body, attachment and quote rows, or
// a message block. Its rows carry no padding — the frame pads every row to the pane's width — and
// its runs are already painted, so measuring one measures the text (an escape sequence is zero
// cells wide).
type Block struct {
	// PostID is the post this block is, and the zero id for a block that is not a post (the help
	// overlay, a state block, a pane header). The selection is a post id, never a row index, so a
	// block has to be able to say which post it is.
	PostID domain.PostID
	// Rows are the block's rows in order: a leading blank separator row when this block follows
	// another, then the post's own rows.
	Rows []string
}

// Selection is the renderer's own value for the row the cursor is on: that row's id, plus the chain of
// containing posts when the row is a post quoted inside another. It is the screen's Target converted by
// the shell at the seam — Path is outermost first and empty for a row of the list itself — so this
// package still imports nothing but internal/domain and internal/text.
//
// ID is a post id and not a string because a selection is a post here: the walk's own Target speaks the
// strings a row is keyed by, and the shell is the one place the two meet.
type Selection struct {
	// ID is the id of the row itself: the post whose block is selected when Path is empty, and the
	// quoted post's own id when it is not.
	ID domain.PostID
	// Path is the outermost containing post first, empty for a row of the list.
	Path []domain.PostID
}

// Blocks builds every block of a list, in list order: the layout is the list's, not the pane's,
// so the pane's first block starts at its first row and the last block's last row may be the
// pane's last row.
//
// Four rules live here. A grammar's pane header is drawn once as the pane's first content rows.
// Content starts at column 2, because column 1 is always the selection column. The first block has
// no separator, one entirely blank row separates every other block from the one above it, and the
// separator is dropped between a post and the continuation that answers it. And every post is
// rendered at render time from the unwrapped content value: nothing was wrapped in the model, so a
// different pane width rewraps rather than reusing lines.
//
// The selection marks rows, and a quoted post is a row: a selection with an empty Path marks every row
// of the block whose post it names (its quoted rows included, which is what makes the quoted text one
// block with the post that quotes it), and a selection whose Path names a quoted post marks only the
// rows of that quoted post — whose rows then carry the selection cell in column 1 followed by the
// quote's own prefix, the selection column plus the quote prefix and not a second quote level.
//
// One composer draws every state: the plain reader hands it Plain and a cover hands it the style's
// Grammar, and nothing else about the call differs. falls is the per-post URL fallback map; the
// zero value (nil) means nothing failed and keeps callers that have no media action byte-identical
// aside from the numbered attachment form itself.
func Blocks(posts []domain.Post, g Grammar, w int, sel Selection, now time.Time, falls Fallbacks) []Block {
	b := &builder{grammar: g.withClock(now), width: w, pal: For(RungFromEnv()), sel: sel, falls: falls}
	blocks := make([]Block, 0, len(posts)+1)
	if len(g.PaneHeader) > 0 {
		// The pane's own header lines, drawn once as the pane's first content rows. They scroll
		// with the content — a sticky row would move the pane's row arithmetic and the anchor for a
		// cosmetic gain — and they never carry the selection mark: they are not a block's rows.
		rows := make([]string, 0, len(g.PaneHeader))
		for _, line := range g.PaneHeader {
			rows = append(rows, " "+b.pal.paint(RoleMuted, line))
		}
		blocks = append(blocks, Block{Rows: rows})
	}
	for i := range posts {
		above := domain.PostID("")
		if continuation(posts, i) {
			above = posts[i-1].ID
		}
		rows := make([]string, 0, 8)
		if i > 0 && above == "" && g.Separator {
			rows = append(rows, "")
		}
		rows = append(rows, b.post(posts[i], posts[i].ID, nil, above, "", false, domain.Post{})...)
		blocks = append(blocks, Block{PostID: posts[i].ID, Rows: rows})
	}
	return blocks
}

// selectionCell is one record's selection cell: the mark when the record is the row the selection
// names, and a space for every other record.
//
// root is the block's own post id and path is the record's own place in the block's quote chain (nil
// for the block itself). A selection with no path marks the whole block — every record whose root it
// names — which is Phase 2's behaviour for a block-level selection and what keeps the stored frames of
// that phase unchanged. A selection with a path marks exactly the record that path reaches, so the
// same post quoted in two places has two separately selectable rows.
func selectionCell(sel Selection, root domain.PostID, p domain.Post, path []domain.PostID) string {
	if sel.ID == "" || p.ID == "" {
		return " "
	}
	if len(sel.Path) == 0 {
		// A block-level selection: the mark goes on every row of the block, the rows of the posts it
		// quotes included.
		if root == sel.ID {
			return MarkSelection
		}
		return " "
	}
	// A quoted row: the selection names it when its own path reaches it, so the same post quoted in two
	// places marks one row and not the other.
	if p.ID == sel.ID && slices.Equal(sel.Path, path) {
		return MarkSelection
	}
	return " "
}

// StateBlock is the one-block pane for a message that is not a post: the loading line, the empty
// state's two lines, and the pane's own words for a failure it has nothing to show for. The head
// line sits where a head row sits (column 2) and the body lines sit where body rows sit, so the
// pane never looks like a different kind of screen, and an empty body renders no body row.
//
// The words are the caller's: copy.go's constants for the empty and loading states, and
// domain.Failure's own classifier sentence for a failure. This function decides the shape only,
// and the shape is the grammar's — a cover's gutter and indent are the ones a post's body carries.
//
// The body is wrapped to the pane's measure like a post body, so it is never wider than a row the
// frame can hold: a sentence longer than the measure — a transport error carrying a URL, say — runs
// over as many body rows as it needs instead of being cut at the pane's edge with no mark. Nothing
// about the words changes; only where they break.
func StateBlock(g Grammar, w int, head, body string) Block {
	b := &builder{grammar: g, width: w, pal: For(RungFromEnv())}
	rows := []string{" " + head}
	for _, line := range text.Wrap(body, g.measure(w, 0, 0)) {
		rows = append(rows, b.bodyRow(" ", "", line))
	}
	return Block{Rows: rows}
}

// continuation reports whether posts[i] is a reply whose parent is the block directly above it:
// the same author, and the post above is the one it answers. The rule is directional on purpose —
// a newest-first payload that puts a reply above its parent marks nothing, rather than marking the
// wrong block — and it needs both accounts to be named, because two posts with no author at all
// are not two posts by the same one.
func continuation(posts []domain.Post, i int) bool {
	if i < 1 || i >= len(posts) {
		return false
	}
	post, above := posts[i], posts[i-1]
	return post.ReplyTo != "" &&
		post.ReplyTo == above.ID &&
		post.Author.ID != "" &&
		post.Author.ID == above.Author.ID
}

// builder is one pass over a list: the grammar with its clock set, the pane's width, the palette every
// run is painted with, and the row the cursor is on.
type builder struct {
	grammar Grammar
	width   int
	pal     Palette
	// sel is the row the cursor is on, as the screen's Target converted by the shell. It is what every
	// record's selection cell is decided from, and nothing else paints a selection.
	sel Selection
	// indent is how many cells every row this builder draws is indented by, after the selection cell
	// and before the quote prefix, the body's own prefix and its indent. It is the conversation's reply
	// step and nothing else: a list's builder leaves it 0, and its rows carry a selection cell in column
	// 1 and the body's own indent behind it, exactly as Phase 2's stored frames show.
	indent int
	// reply is which reply level this builder's rows are at, 0 or 1: it is the measure's reply term and
	// the reason a reply can be indented without its text being cut at the pane's edge. It has no
	// effect on a grammar whose ReplyIndent is 0.
	reply int
	// surface marks the builder as drawing a conversation's rows rather than a list's. The two differ
	// in two ways the 04-UI-SPEC's visibility table states and Phase 2's own frames do not: a record
	// whose content value carries no author and no stamp draws no head row at all — there is nothing in
	// it to print, and the state marker below it is the block's whole content — and that marker stands
	// at the block's own indent rather than at a body's. The one exception is a headless *focal* post:
	// A1-T requires the frame to name the focal post even when its text is unavailable, so that block
	// still draws a head row carrying only the focal marker. A list's tombstone keeps Phase 2's shape
	// (a blank head row and a body row), because its stored frames are that phase's specification.
	surface bool
	// falls is the per-post URL fallback map for this compose pass. A nil map means nothing failed.
	falls Fallbacks
}

// lead is the row's opening cells: the selection cell, then the indent this builder's section adds.
// Both are written before the quote prefix and the body's own indent, which is the UI-SPEC's own order
// for the reply step — a selection cell, a section indent, a quote prefix, a body indent — and it is
// what makes `>` followed by `> ` read as a selection plus a quote rather than as two quote levels.
func (b *builder) lead(mark string) string {
	if b.indent == 0 {
		return mark
	}
	return mark + strings.Repeat(" ", b.indent)
}

// measure is the body measure for a row of this builder's own section and quote depth: the grammar's
// measure with the builder's indent as an extra term, so the reply step moves the indent and the
// measure together. No measurement is done here — internal/text's Width is what the indent is counted
// with, and a run of spaces is exactly as wide as it is long.
func (b *builder) measure(depth int) int {
	m := b.grammar.measure(b.width, depth, b.reply)
	if b.indent > b.grammar.ReplyIndent*b.reply {
		// A caller may set an indent without a reply level (a cover's own record prefix, a one-off
		// indent): the measure follows the cells actually written, so a row can never be indented by
		// more than the measure gave back.
		m -= b.indent - b.grammar.ReplyIndent*b.reply
	}
	return m
}

// post renders one post's rows: the head with the record's own selection cell, the body at the measure,
// the attachment when the grammar keeps it in the body, and the quote.
//
// root is the block's own post id and path is where this record sits in that block's quote chain (nil
// for the block itself): both are what its selection cell is decided from. above is the record this post
// continues, and is empty when it continues nothing; quoting is the record that quotes this one, and is
// empty unless this post is being drawn as a quoted record. The last two are post ids, and both are only
// ever used to spell a marker's {ref}.
func (b *builder) post(p domain.Post, root domain.PostID, path []domain.PostID, above, quoting domain.PostID, focal bool, parent domain.Post) []string {
	g := b.grammar
	mark := selectionCell(b.sel, root, p, path)
	var rows []string
	// A record whose content value carries no author and no stamp draws no head row in a conversation:
	// the state marker below it is its whole content. The one exception is the frame's focal post,
	// because A1-T requires a stripped frame to name the focal post and the state marker cannot carry
	// the marker on its own. The row a headless focal draws is the grammar's own: the plain reader's
	// empty identity and stamp columns and the focal marker, and under a cover whatever that grammar
	// can say about a record that carries no author and no stamp — its own reference, its own state
	// word and its alias for the account nobody named.
	if !(b.surface && headless(p) && !focal) {
		rows = b.head(g.Head, p, mark, "", above, quoting, focal, parent)
	}
	if body, ok := b.placeholder(p); ok {
		if b.surface {
			// The state marker is the block's whole content, at the block's own indent: a tombstone's
			// rows are the selection cell, the section indent and the words, exactly as the UI-SPEC's
			// visibility excerpt draws them.
			return append(rows, b.stateRow(mark, body))
		}
		return append(rows, b.bodyRow(mark, "", body))
	}
	lines := text.Wrap(p.Text, b.measure(0))
	for _, line := range lines {
		rows = append(rows, b.bodyRow(mark, "", line))
	}
	if g.AttachmentRow {
		fall, _ := b.falls.For(p.ID)
		if len(p.Media) > 0 {
			rows = append(rows, b.attachmentRows(mark, "", p.Media, fall)...)
		} else if fall.Permalink {
			// y/o can fail on a post with no attachments; the url row still prints.
			rows = append(rows, b.urlRows(mark, "", fall.Link)...)
		}
	}
	if len(lines) == 0 && len(p.Media) == 0 && p.Quote == nil {
		// Nothing to show at all: the line that says so, rather than a blank space where a post
		// should be. It is reachable with a body the upstream sent empty, which is a different
		// thing from a tombstone.
		rows = append(rows, b.bodyRow(mark, "", g.Markers.MissingText))
	}
	if p.Quote != nil {
		rows = append(rows, b.quote(*p.Quote, 1, root, append(path, p.ID), p.ID)...)
	}
	return rows
}

// placeholder is the body line of a post whose body cannot be rendered: an unavailable post (a
// tombstone) and a post withheld in the reader's country have no readable content, and printing an
// empty body for them would read as a post that was served with nothing in it. Nothing else is
// rendered for such a post: no attachment row, no quote, no head row when the content value carries
// nothing for one (see headless), and no marker beyond the state's own.
func (b *builder) placeholder(p domain.Post) (string, bool) {
	switch p.Visibility {
	case domain.Unavailable:
		return b.grammar.Markers.Unavailable, true
	case domain.Withheld:
		// A list's withheld post keeps Phase 2's body wording (`post unavailable`): the fourth
		// visibility literal is the conversation's own, and a list frame that spoke it would move
		// every stored Phase 2 frame that carries one. A conversation's surface builder uses the
		// grammar's Withheld slot, which is what the UI-SPEC's visibility table draws.
		if b.surface {
			return b.grammar.Markers.Withheld, true
		}
		return b.grammar.Markers.Unavailable, true
	}
	return "", false
}

// quote renders the quoted post's own rows inside the record that quotes it.
//
// A grammar whose QuoteAsRecord is set renders the quoted post as a record of its own: the layout a
// top-level record has, at the same origin, drawn right after the record that quotes it, and
// carrying the back-reference to that record as one of its markers. The list order is untouched
// either way: the quoted post stays where the content puts it.
//
// Every other grammar draws the quote as a nested block or row: the quote prefix at this depth's
// origin, then the quoted post's own head templates, its body at the quote's own indent, and then
// its own quote below them. Nothing is omitted at the depth cap — the prefix simply stops growing.
//
// root and path are what the quoted record's own selection cell is decided from, exactly as they are
// for a top-level record: a quoted post is a row of the block that quotes it, and a selection may name
// it. The path grows by one id per level and the recursion is a chain, so the slice a level appends to
// belongs to that level alone.
func (b *builder) quote(q domain.Post, depth int, root domain.PostID, path []domain.PostID, quoting domain.PostID) []string {
	g := b.grammar
	if g.QuoteAsRecord {
		return b.post(q, root, path, "", quoting, false, domain.Post{})
	}
	mark := selectionCell(b.sel, root, q, path)
	levels := depth
	if levels > g.Markers.QuoteDepthCap {
		levels = g.Markers.QuoteDepthCap
	}
	prefix := strings.Repeat(g.Markers.QuotePrefix, levels)

	rows := b.head(g.QuoteHead, q, mark, prefix, "", "", false, domain.Post{})
	if body, ok := b.placeholder(q); ok {
		return append(rows, b.quoteBodyRow(mark, prefix, body))
	}
	lines := text.Wrap(q.Text, b.measure(depth))
	for _, line := range lines {
		rows = append(rows, b.quoteBodyRow(mark, prefix, line))
	}
	if g.AttachmentRow {
		fall, _ := b.falls.For(q.ID)
		if len(q.Media) > 0 {
			rows = append(rows, b.attachmentRows(mark, prefix, q.Media, fall)...)
		} else if fall.Permalink {
			rows = append(rows, b.urlRows(mark, prefix, fall.Link)...)
		}
	}
	if len(lines) == 0 && len(q.Media) == 0 && q.Quote == nil {
		rows = append(rows, b.quoteBodyRow(mark, prefix, g.Markers.MissingText))
	}
	if q.Quote != nil {
		rows = append(rows, b.quote(*q.Quote, depth+1, root, append(path, q.ID), q.ID)...)
	}
	return rows
}

// head renders one record's head: one row per line of the grammar's head templates, and then the
// marker run placed where the grammar's own placement says it lands.
//
// The templates are the whole of what a style's head is made of, and they are expanded by the one
// placeholder expander below: a style cannot grow a second template language, and no branch in
// this file reads a style's name.
func (b *builder) head(templates []string, p domain.Post, mark, prefix string, above, quoting domain.PostID, focal bool, parent domain.Post) []string {
	g := b.grammar
	run := b.markerRun(p, above, quoting, focal, parent)
	subject := template{post: p, account: p.Author, ref: p.ID}
	if g.MarkerPlace == MarkersInline {
		subject.markers = run
	}
	rows := make([]string, 0, len(templates)+len(run))
	for _, line := range templates {
		if line == "" {
			// A template line that is the empty string is a blank row and not a dropped line: it is
			// the blank git leaves between a commit's header and its message, and column 1 of a
			// blank row is blank even when the block is selected — a mark on an otherwise empty row
			// reads as a bug.
			rows = append(rows, " ")
			continue
		}
		pieces, says := b.expand(line, subject)
		if !says {
			// Its only placeholder names nothing and its text is nothing but that field's padding:
			// a line has to say something to be drawn.
			continue
		}
		rows = append(rows, b.headRows(mark, prefix, pieces)...)
	}
	switch g.MarkerPlace {
	case MarkersHeadLines:
		for _, marker := range run {
			rows = append(rows, b.lead(mark)+b.pal.paint(RoleMuted, prefix+marker))
		}
	case MarkersBodyLines:
		for _, marker := range run {
			rows = append(rows, b.markerRow(mark, prefix, marker))
		}
	}
	return rows
}

// markerRun is the record's markers, in the run's fixed order, each already expanded: the
// relationship markers the record carries, and — for a grammar whose attachment is one of the
// record's markers rather than a line of its own body — its attachment.
//
// The order is Phase 2's for the three markers it had (repost, limited, continued), then Phase 3's
// (a quoted record's back-reference to the record that quotes it, and an attachment marker), and the
// two this phase adds follow them: `focal`, which the post the frame was opened on carries, and
// `reply to {handle}`, which a reply carries when its parent is neither the focal post nor the block
// directly above it. Appending rather than inserting is what keeps every stored covered frame's
// marker order byte-identical.
//
// focal and parent are the conversation's two facts and are both empty for a list's own blocks: the
// zero Post is "this record answers nothing outside the frame". A marker whose template is empty is a
// marker the grammar does not have, and it contributes nothing to the run.
func (b *builder) markerRun(p domain.Post, above, quoting domain.PostID, focal bool, parent domain.Post) []string {
	g := b.grammar
	run := make([]string, 0, 7)
	add := func(line string) {
		if line != "" {
			run = append(run, line)
		}
	}
	if p.RepostedBy != nil {
		// The repost marker names the account that surfaced the post, not its author.
		add(b.expandText(g.Markers.Repost, template{post: p, account: *p.RepostedBy, ref: p.ID}))
	}
	if p.Visibility == domain.Limited {
		add(b.expandText(g.Markers.Limited, template{post: p, account: p.Author, ref: p.ID}))
	}
	if above != "" {
		add(b.expandText(g.Markers.Continued, template{post: p, account: p.Author, ref: above}))
	}
	if quoting != "" {
		add(b.expandText(g.Markers.Parent, template{post: p, account: p.Author, ref: quoting}))
	}
	if len(p.Media) > 0 && !g.AttachmentRow {
		// Cover markers stay the unsplit numbered list: a URL on a covered screen is a
		// DSGT-09 violation, and the shell passes the zero Fallbacks under a cover anyway.
		add(attachmentEntries(g.Markers.Attachment, p.Media))
	}
	if focal {
		add(b.expandText(g.Markers.Focal, template{post: p, account: p.Author, ref: p.ID}))
	}
	if parent.ID != "" {
		// The marker names the post this one answers, so `{handle}` is that post's author — the
		// account the reply is addressed to — and never this record's own.
		add(b.expandText(g.Markers.ReplyTo, template{post: p, account: parent.Author, ref: parent.ID}))
	}
	return run
}

// template is the subject one template line is expanded against: the post the line is about, the
// account its identity placeholders name — the post's author, or the account that reposted it in a
// repost marker — the post its {ref} names, and the marker run its {markers} stands for.
type template struct {
	post    domain.Post
	account domain.User
	ref     domain.PostID
	markers []string
}

// headRun is one run of an assembled head line: its characters and the role they are painted with.
type headRun struct {
	text string
	role Role
}

// parsePlaceholders finds every placeholder occurrence in a template line, in order. It looks for
// the braces rather than for the names, so an unknown field is simply a field that names nothing
// and the closed set of copy.go's placeholders is the whole of the template language.
func parsePlaceholders(line string) []placeholder {
	fields := make([]placeholder, 0, 2)
	for at := 0; at < len(line); {
		open := strings.IndexByte(line[at:], '{')
		if open < 0 {
			break
		}
		start := at + open
		shut := strings.IndexByte(line[start:], '}')
		if shut < 0 {
			break
		}
		body := line[start+1 : start+shut]
		field := placeholder{name: "{" + body + "}", at: start, end: start + shut + 1}
		if colon := strings.IndexByte(body, ':'); colon >= 0 {
			field.name = "{" + body[:colon] + "}"
			if cells, err := strconv.Atoi(body[colon+1:]); err == nil {
				field.cells = cells
			}
		}
		fields = append(fields, field)
		at = field.end
	}
	return fields
}

// placeholder is one `{name}` or `{name:cells}` occurrence in a template line: which field it
// names, the width the template declares for it in cells (0 when it declares none) and where it
// sits in the line, in bytes.
type placeholder struct {
	name  string
	cells int
	at    int
	end   int
}

// expand assembles one template line into its runs, and reports whether the line says anything: a
// line whose only placeholder names nothing is a line of nothing but that field's padding, and it
// is dropped rather than drawn.
//
// The roles are the ones Phase 2 gave the head, and the stored colour frames are where that rule is
// written down: the identity field and the one cell that separates it from the rest of the line are
// the terminal's own foreground, while the prefix, the templates' own text and every other
// placeholder are the line's muted chrome.
func (b *builder) expand(line string, t template) ([]headRun, bool) {
	fields := parsePlaceholders(line)
	if len(fields) == 1 && fields[0].at == 0 && fields[0].end == len(line) {
		if _, says := b.value(fields[0], t); !says {
			return nil, false
		}
	}
	runs := make([]headRun, 0, 2*len(fields)+1)
	at := 0
	for _, field := range fields {
		if field.at > at {
			runs = append(runs, headRun{line[at:field.at], RoleMuted})
		}
		value, _ := b.value(field, t)
		if field.cells > 0 {
			// A field that does not fit is cut at a cluster boundary with the truncation mark in
			// its last cell and then padded to the column: the cells behind it may not move.
			value = text.Pad(cutField(value, field.cells, b.grammar.Markers.Truncation), field.cells)
		}
		role := roleOf(field.name)
		end := field.end
		if role == RoleDominant && end < len(line) && line[end] == ' ' {
			// The identity field keeps the one cell that separates it from the rest of the line: a
			// field's separator is part of the field, which is what Phase 2's head rows show and
			// what the templates are written to say.
			value += " "
			end++
		}
		runs = append(runs, headRun{value, role})
		at = end
	}
	if at < len(line) {
		runs = append(runs, headRun{line[at:], RoleMuted})
	}
	return runs, true
}

// expandText is one template expanded to its characters: a marker's own line, which is chrome
// wherever its placeholders name something and empty when the template names nothing.
func (b *builder) expandText(line string, t template) string {
	runs, _ := b.expand(line, t)
	return runsText(runs)
}

// value is one placeholder's substitution, and whether it names anything at all: the empty string
// is a field that is not there, which is what drops a line that has nothing else.
func (b *builder) value(field placeholder, t template) (string, bool) {
	switch field.name {
	case PlaceholderStamp:
		return b.grammar.Stamp.Stamp(t.post.CreatedAt), true
	case PlaceholderRef:
		if b.grammar.Ref == nil {
			return "", false
		}
		ref := b.grammar.Ref(t.ref)
		return ref, ref != ""
	case PlaceholderHandle:
		return b.identity(t.account).Handle, true
	case PlaceholderName:
		return b.identity(t.account).Name, true
	case PlaceholderEmail:
		return b.identity(t.account).Email, true
	case PlaceholderState:
		word := b.grammar.state(t.post.Visibility)
		return word, word != ""
	case PlaceholderMarkers:
		return inlineMarkers(t.markers), len(t.markers) > 0
	case PlaceholderText:
		return t.post.Text, t.post.Text != ""
	}
	return "", false
}

// roleOf is the role a substitution is painted with: the identity fields and the post's own text
// are the terminal's own foreground, and everything else a head line carries is chrome.
func roleOf(name string) Role {
	switch name {
	case PlaceholderHandle, PlaceholderName, PlaceholderEmail, PlaceholderText:
		return RoleDominant
	}
	return RoleMuted
}

// inlineMarkers joins a marker run the way an inline grammar writes it: a marker template that
// opens with a space carries its own separator, and one that does not is appended behind the head's
// two-cell field gap. That is Phase 2's own rule for the plain grammar's three markers, and it is
// what puts a cover's ` parent={ref}` one cell behind the field before it while the plain reader's
// `continued` sits two behind the stamp.
func inlineMarkers(run []string) string {
	var out strings.Builder
	for _, marker := range run {
		if marker == "" {
			continue
		}
		if !strings.HasPrefix(marker, " ") {
			out.WriteString(strings.Repeat(" ", headGap))
		}
		out.WriteString(marker)
	}
	return out.String()
}

// headRows lays one assembled head line out as pane rows: the line itself when it fits the pane
// behind the selection cell and the quote prefix, or the lines text.Wrap breaks it into when it
// does not, each starting at the head's own indent — exactly as a terminal wraps a long printed
// line, and with the same text.Wrap the body uses, so a cover cannot introduce a second wrap point.
//
// A line that has to wrap is painted as chrome: a wrap point inside a field is not a field, and no
// head this program draws wraps inside its identity column — the widest field is 20 cells, the
// widest quote prefix 6, and the pane's floor is 60.
func (b *builder) headRows(mark, prefix string, pieces []headRun) []string {
	plain := runsText(pieces)
	room := b.width - text.Width(b.lead(mark)) - text.Width(prefix)
	if room < 1 {
		room = 1
	}
	if text.Width(plain) <= room {
		return []string{b.paintHead(mark, prefix, pieces)}
	}
	lines := text.Wrap(plain, room)
	rows := make([]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, b.lead(mark)+b.pal.paint(RoleMuted, prefix+line))
	}
	return rows
}

// runsText is an assembled line's characters with no escapes: the string every width decision about
// a head line is made on, because an escape sequence is zero cells wide and a wrap point decided on
// a coloured string would be a second, hidden measure.
func runsText(runs []headRun) string {
	var out strings.Builder
	for _, run := range runs {
		out.WriteString(run.text)
	}
	return out.String()
}

// paintHead paints one assembled head line: the prefix and every run of chrome as one muted run per
// change of role, the identity field and its separator as they are. Adjacent runs of one role are
// one run, so a line carries one escape pair per change of role and never a pair of escapes around
// nothing.
func (b *builder) paintHead(mark, prefix string, pieces []headRun) string {
	var out strings.Builder
	out.WriteString(b.lead(mark))
	role := RoleMuted
	var run strings.Builder
	run.WriteString(prefix)
	flush := func() {
		if run.Len() == 0 {
			return
		}
		out.WriteString(b.pal.paint(role, run.String()))
		run.Reset()
	}
	for _, piece := range pieces {
		if piece.text == "" {
			continue
		}
		if piece.role != role {
			flush()
			role = piece.role
		}
		run.WriteString(piece.text)
	}
	flush()
	return out.String()
}

// bodyRow is one body row: the selection cell, the quote prefix, the body's own prefix and indent,
// and the line. Body text is the dominant run and is never coloured: it is what the reader came
// for.
func (b *builder) bodyRow(mark, prefix, line string) string {
	g := b.grammar
	return b.lead(mark) + b.pal.paint(RoleMuted, prefix) + b.pal.paint(RoleMuted, g.BodyPrefix) +
		strings.Repeat(" ", g.BodyIndent) + line
}

// stateRow is one state marker row of a conversation: the selection cell, the block's own indent and
// the marker, with no body indent behind it. It is the shape the 04-UI-SPEC's visibility table gives a
// tombstone and a withheld record — the marker is the block's whole content, so it stands where the
// block's rows begin rather than where a body's text does — and it is why the row reads as the block
// itself rather than as a line under it. The grammar's own body prefix is still written: a cover's
// gutter is part of what makes one record one record.
func (b *builder) stateRow(mark, words string) string {
	return b.lead(mark) + b.pal.paint(RoleMuted, b.grammar.BodyPrefix+words)
}

// quoteBodyRow is one body row of a quoted post: the quote prefix at its origin and then the
// quote's own indent, which is not the grammar's body indent — a record's body sits where its style
// puts it, and the text it quotes is indented by the same two cells behind the prefix in every
// style.
func (b *builder) quoteBodyRow(mark, prefix, line string) string {
	return b.lead(mark) + b.pal.paint(RoleMuted, prefix) + strings.Repeat(" ", QuoteBodyIndent) + line
}

// markerRow is one marker line at the body's own origin: the shape a body row has, drawn for a
// grammar whose markers are lines of the body rather than fields of the head.
func (b *builder) markerRow(mark, prefix, marker string) string {
	g := b.grammar
	return b.lead(mark) + b.pal.paint(RoleMuted, prefix+g.BodyPrefix+strings.Repeat(" ", g.BodyIndent)+marker)
}

// attachmentRows is the attachment line for a grammar that keeps it in the body.
// When nothing failed the entries share one row, joined by CopyAttachmentSeparator.
// When any entry is a URL the line splits into one entry per row; each URL wraps at
// measure-4 with continuations at UrlCol, never truncated, and carries an OSC 8
// hyperlink exactly when the palette emits escapes. A Permalink fallback appends a
// `url ` row with the same wrap rule.
func (b *builder) attachmentRows(mark, prefix string, media []domain.MediaRef, fall MediaFallback) []string {
	g := b.grammar
	template, _ := PerEntryMarker(g.Markers.Attachment)
	chrome := func(line string) string {
		return b.lead(mark) + b.pal.paint(RoleMuted, prefix+g.BodyPrefix+strings.Repeat(" ", g.BodyIndent)+line)
	}
	if !anyURLEntry(media, fall) && !fall.Permalink {
		line := attachmentEntries(template, media)
		if line == "" {
			return nil
		}
		return []string{chrome(line)}
	}
	set := failedSet(fall.Failed)
	measure := b.measure(0)
	urlCol := measure - EntCol
	if urlCol < 1 {
		urlCol = 1
	}
	var rows []string
	for i, ref := range media {
		n := i + 1
		if set[n] {
			url := targetURL(ref)
			parts := wrapURL(url, urlCol)
			if len(parts) == 0 {
				parts = []string{""}
			}
			// "[{n}] {noun}" with noun "x" then strip "x" yields the four-cell prefix "[n] ".
			first := strings.TrimSuffix(entryMarker(template, n, "x"), "x")
			rows = append(rows, b.lead(mark)+b.pal.paint(RoleMuted, prefix+g.BodyPrefix+strings.Repeat(" ", g.BodyIndent)+first)+osc8(b.pal, url, parts[0]))
			for _, cont := range parts[1:] {
				// Continuations sit at UrlCol (BodyIndent + EntCol): indent + four spaces.
				pad := strings.Repeat(" ", EntCol)
				rows = append(rows, b.lead(mark)+b.pal.paint(RoleMuted, prefix+g.BodyPrefix+strings.Repeat(" ", g.BodyIndent)+pad)+osc8(b.pal, url, cont))
			}
			continue
		}
		rows = append(rows, chrome(entryMarker(template, n, singularNoun(ref.Kind))))
	}
	if fall.Permalink {
		rows = append(rows, b.urlRows(mark, prefix, fall.Link)...)
	}
	return rows
}

// urlRows is the permalink fallback row kind: at the body indent, prefix `url `
// (EntCol cells), the URL wrapped the same way as an attachment URL entry.
func (b *builder) urlRows(mark, prefix, url string) []string {
	g := b.grammar
	measure := b.measure(0)
	urlCol := measure - EntCol
	if urlCol < 1 {
		urlCol = 1
	}
	parts := wrapURL(url, urlCol)
	if len(parts) == 0 {
		parts = []string{""}
	}
	var rows []string
	rows = append(rows, b.lead(mark)+b.pal.paint(RoleMuted, prefix+g.BodyPrefix+strings.Repeat(" ", g.BodyIndent)+UrlPrefix)+osc8(b.pal, url, parts[0]))
	for _, cont := range parts[1:] {
		pad := strings.Repeat(" ", EntCol)
		rows = append(rows, b.lead(mark)+b.pal.paint(RoleMuted, prefix+g.BodyPrefix+strings.Repeat(" ", g.BodyIndent)+pad)+osc8(b.pal, url, cont))
	}
	return rows
}

// identity is the grammar's alias for an account, before any field of a head line is built from it:
// the repost marker names an account in a free run of text, where no column applies.
func (b *builder) identity(u domain.User) CoverAlias { return b.grammar.Identity(u) }

// stamp is the head's time column for an instant: the grammar's format, in the grammar's zone.
func (b *builder) stamp(t time.Time) string { return b.grammar.Stamp.Stamp(t) }

// cutField cuts s to at most w cells with the truncation mark in its last cell, and returns s
// unchanged when it already fits. w < 1 has no room for anything, not even the mark, so it returns
// nothing. The cut lands on a cluster boundary and the spaces it left at the end are dropped: the
// mark marks the last cell the field occupies, and a trailing space is not a cell the field says
// anything in.
func cutField(s string, w int, mark string) string {
	if w < 1 {
		return ""
	}
	if text.Width(s) <= w {
		return s
	}
	return strings.TrimRight(text.Truncate(s, w-text.Width(mark)), " ") + mark
}
