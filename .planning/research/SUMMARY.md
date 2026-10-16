# Project Research Summary

**Project:** x-terminal
**Domain:** Read-only terminal (TUI) client for x.com — single-user personal tool, reading x.com's own internal web GraphQL endpoints with the user's session cookie
**Researched:** 2026-10-14
**Confidence:** MEDIUM-HIGH overall (stack and code-reading claims HIGH; upstream survivability and two named surface questions MEDIUM/`[UNVERIFIED]`)

## Executive Summary

This is a client for an undocumented, actively hostile-upstream API, wrapped in a terminal UI that is the *easy* half. Four independent research passes converge on the same shape: no existing library, in any language, covers the endpoint set this project needs (`HomeTimeline` is missing from every maintained x.com client; bookmarks from the TypeScript one; likes from the Python and Go ones), so the data layer must be hand-rolled as a **thin, swappable adapter**, borrowing request mechanics from the maintained scrapers (`twscrape`, `@the-convocation/twitter-scraper`, `twikit`) rather than depending on them. Everything above that adapter — key grammar, list/detail navigation, wrapping, golden-frame views — is ordinary terminal-application work this author has already shipped once in `/home/user/repos/rss-tm`.

The recommended approach is **Go 1.27.2 + Bubble Tea v2**, with `internal/domain` declaring the `Source` port, `internal/xweb` owning HTTP mechanics, `internal/xsource` owning x.com response semantics, and `internal/xweb/volatile` — the **containment seam** — owning the three things that rotate on someone else's schedule: the `queryId`/operation map, the `features`/`fieldToggles` bag, and the `x-client-transaction-id` signer. Cursors are opaque, identity is the post id, the timeline lives only in memory, and the sole disk surfaces are the `0600` session file, the volatile snapshot cache, and a SQLite mirror that can only ever hold bookmarks and likes. Media opens in an external program behind a renderer interface; v1 ships zero image code.

The risks are not feasibility risks. They are (a) **account-shaped enforcement**: reads carry the user's real identity, the thresholds are undocumented and unstable, and a single retry loop can convert a 15-minute throttle into a 24-hour cross-endpoint lock; (b) **silent failure by construction** — a stale `queryId` yields an empty `404`, a broken signer 404s *every* endpoint at once, and a rotated ID looks exactly like "no posts"; and (c) a **measured network reality** the four researchers did not have: this host cannot reach `x.com` at all without the user's proxy, so proxy wiring and a loud connectivity self-check are first-phase work, not hardening. The whole plan therefore starts by retiring that risk with one real request path, and carries a first-class repair/diagnosis story (`doctor`, a response classifier, a pin override) as a budgeted phase rather than a nice-to-have.

## Key Findings

### Reconciled Verdict: YES — to build, and a conditional yes to depend on

The four files disagree only in emphasis. **STACK.md says YES** (the *build* is feasible and the stack is settled); **PITFALLS.md says MAYBE** (sustained, un-degraded *use* is not engineerable — only minimisable); FEATURES.md says YES with four conditions; ARCHITECTURE.md says YES with one condition. Reconciled once, and not restated per file:

> **The build is a yes. The stack is decided. The residual risk is survivability and honestly-labelled breakage, not whether the endpoints answer.** It stays a yes only while all of these hold: read-only in every code path; manual refresh only, one request in flight, no timers ever; personal and never distributed (no public repo/binary/image/instance); a budgeted repair path (`doctor` + response classifier + pin override); the UI degrading to "open in browser" instead of pretending; and the account risk stated to the user on first run. Drop any one and the answer drifts toward the Nitter pattern that attracted a cease-and-desist in Aug 2026.

### Recommended Stack

Go 1.27.2 + `charm.land/bubbletea/v2` v2.1.0 (`lipgloss/v2` v2.0.6, `bubbles/v2` v2.2.1, `charmbracelet/x/ansi` v0.11.9), a hand-rolled `net/http` GraphQL client, and `modernc.org/sqlite` v1.60.1 for the mirror. Decisive on this project's merits: the async model *is* the data model (pure `Update` returning a `tea.Cmd`, exactly `rss-tm`'s enforced convention), 0.9 ms measured process start with a 2.35 MB `CGO_ENABLED=0` binary, and the best Unicode-width story of the four candidates (mode 2027 + `displaywidth`). The runner-up — TypeScript on Bun — wins only if you would rather inherit `@the-convocation/twitter-scraper`'s cookie/signing machinery than port it, and it still lacks `HomeTimeline` and bookmarks. Rust is out (no toolchain, dead x.com ecosystem); Python is the wrong shape for a daily driver.

**Core technologies:**
- Go 1.27.2 — language, toolchain, static binary: measured 0.9 ms start, no interpreter, no venv, already installed.
- `charm.land/bubbletea/v2` v2.1.0 + `lipgloss/v2` + `bubbles/v2` — TUI runtime and widgets; v2's `charm.land` import paths are incompatible with v1, so pick v2 everywhere.
- Hand-rolled `internal/xweb` + `internal/xsource` on stdlib `net/http` — no dependency covers the required endpoint set, and the project constraint demands a swappable fetch/parse layer.
- `modernc.org/sqlite` v1.60.1 (BSD-3-Clause, pure Go) — keeps `CGO_ENABLED=0`; `mattn/go-sqlite3` (cgo) breaks the static-binary property.
- `charmbracelet/x/exp/golden` v0.1.0 — golden frames are the view specification, as in `rss-tm`.
- Conditional only: `github.com/bogdanfinn/tls-client` v1.16.0 (or `refraction-networking/utls` v1.8.2) **iff** Phase 1 proves plain Go TLS is challenged on authenticated reads.

### Expected Features

The product's job is the whole read surface with none of the conveniences. Everything else on the list — the differentiators — is restraint expressed in code.

**Must have (table stakes), v1:**
- Following timeline with keyset pagination that never loses position — the reason the tool exists.
- Post detail with paginated reply/thread expansion and inline quotes — a timeline without threads is a headline list.
- Search (posts + users) and profile (bio + recent posts) — reuse the same entity-agnostic list view.
- Bookmarks and likes read from the session, with `whoami` — required, and the one surface whose availability is `[UNVERIFIED]`.
- Cookie auth from a `0600` file with an env override; manual refresh that preserves position; error surfacing as four named UI states (expired / rate-limited / offline / upstream shape change).
- External media opener behind a renderer interface; borderless, low-colour, count-free rendering; a keymap table that feeds help and input from one source.

**Should have (differentiators):** structural timeline-never-persisted enforcement (package boundary + test), "N new posts" refresh banner, visible request budget, `doctor` self-diagnosis, colour degradation ladder, clipboard fallbacks, shell completion from the keymap table.

**Defer (v2+):** SQLite mirror for bookmarks/likes (v1.x trigger: the list feels slow), Kitty-graphics renderer (v1.x trigger: media is opened often), export/pipeline mode, list/community timelines, digest mode, a second source behind the same seam.

**Anti-features, refused by decision:** background polling/streaming, write actions, in-terminal video/audio, inline images by default, notification badges/bell, algorithmic feed as default, multi-account rotation, timeline cache/read-state on disk, engagement counts as chrome, mouse-first interaction, AI summarisation, browser-cookie extraction, themes/plugin systems.

### Architecture Approach

Six layers, data flowing down and effects flowing up as values: `internal/tui` (pure `View`/`Update`, forbidden from importing `net/http`, `os/exec`, `database/sql`, `store`, `xweb`, `xsource`) → `internal/screen` (pure reducers, page cache, dedupe set, single-flight latch) → `internal/domain` (**declares the `Source` port**; `Post`/`Author`/`Thread`/`Page`/`Cursor`) → `internal/xsource` (x.com semantics) → `internal/xweb` (HTTP mechanics) → `internal/xweb/volatile` (the churn seam), with `session`, `store`, `config`, `media` as leaves and `internal/cli` as the single composition root. The consumer-side port pattern is copied from `rss-tm` (`tui.Source`, `daemon.FetchFunc`), never imported from it.

**The containment seam** is `internal/xweb/volatile`: the only package permitted to know a `queryId`, a feature flag, a header name, or `abs.twimg.com`, behind three tiny interfaces (`Ops.URL`, `Features.Features/FieldToggles/Absorb336`, `Signer.Header/Reset`). A pinned, checked-in snapshot is authoritative; runtime bundle-scraping refresh may only *add* to it; an unknown operation is a typed `ErrUpstreamSchema`, never a silent fallback to a stale id; no package above it may name any of the volatile vocabulary, enforced by an AST import scan with anti-vacuity arms. `session` stays separate from `config` because the cookie must not sit on a generally-editable, round-tripped surface; `store`'s insert API accepts only `domain.SavedItem`, so "no timeline on disk" is a type-system fact with one `PRAGMA table_info` test behind it.

### Critical Pitfalls

1. **Cookie-authenticated reads put the user's real account in the enforcement path** — prior art states it plainly (convocation's ban warning; x-native's "use a dedicated account"; Twikit's "reduces but doesn't eliminate"). Mitigate with single-flight requests, one request per action, a ≥1.5 s inter-request floor, page sizes at X's defaults, no timers, no retry loops, a session-wide request cap, and a one-line risk statement on first run.
2. **A naive HTTP client gets `403` (Cloudflare) or an empty `404` (missing signer) and the developer blames the API** — three independent layers must line up (TLS/HTTP fingerprint, `x-client-transaction-id`, header coherence with `ct0` mirrored into `x-csrf-token` and `x-twitter-auth-type: OAuth2Session`). Classify before parsing; never log bodies as the primary diagnosis.
3. **The transaction-ID signer breaks silently, 404ing every endpoint at once** — X has shipped malformed manifests (unquoted keys) and changed chunk-hash lengths. Isolate it behind `Signer`, self-verify at startup, cache the last-known-good key material on disk with a TTL, parse defensively, and report "bundle schema changed" as its own error class.
4. **`queryId` / features / response shapes rotate on different clocks, with no clear error** — a stale ID is a `404` with an empty body, which renders as "no posts". One machine-writable registry (`operationName → {queryId, features, fieldToggles}`), zero `queryId` literals elsewhere, auto-heal then pin, and a specific classifier for `404+empty` / `422|errors[]` / `200`-with-`errors`.
5. **Rate limits and the invisible 24-hour soft ban** — per-endpoint 15-minute budgets plus a fast limiter that trips at ~1 req/s and blocks the account across endpoints. Read `x-rate-limit-remaining`/`-reset`, never retry a 429, treat `code 88` with budget remaining as a soft ban and stop, keep load-shedding a distinct state.
6. **Cookie leakage** — no dumps, no `%+v` of requests, a redacting `Secret` type, `0600` + symlink refusal, a host-scoped jar (twscrape shipped a fix for cookies sent outside x.com), and a leak test that greps captured output for a fixture cookie.
7. **Grapheme-cluster width mismatch corrupts the layout** — one width model, routed through every wrap/pad/truncate; prefer tmux's wcwidth semantics under `$TMUX`, grapheme semantics in bare Ghostty; ASCII markers instead of emoji decorations; golden frames at 80/120 columns inside and outside tmux.

### Upstream Reachability — measured constraint added after the four research files

Measured on this host (2026-10-14, after the researchers were launched): DNS resolves `x.com` → 162.159.140.229 (Cloudflare), but TCP/TLS to `x.com`, `api.x.com`, `twitter.com`, `cdn.syndication.twimg.com` and `abs.twimg.com` all time out, while `https://example.com/` returns HTTP 200 in 0.78 s. No listener on the usual local proxy ports (7890/1080/2080/10808…), no proxy environment variables set, no mihomo/clash/sing-box configuration on disk, Tailscale up but no exit node in evidence. The user runs a local proxy client that is simply not started right now, and has decided the application must therefore (a) support explicit proxy configuration in its config file plus the standard `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY` environment variables, over both HTTP and SOCKS5, and (b) run a **startup connectivity self-check that fails loudly and legibly** — a silent no-op is *the* failure mode to design against.

Three consequences that change the plan: (i) **the Phase 0 "one live request path end-to-end" spike cannot succeed without a working proxy**, so proxy plumbing and the connectivity check belong to that same first phase, not to a later hardening phase; (ii) the check must distinguish its states — no route, proxy refused, proxy reached but upstream blocked, TLS/Cloudflare challenge, `404`-empty, auth rejected (so a dead proxy is never reported as an expired session); and (iii) the empirical terminal-media result already in hand caps v1 at external opening, so **no media capability research or spike remains**. Two grounded implementation notes: Go's stdlib env handling (`vendor/golang.org/x/net/http/httpproxy/proxy.go`) reads only `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` and their lowercase forms — `ALL_PROXY` must be read by this project's own code; but `net/http`'s transport natively accepts `socks5`/`socks5h` proxy URLs (`net/http/transport.go`), so SOCKS5 needs no third-party dialer. A proxy forwards bytes and does not change the TLS ClientHello, so it neither fixes nor worsens fingerprint-based detection.

### Open Spike Questions Carried Forward (unsoftened)

1. **Own-likes availability via the web `Likes` endpoint.** `twscrape` exposes `bookmarks()` but no likes operation; it *removed* `liked_tweets` after the June 2024 private-likes change, while `twikit` and API captures still carry a `Likes` op. `[UNVERIFIED]` — one Phase 1 request settles it. Fallback designed in advance: the likes screen degrades to an honest message, and the mirror must stay useful with bookmarks alone.
2. **Whether plain Go TLS is challenged on authenticated reads.** Expect success (both reference libraries treat fingerprint spoofing as opt-in), but this is exactly the class of claim this project settles by measurement, not documentation. Mitigation is a one-file transport swap behind the existing `RoundTripper` seam — do not add it pre-emptively.

### Contracts That Must Not Be Softened

- **Query-id churn containment.** IDs are *data*, never code: a single generated registry file with markers, zero `queryId` string literals anywhere else in the repo, a pinned snapshot that is authoritative, refresh that only adds, `doctor --refresh-ids`, and a per-operation manual pin override (bookmarks and likes are exactly the operations bundle-scraping cannot see). The test of a fix is that it touches **one file**.
- **Rate-limit and soft-ban discipline.** Single-flight always; a hard inter-request floor of ~1.5 s (1 req/s is already documented as enough to trip the fast limiter, so treat the gap as a floor, not a target); no background prefetch, no parallel media/metadata/link fetches; 429 → show the computed reset time and disable refresh until then; `code 88` with budget remaining → soft-ban message, stop for the session; 5xx/LoadShed → bounded backoff with a *different* message; refresh key debounced so key-repeat cannot burst.

## Implications for Roadmap

Recommended structure: seven vertical-MVP phases. Each delivers something usable end-to-end and retires a named risk; the first retires the one that can invalidate everything.

### Phase 1: Reachable transport and one live read (risk retirement)
**Rationale:** the dominant risk is upstream reachability and request legitimacy, not UI work. Measured fact: nothing works on this host until the proxy path is wired and verified — so proxy handling is this phase's work, not Phase 7's. Everything later assumes "the upstream answers"; this phase makes that true or makes it a named, cheap failure on day one (and captures the real fixture shapes that fix the `Post`/`Cursor` interfaces).
**Delivers:** Go module and package skeleton; `internal/config` (paths, keymap seed, proxy block, media opener); `internal/session` cookie type (`0600` file at `$XDG_CONFIG_HOME/x-terminal/session`, env override, redacting `String`/`MarshalJSON`, mode and symlink checks, leak test); proxy support in `internal/xweb` (config file plus `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY`/`NO_PROXY`, HTTP and SOCKS5); a **startup connectivity self-check** with distinct named outcomes; the query-ID registry (pinned snapshot + refresh script) and the ported `x-client-transaction-id` signer; `cmd/spike` — a throwaway that reads the cookie, issues one authenticated GraphQL read, and prints parsed post text, author and status; fixtures recorded into `xsource/testdata`.
**Addresses:** the Auth requirement; the first half of Error surfacing (upstream-shape-change and connectivity states).
**Avoids:** Pitfalls 3, 4, 5, 6, 7; the newly measured "no route to upstream" class; and quietly answers spike questions 1 and 2 above.
**Build-order note:** ARCHITECTURE's Phase-0b media probe is **already discharged** by `TERMINAL-MEDIA-PROBE.md`; do not re-run it and do not plan media work here.

### Phase 2: Following timeline in the pane (first usable slice)
**Rationale:** this is the product. It is also where every request-discipline rule — pagination, dedupe, stall detection, single-flight, the rate ledger, error states — is written once and reused verbatim by every later screen. Error surfacing must land *before* the "load older" key is bound, because pagination is the first feature that can genuinely hit a limiter.
**Delivers:** `internal/domain` types + the `Source` port; `internal/xsource` decode + `internal/xweb` transport with classifier, rate ledger and the one-shot tid retry on `404`; `internal/tui` shell with pure `Update`/`View`, the keymap table (feeding help overlay and input), the borderless low-colour renderer with **one** width helper and a `NO_COLOR`/`TERM=dumb`/16-colour ladder, and golden frames at 80 and 120 columns; Following timeline with keyset pagination, id-keyed dedupe, `(cursor, entry-id)` stall detection, manual refresh that preserves selection and scroll, and the four named failure states in exactly one quiet status line.
**Uses:** Bubble Tea v2 + `bubbles` viewport/list; `charmbracelet/x/ansi` for width and `OSC 8`.
**Implements:** L0–L5 of the architecture, plus the import-boundary tests in both directions (UI cannot reach I/O; `volatile` cannot reach `domain`/`xsource`).
**Avoids:** Pitfalls 1 (single-flight, no timers), 5 (empty-list-is-not-success), 8 (ledger + debounce), 9 (width), 11 (pagination/dedupe/ordering), 13 (tty, colour, resize).

### Phase 3: Post detail, reply threads and inline quotes
**Rationale:** the fiddliest decoder in the project, and the one that benefits most from the fixtures captured in Phase 1. It requires the navigation stack to be a first-class property, not a per-view hack — retrofitting "back restores list, selection and scroll" after the views exist means rewriting every view's state.
**Delivers:** two-stage load (post then thread page), ancestor chain above the focal post, reply groups with the `ShowMoreThreads` per-module cursor, `SelfThread` handling, inline quoted posts, tombstone/limited-visibility placeholders, and honest partial-thread labelling.
**Avoids:** Pitfall 11 in the thread context; the N+1 reply anti-pattern; silent truncation (explicit continuation affordances).

### Phase 4: Search (posts + users) and profiles — parallel with Phase 3
**Rationale:** reuses pagination and post rendering wholesale; only two new decode paths. Entity-agnostic list view is a Phase 2 property, and this phase is the proof that it was extracted properly.
**Delivers:** `SearchTimeline` with a Latest/Top toggle, the people product returning users through the same list view, `UserByScreenName` header + paginated `UserTweets`, and a `whoami`/session-status line.

### Phase 5: Bookmarks, likes and the mirror-only store
**Rationale:** the only persistence in the system, and therefore the phase where the privacy promise has to be *proven* rather than asserted. It is also the phase gated on spike question 1; sequence it so the mirror is useful with bookmarks alone.
**Delivers:** `internal/store` (two-table schema, WAL, ordered migrations with the version bump in the same transaction, incremental sync that stops at the first page with no unknown ids, full re-list marking `present = 0` instead of deleting, `query_only` reader); the `Saved` screen reading from the mirror when offline; the insert API accepting only `SavedItem`; `PRAGMA table_info` allowlist test; documented pin override for the `Bookmarks`/`Likes` operations.
**Avoids:** Pitfalls 5 (pin override path), 11, 12 (a second file appearing), and any timeline write.

### Phase 6: Media opening, links and clipboard — parallel with Phases 3–5
**Rationale:** the v1 decision is settled and empirically justified; what is left is plumbing that must exist behind an interface so the v1.x Kitty renderer is a one-file addition.
**Delivers:** `media.Renderer` with an external-program implementation (`xdg-open` for URLs; `mpv` for video/animated behind an allowlisted command from config), `OSC 8` hyperlinks on media and post links, a printed-URL fallback whenever an opener is unavailable, and a clipboard chain of `OSC 52` → `wl-copy` (never `xclip`/`xsel`, absent here).
**Explicitly out of v1:** inline image rendering. The probe measured bare APC in tmux failing silently and DCS-wrapped APC rendering with text drawing *across* the image; Ghostty implements no sixel and no iTerm2 images, and Kitty animation is unimplemented in 1.3.1. A future renderer must detect via `TERM_PROGRAM`/`TERM` (never `KITTY_WINDOW_ID`), use Unicode placeholders plus tmux passthrough when `$TMUX` is set, request an explicit placement size, reserve the exact cells, force a repaint, and live only in a dedicated non-scrolling pane.

### Phase 7: Operations, repair path and audit
**Rationale:** the failure modes here are silent by construction, so diagnosis is a real phase of work, not polish. It comes last only because it needs a failure to render *from* — but if daily use starts before it lands, a session expiry will look like a crash.
**Delivers:** `doctor` (signer self-test, `--refresh-ids`, session probe with named outcomes, response classification, snapshot age, store schema version, request budget line — presence facts only, never a token prefix); per-operation pin override UI; the safe-mode session request cap; the leak/redaction audit and a repo-wide `grep` sweep as an acceptance check; runbook entries for ID rotation, features-blob drift, signer break, session expiry, 429, soft ban and accidental distribution; release-hygiene checks (no packaging, no image, no hosted mode).
**Avoids:** Pitfalls 2, 7, 8, 12 at the audit level; makes Pitfalls 4 and 5 one-line repairs in practice.

### Phase Ordering Rationale

- **Phase 1 before everything** because every later design assumes the upstream answers *from this host*, and the proxy requirement means it currently does not. It is also the cheapest way to capture real fixture shapes.
- **Phase 2 before the rest** because pagination, dedupe, request discipline and the width model are written once and reused by four screens; writing the thread decoder first would tempt a second paging implementation into post detail.
- **Phases 3–6 are mutually independent** once 1 and 2 land — different packages, shared only through the `Source` port and the list view. This is the natural fan-out point for parallel work, one package boundary per slice.
- **Phase 7 needs a failure to render**, hence its position; the *error-surfacing UI states* it audits are already required in Phase 2.
- **No phase may add** a write path, a timer, a second persistence file, a second account, or a distribution artifact. That is the guardrail that keeps the whole plan a yes.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1** — the volatile surface is the least-documented part of the build: the transaction-ID algorithm is published only as reverse-engineering write-ups and non-Go ports, `queryId` discovery must handle both Vite and legacy webpack bundle layouts, and no Go implementation is published as a module. Research/port from `twscrape`'s `xclid.py` and `update-gql-ops.py` before writing code.
- **Phase 3** — the `TweetDetail` entry grammar (ancestors, related-tweets rails, per-module cursors, `SelfThread`, visibility wrappers) is the fiddliest decoder here; drive it from recorded fixtures.
- **Phase 5** — confirm the `Likes` endpoint's current behaviour and the `Bookmarks` pin-override path against the live site in the same session that captures fixtures.

Phases with standard patterns (skip research-phase):
- **Phase 2** — the TUI half is well-trodden and already proven in this author's stack: pure `Update`, keymap table, golden frames, borderless rendering.
- **Phase 6** — the media/clipboard/open plumbing is conventional (`xdg-open`, `mpv`, OSC 52, `wl-copy`), and the terminal capability question is already answered empirically.
- **Phase 7** — `doctor`/ops/audit work is ordinary engineering once the error taxonomy exists.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Every version pulled from its registry/repo on 2026-10-14; startup and binary size measured on this host; x.com mechanics corroborated by two independently maintained implementations; the framework ranking itself is a judgement, stated as such. |
| Features | MEDIUM-HIGH | Competitor landscape HIGH (URL-cited, activity dates machine-read); three table-stakes rows rest on upstream surfaces that are unverified until Phase 1; the likes row is `[UNVERIFIED]`. |
| Architecture | MEDIUM-HIGH | HIGH where read out of primary code (twikit, twscrape, `rss-tm` conventions); the churn-seam shape is a design judgement `[INFERENCE]`, backed by dated changelogs showing monthly ID rotation. |
| Pitfalls | MEDIUM | Mechanisms HIGH (live code, issue threads, host probes); X's own thresholds and terms text MEDIUM — X publishes neither. |

**Overall confidence:** MEDIUM-HIGH.

### Gaps to Address

- **Own-likes availability** — settles in Phase 1 with one request; design the mirror so bookmarks alone justify it, and let the likes screen degrade honestly.
- **Plain Go TLS vs Cloudflare on authenticated reads** — one authenticated read in Phase 1 decides whether `bogdanfinn/tls-client` enters the build; the mitigation is a same-seam transport swap.
- **`auth_token` lifetime in 2026** — determines how often the human re-pastes cookies and whether a "cookie likely expired" hint belongs in the UI; observe or inspect the cookie's own expiry once.
- **Whether some `features` flags must be `false` rather than absent** — naive self-healing flips everything to `true` on `(336)` and can loop; force a `(336)` deliberately and observe once.
- **Proxy semantics against the user's actual client** — HTTP CONNECT vs SOCKS5, whether it is on a different port than the usual ones, and whether it needs authentication. Untestable until the client is started; the self-check must report each variant distinctly.
- **Whether a proxy IP changes the bot-detection posture** — a datacenter/VPN egress may be treated differently from the home IP; `[INFERENCE]`, cheap to observe during Phase 1 (compare classification with and without the proxy).
- **Ghostty image retention and `image-storage-limit` under tmux** — only matters if the v1.x image upgrade ships; probe then, not now.
- **Naming reconciliations to fix once, in Phase 1** (the research files differ; the roadmap should assume these): the fetch layer is `internal/xweb` + `internal/xweb/volatile` + `internal/xsource` (ARCHITECTURE's layering) with STACK's `xapi` content folded into them; the session override is an env *path* plus an env *pair* rather than a single cookie variable; "one request per 1.5 s" is the floor, not the target.

## Sources

### Primary (HIGH confidence)

- `.planning/PROJECT.md` — this project's own decisions: read-only v1, manual refresh, no timeline on disk, cookie outside the config surface, not distributed, vertical-MVP phases, English-only, sibling-repo conventions.
- `.planning/research/TERMINAL-MEDIA-PROBE.md` + `evidence/probe-{A,B,C}.png` — host-specific empirical result: Kitty graphics works directly in Ghostty 1.3.1; `KITTY_WINDOW_ID` is not exported; bare APC fails silently inside tmux 3.7c; DCS-wrapped APC renders but tmux's grid does not model image cells, so later text draws across it and scrolling smears.
- Host measurements, 2026-10-14 — Go static binary 2,350,833 bytes / 0.9 ms median start (15 runs); `x.com` and four related hosts unreachable over TCP/TLS while `https://example.com/` returns 200 in 0.78 s; no local proxy listener, no proxy env vars, no proxy client config on disk; `xdg-open`, `wl-copy`, `mpv`, `ffmpeg` present, `xclip`/`xsel`/`chafa`/`viu`/`timg` absent; a non-interactive shell reports `TERM=dumb`, `NO_COLOR=1`.
- Go 1.27.2 source on this host — `vendor/golang.org/x/net/http/httpproxy/proxy.go` (`FromEnvironment` reads only `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` and lowercase forms) and `net/http/transport.go` (transport accepts `socks5`/`socks5h` proxy URLs).
- `/home/user/repos/rss-tm` — the convention reference: hermetic AST import scan with anti-vacuity arms, consumer-side ports, `FetchFunc` seam, ordered migrations with the version bump in the same transaction, WAL read back rather than assumed, golden frames as the view spec.
- x.com internals, from live code: `twscrape` (`account.py`, `queue_client.py`, `xclid.py`, `api.py`, `models.py`, `scripts/update-gql-ops.py`, `changelog.md`), `twikit` (`client/gql.py`, `client/client.py`, `x_client_transaction/transaction.py`), `the-convocation/twitter-scraper` (`src/auth-user.ts`, `src/api-data.ts`, README), `fa0311/TwitterInternalAPIDocument` `docs/json/API.json`; issue threads `twscrape#306` (404-empty without the signer), `twscrape#280` (malformed manifest, features change), `FxEmbed#1283` (per-endpoint signer requirement).
- Enforcement and account risk: `techcrunch.com/2026/09/15/nitter-and-xcancel-are-dead-again-after-xs-latest-legal-actions/`; Nitter repository C&D notice; `nitter#1107` (per-endpoint budgets, ~1 req/s second limiter, 24-hour cross-endpoint block) and `nitter#1349` (opaque 24-hour limit, proxies did not help); Twikit "Account protection"; `hobenshield.com/articles/web-api-re` (header coherence, JA3).
- Terminals and width: Mitchell Hashimoto, "Grapheme Clusters and Terminal Emulators"; `ghostty-org/ghostty#1695`; `mattn/go-runewidth#59`; `charmbracelet/x/ansi` width API; `kovidgoyal/kitty` discussion #4021; `tmux` `CHANGES` (`allow-passthrough`, 3.3); `no-color.org`.

### Secondary (MEDIUM confidence)

- Ghostty capability matrix read from docs **and** the released `v1.3.1` source tree — Kitty graphics, Unicode placeholders, z-index, `image-storage-limit` (320 MB default, no enable/disable toggle), animation unimplemented in 1.3.1, sixel and iTerm2 inline images absent.
- X's Terms of Service (PDF dated 2026-04-10) — quoted from indexed copies; the direct fetch timed out. X API pricing figures quoted as indexed and corroborated by two developer-forum posts.
- Competitor evidence drawn from each project's own README (`xeet`, `XTUI`, `omaX`, `x-tui`, `twit`, plus the dead twterm/rainbowstream/oysttyer lineage and the adjacent-network TUIs `newsboat`/`neomutt`/`aerc`/`tuir`) — the feature exists, quality unverified.

### Tertiary (LOW confidence — needs validation)

- `tweetxvault` (bookmarks/likes archive CLI) as evidence that likes are reachable in some form; it is reported broken by cookie issues, so it is a hypothesis, not a finding.
- Registry/`pkg.go.dev` dates for Go modules, which are MEDIUM because `proxy.golang.org` is unreachable from this host.
- The reading that enforcement targets public alternative front ends rather than single-user personal clients — `[INFERENCE]` from the Nitter and XCancel events, not a stated policy.

---
*Research completed: 2026-10-14*
*Ready for roadmap: yes*
