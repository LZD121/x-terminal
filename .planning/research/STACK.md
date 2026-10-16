# Stack Research

**Domain:** Read-only terminal (TUI) client for x.com — single-user personal tool; data source is x.com's own internal web GraphQL API, authenticated by the logged-in session cookie
**Researched:** 2026-10-14
**Confidence:** HIGH for the framework and persistence choices; MEDIUM-HIGH for the x.com internals (two independent maintained implementations agree on the mechanics) and for the image path (empirically probed on this host, then verified against the Ghostty `v1.3.1` source)

## Executive Verdict (feasibility)

**YES — this is feasible, and the hard part is not the TUI.** Three findings drive every recommendation below:

1. **No library, in any language, covers the endpoint set this project needs.** The single most important requirement — the Following timeline (`HomeTimeline`) — is missing from every maintained x.com client (Python `twscrape` 0.20.1, TypeScript `@the-convocation/twitter-scraper` 0.22.3); bookmarks are missing from the TypeScript one; likes are missing from the Python and Go ones. Every library is a *different subset*. The pragmatic answer is therefore a **hand-rolled thin client** that borrows the request mechanics from the maintained projects, not a dependency.
2. **The x.com layer is a two-file fragility, not a research project.** The moving parts are known exactly: a per-request `x-client-transaction-id` header (algorithm published, ~200–400 lines to port), a per-operation `queryId` table scannable out of the live JS bundle, and a feature-flag map that reports its own drift via error code `(336)`. Everything else (bearer token, header names, cursor pagination, response shape) is stable enough to hard-code.
3. **Go + Bubble Tea v2 is the right stack**, on this project's merits: it is the only candidate whose async model (`Update` returns a `tea.Cmd`; the command runs off the UI loop) matches the "everything is a read" shape of the app, it produces a 2.4 MB static binary that starts in **0.9 ms** on this host, and its terminal-width handling is the most modern of the four (Unicode mode 2027 + `displaywidth`). The runner-up is **TypeScript on Bun** — not because its runtime is better but because `@the-convocation/twitter-scraper` already implements the cookie/transaction-ID plumbing — and it still lacks `HomeTimeline` and bookmarks. Python/Textual is the fastest to prototype but pays a blocking single-event-loop model and interpreter startup; Rust/Ratatui is the best UI *library* but has a dead x.com ecosystem and no toolchain installed.

| Area | Recommendation | Confidence |
|------|----------------|-----------|
| TUI runtime | Go 1.27.2 + `charm.land/bubbletea/v2` v2.1.0 (+ `lipgloss/v2`, `bubbles/v2`) | HIGH |
| x.com data layer | Hand-rolled thin client in `internal/xapi` (stdlib `net/http`), mechanics ported from `twscrape` (MIT) | MEDIUM-HIGH |
| Transaction-ID signing | Port the ~300-line generator; reference implementations exist in Python/TS, plus a Go research port | MEDIUM-HIGH |
| Terminal images (later) | Nothing in v1; external-opener interface + `OSC 8` link (verified supported). Later: Kitty graphics with Unicode placeholders + tmux passthrough | MEDIUM-HIGH |
| Session cookie | `0600` file in `$XDG_CONFIG_HOME/x-terminal/`, env-var override; no keyring | HIGH |
| Persistence | `modernc.org/sqlite` v1.60.1 (pure Go, BSD-3-Clause), CGO-free | HIGH |

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.27.2 (installed) | Language + toolchain | Static binary (measured 2,350,833 bytes, `CGO_ENABLED=0`), 0.9 ms process start on this host, goroutine-per-`Cmd` concurrency, and the author already owns a Go+Bubble Tea v2 codebase (`rss-tm`) |
| `charm.land/bubbletea/v2` | **v2.1.0** (v2.0.0 released 2026-02-23; module path moved from `github.com/charmbracelet/bubbletea` to the `charm.land` vanity domain; `go.mod` requires `go 1.26.0`) | TUI runtime (event loop, renderer, input) | The "Cursed Renderer" is an ncurses-derived incremental renderer; v2 enables **Unicode mode 2027** (wide/emoji width correctness) and **synchronized updates mode 2026** by default, ships progressive keyboard enhancement (Ghostty is an explicitly supported terminal) and OSC 52 clipboard. Declarative `View() tea.View` removes the option/command state machine. |
| `charm.land/lipgloss/v2` | v2.0.6 | Styling/spacing/width arithmetic | Pure (no I/O) in v2 — Bubble Tea owns the terminal, so no colour-query races. Same repo family, same version cadence. |
| `charm.land/bubbles/v2` | v2.2.1 | Widgets: `viewport`, `list`, `key`, `help`, `table`, `textinput`, `spinner`, `paginator` | Package set confirmed at repo root (`viewport`, `list`, `key`, `help`, `table`, `textinput`, `textarea`, `paginator`, `progress`, `spinner`, `stopwatch`, `timer`, `tree`, `filepicker`, `cursor`) — enough for timeline/detail/search/profile panes with no third-party widget code. **No image widget exists** (relevant to Deep Dive 2). |
| `net/http` (stdlib) | Go 1.27.2 | All x.com HTTP | A thin GraphQL client is a `http.Client` with a cookie jar; no framework needed. Keeps `internal/xapi` dependency-free and swappable, which the project constraints demand. |
| `modernc.org/sqlite` | **v1.60.1** | Bookmarks/likes mirror | Pure Go, **BSD-3-Clause**, no cgo → the static-binary story survives (`mattn/go-sqlite3` needs a C toolchain and cgo). Used through `database/sql`. |
| `charmbracelet/x/ansi` | v0.11.9 | ANSI/escape-sequence and width helpers | Already a direct dependency of Bubble Tea v2 — use it instead of hand-rolling escapes |
| `github.com/clipperhouse/displaywidth` | v0.11.0 (arrives transitively via Bubble Tea v2) | Grapheme/emoji display width | This is the width engine v2 uses; do not add a second width library |

### The Framework Decision (item 1 of the brief)

Candidates, each verified against its registry/repo today:

| Criterion | **Go + Bubble Tea v2** | Rust + Ratatui | Python + Textual | TS + OpenTUI / Ink |
|---|---|---|---|---|
| Latest release | `charm.land/bubbletea/v2` **v2.1.0** (v2.0.0 2026-02-23) | `ratatui` **0.30.2** (2026-06-19), split into `ratatui-core` / `ratatui-widgets` / `ratatui-crossterm` | `textual` **8.2.8** (2026-06-30) | `@opentui/core` **0.6.4** (2026-10-13; Zig core behind an FFI, needs Bun ≥1.3.14 or Node ≥26.4.0 with `--experimental-ffi`); `ink` **8.0.0** (2026-10-03) |
| Bubble Tea's *model* in other languages | This stack **is** it | None — you own state and event loop | Textual is its own model (DOM/CSS/reactor), not Elm | `@oakoliver/bubbletea` **1.2.0** (2026-10-04, MIT, zero-dependency port with `bubbles`/`gum`/`glamour`/`huh` siblings) — current but a one-maintainer port with ≈45 weekly downloads, i.e. unproven; not Charm-official |
| Async I/O without blocking the UI | **Native fit:** `Update` returns a `tea.Cmd`; Bubble Tea runs it off the render loop and delivers the result as a `Msg`. Blocking code is impossible to write accidentally — this is exactly `rss-tm`'s enforced convention | No runtime at all: you own the loop (crossterm/termion backend + a thread + channel, or `EventStream`). Full control, more scaffolding | Single event loop: a slow handler **blocks the UI**; correct code must wrap every read in `run_worker()` / `@work(exclusive=…)` (documented, easy to forget) | JS event loop; `async/await` is natural in Ink/OpenTUI |
| Virtualized scrolling of long lists | Slice the visible window yourself; `bubbles/viewport` + `list` handle the frame. O(visible) rendering because the renderer only diffs the frame | Same "slice it yourself" model (`List` + `ListState` offset/selected); third-party `tui-widget-list` crate exists if you want scrolling-list behaviour | Widget-per-item DOM: `ListView`/`DataTable` grow with item count; large feeds need `DataTable` + manual paging | Ink re-renders the tree per frame (`maxFps` default 30) unless `incrementalRendering: true`; `<Static>` exists for append-only output. OpenTUI is a real renderer with scroll boxes. |
| Unicode / wide-char / emoji width | **Best of the four:** v2 enables mode 2027 when the terminal supports it and computes width with `displaywidth` + `uniseg`. Ghostty advertises the Kitty keyboard protocol and is listed as a supported terminal | `ratatui` has explicit grapheme-width handling (recent fixes for VS16 clusters, #2686); VS16 edge cases have historically bitten incremental updates | `rich`/Textual own width logic; correct in practice, but each widget has its own measurement path | JS `string-width`-class libraries; correctness is a per-library patchwork |
| Keyset pagination ergonomics | Cursor string → closure inside a `tea.Cmd`; "load older" is a message, not a state machine. Matches the manual-refresh requirement (no background polling) | Cursor handling is yours in both the fetch thread and the widget state | Cursor must round-trip through worker results; `exclusive=True` cancellation semantics help | Async generators (`for await`) in the-convocation are genuinely pleasant; OpenTUI has no such precedent yet |
| Distribution | Single static binary (`CGO_ENABLED=0`), 2.35 MB measured. **No interpreter, no venv** | Single static binary too — but see toolchain cost below | Needs the interpreter + a venv (`uv`/`pip`); system Python 3.14.7 present, so it works, but "start it from a tmux pane" now depends on Python packaging | Bun can emit a single-file executable (`bun build --compile`, bundles the runtime); Node needs `node_modules` or a bundler |
| Startup latency (measured on this host, medians of 15 runs) | **0.9 ms** (static Go binary) | ~1 ms class (native; not measured — no toolchain here) `[INFERENCE]` | 10.4 ms bare interpreter; 17.4 ms with a couple of stdlib imports (+ Textual/Rich import cost) | Bun 3.8 ms; Node 20.2 ms; Deno 21.6 ms (before framework code) |
| Toolchain cost | Already installed | **Not installed**: rustup (`static.rust-lang.org` reachable from this host — verified HTTP 200) + a stable toolchain download. Not a blocker, just a step `[INFERENCE]` | Installed | Installed (Node 26.8.2, Bun 1.4.3, Deno 2.9.6) |
| x.com client ecosystem | Go port exists (`imperatrona/twitter-scraper` v0.0.18) but is **stale**: hard-coded queryIds, no `x-client-transaction-id`, and its own source carries a hotfix note that the classic web bearer returns 404 without that header | **Dead**: `egg-mode` 0.16.1 (2023-02), `twitter-v2` 0.1.8 (2022-10), `twitter-scraper` 0.3.0 (2022-08). Only `x-client-transaction` 0.1.0 (2025-05) is recent, and it is a single-purpose signing crate | Best-covered: `twscrape` 0.20.1 (2026-08-25) is actively maintained, does cookie auth, transaction-IDs, op-ID refresh, and SQLite session storage | Best-covered for a JS host: `@the-convocation/twitter-scraper` 0.22.3 (2026-04-01) does cookie auth, `x-client-transaction-id` (experimental flag), and ships its endpoint table |

**Recommendation: Go 1.27 + Bubble Tea v2** — one stack, decided.

Why it wins on this project's merits, not on continuity:

- **The async model matches the data model.** Every screen is "fetch then render", gated by a manual refresh. Bubble Tea expresses exactly that: the `Update` function stays pure and every read leaves as a `tea.Cmd`. Python's Textual requires remembering `run_worker`; Rust requires designing the concurrency yourself; both are more code between "press R" and "new posts".
- **A text-only feed with no excuse for lag gets the fastest start and the cheapest binary.** 0.9 ms to first frame-class startup, no interpreter, no venv, no `node_modules`. This is a tool that must feel instant inside a tmux pane.
- **Unicode correctness is the default, not a patch.** `x.com` content is emoji- and CJK-heavy; v2's mode 2027 + `displaywidth` is the most defensible width story of the four, and the failure mode (layout drift on wide glyphs) is exactly the kind of thing this project cannot afford to hand-roll.
- **The x.com library gap is not a downside.** Since no library covers `HomeTimeline` + bookmarks + likes in *any* language, "Go has the weakest scraper library" costs nothing: the work is the same everywhere. Go's disadvantages (no maintained fingerprint-spoofing stack by default) are bounded and have named mitigations (`bogdanfinn/tls-client` v1.16.0, utls v1.8.2).

**Runner-up: TypeScript on Bun + OpenTUI (0.6.4) or Ink (8.0.0).**

The runner-up wins under specific conditions:

- **You want the x.com layer to be someone else's problem for as long as possible.** `@the-convocation/twitter-scraper` is the only package that already implements cookie auth, `x-client-transaction-id` generation, Cloudflare-aware `CycleTLS` fetch, and rate-limit strategies. You would still have to add `HomeTimeline`/`HomeLatestTimeline` and `Bookmarks` to its endpoint table (they are absent), but likes, search, tweet detail and profiles come free.
- **You value startup only behind correctness of the *first* fetch:** Bun starts in 3.8 ms and can compile to a single executable, so the "instant pane" goal is still met.
- **It stops winning** the moment you care about (a) not maintaining a JS dependency tree for a personal tool, (b) the width/emoji correctness story, or (c) avoiding a runtime that must be installed to run the compiled artifact's build pipeline. OpenTUI is young (0.6.x, first stable major not yet reached) and Ink's renderer is the least incremental of the four.

**Third: Python + Textual (8.2.8)** — the fastest path to a working prototype and the best x.com library (`twscrape`), but it is *the wrong shape* for a long-lived daily-driver: a blocked event loop unless disciplined, interpreter startup, and a dependency tree that keeps moving (`twscrape`'s `pyproject.toml` requires `httpx`, `beautifulsoup4`, `fake_useragent`, `aiosqlite`, `pyotp`, `py-machineid`, with `curl-cffi` as an optional extra) — including an **async SQLite driver** for a project whose persistence need is "a few thousand rows of bookmarks and likes". `[INFERENCE]` on the "wrong shape" judgement; the dependency list itself is verified.

**Fourth: Rust + Ratatui (0.30.2)** — the best-instrumented UI *library* (modularized crates, explicit width handling, an active org) and the honest answer if you want a compiled binary with zero runtime story and are willing to (a) install rustup, (b) hand-roll the entire x.com layer with no maintained reference client to copy from, and (c) own the event loop and its concurrency. Not worth it here merely for the language.

Confidence: **HIGH** on the framework choice; **MEDIUM** on the relative ranking of the runner-up (it depends on how much you weight the existing TS library).

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/charmbracelet/x/exp/golden` | v0.1.0 | Golden-frame test helper | From day one — the view *is* the spec; `rss-tm` already proved this pattern for a TUI of this shape |
| `github.com/bogdanfinn/tls-client` | v1.16.0 (or `github.com/refraction-networking/utls` v1.8.2) | Browser-like TLS fingerprint transport | **Only if** an empirical probe shows plain Go TLS is challenged (see §1.4). Do not add pre-emptively. |
| `github.com/zalando/go-keyring` | v0.2.8 (MIT) | libsecret/Secret Service access | Only if the user later decides they cannot live with a 0600 file. See Deep Dive 3. |
| `charmbracelet/x/ansi` | v0.11.9 | Escape sequences, `OSC 8` hyperlinks, width helpers | For the media-link escape and any raw terminal query |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `go build` / `go test` (Go 1.27.2) | Build + test | Ship with `CGO_ENABLED=0`; verify with `go build -ldflags '-s -w'` |
| `net/http/httptest` (stdlib) | Fake the x.com GraphQL server in tests | Fixtures are *recorded JSON bodies*, not live calls — the fetch layer must be testable with zero network |
| Golden-frame tests | Lock the rendered frame byte-for-byte | Combined with `tea.WithWindowSize(w,h)` (new in v2) so tests need no TTY mock |
| `sqlite3` 3.53.4 (installed) | Inspect the bookmarks/likes mirror | Read-only debugging of the one file this project persists |

## Deep Dive 1 — Talking to x.com's internal web GraphQL (the critical path)

### 1.1 What a request actually looks like

Verified against two independently maintained implementations — `twscrape` 0.20.1 (`twscrape/account.py::make_client`, `twscrape/queue_client.py::Ctx.req`, MIT) and `@the-convocation/twitter-scraper` 0.22.3 (`src/auth-user.ts`), read 2026-10-14:

```
GET https://x.com/i/api/graphql/<queryId>/<OperationName>?variables=…&features=…&fieldToggles=…
authorization: Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA
x-csrf-token: <value of the ct0 cookie>
x-client-transaction-id: <computed per request, per path>
x-twitter-active-user: yes
x-twitter-client-language: en
x-twitter-auth-type: OAuth2Session        # the-convocation sets this; twscrape omits it
content-type: application/json
user-agent: <real browser UA string>
cookie: auth_token=<…>; ct0=<…>
```

- The bearer string in this project's brief is byte-identical to `twscrape`'s `TOKEN` constant — it is still the value the web app uses. (The `%3D` is the URL-encoded `=`.)
- **Both cookies are required.** `twscrape` gates on `has_required_cookies()` = `auth_token` and `ct0` present; the-convocation records in-source that `ct0` alone yields `401` on every API call.
- `x-csrf-token` is simply the `ct0` cookie value (both implementations).
- Guest/anonymous mode is a *different* path: `x-guest-token: <gt>` from `POST https://x.com/1.1/guest/activate.json` (or the `gt` cookie the login page's inline script sets), cached ~3h. **Not needed** once a real `auth_token` session exists — `twscrape`'s cookie flow never calls it.

### 1.2 The two things that must be fetched from the live site

**(a) The `queryId` table — the #1 thing that breaks.** X assigns each GraphQL operation a persisted-query hash; IDs rotate whenever the web app ships a new build, and a stale ID returns **404**. Evidence of how volatile this is: the same operation has three different IDs across three currently-published clients —

| Operation | `twscrape` 0.20.1 | `imperatrona/twitter-scraper` (Go, master) | `@the-convocation` 0.22.3 |
|---|---|---|---|
| `TweetDetail` | `XMOz5h24KAZ86qKffKTLdQ` | `VWFGPVAGkZMGRKGe3GFFnA` | `YCNdW_ZytXfV9YR3cJK9kw` |
| `SearchTimeline` | `hyPfJYJ_XAtDYoslQc-Rgg` | `nK1dw4oV3k4w5TdtcAdSww` | `ML-n2SfAxx5S_9QMqNejbg` |
| `UserByScreenName` | `Gb-d6r0vxPOADdG62OEBpQ` | — | `AWbeRIdkLtqTRN7yL_H8yw` |
| `Bookmarks` | `iblrFnKr6PZUR-dWpfXG6g` | `-IyJFt9_jS_9d_vS3NN-fA` | *absent* |
| `HomeTimeline` / `HomeLatestTimeline` | *absent* | `1u0Wlkw6Ru1NwBUD-pDiww` / `9EwYy8pLBOSFlEoSP2STiQ` | *absent* |
| `Likes` (a.k.a. `UserLikedTweets`) | *absent* | *absent* | `Pcw-j9lrSeDMmkgnIejJiQ` |

**Discovery recipe** (from a maintained tool's own documentation, `obnoxiousmods/DeleteTweets/docs/refreshing-query-ids.md`, and `twscrape/scripts/update-gql-ops.py`, the script that regenerates `twscrape/api.py`):

1. `GET` an ordinary logged-in page (`https://x.com/home` or a profile) with the cookie headers.
2. Extract script URLs from the HTML. The build has changed shape over time and the parser must handle both: the current Vite build links `https://abs.twimg.com/x-web/*.js` directly; the legacy webpack build embeds a hash map *and* a chunk-name map in the HTML from which `https://abs.twimg.com/responsive-web/client-web/{name}.{hash}a.js` is reconstructed. (`twscrape/xclid.py::get_scripts_list` documents both, including the regexes and the 7-vs-16-hex-digit hash-width change on 2026-08-24.)
3. Regex the bundles for object literals pairing the two fields — the safe pattern matches `{queryId:"<id>",operationName:"<Op>"}` **as a unit**, because a loose `queryId`-only grep misaligns IDs with operations.
4. Cache the table locally; refresh when any call 404s. This is a build-time/dev-time tool, not something to run on every launch (it costs several HTTP GETs plus bundle downloads).

**(b) The feature-flag map.** Every GraphQL call carries a `features` object that must be *near* the live web build's set. X tells you when you are wrong: error `(336) The following features cannot be null: <names>` names the exact flags to flip to `true`; `twscrape` self-heals by patching those into the request and retrying, and treats "still failing after heal" as a code-update error. A handful of flags must stay `false`. Ship the flag map as data in `internal/xapi`, not as constants threaded through code.

### 1.3 `x-client-transaction-id` — the header that turned a 404 into a research project

What is verified:

- **It is required.** The Go client's own source carries a hotfix note that a web bearer now returns 404 without this header, and `twscrape` regenerates the header and retries whenever a request 404s (issue #248).
- **The algorithm is public and implemented multiple times**: the canonical write-up is a 3-part reverse-engineering series by `iSarabjitDhiman` (`XClientTransaction`, MIT, Python) with a TypeScript sibling (`Lqm1/x-client-transaction-id`, npm **0.3.2**), a Rust port (`langkor/x-client-transaction` 0.1.0, crates.io 2025-05-02), and a Go research port (`ichioda/x-client-transaction-id` — source with a `go.mod` but **not** published/indexed as a Go module, so treat it as reading material, not a dependency).
- **The shape of it** (`twscrape/xclid.py`, 391 lines total including HTML/HTTP helpers, ported from the Python reference under MIT):
  1. Read `<meta name="twitter-site-verification" content="…">` from a logged-in page and base64-decode it to the *verification key bytes*.
  2. Locate the signing script and scrape the *animation indices* from it: legacy builds link `ondemand.s.*.js`; the current build hides `sign.o-*.js` behind a dynamic import inside a bundle chunk, so the loader scans chunks concurrently. The indices come from a regex over the script plus the `d` attributes of SVG paths inside `svg[id^='loading-x-anim'] g:first-child path:nth-child(2)`.
  3. Per request: `ts = now_ms − 1682924400 s`; payload = `METHOD!path!ts` + keyword `obfiowerehiring` + animation key → SHA-256; then `verify_key_bytes ‖ ts_le32 ‖ sha256[:16] ‖ 0x03`, XOR-obfuscated with a random byte prefix and base64-encoded without padding.
- **Practical consequence:** it must be regenerated whenever the key material goes stale; `twscrape` caches a generator per account and rebuilds it on a 404. The generator depends on a *logged-in* page — the anonymous build does not contain the needed material.
- **Cost estimate:** 300–400 lines of Go for the generator (key bytes + index scraping + the cubic-bezier/hash assembly), fully unit-testable against the reference implementations' vectors. `[INFERENCE]` — based on the reference sizes, not measured.

### 1.4 Cloudflare, TLS fingerprints and what "blocked" looks like

- `twscrape` classifies a `≥400` response with `content-type: text/html` as **"Blocked by Cloudflare"** when a `cf-ray` header is present, and separately fails with a precise parse error when the page served is a Cloudflare interstitial (`/cdn-cgi/challenge-platform/` script, no app bundles) — issue #330.
- Authentication endpoints are the ones documented as being behind "Cloudflare's advanced bot detection" (`403`) — the-convocation ships an optional `CycleTLS` fetch entrypoint that impersonates Chrome's TLS fingerprint for that reason; `twscrape` ships an optional `curl_cffi` backend (`TWS_HTTP_BACKEND=curl`) for the same reason. **In both projects the fingerprint-spoofing transport is optional, not default.**
- `[INFERENCE]` For a read-only, low-volume, authenticated client that sends a realistic browser `User-Agent`, plain Go `net/http` will *probably* not be challenged — but this is exactly the kind of claim this project must settle empirically, not from docs (the same reason the media probe was run). If it is challenged, the mitigation is a transport swap, and Go has two: `bogdanfinn/tls-client` v1.16.0 or `refraction-networking/utls` v1.8.2 — behind the same `http.RoundTripper` seam the client already needs.

### 1.5 Rate limits and 429 behaviour

Verified from `twscrape/queue_client.py::_check_rep` (live code, 2026) and `imperatrona`'s README:

| Signal | Meaning | Correct handling |
|--------|---------|------------------|
| `x-rate-limit-remaining: 0` + `x-rate-limit-reset` | Endpoint-level limit hit | Stop; wait until reset. The-convocation documents waits **"up to 13 minutes"** |
| Error `(88) Rate limit exceeded` **with** `remaining > 0` | Ban heuristic, not a normal limit | Mark the session bad; do not retry |
| Error `(326) Authorization: Denied by access control` / `(32) Could not authenticate you` / bare `403` | Session expired or account restricted | Stop and tell the human; refresh cookies |
| Error `(336) …features cannot be null` | Client is stale | Self-heal the flag map (once) or update code |
| Error `(-1) LoadShed` | X is overloaded | Retry with exponential backoff, **without** penalising the account |
| `404` | Stale `queryId` **or** stale `x-client-transaction-id` | Regenerate the transaction-ID first (cheap), then refresh the op table |
| Baseline shape | — | ≤1 request / 1.5 s per account; most endpoints ≈150 requests / 15 min (Go client README, Feb-2024-era figures; treat as order-of-magnitude) |

For this project the design consequence is a *single-user* policy: one in-flight request at a time, a hard floor between requests, and manual refresh — which is what the requirements already demand.

### 1.6 Library survey — and why the answer is a hand-rolled thin client

| Language | Package | Latest | Status | Cookie auth? | Pagination cursors? | Covers *this* project's endpoints? |
|---|---|---|---|---|---|---|
| Python | `twscrape` | **0.20.1** (2026-08-25), MIT | Actively maintained; op-ID refresh script, transaction-ID generator, SQLite account store, curl_cffi backend | Yes (`auth_token`+`ct0`; readme calls cookie setup "the most stable setup") | Yes (cursor-based `_gql_items`, stall detection) | **Partial**: bookmarks + tweet detail + search + user tweets. **No `HomeTimeline`, no likes** |
| Python | `twitter-api-client` | 0.10.22 (2024-04-21) | Effectively unmaintained | Some | — | No — skip |
| Python | `twikit` | 2.3.3 (2025-02-07) | ~20 months stale | Password login, not cookie-first | Yes | No — skip |
| Python | `snscrape` | 0.7.0.20230622 (2023-06) | Dead | No | — | No |
| TS/JS | `@the-convocation/twitter-scraper` | **0.22.3** (2026-04-01), MIT | Actively maintained; ships `cast`/`CycleTLS`/`xpff` machinery; docs warn fixes take days when X changes | Yes (`setCookies`, readme documents cookie auth as the fix for login blocks) | Yes (AsyncGenerators; 13-minute rate-limit strategy) | **Partial**: likes + tweet detail + search + profiles. **No home timeline, no bookmarks.** `x-client-transaction-id` is behind `experimental.xClientTransactionId` |
| TS/JS | `twitter-api-v2` | 1.29.1 (2026-08-04) | Maintained but **official API only** | N/A | N/A | No — pay-per-read, out of scope |
| Go | `imperatrona/twitter-scraper` | v0.0.18 | Semi-maintained; **hard-coded queryIds**, no transaction-ID, source notes the 404 hotfix | Yes (`SetCookies`, `SetAuthToken`) | Yes (channel iterators) | **Partial**: `HomeTimeline`, `HomeLatestTimeline`, `Bookmarks`, `TweetDetail`, `SearchTimeline` — **no likes**, and it will 404 as shipped |
| Go | `michimani/gotwi` v0.18.5 / `g8rswimmer/go-twitter/v2` v2.1.5 | — | Maintained | N/A — official API v2 wrappers | N/A | No |
| Rust | `egg-mode` 0.16.1, `twitter-v2` 0.1.8, `twitter-scraper` 0.3.0 | 2022–2023 | **Dead** | No | — | No |
| Rust | `x-client-transaction` 0.1.0 | 2025-05-02 | Single-purpose crate, low activity | N/A | N/A | Signing only |

**Recommendation: hand-roll `internal/xapi` on stdlib `net/http`, and port the mechanics from `twscrape` (MIT). Confidence: MEDIUM-HIGH.**

Justification, in order:

1. **No dependency covers the requirement set.** The union of features only exists across three libraries, in three languages. Adding one buys you a subset *and* an upgrade treadmill whose API changes are driven by strangers.
2. **The project's own constraints demand a thin, swappable layer.** "The fetch/parse layer must be swappable without touching UI code" is a stated constraint; a library that owns the whole request pipeline works against it.
3. **The mechanics are small and already written down.** Headers: a few lines. Transaction-ID: one self-contained package. Op-ID table: one dev script. Feature flags: a data file. Response parsing: `encoding/json` into a *thin* struct, ignoring everything you do not render.
4. **MIT references are legitimate to port from**, with attribution: `twscrape` (op-ID script, transaction-ID implementation) and the original algorithm write-ups.

What is genuinely stable and should be hard-coded: the GraphQL base URL `https://x.com/i/api/graphql`, the bearer, the header names, the `variables`/`features`/`fieldToggles` query-parameter shape, and the timeline JSON skeleton (`instructions` → `entries` → `content.itemContent.tweet_results.result`, plus `cursorType`/`value` for keyset pagination).

What must be treated as data and refreshed: **queryIds** (breaks first, silently, as 404s), the **feature-flag map** (breaks second, loudly, as `(336)`), the **verification key + animation indices** used for the transaction-ID (breaks third, as 404s on *every* endpoint at once), and the **script-URL layout** in the page HTML (breaks the refresh tool itself).

## Deep Dive 2 — Terminal image rendering (the later upgrade)

Two kinds of evidence, deliberately kept apart: **this repo's own probe** (`.planning/research/TERMINAL-MEDIA-PROBE.md`, 2026-10-14, Ghostty 1.3.1-arch2 + tmux 3.7c — a real payload emitted, screenshotted with `grim`, pixels inspected) and **the upstream capability matrix** (Ghostty docs + the `v1.3.1` source tree, checked 2026-10-14).

### 2.1 What Ghostty 1.3.1 actually implements (verified)

| Capability | Verdict | Evidence |
|---|---|---|
| Kitty graphics protocol | **YES** — documented feature, present since at least tag `v1.0.0` | `https://ghostty.org/docs/features` ("Kitty graphics protocol: Ghostty supports…"); `src/terminal/kitty/graphics.zig` exists at tag `v1.0.0` |
| Config toggle for graphics | **None exists.** No `graphics=` / `kitty-graphics=` key in the option reference; the only image knob is `image-storage-limit` (default **320 MB**, "If this is set to zero, then all image protocols will be disabled") | `https://ghostty.org/docs/config/reference`; `src/config/Config.zig` at `v1.3.1` |
| `a=p` put/display, `z=` z-index, `U=1` **Unicode placeholders** | **YES** (z-index is honoured across three render layers; placeholder codepoint `U+10EEEE` with a full virtual-placement iterator) | `src/terminal/kitty/graphics_command.zig`, `graphics_unicode.zig`, `src/renderer/image.zig` at `v1.3.1` |
| Transmission media `t=d/f/t/s` | All four implemented (`t=t` requires the path to sit in a temp dir and to contain `tty-graphics-protocol`) | `src/terminal/kitty/graphics_image.zig` at `v1.3.1` |
| **Animation** (`a=f`, `a=a`, `a=c`) | **NO in 1.3.1** — parsed, but execution returns `ERROR: unimplemented action`; implemented on `main` after 1.3.1 | `src/terminal/kitty/graphics_exec.zig` at `v1.3.1` vs `main` |
| **Sixel** | **NO.** Zero `sixel` matches in `src/`+`include/`; a collaborator answers the official Q&A with "No and its not planned" | `https://github.com/ghostty-org/ghostty/discussions/5832`; tarball grep of tag `v1.3.1` |
| **iTerm2 inline images (`OSC 1337 File=`)** | **NO.** Only `Copy` and `CurrentDir` are implemented; `File`/`FilePart`/`MultipartFile` fall into the `unimplemented OSC 1337` arm | `src/terminal/osc/parsers/iterm2.zig` at `v1.3.1`; corroborated by issue #13011 |
| `OSC 8` hyperlinks | **YES** — in the VT reference | `https://ghostty.org/docs/vt/reference` |
| Detection signals | `TERM=xterm-ghostty`, `TERM_PROGRAM=ghostty`, `TERM_PROGRAM_VERSION`, `GHOSTTY_RESOURCES_DIR` | `src/config/Config.zig`, `src/termio/Exec.zig` at `v1.3.1` |

The probe independently confirms the detection signal: **Ghostty does not export `KITTY_WINDOW_ID`**, so Kitty-style environment detection is wrong here — use `TERM_PROGRAM`/`TERM`.

### 2.2 The tmux story, and why the probe's hazard has a known fix

The probe found: inside tmux 3.7c, a bare Kitty payload does nothing even with `allow-passthrough on`; the DCS-wrapped form (`ESC Ptmux;` + doubled ESCs + `ESC \`) *does* render, but tmux does not model image cells, so text printed afterwards draws **across** the image, and scrolling smears it.

Upstream's answer (Ghostty collaborator, discussion #7224) names the fix the probe was missing: to make images work under tmux you must (1) use **Unicode placeholders** — so the image occupies real grid cells tmux knows about — and (2) **wrap the sequence in tmux's passthrough** (`set -g allow-passthrough on`, an option added in **tmux 3.3**; this host runs 3.7c). Same thread: under Ghostty you must force the format (`chafa -f kitty`) or tools silently fall back to ASCII.

`[INFERENCE]` Combined reading: the probe's "reserve cells + force repaint" mitigation is a workaround for *not* using placeholders. The supported path is placeholder mode; the workaround is only for a non-placeholder put. Either way the probe's conclusion stands: **images must not live inside a scrolling list.**

### 2.3 Libraries that can emit it (if the seam ever moves)

| Language | Option | Latest | Protocols | Notes |
|---|---|---|---|---|
| Rust | **`ratatui-image`** | **11.1.0** (2026-09-17; `12.0.0-rc.2` 2026-10-05), MIT | Kitty + Sixel + iTerm2 + halfblocks, with detection and font-pixel-size query; its screenshot test suite runs against **ghostty** among others | The most mature option in any language — but requires a Rust toolchain, which is a non-starter here |
| Go | **`blacktop/go-termimg`** | **v0.1.26** (2026-02-23), MIT | Kitty (virtual images, z-index), Sixel, iTerm2, halfblocks, **tmux passthrough**, and a Bubble Tea `ImageWidget` | The most complete Go option; its own README warns the "Kitty Unicode placeholder/relative placement features are currently under construction and not recommended for production use" — i.e. exactly the tmux-relevant path is the weak one |
| Go | **`dolmen-go/kittyimg`** | **v1.0.1-rc1** (2026-08-22); v1.0.0 stable 2026-03-19 | Kitty only; `Fprint`/`Fprintln`/`Transcode` + an `icat` CLI; README claims production-ready | The narrow, honest dependency if you want *only* Kitty, aimed at the `file:`/placeholder path that survives tmux |
| Go | `BourgeoisBear/rasterm` v1.1.2, `mattn/go-sixel` v0.0.14, `eliukblau/pixterm` v1.3.3, `qeesung/image2ascii` v1.0.1 | 2026 / 2026 / 2026 / **2018** | Encoders only; `pixterm` is ANSI halfblocks only; `image2ascii` is dead | Not worth adopting |
| TS | **`@opentui/core` 0.6.4** has a built-in `ImageRenderable` (Kitty / Sixel / block fallback, per-renderer transports, tmux passthrough when a protocol is forced) | 2026-10-13 | Kitty, Sixel, blocks, tmux | Only relevant if you take the TypeScript stack |
| TS | **`terminal-image` 5.0.1** | 2026-07-09, MIT | Kitty (raw APC chunks) + iTerm2 via `term-img` + ANSI fallback; detection via `supports-terminal-graphics` | Node-host option |
| Python | **`textual-image` 0.14.1** | 2026-09-25 | TGP (Kitty) + Sixel, Unicode fallback; needs Pillow | Note: **Textual core has no image widget** (grep of `textual/widgets/__init__.py` finds none) |
| External CLI | **`chafa` 1.18.2-2** (Arch `extra`, LGPL-3.0-or-later; upstream 1.18.3) with `-f kitty`; `viu 1.6.1-2` (`extra`, MIT); `timg 1.6.3` (**AUR only**) | — | Kitty/iTerm/sixel/symbols | Language-agnostic "print bytes to stdout"; nothing here is installed today |

### 2.4 Recommendation (MEDIUM-HIGH confidence; the shape of it is HIGH)

1. **v1 ships zero image code.** The `media.Opener` seam is the deliverable: `Open(ctx, MediaRef) error` with one implementation that `exec.Command`s `mpv` (video/animated) or `xdg-open` (images). The probe proves this is fully functional on this host today, and it is the only path that cannot break the layout.
2. **Render media references as `OSC 8` hyperlinks** (verified supported by Ghostty 1.3.1) so the text itself is clickable — this is the fallback that works everywhere, including inside tmux.
3. **When the upgrade happens, prefer the shell-out or the narrow library over a full framework integration.** Concretely: `exec chafa -f kitty --size <WxH> <file>` and write its stdout, or link `dolmen-go/kittyimg` and own the placeholder layout yourself. This keeps the payload out of your process's terminal-state assumptions and keeps the "bytes in, bytes out" boundary testable.
4. **Non-negotiable requirements for any future renderer:** detect via `TERM_PROGRAM`/`TERM` (never `KITTY_WINDOW_ID`); use **Unicode placeholders** and wrap in tmux passthrough when `$TMUX` is set; request an explicit placement size (otherwise Ghostty scales and interpolates); confine images to a dedicated, non-scrolling pane that repaints; do not attempt animation on 1.3.1 (`a=f/a=a/a=c` are unimplemented there).
5. **Do not build sixel or iTerm2 support.** Both are absent from Ghostty 1.3.1 by verified source inspection, so a fallback chain would be dead code on the only machine that runs this.

## Deep Dive 3 — Session-cookie storage

The credential is a bearer pair: `auth_token` + `ct0` are sufficient to act as the account for as long as X honours the session. `auth_token` is *not* short-lived (X rotated it roughly yearly in older behaviour; `[UNVERIFIED]` for 2026), so the storage decision is a real one, not a cache decision.

Host facts established for this decision (probed 2026-10-14):

- `$XDG_CONFIG_HOME` is **unset** → the correct base is `$HOME/.config`; Go's `os.UserConfigDir()` returns `/home/user/.config` here, and per its own documentation it uses `$XDG_CONFIG_HOME` when non-empty, else `$HOME/.config`, and **errors if `$XDG_CONFIG_HOME` is relative** (a useful anti-footgun).
- A Secret Service provider **is** present and reachable: `org.freedesktop.secrets` is owned by `gnome-keyring-daemon` (PID 1368) on the user D-Bus, `secret-tool` is installed and exits cleanly, and `kwalletd6` is present as an activatable portal backend.
- The installed browser is **Chromium** (`~/.config/chromium/Default/Cookies` exists, 61 KB); **Firefox is not installed**, so the unencrypted `cookies.sqlite` route is not even available here.

| Option | Concrete implementation | Trade-off it accepts | Verdict |
|---|---|---|---|
| **0600 file under the XDG config dir** | `os.UserConfigDir()` + `/x-terminal/cookies.json`, dir `0700`, file `0600`, contents `auth_token=…` / `ct0=…`; `X_TERMINAL_COOKIES` env var overrides the file entirely. Write it atomically (temp file → `fsync` → `rename`) and `chmod` explicitly — note Go's `os.WriteFile` applies the *requested* perm masked by umask and will not tighten an **already-existing** file | Plaintext at rest for anyone who is already the same UID (or root). Anyone who can read the file owns the session. | **RECOMMENDED** |
| Env var only | `X_TERMINAL_COOKIES` | Nothing on disk, but the secret lands in the shell history/process environment (visible in `/proc/<pid>/environ` to the same UID) and must be re-supplied on every launch | Recommended as an **override**, not as the only path |
| OS keyring (libsecret/Secret Service) | `github.com/zalando/go-keyring` **v0.2.8** (MIT, 2026-03-23; pure Go — deps are only `godbus/dbus/v5` + `wincred`, no cgo) or the dormant `99designs/keyring` v1.2.2 (MIT, 2022); Python `keyring` **25.7.0** + `SecretStorage` 3.5.0 (BSD-3); Node `@napi-rs/keyring` **2.1.1** (MIT, 2026-10-10) — **not** `keytar`, which is **archived** (last publish 2022-02-17); Rust `keyring` 4.2.0 | Depends on a Secret Service daemon being installed **and unlocked** at process start. On Arch, **Hyprland itself ships none**: `gnome-keyring` (51.1-1, `provides: org.freedesktop.secrets`) has to be installed and wired up separately — on this host it *is* running on the user bus, but that is a property of this machine, not of the desktop. Adds a daemon dependency to a tool that otherwise needs none; Node's `@napi-rs/keyring` additionally has a silent keyutils fallback that loses the secret on reboot | **Deferred** — the runner-up, see below |
| Firefox-cookie import | `~/.mozilla/firefox/<profile>/cookies.sqlite`, table `moz_cookies`: the `value` column is **plaintext** (no OS encryption). Readers open it read-only (`?mode=ro&nolock=1`) or copy it **together with its `-wal`** file | The easiest browser-import path — and irrelevant on this machine, where **Firefox is not installed** | Not applicable here |
| Chromium-cookie import | Linux stores cookies in `<profile>/Default/Cookies` (not `Login Data`; `Local State` is the Windows key file). AES-128 key = PBKDF2-HMAC-SHA1(passphrase, `saltysalt`, **1 iteration**, 16 bytes), where the passphrase is the Secret Service item `chrome_libsecret_os_crypt_password_v2` (`application=chrome`) or the literal `peanuts` for the older `v10` scheme; values are version-prefixed `v10`/`v11`, AES-128-CBC with a 16-space IV. Maintained references recompute both hard-coded keys and match byte-for-byte | Reads a *different* application's private store, breaks whenever Chromium changes its crypto (parts have moved to AES-GCM), and is a large amount of code for a one-line paste | **Do not build** |

**Why the boring option wins here:** the project's threat model is "a shared office, a personal machine, one user". A 0600 file does not defend against a same-UID attacker, and neither would the keyring (the keyring is unlocked by that same session). What the file buys is simplicity, testability, zero daemons, and a straightforward `env > file` precedence. What it costs is honest: the cookie is readable by anything running as this user.

**The condition under which the runner-up (keyring) wins:** the user wants nothing recoverable from a disk image or a backup — e.g. if the machine is backed up to a location they do not control, or if they routinely let other people use the account session. In that case add keyring support *as a second backend behind the same `credentials.Store` interface*, with the file as fallback and the same env override; do not make it the only path.

**What to redact (applies to logs, errors, panics, debug dumps, and test fixtures):**

| Never emitted | Why |
|---|---|
| `auth_token` and `ct0` values, and the whole `Cookie` header | Either alone is enough to act as the account; `ct0` alone is enough to forge `x-csrf-token` |
| `authorization` (the bearer) | It is stable and public, but it identifies the client as a scraped session; harmless to leak, still refuse to |
| `x-client-transaction-id` | Per-request signed value; logging it invites replay analysis and makes logs correlated with sessions |
| Full URLs with `variables=` query strings | Response/request URLs can embed IDs and, in some operations, search terms the user typed |
| Raw `*http.Request` values via `%+v`, `httputil.DumpRequest`/`DumpRequestOut`, or `Request.Write` | All three print headers **verbatim**. Go's internal `reqWriteExcludeHeader` set excludes only `Host`, `User-Agent`, `Content-Length`, `Transfer-Encoding` and `Trailer` — so `Authorization` and `Cookie` are written; this is the single most common leak in Go HTTP code |

Design rules that make those failures structurally hard: (1) the cookie pair is a type whose `String()`/`Format`/`MarshalJSON` emit `auth_token=***; ct0=***`; (2) the HTTP layer logs a *request descriptor* (method, operation name, cursor presence, status, `x-rate-limit-remaining`) instead of the request; (3) error types wrap `*url.Error` and strip the query string before the URL reaches a log line; (4) a test asserts that no logger output contains the fixture cookie value. Note the Go-specific nuance verified in `net/http`'s own documentation: the `Client` strips `Authorization`/`WWW-Authenticate`/`Cookie` when redirecting to a domain that is not an exact or subdomain match — but it **retains them for a same-host different-scheme or subdomain redirect**, and the docs explicitly warn its redirect behaviour predates the Fetch standard and is permissive. Do not rely on redirects for hygiene. `twscrape`'s debug path (`queue_client.py::dump_rep`, which writes request and response headers into `/tmp/twscrape-*/`) is the exact anti-pattern to avoid in an office machine.

## Deep Dive 4 — Local persistence (bookmarks + likes only)

Requirement: mirror two lists; never write the timeline; keep the file inspectable with the installed `sqlite3` 3.53.4.

| Language | Driver | Version | License | cgo / native build | Static binary? | Verdict for this project |
|---|---|---|---|---|---|---|
| **Go (recommended)** | `modernc.org/sqlite` | **v1.60.1** | **BSD-3-Clause** | None — a transpiled SQLite, pure Go; driver name `sqlite` for `database/sql` | Yes (`CGO_ENABLED=0`) | **Use this.** It keeps the single-static-binary property intact; it does carry a CPU overhead versus the C library `[INFERENCE]`, which is irrelevant at a few thousand rows |
| Go (alternative) | `github.com/mattn/go-sqlite3` | latest (MIT, verified on pkg.go.dev) | MIT | **cgo** — requires a C compiler and a libc; breaks `CGO_ENABLED=0` | No (dynamic linking) | Only if you need a SQLite feature or extension the transpile lacks |
| Go (alternative) | `github.com/ncruces/go-sqlite3` | **v0.35.6** | MIT | WASM build of SQLite, pure Go; registers as driver `sqlite3` | Yes | Credible third option; pick it if `modernc.org/sqlite` shows a bug |
| Rust | `rusqlite` | **0.40.2** | MIT | Bundles SQLite via feature flags | Yes | The standard for a non-async Rust app (not our path) |
| Python | stdlib `sqlite3` | links **SQLite 3.53.4** on this host (3.14.7) | PSF | None | n/a | Zero-dependency, but inherits Python's interpreter cost. Note the `autocommit` attribute added in 3.12 **still defaults to `LEGACY_TRANSACTION_CONTROL`** in 3.14 (verified locally: `-1`) — commit explicitly |
| Node / Bun | `node:sqlite` / `bun:sqlite` | Both **work out of the box** — verified live on Node 26.8.2 (`DatabaseSync`, `StatementSync`, `Session`, `backup`) and Bun 1.4.3 | MIT (Node) / MIT (Bun) | None | n/a | Good options *in those languages* (no `better-sqlite3` build needed) — but `node:sqlite` is documented as **Stability 1.2 – Release candidate** (added v22.5.0; RC since v25.7.0), so if you want a stable API guarantee use `better-sqlite3` **13.0.3** (MIT) |

**Schema shape this implies** (small, boring, and it must be *derivable from* the service rather than authoritative): one `posts` table for shared post fields, one `collections` table keyed `(collection, post_id)` with `collection ∈ {bookmark, like}` plus the position/cursor at the time of capture — so a post that appears in both lists is stored once. Store the raw JSON body *per post* only if you need fields you do not render; `[INFERENCE]` for a mirror of two lists, a normalized subset plus the original JSON blob in one column is the pragmatic middle. Write in a transaction per page; use `PRAGMA journal_mode=WAL`. The timeline must never touch this file, and the simplest enforceable version of that rule is that the timeline package never imports the store package — an import-boundary test, mirroring the sibling repo's convention.

## Alternatives Considered

| Recommended | Alternative | When to Use the Alternative |
|-------------|-------------|-----------------------------|
| Go 1.27 + Bubble Tea v2 | **TypeScript on Bun + OpenTUI 0.6.4** | When you would rather reuse `@the-convocation/twitter-scraper`'s cookie/transaction-ID/Cloudflare machinery than port it, and you are willing to hand-write the missing `HomeTimeline`/`Bookmarks` endpoints and accept a younger UI runtime (0.6.x) |
| Go 1.27 + Bubble Tea v2 | **Rust + Ratatui 0.30.2** | When a compiled binary with a hand-owned event loop is the point, and you accept rustup + writing the entire x.com client with no maintained reference to copy from (the Rust ecosystem for this is dead: 2022–2023) |
| Go 1.27 + Bubble Tea v2 | **Python 3.14 + Textual 8.2.8 + twscrape 0.20.1** | When speed of prototyping dominates and the tool will be re-run from a shell you control; add `run_worker`/`@work` discipline for every read, and accept interpreter startup |
| Hand-rolled `internal/xapi` | **`twscrape` as a subprocess** (CLI + JSON) | A legitimate, low-effort way to prove the concept end-to-end in an afternoon: `twscrape search "…" --limit N` parses to SNScrape-style models. Rejected as the final architecture because it cannot serve `HomeTimeline`, drags a Python runtime into a Go binary, and puts parsing outside the swappable seam |
| `0600` file + env override | **OS keyring (`zalando/go-keyring` v0.2.8)** | When "no recoverable secret at rest" outranks "no daemon dependency" (see Deep Dive 3) |
| `modernc.org/sqlite` v1.60.1 | **`mattn/go-sqlite3` (cgo)** | Only if a required SQLite feature or extension is missing from the transpiled build — and then accept a cgo-dependent binary |
| Kitty graphics only (later) | **External renderer binaries** (`chafa`, `timg`, `viu`) | If you later want a *fallback* for a non-Kitty terminal on some other machine. On this host it is pure downside: none are installed, and `mpv`/`ffmpeg` already cover the actual need |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Official X API SDKs in any language (`twitter-api-v2` 1.29.1, `gotwi` v0.18.5, `g8rswimmer/go-twitter/v2`, `tweepy`) | Pay-per-read (~$0.005/post) — the cost model that made this project necessary; also a different auth path than the session cookie | The internal web GraphQL endpoints, as specified |
| `imperatrona/twitter-scraper` (Go) as a *dependency* | Its queryIds are hard-coded in source (they will 404), it sends no `x-client-transaction-id`, and its own `auth.go` carries a hotfix note that the classic web bearer needs that header. It is also 20+ months behind the current web build | Read it for the header set and the `HomeTimeline`/`HomeLatestTimeline` variable shapes; implement your own client |
| `twitter-api-client` (Python, 0.10.22, 2024-04), `snscrape` (2023-06), `twikit` (2.3.3, 2025-02), `egg-mode`, `twitter-v2`, `MegaHotel/twitter-scraper` | Unmaintained, and in this domain unmaintained means broken: the request shape changes on X's schedule, not yours | A hand-rolled client whose breakage you can fix in minutes |
| `KITTY_WINDOW_ID` for graphics detection | **Empirically false on this host**: Ghostty 1.3.1 does not export it (probe A; `TERM`/`TERM_PROGRAM` are what Ghostty actually sets, per `src/termio/Exec.zig` at `v1.3.1`) | `TERM_PROGRAM=ghostty` / `TERM=xterm-ghostty` |
| Bare Kitty payload inside tmux | **Empirically fails** (probe B) even with `allow-passthrough on`; and a DCS-wrapped *non-placeholder* put renders but leaves tmux's grid empty, so later text draws across the image (probe C) | Unicode-placeholder mode (the image occupies real cells) plus tmux passthrough when `$TMUX` is set — Ghostty discussion #7224; `allow-passthrough` exists from tmux 3.3 |
| Inline images inside a scrolling list | tmux does not model image cells: subsequent text draws across the image and scroll smears it (probe C) | Keep images out of v1; later, confine them to a dedicated pane that reserves cells and repaints |
| sixel renderers (`mattn/go-sixel`, `timg`, sixel fallbacks) | **Ghostty implements no sixel at all** — zero `sixel` matches in the `v1.3.1` source, and a maintainer states "No and its not planned" (discussion #5832) | Kitty graphics protocol only |
| iTerm2 `OSC 1337 File=` inline images | **Ghostty implements no iTerm2 image protocol** — `File`/`FilePart`/`MultipartFile` fall into the `unimplemented OSC 1337` arm (`src/terminal/osc/parsers/iterm2.zig` at `v1.3.1`) | Kitty graphics protocol only |
| `chafa` / `viu` / `timg` / `img2txt` as the v1 path | Not installed here (`timg` is not even in the Arch repos — AUR only); installing them buys nothing that `mpv` + `ffmpeg` do not already do for the stated "open externally" requirement | `exec mpv` / `xdg-open` behind a `media.Opener` interface |
| Kitty **animation** actions (`a=f`, `a=a`, `a=c`) on Ghostty 1.3.1 | Parsed but **unimplemented** in 1.3.1 (`ERROR: unimplemented action`); they landed on `main` only after 1.3.1 | Static images; play animated media in `mpv` |
| Putting the cookie in the general config file, or in anything an editing panel round-trips | The sibling project's config is a panel-editable byte round-trip with no secret handling — this is exactly the surface the cookie must stay out of | A separate `0600` file + env override (Deep Dive 3) |
| `CGO_ENABLED=1` builds / cgo SQLite drivers | Destroys the single-static-binary property that justifies the stack | `modernc.org/sqlite`, pure Go |
| `httputil.DumpRequest*`, `%+v` on requests, or "dump on error" logging | Dumps include the `Cookie` and `Authorization` headers — a leaked session cookie on a shared-office machine | Log a redacted request descriptor; keep a test that asserts the cookie never appears in log output |
| `better-sqlite3` / native Node SQLite builds | Requires a native build step for no benefit on Node 26 | `node:sqlite` (verified working) or `bun:sqlite` |
| A background poller or a timeline cache | Contradicts the product decision (manual refresh, no timeline on disk) and is the fastest route to 429s and account restrictions | Manual refresh; reuse cursors instead of re-fetching |
| Browser automation (Playwright/Puppeteer/rod) as the data path | Heavyweight, obviously non-browser to the TLS/behaviour checks, and it turns a 2 MB tool into a browser-bundling project; even the library ecosystems treat fingerprint spoofing as an optional escape hatch, not the default | Plain HTTP with a realistic UA (+ a fingerprinting transport only if a probe proves it is needed) |

## Stack Patterns by Variant

**If the Go TLS probe is challenged (Cloudflare interstitial / `403` with `cf-ray`):**
- Keep everything; replace only the transport: `http.Client.Transport = tlsclient.NewHttpClient(...)` from `github.com/bogdanfinn/tls-client` v1.16.0 (utls-based) or a `uTLS` `DialTLSContext` with `refraction-networking/utls` v1.8.2.
- Because the whole x.com layer sits behind one `RoundTripper`-shaped seam, this is a one-file change — which is the architectural payoff of keeping the client thin.

**If the user later insists on inline images:**
- Implement `media.Opener` → `media.KittyRenderer`; detect capability from `TERM_PROGRAM`/`TERM`; DCS-wrap when `$TMUX` is non-empty; request an explicit placement size; reserve the cells; repaint the whole frame afterwards; and confine it to the post-detail pane (never a list).

**If a new endpoint is needed (bookmark folders, lists, notifications):**
- Only two things change: the op-ID table (re-run the discovery script) and a thin parse struct + fixture. No UI code moves. That is the test of whether the seam held.

**If requests start 404ing in a burst:**
- Suspect the transaction-ID key material first (it fails on *all* endpoints at once), the op-ID table second (fails per operation), the feature map third (fails loudly with `(336)`).

**If the account's session expires:**
- The only correct behaviour is to stop and say so — see the error table in §1.5. Never retry-with-backoff into a ban, and never fall back to a second account (out of scope by project decision).

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| Go 1.27.2 | `charm.land/bubbletea/v2` v2.1.0 | Bubble Tea v2's `go.mod` declares `go 1.26.0`, so 1.27.2 satisfies it |
| `charm.land/bubbletea/v2` | `charm.land/lipgloss/v2`, `charm.land/bubbles/v2` | **v1 import paths are incompatible**: v2 moved to the `charm.land` vanity domain (`github.com/charmbracelet/bubbletea` → `charm.land/bubbletea/v2`). Mixing a v1 dependency into a v2 app fails to build. |
| `bubbles/v2` | `bubbletea/v2` only | v2 widget APIs use v2 types (`tea.View`, `KeyPressMsg`); there is no v1-compatible mode |
| `modernc.org/sqlite` v1.60.1 | `database/sql`, `CGO_ENABLED=0` | Register/serve under the driver name `sqlite`; keep the DSN boring (`file:…?_pragma=journal_mode(WAL)`) |
| Any dependency | the static binary | One cgo import anywhere (`mattn/go-sqlite3`, some keyring packages, terminal libs) turns the build from static to dynamic — audit with `go list -deps -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' ./...` |
| tmux ≥ 3.3 (`tmux 3.7c` here) | Kitty passthrough | `allow-passthrough on` is necessary but *not sufficient* (probe B); the payload must be DCS-wrapped (probe C) |

## Installation

```bash
# Language toolchain: Go 1.27.2 is already installed (no rustup needed for this stack)

# Core TUI
go mod init github.com/lzd/x-terminal   # any module path; the repo is currently module-less
go get charm.land/bubbletea/v2@v2.1.0
go get charm.land/lipgloss/v2@v2.0.6
go get charm.land/bubbles/v2@v2.2.1

# Support
go get charmbracelet/x/ansi@v0.11.9
go get modernc.org/sqlite@v1.60.1
go get github.com/charmbracelet/x/exp/golden@v0.1.0   # golden-frame view tests

# Conditional — add ONLY if the TLS/Cloudflare probe shows plain net/http is challenged
# go get github.com/bogdanfinn/tls-client@v1.16.0

# Build the shipping artifact
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o ~/.local/bin/x-terminal .

# Runtime, already present on this host: mpv, ffmpeg (media openers)
# Deliberately NOT installed: chafa, viu, timg, img2txt (no v1 need)
```

## Open Questions to Settle Empirically (Phase 1, not by more research)

These are the places where this document is deliberately not the last word. Each has a cheap experiment attached, and each can change an implementation decision (though none of them changes the recommended stack).

| Open question | Why it matters | Cheapest experiment |
|---|---|---|
| Does plain Go `net/http` get challenged for **authenticated reads** (Cloudflare `403` + `cf-ray`, or an interstitial page)? | Decides whether `bogdanfinn/tls-client` v1.16.0 enters the build at all | One `GET` of `HomeTimeline` with the real cookies and a browser UA; inspect the status, `content-type`, and whether `cf-ray` is present. If it returns JSON, the transport story is done. `[INFERENCE]` Expect success — the fingerprint-spoofing transports in both reference libraries are opt-in |
| Do the currently-published `HomeTimeline` / `Bookmarks` / `Likes` op IDs work with a freshly computed `x-client-transaction-id`? | The whole feature set rests on it; also validates the Go port of the generator | Run the discovery script, then issue one request per operation and compare against a known-good UI state |
| How long does `auth_token` actually last in 2026? | Determines how often the human must re-paste cookies, and whether a "cookie likely expired" hint belongs in the UI | Observe, or read the cookie's own expiry (`auth_token` is set with a far-future or short expiry — inspect it once) |
| Are the `features` values that must be `false` (as opposed to absent/`true`) load-bearing? | Naive self-healing flips everything to `true` on `(336)`; if some must be `false`, healing can loop | Force a `(336)` deliberately with a stripped flag map and watch whether the healed request succeeds |
| Does Ghostty 1.3.1 retain Kitty images in scrollback, and what is the practical `image-storage-limit` ceiling when tmux is in the path? | Only matters if the image upgrade ships; would constrain the pane design | Emit a payload, scroll, screenshot; and try a multi-MB image with `image-storage-limit` at its 320 MB default |
| Does the current web build ever serve the *authenticated* page in a shape the transaction-ID scraper cannot parse (e.g. a new bundle layout)? | The scraper already chases X's build changes (Vite vs webpack, `sign.o-*` vs `ondemand.s`, 7-vs-16-hex hashes) | Keep the parse errors from `xclid`-style code *specific* (as `twscrape` does: "Cloudflare challenge page served instead of X web app") so the failure is self-diagnosing |

## Sources

**TUI frameworks**
- `https://charm.land/blog/v2` — Charm's v2 release announcement, dated 2026-02-23; "Cursed Renderer", keyboard enhancements, inline images claim
- `https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0` — "What's New in Bubble Tea v2": declarative `tea.View` struct fields, mode 2027 (Unicode width), mode 2026 (synchronized updates), OSC 52 clipboard, `OpenTTY`, `tea.Raw`, `WithWindowSize`, terminal/version/capability queries; Ghostty listed among keyboard-protocol terminals
- `https://raw.githubusercontent.com/charmbracelet/bubbletea/main/UPGRADE_GUIDE_V2.md` — v1→v2 API deltas (`View() tea.View`, `tea.KeyPressMsg`, `msg.Code`/`Text`/`Mod`, `"space"`, removed program options) and the vanity import paths
- `https://raw.githubusercontent.com/charmbracelet/bubbletea/main/go.mod` — `module charm.land/bubbletea/v2`, `go 1.26.0`, `clipperhouse/displaywidth v0.X`, `charmbracelet/x/ansi v0.11.9`, `x/exp/golden`
- `https://pkg.go.dev/charm.land/bubbletea/v2?tab=versions` — v2.1.0 is the current release; `https://pkg.go.dev/charm.land/lipgloss/v2?tab=versions` — v2.0.6; `https://pkg.go.dev/charm.land/bubbles/v2?tab=versions` — v2.2.1
- `https://api.github.com/repos/charmbracelet/bubbles/contents` (via `gh`) — widget package list (no image widget)
- `https://crates.io/api/v1/crates/ratatui` — 0.30.2, updated 2026-06-19; `https://docs.rs/ratatui/latest/ratatui` + `https://github.com/ratatui/ratatui/blob/main/BREAKING-CHANGES.md` — the `ratatui-core`/`ratatui-widgets`/`ratatui-crossterm` split; `https://github.com/ratatui/ratatui` README — `DefaultTerminal`/`ratatui::init()` pattern
- `https://pypi.org/pypi/textual/json` — 8.2.8, 2026-06-30; `https://textual.textualize.io/guide/workers/` — single event loop, `run_worker`/`@work`, "the app will not be able to process other messages until the request has completed"
- `https://registry.npmjs.org/@opentui/core` — 0.6.4 (2026-10-13), Zig core behind an FFI, Bun ≥1.3.14 / Node ≥26.4.0 + `--experimental-ffi`, "OpenCode uses OpenTUI in production"; `https://registry.npmjs.org/ink` — 8.0.0, `incrementalRendering`, `maxFps: 30`, `<Static>`
- `https://registry.npmjs.org/@oakoliver/bubbletea` — 1.2.0 (2026-10-04), MIT, "zero-dependency port of Charmbracelet's Bubbletea"; `https://api.npmjs.org/downloads/point/last-week/@oakoliver/bubbletea` — 45 downloads in the week ending 2026-10-12 (adoption evidence for the "Bubble Tea in TypeScript" option)
- `https://bun.com/docs/bundler/executables` — `bun build --compile` single-file executables
- Local measurements (this host, 2026-10-14): static Go binary 2,350,833 bytes / 0.9 ms median start; Python 3.14.7 10.4 ms bare, 17.4 ms with stdlib imports; Node 26.8.2 20.2 ms; Bun 1.4.3 3.8 ms; Deno 2.9.6 21.6 ms (15 runs each)
- `https://static.rust-lang.org/rustup/dist/x86_64-unknown-linux-gnu/rustup-init` → HTTP 200 (rustup is installable from this host; the cost is a toolchain download, not a blocker)

**x.com internals**
- `https://raw.githubusercontent.com/vladkens/twscrape/main/twscrape/account.py` — `TOKEN` bearer constant, `has_required_cookies()` (`auth_token`+`ct0`), header set, cookie-only `login_method`
- `https://raw.githubusercontent.com/vladkens/twscrape/main/twscrape/queue_client.py` — per-request `x-client-transaction-id`, 404→regenerate retry (issue #248), Cloudflare `cf-ray` detection, `(336)` feature healing, `(88)`/`(326)`/`(32)`/`403` session-vs-ban classification, `(-1) LoadShed` retry, `x-rate-limit-*` handling
- `https://raw.githubusercontent.com/vladkens/twscrape/main/twscrape/xclid.py` — the transaction-ID algorithm (site-verification key, `sign.o-*.js`/`ondemand.s` index scraping, cubic-bezier animation key, SHA-256 payload assembly, timestamp epoch 1682924400), script-URL discovery for both the Vite and legacy webpack builds (7/16-hex hash change on 2026-08-24), Cloudflare challenge detection (issue #330)
- `https://raw.githubusercontent.com/vladkens/twscrape/main/twscrape/api.py` — operation-ID table as `queryId/OperationName`, `GQL_URL`, `GQL_FEATURES`; `https://raw.githubusercontent.com/vladkens/twscrape/main/scripts/update-gql-ops.py` — the op-ID regeneration workflow; `https://raw.githubusercontent.com/vladkens/twscrape/main/twscrape/http.py` + `pyproject.toml` — httpx default vs `curl-cffi` browser-TLS backend, `TWS_HTTP_BACKEND`
- `https://pypi.org/pypi/twscrape/json` — 0.20.1, 2026-08-25, MIT; `https://raw.githubusercontent.com/vladkens/twscrape/main/readme.md` — cookie setup described as "the most stable setup", `unjar` export, "Cookie accounts that include auth_token and ct0 are activated immediately"
- `https://raw.githubusercontent.com/the-convocation/twitter-scraper/main/src/auth-user.ts` — `x-twitter-auth-type: OAuth2Session`, ct0-vs-auth_token comment, guest-token-from-HTML preflight, CycleTLS/TLS-fingerprint notes, `generateTransactionId` gating; `src/api-data.ts` — the shipped endpoint table (queryIds + features + field toggles) and the absence of `HomeTimeline`/`HomeLatestTimeline`/`Bookmarks`; `src/api.ts`/`src/xctxid.ts`/`src/rate-limit.ts`; README — cookie auth recipe, rate-limit waits "up to 13 minutes", "fixes … typically take at least a few days", Cloudflare/CycleTLS section; `https://registry.npmjs.org/@the-convocation/twitter-scraper` — 0.22.3, 2026-04-01, MIT
- `https://raw.githubusercontent.com/imperatrona/twitter-scraper/master/api.go`, `auth.go`, `tweets.go`, `bookmarks.go`, `search.go`, `profile.go` — Go client: guest-token flow, `X-CSRF-Token` from `ct0`, hard-coded bearer + hard-coded per-operation queryIds, the 404/`x-client-transaction-id` hotfix note, `HomeTimeline`/`HomeLatestTimeline`/`Bookmarks`/`TweetDetail` operation URLs; README — cookie/`SetAuthToken` auth and the "≤1 request per 1.5 s, ~150 requests/15 min" guidance; `https://pkg.go.dev/github.com/imperatrona/twitter-scraper?tab=versions` — v0.0.18
- `https://github.com/obnoxiousmods/DeleteTweets/blob/master/docs/refreshing-query-ids.md` — query IDs rotate with each web build; stale IDs return 404; the browse-profile → grep-bundle → match `{queryId,operationName}` pairs recipe
- `https://github.com/Lqm1/x-client-transaction-id` (npm `x-client-transaction-id` 0.3.2, MIT), `https://github.com/iSarabjitDhiman/XClientTransaction` (the algorithm write-up lineage, MIT; PyPI `XClientTransaction` 0.0.1), `https://github.com/langkor/x-client-transaction` (crates.io 0.1.0, 2025-05-02), `https://github.com/ichioda/x-client-transaction-id` (Go port: has a `go.mod`, not published to pkg.go.dev — verification: `https://pkg.go.dev/github.com/ichioda/x-client-transaction-id` → 404)
- Registry status checks: `https://pypi.org/pypi/twikit/json` (2.3.3, 2025-02-07), `https://pypi.org/pypi/twitter-api-client/json` (0.10.22, 2024-04-21), `https://pypi.org/pypi/snscrape/json` (0.7.0.20230622), `https://crates.io/api/v1/crates/egg-mode|twitter-v2|twitter-scraper`, `https://registry.npmjs.org/twitter-api-v2` (1.29.1, 2026-08-04)
- `https://pkg.go.dev/github.com/bogdanfinn/tls-client?tab=versions` (v1.16.0) and `https://pkg.go.dev/github.com/refraction-networking/utls?tab=versions` (v1.8.2) — the Go fingerprinting escape hatches

**Terminal media**
- `.planning/research/TERMINAL-MEDIA-PROBE.md` + `evidence/probe-{A,B,C}.png` (this repo, 2026-10-14) — **primary, host-specific**: Kitty graphics works in Ghostty 1.3.1; `KITTY_WINDOW_ID` unset; scaling/interpolation unless a placement size is requested; bare sequence fails in tmux 3.7c; DCS-wrapped works but tmux does not model image cells; sixel/iTerm2 untested *at probe time* (both later disproved from source, below); `chafa`/`viu`/`timg` absent and `mpv`/`ffmpeg` present
- Ghostty capability matrix, verified against docs **and** the released `v1.3.1` source tree (tarball `https://codeload.github.com/ghostty-org/ghostty/tar.gz/refs/tags/v1.3.1`), 2026-10-14: `https://ghostty.org/docs/features` (Kitty graphics is a documented feature); `https://ghostty.org/docs/config/reference` (`image-storage-limit`, default 320 MB, `0` disables; **no** `graphics`/`kitty-graphics` toggle); `https://ghostty.org/docs/install/release-notes` (1.3.1 dated 2026-03-13); `src/terminal/kitty/graphics_command.zig`, `graphics_exec.zig`, `graphics_image.zig`, `graphics_unicode.zig`, `src/renderer/image.zig`, `src/config/Config.zig`, `src/termio/Exec.zig` at tag `v1.3.1` (actions, z-index, Unicode placeholders, `t=d/f/t/s`, **animation unimplemented**, env vars); sixel negative: zero matches under `src/`+`include/` plus maintainer answer in `https://github.com/ghostty-org/ghostty/discussions/5832`; iTerm2 negative: `src/terminal/osc/parsers/iterm2.zig` `unimplemented OSC 1337` arm, corroborated by `https://github.com/ghostty-org/ghostty/issues/13011`; `https://ghostty.org/docs/vt/reference` (`OSC 8` supported)
- tmux recipe: `https://github.com/ghostty-org/ghostty/discussions/7224` (Unicode placeholders + tmux passthrough; `chafa -f kitty` must be forced under Ghostty) and `https://raw.githubusercontent.com/tmux/tmux/master/CHANGES` (`allow-passthrough` added in tmux 3.3); protocol background `https://github.com/kovidgoyal/kitty/blob/master/docs/graphics-protocol.rst`
- Per-language emitters, all versions/dates fetched from the registry or upstream tag feed on 2026-10-14: `https://crates.io/api/v1/crates/ratatui-image` (11.1.0, 2026-09-17; 12.0.0-rc.2, 2026-10-05; screenshot-tested against ghostty), `https://crates.io/api/v1/crates/viuer` (0.11.0), `https://crates.io/api/v1/crates/viu` (1.6.1), `https://github.com/blacktop/go-termimg` (`v0.1.26`, 2026-02-23; Bubble Tea `ImageWidget`; README flags the Kitty placeholder path as not production-ready), `https://github.com/dolmen-go/kittyimg` (`v1.0.1-rc1`; v1.0.0 2026-03-19), `https://github.com/BourgeoisBear/rasterm` (v1.1.2), `https://github.com/mattn/go-sixel` (v0.0.14), `https://github.com/eliukblau/pixterm` (v1.3.3), `https://github.com/qeesung/image2ascii` (v1.0.1, 2018 — dead); Go tag dates are MEDIUM confidence because `proxy.golang.org` is unreachable from this host. Node: `https://registry.npmjs.org/terminal-image` (5.0.1, 2026-07-09) + its deps `term-img` 7.1.0 / `supports-terminal-graphics` 0.1.0, `https://opentui.com/docs/components/image` (built-in `ImageRenderable`); Python: `https://pypi.org/pypi/textual-image/json` (0.14.1, 2026-09-25) and `https://raw.githubusercontent.com/Textualize/textual/main/src/textual/widgets/__init__.py` (no image widget in core); external CLIs verified read-only on this host via `pacman -Si` (`chafa 1.18.2-2` `extra` LGPL-3.0-or-later, `viu 1.6.1-2` `extra`, `timg` absent from the repos / AUR-only) plus `https://hpjansson.org/chafa/man/` and `https://raw.githubusercontent.com/hzeller/timg/main/README.md`

**Credentials and persistence**
- Host probes, 2026-10-14: `XDG_CONFIG_HOME` unset; `org.freedesktop.secrets` owned by `gnome-keyring-daemon` on the user bus; `secret-tool` present; `kwalletd6` present; Chromium installed (`~/.config/chromium/Default/Cookies`), Firefox absent
- `go doc os.UserConfigDir` (installed Go 1.27.2) + live call returning `/home/user/.config` — XDG semantics, relative-path error
- `https://pkg.go.dev/github.com/zalando/go-keyring?tab=versions` (v0.2.8, MIT) + `https://raw.githubusercontent.com/zalando/go-keyring/master/go.mod` (deps: `godbus/dbus/v5`, `wincred` — pure Go)
- Chromium's Linux cookie scheme: `https://chromium.googlesource.com/chromium/src/+/c51d7ae19a5e001eb9aea418b3efc83fcc9b4649/components/os_crypt/os_crypt_linux.cc` (`kSalt = "saltysalt"`, Linux iteration count = 1, `v10` fallback passphrase `peanuts`) and `https://docs.rs/chromium-safestorage-core/latest/chromium_safestorage_core/derive/` (platform table: Linux v11 = Secret Service passphrase, 1 round; AES-128); Firefox side: mozilla-central `CookiePersistentStorage.cpp` (the `moz_cookies.value` column is plaintext), plus maintained readers `https://pypi.org/pypi/browser_cookie3/json` (0.20.1, 2025-01-04) and `pycookiecheat` 0.8.0
- Keyring ecosystem, all registry/package metadata fetched 2026-10-14: `https://www.npmjs.com/package/keytar` — 7.9.0, **archived**, last publish 2022-02-17; `https://registry.npmjs.org/@napi-rs/keyring` — 2.1.1 (2026-10-10, MIT); `https://pypi.org/pypi/keyring/json` — 25.7.0; `SecretStorage` 3.5.0; `https://pkg.go.dev/github.com/99designs/keyring?tab=versions` — v1.2.2 (2022, dormant); `https://crates.io/api/v1/crates/keyring` — 4.2.0; Arch package metadata (`pacman -Si gnome-keyring` → 51.1-1, `provides: org.freedesktop.secrets`) with Hyprland 0.56.2-4 / `xdg-desktop-portal-hyprland` / `hyprpolkitagent` shipping no Secret Service implementation
- File-write hygiene: `https://pkg.go.dev/os#WriteFile` (perm is applied **before umask** and an existing file's mode is not tightened) — hence temp+fsync+rename and an explicit `chmod` in the recommended design
- `https://pkg.go.dev/net/http#Request.Write` plus Go's internal `reqWriteExcludeHeader` set (`Host`, `User-Agent`, `Content-Length`, `Transfer-Encoding`, `Trailer`) — i.e. `Request.Write` and the `httputil` dump helpers emit `Authorization` and `Cookie`; `https://pkg.go.dev/net/http` — redirect behaviour: sensitive headers (`Authorization`, `WWW-Authenticate`, `Cookie`) dropped for non-matching domains, retained for subdomains/same host, with the explicit warning that this predates the Fetch standard
- SQLite: `https://pkg.go.dev/modernc.org/sqlite` — v1.60.1, **BSD-3-Clause**, pure Go; `https://pkg.go.dev/github.com/mattn/go-sqlite3` — MIT, cgo; `https://pkg.go.dev/github.com/ncruces/go-sqlite3` — v0.35.6, MIT, driver `sqlite3`; `https://crates.io/api/v1/crates/rusqlite` — 0.40.2 with `license = "MIT"` in its `Cargo.toml`; `https://nodejs.org/docs/latest-v26.x/api/sqlite.html` — `node:sqlite` is **Stability 1.2 – Release candidate** (added v22.5.0, RC since v25.7.0); `https://registry.npmjs.org/better-sqlite3` — 13.0.3 (2026-08-05, MIT); `https://docs.python.org/3/library/sqlite3.html` — `autocommit` added in 3.12 (default still `LEGACY_TRANSACTION_CONTROL` for 3.14, confirmed locally); `platformdirs` 4.13.0 as the Python config-dir helper
- Local runtime checks, 2026-10-14: `node -e "require('node:sqlite')"` on Node 26.8.2 (DatabaseSync/StatementSync/Session/backup all present) and `bun -e "…bun:sqlite…"` on Bun 1.4.3; Python 3.14.7 stdlib `sqlite3` reports SQLite 3.53.4 and `connect().autocommit == -1`; `go run` of a two-line program returning `/home/user/.config` from `os.UserConfigDir()`

---
*Stack research for: read-only terminal client for x.com*
*Researched: 2026-10-14*
