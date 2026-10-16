---
phase: 07-media-links-and-clipboard
verified: 2026-10-15T07:22:35Z
status: passed
score: 18/18 must-haves verified
covered_files:
  - .planning/phases/07-media-links-and-clipboard/07-01-PLAN.md
  - .planning/phases/07-media-links-and-clipboard/07-01-SUMMARY.md
  - .planning/phases/07-media-links-and-clipboard/07-02-PLAN.md
  - .planning/phases/07-media-links-and-clipboard/07-02-SUMMARY.md
  - .planning/phases/07-media-links-and-clipboard/07-03-PLAN.md
  - .planning/phases/07-media-links-and-clipboard/07-03-SUMMARY.md
  - internal/cli/feed.go
  - internal/domain/domain.go
  - internal/domain/domain_test.go
  - internal/fake/media.go
  - internal/keys/keys.go
  - internal/keys/keys_test.go
  - internal/media/clipboard.go
  - internal/media/clipboard_test.go
  - internal/media/external.go
  - internal/media/external_test.go
  - internal/media/media_test.go
  - internal/media/renderer.go
  - internal/media/spawn.go
  - internal/render/block.go
  - internal/render/block_test.go
  - internal/render/copy.go
  - internal/render/corpus_test.go
  - internal/render/cover.go
  - internal/render/cover_test.go
  - internal/render/golden_cover_corpus_test.go
  - internal/render/golden_test.go
  - internal/render/grammar.go
  - internal/render/link.go
  - internal/render/link_test.go
  - internal/render/media.go
  - internal/render/media_test.go
  - internal/render/profile.go
  - internal/render/status.go
  - internal/render/status_test.go
  - internal/render/testdata/TestGoldenFrames/color_16_fallback.golden
  - internal/render/testdata/TestGoldenFrames/color_256_fallback.golden
  - internal/render/testdata/TestGoldenFrames/color_mono_fallback.golden
  - internal/render/testdata/TestGoldenFrames/color_truecolor_fallback.golden
  - internal/render/testdata/TestGoldenFrames/media_copied_80.golden
  - internal/render/testdata/TestGoldenFrames/media_copied_wl_80.golden
  - internal/render/testdata/TestGoldenFrames/media_fallback_120.golden
  - internal/render/testdata/TestGoldenFrames/media_fallback_80.golden
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
  - internal/render/testdata/TestGoldenFrames/media_no_opener_all_80.golden
  - internal/render/testdata/TestGoldenFrames/media_none_80.golden
  - internal/render/testdata/TestGoldenFrames/media_opened_80.golden
  - internal/render/testdata/TestGoldenFrames/media_wrap_exact_80.golden
  - internal/render/testdata/TestGoldenFrames/permalink_fallback_80.golden
  - internal/tui/boundary_test.go
  - internal/tui/corpus_test.go
  - internal/tui/cover_e2e_test.go
  - internal/tui/dispatch_test.go
  - internal/tui/links.go
  - internal/tui/media.go
  - internal/tui/model.go
  - internal/tui/state_test.go
  - internal/tui/tui_test.go
  - internal/tui/update.go
  - internal/tui/view.go
  - internal/xsource/tweets.go
  - internal/xsource/xsource_test.go
covered_digest: "v3:sha256:a60da0035f360597408886df020800e41de978e4da08f328b2848dc3f17f8b7a"
behavior_unverified: 0
behavior_unverified_items: []
coincidental_reliance_items: []
re_verification:
  previous_status: passed
  previous_score: 18/18
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 7: Media, Links and Clipboard Verification Report

**Phase Goal:** The user opens a post's images and video in external programs and copies or opens links — with nothing silently reachable, and cover-breaking actions gated in immersive mode.
**Verified:** 2026-10-15T07:22:35Z
**Status:** passed
**Re-verification:** Yes — one change landed after the previous pass: `internal/tui/update.go`'s `press` was restructured so each action has exactly one arm, and a new guard test file `internal/tui/dispatch_test.go` joined the keymap table to the dispatcher from both directions. `diff -rq` against the previous pass's frozen tree (`/tmp/xv7r/pristine`) names **exactly** that change: 1 file modified, 1 file added, 0 removed, 0 golden frames touched. This pass re-adjudicated all 18 must-have truths, re-ran every falsification the previous report records (22 plants: 20 caught, 2 advisories unchanged), added the guard's own plants, ran an absence arm for each absence claim the report makes, and ran a 423-case differential proving the restructure behaviour-preserving. **No code was changed by this pass.**

## What this pass's change was (measured, not claimed)

`diff -rq --exclude=.git --exclude=.serena --exclude=.planning /tmp/xv7r/pristine <repo>` prints two lines and nothing else:

| Path | Change measured |
|---|---|
| `internal/tui/update.go` | the `slices` import; the three multi-action `case` labels in `press` split into one arm per action with byte-identical bodies (`case keys.ActionBookmarks, keys.ActionLikes:` → two arms each `return m.savedAction(action)`; the four cover digits → four `return m.numberedCover(action)`; the eight movement actions → eight `return m.move(action)`); a new `handledActions` slice (update.go:1297, 30 entries) and exported `HandledActions()` (update.go:1315, `slices.Clone`), plus a comment. **The gate, the panic gate, the prompt shadow, the help shadow and the `update` message switch are byte-identical** — they are above the switch and are not in the diff. |
| `internal/tui/dispatch_test.go` | new. Two tests plus a source sweep: `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` (both directions, dedup, count, `HandledActions()` is a copy) and `TestTheShellDeclaresOnlyActionsTheKeymapDeclares` (every action constant keys.go declares is a row's action, a `handledActions` entry and a `case` label in the shell's parsed sources; the shell names no undeclared action). |

No key, no action, no row, no help literal and no golden moved. `internal/keys/keys.go` is **not in the diff at all**, so the `i`/`y`/`o` rows (keys.go:182–184) are byte-identical to the previous pass.

## Impact on Phase 7, change by change

| Changed path | Effect on this phase |
|---|---|
| `internal/tui/update.go` | Covered here. `press`'s exported behaviour is unchanged (proved by differential, below). The three phase-7 arms (`ActionMedia` → `openMedia`, `ActionCopyLink` → `copyLink`, `ActionOpenPost` → `openPost`) are the same calls at update.go:291/293/295; the gate is the same one check at update.go:261 (`coverBreaking` at update.go:341). Every line the previous pass cited inside `press` moved by the added import line and nothing else. |
| `internal/tui/dispatch_test.go` | Covered here (added to the covered set). It is the guard the author described; its own two plants were re-created this pass and both fail (GA, GB below). |
| Everything else | **Not in the diff.** `internal/media/*`, the renderers, the tui media/links/model/view files, `internal/keys/keys.go`, `internal/cli/feed.go` and both corpora are the bytes the previous pass read. |

## Goal Achievement

### Observable Truths

Roadmap SC mapping: SC1←07-01 dispatch; SC2←07-01 seam; SC3←07-02 fallback; SC4←07-03 y/o; SC5←07-03 gate.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 07-01.1 Media key hands every attachment of the selected post to `internal/media` in order, one call per attachment | ✓ VERIFIED | `TestMediaActionOpensAttachments` PASSES this pass (four renderer calls in the fixture's order, keypress itself changes no frame/selection/pending); the differential's `normal|i` case is identical old↔new |
| 2 | 07-01.2 Image→`config.MediaOpener`, video/gif→`mpv --no-terminal`, URL one argv element, no shell | ✓ VERIFIED | `TestOpenDispatchTable` + `TestOpenNoShellArgv` PASS; a shell-joined dispatch is caught (P9: `external_test.go:71`, `:198`) |
| 3 | 07-01.3 Seam is `Open(ctx, ref) (Result, error)` with Outcome + optional rows; v1 ships no in-terminal image code | ✓ VERIFIED | `renderer.go` is not in this pass's diff: `Renderer` (`Open`/`OpenURL`), `Result{Outcome, Rows, Done}`, 9 named outcomes; the only `kitty`/`APC`/`DCS` occurrences in non-test source are `renderer.go`'s own comments (lines 11, 16, 17, 101) — re-grepped this pass — and a Kitty APC escape planted into the fallback path is caught (A1) |
| 4 | 07-01.4 Playable = highest-bitrate `video/mp4` else HLS; no variants → poster + `OpenedPoster` | ✓ VERIFIED | `TestParseMediaPlayableFromFixture` PASSES (fixture, HLS-only, no-`video_info` arms); `dispatch` sets `OpenedPoster` (external.go:113) |
| 5 | 07-01.5 Pane never blocks: one `tea.Cmd`, status from the returned outcome, reaper Wait goroutine | ✓ VERIFIED | `TestOpenDoesNotWaitOnChild` PASSES; a discarded `Wait` is caught by the Done arm (P3, P4) |
| 6 | 07-01.6 `os/exec` only in `internal/term` + `internal/media`; boundary fails on third package / empty scan / planted source | ✓ VERIFIED | Re-grepped: `os/exec` is imported by exactly `internal/term/tmux.go` and `internal/media/spawn.go` in non-test source (`internal/audit/guards.go:77` is a string element of the audit's own `tuiForbidden` list, not an import); a plant into `internal/render/link.go` → FAIL (`boundary_test.go:267`); revert → PASS (P10) |
| 7 | 07-02.1 Failed opener prints the URL in place, wrapped never truncated | ✓ VERIFIED | Missing/`LookPath`/`Start` paths set `Fall.Failed`; `Result.Done` watched by `mediaExitMsg`; `TestOpenStartAndReapNonZeroExitSurfacesOnDone` and `TestMediaLateNonZeroExitBecomesFallback` PASS this pass; re-plants P3/P4 caught |
| 8 | 07-02.2 Split when any entry is URL; prefix 4 cells / continuation 6; no row > W | ✓ VERIFIED | `EntCol = 4` (media.go:12), `UrlCol = 6` (media.go:15); `TestAttachmentFallbackSplitsAndWraps` + `TestFourFailedAttachmentsWrapAtBothWidths` PASS at 80 and 120; truncating-wrap plant caught (P2) |
| 9 | 07-02.3 OSC 8 iff the palette emits escapes; absent in mono; A1 byte-identity after strip | ✓ VERIFIED | `osc8` returns plain text on `RungMono` (media.go:135–137, unchanged); `TestURLFallbackOSC8AndByteIdentity` PASSES; mono-guard plant caught (P1) |
| 10 | 07-02.4 Outcomes are status `notice` vocabulary only | ✓ VERIFIED | `internal/render/copy.go` is not in this pass's diff; the media block (`CopyNoticeMediaOne`/`Many`/`Poster`/`OpenedImage`/`OpenedVideo`/`OpenedGIF`/`OpenedPoster`/`NoOpener`/`NoMedia`) and the REND-07 constants are as the previous pass read them; `TestNoticeKindsHaveWords` + `TestNoticeOfOutcomeIsTotal` PASS |
| 11 | 07-02.5 Fallback survives later keys/frames; cleared only by the next success; hidden under cover, kept in state | ✓ VERIFIED | `TestMediaFallbackClearedOnlyBySuccess` + `TestMediaFallbackHiddenUnderCoverRestores` + `TestCoveredPaneDropsTheFallbackMapByState` PASS; the seam is `fallbacksFor` (model.go:598–603, unchanged this pass); the unconditional-handover plant is caught under all four covers (P11: `tui_test.go:2183`/`:2192`) and, with a cover grammar flipped to print attachment rows, the seam still drops the map (C1: the tui arm passes, the goldens catch the flip) |
| 12 | 07-02.6 No attachments → no attachment line; every attachment failed → one URL entry each | ✓ VERIFIED | `TestMediaActionNoAttachments` PASSES (nil Cmd, zero renderer calls, `no media in this post`); `media_no_opener_all_80` holds four URL entries |
| 13 | 07-03.1 `y` clipboard chain: OSC 52 unless TERM empty/dumb; wl-copy when WAYLAND+PATH; neither → `no clipboard` + print permalink | ✓ VERIFIED | `TestCopyWLCopyOnlyUnderDumbTERM` PASSES; wl-branch plant caught (P5: `clipboard_test.go:252`); dropping the permalink fall is caught (P8) |
| 14 | 07-03.2 Notice names the mechanisms that ran; only the permalink reaches the clipboard | ✓ VERIFIED | `TestCopyPayloadIsPermalinkOnly` + `TestCopyLinkWLOnlyIsLinkCopied` PASS; `Copied`-arm plant caught (P6: `tui_test.go:1896`/`:2072`); an extra payload appended at the call site is caught (A2) |
| 15 | 07-03.3 `o` opens the handle/id or `i/status` permalink via the same opener, one arg, no shell | ✓ VERIFIED | `render.Permalink` unchanged; `TestOpenURLOneArg`/`TestOpenURLMissingOpener`/`TestOpenURLMissingProgram`/`TestOpenURLNonZeroExit`/`TestOpenPost` PASS; permalink-fall plant caught (P8: `tui_test.go:2135`) |
| 16 | 07-03.4 Immersive: media/`y`/`o` bound but refuse — zero renderer/clipboard calls, nil Cmd, one-field frame diff, `NoticeGated` | ✓ VERIFIED | `TestCoverBreakingActionsGated` PASSES this pass (12/12 subtests: 4 covers × 3 keys, recorder counts 0, nil Cmd, one row diff, the cover's own literal, no forbidden word); the gate is the one check at update.go:261 with `coverBreaking` at update.go:341, both byte-identical to the previous pass's bytes; inert-gate plant caught (P7), the gate *deleted* rather than disabled is caught too (A3), and a URL planted into the gated literal is caught by the corpus arm and the goldens (C2: `corpus_test.go:2442`). The differential's four `immersive-*|{i,y,o}` cases are identical old↔new |
| 17 | 07-03.5 Refusal uses grammar `Gated`; every cover non-empty; covered help keeps three rows with cover wording | ✓ VERIFIED | `TestGatedMarkersNonEmpty` PASSES; `TestCoveredHelpKeepsMediaRows` PASSES at 120×40; re-measured live this pass in a scratch probe at 80×24: 22 of the 30 keymap rows plus the cut line `~ 8 more keys than the pane can show`, the three media rows (`i`/`y`/`o`) among the omitted eight (0/3 visible, all four covers — see Anti-Patterns) |
| 18 | 07-03.6 Panic: three actions + help inert; no media/clipboard | ✓ VERIFIED | `TestPanicIgnoresCoverBreakingKeys` PASSES this pass; the differential's `panic|*` cases are identical old↔new (47 keys, every non-`ctrl+p` key a no-op) |

**Score:** 18/18 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/media/renderer.go` | Outcome/Result/Renderer (+ OpenURL) | ✓ EXISTS + SUBSTANTIVE | Not in this pass's diff; 9 outcomes, `Done <-chan error` |
| `internal/media/external.go` | External dispatch + OpenURL | ✓ EXISTS + SUBSTANTIVE | Not in the diff; `target()` shared by dispatch and the printed URL |
| `internal/media/spawn.go` | StartAndReap / RunWithStdin | ✓ EXISTS + SUBSTANTIVE | Not in the diff; only production `os/exec` in media |
| `internal/media/clipboard.go` | OSC52 → wl-copy chain | ✓ EXISTS + SUBSTANTIVE | Not in the diff; wl-only success returns `Copied` |
| `internal/tui/boundary_test.go` | term+media allowlist + plant + anti-vacuity | ✓ EXISTS + SUBSTANTIVE | Live plant failed as required this pass (P10) |
| `internal/keys/keys.go` | `i`/`y`/`o` + HelpCovered | ✓ EXISTS + SUBSTANTIVE | Rows 182–184 with covered wording; the file is not in this pass's diff |
| `internal/tui/{media,links,update,model,view}.go` | Cmds, gate, falls, Done watch, `fallbacksFor` seam | ✓ EXISTS + SUBSTANTIVE | `update.go` is the only one touched (the restructure); gate one check (update.go:261); `fallbacksFor` (model.go:598) |
| `internal/tui/dispatch_test.go` | keymap⇄dispatcher join, both directions + source sweep | ✓ EXISTS + SUBSTANTIVE | New; 30 rows = 30 handled actions = 30 case labels; plants GA/GB caught |
| `internal/render/{media,link,copy,status,cover,block}.go` | Fallback rows, notices, Gated | ✓ EXISTS + SUBSTANTIVE | None in this pass's diff |
| Golden / corpus frames | Media + gated + permalink (45 phase-7 frames of 263) | ✓ EXISTS + SUBSTANTIVE | Byte-identical to all three prior snapshots (G1) |
| `internal/cli/feed.go` | composition root wires the media seam | ✓ EXISTS + SUBSTANTIVE | `Deps.Media.Opener` = `cfg.MediaOpener` (feed.go:268–274); not in this pass's diff |

**Artifacts:** 10/10 fully verified

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| tui media arm | `media.Renderer.Open` | tea.Cmd in `openMedia` | ✓ WIRED | No `os/exec` in tui (boundary scan passes) |
| External | `config.MediaOpener` | feed.go Deps.Media.Opener | ✓ WIRED | Re-read (feed.go:268–274), not in this pass's diff |
| Playable | mpv / fallback URL | `target` / `targetURL` | ✓ WIRED | Same field on both paths |
| Failure outcomes | `render.Fallbacks` | `mediaAnswered` + `mediaExitAnswered` | ✓ WIRED | Start failures and late non-zero exits |
| Display state | Blocks falls arg | `fallbacksFor` zero under cover | ✓ WIRED | model.go:598–603; asserted by `TestCoveredPaneDropsTheFallbackMapByState`; holds even with a cover grammar that prints attachment rows (C1) |
| Dispatcher gate | media/copy/open-post | `coverBreaking` | ✓ WIRED | One check, update.go:261, byte-identical to the previous pass; the differential proves the arm still returns the gate path |
| Gated notice | cover MarkerSet | `StatusFields.Gated` | ✓ WIRED | status.go, not in the diff |
| Permalink fallbacks | `url ` row | `setPermalinkFall` | ✓ WIRED | Both `y` and `o` failure arms |
| Clipboard outcome | notice | `copyLinkAnswered` | ✓ WIRED | `Copied` → `NoticeLinkCopied`; plant caught (P6) |
| keymap table | dispatcher arms | `handledActions` + `dispatch_test.go` | ✓ WIRED | 30 rows / 30 entries / 30 case labels; both plants caught (GA, GB) |

**Wiring:** 10/10 connections verified

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| Open target URL | `ref.Playable`/`ref.URL` | xsource variants / poster | Fixture + decoder | ✓ FLOWING |
| Clipboard payload | `render.Permalink(post)` | Post ID + author handle | Built at the call site | ✓ FLOWING |
| Fallback print URL | `targetURL(ref)` / `Fall.Link` | Same as the opener target / permalink | Goldens + plants | ✓ FLOWING |
| Gated words | `grammar.Markers.Gated` | `cover.go` literals | Status line | ✓ FLOWING |
| Late exit error | `Result.Done` ← `cmd.Wait` | StartAndReap reaper | TUI `mediaExitMsg` | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build/vet/format | `go build ./...`; `go vet ./...`; `gofmt -l .` | exit 0, no output, no unformatted file | ✓ PASS |
| Full suite | `go test ./... -count=1 -json` (XDG_* pointed at `/tmp/xv8/*`) | **1573** tests passed (694 top-level + 879 subtests) / **22** packages, 0 FAIL, 8 skips (the capture tests that need a real terminal/network) — 2 more top-level tests than the previous pass, the guard's two | ✓ PASS |
| Phase suites netless | `unshare -rn` + `ip link set lo up`: media, render, tui, keys, xsource, domain, fake, cli, config, term | all `ok` | ✓ PASS |
| Named Phase-7 tests | the previous pass's exact `-run` set over media/render/tui/xsource, plus the two guard tests | 59 PASS (51 top-level + 8 subtests), 0 FAIL; the PASS/Fail line set is **identical to the previous pass's record** (`named3.out`) plus the guard's two lines | ✓ PASS |
| Gate arm alone | `go test ./internal/tui -run TestCoverBreakingActionsGated -v` | 12/12 subtests PASS (4 covers × 3 keys) | ✓ PASS |
| Playable-variant arm | `go test ./internal/xsource -run ParseMedia -count=1` | `TestParseMediaPlayableFromFixture` PASS | ✓ PASS |
| Committed covered-frame scan (render) | `TestCorpusAudit`'s DSGT-08 no-http arm | reads **170** covered frames (154 `cover_*` + 16 `media_gated_*`), none carries `http`; the count is what the arm reports when N1b/C-plant fails | ✓ PASS |
| Committed covered-frame scan (tui) | `TestTheCoveredTUIFramesLeakNoPlainVocabulary` + `TestTheCoveredTUIFrameLeakScanCatchesAPlant` | 24 covered frames of 56, each carrying its own alias, none carrying a plain word; the planted row is named by the scan (N2) | ✓ PASS |
| Broadened repair-word scan | `TestCoveredFramesLeakNoRepairWords` + `TestCoveredLeakScanCatchesAPlant` | scans every covered frame (170) and asserts `scanned >= len(covered)`; both plant bases report their plant | ✓ PASS |
| Phase-7 frame byte identity (G1) | `cmp` of the 45 `media_*`/`permalink_*`/`color_*` frames against three snapshots; `diff -rq` of both whole corpora | render corpus and tui corpus **byte-identical to the previous pass's frozen tree** (0 differences); the 45 frames identical across `/tmp/xv7d` (pre-amendment), `/tmp/xv7w/pristine` (post-amendment) and now | ✓ PASS |
| Covered media frame literal scan | lowercase scan of the 16 `media_gated_*` frames for http/https, x-terminal, twitter, browser, clipboard, wl-copy, xdg-open, mpv, attachment, pbs.twimg, aergon, followers:/following:/posts:/joined:, mirror, saved.db, unreadable, @lzd | CLEAN ×16 | ✓ PASS |
| Covered help at the phase's primary width | scratch probe against a `/tmp` copy: 80×24, `?` under each cover | 23 overlay lines: 22 of 30 keymap rows + the cut line `~ 8 more keys than the pane can show`; media rows 0/3 visible, all four covers | ✓ PASS (note, not a truth) |
| Audit gate | `x-terminal audit` (built from this tree, run at the repo root, XDG_* to `/tmp`) | **10** checks, 0 findings, 893 files inspected; exit 0 | ✓ PASS |
| Tree untouched | `diff -rq --exclude=.git --exclude=.serena --exclude=.planning` repo vs the copy made at the start of this pass, after every plant and probe | identical (0 differences) | ✓ PASS |

## Falsification Experiments

Plants ran in `/tmp/xv8/plant`, a copy of the frozen tree remade from `/tmp/xv8/pristine` before each one; the repository itself was never edited. The statements below are the failure output actually observed this pass.

### The guard the author described (new this pass)

| # | Plant | Arm it must trip | Observed failure | Caught |
|---|-------|------------------|------------------|--------|
| **GA** | A **31st keymap row** with a fresh action (`ActionGuardRow = "guard-row"`, key `Z`), no handler, no `handledActions` entry | the join, both directions | `dispatch_test.go:63: the keymap binds "guard-row" (Z) and press has no arm for it: the key is a documented no-op`; `:79: the table holds 31 rows and handledActions names 30 actions`; `:128: keys.go declares ActionGuardRow and handledActions does not name it`; `:159: keys.ActionGuardRow is no case label in the shell` | ✓ |
| **GB** | A **handler with no row** (`ActionGuardHandler`, a `case` arm in `press` returning `m, nil`, named in `handledActions`, no keymap row) | the join, both directions | `dispatch_test.go:72: press handles "guard-handler" and no row of the keymap binds it: the arm is unreachable`; `:79: the table holds 30 rows and handledActions names 31 actions`; `:125: keys.go declares ActionGuardHandler ("guard-handler") and no row binds it` | ✓ |
| **GC** | `case keys.ActionThisPackageDeclaresNoSuchAction:` planted into `press` (the property the guard's comment leans on) | the compiler | `vet: internal/tui/update.go:297:12: undefined: keys.ActionThisPackageDeclaresNoSuchAction` — the exact message the comment quotes | ✓ |

### The restructure is behaviour-preserving (new this pass)

| # | Experiment | Result |
|---|-----------|--------|
| **D1** | One probe test (`fingerprint`: rendered frame + every model/frame field + fallback map, hashed) compiled into **both** the previous pass's `update.go` and the current one; every key the keymap binds (plus 12 it does not) pressed in 9 states — normal, four covers, panic, help, prompt, immersive-help | **423 (state×key) cases, byte-identical old↔new.** 47 keys × 9 states, including `i`/`y`/`o` in all four covers and normal, `esc`/`q`/`ctrl+p`/`ctrl+c` in panic, the five prompt keys under the shadow, and `?` under normal and a cover |
| **D2** | Positive control: `case keys.ActionOpen:` changed to `return m, nil` (a real behaviour change) | **10 cases differ** — the probe would have seen the restructure if it had moved behaviour |
| **D3** | Negative control: the new split movement arms re-merged into the old multi-label `case` shape (semantics-equal) | **identical** to the current tree's output — the old multi-label form and the split arms are behaviourally equal, which is what the restructure claims |

### The previous pass's recorded plants, re-run against the current bytes

| # | Plant | Arm it must trip | Observed failure | Caught |
|---|-------|------------------|------------------|--------|
| P1 | `osc8` mono guard removed | `TestURLFallbackOSC8AndByteIdentity` | `media_test.go:212: mono/dumb frame carries escapes` | ✓ |
| P2 | `wrapURL` truncated to one row | `TestAttachmentFallbackSplitsAndWraps`, `TestLongURLWrapsWithoutLoss` | `media_test.go:107: at 80: no continuation row …`; `media_test.go:194: 4000-cell URL lost characters: got 73 want 4024` | ✓ |
| P3 | `StartAndReap` discards `Wait`'s error | `TestOpenStartAndReapNonZeroExitSurfacesOnDone` | `external_test.go:262: Wait error is nil; false must exit non-zero` | ✓ |
| P4 | The Done watch inverts its nil check | `TestMediaLateNonZeroExitBecomesFallback` | `tui_test.go:1869: notice = 22, want NoticeNoOpener after late non-zero exits` | ✓ |
| P5 | wl-only success → `NoClipboard` | `TestCopyWLCopyOnlyUnderDumbTERM` | `clipboard_test.go:252: TERM="dumb": outcome = no clipboard, want Copied (wl-only success)` | ✓ |
| P6 | `media.Copied` arm dropped from `copyLinkAnswered` | `TestCopyLinkWLOnlyIsLinkCopied`, `TestCopyLink` | `tui_test.go:1896: notice=28, want NoticeLinkCopied (wl-only success)`; `tui_test.go:2072: notice=28, want NoticeLinkCopied` | ✓ |
| P7 | Immersive gate made inert | `TestCoverBreakingActionsGated` (12 subtests) | `tui_test.go:1952: renderer called under cover: open=4 url=0`; `tui_test.go:1955: clipboard called under cover: 1`; `open=0 url=1` | ✓ |
| P8 | `setPermalinkFall` a no-op | `TestCopyLink`, `TestOpenPost` | `tui_test.go:2090/:2135: fallback={Failed:[] Permalink:false Link:} ok=false` | ✓ |
| P9 | `dispatch` joins `sh -c "<opener> <url>"` | `TestOpenDispatchTable`, `TestOpenNoShellArgv` | `external_test.go:71: calls = […name:"sh", args:[]string{"-c", "xdg-open https://…"}]`; `external_test.go:198: argv = []string{"-c", …}` | ✓ |
| P10 | `_ "os/exec"` planted in `internal/render/link.go` | `TestProcessSpawnsAreOnlyInTermAndMedia` | `boundary_test.go:267: internal/render/link.go imports os/exec, and only internal/term and internal/media may` | ✓ |
| P11 | `fallbacksFor` hands the map over unconditionally | `TestCoveredPaneDropsTheFallbackMapByState` | `tui_test.go:2183: under {servicelog,gitlog,ci,internal} the pane is handed a 1-entry fallback map, want the zero map`; `:2192` the URL-bearing rows | ✓ |
| C1 | servicelog grammar's `AttachmentRow` flipped `true` (the accident the seam's doc names) | the seam's arm must **not** fail; the corpus must | `ok internal/tui` (the seam holds), `golden_test.go:352` ×6 on covered frames | ✓ (rule holds) |
| C2 | A URL planted into the servicelog gated literal | render goldens + the gated-corpus arm (+ the tui gate arm) | `corpus_test.go:2442: frame media_gated_servicelog_80 missing gated literal "open denied by local policy https://x.com/lzd/status/1"`; `golden_test.go:352` ×4 | ✓ |
| N1a | `mpv wl-copy xdg-open` appended to a re-blessed 80-wide covered media frame's status row | `TestCorpusGatedMediaFrames`' forbidden-word arm | `corpus_test.go:2459: gated frame media_gated_servicelog_80 holds "wl-copy" / "xdg-open" / "mpv"`; `golden_test.go:352` | ✓ |
| N1b | A `pbs.twimg.com` URL appended to a re-blessed 120-wide covered media frame | the DSGT-08 no-http arm | `corpus_test.go:739: covered frame media_gated_gitlog_120 contains http` (`_tmux` too); `:745: no-http arm read 170 covered frames`; `golden_test.go:352` | ✓ |
| N1c | An arbitrary (`media.invalid`) URL appended to an 80-wide gated frame | `TestCorpusGatedMediaFrames`' URL arm **alone** | **nothing from that arm** (`ok`), i.e. it compares the frame against the *fixture's own* ref URLs; the DSGT-08 no-http arm catches the plant instead | ✗ (advisory, below — unchanged) |
| N2 | `mirror saved.db doctor --pin` appended to a covered tui people frame | the tui scanner | `corpus_test.go:831: the covered tui frame search_people_gitlog_80 holds "mirror" / "saved.db" / "doctor"`; `golden_test.go:345` | ✓ |
| B1 | The three saved-pane constructors (view.go's `savedPane`) handed `render.Plain` again | `TestTheCoveredSavedPaneNamesNothingOfTheProduct` | `cover_e2e_test.go:1467: the covered servicelog frame holds "doctor" / "bookmarks", a word the cover did not author`; `:1476: does not carry … the cover's own words` | ✓ |
| B2 | `keys.ActionFirst` no longer special-cased | `TestGoRestoresTheProfileHeader` | `profile_test.go:334: `g` left the frame moved: the pane is still the anchor's and not the frame's own first row` | ✓ |
| B3b | gitlog cover's `AccountFacts` cleared | `TestGoldenFrames` (tui) | `golden_test.go:345: output does not match` ×2 (`profile_gitlog_{80,120}`) | ✓ |
| B3a | `profileFacts`' `CoverOfGrammar` guard removed | — | **nothing fails**: render, tui and cli all `ok` | ✗ (advisory, below — unchanged) |
| N3 | The stale `ListLatestTweetsTimeline` entry restored into `internal/audit/readonly.go` | the `opnames` check | `audit_test.go:156: opnames: internal/audit/readonly.go:0: the table classifies "ListLatestTweetsTimeline" as a read and the registry does not hold it` | ✓ |

### Absence arms this pass added

| # | Absence claim the report makes | Plant | Caught by |
|---|---|---|---|
| **A1** | v1 ships no in-terminal image code (truth 3) | a Kitty APC graphics escape (`\x1b_Ga=T,f=100;…\x1b\\`) prepended to the fallback text in `internal/render/media.go` | `media_test.go:228: after stripping escapes, colour != mono`; `golden_test.go:352` ×3 | ✓ |
| **A2** | only the permalink reaches the clipboard (truth 14) | an attachment URL appended at `internal/tui/links.go`'s `url := render.Permalink(post)` call site | `tui_test.go:1893: clipboard = 1/"<permalink> https://pbs.twimg.com/media/E1.jpg", want 1/"<permalink>"`; `TestCopyLink` (`tui_test.go:2069`) | ✓ |
| **A3** | cover-breaking actions are refused (truth 16) | the gate **deleted** (not merely made inert) from `press` | `tui_test.go:1952: renderer called under cover: open=4 url=0`; `:1955: clipboard called under cover: 1` | ✓ |

**G1 — the phase's own frames are untouched.** The whole render corpus (263 frames, 16 `media_gated_*`, 45 phase-7 frames) and the whole tui corpus are byte-identical to the previous pass's frozen tree (0 differences), and the 45 phase-7 frames are identical across the pre-amendment snapshot (`/tmp/xv7d`), the post-amendment snapshot (`/tmp/xv7w/pristine`) and now. The 16 `media_gated_*` frames re-scan CLEAN for every word they must not carry (N1a/N1b show the arms that would notice are live). **This pass's change moved no stored frame.**

## The dispatcher restructure, adjudicated

The assignment asked one thing about the change: confirm it is behaviour-preserving for the flows that depend on `press`. What was checked, and how:

1. **The diff itself.** Only case labels changed shape: three multi-action labels split into one arm per action, each body byte-identical (`return m.savedAction(action)`, `return m.numberedCover(action)`, `return m.move(action)`). Go gives `case A, B: body` and `case A: body; case B: body` the same semantics; D3 confirms the re-merge (old shape) and the split produce identical observable output.
2. **A 423-case differential (D1).** The same probe compiled over the previous pass's `update.go` and this one; every bound key in nine states — including the two arms the assignment names, the keymap⇄model join's own keys, the gated actions under each cover, the prompt's five keys, the panic gate's two keys, and the frame push/pop keys (`enter`/`q`/`esc`) — produced identical hashes of the rendered frame, the frame stack, the selection/anchor, the fallback map, the notice, `pending` and `seq`. A real behaviour change (D2, `ActionOpen` → no-op) shows up in 10 cases, so the probe is not blind.
3. **The named arms re-run.** `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` and `TestTheShellDeclaresOnlyActionsTheKeymapDeclares` PASS (the keymap⇄dispatcher exhaustion); `TestCoverBreakingActionsGated` 12/12; `TestPanicIgnoresCoverBreakingKeys` and the panic/blur tests PASS; the prompt tests (`TestSearch*`, `TestPeople*`) PASS; `TestTheBackStackRestoresTheFrame` and `TestAgnosticKeyScriptOverPostsAndAccounts` PASS (frame pushes/pops).
4. **The restructure's claim about itself.** The enumeration is 30 entries, the table 30 rows, the shell 30 case labels over actions; the guard's two plants (GA, GB) both fail, and a bogus case label is refused by the compiler with the message the source comment quotes (GC).

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| MEDIA-01 | 07-01 | External openers for image/video/gif | ✓ SATISFIED | Dispatch + tui media arm (P9) |
| MEDIA-02 | 07-01 | Renderer seam for one-file Kitty later | ✓ SATISFIED | `Renderer` shape + rows half; no in-terminal v1 code (A1) |
| MEDIA-03 | 07-02 | Raw URL when opener unavailable | ✓ SATISFIED | Missing/start-failure paths, plus late non-zero via `Result.Done`/`mediaExitMsg` (P3, P4) |
| REND-07 | 07-03 | Copy link (OSC52/wl-copy) + open post | ✓ SATISFIED | `y`/`o` with wl-only → `Copied` + `NoticeLinkCopied` (P5, P6, P8, A2) |
| DSGT-10 | 07-03 | Cover-breaking actions unavailable in immersive | ✓ SATISFIED | One gate + panic inert + covered help + state-rule `fallbacksFor` (P7, P11, C1, C2, A3) |

**Coverage:** 5/5 requirements satisfied

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/render/corpus_test.go | 2444–2454 | `TestCorpusGatedMediaFrames`' URL arm compares a gated frame against the fixture's own `MediaRef` URLs, so a URL that is *not* one of them passes that arm (N1c, re-run this pass); the catch-all is the DSGT-08 no-http arm, which does catch it | ℹ️ Info | Not a coverage hole today — the DSGT-08 arm reads all 170 covered frames for `http` and the gate arm's forbidden-word list has `http` too — but the arm's name promises more than it checks, and a future non-`http` scheme (e.g. `magnet:`) would have no arm here |
| internal/render/profile.go | 79–85 | The `CoverOfGrammar` guard inside `profileFacts` is unreachable with the shipped covers (all four set `AccountFacts`): removing it fails no test (B3a, re-run this pass), while removing it *and* a cover's clauses fails the tui goldens | ℹ️ Info | A defensive branch in a file this phase does not own: the rule it states is held by the four covers' own clauses (B3b), not by the guard |
| internal/tui/testdata/TestGoldenFrames/profile_*.golden, search_*.golden | body rows | 27 covered frames carry `https://example.invalid/…` | ℹ️ Info | It is a post's own body text in the corpus fixture, byte-identical through every round — not a product literal and not a fallback row (no tui golden holds `no opener`, i.e. none carries a media fallback row). The DSGT-08 no-http arm is scoped to the covered render corpus by design; a naive `http` grep over the tui corpus flags these plain-content rows too |
| internal/render/help.go (with internal/tui/view.go) | help cut row | At 80×24 a covered `?` overlay shows 22 of 30 rows plus `~ 8 more keys than the pane can show`; re-measured live this pass in a `/tmp` probe, the three media rows (`i`/`y`/`o`) are among the omitted eight (0/3 visible, all four covers) | ℹ️ Info | Not a regression of truth 17, whose arm asserts at 120×40. But the covered wording of the three keys is not visible at the phase's primary width |

**Anti-patterns:** 4 found (0 blockers, 0 warnings, 4 info)

### Advisory (New Scope, Unevidenced)

Two, both about arms rather than product behaviour, and neither a Phase-7 gap; both re-run this pass and unchanged:

1. **`internal/render/profile.go`'s cover guard is not falsifiable with the shipped covers** (B3a: removing it fails nothing; removing it *and* a cover's clauses fails the tui goldens). A file this phase does not own; the outcome it defends is pinned by the tui goldens, the branch itself is not. Recorded for the milestone.
2. **`TestCorpusGatedMediaFrames`' URL arm is narrower than its name** (N1c): it looks only for the fixture's own attachment URLs, so an arbitrary URL in a covered media frame is caught by the DSGT-08 arm and the stored bytes instead. Worth tightening when the corpus is next touched; not a defect in the shipped behaviour.

## Human Verification Required

None — all verifiable items checked programmatically. The two optional smokes stand as before and are not required for status:

### 1. Live opener smoke (optional)
**Test:** On a normal-mode feed post with image+video, press `i`, then `y`, then `o`.
**Expected:** Image opens via the configured opener, video via mpv; the clipboard receives only the permalink; the browser opens the permalink.
**Why human:** Real desktop opener/`wl-copy`/terminal OSC 52 behaviour.

### 2. Immersive refusal (optional)
**Test:** Press `m`, then `i`/`y`/`o`.
**Expected:** Cover `Gated` notice only; no external windows; no clipboard change.
**Why human:** Confirms end-to-end with a real clipboard manager watching.

## Gaps Summary

**No gaps found.** All 18 must-have truths hold on the frozen tree. This pass's change — the `press` restructure plus the keymap⇄dispatcher guard — is behaviour-preserving, and this was measured rather than assumed: the whole code diff since the previous pass is two files (`update.go`, `dispatch_test.go`), a 423-case differential over nine states and every bound key is byte-identical between the old and new `press` with both a positive control (a real change shows in 10 cases) and a negative control (the old multi-label shape re-merged is identical), no key, row, action, help literal or golden moved, and both corpora are byte-identical to the previous pass's frozen tree. `i`/`y`/`o` route exactly as before: the same three arms (update.go:291/293/295) behind the same one gate at update.go:261, with `keys.go`'s rows untouched (not in the diff at all). Every falsification the previous report records was re-run against the current bytes with the same outcomes — 20 caught, the two advisories (N1c, B3a) unchanged — plus the guard's own two plants (GA, GB) and three absence arms (A1–A3), all caught, and the compiler-refusal property the guard's comment leans on (GC). Two recorded advisories stand: N1c and B3a, both P3 notes, not gaps. The digest was recomputed over **77** covered paths (70 implementation files + this phase's 6 PLAN/SUMMARY files + `internal/tui/dispatch_test.go`, **the one path added this pass** — the guard is a new file the report cites as an artifact).

### Closed Gaps (Prior Verifications)

1. **Wl-only clipboard success reported as `no clipboard`** — `Clipboard.Copy` returns `Copied` when `wl && !wrote`; re-planted and caught (P5, P6).
2. **Production reaper cannot surface a non-zero opener exit** — `StartAndReap` returns `Wait` on a channel; `Result.Done` watched by `mediaExitMsg`; re-planted and caught (P3, P4).
3. **Notice lied after wl-only success** — `copyLinkAnswered` maps `Copied` → `NoticeLinkCopied`; re-planted and caught (P6).
4. **(This pass, recorded for the milestone)** Nothing from the previous round reopened; the dispatcher restructure introduces no gap — its own guard's plants fail in both directions (GA, GB) and the differential shows no behaviour moved.

## Falsification Attempts (attacks that did not find gaps)

- Shell / string-joined command lines in media: not found; argv-slice only (P9).
- Third `os/exec` package: not present; a plant in `internal/render/link.go` is caught by the boundary scan (P10). The only other non-test mention of the string is `internal/audit/guards.go:77`, an element of the audit's own `tuiForbidden` list — not an import.
- In-terminal image code in v1: not found; a Kitty APC escape planted into the fallback path is caught (A1).
- Immersive gate still calling the renderer or clipboard: not found; an inert gate is caught (P7), a *deleted* gate is caught (A3), and so is a URL in the gated literal (C2).
- Clipboard receiving anything but the permalink: not found; an extra payload at the call site is caught (A2).
- Truncated fallback URL: not found (P2).
- OSC 8 in the mono rung: not found (P1).
- Covered pane handed the fallback map: not found (P11) — and the rule survives a cover grammar that *does* print attachment rows (C1), which is the accident the seam's documentation is about.
- Covered media frame carrying a product literal or a URL: not found — the 16 covered media frames are byte-identical to both snapshots, they scan clean for every forbidden word, the committed scan reads all 170 covered frames, and plants into real frames were caught by three different arms (N1a, N1b, C2).
- Covered tui frame carrying a plain word: not found, and the arm is non-vacuous — the scanner reads all 24 covered tui frames, requires each to carry its own alias, and names the planted row by frame and word (N2).
- Vacuous planted-violation boundary test: the anti-vacuity arms plus the live plant failure prove the scan is non-vacuous.
- The `opnames` audit check being vacuous: not the case — restoring the deleted stale entry makes it report (N3).
- The restructure moving behaviour: not the case — 423 differential cases identical (D1), with a positive (D2) and a negative (D3) control.
- A key bound with no dispatcher arm, or an arm no key reaches: not the case in the shipped tree, and the guard that says so is non-vacuous (GA, GB).
- The claim that the change is only `update.go` + `dispatch_test.go`: measured — `diff -rq` against the previous pass's frozen tree prints exactly two files and no `testdata/` path.

## Verification Metadata

**Verification approach:** Goal-backward re-verification from the prior report's must-haves + ROADMAP Phase 7 success criteria + the three PLAN `must_haves` (SUMMARYs untrusted)
**Must-haves source:** prior 07-VERIFICATION.md must-haves (re-verification) + 07-01/02/03-PLAN.md frontmatter + ROADMAP Phase 7 success criteria
**Covered files added this pass:** `internal/tui/dispatch_test.go` — the new guard file, cited as an artifact. Nothing removed (77 paths total, all present).
**Automated checks:** `go build`/`go vet`/`gofmt` clean; `go test ./... -count=1` → 1573 tests / 22 packages, 0 fail, 8 skips; ten packages pass under `unshare -rn` with `lo` up; the previous pass's named Phase-7 `-run` set plus the two guard tests → 59 PASS / 0 fail (identical to the previous record); the gate arm 12/12 subtests; the 22 recorded plant experiments re-run (20 caught, 2 advisories unchanged) plus the guard's 2 plants, 3 absence arms and 1 compile-refusal plant (all caught) plus the 3 differential runs (423 cases identical; controls differ/identical as required)
**Human checks required:** 0 (optional smoke only)
**Could not verify:** Live desktop `xdg-open`/`mpv`/`wl-copy` against a real compositor; the historical Kitty/tmux smear measurements in `.planning/research/TERMINAL-MEDIA-PROBE.md` (accepted as research, not re-measured); MVP User Flow Coverage (the goal is not in User Story form); the `internal/audit` change's own arms beyond the gate run plus N3 (that surface belongs to the phase that owns it). The differential probe (D1–D3) is a scratch file in `/tmp`, not a committed test: it hashes model state and the rendered frame, so it cannot see a difference that changes neither. A human would need a real desktop, a Wayland session with `wl-copy` and a terminal that honours OSC 52 to exercise the two optional smokes.

---

*Verified: 2026-10-15T07:22:35Z*
*Verifier: Claude (subagent)*
