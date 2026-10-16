---
phase: 04-post-detail-threads-and-quotes
plan: 02
status: complete
requirements: [THRD-02]
created: "2026-10-14"
---

# 04-02 — The walk, once; the selection a Target; the thread frame's state

## What landed

| File | What it holds |
|------|---------------|
| `internal/screen/walk.go` (new) | `Target` (`ID string`, `Path []string`, `Top`, `Equal`, `String`), `Walk[T, P]` with the rows, the page cache, the cursor, the dedupe set, the spent cursors, the in-flight cursor, the single-flight latch, the sticky failure, the refresh count, the stall flag, the head ids, the selected `Target` and the anchor; the sealed `Request` union (`FeedRequest`, `ThreadRequest`), `RefreshMode`, `postTargets`, `postFor`. This is the one file the later phases extend in place. |
| `internal/screen/absorb.go` | the home frame's vocabulary: the sealed `Event` set (`Select` now carries a `Target`) and six folds, each one a call into the walk. No rule of its own. |
| `internal/screen/list.go` | `List` = `Kind domain.HomeKind` + one `Walk`; the accessors it always had, plus `Selected() Target`, `Targets()`, `Post(Target)`, `Move(int)`, `Anchor()`, `WithAnchor(int)`, and `Index(string)`. `FirstID`/`LastID`/`NextID`/`PrevID`/`Has` are gone: movement walks the frame's own row sequence, so the id lookups had no caller left. |
| `internal/screen/effect.go` | `Effect.Req Request` replaces `Feed domain.HomeKind`; `Head` is `[]string` (the strings the walk keys rows by — Phase 5 asks the same question about accounts). |
| `internal/screen/thread.go` (new) | `Thread` (the focal copy, the chain above, the continuations, one `Walk[domain.Post, domain.ThreadPage]`, the read flag), `NewThread`, the four thread events, the two-stage load, the trigger, the refresh, the three partial-thread questions, `Position`. |
| `internal/render/block.go` | `Selection{ID domain.PostID; Path []domain.PostID}`; `Blocks(…, sel Selection, …)`; the per-record selection cell (`selectionCell`), with the block's own row path threaded through `post`/`quote`. A block-level selection marks the whole block including its quote rows (Phase 2 unchanged); a selection whose `Path` names a quoted post marks only that post's rows. |
| `internal/tui/model.go`, `update.go`, `view.go` | `frame{id, List}` (the anchor lives in the walk now), `selectionOf`, `heads`/`rowsAbove`/`scroll`/`reanchor` over Targets, movement through `Targets()` + `Move` with `step` measured in rendered rows, and `request()` — the dispatch over the request union. |

Tests added: `internal/screen/walk_test.go` (15), `internal/screen/thread_test.go` (12), plus pins in `list_test.go`
(a quoted post is a row; the frame's row order) and `block_test.go`/`golden_test.go` re-run unchanged.

## Verification (exact commands)

- Task 1: `go test ./internal/screen/ -count=1 -run 'TestWalk|TestPostTargets|TestTarget' && go vet ./internal/screen/` → **15 passed**.
- Task 2: `go test ./internal/screen/ ./internal/render/ -count=1 && go vet ./internal/screen/ ./internal/render/` → **201 passed** (before Task 3 the tree still fails to build in `internal/tui`, as the plan says it will).
- Task 3: `go build ./... && go test ./internal/screen/ ./internal/tui/ -count=1` → **124 passed**.
- Whole plan: `go build ./... && go test ./... -count=1` → **730 passed, 19 packages** (baseline was 703); `gofmt -l internal cmd` empty.
- Acceptance greps: `type Target struct`/`type Walk[`/`type Request interface`/`func NewWalk[` = 1 each; `usedCursors` in `walk.go` only (0 in every other non-test file); `Req Request` in `effect.go` = 1 and `Feed domain.HomeKind` = 0; `func (l List) Selected() Target`, `func (l List) Targets()`, `func (l List) Post(t Target)`, `Thread.Post`, `type Selection struct` all 1; `func NewThread` 1; `ThreadRequest` in `walk.go` 3; `func (t Thread) Position()` 1; `PartialStopped|PartialAbove|PartialHidden` 6.
- Purity: no `net/http`, no `os.`, no `time.Now`, no `io.` in `internal/screen` (non-test); the package imports only `domain`, `slices`, `strings`, `time`. `internal/render` (non-test) still imports only `domain`, `text` and the standard library.
- Phase 2's stored frames and the back stack's four observables: unchanged, green (`internal/render`'s golden corpus and `internal/tui`'s round-trip tests were not edited except for the selection's type).

## Tests that were watched failing (broken deliberately, then reverted)

1. `walk.go`'s stall rule, `added == 0` removed → `TestWalkStallsOnAPageThatAddsNothingUnseen` failed: "a page that added no unseen row did not end the walk…".
2. `list.go` given one occurrence of `usedCursors` → `TestWalkPolicyLivesInOneFile` failed: "list.go names the walk's "usedCursors"…" (the one-implementation rule is a test, not an intention).
3. `Thread.requested`'s trigger gate removed → `TestThreadPaginationIsTheReplyWalk` failed: "a press with the selection on 1 produced {1 {7 c1} c1 []}…" (an ancestor row is not a trigger).
4. `Thread.PartialHidden`'s count clause replaced by `false` → `TestThreadPartialHidden` and `TestThreadReadsTheReplyCountWithoutPrintingIt` both failed.

## Decisions the plan did not settle (and the choices made)

- **`Target` is not comparable** (it holds a slice), so it carries `Equal`. The shell and the walk compare targets through it; nothing uses `==`.
- **`absorb`/`refresh` signatures**: the plan's `absorb(rows []T, next domain.Cursor, requested int)` and `refresh(rows []T, requested int, mode RefreshMode)` name a count the walk's rules never read (Phase 2's `pageFetched` stalls on the row the walk did not hold, the cursor it was asked with, and the spent set) and omit the two values the folds do need. They are now `absorb(page P, rows []T, next domain.Cursor)` — the page value is what fills the walk's own page cache, which the must-haves require — and `refresh(rows []T, next domain.Cursor, mode RefreshMode)`, because `RefreshReplace` moves the walk's cursor to the answer's (`04-UI-SPEC`). Phase 2's stall rule is verbatim.
- **The walk's page gate asks for page one**: `next == ""` refuses a page only once a page has been absorbed, so a frame the shell has just pushed can ask for its own first page through the same gate as every later one. Nothing in the shell changes (its own trigger requires a loaded list); the thread's first page depends on it.
- **The anchor is declared once, in the walk** (must-have: "declared once in `internal/screen/walk.go`"). Task 3's "`frame` keeps `Anchor int` for now" is superseded: the shell reads and writes `List.Anchor()`/`List.WithAnchor`, and 04-03's frame will do the same through its own list/thread.
- **`Thread.LoadingOlder()` was added** to the plan's "exactly these names": 04-03's status line prints `loading replies` while a reply page is out, `internal/screen/thread.go` is not in 04-03's file list, and the predicate is the walk's own.
- **The thread's trigger gate lives in the frame** ("reaching a row above the reply walk asks for nothing"): the frame is where the walk's cursor is, so a caller that asks at the wrong moment gets nothing rather than a request. The shell may therefore send `ThreadRequested` on a forward move without duplicating the rule.
- **`RefreshReplace` clears the stall flag**: the walk's position becomes the answer's, so the evidence that ended it (a cursor) is gone with it. A later page that repeats a cursor stalls again on its own. `RefreshPrepend` is Phase 2's rule verbatim and touches neither the stall flag nor the cursor.
- **`List.Move`/`Anchor`/`WithAnchor`** join the plan's widened accessor list (the shell needs them once the anchor is the walk's), and `FirstID`/`LastID`/`NextID`/`PrevID`/`Has` were deleted rather than kept as a second way to move.
- **The plan's walk_test note is wrong on one point**: "a page whose cursor equals the one it was asked with does not stall by itself unless it also added nothing" contradicts Phase 2's `pageFetched` and the must-haves ("a cursor that comes back a second time **or** a page that adds nothing unseen ends the walk"). Phase 2's rule is implemented and tested: the repeated cursor ends the walk whether or not the page carried rows the walk did not hold.
- **The shell's thread arm is not in this plan**: `request()` has an explicit `ThreadRequest` case that returns no command, and the comment names 04-03 as the plan that brings the read, the frame it lands in and the notices. Nothing in this build produces a `ThreadRequest` (no port for it is in `Deps` yet).

## What 04-03 inherits (unverified here)

- The frame renders from the copy in hand and page one replaces it; the request count for an open that only renders is asserted in `TestThreadOpeningCostsNoRequest` at the frame level (the source's call counter). The same claim at the shell level — `fake.Thread.Calls()` unchanged across a push — is 04-03's test.
- `Thread.Targets()` is the order the shell moves through and `Thread.Post(quote target)` is the copy a quoted conversation is opened from; `Thread.Position()` is the status line's `{selected}/{loaded}`, and it is this plan's reading of "the chain above, the focal post, its continuations and the replies read so far".
- No engagement count is reachable through any accessor a renderer calls (`TestThreadReadsTheReplyCountWithoutPrintingIt`): the reply count decides one bool, and the frame's vocabulary names no count. Post values still carry `Counts` as content; the grammars print no field of them (04-03's corpus audit).
