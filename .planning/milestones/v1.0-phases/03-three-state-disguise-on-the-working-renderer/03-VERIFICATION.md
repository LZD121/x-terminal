---
phase: 03-three-state-disguise-on-the-working-renderer
verified: 2026-10-15T07:47:08Z
status: passed
score: 19/19 must-haves verified
re_verification:
  previous_status: passed
  previous_score: 19/19
  gaps_closed: []
  gaps_remaining: []
  regressions: []
covered_files: # 244 paths (see 'Covered-file set' below)
  - .planning/phases/03-three-state-disguise-on-the-working-renderer/03-01-PLAN.md
  - .planning/phases/03-three-state-disguise-on-the-working-renderer/03-01-SUMMARY.md
  - .planning/phases/03-three-state-disguise-on-the-working-renderer/03-02-PLAN.md
  - .planning/phases/03-three-state-disguise-on-the-working-renderer/03-02-SUMMARY.md
  - .planning/phases/03-three-state-disguise-on-the-working-renderer/03-03-PLAN.md
  - .planning/phases/03-three-state-disguise-on-the-working-renderer/03-03-SUMMARY.md
  - .planning/phases/03-three-state-disguise-on-the-working-renderer/03-CONTEXT.md
  - .planning/phases/03-three-state-disguise-on-the-working-renderer/03-UI-SPEC.md
  - internal/cli/feed.go
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/keys/keys.go
  - internal/keys/keys_test.go
  - internal/render/alias.go
  - internal/render/alias_test.go
  - internal/render/block.go
  - internal/render/block_test.go
  - internal/render/capture_panic_test.go
  - internal/render/capture_pty_test.go
  - internal/render/capture_tmux_test.go
  - internal/render/copy.go
  - internal/render/corpus_test.go
  - internal/render/cover.go
  - internal/render/cover_test.go
  - internal/render/frame.go
  - internal/render/golden_cover_corpus_test.go
  - internal/render/golden_test.go
  - internal/render/grammar.go
  - internal/render/help.go
  - internal/render/help_test.go
  - internal/render/panic.go
  - internal/render/panic_test.go
  - internal/render/profile.go
  - internal/render/prompt_test.go
  - internal/render/saved.go
  - internal/render/saved_test.go
  - internal/render/status.go
  - internal/render/status_test.go
  - internal/render/testdata/TestGoldenFrames/cover_ci_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_help_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_help_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_media_detail_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_media_detail_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_empty_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_failed_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_failed_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_unreadable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_wrong_account_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_saved_wrong_account_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_help_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_help_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_media_detail_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_media_detail_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_offline_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_offline_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_empty_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_failed_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_failed_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_unreadable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_wrong_account_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_saved_wrong_account_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_help_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_help_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_media_detail_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_media_detail_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_empty_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_failed_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_failed_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_unreadable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_wrong_account_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_saved_wrong_account_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_bookmarks_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_bookmarks_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_bookmarks_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_bookmarks_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_help_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_help_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_media_detail_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_media_detail_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_empty_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_empty_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_failed_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_failed_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_unavailable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_unavailable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_unreadable_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_unreadable_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_wrong_account_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_saved_wrong_account_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_thread_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_thread_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_ci_120.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_ci_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_ci_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_ci_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_gitlog_120.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_gitlog_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_gitlog_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_gitlog_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_internal_120.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_internal_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_internal_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_internal_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_servicelog_120.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_servicelog_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_servicelog_80.golden
  - internal/render/testdata/TestGoldenFrames/media_gated_servicelog_80_tmux.golden
  - internal/render/testdata/TestGoldenFrames/panic_120.golden
  - internal/render/testdata/TestGoldenFrames/panic_80.golden
  - internal/render/testdata/cover-alias-pins.txt
  - internal/term/term.go
  - internal/term/term_test.go
  - internal/term/tmux.go
  - internal/tui/boundary_test.go
  - internal/tui/cover_e2e_test.go
  - internal/tui/model.go
  - internal/tui/profile_test.go
  - internal/tui/state_test.go
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
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_ci_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_gitlog_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_internal_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_likes_hedged_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_sync_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_sync_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_absent_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_absent_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_complete_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_complete_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_stalled_notice_120.golden
  - internal/render/testdata/TestGoldenFrames/cover_servicelog_stalled_notice_120_tmux.golden
  - internal/render/testdata/TestGoldenFrames/likes_hedged_80.golden
  - internal/render/testdata/TestGoldenFrames/likes_hedged_120.golden
  - internal/tui/corpus_test.go
  - internal/tui/dispatch_test.go
  - internal/screen/saved.go
  - internal/screen/saved_test.go
covered_digest: "v3:sha256:44b866f8eb619ebec6d530987c637e18d4c388d1dd8d7c2fd04233bf60630448"
behavior_unverified: 0
---

# Phase 3: Three-State Disguise on the Working Renderer Verification Report

**Phase Goal:** The user can wear the feed as work-software output or replace the pane with a one-key panic frame — identities aliased, nothing on screen naming the real product — and step back without losing their place.
**Verified:** 2026-10-15T07:47:08Z
**Status:** passed
**Re-verification:** Yes — the pre-archive pass against the frozen tree, after the **keymap-dispatch guard** (the one change since the last pass). The whole-tree diff against the previous pass's own byte-copy of the pre-change tree (`/tmp/xtv5`) is exactly two paths: `internal/tui/update.go` restructured and `internal/tui/dispatch_test.go` added — nothing else in the source tree moved, and **neither `testdata` tree differs** (`diff -rq` rc=0 both). Every falsification the last report recorded was re-run in a fresh scratch copy (**33 of 33 caught**), the guard's own plants were added (**7 of 7 meaningful ones caught**, 2 passing plants recorded as notes), and the refactor was proved behaviour-preserving **two independent ways**: a differential sweep of every bound keystroke through every dispatcher state, old `press` vs new `press`, **1558 pairs byte-identical**, and the shipped binary driven under tmux side by side with the pre-refactor binary, every capture byte-identical. Nineteen of nineteen truths hold. What remains is recorded in *Accepted tech debt* — nothing in it is a must-have failure.

## The dispatcher guard, adjudicated against the behavioural truths

The change is small and its own diff is the whole of it (established without git: `/tmp/xtv5` is this repository's byte-copy from the previous pass, still holding the old `press`): the four combined case labels in `press` (`ActionBookmarks, ActionLikes`; `ActionCover1..ActionCover4`; the eight movement actions) are split into **one arm per action**, each arm calling the same handler with the same `action`; `handledActions` (30 entries, `update.go:1297-1309`) enumerates the dispatched set beside the switch, `HandledActions()` (`update.go:1315`) exports a `slices.Clone` of it, and `dispatch_test.go` (new) joins the enumeration to the keymap both ways plus a source sweep. **No gate, no handler body, no key and no frame was touched** — the diff's only non-labelling hunks are the new `slices` import, one comment and the two new declarations.

1. **The restructure is behaviour-preserving, proved by a differential sweep.** A scratch test (`/tmp/xtv6`, a copy) builds a shell in every dispatcher state — plain; each of the four covers; help open (plain and covered); the prompt open, empty and seeded; panic entered from plain and from a cover; a thread frame; a search-results frame; a people-results frame; a saved frame; and one 120×36 wide case — then presses **every keystroke the keymap binds** plus six unbound ones (`z`, `0`, `9`, `ctrl+x`, `Z`, `Q`) and records the resulting state: `state`, `style`, `help`, the prompt value, frame count and the top frame's id/kind/Moved/Anchor/Read, the notice pair, the failure kind, the counted pane rows, a hash of the rendered frame, the status line, and the terminal sink's and persist writer's logs. The same file was dropped into the pre-refactor tree and this one: **1558 (state, key) pairs, and `cmp` reports the two outputs byte-identical** (1559 lines each). That covers the keymap⇄model exhaustion arm (unbound and screen-excluded keys are no-ops in both), the gated actions under each cover, the prompt's keys, the help shadow and the panic gate as *values*, not as prose.
2. **The gated cover-breaking keys behave exactly as before, at the model and in the shipped binary.** `TestCoverBreakingActionsGated` passes with its full 12 subtests (four covers × `i`/`y`/`o`): the gated key returns no command, the renderer's and clipboard's call counts stay 0, only the status row changes, the notice is `NoticeGated` and the status carries the style's own refusal literal (`open denied by local policy`, `operation refused`, `step skipped: policy`, `request denied`). On the shipped path this pass drove the real binary under tmux in a private namespace: under `gitlog` the `y` row reads `proxy refused  start the proxy  operation refused  branch=main  1 request, 0 rate-limited`, under `servicelog` `i` reads `open denied by local policy  stream=in`, and the **pre-refactor binary's captures are byte-identical to the new one's** (`cmp` over all seven captures).
3. **The panic gate behaves exactly as before.** `TestPanicSwallowsKeys` (every bound key swallowed, frame unchanged, no request, help not opened, selection unmoved) and `TestPanicIgnoresCoverBreakingKeys` pass; so does `TestPanicEntersInOneFrameAndLeavesExactly`. In the shipped runs, `C-p` from the covered frame painted the constant frame (`ok corp/…` rows), the title and window name became `make`, six keys (`m`, `q`, `1`, `i`, `z`, `?`) left the capture **byte-identical**, and `C-p` restored the covered frame byte-for-byte — the post-`C-p` capture is `cmp`-identical to the one taken before the panic, window name `git` and the gated notice included.
4. **The prompt's keys and the frame pushes/pops are unchanged.** The differential sweep's `prompt_normal`, `prompt_seeded`, `search_results`, `people_results`, `thread`, `saved_bookmarks` and their covered variants all match the old tree byte for byte, and `TestStateTransitions` (16 subtests, T1–T13), `TestPanicRoundTripIsExact`, `TestTheWholeSurfaceUnderACover` and `TestTheBackStackRestoresTheFrame` are green.
5. **The guard's own plants fail, and the join is a real one.** Plant **G1** (a 31st keymap row with a fresh action) fails the join at `dispatch_test.go:63: the keymap binds "zz-plant" (z) and press has no arm for it` and the sweep at `dispatch_test.go:128`/`:159`; plant **G1b** (an action declared and bound by nothing) fails the sweep at `:125`; plant **G1d**/**G1f** (an action declared outside the const block that the shell does name) fails at `:147: the shell names keys.ActionZzOutside, which keys.go does not declare`; plant **G2** (a handler and an enumeration entry with no row) fails the join at `dispatch_test.go:72: press handles "zz-ghost" and no row of the keymap binds it`; plant **G3** (a case label naming an undeclared action) fails the build at `internal/tui/update.go:268:12: undefined: keys.ActionThisPackageDeclaresNoSuchAction` — the compiler arm the sweep's comment claims. Two plants pass and are recorded: **G1c/G1e** (an action declared in a shape the sweep's regex does not read, named nowhere) — inert, nothing dispatches it — and **G4** (the `ActionCopyLink` arm's *body* replaced by a no-op, case label intact): both guard tests stay green and the behavioural tests fail instead (`tui_test.go:2062: y returned nil command`), which is the honest boundary of what a structural join can prove (tech debt 9).

## What was run

All repository access was read-only, and no git command was run in the repository. The change under adjudication was established by whole-tree comparison, not by git: `/tmp/xtv5` is the previous pass's own byte-copy of this repository, and `diff -rq` against it reports exactly `internal/tui/update.go` (changed) and `internal/tui/dispatch_test.go` (added) in the source tree, with `.planning` excluded. Plants, probes and the differential sweep were all run outside the repository: a pristine base at `/tmp/xtv7/base` (a byte-copy with `.git` removed) and two throwaway copies of it, `/tmp/xtv6` (the sweep) and `/tmp/xtv7/work` (re-created per plant). The repository therefore ends byte-identical to how this pass found it except for this report file — `diff -rq` against the base reports no source or `testdata` difference at all, and `git status` shows only report files modified and nothing untracked under `internal/`. Two process notes carried over: the plant base must not be nested inside itself (a first `cp -a` into an existing copy inflated the audit's file count 893 → 1784 and was discarded before any outcome was recorded), and the shipped-run comparisons need `C-p` spelled as tmux's own key name — a literal `"ctrl+p"` is eight bytes of text and drives nothing.

| Check | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0, no output |
| `gofmt -l internal cmd` | empty |
| `go test ./... -count=1` | **1573 tests passed, 0 failures**, 22 packages (19 `ok`, 3 with no test files) — was 1571 last pass, **+2 from `dispatch_test.go`'s two tests** |
| `unshare -rn` (`lo` up, verified with `ip addr show lo`) + `go test ./... -count=1` | exit 0 — the suite is still network-free |
| `unshare -rn` (`lo` up) + the phase suites (`./internal/render/... ./internal/tui/... ./internal/keys/... ./internal/config/... ./internal/term/...`) | all `ok`, exit 0 |
| `go test ./internal/tui -run TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` and `…TestTheShellDeclaresOnlyActionsTheKeymapDeclares` | PASS — the guard, both directions and the source sweep, on the frozen tree |
| the **differential sweep** (`/tmp/xtv6`, scratch): every bound keystroke × every dispatcher state, pre-refactor `press` (`/tmp/xtv5`) vs this tree | **1558 pairs, outputs `cmp`-identical** (1559 lines each) — the behaviour-preservation evidence |
| `go test ./internal/render -run TestCorpusAudit -v` | PASS — 1 parent + **16 subtests**; the no-http arm logged `read 170 covered frames` (unchanged) |
| `go test ./internal/render -run TestCapturePanic -v` | PASS — multiplexer leg `62 labelled lines → 0` (13 more sit in the alternate screen's own saved copy, which no clear reaches), the control `leaked: the pane holds 41 labelled lines with the clear disabled`, the four later covered screens each `62 → 0`; the bare leg skipped with its printed reason. The control leg still logs two `kill-server … exit status 1` lines on a passing run — cosmetic; see tech debt |
| `go test ./internal/render -run TestGoldenFrames` (no `-update`) | PASS over **263 stored frames** (154 `cover_*`, 16 `media_gated_*`, 2 `panic_*`, 91 plain) |
| `go test ./internal/tui -run TestGoldenFrames` (no `-update`) | PASS over the **56** `internal/tui` frames (24 profile, 32 search) |
| `go test ./internal/tui -run TestTheCoveredSavedPaneNamesNothingOfTheProduct -v` | PASS — **16 state/width subtests** (8 saved states × 2 widths) and 64 inner rows (four covers inside each), the `the store cannot be read` state among them |
| `go test ./internal/tui -run TestCoverBreakingActionsGated`, `…TestPanicIgnoresCoverBreakingKeys`, `…TestPanicSwallowsKeys`, `…TestStateTransitions`, `…TestPanicEntersInOneFrameAndLeavesExactly`, `…TestTheQuitKeysWriteNothing` | PASS — the restructure's own risk surface, green |
| `x-terminal audit` (temp `XDG_CONFIG_HOME`/`XDG_DATA_HOME`) | **10 checks, 0 findings, 893 files inspected**, exit 0 — unchanged from last pass |
| `diff -rq /tmp/xtv5/<tree> <tree>` (source, excluding `.git`/`.planning`) | only `internal/tui/update.go` (changed) and `internal/tui/dispatch_test.go` (added) — nothing else in the tree moved |
| `diff -rq /tmp/xtv5/internal/{render,tui}/testdata <tree>` | rc=0 both — **no stored frame's bytes moved** this pass either |
| plants — **33** (every falsification the last report recorded, re-run) plus the guard's own **7 meaningful** plants, each in a scratch copy, each arm on a fresh copy of a pristine base | **40 of 40 meaningful plants caught**; 2 further plants (G1c/G1e) pass and are recorded as notes, not as gaps |
| the shipped binary, 90×24 tmux pane, private `HOME`/`XDG_CONFIG_HOME`/`XDG_DATA_HOME`, own tmux socket, `unshare -rmn`, session from `X_TERMINAL_AUTH_TOKEN`/`X_TERMINAL_CT0`, `display.state = immersive` + `display.cover = gitlog` in `config.toml` | **2 runs, the pre-refactor binary and this one, side by side**: start, the three gated keys `i`/`y`/`o`, `C-p`, six swallowed keys (`m q 1 i z ?`), `C-p` back — **all seven captures byte-identical between the two binaries**; the panic frame is untouched by the six keys; the frame after `C-p` is byte-identical to the frame before it (`git` on the window, `operation refused  branch=main` on the status line) |
| `verification.fingerprint` | recomputed over the 244-path set below: `44b866f8…`, carried in this report |
| `verification.status <phase dir>` | `passed` after this report was written |

## Covered-file set

**244 paths**, and the digest above is `verification.fingerprint`'s own output over exactly this set. This pass **adds 1** — the guard's own test file, whose failure lines this report cites:

- `internal/tui/dispatch_test.go` — the keymap⇄dispatcher join this pass adjudicates; plants G1, G1b, G1d, G1f, G2 fail inside it and its own line numbers are quoted in the guard section. Without it the report would cite failure lines from a file outside its covered set, and a digest that ignores the phase's newest test file is a digest over a tree that no longer exists.

The previous pass declared 243 paths (241 the pass before), and its additions stand as recorded below, both of them files the fix round changed and this report's own evidence rests on:

- `internal/screen/saved.go` — the release rule the covered saved frame's states are now produced under. The covered test's reach comment names it (`the frame's own release rule (screen.Saved)`), the fix round's third claim turns on it, and the shipped runs' capability state is the state it governs.
- `internal/screen/saved_test.go` — the rule's own arm (`TestSavedCapabilityGoesWhenTheFramesOwnActionSucceeds`, added this round), which is where plants B1 and B2 fail. Without it the report would cite failure lines from a file outside its covered set.

Considered and **not** added: `internal/store/{read.go,schema.go,migrate.go}` (the `FileError` mark's producers — the mirror's storage is phase 6's evidence base, and this report observes the mark only through `internal/cli/feed.go`'s already-covered mapping, its own test arm and the shipped runs); `internal/audit/{audit.go,readonly.go}` (the new `opnames` check — milestone tooling, falsified here as plant C exactly as last pass falsified `source.go` as plant 26, but not part of the cover's evidence base); `internal/xweb/volatile/pins.go` and `internal/domain/page.go` (comment-only moves on promises that are not this phase's); and the `search_people_*` covered frames (byte-unchanged again; the covered people field is the plain spelling by design).

## The amendment's claims, adjudicated against the cover truths

(The round before this one; every claim below was re-checked on this tree and every plant re-run — see the table. Nothing here regressed.)

1. **A covered likes frame no longer resolves its own ambiguity.** `SavedEmptyState` (`saved.go:173`) branches on `kind == domain.Like` *first*: under a cover it returns the one-line `StateBlock(g, w, CopyCoverEmptyHedged, "")` (`no records in this view, or the view is no longer served`), never the heading+body that says the read succeeded. The 8 stored covered frames carry exactly that one line; `TestTheCoveredLikesHedgeIsInTheCorpus` requires the hedge, forbids `CopyCoverEmptyBody`/`CopyEmptyLikesHedged`/`CopyListLikes`, and requires the frame to be one block of one row. **Falsified**: plant 23 (the per-kind branch deleted) fails `TestGoldenFrames` on all 8 likes-hedged frames, and — with those frames regenerated as the regression would ship them — the corpus arms too: `corpus_test.go:2340: the covered likes-empty frame cover_servicelog_likes_hedged_80 does not carry the cover's own hedge`, `:2344: … holds "no likes, or the list is no longer served"`, and `:390: the stored frame … prints "likes", the product's own word (from CopyListLikes)`.
2. **The plain-vocabulary scan covers every covered frame in both corpora.** Render side: `TestCoveredFramesLeakNoRepairWords` loops `covered` from `partition(storedFrames)` with no pane filter, requires `scanned == len(covered)` and ≥60. Tui side: `TestTheCoveredTUIFramesLeakNoPlainVocabulary` (`internal/tui/corpus_test.go`) requires ≥24 covered frames, scans each for `tuiLeakWords()` + the corpus's own account identities, and requires every covered frame to carry an alias of its own style. Both have self-plant arms over real frames written to temp dirs. **Falsified externally**: plant 21 (`saved.db doctor` into `cover_gitlog_thread_80`, a *thread* frame) and plant 22 (`saved.db` into the tui's `search_gitlog_80`) each fail by frame name.
3. **`frameKinds()` is joined to the whole-surface table both ways.** `TestEveryFrameKindIsCheckedUnderACover` (`cover_e2e_test.go:730`) counts the table's rows per declared kind, requires exactly one each way, then walks each kind under one cover (gitlog). **Falsified both directions**: plant 20a (`the shell declares the frame kind 99 and the covered check table has 0 rows for it`) and plant 20b (`the covered check table has 6 rows and the shell declares 5 frame kinds`).
4. **The four covered notices have frames and a totality arm.** 32 stored frames (4 styles × 4 notices at 120, plus tmux twins). `TestTheCoveredNoticesAreOnScreen` requires every covered frame whose state carries a notice to carry its style's own words and not the plain reader's; `TestTheCoveredCopyIsTotal` (`status_test.go:971`) requires every covered notice at three counts to differ from the plain reader's. **Falsified**: plant 19 (all four `NoticeLine` cover arms reverted) fails three tests; first line `corpus_test.go:2286: the covered frame cover_servicelog_sync_notice_120 does not carry "syncing, 40 new", its own style's words for the notice`.
5. **A covered profile names one account consistently.** `profileAccount` (`view.go:447`) returns the header's account once it has answered and `(domain.User{}, false)` before. **Falsified**: plant 24 fails with the old finding-3 symptom verbatim (`the covered profile frame names an account before its header has answered: "account: pump-sign-57 …"`).
6. **Every cover stamp sets `MonthYear`.** `cover.go:195-210`: all four stamps set it. **Falsified**: plant 25 (`MonthYear` dropped from `coverGitStamp`) fails `TestGoldenFrames/profile_gitlog_{80,120}`.
7. **Hygiene.** `capture_tmux_test.go` kills its server and then asserts both halves of the cleanup. The 9 dead `copy.go` constants are gone. The audit's `source` check runs first and its claim is falsified by plant 26.

## The blocker, still closed

The earlier pass's blocker (the covered saved pane's capability, empty and failed-with-empty-mirror states printing the product's repair vocabulary) stays closed, re-measured three ways on this tree: `TestTheCoveredSavedPaneNamesNothingOfTheProduct` passes with its 16 subtests (8 states × 2 widths, four covers each, the likes-hedged state a seventh row and the amended `store cannot be read` state a third); plant 1 (the three builders handed `render.Plain` again) fails it for every state at both widths; and five hermetic shipped runs print the cover's own words under all four covers with every leak word counting 0, while the plain run prints the real ones — including the state the fix round repaired, reached in the real binary by an unopenable `saved.db`.

## Plants

Forty, each on a fresh copy of a pristine base (`/tmp/xtv7/base`, itself a byte-copy of the frozen tree with `.git` removed), each discarded afterwards — the repository was never the plant site, so nothing needed restoring in it (`diff -rq` against the base reports only this report file and other phases' reports). The outcome quoted is the run's own first failure line(s). Rows **1–27** are the last report's recorded falsifications re-run, **A–E** the previous round's own re-run, and **G1–G4** the guard's plants, added this pass.

| # | Plant | Outcome |
|---|---|---|
| 1 | `view.go` `savedPane` hands `render.Plain` to the three builders | `--- FAIL: TestTheCoveredSavedPaneNamesNothingOfTheProduct` — all states × both widths; first line `cover_e2e_test.go:1467: the covered servicelog frame holds "doctor", a word the cover did not author (state "the list's operation is missing")` |
| 2 | `x-terminal` into a blank row of `cover_gitlog_saved_unreadable_80.golden` | `corpus_test.go:2063: the covered frame cover_gitlog_saved_unreadable_80 holds "x-terminal", a word the cover did not author` |
| 3 | `👤` into row 2 of `cover_ci_saved_empty_80.golden` | `corpus_test.go:149: row 2 of the stored frame cover_ci_saved_empty_80 holds "👤" (U+1F464) …` under `the_absences` |
| 4 | `👤` in `panic.go`'s own literals | `panic_test.go:63: PanicFrame(80, 24): row 21 holds "👤" (U+1F464) …` at every size |
| 5 | `helpRows` passes `false` instead of `covered` | `state_test.go:517: the covered overlay does not carry "next record"` (+ the other covered wordings) |
| 6 | `Help` returns the uncut overlay | `help_test.go:173: the overlay is 30 rows for a 23-row pane` |
| 7 | `emptyBlock` passes `""` to `FailureWords` | `--- FAIL: TestTheCoveredFailurePaneNamesNothingOfTheUpstream` under all four covers |
| 8 | `lzd` into a row of `panic_80.golden` | `corpus_test.go:385: the stored frame panic_80 prints "lzd", an identity of the content it was built from` |
| 9 | `coverGitFacts` replaced by the plain reader's clauses | `TestGoldenFrames/profile_gitlog_{80,120}` byte mismatch |
| 10 | `ListLabelProfile`'s covered arm prints `u.ScreenName` | `TestGoldenFrames/profile_servicelog_{80,120}` (and the other three styles') byte mismatches |
| 11 | `CapabilityWord` ignores the cover | `corpus_test.go:2098: the covered capability frame cover_servicelog_saved_unavailable_80 does not carry its own state's word` — 16 findings |
| 12 | `StateBlockWithMirror` passes `""` to `savedNeverMirrored` | `TestGoldenFrames/cover_{servicelog,gitlog,ci,internal}_saved_failed_{80,120}` — exactly the 8 frames |
| 13 | `cover_internal_saved_empty_120.golden` deleted | `corpus_test.go:104: reading …: open …: no such file or directory` |
| 14 | the `saved_empty` shape renamed to `saved_nil` (the 8 corpus files) | `corpus_test.go:104: reading testdata/TestGoldenFrames/cover_servicelog_saved_empty_80.golden: open …: no such file or directory` (the corpus names the frames it expects; a renamed file is both a missing frame and a stale one) |
| 15 | `CoverAliasOf` returns the real handle | `alias_test.go:58: CoverAliasOf("servicelog", aergon) = {Handle:aergon …}, want {Handle:agent-rollback-70 …}` in all four styles |
| 16 | `_ "os/exec"` imported by `internal/render/cover.go` | `boundary_test.go:267: internal/render/cover.go imports os/exec, and only internal/term and internal/media may` |
| 17 | `fallbacksFor` returns `f.Fall` under a cover | `tui_test.go:2183: under servicelog the pane is handed a 1-entry fallback map, want the zero map` (all four covers) |
| 18 | `ctrl+d` → `ctrl+x` in `cover_ci_help_80.golden` | `corpus_test.go:783: key column row 3 differs: cover_servicelog_help_80="ctrl+d          " cover_ci_help_80="ctrl+x          "` |
| 19 | the four `NoticeLine` covered arms reverted to the plain vocabulary | `corpus_test.go:2286` and `:2289` on the sync notice (+ `TestGoldenFrames` and `TestTheCoveredCopyIsTotal` fail) |
| 20a | a sixth frame kind appended to `frameKinds()` | `cover_e2e_test.go:739: the shell declares the frame kind 99 and the covered check table has 0 rows for it` (+ the length arm) |
| 20b | a check-table row for `frameKind(77)` | `cover_e2e_test.go:752: the covered check table has 6 rows and the shell declares 5 frame kinds` |
| 21 | `saved.db doctor` into `cover_gitlog_thread_80.golden` — a covered *thread* frame | `corpus_test.go:2063: the covered frame cover_gitlog_thread_80 holds "saved.db"` and `"doctor"` |
| 22 | `saved.db` into `internal/tui`'s `search_gitlog_80.golden` | `corpus_test.go:831: the covered tui frame search_gitlog_80 holds "saved.db", a string the cover of gitlog did not author` |
| 23 | the covered likes-hedge branch deleted from `SavedEmptyState` | code-only: `TestGoldenFrames` fails on all 8 likes-hedged frames. Regenerated (`-update`) as the regression would ship: `corpus_test.go:2340` / `:2344` (the hedge gone, the plain line and `"likes"` in it) and `corpus_test.go:390` (the product's own word) |
| 24 | `profileAccount` names the opened handle under a cover before the header | `cover_e2e_test.go:835: the covered profile frame names an account before its header has answered: "account: pump-sign-57 …"` — the old finding-3 symptom verbatim |
| 25 | `MonthYear` dropped from `coverGitStamp` | `TestGoldenFrames/profile_gitlog_{80,120}` byte mismatch (the `Since:` clause gone) |
| 26 | `internal/render/cover.go` made unparseable | `x-terminal audit`: `source` reports `internal/render/cover.go:523: cannot parse: expected ')', found '{'`; exit 1 |
| A | the pre-fix "store cannot be read" reach restored (a page answered while the frame wears `CapMirrorUnreadable`) | `--- FAIL: TestTheCoveredSavedPaneNamesNothingOfTheProduct/the_store_cannot_be_read/{80,120}` — `cover_e2e_test.go:1454: the plain frame of "the store cannot be read" does not carry "move saved.db aside": the covered absence below would be vacuous` (the release rule clears the state, so the old pair cannot be produced) |
| B1 | the `MirrorPage` release rule deleted (`screen.Saved`) | `saved_test.go:85: capability after a page arrived = mirror unreadable, want CapNone: the read succeeded, so the file state it refutes is over` and `:101: … how the state was set does not change what a successful read of the list proves` |
| B2 | the `SyncFinished` release rule deleted (`screen.Saved`) | `saved_test.go:118: capability operation missing / wrong account / mirror unreadable after a sync that finished = …, want CapNone` |
| C | the stale `ListLatestTweetsTimeline` back in the classification table | `x-terminal audit`: `internal/audit/readonly.go: the table classifies "ListLatestTweetsTimeline" as a read and the registry does not hold it: an entry for an operation this build cannot resolve`; exit 1 |
| D | a `*xsource.ParseError` accepted as a failure of the file again (the blacklist's hole) | `feed_test.go:498: a shape-drifted answer (…) was read as a failure of the mirror`, `:520: the status line is "mirror unreadable  move saved.db aside, then sync …", want the state for a shape this build has no rule for ("unclassified")` |
| E | `volatile.PinsFileName` renamed | `config_test.go:741: the pins file's name is "pins.jsonx", want pins.json: an operator's pins file is at that name, so a rename is a migration` and `:745: PinsPath = "/tmp/x-data/pins.jsonx", want "/tmp/x-data/pins.json"` — the path follows the constant |
| G1 | a 31st keymap row with a fresh action (`ActionZzPlant`, declared in the const block, bound to `z`, handled by nothing) | join: `dispatch_test.go:63: the keymap binds "zz-plant" (z) and press has no arm for it: the key is a documented no-op` + `:79: the table holds 31 rows and handledActions names 30 actions`; sweep: `:128: keys.go declares ActionZzPlant and handledActions does not name it` + `:159: keys.ActionZzPlant is no case label in the shell` |
| G1b | an action declared and bound by nothing at all (`ActionZzOrphan`) | sweep: `:125: keys.go declares ActionZzOrphan ("zz-orphan") and no row binds it: an action with no key is not an action` (+ `:128`, `:159`) — the join test's own direction cannot see it, the sweep does |
| G1c | a fresh action declared outside the const block, in a shape the sweep's regex reads (`const ActionZzOutside Action = "zz-outside"`), named nowhere | **passes** (both guard tests `ok`): an undeclared-but-unused name is inert; recorded as tech debt 8, not a gap |
| G1d | the same declaration **plus** a dispatcher arm naming it | sweep: `dispatch_test.go:147: the shell names keys.ActionZzOutside, which keys.go does not declare` — the refs arm closes the shape hole the moment anything dispatches it |
| G1e | a `var ActionZzTail Action = "zz-tail"` appended outside the block, named nowhere | **passes** (both guard tests `ok`) — the inert half of G1c/G1d; recorded as tech debt 8 |
| G1f | the same `var` shape **plus** a dispatcher arm | sweep: `dispatch_test.go:147: the shell names keys.ActionZzTail, which keys.go does not declare` |
| G2 | a dispatcher arm and a `handledActions` entry for an action no row binds (`ActionZzGhost`) | join: `dispatch_test.go:72: press handles "zz-ghost" and no row of the keymap binds it: the arm is unreachable` + `:79`; sweep: `:125: keys.go declares ActionZzGhost ("zz-ghost") and no row binds it` |
| G3 | a case label naming an action the keymap package does not declare | `go build ./internal/tui`: `internal/tui/update.go:268:12: undefined: keys.ActionThisPackageDeclaresNoSuchAction` — the compiler arm `press`'s own comment claims; `rc=1` |
| G4 | the `ActionCopyLink` arm's body replaced by a no-op, its case label intact | **both guard tests pass** — the join is structural; the behavioural tests are the arm: `tui_test.go:1893: clipboard = 0/"", want 1/"https://x.com/a_15_cell_handl/status/adv-0"` and `tui_test.go:2062: y returned nil command`. Recorded as tech debt 9 (the guard's honest boundary) |

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 03-01.1 the same post renders through five Grammar values with the text, order, relationship and instant untouched | ✓ VERIFIED | `TestCoverGrammarsAreValues`, `TestGoldenFrames` over the byte-exact renderings (263 frames), `assertSameState`; `CoverStyles()` is still exactly four |
| 2 | 03-01.2 no covered rendering contains a real handle or display name; aliases readable and stable across calls, processes and restarts | ✓ VERIFIED | `TestCoveredRenderingsHideTheIdentity`, the V2 scan, `TestCoverAliasesAcrossProcesses`; plant 15 fails the worked examples in all four styles |
| 3 | 03-01.3 `internal/render` cannot write, asserted by parsing the package's own source | ✓ VERIFIED (as landed: `os` is read-only, not absent) | `TestRenderCannotWrite`; plant 16 fails `alias_test.go:365`/`boundary_test.go:267` |
| 4 | 03-01.4 every covered frame and the panic frame are composed by the same composer, no style name in a condition, one Grammar value is the whole of what a style changes | ✓ VERIFIED | the saved builders still take a `Grammar`; `grep -rnE '\.Name *==' internal/render/*.go` (non-test) → only `CoverOfGrammar`'s lookup; plant 16 fails the boundary arm |
| 5 | 03-01.5 the panic frame is one constant frame of H rows of W cells, byte-identical at 80 and 120, zero ESC, no feed text, no handle, no clock, no prompt, no product noun | ✓ VERIFIED | `TestPanicFrameIsExactlyHRowsOfWCells`, the constancy test, V5; plants 4 and 8 fail its glyph scan and the corpus identity arm |
| 6 | 03-01.6 the plain reader's frames are byte-identical to Phase 2's stored frames, and no covered frame renders the real identity, the list names, the notices or the failure states in the product's voice | ✓ VERIFIED | First conjunct: `TestGoldenFrames` passes with no `-update`, and both `testdata` trees are byte-identical to the pre-fix copy (no plain frame moved this round either). Second conjunct: the covered scan runs over every covered frame in both corpora (plants 21, 22), the notices are covered words on screen (19), the likes hedge leaves the ambiguity open (23), the covered profile names one account (24) |
| 7 | 03-02.1 one key switches normal → immersive and back, one key walks the four styles, one key replaces the pane in a single frame, no request | ✓ VERIFIED | `TestStateTransitions` (16 subtests, green after the refactor), `TestPanicEntersInOneFrameAndLeavesExactly`, `TestStartupStateIsTheConfiguredOne`; the differential sweep matches the pre-refactor tree on every state/key pair |
| 8 | 03-02.2 the same key leaves panic and the screen comes back exactly, with the source's call count unchanged | ✓ VERIFIED | `TestPanicRoundTripIsExact` — green |
| 9 | 03-02.3 entering panic writes one ordered byte string (cells → ED 3 → title), and only then clears the multiplexer's history and renames the window; a missing pane variable starts no subprocess and is not an error | ✓ VERIFIED | `TestEntryOrder`, `TestEnterPanicIsOneWrite`, `TestEnterPanicOrderAcrossEffects`, `TestPaneTable`; the multiplexer leg re-ran this pass (62 → 0, control leaked 41) |
| 10 | 03-02.4 losing focus enters panic; regaining changes nothing; a second blur does not re-enter; a terminal reporting no focus never panics by itself | ✓ VERIFIED at the model level (the real focus event is not drivable here) | `TestBlurPanicsAndFocusDoesNotRestore`, `TestPanicSwallowsKeys`; the sweep's `panic_*` scenarios match the old tree — see Human Verification |
| 11 | 03-02.5 the title is a function of the state, the tmux window name follows it, both are restored on exit | ✓ VERIFIED | `TestTheTitleIsAFunctionOfTheState`, green |
| 12 | 03-02.6 `os/exec` appears in exactly one package, and the boundary test fails on a planted violation | ✓ VERIFIED (as landed — the allowlist is `internal/term` + `internal/media`; see drift) | `TestProcessSpawnsAreOnlyInTermAndMedia`; plant 16 |
| 13 | 03-02.7 the display state and cover are read at start-up and written only on a deliberate change; a panic entered by a key or by focus loss writes no file | ✓ VERIFIED | `TestThePersistedStateIsTheDeliberateOne`, `TestNoAliasTableOnDisk`; this pass's shipped runs opened covered from the two config keys |
| 14 | 03-03.1 the panic round trip is byte-exact in frames, requests, stack depth, selection, anchor, latch and failure state | ✓ VERIFIED | as #8, and on the shipped path: the tmux run's post-`C-p` capture is byte-identical to the covered capture it left, with the window name back to `git` |
| 15 | 03-03.2 no covered or panic frame contains a real handle or display name, and each carries at least one expected alias so an empty frame cannot pass | ✓ VERIFIED | the V2 arms over the grown corpus plus the tui half's own scan with its every-covered-frame-carries-an-alias arm; plants 8, 15, 22 |
| 16 | 03-03.3 every Phase 2 state has a covered counterpart from the same model with the same keys and an identical request count; the plain frames stay byte-identical | ✓ VERIFIED **and the enumerator is joined** | `TestTheWholeSurfaceUnderACover` over the nine Phase 2 screens (green); `TestEveryFrameKindIsCheckedUnderACover` joins `frameKinds()` (all five kinds, saved included) to per-kind covered checks both ways (plants 20a/20b). Residual: the full `assertSameState` + equal-request-count property is still `screens()`'s nine screens only — see tech debt 1 |
| 17 | 03-03.4 the entry's bytes place the cells before ED 3 before the title, the tmux work starts only after the write, a start-up panic runs the same sequence | ✓ VERIFIED | #9's tests plus `TestPanicEntryOrderAndBranches`; this pass's tmux run cleared 62 labels and the control leaked 41, and the shipped `C-p` set the window name to `make` then back to `git` |
| 18 | 03-03.5 a whole covered run creates or modifies no file except the two config keys, and no written file contains an alias, handle or display name | ✓ VERIFIED | `TestNoAliasTableOnDisk`, green |
| 19 | 03-03.6 in the real environment the clear works, with a control, on a bare terminal and under a multiplexer | ✓ VERIFIED (multiplexer re-run by me; bare-terminal leg from the author's evidence) | the multiplexer leg re-ran this pass: `before=62 labelled lines → after=0`, the control `41`, the later covered screens each `62 → 0`; the bare leg skipped with its printed reason |

**Score:** 19/19 truths verified (0 FAILED; 0 present, behavior-unverified) — the same nineteen as last pass, each re-adjudicated on this tree; the dispatcher refactor moved no truth, and the guard adds structure, not a truth

### Required Artifacts

| Artifact | Status | Details |
|---|---|---|
| `internal/render/grammar.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | `StampFormat.MonthYear` declared with the `{joined}` rule; all named fields present |
| `internal/render/alias.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | FNV-1a over the versioned purpose+key; plant 15 fails the worked examples |
| `internal/render/cover.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | four grammar values, four facts rows each with a fourth clause, every stamp sets `MonthYear` (plant 25); `CoverStyles()` in the frozen order; plants 9, 16, 26 |
| `internal/render/panic.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | 21 lines × three columns; plant 4 fails its glyph scan |
| `internal/render/copy.go` + `status.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | `FailureWords` the one covered-aware decider; the four notice arms asserted three ways (plant 19); `ListLabelProfile`'s covered arm (plant 10) |
| `internal/render/saved.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | the three builders take a `Grammar`; the likes per-kind branch (plant 23); plants 11 and 12 fail over it |
| `internal/render/profile.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | `profileFacts` renders a grammar's own clauses, omitting `{joined}` only when the stamp has no `MonthYear` — the pin plant 25 breaks |
| `internal/term` | ✓ EXISTS + SUBSTANTIVE + WIRED | the entry, the title, the three tmux commands, injectable seams |
| `internal/keys` | ✓ EXISTS + SUBSTANTIVE + WIRED | the phase's rows, `Row.HelpCovered` + `HelpText(covered)`; plant 5 fails the covered wording; `ByKeystroke` has no production caller |
| `internal/config` | ✓ EXISTS + SUBSTANTIVE + WIRED | the two vocabularies, validation, the in-place writer; `PinsPath` built from `volatile.PinsFileName` (plant E); the shipped runs read `display.state`/`display.cover` this pass |
| `internal/tui` | ✓ EXISTS + SUBSTANTIVE + WIRED | the three states, transitions, panic overlay, focus handling, `fallbacksFor` (plant 17); `profileAccount` names one account or none under a cover (plant 24); `frameKinds()` is joined by the check table (plants 20a/20b); the saved surface's eight-state table (plants 1, A) |
| `internal/cli/feed.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | sink, start-up state, exit-time restore; the capability state's whitelist mapping (plant D); the shipped runs walked the capability branch for real |
| `internal/screen/saved.go` | ✓ EXISTS + SUBSTANTIVE + WIRED — **added to the set this pass** | the stickiness/release rule the covered saved frame's states are produced under (plants B1, B2) |
| `internal/render/testdata/TestGoldenFrames/{cover_*,media_gated_*,panic_*,likes_hedged_*}` | ✓ EXISTS + SUBSTANTIVE | **263** stored frames (154 covered), byte-compared with no `-update`, **byte-identical to the pre-fix copy**; plants 2, 3, 12, 13, 18, 21, 23, 25 fail over these bytes |
| `internal/render/corpus_test.go`, `golden_cover_corpus_test.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | `TestCorpusAudit`'s 16 subtests; the covered-frame scan over every covered frame; the notice and likes-hedge arms with their own floors |
| `internal/tui/corpus_test.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | the covered-frame scan over the tui half, with its ≥24-frame floor, its plain arm, its every-frame-carries-an-alias arm and its self-plant arm; plant 22 |
| `internal/tui/cover_e2e_test.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | the round trip, the whole surface, the frame-kind join, the covered profile's one account, the saved surface's own eight-state table with the amended `store cannot be read` reach; plants 1, 7, 20a, 20b, 24, A |
| `internal/tui/dispatch_test.go` | ✓ EXISTS + SUBSTANTIVE + WIRED — **added to the set this pass** | the keymap⇄dispatcher join both ways (a bound action with no arm; an arm with no row), the anti-vacuity floor (30 rows/30 actions) and the copy check on `HandledActions()`, plus the source sweep over `keys.go`'s declarations, the shell's refs and its case labels; plants G1, G1b, G1d, G1f, G2 fail inside it, G3 in the compiler |
| `internal/tui/profile_test.go` | ✓ EXISTS + SUBSTANTIVE + WIRED | `TestGoRestoresTheProfileHeader`, green |
| `internal/render/capture_panic_test.go` (+ `capture_tmux_test.go`, `capture_pty_test.go`) | ✓ EXISTS + SUBSTANTIVE + WIRED | both legs, the control inside the same test, the socket-file cleanup asserted; re-run this pass |

**Artifacts:** 19/19 present and wired (the screen package's saved frame joined the set).

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `internal/tui` | `render.Blocks` | `pane()` → `m.grammar()` | ✓ WIRED | the grammar is the only difference between plain and covered |
| the list, thread, search and profile failure panes | the pane's two strings | `view.go:196` → `render.FailureWords` | ✓ WIRED | plant 7 fails the covered pane's names |
| the saved pane's capability, empty and failure branches | the pane's words | `view.go:404-424` → `CapabilityBlock(m.grammar(), …)` / `StateBlockWithMirror(m.grammar(), …)` / `SavedEmptyState(m.grammar(), …)` | ✓ WIRED | plant 1 fails the arm for every state |
| the saved frame's first page in flight | the pane's line | `view.go:420` → `render.SavedLoadingLine(m.covered())` | ✓ WIRED | the saved state table's seventh row carries the covered line |
| the covered saved frame's notices | the status line's notice field | `view.go` → `render.NoticeLine(m.covered(), …)` | ✓ WIRED + ASSERTED | 16 stored frames + `TestTheCoveredNoticesAreOnScreen` + the totality arm; plant 19 |
| the covered profile/search status fields | the cover's own words | `view.go` → `ListLabelPeople/Search/Profile(m.grammar(), …)` | ✓ WIRED | plants 9 and 10 fail the covered goldens; `profileAccount` names the header's account or none (plant 24) |
| a covered pane's fallback rows | the composer | `model.go` `fallbacksFor` | ✓ WIRED | by state, not by the covers happening to set `AttachmentRow false`; plant 17 |
| the covered help overlay | the table's own wording | `view.go:222` `helpRows(covered)` + `render.Help(rows, w, paneRows())` | ✓ WIRED | plants 5, 6, 18 |
| the stored covered failure frames | the shell's own `Render` | the same `FailureWords` | ✓ WIRED | plant 12 fails `TestGoldenFrames` |
| covered frames | `text.Wrap`/`text.Pad` | the expander + the measure | ✓ WIRED | the width arm holds every stored frame to exactly W cells; plant 3 trips it |
| the shell | `internal/term` | one value in, nothing back, from a command | ✓ WIRED | `TestTheEntryIsCommandedNotCalled` |
| `internal/config`'s vocabulary | the shell's and the render package's | `TestDisplayVocabularyAgrees` | ✓ WIRED | every accepted value resolves |
| the fake source's counter | "no request was issued" | `s.src.Calls()` | ✓ WIRED | read numerically in the round trip and the whole-surface comparison |
| the shell's frame-kind enumeration | the covered per-kind table | `frameKinds()` ↔ `frameKindChecks()` | ✓ WIRED both ways | `TestEveryFrameKindIsCheckedUnderACover`; plants 20a and 20b fail the two directions |
| the keymap's rows | the dispatcher's arms | `handledActions` ↔ `keys.Rows()` ↔ the shell's case labels, in `dispatch_test.go` | ✓ WIRED both ways — **new this pass** | plants G1 (a 31st row), G1b (an action no row binds), G2 (an arm no row binds) each fail a named line; G3's bogus case label fails the build |
| an unopenable `saved.db` | the frame's capability and its covered words | `openSaved` → `store.FileError` → `savedMirrorError` → `Deps.SavedCapability`/`SavedError` → `CapabilityBlock(m.grammar(), …)` | ✓ WIRED — **new this pass** | plant D fails the predicate's negative arm; five hermetic shipped runs render the covered state under all four covers with 0 leak words |

**Wiring:** 15/15 connections verified.

## Requirements Coverage

| Requirement | Status | Blocking Issue |
|-------------|--------|----------------|
| DSGT-01 three states, one key each, one-frame panic, exact restore without a request | ✓ SATISFIED | - |
| DSGT-02 panic clears what came before (scrollback + the multiplexer's history) | ✓ SATISFIED | re-measured this pass under tmux with the control |
| DSGT-03 the title (and tmux window name) becomes the cover's own name, restored on exit | ✓ SATISFIED | - |
| DSGT-04 losing focus enters panic; regaining does not restore | ✓ SATISFIED | model-level; the real event needs a focus-reporting terminal — recorded, not faked |
| DSGT-05 four cover styles, switchable at runtime | ✓ SATISFIED | `CoverStyles()` is still exactly four |
| DSGT-06 identities replaced throughout; no real handle or display name rendered anywhere on screen | ✓ SATISFIED | re-planted (15, 8, 22); the covered profile's account field no longer names two accounts (plant 24) |
| DSGT-07 deterministic aliases, no alias table on disk | ✓ SATISFIED | - |
| DSGT-08 the whole read surface works under a cover with unchanged keys | ✓ SATISFIED | the nine-screen table plus the frame-kind join over all five kinds (saved included) with per-kind covered checks; the saved surface's eight states × four covers × two widths |
| DSGT-09 nothing in immersive or panic reveals the real product or its purpose | ✓ SATISFIED | the covered scan over every covered frame in both corpora (21, 22); the notices (19), the likes hedge (23), the stamps (25) and the profile's account (24) each carry a failing plant; the amended saved state is reachable in the shipped binary with 0 leak words |
| DSGT-11 the state and cover persist; the user picks the start-up state | ✓ SATISFIED | the shipped runs opened covered from the config's two keys |
| ROADMAP SC1 single keys switch and restore exactly, without a request | ✓ SATISFIED | - |
| ROADMAP SC2 panic leaves nothing reachable behind and renames the title/window | ✓ SATISFIED | - |
| ROADMAP SC3 losing focus panics; regaining does not restore | ✓ SATISFIED | as DSGT-04 |
| ROADMAP SC4 four styles, deterministic aliases, no alias table, and no real handle, product name, status line, hint, engagement count, avatar or icon while covered | ✓ SATISFIED | every conjunct re-planted this pass |
| ROADMAP SC5 the whole surface works under a cover; state and style persist | ✓ SATISFIED | as DSGT-08 |

**Coverage:** 10/10 requirement rows satisfied; 5/5 roadmap success criteria.

## Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/render/alias_test.go` | DSGT-06, 07 | yes | 0 | no | Value + behavioral (two processes) | Adequate — plant 15 |
| `internal/render/panic_test.go` | DSGT-09 | yes | 0 | no | Value + absence + glyph | Adequate — plant 4 |
| `internal/render/corpus_test.go` | DSGT-01, 06, 08, 09 | yes | 0 | no (reads stored bytes) | Behavioral over 263 stored frames, anti-vacuity floors, self-plant arms | Adequate — every covered frame (2, 3, 8, 13, 14, 21) plus the notice (19) and likes-hedge (23) arms; a *generator* change still only the golden test sees (plant 12) — by design, stated in the test's own comment |
| `internal/render/status_test.go` | DSGT-09 | yes | 0 | no | Value + behavioral + totality | Adequate — `TestTheCoveredCopyIsTotal` fails on plant 19 |
| `internal/render/saved_test.go` | DSGT-09 | yes | 0 | no | Value (plain sentinel) | Adequate for the plain half; the covered half is the corpus frames' |
| `internal/render/prompt_test.go` | DSGT-08 | yes | 0 | no | Value (plain sentinel) | Adequate; the covered labels are the tui frames' (plant 10) |
| `internal/render/cover_test.go` | DSGT-05, 06, 09 | yes | 0 | no | Value (byte-exact) | Adequate |
| `internal/render/help_test.go` (+ the shell's overlay tests) | DSGT-08 | yes | 0 | no | Value + behavioral | Adequate — plants 5, 6, 18 |
| `capture_panic_test.go`, `capture_tmux_test.go`, `capture_pty_test.go` | DSGT-02 | multiplexer leg yes | bare leg gates on a Ghostty window (printed reason) | no | Behavioral, with a control | Adequate; cleanup asserted (no server, no socket file); the bare leg is a human checkpoint by design |
| `internal/tui/corpus_test.go` | DSGT-06, 09 | yes | 0 | no | Behavioral over the tui half's 56 frames + self-plant | Adequate — plant 22 |
| `internal/tui/cover_e2e_test.go` | DSGT-01, 08, 09, 11 | yes | 0 | no | Behavioral | Adequate — the frame-kind join (20a/20b), the covered profile's one account (24), the saved surface's eight-state table (1, A) |
| `internal/tui/dispatch_test.go` (new) | DSGT-01, 05, 08 (dispatch) | yes | 0 | no (reads sources as syntax) | Structural + value, with anti-vacuity floors | Adequate — 7 plants attempted, 5 fail inside it (G1, G1b, G1d, G1f, G2), 1 in the compiler (G3); a gutted *arm body* stays green here by design (G4) and is caught by the behavioural tests — tech debt 9 |
| `internal/tui/tui_test.go`, `state_test.go`, `boundary_test.go` | DSGT-01, 04, 07, 11, boundary | yes | 0 | no | Behavioral + structural | Adequate — plants 16, 17, B1/B2's composition arm; see tech debt 6 |
| `internal/tui/profile_test.go` | DSGT-08 | yes | 0 | no | Behavioral (pane rows) | Adequate |
| `internal/keys`, `internal/config`, `internal/term` tests | DSGT-05, 11, 02 | yes | 0 | no | Value + behavioral | Adequate — plant E joins the pins name to its path |

**Disabled tests on requirements:** 0 · **Circular patterns:** 0 · **Insufficient assertions:** 0 · **Unasserted arms found this pass:** 0

## Later-phase drift (checked; not this pass's regression)

1. **`os/exec` is in two packages, not one (truth 03-02.6).** Unchanged; plant 16 re-confirms the failure at `boundary_test.go:267`.
2. **The stored corpus is 319 frames across both corpora** (263 render + 56 tui), the same counts as last pass — this round moved no frame and this pass moved none either.
3. **`internal/audit` pins 10 checks** (was 9 last round): `opnames` was added third and inspects one file (the registry), which is why the audit's own file total is 893 while every other count is unchanged. Falsified by plant C, re-run this pass on the frozen tree.
4. **`internal/cli/feed.go`'s predicate is a whitelist now**, so a `*xsource.ParseError` prints the parser's own state instead of `mirror unreadable` (plant D) — a deliberate change of behaviour recorded in the shipped runs' plain arm, which now only ever says `mirror unreadable` for a marked store failure.

## Accepted tech debt (P2/P3 — recorded for the archive, none a must-have failure)

1. **The full `assertSameState` + equal-request-count property (DSGT-08's strongest form) is asserted over `screens()`'s nine Phase 2 screens only.** The saved surface's covered equivalence is asserted by the frame-kind join and by the eight-state saved table — but not by the request-count equality and cover-off byte-identity arms of `TestTheWholeSurfaceUnderACover`. A regression that made the covered saved walk spend an extra request would not fail a test. P3.
2. **The tui corpus scan drops an identity string the corpus's own writers' text holds as a word** (`the identity "Fifteen" is dropped from the scan's targets: the corpus's own writer text holds it as a word`). Principled and commented in the test, but a handle spelled like a content word is scanned only by the render half. P3.
3. **No CI exists**, so "byte-compared in CI" remains a `go test` run, and the tmux harness's session names are per-test: two concurrent `go test ./internal/render` processes collide. Related cosmetic noise: `TestCapturePanic`'s control leg logs two `kill-server … exit status 1` lines on a passing run. P3.
4. **Plan-truth wording drift (03-02.6).** The plan's literal says one `os/exec` package; the landed rule is two (`internal/term` + `internal/media`). The property the must-have exists for — the shell cannot reach `os/exec` — holds. P3.
5. **Process notes carried with the record.** (a) Plants 11 and 12 were run concurrently against the same file two passes ago and its first reading mixed their signatures; they are serial here. (b) Three of this pass's first-attempt plants were malformed (6 named a non-existent test, 10 edited `profile.go` instead of `copy.go`, 9's "plain" clauses were the covered ones) and were corrected before their outcomes were recorded. (c) Plant 14's renamed corpus files had to be cleaned out before later plants, or they mask them with stale-frame errors. No repository bytes were touched by any of this.
6. **The tui arm of the release rule pins the composition, not either clause (new, from the fix round).** `TestASavedFramesCapabilityGoesWhenItsOwnActionSucceeds` (`internal/tui/tui_test.go`) passes with the `SyncFinished` release deleted and with the `MirrorPage` release deleted, because its scenario's walk delivers a page as well as reaching its own end — the page rule alone satisfies it. The clauses are isolated by `internal/screen/saved_test.go` (plants B1, B2), and the tui test's own comment reads a little stronger than its arm can prove. P3.
7. **The `keys.ByKeystroke` comment undercounts its callers (from the fix round).** It says "The two tests that use it"; four test functions do (`TestTheDisguisesKeysResolve`, `TestTheSearchRowsResolve`, `TestEveryKeyRoundTripsThroughTheMatcher`, `TestTheMatcherBindsNothingElse`). The substantive claim — no production caller — is verified. P3.
8. **The guard's source sweep reads one declaration shape (new, from this round).** `dispatch_test.go`'s `actionDecl` regex reads `ActionX Action = "…"` at the start of a line, which is the shape `keys.go`'s const block uses for all 30 actions. A new action declared in another shape — `var ActionX Action = "…"` outside the block — is not read by the sweep, so a declaration named nowhere would pass (plants G1c/G1e). It cannot reach the dispatcher without being named in the shell's sources, and then the refs arm fails it (plants G1d/G1f, `dispatch_test.go:147`), so the hole is inert code rather than a live drift. P3.
9. **The guard pins the join's structure, not an arm's behaviour (new, from this round).** Replacing an arm's *body* with a no-op keeps its case label, so both guard tests stay green while the key stops working (plant G4: `y` returns nil and `TestCopyLink` fails instead). That is the boundary the guard's own comment draws — it exists for the drift the compiler cannot see — and the behavioural tests are the arm for it; recorded so the guard is not read as more than it is. P3.


## Human Verification Required

Both items are recorded and neither is a gap; both name behaviour this pass's environment cannot drive.

### 1. The bare-terminal leg of `TestCapturePanic` (DSGT-02, in a real window)
**Test:** in a Ghostty window with no multiplexer (`TMUX` unset, `TERM_PROGRAM=ghostty`, `WAYLAND_DISPLAY` set, `wtype` and `grim` on `PATH`), run `go test ./internal/render -run TestCapturePanic -count=1 -v` and watch the window.
**Expected:** the bare leg reports it ran; after the panic the window shows the 21 `ok corp/… 0.…s` rows, the title reads `make`, six injected `Shift+Page_Up` reveal nothing of the flood, and the same scroll-up *without* the clear reveals it.
**Why human:** this host has no Ghostty window in the test process's ancestry, so the leg skipped with its printed reason — re-confirmed this pass.

### 2. Focus-driven panic on a real terminal (DSGT-04)
**Test:** in a bare Ghostty window, run the app covered, click another window, come back.
**Expected:** the blur paints the panic frame; returning focus leaves it in place; `ctrl+p` is what leaves.
**Why human:** model-level only here; no terminal in this environment can be blurred and refocused without stealing focus from the user's session.

## Gaps Summary

**No critical gaps.** The one change since the last pass — `press` restructured to one arm per action with the handled set enumerated, plus the guard test — is **behaviour-preserving by two independent measurements**: a differential sweep of every bound keystroke in every dispatcher state (1558 pairs, old vs new, byte-identical) and the shipped binary driven side by side with the pre-refactor binary under tmux (seven captures, byte-identical). The gated cover-breaking keys and the panic gate behave exactly as before, at the model and on the shipped path. Every falsification the last report recorded was re-run and caught (33 of 33), the guard's own plants fail as claimed (5 in the guard test, 1 in the compiler, 1 in the behavioural tests), and no stored frame's bytes moved. Nineteen of nineteen truths hold. The nine accepted tech-debt items above are P2/P3 and recorded for the archive.

## Verification Metadata

**Verification approach:** Goal-backward — the phase goal, the five ROADMAP success criteria, DSGT-01..09/11, and every `must_haves` entry of the three plans, each adjudicated again on the frozen tree; the fix round's six claims re-adjudicated from the code and falsified with their own plants.
**Must-haves source:** the three PLANs' `must_haves` frontmatter, merged with ROADMAP.md's Phase 3 success criteria and REQUIREMENTS.md's DSGT rows; this pass adds the guard's own claim (the keymap⇄dispatcher join) as a behaviour-preservation obligation of the same truths, not as a new truth.
**Automated checks:** `go build`, `go vet`, `gofmt`, `go test ./...` (**1573 tests / 22 packages**), the whole suite again inside `unshare -rn`, the phase suites again inside `unshare -rn`, the two guard tests, the **1558-pair differential sweep** of old vs new `press`, `TestCorpusAudit -v` (1 + 16 subtests, 170 covered frames), `TestCapturePanic -v`, `TestGoldenFrames` over 263 render + 56 tui frames, `TestTheCoveredSavedPaneNamesNothingOfTheProduct -v` and the gate tests by name, `x-terminal audit` (10 checks, 0 findings, 893 files), `diff -rq` of the whole source tree and both `testdata` trees against the previous pass's byte-copy, **40 planted defects each caught by a named test or finding** (2 further plants pass and are recorded as notes), and the shipped binary driven in a hermetic namespace and a private 90×24 tmux pane **twice — this build and the pre-refactor build — with all seven captures byte-identical**.
**Human checks required:** 2 (the bare-Ghostty leg and focus-driven panic) — recorded, neither verified nor a gap.
**Total verification time:** ~70 min.
**What a human would still need to run:** the bare-Ghostty capture leg and the focus-driven panic on a real terminal. Everything else in this report was executed by this pass on this tree.

---

*Verified: 2026-10-15T07:47:08Z*
*Verifier: omp subagent (gsd-verifier persona)*
