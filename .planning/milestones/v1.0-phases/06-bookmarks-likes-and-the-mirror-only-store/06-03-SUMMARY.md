---
phase: 06-bookmarks-likes-and-the-mirror-only-store
plan: 03
type: summary
status: complete
requirements: [SAVE-04]
verified: |
  gofmt -l internal cmd                         empty
  go build ./...                                ok
  go vet ./internal/store/ ./internal/config/ ./internal/cli/ ./internal/xsource/  ok
  go test ./... -count=1                        974 tests / 20 packages ok
  unshare -rn (lo up) go test ./internal/store/... ./internal/cli/... -run TestSync|TestWalk|TestFull|TestOwner  ok
  go list -deps ./internal/store | grep -c xsource  0
  grep -c 'DELETE' internal/store/*.go          0
  TestFullNeverDeletes fail-arm (DELETE planted) FAIL then revert PASS
  go run ./cmd/x-terminal --help | grep sync    lists --full / --kind
  live: go run ./cmd/x-terminal sync --kind bookmarks  exit 0
    "bookmarks  pages 40  items 800  absent 0  page cap"
    mirror /home/user/.local/share/x-terminal/saved.db
unverified: |
  Pane `r` driving the same Begin/Step/Abandon — plan 04.
  doctor mirror line — Phase 8.
decision: |
  SortIndex is a walk-wide descending decimal key from page order: domain.Post
  does not carry upstream sortIndex, and the mirror reader orders by sort_index DESC.
  MarkAbsent only on completed full stops (EndOfList, PageCap); StopStalled and
  Abandon mark nothing.
  likesReady stamps WithViewerID from Begin's Viewer read so Likes has userId
  without a second identity call outside the walk.
  Sync creates DataDir (MkdirAll 0700) before Open, matching notice/pins writers.
---
# 06-03 SUMMARY — incremental/full walk and `sync` verb (SAVE-04)

## Walk (`Begin` / `Step` / `Abandon`)

`internal/store/sync.go:316,330,341,439,518` — `SavedFetcher`, `Viewer`, `pageCap=40`
(a cap, not a target), `Mirror`/`NewMirror`, `run.Step`/`Abandon`. Incremental stop
at the first page with `added == 0` (`StopNoUnknownIDs`); full re-list marks
`present = 0` at the run's start instant and never deletes. Failed/abandoned runs
write their own `sync_runs` row and leave last-success untouched. Store never
imports `xsource`.

## Owner and Viewer

`ErrWrongOwner` (`sync.go:310`) refuses a foreign session after one Viewer read
and before any page fetch. `(*Adapter).Viewer` (`user.go:167`) is the one
identity read; whoami keeps its direct call.

## Verb

`config.SavedDBPath()` (`config.go:196`) — `saved.db` beside pins/notice.
`internal/cli/sync.go:61` — `--full` / `--kind` / `--json` / `--quiet`;
`openMirror` seam; one `Begin`/`Step` loop per kind; presence facts only.
`cli.go` gains one `sync` row.

## Proofs that can fail

- Never-delete: three seeded rows, full walk sees two → file still holds three,
  `Page` returns two, `Status.Absent=1`, re-see restores present; planted `DELETE`
  made `TestFullNeverDeletes` fail, then reverted.
- Owner refusal: fetcher call count unchanged; both handles named.
- Verb: incremental call count 2 then 1; `--full` absent facts; failing fetch exit
  1 with no post text on stdout; `--quiet`/`--json`.

## Live

`x-terminal sync --kind bookmarks` → exit 0, 40 pages / 800 items / page cap,
mirror at `~/.local/share/x-terminal/saved.db` (800 present rows).
