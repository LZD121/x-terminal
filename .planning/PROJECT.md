# x-terminal

## What This Is

A read-only x.com client that runs inside a terminal pane. It renders timelines,
threads, search results, profiles and bookmark/like lists as plain text, with no
app-like chrome: it should read like a log viewer, not a social app. Single user,
personal tool — built for one person who already lives in a terminal and wants the
feed without the browser.

## Core Value

Browse x.com from inside a terminal pane — timelines, threads, search, profiles and
bookmarks — with no browser, no algorithm-driven app shell, and nothing on screen that
looks like social media. The feed can be worn as a work-software cover while being read,
and hidden entirely with one key.

## Requirements

### Validated

- ✓ Auth: the session cookie is supplied once (0600 file at `$XDG_CONFIG_HOME/x-terminal/session`, loose permissions refused with the fix command, symlinks refused, `X_TERMINAL_AUTH_TOKEN`/`X_TERMINAL_CT0` override), the credential cannot be printed, `whoami` names the account and `logout` deletes the local copy — Phase 1
- ✓ One real authenticated read from this host through the proxy, with every failure mode a named state rather than an empty timeline, and `doctor` reporting five distinct connectivity outcomes — Phase 1
- ✓ Following timeline with keyset pagination that never loses position, manual refresh that preserves selection and scroll, and the four named failure states in one status line — Phase 2
- ✓ Borderless, unbranded, count-free rendering with one width model (`$TMUX`-aware), a colour ladder honouring `NO_COLOR`/`TERM=dumb`, a one-table keymap feeding input/help/completion, a real back stack, and golden frames at 80 and 120 columns as the view specification — Phase 2
- ✓ Three display states (normal / immersive / panic) with four switchable work-software covers, deterministic identity aliases with no table on disk, a panic entry proven in a real terminal and a real tmux pane (scrollback and tmux history both cleared, restore byte-identical with no request), and no covered frame naming the product or any real handle — Phase 3
- ✓ Post detail and threads: the thread frame is the detail view, ancestors → focal → self-thread → replies, quotes openable with `enter`, opening costs no request, and a partially-reachable thread says so only when it is — Phase 4
- ✓ Search and profiles: `/` searches posts with latest/top in place, `u` finds people on the same walk, `a`/`enter` open a profile in two ordered requests, absent counts are omitted never zeroed, and the list machinery is entity-agnostic — Phase 5
- ✓ Bookmarks and likes: a sealed mirror-only store the timeline cannot import, reads by operation name never query id, a sync that marks absent never deletes, and a saved frame that opens with zero requests — Phase 6
- ✓ Media, links and clipboard: external open through one Renderer seam, in-place URL fallback when the opener fails, an honest OSC 52/`wl-copy` chain, and cover-breaking actions gated under immersive — Phase 7
- ✓ Operations and repair: an audit verb whose every check can fail, a doctor that names the repair, and a runbook that cannot rot silently — Phase 8

### Active



### Out of Scope

- Write actions (post, reply, like, repost) — v1 is read-only; avoid the write-auth path and the extra ToS exposure it carries
- Official X API — pay-per-use pricing (≈ $0.005 per post read) makes read-heavy browsing uneconomic; the free tier is insufficient for reading
- Direct messages and notifications — not part of the "read the feed quietly" job
- In-terminal video/audio playback — delegated to external players instead
- Multi-account support — single user, single session
- Persisting the timeline to disk — deliberate: no browsing traces left in the office environment
- Export / pipeline integration (pipe posts to `grep`, dump threads to markdown) — not selected for v1; revisit if it turns out to be the real motivation
- Distribution as a *distributable* artifact — no container image, no hosted instance, no installer, no package recipe. Internal-endpoint usage is against X's terms, so nothing a stranger can pick up and run may exist. **Narrowed 2026-10-15**, when the project was pushed to a private repository with a CI/release job: a GitHub Release carrying source and a Go binary, built out of a private repository for the operator's own machines, is the build this project already was and not a distributable form of it. **Re-opened and accepted 2026-10-16**, when the repository was made *public*: that narrowing's premise — a private repository — is gone, and the Release's source and three platform binaries are now downloadable by anyone with no credential, so a stranger can in fact pick this program up and run it. The author accepted that exposure deliberately, having been shown what a fork makes irreversible. `internal/audit`'s `distribution` check still enforces the same set (image / package / installer / hosted mode) and its comment records this second change rather than going on printing the private-repository reasoning; `RUNBOOK.md`'s "Accidental distribution" section is the repair when something does escape.

### The second display mode (declared exception, 2026-10-17)

The project's plain mode is built so that nothing on screen looks like social media: the reader is a
log viewer, the four covers are work software, and the panic frame is a finished command's output. That
rule is enforced by the banned-vocabulary tests in `internal/render` (cover, corpus, status), and it
continues to hold for every one of those surfaces.

There is now a second presentation — `internal/vivid`, entered only by a hidden key sequence, with no
flag, config key or environment variable anywhere — worn where no disguise is needed. It is a
**declared, narrow exception** to that rule and to nothing else:

- It prints real identities and product wording because it is not a disguise. The exception is a
  different grammar, not a flag on the plain one, so no plain or covered frame can acquire it.
- It stays **read-only**. The write-operation checks are unchanged and vivid mode adds no operation.
- It adds **no persistence**: the marks, the reading position and the images are session memory.
- Its entry is not documented in the repository. `internal/audit`'s `hidden` check holds the two ways
  a start-up path could reach it.

## Context

- **Data source is unofficial.** x.com's own web front end talks to internal
  GraphQL endpoints authenticated by the logged-in session cookie. Reusing those
  is the only zero-cost path that preserves full read fidelity (timeline ranking,
  threads, search, bookmarks). It is also the most brittle part of the system:
  query IDs, required headers and payload shapes change without notice, so the
  parser must stay thin and isolated.
- **X API costs, for the record:** v2 is pay-per-usage — approximately $0.005 per
  post read and $0.015 per post created ($0.20 when the post contains a link),
  with the older $200/month Basic tier as the entry point. Reading a 1,000-post
  timeline costs roughly $5. This is why the official API is out of scope.
- **Prior art exists but is fragile by construction:** the Nitter lineage, RSS
  bridges, and per-call aggregators (e.g. twitterapi.io) all trade stability for
  access. No mature terminal client for x.com exists today, which is part of why
  this is being built rather than adopted.
- **Sibling prior art on this machine:** `/home/user/repos/rss-tm` is the same
  author's Go 1.27 + Bubble Tea v2 terminal *feed* reader (RSS + quote sources),
  16 `internal/` packages, ~2k lines of load-bearing convention: a pure-`Update`
  Model with every read expressed as a `tea.Cmd` (enforced by an import-boundary
  test), a data-table key grammar that the help overlay renders, golden frames as
  the specification for a view, a format-agnostic feed value behind a
  `FetchFunc` function type, and a `quote` package that already demonstrates
  plugging a second, entirely different source into the same scheduler. It proves
  the hard parts of this project are solvable here. It is **not** a constraint: no
  shared code, no keybinding/theme alignment (decided 2026-10-14). Its one real
  gap is secrets — it has no cookie/token mechanism anywhere, so this project has
  no precedent to copy for session-cookie storage.
- **Workstation environment (verified):** Omarchy (Arch) on Hyprland; terminal is
  **Ghostty 1.3.1**, which supports the Kitty graphics protocol. `mpv` and
  `ffmpeg` are installed; `chafa`/`viu`/`timg` are not. `tmux` is installed but
  was not active when checked.
- **Toolchains (verified):** Go 1.27.2, Python 3.14.7, Node 26.8.1, Bun 1.4.3,
  Deno 2.9.6, sqlite3 3.53.4. **No Rust toolchain is installed** (`cargo`/`rustc`
  absent) — a Rust stack implies a rustup install first.
- **Motivation, in the user's words:** they are in a terminal/multiplexer all day;
  they want a low-distraction plain-text feed; the web client is heavy and the
  browser is conspicuous in a shared office.

## Constraints

- **Tech**: upstream endpoints are undocumented and unstable — the fetch/parse
  layer must be swappable without touching UI code
- **Cost**: zero recurring API spend; the only budget is the machine's own resources
- **Legal**: use of internal endpoints violates X's terms — this stays a
  single-user personal tool with no support and no product behind it. Published
  publicly (source and release binaries) since 2026-10-16; that exposure was
  accepted deliberately, and it changes nothing about who maintains it: nobody does
- **Privacy/environment**: no timeline data on disk; the office setting drives both
  the no-persistence choice and the unbranded visual style
- **Compatibility**: must run in Ghostty on Linux; terminal image capability is
  unverified and needs empirical testing, not documentation-based assertions
- **Performance**: startup and scroll must feel instant in a tmux pane; a
  text-only feed has no excuse for lag
- **Secrets**: the sibling project's config round-trips bytes through a
  panel-editable file and has no secret-handling mechanism at all — the session
  cookie must therefore live outside any generally-editable config surface
- **UI language**: English only — no i18n layer; the interface must read as ordinary tool output
- **Project structure**: vertical MVP — each phase delivers one end-to-end usable capability, starting with the live data path
- **Network**: x.com and every `*.twimg.com` host are unreachable from this host
  without a proxy — measured: DNS resolves (`x.com` → 162.159.140.229) while TCP/TLS
  times out, and `https://example.com/` returns 200 in 0.78 s. The user's proxy is
  Clash Verge Rev (`verge-mihomo`) on **`127.0.0.1:7897`** — a *mixed* port that
  serves both HTTP and SOCKS5, verified working for `https://x.com/` (200 in 0.39 s,
  egress Hong Kong). The port is 7897, not the usual 7890, and the GNOME system-proxy
  setting is configured but disabled (`mode=none`), so nothing inherits it: the client
  must take its proxy from configuration or the standard `*_PROXY` variables, and must
  fail loudly when the proxy is not running

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Use x.com's internal web GraphQL endpoints with session-cookie auth | Only zero-cost path with full read fidelity (timeline, threads, search, bookmarks) | — Pending |
| Read-only v1 | Avoids the write-auth path and the extra ToS exposure of mutating an account; the stated job is reading quietly | — Pending |
| Cookie supplied via config file, env-var override | Simplest working mechanism; browser-cookie extraction adds keyring/decryption complexity for no v1 benefit | — Pending |
| Manual refresh only (explicit key) | Fewest upstream requests, lowest chance of tripping bot defenses, no surprising timeline rewrites | — Pending |
| Timeline never persisted; only bookmarks/likes mirrored locally | Leaves no browsing trace on disk in the office environment | — Pending |
| Borderless, unbranded, low-color UI | Reads as generic tool output rather than a social app | — Pending |
| Media opens externally behind an abstracted renderer interface | Guarantees v1 works everywhere; in-terminal rendering (Ghostty/Kitty protocol) can be enabled once empirically verified | — Pending |
| Stack chosen on this project's merits, treating rss-tm as evidence rather than a constraint | The user asked for the best fit for the function, not continuity with an older repo; prior art still de-risks the TUI/render/test layers | — Pending |
| Session cookie kept out of the general config surface (own 0600 file, env override) | The account cookie is a bearer credential for the user's whole x.com session; it must not be readable or round-tripped by ordinary config editing | — Pending |
| Published publicly, but not a product | The repository and its Release (source + three static binaries) went public on 2026-10-16, so a stranger can pick the program up and run it — the exposure the earlier "not distributed" stance refused, accepted deliberately once its premise (a private repository) was pointed out. Internal-endpoint usage is still against X's terms and this stays a single-user personal tool with no support, no roadmap and no cadence | — Accepted 2026-10-16 |
| The Following timeline is read with `HomeLatestTimeline`, whose operation id comes from an external pinned source | The authenticated web bundle deliberately omits it, and the `GenericTimelineById` + `timelineId` path the bundle implies answers `200` with `Internal server error` from a raw client | — Pending |
| The operation registry is data: 107 operations with provenance, add-only on refresh, pins separate | Ids rotate on someone else's schedule; keeping them in one JSON file makes a repair a one-file change and lets a bundle refresh never delete an externally sourced entry | — Pending |
| The `features` bag is optional and is sent only when the registry declares it | Measured: `HomeLatestTimeline` returned 155 posts with no `features` parameter at all, so the feared `(336)` drift is far smaller than modelled | — Pending |
| Go module fetches run through the local proxy | `proxy.golang.org` is unreachable directly from this host and works through Clash on `127.0.0.1:7897` | — Pending |
| The credential's `%p` formatting boundary is accepted, documented and pinned by exact count | `fmt` resolves `%p` before any method lookup, so no type-level mitigation exists while the value keeps a string representation; no caller uses it | — Pending |
| Bubble Tea v2 parses view content into cells, so a raw escape cannot be embedded in a frame | Discovered while planning the disguise phase: a panic entry that must emit `ESC[3J` and a title change cannot be ordered from inside `View`, so it is one `Write` by a dedicated terminal package, and a new module-wide AST test keeps `os/exec` confined to that package | — Pending |
| Every request answer is matched to the frame that asked for it | A discarded frame's answer was being absorbed by its replacement, which made a healthy pagination look like a stalled cursor — the pane lying about what happened | — Pending |
| A recorded failure is cleared by the next successful answer | The status line outlived the condition it described; recovery must visibly recover | — Pending |
| A covered pane never receives the classifier's own sentence | The disguise's whole value is that the screen names nothing; a covered failure pane now shows the cover's own line, while the plain reader keeps the honest detail byte for byte | — Pending |
| The alias mapping is a pure function over compiled-in word lists, asserted by a no-write test | A persisted alias table would be a list of the accounts the user follows — exactly the trace this project refuses to leave | — Pending |
| Absence claims are asserted with arms, never by golden byte-comparison alone | A golden compare is a tautology: it passes as soon as the frame is re-blessed, so a planted icon or identity must fail an explicit scan | — Pending |
| The pane's rows are produced by one constructor that both the painter and every measurer call | A cover changes how many rows a post occupies; measuring in one grammar and painting in another moved the selected block out of the pane under gitlog — a disguised pane must hide the content, not the reader's place | — Pending |
| Vertical MVP phase structure (end-to-end slice per phase, live data path first) | The dominant risk is upstream access, not UI work — proving one real request path early beats building layers that may not fit the real payloads | — Pending |
| English-only UI copy, no i18n layer | Keeps the text surface plain and unremarkable in a shared office; i18n is cost with no user | — Pending |
| Greenfield build, no shared code with rss-tm | Decided with the user: pick the stack on this project's merits; the sibling repo is evidence and a convention reference only | — Pending |
| Proxy support as a first-class transport concern (config file plus `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY`, HTTP and SOCKS5) | x.com is unreachable here without the local proxy; treating it as later hardening would leave the whole client silently dead on arrival | — Pending |
| A `doctor` subcommand plus a lightweight startup self-check | This upstream's signature failure is an empty HTTP 404 that renders as "no new posts" — silent wrongness, not an error; the tool must distinguish "no network path", "expired cookie", "stale query id" and "genuinely empty" | — Pending |
| Three display states (normal / immersive / panic) instead of one hide key | The real goal is reading the feed continuously in an open office, not only hiding on approach: immersive serves the reading case, panic serves the interruption case | — Pending |
| The immersive cover aliases author identities too | A visible real name or @handle defeats the purpose; deterministic hashing keeps aliases stable across restarts without writing an alias table to disk | — Pending |
| Panic clears terminal scrollback and tmux history, and focus loss triggers it | Swapping only the current screen leaves the timeline one scroll-up away; auto-restoring on focus return is unsafe, because un-covering must be deliberate | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-10-16 — repository made public; fixtures sanitised and the distribution promise amended to match*
