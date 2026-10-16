# Roadmap: x-terminal

## Overview

A read-only x.com client for a terminal pane, delivered as eight vertical slices. Every phase ends with something the user can actually do, and each retires a named risk. One standing rule binds all of them: no phase may add a write path to the account, a timer, a second persistence file, a second account, or a distribution artifact.

Phase 1 proves the one thing that can invalidate the project — that a real authenticated request reaches x.com from this host, through the user's proxy — and captures the real payload shapes. Phase 2 turns that into the product: the Following timeline in a pane, carrying the pagination, request discipline, width model and navigation shell that every later surface reuses. Phase 3 adds the three-state disguise (normal / immersive / panic) on top of that working renderer; it sits third rather than last so the cover is a renderer mode that later surfaces inherit, instead of something retrofitted into finished views. Phases 4–7 are the rest of the read surface — threads, search and profiles, bookmarks and likes with the mirror-only store, and media / links / clipboard — and are mutually independent: different packages, sharing only the `Source` port, the list view, the navigation shell and the cover layer, so they can run in parallel. Phase 8 is the repair path: `doctor`, the runbook, and the audits that prove read-only, no-timeline-on-disk and no-distribution still hold.

The research recommendation was seven phases; the disguise requirements (DSGT-*) arrived after that research was written and had no home in it. The structure here is those seven plus one dedicated disguise phase, sequenced immediately after the first working renderer — Phase 3's notes carry that argument and the measured DSGT-02 evidence behind it.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Reachable Transport and One Live Read** - Proxy wiring, named connectivity states, session-cookie handling, the volatile containment seam, and one real authenticated Following-timeline read (completed 2026-10-14)
- [x] **Phase 2: Following Timeline in the Pane** - The default list with keyset pagination, manual refresh, named failure states, and the rendering / navigation shell every later surface reuses (completed 2026-10-14)
- [x] **Phase 3: Three-State Disguise on the Working Renderer** - Normal / immersive / panic states with aliased identities, cover styles and one-key restoration (completed 2026-10-14)
- [x] **Phase 4: Post Detail, Threads and Quotes** - Open a post, page its replies, read inline quotes, label tombstones and partial threads honestly (completed 2026-10-14)
- [x] **Phase 5: Search and Profiles** - Keyword search with a Latest/Top toggle, user lookup, profile bio and recent posts (completed 2026-10-15)
- [x] **Phase 6: Bookmarks, Likes and the Mirror-Only Store** - The saved lists from the session, mirrored to `saved.db`, with the store structurally unable to hold anything else (completed 2026-10-15)
- [x] **Phase 7: Media, Links and Clipboard** - External openers behind a renderer interface, link copy / open, and cover-breaking actions gated (completed 2026-10-15)
- [x] **Phase 8: Operations, Repair Path and Audit** - `doctor`, the runbook, and the whole-tree audits (completed 2026-10-15)

## Phase Details

### Phase 1: Reachable Transport and One Live Read

**Goal**: The client reaches x.com through the user's proxy and completes one real authenticated read, and every way that can fail is a named, distinct state rather than an empty timeline.
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: AUTH-01, AUTH-02, AUTH-03, AUTH-04, NET-01, NET-02, NET-03, XAPI-01, XAPI-02, XAPI-03, XAPI-05, XAPI-06, XAPI-07, SEC-01, OPS-05
**Success Criteria** (what must be TRUE):
  1. User can put the session cookie in a `0600` file at `$XDG_CONFIG_HOME/x-terminal/session` (the env path/cookie override wins), the app refuses a looser mode or a symlink with a named error, one command deletes the local copy, and the first run states the account risk.
  2. On startup the app reports exactly one named connectivity outcome — no route to x.com, proxy refused, proxy reached but upstream blocked, TLS/Cloudflare challenge, authenticated read OK — so a dead proxy is never mistaken for an expired session.
  3. A real Following-timeline response is fetched through the proxy with the session cookie and printed as parsed author/text/status (`whoami` names the account the session belongs to): no API key, no paid tier.
  4. Every upstream failure class is its own named outcome — `404`-with-empty-body, `422`, `200`-with-`errors[]`, `429`, load-shedding, broken signer — and none of them renders as an empty timeline.
  5. Operation ids and the features/fieldToggles bag are data in one registry file (no `queryId` literal anywhere else in the repo) that a refresh may only add to, requests obey single-flight / a ≥1.5 s floor / no retry on `429` / a session cap, and the cookie appears in no log, error or request dump (a leak test greps captured output for a fixture cookie).

**Notes**: Research placed `whoami` in the search/profiles phase; it lands here instead, because the self-check's "authenticated read OK" has to name the account it read as (deviation). XAPI-04 (manual per-operation pin) is a property of this phase's registry, but its motivating operations — bookmarks and likes — arrive in Phase 6, so the requirement is mapped there. The fetch layer is `internal/xweb` + `internal/xweb/volatile` (the churn seam) + `internal/xsource`, with `internal/domain` declaring the `Source` port; `cmd/spike` is a throwaway that captures fixtures into `internal/xsource/testdata` and is deleted once Phase 2's screens replace it. Open question settled here by measurement: whether the web `Likes` operation answers at all — Phase 6 must degrade honestly either way.
**Plans**: TBD

### Phase 2: Following Timeline in the Pane

**Goal**: The user reads their Following timeline in a terminal pane — scrolling back through older posts and refreshing deliberately — and the pane never lies about what it is showing.
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: FEED-01, FEED-02, FEED-03, FEED-04, FEED-05, REND-01, REND-02, REND-03, REND-04, REND-05, REND-06, REND-08, OPS-01, OPS-02, OPS-04
**Success Criteria** (what must be TRUE):
  1. The Following timeline is the default list, and the algorithmic feed is reachable only through an explicit toggle.
  2. Scrolling with keys loads older posts keyset page by page; the viewport never jumps when a late page arrives, no post appears twice, and a stalled cursor is reported instead of looping.
  3. Refresh is an explicit key: it preserves selection and scroll and announces "N new posts" instead of silently replacing what is on the list; a fetch that fails partway still renders what was received, with the failure stated in one quiet status line.
  4. The four named failure states each render with a distinct remedy — expired session, rate limited (with the computed reset time), offline / no route, upstream shape change — alongside a live request-budget line, and a soft ban (`code 88` with budget left) stops the session and says so rather than being retried.
  5. The renderer and navigation contract holds: borderless, unbranded and engagement-count-free; one width model shared by wrap, pad and truncate and correct for CJK, emoji and grapheme clusters; correct reflow on resize; colour degrading truecolor → 256 → 16 → mono honouring `NO_COLOR` and `TERM=dumb` with no meaning carried by colour alone; vim-ish keys, `?` help, shell completion and a real back stack all derived from one keymap table; golden frames at 80 and 120 columns, inside and outside tmux, as the view specification.

**Notes**: The navigation shell and back stack land here (REND-06), not with the first drill-down: Phases 4–7 execute in parallel and all of them push and pop the same stack, so it has to be a Phase 2 property rather than something the thread phase introduces (deviation from the research, which placed the stack with post detail). The classifier is built in Phase 1; this phase renders its states and adds the rate ledger, the budget line and the soft-ban stop.
**Plans**: TBD
**UI hint**: yes

### Phase 3: Three-State Disguise on the Working Renderer

**Goal**: The user can wear the feed as work-software output or replace the pane with a one-key panic frame — identities aliased, nothing on screen naming the real product — and step back without losing their place.
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: DSGT-01, DSGT-02, DSGT-03, DSGT-04, DSGT-05, DSGT-06, DSGT-07, DSGT-08, DSGT-09, DSGT-11
**Success Criteria** (what must be TRUE):
  1. Single keys switch normal → immersive → panic and back; panic renders in one frame, and the same key restores the exact previous list, selection and scroll without issuing a request.
  2. Panic leaves nothing reachable behind: the scrollback is cleared and repainted (under tmux, the multiplexer's own history buffer too) — verified by scrolling up under the probe harness — and the terminal title, plus the tmux window name where applicable, becomes the cover's own name, restored on exit.
  3. Losing window focus enters panic automatically, and regaining focus does not restore the cover by itself.
  4. At least four immersive cover styles (timestamped service log, git-log/commit entries, CI/build output, generic internal-tool layout) are selectable and switchable at runtime; author names and handles are replaced throughout by deterministic aliases, stable across restarts, with no alias table written to disk; and no real handle, product name, status line, hint, engagement count, avatar or icon is rendered anywhere while covered.
  5. The whole surface that exists works under the cover with unchanged keys (list, pagination, refresh, help) because the cover is a renderer mode rather than a per-screen copy, and the active display state plus cover style persist in the config file so the user picks the startup state.

**Notes**: Sequential home for DSGT-*: the disguise requirements postdate the research, and retrofitting a cover into surfaces built one at a time is the failure mode to avoid. Sitting immediately after Phase 2 builds the cover on a working renderer and makes every surface from Phase 4 on written against it — that is DSGT-08's whole-surface guarantee, re-audited in Phase 8. DSGT-10 is deliberately not here: it gates browser/media/clipboard actions, which arrive in Phase 7.
**DSGT-02 evidence (measured, not assumed)**: Probe D/E (`TERMINAL-MEDIA-PROBE.md`, 2026-10-14) settles the mechanism, with a control. Direct Ghostty 1.3.1: `ESC[3J` (ED 3) clears the scrollback — after six injected Shift+Page_Up no labelled feed line survives, while the same run without ED 3 leaks the feed immediately. Under tmux 3.7c, ED 3 does nothing to tmux's own history (401 matching `capture-pane -S -5000` lines before and after); `tmux clear-history -t <pane>` drops it to the 17 lines still on the visible pane. What the plans inherit: (1) order matters — paint the cover first, then clear history, because clearing while the feed is still on the visible pane leaves that pane as the only remaining history; (2) two mechanisms satisfy one requirement — ED 3 on a direct terminal, `tmux clear-history -t $TMUX_PANE` under tmux, run as a subprocess behind the same effect/`Cmd` boundary as any other I/O, with a missing `$TMUX_PANE` treated as "no tmux history to clear" rather than an error; (3) the probe's harness (`wtype` input + `grim` capture) is the acceptance check — scrollback behaviour is testable without a human watching.
**Plans**: TBD
**UI hint**: yes

### Phase 4: Post Detail, Threads and Quotes

**Goal**: The user opens a post and reads its conversation — replies a page at a time, quotes inline, tombstones and unavailable replies labelled honestly.
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: THRD-01, THRD-02, THRD-03
**Success Criteria** (what must be TRUE):
  1. Opening a post reads the focal post and its reply thread page by page — never one unbounded burst — and leaving it restores the previous list, selection and scroll.
  2. Inline quoted posts render in place and can be opened as their own conversation.
  3. Tombstones and limited-visibility placeholders render as themselves, and a reply section that is structurally unreachable says "partial thread" instead of truncating silently.

**Notes**: Parallel-safe: independent of Phases 5, 6 and 7 (different packages; shares only the `Source` port, the list view, the navigation shell and the cover layer). Drive the decoder from the fixtures captured in Phase 1.
**Plans**: TBD
**UI hint**: yes

### Phase 5: Search and Profiles

**Goal**: The user searches posts by keyword and finds people, then reads a profile's bio and recent posts through the same list they already know.
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: DISC-01, DISC-02, DISC-03
**Success Criteria** (what must be TRUE):
  1. Keyword search returns posts with an explicit Latest/Top toggle, and its results page the same way the timeline does — no duplicates, no viewport jumps.
  2. Users are findable by handle or display name.
  3. A profile shows bio, counts and join date, and that account's recent posts scroll and page with the same keys as the timeline.

**Notes**: Parallel-safe: independent of Phases 4, 6 and 7. The `whoami`/session-status line the research placed here moved to Phase 1 (see Phase 1's notes); this phase proves the list view was extracted entity-agnostically in Phase 2.
**Plans**: TBD
**UI hint**: yes

### Phase 6: Bookmarks, Likes and the Mirror-Only Store

**Goal**: The user reads their own bookmarks and likes from the active session and can open that mirror on disk with the network down — while the timeline stays structurally unable to reach disk.
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: SAVE-01, SAVE-02, SAVE-03, SAVE-04, SAVE-05, SEC-02, XAPI-04
**Success Criteria** (what must be TRUE):
  1. The bookmarks list reads from the active session and pages like any other list.
  2. The likes list reads from the session; if that endpoint is unavailable, the list degrades to an honest message and nothing else in the app breaks.
  3. Bookmarks and likes mirror into `saved.db`, which opens and reads with the network down; sync is incremental — it stops at the first page containing no unknown ids — and a full re-list marks rows `present = 0` instead of deleting them.
  4. `saved.db` can only ever hold bookmarks and likes: the store's write API accepts nothing but a saved item, a schema-allowlist test fails on any unlisted table or column, and an import-boundary test proves the timeline model never reaches the store.
  5. When bundle-scraping cannot see the Bookmarks/Likes operations, the user can pin those operation ids by hand in the registry — no code change — and the mirror stays worth having with bookmarks alone.

**Notes**: Parallel-safe: independent of Phases 4, 5 and 7. This is the only persistence in the system, so the privacy promise is proven here rather than asserted. XAPI-04's pin mechanism is built in Phase 1 and mapped here because bookmarks and likes are the operations it exists for.
**Plans**: TBD
**UI hint**: yes

### Phase 7: Media, Links and Clipboard

**Goal**: The user opens a post's images and video in external programs and copies or opens links — with nothing silently unreachable, and cover-breaking actions gated in immersive mode.
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: MEDIA-01, MEDIA-02, MEDIA-03, REND-07, DSGT-10
**Success Criteria** (what must be TRUE):
  1. A post's media opens in an external program — `xdg-open` for images and links, `mpv` for video and animated media.
  2. Media opening sits behind a renderer interface, so an in-terminal implementation is one new file and no change to the list or detail code.
  3. With no opener available, the raw URL prints instead, so media is never silently unreachable.
  4. The user can copy a post's link (OSC 52, falling back to `wl-copy`) and open the post in a browser.
  5. In immersive mode, cover-breaking actions — browser open, media open, copying a handle-bearing URL — are unavailable until the user deliberately leaves the cover.

**Notes**: Parallel-safe: independent of Phases 4, 5 and 6. v1 ships no in-terminal image code — settled empirically, not assumed; the renderer interface exists so the v1.x Kitty-graphics implementation is a one-file addition.
**Plans**: TBD
**UI hint**: yes

### Phase 8: Operations, Repair Path and Audit

**Goal**: When the upstream breaks — a rotated query id, a broken signer, an expired cookie, a rate limit — the user can find out exactly what broke and repair it without touching code.
**Mode:** mvp
**Depends on**: Phases 1–7 (audits the whole build and reports the store schema)
**Requirements**: OPS-03, OPS-06, SEC-03
**Success Criteria** (what must be TRUE):
  1. `doctor` prints presence facts only — proxy reachability, session validity, signer self-test, snapshot age, store schema version, request budget — and never a token value.
  2. The runbook covers operation-id rotation, features-blob drift, signer break, session expiry, `429`, soft ban and accidental distribution, and following it repairs each case without a code change.
  3. A repo-wide audit proves no post, reply, like, repost, follow or bookmark write call exists anywhere in the build, and re-runs the cookie-leak sweep across `doctor` output, logs, error paths and crash output.
  4. Everything delivered since the disguise phase is re-checked under the cover: no real handle, product name or hint chrome, all keys unchanged, cover-breaking actions still gated.
  5. The guardrail audit passes across the tree: no write path, no timer, no second persistence file, no second account, no distribution artifact — including no packaging, no image and no hosted mode.

**Notes**: Last because it needs a failure to render from and a store schema to report; the error taxonomy and rate ledger it reads were built in Phases 1–2. The research's "per-operation pin override UI" and "safe-mode request cap" are already delivered as data and enforcement in Phase 1 (XAPI-04, XAPI-07).
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8.
Parallel fan-out point: once Phase 3 lands, Phases 4, 5, 6 and 7 are mutually independent and may execute in parallel.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Reachable Transport and One Live Read | 3/3 | Complete    | 2026-10-14 |
| 2. Following Timeline in the Pane | 5/5 | Complete    | 2026-10-14 |
| 3. Three-State Disguise on the Working Renderer | 3/3 | Complete    | 2026-10-14 |
| 4. Post Detail, Threads and Quotes | 3/3 | Complete    | 2026-10-14 |
| 5. Search and Profiles | 3/3 | Complete    | 2026-10-15 |
| 6. Bookmarks, Likes and the Mirror-Only Store | 4/4 | Complete    | 2026-10-15 |
| 7. Media, Links and Clipboard | 3/3 | Complete    | 2026-10-15 |
| 8. Operations, Repair Path and Audit | 3/3 | Complete    | 2026-10-15 |
