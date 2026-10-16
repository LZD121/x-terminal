package render

import (
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// This file is the conversation's view: the block order of a thread frame, the reply step, the
// relationship markers, the visibility rows and the three conditional pane statements. It is not a
// second renderer — every row is composed by the builder in block.go, at the same measure, with the
// same palette, the same selection cell and the same quote prefix — and it is not a second block
// builder either: what it decides is the order of the blocks and which block a row belongs to.
//
// The frame is one top-to-bottom list, which is the phase's central reading: there is no separate
// "post detail" screen, so the chain above the post, the post itself, its own continuations and one
// page of its replies are rows of one pane. Nothing here writes an escape sequence by hand and
// nothing here measures a string except through internal/text.

// Conversation is what the renderer needs to draw a conversation, and nothing else. Every field is a
// value a screen already decided: the posts in the answer's own order, and the flags the screen
// computed from the walk and from the content.
//
// No string, no count and no index of a screen value is copied here — the renderer derives the
// statements it prints from these flags and never re-derives a flag from the posts. That split is
// what makes the partial-thread statement honest: whether the reply walk stopped is a fact about a
// cursor the walk spent, and a renderer that guessed at it from a post count would be inventing a
// fact about the upstream.
type Conversation struct {
	// Ancestors is the chain above the focal post, root first, in the answer's own order, at indent
	// 0 above the focal block. It may be empty and it may hold a tombstone, which keeps its place.
	Ancestors []domain.Post
	// Focal is the post the frame was opened on: an ordinary block at indent 0 whose head row carries
	// `focal` (Markers.Focal), and the block the selection starts in.
	Focal domain.Post
	// Continued is the focal post's own continuations, in the answer's own group order, at indent 0,
	// each carrying `continued`. There is no blank row between the focal block and them: a
	// continuation answers the block directly above it, which is Phase 2's continuation rule.
	Continued []domain.Post
	// Replies is one page of the conversation under the focal post, in the answer's own ranked order,
	// one block per post, every row at the reply step. Nothing here re-sorts it and nothing derives a
	// tree from it: a reply whose parent is not in the frame keeps its place, and the frame says so
	// with `reply to {handle}` rather than by indenting.
	Replies []domain.Post
	// Loading is a first page in flight with no answer yet: the pane says `loading thread...` below
	// the focal block, because the copy the frame was opened with is already on screen.
	Loading bool
	// Read records that an answer has come back at all. It is the difference between "the answer
	// carried no replies" and "nothing has been read", which is the difference the empty-reply
	// statement turns on.
	Read bool
	// Stalled is the first partial-thread condition: the reply walk ended on a cursor the upstream
	// would not advance.
	Stalled bool
	// NoReplies is a read that succeeded and brought no reply at all: `no replies` with Phase 2's
	// empty-read body row under it. It is suppressed by PartialHidden, which states the same emptiness
	// as a fact about the conversation rather than about the section.
	NoReplies bool
	// PartialAbove is the second partial-thread condition: the focal post answers a post the frame
	// does not hold, so the chain above it is not reachable. Its statement is drawn directly above the
	// focal block, preceded by one blank row.
	PartialAbove bool
	// PartialHidden is the third partial-thread condition: the answer carried no replies and no reply
	// cursor while the focal post's own reply count says there are replies. The count is consulted as
	// evidence and is never printed (REND-01): nothing in this file knows the number.
	PartialHidden bool
}

// ThreadBlocks builds a conversation's blocks, in the UI-SPEC's order: the chain above, the focal
// post, the focal post's own continuations, the replies, and the statements.
//
// One blank row separates two blocks, exactly as in a list, with one exception: no separator sits
// between the focal block and a continuation of it, because a continuation answers the block directly
// above it and Phase 2's rule reads that relationship off the absence of a separator. There is no
// leading separator, none after the last row, and an empty part renders no gap at all — a
// conversation with no ancestors starts at the focal post.
//
// w is the pane's width: the reply step and the measure both come from it, so a narrower pane wraps
// the same reply rather than cutting it. sel is the row the cursor is on, as the shell converted it:
// a selection naming a quoted post marks that quoted post's rows, and one naming a top-level post
// marks every row of its block, its quote's rows included.
func ThreadBlocks(c Conversation, g Grammar, w int, sel Selection, now time.Time, falls Fallbacks) []Block {
	g = g.withClock(now)
	blocks := make([]Block, 0, len(c.Ancestors)+len(c.Continued)+len(c.Replies)+4)
	base := threadSection{grammar: g, width: w, sel: sel, surface: true, falls: falls}
	reply := base
	reply.indent, reply.reply = g.ReplyIndent, 1

	// (1) The chain above: ordinary blocks at indent 0, root first.
	for i := range c.Ancestors {
		blocks = append(blocks, threadBlock(base, c.Ancestors[i], i > 0))
	}
	// (2) The chain-cut statement: directly above the focal block, preceded by one blank row — which
	// is the composer's own separator, because a statement is a block like any other and the blank row
	// between two blocks is the composer's rule. Nothing precedes it when the chain above is empty, so
	// then there is no blank row either: a frame has no leading separator.
	if c.PartialAbove {
		blocks = append(blocks, statementBlock(g, w, 0, g.Markers.PartialAbove, len(blocks) > 0))
	}
	// (3) The focal post, and then its own continuations with no separator between them.
	focal := base
	focal.focal = true
	blocks = append(blocks, threadBlock(focal, c.Focal, len(blocks) > 0))
	above := c.Focal.ID
	for i := range c.Continued {
		section := base
		section.continued = above
		blocks = append(blocks, threadBlock(section, c.Continued[i], false))
		above = c.Continued[i].ID
	}
	// (4) The replies, one block per post, every row at the reply step.
	for i := range c.Replies {
		section := reply
		section.replyTo = replyParent(c, i)
		blocks = append(blocks, threadBlock(section, c.Replies[i], true))
	}
	// (5) The statements, at the reply step. The end of the reply list prints nothing at all: the
	// upstream said there is nothing more, and the frame repeats that by saying nothing.
	switch {
	case c.PartialHidden:
		blocks = append(blocks, statementBlock(g, w, g.ReplyIndent, g.Markers.PartialHidden, true))
	case c.NoReplies && c.Read:
		// The empty section's two rows are one block: the statement, and Phase 2's empty-read body
		// row under it, in the words the style on screen uses.
		style, _ := CoverOfGrammar(g)
		blocks = append(blocks, statementPair(g, w, g.ReplyIndent, g.Markers.NoReplies, EmptyBody(style)))
	}
	if c.Stalled {
		blocks = append(blocks, statementBlock(g, w, g.ReplyIndent, g.Markers.PartialStopped, true))
	}
	// The first page in flight, at indent 0 below the focal block. It is drawn last because a frame
	// cannot both be waiting for its first page and have read a reply list: the two states do not
	// coexist, and one of them is always absent.
	if c.Loading {
		blocks = append(blocks, statementBlock(g, w, 0, g.Markers.LoadingThread, len(blocks) > 0))
	}
	return blocks
}

// threadSection is one block's own place in a conversation: the grammar, the pane's width, the row
// the cursor is on, the section's indent and reply level, and the three facts only a conversation has
// — whether this is the focal post, the record it continues, and the post it answers elsewhere.
type threadSection struct {
	grammar Grammar
	width   int
	sel     Selection
	// surface marks a conversation's rows, which differ from a list's in two ways the UI-SPEC's
	// visibility table states: a record whose content value carries no author and no stamp draws no
	// head row at all (there is nothing in it to print, and the state marker below it is the block's
	// whole content), and that marker stands at the block's own indent rather than at a body's.
	surface bool
	// indent is the reply step this block's rows carry, after the selection cell: 0 for the chain
	// above, the focal post and its continuations, and the grammar's ReplyIndent for a reply.
	indent int
	// reply is the measure's reply level, 0 or 1, which the indent follows.
	reply int
	// focal marks the post the frame was opened on: exactly one block of one conversation carries it.
	focal bool
	// continued is the post this block directly answers within the frame, and is non-empty only for
	// the focal post's own continuations: they carry `continued`, and no separator precedes them.
	continued domain.PostID
	// replyTo is the reply-elsewhere marker's subject when this block carries one: the post this
	// reply answers, which is neither the focal post nor the block directly above it.
	replyTo domain.Post
	// falls is the per-post URL fallback map for this compose pass.
	falls Fallbacks
}

// threadBlock is one post's block in a conversation: the rows the composer builds for it, and the
// post's own id so that the pane's row arithmetic can find the block (the shell's anchor, and the
// status line's position).
//
// separator asks for the blank row in front of the block: the composer writes it, because the
// separator is the composer's rule and a second copy of it here would be a second layout.
func threadBlock(s threadSection, post domain.Post, separator bool) Block {
	b := &builder{
		grammar: s.grammar,
		width:   s.width,
		pal:     For(RungFromEnv()),
		sel:     s.sel,
		indent:  s.indent,
		reply:   s.reply,
		surface: s.surface,
		falls:   s.falls,
	}
	rows := make([]string, 0, 8)
	if separator && s.grammar.Separator {
		rows = append(rows, "")
	}
	rows = append(rows, b.post(post, post.ID, nil, s.continued, "", s.focal, s.replyTo)...)
	return Block{PostID: post.ID, Rows: rows}
}

// statementBlock is one pane statement: a block like any other, so the frame composer pads and paints
// it, and never a post — its PostID is the zero id, which is what keeps the selection and the
// position arithmetic off it, and what makes it unselectable by construction.
//
// A statement is muted chrome and never accented: it is a statement of fact about the frame, not a
// failure state, and the status line's accent belongs to the state word alone.
func statementBlock(g Grammar, w, indent int, words string, separator bool) Block {
	return statementRows(g, w, indent, []string{words}, separator)
}

// statementPair is one statement block of two rows: the empty reply section's statement and the body
// row under it. They are one block rather than two so that no separator and no selection cell can
// come between them.
func statementPair(g Grammar, w, indent int, head, body string) Block {
	if body == "" {
		return statementBlock(g, w, indent, head, true)
	}
	return statementRows(g, w, indent, []string{head, body}, true)
}

// statementRows composes a statement's rows: every row at the statement's own indent, in the grammar's
// own body shape, so the words are measured, wrapped and painted by the one composer.
func statementRows(g Grammar, w, indent int, words []string, separator bool) Block {
	b := &builder{grammar: g, width: w, pal: For(RungFromEnv()), indent: indent}
	rows := make([]string, 0, len(words)+1)
	if separator && g.Separator {
		rows = append(rows, "")
	}
	for _, line := range words {
		for _, wrapped := range text.Wrap(line, g.measure(w, 0, 0)-indent) {
			rows = append(rows, b.bodyRow(" ", "", wrapped))
		}
	}
	return Block{Rows: rows}
}

// replyParent is the post a reply answers, when that parent is one the frame holds and is neither the
// focal post nor the block directly above this reply. It is the whole of the `reply to {handle}`
// marker's rule, and it is why the marker can name a handle: the parent post carries the author the
// marker is addressed to.
//
// "The block directly above it" is the frame's own block order: the previous reply, or — for the
// first reply — the focal post's last continuation, or the focal post itself when it has none.
//
// A reply whose parent is the focal post carries nothing: answering the block you are under is what
// the reply step already says, and a marker repeating it would be a second word for one fact. A reply
// whose parent the frame does not hold carries nothing either, and for the same reason the chain-cut
// statement exists: naming a handle needs the post that carries it, and inventing one would be a
// guess. The statement above the focal block is what speaks for a missing ancestor.
func replyParent(c Conversation, i int) domain.Post {
	reply := c.Replies[i]
	if reply.ReplyTo == "" || reply.ReplyTo == c.Focal.ID {
		return domain.Post{}
	}
	directlyAbove := c.Focal.ID
	if len(c.Continued) > 0 {
		directlyAbove = c.Continued[len(c.Continued)-1].ID
	}
	if i > 0 {
		directlyAbove = c.Replies[i-1].ID
	}
	if reply.ReplyTo == directlyAbove {
		return domain.Post{}
	}
	for _, posts := range [][]domain.Post{c.Ancestors, c.Continued, c.Replies} {
		for j := range posts {
			if posts[j].ID == reply.ReplyTo {
				return posts[j]
			}
		}
	}
	return domain.Post{}
}

// headless reports whether a record's content value carries nothing for a head row: no author and no
// stamp. It is a rule about what the value holds and not about tombstones — a tombstone that carries
// an author and a date draws its head row like any other record, which is what the UI-SPEC's
// visibility table says and what Phase 2's own frame draws. A headless record is drawn without a head
// row except when it is the frame's focal post: that one draws the grammar's own head row, which for
// the plain reader is the focal marker alone, because A1-T requires the frame to name the focal post
// even when its text is unavailable and the state marker below it cannot carry the marker.
func headless(p domain.Post) bool {
	return p.Author.ID == "" && p.Author.ScreenName == "" && p.CreatedAt.IsZero()
}

// bodyCells is the widest text a row of a conversation may carry at a quote depth and a reply level:
// the selection cell, the reply step, the body's own prefix and indent, two cells per quote level and
// the measure. It is the composer's own arithmetic as a value, so a test can assert the width
// property against the formula the composer used rather than against a second copy of it.
func bodyCells(g Grammar, w, depth, replyLevel int) int {
	depth = min(depth, g.Markers.QuoteDepthCap)
	return 1 + g.ReplyIndent*max(replyLevel, 0) + text.Width(g.BodyPrefix) + g.BodyIndent +
		2*depth + g.measure(w, depth, replyLevel)
}
