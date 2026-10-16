---
phase: 05-search-and-profiles
plan: 03
type: summary
status: complete
requirements: [DISC-02]
verified: |
  gofmt -l internal cmd                          empty
  go test ./... -count=1                         916 passed in 19 packages
  go test ./internal/tui -count=1 -run 'People|Account|Agnostic|Golden|Corpus' PASS
  go test ./internal/screen -count=1 -run Agnostic PASS
  go test ./internal/xsource -count=1 -run People PASS
  go test ./internal/fake -count=1 -run People PASS
  grep -c 'type Results' internal/screen/search.go   → one Results[T,P] type
  grep -rn 'type Walk\[' internal/screen/*.go | wc -l → 1
  sha256 corpus stable across non-update Golden|Corpus run
unverified: |
  Live SearchTimeline People capture (X_TERMINAL_CAPTURE=1) not run; fixture is
  synthetic (fail-closed). Provenance queryId names the registry id for
  documentation only; no queryId literal appears in non-test .go (XAPI-02).
  TimelineTimelineCursor case arms remain in detail.go (thread walker) and the
  shared foldEntry in tweets.go — people adds no second cursor reader.
  Human frame approval (Task 3 checkpoint) still pending: run
  `go test ./internal/tui -run TestShowNewFrames -v -count=1`.
---
# 05-03 SUMMARY — people search: same walk, same reducer, one row per account

## 1. Port and value

`SearchProduct.People` (`String` → `people`). `UserPage{Rows, Next, Requested,
FetchedAt}` mirrors `PostPage`. `Source.SearchPeople(ctx, query, cursor)`.

## 2. Shared instruction walk + ParseSearchPeople

`walkInstructions` / `foldEntry` / `entrySink` in `tweets.go`: one entry loop,
one cursor typename arm. `Page` and `peoplePage` are sinks. `ParseSearchPeople`
reads `TimelineUser` via `user_results` → `userFrom` (same as post authors).
Non-account items skipped; unreadable account → typed path error; neither rows
nor cursors → typed error. `wireProduct(People)="People"`.
`Adapter.SearchPeople` → `SearchTimeline` by name + measured four variables.

Fixture: `internal/xsource/testdata/fixture-search-people.json` (synthetic,
fail-closed; gated capture stubs refuse a live session). Provenance row:
`operation=SearchTimeline`, `source=synthetic`, variables
`{rawQuery:people,count:20,product:People,cursor:""}`.

## 3. Fake

`SearchPeople` mirrors Search knobs (counter, failNext, emptySearch, stall).
`AdversarialPeople()`: cut display name, empty name, long handle, ≥2 pages.
`pages-search-people.json` from `ParseSearchPeople` via `-update-search-people`.
Parity + `TestAdversarialPeopleCarriesRequiredCases`.

## 4. Keys + shell

`u` → `search-people` / "find people" (plain HelpCovered empty — no "account"
leak). People frame = `frameSearch` + `People Results[User,UserPage]` beside
`Search`. Prompt seeded from either product. Pane: `UserRows` vs `Blocks`.
`enter`/`a` on account → profile (2 requests). `s` on people →
`nothing to switch`, 0 requests. Empty success → `no results` notice.

## 5. Agnostic proofs

`screen/agnostic_test.go`: one script table (first page, dedupe, stall, fail,
refresh new/none, select, end) over `Walk[Post]` and `Walk[User]` — same
expected outcomes. Dedupe arm proven non-vacuous (raw concat ≠ walk absorb).
`tui/agnostic_test.go`: same key script (`/|u`, `j`, `G`/`pgdown`, `q`) over
both kinds — 1 open request, move parity, 0 pop requests, position ≥ 1.

## 6. Account row

`UserRow` / `UserRows` via `UserHead` (`{handle:16} {name}`); name truncated with
mark; empty name prints nothing. No bio/counts/badge/avatar/stamp.

## 7. Golden corpus + audit

36 frames under `internal/tui/testdata/TestGoldenFrames/` driven through
`tui.Model` Update/Render with pinned env+clock (`2026-10-14T09:12:00Z`).
Names: search_{60,80,120,200}, search_top_80, search_people_{80,120},
search_empty_80, profile_{60,80,120,200}, profile_{empty,bare,loading,scrolled}_80,
prompt_{80,120}, search_stalled/failure_80, colour rung matrix, one per cover
grammar for people+profile. `TestCorpusAudit` over stored bytes: CJK+emoji
query, long query, cut name, bio, bare header, selection, `+`, absences
(no likes/rank/result counts in lists), A1-S/A2-S/A4, corpus size arm.
Planted `likes` violation detected then restored.

## 8. Grammar completeness

`TestPhase5MarkersFillEveryGrammar` + covered people frame scan for plain
loading/empty literals.

## Human verify

`go test ./internal/tui -run TestShowNewFrames -v -count=1` prints search_80,
search_people_80, profile_80, prompt_80. Resume-signal: "approved".

**Deferred (2026-10-15):** operator visual approval of TestShowNewFrames is
deferred — not a progress blocker. Offline goldens plus the corpus absence
arms (bio-in-people-row, match-term attribute, planted-violation) hold the
spec; resume when an operator can run ShowNewFrames and record approval.
