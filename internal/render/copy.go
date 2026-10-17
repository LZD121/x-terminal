// Package render is the phase's view: one plain grammar, the post-block layout, the status line,
// the help table and the frame composer, with every user-visible string in this one file and the
// colour ladder behind one rung value.
//
// The package reads domain values and measures through internal/text, and it does nothing else:
// no I/O, no clock of its own, no upstream JSON, no string measured by any call other than the
// four in internal/text. It imports internal/domain and internal/text, and nothing else of this
// module.
//
// Content and presentation are split on purpose. A domain value owns what is true about a post —
// its id, its author, its text, its instant, its relationships, its media kinds — and every
// character the user sees is decided here. One Grammar value is one display style: the plain
// reader's grammar is Plain, and a cover grammar is another Grammar value, not another renderer.
//
// Painting. Rows are composed plain and coloured per run, because internal/text wraps and cuts
// text while internal/text.Width counts an escape sequence as zero cells: a wrap point may not be
// decided on a coloured string. A run's colour comes from the palette (palette.go), whose rung is
// read from the environment at the moment the row is built. That is one decision and not a
// per-frame one — the environment of a process does not change under it — and it is never read
// from the config, never from a terminal query and never re-decided from anything the session
// did. A test that compares a row to the literal copy pins the environment (TERM=dumb and
// NO_COLOR=1), and a test that cares about colour strips the escapes, which is the property every
// colour assertion in this package is built on.
package render

import (
	"strconv"
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
)

// Every string the user can see, and the placeholders the templates below use: {n} a count, {m}
// the rate-limited count, {noun} the noun a counted media kind takes, {handle} a handle, {clock}
// a local wall-clock time, {date} a local dated time, {selected} and {loaded} the position's two
// numbers. No call site builds a sentence by concatenation except through one of these.
const (
	// The empty state. The heading names the feed, so there is one per feed.
	CopyEmptyHeadingFollowing = "no posts in following"
	CopyEmptyHeadingForYou    = "no posts in for you"
	// CopyEmptyBody is the second line of the empty state. It says what happened (the read
	// succeeded) and what to do about it, because an empty timeline and a failed read look the
	// same on screen and only one of them is true.
	CopyEmptyBody = "this is an empty read that succeeded - press r to refresh"

	// Loading, first page: one line per feed.
	CopyLoadingFollowing = "loading following..."
	CopyLoadingForYou    = "loading for you..."

	// CopyLoadingSaved is a saved frame's first line while its mirror page is out. The frame's list
	// is the status line's own field, so the pane's line is the state's, one line for both lists.
	CopyLoadingSaved = "loading..."

	// CopyPaneTooNarrow is the pane's one narrow line, below MinPaneCols.
	CopyPaneTooNarrow = "widen the pane to at least 60 columns"

	// CopyHelpClipped is the help overlay's cut row: how many of the keymap's rows the pane cannot
	// hold, said in the pane rather than left for the reader to notice. The overlay has no scroll of
	// its own, so the count is the whole of what it can honestly offer.
	CopyHelpClipped = "{n} more keys than the pane can show"

	// MarkTruncation is the truncation mark: it goes in the last cell a truncated field may
	// occupy, and internal/text.Truncate appends nothing on its own, so a cut field can be told
	// from a naturally short one.
	MarkTruncation = "~"
	// MarkSelection is the selection mark, in column 1 of every row of the selected block. It is
	// what carries the selection; the colour only repeats it.
	MarkSelection = ">"

	// The markers of the plain grammar. Repost is the only one with a placeholder.
	MarkerRepost      = "reposted by {handle}"
	MarkerLimited     = "limited"
	MarkerContinued   = "continued"
	MarkerUnavailable = "post unavailable"
	MarkerMissingText = "post text missing"
	// MarkerQuotePrefix is one quote level's prefix: repeated once per level, at the content
	// origin, and capped at MarkerSet.QuoteDepthCap levels.
	MarkerQuotePrefix = "> "

	// The attachment line is one numbered entry per attachment: `[{n}] {noun}`
	// rendered once per entry and joined by CopyAttachmentSeparator when nothing
	// failed. The `attachment: ` prefix is gone; every entry is singular.
	CopyAttachment          = "[{n}] {noun}"
	CopyAttachmentSeparator = ", "
	NounImage               = "image"
	NounImages              = "images"
	NounVideo               = "video"
	NounVideos              = "videos"
	NounGIF                 = "gif"
	NounGIFs                = "gifs"

	// The status line's vocabulary. The budget line is singular at one request.
	CopyBudgetOne     = "{n} request, {m} rate-limited"
	CopyBudgetMany    = "{n} requests, {m} rate-limited"
	CopyListFollowing = "following"
	CopyListForYou    = "for you"
	CopyPosition      = "{selected}/{loaded}"
	// CopyPositionMore is the position while a next cursor is stored: the pane never implies
	// there is more than the cursor says.
	CopyPositionMore       = "{selected}/{loaded}+"
	CopyNoticeNewPost      = "{n} new post"
	CopyNoticeNewPosts     = "{n} new posts"
	CopyNoticeNoNewPosts   = "no new posts"
	CopyNoticeLoadingOlder = "loading older posts"
	CopyNoticeStalled      = "no older posts (cursor stalled)"
	// CopyNoticeNoResults is a search that read successfully and carried no rows: a successful
	// outcome, said in the notice field, and never a failure's words. CopyNoticeNothingToSwitch is
	// the one honest line `s` reports where there is no order to switch.
	CopyNoticeNoResults       = "no results"
	CopyNoticeNothingToSwitch = "nothing to switch"
	// CopyListSearch is the status line's list field for a results frame: the product and the
	// query, named once, quoted. The echo is capped at ListLabelQueryCells before it gets here.
	CopyListSearch = "search: {product} \"{query}\""
	// CopyListPeople is the status line's list field for a people results frame.
	CopyListPeople = "people: \"{query}\""
	// CopyListAccounts is the status line's list field for one account list: which list, and whose.
	// The list's own word comes from domain.UserListKind, so the three can never disagree with the
	// operation they name.
	CopyListAccounts = "{list}: {handle}"
	// CopyListMedia is the status line's list field for a profile's media tab.
	CopyListMedia = "media: {handle}"
	// MarkerPromptSearch is the prompt row's prefix for the posts product, as a template: the
	// product is named before the request is paid for. MarkerPromptPeople is the people
	// product's own prefix. MarkerLoadingResults is the pane's own statement while a results
	// frame's first page is out.
	MarkerPromptSearch   = "search: {product}"
	MarkerPromptPeople   = "people:"
	MarkerLoadingResults = "loading results..."
	// Profile surface literals: the pane's loading statement, the facts clauses, the list field,
	// and the notices a profile frame carries.
	MarkerLoadingProfile   = "loading profile..."
	MarkerNoAuthor         = "no author to open"
	MarkerNoPosts          = "no posts"
	CopyNoticeLoadingPosts = "loading posts"

	// Phase 6: saved lists, sync notices, capability states.
	CopyEmptyHeadingBookmarks = "no posts in bookmarks"
	CopyEmptyBodyNeverSynced  = "nothing is mirrored for {list} yet - press r to sync it"
	CopyEmptyLikesHedged      = "no likes, or the list is no longer served"
	CopyListBookmarks         = "bookmarks"
	CopyListLikes             = "likes"
	CopySyncing               = "syncing, {n} new"
	CopyNoticeNoLongerSaved   = "{n} no longer saved"
	CopyNoticeMirrorComplete  = "mirror complete"
	CopyNoticeSyncStalled     = "sync stopped (cursor stalled)"
	CopyCapUnavailable        = "{list} unavailable"
	CopyCapWrongAccount       = "wrong account"
	CopyCapMirrorUnreadable   = "mirror unreadable"
	CopyCapRemedyPin          = "doctor --pin, then restart"
	CopyCapRemedyRepoint      = "delete saved.db to re-point it"
	CopyCapRemedyAside        = "move saved.db aside, then sync"
	CopyCapBodyMissing        = "no operation id for {list} in this build - pin one with doctor --pin, then restart"
	CopyCapBodyWrongAccount   = "this mirror was synced from @{owner} - delete {path} to re-point it, or keep reading it offline"
	CopyCapBodyUnreadable     = "the mirror at {path} could not be opened - move it aside and read this list once online to start a new one"
	MarkerSync                = "synced"
	MarkerSyncNever           = "never synced"
	CoverSyncCache            = "cached"
	CoverSyncNever            = "never cached"
	// Covered list names for the saved surfaces: the same two lists, in each
	// style's own vocabulary, so a covered bookmarks/likes frame never prints
	// the product words `bookmarks` / `likes` (DSGT-08 / Phase 8 corpus).
	CoverServiceLogListBookmarks = "stream=saved"
	CoverServiceLogListLikes     = "stream=fav"
	CoverGitListBookmarks        = "ref=saved"
	CoverGitListLikes            = "ref=starred"
	CoverCIListBookmarks         = "queue=saved"
	CoverCIListLikes             = "queue=starred"
	CoverInternalListBookmarks   = "view=saved"
	CoverInternalListLikes       = "view=starred"

	// The covered saved surface: the capability block's three states, the empty read's own
	// sentence, and the four notices a mirror read leaves. A cover supplies its own words for
	// every one of them, because every plain one names something a covered frame may not hold —
	// the mirror, the file it lives in, the program that pins an operation, the account a cache
	// was filled from, and the two lists' product names. They sit here as one family rather than
	// one per style for the reason CopyCoverFailureBody does: the sentence is one sentence, and
	// only the style's own letters would differ.
	CopyCoverCapUnavailable      = "list unavailable"
	CopyCoverCapWrongAccount     = "wrong account"
	CopyCoverCapUnreadable       = "cache unreadable"
	CopyCoverCapRemedyPin        = "pin the operation, then restart"
	CopyCoverCapRemedyRepoint    = "re-point the cache, or read it offline"
	CopyCoverCapRemedyAside      = "move the cache aside, then sync"
	CopyCoverCapBodyMissing      = "no operation for this list in this build - pin one and restart"
	CopyCoverCapBodyWrongAccount = "this cache was filled by another account - re-point it, or keep reading it offline"
	CopyCoverCapBodyUnreadable   = "the cache could not be opened - move it aside and read this list once online"
	// CopyCoverEmptyNeverMirrored is the empty read's body when nothing has ever filled the store:
	// the plain reader's names the list and the mirror, so a cover's says the same fact without
	// either. The covered heading and the synced-empty body are the home feeds' own
	// (CopyCoverEmptyHeading, CopyCoverEmptyBody): an empty read is one state in every view.
	CopyCoverEmptyNeverMirrored = "nothing is cached for this view yet - press r to refresh"
	// The covered notices of a mirror read. Every plain one names the mirror or a product word
	// (`saved`), and each is reachable under a cover because `r` is not a cover-breaking key.
	CopyCoverSyncing      = "reading, {n} new"
	CopyCoverNoLongerHeld = "{n} no longer held"
	CopyCoverReadComplete = "read complete"
	CopyCoverReadStalled  = "read stopped (cursor stalled)"
	// CopyCoverListProfile is the status line's list field for a profile frame under a cover: the
	// account's own alias, because the plain field names the handle the frame was opened on and a
	// covered frame names no handle that exists on the upstream. The word is the covers' own word
	// for an account (their loading statements say `loading account...`).
	CopyCoverListProfile = "account: {handle}"

	CopyNoticeNoPosts  = "no posts"
	CopyNoticeNoAuthor = "no author to open"
	// Media action notices (MEDIA-03). One attachment uses CopyNoticeMediaOne with
	// {noun}; a set uses CopyNoticeMediaMany with the count of successes; a missing
	// opener prints the URL in the pane and says so here.
	CopyNoticeMediaOne     = "opened {noun}"
	CopyNoticeMediaMany    = "opened {n} media"
	CopyNoticePoster       = "opened poster frame"
	CopyNoticeOpenedImage  = "opened image"
	CopyNoticeOpenedVideo  = "opened video"
	CopyNoticeOpenedGIF    = "opened gif"
	CopyNoticeOpenedPoster = CopyNoticePoster
	CopyNoticeNoOpener     = "no opener - URL shown"
	CopyNoticeNoMedia      = "no media in this post"
	// Permalink / browser outcomes (REND-07). The covered refusal is not here:
	// NoticeGated reads StatusFields.Gated from the active grammar's MarkerSet.
	CopyNoticeOpenedBrowser = "opened in browser"
	CopyNoticeLinkCopied    = "link copied"
	CopyNoticeLinkCopiedWL  = "link copied with wl-copy"
	// The translation action's two outcomes (they are a pair for the same reason
	// CopyNoticeNewPosts/CopyNoticeNoNewPosts are: an action that spends a request says what it got,
	// and silence after one would be indistinguishable from an action that did nothing). The covered
	// pair is per style, below.
	CopyNoticeTranslated    = "showing a translation"
	CopyNoticeNoTranslation = "no translation available"
	CopyNoticeNoClipboard   = "no clipboard"
	CopyListProfile         = "profile: {handle}"
	// PlaceholderJoined is the one facts placeholder that is also matched by name: Plain's clauses
	// carry it in their own templates, and expandFact reads it as the second spelling of `{joined}`.
	// The three other placeholders are matched as the literal text of the clause that carries them
	// (profile.go's expandFact), so there is no constant for them and none is needed: a constant
	// nothing reads is a second copy of a string the renderer already spells.
	PlaceholderJoined = "{joined}"
	// The rate-limited state's composed form and its two remedies. The suffix is the local
	// wall-clock time when the reset is on the clock's own day and a dated time otherwise.
	CopyStateRateLimited  = "rate limited"
	CopyResetUntilClock   = "until {clock}"
	CopyResetUntilDate    = "until {date}"
	CopyRemedyRateLimited = "no retry"
	// CopyRemedyRateLimitedUnknown is the remedy when no reset header was sent: a wait with no
	// end is not a wait, and the line has to say so.
	CopyRemedyRateLimitedUnknown = "no retry; reset unknown"

	// The placeholders.
	PlaceholderCount       = "{n}"
	PlaceholderRateLimited = "{m}"
	PlaceholderNoun        = "{noun}"
	PlaceholderHandle      = "{handle}"
	PlaceholderClock       = "{clock}"
	PlaceholderDate        = "{date}"
	PlaceholderSelected    = "{selected}"
	PlaceholderLoaded      = "{loaded}"

	// The placeholders a cover's templates are written with. A template names a field and, after a
	// colon, the column it occupies: `{handle:16}` is the identity column, `{ref:8}` a queue's id
	// column, `{state:7}` a table's state column. The set is closed — a placeholder that is not one
	// of these names nothing — so a template cannot grow a second template language, and one
	// mechanism, this table and internal/text.Pad, places every field of every head.
	PlaceholderStamp = "{stamp}"
	PlaceholderRef   = "{ref}"
	PlaceholderName  = "{name}"
	// PlaceholderList is the account-list field's own word: following, followers, verified
	// followers. It is a placeholder like the rest, so the literal above stays a template a test can
	// read rather than a string assembled in the shell.
	PlaceholderList    = "{list}"
	PlaceholderEmail   = "{email}"
	PlaceholderMarkers = "{markers}"
	PlaceholderState   = "{state}"
	PlaceholderText    = "{text}"
	// PlaceholderProduct is the search product's word in the two places it appears — the prompt's
	// prefix and the status line's list field — and PlaceholderQuery is the query echo's slot in
	// the list field.
	PlaceholderProduct = "{product}"
	PlaceholderQuery   = "{query}"
	// The conversation's markers and pane statements (04-UI-SPEC's copywriting table). Every one of
	// them has a covered twin in cover.go, because a marker and a pane statement render under a cover
	// too: a frame that fell back to the plain literal would speak in the product's voice under it.
	//
	// MarkerReplyTo is the one with a placeholder: `{handle}` expands through Grammar.Identity — the
	// alias seam — so a marker cannot leak a real handle under a cover, and a cover that prefers its
	// own token writes `{ref}` in its template instead.
	MarkerFocal          = "focal"
	MarkerReplyTo        = "reply to {handle}"
	MarkerNoReplies      = "no replies"
	MarkerLoadingThread  = "loading thread..."
	MarkerPartialStopped = "partial thread - the reply walk stopped here"
	MarkerPartialAbove   = "partial thread - the post above is not reachable"
	MarkerPartialHidden  = "partial thread - replies are not reachable"
	// MarkerWithheld is the fourth visibility state's body line. Phase 2's marker table names
	// Limited, Unavailable and MissingText and carries no literal for Withheld, so this row is added
	// rather than inherited: a state without a literal would render as a blank row.
	MarkerWithheld = "post withheld"

	// The conversation's status-line vocabulary: the list's own word in a thread frame, the notices a
	// conversation has besides Phase 2's, and the one honest line an applicable key with nothing to
	// act on reports. `nothing to open` is a notice and not a pane statement: it says what the key
	// did, and the pane's own words are the conversation's.
	CopyListThread           = "thread"
	CopyNoticeLoadingReplies = "loading replies"
	CopyNoticeNewReply       = "{n} new reply"
	CopyNoticeNewReplies     = "{n} new replies"
	CopyNoticeNoNewReplies   = "no new replies"
	CopyNoticeNothingToOpen  = "nothing to open"

	// The cover's copy: the same states and the same two lists, in the words a work tool uses. Every
	// one of them is reached through the lookups below and never by a caller choosing one of two
	// constants itself, so a covered frame cannot print the plain reader's words by accident and a
	// plain frame cannot print a cover's.
	CopyCoverEmptyHeading = "no records in this view"
	CopyCoverEmptyBody    = "the read succeeded and found nothing - press r to refresh"
	// CopyCoverEmptyHedged is the likes list's empty answer under a cover: the plain reader's
	// `no likes, or the list is no longer served` names the list by its product word, and a covered
	// frame may not. What it may not change is the *shape* of the answer — one line and no body —
	// because the ambiguity is about the emptiness itself (06-UI-SPEC:263-266): a second line saying
	// the read succeeded and found nothing would resolve exactly what the line exists to leave open,
	// and a likes list whose endpoint the upstream may no longer serve cannot claim that.
	CopyCoverEmptyHedged = "no records in this view, or the view is no longer served"

	// The covered loading lines: the first page and a page of older records.
	CopyCoverLoading      = "loading records..."
	CopyCoverLoadingOlder = "loading older records"

	// CopyCoverFailureBody is the covered pane's failure line: what a tool says about a read that did
	// not complete. The head row above it carries the classifier's state word — the one field a cover
	// does not author, with the host substituted — and that word says which failure it was. The
	// classifier's own sentence stays out of a covered pane entirely: it is not the cover's to print,
	// because it names the host, the upstream's operation ids and the product's own words, and a
	// sentence a cover does not author is a sentence that can leak. The plain reader's pane keeps
	// that sentence byte for byte.
	CopyCoverFailureBody = "the last read did not complete"

	// The covered notices. The count is a count of records — not an engagement count, and not a
	// result the user asked for twice.
	CopyCoverNoticeNewRecord    = "{n} new record"
	CopyCoverNoticeNewRecords   = "{n} new records"
	CopyCoverNoticeNoNewRecords = "no new records"
	CopyCoverNoticeStalled      = "no older records (cursor stalled)"

	// The covered list names: the same two lists, in the same order, with the same behaviour, named
	// the way each style's own kind of tool would name them. The field stays droppable by the status
	// line's own declared rule when the longer names do not fit.
	CopyCoverListServiceLogFollowing = "stream=in"
	CopyCoverListServiceLogForYou    = "stream=all"
	CopyCoverListGitLogFollowing     = "branch=main"
	CopyCoverListGitLogForYou        = "branch=next"
	CopyCoverListCIFollowing         = "queue=ci"
	CopyCoverListCIForYou            = "queue=next"
	CopyCoverListInternalFollowing   = "view=mine"
	CopyCoverListInternalForYou      = "view=all"

	// CopyUpstreamHost is the host no covered frame may hold, and CopyUpstreamHostWord is what a
	// cover calls it instead: somebody else's hostname is not part of a failure, and the state still
	// says that a failure happened and which one.
	CopyUpstreamHost     = "x.com"
	CopyUpstreamHostWord = "upstream"
	// The covered copy of the conversation's surface: the same markers, statements, list word and
	// notices, in each style's own vocabulary. A covered frame that fell back to a plain literal would
	// speak in the product's voice under the cover (DSGT-09), and the marker set is per grammar, so
	// every style carries its own row for all eight.
	//
	// The servicelog style writes key=value like every other field of its head; the gitlog style writes
	// git's own header lines; the ci style writes bracketed log notes; and the internal style writes
	// detail lines under its row.
	CoverServiceLogFocal        = "focus=1"
	CoverServiceLogReplyTo      = " reply_to={handle}"
	CoverServiceLogNoReplies    = "responses=0"
	CoverServiceLogLoading      = "loading conversation..."
	CoverServiceLogStopped      = "partial=true stopped=cursor"
	CoverServiceLogAbove        = "partial=true parent=absent"
	CoverServiceLogHidden       = "partial=true responses=absent"
	CoverServiceLogListThread   = "stream=thread"
	CoverServiceLogLoadingRepl  = "reply read in flight"
	CoverServiceLogNewReply     = "{n} new response"
	CoverServiceLogNewReplies   = "{n} new responses"
	CoverServiceLogNoNewReplies = "no new responses"
	CoverServiceLogNothingOpen  = "nothing selected"

	CoverGitFocal        = "Focal: yes"
	CoverGitReplyTo      = "Responds-to: {name}"
	CoverGitNoReplies    = "(no replies)"
	CoverGitLoading      = "loading the conversation..."
	CoverGitStopped      = "Partial: reply walk stopped"
	CoverGitAbove        = "Partial: parent commit absent"
	CoverGitHidden       = "Partial: replies absent"
	CoverGitListThread   = "scope=conversation"
	CoverGitLoadingRepl  = "reading the conversation"
	CoverGitNewReply     = "{n} new response"
	CoverGitNewReplies   = "{n} new responses"
	CoverGitNoNewReplies = "no new responses"
	CoverGitNothingOpen  = "no commit selected"

	CoverCIFocal        = " focus=1"
	CoverCIReplyTo      = " responds_to={handle}"
	CoverCINoReplies    = " responses=0"
	CoverCILoading      = "loading conversation..."
	CoverCIStopped      = " note=walk stopped"
	CoverCIAbove        = " note=parent absent"
	CoverCIHidden       = " note=responses absent"
	CoverCIListThread   = "job=thread"
	CoverCILoadingRepl  = "fetching responses"
	CoverCINoNewReplies = "no new responses"
	CoverCINothingOpen  = "nothing selected"

	CoverInternalFocal        = "selected=true"
	CoverInternalReplyTo      = "responds to {handle}"
	CoverInternalNoReplies    = "no responses"
	CoverInternalLoading      = "loading conversation..."
	CoverInternalStopped      = "partial: walk stopped"
	CoverInternalAbove        = "partial: parent missing"
	CoverInternalHidden       = "partial: responses missing"
	CoverInternalListThread   = "view=thread"
	CoverInternalLoadingRepl  = "loading responses"
	CoverInternalNewReply     = "{n} new response"
	CoverInternalNewReplies   = "{n} new responses"
	CoverInternalNoNewReplies = "no new responses"
	CoverInternalNothingOpen  = "no request selected"

	// The covered copy of the search surface: the two new notices, the prompt's prefix and the
	// results pane's loading statement, in each style's own vocabulary. A covered frame that fell
	// back to a plain literal would speak in the product's voice under the cover (DSGT-09), so each
	// style carries its own row for all four. The product word itself (`latest`, `top`) is ordering
	// vocabulary, not a name, and is the {product} slot's content in every style.
	CoverServiceLogNoResults       = "matches=0"
	CoverServiceLogNothingToSwitch = "no alternate stream"
	CoverServiceLogPromptSearch    = "filter={product}"
	CoverServiceLogPromptPeople    = "people:"
	CoverServiceLogLoadingResults  = "loading records..."

	CoverGitNoResults       = "(no matches)"
	CoverGitNothingToSwitch = "no other branch"
	CoverGitPromptSearch    = "grep: {product}"
	CoverGitPromptPeople    = "people:"
	CoverGitLoadingResults  = "loading the records..."

	CoverCINoResults       = " matches=0"
	CoverCINothingToSwitch = "no alternate job"
	CoverCIPromptSearch    = "filter={product}"
	CoverCIPromptPeople    = "people:"
	CoverCILoadingResults  = "loading records..."

	CoverInternalNoResults       = "no requests match"
	CoverInternalNothingToSwitch = "no alternate view"
	CoverInternalPromptSearch    = "filter: {product}"
	CoverInternalPromptPeople    = "people:"
	CoverInternalLoadingResults  = "loading requests..."

	// Covered profile notices and markers: each style's own vocabulary so a covered frame cannot
	// speak in the product's voice (DSGT-09).
	CoverServiceLogLoadingProfile = "loading account..."
	CoverServiceLogLoadingPosts   = "loading records"
	CoverServiceLogNoPosts        = "records=0"
	CoverServiceLogNoAuthor       = "no author"
	CoverGitLoadingProfile        = "loading the account..."
	CoverGitLoadingPosts          = "loading the records"
	CoverGitNoPosts               = "(no records)"
	CoverGitNoAuthor              = "no author on this record"
	CoverCILoadingProfile         = "loading account..."
	CoverCILoadingPosts           = "loading records"
	CoverCINoPosts                = " records=0"
	CoverCINoAuthor               = "no author"
	CoverInternalLoadingProfile   = "loading request owner..."
	CoverInternalLoadingPosts     = "loading requests"
	CoverInternalNoPosts          = "no requests"
	CoverInternalNoAuthor         = "no owner on this record"

	// The covered pair of the translation action's notices. Every style's wording is the style's own
	// words for a rendered payload — the same four styles that already answer a missing body with
	// their own marker (`CoverServiceLogNoPayload`, `CoverGitNoMessage`, `CoverCINoResults`,
	// `CoverInternalNoResults`) — because a covered frame carries no plain-reader word and a
	// translation is one more body on that screen.
	CoverServiceLogTranslate   = "payload rendered"
	CoverServiceLogNoTranslate = "no rendered payload"
	CoverGitTranslate          = "message rendered"
	CoverGitNoTranslate        = "no rendered message"
	CoverCINoTranslate         = "no rendered log"
	CoverCITranslate           = "log rendered"
	CoverInternalTranslate     = "response rendered"
	CoverInternalNoTranslate   = "no rendered response"
)

// stateWords is the status line's state column: one row per failure kind, and a kind without a
// row is a state with no words, which state_test.go fails on. The words are the classifier's own
// (xweb.Outcome.Error, netcheck.Outcome) except for the four documented renames:
//
//	xweb.OutcomeAuthRejected    "auth rejected"                -> "session expired"
//	xweb.OutcomeUnknownOperation "unknown operation"           -> "operation id stale"
//	netcheck.OutcomeUpstreamBlocked "proxy reached but upstream blocked" -> "upstream blocked"
//	netcheck.OutcomeSessionSkipped  "session not configured"   -> "no session file"
//
// The renames are prose, not a second vocabulary: each one says what the user's account is not
// (a bad account, a typo, a healthy proxy, a configured session) and none of them changes what
// the state is. Where the UI-SPEC's table and the classifier disagree on spelling — its
// "auth or signer rejected" against xweb's "signer or auth rejected" — the classifier's spelling
// wins, because the state field prints the classifier verbatim.
var stateWords = map[domain.FailureKind]string{
	domain.FailureSessionExpired:    "session expired",
	domain.FailureSignerOrAuth:      "signer or auth rejected",
	domain.FailureOperationStale:    "operation id stale",
	domain.FailureEndpointRetired:   "endpoint retired",
	domain.FailureRateLimited:       "rate limited",
	domain.FailureSoftBan:           "soft ban",
	domain.FailureLoadShed:          "load shed",
	domain.FailureBadRequest:        "bad request",
	domain.FailureEmptyBody:         "empty response body",
	domain.FailureChallenge:         "edge challenge",
	domain.FailureUnauthorized:      "unauthorized",
	domain.FailureUnclassified:      "unclassified",
	domain.FailureBudgetSpent:       "budget spent", // xweb.ErrRequestCap
	domain.FailureNoRoute:           "no route to x.com",
	domain.FailureProxyRefused:      "proxy refused",
	domain.FailureUpstreamBlocked:   "upstream blocked",
	domain.FailureTLSChallenge:      "TLS/Cloudflare challenge",
	domain.FailureTransportUnusable: "transport unusable",
	domain.FailureNoSession:         "no session file",
}

// remedies is the status line's terse remedy column, one row per failure kind. It is deliberately
// not xweb.Outcome.Remedy: that sentence is what the pane prints when it has nothing else to show
// (domain.Failure.Detail), and this column says what to do in as few cells as the line can afford.
var remedies = map[domain.FailureKind]string{
	domain.FailureSessionExpired:    "copy the cookie again",
	domain.FailureSignerOrAuth:      "add a transaction id",
	domain.FailureOperationStale:    "doctor --refresh-ids",
	domain.FailureEndpointRetired:   "do not retry",
	domain.FailureRateLimited:       CopyRemedyRateLimited,
	domain.FailureSoftBan:           "stop for the session",
	domain.FailureLoadShed:          "backoff, then retry",
	domain.FailureBadRequest:        "do not retry",
	domain.FailureEmptyBody:         "re-check the route",
	domain.FailureChallenge:         "an edge page, not the API",
	domain.FailureUnauthorized:      "see doctor for the message",
	domain.FailureUnclassified:      "record this shape first",
	domain.FailureBudgetSpent:       "restart to reset the budget",
	domain.FailureNoRoute:           "check the proxy",
	domain.FailureProxyRefused:      "start the proxy",
	domain.FailureUpstreamBlocked:   "check the proxy",
	domain.FailureTLSChallenge:      "check the proxy",
	domain.FailureTransportUnusable: "fix the proxy setting",
	domain.FailureNoSession:         "write the session file",
}

// StateWord is the state column's word for a kind: the classifier's own name for the state, or
// the empty string for FailureNone, which is not a row of either table (there is nothing to say).
func StateWord(kind domain.FailureKind) string { return stateWords[kind] }

// Remedy is the terse remedy column's row for a kind, and the empty string for FailureNone. The
// rate-limited row's other form — CopyRemedyRateLimitedUnknown, when no reset header was sent —
// is the status line's business: it depends on the failure's Reset, not on the kind.
func Remedy(kind domain.FailureKind) string { return remedies[kind] }

// FailureWords is the failure pane's own two strings for a style: the head row's state word and the
// body line under it. It is the one place the pane's failure words are decided, so the shell's pane
// and the stored corpus cannot disagree about them.
//
// Under the plain reader they are the classifier's: the state's own name, and the sentence the caller
// copied out of the classifier or out of the failing reachability check, byte for byte — the reader is
// owed what actually failed.
//
// Under a cover the head row is that name with the upstream's host substituted, which is the one edit
// a cover makes to words it did not author, and the body is the cover's own line. The detail is not
// edited here, not scrubbed here and not read at all on that path: a covered pane never carries a
// sentence the cover did not write, so nothing the classifier says about somebody's hostname — or
// about the product's own feeds and operations — can reach a covered screen.
func FailureWords(cover CoverStyle, kind domain.FailureKind, detail string) (word, body string) {
	if cover == "" {
		return StateWord(kind), detail
	}
	return HostSubstituted(StateWord(kind)), CopyCoverFailureBody
}

// EmptyHeading is the empty state's heading for a feed: the cover's words under a cover, and under
// the plain reader the feed's own, because there the heading names the feed.
func EmptyHeading(cover CoverStyle, kind domain.HomeKind) string {
	plain := CopyEmptyHeadingFollowing
	if kind == domain.ForYou {
		plain = CopyEmptyHeadingForYou
	}
	return coverOr(cover, CopyCoverEmptyHeading, plain)
}

// EmptyBody is the empty state's second line: it says what happened (the read succeeded) and what to
// do about it, under every cover, because an empty timeline and a failed read look the same on
// screen and only one of them is true.
func EmptyBody(cover CoverStyle) string {
	return coverOr(cover, CopyCoverEmptyBody, CopyEmptyBody)
}

// LoadingLine is the first page's loading line: the cover's words under a cover, the feed's own
// under the plain reader.
func LoadingLine(cover CoverStyle, kind domain.HomeKind) string {
	plain := CopyLoadingFollowing
	if kind == domain.ForYou {
		plain = CopyLoadingForYou
	}
	return coverOr(cover, CopyCoverLoading, plain)
}

// SavedLoadingLine is a saved frame's own statement while its first page is out: the cover's words
// under a cover, the plain reader's under it. The list being read is named by the status line's own
// field, which is where a saved frame's list name lives, so the pane's line names the state and not
// the list.
func SavedLoadingLine(cover CoverStyle) string {
	return coverOr(cover, CopyCoverLoading, CopyLoadingSaved)
}

// NoticeLine is the notice field's words for a set of fields: the program's own announcement about
// the list, never a result the user asked for twice. The count is a count of records.
//
// The conversation's notices are here with the list's because the field is one field and its drop
// rule is the line's: `loading replies` while a reply page is out, and the refresh's own answer after
// `r` — the `{n} new reply` pair, and the zero that says "no new replies" out loud.
func NoticeLine(cover CoverStyle, notice NoticeKind, count int) string {
	switch notice {
	case NoticeNewPosts:
		plain := CopyNoticeNewPosts
		if count == 1 {
			plain = CopyNoticeNewPost
		}
		covered := CopyCoverNoticeNewRecords
		if count == 1 {
			covered = CopyCoverNoticeNewRecord
		}
		return strings.ReplaceAll(coverOr(cover, covered, plain), PlaceholderCount, strconv.Itoa(count))
	case NoticeNoNewPosts:
		return coverOr(cover, CopyCoverNoticeNoNewRecords, CopyNoticeNoNewPosts)
	case NoticeLoadingOlder:
		return coverOr(cover, CopyCoverLoadingOlder, CopyNoticeLoadingOlder)
	case NoticeStalled:
		return coverOr(cover, CopyCoverNoticeStalled, CopyNoticeStalled)
	case NoticeLoadingReplies:
		return coverOr(cover, coverLoadingReplies(cover), CopyNoticeLoadingReplies)
	case NoticeNewReplies:
		plain := CopyNoticeNewReplies
		if count == 1 {
			plain = CopyNoticeNewReply
		}
		return strings.ReplaceAll(coverOr(cover, coverNewReplies(cover, count), plain), PlaceholderCount, strconv.Itoa(count))
	case NoticeNoNewReplies:
		return coverOr(cover, coverNoNewReplies(cover), CopyNoticeNoNewReplies)
	case NoticeNothingToOpen:
		return coverOr(cover, coverNothingToOpen(cover), CopyNoticeNothingToOpen)
	case NoticeNoResults:
		return coverOr(cover, coverNoResults(cover), CopyNoticeNoResults)
	case NoticeNothingToSwitch:
		return coverOr(cover, coverNothingToSwitch(cover), CopyNoticeNothingToSwitch)
	case NoticeLoadingPosts:
		return coverOr(cover, coverLoadingPosts(cover), CopyNoticeLoadingPosts)
	case NoticeNoPosts:
		return coverOr(cover, coverNoPosts(cover), CopyNoticeNoPosts)
	case NoticeNoAuthor:
		return coverOr(cover, coverNoAuthor(cover), CopyNoticeNoAuthor)
	// The four notices a mirror read leaves. They are reachable under a cover — `r` on a saved
	// frame is not a cover-breaking key — and every plain one names the mirror or a product word
	// (`saved`), so each carries the cover's own row here.
	case NoticeSyncing:
		return strings.ReplaceAll(coverOr(cover, CopyCoverSyncing, CopySyncing), PlaceholderCount, strconv.Itoa(count))
	case NoticeNoLongerSaved:
		return strings.ReplaceAll(coverOr(cover, CopyCoverNoLongerHeld, CopyNoticeNoLongerSaved), PlaceholderCount, strconv.Itoa(count))
	case NoticeMirrorComplete:
		return coverOr(cover, CopyCoverReadComplete, CopyNoticeMirrorComplete)
	case NoticeSyncStalled:
		return coverOr(cover, CopyCoverReadStalled, CopyNoticeSyncStalled)
	case NoticeOpenedImage:
		return CopyNoticeOpenedImage
	case NoticeOpenedVideo:
		return CopyNoticeOpenedVideo
	case NoticeOpenedGIF:
		return CopyNoticeOpenedGIF
	case NoticeOpenedPoster:
		return CopyNoticeOpenedPoster
	case NoticeOpenedMedia:
		return strings.ReplaceAll(CopyNoticeMediaMany, PlaceholderCount, strconv.Itoa(count))
	case NoticeNoOpener:
		return CopyNoticeNoOpener
	case NoticeNoMedia:
		return CopyNoticeNoMedia
	case NoticeOpenedBrowser:
		return CopyNoticeOpenedBrowser
	case NoticeLinkCopied:
		return CopyNoticeLinkCopied
	case NoticeLinkCopiedWL:
		return CopyNoticeLinkCopiedWL
	case NoticeNoClipboard:
		return CopyNoticeNoClipboard
	// The translation action's two outcomes. They are reachable under a cover — `T` is not a
	// cover-breaking key — and the plain wording names the act of translating, which a covered frame
	// says in its own vocabulary for a rendered payload.
	case NoticeTranslated:
		return coverOr(cover, coverTranslated(cover), CopyNoticeTranslated)
	case NoticeNoTranslation:
		return coverOr(cover, coverNoTranslation(cover), CopyNoticeNoTranslation)
	case NoticeGated:
		// Words come from StatusFields.Gated via statusParts; NoticeLine stays empty.
		return ""
	}
	return ""
}

// The conversation's covered notices, one row per style. They are read through the lookups below
// rather than by a caller choosing among constants, so no covered frame spells the plain reader's
// word by accident.
func coverLoadingReplies(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogLoadingRepl
	case CoverGitLog:
		return CoverGitLoadingRepl
	case CoverCI:
		return CoverCILoadingRepl
	case CoverInternal:
		return CoverInternalLoadingRepl
	}
	return CopyNoticeLoadingReplies
}

// coverNewReplies is the replies-found notice: the style's own singular at one and plural otherwise.
// The ci style answers in the servicelog's pair on purpose: the two styles' words for a response are
// the same two words, and a second pair of constants spelling them again was a duplicate nothing
// read — the case is one case because the vocabulary is one vocabulary.
func coverNewReplies(cover CoverStyle, count int) string {
	one := count == 1
	switch cover {
	case CoverServiceLog, CoverCI:
		if one {
			return CoverServiceLogNewReply
		}
		return CoverServiceLogNewReplies
	case CoverGitLog:
		if one {
			return CoverGitNewReply
		}
		return CoverGitNewReplies
	case CoverInternal:
		if one {
			return CoverInternalNewReply
		}
		return CoverInternalNewReplies
	}
	return CopyNoticeNewReplies
}

// coverNoNewReplies is the style's own "nothing new" for a conversation.
func coverNoNewReplies(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogNoNewReplies
	case CoverGitLog:
		return CoverGitNoNewReplies
	case CoverCI:
		return CoverCINoNewReplies
	case CoverInternal:
		return CoverInternalNoNewReplies
	}
	return CopyNoticeNoNewReplies
}

// coverNothingToOpen is the style's own line for an applicable key whose action has nothing to act
// on.
func coverNothingToOpen(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogNothingOpen
	case CoverGitLog:
		return CoverGitNothingOpen
	case CoverCI:
		return CoverCINothingOpen
	case CoverInternal:
		return CoverInternalNothingOpen
	}
	return CopyNoticeNothingToOpen
}

// coverNoResults is the style's own line for a successful read that carried no rows.
func coverNoResults(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogNoResults
	case CoverGitLog:
		return CoverGitNoResults
	case CoverCI:
		return CoverCINoResults
	case CoverInternal:
		return CoverInternalNoResults
	}
	return CopyNoticeNoResults
}

// coverTranslated is the style's own line for a body the translation service rendered: the action's
// outcome in the words a work tool uses for a payload it has just produced.
func coverTranslated(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogTranslate
	case CoverGitLog:
		return CoverGitTranslate
	case CoverCI:
		return CoverCITranslate
	case CoverInternal:
		return CoverInternalTranslate
	}
	return CopyNoticeTranslated
}

// coverNoTranslation is the same action's other outcome: the service answered and carried nothing to
// render, so this record has no payload in the configured language.
func coverNoTranslation(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogNoTranslate
	case CoverGitLog:
		return CoverGitNoTranslate
	case CoverCI:
		return CoverCINoTranslate
	case CoverInternal:
		return CoverInternalNoTranslate
	}
	return CopyNoticeNoTranslation
}

// coverNothingToSwitch is the style's own line for `s` where there is no order to switch.
func coverNothingToSwitch(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogNothingToSwitch
	case CoverGitLog:
		return CoverGitNothingToSwitch
	case CoverCI:
		return CoverCINothingToSwitch
	case CoverInternal:
		return CoverInternalNothingToSwitch
	}
	return CopyNoticeNothingToSwitch
}

// ThreadListName is the status line's list field in a conversation: the thread's own word, which
// names the surface rather than a feed, in the words the style on screen uses. The feed's own names
// are ListName's.
func ThreadListName(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogListThread
	case CoverGitLog:
		return CoverGitListThread
	case CoverCI:
		return CoverCIListThread
	case CoverInternal:
		return CoverInternalListThread
	}
	return CopyListThread
}

// ListName is the status line's list field: the list the model holds, named the way the style on
// screen names it. The covered names are the tool's own words for the same two lists rather than
// abbreviations of the plain ones — a field a reader cannot act on is worse than a missing one — and
// the field stays droppable by the status line's own declared rule when they do not fit.
func ListName(cover CoverStyle, kind domain.HomeKind) string {
	forYou := kind == domain.ForYou
	switch cover {
	case CoverServiceLog:
		return coverList(CopyCoverListServiceLogFollowing, CopyCoverListServiceLogForYou, forYou)
	case CoverGitLog:
		return coverList(CopyCoverListGitLogFollowing, CopyCoverListGitLogForYou, forYou)
	case CoverCI:
		return coverList(CopyCoverListCIFollowing, CopyCoverListCIForYou, forYou)
	case CoverInternal:
		return coverList(CopyCoverListInternalFollowing, CopyCoverListInternalForYou, forYou)
	}
	return ListHome(kind)
}

// ListHome is the list field's plain name for a home timeline: Phase 2's two literals, factored
// out of ListName so the covered branch and the plain branch read the field's two names from one
// place each.
func ListHome(kind domain.HomeKind) string {
	if kind == domain.ForYou {
		return CopyListForYou
	}
	return CopyListFollowing
}

// ListLabelQueryCells is how wide the query echo in the status line's list field may be: enough
// to recognise the query, little enough that the field stays droppable by the line's own rule.
const ListLabelQueryCells = 32

// ListLabelSearch is the status line's list field for a results frame: the product and the query,
// named once — the one place the two are composed, so no call site builds the sentence. The echo
// is capped at ListLabelQueryCells with the truncation mark in its last cell, cut at a cluster
// boundary: a query longer than that is still the frame's, and `/` re-seeds the prompt with the
// whole of it.
//
// Under a cover the field is the cover's own: the prefix is the same one the prompt row carries
// (Markers.PromptSearch — `filter={product}` for a log tail, `grep: {product}` for git), read off
// the grammar rather than spelled again here, so the status line and the field the user types into
// cannot drift apart. The product word itself is ordering vocabulary under every style and stays.
func ListLabelSearch(g Grammar, p domain.SearchProduct, query string) string {
	echo := cutField(query, ListLabelQueryCells, MarkTruncation)
	prefix := CopyListSearch
	if _, ok := CoverOfGrammar(g); ok {
		prefix = prefixOrEmpty(g.Markers.PromptSearch, MarkerPromptSearch) + " \"{query}\""
	}
	label := strings.ReplaceAll(prefix, PlaceholderProduct, p.String())
	return strings.ReplaceAll(label, PlaceholderQuery, echo)
}

// ListLabelPeople is the status line's list field for a people results frame: the query,
// named once — the one place the echo is composed. Capped exactly as ListLabelSearch caps it.
//
// Under a cover the prefix is the grammar's own prompt word for the people product
// (Markers.PromptPeople), which is `people:` in all four styles this phase ships: the field is the
// same field, and it is read off the grammar so a style that renames the prompt renames this with
// it.
func ListLabelPeople(g Grammar, query string) string {
	echo := cutField(query, ListLabelQueryCells, MarkTruncation)
	label := CopyListPeople
	if _, ok := CoverOfGrammar(g); ok {
		label = prefixOrEmpty(g.Markers.PromptPeople, MarkerPromptPeople) + " \"{query}\""
	}
	return strings.ReplaceAll(label, PlaceholderQuery, echo)
}

// ListLabelProfile is the status line's list field for a profile frame: the account named once —
// the handle the frame was opened on under the plain reader, and the account's own alias under a
// cover, because no covered frame names a handle that exists on the upstream.
// ListLabelAccounts is the status line's list field for an account-list frame: which list it is and
// whose. Under a cover the handle is the cover's alias, and the list word stays: "following" is not
// a word of the product, and a covered frame that printed no list at all would leave the reader
// with a position in nothing.
func ListLabelAccounts(g Grammar, kind domain.UserListKind, u domain.User) string {
	handle := u.ScreenName
	if _, ok := CoverOfGrammar(g); ok {
		handle = g.Identity(u).Handle
	}
	label := strings.ReplaceAll(CopyListAccounts, PlaceholderList, kind.String())
	return strings.ReplaceAll(label, PlaceholderHandle, handle)
}

// ListLabelMedia is the status line's list field for a media frame, on the same rule.
func ListLabelMedia(g Grammar, u domain.User) string {
	handle := u.ScreenName
	if _, ok := CoverOfGrammar(g); ok {
		handle = g.Identity(u).Handle
	}
	return strings.ReplaceAll(CopyListMedia, PlaceholderHandle, handle)
}

func ListLabelProfile(g Grammar, u domain.User) string {
	if _, ok := CoverOfGrammar(g); ok {
		return strings.ReplaceAll(CopyCoverListProfile, PlaceholderHandle, g.Identity(u).Handle)
	}
	return strings.ReplaceAll(CopyListProfile, PlaceholderHandle, u.ScreenName)
}

// prefixOrEmpty is a grammar's own marker where it carries one and the plain reader's otherwise,
// which is what keeps a half-filled grammar from rendering a field with no words in it.
func prefixOrEmpty(marker, fallback string) string {
	if marker == "" {
		return fallback
	}
	return marker
}

func coverLoadingPosts(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogLoadingPosts
	case CoverGitLog:
		return CoverGitLoadingPosts
	case CoverCI:
		return CoverCILoadingPosts
	case CoverInternal:
		return CoverInternalLoadingPosts
	}
	return CopyNoticeLoadingPosts
}

func coverNoPosts(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogNoPosts
	case CoverGitLog:
		return CoverGitNoPosts
	case CoverCI:
		return CoverCINoPosts
	case CoverInternal:
		return CoverInternalNoPosts
	}
	return CopyNoticeNoPosts
}

func coverNoAuthor(cover CoverStyle) string {
	switch cover {
	case CoverServiceLog:
		return CoverServiceLogNoAuthor
	case CoverGitLog:
		return CoverGitNoAuthor
	case CoverCI:
		return CoverCINoAuthor
	case CoverInternal:
		return CoverInternalNoAuthor
	}
	return CopyNoticeNoAuthor
}

// HostSubstituted replaces the upstream host, and only it, with the cover's generic word: every
// `x.com` and every `<label>.x.com`, case-insensitively, becomes `upstream`, and nothing else about
// the sentence changes. It is the one edit a cover makes to words it did not author — the
// classifier's state word and its pane sentence, and a reachability check's detail — so those keep
// their own vocabulary and lose only a proper noun that is not the reader's business.
//
// The whole hostname goes, so `api.x.com` is `upstream` and not `api.upstream`: the labels in front
// of a host are the same host's, and a rename that left `api.` standing would leave a name that still
// points at somebody. A string with no host in it comes back byte for byte.
//
// The scan walks the string's own bytes: a hostname is one run of label bytes, and the run is
// rewritten as a whole or not at all, so the only offsets this file uses are the run's two ends and
// nothing here counts a character. internal/text is the program's one measure, and a byte offset in
// a sentence is not a width.
func HostSubstituted(s string) string {
	var out strings.Builder
	written := 0 // how many bytes of s the builder has accounted for
	start := -1  // where the run of label bytes in hand began, or -1 between two runs
	for at := range s {
		if labelByte(s[at]) {
			if start < 0 {
				start = at
			}
			continue
		}
		if start >= 0 {
			if isHostRun(s[start:at]) {
				out.WriteString(s[written:start])
				out.WriteString(CopyUpstreamHostWord)
				written = at
			}
			start = -1
		}
	}
	if start >= 0 && isHostRun(s[start:]) {
		// The last run reaches the end of the string: there is nothing behind it to copy.
		out.WriteString(s[written:start])
		out.WriteString(CopyUpstreamHostWord)
		return out.String()
	}
	out.WriteString(s[written:])
	return out.String()
}

// isHostRun reports whether one run of hostname labels is the upstream host or one of its
// subdomains: it ends with the host, case-insensitively, and everything in front of the host is a
// dot-terminated run of well-formed labels. The caller has established where the run starts — the
// byte in front of a run is not a label byte — so `x.com` and `api.eu.x.com` are the host, while a
// run that is a different host whose name merely ends in the host's letters (`notx.com` is one
// label, not a subdomain) and a host followed by more of a label (`x.comfoo`) are both not.
func isHostRun(run string) bool {
	lower := strings.ToLower(run)
	if !strings.HasSuffix(lower, CopyUpstreamHost) {
		return false
	}
	labels := strings.TrimSuffix(lower, CopyUpstreamHost)
	if labels == "" {
		return true
	}
	if !strings.HasSuffix(labels, ".") {
		return false
	}
	for _, label := range strings.Split(strings.TrimSuffix(labels, "."), ".") {
		if !wellFormedLabel(label) {
			return false
		}
	}
	return true
}

// labelByte is whether a byte can be part of a hostname label: the letters, the digits, the hyphen
// and the dot that separates two labels.
func labelByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '.':
		return true
	}
	return false
}

// wellFormedLabel is whether one hostname label is one: non-empty, letters, digits and hyphens, and
// no hyphen at either end.
func wellFormedLabel(label string) bool {
	if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}
	for at := range label {
		if c := label[at]; !labelByte(c) || c == '.' {
			return false
		}
	}
	return true
}

// coverOr picks the cover's string under a cover and the plain reader's without one: the zero
// CoverStyle is the plain reader, so no caller has to ask whether a cover is worn.
func coverOr(cover CoverStyle, covered, plain string) string {
	if cover != "" {
		return covered
	}
	return plain
}

// coverList picks one of a style's two list names for a feed.
func coverList(following, forYou string, forYouList bool) string {
	if forYouList {
		return forYou
	}
	return following
}

// mediaNoun is the noun a counted media kind takes: singular at one, plural otherwise.
func mediaNoun(kind domain.MediaKind, n int) string {
	one := n == 1
	switch kind {
	case domain.Video:
		if one {
			return NounVideo
		}
		return NounVideos
	case domain.GIF:
		if one {
			return NounGIF
		}
		return NounGIFs
	default:
		if one {
			return NounImage
		}
		return NounImages
	}
}
