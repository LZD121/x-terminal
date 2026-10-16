package render

import (
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/text"
)

// StampCells is the width of the head row's time column, in cells. Both absolute forms are
// exactly this wide, so the column never moves.
const StampCells = 12

// QuoteBodyIndent is how far a quoted post's text sits behind its quote prefix, in cells: one
// nesting level of the spacing scale, and the same number under every grammar. It is deliberately
// not the grammar's own BodyIndent. A record's body sits where its style puts it — git's 4-cell
// message indent, a ci tail's `| ` gutter at indent 0, a log's 2-cell continuation — and the text
// that record quotes is still indented by these two cells behind the quote prefix, which is what
// the phase's worked renderings show and what the plain grammar's own 2-cell body indent happens to
// agree with.
const QuoteBodyIndent = 2

// headGap is the two cells a marker sits behind the field before it in a grammar whose marker
// templates do not carry their own separator: Phase 2's rule for the plain grammar's three
// markers, and the same two cells the status line puts between its fields.
const headGap = 2

// StampFormat renders an instant as an absolute form. There is no relative form and no relative
// word ("2h ago"): a frame has to be reproducible, and a relative stamp would make it a function of
// when it was read.
type StampFormat struct {
	// Clock is the layout for an instant in the clock's own year: `Jan 02 15:04`.
	Clock string
	// Date is the layout for every other year: `Jan 02  2006`.
	Date string
	// MonthYear is the layout the profile's `{joined}` placeholder renders through: `Jan 2006`.
	// A grammar whose clauses name no join date leaves it empty, and then the clause is omitted —
	// but a grammar that declares the clause and leaves this empty would declare a fact no frame of
	// it could ever print, so every shipped grammar that has the clause sets this.
	MonthYear string
	// Location is the zone a stamp is rendered in, or nil for the clock's own. A grammar that
	// names its zone sets it — a service log spells UTC, and the `Z` is part of the instant — while
	// a grammar whose zone is the reader's own leaves it nil. Either way a cover spells the same
	// instant the plain reader spells, in another zone's letters.
	Location *time.Location
	// At is the clock every stamp is rendered against: its year picks the layout and its location
	// renders the instant, so a stamp is in the reader's local time. A grammar carries no clock —
	// Blocks fills At from its own now argument — which is what makes a frame deterministic.
	At time.Time
}

// Stamp renders t as an absolute stamp in the format's own width: the clock form when t falls in the
// clock's own year, the dated form otherwise, in the format's zone. A form is exactly as wide as it
// is and is never padded to a column: the plain reader's column is 12 cells because its head
// template declares `{stamp:12}` and its own two forms are 12 cells wide, not because the format pads
// whatever it is given — a build log's `[09:12:00]` is eight cells and stays eight, and a git date is
// 28 and is never cut.
//
// The zero instant has no date — it is the absence of one, which is what a tombstone carries — and
// renders as StampCells of nothing rather than as a date the upstream never sent: a blank field is
// the truthful one, and StampCells is the width of the column the plain reader's own head declares.
func (f StampFormat) Stamp(t time.Time) string {
	if t.IsZero() {
		return strings.Repeat(" ", StampCells)
	}
	zone := f.Location
	if zone == nil {
		zone = f.At.Location()
	}
	local := t.In(zone)
	layout := f.Date
	if local.Year() == f.At.In(zone).Year() {
		layout = f.Clock
	}
	return local.Format(layout)
}

// MarkerPlace is where a grammar puts the markers of a post's relationships: appended to the head
// line they belong to, or one line each at the head's indent, or one line each at the body's own
// indent. It is one value for the whole run, because a style that writes a service log's `key=value`
// fields inline writes all of them inline, and a style that writes a table's detail lines writes all
// of them as lines.
type MarkerPlace uint8

const (
	// MarkersInline appends the run to the head's last line, where the head template's {markers}
	// stands for it.
	MarkersInline MarkerPlace = iota
	// MarkersHeadLines draws one line per marker at the head's own indent: a git header's
	// `Replayed-by:` line, a CI tail's `origin=`.
	MarkersHeadLines
	// MarkersBodyLines draws one line per marker at the body's own indent: an internal tool's
	// `from <owner>` detail line under its row.
	MarkersBodyLines
)

// MarkerSet is every marker literal of one grammar, as templates over the placeholders of copy.go.
// The selection mark is deliberately not here — the mark lives in the row and not in a literal, so
// a grammar cannot lose it — and neither is the truncation mark's role: it is the mark of any field
// a template cuts short.
//
// A slot that is the empty string is a marker the grammar does not have, and the composer draws
// nothing for it: a service log folds the limited state into its level word, a table renders it in
// its state column, and neither then prints a second marker about it.
type MarkerSet struct {
	// Repost is the provenance marker on the record of a reposted post: the original author is the
	// record's identity, and this names who surfaced it.
	Repost string
	// Continued marks a reply whose parent is the record directly above it.
	Continued string
	// Limited marks a post the upstream is holding back. Its body is still served and is printed.
	Limited string
	// Unavailable is the body of a tombstone: a post that used to exist.
	Unavailable string
	// MissingText is the body of a post that has no text, no media and no quote.
	MissingText string
	// Attachment is the attachment line's template, one counted noun per kind that is present: a
	// body line of its own under the plain grammar, one of the record's markers under a cover.
	Attachment string
	// Parent is the back-reference a quoted post carries when the grammar renders it as a record of
	// its own, and names the record that quotes it. It is empty in a grammar whose quote is a nested
	// block or row instead.
	Parent string
	// StateInfo, StateWarn and StateError are the three words a grammar has for a post's
	// visibility — a log level (`INFO`/`WARN`/`ERROR`) or a table's state column (`open`/`limited`/
	// `closed`) — and are empty in a grammar that renders the relation as a marker instead.
	StateInfo string
	// StateWarn is the word for a limited post.
	StateWarn string
	// StateError is the word for an unavailable or withheld post.
	StateError string
	// QuotePrefix is one quote level's prefix, repeated once per level at the content origin, and
	// capped at QuoteDepthCap levels. A cover writes its own: a nested block sits where its style
	// puts a nested block.
	QuotePrefix string
	// QuoteDepthCap is the deepest prefix a quote may carry: deeper content keeps the prefix it
	// has and its text is still rendered, so nothing is omitted and the measure cannot collapse.
	QuoteDepthCap int
	// Truncation is the mark for the last cell of a truncated field.
	Truncation string
	// The conversation's markers. They are slots like the rest, and a slot that is the empty string
	// is a marker the grammar does not have (the composer draws nothing for it), which is why every
	// shipped grammar sets all eight: a pane statement or a relationship marker without a literal
	// would render as a blank row, and TestTheMarkerSetsAreTotal is what holds that.
	//
	// Focal marks the post the frame was opened on: exactly one block of one conversation carries it.
	Focal string
	// ReplyTo marks a reply whose parent is neither the focal post nor the block directly above it.
	// Its template carries `{handle}`, which expands through Identity — the alias seam — so a cover
	// cannot leak a real handle through a marker.
	ReplyTo string
	// NoReplies is the empty reply section's statement: a read that succeeded and brought nothing.
	NoReplies string
	// LoadingThread is the conversation's first-page statement.
	LoadingThread string
	// PartialStopped, PartialAbove and PartialHidden are the three partial-thread conditions, each a
	// statement of fact: the reply walk stopped on a cursor the upstream would not advance, the post
	// above the focal post is not reachable, or replies are not reachable while the focal post's own
	// reply count says there are some. None of them is an error and none names a count.
	PartialStopped string
	// PartialAbove is the chain-cut statement, drawn directly above the focal block.
	PartialAbove string
	// PartialHidden is the third condition's statement, drawn at the reply step.
	PartialHidden string
	// Withheld is the body of a post the upstream will not serve in the reader's country. Phase 2's
	// marker table names Limited, Unavailable and MissingText; this row is added, because a state
	// without a literal renders as a blank row.
	Withheld string
	// PromptSearch is the prompt row's prefix for the posts product, as a template over
	// {product}: the product is named before the request is paid for. It is a marker and not a
	// constant so that a cover can rename the prompt without a code change.
	PromptSearch string
	// PromptPeople is the prompt row's prefix for the people product (`people:` under Plain).
	PromptPeople string
	// LoadingResults is the pane's own statement while a results frame's first page is out.
	LoadingResults string
	// LoadingProfile is the pane's own statement while a profile header is out.
	LoadingProfile string
	// UserRow is the user-row / profile-identity template when a grammar wants their shape to
	// differ from UserHead's. Empty means "use UserHead".
	UserRow string
	// NoAuthor is the notice for `a` where the selected row has no author.
	NoAuthor string
	// NoPosts is the notice for a profile whose posts read succeeded and carried none.
	NoPosts string
	// Sync is the synced field's word ("synced"); a cover may call it a cache.
	Sync string
	// SyncNever is the never-synced form ("never synced").
	SyncNever string
	// Gated is the notice written when a cover-breaking action is refused under a
	// cover. Empty in the plain grammar, where that notice is unreachable; every
	// cover grammar carries its own non-empty wording so a refused key is never a
	// blank status field.
	Gated string
}

// Grammar is one display style. The block layout, the width model and the reducers are shared by
// every grammar: a grammar decides what a record's head is made of, where its markers land, how far
// its body is indented, what its stamp spells and what it calls an account, and changes nothing
// about what is true about a post. There is one composer — Blocks — and a cover is a value handed
// to it, never a second rendering path.
type Grammar struct {
	// Name is the grammar's name, for a test's failure message rather than for a frame. Nothing in
	// the composer reads it: a style is its fields, not its name.
	Name string
	// Head is the record's head, line by line, as templates: the identity field, the stamp and the
	// marker run, laid out the way the style lays them out. A template line that is the empty
	// string is a blank row (git separates its header from the commit message with one), and one
	// whose only placeholder is empty is dropped.
	Head []string
	// QuoteHead is the head a nested quoted post uses, in a grammar that draws its quote as a nested
	// block or row. A grammar that renders the quoted post as a record of its own carries the same
	// templates as Head, because there the quoted post is simply a record.
	QuoteHead []string
	// QuoteAsRecord is whether the quoted post is a record of its own, drawn after the record that
	// quotes it at the same origin and carrying the back-reference in Parent, rather than a nested
	// block or row. The list order is untouched either way.
	QuoteAsRecord bool
	// MarkerPlace is where the marker run lands.
	MarkerPlace MarkerPlace
	// PaneHeader is the lines a table-style pane draws once as its first content rows: they scroll
	// with the content — a sticky row would move the pane's row arithmetic and the anchor for a
	// cosmetic gain — and never carry the selection mark.
	PaneHeader []string
	// BodyIndent is the number of cells a body row is indented by, after the selection column and
	// after BodyPrefix.
	BodyIndent int
	// BodyPrefix is a literal prefixed to every body row before its indent: a build log's `| `
	// gutter, which is what makes one record unmistakably one record.
	BodyPrefix string
	// AttachmentRow is whether a post's attachment is a body line of its own — Phase 2's
	// `attachment: 2 images`, drawn under the body — or one of the record's markers (` att=2
	// images`, `Files: 2 images`), drawn where MarkerPlace puts the run. It is its own field and not
	// a second reading of MarkerPlace because the plain grammar needs both at once: its head
	// markers are inline and its attachment is a row of the body.
	AttachmentRow bool
	// MeasureCap is the widest a body line may be, in cells, whatever the pane's width: the
	// measure is the typographic limit, not the layout's leftover.
	MeasureCap int
	// Separator is whether one entirely blank row separates two blocks.
	Separator bool
	// Stamp is the head's time column.
	Stamp StampFormat
	// Markers is every marker literal, as templates.
	Markers MarkerSet
	// Identity is the one seam between content and presentation that a display style replaces: the
	// plain reader's grammar prints the handle, and a cover prints whatever that cover calls the
	// account. It answers with all four of the alias's projections at once, because a style may
	// spell an identity in more than one field of the same head.
	Identity func(domain.User) CoverAlias
	// Ref is the cover's own reference token for a post — a log's sequence number, git's abbreviated
	// hash, a build number, a case id — and nil in the plain grammar, which has no such token.
	Ref func(domain.PostID) string
	// ShowCounts is whether the grammar prints a post's engagement numbers. It is false in every
	// grammar this phase ships — the list grammar prints no count at all (REND-01) — and it is
	// here because it is the switch a cover turns on.
	ShowCounts bool
	// ReplyIndent is how many cells every row of a reply block is indented by, after the selection
	// column and before the body's own prefix, indent and quote prefixes. It is the reply *section*
	// marker and not a computed tree depth: a reply group arrives as a flat ranked list, and there is
	// exactly one reply level, so the deepest legal row of any conversation is a quote inside a reply
	// — ReplyIndent plus two quote prefix steps, six cells, whatever the conversation's length.
	//
	// It is a field rather than a constant because a cover supplies its own record prefix or its own
	// number, and because the measure's reply term is written in terms of it.
	ReplyIndent int
	// UserHead is the user-row and profile-identity template: `{handle:16} {name}` under Plain.
	// One template for both surfaces, so a cover can render its own directory form without a code
	// change. Markers.UserRow overrides it when a grammar wants their shapes to differ.
	UserHead []string
	// AccountFacts is the facts line's clauses in fixed order, joined by two spaces. Each clause is
	// omitted when its value is absent, and a grammar with no clause list at all renders no facts
	// row — which is what a grammar that must not print audience numbers sets. A grammar that does
	// print them writes its own words for them: the plain reader's clauses are `followers:`,
	// `following:`, `posts:` and `joined:`, and a covered frame may hold none of the four, so every
	// cover carries its own (cover.go). A grammar that has no clauses of its own and is not the
	// plain reader's gets none rather than the plain reader's words (profile.go's profileFacts).
	AccountFacts []string
}

// PlainMarkers is the plain reader's marker set: every literal of the UI-SPEC's marker table,
// including the conversation's eight.
var PlainMarkers = MarkerSet{
	Repost:         MarkerRepost,
	Continued:      MarkerContinued,
	Limited:        MarkerLimited,
	Unavailable:    MarkerUnavailable,
	MissingText:    MarkerMissingText,
	Attachment:     CopyAttachment,
	QuotePrefix:    MarkerQuotePrefix,
	QuoteDepthCap:  2,
	Truncation:     MarkTruncation,
	Focal:          MarkerFocal,
	ReplyTo:        MarkerReplyTo,
	NoReplies:      MarkerNoReplies,
	LoadingThread:  MarkerLoadingThread,
	PartialStopped: MarkerPartialStopped,
	PartialAbove:   MarkerPartialAbove,
	PartialHidden:  MarkerPartialHidden,
	Withheld:       MarkerWithheld,
	PromptSearch:   MarkerPromptSearch,
	PromptPeople:   MarkerPromptPeople,
	LoadingResults: MarkerLoadingResults,
	LoadingProfile: MarkerLoadingProfile,
	NoAuthor:       MarkerNoAuthor,
	NoPosts:        MarkerNoPosts,
	Sync:           MarkerSync,
	SyncNever:      MarkerSyncNever,
}

// Plain is the plain reader's grammar: the handle is the identity, the stamp is local and
// absolute, the body indents by two cells, the attachment is a line of the body, the measure is
// capped at 96 cells and one blank row separates two blocks. It is the grammar Phase 2 ships and
// the zero Grammar is not it — a Grammar nobody filled in would render a frame with no head at all.
//
// Its head templates are the seam's own statement of Phase 2's post block: the identity column is
// 16 cells and the stamp 12, both declared by the template rather than by two fields of this
// struct, so one mechanism — the template and text.Pad — places both. The stored frames are
// byte-identical to Phase 2's, which is the change's own proof that the mechanism is the same one.
var Plain = Grammar{
	Name: "plain",
	Head: []string{
		"{handle:16} {stamp:12}{markers}",
	},
	QuoteHead: []string{
		"{handle:16} {stamp:12}{markers}",
	},
	UserHead: []string{
		"{handle:16} {name}",
	},
	AccountFacts: []string{
		"followers: {followers}",
		"following: {following}",
		"posts: {posts}",
		"joined: {joined}",
	},
	MarkerPlace:   MarkersInline,
	ReplyIndent:   2,
	BodyIndent:    2,
	AttachmentRow: true,
	MeasureCap:    96,
	Separator:     true,
	Stamp:         StampFormat{Clock: "Jan 02 15:04", Date: "Jan 02  2006", MonthYear: "Jan 2006"},
	Markers:       PlainMarkers,
	Identity: func(u domain.User) CoverAlias {
		return CoverAlias{Handle: u.ScreenName, Name: u.Name}
	},
	ShowCounts: false,
}

// withClock returns the grammar with its stamp format's clock set, which is how Blocks turns its
// now argument into the year and the location every stamp in the frame is rendered against.
func (g Grammar) withClock(now time.Time) Grammar {
	g.Stamp.At = now
	return g
}

// state is the grammar's word for a post's visibility: the level a log style prints in its head
// line, the state a table prints in its column, and nothing at all in a grammar that renders the
// relation as one of its markers instead.
func (g Grammar) state(v domain.Visibility) string {
	switch v {
	case domain.Limited:
		return g.Markers.StateWarn
	case domain.Unavailable, domain.Withheld:
		return g.Markers.StateError
	}
	return g.Markers.StateInfo
}

// measure is the body measure at a quote depth and a reply level: the pane minus the selection cell,
// the body's own prefix and indent, two cells per quote level and the reply section's own step, capped
// at the grammar's measure. The plain grammar's own values (indent 2, no prefix, ReplyIndent 2) give
// Phase 2's min(W - 3 - 2·depth, 96) exactly when the row is not in a reply, which is the regression
// the stored frames check and TestThePlainMeasureIsPhaseTwos pins.
//
// depth is clamped to the grammar's own QuoteDepthCap, because the prefix stops growing at the cap and
// the measure has to stop with it: a measure that kept shrinking while the prefix did not would cut
// text off a row the prefix still had room for.
func (g Grammar) measure(w, depth, replyLevel int) int {
	if depth > g.Markers.QuoteDepthCap {
		depth = g.Markers.QuoteDepthCap
	}
	measure := w - 1 - g.BodyIndent - 2*depth - text.Width(g.BodyPrefix) - g.ReplyIndent*replyLevel
	if measure > g.MeasureCap {
		measure = g.MeasureCap
	}
	return measure
}
