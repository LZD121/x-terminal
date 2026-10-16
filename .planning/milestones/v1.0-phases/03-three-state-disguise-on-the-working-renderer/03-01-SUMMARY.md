---
phase: 03-three-state-disguise-on-the-working-renderer
plan: 01
subsystem: ui
tags: [grammar-seam, cover-grammars, alias-function, panic-frame, covered-copy, status-line, DSGT-06, DSGT-07, DSGT-09, A6, golden-frames]

# Dependency graph
requires:
  - phase: 02-03
    provides: internal/render — Blocks/StateBlock, Frame, StatusFields, the help table, the palette, Plain and its measure; Phase 3's cover is a value handed to that one composer
  - phase: 02-02
    provides: internal/text — Width/Wrap/Pad/Truncate, still the only measurement any frame is made with
  - phase: 02-01
    provides: internal/domain (Post/User/PostID/HomeKind/Visibility/FailureKind) and internal/screen.List
provides:
  - "internal/render/grammar.go: the template/placement seam — Head, QuoteHead, QuoteAsRecord, MarkerPlace, PaneHeader, BodyIndent, BodyPrefix, AttachmentRow, MeasureCap, Separator, Stamp (with StampFormat.Location), Markers, Identity func(domain.User) CoverAlias, Ref, ShowCounts; HandleCols deleted"
  - "internal/render/block.go: the placeholder expander, head/marker/body placement and Blocks/StateBlock reading the seam; no style name is compared anywhere in it"
  - "internal/render/alias.go: CoverStyle, CoverAlias, CoverAliasOf, the FNV-1a fingerprint and the four frozen word lists"
  - "internal/render/cover.go: ServiceLog/GitLog/CI/Internal Grammar values, the four ids, titles, CoverStyles/CoverStyleOf and the per-style Ref functions"
  - "internal/render/panic.go: PanicFrame(w, h), the 21 lines, the three columns, CopyTitlePlain and CopyTitlePanic"
  - "internal/render/copy.go + status.go: the covered copy table, the per-style list names, the covered notices/empty/loading strings, HostSubstituted, and StatusFields.Cover"
affects: [03-02, 03-03, phase-04, phase-05, phase-06, phase-07, phase-08]

# Actuals — chars/4 over the realized diff (tracked insertions + the new files)
actuals:
  tokens: 44000
  tasks: 3
  commits: 0

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A display style is a Grammar value: one composer (Blocks), five grammars, and no branch on a style's name anywhere — asserted by renaming a grammar and re-rendering"
    - "The identity seam answers with the alias's four projections at once, so a style picks a slot and never spells an account"
    - "Purity by shape: internal/render cannot write (no io/os-exec/path/math-rand/maphash, no os call but Getenv, no time.Now), proven by a test that parses the package's own source with anti-vacuity arms"
    - "Every visible string is a named constant reached through a lookup that takes the style, so a covered frame cannot print the plain words by accident"
    - "Absence assertions armed with the strings they scan for: every word a frame must not hold is first shown to be in the fixture's content or in a feed rendering"

key-files:
  created:
    - internal/render/alias.go
    - internal/render/alias_test.go
    - internal/render/cover.go
    - internal/render/cover_test.go
    - internal/render/panic.go
    - internal/render/panic_test.go
  modified:
    - internal/render/grammar.go
    - internal/render/block.go
    - internal/render/block_test.go
    - internal/render/copy.go
    - internal/render/status.go
    - internal/render/status_test.go
    - internal/render/corpus_test.go
    - internal/render/state_test.go

key-decisions:
  - "HandleCols is deleted and the plain grammar's 16-cell column is `{handle:16}` in Head: one mechanism (the template plus text.Pad) places every field of every head"
  - "AttachmentRow is a Grammar field (beyond the plan's field table): the plain grammar's attachment is a body row while its markers are inline, and one MarkerPlace cannot express both"
  - "The A6 violation was fixed in the scrub, not in the rule: HostSubstituted is now a byte-run scan with no len() on any string; the scan, its arms and its rules are untouched"
  - "The panic frame is its own function (PanicFrame), as the plan's task text and the UI-SPEC require: Frame always appends a status line and the panic frame owns rows 1..H. It still uses the shared measure (text.Wrap/text.Pad) and the plain reader's own floor copy"
  - "CopyTitlePlain/CopyTitlePanic live in panic.go and CoverStyle.Title() returns them, so the name of the thing on screen is spelled once"
  - "internal/render may call os.Getenv (Phase 2's colour ladder) and nothing else on os; TestRenderCannotWrite asserts exactly that rather than 'no os import'"
  - "The gitlog stamp follows git's own format, so the UI-SPEC's fenced `Mon Oct 14 … 2026` renders as `Wed Oct 14 … 2026`: the format is the specification and the example text in the fence is not"
  - "At 80 columns the covered `operation id stale` line is byte-identical to the plain one: the only covered difference on that line is the list field, which both drop there"

requirements-completed: [DSGT-06, DSGT-07, DSGT-09]

coverage:
  - id: D1
    description: "The template/placement seam: Grammar extended, HandleCols deleted, StampFormat.Location, the three MarkerPlace values, the closed placeholder set with {x:N} padding, marker/head/body placement, PaneHeader rows"
    requirement: DSGT-09
    verification:
      - kind: unit
        ref: "internal/render/block_test.go#TestThePlaceholderExpander"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheMeasureIsTheGrammarsOwn"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheMarkerPlaceIsWhereTheMarkersLand"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestTheQuoteIsARecordOrANestedBlock"
        status: pass
      - kind: unit
        ref: "internal/render/block_test.go#TestThePaneHeaderIsAContentRow"
        status: pass
      - kind: integration
        ref: "go test ./internal/render/... -count=1 -run TestGoldenFrames (no -update)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The deterministic, total alias function in a package that cannot write"
    requirement: DSGT-07
    verification:
      - kind: unit
        ref: "internal/render/alias_test.go#TestCoverAliasWorkedExamples"
        status: pass
      - kind: unit
        ref: "internal/render/alias_test.go#TestCoverAliasIsStableAndTotal"
        status: pass
      - kind: unit
        ref: "internal/render/alias_test.go#TestCoverWordLists"
        status: pass
      - kind: unit
        ref: "internal/render/alias_test.go#TestRenderCannotWrite"
        status: pass
    human_judgment: false
  - id: D3
    description: "The four covers as Grammar values, and their eight worked renderings byte for byte at 80 and 120 columns, with the identity replaced by the alias"
    requirement: DSGT-06
    verification:
      - kind: unit
        ref: "internal/render/cover_test.go#TestCoverRenderings"
        status: pass
      - kind: unit
        ref: "internal/render/cover_test.go#TestCoverGrammarsAreValues"
        status: pass
      - kind: unit
        ref: "internal/render/cover_test.go#TestCoveredRenderingsHideTheIdentity"
        status: pass
      - kind: unit
        ref: "internal/render/cover_test.go#TestTheStyleVocabulary"
        status: pass
      - kind: unit
        ref: "internal/render/cover_test.go#TestNoTabByteUnderAnyGrammar"
        status: pass
    human_judgment: false
  - id: D4
    description: "The panic frame: one constant frame of H rows of W cells, no feed character, no ESC byte, no clock, no prompt, no product noun"
    requirement: DSGT-09
    verification:
      - kind: unit
        ref: "internal/render/panic_test.go#TestPanicFrameIsExactlyHRowsOfWCells"
        status: pass
      - kind: unit
        ref: "internal/render/panic_test.go#TestPanicFrameIsConstantAboveTheLongestLine"
        status: pass
      - kind: unit
        ref: "internal/render/panic_test.go#TestPanicFrameShowsItsLastRowsWhenTheContentIsTallerThanThePane"
        status: pass
      - kind: unit
        ref: "internal/render/panic_test.go#TestPanicFrameIsThePlainReadersFloorLineBelowThePaneFloor"
        status: pass
      - kind: unit
        ref: "internal/render/panic_test.go#TestPanicFrameHoldsNoCharacterOfTheFeed"
        status: pass
    human_judgment: false
  - id: D5
    description: "The covered status line: Phase 2's order and drop rule with the two named copy changes and only those, and the host substitution"
    requirement: DSGT-09
    verification:
      - kind: unit
        ref: "internal/render/status_test.go#TestCoveredStatusLinesWorked"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestTheCoveredDropCostsTheListAtEightyColumns"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestTheHostSubstitution"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestNoCoveredStatusLineNamesTheHost"
        status: pass
      - kind: unit
        ref: "internal/render/status_test.go#TestTheCoveredCopyIsTotal"
        status: pass
    human_judgment: false
  - id: D6
    description: "A6 (internal/text is the only measure) still holds in internal/render after the scrub was rewritten"
    requirement: DSGT-06
    verification:
      - kind: integration
        ref: "go test ./internal/tui/ -count=1 -run TestNoSecondMeasurement"
        status: pass
      - kind: integration
        ref: "planted len(CopyUpstreamHost) in internal/render/copy.go: the scan failed; reverted and re-run green"
        status: pass
    human_judgment: false
---

# Phase 3: Three-State Disguise on the Working Renderer Summary (plan 01, render side)

## Performance

3 tasks in scope; all three complete. 546 tests pass in 18 packages (131 of them in `internal/render`), from
533 passing + 2 failing + 1 skipped when this session picked the tree up. `go build ./...`, `go vet ./...`
and `gofmt -l internal cmd` are all clean.

## Accomplishments

The plan's three tasks, and what the previous (killed) agent had left:

**Already written and correct — checked, not rewritten.** The seam in `grammar.go` (every field the
UI-SPEC's table names, `HandleCols` deleted, the plain grammar's `{handle:16}` reproducing Phase 2's
bytes), the placeholder expander and placement logic in `block.go`, `alias.go` with its fingerprint and
frozen lists, `cover.go` with the four grammars and the eight worked renderings, `panic.go`, the covered
copy in `copy.go`, `StatusFields.Cover` in `status.go`, and the tests `alias_test.go`, `cover_test.go`.
Two defects in that work were found and fixed (below), and the plan's task-3 tests were missing entirely.

**Finished in this session**

1. **`internal/render/panic_test.go` — new, the plan's whole task-3 panic evidence.** The size table
   (80/120/60/79/200 × 24, H = 12 and 40, both sides of the pane's floor), H rows of exactly W cells with
   not one ESC byte, the degenerate `H = 0` frame and the `H = 1` frame, the constant-frame property with
   every line held to the frame's own three columns, the scroll rule (`H` rows from the bottom, no
   truncation mark) and the wrap rule (reached with a line of the test's own, because all 21 shipped
   lines are 46 cells and the floor is 60 — the fact is asserted and the branch is exercised), the floor
   branch compared against the plain reader's own `Frame` floor row, and the absence scan for handles,
   display names, body words, aliases, the host, the product's own nouns, a prompt, a path and a clock.
2. **`internal/render/status_test.go` — the covered status line (task 3).** The UI-SPEC's six worked
   covered lines at 80 columns byte for byte; the measured drop consequence (the list disappears in three
   lines under a cover against one in the plain reader, and in none at 120 columns); the substitution's
   fifteen cases with three armed counts; the host absence over the closed taxonomy × four covers with
   the plain reader as the arm; and the totality of the covered copy over every state, feed and notice.
3. **`internal/render/cover_test.go` — the style vocabulary** (ids, frozen order, `CoverStyleOf`
   refusing everything that is not a style, the four titles and both non-cover titles as literals), and
   the tab-byte scan extended to the stored corpus.
4. **The A6 violation, fixed in the code.** See below.
5. **The wrap-case expectation in `TestThePlaceholderExpander`**, which the killed agent had left two rows
   short (it listed the head's two wrapped rows and forgot the body's two), and the stale `HandleCols`
   mention in a `block_test.go` comment, which made the plan's own `grep` check non-empty.

## The seam as landed (`internal/render/grammar.go`)

`Grammar`: `Name`, `Head []string`, `QuoteHead []string`, `QuoteAsRecord bool`, `MarkerPlace MarkerPlace`,
`PaneHeader []string`, `BodyIndent int`, `BodyPrefix string`, `AttachmentRow bool`, `MeasureCap int`,
`Separator bool`, `Stamp StampFormat`, `Markers MarkerSet`, `Identity func(domain.User) CoverAlias`,
`Ref func(domain.PostID) string`, `ShowCounts bool`. **`HandleCols` is deleted** and appears nowhere in
the tree.

`StampFormat`: `Clock`, `Date`, `Location *time.Location` (nil = the clock's own zone), `At time.Time`
(filled from `Blocks`'s `now`, so a grammar carries no clock). `Stamp` renders `t.In(location)` where the
location is `Location` when set and `At.Location()` otherwise, picks the clock form when the stamp falls
in the clock's own year, and renders the zero instant as `StampCells` of nothing.

`MarkerPlace`: `MarkersInline`, `MarkersHeadLines`, `MarkersBodyLines`.

`MarkerSet`: `Repost`, `Continued`, `Limited`, `Unavailable`, `MissingText`, `Attachment`, `Parent`,
`StateInfo`, `StateWarn`, `StateError`, `QuotePrefix`, `QuoteDepthCap` (2), `Truncation` (`~`).

Placeholders (closed set, `copy.go`): `{stamp}`, `{ref}`, `{handle}`, `{name}`, `{email}`, `{markers}`,
`{state}`, `{text}`; `{x:N}` pads with `text.Pad` and cuts with `~`. A template line that is the empty
string is a blank row; a line whose only placeholder is empty is dropped; a line wider than `W-1` wraps
with `text.Wrap` at the head's own indent; no tab byte is ever emitted.

Measure: `min(W - 1 - BodyIndent - 2·depth - text.Width(BodyPrefix), MeasureCap)`; the plain grammar's
`BodyIndent == 2`, `BodyPrefix == ""` reproduces Phase 2's `min(W - 3 - 2·depth, 96)`.

## The five grammars, exactly as landed

| Field | plain | servicelog | gitlog | ci | internal |
|---|---|---|---|---|---|
| `Name` | `plain` | `servicelog` | `gitlog` | `ci` | `internal` |
| `Head` | `["{handle:16} {stamp:12}{markers}"]` | `["{stamp} {state}  svc={handle} seq={ref}{markers}"]` | `["commit {ref}", "Author: {name} <{email}>", "Date:   {stamp}", ""]` | `["[{stamp}] build={ref} job={handle}{markers}"]` | `["{ref:8}  {state:7}  {handle:20}  {stamp}"]` |
| `QuoteHead` | as `Head` | as `Head` (unused) | `["{name}  {stamp}"]` | `["[{stamp}] build={ref} job={handle}"]` | as `Head` |
| `QuoteAsRecord` | false | **true** | false | false | false |
| `MarkerPlace` | `MarkersInline` | `MarkersInline` | `MarkersHeadLines` | `MarkersInline` | `MarkersBodyLines` |
| `PaneHeader` | — | — | — | — | `["id        state    owner                 updated"]` |
| `BodyIndent` / `BodyPrefix` | 2 / `""` | 2 / `""` | 4 / `""` | 0 / `"\| "` | 2 / `""` |
| `AttachmentRow` | true | false | false | false | false |
| `MeasureCap` / `Separator` | 96 / true | 96 / false | 96 / true | 96 / false | 96 / false |
| `Stamp` | `Jan 02 15:04` / `Jan 02  2006` | `2006-01-02T15:04:05Z`, **UTC** | `Mon Jan 2 15:04:05 2006 -0700` | `15:04:05` (local) | `Jan 02 15:04` |
| `Ref` | nil | `seq` = `1000000 + H(id) mod 9000000` | `sha` = 12 lowercase hex of `H(id)` | `build` = `1000 + H(id) mod 9000` | `REQ-` + the same build number |
| `Identity` | the handle | the alias's handle | the alias's `Name`/`Email` | the alias's handle | the alias's login |
| `ShowCounts` | false | false | false | false | false |

Markers: servicelog ` replayed_by={handle}`, ` follows={ref}`, ` parent={ref}`, ` att={n} {noun}`, levels
`INFO`/`WARN`/`ERROR`, bodies `record withdrawn`/`no payload`; gitlog `commit`/`Author:`/`Date:` header,
`Replayed-by: {name}`, `Continued-from: {ref}`, `Visibility: restricted`, `Files: {n} {noun}`,
`record withdrawn`/`(no message)`, quote `    > `; ci ` origin={handle}`, ` follows={ref}`,
` note=partial access`, ` artifacts={n} {noun}`, `no output captured`/`(no output)`, gutter `| `, quote
`| > `; internal `from {handle}`, `follows {ref}`, `files {n} {noun}`, `record withdrawn`/`no text`,
states `open`/`limited`/`closed`, and the pane header above.

## The alias function (`internal/render/alias.go`)

`type CoverStyle string` (declared here, the four values in `cover.go`; the zero value is the plain
reader). `type CoverAlias struct{ Handle, Name, Login, Email string }` and
`func CoverAliasOf(style CoverStyle, u domain.User) CoverAlias`.

**The key**: `u.ID`; else `strings.ToLower(strings.TrimPrefix(u.ScreenName, "@"))`; else the literal
`"\x00anonymous"`.

**The fingerprint**: `coverDigest(purpose, key string) uint64` = FNV-1a 64-bit
(`offset 14695981039346656037`, `prime 1099511628211`) over `"xt-cover/v1/" + purpose + "\x00" + key`, in
that order. Purposes: `slug`, `name` (the identity), `seq`, `sha`, `req` (the reference tokens).
No `hash/maphash` (seeded per process), no `math/rand`, no clock, no map iteration, no list — asserted by
`TestRenderCannotWrite`.

**The projections** — `h1 = coverDigest("slug", key)`, `h2 = coverDigest("name", key)`:
`Handle = ROLES[h1 mod 24] + "-" + NOUNS[(h1>>8) mod 64] + "-" + twoDigits((h1>>16) mod 100)`;
`Name = FIRST[h2 mod 48] + " " + SURNAMES[(h2>>8) mod 64]`; `Login` = the lowercased `first.last`;
`Email = Login + the two digits + "@corp.internal"`. The style argument is not read: all four projections
are computed for every style, and a style's identity function picks a slot.

**Frozen lists** (fixed-length arrays, ASCII, no spaces): `coverRoles` 24, `coverNouns` 64, `coverFirst`
48, `coverSurnames` 64. The longest slug is 18 cells and the longest login 17, which is why a cover's
identity column is 20 — both asserted.

**The two worked examples** (asserted byte for byte for all four styles, `TestCoverAliasWorkedExamples`):

| account | Handle | Name | Login | Email |
|---|---|---|---|---|
| `aergon`, id `44196397` | `agent-rollback-70` | `Anya Karlsson` | `anya.karlsson` | `anya.karlsson70@corp.internal` |
| `lzd`, id `14706153` | `guard-migrate-48` | `Clara Sorensen` | `clara.sorensen` | `clara.sorensen48@corp.internal` |

## The panic frame's constants (`internal/render/panic.go`)

`panicStatusCol = 0`, `panicPackageCol = 8`, `panicDurationCol = 40`, `panicPackageCells = 32` (the
package field, all four multiples of 4, none sized from content). `panicLines` is a `[21]string` of
`ok <package> <duration>` lines of the fictional `corp/...` monorepo — **every line is exactly 46 bytes**,
the status field in bytes 0..7, the 32-cell package field in bytes 8..39, the duration from byte 40, and
the spacing is spaces only. `CopyTitlePlain = "x-terminal"`, `CopyTitlePanic = "make"`.

`PanicFrame(w, h)`: `h < 1` → `""`; `w < MinPaneCols` (60) → `text.Pad(cutField(" "+CopyPaneTooNarrow, w,
"~"), w)` on row 1 and blank rows under it, still H rows; otherwise the run, each line wrapped with
`text.Wrap` when it does not fit, trimmed to its **last** H rows when the content is taller (no
truncation mark), blank rows below it, every row padded to W. Not one ESC byte. The 80- and 120-column
forms are byte-identical over the lines the frame draws.

## The covered copy (`internal/render/copy.go`)

`CopyCoverEmptyHeading = "no records in this view"`, `CopyCoverEmptyBody = "the read succeeded and found
nothing - press r to refresh"`, `CopyCoverLoading = "loading records..."`, `CopyCoverLoadingOlder =
"loading older records"`, `CopyCoverNoticeNewRecord(s) = "{n} new record"` / `"{n} new records"`,
`CopyCoverNoticeNoNewRecords = "no new records"`, `CopyCoverNoticeStalled = "no older records (cursor
stalled)"`; list names `stream=in`/`stream=all`, `branch=main`/`branch=next`, `queue=ci`/`queue=next`,
`view=mine`/`view=all`; `CopyUpstreamHost = "x.com"`, `CopyUpstreamHostWord = "upstream"`. The lookups are
`EmptyHeading(cover, kind)`, `EmptyBody(cover)`, `LoadingLine(cover, kind)`,
`NoticeLine(cover, notice, count)`, `ListName(cover, kind)` — each takes the style, so a caller cannot pick
the wrong vocabulary, and the zero style returns Phase 2's constants byte for byte.

`StatusFields.Cover CoverStyle` (zero = plain reader): the state word's one rename
(`no route to x.com` → `no route to upstream`, via `HostSubstituted`), the style's list name, and the
covered notices. The remedy column is deliberately untouched (the UI-SPEC's named residue), and the field
order, the drop rule (`notice`, `position`, `list`, in that order) and the never-dropped fields (`state`,
`remedy`, `budget`) are Phase 2's.

## The A6 violation, and how it was fixed

`TestNoSecondMeasurement` (`internal/tui`) reported `internal/render/copy.go: measures the string
CopyUpstreamHost with len()`. The old scrub found host occurrences with `for at := 0; at < len(s);` and
suffix-matched with `run[len(run)-len(CopyUpstreamHost):]`, then checked labels with `len(label)`.

The code was fixed, never the rule. `HostSubstituted` is now a scan over the string's own runs of
hostname-label bytes: `for at := range s` walks the byte offsets of each rune, a run is remembered where it
starts and handed to `isHostRun` where it ends (or at the end of the string), `isHostRun` compares with
`strings.ToLower`/`HasSuffix`/`TrimSuffix` and `strings.Split` instead of counting bytes, and
`wellFormedLabel` uses `strings.HasPrefix`/`HasSuffix` and `for at := range label`. **`copy.go` now
contains no `len(` call at all** (`grep -n "len(" internal/render/copy.go` → no output), which is stricter
than the scanner's own rule about declared string names. The semantics are unchanged, including the
negative cases (`notx.com`, `x.comfoo`, `..x.com`, `-a.x.com`, `x.competitor`) and the no-op for a string
with no host in it. No exemption, no allowlist, no edit to the scanner, and no change to its arms: the
planted violation below still fails it.

## Tests that were made to fail, by hand

| What was planted | What was run | Result |
|---|---|---|
| `len(CopyUpstreamHost)` in `internal/render/copy.go`'s `isHostRun` | `go test ./internal/tui/ -run TestNoSecondMeasurement` | FAIL: `internal/render/copy.go: measures the string CopyUpstreamHost with len()` — A6's arms are intact after the rewrite |
| `import "io"` + `_, _ = io.WriteString(io.Discard, time.Now().String())` in `copy.go` | `go test ./internal/render/ -run TestRenderCannotWrite` | FAIL: `internal/render/copy.go imports io` and `… calls time.Now` |
| a handle and host appended to `panicLines[0]` | `go test ./internal/render/ -run TestPanicFrameHoldsNoCharacterOfTheFeed` | FAIL: `holds "aergon" …` and `holds the product's own word "x.com"` at every width |
| `CI.Identity` returning the account's real handle | `go test ./internal/render/ -run TestCoveredRenderingsHideTheIdentity\|TestCoverRenderings` | FAIL, rendered: `[09:12:00] build=4655 job=aergon` where the alias belongs — the no-real-identity claim is a scan of rendered frames and not a reading of the diff |
| (Phase 2's) `-update` never run | `go test ./internal/render/ -run TestGoldenFrames` | PASS: the plain reader's stored frames are byte-identical after the seam change |

Every plant was reverted and the affected test re-run green.

## Deviations from the plan, and things a reviewer should know

1. **`AttachmentRow` is a new `Grammar` field**, not in the plan's field table. The plain grammar's
   attachment is a body row (`attachment: 2 images`) while its three markers are inline, and the plan's
   single `MarkerPlace` cannot express both. The `MarkerSet.Attachment` doc already distinguished the two
   ("a body line of its own under the plain grammar, one of the record's markers under a cover"), so the
   field is the honest minimum.
2. **`PanicFrame` is a function of its own.** The plan's task text defines it that way and the UI-SPEC's
   panic frame owns rows 1..H with no status line, which `Frame` (which always appends one) cannot do. The
   plan's must-have sentence that "the panic frame [is] composed by the same Blocks/Frame functions" is
   therefore read as "no second *content* path": the frame uses the shared measure (`text.Wrap`/`text.Pad`)
   and the plain reader's own floor copy, and draws no character of the feed.
3. **The gitlog weekday.** The UI-SPEC's fenced gitlog block reads `Mon Oct 14 09:12:00 2026 +0000`, but
   2026-10-14 is a Wednesday; `Mon Jan 2 15:04:05 2006 -0700` is git's own format and the format is what
   the style is pinned to. The rendering is `Wed Oct 14 …` and the divergence is held by
   `TestTheGitDateDivergesFromTheUISpecsWeekday` (with the instant's weekday asserted, so the test cannot
   rot silently).
4. **The covered `operation id stale` line is the plain line at 80 columns.** Both states drop the list
   there (the plain line is 82 cells, Phase 2's already-recorded arithmetic issue with that fence), and the
   list name is the only covered difference on it, so the two lines coincide. The test's arm says exactly
   that instead of demanding a difference that cannot exist.
5. **The UI-SPEC's six worked covered lines all hold at 80 columns** — they were transcribed and pass
   byte for byte. The named consequence holds too, measured: the list field is dropped in three of the six
   under a cover (`session expired`, `no route`, `operation id stale`) against one in the plain reader
   (`operation id stale`).
6. **`TestRenderCannotWrite` forbids `os/exec` and every `os` call except `os.Getenv`,** because Phase 2's
   colour ladder reads the environment and nothing else in the package touches `os`. The plan's must-have
   truth ("imports no `os`") is narrower than the landed rule; the reason is recorded in the test's own
   comment and the arm asserts that the `os.Getenv` call really is there.
7. **The two-namespace agreement test lives in `alias_test.go`,** not `cover_test.go` as the plan's task-1
   text said: `TestCoverAliasWorkedExamples` loops over all four styles and asserts the same four
   projections for one account, which is the same claim with one expectation instead of four.
8. **The panic frame's wrap branch is unreachable with the shipped lines** (longest 46 cells, floor 60).
   The rule is still implemented and is exercised by a test that injects a longer line into `panicLines`
   and restores it with `t.Cleanup`; the fact that the shipped content cannot reach it is asserted, not
   hidden.
9. **Not verified: the stored golden files' history.** No git command was run in this session and the work
   is uncommitted, so "unchanged by this plan's commits" could not be a diff. Instead: `TestGoldenFrames`
   passes with no `-update`, and all 31 `.golden` files share one mtime (2026-10-14 18:20:40) that
   predates every source edit of this plan (18:49–19:25), so none of them was regenerated by this work.
10. **Out of this plan's scope:** nothing in `internal/tui` or `internal/screen` yet wears a cover, switches
    a state or calls `PanicFrame`; the shell's wiring, keymap rows and config are plans 03-02/03-03's.
11. **Where `len()` of a string still appears, and why it is not a width.** The plan's rule is enforced
    by the A6 scanner, which by its own documented terms collects declared string names (fields, vars,
    consts and string-returning functions) and not function parameters or range variables. After the
    scrub's rewrite, `internal/render/copy.go` has no `len(` at all; the two remaining byte-offset loops
    are `parsePlaceholders`/`expand` in `block.go` (finding `{` and `}` in a template) and the FNV-1a
    byte walk in `alias.go` (`for i := range len(part)`) — a byte count of a hash input and of a byte
    offset in a template, neither of them a width, and neither reachable by the scanner's rule as
    written. Flagging them would mean widening the rule (collecting parameters), which is a change to
    the rule and not a fix to the code, so it was left alone and recorded here.

## Verification as run

The plan's own `<verification>` block, for real:

| Check | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0, no output |
| `gofmt -l internal cmd` | empty |
| `go test ./... -count=1` | **546 passed in 18 packages** (0 failed, 0 skipped) |
| `go test ./internal/render/... -count=1` | 131 passed |
| `go test ./internal/render/... -count=1 -run 'Golden\|Corpus'` | 40 passed, **without `-update`** |
| `go test ./internal/render/... -count=1 -run 'TestGoldenFrames\|TestPlaceholder\|TestMeasure\|TestCoverAlias\|TestRenderCannotWrite\|TestNoTab'` | 36 passed |
| `go test ./internal/... -count=1` | 546 passed in 15 packages |
| `go test ./internal/render/... -count=1 -run 'TestCover\|TestPlain'` | 7 passed |
| `go test ./internal/render/... -count=1 -run 'TestPanicFrame\|TestCovered\|TestHostSubstitution\|TestGoldenFrames'` | 39 passed |
| `grep -rn "HandleCols" internal/ cmd/` | no matches (exit 1) |
| `sha256sum internal/render/testdata/TestGoldenFrames/*.golden \| sha256sum` | `843b3315d556d6c37f2850bd56dc8f4afb0a4ff30789e79341dfae6b794254ed` (31 files) |
| `grep -n "len(" internal/render/copy.go` | no matches — the scrub counts no string |
| the five plants above | each test failed, each file reverted, each test green again |

## Next Phase Readiness

- **The one composer is unchanged in shape.** `Blocks(posts []domain.Post, g Grammar, w int, selected
  domain.PostID, now time.Time) []Block` and `StateBlock(g Grammar, w int, head, body string) Block` are
  what plans 03-02/03-03 call; a cover is `CoverStyle.Grammar()` handed to them, and the plain reader is
  `Plain` or the `CoverStyle("").Grammar()`.
- **What the shell needs from this plan**: `CoverStyles()` (the frozen cycle order), `CoverStyleOf(id)`
  for the config, `style.Title()` for the terminal title and tmux window name, `CopyTitlePlain` /
  `CopyTitlePanic` for the other two states, `PanicFrame(w, h)` for the panic state, and `StatusFields.Cover`
  to re-word the line.
- **The alias is the only identity a cover may print**, and it is computed per account per frame from
  `u.ID` — nothing about it is stored, and `internal/render` cannot write anything, which is the structural
  half of "no alias table is ever persisted".
- **`HostSubstituted` is the one edit a cover makes to a sentence it did not author.** Plan 03-02 must run
  the classifier's pane sentence and any reachability detail through it before they reach `StateBlock`;
  this plan's tests pin the function, not the call sites.
- **No new dependency, no change to `go.mod`/`go.sum`**, and the stored corpus is byte-identical to Phase 2's.
