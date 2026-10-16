---
phase: 06-bookmarks-likes-and-the-mirror-only-store
verified: 2026-10-15T07:24:00Z
status: passed
score: 29/29 must-haves verified
covered_files:
  - .planning/phases/06-bookmarks-likes-and-the-mirror-only-store/06-01-PLAN.md
  - .planning/phases/06-bookmarks-likes-and-the-mirror-only-store/06-01-SUMMARY.md
  - .planning/phases/06-bookmarks-likes-and-the-mirror-only-store/06-02-PLAN.md
  - .planning/phases/06-bookmarks-likes-and-the-mirror-only-store/06-02-SUMMARY.md
  - .planning/phases/06-bookmarks-likes-and-the-mirror-only-store/06-03-PLAN.md
  - .planning/phases/06-bookmarks-likes-and-the-mirror-only-store/06-03-SUMMARY.md
  - .planning/phases/06-bookmarks-likes-and-the-mirror-only-store/06-04-PLAN.md
  - .planning/phases/06-bookmarks-likes-and-the-mirror-only-store/06-04-SUMMARY.md
  - internal/audit/audit.go
  - internal/audit/audit_test.go
  - internal/audit/guards.go
  - internal/audit/readonly.go
  - internal/audit/source.go
  - internal/cli/audit.go
  - internal/cli/doctor.go
  - internal/cli/feed.go
  - internal/cli/feed_test.go
  - internal/cli/sync.go
  - internal/cli/sync_test.go
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/domain/saved.go
  - internal/domain/source.go
  - internal/fake/mirror.go
  - internal/fake/parity_test.go
  - internal/fake/saved.go
  - internal/fake/testdata/pages-saved.json
  - internal/keys/keys.go
  - internal/render/capture_panic_test.go
  - internal/render/capture_tmux_test.go
  - internal/render/copy.go
  - internal/render/corpus_test.go
  - internal/render/cover.go
  - internal/render/golden_cover_corpus_test.go
  - internal/render/golden_test.go
  - internal/render/grammar.go
  - internal/render/profile.go
  - internal/render/prompt_test.go
  - internal/render/saved.go
  - internal/render/saved_test.go
  - internal/render/status.go
  - internal/render/status_test.go
  - internal/render/testdata/TestGoldenFrames/bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_absent_rows_80.golden
  - internal/render/testdata/TestGoldenFrames/bookmarks_never_synced_80.golden
  - internal/render/testdata/TestGoldenFrames/empty_120.golden
  - internal/screen/saved.go
  - internal/screen/saved_test.go
  - internal/store/boundary_test.go
  - internal/store/migrate.go
  - internal/store/migrate_test.go
  - internal/store/read.go
  - internal/store/schema.go
  - internal/store/schema_test.go
  - internal/store/store_test.go
  - internal/store/sync.go
  - internal/store/sync_test.go
  - internal/tui/corpus_test.go
  - internal/tui/cover_e2e_test.go
  - internal/tui/dispatch_test.go
  - internal/tui/model.go
  - internal/tui/profile_test.go
  - internal/tui/saved.go
  - internal/tui/tui_test.go
  - internal/tui/update.go
  - internal/tui/view.go
  - internal/xsource/saved.go
  - internal/xsource/saved_test.go
  - internal/xsource/testdata/fixture-bookmarks.json
  - internal/xsource/testdata/fixture-likes.json
  - internal/xsource/user.go
covered_digest: "v3:sha256:917004abc5b7e8536d05cb94924f10eda5bf8af3119bc5480baa6b407a63c595"
behavior_unverified: 0
behavior_unverified_items: []
coincidental_reliance_items: []
---

# Phase 6: Bookmarks, Likes and the Mirror-Only Store Verification Report

**Phase Goal:** The user reads their own bookmarks and likes from the active session and can open that mirror on disk with the network down — while the timeline stays structurally unable to reach disk.
**Verified:** 2026-10-15T07:24:00Z (sixth refresh; the dispatcher-guard commit verified over the frozen tree)
**Status:** passed

## Refresh Notes (why this report moved again)

This is the sixth refresh, and the first that re-verifies a **guard commit** rather than a plan or a
fix round. One commit (`df80562`, "test: join the keymap to the shell's dispatch, both directions")
added `internal/tui/dispatch_test.go` and restructured the `press` switch in
`internal/tui/update.go` so each action has exactly one arm; `handledActions` and the exported
`HandledActions()` (`update.go:1286-1315`) are the join point. No action, no key, no golden and no
other file moved (`git diff d686f36..df80562 --stat` = those two files). Every claim below was
re-derived from the frozen tree and then falsified by planting what it forbids in a `/tmp` copy; the
repository was left byte-identical (a pristine copy was kept and compared after every batch).

**1. The dispatcher refactor is behaviour-preserving, proven by differential execution, not by
reading the diff.** The multi-label arms became one arm per action with the identical body:
`case ActionBookmarks, ActionLikes: return m.savedAction(action)` became two arms of the same call,
`ActionCover1..4` four arms of `m.numberedCover(action)`, and the eight movement actions one arm
each of `m.move(action)`. A probe was compiled into the package that drives **86 checkpoints** —
every bound key, two unbound keys, the prompt's own keys, the panic gate's keys and its no-op set,
`b`/`l` open, paging, `r` sync on a saved frame (plain and covered), the pop, the unreadable-mirror
pair and the covered gate — and dumps each step's frame kind, frame depth, display state,
help/prompt flags, source call count, persist log and the **rendered frame bytes**. Run with the new
`press` and again with `d686f36`'s `press` from the same file positions, the two transcripts are
byte-identical: `diff` over 182,165 bytes / 86 checkpoint blocks is empty. Every flow the check names
appears in it — the keymap⇄model exhaustion arm (an unknown key still returns `m, nil`), the
gated actions under a cover (`i`/`y`/`o` → the gate notice), the prompt's keys (`g` as text,
`ctrl+w`, `ctrl+u`, `enter`, `esc`), the panic gate's two keys and its no-ops, and the frame
pushes/pops (`enter`, `a`, `b`, `l`, `q`).

**2. The guard fails in both directions, and its own plants were re-run.** A 31st keymap row with a
fresh action fails the join twice (`dispatch_test.go:63: the keymap binds "frobnicate" (f9) and press
has no arm for it: the key is a documented no-op`, plus the `:79` 31-vs-30 anti-vacuity arm) and the
source sweep twice (`:128`, `:159`). A handler with no row fails it from the other side
(`:72: press handles "orphan" and no row of the keymap binds it: the arm is unreachable`; `:125`). A
planted bogus case label (`case keys.ActionThisPackageDeclaresNoSuchAction:`) is refused by the
compiler (`undefined: keys.ActionThisPackageDeclaresNoSuchAction`), the backstop the file's own
comment claims. Removing one real arm (`case keys.ActionLikes`) leaves the guard **green** — the
sweep is an AST read of case-clause labels and the action is still a `case` in `saved.go` — and the
behavioural tests catch it instead (`TestSavedFrameIssuesNoSourceRequest`,
`TestLikesUnavailableLeavesOthersAlone`, `TestTheCoveredSavedPaneNamesNothingOfTheProduct` FAIL).
Two comment claims overstate what is pinned; both are recorded, not gaps (Accepted Tech Debt 4-5).

**3. The whitelist and the release rules were re-run and re-failed exactly as recorded.** Predicate
broadened to `err != nil` → the same cli set FAILs; `*xsource.ParseError` re-admitted →
`TestAShapeDriftedAnswerIsNotAFactAboutTheFile` FAILs at `feed_test.go:498/:520/:526`; `Open`'s and
`Page`'s `FileError` wraps removed → `feed_test.go:694` / `store_test.go:189` FAIL; `Status`'s
wrap removed → still green (the recorded hole, Accepted Tech Debt 3). The four release rules
(screen: no page release; a page clearing every state; `StopFailure` releasing too) fail
`saved_test.go:85/:101`, `:108`, `:128`; clearing on `StatusRead` fails the shell at
`tui_test.go:1576` while `internal/screen` alone stays green. The covered `store cannot be read`
state without its failing read fails `cover_e2e_test.go:1454` in both widths. `DELETE` after the
`MarkAbsent` UPDATE, the old banding rule, the timeline-import plant, the synced-field golden plant,
the plain-grammar plant at the saved pane's call sites, `ActionFirst` disabled, the plain clause in
`coverGitFacts` and the stale `ListLatestTweetsTimeline` in `opClass` all reproduce their recorded
failures text for text (the arms below; the four for the guard are recorded there too).

**4. The mirror truths were re-adjudicated, not assumed.** This pass ran the phase's named tests
by name over the frozen tree — the `-run` sets below total 117 `--- PASS:` lines including subtests
(44 `internal/tui`, 27 `internal/render`, 12 `internal/cli`, 18 `internal/store`, and 16 across
`internal/xsource`/`internal/fake`/`internal/screen`) — and re-derived 22-25 and
27-28 from `press`/`savedAction`/`refresh`'s saved branch plus those tests and the probe. Truths the
named arms do not reach directly were re-checked by their packages' full runs (all 19 `ok`). All 29
hold; the holes remain the recorded notes (the unpinned `Status` mark, and the guard's two
overstated comment claims this pass adds to Accepted Tech Debt).

**Nothing else moved.** `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty;
`go test ./... -count=1 -v` → **1573 `--- PASS:` lines including subtests, 0 failures** (22
packages; the 19 carrying tests all `ok`; the 1571 of the last pass plus the guard's two tests); the
phase's seven packages re-run inside `unshare -rn` with `lo` up (store, xsource, fake, tui, render,
cli, screen — all `ok`); `go run ./cmd/x-terminal audit` → **10 checks, 0 findings, 893 files**,
exit 0. **No golden changed this round.**

`covered_files` gained two paths this pass: the guard itself (`internal/tui/dispatch_test.go`) and
the table it joins against (`internal/keys/keys.go`) — the join reads the declarations out of
`keys.go` and holds the dispatcher to all 30 of its rows, so that file is part of this claim's
surface.

## Goal Achievement

### Observable Truths

Roadmap success criteria are satisfied by the plan truths below (SC1←06-02; SC2←06-04
likes capability; SC3←06-03+06-04 offline/zero-request; SC4←06-01 seal/boundary;
SC5←06-02 pin path). This pass re-derived the truths the guard commit touches (22, 23, 27 — the
`press` routes `b`/`l`/`r` ride) plus the mirror truths it re-ran (3, 4, 16, 20, 24, 25, 28, 29),
falsified each, and regression-checked the other 21 with the full suite, the phase's named tests,
the 86-checkpoint differential probe and the offline runs; all 29 hold. The holes are the two
recorded notes (the unpinned store `Status` mark; the guard comment's two overstated claims), not
gaps.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | One saved item written through the store write API comes back through a second `query_only` connection as a `domain.Post`, and `journal_mode` reads back as `wal` | ✓ VERIFIED | `TestRoundTripOneSavedItem` reads `PRAGMA journal_mode` back and fails unless `wal` (store_test.go:52-58), then round-trips post text/author/FetchedAt; `TestOpenSetsQueryOnlyOnReader` refuses a reader `INSERT`. Both PASS by name this pass (and the whole `internal/store` package under `unshare -rn`) |
| 2 | File holds exactly the three declared tables/columns and closed `meta` keys; extras fail allowlist | ✓ VERIFIED | `declaredTables` = meta(2)+saved_items(7)+sync_runs(9); `declaredMetaKeys` = schema_version, owner, last_success_bookmark, last_success_like. `TestSchemaAllowlist` (+ its fail arms) PASS by name this pass |
| 3 | Store exported surface is sealed: mutating APIs take `[]domain.SavedItem` / `SavedKind`+instant / `SyncResult` / owner; no exported DB handle or arbitrary statement runner | ✓ VERIFIED | `TestExportedSurfaceAllowlist` PASS by name this pass — the allowlist is eleven names (`Open`, `Close`, `Path`, `Page`, `Status`, `Upsert`, `MarkAbsent`, `RecordRun`, `SetOwner`, `ReadFacts`, `FileError`, schema_test.go:68-72) with the ≥4-mutator anti-vacuity arm; `go doc -all ./internal/store` lists `FileError` as the surface's one exported type and no `*sql.DB`, statement runner or arbitrary key/value writer in any signature. **Re-planted this pass**: the type renamed (`FileError`→`fileError`, the allowlist's own entry put back) → FAIL `schema_test.go:76: exported surface missing "FileError" (have [Error Unwrap Page Status ReadFacts Store Open Close Path Upsert MarkAbsent RecordRun SetOwner SavedFetcher Viewer Mirror NewMirror Page Status Path Begin Step Abandon])`; reverted |
| 4 | AST scan: no home/thread/search surface imports `internal/store`; planted fixture fails; anti-vacuity arms fire | ✓ VERIFIED | `TestBoundaryImportScan` PASS; **re-planted this pass** `_ "…/internal/store"` into `internal/xsource/home.go` → FAIL `boundary_test.go:45: timeline surfaces import internal/store`; reverted |
| 5 | Full fake timeline session leaves data dir with only prior files + `saved.db`(-wal/-shm) | ✓ VERIFIED | `TestBoundaryDataDirAfterTimelineSession` PASS by name (also asserts `saved_items` holds 0 rows afterwards) |
| 6 | Schema upgrade keeps rows; newer schema refused with `ErrSchemaNewer` | ✓ VERIFIED | `TestMigrateKeepsRowsAcrossUpgrade`, `TestMigrateRefusesNewerSchema` PASS by name |
| 7 | DDL + `meta.schema_version` bump are one transaction; intentional mid-migration failure rolls back | ✓ VERIFIED | `TestMigrateTornUpgradeRollsBack` PASS by name; `applyMigration` runs DDL and the bump inside one `BeginTx` |
| 8 | With network unshared (`unshare -rn`, lo up) store opens/writes/serves | ✓ VERIFIED | Re-run this pass: `unshare -rn` (lo up, 127.0.0.1/8) store/xsource/fake/tui/render/cli/screen all `ok` |
| 9 | Bookmarks arrive as ordered domain posts via `Source.Saved`, same entry walk as home timelines, no network/session in unit tests | ✓ VERIFIED | `TestSavedDecodesTheBookmarksFixture` PASS by name; `opBookmarks`/`opLikes` remain the only op names in `xsource/saved.go` |
| 10 | `Bookmarks` read by operation name never query id; missing registry op is typed error | ✓ VERIFIED | `TestPinPathIsBothHalvesOfXAPI04` missing-op arm → `volatile.ErrUnknownOperation`, call count 0. No 22-char operation-id literal in non-test source |
| 11 | Wrong pin → named stale failure (never empty success); measured pin → success; no code change between | ✓ VERIFIED | Same test: wrong pin → `*xweb.APIError` with `OutcomeUnknownOperation` and an empty page + non-nil error; then the measured `queryId` → non-empty page, 1 request each |
| 12 | Likes is one operation name + one decoder; port returns page or classifier state | ✓ VERIFIED | `TestSavedDecodesTheLikesFixture`, `TestSavedPassesAFailedAnswerThrough` PASS by name; fixtures + provenance rows under `xsource/testdata/` |
| 13 | No fixture carries viewer identity/cookie/real saved body | ✓ VERIFIED | `TestSavedFixturesCarryNoCapturedBodies` PASS (its final session arm skips without a real session); audit `leak` check sweeps the fixture cookie across the named surfaces → 0 findings |
| 14 | Fake saved pages equal real parser extraction from fixture | ✓ VERIFIED | `TestFakeSavedPagesMatchTheFixture` PASS by name |
| 15 | Incremental sync stops at first page with no unknown ids; page one costs one request | ✓ VERIFIED | `TestWalkIncrementalFirstPageAddsNothing` (call count 1), `TestSyncIncrementalStopsEarly` PASS by name; `Step` ends on `pageAdded == 0 && !full` → `StopNoUnknownIDs` |
| 16 | Full re-list marks unseen `present=0`, never deletes; row count only grows | ✓ VERIFIED | `TestFullNeverDeletes`, `TestMarkAbsentNeverDeletes` PASS by name; **re-planted this pass** a second `DELETE FROM saved_items WHERE present = 0` issued after the `MarkAbsent` UPDATE → both FAIL (`sync_test.go:142: Status present/absent = 2/0, want 2/1`; `sync_test.go:596: rows before/after = 3/2, want 3/3 (never-delete)`); reverted |
| 17 | Every run recorded; failure writes its row and leaves last-success untouched | ✓ VERIFIED | `TestRecordRunWritesLastSuccessOnlyOnSuccess`, `TestWalkRecordsRunAndLastSuccess`, `TestWalkFetchFailureMidWalk` PASS |
| 18 | Failed page not written; prior pages kept; error reaches caller | ✓ VERIFIED | `TestWalkFetchFailureMidWalk`: `Step` records `StopFailure` then returns the fetch error unwrapped; PASS by name |
| 19 | Pane and verb share one `Begin`/`Step`/`Abandon`; verb prints run facts never bodies/tokens | ✓ VERIFIED | `store.NewMirror` constructed only in `cli/sync.go` and `cli/feed.go` (+ cli tests); `TestSyncFailingFetchExitsOneWithoutPostText` PASS by name; tui `TestSavedSyncOnRefresh` PASS |
| 20 | Mirror records owner and refuses wrong account; offline read still works | ✓ VERIFIED | `TestOwnerRules`, `TestOwnerRefusesRepoint`, `TestSyncOwnerRefusal` PASS by name; `cli/feed.go` maps `store.ErrWrongOwner` to `CapWrongAccount` without closing the read path, and maps **only** the store's own `FileError` to `CapMirrorUnreadable` (`feed.go:243-251`), so a refused proxy, a spent request cap, a missing session and a shape-drifted 200 each keep their own state and remedy. **Re-planted this pass (re-run, same failures)**: the predicate broadened to `err != nil` → six cli FAILs incl. `feed_test.go:498: a shape-drifted answer (…) was read as a failure of the mirror`; `*xsource.ParseError` re-admitted → the same test FAILs at `feed_test.go:498/:520/:526`; `Page`'s mark unwrapped → `store_test.go:189`; `Open`'s mark unwrapped → `feed_test.go:694`; `Status`'s mark unwrapped → full suite still green (note 3); all reverted |
| 21 | `x-terminal sync [--full] [--kind …]` headless; exit 0/1 | ✓ VERIFIED | `TestSyncIncrementalStopsEarly`, `TestSyncFullMarksAbsent`, `TestSyncFailingFetchExitsOneWithoutPostText`, `TestSyncOwnerRefusal`, `TestSyncKindLikesUnknownOpLeavesBookmarks`, `TestSyncQuietAndJSON`, `TestSyncUnknownKindIsUsage` PASS by name |
| 22 | `b`/`l` open saved frame with zero Source requests across open/scroll/page/pop | ✓ VERIFIED | `TestSavedFrameIssuesNoSourceRequest` PASS (also under `unshare -rn`). The route is `press`' `case ActionBookmarks:` / `case ActionLikes:` → `m.savedAction(action)` (`update.go:287-290`) → `pushSaved` (`saved.go:246-254`), the identical body the pre-split shared arm carried; the probe's `b`/`l`/`G`/`j`/`k`/`g`/`ctrl+d`/`q` checkpoints are byte-identical to `d686f36`'s |
| 23 | `q`/`esc` pops saved frame restoring prior frame byte-identically with zero new requests | ✓ VERIFIED | Same test: `s.frame() != before` fails; call count unchanged after pop. `q`/`esc` still route `ActionBack` → `m.back()` (`update.go:271-272`), an arm the split did not touch; the probe's pop checkpoints match `d686f36`'s |
| 24 | Every saved frame carries `synced`/`never synced`; no other frame does (corpus) | ✓ VERIFIED | `TestSyncedFieldCorpusRule` PASS by name this pass with both anti-vacuity arms. **Re-planted this pass** `synced Jan 02` over a row of the non-saved `empty_120.golden` → FAIL `corpus_test.go:2012: non-saved frame empty_120 carries the synced field`; reverted. The covered frames carry the per-grammar markers (`cached`/`never cached`) and pass the same rule |
| 25 | Three capability states print with remedies; likes unavailable leaves other lists working | ✓ VERIFIED | `TestCapabilityBlocksAt80`, `TestLikesUnavailableLeavesOthersAlone`, `TestSyncKindLikesUnknownOpLeavesBookmarks` PASS by name; covered pane/status carry the cover's own state word and remedy. **The release rule is re-pinned from both ends this pass**: `screen.Saved` releases the file state on a page that delivered (`saved.go:122-127`), releases whatever the frame wore when a sync finished with any stop but `StopFailure` (`saved.go:153-159`), leaves it after a failed action, does not clear it on `StatusRead` (`saved.go:162-167`), and never clears the account state on a page read — `TestSavedCapabilityGoesWhenTheFramesOwnActionSucceeds` (5 subtests, `saved_test.go:85/:101/:108/:128` FAIL under the four plants) plus the shell's `TestASavedFramesCapabilityGoesWhenItsOwnActionSucceeds` (PASS by name this pass). Four plants, four failures: no page release (`saved_test.go:85`, `:101`), a page clearing everything (`:108`), `StopFailure` releasing too (`:128`), `StatusRead` clearing (`tui_test.go:1576`). The covered `store cannot be read` state now stands on a reachable pair (failing read + failing map); removing the failing read degrades both frames to the empty-succeeded state and fails `cover_e2e_test.go:1454`. **Also re-planted**: `render.Plain` at `view.go`'s four saved-pane call sites → `TestTheCoveredSavedPaneNamesNothingOfTheProduct` FAIL (`the covered servicelog frame holds "doctor", a word the cover did not author`); reverted |
| 26 | Empty mirror vs empty successful read distinguishable; likes hedge is one line | ✓ VERIFIED | **Closed in the amendment round (was the blocker); its named tests re-run green this pass.** Plain half: `TestSavedEmptyStatesDiffer`, `TestLikesHedgedIsOneLine` PASS. Covered half: `SavedEmptyState`'s covered branch reads `kind` first (`saved.go:173-179`) and returns `CopyCoverEmptyHedged` — one line, no body — for a covered `l` frame, at both synced states and all four styles. Pinned three ways: the unit arm (`TestTheCoveredLikesHedgeIsOneLine`), the stored bytes of all 8 `cover_*_likes_hedged_*` frames (`TestTheCoveredLikesHedgeIsInTheCorpus`), and the shipped path through the `l` key (`savedFrameStates`). Falsified: reverting the branch fails all three (8 frames × 3 messages, 4 styles × 2 widths, 4 styles × 2 widths); see the arm list below |
| 27 | `r` on saved frame is incremental sync via same `Begin`/`Step`/`Abandon` | ✓ VERIFIED | `TestSavedSyncOnRefresh` PASS by name this pass (also in the tui package under `unshare -rn`), driving the pane's `r` through `Deps.Saved`; the route is `press`' `case ActionRefresh: return m.refresh()` (`update.go:285-286`, untouched by the split) → `refresh`'s `case frameSaved: return m.refreshSaved()` (`update.go:670-671`) → `Deps.Saved` `Begin`/`Step`/`Abandon`; the plain notice frames are byte-identical and the probe's `saved-r`/`likes-r` checkpoints (notice, rows, capability) match `d686f36`'s |
| 28 | `present=0` rows never render and are not counted in position | ✓ VERIFIED | Reader SQL is `present = 1` only (read.go:37,43); `Status` counts present/absent separately; golden `bookmarks_absent_rows_80` shows `3 no longer saved` beside position `2/5+`. **Re-planted this pass** `bandTop: sortKeyBase + sortBand` (the old rule) → `TestWalkBandsKeepSortIndexUnique` FAIL (`sync_test.go:1033: after the incremental walk: 803 rows, 800 distinct keys, want 803/803`); reverted. `TestFileWrittenUnderTheOldRuleStillReads` PASS |
| 29 | Phase strings are `copy.go` constants; `Markers.Sync`/`SyncNever` non-empty every grammar; covers leak no `x.com`/`doctor`/`saved.db`/`pins.json` | ✓ VERIFIED | `MarkerSync`/`MarkerSyncNever` at copy.go:146-147; `TestMarkerSyncNonEmptyEveryGrammar` (5 grammars) PASS by name this pass; the plain `coverGitFacts` clause plant still fails the two covered gitlog goldens. `TestCoveredFramesLeakNoRepairWords` scans **every** covered frame (≥60) for 8 repair words + the built-from accounts, with a shape-coverage arm (all six covered saved shapes), capability/failure arming and plain-prints-it arms; `TestCoveredLeakScanCatchesAPlant` plants all 9 strings into a covered **thread** frame and a covered saved row. `internal/tui/corpus_test.go`'s `TestTheCoveredTUIFramesLeakNoPlainVocabulary` does the same over its half (≥24 frames, its own plant base). **Falsified this pass**: a plain row planted into `cover_gitlog_thread_80` and into `search_people_gitlog_80` (both re-blessed) → both scans FAIL naming the frame and the string. Independently, 0/96 covered states leaked in last pass's probe and the hedged/notice frames pass the same scan now |

**Score:** 29/29 truths verified (0 present, behavior-unverified)

No truth was marked by override; no override entry exists in this report.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/domain/saved.go` | Saved kinds, ports, capability | ✓ EXISTS + SUBSTANTIVE | Mirror/SyncRun/Capability/SavedItem, MirrorFacts |
| `internal/store/schema.go` | Open/Close, WAL read-back, three tables | ✓ EXISTS + SUBSTANTIVE | writer pool + `query_only` reader; `declaredTables`/`declaredMetaKeys` |
| `internal/store/migrate.go` | Ordered migrations, same-tx bump | ✓ EXISTS + SUBSTANTIVE | `ErrSchemaNewer`; no `user_version` |
| `internal/store/read.go` | Page/Status present=1; `ReadFacts` | ✓ EXISTS + SUBSTANTIVE | keyset pages; present/absent counts; `ReadFacts` mode=ro+query_only |
| `internal/store/sync.go` | Upsert/MarkAbsent/walk; per-run `sort_index` band | ✓ EXISTS + SUBSTANTIVE | UPDATE present=0 only; Begin/Step/Abandon; pageCap 40; sortFloor + sortBand=800 |
| `internal/store/boundary_test.go` | AST + data-dir proofs | ✓ EXISTS + SUBSTANTIVE | plant + two positive controls + anti-vacuity |
| `internal/store/{schema,migrate,sync,store}_test.go` | Schema/migrate/sync proofs | ✓ EXISTS + SUBSTANTIVE | allowlist fail arms; never-delete; banding + old-file-read arms |
| `go.mod` sqlite pin | `modernc.org/sqlite v1.60.1` | ✓ EXISTS + SUBSTANTIVE | `CGO_ENABLED=0 go build ./...` ok |
| `internal/xsource/saved.go` | Saved adapter | ✓ EXISTS + SUBSTANTIVE | `opBookmarks`/`opLikes` names only |
| `internal/xsource/testdata/fixture-{bookmarks,likes}.json` | Redacted fixtures | ✓ EXISTS + SUBSTANTIVE | provenance rows; redaction tests |
| `internal/domain/source.go` Saved | Port method | ✓ EXISTS + SUBSTANTIVE | method present |
| `internal/fake/saved.go` + parity | Fake + pages-saved | ✓ EXISTS + SUBSTANTIVE | call counter; parity test |
| `internal/store` Mirror walk + `cli/sync.go` | Verb + walk | ✓ EXISTS + SUBSTANTIVE | command table row; flags |
| `internal/config` SavedDBPath | Path beside pins | ✓ EXISTS + SUBSTANTIVE | used by sync + feed + doctor |
| `internal/screen/saved.go` | Saved frame state | ✓ EXISTS + SUBSTANTIVE | reducers + NoticeFor |
| `internal/fake/mirror.go` | Scripted Mirror | ✓ EXISTS + SUBSTANTIVE | counters |
| `internal/render/saved.go` + status/copy/cover/grammar/profile + goldens | Synced field, capability/empty/failure blocks in the pane's own grammar | ✓ EXISTS + SUBSTANTIVE | the three builders take a `Grammar`; the covered empty branch now *also* takes `kind`, so the likes hedge is one line under every grammar; covered branches exist for every state |
| `internal/tui/{model,update,view,saved}.go` + `cli/feed.go` | Keys, dispatch, openSaved, `g` | ✓ EXISTS + SUBSTANTIVE | composition root only opens store; `ActionFirst` restores the opening state; `profileAccount` names one account |
| `internal/tui/cover_e2e_test.go` | The saved surface's own covered table + B1's shipped-path arm + the kinds join | ✓ EXISTS + SUBSTANTIVE | `savedFrameStates` (8 states incl. the likes-empty one), `savedLeakWords` (9), plain arm per state, `frameKindChecks` joined to `frameKinds()` |
| `internal/render/golden_cover_corpus_test.go` | Covered cross-product incl. the saved non-row states and the four notices | ✓ EXISTS + SUBSTANTIVE | 13 saved shapes × 4 styles × 2 widths (notices at 120), incl. `likes_hedged` |
| `internal/audit/source.go` | Parse-honesty check | ✓ EXISTS + SUBSTANTIVE | first check in the registry; names a non-parsing file, its line and the parser's message |

**Artifacts:** all listed plan artifacts verified

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| Write API | `domain.SavedItem` only | Upsert signature | ✓ WIRED | No timeline type fits |
| Open | Migrate + WAL read-back | `Open` path | ✓ WIRED | refuses non-wal with a named error |
| Timeline packages | must not import store | AST boundary | ✓ WIRED | plant fails; `go list` importers = `internal/cli` (+ store tests) |
| Adapter Saved | volatile registry | op name lookup | ✓ WIRED | pin test exercises `Lookup`; missing op = typed error |
| Fixture body | fake pages | parity test | ✓ WIRED | `TestFakeSavedPagesMatchTheFixture` |
| Sync walk | SavedFetcher port | no xsource import | ✓ WIRED | `internal/store` declares the port; adapter passed by cli |
| MarkAbsent | run start instant | full finish path | ✓ WIRED | `TestFullNeverDeletes`; `finish` gates on `full && (EndOfList\|PageCap)` |
| Run's keys | file's existing keys | `sortFloor` → `bandTop` | ✓ WIRED | `Begin` reads the floor and fixes the band; `Step` writes from it |
| Verb + pane `r` | same Mirror Begin/Step | cli + tui | ✓ WIRED | both construct `store.NewMirror` / `Deps.Saved` |
| Saved frame rows | Mirror.Page | pushSaved | ✓ WIRED | Source call count unchanged |
| **Saved pane's empty branch** | **the kind, then the pane's own grammar** | `view.go:421` → `SavedEmptyState` → `saved.go:175-179` | ✓ WIRED | the blocker's link: `kind == domain.Like` is read before `cover`, so a covered `l` frame prints the cover's hedge; reverting it fails the three arms above |
| Saved pane's other branches | the pane's own grammar | `view.go:410/415` → the two builders, `status.go` → the cover's state word/remedy | ✓ WIRED | plant fails `TestTheCoveredSavedPaneNamesNothingOfTheProduct` |
| Covered facts row | each style's own `AccountFacts` + its own `MonthYear` | `cover.go` → `profile.go:129-132` | ✓ WIRED | the fourth clause now renders (`Since: Mar 2011` etc.); clearing `coverGitStamp.MonthYear` fails the two covered gitlog profile goldens |
| `g` / `ActionFirst` | the frame's opening state | `update.go` `move` → `model.go` `skip` | ✓ WIRED | plant fails `TestGoRestoresTheProfileHeader` (10 subtests) |
| Covered profile field | one account, one name | `view.go:447-455` `profileAccount` | ✓ WIRED | plant fails `TestTheCoveredProfileNamesOneAccount` in all 4 styles |
| `frameKinds()` | the covered check table | `cover_e2e_test.go:729-743` | ✓ WIRED | a sixth kind added to the enumerator fails the join both ways |

| `b`/`l` key | `pushSaved` | `press` `case ActionBookmarks`/`ActionLikes` → `savedAction` | ✓ WIRED | one arm per action now; the probe's `b`/`l` open/pop checkpoints are byte-identical to `d686f36`'s `press` (truth 22) |
| `r` on a saved frame | `store.NewMirror`'s `Begin`/`Step`/`Abandon` | `press` `case ActionRefresh` → `refresh` → `refreshSaved` | ✓ WIRED | `update.go:285-286` → `:670-671` → `saved.go:135-160`; `TestSavedSyncOnRefresh` PASS, probe `saved-r`/`likes-r` match (truth 27) |
| `handledActions` | the keymap table | `dispatch_test.go`'s two-direction join + AST sweep | ✓ WIRED | 30 arms = 30 rows; a 31st row, an orphan arm, and a bogus case label each fail it (`dispatch_test.go:63/:72`, compiler `undefined:`); one real arm removed is caught by the behavioural tests instead (Tech Debt 5) |
| `savedMirrorError` | the store's `FileError` alone | `errors.As` at `feed.go:129` | ✓ WIRED | `Open`/`Page`/`Status` wrap; nothing else wears the mark. Unwrapping `Open` fails the cli predicate test (`feed_test.go:694`), unwrapping `Page` fails the store's mark test (`store_test.go:189`); broadening the predicate fails six cli tests |
| a frame's capability | the frame's own successful action | `saved.go:122-127` (a delivered page), `:153-159` (a finished sync), `:162-167` (status) | ✓ WIRED | four plants fail `TestSavedCapabilityGoesWhenTheFramesOwnActionSucceeds` and `TestASavedFramesCapabilityGoesWhenItsOwnActionSucceeds` by name (truth 25) |
| the classification table | the checked-in registry | `checkOpNames` (`readonly.go:171-200`) + `audit_test.go`'s ten-name pin | ✓ WIRED | the stale name back in the table → `opnames` reports it and the audit exits 1 |

**Wiring:** all plan key_links verified

## Requirements Coverage

| Requirement | Status | Blocking Issue |
|-------------|--------|----------------|
| SAVE-01: read bookmarks from session | ✓ SATISFIED | - |
| SAVE-02: likes list / honest degrade | ✓ SATISFIED | - (the covered likes-empty hedge is restored and pinned three ways) |
| SAVE-03: mirror opens with network down | ✓ SATISFIED | - |
| SAVE-04: incremental stop; full marks present=0 | ✓ SATISFIED | - |
| SAVE-05: write API accepts only saved items | ✓ SATISFIED | - |
| SEC-02: timeline structurally unrepresentable on disk | ✓ SATISFIED | - |
| XAPI-04: manual pin of Bookmarks/Likes ops | ✓ SATISFIED | - |

**Coverage:** 7/7 requirements satisfied

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/render/copy.go | 55-61 | The four mirror-read notices are cover-respelled | ℹ️ Info | Deliberate and documented; deviates from `06-UI-SPEC.md:605` ("news is not cover-respelled"), closes a `DSGT-09` leak on a reachable path (`r` is not cover-breaking); plain notices untouched — see Accepted Tech Debt |
| internal/render/copy.go | 296-304 | The covered hedge's nouns ("records", "view") are the shared covered heading's, not each cover's own word for the list (`queue=`/`ref=starred`/`stream=fav`) | ℹ️ Info | One shared covered sentence, the same shape as the `CopyCoverEmptyHeading`/`CopyCoverEmptyBody` pair it extends; not introduced by any round beyond extending it — see Accepted Tech Debt |
| internal/tui/cover_e2e_test.go | 1362 | The likes-empty state's `covered` list asserts `CopyCoverEmptyHeading`, which the hedge already contains as a prefix | ℹ️ Info | The assertion is entailed rather than independent; the independent arms (hedge present, three forbidden strings, one row) are in the same row and in the corpus — see Accepted Tech Debt |

**Anti-patterns:** 3 found (0 blockers, 0 warnings, 3 info). This pass adds no source
anti-pattern: the only new finding is a documentation overreach in a comment, recorded under
Accepted Tech Debt 4-5 rather than as an anti-pattern.

The three earlier warnings stay closed and are recorded in Accepted Tech Debt rather than as
findings: `frameKinds()` has its join test, every cover stamp sets `MonthYear`, and the covered
likes-empty frame now has three independent arms.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/store/*_test.go` | SAVE-04/05 SEC-02 | yes | no | no | Behavioral + plant fail arms | OK |
| `internal/xsource/saved_test.go` | SAVE-01 XAPI-04 | yes | no | no | Value/behavioral | OK |
| `internal/tui/tui_test.go` saved | SAVE-02/03 | yes | no | no | Behavioral call counters | OK |
| `internal/render/corpus_test.go` + `saved_test.go` | SAVE-02/03 | yes | no | no | Corpus presence/absence + leak scan with shape/arm/plant legs + the covered hedge's unit, corpus and one-line arms | OK |
| `internal/tui/cover_e2e_test.go` (`TestTheCoveredSavedPaneNamesNothingOfTheProduct`, `TestEveryFrameKindIsCheckedUnderACover`, `TestTheCoveredProfileNamesOneAccount`) | SAVE-02/03 DSGT-09 D5 | yes | no | no | Shipped-path scan, plain arm per state, cover's own words asserted; all 8 saved states reached through shipped keys (`b`/`l`) | OK |
| `internal/tui/corpus_test.go` (`TestTheCoveredTUIFramesLeakNoPlainVocabulary` + plant) | SAVE-02/03 DSGT-09 | yes | no | no | Covered-frame scan over this corpus's half + its own plant proof | OK |
| `internal/cli/sync_test.go` | SAVE-04 | yes | no | no | Behavioral | OK |
| `internal/cli/feed_test.go` (`TestSavedMirrorErrorNamesOnlyTheFile`, `TestAShapeDriftedAnswerIsNotAFactAboutTheFile`) | SAVE-02/03 | yes | no | no | Predicate over production errors + the drifted answer driven through a real store and the shipped mapping | OK |
| `internal/screen/saved_test.go` (`TestSavedCapabilityGoesWhenTheFramesOwnActionSucceeds`) | SAVE-02/03 | yes | no | no | State-transition subtests, one per release rule, each a plant target | OK |
| `internal/audit/audit_test.go` (`TestCheckSetIsPinned`) | (cross-check) | yes | no | no | The ten names and their order, by hand | OK |
| `internal/fake/parity_test.go` | SAVE-01 | yes | no | no | Cross-implementation parity | OK |
| `internal/tui/dispatch_test.go` (`TestEveryBoundActionIsHandledAndEveryHandledActionIsBound`, `TestTheShellDeclaresOnlyActionsTheKeymapDeclares`) | (cross-check, this round) | yes | no | no | Two-direction join over the keymap table and the `handledActions` enumeration, plus an AST sweep of the action declarations in `keys.go` against the shell's references and case labels; anti-vacuity floors; both plants fail it | OK |
| `internal/audit/source_test`/`audit_test.go` | (cross-check) | yes | no | no | Registry + finding shape | OK |

**Disabled tests on requirements:** 0 blockers (the `X_TERMINAL_CAPTURE=1` capture tests
skip without the gate by design; `TestNoFixtureCarriesViewerIdentity` skips only its last
arm, which needs a real session)
**Circular patterns detected:** 0
**Insufficient assertions:** 0 — the covered likes-empty frame was this report's single
entry last pass; the state now has three arms and its own stored frames.

Planted-violation arms exercised in this refresh (all in `/tmp` copies; the repository was left
byte-identical — a pristine copy was kept and compared after every batch). Items marked *(re-run)*
repeat an arm the previous report recorded and reproduced its failure text; items 20-23 are this
round's, for the dispatcher guard, and item 0 is the behaviour-preservation differential:

0. **The dispatcher differential** — a probe compiled into `internal/tui` driving 86 checkpoints
   (every bound key, two unbound keys, the prompt's keys, the panic gate's two keys and its no-op
   set, `b`/`l` open+paging+`r` sync+pop, the covered gate, the unreadable-mirror pair), dumping each
   step's frame kind, depth, display state, help/prompt flags, source call count, persist log and
   rendered frame bytes. Run against `df80562`'s `press` and, with `dispatch_test.go` removed,
   against `d686f36`'s `press`: `diff` over 182,165 bytes / 86 blocks is **empty**.
1. **The predicate broadened** (`savedMirrorError` → `return err != nil`; *(re-run)* of the recorded
   `return true` arm) → six cli FAILs: `TestAShapeDriftedAnswerIsNotAFactAboutTheFile`
   (`feed_test.go:498: a shape-drifted answer (xsource: tweet-1: content.itemContent.tweet_results
   …) was read as a failure of the mirror`; `:520` and `:526` show the status line
   `mirror unreadable  move saved.db aside, then sync`), the three
   `TestASavedFrameNamesTheRequestPathsOwnStates` rows (`a spent request cap`, `a missing session`,
   `a refused proxy`) and `TestSavedMirrorErrorNamesOnlyTheFile`.
2. **The shape-drifted 200 claims the mirror** — `*xsource.ParseError` re-admitted into the
   predicate → `TestAShapeDriftedAnswerIsNotAFactAboutTheFile` FAIL (`:498` `was read as a failure
   of the mirror`, `:520` the unreadable status line, `:526` the pane). This is the case a blacklist
   of the request path's four named shapes missed.
3. **`Page`'s mark removed** (`read.go:55`) → `TestTheFilesOwnReadFailuresCarryTheMark` FAIL
   (`store_test.go:189: the read failure store: page: sql: database is closed (*fmt.wrapError) does
   not carry the file's mark`); the cli package stays green.
4. **`Open`'s mark removed** (`schema.go:43`) → `TestSavedMirrorErrorNamesOnlyTheFile` FAIL
   (`feed_test.go:694: the store's own error (store: open …/saved.db: store: read schema version:
   file is not a database (26)) is the mirror's and the predicate does not see it`).
5. **`Status`'s mark removed** (`read.go:132`) → **no test fails**: `go test ./... -count=1` → 22
   packages `ok`. Recorded as a note (see Accepted Tech Debt 3).
6. **No page release** — the `MirrorPage` branch of `screen.Saved.Update` left as `s.list = next`
   only → `TestSavedCapabilityGoesWhenTheFramesOwnActionSucceeds/a_read_that_delivered_its_page_clears_the_file_state`
   FAIL (`saved_test.go:85: capability after a page arrived = mirror unreadable, want CapNone…`) and
   `/a_capability_the_frame_was_handed_is_cleared_the_same_way` FAIL (`saved_test.go:101`).
7. **A page read clears every state** (`s.capability = domain.CapNone` in the `MirrorPage` branch) →
   `/a_page_read_does_not_clear_the_account_state` FAIL (`saved_test.go:108: capability after a page
   arrived = , want CapWrongAccount…`).
8. **`SyncFinished` releases a failed run too** (the `Stop != StopFailure` guard removed) →
   `/a_run_recorded_as_a_failure_clears_nothing` FAIL (`saved_test.go:128: capability after a walk
   that recorded a failure = , want it to stay CapMirrorUnreadable`).
9. **`StatusRead` clears the capability** (the `e.Cap != CapNone` guard removed from the
   `StatusRead` case only) → the shell's `TestASavedFramesCapabilityGoesWhenItsOwnActionSucceeds`
   FAIL (`tui_test.go:1576: capability after a failed read = , want CapMirrorUnreadable`) and
   `TestSavedFrameIssuesNoSourceRequest` FAIL. The screen package alone stays green: this rule is
   pinned where it can be observed, in the shell.
10. **`FileError` is not exported** (the type renamed in every store file and in `feed.go`, the
    allowlist's own entry put back) → `TestExportedSurfaceAllowlist` FAIL
    (`schema_test.go:76: exported surface missing "FileError"`).
11. **The covered `store cannot be read` state without its failing read** (the `FailPage` line of
    `savedFrameStates`' capability arm replaced by a discarded `errors.New`) → 
    `TestTheCoveredSavedPaneNamesNothingOfTheProduct/the_store_cannot_be_read/{80,120}` FAIL
    (`cover_e2e_test.go:1454: the plain frame of "the store cannot be read" does not carry "move
    saved.db aside": the covered absence below would be vacuous`, the pane showing
    `no posts in bookmarks` / `this is an empty read that succeeded`).
12. **A stale operation name** (`"ListLatestTweetsTimeline": "read"` back in `opClass`) → the audit
    reports `internal/audit/readonly.go: the table classifies "ListLatestTweetsTimeline" as a read
    and the registry does not hold it: an entry for an operation this build cannot resolve`,
    **10 checks, 1 findings, 893 files inspected**, exit 1; removed, back to 10 checks / 0 findings
    / 893 files / exit 0.
13. **The timeline's adapter imports `internal/store`** (`_ "…/internal/store"` into
    `internal/xsource/home.go`; *(re-run)*) → `TestBoundaryImportScan` FAIL
    (`boundary_test.go:45: internal/xsource/home.go imports github.com/LZD121/x-terminal/internal/store`).
14. **The plain grammar at the saved pane's call sites** (`render.Plain` for `m.grammar()` in
    `view.go`'s `savedPane`; *(re-run)*) → `TestTheCoveredSavedPaneNamesNothingOfTheProduct` FAIL
    (`the covered servicelog frame holds "doctor", a word the cover did not author`, over the
    `bookmarks unavailable` frame).
15. **`ActionFirst` disabled** (its `case` removed from `update.go`; *(re-run)*) →
    `TestGoRestoresTheProfileHeader` FAIL in all 10 subtests (`profile_test.go:334: g left the
    frame moved: the pane is still the anchor's and not the frame's own first row`).
16. **A plain clause added to `coverGitFacts`** (`"published {joined}"`; *(re-run)*) → tui
    `TestGoldenFrames/profile_gitlog_80` FAIL against the stored bytes.
17. **`MarkAbsent` deletes what it marks** (a `DELETE FROM saved_items WHERE present = 0` after the
    `UPDATE`; *(re-run)*) → `TestMarkAbsentNeverDeletes` FAIL (`sync_test.go:142: Status
    present/absent = 2/0, want 2/1`) and `TestFullNeverDeletes` FAIL (`sync_test.go:596: rows
    before/after = 3/2, want 3/3 (never-delete)`). Placed *before* the `UPDATE` the same statement
    deletes nothing and both tests pass — the arm only bites where the rows exist.
18. **The old banding rule** (`bandTop: sortKeyBase + sortBand` instead of `floor + sortBand`;
    *(re-run)*) → `TestWalkBandsKeepSortIndexUnique` FAIL (`sync_test.go:1033: after the incremental
    walk: 803 rows, 800 distinct keys, want 803/803`).
19. **A non-saved frame carries the `synced` field** (`synced Jan 02`) overlaid on the first row of
    `empty_120.golden`; *(re-run)*) → `TestSyncedFieldCorpusRule` FAIL
    (`corpus_test.go:2012: non-saved frame empty_120 carries the synced field`).

20. **A 31st keymap row with a fresh action** (`ActionFrobnicate`) → `dispatch_test.go:63: the keymap
   binds "frobnicate" (f9) and press has no arm for it: the key is a documented no-op`, `:79: the
   table holds 31 rows and handledActions names 30 actions`, `:128` and `:159` (sweep).
21. **A handler with no row** (`ActionOrphan` in `press` + `handledActions`) → `:72: press handles
   "orphan" and no row of the keymap binds it: the arm is unreachable`, `:79`, `:125`.
22. **A bogus case label** (`case keys.ActionThisPackageDeclaresNoSuchAction:`) → the compiler refuses:
   `internal/tui/update.go:327:12: undefined: keys.ActionThisPackageDeclaresNoSuchAction`.
23. **One real arm removed** (`case keys.ActionLikes` deleted from `press`) → the guard stays green
    (the sweep is an AST read of case-clause labels; `ActionLikes` is still a `case` in `saved.go`),
    and the behavioural tests catch it: `TestSavedFrameIssuesNoSourceRequest`,
    `TestLikesUnavailableLeavesOthersAlone`, `TestTheCoveredSavedPaneNamesNothingOfTheProduct` FAIL.

### Audit cross-check

`go run ./cmd/x-terminal audit` → **10 checks, 0 findings, 893 files inspected** (exit 0; re-run this pass).
The checks that overlap this phase's promises agree with it:

- **source** — every Go file the audit reads parses, and one that does not is a finding naming the
  file and the line rather than a silent skip. This is what makes the checks below trustworthy:
  they used to skip an unparseable file in place.
- **opnames** — the classification table is a subset of the checked-in registry,
  asserted rather than left to review because the two files are edited independently; re-proven this
  pass by plant 12 *(re-run)*, which puts the stale `ListLatestTweetsTimeline` back: `internal/audit/readonly.go:
  the table classifies "ListLatestTweetsTimeline" as a read and the registry does not hold it: an entry for
  an operation this build cannot resolve`, **10 checks, 1 finding, 893 files**, exit 1 — and back to 0
  findings on its removal.
- **persistence** — every write resolves to one of the six named destinations; the only
  `sql.Open` call sites remain `store/schema.go:50,56` and `store/read.go:198`, all
  classified `saved` (truths 3, 16).
- **boundaries** — `internal/tui` is checked against a forbidden-import list containing
  `database/sql` and `internal/store`, with a positive control; same promise as truth 4.
- **readonly** — `opClass` maps `Bookmarks`/`Likes` to `read` and lists
  `CreateBookmark`/`DeleteBookmark`/`FavoriteTweet`/… as `write` (truths 9–11).
- **leak** — sweeps the fixture cookie across the named surfaces including the store
  sources (truth 13), and now also asserts the covered notices' words on screen
  (`TestTheCoveredNoticesAreOnScreen`).

No check contradicts a truth above, and the covered likes-empty frame is now inside the
covered-frame scan rather than outside every corpus.

## Human Verification Required

None. Every claim in this pass was checked programmatically: the guard's join and its sweeps, the
dispatcher differential (a compiled probe, not a reading of the diff), and the re-run plant arms.
Two comment claims in `update.go`/`dispatch_test.go` overstate what the guard pins (one action per
arm; the sweep rejecting a merged arm) — they are documentation, recorded as notes, and no must-have
depends on them. The ordinary residual checks a human may still run are listed under Could Not
Verify.

## Accepted Tech Debt and Notes (carried into the archive)

Recorded so the milestone archive carries them; none is a must-have failure.

1. **The four mirror-read notices are cover-respelled** (`internal/render/copy.go:55-61`,
   ℹ️ Info). They deviate from `06-UI-SPEC.md:605` ("news is not cover-respelled"). The
   deviation is deliberate and documented in the file: `r` is not cover-breaking, so a
   plain notice on a covered saved frame would leak the product's word for the
   operation it names; the plain notices are untouched and their frames are
   byte-identical. Accepted as-is.
2. **The covered hedge's nouns are the shared covered heading's** — "records" and
   "view" (`copy.go:296,304`) — while the four covers name their own saved list
   `queue=starred`, `ref=starred`, `view=starred` and `stream=fav` (`copy.go:157-163`).
   Only the `internal` cover's word matches the hedge's "view". This is the same
   sharing the covered heading/body pair already had; a per-style hedge row would be a
   larger change than the defect needed. Not a gap: no must-have names the word, and
   the cover's own heading already reads "no records in this view".
3. **`internal/store`'s `Status` mark is not pinned by a test** (`read.go:132`, ℹ️ Info). Removing
   `Status`'s `&FileError{Err: err}` wrap leaves `go test ./... -count=1` green across all 22
   packages: `TestTheFilesOwnReadFailuresCarryTheMark` exercises `Page` only (against a closed
   store) and the cli predicate's positive arm uses `Open`'s error (`storeOpenError`), while the
   shell's unreadable-mirror arms go through the `unreadableMirror` stand-in, which exists only for
   the open failure. The behaviour is present and correct — a `Status` read of a broken or closed
   file does reach `CapMirrorUnreadable` through `savedError` — but nothing would notice its
   removal. Small, honest coverage debt; closing it means a store arm that calls `Status` on a
   closed store and a cli row that drives a real `Status` failure through the shipped mapping.
4. **A comment above `coverBreaking` overstates what the sweep needs and what it is**
   (`internal/tui/update.go:331-337`, ℹ️ Info). It says "one action per arm is what keeps that sweep
   a line match rather than a parser". The sweep is an AST read of case clauses
   (`dispatch_test.go:shellActionCases`), not a line match, and it accepts a merged arm: merging the
   two saved arms back into `case keys.ActionBookmarks, keys.ActionLikes:` leaves both guard tests
   `ok`. The comment's other half is true and was planted — a bogus case label
   (`keys.ActionThisPackageDeclaresNoSuchAction`) is refused by the compiler
   (`undefined: …`). Prose, not behaviour: no must-have reads a comment, and nothing else in the
   tree depends on the claim. Honest one-line fix: say the sweep reads case clauses and that a bogus
   label is the compiler's.
5. **`handledActions`' doc claims more than the sweep enforces** (`update.go:1286-1295`, ℹ️ Info).
   It says a new action lands "in the table and in a handler arm at once or not at all". The sweep's
   case-label leg only requires *a* `case` for the action somewhere in the package, so deleting
   `press`' live `case keys.ActionLikes:` leaves the guard green — `internal/tui/saved.go:250`'s
   routed switch still names it. The behaviour is caught, but by the feature tests, not by the
   guard: `TestSavedFrameIssuesNoSourceRequest`, `TestLikesUnavailableLeavesOthersAlone` and
   `TestTheCoveredSavedPaneNamesNothingOfTheProduct` all FAIL under that plant. The test file's own
   `shellActionCases` comment ("the dispatcher's arms and every routed switch's") states the real
   reach; the update.go doc should say the same, or the leg should be restricted to `press`.
6. **The likes-empty state's `covered` assertion is partly entailed**
   (`internal/tui/cover_e2e_test.go:1362`): `CopyCoverEmptyHedged` contains
   `CopyCoverEmptyHeading` as a prefix, so asserting the heading adds nothing to that
   row. The independent arms are the hedge itself, the forbidden trio and the
   one-row/one-block checks in the same row and in the corpus. Harmless redundancy.

Closed in the guard commit (kept here so the archive shows what moved): the keymap and the
dispatcher are now joined in both directions by `internal/tui/dispatch_test.go`, with the
enumeration (`handledActions`) and its exported read (`HandledActions()`) beside the switch, and a
third sweep that reads the action declarations out of `keys.go` — the seam AUTH-03 shipped with, now
guarded. Still open from earlier rounds and recorded above: the `Status` mark's missing test arm
(3), the partly entailed likes-empty assertion (6), and the two comment overstatements this pass
found (4, 5).

Closed in earlier rounds (kept here so the archive shows what moved): the covered likes-empty
hedge (the previous blocker — now three arms and 8 stored frames); `frameKinds()`'s
dead-code warning (now joined by `TestEveryFrameKindIsCheckedUnderACover`); the
unreachable fourth facts clause (every cover stamp now sets `MonthYear`, so the covered
profile frames show `Since: Mar 2011` / `[since 2011-03]` / `since Mar 2011` /
`since=2011-03`); and the `capture_tmux_test.go` socket left behind per run (now
unlinked after the server is confirmed stopped).

## Gaps Summary

**No gaps found.** Phase goal achieved. All 29 must-have truths verified on the frozen
tree; the dispatcher-guard commit is behaviour-preserving under an 86-checkpoint differential and
the guard fails in both directions on its own plants; 1573 tests pass in 22 packages; the audit is
clean at 10 checks / 0 findings over 893 files. The mirror truths were re-derived and their plants
re-run (the `Status` mark's missing arm and the guard's two comment overstatements are recorded as
notes 3-5, not gaps).

## Could Not Verify

- Interactive live TUI with the real network down (no proxy) while reading a production
  `saved.db` — not run; equivalent proven via fake-mirror call counters plus the packed
  `unshare -rn` runs.
- Live capture refresh (`X_TERMINAL_CAPTURE=1`) — intentionally skipped without the
  gate; fixtures + provenance rows are already in the tree.
- The stored frames were read as data, not painted against a live terminal; their bytes
  are the contract the corpus rules assert. The tmux leg that verifies the painting
  step is itself gated on `tmux` being on PATH (it skips, loudly, when it is not).
- The differential probe and the plant batches ran in a `/tmp` copy of the tree with
  `XDG_CONFIG_HOME`/`XDG_DATA_HOME` pointed at a temp dir; the operator's real session file and
  `saved.db` were never opened (the real session file is gone and was not touched or restored).
- The dispatcher differential is a compiled probe over 86 observable checkpoints, not a proof over
  all inputs: the argument is also structural — the four multi-label arms were re-expressed as one
  arm per action with the identical body and receiver (`m.savedAction(action)`, `m.numberedCover(action)`,
  `m.move(action)`), which Go evaluates identically — and the probe covers every bound key, both
  unbound keys the tests use, both shadowed states and every routed push/pop.
- A human would need to run `X_TERMINAL_CAPTURE=1 go test ./internal/xsource/ -run
  TestCapture -count=1` (proxy up + real session) to refresh fixtures, and an end-to-end
  `feed` against a real synced mirror with the network physically down.

## Verification Metadata

**Verification approach:** Goal-backward from ROADMAP SC + four PLAN `must_haves`;
re-verification mode — the truths the guard commit touches (22, 23, 27) plus the mirror truths
re-run (3, 4, 16, 20, 24, 25, 28, 29) re-derived from the tree and falsified, the other 21
regression-checked (named tests, the full suite, the 86-checkpoint differential, the offline runs)
**Must-haves source:** PLAN.md frontmatter (06-01..06-04) merged with ROADMAP success criteria
**Automated checks:** `go build ./...` ok; `go vet ./...` ok; `gofmt -l .` empty;
`go test ./... -count=1 -v` → 22 packages (the 19 with tests all ok), **1573 `--- PASS:` lines
including subtests, 0 `--- FAIL:`** (the last pass's 1571 plus the guard's two tests);
`unshare -rn` with lo up: store/xsource/fake/tui/render/cli/screen all ok; `go run ./cmd/x-terminal
audit` → 10 checks, 0 findings, 893 files, exit 0; `go list ./...` → 22 packages
**Independent controls:** no golden file differs in this round (a pristine copy was kept and
compared after every plant batch, including both `testdata` trees); the dispatcher differential
reproduces the pre-refactor `press` byte for byte over 86 checkpoints; the audit's `opnames` check
turns red on a single re-added stale name and green again on its removal, with the file count
unchanged at 893
**Human checks required:** 0
**Falsification attempts:** twenty-three planted arms this round, all in a `/tmp` copy — the
nineteen the previous report recorded (re-run and re-failed with their recorded text) plus four new
for the guard — and the differential (arm 0), which came back identical. New: the 31st keymap row
with a fresh action (guard FAILs both directions), a handler with no row (FAILs), a bogus case label
(the compiler refuses), and one live `savedAction` arm removed from `press` (the guard stays green —
the behavioural tests catch it; note 5). Only one arm fails to fail as a defect: unwrapping the
store's `Status` mark, which nothing catches (note 3). The repository was left byte-identical (a
pristine copy compared after every batch; no golden differs)

---

*Verified: 2026-10-15T07:24:00Z*
*Verifier: Claude (subagent)*
