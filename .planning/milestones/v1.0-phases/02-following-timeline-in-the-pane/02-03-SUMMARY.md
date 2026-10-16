---
phase: 02-following-timeline-in-the-pane
plan: 03
subsystem: ui
tags: [render, grammar, status-line, colour-ladder, palette, help-overlay, truncation, REND-01, REND-04]

# Dependency graph
requires:
  - phase: 02-01
    provides: internal/domain (Post/User/PostPage/FailureKind) and internal/screen (the list state machine) this package projects into rows
  - phase: 02-02
    provides: "internal/text — Width, Wrap, Pad, Truncate: the only measurement this package is allowed to use"
provides:
  - "internal/render — Grammar/Plain, Blocks/Block, StatusLine/StatusFields, Frame, Help/HelpRow, Palette/Rung/Role, StateWord/Remedy, StateBlock"
  - "the UI-SPEC's two worked frames (80 and 120 columns) transcribed as literal test expectations, with the two places the fenced text cannot be the width model's output measured and recorded"
  - "the colour ladder (truecolor/256/16/mono) with the roles dominant/muted/state/state-stop/selection, and A1/A2/A3 asserted at the unit, block and frame levels"
  - "internal/xweb.APIError.Reset — the upstream's x-rate-limit-reset instant, which the rate-limited state line prints"
affects: [02-04-tui-keys, 02-05-golden-frames, phase-03-grammars]

# Actuals — chars/4 over the files actually changed
actuals:
  tokens: 43500
  tasks: 3
  commits: 0

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "one Grammar value per display style: identity, stamp, indents, measure cap, markers and separator are the only things a cover changes"
    - "every user-visible string in copy.go, reached only through a template with a documented placeholder"
    - "rows are composed plain and painted per run, because internal/text measures and wraps text while it counts an escape as zero cells"
    - "the rung is read from the environment at paint time (one decision per process, never from config, never a terminal query)"
    - "the state words are pinned to the classifiers by a test that reads xweb.Outcome.Error() and netcheck.Outcome, so the screen cannot drift"

key-files:
  created:
    - internal/render/copy.go
    - internal/render/grammar.go
    - internal/render/block.go
    - internal/render/status.go
    - internal/render/frame.go
    - internal/render/palette.go
    - internal/render/help.go
    - internal/render/block_test.go
    - internal/render/status_test.go
    - internal/render/palette_test.go
    - internal/render/help_test.go
    - internal/render/state_test.go
  modified:
    - internal/xweb/classify.go
    - internal/xweb/graphql.go
    - internal/xweb/classify_test.go

key-decisions:
  - "StatusFields carries a Now time.Time (the plan's struct did not): the rate-limited state's `until 14:32` / `until Jan 02 15:04` forms need a reference day, and a wall-clock read inside the renderer would make a frame non-reproducible"
  - "the block builder paints its own runs (the palette rung is read per call) and Frame paints only the selection bar from the row's own `>` mark: the environment cannot change under a process, and a test that wants literals pins TERM=dumb/NO_COLOR=1"
  - "StampFormat.Stamp returns at least 12 cells rather than exactly 12: the plain grammar's two forms are 12, and a cover's ISO instant (20) keeps its own width instead of being cut"
  - "the zero instant has no date: a tombstone's head row is blank rather than `Jan 01  0001`"
  - "the attachment template is read the way its own shape reads (words before the first {n} printed once, the rest repeated per kind), so the line is `attachment: 2 images, 1 video` and not the words repeated"
  - "cutField and cutStatus drop the spaces a cut left before appending the mark, so the mark is the last cell the field occupies"
  - "a marker or a key list that cannot fit is cut with the truncation mark, because Frame truncates no content row and a silently cut field would be a lie"

patterns-established:
  - "the UI-SPEC's fenced frames are transcribed as literal rows, and where they contradict the document's own width model the contradiction is asserted with its numbers instead of being papered over"
  - "anti-vacuity arms are run by hand against the shipped code: eight breaks, each watched to fail, each reverted"

requirements-completed: [REND-01, REND-04]

# Coverage metadata
coverage:
  - id: D1
    description: "The plain grammar's post-block layout: 16-cell identity column, 12-cell absolute stamp, 2-cell indent, capped measure, quote prefix capped at two levels, one blank separator (dropped between a post and its continuation), placeholders for a tombstone and a missing body, and an attachment line counted in the fixed noun order"
    requirement: REND-01
    verification:
      - kind: unit
        ref: "internal/render/block_test.go#TestBlocksTheUISpecFrameAt80Columns"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestBlocksTheUISpecFrameAt120Columns"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheTwoGrammarsExample"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheHandleColumnIsFixed"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestAContinuationCarriesNoSeparator"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestMarkerPlaceholders"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheAttachmentLineCountsEveryKind"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheQuotePrefixStopsGrowingAtTheCap"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheListGrammarPrintsNoCountAndNoURL"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestSelectionIsTheMarkAndOnlyTheMark"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheMeasureIsTheCapNotThePane"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestABlockRowNeverExceedsThePane"
        status: pass
    human_judgment: false
  - id: D2
    description: "The status line: fixed field order, two-space separators, no adjacent gaps, drop order notice then position then list, state/remedy/budget never dropped, never wrapped, cut with the mark when even they do not fit, and the rate-limited state composed with the upstream's reset instant"
    requirement: REND-04
    verification:
      - kind: unit
        ref: "internal/render/status_test.go#TestStatusLineWorkedLines"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestTheFifthWorkedLineIsWiderThanItsPane"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestStatusLineDropsInOrder"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestStatusLineNeverWraps"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestStatusLineRateLimitedForms"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestStatusLineBudgetNeverMoves"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestTheStateWordsAreTheOnesOnTheLine"
        status: pass
    human_judgment: false
  - id: D3
    description: "The frame composer: exactly h rows of exactly w cells, the status line last, the pane's blocks in order, the selection bar painted from the row's own mark, the narrow-pane line below 60 columns, and clipping rather than scrolling"
    requirement: REND-01
    verification:
      - kind: unit
        ref: "internal/render/status_test.go#TestFrameIsExactlyThePanesSize"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestFrameTheUISpecFrame"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestThePaneDoesNotChangeWhenTheStateDoes"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestFrameTooNarrow"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestFrameClipsBlocksToThePane"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestFramePaintsTheSelectionBar"
        status: pass
    human_judgment: false
  - id: D4
    description: "The colour ladder: NO_COLOR/dumb win, then COLORTERM truecolor, then TERM 256color, then 16; mono emits no colour at all; the selection keeps reverse video unless TERM=dumb; stripping a coloured frame leaves the mono frame's characters"
    requirement: REND-04
    verification:
      - kind: unit
        ref: "internal/render/palette_test.go#TestRungFromEnvIsTheLadder"
        status: pass
      - kind: unit
        ref: "internal/render/palette_test.go#TestSGRIsTheUISpecsTable"
        status: pass
      - kind: unit
        ref: "internal/render/palette_test.go#TestNoMeaningIsCarriedByColourAlone"
        status: pass
      - kind: unit
        ref: "internal/render/palette_test.go#TestTheMonoRungEmitsNoColour"
        status: pass
      - kind: unit
        ref: "internal/render/palette_test.go#TestPaintNestsWithoutEatingTheBar"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestColourNeverCarriesMeaningAlone"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestFrameColourIsOnlyASecondStatement"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestFrameKeepsTheStateWordWhole"
        status: pass
    human_judgment: false
  - id: D5
    description: "The state and remedy words are the classifiers' own, with four documented renames, one row per failure kind, and the four OPS-01 named states spelled as the UI-SPEC's table spells them"
    requirement: REND-04
    verification:
      - kind: unit
        ref: "internal/render/state_test.go#TestEveryKindHasItsTwoRows"
        status: pass
      - kind: unit
        ref: "internal/render/state_test.go#TestTheStateWordsAreTheClassifiers"
        status: pass
      - kind: unit
        ref: "internal/render/state_test.go#TestTheRemediesAreDistinct"
        status: pass
      - kind: unit
        ref: "internal/render/state_test.go#TestTheBudgetSpentRowNamesItsSource"
        status: pass
      - kind: unit
        ref: "internal/render/state_test.go#TestTheFourNamedStatesAreTheFourOPSRows"
        status: pass
    human_judgment: false
  - id: D6
    description: "internal/xweb carries the upstream's reset instant on the classified path: rateLimitReset reads x-rate-limit-reset, APIError.Reset holds it (zero when absent), and the error's text prints the instant, never a countdown"
    requirement: REND-04
    verification:
      - kind: unit
        ref: "internal/xweb/classify_test.go#TestRateLimitResetReadsTheInstant"
        status: pass
      - kind: unit
        ref: "internal/xweb/classify_test.go#TestClassifyCarriesTheResetOnARateLimit"
        status: pass
    human_judgment: false
  - id: D7
    description: "The help overlay is one table: one row per binding in order, a 16-cell keys column, no heading, no border, no rule, no group column, and rows that fit the pane or are cut with the mark"
    requirement: REND-01
    verification:
      - kind: unit
        ref: "internal/render/help_test.go#TestHelpIsTheTable"
        status: pass
      - kind: unit
        ref: "internal/render/help_test.go#TestHelpFitsThePane"
        status: pass
      - kind: unit
        ref: "internal/render/help_test.go#TestHelpCutsAKeyListItCannotHold"
        status: pass
      - kind: unit
        ref: "internal/render/help_test.go#TestHelpColourIsOnlyASecondStatement"
        status: pass
    human_judgment: false
  - id: D8
    description: "The renderer measures only through internal/text: no fmt, no unicode/utf8, no len() on a string, no forbidden import, and the environment's measure switch is visible in the rows"
    requirement: REND-01
    verification:
      - kind: unit
        ref: "internal/render/block_test.go#TestTheMeasureFollowsTheEnvironment"
        status: pass
      - kind: other
        ref: "grep -rn --include='*.go' --exclude='*_test.go' 'fmt\\.\\|unicode/utf8' internal/render/ (no matches); every len() call is on a slice or a map"
        status: pass
      - kind: other
        ref: "grep -rn --include='*.go' --exclude='*_test.go' 'bubbletea|net/http|os/exec|internal/xsource|lipgloss|bubbles|bold' internal/render/ (no matches)"
        status: pass
    human_judgment: false

# Metrics
duration: 1h50m
completed: 2026-10-14
status: complete
---

# Phase 2: Following timeline in the pane — Plan 03 Summary

## Performance

| Metric | Value |
|--------|-------|
| Wall time | ~1h50m (single writer, three tasks in one pass) |
| Test cases in `internal/render` | 61 |
| Full suite | 15 packages, all green (`go test ./... -count=1`) |
| Non-test bytes written | 53,045 in `internal/render` (7 files) |
| Test bytes written | 95,496 in `internal/render` (5 files) |

## Accomplishments

- **`internal/render`** exists and is the phase's view: the plain grammar, the post-block layout, the
  status line, the frame composer, the colour ladder and the help table, with every user-visible string
  in `copy.go`.
- The **UI-SPEC's 80-column frame is reproduced character for character** (one body line's wrap point
  excepted, see Deviations) and the 120-column frame under the capped measure it specifies.
- The **status line** never wraps, drops fields in the documented order, keeps the budget in every
  frame, and prints the classifier's own words with four documented renames.
- The **colour ladder** answers to the environment and to nothing else, and no meaning is carried by
  colour alone (asserted by stripping, at the unit, block and frame levels).
- **`internal/xweb`** now carries the upstream's `x-rate-limit-reset` instant on the classified path,
  which is the number the rate-limited state line prints.

## The landed shapes (the plan's `<output>` contract)

```go
// internal/render/grammar.go
type Grammar struct {
	Name       string
	HandleCols int
	Stamp      StampFormat
	BodyIndent int
	MeasureCap int
	Separator  bool
	Markers    MarkerSet
	Identity   func(domain.User) string
	ShowCounts bool
}
type MarkerSet struct {
	Repost, Continued, Limited, Unavailable, MissingText, Attachment string
	QuotePrefix   string
	QuoteDepthCap int
	Truncation    string
}
type StampFormat struct{ Clock, Date string; At time.Time }
func (f StampFormat) Stamp(t time.Time) string
var Plain Grammar // Name "plain", HandleCols 16, BodyIndent 2, MeasureCap 96, Separator true, ShowCounts false
var PlainMarkers MarkerSet
const StampCells = 12

// internal/render/block.go
type Block struct{ PostID domain.PostID; Rows []string }
func Blocks(posts []domain.Post, g Grammar, w int, selected domain.PostID, now time.Time) []Block
func StateBlock(g Grammar, head, body string) Block

// internal/render/status.go
type NoticeKind uint8 // NoticeNone, NoticeNewPosts, NoticeNoNewPosts, NoticeLoadingOlder, NoticeStalled
type StatusFields struct {
	Failure domain.Failure
	Notice NoticeKind
	NoticeCount int
	List domain.HomeKind
	Selected, Loaded int
	HasMore bool
	Used, Limited int
	Now time.Time // added by this plan: the clock the reset suffix is read against
}
func StatusLine(f StatusFields, w int) string

// internal/render/frame.go
const MinPaneCols = 60
func Frame(blocks []Block, status StatusFields, w, h int) string

// internal/render/help.go
const HelpKeyCols = 16
type HelpRow struct{ Keys, Text, Group string }
func Help(rows []HelpRow, w int) []string

// internal/render/palette.go
type Rung uint8      // RungTrueColor, Rung256, Rung16, RungMono
type Role uint8      // RoleDominant, RoleMuted, RoleState, RoleStateStop, RoleSelection
type Palette struct{ /* rung, dumb */ }
func RungFromEnv() Rung
func For(rung Rung) Palette
func (p Palette) Rung() Rung
func (p Palette) SGR(role Role) string
func (p Palette) Reset() string

// internal/render/copy.go
func StateWord(kind domain.FailureKind) string
func Remedy(kind domain.FailureKind) string

// internal/xweb
func rateLimitReset(headers http.Header) (time.Time, bool)   // classify.go, unexported
type APIError struct{ Operation string; Outcome Outcome; Status int; Detail string; Reset time.Time }
```

**Copy constant names** (`internal/render/copy.go`, the only file with user-visible sentences):
`CopyEmptyHeadingFollowing`, `CopyEmptyHeadingForYou`, `CopyEmptyBody`, `CopyLoadingFollowing`,
`CopyLoadingForYou`, `CopyPaneTooNarrow`, `MarkTruncation` (`~`), `MarkSelection` (`>`),
`MarkerRepost` (`reposted by {handle}`), `MarkerLimited`, `MarkerContinued`, `MarkerUnavailable`,
`MarkerMissingText`, `MarkerQuotePrefix` (`> `), `CopyAttachment` (`attachment: {n} {noun}`),
`CopyAttachmentSeparator` (`", "`), `NounImage`/`NounImages`, `NounVideo`/`NounVideos`,
`NounGIF`/`NounGIFs`, `CopyBudgetOne`/`CopyBudgetMany`, `CopyListFollowing`/`CopyListForYou`,
`CopyPosition`/`CopyPositionMore`, `CopyNoticeNewPost`/`CopyNoticeNewPosts`/`CopyNoticeNoNewPosts`/
`CopyNoticeLoadingOlder`/`CopyNoticeStalled`, `CopyStateRateLimited`, `CopyResetUntilClock`/
`CopyResetUntilDate`, `CopyRemedyRateLimited`/`CopyRemedyRateLimitedUnknown`, and the placeholders
`PlaceholderCount` `{n}`, `PlaceholderRateLimited` `{m}`, `PlaceholderNoun` `{noun}`,
`PlaceholderHandle` `{handle}`, `PlaceholderClock` `{clock}`, `PlaceholderDate` `{date}`,
`PlaceholderSelected` `{selected}`, `PlaceholderLoaded` `{loaded}`.

**The two state tables** (`stateWords`, `remedies`, both in `copy.go`, keyed by `domain.FailureKind`,
19 rows — every kind but `FailureNone`, which has no row).

## The state-word pins and the four renames

`state_test.go` reads the classifiers: for every row whose word comes from an `xweb.Outcome` it
asserts equality with that outcome's own string, and for every `netcheck.Outcome` row the same.
The four renames are asserted to *differ* from the constant they came from, so a rename that was
never made fails too:

| Kind | Source constant | Constant's word | The copy table's word |
|------|-----------------|-----------------|------------------------|
| `FailureSessionExpired` | `xweb.OutcomeAuthRejected` | `auth rejected` | `session expired` |
| `FailureOperationStale` | `xweb.OutcomeUnknownOperation` | `unknown operation` | `operation id stale` |
| `FailureUpstreamBlocked` | `netcheck.OutcomeUpstreamBlocked` | `proxy reached but upstream blocked` | `upstream blocked` |
| `FailureNoSession` | `netcheck.OutcomeSessionSkipped` | `session not configured` | `no session file` |

Verbatim rows include `xweb.OutcomeSignerOrAuth` — the UI-SPEC's table spells this row
`auth or signer rejected`, and the plan's rule is that the classifier's spelling wins, so the row
reads `signer or auth rejected`. `FailureBudgetSpent` has no outcome: it is keyed to
`xweb.ErrRequestCap` in the row's own comment, and `TestTheBudgetSpentRowNamesItsSource` reads
`copy.go` and fails if that name leaves the comment.

## The ladder's actual check order

`RungFromEnv`, in this order (the order is the contract, and the comment in `palette.go` says why):

1. `NO_COLOR` set and non-empty → `RungMono` (the opt-out wins over any advertisement);
2. `TERM` empty or `dumb` → `RungMono`;
3. `COLORTERM` contains `truecolor` or `24bit` → `RungTrueColor`;
4. `TERM` contains `256color` → `Rung256`;
5. otherwise, with a `TERM` at all → `Rung16`.

Values: muted `#8a8a8a` / `38;5;245` / `90`; state `#d7af00` / `38;5;178` / `33`; state-stop
`#d75f5f` / `38;5;167` / `31`; selection `SGR 7`; dominant emits nothing at every rung. In `RungMono`
only the selection paints, and `TERM=dumb` silences even that. `Palette.Reset()` is `\x1b[0m` unless
the rung paints nothing at all (mono on a dumb terminal). Runs are closed with a *specific* off code
(`39` for a foreground, `27` for reverse) rather than a full reset, so a muted stamp inside the
selected block's bar does not end the bar halfway across the pane.

## The `APIError.Reset` change and what the shell plans must call

`internal/xweb/classify.go` gained `rateLimitReset(headers) (time.Time, bool)` (beside
`rateLimitRemaining`); `internal/xweb/graphql.go` populates `APIError.Reset` at the one classified
failure site and appends `reset 2026-10-14T14:32:00Z` (RFC3339, UTC) to `APIError.Error()` — an
instant, never a countdown.

For plan 02-04 (the shell): map `*xweb.APIError` by `Outcome`, copy `Detail` from
`Outcome.Remedy()` and `Reset` from `APIError.Reset` for the rate-limited kind, then render with

```go
blocks := render.Blocks(frame.Posts, render.Plain, w, frame.Selected, m.now)
status := render.StatusFields{Failure: f, Notice: …, NoticeCount: …, List: frame.Kind,
                              Selected: …, Loaded: len(frame.Posts), HasMore: frame.Next != "",
                              Used: used, Limited: limited, Now: m.now}
frame := render.Frame(blocks, status, w, h)      // exactly w×h, status line last
```

The help modal is `render.Frame([]render.Block{{Rows: render.Help(rows, w)}}, status, w, h)` — the
overlay's rows are pane rows, already carrying their selection column. A cold pane (loading, empty
read, a failure with nothing to show) is `render.StateBlock(render.Plain, head, body)` with the
words from `copy.go` (`CopyLoadingFollowing`, `CopyEmptyHeadingFollowing` + `CopyEmptyBody`) and,
for a failure, `render.StateWord(f.Kind)` as the head and `f.Detail` as the body. `Plain` is the
grammar this phase ships; `render.StateWord`/`render.Remedy` are the only way a caller names a state.

## Every literal frame the tests now hold

- **`TestBlocksTheUISpecFrameAt80Columns`** — the UI-SPEC's 80-column frame, 23 rows, transcribed
  literally with the leading selection cell: the head rows, the depth-1 quote's two rows, the repost
  head with `reposted by aergon`, the two-image attachment line, the continuation pair with no
  separator row between the blocks, the CJK/emoji/URL wrap, and the tombstone.
- **`TestBlocksTheUISpecFrameAt120Columns`** — the same list and selection at 120 columns under the
  96-cell cap.
- **`TestTheTwoGrammarsExample`** — the UI-SPEC's (A) rendering (aergon quoting lzd) as 5 literal rows.
- **`TestFrameTheUISpecFrame`** — the whole 24-row frame at 80×24, right-trimmed for reading and each
  row asserted to be exactly 80 cells: the 23 pane rows above plus
  `rate limited until 14:32  no retry  following  6 requests, 1 rate-limited` as the last row.
- **`TestStatusLineWorkedLines`** — the UI-SPEC's six worked status lines, five of them byte-exact at
  80 columns with `used = 200`, `limited = 12`.
- **`TestTheFifthWorkedLineIsWiderThanItsPane`** — the sixth line (`operation id stale …`) is 82 cells
  in the UI-SPEC's own fenced block, and is asserted at the width that holds it; at 80 columns the
  engine drops the list field, which is the third field the rule allows it to drop.
- **`TestTheMeasureFollowsTheEnvironment`** — the same list, the same width, two environments: the
  ZWJ family is 2 cells bare and 6 under a multiplexer, so the rows move. This is the arm that makes
  "no second measurement" falsifiable at the renderer.

## Evidence

| Command | Result |
|---------|--------|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | all 15 packages ok (render 61 cases) |
| `go test ./internal/render/... -count=1` under `env -u NO_COLOR -u TMUX TERM=xterm-256color`, `NO_COLOR=1 TERM=dumb`, `TMUX=… TERM=screen-256color`, `NO_COLOR=1 TMUX=… TERM=dumb`, `env -u NO_COLOR -u TERM` | ok in all five |
| `gofmt -l internal cmd` | no output |
| `grep -rn --include='*.go' --exclude='*_test.go' 'lipgloss\|bubbles\|bold' internal/render/` | no matches |
| `grep -rn --include='*.go' --exclude='*_test.go' 'bubbletea\|net/http\|os/exec\|internal/xsource' internal/render/` | no matches |
| `grep -rn 'fmt\.\|unicode/utf8' internal/render/*.go` (non-test) | no matches; every `len()` call is on a slice or a map |
| `go test ./internal/render/ -count=1 -v \| grep -c 'PASS'` | 61 |

Anti-vacuity, by hand, each break applied to the shipped code, watched to fail, then reverted
(each revert verified by the suite going green again in the same run):

1. a verbatim state word changed (`endpoint retired` → `endpoint retired!`):
   `TestTheStateWordsAreTheClassifiers` fails by name and value.
2. a rename reverted to its constant (`session expired` → `auth rejected`): the same test fails with
   *"the row reads "auth rejected", which is the constant itself: the rename is missing"*.
3. the notice removed from the droppable list: `TestStatusLineDropsInOrder` fails (the budget is cut
   at 88 columns instead).
4. the measure cap disabled: `TestBlocksTheUISpecFrameAt120Columns` fails (21 rows, want 22).
5. the truncation mark dropped from `cutField`: `TestABlockRowNeverExceedsThePane` fails on the cut
   head row.
6. the renderer stops wrapping (one line per body): `TestTheMeasureFollowsTheEnvironment` fails with
   *"every row is identical in both environments"*.
7. the reset instant never read from the response: `TestClassifyCarriesTheResetOnARateLimit` fails
   (*"the error carries the reset 0001-01-01 …"*); and `rateLimitReset` made to return the zero time:
   `TestRateLimitResetReadsTheInstant` fails.
8. a coloured run made to change the characters (a leading space in the truecolor muted escape):
   `TestFrameColourIsOnlyASecondStatement` fails on the strip comparison; the selection mark dropped
   from the row: `TestSelectionIsTheMarkAndOnlyTheMark` fails.

Two arms were *first written wrong* and are worth recording: adding a cell inside `paneRow`'s
argument changes both rungs, so the A1 comparison could not see it (the break has to be inside the
painting path); and a fixed measure (`text.Wrap(s, 77)`) still moves with `$TMUX`, because the
cluster widths inside `Wrap` follow the environment — so the environment arm's break had to remove
the wrap, not the number.

## Deviations from the plan / the UI-SPEC (reported, not silently adjusted)

1. **The UI-SPEC's 80-column frame breaks one body line one token early.** Its first body line is
   `… on a cursor.` (74 cells of text) although the next token fits the measure exactly
   (74 + 1 + 2 = 77 = `min(80-3, 96)`). The engine puts `if` on that line, so the first post's body
   is 2 lines ending `… a cursor. if` / `… the walk.` — every other row of the frame is
   byte-identical. Recorded with its numbers in
   `TestTheFrameDivergencesFromTheUISpecAreTheMeasuredOnes`.
2. **The UI-SPEC's 120-column frame ignores its own 96-cell cap.** Its fenced rows carry 114 and 109
   cells of text, wider than the cap and wider than what `A7` audits; a renderer that reproduced them
   would be ignoring the cap at exactly the width the cap exists for. The engine wraps at
   `min(W-3, 96)`, which is what the plan's measure formula and `A7` require; the two fenced rows and
   the rows the cap produces are both asserted, with the cell counts.
3. **The fifth worked status line does not fit 80 columns.** It is 82 cells in the UI-SPEC's own
   block; the plan's own rule ("after all three droppable fields are gone, truncate") cannot make it
   fit without dropping `list`, which the drop rule does at 80 columns. Both the literal line (at the
   width that holds it) and the 80-column output are asserted.
4. **`StatusFields` gained a `Now time.Time`.** The plan's struct has no clock, but the
   `until 14:32` / `until Jan 02 15:04` rule needs a reference day, and reading the wall clock inside
   the renderer would make a frame non-reproducible (which plan 05 forbids). A zero `Now` takes the
   dated form, which is true whenever it is read. The shell must pass its injected clock.
5. **`StampFormat.Stamp` returns *at least* 12 cells, not exactly 12.** The plan's two forms are 12
   cells each, which is what the plain grammar needs; a cover whose instant is an ISO timestamp (20
   cells) keeps its own width instead of being cut — the UI-SPEC's own (B) rendering requires it.
6. **A tombstone's head row is blank, not `Jan 01  0001`.** The zero instant is the absence of a
   date; printing a date the upstream never sent would be the pane lying.
7. **`Withheld` renders like `Unavailable`.** The plan gives no rule for a withheld post and the
   copy table has no word for it, so it takes the tombstone's line rather than inventing copy. No
   recorded response carries one yet.
8. **`StateBlock` was added.** The plan lists no helper for the pane's loading/empty/cold-failure
   states, but the UI-SPEC requires them "in the same block shape" and the shell may not compose the
   indent bytes itself ("the shell adds no bytes"). It is one exported function in `block.go`.
9. **The attachment template is read as shape.** `attachment: {n} {noun}` printed once per kind
   would produce `attachment: 1 image, attachment: 1 video`; the words before the first `{n}` are
   printed once and the rest repeats, which is what the UI-SPEC's own template notation shows.
10. **`ShowCounts` is declared and false everywhere; nothing in this phase reads it.** There is no
    copy for an engagement line in the copywriting contract, so implementing one would mean inventing
    words. The field is the switch a Phase 3 cover flips, and `block_test.go` asserts it is false.
11. **The palette is read from the environment at paint time** (per call), not once into a variable:
    the environment of a process cannot change under it, so it is still one decision, and this is
    what lets the tests pin a rung with `t.Setenv` and lets plan 05's corpus store one frame per rung.
12. **`Blocks` and `Help` paint their own runs**, and `Frame` paints only the selection bar (from the
    row's own `>` mark, after the padding, so the bar runs to the pane's edge). A caller that wants
    plain rows pins the environment; a caller that wants colour strips, which is what every colour
    assertion does.

## Files Created/Modified

Created: `internal/render/{copy,grammar,block,status,frame,palette,help}.go` and
`internal/render/{block,status,palette,help,state}_test.go`.
Modified: `internal/xweb/classify.go` (the reset reader), `internal/xweb/graphql.go` (`Reset` field,
`Error()` text, population), `internal/xweb/classify_test.go` (the two new tests). Nothing outside
`files_modified` was touched, and no file under `.planning/` other than this summary.

## Issues Encountered

- `t.Setenv` makes the measure live mid-test: a frame captured in the bare arm and *measured* after
  switching `$TMUX` is a different number (plan 02-02 hit the same trap). The environment test now
  records each arm's widths while its own environment is set.
- The block builder paints, so a list built under one rung and rendered under another is not a render
  of that rung: `TestFrameColourIsOnlyASecondStatement` builds its blocks inside each environment.

## User Setup Required

None — no new dependency, no configuration, no external service. The module is unchanged.

## Next Phase Readiness

- Plan 02-04 (the shell) can build its `Render(w, h)` from `Blocks` + `StatusFields` + `Frame`, its
  help overlay from `Help`, and its cold panes from `StateBlock` + `StateWord`; it must pass its
  injected clock as `StatusFields.Now`.
- Plan 02-05 (the golden corpus) can render every frame through `Frame` with the environment pinned
  (`TMUX`, `TERM`, `COLORTERM`, `NO_COLOR` — the renderer reads all four), and should render the
  `narrow_79` frame at `render.MinPaneCols - 1` columns if it expects the narrow-pane line, since the
  floor is 60 by the UI-SPEC's own rule.
- Phase 3 can add a cover by writing another `Grammar` value: `Identity`, `Stamp`, `Markers`,
  `Name`, `ShowCounts` and the palette are the seams, and `TestTheGrammarIsTheOnlyThingThatChanges`
  is the test that shows a grammar change moves the chrome and leaves the content alone.
