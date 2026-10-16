---
phase: 04-post-detail-threads-and-quotes
verified: 2026-10-15T07:12:26Z
status: passed
score: 22/22 must-haves verified
covered_files:
  - .planning/phases/04-post-detail-threads-and-quotes/04-01-PLAN.md
  - .planning/phases/04-post-detail-threads-and-quotes/04-01-SUMMARY.md
  - .planning/phases/04-post-detail-threads-and-quotes/04-02-PLAN.md
  - .planning/phases/04-post-detail-threads-and-quotes/04-02-SUMMARY.md
  - .planning/phases/04-post-detail-threads-and-quotes/04-03-PLAN.md
  - .planning/phases/04-post-detail-threads-and-quotes/04-03-SUMMARY.md
  - internal/cli/feed.go
  - internal/cli/feed_test.go
  - internal/domain/thread.go
  - internal/domain/thread_test.go
  - internal/fake/testdata/thread-shapes.json
  - internal/fake/testdata/thread-tombstone.json
  - internal/fake/testdata/thread.json
  - internal/fake/thread.go
  - internal/fake/thread_parity_test.go
  - internal/keys/keys.go
  - internal/keys/keys_test.go
  - internal/render/block.go
  - internal/render/block_test.go
  - internal/render/copy.go
  - internal/render/corpus_test.go
  - internal/render/cover.go
  - internal/render/golden_thread_test.go
  - internal/render/grammar.go
  - internal/render/status.go
  - internal/render/testdata/TestGoldenFrames/cover_ci_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_120.golden
  - internal/render/testdata/TestGoldenFrames/thread_200.golden
  - internal/render/testdata/TestGoldenFrames/thread_60.golden
  - internal/render/testdata/TestGoldenFrames/thread_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_adversarial_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_chaincut_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_color_16.golden
  - internal/render/testdata/TestGoldenFrames/thread_color_256.golden
  - internal/render/testdata/TestGoldenFrames/thread_color_mono.golden
  - internal/render/testdata/TestGoldenFrames/thread_color_truecolor.golden
  - internal/render/testdata/TestGoldenFrames/thread_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_narrow_79.golden
  - internal/render/testdata/TestGoldenFrames/thread_partial_120.golden
  - internal/render/testdata/TestGoldenFrames/thread_partial_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_push_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_sel_ancestor_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_sel_quote_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_sel_reply_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_status_failure_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_status_noplus_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_status_plus_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_tombstone_focal_80.golden
  - internal/render/testdata/TestGoldenFrames/thread_visibility_80.golden
  - internal/render/thread.go
  - internal/render/thread_test.go
  - internal/screen/absorb.go
  - internal/screen/effect.go
  - internal/screen/list.go
  - internal/screen/list_test.go
  - internal/screen/thread.go
  - internal/screen/thread_test.go
  - internal/screen/walk.go
  - internal/screen/walk_test.go
  - internal/tui/cover_e2e_test.go
  - internal/tui/model.go
  - internal/tui/tui_test.go
  - internal/tui/update.go
  - internal/tui/view.go
  - internal/xsource/capture_test.go
  - internal/xsource/detail.go
  - internal/xsource/detail_test.go
  - internal/xsource/fixture_test.go
  - internal/xsource/testdata/fixture-tweetdetail-empty.json
  - internal/xsource/testdata/fixture-tweetdetail-shapes.json
  - internal/xsource/testdata/fixture-tweetdetail-tombstone.json
  - internal/xsource/testdata/fixture-tweetdetail.json
  - internal/xsource/testdata/provenance.json
  - internal/xsource/tweets.go
  - internal/tui/dispatch_test.go
covered_digest: "v3:sha256:6530733f059668662e11ddaac04b5ef28d8c95c2975af315bae5a548e4cd652c"
behavior_unverified: 0
coincidental_reliance_items: []
---
# Phase 4: Post Detail, Threads and Quotes Verification Report

**Phase Goal:** The user opens a post and reads its conversation — replies a page at a time, quotes inline, tombstones and unavailable replies labelled honestly.
**Verified:** 2026-10-15T07:12:26Z
**Status:** passed

This is the sixth pass over the **frozen** tree. Since the previous pass (06:22:16Z) exactly one change
landed, and it is a guard over this phase's own dispatcher: **`internal/tui/update.go` restructured plus
one new file, `internal/tui/dispatch_test.go`.** The change set was read off the filesystem, never
trusted: `diff -rq` between `/tmp/x4v5_pristine` — the byte copy the previous pass's own experiments ran
from — and the repository reports **exactly two source paths** (`internal/tui/update.go` modified,
`internal/tui/dispatch_test.go` present only in the repository) plus the eight phase `VERIFICATION.md`
files the parallel passes write. **No thread-frame source file and no golden is in that set:** all **31**
thread goldens (23 `thread_*` + 8 `cover_*_thread_*`) are byte-identical to that copy and to the
pre-fix copy (`cmp`, 31 compared, 0 differing against each), and the four stored corpora
(`internal/render/testdata`, `internal/tui/testdata`, `internal/fake/testdata`,
`internal/xsource/testdata`) are byte-identical too (`diff -rq`, no output). `internal/keys/keys.go` is
byte-identical (`cmp`), so **no key binding moved**; `internal/render/{thread,block,grammar,cover,status}.go`,
`internal/screen/*` and `internal/xsource/*` are untouched.

## Is the dispatcher refactor behaviour-preserving? Yes — three ways

`diff -u` of `internal/tui/update.go` against the previous pass's copy is **three hunks and nothing
else**:

1. the `slices` import for the new `HandledActions` copy;
2. **three multi-action `case` arms split into one arm per action** — `case ActionBookmarks, ActionLikes:`
   became two arms, `case ActionCover1, …, ActionCover4:` became four, and the eight movement actions
   became eight — each new arm carrying **the identical body the shared arm had** (`m.savedAction(action)`,
   `m.numberedCover(action)`, `m.move(action)`);
3. the new comment block, the `handledActions` slice and `func HandledActions()`.

The panic branch, the prompt branch (`m.promptKey(msg)`), the help branch, and the `coverBreaking` gate
are **outside every hunk** — their line ranges are unchanged modulo the additions below them. In Go a
`switch` over a typed string with `case A, B: return f(action)` and `case A: return f(action); case B:
return f(action)` are the same program for every input, and no case was reordered or rebound:

- **the keymap⇄model exhaustion arm**: `keys.Rows()` (`internal/keys/keys.go`) is byte-identical, and
  `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` passes, so every row's action still has
  an arm and every arm still has a row; the join is now total in both directions (30 rows ⇄ 30 actions);
- **the gated actions under a cover**: `coverBreaking` and its call site are unchanged, and
  `TestCoverBreakingActionsGated`, `TestPanicIgnoresCoverBreakingKeys` and `TestMediaFallbackHiddenUnderCoverRestores`
  pass — `m` no longer reaches any action under a cover than it did;
- **the prompt's keys**: the prompt branch precedes the lookup exactly as before, and
  `TestSearchPromptSpendsNothing`, `TestSearchSlashReseedsThePrompt`, `TestSearchEmptyQueryStaysOpen`,
  `TestSearchOpensAQueryAndRendersResults` pass;
- **the panic gate's keys**: the panic branch is byte-identical, and `TestPanicRoundTripIsExact`,
  `TestPanicEntersInOneFrameAndLeavesExactly`, `TestPanicIgnoresCoverBreakingKeys`, `TestPanicSwallowsKeys`
  and `TestBlurPanicsAndFocusDoesNotRestore` pass;
- **the frame pushes/pops**: `enter → ActionOpen → m.open()` (`update.go:855`), `q`/`esc → ActionBack →
  m.back()` (`update.go:984`), `g`/`home → ActionFirst → m.move(action)` (`update.go:373`) — the same
  handler each — and `TestEnterOpensAConversationWithLoadingThenPage`,
  `TestEnterOnAQuotedPostPushesTheQuote`, `TestEnterOnATombstoneOpensItsConversation`,
  `TestEnterWithNothingToOpenPrintsTheNotice`, `TestQRestoresTheFrameWithZeroThreadRequests` and
  `TestTheBackStackRestoresTheFrame` all pass; `TestTIsANoopInAThreadAndOverlayListsEnter` still holds;
- the kind join (`TestEveryFrameKindIsCheckedUnderACover`, 5/5 kinds) and the geometry guard
  (`TestTheSelectedHeadStaysInsideThePaneUnderEveryCover`, **20/20 subtests**) pass unchanged.

24 named dispatcher-flow tests were run in one batch on the repository and all 24 passed (the list
above). The one thing the refactor *added* is the enumeration itself, and the reference-build failure it
leans on is real: a bogus case label fails the build with
`undefined: keys.ActionThisPackageDeclaresNoSuchAction` (plant G3 below).

## The guard round's own claims, adjudicated

| The guard's claim | Adjudication |
|---|---|
| **`press` now has exactly one arm per action, so the handled set is enumerable** | **True.** The three shared arms are split (`update.go:286-288` bookmarks/likes, `301-308` the four covers, `311-326` the eight movements); every arm's body is the one call the shared arm made, with the same `action` value |
| **`handledActions` + `HandledActions()` name the dispatcher's set** (`update.go:1297`, `1315`) | **True and live.** The slice names 30 actions; `HandledActions()` returns `slices.Clone(handledActions)`, and the test proves the copy is a copy (`exported[0] = "scribbled"` does not move `handledActions`) |
| **`dispatch_test.go` joins the keymap to the dispatcher in both directions** | **True.** `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` walks `keys.Rows()` → the set and the set → `keys.Rows()`, plus two anti-vacuity floors (row count equals the set's; ≥30 actions); `TestTheShellDeclaresOnlyActionsTheKeymapDeclares` sweeps the *sources* — every `Action…` constant `keys.go` declares must be a row, a set member and a case label, and every action name the shell's sources reference must be declared (read through Go's parser, so the planted obituary prose at `update.go:333` is not a reference) |
| **A 31st row with a fresh action fails it** | **True.** Plant G1 (an `ActionProbe` const plus a 31st row on `f13`): `the keymap binds "probe" (f13) and press has no arm for it`, `the table holds 31 rows and handledActions names 30 actions`, `keys.go declares ActionProbe and handledActions does not name it`, `keys.ActionProbe is no case label in the shell` |
| **A handler with no row fails it** | **True.** Plant G2 (an `ActionGhost` const, an enumeration entry and a `case` arm, no row): `press handles "ghost" and no row of the keymap binds it: the arm is unreachable`, `the table holds 30 rows and handledActions names 31 actions`, `keys.go declares ActionGhost ("ghost") and no row binds it` |
| **The compiler refuses an arm for an action that does not exist** | **True.** Plant G3: `go build ./...` fails with `internal/tui/update.go:311:12: undefined: keys.ActionThisPackageDeclaresNoSuchAction` |

## The fix round (13:56–14:09) — unchanged since the previous pass, and re-checked

The previous pass adjudicated the 20-file fix round in full; its files are byte-identical to this pass's
reference copy, so that adjudication stands, and this pass re-ran the two arms of it that touch this
phase's covered set:

- `savedMirrorError` reverted to `return err != nil` → **4 `internal/cli` tests fail**
  (`TestSavedMirrorErrorNamesOnlyTheFile`, `TestAShapeDriftedAnswerIsNotAFactAboutTheFile`,
  `TestASavedFrameNamesATransportFailureAsTheOrdinaryState`, `TestASavedFrameNamesTheRequestPathsOwnStates`);
  the conversation frame is unaffected either way.
- `volatile.PinsFileName` renamed to `"pinsX.json"` → `TestPinsPathIsTheRegistersName` **fails**
  (`the pins file's name is "pinsX.json", want pins.json … a rename is a migration`);
  same result as recorded.

Of the fix's 20 files, **five are in this phase's covered set** (`internal/cli/feed.go`,
`internal/cli/feed_test.go`, `internal/keys/keys.go`, `internal/tui/cover_e2e_test.go`,
`internal/tui/tui_test.go`), and none of them changed this round.

## The amendment (the post-04:18 round) — unchanged, re-checked

Its files are byte-identical to this pass's reference copy, so the previous adjudication stands. The two
arms that bear on this phase were re-run and reproduce: the widened leak scan's plant
(`TestCoveredFramesLeakNoRepairWords`, **24 findings naming all eight** covered thread frames) and the
kind join's plant (a sixth kind fails `TestEveryFrameKindIsCheckedUnderACover` with both messages). Two
citations moved with the dispatcher restructure: `TestEveryFrameKindIsCheckedUnderACover` is still at
`cover_e2e_test.go:730` (`frameKinds()` call at `732`) and `profileAccount` at `view.go:447`; both files
are untouched this round.

## What was run

| Check | Command | Result |
|---|---|---|
| Build | `go build ./...` | clean |
| Vet | `go vet ./...` | clean |
| Formatting | `gofmt -l internal cmd` | empty |
| Suite | `go test ./... -count=1 -v` | **1573 passed** (1581 `=== RUN`), **0 failed**, 8 skipped, 19 packages `ok` + 3 without tests = **22 packages** (the previous pass's 1571 + the two new dispatch tests) |
| Offline | `unshare -rn sh -c 'ip link set lo up; go test ./... -count=1 -v'` | **1573 passed**, 0 failed, 8 skipped, 19 packages `ok` — no test needs the network |
| Phase suites offline | `unshare -rn … go test ./internal/{xsource,domain,screen,render,fake,keys,tui,cli} -count=1` | all `ok` |
| Named Phase-4 tests | the phase's named set offline (`-run 'Thread\|Quote\|Tombstone\|Visibility\|Partial\|Focal\|Walk\|Enter\|GoldenFrames\|CorpusAudit\|Covered\|FrameKind\|Selected\|Provenance\|Fixture\|Identity'` over the eight packages) | **604 ran, 603 passed, 0 failed, 1 skipped** (the identity arm) — the same counts as the previous pass |
| Dispatcher-flow tests | `go test ./internal/tui -run '<24 names>' -v` | **24/24 pass**: gated actions under a cover, the prompt's keys, the panic gate's keys, the frame pushes/pops, the help overlay, the kind join, the geometry guard, both dispatch joins |
| Dispatch joins | `go test ./internal/tui -run 'TestEveryBoundActionIsHandledAndEveryHandledActionIsBound\|TestTheShellDeclaresOnlyActionsTheKeymapDeclares' -v` | **2 passed** |
| Geometry guard (shipped) | `go test ./internal/tui -run TestTheSelectedHeadStaysInsideThePaneUnderEveryCover -v` | **20/20 subtests pass** (5 grammars × {80,120} × {list,thread}) |
| Geometry, all widths (scratch) | probe repeating the guard at W ∈ {60,80,120,200} × 5 grammars × 2 frames | **40/40 assertion pass, 40/40 moved; 8/8 plain controls pass**; 1 non-discriminating arm (`servicelog/60/thread`, as recorded) |
| Cold-frame opening rule (scratch) | 04-UI-SPEC "On push: where the frame opens", 5 grammars × {60,80,120,200} × pane heights {5,6,8,12,24} | **100 combinations, 0 bad** — 48 frames present their own first row, 52 put the head at pane row 0 |
| Covered thread surface (scratch) | every cover, loaded / failed / stalled, scan for host, URL and repair words | **pass** in all 15 combinations (5 grammars × 3 states) |
| `enter` on the thread's own focal (scratch) | probe: `Path`-less focal + `enter` | no frame pushed, 0 thread requests |
| Leak scans (shipped) | render `TestCoveredFramesLeakNoRepairWords\|TestCoveredLeakScanCatchesAPlant\|TestCorpusAudit`; tui `TestTheCoveredTUIFramesLeakNoPlainVocabulary\|TestTheCoveredTUIFrameLeakScanCatchesAPlant\|TestEveryFrameKindIsCheckedUnderACover` | all green over the clean tree; the render no-http arm logs "read **170** covered frames"; the kind join passes 5/5 |
| Focal + audit binary | `TestGoldenFrames/thread_tombstone_focal_80$`; `go run ./cmd/x-terminal audit` | golden green clean; **audit exits 0: 10 checks, 0 findings, 893 files inspected** (`source`, `readonly`, `opnames`, `transport`, `timers`, `persistence`, `distribution`, `account`, `boundaries`, `leak`) |
| Identity arm | `TestNoFixtureCarriesViewerIdentity` with `XDG_CONFIG_HOME`/`XDG_DATA_HOME` under `/tmp` | SKIP without a session; with `X_TERMINAL_AUTH_TOKEN`/`CT0` + `X_TERMINAL_VIEWER_HANDLE` it **runs and passes**, and a planted handle makes it **FAIL** |
| Independent census | python over the raw recorded body | **202,462 bytes** (`stat`), sha256 `526ca1f8081f…` — identical for `internal/xsource/testdata/fixture-tweetdetail.json` and `.planning/research/evidence/fixture-tweetdetail.json`; 43 entries = 41 `conversationthread-` + 1 `tweet-` + 1 `cursor-` |
| Provenance census | python over `provenance.json` | 11 rows under `files` for 11 fixtures, every `redaction` non-empty |
| Corpus census | name census over both testdata trees | render **263** frames (**154** `cover_*` + 24 `media_*`) + tui **56** = 319 goldens; **31 thread frames, of which 8 are covered** |
| Thread frames unchanged | `cmp` of all 23 `thread_*.golden` + 8 `cover_*_thread_*.golden` against `/tmp/x4v5_pristine` (the previous pass's copy) and against `/tmp/x4r_pristine` (the pre-fix copy) | **31 compared, 0 differing** against each |
| Guard round's change set | `diff -rq /tmp/x4v5_pristine .` (minus `.git`) | exactly `internal/tui/update.go` + `internal/tui/dispatch_test.go` (and the parallel passes' `VERIFICATION.md` files); the four stored corpora byte-identical |
| Reproducibility | `go test ./internal/fake -run TestThreadParity -update`; `go test ./internal/render ./internal/tui -run TestGoldenFrames -update` | no diff in the three generated fixtures; no diff in either testdata tree |
| Fingerprint | `gsd-tools query verification.fingerprint …` | **82 files**, `v3:sha256:6530733f…` (the previous pass's 81-file digest was `c9069cf5…`; only `update.go` changed and `dispatch_test.go` was added) |
| Phase completeness | `gsd-tools verify phase-completeness 4` | `{"complete": true, "plan_count": 3, "summary_count": 3, "errors": [], "warnings": []}` |

All destructive experiments ran on scratch copies (`/tmp/x4v6`, reset from `/tmp/x4v6_pristine` — a byte
copy of the current tree taken before any experiment — between plants; the probes and the plants live
only there). `diff -rq` between the repository and that copy reports **only the scratch probe file**;
the repository differs from this pass's own pre-pass byte copy in **nothing outside `.planning`**: no
repository source file, fixture or golden was edited by this pass, only this report was written. The
operator's real `~/.config/x-terminal/session` was never read, printed or deleted — every run that could
touch a session was given `XDG_CONFIG_HOME`/`XDG_DATA_HOME` under `/tmp`, and
`~/.local/share/x-terminal/saved.db` was never opened.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 04-01.1 | One thread read is one page; nothing in the read path expands a cursor into a further request of its own | ✓ VERIFIED | `TestTheAdapterAsksForOnePage` passes on the current tree: exactly 1 `GraphQL` call for page one (operation `TweetDetail`, `focalTweetId` = the id, empty `cursor`), a second call only when the caller passes the answer's own `Next`; `ParseThread`/the walker contain no loop over a cursor. `internal/xsource` is untouched by the guard round |
| 04-01.2 | The recorded 202,462-byte body decodes to 42 posts out of 43 entries, non-posts skipped and counted | ✓ VERIFIED | `stat`/`sha256sum` re-run this round: 202,462 bytes, `526ca1f8081f…`, identical for fixture and evidence; the census re-counts 43 entries = 41 `conversationthread-` + 1 `tweet-` + 1 `cursor-`; `TestTheRecordedThreadIsOnePage` pins `Requested` 42 / `Skipped` 6 |
| 04-01.3 | The entries that are not posts are named and dropped | ✓ VERIFIED | `TestNoPostFromTheRelatedRailOrAnAdvertisementReachesThePage` passes; planting `if false && strings.Contains(item.EntryID, promotedItemMark)` (`detail.go:294`) fails **6 top-level tests** this round (5 in `internal/xsource`, plus `TestThreadParity`) |
| 04-01.4 | A moved post is a typed `ParseError` naming the JSON path; a first page without the focal post is refused | ✓ VERIFIED | `TestAnUnknownTweetShapeIsATypedError`, `TestAMissingFocalFirstPageIsRefused` pass in the current tree |
| 04-01.5 | Tombstones, limited and withheld posts decode as themselves; none is an error, none dropped | ✓ VERIFIED | `TestTheTombstoneFocalIsAnUnavailableFocal`, `TestTheSyntheticShapesAreAllRead` pass; the withheld rule is still `detail.go:369` (`node.Legacy != nil && len(WithheldInCountries) > 0`), recorded as the one inference |
| 04-01.6 | Every fixture has exactly one provenance row naming operation, date, query id, redaction, sha256; the recorded one names the evidence file | ✓ VERIFIED | 11 provenance rows for 11 fixtures, all with non-empty `redaction`, the recorded `fixture-tweetdetail.json` row naming `.planning/research/evidence/fixture-tweetdetail.json` (re-read this round); deleting one `conversationthread` entry fails 5 top-level tests (re-run this round) |
| 04-01.7 | Nothing in the checked-in fixtures is the viewer's own identity or a cookie; the capture harness rewrites the id and handle; a test that knows the session says so | ✓ VERIFIED | Always-on arms pass (no `auth_token`/`ct0` keys, every `redaction` non-empty). The identity arm SKIPs when no session is reachable and **runs** when identity is supplied through `X_TERMINAL_AUTH_TOKEN`/`CT0` + `X_TERMINAL_VIEWER_HANDLE`: passing clean, failing with "testdata/fixture-tweetdetail-shapes.json carries the viewer's own handle" under a planted `"screen_name": "probehandle"` (all three re-run this round). See (a) |
| 04-02.1 | One walk exists and every frame rides it; a test fails if a second implementation appears | ✓ VERIFIED | `TestWalkPolicyLivesInOneFile` passes; `internal/screen/walk.go` holds the policy and `absorb.go` a mention of `usedCursors` fails it with the documented message (re-run this round). `walk.go` is untouched by the guard round |
| 04-02.2 | A quoted post is a row; `Path` tells two quotations of the same post apart | ✓ VERIFIED | `TestPostTargetsWalksTheQuoteChain`, `TestPostTargetsTheSamePostQuotedTwice`, `TestTargetTopNamesTheContainingBlock` pass; `render.Selection{ID, Path}` / `Target{ID, Path}` remain the selectable unit; the scratch probe re-confirms `enter` on a `Path`-less focal pushes nothing and spends 0 conversation requests |
| 04-02.3 | The thread frame holds the answer's order; never re-sorted, never re-derived into a tree; a reply whose parent is absent keeps its place | ✓ VERIFIED | `TestTheOrderIsTheAnswersOrder` compares the page against the raw payload's own module order and reply-to ids; re-run this round, no `sort.` call exists in `internal/xsource/detail.go`, `internal/screen/thread.go` or `internal/render/thread.go` |
| 04-02.4 | Opening renders from the copy already held, with `loading thread...`; page one replaces it; a failed first page keeps the copy and puts the failure on the status line | ✓ VERIFIED | `TestEnterOpensAConversationWithLoadingThenPage`, `TestFailedFirstThreadPageKeepsFocalAndFailure` pass; `TestQRestoresTheFrameWithZeroThreadRequests` pins the zero-request open and the byte-identical restore. `skip` (`model.go:699`) is byte-identical to the previous pass's copy, `open` is `update.go:855` (moved by the restructure's added arms; the arm that reaches it, `enter → ActionOpen`, is unchanged), and the cold-frame probe asserts the pane the open draws in all 100 combinations (48 own-first-row, 52 head-at-row-0) |
| 04-02.5 | Pagination is Phase 2's rules verbatim: single-flight, a repeated cursor or an empty page ends the walk, an ancestor row is not a trigger | ✓ VERIFIED | `TestWalkSingleFlight`, `TestWalkStallsOnACursorThatComesBack`, `TestWalkStallsOnAPageThatAddsNothingUnseen`, `TestThreadPaginationIsSingleFlight`, `TestThreadStallPrintsConditionAndNeverRetries` pass |
| 04-02.6 | The three partial-thread conditions are computed from the frame's own state only; the reply count is evidence, never printed | ✓ VERIFIED | `TestThreadPartialAbove`, `TestThreadPartialHidden`, `TestThreadReadsTheReplyCountWithoutPrintingIt` pass; `conversationOf` (`model.go:641`) hands the renderer no count — its fields are the posts, the flags and the statements only |
| 04-02.7 | Tombstones, limited and withheld posts are ordinary rows: they keep their place and indent, are selectable, and raise no failure | ✓ VERIFIED | `TestThreadVisibilityStatesAreOrdinaryRows` passes; `thread_visibility_80.golden` shows `limited`, `post unavailable` and `post withheld` each at the block's own indent, in payload order; the frame is byte-identical to the previous pass's copy |
| 04-03.1 | The conversation frame is the detail view: one frame, one list, no second post-only screen and no second back-stack step | ✓ VERIFIED | The kind set holds five surfaces and **none of them is post-only**: `frameThread` is the one that holds a `Thread`, and the join test (`cover_e2e_test.go:730`) walks a shipped key to every declared kind and requires its covered rendering, 5/5. `Model.open` (`update.go:855`) pushes exactly one frame per press and its Kind is `frameThread`; `q` pops one via `m.back()` (`update.go:984`). `view.pane`'s `case f.Kind == frameThread: return m.scrolled(m.blocksAt(f, w), f)` (`view.go:117-118`) — one list, the thread's own. The guard round's new join (`dispatch_test.go`) adds that the arm reaching `open` is the only arm for `ActionOpen`, and that `ActionOpen` is the only action bound to `enter` |
| 04-03.2 | A tombstone renders `post unavailable`, a withheld post `post withheld` as the block's only body row at its own indent; a limited post keeps its text with `limited`; all selectable, `enter` on a tombstone opens it | ✓ VERIFIED | `TestGoldenFrames/thread_tombstone_focal_80` passes with the head row `> … focal` above `>post unavailable`; `TestEnterOnATombstoneOpensItsConversation` passes. Re-planting `if !(b.surface && headless(p))` (dropping the `!focal` exception, `block.go:233`) fails that golden exactly as recorded — the head+focal rows are gone, `>post unavailable` alone |
| 04-03.3 | `partial thread` prints only under the three conditions, each at its documented place, and the end of the reply list prints nothing | ✓ VERIFIED | `thread_partial_80/120`, `thread_empty_80`, `thread_chaincut_80` are stored and byte-compare (byte-identical to the previous pass's copy); `TestCorpusAudit`'s thread-absence arm asserts the stopped/above/hidden statements and the recording prints no `partial` word; deleting `thread_partial_80.golden` fails 10 top-level tests (re-run this round) |
| 04-03.4 | A ranked reply list is never presented as chronological: no per-group heading, no sort word, no numbering | ✓ VERIFIED | `TestCorpusAudit` forbids `load more`, `ranked`, `chronological`, `avatar` over every thread frame (`corpus_test.go:683-697`); re-adding `ranking` fails it on `thread_partial_80` and `thread_partial_120` row 8 with the documented message (re-run this round) |
| 04-03.5 | A quoted post renders in place, indented, never elided or expanded into a second top-level row, at most two quote levels deep, and `enter` opens it | ✓ VERIFIED | `TestEnterOnAQuotedPostPushesTheQuote` passes (a `Path`-bearing row pushes a frame whose focal post is the quoted id, at zero request cost). The depth claim is the prefix, not the parser: every shipped grammar caps the prefix at `Markers.QuoteDepthCap = 2` (`grammar.go:292`), while the parser's own bound is `maxQuoteDepth = 4` (`tweets.go:228`). `thread_sel_quote_80` / `thread_sel_ancestor_80` / `thread_sel_reply_80` are stored and byte-compare |
| 04-03.6 | The thread's frames contain no engagement count anywhere, and no partial-thread statement prints the number it came from | ✓ VERIFIED | `TestThreadPrintsNoCountDigit` and the corpus audit's count scans pass (`corpus_test.go:146` over every stored frame, `corpus_test.go:711` over the thread frames). Planting the focal reply count into `ThreadBlocks` and re-blessing with `-update` still fails the audit: "the stored frame thread_60/80/120 prints 596, an engagement count its own posts carry" |
| 04-03.7 | Every literal the thread frame can print has both spellings | ✓ VERIFIED | `TestTheMarkerSetsAreTotal` walks every shipped grammar and requires all eight new slots non-empty; `thread_color_{truecolor,256,16,mono}.golden` exist. The covered thread surface carries no host, URL, product or repair word in the loaded, failed or stalled states (scratch probe, 15/15), and the eight covered thread frames are byte-identical to the previous pass's copy |
| 04-03.8 | The stored frames are the specification at 60/80/120/200 with the adversarial content, and the shell composes no bytes of its own | ✓ VERIFIED | `golden_thread_test.go` stores `thread_60/80/120/200` through Phase 2's loader; **23 `thread_*.golden` + 8 `cover_*_thread_*.golden`** exist and byte-compare, and all 31 are byte-identical to **both** the previous pass's copy and the pre-fix copy; `view.pane`'s `case f.Kind == frameThread` hands `m.scrolled(m.blocksAt(f, w), f)` to the pane, the same grammar the geometry is measured in, and the shell writes no row of its own. The guard passes 20/20, the scratch probe repeats it at W ∈ {60,80,120,200} for 40/40 (40/40 moved), and the cold-frame arm passes 100/100. `-update` over both testdata trees produced **no diff** |

**Roadmap Success Criteria (ROADMAP.md, Phase 4) — the non-negotiable contract**

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| SC1 | Opening a post reads the focal post and its reply thread page by page, never one unbounded burst, and leaving restores the previous list, selection and scroll | ✓ VERIFIED | 1 `Thread` call per page, 0 on push (`TestQRestoresTheFrameWithZeroThreadRequests`); `q` restores the list frame byte-identically with the same target selected and the same anchor. The key that opens it (`enter → ActionOpen → m.open()` at `update.go:855`) and the key that leaves (`q → ActionBack → m.back()` at `update.go:984`) are the same arms they were, as the mechanical diff of the restructure shows |
| SC2 | Inline quoted posts render in place and can be opened as their own conversation | ✓ VERIFIED | Quote rows carry the selection cell and the quote prefix (`TestThreadBlocksTheUISpecFrameAt80Columns`); `enter` on one pushes the quote's frame (`TestEnterOnAQuotedPostPushesTheQuote`) |
| SC3 | Tombstones and limited-visibility placeholders render as themselves, and a structurally unreachable reply section says `partial thread` instead of truncating silently | ✓ VERIFIED | Visibility frames + the three conditions; focal tombstone carries `focal` |

**Score:** 22/22 must-haves verified (0 present-and-behaviour-unverified).

### Required Artifacts

`gsd-tools verify artifacts` returns `total: 0` for all three plans: their `must_haves.artifacts` are free-form strings, not the `{path, provides}` objects the verb parses, so the tool is not applicable and every artifact below was checked by hand.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/xsource/testdata/fixture-tweetdetail.json` | the recorded body, byte-identical to the evidence file | ✓ EXISTS + SUBSTANTIVE | sha256 `526ca1f8081f…`, identical on both sides (re-run this round) |
| `…-shapes.json`, `…-empty.json`, `…-tombstone.json` | synthetic bodies for the shapes no recording carries | ✓ EXISTS + SUBSTANTIVE | 15/1/3 entries; carry `SelfThread`, `ShowMoreThreads`, `TweetWithVisibilityResults`-wrapped, `TweetTombstone`, `withheld_in_countries`, the rail, prompt, composer, promoted and `who-to-follow` entries |
| `internal/xsource/testdata/provenance.json` | one row per fixture | ✓ EXISTS + SUBSTANTIVE | 11 rows / 11 fixtures; every row has a non-empty `redaction` |
| `internal/xsource/capture_test.go` | gated capture harness | ✓ EXISTS + SUBSTANTIVE | `X_TERMINAL_CAPTURE` gate: `--- SKIP: TestCaptureTweetDetail` names the variable |
| `internal/xsource/fixture_test.go` | provenance, source-naming, identity tests | ✓ EXISTS + SUBSTANTIVE | all with real assertions; the identity arm ran and failed correctly under a planted handle this round |
| `internal/domain/thread.go` | `ThreadPage`, `Group`, `GroupKind`, the `Threads` port | ✓ EXISTS + WIRED | `var _ domain.Threads = (*Adapter)(nil)` compiles |
| `internal/xsource/detail.go` | `ParseThread` and `(*Adapter).Thread` | ✓ EXISTS + WIRED | consumed by `internal/fake/thread_parity_test.go` and the composition root |
| `internal/screen/walk.go` | `Target`, `Walk[T,P]`, the sealed `Request` union, `RefreshMode`, the whole policy | ✓ EXISTS + WIRED | both the list and the thread frames ride it; later phases added their own `Request` cases in the same file |
| `internal/screen/thread.go` | the thread reducer, the two-stage load, the three conditions, refresh | ✓ EXISTS + WIRED | `internal/tui` drives it through `ThreadRequested`/`ThreadFetched`/… |
| `internal/screen/effect.go` | `Effect.Req Request` | ✓ EXISTS + WIRED | `Req Request` present; `request()` switches on the union |
| `internal/render/block.go` | selection widened to id + containing chain; headless focal draws grammar head + `focal` | ✓ EXISTS + WIRED | `Selection{ID, Path}`; `!(b.surface && headless(p) && !focal)` at `block.go:233` is the A1-T exception (re-planted and re-falsified this round) |
| `internal/render/thread.go`, `grammar.go`, `copy.go`, `status.go` | the conversation's blocks, `ReplyIndent`, the markers, the literals | ✓ EXISTS + WIRED | `ThreadBlocks` is the shell's thread pane; `TestThePlainMeasureIsPhaseTwos` pins Phase 2's measure at replyLevel 0; every thread golden is byte-identical |
| `internal/render/golden_thread_test.go` + `testdata/TestGoldenFrames/` | the stored frames | ✓ EXISTS + SUBSTANTIVE | 23 `thread_*.golden` at 60/80/120/200 plus the four rungs, plus 8 `cover_*_thread_*.golden`; the corpus holds 263 render frames and byte-compares, the corpus audit, the widened leak scan and `-update` are all green with no diff |
| `internal/render/corpus_test.go` | the adversarial audit extended over the thread frames | ✓ EXISTS + SUBSTANTIVE | fails on a one-cell edit, a deleted frame, a re-blessed violation, a re-added forbidden word and a re-blessed leak (all re-planted this round) |
| `internal/fake/thread.go` + `testdata/thread*.json` | the fixture-backed conversation source | ✓ EXISTS + WIRED | `ThreadRecorded/Shapes/Tombstone`, `Calls`, `FailNext`, `StallWith`; `-update` regenerates the three fixtures byte-identically (re-run: no diff) |
| `internal/keys/keys.go` | the `enter` row, `Screens`, the `t` row | ✓ EXISTS + WIRED | **byte-identical this round** (`cmp` against the previous pass's copy): `enter → open` (screens `all`) remains the only open-a-conversation row; `t → switch-list` (screens `home`). The fix round touched only the `ByKeystroke` comment |
| `internal/tui/model.go`, `update.go`, `view.go` | the frame kind, `enter`, the thread status line and identity; `blocksIn`/`blocksAt` unify measure and paint | ✓ EXISTS + WIRED | geometry/`rowsAbove`/`skip`/`scroll`/`step`/`pane`/`scrolled` all call `blocksAt`; `skip` (`model.go:699`), the thread status field (`view.go:292`) and the kind enumerator (`model.go:196`) are byte-identical to the previous pass's copy. `update.go` is the guard round's one modified file: `press`'s three shared arms were split one-arm-per-action, and the new `handledActions`/`HandledActions()` sit beside it |
| `internal/tui/dispatch_test.go` | **new this round** — the keymap⇄dispatcher join, both directions plus the source sweep | ✓ EXISTS + SUBSTANTIVE + WIRED | 259 lines, two top-level tests; a 31st row (G1) and a handler with no row (G2) each fail it with the documented messages, and a bogus case label (G3) fails the build. **Added to this report's covered set** |
| `internal/cli/feed.go` | the second source seam | ✓ EXISTS + WIRED | `threadSource` beside `feedSource`, both handed to `tui.Deps`; the thread seam and the `Threads:` field are unchanged by the fix round in the diff |

**Artifacts:** 18/18 verified. The eight `internal/render/testdata/TestGoldenFrames/cover_{servicelog,gitlog,ci,internal}_thread_{80,120}.golden` frames are thread frames the phase's renderer produces, audited by the same corpus scan, read by the widened leak scan and exercised by the geometry guard; all 31 thread goldens are byte-identical to both reference copies. `internal/tui/dispatch_test.go` was added to the covered set this round; no path was removed.

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `detail_test.go` | the recorded body | `readFixture(recordedDetailPath)` + raw-JSON arithmetic | ✓ WIRED | the decoder's answers are compared against the payload, never against themselves |
| the adapter | the volatile registry | `opTweetDetail` name only | ✓ WIRED | no `.go` file in the package spells the query id; the `opnames` audit check enforces the same totality over the classification table, and the audit exits 0 |
| `domain.ThreadPage` | `internal/screen` → `internal/render` | the port | ✓ WIRED | no module kind, instruction name or cursor type is a field of the conversation's values; the only `SortIndex` in `internal/domain` is `saved.go:127`, the mirror's own value |
| `screen.Walk` | both frames | the page cache, dedupe, stall, latch | ✓ WIRED | answers are routed by opaque frame id and dropped when no frame holds the id |
| the shell's movement | the frame's own row sequence | `Targets()` + `Move` | ✓ WIRED | a quote is exactly one step from the post that quotes it; `g` leaves the frame unmoved so the pane shows the frame's own first row |
| `walk.Next` | the next request | `Effect.Cursor` | ✓ WIRED | the decoder's own cursor is stored and handed back untouched |
| `xsource.ParseThread` | `internal/fake`'s fixtures | `thread_parity_test.go -update` | ✓ WIRED | `-update` produced no diff on the current tree (re-run this round) |
| the shell | `internal/render` | `view.pane` → `blocksAt` → `ThreadBlocks` | ✓ WIRED | measure and paint share `blocksAt`; the shell composes no bytes of the conversation |
| `screen.Thread`'s three questions | `Grammar`'s markers | `conversationOf` → `ThreadBlocks` | ✓ WIRED | the flags come from the frame, the words from the grammar; `conversationOf` carries no count |
| `internal/keys`'s table | `press`'s arms | `handledActions` + `dispatch_test.go` | ✓ WIRED | the new join: every row's action has exactly one arm and every arm a row, both directions, with the declarations swept from source; plants G1/G2/G3 show it fails when either end drifts |

**Wiring:** 10/10 connections verified.

## Requirements Coverage

| Requirement | Status | Blocking Issue |
|-------------|--------|----------------|
| THRD-01: open a post and read its reply thread, paginated page by page | ✓ SATISFIED | One request is one page; the walk's own cursor; full suite green |
| THRD-02: read inline quoted posts and open a quote as its own conversation | ✓ SATISFIED | Quote rows are selectable; `enter` pushes the quote's frame at no request cost |
| THRD-03: tombstones and limited-visibility placeholders, and an honest "partial thread" | ✓ SATISFIED | Visibility frames; focal tombstone carries `focal`; the three conditions each print their own line |
| REND-01 (inherited): no engagement counts | ✓ SATISFIED | two corpus scans over the stored frames; the reply count is evidence only |
| SEC-01 (inherited): the cookie never appears in a fixture or output | ✓ SATISFIED | no session-shaped key and no session cookie value in any fixture; the handle/id arm runs and fails correctly when identity is supplied |

**Coverage:** 5/5 requirements satisfied.

## The implementer's four "missed by the plan" items, adjudicated

1. **A withheld post in a *list* keeps Phase 2's wording (`post unavailable`), not `post withheld`** — **acceptable reading.** `domain.Withheld` is produced on exactly one line, in the thread decoder; no list read can produce it. The must-have binds the conversation frame, and `block.go`'s `surface` branch gives it `post withheld` there.
2. **A headless tombstone *focal* draws no head row, so no `focal` marker** — **closed.** `thread_tombstone_focal_80.golden` starts with `> … focal` then `>post unavailable`; re-planting the pre-fix `!(surface && headless)` fails that golden (re-run this round).
3. **`ranking` was dropped from the audit's forbidden substrings** — **closed as documentation.** The comment and failure message name the post-body collision; re-adding `ranking` fails those frames (re-run this round). `ranked`/`chronological` still enforce the must-have.
4. **`CopyEmptyBody` is reachable on a thread frame's `no replies`** — **acceptable reading.** It is what the plan's Task 1 asked for, and `assertEmptyCopyOnlyAfterAnEmptyRead` is widened deliberately: only the empty-state *heading* is exclusive to the empty pane.

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/tui/model.go` | 196 | `frameKinds()` — **previously dead, live since the amendment**: its comment's claim is exactly what `TestEveryFrameKindIsCheckedUnderACover` (`cover_e2e_test.go:730`) enforces in both directions | ℹ️ closed | the third pass's warning stays retired: a sixth kind planted in both places still **fails** that test (re-run this round) |
| `internal/render/corpus_test.go` | 2036-2196 | the leak scan — **previously narrowed to the saved frames, now over every covered frame**, with its own plant arm planting into a covered conversation frame | ℹ️ closed | the third pass's warning stays retired: the plant fails it, naming all eight covered thread frames (24 findings, re-run this round) |
| `internal/tui/update.go` | 1297-1315 | the guard round's `handledActions` is **data the compiler cannot check**; the join test is the only thing holding it to the switch | ℹ️ none | not a silent no-op today: the set is joined to the keymap in both directions, every declared action must be a case label, and a 31st row and an unbound arm each fail it (plants G1/G2) |
| all round-touched files | – | `TODO`/`FIXME`/`XXX`/`HACK`/stub markers; dead constants or dead helpers added | ℹ️ none | re-scanned this round over `update.go`, `dispatch_test.go` and this phase's covered files: no marker hits. The guard round adds no dead weight: `handledActions` is read by the test, `HandledActions` by the test too (see note 7) |

**Anti-patterns:** 0 open (2 previous warnings closed, 0 new).

## Falsification attempts (re-run 2026-10-15 on the frozen tree, plant-by-plant on `/tmp/x4v6`, reset from `/tmp/x4v6_pristine` between plants)

| Planted / attacked | Result |
|---|---|
| **Guard: a 31st keymap row carrying a fresh action** (`ActionProbe` const + a row on `f13`, no arm, no enumeration entry) | **Both dispatch joins FAIL**: `the keymap binds "probe" (f13) and press has no arm for it: the key is a documented no-op`, `the table holds 31 rows and handledActions names 30 actions`, `keys.go declares ActionProbe and handledActions does not name it`, `keys.ActionProbe is no case label in the shell` |
| **Guard: a handler with no row** (`ActionGhost` const + enumeration entry + `case` arm, unbound) | **Both dispatch joins FAIL**: `press handles "ghost" and no row of the keymap binds it: the arm is unreachable`, `the table holds 30 rows and handledActions names 31 actions`, `keys.go declares ActionGhost ("ghost") and no row binds it` |
| **Guard: a case label for an action that does not exist** | `go build ./...` FAILS: `internal/tui/update.go:311:12: undefined: keys.ActionThisPackageDeclaresNoSuchAction` |
| **Geometry: the pane measuring with the plain reader's rows while painted in the cover's grammar** (`rowsAbove` in `model.go` and `scrolled` in `view.go` measuring `render.Plain`) | The shipped guard FAILED in **all 16 cover subtests** (4 plain controls passed), e.g. `servicelog/80/list`: "the selected block's head is not a row of the pane: pane row 22 of 23, 7 rows dropped … the pane measured itself in one grammar and painted itself in another"; the all-width probe failed **31/32** cover cases at 60/80/120/200 (8/8 plain controls passed) — the one case that did not fire is still the recorded blind combination `servicelog/60/thread` |
| **Pane opening rule: the pre-fix `skip` = `max(0, above-Anchor)`** (the frame's opening state ignored, `model.go:699`) | The shipped guard passed (its frames are all moved frames) and the **cold-frame probe failed 48/100**, each "the head N rows down with M pane rows: the pane drops N rows, want 0" — the same count as recorded |
| **The widened leak scan: `saved.db doctor --pin mirror` in the covered thread corpus's focal body, re-blessed with `-update`** | **`TestCoveredFramesLeakNoRepairWords` FAILS** — the only failing top-level test in `internal/render` + `internal/tui`: **24 findings naming all eight** covered thread frames (`cover_{servicelog,gitlog,ci,internal}_thread_{80,120}` hold `mirror`, `saved.db`, `doctor`) |
| **The tui half's scan: a leak appended to a stored covered tui frame** (`search_people_gitlog_80.golden`) | The scanner FAILS, naming the frame and each word ("the covered tui frame search_people_gitlog_80 holds \"mirror\", a string the cover of gitlog did not author") |
| **`frameKinds`'s claim: a sixth `frameKind` added to the const block *and* to `frameKinds()`** | `go build ./...` clean, and `TestEveryFrameKindIsCheckedUnderACover` **FAILS** with both messages ("the shell declares the frame kind 5 and the covered check table has 0 rows for it …" and "the covered check table has 5 rows and the shell declares 6 frame kinds") |
| **Focal: re-planted `if !(b.surface && headless(p))` (no `!focal`, `block.go:233`)** | `TestGoldenFrames/thread_tombstone_focal_80` failed — the head+focal rows are gone, `>post unavailable` stands alone and the rest of the conversation shifts up |
| **Audit: re-added `ranking` to the forbidden list** | `TestCorpusAudit` failed on `thread_partial_80` and `thread_partial_120` row 8 with the corrected failure message |
| **Decoder: kept the promoted module items (`if false && …promotedItemMark`, `detail.go:294`)** | **6** top-level tests failed: `TestTheRecordedThreadIsOnePage`, `TestTheRecordedAnswerKeepsOnlyItsOwnConversation`, `TestTheOrderIsTheAnswersOrder`, `TestTheSyntheticShapesAreAllRead`, `TestNoPostFromTheRelatedRailOrAnAdvertisementReachesThePage` and `TestThreadParity` |
| **Fixture: deleted one `conversationthread` entry from `fixture-tweetdetail.json`** | 5 top-level tests failed: `TestTheRecordedThreadIsOnePage`, `TestEveryFixtureIsProvenanced`, `TestTheRecordedFixtureNamesItsSource`, `TestTheRecordedFixtureIsTheOneWeThinkItIs` and `TestThreadParity` |
| **Golden: one-cell edit in `thread_80.golden`** | `TestGoldenFrames/thread_80` failed the byte compare |
| **Golden: deleted `thread_partial_80.golden`** | **10** top-level tests failed, including `TestGoldenFrames` and `TestCorpusAudit` ("reading testdata/TestGoldenFrames/thread_partial_80.golden: no such file or directory") |
| **Count: the focal reply count planted in `ThreadBlocks`, then re-blessed with `-update`** | The byte compare passed after blessing, and `TestCorpusAudit` **still** failed: "the stored frame thread_60/80/120 prints 596, an engagement count its own posts carry" |
| **Walk: planted a `usedCursors` mention in `absorb.go`** | `TestWalkPolicyLivesInOneFile` failed: `absorb.go names the walk's "usedCursors": …` |
| **Identity: planted `"screen_name": "probehandle"` in a fixture, with `X_TERMINAL_AUTH_TOKEN`/`CT0`/`X_TERMINAL_VIEWER_HANDLE` set** | `TestNoFixtureCarriesViewerIdentity` failed: "testdata/fixture-tweetdetail-shapes.json carries the viewer's own handle"; the same run on the clean tree PASSES with the identity supplied and SKIPs without it |
| **Shell: `enter` on the thread's own focal post (`Path` empty)** | no frame pushed (stack depth unchanged), 0 `Thread` calls — scratch probe, passes on the clean tree |
| **Parity: `go test ./internal/fake -run TestThreadParity -update`** | no diff in the three generated fixtures |
| **Corpus: `TestGoldenFrames -update` over both testdata trees** | **no diff at all** — the stored frames, thread frames included, are exactly what the current tree renders |
| **Offline: the whole suite inside `unshare -rn` with only `lo` up** | 1573 passed, 0 failed, 8 skipped, 19 packages `ok` — no test touches the network |
| **The fix's whitelist: `savedMirrorError` reverted to `return err != nil`** | `internal/cli` **fails 4 tests**, naming the fix's own cases (`TestSavedMirrorErrorNamesOnlyTheFile`, `TestAShapeDriftedAnswerIsNotAFactAboutTheFile`, `TestASavedFrameNamesATransportFailureAsTheOrdinaryState`, `TestASavedFrameNamesTheRequestPathsOwnStates`) — the conversation frame is unaffected either way |
| **The fix's pins promise: `volatile.PinsFileName` renamed to `"pinsX.json"`** | `TestPinsPathIsTheRegistersName` **fails**: "the pins file's name is \"pinsX.json\", want pins.json: an operator's pins file is at that name, so a rename is a migration" and `PinsPath = "/tmp/x-data/pinsX.json"` |
| **The guard round's change set itself: `diff -rq /tmp/x4v5_pristine .`** | exactly `internal/tui/update.go` (modified) and `internal/tui/dispatch_test.go` (new), plus the parallel passes' `VERIFICATION.md` files; `internal/keys/keys.go` byte-identical (`cmp`); the four stored corpora and all 31 thread goldens byte-identical (`cmp`/`diff -rq`) |

## Human Verification Required

### (a) The viewer's own identity in the fixtures (SEC-01, 04-01.7) — **ACCEPT**
**Verified here:** always-on arms green; every `redaction` non-empty; the identity arm SKIPs when no session is reachable (re-run with `XDG_CONFIG_HOME`/`XDG_DATA_HOME` under `/tmp`, so the operator's real session was never opened); supplying `X_TERMINAL_AUTH_TOKEN`/`X_TERMINAL_CT0`/`X_TERMINAL_VIEWER_HANDLE` makes the arm run, pass clean, and **FAIL** under a planted handle, so the arm is live rather than vacuous.
**Human would still run (optional):** the same test against a real session that names a handle/id, or after a live capture. Not a blocker.

### (b) Live capture harness and tmux twin — **DEFER (acceptable)**
**Why defer:** `X_TERMINAL_CAPTURE=1` spends a real request from the account budget; the tmux twin needs a multiplexer session. Neither is graded-suite work (both SKIP in the offline run, by name).
**What a human runs:** `X_TERMINAL_CAPTURE=1 go test ./internal/xsource -run TestCaptureTweetDetail -v`, then re-check fixtures; capture thread frames inside/outside tmux at 80/120 and diff to `thread_*.golden`.

### (c) Shipped binary cannot take a fake source — **ACCEPT**
End-to-end composition was exercised through the shell with the composition's own `Deps` and the fake thread source. Pointing the shipped binary at a fake source is out of product design; not a verification gap for Phase 4.

### (d) The geometry and opening-rule arms' reach — **NOTED, not a gap**
The shipped guard asserts {80,120}; the scratch probe repeats it at {60,80,120,200} and passed 40/40 with 40/40 moved frames. Three honest limits, all re-measured this round: (i) the probe's plain-measurement arm is not *discriminating* in exactly one combination — `servicelog/60/thread`, measured again as the single cover case that survives the plant (31/32 fire), while the shipped guard's arm is discriminating in all 16 of its cover combos; (ii) the shipped guard exercises only **moved** frames, so 04-UI-SPEC's cold-frame rule has no shipped arm for the *conversation* frame — the arm for it lives in this pass's scratch probe (100/100 over 5 grammars × 4 widths × 5 pane heights, both branches, 48/100 under the pre-fix rule); (iii) the covered-conversation host scan is non-discriminating (the plain twin carries none of those strings either).

## Gaps Summary

### Critical Gaps (Block Progress)

None. All 22 must-haves verified; every falsification this report records was re-run against the frozen tree and either held (clean) or failed exactly as documented under its plant. **No regression from the guard round was found:** its whole change set is `internal/tui/update.go` plus `internal/tui/dispatch_test.go`, the restructure is provably behaviour-preserving (three hunks; one arm per action with each arm's body the one the shared arm had; no case reordered; the panic, prompt, help and cover-gate regions untouched), 24 named dispatcher-flow tests pass, and the conversation frame's bytes — all 31 thread goldens, and every thread-frame source file — are byte-identical to the previous pass's and the pre-fix copies.

### Non-Critical Gaps (Can Defer)

1. **CLOSED — `frameKinds()` was dead code with a false claim** (`internal/tui/model.go`). The amendment wired it; re-falsified this round: a sixth kind planted in both places fails `TestEveryFrameKindIsCheckedUnderACover` with both messages.
2. **CLOSED — the corpus leak scan had been narrowed to the covered saved frames.** It is now over every covered frame (`corpus_test.go:2036`, 170 frames) with an arm that plants every word into a covered **conversation** frame; `internal/tui` has its own scanner. Re-falsified this round: the plant fails, naming all eight covered thread frames, and the tui plant fails its scanner too.
3. **NOTE (P3) — the leak vocabulary is two hand-kept lists** — `coveredLeakWords` (`render/corpus_test.go:2145`) and `tuiLeakWords` (`tui/corpus_test.go:661`). They are identical today (8 words). The residual risk is drift: a word added to one list and not the other leaves that corpus's covered frames unscanned. Guard-hardening, no current behaviour impact.
4. **NOTE (P3) — the shipped geometry and opening-rule arms are narrower than the stored corpus** — the guard asserts {80,120} while the phase stores 60/80/120/200 (the other two widths live only in this pass's scratch probe), it walks only moved frames (the conversation frame's cold-frame branch has no shipped arm), one `servicelog/60/thread` combination is non-discriminating, and the covered-conversation host scan is non-discriminating. All measured this round and recorded under (d).
5. **NOTE (P3) — the covered thread frames carry no facts row** — the amendment's `MonthYear` claim renders the fourth facts clause for the covered *profile* frames only (`profileFacts` is the sole `AccountFacts` consumer). By design for a conversation (its status field is `ThreadListName`), recorded so the next reader does not look for a `since=` clause there.
6. **NOTE (P3) — the fix round's own promises are other phases' truths.** The `FileError` whitelist, the saved frame's capability releases, the `opnames` audit check, the pins-path constant and the `PostPage.Requested` doc all belong to the saved, audit and config surfaces (phases 6 and 8). Phase 4's must-haves do not rest on any of them; the guard round, by contrast, is this phase's own dispatcher and *is* evidence for SC1.
7. **NOTE (P3) — `HandledActions()`'s only caller today is the in-package dispatch test.** Its doc says it exists so the `keys` subcommand and the help overlay's golden *can* ask for the join from their own side; neither does yet, and no test outside `internal/tui` calls it. The exported seam is deliberate and exercised (the test asserts the copy is a copy), so it is not dead weight — but a reader who greps for consumers finds one test. Worth a consumer or a narrower doc if either surface is ever to read the join.

## Recommended Fix Plans

None — the phase goal is achieved, no must-have failed, and the guard round closed nothing that was open here while breaking nothing. The remaining items are P3 notes about the width of the guards, one hand-kept word list, and one exported seam with a single caller.

## Covered set

The **82** paths in the frontmatter are this phase's three PLAN/SUMMARY pairs plus the implementation, test and evidence files the phase's truths rest on — including the eight `cover_*_thread_*.golden` frames. **One path was added this round: `internal/tui/dispatch_test.go`**, the new join that now guards the dispatcher every one of this phase's key truths routes through (`enter`/`q`/`g`, SC1). The digest moved for two reasons: `internal/tui/update.go` changed (the restructure) and `dispatch_test.go` joined the set (the 82-file digest went `c9069cf5…` → `6530733f…`; the previous 81-file digest was `c9069cf5…`). The fix round's other 15 files deliberately stay out: `internal/store/*` and `internal/screen/saved.go` are the mirror's own evidence, `internal/audit/*` is the audit's, `internal/config/*` and `internal/xweb/volatile/pins.go` are the config and registry surfaces' — and none of them is a thread-frame path.

## Verification Metadata

**Verification approach:** Goal-backward re-verification against each PLAN's `must_haves` and the ROADMAP Success Criteria; full suite (online + `unshare -rn` offline); a byte diff of the guard round against the previous pass's own copy (two source files, no golden) and a `cmp` of all 31 thread goldens against two reference copies; a mechanical read of the `press` restructure plus 24 named dispatcher-flow tests for the flows that ride it; three plants against the new guard (a 31st row, an unbound arm, a bogus case label); adversarial re-plant of every recorded defect and of the fix round's two claims; four scratch probes (geometry at every stored width, the cold-frame opening rule, the covered conversation surface, `enter` on the focal); a corpus census; stored-frame `-update` comparison; the shipped audit; digest recomputation.
**Must-haves source:** the three PLANs' frontmatter (`truths`, `artifacts`, `key_links`) plus ROADMAP.md's three Success Criteria; the plans declare no `prohibitions`.
**Automated checks:** 1573 tests / 22 packages green offline; geometry/focal/audit/walk/identity/parity/leak-scan/kind-join/dispatch-join guards; 22 planted-violation experiments; the audit binary's 10 checks; fingerprint and phase-completeness via `gsd-tools`.
**Human checks required:** none blocking; (b) remains an optional live/tmux check a human may still run.
**Not verified, and what would be needed:** live x.com capture and tmux twin byte-compare (deferred above). The shipped binary was not pointed at a fake source (accepted). The cold-frame opening rule has no shipped arm for the conversation frame — the evidence is this pass's scratch probes (`/tmp/x4v6`), reproducible from the commands above. The real `~/.config/x-terminal/session` and `~/.local/share/x-terminal/saved.db` were never opened.
**Report-only guarantee:** the working tree was not modified except for this file. Every destructive experiment used `/tmp/x4v6`, reset from `/tmp/x4v6_pristine` between plants; the repository differs from this pass's own pre-pass byte copy in nothing outside `.planning`, and the repository's goldens, fixtures and sources are byte-identical to how they were found.

---

*Verified: 2026-10-15T07:14:00Z*
*Verifier: omp (subagent), sixth re-verification over the three plans and the ROADMAP against the frozen tree, adjudicating the guard round's `press` restructure and its new dispatch join*
