---
phase: 05-search-and-profiles
plan: 01
type: summary
status: complete
requirements: [DISC-01]
verified: |
  gofmt -l internal cmd                          empty
  go test ./... -count=1                         845 passed in 19 packages
  grep -rn 'irezafrnzav4Ulf1vDkEBg' internal/ --include='*.go'   empty
  grep -c 'domain.Cursor\|make(map' internal/screen/search.go   0
  grep -rn 'type Walk\[' internal/screen/*.go | wc -l           1
  grep -rn 'type .*Cursor' internal/domain/*.go | wc -l         1
  grep -rn 'homeCount' internal/ --include='*.go'               empty
  go test ./internal/tui -count=1 -run 'TestSearch'             9 PASS
  go test ./internal/xsource -count=1 -run TestCaptureSearch -v SKIP (gate off)
unverified: |
  Live SearchTimeline capture (X_TERMINAL_CAPTURE=1) — assignment forbids live
  session use here; fixture-search.json / pages-search.json / provenance row not
  written. Decoder + adapter covered by minimalSearchBody fixtures instead.
  Plan's parity -update-search path and "fixture in place" acceptance wait on that
  capture. TimelineTimelineCursor sites remain 3 (tweets.go once + detail.go's
  own conversation walker twice) — search shares tweets.go's one reader.
---
# 05-01 SUMMARY — keyword search: prompt, results walk, Latest/Top toggle

## 1. Domain and the sealed Request case

`internal/domain/profile.go` — `SearchProduct` (`Latest` zero, `Top`) with
`String()` → `latest`/`top`. No `Media`. `internal/domain/source.go` —
`Source.Search(ctx, query, product, cursor) (PostPage, error)`.

`internal/screen/walk.go` — append-only `SearchRequest{Query, Product, Cursor}`
with `At(c)`; `Request` gained `At` on the union; `ResultsFetched` /
`ResultsRefreshed` generic events; `PostTargets` / `PostFor` exported for the
shell. One walk, one cursor type — no second latch/dedupe/stall.

## 2. Results frame and prompt

`internal/screen/search.go` — `Results[T,P]` holds only `{query, product, walk,
req}`; `NewResults` / `NewPostResults` return the one opening page effect.
`Update` translates `ResultsRequested` / `ResultsRefreshRequested` /
`ResultsFetched` / `ResultsRefreshed` / `ResultsFailed` into the walk. Grep shows
zero `domain.Cursor` / `make(map` of its own.

`internal/screen/prompt.go` — product + text; `Insert` (newlines → spaces),
`Backspace` via `text.DropLastCluster`, `DeleteWord`, `Clear`, `Submitted`
(empty/all-space = no-op). `internal/text/drop.go` — `DropLastCluster`, `Tail`.

## 3. Live read path (adapter + shared walk)

`internal/xsource/tweets.go` — `Page.walkTimeline` extracted; `ParseTimeline` and
`ParseSearch` both feed it. `homeCount` renamed `pageCount` (shared).

`internal/xsource/search.go` — `opSearch = "SearchTimeline"` (name only);
`ParseSearch` at
`data.search_by_raw_query.search_timeline.timeline.instructions`;
`Adapter.Search` sends `{rawQuery, count: pageCount, product: Latest|Top,
cursor}` and nothing invented. Wire spellings stay below the port.

**Capture:** `TestCaptureSearch` skips without `X_TERMINAL_CAPTURE=1`. This run
did not capture — no `fixture-search.json`, no provenance append, no
`pages-search.json`. Decoder tests use `minimalSearchBody`.

`internal/fake/search.go` — `Search` on the fake with call counter,
`LastSearch`, `EmptySearch`, `FailNext`/`StallWith`; `RecordedSearch` fails
closed until the capture document exists.

## 4. Render, keys, shell

`internal/render/prompt.go` — `PromptRow` (grammar `Markers.PromptSearch`, tail,
caret in `1..W`). `ListHome` / `ListLabelSearch` (32-cell echo + `~`). Notices
`NoticeNoResults` / `NoticeNothingToSwitch`. Covered literals in `cover.go` for
all four styles. `Markers.PromptSearch` / `LoadingResults` on every grammar.

`internal/keys` — `/` → `search-posts`, `s` → `search-order`, both `all`, with
`HelpCovered`.

`internal/tui` — `frameSearch` on the existing stack; `prompt *screen.Prompt`
(modal, not a frame); `/` opens (re-seeds from a search frame); printable keys
before keymap lookup while open; `enter` submits one `SearchTimeline`; `esc`
restores pane byte-identically at 0 requests; `s` push/pop Latest↔Top (one
request / zero); elsewhere `nothing to switch`. Empty success → empty pane +
`no results` in notice; failure never says those words. Cursor placed only for
the prompt (`tea.NewCursor`).

## 5. Traced request counts

| Test | Asserted |
|------|----------|
| `TestSearchOpensAQueryAndRendersResults` | `/` + type = 0; `enter` = +1; status `search: latest "page cache"` |
| `TestSearchPromptSpendsNothing` | printable cmds as text, 0 requests |
| `TestSearchEscapeRestoresThePane` | `esc` = 0, frame byte-identical |
| `TestSearchEmptyQueryStaysOpen` | empty `enter` = 0, prompt stays |
| `TestSearchOrderToggleIsAFrameToggle` | `s` latest→top = +1; `s` top→pop = 0, pane identical |
| `TestSearchOrderSaysNothingToSwitch` | home `s` = 0 + notice |
| `TestSearchEmptyIsNoResultsInTheNotice` | empty pane, notice only |
| `TestSearchFailureNeverSaysNoResults` | state words, no empty copy |
| `TestSearchSlashReseedsThePrompt` | seed = frame query, 0 requests |

## Decisions the plan missed / reported

1. **No live search fixture this run** — reported, not invented. Operator:
   `X_TERMINAL_CAPTURE=1 go test ./internal/xsource -run TestCaptureSearch`, then
   extend the harness to write `fixture-search.json` + provenance, then
   `-update-search` for `pages-search.json`.
2. **CONTEXT vs UI-SPEC on `t` vs `s`** — followed UI-SPEC / plan: `/` + `s`
   push/pop, not CONTEXT's in-place `t` cycle.
3. **`TimelineTimelineCursor` count** — plan wanted 1 across `*.go`; detail.go
   keeps its conversation walk. Search did not add a fourth.
