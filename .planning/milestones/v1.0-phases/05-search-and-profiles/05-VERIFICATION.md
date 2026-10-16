---
phase: 05-search-and-profiles
verified: 2026-10-15T07:24:07Z
status: passed
score: 24/24 must-haves verified
covered_files:
  - .planning/phases/05-search-and-profiles/05-01-PLAN.md
  - .planning/phases/05-search-and-profiles/05-01-SUMMARY.md
  - .planning/phases/05-search-and-profiles/05-02-PLAN.md
  - .planning/phases/05-search-and-profiles/05-02-SUMMARY.md
  - .planning/phases/05-search-and-profiles/05-03-PLAN.md
  - .planning/phases/05-search-and-profiles/05-03-SUMMARY.md
  - .planning/phases/05-search-and-profiles/05-CONTEXT.md
  - .planning/phases/05-search-and-profiles/05-UI-SPEC.md
  - internal/domain/profile.go
  - internal/domain/source.go
  - internal/fake/parity_test.go
  - internal/fake/profile.go
  - internal/fake/search.go
  - internal/fake/testdata/pages-profile.json
  - internal/fake/testdata/pages-search-people.json
  - internal/fake/testdata/pages-search.json
  - internal/keys/keys.go
  - internal/keys/keys_test.go
  - internal/render/copy.go
  - internal/render/corpus_test.go
  - internal/render/cover.go
  - internal/render/golden_test.go
  - internal/render/grammar.go
  - internal/render/grammar_phase5_test.go
  - internal/render/profile.go
  - internal/render/profile_test.go
  - internal/render/prompt.go
  - internal/render/prompt_test.go
  - internal/render/saved.go
  - internal/render/status.go
  - internal/render/user.go
  - internal/screen/agnostic_test.go
  - internal/screen/profile.go
  - internal/screen/profile_test.go
  - internal/screen/prompt.go
  - internal/screen/prompt_test.go
  - internal/screen/saved.go
  - internal/screen/search.go
  - internal/screen/search_test.go
  - internal/screen/walk.go
  - internal/text/drop.go
  - internal/tui/agnostic_test.go
  - internal/tui/corpus_test.go
  - internal/tui/cover_e2e_test.go
  - internal/tui/dispatch_test.go
  - internal/tui/golden_test.go
  - internal/tui/model.go
  - internal/tui/people_test.go
  - internal/tui/profile_test.go
  - internal/tui/saved.go
  - internal/tui/search_test.go
  - internal/tui/testdata/TestGoldenFrames/profile_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_200.golden
  - internal/tui/testdata/TestGoldenFrames/profile_60.golden
  - internal/tui/testdata/TestGoldenFrames/profile_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_bare_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_bio_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_ci_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_ci_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_color_16.golden
  - internal/tui/testdata/TestGoldenFrames/profile_color_256.golden
  - internal/tui/testdata/TestGoldenFrames/profile_color_mono.golden
  - internal/tui/testdata/TestGoldenFrames/profile_color_truecolor.golden
  - internal/tui/testdata/TestGoldenFrames/profile_empty_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_gitlog_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_gitlog_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_internal_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_internal_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_loading_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_quote_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_repost_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_scrolled_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_servicelog_120.golden
  - internal/tui/testdata/TestGoldenFrames/profile_servicelog_80.golden
  - internal/tui/testdata/TestGoldenFrames/profile_tombstone_80.golden
  - internal/tui/testdata/TestGoldenFrames/prompt_120.golden
  - internal/tui/testdata/TestGoldenFrames/prompt_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_200.golden
  - internal/tui/testdata/TestGoldenFrames/search_60.golden
  - internal/tui/testdata/TestGoldenFrames/search_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_ci_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_ci_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_color_16.golden
  - internal/tui/testdata/TestGoldenFrames/search_color_256.golden
  - internal/tui/testdata/TestGoldenFrames/search_color_mono.golden
  - internal/tui/testdata/TestGoldenFrames/search_color_truecolor.golden
  - internal/tui/testdata/TestGoldenFrames/search_empty_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_failure_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_gitlog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_gitlog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_internal_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_internal_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_ci_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_ci_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_gitlog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_gitlog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_internal_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_internal_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_servicelog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_people_servicelog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_servicelog_120.golden
  - internal/tui/testdata/TestGoldenFrames/search_servicelog_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_stalled_80.golden
  - internal/tui/testdata/TestGoldenFrames/search_top_80.golden
  - internal/tui/tui_test.go
  - internal/tui/update.go
  - internal/tui/view.go
  - internal/xsource/capture_test.go
  - internal/xsource/profile.go
  - internal/xsource/profile_test.go
  - internal/xsource/search.go
  - internal/xsource/search_test.go
  - internal/xsource/testdata/fixture-profile-header.json
  - internal/xsource/testdata/fixture-profile-posts.json
  - internal/xsource/testdata/fixture-search-people.json
  - internal/xsource/testdata/fixture-search.json
  - internal/xsource/testdata/provenance.json
  - internal/xsource/tweets.go
covered_digest: "v3:sha256:b0499f21f26c78ab977511f770954db80aec8b39a0d1c74451768c8803ff9b2c"
behavior_unverified: 0
behavior_unverified_items: []
coincidental_reliance_items: []
re_verification:
  previous_status: passed
  previous_score: 24/24
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  this_round: >-
    the dispatcher's `press` was restructured to one arm per action with an enumerable
    `handledActions` set, and `internal/tui/dispatch_test.go` was added to hold the join; no
    action was added, no key changed and no golden was regenerated, and the restructure is
    behaviour-preserving on every flow this phase depends on (re-measured by plants
    W1-W7/X/Y/Z/BK/TT/SL and the phase suites)
  deferred:
  - truth: "05-03 human checkpoint: approve TestShowNewFrames frames"
    addressed_in: "still DEFER-only; the frame it defers (profile_80) draws the header, so the checkpoint is worth running"
    evidence: "05-03-SUMMARY deferred operator approval 2026-10-15; the corpus is byte-equal to the shipped render (TestGoldenFrames)"
---
# Phase 5: Search and Profiles Verification Report

**Phase Goal:** The user searches posts by keyword and finds people, then reads a profile's bio and recent posts through the same list they already know.
**Verified:** 2026-10-15T07:24:07Z
**Status:** passed
**Re-verification:** Yes — the final milestone-archive re-check over the frozen tree. Since the previous pass the only change is one behaviour-preserving guard: `internal/tui/update.go`'s `press` was restructured so each action has exactly one arm and the set it handles is enumerable (`handledActions`, plus the exported `HandledActions()`), with `internal/tui/dispatch_test.go` asserting the keymap⇄dispatcher join from both directions plus a source sweep over the action declarations. No action was added, no key changed, no golden was regenerated. Every flow this phase depends on still routes exactly as before: the exhaustion arm (`TestEveryBoundActionIsHandledAndEveryHandledActionIsBound`), the panic gate, the help overlay's shadow, the cover gate, the prompt's shadow, and `s`/`u`/`a`/`enter`/`g` — each pinned by removing its arm (plants W1–W7) and by the phase suites, all PASS; the 56-frame golden byte-equality guard is unchanged. Before it, for the record, came the fix round that touched this phase's ground: the saved frame's `mirror unreadable` state became a whitelist (`internal/store`'s `FileError`, `internal/cli/feed.go`'s predicate), a capability can now be released (`internal/screen/saved.go`, `internal/tui/saved.go`), and this phase's own `internal/tui/cover_e2e_test.go` had to move its "store cannot be read" scenario onto a reachable pair (a failing read plus the failing map) because a mirror that answered while the frame wore the state is a pair the shipped path cannot produce. Nothing in the profile or search path changed. Every claim below was re-measured against this tree: the full suite (1573 passing, 0 FAIL — the two new arms of `dispatch_test.go` and nothing else), the phase suites (1033 passing), the audit (10 checks, 0 findings, 893 files), all 22 falsifications the previous pass recorded (re-run verbatim from its own `plants2.sh`, all refused at the same sites — only plant O's run moved from the `screen` package to `tui`, where its assertion lives) plus one new plant of mine for the fix round's own scenario (plant T, and the pristine-copy-only probe files were removed), fifteen new plants for this round's restructure — its own two (U, V: a 31st keymap row with a fresh action, and a handler for an action no row binds, both refused), the phase's keys and the two movement arms (W1–W7: `s`, `u`, `a`, `enter`, the prompt shadow, `j`, `g`), the three other shadows/gates (X, Y, Z), and `q`/`t`/`/` (BK, TT, SL) — a bogus case label (proving the compiler refuses one, the property `shellActionCases` leans on), and the three probes of my own (the covered facts row over all five grammars, `g`/`home` at 12 sizes × 2 keys (24/24), the in-flight instrument). **The dispatcher refactor is behaviour-preserving**: `press`'s arms dispatch the same actions to the same methods in the same order (panic gate → prompt shadow → keymap lookup → help shadow → cover gate → the one switch), the golden corpus (56 frames) is byte-identical, and removing any one of `s`/`u`/`a`/`enter`/`g`/`/`/`q`/`t`'s arm or either shadow/gate fails the tests that pin that flow (plants W1–W7/X/Y/Z/BK/TT/SL). One recorded detail is clarified rather than contradicted: the covered saved pane's state table has eight states while its own comment names seven (note 4).

## Goal Achievement

### Observable Truths

Roadmap success criteria (R1–R3) plus every PLAN `must_haves.truths` entry (21 plan truths; 24 rows).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| R1 | Keyword search returns posts with an explicit Latest/Top toggle, and its results page the same way the timeline does — no duplicates, no viewport jumps | ✓ VERIFIED | Posts and accounts are ONE `Results[T,P]` (`internal/screen/search.go:19`) over the ONE `Walk[T,P]` (`internal/screen/walk.go:72`) the timeline, thread and saved frames also use (`list.go:28`, `thread.go:33`, `profile.go:18`, `saved.go:35`); `TestSearch*`, `TestGoldenFrames/search_top_80` PASS; deleting the dedupe check in `Walk.marked` still fails the shared arm for BOTH kinds (`agnostic_test.go:155/158`, plant N this pass) |
| R2 | Users are findable by handle or display name | ✓ VERIFIED | `u` → `SearchPeople` → `Results[domain.User, domain.UserPage]`; account rows are `{handle:16} {name}` via `UserHead`; `TestPeopleSearchRendersOneRowPerAccount`, `TestAccountRowOpensTheProfile` PASS (re-run this pass) |
| R3 | A profile shows bio, counts and join date, and that account's recent posts scroll and page with the same keys as the timeline | ✓ VERIFIED | Re-measured this pass: a cold profile frame's pane rows 1..3 ARE the identity row, the bio and the facts row at 60/80/120/200 × H24/30/40 (12/12; `TestColdProfileFramePaintsItsHeaderAtTheTop`), and every stored profile frame carries them (`profile_60/80/120/200`, `profile_ci/gitlog/internal/servicelog_{80,120}`); the plain facts row prints all four clauses (`followers: 1204  following: 312  posts: 8901  joined: Mar 2011` in `profile_80`), and **the covered rows now print their four in the cover's own spelling** — `watchers=1204  watching=312  records=8901  since=2011-03`, `Watchers: 1204  Watching: 312  Commits: 8901  Since: Mar 2011`, `[1204 watchers]  [312 watching]  [8901 records]  [since 2011-03]`, `watchers 1204  watching 312  requests 8901  since Mar 2011` (stored bytes at 80 and 120, and a render-layer probe over all five grammars asserting the month is in the row and a zero `Joined` drops the clause). The previous pass's "the covered row prints three of the four" is closed: clearing one or all cover stamps' `MonthYear` fails `TestGoldenFrames` on exactly those frames (plants Q, Q2) |
| 01.1 | `/` opens a one-row prompt naming the product before any request, spends nothing, leaves the status line (budget included) exactly as it was | ✓ VERIFIED | `TestSearchOpensAQueryAndRendersResults` (0 calls, budget string unchanged, row contains `search: latest`), `TestSearchPromptSpendsNothing` (re-run this pass) |
| 01.2 | `enter` shows the query's posts as the timeline's own blocks; status names product and query; exactly one more request | ✓ VERIFIED | `TestSearchOpensAQueryAndRendersResults` (+1 request, `search: latest "…"` in the list field); the stored plain `search_80` still ends `search: latest "日本語 🔍 timeline"  1/9+  2 requests, 0 rate-limited` — the amendment moved the covered spellings only |
| 01.3 | `esc` restores the pane byte-identically with zero requests; empty `enter` stays open; every printable key is text | ✓ VERIFIED | `TestSearchEscapeRestoresThePane` (frame string compared, 0 calls), `TestSearchEmptyQueryStaysOpen`, `TestSearchPromptSpendsNothing` (`?/usartjk` typed as text), re-run this pass |
| 01.4 | `s` latest→top is one request and a push; top→pop is byte-identical with zero; elsewhere `nothing to switch` with zero | ✓ VERIFIED | `TestSearchOrderToggleIsAFrameToggle`, `TestSearchOrderSaysNothingToSwitch`, `TestPeopleOrderToggle` (re-run this pass) |
| 01.5 | Empty success is `no results` in the notice; a failure never prints those words | ✓ VERIFIED | `TestSearchEmptyIsNoResultsInTheNotice`, `TestSearchFailureNeverSaysNoResults`, `TestPeopleEmptyIsNoResultsInTheNotice` (re-run this pass) |
| 01.6 | Pagination, dedupe, stall and single-flight on a results list are the same implementation the timeline uses | ✓ VERIFIED | One `type Walk[` and one `type Results[` in `internal/screen`; `TestAgnosticWalkBehaviour` drives ONE `want` table over posts and accounts; the dedupe plant fails that table for both kinds (plant N, this pass) |
| 01.7 | A result carries no rank number, no result count, no attribute on the matched term; product and query named once, in the status | ✓ VERIFIED | `scanForbiddenListAbsences` + `assertNoMatchTermAttribute` wired into `TestCorpusAudit`; an SGR-wrapped match term in the stored `search_80` is still caught (`corpus_test.go:230 search_80 search frame wraps match term "timeline" in an SGR attribute near offset 1952`, plant I this pass) |
| 02.1 | `a` opens the profile in exactly two requests, header then page one addressed by the numeric id | ✓ VERIFIED | `TestProfileOpensInTwoRequests` (2 calls; `LastUserPosts()==header.User.ID`; second `a` = 0 — re-run this pass), `TestProfileOpenIssuesOnlyTheHeader`, `TestProfileHeaderFetchedIssuesUserPostsByNumericID` |
| 02.2 | The header renders as soon as it arrives, with the post list below it in `loading posts`; a failure is the ordinary one-line state; a header that arrived is never taken off screen; a failed header spends no posts request | ✓ VERIFIED | Re-confirmed this pass: the walk is latched when the header answer asks for page one (`screen/profile.go:120-123` `p.walk.Page(req, now)`), so `TestProfileInFlightPostsSaysLoadingPosts` sees `Walk().InFlight()==true`, the notice is `loading posts` and the status reads `loading posts  profile: alice  1 request, …`. Un-latching it reproduces the regression exactly (plant O this pass: `profile_test.go:190 the walk is not latched while its first post page is out`, `:193 notice is 12, want 11`, `:197 status is "no posts  profile: alice  1 request, …", want it to carry "loading posts"`, `:200 it claims an empty read succeeded while a request is out` — the frame claimed an empty read succeeded while a request was out) |
| 02.3 | A profile shows the bio wrapped with every line, then one facts row in the fixed order, absent values omitted never zeroed, identity row alone when nothing survived | ✓ VERIFIED | The rule is intact and enforced at the render layer (`render/profile.go:78-137`; making an absent follower print `0` still fails `TestProfileHeaderFullBareBioFacts` at `profile_test.go:46`, plant M this pass) and it is PAINTED: the cold-frame measurement reads the pane's own bytes against `render.ProfileHeader`'s rows; `profile_empty_80` (no bio, no facts) is the identity row alone, `profile_bio_80` is the three-line bio, `profile_bare_80` the bare account. **The covers' own clauses now render their fourth clause** (`cover.go:154-183` declare them, `cover.go:195-210` give every stamp the `MonthYear` the clause renders through): the render-layer probe asserts all five grammars print the month and that a zero `Joined` drops the clause under each. The previous pass's unreachable-clause note is closed |
| 02.4 | Identity is `{handle:16} {name}` via the shared template; counts only here; no verification, website, avatar or banner grammar | ✓ VERIFIED | `UserHead` shared by `render.UserRow`/`ProfileHeader`; counts printed only by `profileFacts`; `TestProfileHeaderFullBareBioFacts`, grammar sources. Under a cover the identity row is the alias of the same account the status field names (see 03.7): one account, one spelling |
| 02.5 | The account's posts are ordinary blocks paginating with the timeline's keys, trigger, latch and stall rule; position counts posts; `enter` opens the conversation | ✓ VERIFIED | `screen.Profile` embeds `Walk[domain.Post, domain.PostPage]` with no local cursor/dedupe/stall field; position is posts-only (`1/9+` in the goldens); `tui` position tests PASS |
| 02.6 | The header is a block of the list, not chrome: scrolls away, never pinned, repeated or re-drawn, no box/title/banner/second status line, and `g` returns the view to the frame's first row | ✓ VERIFIED | `move` (`update.go:373-427`; the `ActionFirst` reset arm at `:406-413`) leaves the frame unmoved on `keys.ActionFirst` (`f.Moved, f.Anchor = false, 0`), so `skip` (`model.go:699-706`) presents the frame's own first row again. Re-measured this pass: after `G` (and after `j`×4 then `G`), `g` and `home` put the identity row back as pane row 1 at 24/24 size/key pairs (60/80/120/200 × H24/30/40 × both keys, own probe) and in all five grammars at 80/120 (`TestGoRestoresTheProfileHeader`, 10 subtests PASS), and `k` from the first post cannot take it away. Deleting the `ActionFirst` arm fails all 10 subtests (`profile_test.go:334`, plant A). The not-chrome half still holds: the header is `blocks[0]` with `PostID==""`, and once scrolled no pane row carries it (the cold test's second half) |
| 02.7 | `a` on a row with no author or on the already-open profile spends nothing; `r` re-reads header then page one and prepends only unseen ids, announcing `{n} new posts` / `no new posts` | ✓ VERIFIED | `TestOpenAuthorWithoutAnAuthor`, second-`a` no-op in `TestProfileOpensInTwoRequests`, `TestProfileRefreshIsTheTwoRequestPair`; driven: `r` on an open profile spends exactly 2 (header, page one) and the status reads `no new posts` (re-run this pass) |
| 03.1 | `u` opens the same prompt with `people:` before the request; one row per account, handle+name only | ✓ VERIFIED | `TestPeopleSearchRendersOneRowPerAccount` (re-run this pass); a bio line planted under an account row of the stored `search_people_80` still fails the corpus audit (`corpus_test.go:227 search_people_80 people frame row 1 carries a bio/body continuation under an account row`, plant H this pass); `TestCoveredFramesOmitPlainPeopleLiterals` |
| 03.2 | An account row is a row in the same list: same cursor, dedupe, stall, latch, selection, scroll, anchor and back stack | ✓ VERIFIED | `screen/agnostic_test.go` (one script, one `want` table, posts and accounts) and `tui/agnostic_test.go` (one key script over both); the dedupe plant fails both kinds identically (plant N, this pass) |
| 03.3 | `enter`/`a` on an account opens its profile (two ordered requests); `s` on people prints `nothing to switch` and spends nothing | ✓ VERIFIED | `TestAccountRowOpensTheProfile` (exactly 2 past the search — re-run this pass), `TestPeopleOrderToggle` |
| 03.4 | Empty people read is `no results` in the notice; a failure never prints those words | ✓ VERIFIED | `TestPeopleEmptyIsNoResultsInTheNotice` (re-run this pass); the notice vocabulary is per-product through `noticeFor` |
| 03.5 | The stored golden frames are this phase's view specification at every named width/state | ✓ VERIFIED | 56 frames under `internal/tui/testdata/TestGoldenFrames/`; `TestGoldenFrames` compares each against the shipped shell's `Render` byte for byte and PASSes (56/56 this pass). The amendment regenerated the eight covered profile frames at 80/120 with the fourth facts clause (`since=`, `Since:`, `[since …]`, `since `); the covered search/people frames are unchanged |
| 03.6 | The corpus audit asserts the adversarial content, the absences, and that no stored frame holds both a user row and a post block | ✓ VERIFIED | All four restored cases are stored AND asserted over the stored bytes by name (`corpus_test.go:140-178`): blanking one bio line of the stored `profile_bio_80` fails at `corpus_test.go:165` (plant G this pass), and `TestCorpusAuditFailsOnPlantedViolation`'s eight arms (likes/rank in a people frame, bio in a people row, SGR match term, blanked bio line, blanked repost marker, blanked quote row, blanked tombstone, user row in a posts frame) all PASS. Deleting `profile_tombstone_80.golden` fails the shrink arm (`stored corpus has 55 frames, table names 56`, `corpus_test.go:105`, plant J). The homogeneity rule still runs BOTH ways: a user row planted into the stored `search_80` is caught by `scanUserRowsInPostsFrame` (`corpus_test.go:205 search_80 posts frame carries a user row`, plant F this pass), and the control at `corpus_test.go:187-199` keeps that arm from being vacuous |
| 03.7 | Both new surfaces render through Phase 3's grammar/cover seam; a covered frame holds none of the plain literals | ✓ VERIFIED | The four covers carry their own `AccountFacts` AND the stamps to render them (`cover.go:154-183`, `:195-210`), so the eight covered profile frames hold the cover's four clauses and no plain literal (`followers:`/`following:`/`posts:`/`joined:` appear in none of them), and `ListLabelSearch/People/Profile` take a `Grammar`, so the covered status fields read `grep: latest "…"`, `filter=latest "…"`, `filter: latest "…"`, `people: "…"` and `account: <alias>` instead of the plain spellings (stored bytes at 80 and 120 — `account: batch-restore-09` in three styles, `account: nils.okafor` under internal, the alias of the header's own account in each). Reverting either — covers to `Plain.AccountFacts` (plant B), or the labels to their plain path (plant C) — fails `TestGoldenFrames` for all four styles at both widths. **The account the covered status field names is one account one way**: `profileAccount` (`view.go:447-455`) names the header's own account once its answer has arrived and nothing before it under a cover; `TestTheCoveredProfileNamesOneAccount` pins both halves, and reverting it fails at `cover_e2e_test.go:835` (`account: pump-sign-57` before the header, `account: batch-restore-09` after — plant R). **The amendment's tui-corpus scan** (`TestTheCoveredTUIFramesLeakNoPlainVocabulary`, `corpus_test.go:792-905`) scans every covered frame of this corpus — the 24 search/people/profile frames — with each frame's own aliases masked, and its own plant test plants every leak word plus `alice` into a covered people frame; planting a repair word (`doctor`/`mirror`/`saved.db`) or the profile's own identity (`alice`/`Maria`) into a stored covered profile frame fails the scan by the frame's name (plants S1/S2 this pass). The saved surface's three non-row branches pass the grammar through the shell (`view.go:412/417/423`), so no covered saved pane holds the mirror's path, `mirror`, `saved.db`, the host, the product's list names or the store's owner (`TestTheCoveredSavedPaneNamesNothingOfTheProduct`, 80 subtests (16 state subtests — 8 states × 2 widths — and 64 style subtests inside them); reverting any branch fails it — plants D/E this pass; see also note 4). **The unreadable state's own reachability is pinned**: this round's fix made `CapMirrorUnreadable` releaseable (`internal/screen/saved.go:125`, the frame's own release rule the test's comment names) and reachable only through a failing read plus the composition root's mapping (`internal/tui/saved.go`'s `savedFailed` → `screen.MirrorFailed`), and dropping the failing read from the scenario fails the state's plain arm — `cover_e2e_test.go:1454 the plain frame of "the store cannot be read" does not carry "move saved.db aside": the covered absence below would be vacuous` (plant T this pass) — so the covered absence over that state is neither vacuous nor asserted over an unreachable state |

**Score:** 24/24 truths verified

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Approve `TestShowNewFrames` frames | **still DEFER-only** | The checkpoint is `search_80, search_people_80, profile_80, prompt_80`; `profile_80` draws the header, so the deferred frame is worth an operator's eye — but nothing blocks on it (`TestGoldenFrames` is byte-equal to the shipped render) |
| 2 | Live `UserByScreenName`/`UserTweets`/`SearchTimeline` capture | Human Verification 1 | Forbidden in this pass (no network, the session must not be read); the fixtures stand in for it |

### Advisory (New Scope, Unevidenced)

None.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/domain/profile.go` | SearchProduct, Profile, UserPage, port methods | ✓ EXISTS + SUBSTANTIVE | Latest/Top/People; `*int` counts; `UserPage` mirrors `PostPage` |
| `internal/xsource/search.go` / `tweets.go` | Search/SearchPeople over the ONE instruction walk | ✓ EXISTS + SUBSTANTIVE | `walkInstructions`/`foldEntry`/`entrySink` shared by `ParseTimeline` and `ParseSearch`; one cursor reader |
| `internal/xsource/profile.go` | ParseProfile + Adapter.Profile/UserPosts | ✓ EXISTS + SUBSTANTIVE | nil counts when absent, pointer-to-zero when the answer carried 0; one walk for the posts page |
| `internal/xsource/testdata/fixture-*.json` + `provenance.json` | Redacted, provenanced fixtures | ✓ EXISTS | Removing `fixture-search.json` still fails the parse test with the file name (`search_test.go:447`, plant K) |
| `internal/fake/search.go` / `profile.go` + `testdata/pages-search*.json`, `pages-profile.json` | Fixture-backed, fail-closed | ✓ EXISTS + WIRED | Removing `pages-search.json` still fails parity closed with the reason spelled out (`parity_test.go:598`, plant L) |
| `internal/screen/search.go` | One `Results[T,P]` for both entity kinds | ✓ EXISTS + SUBSTANTIVE | `NewPostResults`/`NewPeopleResults` over the one `Walk` |
| `internal/screen/profile.go` | Header state + `Walk[Post,PostPage]` | ✓ EXISTS + SUBSTANTIVE | `Open` → header only; `HeaderFetched` → `UserPostsRequest{ID}` through `Walk.Page`, so the latch is set (02.2) |
| `internal/screen/agnostic_test.go`, `internal/tui/agnostic_test.go` | The same assertions over both entity kinds | ✓ EXISTS + SUBSTANTIVE | One shared expected table / one key script |
| `internal/render/prompt.go` / `profile.go` / `user.go` | Prompt row, header block, account row | ✓ EXISTS | identity via `UserHead`; facts clauses omit absent values; the covers' clauses and the grammar-aware list labels are the fix's, and the fourth clause renders now that every cover stamp sets `MonthYear` |
| `internal/tui` search/people/profile + goldens + corpus | Shell wiring, 56 stored frames, the audit | ✓ EXISTS + SUBSTANTIVE | The frames match the shipped renderer byte for byte and carry the header and the covered wording; the four content cases, the two-way homogeneity arm, the `g` restoration and the amendment's plain-vocabulary scan over every covered frame are asserted |
| `internal/render/user_test.go` | Account-row render tests (plan 05-03 declared it) | ✗ MISSING | The plan lists the file (`05-03-PLAN.md:27`); it does not exist, and no test asserts the account row at the render layer (the shape is pinned at the tui layer and by the corpus bio arm). Carried from the previous pass; non-blocking |

**Artifacts:** 10/11 verified.

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| tui search frame | `Source.Search` | `Results` → `Walk` → `SearchRequest` → `Adapter.Search` → `SearchTimeline` by name | ✓ WIRED | `update.go` request union; no `queryId` literal outside the registry |
| tui people frame | `Source.SearchPeople` | the same reducer, `PeopleRequest` | ✓ WIRED | one dispatch, one walk |
| tui profile | `Profile` then `UserPosts` | `ProfileRequest` → `HeaderFetched` → `UserPostsRequest{ID}` | ✓ WIRED | the posts request exists only as the header answer's effect; numeric id only |
| profile frame pane | `render.Profile`'s block list | `blocksIn` → `m.skip` with the frame's opening state and anchor | ✓ WIRED | The block list is right (header first), the pane starts at the frame's first row while the frame is unmoved (`model.go:700-708`, `view.go:246`), and `ActionFirst` returns the frame to that state (`update.go:406-413`), so the header is one key away again |
| query echo | prompt row + list field | `render.PromptRow`, `ListLabelSearch/People` | ✓ WIRED | one composition site each, and the covered spelling is read off the same grammar the prompt row uses (`copy.go:829-853`) |
| account row + profile identity | `UserHead` | one template | ✓ WIRED | `user.go`/`profile.go`/`grammar.go` |
| keys `/` `s` `u` `a` `g` | `keys.Rows()` | help + `keys` subcommand + `press` | ✓ WIRED | one keymap table; all of this phase's rows appear in the stored `help_80`/`help_120` frames; `press` routes each through its own single arm and `handledActions` enumerates the set, held to the table from both directions by `dispatch_test.go` (`TestEveryBoundActionIsHandledAndEveryHandledActionIsBound`, `TestTheShellDeclaresOnlyActionsTheKeymapDeclares`); removing `/`, `s`, `u`, `a` or `g`'s arm fails that key's own tests (plants SL, W1–W4, W7) |
| goldens | shipped shell | `Update`/`Render` + fake + pinned clock/env | ✓ WIRED | `TestGoldenFrames` byte equality (56/56) |
| covered saved pane | cover grammar | `savedPane` → `CapabilityBlock`/`StateBlockWithMirror`/`SavedEmptyState` with `m.grammar()` | ✓ WIRED | the three branches pass the pane's grammar (`view.go:412/417/423`); reverting any one to `render.Plain` fails `TestTheCoveredSavedPaneNamesNothingOfTheProduct` (plants D/E) |

**Wiring:** 9/9 verified.

## Requirements Coverage

| Requirement | Status | Blocking Issue |
|-------------|--------|----------------|
| DISC-01 keyword search + Latest/Top | ✓ SATISFIED | - |
| DISC-02 find users by handle/display name | ✓ SATISFIED | - |
| DISC-03 profile bio/counts/join + recent posts | ✓ SATISFIED | the surface is readable on open, the header is reachable again after a scroll, the posts page on the one walk, and the join date now renders under every cover |
| REND-01 no engagement/avatars in lists | ✓ SATISFIED | absence scanner + bio-in-people-row arm, both non-vacuous |
| REND-05 one keymap | ✓ SATISFIED | `keys.Rows()`; this phase's rows are in the stored help frames, and the keymap's key side is held to the dispatcher's from both directions by this round's `dispatch_test.go` (plants U/V) |
| REND-06 back stack | ✓ SATISFIED | agnostic/pop tests; profile push/pop costs 2 then 0 |
| REND-08 goldens | ✓ SATISFIED | 56 stored frames, byte-equal to the shipped render |
| SEC-01 session cookie | ✓ inherited | the suite does not print the session; every run here pointed `XDG_CONFIG_HOME`/`XDG_DATA_HOME` at a temp directory and never opened the operator's real files |

**Coverage:** 8/8 requirements satisfied.

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | none this pass | — | The previous pass's one warning (the covered facts row declared a fourth `{joined}` clause no cover stamp could render) is closed by the amendment — every cover stamp sets `MonthYear` (`cover.go:195-210`) and the clause renders in all four styles (R3/02.3 evidence, plants Q/Q2) |

**Anti-patterns:** 0 blockers, 0 warnings.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `tui/search_test.go` | DISC-01 | yes | no | no | behavioural budgets + frame bytes | OK |
| `tui/people_test.go` | DISC-02 | yes | no | no | behavioural budgets | OK |
| `tui/profile_test.go` | DISC-03 | yes | no | no | call counts, header state, the pane's own bytes at 12 sizes, the in-flight notice, `g`'s restoration in 5 grammars × 2 widths | OK |
| `tui/cover_e2e_test.go` | DSGT-08/DSGT-09 | yes | no | no | the covered pane's own bytes vs. a plain arm, 8 saved states × 2 widths × 4 styles (the table's own doc comment says seven — note 4); the covered profile's one-account rule over the shipped path with the header request held; the frame-kind join | OK (non-vacuous: each state's plain arm asserts the leak string is there; the identity test's plain arm asserts the field renders pre-answer) |
| `screen/agnostic_test.go` | entity-agnostic | yes | no | no | value tables + non-vacuity arm | OK |
| `tui/agnostic_test.go` | entity-agnostic | yes | no | no | key script over both kinds | OK |
| `tui/corpus_test.go` | REND-08/absences | yes | no | no | stored bytes + fail-on-plant, 8 arms + the shrink and homogeneity arms + the plain-vocabulary scan over every covered frame (24) with its own through-a-file plant test | OK |
| `tui/dispatch_test.go` | REND-05 one keymap (key side) | yes | no | no | the keymap⇄`press` join both ways + a source sweep over `keys.go`'s action declarations and the shell's case labels; both of its own plant directions fail (`dispatch_test.go:63/72/79/125/128`) | OK (non-vacuous: the sweep reads the declarations out of the source, so an action that lands in only one of the three places fails) |
| `render/profile_test.go` | DISC-03 render rule | yes | no | no | row counts + clause strings | OK |
| `render/grammar_phase5_test.go` | cover seam | yes | no | no | marker totality + covered-literal absence | OK (two literals only — the covered facts-row labels are pinned by golden equality; see the non-critical notes) |
| `render/corpus_test.go` | covered-frame rule (render half) | yes | no | no | the scan over every covered frame of the render corpus (≥60, the saved shapes state by state) with its own plant test into a covered thread frame and a covered saved-row frame | OK |
| `xsource` Capture*/parse fixtures | fixtures | skip without gate | expected | n/a | existence + synthetic body | OK |
| `fake` search/profile parity | fail-closed | yes | no | no | document parity | OK |

**Disabled tests on requirements:** 0 blockers (captures are gated operator tests).
**Circular patterns detected:** 0.

## Falsification Experiments (scratch copy `/tmp/p5v`, edit-and-revert)

Every plant ran in a copy of the tree (`/tmp/p5v`, taken after the guard round); each was reverted after its run, and the repository itself was never edited — the only write anywhere is this report. The final `diff -r` against the repository names no file except the `*-VERIFICATION.md` reports themselves (each parallel pass's own, in the repository) and the copy's two probe files and `.git/index`, which the copy's own stray `git` metadata accounts for; no plant left a `*.p5bak` behind and `git status` in the repository shows only the reports. Plants marked **FAIL** are tests that correctly refused the planted violation. All 22 plants the previous pass re-ran, plus the one it added, are re-run this pass, verbatim from its own `plants2.sh`: A–P are the sixteen that pass before it recorded, Q, Q2, R, S1, S2 and 3 are the amendment round's own checks of the covered profile, and T is the previous pass's own check of the fix round's scenario. Fifteen more are this pass's own, for the guard the round added: U and V are the author's two (a 31st keymap row with a fresh action; a handler for an action no row binds); W1–W7 remove one dispatcher arm each for the flows this phase depends on (`s`, `u`, `a`, `enter`, the prompt shadow, `j`, `g`); X, Y and Z remove the help shadow, the panic gate and the cover gate; BK, TT and SL remove the back, toggle and `/` arms.

| Experiment | Result |
|------------|--------|
| **A — delete the `ActionFirst` arm from `move` (`update.go`)** | **FAIL** `TestGoRestoresTheProfileHeader` at 10/10 subtests (`plain/80 … internal/120`: `profile_test.go:334 \`g\` left the frame moved`); restored |
| **B — set every cover's `AccountFacts` back to `Plain.AccountFacts`** | **FAIL** `TestGoldenFrames` on `profile_{servicelog,gitlog,ci,internal}_{80,120}` (`golden_test.go:345 output does not match`); restored |
| **C — drop the grammar branch from `ListLabelProfile`/`ListLabelSearch`** | **FAIL** `TestGoldenFrames` on the covered profile set and on the covered search set (all four styles at 80/120); restored |
| **D — pass `render.Plain` to `CapabilityBlock` in `savedPane`** | **FAIL** `TestTheCoveredSavedPaneNamesNothingOfTheProduct` (`cover_e2e_test.go:1467` the covered frame holds `doctor`/`bookmarks` in all four styles; `:1476` missing the cover's own capability words); restored |
| **E — pass `render.Plain` to `StateBlockWithMirror` and `SavedEmptyState`** | **FAIL** the same test (`:1467` holds `mirror`/`bookmarks`; `:1476` does not carry `no records in this view` / `nothing is cached for this view yet`); restored |
| **Q — clear `MonthYear` on all four cover stamps (the amendment's clause, absent)** | **FAIL** `TestGoldenFrames` on exactly the eight covered profile frames (4 styles × 80/120; `golden_test.go:345`); the plain frames and the whole covered search/people set still match, and the render-layer corpus does not notice (it holds no profile frame); restored |
| **Q2 — clear `MonthYear` on `coverGitStamp` alone** | **FAIL** `TestGoldenFrames` on `profile_gitlog_{80,120}` only; the other three styles' frames still match — the clause is per-style, in the style's own `MonthYear`; restored |
| **R — revert `profileAccount` to name the opened handle under a cover** | **FAIL** `TestTheCoveredProfileNamesOneAccount` (`cover_e2e_test.go:835 the covered profile frame names an account before its header has answered: "account: pump-sign-57 …"` in three styles, `"account: viktor.rahman …"` under internal — and the header's answer would spell the same account `batch-restore-09`: two names for one account); restored |
| **S1 — plant `doctor mirror saved.db` into the stored `profile_gitlog_80`** | **FAIL** `TestTheCoveredTUIFramesLeakNoPlainVocabulary` (`corpus_test.go:831 the covered tui frame profile_gitlog_80 holds "mirror" … "saved.db" … "doctor"`); restored |
| **S2 — plant the profile's own identity row (`alice            Maria`) into the stored `profile_gitlog_80`** | **FAIL** the same test (`corpus_test.go:831 the covered tui frame profile_gitlog_80 holds "alice" … "Alice"`); restored |
| **3 — remove the profile row from the covered check table (`frameKindChecks`)** | **FAIL** `TestEveryFrameKindIsCheckedUnderACover` (`cover_e2e_test.go:739 the shell declares the frame kind 3 and the covered check table has 0 rows for it`, and `:739 the shell declares the frame kind 0 and the covered check table has 2 rows for it` — the row was typed as `frameHome`); restored |
| **F — plant a user row into the stored `search_80.golden`** | **FAIL** `TestCorpusAudit` (`corpus_test.go:205 search_80 posts frame carries a user row: ">alice            Alice"`) — the reverse-homogeneity arm; restored |
| **G — blank one bio line of the stored `profile_bio_80.golden`** | **FAIL** `corpus_test.go:165 profile_bio_80: the header's bio rows reassemble into "reads timelines in a terminal, mostly the slow ones, and writes down what the  usually is", want "… what the cursor did when it came back around …"`; restored |
| **H — insert a bio continuation under an account row of the stored `search_people_80.golden`** | **FAIL** `corpus_test.go:227 search_people_80 people frame row 1 carries a bio/body continuation under an account row: "   reads timelines in a terminal, mostly the slow ones, and writes down what the"`; restored |
| **I — wrap the match term `timeline` in SGR in the stored `search_80.golden`** | **FAIL** `corpus_test.go:230 search_80 search frame wraps match term "timeline" in an SGR attribute near offset 1952`; restored |
| **J — delete `profile_tombstone_80.golden`** | **FAIL** `corpus_test.go:105 stored corpus has 55 frames, table names 56`, and `TestCorpusAuditFailsOnPlantedViolation` fails reading the frame back (`corpus_test.go:384 reading profile_tombstone_80: open testdata/TestGoldenFrames/profile_tombstone_80.golden: no such file or directory`); restored |
| **K — remove `internal/xsource/testdata/fixture-search.json`** | **FAIL** `search_test.go:447 reading fixture-search.json: open testdata/fixture-search.json: no such file or directory`; restored |
| **L — remove `internal/fake/testdata/pages-search.json`** | **FAIL** `parity_test.go:598 the fake's recorded search page set is empty: … testdata/pages-search.json is not there … (fail-closed)`; restored |
| **M — make an absent follower count print `0` in `render/profile.go`** | **FAIL** `profile_test.go:46 bare header has 2 rows, want 1`; restored |
| **N — delete the dedupe check in `Walk.marked`** | **FAIL** `screen/agnostic_test.go:155 posts step 1: got {Rows:5 …} want {Rows:4 …}` and `:158 users step 1: …` — both kinds; restored |
| **O — un-latch the profile's first post page (`HeaderFetched` returns the effect without `Walk.Page`)** | **FAIL** `profile_test.go:190 the walk is not latched while its first post page is out`, `:193 notice is 12, want 11`, `:197 status is "no posts  profile: alice  1 request, …", want it to carry "loading posts"`, `:200 it claims an empty read succeeded while a request is out`; restored |
| **P — restore the old pane rule (`skip = above - Anchor`, no opening state)** | **FAIL** `TestColdProfileFramePaintsItsHeaderAtTheTop` at every size (`profile_test.go:54 pane row 1 is ">a_15_cell_handl  Oct 14 09:12", want the header's own row " alice            Maria"`); restored |
| **T — drop the failing read from the "store cannot be read" scenario (`cover_e2e_test.go`, this round's own change)** | **FAIL** `TestTheCoveredSavedPaneNamesNothingOfTheProduct/the_store_cannot_be_read/{80,120}` at `cover_e2e_test.go:1454 the plain frame of "the store cannot be read" does not carry "move saved.db aside": the covered absence below would be vacuous` — with a mirror that answers, the frame's own release rule clears the capability, so the pair the fix retired is unreachable; restored |
| **U — add a 31st keymap row with a fresh action `test-plant` (`keys.go`) [this round's guard, the author's own plant]** | **FAIL** `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` (`dispatch_test.go:63 the keymap binds "test-plant" (ctrl+t) and press has no arm for it: the key is a documented no-op`, `:79 the table holds 31 rows and handledActions names 30 actions`) and `TestTheShellDeclaresOnlyActionsTheKeymapDeclares` (`:128 keys.go declares ActionTestPlant and handledActions does not name it: the enumeration went stale`); restored |
| **V — add a handler for an action no row binds (`keys.go` + `update.go`) [this round's guard, the author's own plant]** | **FAIL** the same two tests (`dispatch_test.go:72 press handles "test-orphan" and no row of the keymap binds it: the arm is unreachable`, `:79 the table holds 30 rows and handledActions names 31 actions`, `:125 keys.go declares ActionTestOrphan ("test-orphan") and no row binds it`); restored |
| **W1 — remove `case keys.ActionSearchOrder: return m.switchOrder()` (`s`)** | **FAIL** (`search_test.go:167 \`s\` from latest issued 0 requests past 2, want exactly one`; `:203` the status line reads `following …`, without `nothing to switch`); restored |
| **W2 — remove `case keys.ActionSearchPeople: return m.openPrompt(domain.People)` (`u`)** | **FAIL** (`people_test.go:49 \`u\` did not open the prompt`; `:174` the empty people read's status line is a thread's, without `no results`); restored |
| **W3 — remove `case keys.ActionOpenAuthor: return m.openAuthor()` (`a`)** | **FAIL** (`profile_test.go:109 \`a\` issued 0 requests past 1, want exactly 2`; `:272 notice=0, want NoticeNoAuthor`); restored |
| **W4 — remove `case keys.ActionOpen: return m.open()` (`enter`)** | **FAIL** (`people_test.go:130 \`enter\` on an account row left kind 2, want profile`; `tui_test.go:1065 enter asked for nothing`); restored |
| **W5 — remove the prompt's shadow block (`if m.prompt != nil { return m.promptKey(msg) }`)** | **FAIL** across `TestSearch`: `search_test.go:47 typing into the prompt issued 2 requests: every printable key is text`, `:107 the prompt's text is "", want the keys typed in as text`, `:124 \`esc\` left the prompt open`; restored |
| **W6 — carry the wrong action in `j`'s arm (`move(action)` → `move(keys.ActionPrev)`)** | **FAIL** `TestGoldenFrames` (`golden_test.go:345 output does not match`, `TestGoldenFrames/profile_scrolled_80`) — the 56-frame byte equality is what catches a mis-routed movement; restored |
| **W7 — carry the wrong action in `g`'s arm (`move(action)` → `move(keys.ActionLast)`)** | **FAIL** `TestGoRestoresTheProfileHeader` at 10/10 (`profile_test.go:334 \`g\` left the frame moved`); restored |
| **X — remove the help overlay's shadow block** | **FAIL** `TestTheHelpOverlayIsTheOneTable` (`tui_test.go:700 a movement key under the overlay moved the selection to "adv-1"`, `:708 back under the overlay returned a command, so it quit rather than closing`); restored |
| **Y — remove the panic gate** | **FAIL** `TestPanicSwallowsKeys` (`state_test.go:311 the key "G"/"enter"/"a" asked for a command while panicked`); restored |
| **Z — remove the cover gate (`coverBreaking`)** | **FAIL** `TestCoverBreakingActionsGated` (`tui_test.go:1952 renderer called under cover: open=4 url=0`, `:1955 clipboard called under cover: 1`); restored |
| **BK — remove `case keys.ActionBack: return m.back()` (`q`/`esc`)** | **FAIL** `TestQRestoresTheFrameWithZeroThreadRequests` (`tui_test.go:1168 q did not restore the frame byte-identically`); restored |
| **TT — remove `case keys.ActionSwitchList: return m.toggleFeed()` (`t`)** | **FAIL** `TestTheToggleSwitchesFeedsAndReturns` (`cli/feed_test.go:242 the toggled frame read following, want ForYou`); restored |
| **SL — remove `case keys.ActionSearchPosts: return m.openPrompt(domain.Latest)` (`/`)** | **FAIL** (`search_test.go:29 \`/\` did not open the prompt`; `:127 \`esc\` issued 2 requests`); restored |
| Compiler plant: add `case keys.ActionThisPackageDeclaresNoSuchAction:` to `press` | **FAIL the build** — `internal/tui/update.go:236:13 undefined: keys.ActionThisPackageDeclaresNoSuchAction` (and `:268:12`) — the property the dispatch sweep's source read leans on: a bogus case label cannot exist; reverted |
| Own probe: the covered facts row at the render layer, all five grammars, joined = 2011-03-01 | **PASS**: every grammar prints four clauses with the month in its own spelling (`followers: 1204  following: 312  posts: 8901  joined: Mar 2011` / `since=2011-03` / `Since: Mar 2011` / `[since 2011-03]` / `since Mar 2011`), and a zero `Joined` drops the clause under each — the previous pass's three-clause covered row is gone. Re-run this pass |
| Own probe: `g`/`home` from a scrolled profile at 60/80/120/200 × H24/30/40 (re-run this pass) | **PASS** 24/24: pane row 1 is the identity row after both keys, and after `k` at the first post |
| Instrument the "header arrived, posts in flight" state | `TestProfileInFlightPostsSaysLoadingPosts` PASSES: `outstanding()==1`, `Walk().InFlight()==true`, notice `loading posts`, status `loading posts  profile: alice  1 request, …`; after the empty answer the notice is `no posts` |

## Automated Checks Run

| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `gofmt -l .` | PASS (no files) |
| `go test ./... -count=1` | PASS — 19 test packages of 22 ok (3 have no test files), 0 FAIL, 8 SKIP; verbose: 1581 `=== RUN`, 1573 `--- PASS`, 0 FAIL (the two more than the previous pass are `dispatch_test.go`'s own arms, counted the same way: every `--- PASS` at any nesting) |
| `go test ./internal/{tui,screen,render,fake,xsource,keys,domain} -count=1` | PASS — 1033 `--- PASS`, 8 SKIP, 0 FAIL |
| Same full suite under `unshare -rn` with `lo` up (`ip link set lo up`) | PASS — exit 0, 19 packages ok, no FAIL (re-run this pass) |
| Same phase suites under `unshare -rn` with `lo` up | PASS — exit 0, all 7 packages ok, 0 FAIL (re-run this pass) |
| `TestGoldenFrames` (56 frames, byte equality) | PASS |
| `TestGoRestoresTheProfileHeader` (5 grammars × 80/120) | PASS — 10 subtests |
| `TestColdProfileFramePaintsItsHeaderAtTheTop` (4 widths × 3 heights) | PASS — 12 subtests |
| `TestTheCoveredSavedPaneNamesNothingOfTheProduct` | PASS — 80 `--- PASS` lines (16 state subtests: 8 states × 2 widths; 64 style subtests inside them, one per style, plus each state's own plain arm asserted inside its parent); the table's own comment names seven states (note 4) |
| `TestTheCoveredProfileNamesOneAccount` / `TestEveryFrameKindIsCheckedUnderACover` / `TestTheWholeSurfaceUnderACover` / `TestTheCoveredFailurePaneNamesNothingOfTheUpstream` / `TestCoveredFramesOmitPlainPeopleLiterals` | PASS |
| `TestEveryBoundActionIsHandledAndEveryHandledActionIsBound` / `TestTheShellDeclaresOnlyActionsTheKeymapDeclares` (this round's guard) | PASS — the keymap⇄dispatcher join both ways plus the source sweep over the action declarations; both of the author's plants (a 31st row; a handler with no row) are refused (plants U, V), and a bogus case label is refused by the compiler |
| `TestTheCoveredTUIFramesLeakNoPlainVocabulary` / `TestTheCoveredTUIFrameLeakScanCatchesAPlant` (tui half) and `TestCoveredFramesLeakNoRepairWords` / `TestCoveredLeakScanCatchesAPlant` (render half) | PASS |
| `TestCorpusAudit` (incl. `TestCorpusAuditFailsOnPlantedViolation`, 8 arms) | PASS |
| `x-terminal audit` | 10 checks, 0 findings, 893 files inspected, exit 0 (run with `XDG_*` pointed at a temp directory; the new `opnames` check requires every name in the audit's classification table — `SearchTimeline`, `UserByScreenName`, `UserTweets` among them — to be one the volatile registry resolves, which is this phase's own `Source` union) |
| `TestASavedFramesCapabilityGoesWhenItsOwnActionSucceeds` (the fix round's arm for the stickiness rule, in the covered `tui_test.go`) | PASS — phase 6's truth, not this phase's; re-run as part of the phase suites |
| Decision coverage gate | skipped — `no trackable decisions` in CONTEXT.md |

## Human Verification Required

### 1. Live capture of the profile pair and the search reads (unchanged)
**Test:** `X_TERMINAL_CAPTURE=1` for `UserByScreenName` + `UserTweets` and for `SearchTimeline`, with a real session.
**Expected:** the matched pairs the fixtures stand in for.
**Why human:** forbidden in this pass (no network, the session must not be read).

(The previous pass's second human item — reading a covered profile frame to judge whether the covers carry the join date — is withdrawn: the tree now settles it. Every covered profile frame carries the fourth facts clause in its style's own spelling, the clause is what 05-UI-SPEC:632 asks each cover to render, and clearing a style's `MonthYear` fails `TestGoldenFrames` on exactly that style's frames.)

## Gaps Summary

**No critical gaps.** Every must-have truth is verified and no blocker remains. This pass found no broken must-have: the round's only change is the dispatcher's one-arm-per-action restructure, which is behaviour-preserving for every flow this phase's truths depend on (pinned by plants W1–W7, X, Y, Z, BK, TT, SL and the unchanged 56-frame golden byte equality), and every re-measurement agrees with the record. No new note is raised this pass; the notes below are carried.

### Non-Critical Gaps (Can Defer)

1. **CLOSED by the amendment — the covers' `{joined}` clause was unreachable**
   - Was: each cover declared a fourth facts clause (`since={joined}`, `Since: {joined}`, `[since {joined}]`, `since {joined}`) but `expandFact` expanded `{joined}` only when `g.Stamp.MonthYear` was set and no cover stamp set it, so every covered profile frame printed three clauses for an account whose answer carried four.
   - Now: every cover stamp sets `MonthYear` (`cover.go:195-210`) and the clause renders in the style's own spelling in all eight covered profile goldens at 80/120 and in the render-layer probe; clearing one or all four fails `TestGoldenFrames` on exactly those frames (plants Q/Q2). The pre-amendment plain-clause fallback would have dropped the clause too, so this was never a regression of the previous fix round — the record stands as written.
2. **The account-row render test plan 05-03 declared (`internal/render/user_test.go`)**
   - Issue: the file does not exist (`05-03-PLAN.md:27` lists it) and no render-layer test asserts the account row's bytes.
   - Impact: the shape is pinned at the tui layer and by the corpus bio arm, so the risk is coverage rather than correctness.
   - Recommendation: add the file or strike it from the plan (carried from the previous pass; unchanged this pass).
3. **The covered facts row is pinned only by golden equality (P3, recorded note)**
   - Issue: no non-golden arm asserts a cover's facts row against the values the profile carries (the previous pass's fix-plan task 3 was not taken, and this pass only re-ran the same check as a scratch probe — the arm is still not in the tree). Plant Q2 shows golden equality catches a drift of the shipped render from the stored bytes, but a re-blessed drift would pass, and the plain-vocabulary scan does not list the facts-row labels (`followers:`/`joined:` are not in `tuiLeakWords`/`coveredLeakWords`).
   - Impact: a cover's facts clauses could silently become the plain reader's if someone regenerated the goldens under a broken tree; nothing today refuses that except review.
   - Recommendation: accepted as recorded tech debt for the archive — a render-layer arm (one table, five grammars, the four clauses against the account's own values) is one small test.
4. **The covered saved-frame state table's own comment says seven states (P3, new this pass)**
   - Issue: `savedFrameStates`' doc comment (`cover_e2e_test.go:1302-1303`) enumerates "the three capability states, the two empty reads, the failure with an empty store, and the first page in flight" — seven — while the table lists eight: the likes list with an empty store is the eighth (added with the likes hedge, before this round). The previous pass's test-quality row repeated the seven.
   - Impact: none on behaviour (all eight states are asserted, 80 subtests); a reader counting the comment would under-read the covered surface.
   - Recommendation: say eight in the comment, or name the likes state; a P3 doc nit, no plan needed.
   - Counting convention, for the record: the previous pass's "17 subtests" is the 16 state/width pairs plus the top-level test; with the per-style arms the test's `--- PASS` lines number 80.
5. **The covered pre-answer profile frame has no stored golden (P3, recorded note)**
   - Issue: the state `profileAccount` exists for — a covered profile before its header has answered names nothing — is pinned behaviorally by `TestTheCoveredProfileNamesOneAccount` but is not a stored frame. The corpus scan's alias-presence arm requires every covered frame to carry an alias of its style, which the pre-answer frame has none of, so a stored pre-answer covered frame would trip the scan as written.
   - Impact: none while the behavioral test lives; a reader of the corpus alone would not see the state.
   - Recommendation: accepted as recorded tech debt; making the scan's alias arm state-aware is the precondition for a stored frame.

## Recommended Fix Plans

The previous pass's `05-04-PLAN.md` sketch is now obsolete: its tasks 1–2 (set `MonthYear` on each cover stamp; regenerate the eight covered profile frames and re-run the golden/corpus/covered-pane tests) are exactly what the amendment round landed, and their refusal tests are recorded above (plants Q/Q2). Its task 3 — the render-layer facts-row arm — was not taken and is restated as non-critical note 3; it is a one-table test and no longer needs a plan.

## Could Not Verify

- **That this round's fix holds on its own ground** (`internal/store`'s `FileError` mark, `internal/cli/feed.go`'s `errors.As(err, &store.FileError{})` predicate, the release rule in `internal/screen/saved.go`): phase 6's report owns those truths. What this pass verified is the part that touches this phase — that the covered saved pane's `mirror unreadable` state is reachable through the shell at all (plant T) and that its covered absence is not vacuous (plants D/E). `cli/feed.go:129` and `store/read.go:23-41` were read for effect and none of this phase's truths depends on which errors wear the mark: this phase's evidence drives the capability through the shell's own `SavedError` seam.
- Live `UserByScreenName` / `UserTweets` / `SearchTimeline` capture against a real session (forbidden in this pass). The synthetic fixtures satisfy the offline contract, and the gated capture stubs refuse a live body rather than inventing one.
- **That the amendment's own claims about *history* are true** — "every cover stamp now sets `MonthYear`", "the covered profile names one account", "the goldens were regenerated". This pass is read-only and ran no `git` command, so the *before* state cannot be diffed; what is verified here is the end state (the stamps, `profileAccount`, the stored bytes at 80/120) plus the failure each plant produces when the claim is taken away.
- Interactive visual judgment of the stored frames (offline bytes plus the deterministic golden comparison stand in).
- **That the restructure is a *change* at all, and that it is *only* a restructure** — the assignment forbids `git diff`/`git show`/`jj diff`, so the *before* of `press`, `handledActions` and `dispatch_test.go` cannot be diffed (the only git invocation this pass made was `git status --porcelain`, to confirm no source file changed). What is verified is the end state and its behaviour: the arms dispatch the same actions to the same methods, the two-directional join holds and both of its plant directions fail, a bogus case label is refused by the compiler, and the 56-frame golden corpus plus every key-flow test are unchanged. A human with git can confirm no key or action was added by diffing `keys.Rows()`/`press` against the previous commit.
- That a failed **header** at the TUI Source-counter layer costs exactly `before+1` with no `LastUserPosts` (the reducer proves it — `TestProfileHeaderFailedSpendsNoPostsRequest` — and no tui-level FailNext+`a` test exists).
- The help overlay at every height: the stored help frames are 80/120-wide at the corpus height, and the phase's rows are all present there.

## Decision Coverage

No trackable decisions (`gsd-tools` decision-coverage-verify skipped: no trackable decisions in CONTEXT.md).

## Verification Metadata

**Verification approach:** Goal-backward re-adjudication of all 24 must-haves against the frozen tree after the round that restructured `press` into one arm per action (and added `handledActions`/`HandledActions()` and `dispatch_test.go`). All 22 falsifications the previous pass recorded (A–P, Q, Q2, R, S1, S2, 3) were re-run **verbatim from that pass's own `plants2.sh`** in a fresh `/tmp/p5v` copy — every one still refused at the site named above (D/E and R's line numbers unchanged; plant O's run moved from the `screen` package to `tui`, where `TestProfileInFlightPostsSaysLoadingPosts` lives) — plus plant T (the fix round's own claim). Fifteen plants of my own pin the restructure's behaviour-preservation: U and V (the guard's two directions, both refused), W1–W7 (one dispatcher arm removed for `s`, `u`, `a`, `enter`, the prompt shadow and both movement arms), X/Y/Z (the help shadow, the panic gate and the cover gate), BK/TT/SL (back, toggle, `/`), and a bogus case label (refused by the compiler). Three probes of my own, re-run: the covered facts row at the render layer over all five grammars, `g`/`home` at 12 sizes × 2 keys (24/24), and the in-flight instrument (control run). The restructure's own flows were re-adjudicated one by one against their tests; none changes what any of this phase's truths assert, and the 56-frame golden corpus is byte-identical.
**Must-haves source:** 05-01/02/03-PLAN.md frontmatter (21 truths) + ROADMAP Phase 5 success criteria (3). The score is corrected to 24/24: the previous pass's `21/21` counted only the numbered plan rows while its own table carried all 24.
**Covered-file set:** 121 paths — the previous pass's 120, unchanged and none removed, plus `internal/tui/dispatch_test.go` (this round's guard over the dispatcher that routes every one of this phase's keys; its keymap-side file `internal/keys/keys.go` and the dispatcher `internal/tui/update.go` were already covered) — and that pass's own two additions were: `internal/screen/saved.go` (the frame's own release rule, which the covered test's comment names as what makes the unreadable state's failing read the reachable pair) and `internal/tui/saved.go` (the shell plumbing — `pushSaved`/`savedPageAnswered`/`savedFailed` — that turns that read into the capability state the covered pane test reaches). The fix's other two files (`internal/store`, `internal/cli/feed.go`) are not on this phase's evidence path, so they were not added (see "Could Not Verify").
**Automated checks:** build/vet/gofmt/full suite (22 packages, 19 with tests, 1573 passing, 0 FAIL, 8 SKIP) and the phase suites (1033 passing, 8 SKIP), each also under `unshare -rn` with `lo` up: PASS. 39 plants (23 re-run from the previous pass + 15 of mine + the compiler plant) and 3 probes recorded above. `x-terminal audit`: 10 checks, 0 findings, 893 files, exit 0.
**Human checks required:** 1 (the live capture).
**Tree mutation:** only this report was written. Every plant ran in a fresh `/tmp/p5v` copy (taken after the guard round); the copy's two probe files were removed after their runs, `git status` in the repository shows only the phase reports, and no test run left a file in the repository.

---

*Verified: 2026-10-15T07:24:07Z*
*Verifier: Claude (subagent)*
