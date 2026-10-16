package render

import (
	"time"

	"github.com/LZD121/x-terminal/internal/domain"
)

// This file is the cover half of the phase's visible surface, as values: four display styles, each
// one Grammar the one composer draws, with every literal here or in copy.go. Nothing in it renders
// anything — there is no per-style function, no column sized from content, and no field of a cover
// derived from the wall clock, a counter, a map's order or a random source. A post renders the same
// bytes on every run, which is what lets the stored frames pin a cover exactly.
//
// The four styles each name the same content differently: a service's structured log tailed in a
// terminal, a git log, a CI job stream, and an internal request queue. What they have in common is
// what they may not change — the text, the order, the page boundaries, the cursors, the selection,
// the status line's row and field order, and the identity of every account, which is a cover alias
// and never a handle that exists on the upstream.

// The four styles' ids: the values `display.cover` carries in the config file, the id the `keys`
// subcommand's help names, and the value the cycle key steps through. The zero CoverStyle is the
// plain reader.
const (
	// CoverServiceLog is a service's structured application log, tailed in a terminal.
	CoverServiceLog CoverStyle = "servicelog"
	// CoverGitLog is `git log` of an internal repository.
	CoverGitLog CoverStyle = "gitlog"
	// CoverCI is a CI job-log stream, several builds concatenated.
	CoverCI CoverStyle = "ci"
	// CoverInternal is an internal request queue with a per-row detail line.
	CoverInternal CoverStyle = "internal"
)

// The titles. Every state's title names what is on screen — the style's own name while a cover is
// worn, `x-terminal` for the plain reader and `make` for the panic frame (panic.go) — so no state's
// title ever lies about its frame, and no covered title names the product.
const (
	// CoverTitleServiceLog is the servicelog style's title and tmux window name.
	CoverTitleServiceLog = "svc-log"
	// CoverTitleGitLog is the gitlog style's title and tmux window name.
	CoverTitleGitLog = "git"
	// CoverTitleCI is the ci style's title and tmux window name.
	CoverTitleCI = "pipeline"
	// CoverTitleInternal is the internal style's title and tmux window name.
	CoverTitleInternal = "ops-console"
)

// The servicelog literals: a structured log line whose level is a rename of the post's visibility,
// whose identity is the account's machine name, and whose reference is a sequence number. Every
// inline marker carries its own leading separator, and a level word carries the limited state, so
// the style prints no second marker about it.
const (
	CoverServiceLogHead       = "{stamp} {state}  svc={handle} seq={ref}{markers}"
	CoverServiceLogReplayedBy = " replayed_by={handle}"
	CoverServiceLogFollows    = " follows={ref}"
	CoverServiceLogParent     = " parent={ref}"
	CoverServiceLogAttachment = "[{n}] {noun}"
	CoverServiceLogWithdrawn  = "record withdrawn"
	CoverServiceLogNoPayload  = "no payload"
	CoverServiceLogLevelInfo  = "INFO"
	CoverServiceLogLevelWarn  = "WARN"
	CoverServiceLogLevelError = "ERROR"
)

// The gitlog literals: git's own header, in git's own order, with a blank row between the header
// and the message, the message indented by four cells, and a commit's hash as its reference. The
// quoted post is a `>` attribution inside the message, which is the convention every developer
// already reads as "this message quotes another".
const (
	CoverGitCommit     = "commit {ref}"
	CoverGitAuthor     = "Author: {name} <{email}>"
	CoverGitDate       = "Date:   {stamp}"
	CoverGitQuoteHead  = "{name}  {stamp}"
	CoverGitReplayedBy = "Replayed-by: {name}"
	CoverGitContinues  = "Continued-from: {ref}"
	CoverGitRestricted = "Visibility: restricted"
	CoverGitFiles      = "[{n}] {noun}"
	CoverGitWithdrawn  = "record withdrawn"
	CoverGitNoMessage  = "(no message)"
	CoverGitQuoteMark  = "    > "
)

// The ci literals: one build's log lines, a `| ` gutter on every body line, and the quoted post as
// a nested block inside the record. The gutter is what makes one record unmistakably one record,
// and it is the reason this style is not confusable with servicelog even though both carry a clock.
const (
	CoverCIHead        = "[{stamp}] build={ref} job={handle}{markers}"
	CoverCIQuoteHead   = "[{stamp}] build={ref} job={handle}"
	CoverCIOrigin      = " origin={handle}"
	CoverCIFollows     = " follows={ref}"
	CoverCIPartial     = " note=partial access"
	CoverCIArtifacts   = "[{n}] {noun}"
	CoverCIWithdrawn   = "no output captured"
	CoverCINoOutput    = "(no output)"
	CoverCIGutter      = "| "
	CoverCIQuotePrefix = "| > "
)

// The internal literals: an aligned table of requests, the header row drawn once at the top of the
// pane and scrolled with it, the owner column carrying the account's login, and the post's detail
// prose indented under its row. The columns are fixed at 8, 2, 7, 2, 20, 2 and 12 cells — 53 before
// the stamp — and none of them is sized from the content.
const (
	CoverInternalHead       = "{ref:8}  {state:7}  {handle:20}  {stamp}"
	CoverInternalPaneHeader = "id        state    owner                 updated"
	CoverInternalFrom       = "from {handle}"
	CoverInternalFollows    = "follows {ref}"
	CoverInternalFiles      = "[{n}] {noun}"
	CoverInternalWithdrawn  = "record withdrawn"
	CoverInternalNoText     = "no text"
	CoverInternalOpen       = "open"
	CoverInternalLimited    = "limited"
	CoverInternalClosed     = "closed"
)

// The reference tokens. Each is one purpose string of coverDigest over the post's own id, so a post
// keeps the same token in every style that renders it and a restart cannot change it. The values
// are the ranges a log, a repository and a build system really show: a seven-digit sequence
// number, twelve hex digits of a commit, and a four-digit build or request number.
const (
	// CoverSeqBase and CoverSeqSpan put a sequence number in [1000000, 9999999].
	CoverSeqBase = 1000000
	CoverSeqSpan = 9000000
	// CoverBuildBase and CoverBuildSpan put a build or request number in [1000, 9999].
	CoverBuildBase = 1000
	CoverBuildSpan = 9000
	// CoverRequestPrefix is what an internal queue puts in front of a request's number.
	CoverRequestPrefix = "REQ-"
	// CoverSHACells is how many hex digits git abbreviates a commit to.
	CoverSHACells = 12
)

// The gated-refusal notices (REND-07 / DSGT-10). Each cover speaks in its own
// voice; none names the product, a purpose, a handle, a program or the way out.
const (
	CoverServiceLogGated = "open denied by local policy"
	CoverGitGated        = "operation refused"
	CoverCIGated         = "step skipped: policy"
	CoverInternalGated   = "request denied"
)

// The four styles' facts rows (Grammar.AccountFacts): the account's own numbers and its join date,
// in the words each style uses for them. The plain reader's clauses are `followers: {followers}`,
// `following: {following}`, `posts: {posts}` and `joined: {joined}` — a covered frame may hold none
// of those four labels, and `posts` is the product's own word besides — so each style carries its
// own row. The clauses are the same four facts in the same order under every style, and each is
// omitted when the value it names is absent, exactly as the plain reader's are. The fourth clause
// renders through the style's own Stamp.MonthYear, so every style below sets one: a clause whose
// placeholder no form can render is a declared fact the frame never shows.
var (
	// coverServiceLogFacts is a structured log's fields: key=value, the shape every other field of
	// its head takes.
	coverServiceLogFacts = []string{
		"watchers={followers}",
		"watching={following}",
		"records={posts}",
		"since={joined}",
	}
	// coverGitFacts is git's own prose: a labelled line per fact, the shape of its `Author:` line.
	coverGitFacts = []string{
		"Watchers: {followers}",
		"Watching: {following}",
		"Commits: {posts}",
		"Since: {joined}",
	}
	// coverCIFacts is a build log's bracketed summary: the same shape its attachment lines
	// (`[3] images`) and its head (`[15:04:05] build=...`) carry.
	coverCIFacts = []string{
		"[{followers} watchers]",
		"[{following} watching]",
		"[{posts} records]",
		"[since {joined}]",
	}
	// coverInternalFacts is a request queue's detail lines, the shape its `from {handle}` and
	// `follows {ref}` rows carry: the field's name and then its value.
	coverInternalFacts = []string{
		"watchers {followers}",
		"watching {following}",
		"requests {posts}",
		"since {joined}",
	}
)

// The styles' stamps. Every stamp names its zone when it is not the plain grammar's local time, so
// a cover never shows a different instant from the plain reader — only a different spelling of it.
//
// Every one of them sets MonthYear as well as its two head forms: that is the form the facts row's
// fourth clause renders through, so a style that left it empty would declare a clause its own frames
// could never print (each style's is the same month the plain reader's `joined:` clause shows, in
// the style's own spelling).
var (
	// coverISOStamp is the servicelog stamp: the instant the plain reader spells locally, spelled in
	// UTC with the `Z` explicit, width for width.
	coverISOStamp = StampFormat{
		Clock: "2006-01-02T15:04:05Z", Date: "2006-01-02T15:04:05Z",
		MonthYear: "2006-01", Location: time.UTC,
	}
	// coverGitStamp is git's own default date format, in the reader's zone, with the offset spelled:
	// `Mon Jan 2 15:04:05 2006 -0700` for the head, and git's own `Jan 2006` for a month.
	coverGitStamp = StampFormat{
		Clock: "Mon Jan 2 15:04:05 2006 -0700", Date: "Mon Jan 2 15:04:05 2006 -0700",
		MonthYear: "Jan 2006",
	}
	// coverClockStamp is a build log's clock time: the same clock the plain grammar shows, with the
	// seconds a pipeline's log carries. Its month is a pipeline's own dated stamp, `2006-01`.
	coverClockStamp = StampFormat{Clock: "15:04:05", Date: "15:04:05", MonthYear: "2006-01"}
	// coverTableStamp is a queue row's updated column: the plain reader's own dated form, in the
	// reader's own zone, and the same spelling with no day for the queue's join column.
	coverTableStamp = StampFormat{Clock: "Jan 02 15:04", Date: "Jan 02 15:04", MonthYear: "Jan 2006"}
)

// ServiceLog is the servicelog cover's grammar. The quoted post is a record of its own, carrying the
// back-reference to the record that quotes it (QuoteAsRecord), the body is a two-cell continuation,
// and the level word in the head is a rename of the post's visibility — which is why the limited
// state needs no second marker.
var ServiceLog = Grammar{
	Name:          string(CoverServiceLog),
	Head:          []string{CoverServiceLogHead},
	QuoteHead:     []string{CoverServiceLogHead},
	QuoteAsRecord: true,
	MarkerPlace:   MarkersInline,
	ReplyIndent:   2,
	BodyIndent:    2,
	MeasureCap:    96,
	Separator:     false,
	Stamp:         coverISOStamp,
	Markers: MarkerSet{
		Repost:         CoverServiceLogReplayedBy,
		Continued:      CoverServiceLogFollows,
		Parent:         CoverServiceLogParent,
		Unavailable:    CoverServiceLogWithdrawn,
		MissingText:    CoverServiceLogNoPayload,
		Attachment:     CoverServiceLogAttachment,
		StateInfo:      CoverServiceLogLevelInfo,
		StateWarn:      CoverServiceLogLevelWarn,
		StateError:     CoverServiceLogLevelError,
		QuotePrefix:    MarkerQuotePrefix,
		QuoteDepthCap:  2,
		Truncation:     MarkTruncation,
		Focal:          CoverServiceLogFocal,
		ReplyTo:        CoverServiceLogReplyTo,
		NoReplies:      CoverServiceLogNoReplies,
		LoadingThread:  CoverServiceLogLoading,
		PartialStopped: CoverServiceLogStopped,
		PartialAbove:   CoverServiceLogAbove,
		PartialHidden:  CoverServiceLogHidden,
		Withheld:       CoverServiceLogWithdrawn,
		PromptSearch:   CoverServiceLogPromptSearch,
		PromptPeople:   CoverServiceLogPromptPeople,
		LoadingResults: CoverServiceLogLoadingResults,
		LoadingProfile: CoverServiceLogLoadingProfile,
		NoAuthor:       CoverServiceLogNoAuthor,
		Sync:           CoverSyncCache,
		SyncNever:      CoverSyncNever,
		NoPosts:        CoverServiceLogNoPosts,
		Gated:          CoverServiceLogGated,
	},
	AccountFacts: coverServiceLogFacts,
	Identity:     func(u domain.User) CoverAlias { return CoverAliasOf(CoverServiceLog, u) },
	Ref:          serviceLogSeq,
}

// GitLog is the gitlog cover's grammar. The head is git's header — the abbreviated commit, the
// author and the date, then the blank row git leaves before the message — the message is indented
// by four cells the way git indents it, one blank row separates two commits, and the quoted post is
// a `>` attribution inside the message.
var GitLog = Grammar{
	Name: string(CoverGitLog),
	Head: []string{
		CoverGitCommit,
		CoverGitAuthor,
		CoverGitDate,
		"",
	},
	QuoteHead:   []string{CoverGitQuoteHead},
	MarkerPlace: MarkersHeadLines,
	ReplyIndent: 2,
	BodyIndent:  4,
	MeasureCap:  96,
	Separator:   true,
	Stamp:       coverGitStamp,
	Markers: MarkerSet{
		Repost:         CoverGitReplayedBy,
		Continued:      CoverGitContinues,
		Limited:        CoverGitRestricted,
		Unavailable:    CoverGitWithdrawn,
		MissingText:    CoverGitNoMessage,
		Attachment:     CoverGitFiles,
		QuotePrefix:    CoverGitQuoteMark,
		QuoteDepthCap:  2,
		Truncation:     MarkTruncation,
		Focal:          CoverGitFocal,
		ReplyTo:        CoverGitReplyTo,
		NoReplies:      CoverGitNoReplies,
		LoadingThread:  CoverGitLoading,
		PartialStopped: CoverGitStopped,
		PartialAbove:   CoverGitAbove,
		PartialHidden:  CoverGitHidden,
		Withheld:       CoverGitWithdrawn,
		PromptSearch:   CoverGitPromptSearch,
		PromptPeople:   CoverGitPromptPeople,
		LoadingResults: CoverGitLoadingResults,
		LoadingProfile: CoverGitLoadingProfile,
		NoAuthor:       CoverGitNoAuthor,
		Sync:           CoverSyncCache,
		SyncNever:      CoverSyncNever,
		NoPosts:        CoverGitNoPosts,
		Gated:          CoverGitGated,
	},
	AccountFacts: coverGitFacts,
	Identity:     func(u domain.User) CoverAlias { return CoverAliasOf(CoverGitLog, u) },
	Ref:          gitLogSHA,
}

// CI is the ci cover's grammar. Every body line carries a `| ` gutter, the head is one bracketed
// clock time with the build and the job, one record follows another with no blank row between them,
// and the quoted post is a nested block inside the record.
var CI = Grammar{
	Name:        string(CoverCI),
	Head:        []string{CoverCIHead},
	QuoteHead:   []string{CoverCIQuoteHead},
	MarkerPlace: MarkersInline,
	ReplyIndent: 2,
	BodyPrefix:  CoverCIGutter,
	MeasureCap:  96,
	Separator:   false,
	Stamp:       coverClockStamp,
	Markers: MarkerSet{
		Repost:         CoverCIOrigin,
		Continued:      CoverCIFollows,
		Limited:        CoverCIPartial,
		Unavailable:    CoverCIWithdrawn,
		MissingText:    CoverCINoOutput,
		Attachment:     CoverCIArtifacts,
		QuotePrefix:    CoverCIQuotePrefix,
		QuoteDepthCap:  2,
		Truncation:     MarkTruncation,
		Focal:          CoverCIFocal,
		ReplyTo:        CoverCIReplyTo,
		NoReplies:      CoverCINoReplies,
		LoadingThread:  CoverCILoading,
		PartialStopped: CoverCIStopped,
		PartialAbove:   CoverCIAbove,
		PartialHidden:  CoverCIHidden,
		Withheld:       CoverCIWithdrawn,
		PromptSearch:   CoverCIPromptSearch,
		PromptPeople:   CoverCIPromptPeople,
		LoadingResults: CoverCILoadingResults,
		LoadingProfile: CoverCILoadingProfile,
		NoAuthor:       CoverCINoAuthor,
		Sync:           CoverSyncCache,
		SyncNever:      CoverSyncNever,
		NoPosts:        CoverCINoPosts,
		Gated:          CoverCIGated,
	},
	AccountFacts: coverCIFacts,
	Identity:     func(u domain.User) CoverAlias { return CoverAliasOf(CoverCI, u) },
	Ref:          ciBuild,
}

// Internal is the internal cover's grammar. It is a table: a header row drawn once at the top of the
// pane, then one row per request with its state, its owner's login and its updated column, and the
// post's detail prose indented under the row it belongs to. The state column is where the post's
// visibility is renamed (`open`, `limited`, `closed`), and the markers are detail lines rather than
// fields of the row.
var Internal = Grammar{
	Name:        string(CoverInternal),
	Head:        []string{CoverInternalHead},
	QuoteHead:   []string{CoverInternalHead},
	MarkerPlace: MarkersBodyLines,
	ReplyIndent: 2,
	PaneHeader:  []string{CoverInternalPaneHeader},
	BodyIndent:  2,
	MeasureCap:  96,
	Separator:   false,
	Stamp:       coverTableStamp,
	Markers: MarkerSet{
		Repost:         CoverInternalFrom,
		Continued:      CoverInternalFollows,
		Unavailable:    CoverInternalWithdrawn,
		MissingText:    CoverInternalNoText,
		Attachment:     CoverInternalFiles,
		StateInfo:      CoverInternalOpen,
		StateWarn:      CoverInternalLimited,
		StateError:     CoverInternalClosed,
		QuotePrefix:    MarkerQuotePrefix,
		QuoteDepthCap:  2,
		Truncation:     MarkTruncation,
		Focal:          CoverInternalFocal,
		ReplyTo:        CoverInternalReplyTo,
		NoReplies:      CoverInternalNoReplies,
		LoadingThread:  CoverInternalLoading,
		PartialStopped: CoverInternalStopped,
		PartialAbove:   CoverInternalAbove,
		PartialHidden:  CoverInternalHidden,
		Withheld:       CoverInternalWithdrawn,
		PromptSearch:   CoverInternalPromptSearch,
		PromptPeople:   CoverInternalPromptPeople,
		LoadingResults: CoverInternalLoadingResults,
		LoadingProfile: CoverInternalLoadingProfile,
		NoAuthor:       CoverInternalNoAuthor,
		Sync:           CoverSyncCache,
		SyncNever:      CoverSyncNever,
		NoPosts:        CoverInternalNoPosts,
		Gated:          CoverInternalGated,
	},
	AccountFacts: coverInternalFacts,
	Identity: func(u domain.User) CoverAlias {
		// The owner column holds what a tool's owner column holds: the login, not the display name
		// and not the machine name.
		alias := CoverAliasOf(CoverInternal, u)
		alias.Handle = alias.Login
		return alias
	},
	Ref: internalRequest,
}

// CoverStyles is the four styles, in the frozen order the cycle key steps through them and the
// `cover 1..4` keys number them. The list is built here rather than read out of a map, so the order
// is a fact about the program and not about a hash table.
func CoverStyles() []CoverStyle {
	return []CoverStyle{CoverServiceLog, CoverGitLog, CoverCI, CoverInternal}
}

// CoverStyleOf reads a style id, and reports whether it is one at all. An unknown id — including the
// empty string, which is the plain reader and not a cover — is "not a style", so a caller keeps the
// default it already had rather than guessing at a style it has never heard of.
func CoverStyleOf(id string) (CoverStyle, bool) {
	for _, style := range CoverStyles() {
		if string(style) == id {
			return style, true
		}
	}
	return "", false
}

// Title is the terminal title (and tmux window name) this style is worn under: the style's own
// name, because the title names what is on screen. The zero style is the plain reader, whose title
// is the program's own name.
func (s CoverStyle) Title() string {
	switch s {
	case CoverServiceLog:
		return CoverTitleServiceLog
	case CoverGitLog:
		return CoverTitleGitLog
	case CoverCI:
		return CoverTitleCI
	case CoverInternal:
		return CoverTitleInternal
	}
	return CopyTitlePlain
}

// Grammar is the style's display style, and the whole of what wearing a cover changes: the one
// composer is handed this value instead of Plain. The zero style is the plain reader.
//
// The four styles are compared in one table in cover_test.go rather than branched on anywhere, and
// that is the phase's central structural claim: a style is a value, and there is no second rendering
// path to keep in step with the first.
func (s CoverStyle) Grammar() Grammar {
	switch s {
	case CoverServiceLog:
		return ServiceLog
	case CoverGitLog:
		return GitLog
	case CoverCI:
		return CI
	case CoverInternal:
		return Internal
	}
	return Plain
}

// serviceLogSeq is a servicelog record's sequence number: seven digits, the shape a tailed log
// prints, computed from the post's own id so the same post keeps the same number for as long as the
// account exists.
func serviceLogSeq(id domain.PostID) string {
	return coverReference(CoverSeqBase, CoverSeqSpan, coverPurposeSeq, string(id))
}

// ciBuild is a build log record's number, and internalRequest the same number in the shape a
// request queue prints it. One purpose string for both, because the two styles show one field: the
// phase's own renderings carry `build=4655` and `REQ-4655` for one post, and a record whose number
// changed when the style changed would not be a record.
func ciBuild(id domain.PostID) string {
	return coverReference(CoverBuildBase, CoverBuildSpan, coverPurposeReq, string(id))
}

// internalRequest is an internal queue row's request id: the build number with the queue's own
// prefix on it.
func internalRequest(id domain.PostID) string {
	return CoverRequestPrefix + ciBuild(id)
}

// gitLogSHA is a git record's abbreviated commit hash: the first twelve hex digits of the
// fingerprint, which is the digest's own leading twelve digits and not a second hash of anything.
func gitLogSHA(id domain.PostID) string {
	const hexDigits = "0123456789abcdef"
	value := coverDigest(coverPurposeSHA, string(id)) >> 16
	out := make([]byte, CoverSHACells)
	for i := range len(out) {
		out[len(out)-1-i] = hexDigits[value&0xf]
		value >>= 4
	}
	return string(out)
}

// CoverOfGrammar is the style a grammar value is, and false for the plain reader's own grammar and for
// a grammar that is no style this program ships (a test's own, say). A grammar is a value and carries
// no style id, so the mapping is by name — the one field a style carries for a failure message — and it
// is what lets the conversation's composer reach the words that are functions of the style rather than
// fields of it, like the empty read's body row.
func CoverOfGrammar(g Grammar) (CoverStyle, bool) {
	for _, style := range CoverStyles() {
		if style.Grammar().Name == g.Name {
			return style, true
		}
	}
	return "", false
}
