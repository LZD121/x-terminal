---
phase: 02-following-timeline-in-the-pane
verified: 2026-10-15T07:45:00Z
status: passed
score: 23/24 must-haves verified
covered_files:
  - .planning/phases/02-following-timeline-in-the-pane/02-01-PLAN.md
  - .planning/phases/02-following-timeline-in-the-pane/02-01-SUMMARY.md
  - .planning/phases/02-following-timeline-in-the-pane/02-02-PLAN.md
  - .planning/phases/02-following-timeline-in-the-pane/02-02-SUMMARY.md
  - .planning/phases/02-following-timeline-in-the-pane/02-03-PLAN.md
  - .planning/phases/02-following-timeline-in-the-pane/02-03-SUMMARY.md
  - .planning/phases/02-following-timeline-in-the-pane/02-04-PLAN.md
  - .planning/phases/02-following-timeline-in-the-pane/02-04-SUMMARY.md
  - .planning/phases/02-following-timeline-in-the-pane/02-05-PLAN.md
  - .planning/phases/02-following-timeline-in-the-pane/02-05-SUMMARY.md
  - .planning/phases/02-following-timeline-in-the-pane/02-CONTEXT.md
  - .planning/phases/02-following-timeline-in-the-pane/02-UI-SPEC.md
  - go.mod
  - go.sum
  - internal/audit/audit.go
  - internal/audit/guards.go
  - internal/audit/leak.go
  - internal/audit/readonly.go
  - internal/audit/source.go
  - internal/cli/cli.go
  - internal/cli/cli_test.go
  - internal/cli/doctor.go
  - internal/cli/doctor_test.go
  - internal/cli/failure.go
  - internal/cli/failure_test.go
  - internal/cli/feed.go
  - internal/cli/feed_test.go
  - internal/cli/keys.go
  - internal/cli/keys_test.go
  - internal/cli/notice.go
  - internal/cli/notice_test.go
  - internal/cli/session.go
  - internal/cli/session_test.go
  - internal/cli/whoami.go
  - internal/cli/whoami_test.go
  - internal/domain/domain.go
  - internal/domain/domain_test.go
  - internal/domain/failure.go
  - internal/domain/page.go
  - internal/domain/source.go
  - internal/fake/fake.go
  - internal/fake/parity_test.go
  - internal/fake/testdata/pages.json
  - internal/keys/keys.go
  - internal/keys/keys_test.go
  - internal/render/alias.go
  - internal/render/block.go
  - internal/render/block_test.go
  - internal/render/capture_panic_test.go
  - internal/render/capture_pty_test.go
  - internal/render/capture_tmux_test.go
  - internal/render/copy.go
  - internal/render/corpus_test.go
  - internal/render/cover.go
  - internal/render/frame.go
  - internal/render/golden_cover_corpus_test.go
  - internal/render/golden_test.go
  - internal/render/grammar.go
  - internal/render/help.go
  - internal/render/help_test.go
  - internal/render/link.go
  - internal/render/media.go
  - internal/render/palette.go
  - internal/render/palette_test.go
  - internal/render/panic.go
  - internal/render/profile.go
  - internal/render/prompt.go
  - internal/render/saved.go
  - internal/render/saved_test.go
  - internal/render/state_test.go
  - internal/render/status.go
  - internal/render/status_test.go
  - internal/render/testdata/TestGoldenFrames/color_16.golden
  - internal/render/testdata/TestGoldenFrames/color_16_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_256.golden
  - internal/render/testdata/TestGoldenFrames/color_256_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_mono.golden
  - internal/render/testdata/TestGoldenFrames/color_mono_tmux.golden
  - internal/render/testdata/TestGoldenFrames/color_truecolor.golden
  - internal/render/testdata/TestGoldenFrames/color_truecolor_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/empty_120.golden
  - internal/render/testdata/TestGoldenFrames/empty_80.golden
  - internal/render/testdata/TestGoldenFrames/expired_120.golden
  - internal/render/testdata/TestGoldenFrames/expired_80.golden
  - internal/render/testdata/TestGoldenFrames/help_120.golden
  - internal/render/testdata/TestGoldenFrames/help_80.golden
  - internal/render/testdata/TestGoldenFrames/likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/loading_80.golden
  - internal/render/testdata/TestGoldenFrames/longdetail_80.golden
  - internal/render/testdata/TestGoldenFrames/narrow_79.golden
  - internal/render/testdata/TestGoldenFrames/offline_120.golden
  - internal/render/testdata/TestGoldenFrames/offline_80.golden
  - internal/render/testdata/TestGoldenFrames/schemachange_120.golden
  - internal/render/testdata/TestGoldenFrames/schemachange_80.golden
  - internal/render/testdata/TestGoldenFrames/stalled_80.golden
  - internal/render/testdata/TestGoldenFrames/timeline_120.golden
  - internal/render/testdata/TestGoldenFrames/timeline_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/timeline_200.golden
  - internal/render/testdata/TestGoldenFrames/timeline_200_tmux.golden
  - internal/render/testdata/TestGoldenFrames/timeline_60.golden
  - internal/render/testdata/TestGoldenFrames/timeline_60_tmux.golden
  - internal/render/testdata/TestGoldenFrames/timeline_80.golden
  - internal/render/testdata/TestGoldenFrames/timeline_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/timeline_last_80.golden
  - internal/render/thread.go
  - internal/render/user.go
  - internal/screen/absorb.go
  - internal/screen/effect.go
  - internal/screen/list.go
  - internal/screen/list_test.go
  - internal/screen/walk.go
  - internal/store/read.go
  - internal/text/agreement_test.go
  - internal/text/pad.go
  - internal/text/text_test.go
  - internal/text/truncate.go
  - internal/text/width.go
  - internal/text/wrap.go
  - internal/tui/boundary_test.go
  - internal/tui/corpus_test.go
  - internal/tui/cover_e2e_test.go
  - internal/tui/dispatch_test.go
  - internal/tui/golden_test.go
  - internal/tui/model.go
  - internal/tui/profile_test.go
  - internal/tui/testdata/TestGoldenFrames/profile_ci_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_ci_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_gitlog_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_gitlog_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_internal_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_internal_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_servicelog_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_servicelog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_ci_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_ci_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_gitlog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_gitlog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_internal_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_internal_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_servicelog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_servicelog_80.golden
  - internal/tui/tui_test.go
  - internal/tui/update.go
  - internal/tui/view.go
  - internal/xsource/home.go
  - internal/xsource/tweets.go
  - internal/xsource/user.go
  - internal/xsource/xsource_test.go
covered_digest: "v3:sha256:c37bef60587cd232b02de37f7cd404bb087785f7b85638eed79be441fcc43525"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 23/24
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  pass: 8
  changed_since_previous: "the keymap-dispatch guard: internal/tui/update.go's press was restructured so each action has exactly one arm and the handled set is enumerated (handledActions, plus an exported HandledActions()), and internal/tui/dispatch_test.go was added, asserting the keymap<->dispatcher join in both directions plus a source sweep over keys.go's declarations, its rows, the enumeration and the shell's case labels. A whole-tree diff against the seventh pass's pristine copy (/tmp/xtv2-base) shows exactly these two paths changed and nothing else in the repository."
deferred:
  - truth: "In a bare Ghostty window the same frame lands the cursor on the frame's last row, which is only true while no row wraps and the measure stays capped"
    addressed_in: "human check"
    evidence: "Re-run this pass: `go test ./internal/render -run TestCaptureInGhostty -count=1 -v` on this host prints `--- SKIP: TestCaptureInGhostty` (with its self-test subtest, `the frame is checked before it is written`, passing). The arm's own skip sentence (`capture_pty_test.go:69-71`) needs `TMUX` unset and `TERM_PROGRAM=ghostty`, i.e. a real 80-column Ghostty window; this host has neither a Wayland session nor a Ghostty binary. Deferred to the human check, not failed and not counted as verified."
coincidental_reliance_items:
  - truth: "Scrolling back never shows the same post twice, and a cursor that repeats or a page that contributes nothing unseen ends the walk with a named stall instead of looping"
    reason: fixture-only
    harden: "The dedupe and stall proofs run on internal/fake's page sets and on synthetic rows; the real adapter's cursor semantics (page.Next from the upstream's own bottom cursor) are proven only by the recorded fixture's parse, not by a repeated cursor the real API produced. Promote it into the adapter's contract or into a recorded repeated-cursor response."
---
# Phase 2: Following Timeline in the Pane Verification Report

**Phase Goal:** The user reads their Following timeline in a terminal pane — scrolling back through older posts and refreshing deliberately — and the pane never lies about what it is showing.
**Verified:** 2026-10-15T07:45:00Z
**Status:** passed
**Score:** 23/24 truths verified (1 deferred/uncertain: P05.3), 0 failed
**Re-verification:** Yes — eighth pass, on the frozen tree after the **keymap-dispatch guard**. The one **P2** this report carried (note 8 last pass: no arm joined the keymap table to the model's dispatch) is now closed, and closed by structure rather than by prose. `internal/tui/update.go`'s `press` was restructured so each action has exactly one arm — the four combined case labels (`ActionBookmarks, ActionLikes`; `ActionCover1..ActionCover4`; the eight movement actions) split into one arm each, every arm calling the same handler with the same `action` — the handled set is enumerated in `handledActions` beside the dispatcher with an exported `HandledActions()` read of a copy, and the new `internal/tui/dispatch_test.go` asserts the join in both directions plus a source sweep that joins `internal/keys/keys.go`'s action declarations, its rows, the enumeration and the shell's case labels. **The refactor is behaviour-preserving, and that is measured rather than assumed:** a whole-tree diff against the seventh pass's own pristine copy (`/tmp/xtv2-base`) shows the change is exactly the three combined case labels split into one-arm-per-action with identical handler bodies, the `slices` import, the enumeration and two comments — the gate order (panic → prompt → lookup → help → cover gate → switch), the wording of every arm and every other line of `press` and `move` are byte-identical, which the diff of lines 227–330 shows directly. The flows that depend on `press` were each re-exercised by their own named tests (the panic gate through `TestPanicSwallowsKeys`/`TestPanicIgnoresCoverBreakingKeys`, the gated actions through `TestCoverBreakingActionsGated`, the prompt through the four `TestSearch*` prompt tests, the pushes/pops through `TestTheBackStackRestoresTheFrame`, `TestADiscardedFramesAnswerIsDropped`, `TestThePaginationTriggerIsSingleFlight`, `TestRefreshIsDebouncedAndAnnounced`), all green, and by plants (G4: every arm whose case label lives in two switches removed in turn, each caught by its own behaviour tests; G7: an arm with only one case label removed, caught by the guard and by the behaviour tests). No truth changed status: the 23 verified truths stay verified, P05.3 stays deferred.

**Carry-over:** the seventh pass's structure, its truth table, its requirement mapping, its human-verification section and the technical-debt note section are preserved. This pass changes four things: the P2 note (8) becomes a resolved entry, a new subsection records what the guard does and what its plants measured, a new falsification block records this pass's arms (G1–G7 plus the reversed B1), and the covered-file set gains one path (`internal/tui/dispatch_test.go`, new in the guard). No code changed since the seventh pass except the two paths the guard touched.

**MVP mode.** The phase is `mode: mvp` in ROADMAP.md, but its goal is not in User Story form: `gsd-tools query user-story.validate` answers `valid: false` with all three slot errors. Following the reading the other phases' passes took, this report keeps the goal-backward structure against the roadmap success criteria and the plans' `must_haves` rather than writing a User Flow Coverage table against a non-story goal.

## What the amendment does, measured from the tree

*(The sixth pass's own measurement of the amendment round, preserved; its claims and arms are the ones the seventh pass re-ran, all still biting correctly — see the falsification section.)*

Each claim below was checked against the code and then **falsified by breaking it**: the arm that must fire is named, and its failure output is quoted in the falsification section.

1. **A covered likes frame keeps its own hedge.** `CopyCoverEmptyHedged` (`copy.go:304`, new) is the cover's own one-line answer, and `SavedEmptyState`'s likes branch picks it per grammar (`saved.go:176-179`: `if cover != "" { StateBlock(g, w, CopyCoverEmptyHedged, "") } else { … CopyEmptyLikesHedged … }`). 8 stored frames pin it (`cover_{servicelog,gitlog,ci,internal}_likes_hedged_{80,120}`), `TestTheCoveredLikesHedgeIsInTheCorpus` reads them from disk (one block, one row, no `CopyCoverEmptyBody`, no plain sentence, no `CopyListLikes`) and `TestTheCoveredLikesHedgeIsOneLine` states the rule per style; the plain arm (`likes_hedged_80/120`, itself pinned by the scan) keeps the absence from being vacuous. **Arms:** adding the body back fails 5 frames + the unit test (N1a); making the covered hedge the plain sentence fails both (N1b).
2. **The plain-vocabulary scan covers every covered frame in both corpora.** `TestCoveredFramesLeakNoRepairWords` (render) iterates `partition(storedFrames)` and fatals unless it scanned every covered frame (`scanned < len(covered)`), with its own shapes arm; `TestTheCoveredTUIFramesLeakNoPlainVocabulary` (`internal/tui/corpus_test.go:803`, new) is the tui corpus's first scanner, fatals under 24 covered frames (the corpus holds exactly 24 by style naming: 4 styles × profile/search/people × 2 widths), and requires every scanned frame to print one of its own style's aliases. Both plant arms now base on a frame a leak could hide in: `cover_gitlog_thread_80` (a conversation) and `search_people_gitlog_80` (a covered tui frame). **Arms:** a plain row planted through the committed corpus is caught in each (N2a, N2b — the rendered-and-re-blessed scenario the docstring narrates).
3. **`frameKinds()` is no longer dead code.** `model.go:196` returns the five kinds and its comment names the join; `frameKindChecks()` (`cover_e2e_test.go:634`) has exactly one row per kind, and `TestEveryFrameKindIsCheckedUnderACover` requires `counts[kind] == 1` for each declared kind **and** `len(table) == len(declared)`, then walks the shipped keys to each surface under a cover and requires a different frame, no product word and the surface's own status field in the style's words. **Arms:** a sixth kind added to the enumerator fails (`the shell declares the frame kind 5 and the covered check table has 0 rows for it`, N3a); a row's kind changed so one kind has none fails the other direction (`the shell declares the frame kind 4 and the covered check table has 0 rows for it`, N3b); deleting the function no longer leaves the suite green — the test does not compile (`undefined: frameKinds`, F19, where the last pass measured the whole suite passing).
4. **The four covered notices have frames and a totality arm.** 16 frames × their tmux twins = 32 stored files (`cover_*_{sync,absent,complete,stalled}_notice_120`), generated from `golden_cover_corpus_test.go`'s `noticeState` table; `TestTheCoveredNoticesAreOnScreen` asserts each covered frame carrying a notice carries its **own style's** words (and not the plain reader's) and that every one of the four kinds is carried by some frame. **Arms:** reverting `NoticeSyncStalled` to the plain constant fails 8 frames with both messages (N4a); giving a kind no frame fails the totality arm (`no covered frame in the corpus carries a read in progress: its words are asserted over no frame`, N4c).
5. **A covered profile names one account consistently.** `profileAccount` (`view.go:447`) answers the header's account once it has arrived, the opened handle only for the plain reader (`m.covered() == ""`), and nothing under a cover before the answer; `TestTheCoveredProfileNamesOneAccount` reads the field in the state the shipped path leaves it in (header request unrun) and after the answer, for all four styles, with the plain arm as the counter-case. **Arm:** returning the opened handle under a cover fails every style (`the covered profile frame names an account before its header has answered: "account: pump-sign-57  0 requests, 0 rate-limited"`, N5).
6. **Every cover stamp sets `MonthYear`, so the fourth facts clause renders.** `coverISOStamp`/`coverGitStamp`/`coverClockStamp`/`coverTableStamp` each carry a layout (`cover.go:197/203/207/210`) and `expandFact` refuses `{joined}` when `g.Stamp.MonthYear == ""` (`profile.go:129`). The frames say so: `profile_gitlog_80` now reads `Watchers: 1204  Watching: 312  Commits: 8901  Since: Mar 2011` where the last pass measured three clauses and a dead one. **Arm:** blanking one layout fails that style's two stored frames (N6).
7. **Hygiene.** (a) The nine dead `copy.go` constants are gone (`CopyFactFollowers/Following/Posts/Joined`, `PlaceholderFollowers/Following/Posts`, `CoverCINewReply(s)`: 0 references in the tree) and an independent scan of `copy.go`'s 250 constants finds **0 with no reader**; the CI replies' single-case fallback is now documented as one vocabulary rather than a duplicate pair. (b) The capture harness unlinks its per-run socket and waits for the server to stop answering before it goes (`tmuxKill` → `removeTmuxSocket`, `capture_tmux_test.go:352-395`): measured **0 growth** in `/tmp/tmux-1000` across six capture runs that pass (280 files before and after each of those six; this pass measures the directory at **0** files), two concurrent runs both exit 0, and no server survives. One file in the harness's own naming scheme (pid 2809916, created 13:08:12 UTC) appeared in the directory during the pass's window, between two of my measurements, and was not left by any run of this tree's harness — every measured run, including two taken after it appeared, shows 0 growth, and the harness's own assertion (N7b) would have failed any run of it that left one; the likeliest writer is a stale process or a parallel pass on another copy. **Arm:** neutralising the removal makes the harness's own assertion fire (`the harness left its socket file behind at /tmp/…/xterm-golden-2810555-…: a later run's client would find a socket no server answers on`, N7b). (c) `internal/audit`'s new first check is `source`: `x-terminal audit` reports `9 checks, 0 findings, 892 files` and exits 0 on this tree, and a planted Go file that does not parse is reported (`internal/render/zz_broken_plant.go:3: cannot parse: expected ')', found 'EOF'`, exit 1) instead of being skipped — a whole-tree check, not this phase's surface, so its files are not in this report's covered set.

**Two places the amendment's prose is looser than the tree** (both P3, neither a gap): its claim 3 says the enumerator is joined to "the whole-surface table"; the join is to `frameKindChecks()`, a per-kind table, while `screens()` (the per-state whole-surface table) is still hand-kept and has no kind field — the substance holds (a sixth kind fails the build), the name does not. And the four covered notices' frames are 32 files, 16 frames with tmux twins, which is what the brief's "16 frames" counts.

## What the fix round does, measured from this phase's surface

Each claim was checked against the code and then **falsified by breaking it**: the arm that must fire is named and its output is quoted in the falsification section below.

1. **The saved frame's "mirror unreadable" state is a whitelist, not a blacklist.** `feed.go:129-132` is now `var file *store.FileError; return errors.As(err, &file)` — the blacklist and `savedTransportError` are gone — and `internal/store/read.go:37` is where the store marks its own file failures (the page open at `read.go:55`, the status read at `read.go:132`, the schema write at `schema.go:43`). The other direction is unchanged: `errors.Is(err, store.ErrWrongOwner)` still answers `CapWrongAccount` (`feed.go:244-246`). **Arm (new this pass):** teaching the predicate that a `*xsource.ParseError` is the file too fails `TestAShapeDriftedAnswerIsNotAFactAboutTheFile` (`a shape-drifted answer … was read as a failure of the mirror: the pane would tell the reader to move a healthy saved.db aside`), the status line rendering `mirror unreadable  move saved.db aside, then sync` where the shape's own state belongs.
2. **The audit's tenth check.** `internal/audit/audit.go:187` registers `opnames` ("every operation name the classification table knows is one the registry resolves"); `checkOpNames` (`internal/audit/readonly.go:171`) intersects `opClass` with the registry snapshot and was written for the stale `ListLatestTweetsTimeline` entry. Re-measured live: `x-terminal audit` from the repository root prints `10 checks, 0 findings, 893 files inspected`, exit 0. **Arm (new this pass):** putting the stale entry back makes the same run print `internal/audit/readonly.go: the table classifies "ListLatestTweetsTimeline" as a read and the registry does not hold it: an entry for an operation this build cannot resolve` and exit 1, and fails `TestOpNamesPassesOnTheRealTableAndSnapshot`.
3. **`keys.ByKeystroke`'s comment is now true.** `internal/keys/keys.go:215-219` says it is the keymap's own tests' matcher with no production caller; measured: the only references outside `keys.go` are `internal/keys/keys_test.go:35,57,196,213`, input handling goes through `keys.Lookup` (`internal/tui/update.go:325-328`), and an audit of `internal/render`'s non-test imports finds no keymap import.
4. **No golden changed.** The fix round regenerates no stored frame: the corpus re-counts at **263 render frames + 56 tui frames = 319 files** (the sixth pass's 279 → 319 stands), this phase's 31 are all still stored and still compared, and `TestGoldenFrames` passes byte for byte in both packages — so the round's three changed covered files moved no frame's bytes.
5. **The shared tui corpus's saved scenario drives a reachable state.** `internal/tui/cover_e2e_test.go`'s `CapMirrorUnreadable` leg sets the store's own failing read (`mirror.FailPage`) beside the capability map, so the frame's read is what answers with the failure the state names, rather than a mirror that answers while the frame claims otherwise. **Arm (new this pass):** removing the failing read leaves the state unreachable and the corpus's own anti-vacuity arm fires (`the plain frame of "the store cannot be read" does not carry "move saved.db aside": the covered absence below would be vacuous`).

## What the keymap-dispatch guard does, measured this pass

The guard is the fix for the P2 the seventh pass recorded (note 8): nothing joined the keymap table
to the model's dispatch, so a row could be added, documented by `?` and by `keys`, and do nothing.
It is two files.

1. **`press` is one arm per action, and the handled set is spelled out beside it.**
   `internal/tui/update.go`'s dispatch now has a separate `case` for each action — the four combined
   labels (`ActionBookmarks, ActionLikes`; `ActionCover1..ActionCover4`; the eight movement actions)
   were split, each new arm calling the same handler with the same `action` — and `handledActions`
   (`update.go:1294-1309`, 30 entries) names the set, with `HandledActions()` (`update.go:1315`)
   returning a `slices.Clone` of it for callers outside the package. The restructure is mechanical:
   a whole-tree diff against `/tmp/xtv2-base` (the seventh pass's pristine copy, confirmed to be the
   pre-guard tree) leaves `/tmp/xtv3-update-diff.txt` showing only the three split labels (with
   identical handler bodies), the added `"slices"` import, the enumeration and two comments. The
   gate preamble (panic → prompt → lookup → help → cover gate → switch) and every other arm are
   untouched, so the change cannot alter which key reaches which handler.
2. **`internal/tui/dispatch_test.go` joins the two ends, three ways.**
   `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` reads `handledActions` and
   `keys.Rows()` and fails both directions (a row whose action is not handled; an action handled
   with no row), with anti-vacuity floors (`len(keys.Rows()) == len(handled)` and `len(handled) >= 30`)
   and a copy check on `HandledActions()`. `TestTheShellDeclaresOnlyActionsTheKeymapDeclares` walks
   the sources: it reads the action *declarations* out of `internal/keys/keys.go` by regex, then
   requires every declared action to be a row's action, a member of `handledActions` and a `case`
   label somewhere in the shell's non-test files (parsed with `go/ast`, so the planted arm's prose
   in `update.go` is not counted as a reference), and the reverse (a `keys.Action…` selector the
   shell names but the keymap does not declare). Both tests cover exactly the defect class AUTH-03
   shipped, where a verb was implemented and its table row deleted.

**What its plants measured (this pass, on fresh `/tmp` copies).** Each plant is a fresh copy of
`/tmp/xtv3-base`, the plant applied, the guard's own tests run; the quoted line is what the test
printed.

- **G1 — a 31st row with an action nothing handles** (the exact plant the seventh pass ran as B1,
  when it passed everything): `ActionNudge` added to `keys.go` and bound to `ctrl+n`.
  `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` FAILs with `the keymap binds "nudge"
  (ctrl+n) and press has no arm for it: the key is a documented no-op` and `the table holds 31 rows
  and handledActions names 30 actions`; `TestTheShellDeclaresOnlyActionsTheKeymapDeclares` FAILs
  with `keys.go declares ActionNudge and handledActions does not name it: the enumeration went
  stale` and `keys.ActionNudge is no case label in the shell: nothing dispatches it, and
  handledActions is data without an arm`. **So the seventh pass's B1 now fails — the gap it
  recorded is closed.**
- **G2 — a handler for an action no row binds** (the guard's other direction, as the author
  reported it fails): `ActionNudge` added, a `case keys.ActionNudge: return m, nil` arm written in
  `press` and the action added to `handledActions`, but no keymap row. FAILs: `press handles
  "nudge" and no row of the keymap binds it: the arm is unreachable`, `the table holds 30 rows and
  handledActions names 31 actions`, and `keys.go declares ActionNudge ("nudge") and no row binds
  it: an action with no key is not an action`.
- **G3 — the arm the bogus case label's obituary claims** — `case
  keys.ActionThisPackageDeclaresNoSuchAction: return m, nil` planted in the switch: `go build ./...`
  and `go vet ./internal/tui` both refuse it with `undefined:
  keys.ActionThisPackageDeclaresNoSuchAction` (`update.go:327:12`). The compiler refusing a bogus
  case label is what lets the sweep read handledActions as text rather than parse it.
- **G5 — the sweep's own floor** — one declaration reshaped so the regex sees 29:
  `the sweep read 29 action declarations out of keys.go, fewer than the table's rows: it is not
  reading the source` — the sweep is not vacuous.
- **G6 — an action dropped from the enumeration only** — `ActionHelp` removed from
  `handledActions`: FAILs `the keymap binds "help" (?) and press has no arm for it`, the count
  mismatch, and the floor `handledActions names 29 actions`.
- **G7 — a one-case-label arm removed** — `case keys.ActionSwitchList: return m.toggleFeed()`
  deleted: the guard FAILs `keys.ActionSwitchList is no case label in the shell: nothing dispatches
  it, and handledActions is data without an arm`, and the behaviour tests fail with it
  (`TestTheBackStackRestoresTheFrame`, `TestADiscardedFramesAnswerIsDropped`, `TestPanicRoundTripIsExact`).
- **G4 — a two-case-label arm removed (the guard's one blind spot, measured)** — `ActionCopyLink`,
  `ActionMedia`, `ActionOpenPost`, `ActionQuit` and `ActionPanic` each have their `case` label in a
  second switch too (`coverBreaking`, the panic gate), so deleting the main switch's arm leaves the
  sweep's `shellActionCases` satisfied (it reads case labels package-wide, not per switch); the
  guard's own tests are **green** under each of those five plants. Each is caught anyway, by the
  behaviour tests that press the key: `ActionCopyLink` → `TestCopyLink`, `TestCopyLinkWLOnlyIsLinkCopied`;
  `ActionMedia` → the six `TestMedia*`; `ActionOpenPost` → `TestOpenPost`; `ActionQuit` →
  `TestTheQuitKeysWriteNothing`, `TestAQuietKeyAndTheTwoExits`; `ActionPanic` →
  `TestPanicRoundTripIsExact`, `TestPanicSwallowsKeys` and five more. Recorded as a note (the
  guard is a structural join "declared ⇔ handled ⇒ a case label exists", and for those five actions
  a case label does still exist), not as a gap: the phase's truth (P04.2, the three surfaces read
  one table) is unaffected, and no such arm can ship unnoticed because the key's own test fires.

## Goal Achievement

### Observable Truths

Roadmap success criteria (the contract) first, then every PLAN `must_haves.truths` entry in its own words.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| R1 | The Following timeline is the default list, and the algorithmic feed is reachable only through an explicit toggle | ✓ VERIFIED | `Feed` rejects every argument (`if len(args) > 0` → usage, exit 2, re-measured live this pass: `feed: unknown argument nope`), so the feed verb and the argument-less invocation are the only entry points and both open on Following; the toggle is the only producer of a second feed frame (`otherFeed`'s answer is pushed from `toggleFeed` and nowhere else) and `internal/keys/keys.go` still scopes the `t` row to `Screens: home`. Live this pass: the binary built from this tree, in a real tmux pane inside `unshare -rn` with only `lo` up and an absent session, opened on Following with the status line `no session file  write the session file  following  0 requests, 0 rate-limited` (24 rows, row 24 the line, 80 cells). Re-falsified: making the bare invocation print the usage fails `TestABareInvocationOpensTheFeed` with `a bare invocation exited 2, want 0` (F9). |
| R2 | Scrolling with keys loads older posts keyset page by page; the viewport never jumps when a late page arrives, no post appears twice, and a stalled cursor is reported instead of looping | ✓ VERIFIED | The machinery is the shared `internal/screen/walk.go`: `absorb` de-duplicates on the element's id map and stalls on a repeated or unproductive cursor; `Page`/`Refresh` hold the single-flight latch and the request gate; `TestThePaginationTriggerIsSingleFlight` pages and asserts the late page moved neither the selection, the anchor, nor the rows above it. Re-falsified this pass: routing answers by the top frame's id instead of the asking frame's loads 9 posts into the wrong frame and reports a stalled walk on a list that was asked once (`tui_test.go:964-979`, F11). |
| R3 | Refresh is an explicit key: it preserves selection and scroll and announces "N new posts" instead of silently replacing what is on the list; a fetch that fails partway still renders what was received, with the failure stated in one quiet status line | ✓ VERIFIED | `TestRefreshIsDebouncedAndAnnounced` (one request for two presses, `2 new posts`, the selected id unchanged, its index moved down 2, pane byte-identical before and after) and `TestAFailedPageKeepsThePosts` (one request, pane byte-identical, the line carrying the state word). Re-falsified this pass: deleting `Walk.refresh`'s `defaultSelection()` fails `TestARefreshIntoAnEmptyListSelectsTheFirstPost` (`the refresh filled an empty list and selected "", want the first post "new-0"`, F12); deleting `answered`'s `m.failure = domain.Failure{}` leaves a stale `session expired` on the line after a successful fetch (F10). |
| R4 | The four named failure states each render with a distinct remedy — expired session, rate limited (with the computed reset time), offline / no route, upstream shape change — alongside a live request-budget line, and a soft ban (`code 88` with budget left) stops the session and says so rather than being retried | ✓ VERIFIED | `TestTheFourNamedStatesRenderTheirOwnWords` + `TestTheFourNamedStatesAreTheFourOPSRows` + `TestStatusLineRateLimitedForms` (the upstream's own reset is what the line prints) + corpus A2 over the stored frames. Offline is reachable in the shipped app: `feed.go:279` still builds `Check: feedReach(client, cfg)` through the same client the pane reads through — re-falsified in both directions this pass (`Check: nil` fails `TestTheFeedClassifiesAReachabilityFailure` with `the status line is "unclassified  record this shape first  following  1 request, 0 rate-limited", want it to carry "no route to x.com"` and `the reachability check ran 0 times for one unnameable failure, want once`, F13). |
| R5 | The renderer and navigation contract holds: borderless, unbranded and engagement-count-free; one width model shared by wrap, pad and truncate and correct for CJK, emoji and grapheme clusters; correct reflow on resize; colour degrading truecolor → 256 → 16 → mono honouring `NO_COLOR` and `TERM=dumb` with no meaning carried by colour alone; vim-ish keys, `?` help, shell completion and a real back stack all derived from one keymap table; golden frames at 80 and 120 columns, inside and outside tmux, as the view specification | ✓ VERIFIED (one substitution) | The corpus now stores **263 render frames** (was 223; the amendment added 40 covered frames) plus **56 tui frames** — 319 files, matching the brief's 279 → 319 — and this phase's 31 are all still stored and still byte-compared by `TestGoldenFrames`. The four absence arms were re-planted into stored frames this pass, in place so the row count stays 24, and each fired: an engagement count (`the stored frame timeline_80 prints 3813, an engagement count its own posts carry`), a media location (`the stored frame media_opened_80 prints the media location "https://pbs.twimg.com/media/a.jpg"`), a display name (`prints the display name "name of a_15_cell_handl"`), a non-ASCII glyph (`row 4 … holds "â" (U+00E2)`) — F7a–F7d. The help overlay's cut was re-falsified (F8). The substitution is REND-05's third surface (the `keys` verb, not a completion script), documented in the requirements table below. |
| P01.1 | The Following timeline arrives as an ordered page of domain posts through one port, and the same port is satisfied from the recorded fixture with no network and no session | ✓ VERIFIED | `internal/domain/source.go` (exactly `Home`/`User`), `xsource.Adapter`, `fake.Source`, and `TestFakePagesMatchTheFixture`. This pass: the **whole** suite again inside `unshare -rn` with only `lo` up, `GOPROXY=off`, `XDG_*` in `/tmp` and `GOMODCACHE` pointed at the real module cache — 19 packages `ok`, 3 with no test files, 0 failures — plus the repository run outside the namespace: **1571 passed, 22 packages, 0 failures** (19 `ok`, 3 with no test files; 8 skips, the deferred Ghostty capture among them). |
| P01.2 | Scrolling back never shows the same post twice, and a cursor that repeats or a page that contributes nothing unseen ends the walk with a named stall instead of looping | ✓ VERIFIED | `walk.go`'s dedupe set, `usedCursors` and stall rule, reached from `absorb.go`'s folds; `TestPaginationAbsorbsEachPostOnce`, `TestARepeatedCursorStallsTheWalk`, `TestAPageThatAddsNothingUnseenStallsTheWalk`, `TestWalkAbsorbsEachRowOnce`, `TestWalkStallsOnACursorThatComesBack`. An adversarial read of the folds found no event sequence that re-enters an id or returns an effect after a stall; the F11 arm shows the stall is reachable from the shipped shell as well as from the walk's own tests. |
| P01.3 | A page fetch that fails keeps every post already loaded and records exactly one failure for the status line | ✓ VERIFIED | `pageFailed` touches only the three latches, `requested` and `failure`; `TestAFailedPageKeepsEveryPostAndTheSelection`, `TestAFailedPageKeepsThePosts`, `TestTheResultsFrameFailureKeepsTheRows`; unchanged by the amendment and green in both suite runs. |
| P01.4 | After a soft ban, no page request and no refresh is ever issued again for the life of the process, and the list says the session stopped | ✓ VERIFIED | `blocked()` returns `true` for `failure.Stopped` before anything else and both `Page` and `Refresh` are gated on it; `requestAllowed` in the shell is the same gate for every later frame; `TestASoftBanStopsTheSessionForTheProcess` (screen) and `TestASoftBanStopsTheSession` (tui, a flat call count across `G`/`r`/`t`); unchanged by the amendment and green. |
| P02.1 | A post body wraps to the pane the same way the pane measures it, and a CJK run, an emoji and a ZWJ family all measure as what the running terminal will actually draw (nothing wider than the line ever reaches the screen) | ✓ VERIFIED | `TestEnvironmentMovesTheMeasure` (ZWJ family 2 bare / 6 tmux, CJK 16 in both); every `Wrap` line ≤ w over the 23-entry corpus at 8 widths in both `$TMUX` arms; corpus A7 re-measures every stored frame's rows against its grammar's own budget. Re-falsified this pass through the width model itself (F1–F5), each on a fresh copy. The hand-built adversarial corpus an earlier pass used was not rebuilt; the evidence is the in-tree corpus. |
| P02.2 | The same string padded or truncated to a width reports exactly that width back, and a truncated field ends at a cluster boundary with no invented mark | ✓ VERIFIED | The four former counterexamples are still fixed; each was re-falsified separately this pass with its failure message quoted (F1–F5): `Pad`'s absorbed fill (`Width(Pad(s, 2)) = 1, want exactly 2 … (s = "a۝")`), the unterminated escape's drop, `Wrap`'s content test on `flush` (`Wrap(s, 8) does not carry the non-space characters of s in order: got "", want "́"`), and both of `Truncate`'s guards (`Width(Truncate(s, 2)) = 6, wider than the measure (s = "👨‍👩‍👧")`; `Truncate(s, 1) = "a", which is not a prefix of s = "abcdef"`). `TestA4Agreement` + `TestCorpusIsWhatItClaims` pass over the 23-entry corpus in both arms. |
| P02.3 | Under tmux the measure follows wcwidth semantics and in a bare Ghostty it follows grapheme-cluster semantics; both hold every property, only the wrap points move | ✓ VERIFIED (one measured divergence) | `TestEnvironmentMovesTheMeasure` (2 vs 6 cells; 1 vs 0 lines at w = 4); `TestCaptureTmux` passes against real tmux 3.7c (`timeline_80` + `timeline_120` subtests), which itself measures the ZWJ family as 2 cells where the model's tmux arm says 6 — documented in `capture_tmux_test.go` and 02-05-SUMMARY, and the model's error direction over-counts, so it cannot wrap a row. |
| P03.1 | A list of domain posts renders as the UI-SPEC's own 80-column and 120-column frames, byte for byte, with the block layout, the 16-cell handle column, the 12-cell absolute stamp, the 2-cell indent, the capped measure, the quote prefix and the one blank separator row | ✓ VERIFIED (two documented divergences) | `TestBlocksTheUISpecFrameAt80Columns`/`120Columns` transcribe the frames and still pass; the two places the fenced text cannot be the width model's output are asserted with their cell counts in `TestTheFrameDivergencesFromTheUISpecAreTheMeasuredOnes`. The 02-UI-SPEC.md is in the covered set for this row. The regenerated `help_80`/`help_120` are not a third divergence: the UI-SPEC fences no help frame; this pass re-read them live (the overlay's own cut sentence, below). |
| P03.2 | The frame is always exactly the terminal's rows by its columns, the status line is always its last row and never wraps, and the selection is legible from the `>` in column 1 alone when every escape sequence is stripped | ✓ VERIFIED | `TestFrameIsExactlyThePanesSize`, `TestStatusLineNeverWraps`, `TestFrameTooNarrow` (`MinPaneCols = 60`), corpus row-count + golden byte-equality, corpus A3 (the mark in column 1 with escapes stripped) and `TestFramePaintsTheSelectionBar`. Live this pass: the shipped pane at 80, 60 and 40 columns is exactly 24 rows, no row wider than the pane, the status line last; at 40 columns the too-narrow line (`widen the pane to at least 60 columns`) replaces the list and the status row is cut, not wrapped. (The sixth pass wrote "every row its own width (80/60/40 cells)"; `capture-pane` only reports the rows the renderer wrote, so the measured statement is the one above — the frame's uniform padding is the stored goldens' and `TestFrameIsExactlyThePanesSize`'s.) |
| P03.3 | Colour degrades truecolor to 256 to 16 to mono from the environment, a mono rung emits no colour at all, and stripping every escape sequence from a coloured frame leaves exactly the same characters as the mono frame | ✓ VERIFIED | `TestRungFromEnvIsTheLadder` (the four rungs, from the environment and nothing else), `TestColourNeverCarriesMeaningAlone`, `TestFrameColourIsOnlyASecondStatement`, and corpus A1 over the stored bytes (the arms that the truecolor frame carries escapes and that a selected block is present). Unchanged by the amendment; A1 is also the check my plants trip, so it is exercised in every plant arm above. |
| P04.1 | Running the binary with no arguments opens the Following timeline; the algorithmic feed is one key away and one key back, and the frame returned to is byte-identical to the frame that was left | ✓ VERIFIED | `TestABareInvocationOpensTheFeed`; live this pass in a real tmux pane inside an empty network namespace with an absent session (the bare binary opened the pane, `q` quit it with `EXIT=0`, recorded through a wrapper because the tmux server leaves with it). Re-falsified: making the bare invocation take the usage path fails the test with `a bare invocation exited 2, want 0` (F9); `TestTheBackStackRestoresTheFrame` still holds the byte-identity leg. |
| P04.2 | Keys, the `?` overlay and the `keys` subcommand all read one table, so adding a key is one row and the three surfaces cannot disagree | ✓ VERIFIED | `internal/keys/keys.go` holds 30 rows, `keys` printed 30 tab-separated lines this pass (`j, down  next  next post` …), the overlay is built from `keys.ForScreen` via `helpRows`, and `internal/render` still never imports the keymap. The overlay names its cut rather than clipping silently: `render.Help(rows, w, pane)` keeps the pane's last row for its cut sentence (`CopyHelpClipped`), which the live pane printed as `~ 8 more keys than the pane can show`. Live, the `?` pane at 80×24 matched `help_80.golden`'s rows 1–23 in every character after trailing padding is trimmed (tmux's `-N` capture preserves trailing spaces only where its own line buffer did, which is a capture artifact and not the program's rows — the raw capture shows rows 40 and 80 cells wide alternately while the content is identical). **Re-measured this pass:** 30 rows, `keys` printed 30 lines, the live `?` pane matched `help_80.golden` rows 1–23 with **0 mismatched characters**, and the three surfaces are now joined to the *shell* as well — `internal/tui/dispatch_test.go` holds the keymap table’s actions and the dispatcher’s handled set to each other in both directions, plus a source sweep (G1–G7). |
| P04.3 | The selected block stays on screen when the terminal is resized and when a late page arrives, the selection is a post id rather than a row, and a page that arrives never moves what the user is reading | ✓ VERIFIED | `TestResizeRerendersWithoutFetching` (resized frame == cold frame, no fetch), `TestThePaginationTriggerIsSingleFlight`, `TestTheSelectedHeadStaysInsideThePaneUnderEveryCover` (parent + 20 subtests, re-run this pass), and `TestTheBackStackRestoresTheFrame`. **Re-read against the amendment:** the `g` branch that clears `Moved`/`Anchor` only re-enters the opening rule, which this phase's truth already relies on; my own probe drives a plain 80×24 timeline six moves down and presses `g` — the frame is unmoved, the selection back at the first loaded post, no request spent, the pane byte-identical to a fresh frame's pane (F21), and neutralising the branch fails that probe as well as `TestGoRestoresTheProfileHeader` (F21b, F17). |
| P04.4 | One quiet status line carries the state, the remedy, the list, the position and the request budget in every frame, including every failure frame, and the four named failure states each render with their own words | ✓ VERIFIED | The line composes in the fixed field order and drops in the fixed order — notice, position, list, synced (`status.go`; `synced` is phase 6's addition to the droppable set, after `list`, so this phase's three-field order is unchanged) — while `state`, `remedy` and `budget` are never dropped; below the width that holds them the row is cut with `~`, which is what 02-UI-SPEC.md asks for. Live this pass: at 60 columns the last row was `no session file  write the session file  0 requests, 0 rate~`; at 40 columns the pane was the too-narrow line and the row was cut at `write the session file~`. |
| P04.5 | After a soft ban the session stops: the state line says so, the state word takes the stopping accent, and no further request is issued | ✓ VERIFIED | `TestASoftBanStopsTheSession`, `blocked()`'s `Stopped` case, `stateRole`'s `RoleStateStop`; not reproducible against the live upstream (a real soft ban would be needed). Unchanged by the amendment and green in both suite runs. |
| P05.1 | The stored frames are the view specification: the same state renders the same bytes at 80 and 120 columns, and every one of the UI-SPEC's adversarial cases is visible in a named frame — a width bug cannot hide | ✓ VERIFIED | `TestGoldenFrames` byte-compares the stored corpus: **263 render frames now** (the amendment added 40 covered frames; the corpus was 223 at the last pass) and **56 tui frames**, 319 files in all; this phase's 31 are all still stored and still checked. The audit's required cases all resolve, `longdetail_80` included. Re-falsified this pass: `StateBlock`'s body printed verbatim fails `TestGoldenFrames/longdetail_80` (F6), each of the four absences fires on a planted stored frame (F7a–F7d), and the amendment's own covered goldens are the guard for its claims (N1a, N1b, N4a, N4c, N5, N6). |
| P05.2 | The frames written into a tmux pane at 80×24 and 120×24 come back from `capture-pane` with the same text, the same row count and the same wrap points, so the terminal agrees with the model | ✓ VERIFIED (**correction reversed: the harness now cleans up after itself**) | `TestCaptureTmux` re-run this pass: `--- PASS` with the `timeline_80` and `timeline_120` subtests, against real tmux 3.7c on the harness's per-run socket, and two concurrent `-run 'TestCaptureTmux|TestCapturePanic'` runs both exited 0. The last pass had to narrow the wording because each run left a 0-byte socket file behind: **that is fixed.** `/tmp/tmux-1000` holds **281** files again this pass — an external accumulation since the seventh pass (the oldest is `xterm-golden` from 10:07 UTC, eight `xterm-golden-*` pairs from 10:08–10:45, and one from 13:08; none dated within this pass, none attributable to a run of this tree, whose own runs all leave their private `TMUX_TMPDIR` empty), so the harness is clean but the *host* directory is not, and the seventh pass’s “0 files” was a moment, not a property, no server answers, and the harness now asserts its own cleanup — neutralising `removeTmuxSocket` makes `capture_tmux_test.go:148` fail with `the harness left its socket file behind at …/xterm-golden-2810555-…: a later run's client would find a socket no server answers on` (N7b). |
| P05.3 | In a bare Ghostty window the same frame lands the cursor on the frame's last row, which is only true while no row wraps and the measure stays capped | ? UNCERTAIN — deferred | Re-run this pass: the arm skips with its own printed sentence (`capture_pty_test.go:69-71`), which needs `TMUX` unset and `TERM_PROGRAM=ghostty`; this host has no Wayland session and no Ghostty. Not verified, not counted as a gap; see Human Verification. |
| P05.4 | Stripping every escape sequence from a coloured frame leaves exactly the mono frame's characters: no state, marker, list name or selection is carried by colour alone | ✓ VERIFIED | Corpus A1 over the stored corpus (mono == truecolor == plain after stripping, with the arms that the truecolor frame carries escapes and that a selected block is present), `TestColourNeverCarriesMeaningAlone`, `TestFrameColourIsOnlyASecondStatement`. |

**Score:** 23/24 truths verified (1 deferred/uncertain: P05.3), 0 failed — unchanged by the keymap-dispatch guard, which restructures the dispatcher P04.2 rests on without adding or removing a key. All 24 were re-adjudicated on the frozen tree this pass. The two rows whose *evidence* moved are the guard’s own: **P04.2** (the keymap is now held to the shell by `internal/tui/dispatch_test.go`, both directions plus a source sweep — G1–G7), and the behaviour preservation of `press` itself (the whole-tree diff against the seventh pass’s pristine copy `/tmp/xtv2-base`, plus the named flows re-run and G4/G7 breaking each duplicated-label and single-label arm). No other row changed status or evidence: the guard adds and removes no action, changes no key, regenerates no golden, and touched no other file. Every arm the seventh pass recorded was re-run and still breaks what it names, with the single exception of B1, which the guard deliberately converts into a failing arm (G1).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/domain/source.go` | the Source port (cursor in, page out, plus User) | ✓ EXISTS + SUBSTANTIVE | exactly two methods, no wire vocabulary |
| `internal/domain/page.go` | PostID, Cursor, HomeKind, PostPage | ✓ EXISTS + SUBSTANTIVE | `HomeKind(0).String() == "following"` tested |
| `internal/domain/failure.go` | closed FailureKind enum + Failure | ✓ EXISTS + SUBSTANTIVE | 20 kinds + none, totality test, zero Failure non-retryable |
| `internal/xsource/home.go` | the port over an injected GraphQL caller | ✓ EXISTS + SUBSTANTIVE | `var _ domain.Source`; no `queryId` literal anywhere in the four packages |
| `internal/fake/fake.go` | recorded fixture + adversarial page set, counters | ✓ EXISTS + SUBSTANTIVE | parity test against the recorded fixture; `Calls`/`FailNext`/`StallWith` |
| `internal/screen/list.go`, `walk.go` | the frame's policy over the shared walk | ✓ EXISTS + SUBSTANTIVE | effects only, never calls a Source; dedupe/stall/single-flight/refresh diff live once, held by `TestWalkPolicyLivesInOneFile` |
| `internal/text/width.go`, `wrap.go`, `pad.go`, `truncate.go` | the single measurement and the three field operations | ✓ EXISTS + SUBSTANTIVE | the only `$TMUX` branch; all three guards present and each re-falsified this pass (F1–F5) |
| `internal/text/agreement_test.go` | the A4 property suite, both environments, anti-vacuity arms | ✓ EXISTS + SUBSTANTIVE | 23-entry named corpus, 8 widths, two arms |
| `internal/render/copy.go` | every user-visible string, both state tables | ✓ EXISTS + SUBSTANTIVE, **cleaned** | the covered vocabulary for the saved surface and the four styles' notice words, plus the new `CopyCoverEmptyHedged`; the nine dead constants the last pass listed are deleted and an independent scan of its 250 constants finds **none** without a reader. `CopyStateRateLimited` is still stated twice (`copy.go:227` and the state table at `copy.go:473`) and the two still agree — recorded as a P3 note, not a defect |
| `internal/render/grammar.go`, `block.go` | plain grammar + post-block layout | ✓ EXISTS + SUBSTANTIVE | `Plain{HandleCols:16, MeasureCap:96}`, stamp 12, quote cap 2; `StateBlock` wraps its body to the measure; `AccountFacts` is the grammar's own field |
| `internal/render/status.go`, `frame.go` | status line, drop order, full-height frame | ✓ EXISTS + SUBSTANTIVE | the drop order is notice→position→list→synced, one row at every width, frame exactly h×w, `MinPaneCols = 60` |
| `internal/render/palette.go` | rung ladder + roles | ✓ EXISTS + SUBSTANTIVE | NO_COLOR/TERM=dumb win first, then COLORTERM, then 256color, then 16 |
| `internal/render/help.go` | help overlay table | ✓ EXISTS + SUBSTANTIVE | 16-cell keys column, one row per input row, cut with the mark; `Help` takes the pane's row count and spends its last row on the cut sentence; re-verified live this pass |
| `internal/render/cover.go`, `saved.go`, `profile.go` | the covered grammar, the saved surface's state blocks, the header's facts row | ✓ EXISTS + SUBSTANTIVE, **amended** | four `AccountFacts` tables, the three grammar-taking block builders, `expandFact`; every stamp now carries a `MonthYear` layout so the fourth facts clause renders (N6 shows the frames carrying `Since: Mar 2011`), and the likes branch hands each grammar its own hedge (N1a/N1b) |
| `internal/keys/keys.go` | the keymap table + matcher | ✓ EXISTS + SUBSTANTIVE | 30 rows; `keys` printed 30 lines whose first two fields match the table; this pass: the table is now held to the dispatcher by `internal/tui/dispatch_test.go` (both directions plus a source sweep), and `ByKeystroke` still has no production caller |
| `internal/tui/model.go`, `update.go`, `view.go` | pure Model, frame stack, anchor, effect→command | ✓ EXISTS + SUBSTANTIVE, **amended** | see P04.x; `Render` returns `render.Frame` verbatim; the answer is matched on the asking frame's id; `Model.skip`/`reanchor` are the scroll rule; `move`'s `ActionFirst` branch, `savedPane`'s `m.grammar()`, `profileAccount` and `frameKinds()` (now joined to a per-kind table both ways — N3a/N3b/F19) are all live; **this pass:** `update.go`’s `press` is one arm per action, `handledActions` enumerates the set and `HandledActions()` exports a copy, so the keymap⇄dispatch join is now a test (`dispatch_test.go`), not an absence |
| `internal/tui/boundary_test.go` | import scan + A6 scan, with anti-vacuity arms | ✓ EXISTS + SUBSTANTIVE | both scans are live directory scans with arms that fire |
| `internal/cli/failure.go`, `feed.go` | every error class → domain.Failure; the composition root | ✓ EXISTS + SUBSTANTIVE | one limiter, `Budget: limiter.Budget`, a session error becomes a state and no request; `Check: feedReach(client, cfg)` runs the check's unauthenticated stages through that client |
| `internal/render/testdata/TestGoldenFrames/*.golden` | the model corpus at 60/80/120/200, twins where the measures differ | ✓ EXISTS + SUBSTANTIVE | **263 files now** (this phase's 31 + the corpus of phases 3–7, of which the amendment added the 8 likes-hedged and 32 covered-notice frames); the twin rule is asserted in both directions and every frame this phase names is still stored |
| `internal/render/golden_test.go`, `golden_cover_corpus_test.go` | generation + byte comparison via x/exp/golden, pinned env + injected clock | ✓ EXISTS + SUBSTANTIVE | `TestGoldenFrames` compares, `-update` writes, TZ/TMUX/TERM/COLORTERM/NO_COLOR pinned per render; the covered table now carries the four notice states and the likes hedge |
| `internal/render/corpus_test.go` | adversarial-content audit + absences + A1/A2/A3/A7 over stored bytes | ✓ EXISTS + SUBSTANTIVE | reads files only; all four absence arms re-planted against this pass and all four fired (F7a–F7d); the covered-leak scan now reads every covered frame (N2a) |
| `internal/tui/corpus_test.go` | the tui corpus's own covered-vocabulary scan | ✓ EXISTS + SUBSTANTIVE, **new in the amendment** | `TestTheCoveredTUIFramesLeakNoPlainVocabulary` + `…CatchesAPlant`; the corpus's half of the frames had no scanner before, which is the hole the amendment closed (N2b) |
| `internal/render/capture_tmux_test.go`, `capture_pty_test.go` | real-terminal captures | ✓ EXISTS + SUBSTANTIVE (pty leg deferred) | the tmux leg passes on its own per-run socket, **takes the socket file with it** (0 growth over five runs; its own assertion fires when the removal is neutralised, N7b) and leaves no server; the pty leg runs its self-test arm and skips with a reason |

**Artifacts:** 22/22 verified.

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `xsource.Adapter` | `xweb.Client.GraphQL` | operation names resolved in the volatile registry | ✓ WIRED | `volatile.Lookup` returns the typed `ErrUnknownOperation`, never a fallback id |
| `internal/fake` + `xsource.Adapter` | `domain.Source` | both satisfy the port; parity test | ✓ WIRED | compile-time assertion + `TestFakePagesMatchTheFixture` |
| `internal/screen` | the shell | Effect values only | ✓ WIRED | `go list -deps` shows no tea, no http, no source; every event the shell sends is declared, every effect kind handled |
| `internal/text` | the rest of the tree | the only measurement | ✓ WIRED | the A6 scan over tui/screen/render/keys passes, including its `%.*s` precision arm |
| cluster iterator | measurement | both from `charmbracelet/x/ansi@v0.11.9` | ✓ WIRED | `FirstGraphemeCluster` + `Method.StringWidth`/`Truncate`, with the re-measure and prefix guards |
| `internal/render` | `internal/domain`, `internal/text` | imports | ✓ WIRED | `go list -deps` still shows exactly those two module imports, with the amendment's `cover.go`/`saved.go`/`copy.go` included |
| state words | classifier | `state_test.go` pin | ✓ WIRED | `TestTheStateWordsAreTheOnesOnTheLine`, `TestTheStateWordsAreTheClassifiers` |
| `xweb.APIError` | the status line | `Reset` from `x-rate-limit-reset` | ✓ WIRED | the rate-limited frame prints `until 14:32` |
| `internal/cli` | config/session/xweb/xsource/tui | the only composer, one limiter | ✓ WIRED | one `*xweb.Client` and one limiter; no second request path in the tree |
| `internal/tui.Render` | `render.Frame` | direct | ✓ WIRED | `Render` adds no bytes; the live `?` pane matched `help_80.golden`'s rows 1–23 character for character (padding aside) |
| budget field | the live limiter | `Deps.Budget = client.Limiter.Budget` | ✓ WIRED | `TestTheBudgetIsInEveryFrame`; live: `0 requests, 0 rate-limited` on a run that issued none |
| the corpus | `internal/render` | composed by the same functions (`Blocks`/`StateBlock`/`Help`, `StatusFields`, `Frame`) | ✓ WIRED | `golden_test.go` calls `Frame(...)` exactly as the shell does; `Help` takes the same pane row count the shell passes |
| width model | frames and captures | `internal/text` only | ✓ WIRED | the capture harness measures through `text.Width`; corpus A7 measures the same over every stored frame |
| `Model.failureOf` | `netcheck.CheckWith` | `Deps.Check = feedReach(client, cfg)`; run only for an error the mapper cannot name | ✓ WIRED | re-falsified this pass (`Check: nil` → FAIL, F13) |
| `pageMsg.Frame` | `frame.id` | `frameIndexID` in `answered` | ✓ WIRED | an answer whose frame is gone is dropped without marking `read[]`; re-falsified this pass (F11) |
| `frameKinds()` | `frameKindChecks()` | `TestEveryFrameKindIsCheckedUnderACover`, both directions | ✓ WIRED, **new in the amendment** | a declared kind with no row fails, a row for an undeclared kind fails, and deleting the enumerator no longer compiles (N3a, N3b, F19) |
| `keys.Rows()` | `handledActions` | `internal/tui/dispatch_test.go` (`update.go:1294`, the enumeration beside the dispatcher) | ✓ WIRED, **new this pass** | the join runs both ways plus a source sweep over `keys.go`’s declarations, the rows, the enumeration and the shell’s case labels; the seventh pass’s 31st-row plant (B1) now fails (G1) and a handler with no row fails (G2). One measured limit, note 10: for the five actions whose label also appears in a second switch the sweep alone cannot see the main arm deleted (G4), so the behaviour tests are what catch that shape |
| injected clock + pinned env | reproducibility | `corpusNow`, `environment.pin` | ✓ WIRED | stored frames byte-stable across runs; stamps read `2026-10-14` |

**Wiring:** 18/18 connections verified.

### Requirements Coverage

| Requirement | Status | Blocking Issue |
|-------------|--------|----------------|
| FEED-01: Following as the default screen, algorithmic feed only behind an explicit toggle | ✓ SATISFIED | `t` is the only producer of a ForYou frame and `feed` rejects arguments; live this pass |
| FEED-02: load older posts with keyset pagination, viewport never jumps | ✓ SATISFIED | `walk.go` + shell tests; the anchor is a rows-above distance and a late page moves nothing |
| FEED-03: never duplicate a post; detect a stalled cursor instead of looping | ✓ SATISFIED | id-keyed dedupe, `usedCursors`, `stalled`; no counterexample found |
| FEED-04: refresh key preserves selection and scroll, announces "N new posts" | ✓ SATISFIED | `TestRefreshIsDebouncedAndAnnounced` + the refresh-into-empty default (F12) |
| FEED-05: a partial failure still renders what was received with one quiet status line | ✓ SATISFIED | `TestAFailedPageKeepsThePosts`; pane byte-identical after the failure |
| REND-01: borderless, unbranded, no avatars/counts/decorations | ✓ SATISFIED | the four absence checks re-run over all 319 stored frames and each re-planted against this pass (F7a–F7d) |
| REND-02: one width model, correct for CJK/emoji/graphemes | ✓ SATISFIED | the four counterexamples are still fixed and re-falsified (F1–F5); both `$TMUX` arms hold every A4 property |
| REND-03: reflow on resize | ✓ SATISFIED | A5 + `TestResizeRerendersWithoutFetching` + the live reflow of the shipped binary |
| REND-04: colour ladder, NO_COLOR/TERM=dumb, no meaning by colour alone | ✓ SATISFIED | ladder order read in `palette.go`; A1 re-run |
| REND-05: keys, `?` overlay and shell completion from one keymap table | ✓ SATISFIED (substitution) | the third surface is the `keys` subcommand, not a completion script; the one-table discipline holds over 30 rows |
| REND-06: vim-ish navigation with a real back stack restoring list, selection and scroll | ✓ SATISFIED | frame-stack tests; live `q` from the bare pane quit with `EXIT=0` |
| REND-08: golden frames at 80 and 120 columns, inside and outside tmux | ✓ SATISFIED (one leg deferred) | the tmux leg passes on its own per-run socket and now cleans it up; the Ghostty leg is the deferred human check |
| OPS-01: four named failure states with distinct remedies | ✓ SATISFIED | all four states are reachable, the offline one proven through the shipped composition (F13) |
| OPS-02: the request budget in one line | ✓ SATISFIED | `TestTheBudgetIsInEveryFrame`; below ~66 cells the row is cut through it per the UI-SPEC's own rule |
| OPS-04: a soft ban is detected, stops the session, and is not retried | ✓ SATISFIED | `blocked()` + tests |

**Coverage:** 15/15 requirements satisfied.

## Anti-Patterns Found

Four entries the sixth pass carried against exactly the files that amendment touched stay **resolved**, and that is measured, not taken on trust (the arm that pins each is named). This pass closes the one **P2** the seventh pass carried — the keymap has no arm joining it to the model's dispatch — because the guard now exists and its plants fire (G1, G2, G5, G6, G7; the reversed B1 is G1). The seventh pass's P3 notes still stand. No must-have truth failed, and there are no blockers.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/tui/model.go` | 186-198 | ~~`frameKinds()` is dead code and the check its own comment promises does not exist~~ | ✅ **resolved** | the enumerator is now joined to `frameKindChecks()` in both directions by `TestEveryFrameKindIsCheckedUnderACover`: a sixth kind fails (`N3a`), a kind with no row fails (`N3b`), and deleting the function breaks the build (`F19`), where the last pass measured the whole suite green after deleting it. The wording in the amendment's prose that calls this "the whole-surface table" is loose — the join is to a per-kind table, and `screens()` (per-state) remains hand-kept — but the substance is what was asked for |
| `internal/render/copy.go` | 220-227, 384-385 | ~~seven orphaned constants and a dead CI reply pair~~ | ✅ **resolved** | all nine are deleted (0 references to `CopyFact*`, `Placeholder*`, `CoverCINewReply(s)`), `coverNewReplies` documents its single `case CoverServiceLog, CoverCI:` as one vocabulary rather than a duplicate, and an independent scan of `copy.go`'s 250 constants reports **0 with no reader** |
| `internal/render/cover.go` / `profile.go` | 197-211 / 129 | ~~each cover declares a fourth `AccountFacts` clause naming `{joined}`, and no cover stamp sets `MonthYear`~~ | ✅ **resolved** | all four stamps carry a layout; the stored covered profile frames now read `… Commits: 8901  Since: Mar 2011` (N6); blanking one layout fails that style's two frames |
| `internal/render/capture_tmux_test.go` | 352-395 | ~~the per-run socket name is never unlinked (+1 file per run)~~ | ✅ **resolved** | `tmuxKill` waits for the server to stop answering and then removes the socket and its `-stand-in`; measured 0 growth across six runs in that pass, and this pass measures a clean run leaving its private `TMUX_TMPDIR` empty and no server surviving (the default `/tmp/tmux-1000` itself holds **281** files again, none from a run of this pass — see the entry below), and the harness's own assertion fires when the removal is neutralised (`N7b`). One file in the harness's naming scheme appeared in the directory between two measurements during the pass (created 13:08:12 UTC, pid 2809916) — not attributable to any measured run of this tree's harness (all of them show 0 growth, including two after it appeared), so recorded here rather than against the tree |
| `internal/render/status_test.go` | 974-984 | ~~the reply notices were outside the covered-copy totality list~~ | ✅ **resolved** | the list now holds all sixteen notice kinds including `NoticeLoadingReplies`/`NoticeNewReplies`/`NoticeNoNewReplies`, and the comment records the hole this pass's F22 found |
| `internal/render/copy.go` | 227 vs 473 | the same user-visible word stated twice | ℹ️ Info | they still agree (`CopyStateRateLimited = "rate limited"` and the state table's own entry); the pin covers the table, not the constant |
| `internal/screen/walk.go`, `internal/tui/update.go` | `headIDs`/`Effect.Head` vs the refresh | write-only data | ℹ️ Info | re-checked this pass: `Refresh` puts every loaded id in `Effect.Head`, the only non-test reader is `internal/screen/profile.go:168`, which forwards it, and `internal/xsource/home.go:96` (plus `internal/cli/doctor.go:297`) still hardcodes `seenTweetIds: []string{}`, so the refresh diff never reaches the wire; the local diff still works and the dead field costs a reader's attention |
| `internal/cli/keys.go` | 9-11 | comment says "the shipped shell completion scripts read it" | ℹ️ Info | no completion script exists anywhere in the tree (re-checked: `grep -rl completion` finds only the two keys.go comments, no bash/zsh/fish reference); the surface is the `keys` subcommand, the documented CONTEXT substitution |
| `internal/render/help.go` + `internal/keys/keys.go` | 43-62 / 30 rows | the overlay is taller than the pane | ℹ️ resolved earlier | the overlay spends its last row on `~ 8 more keys than the pane can show`, so the eight bindings below the fold at 80×24 are named inside the program; re-verified live this pass |
| `internal/render/testdata/TestGoldenFrames/narrow_79.golden` | — | the name says 79, the bytes say 59 columns | ℹ️ Info | re-measured this pass: 24 rows, every row 59 cells; the name (from the UI-SPEC's list) and the corpus rule "the width the name carries" disagree with the stored bytes; documented in 02-05-SUMMARY as a transposition, and the audit asserts the narrow-pane line's presence, which is why 59 is right for the content |
| `internal/keys/keys.go` + `internal/tui/update.go` | keys.go:1-7 (package doc) / update.go:294-326 (the dispatch) | ~~a keymap row whose action the shell does not dispatch is a documented key that does nothing~~ | ✅ **resolved** | the guard is in the tree now: `press` is one arm per action, `handledActions`/`HandledActions()` name the set, and `dispatch_test.go` joins it to the table in both directions plus a source sweep. Re-measured this pass: the 31st-row plant (the seventh pass's B1) now **fails** — `the keymap binds "nudge" (ctrl+n) and press has no arm for it` and `the enumeration went stale` (G1) — as does a handler with no row (G2), a dropped enumeration entry (G6) and a removed one-label arm (G7). One measured limit, recorded as note 10: for the five actions whose `case` label also appears in a second switch (`ActionCopyLink`, `ActionMedia`, `ActionOpenPost`, `ActionQuit`, `ActionPanic`), deleting the main arm leaves the sweep satisfied — the key's own behaviour tests catch each (G4) |
| tmux capture (measurement note) | — | `capture-pane -p -N` preserves trailing spaces only where its line buffer did | ℹ️ Info | the live `?` overlay's raw capture alternates 40- and 80-cell rows while every character matches `help_80.golden`'s rows 1-23 after trimming; a "byte for byte" claim about a live capture must say "after trailing padding", and the stored frames themselves are uniformly padded. Not a defect in the tree — recorded so a later pass does not read the artifact as a regression |
| `/tmp/tmux-1000` (host, not the tree) | — | the harness’s own socket files in the *default* directory | ℹ️ Info | re-measured this pass: the directory holds **281** files again (the oldest `xterm-golden` dated 10:07 UTC, plus `xterm-golden-*`/`*-stand-in` pairs from 10:08–10:45 and one from 13:08) — i.e. the seventh pass’s “0 files” was true then and is not now, but **none of them is from this pass**: every `TestCaptureTmux` run this pass used a private `TMUX_TMPDIR` (0 sockets left, two concurrent runs both exit 0), and the default directory’s newest entry predates this pass’s first run. An earlier agent’s `tmux -L` runs and other phases’ probes share this name space, so the number is a host measurement, not a tree property; not attributable to this tree’s harness |

**Anti-patterns:** 13 entries (0 blockers, 0 warnings — the seventh pass's single P2 is resolved — 7 resolved, 5 info notes and 1 measured-away). No must-have truth failed.

### Human Verification Required

#### 1. The bare-terminal capture in Ghostty (deferred, blocking in plan 02-05)
**Test:** In an 80-column Ghostty window, in the repository root, run `go test ./internal/render -run TestCaptureInGhostty -count=1 -v`; then look at the frame the harness printed into the window.
**Expected:** the test reports the PTY arm ran (not skipped), the cursor came back on row 24 and column 81 with no wrap, the selected block is findable from its `>` margin alone, the status line is the last row and nothing runs past column 80; then compare against `internal/render/testdata/TestGoldenFrames/timeline_80.golden`.
**Why human:** the arm needs a real terminal emulator to answer the DSR query, and this host has no Wayland session and no Ghostty binary. Everything else in the arm runs here: re-run this pass, the self-test subtest `the frame is checked before it is written` passes and the PTY arm skips with its printed sentence naming `TMUX` and `TERM_PROGRAM` (`capture_pty_test.go:69-71`).

#### 2. Confirm the intended reading of the two render divergences
**Test:** decide whether the UI-SPEC's fenced 80- and 120-column frames are binding over its own measure rule.
**Expected:** either accept the engine's reading (already asserted with cell counts in `TestTheFrameDivergencesFromTheUISpecAreTheMeasuredOnes`) or record an override for plan 02-03's truth 1.
**Why human:** the two documents contradict each other and no deterministic check can pick. (The help overlay's overflow, which an earlier pass raised here as a third item, is decided: the overlay names its cut.)

## Gaps Summary

**No critical gaps found, and no non-critical gap this phase can close is left open.** Every one of the 24 must-haves is adjudicated: 23 verified, and P05.3 (the bare-Ghostty capture leg) deferred to the human check above — neither verified nor a gap. This pass is a verification-gap fix, and the gap it closes is the seventh pass’s **P2** (note 8, now the resolved anti-pattern row): the keymap⇄model dispatch joint. It is closed by structure — `press` is one arm per action and `handledActions`/`HandledActions()` name the set — and by `internal/tui/dispatch_test.go` asserting the join both ways plus a source sweep, and the closure was measured by re-running the very plant that used to pass: the 31st-row plant (G1) now **fails**. **The refactor is behaviour-preserving:** a whole-tree diff against the seventh pass’s pristine copy (`/tmp/xtv2-base`) shows only the three combined case labels split into one arm each with identical handler bodies, the `slices` import, the enumeration and two comments — the gate order and every other arm are byte-identical — and the flows that depend on `press` (the keymap⇄model exhaustion arm, the gated actions under a cover, the prompt’s keys, the panic gate’s keys, the frame pushes/pops) each pass their own named tests and were each broken in turn by a plant (G4, G7). This pass re-ran the arms the seventh pass recorded — the width model (F1–F5), the frame absence checks (F6, F7a–F7d), the help cut (F8), the CLI entry paths (F9, F13, F14), the shell’s answer routing and refresh default (F10–F12), the saved-pane and profile grammars (F16, F18), the `g` branch (F17, F21, F21b), the corpus measurements (F19, F20, F22), the amendment’s ten (N1a–N7b), the audit and the whitelist (A1, A2), the saved scenario (A3), the matcher (C1), the capability release (D1) and the keymap bundle (E1) — and every one still breaks what it names; the one outcome that changed is B1, which the guard replaces with a failing arm (G1). No truth depended on a re-measured line, and no P2/P3 note became a gap.

### Non-Critical Gaps (Can Defer)

1. **"The budget is visible in every frame" is looser than the spec under the cut** — the budget is never *dropped*, but below the width that holds state + remedy + budget the row is cut through it (`… 0 rate~` at 60 cells, live this pass), which is what 02-UI-SPEC.md itself specifies. Recommendation: amend the comment and 02-04-PLAN's sentence to say "never dropped; cut only below the width that holds state, remedy and budget".
2. **REND-05's third surface is the `keys` subcommand, not shell completion** — a documented CONTEXT decision; no completion script exists anywhere in the tree, though `internal/cli/keys.go`'s comment still says one reads the table.
3. **Write-only data** — `Walk.headIDs`/`Effect.Head` have no reader outside the tests (the one non-test consumer forwards them) and `internal/xsource/home.go:96` still hardcodes `seenTweetIds: []string{}`, so the refresh diff never reaches the wire; the local diff still works and the dead field costs a reader's attention.
4. **`narrow_79.golden` is stored at 59 columns** — re-measured; the name (from the UI-SPEC's list) and the corpus rule disagree with the bytes; documented in 02-05-SUMMARY as a transposition and asserted by the audit as the narrow-pane case.
5. **`CopyStateRateLimited` is stated twice** (`copy.go:227` and the state table) — the two still agree and a test pins the table's entry; the constant itself has no pin beyond its use in `corpus_test.go:261`.
6. **The `?` overlay's live capture needs a trailing-padding trim** — a tmux `capture-pane -N` artifact, recorded above so a later pass does not read it as a regression. The program's own rows are uniformly padded.
7. **The amendment's claim 3 names the wrong table** — `frameKinds()` is joined to `frameKindChecks()`, not to `screens()`; substance correct, wording wrong. Recorded so the milestone's narrative is accurate.
8. **~~No arm joins the keymap table to the model's dispatch~~ — resolved this pass.** The seventh pass measured the absence (note 8 / arm B1: a 31st row passed `keys`, `tui` and `cli` and moved only the help goldens). The guard now in the tree closes it: `press` is one arm per action, `handledActions` enumerates the set beside the dispatcher, and `dispatch_test.go` joins it to the table both ways plus a source sweep over `keys.go`'s declarations, its rows, the enumeration and the shell's case labels. Re-run this pass, the same 31st-row plant **fails** (G1), as does a handler with no row (G2); see the resolved anti-pattern row and the guard section above. Kept here, in place, so a later pass sees the record was closed rather than dropped.
9. **The corpus's four absence plants had two strings wrong in this report, not in the tree** — F7b's and F7c's plant strings (above). Recorded so a later pass derives them from `internal/fake/fake.go` and the corpus rather than from this report's prose.
10. **The guard's source sweep reads case labels package-wide, so five actions keep a second label that can cover a deleted arm (measured this pass, P3)** — `ActionCopyLink`, `ActionMedia` and `ActionOpenPost` (in `coverBreaking`) and `ActionQuit` and `ActionPanic` (in the panic gate) each have a `case` label in more than one switch, so deleting the main switch's arm leaves the sweep satisfied and the guard's own two tests green (G4). Every one is still caught by the behaviour tests that press the key (`TestCopyLink`, `TestCopyLinkWLOnlyIsLinkCopied`, the six `TestMedia*`, `TestOpenPost`, `TestTheQuitKeysWriteNothing`, `TestAQuietKeyAndTheTwoExits`, `TestPanicRoundTripIsExact`, `TestPanicSwallowsKeys`), which is why this is a note and not a gap: the guard joins “declared ⇔ handled ⇒ a case label exists”, and for these five a label does still exist. If the join is ever tightened to per-switch arms, G4 is the arm to reproduce the tightening against.

## Recommended Fix Plans

None blocking: no must-have failed. The items above are documentation drift, one cross-phase dead field, a fixture-naming transposition, two wording corrections in this report and the milestone’s narrative, and &#8212; the seventh pass’s one follow-up &#8212; **the keymap⇄dispatch guard, which this pass finds already in the tree**: `press` is one arm per action, `handledActions`/`HandledActions()` name the set, and `internal/tui/dispatch_test.go` joins the two ends both ways (G1–G7 measure it, including the reversal of the seventh pass’s B1). The remaining note 10 (five actions whose case label appears in a second switch) is P3 and is covered by the behaviour tests of each key. Nothing here is a phase-2 fix plan.

## Verification Metadata

**Verification approach:** Goal-backward — ROADMAP success criteria + every PLAN `must_haves` entry, adjudicated against the current repository, a live binary, the arms the seventh pass recorded (re-run), and seven plants of this pass’s own for the keymap-dispatch guard (G1–G7, the first of which is the reversal of the seventh pass’s B1).
**Must-haves source:** ROADMAP.md phase 2's five success criteria + the five PLANs' frontmatter truths.
**Covered-file set:** 189 paths — the seventh pass’s 188 plus the one path this pass added because the guard’s truth now depends on it: **`internal/tui/dispatch_test.go`**, the new file the guard consists of (this report cites its two tests and the arms that break them, and it is the file whose failure is the joint’s whole evidence). `internal/tui/update.go` was already covered by the seventh pass and stays; the digest now also covers it at its post-guard content. Computed as `node ~/.agents/gsd-core/bin/gsd-tools.cjs query verification.fingerprint .planning/phases/02-following-timeline-in-the-pane <189 paths>` \u2192 `v3:sha256:c37bef60587cd232b02de37f7cd404bb087785f7b85638eed79be441fcc43525`.
**Automated checks:** `go build ./...` clean; `go vet ./...` clean; `gofmt -l .` empty; the whole suite with `-count=1 -v` and `XDG_CONFIG_HOME`/`XDG_DATA_HOME` in `/tmp` — **1571 `--- PASS` lines (subtests included), 19 packages `ok` + 3 with no test files = 22, 0 failures, 8 skips** (`TestCaptureInGhostty` the deferred one); the whole suite again inside `unshare -rn` with only `lo` up, `GOPROXY=off`, `GOMODCACHE` at the real module cache and `XDG_*` in `/tmp` — 19 `ok`, 3 with no test files, 0 failures; the phase-specific suites inside that namespace as one run — `internal/text`, `internal/keys`, `internal/tui`, `internal/render`, `internal/cli`, `internal/screen`, `internal/audit` all `ok`; `TestCaptureTmux` PASS on a private `TMUX_TMPDIR` leaving the directory empty, two concurrent capture runs both exit 0, `TestCapturePanic` PASS, `TestTmuxSocketIsPrivateToThisRun` PASS; `x-terminal audit` from the repository root: `10 checks, 0 findings, 893 files inspected`, exit 0, and 1 finding + exit 1 with an unparsable file or a stale operation name planted; the live pane at 80/60/40 columns and the `?` overlay against `help_80.golden` (0 mismatched characters over rows 1–23); `keys` printing 30 lines; 45 falsification arms (38 re-run, 7 new), each quoted in the section below; the corpus re-counted at 263 render + 56 tui frames.
**Human checks required:** 2 (the deferred Ghostty leg, the two-document rendering divergence).
**Total verification time:** ~50 minutes this pass.

### Falsification attempts made (this pass)

Every breakage ran on a fresh copy under `/tmp` (`/tmp/xtv-work`, restored from `/tmp/xtv-base` between arms); the workspace copy was deleted afterwards, the planted `zz_broken_plant.go` was removed, and the repository was never modified — the only file this pass wrote inside it is this report. The failure output quoted is what the test printed. F1–F18 and F21–F22 are the arms the previous report recorded (re-run here on the current tree); F19, F20 and F22 changed outcome because the amendment fixed what they measured; F21b is this pass's own control arm; N1a–N7b are the amendment's own claims, each broken in turn. *(Seventh-pass record, preserved; this pass's own arms — G1–G7 and the pre-guard diff — are the two bullets immediately after this block, and this pass re-ran every arm in the list below.)*

Every breakage ran on a fresh copy under `/tmp` (`/tmp/xtv3-base` as the pristine reference, `/tmp/xtv3-work` restored from it between arms), with `XDG_CONFIG_HOME`/`XDG_DATA_HOME` pointed at `/tmp/xtv3-xdg`; the workspace copies were deleted afterwards, the planted `zz_broken_plant.go` and the scratch probe test were removed with them, and the repository was never modified — the only file this pass wrote inside it is this report. The failure output quoted is what the test printed this pass. **This pass is about the keymap-dispatch guard, so its own seven plants (G1–G7) are the arms the closure of the P2 rests on; the seventh pass’s arms were then re-run and every one still breaks what it names** (F1–F22, N1a–N7b, A1–A3, C1, D1, E1), with the one exception of the arm the guard replaces: B1 now fails (it is G1). Whole-tree scope was checked against `/tmp/xtv2-base` with `diff -r -q`, which is why the behaviour-preservation claim is a diff and not an inference.

- **G1–G7 The keymap-dispatch guard's arms (new this pass)** — the full output of each is in the guard section above. **G1** a 31st row with a fresh action (`ActionNudge`, `ctrl+n`) now FAILS `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` (`the keymap binds "nudge" (ctrl+n) and press has no arm for it: the key is a documented no-op`; `the table holds 31 rows and handledActions names 30 actions`) and `TestTheShellDeclaresOnlyActionsTheKeymapDeclares` (`keys.go declares ActionNudge and handledActions does not name it: the enumeration went stale`; `keys.ActionNudge is no case label in the shell: nothing dispatches it, and handledActions is data without an arm`) — the reversal of the seventh pass's B1, which used to pass everything but the help goldens; **G2** a handler with no row FAILS (`press handles "nudge" and no row of the keymap binds it: the arm is unreachable`; `the table holds 30 rows and handledActions names 31 actions`; `keys.go declares ActionNudge ("nudge") and no row binds it: an action with no key is not an action`); **G3** the planted bogus case label refuses to compile (`undefined: keys.ActionThisPackageDeclaresNoSuchAction`, `update.go:327:12`, from both `go build` and `go vet`); **G5** the sweep reads 29 declarations and fails its own floor (`the sweep read 29 action declarations out of keys.go, fewer than the table's rows: it is not reading the source`); **G6** an action dropped from the enumeration only FAILS (`the keymap binds "help" (?) and press has no arm for it`); **G7** a one-label arm removed FAILS (`keys.ActionSwitchList is no case label in the shell: nothing dispatches it, and handledActions is data without an arm`) and the behaviour tests too; **G4** the guard's one blind spot, measured: for the five actions whose label also lives in a second switch (`ActionCopyLink`, `ActionMedia`, `ActionOpenPost`, `ActionQuit`, `ActionPanic`) the guard's own tests stay green and each is caught by the behaviour tests that press the key (`TestCopyLink`, `TestCopyLinkWLOnlyIsLinkCopied`, the six `TestMedia*`, `TestOpenPost`, `TestTheQuitKeysWriteNothing`, `TestAQuietKeyAndTheTwoExits`, `TestPanicRoundTripIsExact`, `TestPanicSwallowsKeys`).
- **The behaviour-preservation diff (new this pass)** — `diff /tmp/xtv2-base/internal/tui/update.go internal/tui/update.go` is 80 lines and all of them are the three combined case labels split into one arm per action with identical handler bodies, the added `"slices"` import, the `handledActions`/`HandledActions()` block and two comments; `diff -r -q -x .git /tmp/xtv2-base <repo>` reports exactly two paths differing in the whole tree: `internal/tui/update.go` and the new `internal/tui/dispatch_test.go`. The gate preamble (panic → prompt → lookup → help → cover gate → switch) is byte-identical between the two files, and each split arm calls the same handler the combined arm did, with the same `action` value — so no key can have changed target.
- **F1 Pad's absorbed fill** — the naive `s + strings.Repeat(" ", w-Width(s))` fill → `TestA4Agreement` FAIL: `Width(Pad(s, 5)) = 4, want exactly 5 … (s = "a۝", Pad = "a۝    ")` (re-measured this pass, same output).
- **F2 The unterminated escape's drop** — `default:` returning the filled string → FAIL: `Width(Pad(s, 3)) = 2, want exactly 3 … (s = "ab\x1b[", Pad = "ab\x1b[ ")`, in both `$TMUX` arms (re-measured).
- **F3 Wrap's content test** — `flush`'s gate `cur.Len() == 0` → `curW == 0` → FAIL: `Wrap(s, 8) does not carry the non-space characters of s in order: got "", want "́" (s = "́")` (re-measured).
- **F4 Truncate's re-measure dropped** — the guard reduced to the prefix check → `TestA4Agreement/tmux_pane/ZWJ_family` FAIL: `Width(Truncate(s, 2)) = 6, wider than the measure (s = "👨‍👩‍👧", Truncate = "👨‍👩‍👧")` (re-measured; the same arm also trips the Pad assertions on the same string). **A note on the recipe:** the plant must target the condition `Width(cut) <= w && strings.HasPrefix(s, cut)` — an earlier attempt of mine matched `if Width(cut)…`, which that line does not start with, and the substitution silently did nothing, so the arm "passed" without breaking anything. Re-applied to the real condition it fires; recorded because a silent no-op plant is exactly how an absence claim goes untested.
- **F5 Truncate's prefix check dropped** — `strings.HasPrefix(s, cut)` → `strings.HasPrefix(cut, cut)` → FAIL: `Truncate(s, 1) = "a\x1b[31m", which is not a prefix of s = "abcdef\x1b[31m"` (re-measured).
- **F6 The stored long-detail frame** — `StateBlock`'s body printed verbatim → `TestGoldenFrames/longdetail_80` FAIL (byte mismatch on the wrapped transport sentence) (re-measured).
- **F7a–F7d The four absence arms, one in-place plant each in a stored frame** (all four re-planted and re-measured this pass, each in a frame a leak could hide in, each keeping its 24 rows and its 80 cells) — `3813` into `timeline_80` row 2 → `TestCorpusAudit` FAIL: `the stored frame timeline_80 prints 3813, an engagement count its own posts carry`; `https://pbs.twimg.com/media/a.jpg` into `media_opened_80` row 3 → FAIL: `prints the media location "https://pbs.twimg.com/media/a.jpg"` (the sixth pass's line wrote `https://example.invalid/a.jpg`, a URL no corpus state carries — the plant only fires with a URL the frame's own media carry, and this pass used the one the frame holds); `name of a_15_cell_handl` into `timeline_80` row 3 → FAIL: `prints the display name "name of a_15_cell_handl"` (the sixth pass's line wrote `name of a_15_cell_handle`, which is not the handle the corpus holds: `internal/fake/fake.go:307` builds `a_15_cell_handl`, a 15-cell handle, and `fake.go:386` names it `"name of " + handle`); `â` into row 5 → FAIL: `row 5 of the stored frame timeline_80 holds "â" (U+00E2): a frame carries no avatar, icon or emoji of its own …`. An appended line is rejected earlier by the loader, which is itself a check.
- **F8 The help overlay's cut** — `Help` back to silently clipping (`return out[:pane]`, with `_ = cut` to compile) → `TestGoldenFrames/help_80` FAIL, `TestGoldenFrames/help_120` FAIL and `TestHelpNamesTheRowsThePaneCannotHold` FAIL: `the last row is " 3               cover style 3", want it to carry "8 more keys than the pane can show" for 8 rows that do not fit` (re-measured).
- **F9 The bare invocation** — `return Feed(ctx, nil)` → the usage path → `TestABareInvocationOpensTheFeed` FAIL: `a bare invocation exited 2, want 0: it opens the feed rather than printing the usage` (re-measured).
- **F10 The stale failure** — deleting `m.failure = domain.Failure{}` from `answered` → `TestASuccessfulFetchClearsTheState` FAIL: `the status line still says "session expired" after a fetch succeeded: "session expired  copy the cookie again  following  0 requests, 0 rate-limited   "` (re-measured).
- **F11 The toggle's answer routing** — `frameIndexID(msg.Frame)` → `frameIndexID(m.top().ID)` (planted at every occurrence this pass) → `TestADiscardedFramesAnswerIsDropped` FAIL: `the discarded frame's answer loaded 9 posts into another frame, want none` / `a dropped answer marked the live frame read` (re-measured).
- **F12 The refresh default** — deleting the second `w.defaultSelection()` (`Walk.refresh`) → `TestARefreshIntoAnEmptyListSelectsTheFirstPost` FAIL: `the refresh filled an empty list and selected "", want the first post "new-0"` (re-measured).
- **F13 The offline state** — `Check: feedReach(client, cfg)` → `Check: nil` in `feed.go` → `TestTheFeedClassifiesAReachabilityFailure` FAIL: `the status line is "unclassified  record this shape first  following  1 request, 0 rate-limited", want it to carry "no route to x.com"` and `the reachability check ran 0 times for one unnameable failure, want once` (re-measured).
- **F14 The CLI table** — the `feed` row renamed to `feeds` → `TestEveryVerbBodyIsReachableThroughCommands` FAIL: `commands() has no "feed" row: … Feed is exported, implemented and unreachable from Main` and `commands() names "feeds", which is not a verb body in this package` (re-measured).
- **F15 The tmux socket** — re-measured this pass: `TestCaptureTmux` on a private `TMUX_TMPDIR` exits 0 and leaves the directory **empty**, and two concurrent `go test ./internal/render -run 'TestCaptureTmux|TestCapturePanic'` runs **both exited 0** with no leftover. The harness is clean: on a private `TMUX_TMPDIR` it exits 0 leaving **0** sockets, and two concurrent runs both exit 0. The default `/tmp/tmux-1000` holds **281** files again this pass (none from a run of this pass — its newest entry is from 13:08 UTC, before this pass began — so the seventh pass’s “0” was a moment rather than a property of the tree; recorded as an info note above).
- **F16 B1's own claim** — `savedPane` handing `render.Plain` to `CapabilityBlock`/`StateBlockWithMirror`/`SavedEmptyState`/`render.StateBlock` → `TestTheCoveredSavedPaneNamesNothingOfTheProduct` FAIL, naming the leak on every style and width: `the covered servicelog frame holds "doctor", a word the cover did not author (state "the list's operation is missing")` (re-measured).
- **F17 B2's own claim** — the `ActionFirst` branch neutralised → `TestGoRestoresTheProfileHeader` FAIL on all ten subtests: `` `g` left the frame moved: the pane is still the anchor's and not the frame's own first row `` (re-measured).
- **F18 B3's own claim** — `clauses := g.AccountFacts` → `clauses := Plain.AccountFacts` (`internal/render/profile.go:79`) → `TestGoldenFrames` (tui) FAIL on `profile_{servicelog,gitlog,ci,internal}_{80,120}`, each printing the plain reader's clauses in a covered frame (re-measured).
- **F19 `frameKinds()` is no longer dead (outcome changed)** — re-measured: deleting the function does not compile, `internal/tui/cover_e2e_test.go:732:14: undefined: frameKinds`.
- **F20 The nine dead copy constants are gone (outcome changed)** — re-measured: 0 references to all nine (`CopyFact{Followers,Following,Posts,Joined}`, `Placeholder{Followers,Following,Posts}`, `CoverCINewReply`, `CoverCINewReplies` across every `.go` file in the tree) and an independent scan of `copy.go`'s 250 constants (declaration line + every reference outside it) reports **0 with no reader**. (A first scan of mine reported 75 orphans because it counted a reference *in copy.go itself* as a reader and then excluded the file — the corrected scan counts the declaration line and nothing else. Recorded because the corrected number is the one the claim rests on.)
- **F21 My own probe of the `ActionFirst` branch on this phase's surface** — re-run this pass as a scratch test in the `/tmp` copy: a plain 80×24 timeline six moves down, then `g` → `ok` (the frame unmoved, the selection the first loaded post, no request spent, the pane byte-identical to a fresh frame's pane).
- **F21b The same probe against the neutralised branch** — FAIL: `the frame is still marked moved after g: anchor=0 moved=true`, so the probe is a test and not a tautology.
- **F22 The covered reply vocabulary (outcome changed)** — re-measured: `CoverCINewReply`/`CoverCINewReplies` are gone from the tree (0 references), the single case is documented in `coverNewReplies` (`copy.go:682`), and `TestTheCoveredCopyIsTotal`'s list (`internal/render/status_test.go:982-984`) now holds `NoticeLoadingReplies`, `NoticeNewReplies` and `NoticeNoNewReplies`, its comment crediting this report's F22 for the hole.
- **N1a The cover's hedge** — the covered likes branch given the body-claiming answer back → `TestTheCoveredLikesHedgeIsInTheCorpus` FAIL: `the covered likes-empty frame cover_servicelog_likes_hedged_80 is 2 rows, want one: a second row resolves what the line leaves open` (5 frames named; re-measured).
- **N1b The hedge's own words** — `CopyCoverEmptyHedged` set to the plain sentence → FAIL: `does not carry the cover's own hedge "no likes, or the list is no longer served"` (re-measured).
- **N2a The covered-vocabulary scan over a conversation frame** — `saved.db doctor --pin mirror` planted in place in `cover_gitlog_thread_80` → `TestCoveredFramesLeakNoRepairWords` FAIL: `the covered frame cover_gitlog_thread_80 holds "mirror", a word the cover did not author` (and `saved.db`; re-measured).
- **N2b The tui corpus's own scanner** — the same plant in `search_people_gitlog_80` → `TestTheCoveredTUIFramesLeakNoPlainVocabulary` FAIL: `the covered tui frame search_people_gitlog_80 holds "mirror", a string the cover of gitlog did not author` (re-measured).
- **N3a A sixth frame kind** — `frameSixth` added to the consts and to `frameKinds()` → `TestEveryFrameKindIsCheckedUnderACover` FAIL: `the shell declares the frame kind 5 and the covered check table has 0 rows for it` and `the covered check table has 5 rows and the shell declares 6 frame kinds` (re-measured).
- **N3b A row for an undeclared kind** — the table's `frameSaved` row given `kind: frameHome` instead → FAIL in both directions: `the shell declares the frame kind 0 and the covered check table has 2 rows for it` / `the shell declares the frame kind 4 and the covered check table has 0 rows for it` (re-measured).
- **N4a A covered notice arm reverted** — `NoticeSyncStalled` returning the plain constant → `TestTheCoveredNoticesAreOnScreen` FAIL on 8 frames: `the covered frame cover_servicelog_stalled_notice_120 does not carry "sync stopped (cursor stalled)", its own style's words for the notice` / `prints the plain reader's own notice` (re-measured).
- **N4b A covered notice frame deleted** — removing `cover_gitlog_stalled_notice_120.golden` → the corpus load fails by name: `reading testdata/TestGoldenFrames/cover_gitlog_stalled_notice_120.golden: no such file or directory` (re-measured).
- **N4c A notice kind with no frame** — the `sync_notice` row's base changed to another kind → `TestTheCoveredNoticesAreOnScreen` FAIL: `the covered frame cover_servicelog_sync_notice_120 does not carry "read complete", its own style's words for the notice` (re-measured). **Correction to the sixth pass's line:** this arm produced the *word* mismatch, not the totality arm's sentence (`no covered frame in the corpus carries a read in progress`); the totality loop is still reached over the corpus, but the message this plant actually prints is the one above. (The totality arm itself is a live `seen`/`found` check at `corpus_test.go:2292-2297`.)
- **N5 `profileAccount` before the header** — the covered branch returning the opened handle → `TestTheCoveredProfileNamesOneAccount` FAIL in all four styles: `the covered profile frame names an account before its header has answered: "account: pump-sign-57  0 requests, 0 rate-limited"` (re-measured).
- **N6 A cover's `MonthYear` blanked** — `coverGitStamp`'s layout removed → `TestGoldenFrames` (tui) FAIL on `profile_gitlog_80`/`120` with the expected frame losing its fourth clause (re-measured).
- **N7b The harness's socket removal neutralised** — `removeTmuxSocket(t, tmuxSocket)` → `_ = tmuxSocket` → `TestCaptureTmux` FAIL with the harness's own assertion: `the harness left its socket file behind at /tmp/xtv2-tmuxcap/tmux-1000/xterm-golden-3118886-…: a later run's client would find a socket no server answers on`, and the file is left in the directory (re-measured).
- **The live probes** — the shipped binary, built from this tree, run in a real tmux pane (3.7c) inside `unshare -rn` with only `lo` up and a synthetic (absent) session, `XDG_CONFIG_HOME`/`XDG_DATA_HOME`/`TMUX_TMPDIR` in `/tmp`: the bare pane opened on Following at 80, 60 and 40 columns, each **24 rows**, no row wider than the pane, and the status line the last row — 80: `no session file  write the session file  following  0 requests, 0 rate-limited`; 60: `no session file  write the session file  0 requests, 0 rate~`; 40: the too-narrow line `widen the pane to at least 60 columns` replaces the list and the status row is cut at `write the session file~`. The `?` pane at 80×24 matched `help_80.golden`'s rows **1–23 with 0 mismatched characters** after trailing padding is trimmed (rows 1–22 the bindings, row 23 `~ 8 more keys than the pane can show`), with row 24 the live state's own status line (the golden's state is the rate-limited one). `q` quit the pane with `EXIT=0`; `feed nope` printed `feed: unknown argument nope` and the usage, exit 2; `keys` printed **30** tab-separated lines (`j, down  next  next post`). **Corrected wording:** the sixth pass wrote "every row its own width (80/60/40 cells)" — `capture-pane` reports only the rows the renderer actually wrote (blank rows come back empty, and a written row's trailing blanks only with `-N`), so the measured statement is "24 rows, nothing past the pane's width, the status line last"; the *frame*'s uniform padding is what the stored goldens and `TestFrameIsExactlyThePanesSize` hold.
- **The audit and its checks** — re-measured this pass: `x-terminal audit` from the repository root prints `10 checks, 0 findings, 893 files inspected` and exits 0. With an unparsable Go file planted in a `/tmp` copy the first check reports `internal/render/zz_broken_plant.go:3: cannot parse: expected ')', found '{'` (100 files in that check where the clean tree has 99) and the run exits 1.
- **A1 The `opnames` check's own plant (new this pass)** — `ListLatestTweetsTimeline` put back into `opClass` → the audit prints `internal/audit/readonly.go: the table classifies "ListLatestTweetsTimeline" as a read and the registry does not hold it: an entry for an operation this build cannot resolve`, **exit 1**, and the package test `TestOpNamesPassesOnTheRealTableAndSnapshot` fails with the same finding.
- **A2 The whitelist predicate (new this pass)** — `savedMirrorError` taught that a `*xsource.ParseError` is also the file → `TestAShapeDriftedAnswerIsNotAFactAboutTheFile` FAIL: `a shape-drifted answer (…) was read as a failure of the mirror: the pane would tell the reader to move a healthy saved.db aside`, the status line rendering `mirror unreadable  move saved.db aside, then sync` where the shape's own state belongs. So the whitelist is what makes the frame tell the truth about *which* failure it is, and the blacklist cannot come back unnoticed.
- **A3 The changed saved scenario's anti-vacuity arm (new this pass)** — the failing read removed from the shared corpus's `CapMirrorUnreadable` leg (so the mirror answers while the frame claims it cannot be read) → `TestTheCoveredSavedPaneNamesNothingOfTheProduct` FAIL: `the plain frame of "the store cannot be read" does not carry "move saved.db aside": the covered absence below would be vacuous`. The scenario now drives a state the shipped path can produce, and it says so when it stops doing that.
- **B1 The keymap-exhaustion plant (new this pass — the seam this phase's keymap truth rests on)** — a 31st row added to `keys.rows` (`ActionNudge`, `ctrl+n`), with `TestTheTableIsTheUISpecsRows`' pin bumped to 31 so the count is not what is measuring: `go test ./internal/keys ./internal/tui ./internal/cli` **all pass** — the model's dispatch (`internal/tui/update.go:264-307`) ends in `return m, nil`, so an action the shell does not handle is a silent no-op, and the key is still listed by the `?` overlay and by the `keys` subcommand. The only check that fires is the help-overlay golden (`TestGoldenFrames/help_{80,120}` and the six `cover_*_help_*` frames, all regenerable with `-update`). **So there is no keymap⇄model exhaustion arm in the tree**: nothing joins "every action the table declares" to "every action the model dispatches". Recorded as a note below, not a failed must-have — the phase's truth (P04.2) is that the three *surfaces* read one table, and that holds.
- **C1 The matcher's anti-vacuity arm, broken (new this pass)** — `ByKeystroke`'s fallback `return "", false` → `return ActionNext, true` → `TestTheMatcherBindsNothingElse` FAIL: `the matcher answers "next" for "z", which is not a row of the table` (and for `""`, `ctrl+z`, `shift+g`, `j down`) — so the round-trip arm beside it cannot pass for the wrong reason. `internal/tui`, `internal/cli` and `internal/render` stay green under this plant (they do not use `ByKeystroke`), which is also the measurement behind the fix round's comment claim.
- **D1 The capability release, removed (new this pass — the round's claim about the saved surface, measured on the screen package that owns it)** — `Saved.Update`'s `MirrorPage` release dropped → `TestSavedCapabilityGoesWhenTheFramesOwnActionSucceeds` FAIL: `capability after a page arrived = mirror unreadable, want CapNone: the read succeeded, so the file state it refutes is over`.
- **E1 The keymap arms on the shipped tree (new this pass)** — `TestTheTableIsTheUISpecsRows` (30 rows), `TestEveryKeyRoundTripsThroughTheMatcher`, `TestNoKeystrokeIsBoundTwice`, `TestEveryRowIsWellFormed`, `TestARowsKeysAreJoinedTheOneWay`, `TestKeysPrintsTheWholeTable`, `TestTheKeymapMatchesWhatTheDecoderReports` (the seam between the table and Bubble Tea's own spellings), `TestPanicSwallowsKeys` (all 28 other rows driven through the model under panic, with its floor of 18) and `TestTheHelpOverlayIsTheOneTable` all pass; the `t` row is still `Screens: ["home"]` (`keys.go:154`).

### What could not be verified, and what would be needed

- A live Following timeline through the shipped binary: needs a real session cookie and network (both withheld by the rules). The live evidence is the pane's failure path; the success path is carried by the corpus and the model-level tests.
- The bare-Ghostty capture leg: needs an 80-column Ghostty window on a Wayland session (see Human Verification).
- A real soft ban, a real 429 with a reset header, and a real stalled cursor: the gates and reducers are proven by tests; the wire facts are not reproducible here.
- The hand-built adversarial width corpus an earlier pass built in a throwaway copy was not rebuilt; this pass's width evidence is the in-tree 23-entry corpus and A7 over the stored frames, plus the five width arms.
- The amendment's *history*: which round introduced the nine dead constants or the socket-file leak cannot be established from the tree (it carries no snapshot of the pre-fix state), so the anti-pattern table records the state of the tree and the arm that now pins each fix, not a culprit. Nor can the fix round's history be recovered: the tree carries no pre-fix snapshot, so each of its claims was re-derived from the code and falsified rather than diffed.
- The tui corpus's exact count of covered frames as the scan sees them (24 by style naming, which is the scan's own floor) — the scan asserts it covered every frame it partitioned rather than printing the number, so the number here is derived from the file names, not from the test's output.
- Whether any *other* phase's surface regressed: out of scope for this pass, which read only this phase's files and the shared corpora. The neighbouring packages are green in both suite runs and `internal/audit`'s ten checks are clean, but each phase's own report owns its own surface — including the two files the fix round moved that this phase does not touch (`internal/screen/saved.go`, `internal/tui/saved.go`, whose behaviour the round's own arm D1 of this pass measures but whose phase owns the truth).
- The keymap⇄model seam is **no longer** an unverified item: it is covered by `internal/tui/dispatch_test.go` in the tree, and this pass broke it seven ways (G1–G7). What remains unverified there is only the P3 limit note 10 records (five actions whose `case` label also appears in a second switch, so the sweep alone cannot see their main arm deleted; the key’s own behaviour test does). No human decision is needed for the P2 itself, which is closed.

### Files touched

Only this report. Every experiment ran on copies under `/tmp` (`/tmp/xtv3-base` as the pristine reference — a `tar` copy of the repository excluding `.git` — and `/tmp/xtv3-work`, restored from it between arms; the seventh pass’s own pristine copy `/tmp/xtv2-base` was used read-only for the pre-guard diff and the old golden bytes, and was not modified). The binaries were built at `/tmp/xtv3-bin/x-terminal`; the capture text files at `/tmp/xtv3-live/*.txt`; the private `TMUX_TMPDIR`s at `/tmp/xtv3-tmuxcap`, `/tmp/xtv3-tmuxcap2`, `/tmp/xtv3-concur/*`; `XDG_CONFIG_HOME`/`XDG_DATA_HOME` at `/tmp/xtv3-xdg`; `HOME` left real so the module cache stayed usable. The planted `zz_broken_plant.go`, the scratch probe `zz_probe_test.go` and the planted golden bytes were written into `/tmp` copies only and removed with them (the golden corpus was restored from `/tmp/xtv3-goldens-backup` byte for byte after the four absence plants). The workspace copies were deleted afterwards. The repository was never written to except this report: the real `~/.config/x-terminal/session` was never read, printed or removed (every run pointed `XDG_CONFIG_HOME` at `/tmp`), and the operator’s real `~/.local/share/x-terminal/saved.db` was never opened.

---

*Verified: 2026-10-15T07:45:00Z*
*Verifier: gsd-verifier (subagent), goal-backward against the roadmap criteria and the plans, on the frozen tree after the keymap-dispatch guard; eighth pass*
