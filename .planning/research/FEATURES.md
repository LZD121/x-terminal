# Feature Research

**Domain:** Read-only terminal (TUI) client for x.com — single-user personal tool, unofficial session-cookie data source
**Researched:** 2026-10-14
**Confidence:** MEDIUM-HIGH — competitor landscape HIGH (every claim URL-cited, activity dates fetched 2026-10-14); feature set MEDIUM (three table-stakes rows rest on upstream surfaces that are unverified until the fetch layer is spiked)

## Verdict

**Feasibility: YES, with four named conditions.** The UI half of this project is a solved problem in this author's stack (Bubble Tea v2 + a pure-`Update` model, proven in `/home/user/repos/rss-tm`). The read surfaces are demonstrably reachable today with `auth_token` + `ct0`: `twscrape` exposes `tweet_details`, `tweet_replies`, `tweet_thread`, `search`, `user_by_login`, `user_tweets`, `bookmarks`, and `following` against X's internal GraphQL from browser cookies and is under active development (<https://github.com/vladkens/twscrape>, pushed 2026-10-05). Two live TUIs (`xeet`, `XTUI`) prove the whole loop — cookie → GraphQL → rendered feed — works for a read-heavy client, and one of them (`XTUI`) is already read-only (<https://github.com/Panther114/XTUI>).

The conditions:

1. A fetch-layer spike must confirm, against the live endpoints, that the Following timeline, thread/reply, search, profile and bookmarks operations still answer with today's query IDs from a Go client (not a Python one).
2. The **likes** list is the weakest of the in-scope surfaces. `twscrape`'s published API surface includes `bookmarks()` but lists no likes operation (<https://github.com/vladkens/twscrape>), so "read my likes" needs its own verification before it is promised. `[UNVERIFIED]`
3. Query-ID discovery and caching must be designed in from day one, because X rotates internal operation IDs — `xeet` discovers and caches them and offers a manual override (<https://github.com/melqtx/xeet>, "details: query ids and retries"). This is a feature of the fetch layer, not an optimisation.
4. Rate-limit and expired-session handling must be first-class UI states, not log lines. Every surviving client in this space treats them as required behaviour (`xeet` ships offline/timeout/outage/rate-limit/expired-session recovery paths explicitly; `twscrape` builds account rotation and rate-limit waiting into the pool).

**Where the differentiation lives** (detail in "Differentiation Statement"): read-only by construction, a browser-independent cookie file, a feed with no images, no counts and no chrome, and a timeline that provably never touches disk. Each existing tool breaks at least one of those.

## Competitor Landscape

Everything in this section was fetched on 2026-10-14. "Last activity" is the newest commit pushed to the default branch (GitHub REST API where the anonymous rate limit allowed, otherwise the repository's `commits/<branch>.atom` feed) unless stated otherwise.

### A. Terminal / TUI clients for X (Twitter), past and present

| Project | URL | Lang | Data source | Read-only? | Last activity | What it does well | What limits it |
|---|---|---|---|---|---|---|---|
| **xeet** | <https://github.com/melqtx/xeet> | Go | X internal GraphQL, browser session borrowed from Chrome/Firefox/Brave/Helium/Zen, secret kept in macOS Keychain / Linux Secret Service, never in the YAML config | **No** — posts, replies, likes, reposts, quotes | pushed **2026-09-17** (73★) | The most complete live X TUI. Following + For You tabs, bookmarks, search, profiles, threads, quotes, inline truecolor image previews, tmux passthrough, clipboard on X11/Wayland with a file fallback, `doctor`/`whoami`, bounded retries, session-expiry detection, notification panel, theme picker | Browser must exist and be logged in; session lives in a keyring; deliberately includes the write path and account-level access; ASCII banner + images + notifications are the opposite of an unbranded office tool; `TODO.md` still lists GIF animation and Windows DPAPI import |
| **XTUI** | <https://github.com/Panther114/XTUI> | Rust | X internal GraphQL through a browser **extension + native messaging host** (Chrome/Edge only); optional official API | **Yes** — "Not included: posting, likes, follows, or DMs" | pushed **2026-08-13** (1★) | The closest thing to a read-only X TUI that exists: Home/Explore/Mentions/Bookmarks/Lists tabs, Following vs For You toggle, search, threads, profiles, help overlay, `M`/`O` media preview-or-open, black-and-white styling | Requires installing a load-unpacked browser extension and a native host — unusable over SSH or from another machine; 1★, single contributor, Firefox explicitly unsupported; ships a Rust toolchain requirement the workstation lacks |
| **omaX** | <https://github.com/MoizIbnYousaf/omaX> | TypeScript (Bun) + QML bar plugin | Chromium/Firefox-family profile cookies decrypted in-process via `secret-tool`; keeps cookie + keyring material in memory | **No** — like, bookmark, reply, compose | pushed **2026-08-31** (0★) | Proof that this exact workstation (Omarchy + Kitty + Bun) can host an X client: Kitty graphics for profile photos/media, reads the active Omarchy `colors.toml` and re-themes live, an agent-facing JSON CLI, `doctor` output, demo mode with fixtures | Targets Omarchy/Kitty specifically; theme-coupled rather than unbranded; write actions with an "explicit authorization boundary" for agents; 0★, unproven at any scale |
| **x-tui** | <https://github.com/iola1999/x-tui> | TypeScript (Bun + Ink) | Shells out to a resident Python `twitter-cli` daemon (uv), also maintained in-tree | **No** — like, retweet, bookmark, reply, quote, follow | pushed **2026-04-24** (0★) | Best-in-class interaction detail to steal from: stale-while-revalidate caching so back-navigation never refetches, per-list focus index preserved, optimistic writes with rollback, `Ctrl+R` refresh, `?` help, `y` copies the link via OSC 52, image viewer on `i` with Kitty/iTerm2/sixel/halfblock auto-detect | Write-heavy; needs a second runtime (Python/uv) plus a daemon; mouse-first; reply depth intentionally truncated to one page |
| **twit** | <https://github.com/blacktop/twit> | Rust | X internal GraphQL with `auth_token` + `ct0` pasted into `config.yaml` | Unclear/partial — README foregrounds AI summarisation and TTS, warns to use a burner account and that 2FA must be off | last commit **2026-09-05** (commits feed); 14★ | The *only* live X TUI whose README documents the config-file cookie mechanism this project has chosen, and it recommends Ghostty by name | Demands a burner account ("do NOT reuse your primary account"), 2FA disabled; pushes post text to third-party AI providers; config-file cookie is treated as a hazard rather than a deliberate design |
| **plume-tui** | <https://github.com/InstaZDLL/plume-tui> | Python | Cookie login, X + Instagram | No — browsing, likes, downloads | pushed **2026-04-28** (2★) | Shows a single terminal client spanning two hostile platforms with cookie auth | Scraper-shaped rather than reader-shaped; two platforms at once is exactly the scope trap this project avoids |
| **k41-vibe/X-CLI** | <https://github.com/k41-vibe/X-CLI> | Python | Reverse-engineered X endpoints | Unknown | pushed **2026-07-19** (0★) | Independent confirmation that reverse-engineered X CLIs keep getting written | No maintenance signal, no tests, no releases |
| **xummary** | <https://github.com/kernoeb/xummary> | Rust CLI + SwiftUI | Not the internal GraphQL path — daily digest | N/A | pushed **2026-10-01** (0★) | Articulates the anti-scroll thesis: "Your X timeline, read once a day instead of scrolled all day" | Not a browser-replacement reader; a summariser |
| **twterm** | <https://github.com/ryota-ka/twterm> | Ruby | Official API (v1.1 era; v2 after) | No — full-featured incl. posting | pushed **2023-12-15**; gem 2.10.2 released 2021-05-05 (<https://rubygems.org/gems/twterm>) | The historical high-water mark of *TUI craft* for Twitter: 244★, MIT, multi-tab lists, streaming, an `:command` line. Its README structure and tab model are worth reading before designing the navigation stack | Died of the API. RubyGems shows 43 versions ending in 2021; the last push predates the paid-tier collapse of read access |
| **x-cli** (formerly `t`) | <https://github.com/sferik/x-cli> (old path <https://github.com/sferik/t> redirects) | Ruby | Official X API v2 | No | pushed **2026-10-05** (5.6k★) | The most-starred live Twitter CLI, actively maintained across two API generations; a useful catalogue of the read verbs people actually want (`timeline`, `search`, `users`, `thread`) | Its life depends on paid official-API reads — the exact economics this project rejects |
| **twty** | <https://github.com/mattn/twty> | Go | Official X API v2, OAuth 2.0 + PKCE | No | 291★, `master`, 179 commits — active | Documents the "read one post from the terminal" use case explicitly, including that AI assistants are a caller ("x.com refuses direct fetching, so let them run `twty -read URL`") | Same official-API dependency; OAuth app registration is a non-starter for a zero-cost personal tool |
| **rainbowstream** | <https://github.com/orakaro/rainbowstream> | Python | Official API v1.1 + streaming, later reduced to 90-second polling | No — full interaction | pushed **2022-09-29** (3.5k★) | The most-starred terminal Twitter client ever; its docs are still the canonical "here is what a terminal Twitter client must feel like" write-up (<https://opensource.com/article/20/1/tweet-terminal-rainbow-stream>) | Its polling fallback exists only because Twitter killed the stream API for third parties; the 2022 stop is the end of the line |
| **oysttyer** | <https://github.com/oysttyer/oysttyer> | Perl | Official API | No | **archived**; last push **2023-02-02** (337★) | The `TTYtter` lineage: a single-file Perl client with a command language, showing how far a *text* client can go without a TUI framework | Archived. Same API wall |
| **nekome** | <https://github.com/arrow2nd/nekome> | Go | Official API v2 with embedded consumer keys | No | **archived 2023-04-14** (119★) | The design reference for *this* stack: a Bubble Tea-era Go TUI Twitter client with per-timeline panes, flexible styling, command list, keybinding docs | The README states development ended **2023-02-09** because Twitter abolished the free API plan; embedded tokens are likely dead |
| **earthquake** | <https://github.com/jugyo/earthquake> | Ruby (ncurses) | Official streaming API | No | 2018-11-29 (661★) | A ncurses Twitter client with streaming — the "live tail" feel | Streaming is gone from the third-party surface; abandoned |
| **tyrs** | <https://github.com/Nic0/tyrs> | Python (ncurses) | Official API (Twitter + Identica) | No | 2021-03-01 | Early precedent for a multi-source terminal social client | Abandoned |
| **turses** | <https://github.com/louipc/turses> / <https://pypi.org/project/turses/> | Python | Official API | No | pushed **2023-05-16** | The TUI whose config-file-plus-keybinding shape later appeared in every Python client | Abandoned; the canonical repo (`lwc/turses`) now 404s |
| **twnyan** | <https://github.com/arrow2nd/twnyan> | Go | Official API v2 | No | **archived**, pushed 2022-12-31 | Small, clean Go CLI reader | Archived |
| **tweetstreem** | <https://github.com/Setheck/tweetstreem> | Go | Official API | No | pushed **2023-01-07** (4★) | Cross-platform terminal client attempt | Died with the v1.1 API |
| **gockel** | <https://github.com/akrennmair/gockel> | Go | Official API | No | **archived**, 2014-03-09 | One of the first Go terminal Twitter clients | Archived, ancient |
| **twtr** | <https://github.com/maaslalani/twttr> | Go | Official API (presumed) | No | 13★, minimal | A "Twitter in the terminal" toy in the Bubble Tea ecosystem | No read fidelity; not a reader |

**Reading of the table.** The X terminal-client field is not empty — it is *young and thin*. Excluding dead projects, everything that works today was started after the API collapse and authenticates with a **browser session cookie**: `xeet` (73★), `twit` (14★), `XTUI` (1★), `omaX` (0★), `x-tui` (0★), `plume-tui` (2★). Every one of them either (a) requires a local logged-in browser, extension or keyring, or (b) is write-capable, or (c) pushes post text to third-party services. Nothing in the field is aimed at a reader who wants plain text in a tmux pane and nothing else.

### B. Non-TUI prior art the fetch layer shares DNA with

These are not competitors — they are the closest evidence about the data path, and two of them (`unjar`, `tweetxvault`) touch decisions this project has already made.

| Project | URL | Relevance |
|---|---|---|
| **twscrape** | <https://github.com/vladkens/twscrape> (2.9k★, MIT, pushed **2026-10-05**) | The reference implementation of the exact access path: "async Python library and CLI for X/Twitter Search and GraphQL endpoints… runs on your own account pool, keeps sessions in SQLite, rotates accounts when an endpoint is rate-limited"; takes `auth_token`+`ct0` cookies; ships mocked fixtures for 20+ operations and a `scripts/update-gql-ops.py` for query-ID drift; optional `curl-cffi` backend "for browser-like TLS fingerprinting". Read it before designing the fetch seam |
| **unjar** | <https://github.com/vladkens/unjar> (MIT, Rust; twscrape documents using `unjar x.com -f header` piped into `twscrape add_cookie my_account`) | The competing answer to "where does the cookie come from": read it from the browser profile and decrypt it. This project deliberately does the opposite (a 0600 file + env override); `unjar` is the thing to point users at if they want automation |
| **twitter-web-exporter** | <https://github.com/prinsss/twitter-web-exporter> (releases through **2026-08-31**) | Userscript that re-uses the logged-in web app session to export tweets/bookmarks/lists. Confirms the bookmark endpoint is reachable from a session cookie, and that a browser is not required to *be* the UI — only, in its case, to host the script |
| **tweetxvault** | <https://pypi.org/project/tweetxvault/> (0.2.5, released 2026-07-20) | "Archive X bookmarks, likes, authored tweets" — a maintained CLI that does exactly the bookmark/like mirroring this project scopes for local persistence. Independent evidence that (a) bookmarks and likes are the two lists worth keeping offline, and (b) likes *are* reachable in some form — but this tool has been reported broken by cookie/DPAPI issues, so treat the likes path as requiring its own spike (`[UNVERIFIED]`) |
| **twitterapi.io** | <https://twitterapi.io/pricing> and <https://twitterapi.io/pricing/details> | The per-call aggregator route: "$0.00015/tweet" sticker price, no subscription. It is the only zero-setup way to read X at scale, and it is still a third party reading on your behalf — the reason this project prefers its own session |
| **X official API pricing** | <https://docs.x.com/x-api/getting-started/pricing> (table indexed: Posts: Read $0.005 per resource, User: Read $0.010, capped at 3M Post reads/month) and the announcement <https://devcommunity.x.com/t/announcing-the-x-api-pay-per-use-pricing-pilot/250253> ("Post (Read): $0.005 per Post fetched") | The price floor that makes official-API clients (`x-cli`, `twty`) unaffordable for read-heavy personal use. Note: direct fetch of `docs.x.com` failed from this host (ECONNRESET); the figures are quoted from the vendor page as indexed and corroborated by the developer-forum announcement and the April 2026 update <https://devcommunity.x.com/t/x-api-pricing-update-owned-reads-now-0-001-other-changes-effective-april-20-2026/263025> |

### C. TUIs for neighbouring networks — the design references that actually matter

| Project | URL | Lang | Read-only? | Last activity | What to steal / what to learn |
|---|---|---|---|---|---|
| **newsboat** | <https://github.com/newsboat/newsboat> (3.9k★, MIT) | C++ | Yes | pushed **2026-10-10** | The longevity model for a text-terminal reader: separate `urls` and `config` files, `~/.newsboat/config` keybindings, **macros** ("execute sequences of actions with just two keystrokes"), **query feeds** (saved filters across sources), a built-in HTML renderer so you never need the browser (<https://github.com/newsboat/newsboat/blob/master/README.md>) |
| **neomutt** | <https://github.com/NeoMutt/neomutt> (3.8k★) | C | Yes | pushed **2026-10-13** | The "index menu → pager → back" navigation contract, `?` to show current bindings in any menu, and notmuch-style **saved queries** as first-class views (<https://neomutt.org/guide/gettingstarted.html>, <https://neomutt.org/feature/notmuch>) |
| **aerc** | <https://git.sr.ht/~rjarry/aerc> (<https://aerc-mail.org/>) | Go | Yes | active | Go TUI mail client: panes, `:commands`, per-account config, and the discipline of never blocking the UI on a slow backend. The best proof that "text list + detail pane + command line" scales to real daily use |
| **scli** | <https://github.com/isamert/scli> (538★, GPL-3.0) | Python/urwid | Mixed | active | Terminal chat reader with vim-like navigation, command entry, per-contact buffers, history saving and an optional terminal-bell notification. Its *optional* history is a nice template for "persist only if asked" |
| **iamb** | <https://github.com/ulyssa/iamb> | Rust | Mixed | pushed **2026-10-13** | Modern matrix TUI: window/thread model, lazy room loading, keybinding customisation. Good reference for a navigation stack that survives deep drill-downs |
| **tooi** | <https://codeberg.org/ihabunek/tooi> (Go, release 0.27.0 on **2026-07-04**) | Go | Mixed | active | The successor to `toot`'s dropped TUI — evidence that a CLI with a bolted-on TUI gets its TUI split out into its own program |
| **tut** | <https://github.com/RasmusLindroth/tut> (503★, MIT) | Go | Mixed | pushed **2023-12-18** | Bubble Tea Mastodon TUI with a full `:` command set, tabs/panes, templates, mouse support, password-manager hooks, `xdg-open` media. Dormant, but its **config + `:` commands + help** triad is the shape this project should copy |
| **mastui** | <https://github.com/kimusan/mastui> | Python/Textual | Mixed | release **2026-09-03** | The modern-feature catalogue: multi-column timelines that collapse on narrow terminals, per-profile config/keymap/cache directories, SQLite cache for offline reads, ANSI/Sixel/**TGP** image rendering with auto-detect, a URL-extractor modal for copying links, `F12` log viewer (<https://raw.githubusercontent.com/kimusan/mastui/main/README.md>) |
| **toot** | <https://github.com/ihabunek/toot> (1.3k★, GPL-3.0) | Python | Mixed | pushed **2026-09-12** | Its README states the TUI "is no longer being developed" and redirects to `tooi` (<https://raw.githubusercontent.com/ihabunek/toot/master/README.md>) — a live example of a project dropping its TUI instead of maintaining two front ends |
| **hackernews-tui** | <https://github.com/aome510/hackernews-TUI> (722★, MIT) | Rust/Cursive | Yes | pushed **2026-03-29** | A read-only, single-source terminal reader that stays alive: config file with per-key colour overrides, documented default shortcuts per view, story/comment/article/search views, and a roadmap in the README |
| **tuisky** | <https://github.com/sugyan/tuisky> (Rust, MIT) | Rust | Mixed | pushed **2025-12-28** | Bluesky TUI, Homebrew-packaged |
| **bsky_tui** | <https://github.com/ksk001100/bsky_tui> | Rust | Mixed | pushed **2026-08-30** | Bluesky TUI, crates.io-published |
| **skyter** | <https://github.com/hdb/skyter> (299 commits) | Rust | Mixed | pushed **2026-08-19** | Bluesky TUI with the highest commit velocity of the three |
| **bskytui** | <https://pypi.org/project/bskytui> | Python | Mixed | on PyPI | Confirms the pattern: every open protocol gets a handful of terminal clients, none of which dominate |
| **tuir** (fork of `rtv`) | <https://gitlab.com/ajak/tuir> (180★) | Python | Mixed | last commit **2020-04-26**; 42 open issues | Read-only-shaped Reddit reader whose **mailcap** media opener, `$TUIR_BROWSER`/`$TUIR_URLVIEWER` env hooks, `clipboard_cmd` and theme files are the cleanest pre-existing spec for "hand media and links to external programs" (<https://gitlab.com/ajak/tuir/-/raw/master/README.md>) |
| **rtv** | <https://github.com/michael-lazar/rtv> (4.6k★) | Python | Mixed | **archived 2023-02-20** | The cautionary tale; see "Dead Competitors" below |
| **rss-tm** (local) | `/home/user/repos/rss-tm` | Go 1.27 + Bubble Tea v2 | Yes | this author, present | Same author, same stack: pure-`Update` model with every read as a `tea.Cmd` (import-boundary enforced), data-table key grammar rendered by the help overlay, golden frames as the view specification, a `FetchFunc` seam with a second source already plugged in. The pattern to reuse; explicitly not a coupling |


## Dead Competitors and Why

This is the failure mode the project has to dodge, so it is worth stating precisely. Five distinct executioners, in order of how many projects they killed:

**1. Platform access policy (the big one).**
On 2023-01-19/20 Twitter cut off third-party clients and rewrote its developer terms to bar use of the API "to create or attempt to create a substitute or similar service or product to the Twitter Applications"; Twitterrific and Tweetbot pulled their apps, Talon announced it would cease to work, and Fenix was removed the same week (<https://techcrunch.com/2023/01/19/twitterrific-tweetbot-app-store-removal-twitter-api/>). That single event ends most of the table above: `twterm`'s last push is 2023-12-15 and its RubyGems release history stops in 2021 (<https://rubygems.org/gems/twterm>); `oysttyer` is archived; `rainbowstream` stopped in 2022 after the stream API was withdrawn and it fell back to 90-second polling (<https://github.com/abakely/twix>); `tweetstreem` stopped 2023-01-07.

**2. Removal of the free API tier.**
`nekome`'s README states development ended on 2023-02-09 because the Twitter API free plan was abolished, and that its embedded tokens are probably already invalid (<https://github.com/arrow2nd/nekome>). `twnyan`, by the same author, was archived the same year. Any client whose viability depends on a free official tier has a scheduled death date.

**3. Legal escalation against alternative front ends.**
X sent cease-and-desist letters to Nitter on 2026-08-24; the project announced it would continue after legal advice, went down and came back, and the GitHub repository was **archived on 2026-09-11**; XCancel suspended service on 2026-09-14/15 (<https://github.com/zedeus/nitter>, <https://techcrunch.com/2026/09/15/nitter-and-xcancel-are-dead-again-after-xs-latest-legal-actions/>). Nitter's own README now reads, verbatim:

> DATA_9f4c2kq7_START
> On 24 August 2026, cease and desist letters were sent by X Corp. demanding a permanent takedown of Nitter instances and the project's repository. **UPDATE 10 October 2026:** I'm seeking funding and legal help.
> DATA_9f4c2kq7_END

(source: <https://github.com/zedeus/nitter>, fetched 2026-10-14). Note what Nitter *was*: a **public, hosted alternative front end** for many users, which is precisely the shape X's terms target. It is the reason this project's "personal, not distributed" decision is load-bearing rather than cosmetic. `[INFERENCE]` — I am inferring intent-to-enforce against single-user clients from enforcement against a public service; the observed facts are the C&D and the archive.

**4. Maintainer burnout and dependency rot.**
`rtv` is the cleanest case study. It was not killed by Twitter; its author shut it down on 2019-06-03 and the repository was archived in 2023. From the shutdown issue, verbatim:

> DATA_3p8wz1vn_START
> Lately, RTV has stopped being fun for me to work on. I suspect this is the typical burnout that many open source maintainers face… More concerning, I no longer wish to support reddit as a platform… I will not be offering any future security patches or bugfixes.
> DATA_3p8wz1vn_END

(<https://github.com/michael-lazar/rtv/issues/696>). The fork `tuir` then stalled on its own dependency: its README says it depends on a bundled PRAW 3.6.1 fork that is out of support, that upgrading across PRAW major versions is "slow going", that the author is "a relatively inexperienced developer and I have less time than I'd like to devote to software", and that feature requests are therefore not being worked on — last commit 2020-04-26 (<https://gitlab.com/ajak/tuir/-/raw/master/README.md>, <https://gitlab.com/api/v4/projects/ajak%2Ftuir/repository/commits>).

**5. Session/endpoint fragility (a slow executioner, still running).**
Query IDs rotate. `xeet` must "discover and cache current ids for posting, timelines, likes, and unlikes" and documents a manual fallback (<https://github.com/melqtx/xeet>); `twscrape` ships `scripts/update-gql-ops.py` for the same reason. Nitter's README notes that after the 2024 restrictions, "those who wanted to host a Nitter instance had to connect it to a real X account" (<https://techcrunch.com/2026/09/15/nitter-and-xcancel-are-dead-again-after-xs-latest-legal-actions/>). A client that hard-codes one query ID per operation dies on the next rotation.

**How this project dodges all five.** (i) Personal and never distributed, so there is no public-service target and no user base to disappoint — and read-only removes the write-auth exposure on top. (ii) Zero dependence on official-API tiers or pricing; the cookie is the user's own session. (iii) Query-ID discovery + caching + manual override is a designed feature, and each endpoint is isolated so a broken one degrades a single view instead of the app. (iv) The scope is small enough for one person to keep alive, and the feature list below is deliberately short for that reason. (v) No browser, keyring or extension dependency means breakage is in the HTTP layer, which is one package, not in a browser-integration matrix.

## Differentiation Statement

Given everything above: **what can this be that nothing else is?**

Today, every working X terminal client is one of two things. Either it is a *browser-dependent, write-capable social app* (`xeet`, `omaX`, `x-tui`, `twit`) — session borrowed from a keyring or an extension, images and notifications, posting and liking — or it is a *dead or toy* project. The single read-only exception, `XTUI`, depends on installing a browser extension plus a native messaging host, supports only Chrome/Edge, and is a 1★ single-author effort. Nothing in the field is built for the person this project is for.

This project can be **the reader**: a client whose only job is to put the Following timeline, a thread, a search, a profile, your bookmarks and your likes on screen as text, and then get out of the way. Concretely, the combination that does not exist anywhere else:

1. **Read-only by construction**, not by policy — no write code paths exist, so there is nothing to mis-fire and no write-scope exposure.
2. **Auth from a 0600 cookie file with an env override**: no browser needed at request time, no keyring, no extension, no native host. It works over SSH, in a tmux pane, on a machine where the browser is closed. (`twit` is the only tool that documents this mechanism, and it tells you to use a burner account.)
3. **A feed with no chrome**: no images inline, no like/repost/view counts, no avatars, no banners, no brand colours, no border. It reads like a log viewer in a shared office — which is the stated motivation, and which no competitor even attempts (`omaX` re-themes to the desktop palette; `xeet` prints an ASCII logo).
4. **Provably no timeline on disk** while bookmarks and likes *are* mirrored locally for offline reading and fast local filtering — a split no competitor makes, because they either persist everything (offline caches in `mastui`, `x-tui`) or nothing.
5. **Explicit refresh with an "N new posts" affordance** and no polling, no stream, no auto-advance; the reading position is never mutated by a background event.
6. **A request budget you can see**: single-flight, deduped, rate-limit-aware, with per-session request accounting surfaced in the UI. Every competing client hammers endpoints in the background; this one can promise restraint because nothing else needs the bandwidth.
7. **One keymap grammar, one help overlay, one completion source.** The keys, the in-app help, and the shell completion come from a single table, so they cannot disagree — the pattern already proven in `/home/user/repos/rss-tm`.

The honest counterweight: this is a smaller product than `xeet`. It will not post, will not show you an image unless you ask an external program to, and will break when X rotates a query ID until the fetch layer is updated. That is the trade: it is the only one of these that can be left open in an office without anyone realising what it is.

## Feature Landscape

Complexity is cost-to-build-and-keep-working *in this project* (Go, Bubble Tea v2, from scratch, no shared code with `rss-tm`):
**LOW** = one package, an afternoon, low upstream risk. **MEDIUM** = a package plus tests plus config or error semantics. **HIGH** = the load-bearing part of the project, with live upstream risk.

### Table Stakes (Users Expect These)

Missing any of these and the tool is not a replacement for the browser tab, it is a demo.

| Feature | Why Expected | Complexity | Notes |
|---|---|---|---|
| **Following timeline as the default view** | This is the thing being replaced. `twscrape` exposes the operation (`search`, `user_tweets`); `xeet` and `XTUI` both land on a feed | HIGH | The single largest slice of work: session → GraphQL → parse → model → wrap → render. Depends on auth, the fetch seam, the post model and the list view. Must render *something* useful from a partial/failed fetch |
| **Keyset pagination ("load older") without losing position** | Any feed reader that cannot scroll back is not a reader; `newsboat`, `aerc`, `neomutt`, `tut` and every X client above do it | MEDIUM | Cursor comes from the last entry of the returned page; the in-flight page must be anchored so a late response cannot scroll the viewport. Depends on the cursor primitive and the list view. Guard against duplicate entries when X's ranking shifts between pages |
| **Post detail: full reply thread + inline quoted post** | Threads are the reason people open a post at all. `xeet` (`Q` opens the embedded quote as a full conversation), `x-tui` (tweet detail), `twit` all implement it | MEDIUM-HIGH | Two operations (details + replies/thread) plus quote inlining; must share the link/handle parsing with search and profile. `x-tui` documents a deliberate one-page reply limit for latency — copy that: paginate replies, do not "expand" them |
| **Search: posts by keyword, users by handle/name** | Explicitly in scope in `PROJECT.md`; present in every live competitor | MEDIUM | `twscrape` shows search takes a product switch (`Latest` default, `Top`, `Media`) — expose that as a small toggle, not a setting. User search is a separate operation returning a different entity type, so the list view must be entity-agnostic |
| **Profile: bio + recent posts** | Same surfaces as `xeet` ("a readable bio, exact follower/following and post counts, location, website, and join date") at a fraction of the polish | MEDIUM | Bio block + paginated post list. Depends on the list view and the entity-agnostic model. Counts are read-only decoration: show them plainly, do not style them |
| **Bookmarks list (own account)** | Named in `PROJECT.md`; `twscrape.bookmarks()` and `twitter-web-exporter` both prove the endpoint; `xeet`/`XTUI`/`x-tui` all ship it | MEDIUM | Read via the authenticated session. This is also the list that gets mirrored locally, so it must have a stable identity (post ID + timestamp) from day one |
| **Likes list (own account)** | Named in `PROJECT.md` | MEDIUM (risk: **HIGH uncertainty**) | **Weakest link.** `twscrape`'s documented surface has `bookmarks()` but no likes operation; X removed the public-facing Likes tab for other accounts. Verify before promising. Mitigation if the endpoint is unavailable: local mirror is still valuable once imported, and the feature can degrade to "bookmarks only" without touching UI code |
| **Session-cookie auth from a 0600 file, env-var override** | `PROJECT.md` decision; the account cookie is a bearer credential for the whole session | MEDIUM | No prior art in `rss-tm` (it has no secret mechanism at all). Requirements: file created `0600`, never round-tripped through the panel-editable config, env var wins, redaction in every error path and log line, `logout` that deletes only this copy. `xeet`'s TODO lists "atomic `0600` config writes, symlink refusal" as shipped behaviour — copy that threat model exactly |
| **Manual refresh on an explicit key** | `PROJECT.md` decision; the whole no-surprise-mutation posture depends on it | LOW | Refresh must preserve the selected post and scroll offset, and must not re-order what the user is reading. Prefer "N new posts above — `R` to load" over silent replacement |
| **Config file** | Every terminal client in the landscape has one, in a documented location with a documented schema (`newsboat` `~/.newsboat/config`, `tut` `XDG_CONFIG_HOME/tut/config.toml`, `hackernews-tui` `hn-tui.toml`) | MEDIUM | Owns: paths, keymap, colour role mapping, media opener command, reply-page size, cache/DB location. Must be byte-stable on rewrite and must never hold the cookie |
| **Error surfacing: expired session / rate limit / network down / upstream shape change** | `xeet` ships "actionable offline, timeout, service outage, rate-limit, and expired-session recovery" as a first-class feature; `twscrape` waits or raises on rate limits | MEDIUM | Four distinct states with distinct remedies: re-auth, wait (show when), retry, update query IDs. Must be visible without quitting to a log file, and must never leave a spinner that means "hung" |
| **In-app help overlay generated from the keymap** | `?` is universal here — `tuir`, `tut`, `hackernews-tui`, `x-tui`, `XTUI`, `neomutt` (per-menu bindings) | MEDIUM | Single source of truth for keys, help text and completion. `rss-tm` already renders help from a data-table key grammar — reuse the idea, not the code |
| **Vim-ish navigation grammar + a real back stack** | `j`/`k`, `Enter` open, `Esc`/`q` back, `g`/`G`; identical across `tut`, `tuir`, `hackernews-tui`, `scli`, `xtui` | LOW | The subtle part is the stack: returning from a profile reached *from a post* reached *from search* must restore list, selection and scroll. `xeet` advertises exactly this ("`esc` restores the previous screen and reading position, including when browsing another author") |
| **Long-post text rendering: wrapping, quoting, links, unicode width, resize** | Posts are long, contain CJK/emoji/RTL, and arrive at any window size | MEDIUM | This is where a "plain text" claim gets tested. Needs a width-aware wrapper, a quote/URL tokenizer, and reflow on `SIGWINCH`. Ghostty renders grapheme clusters and RTL correctly for LTR-only text (<https://ghostty.org/docs/features>), but the app still owns wrapping |
| **External media opening behind an abstraction** | `PROJECT.md` decision; `tuir` does it with mailcap, `tut` with `xdg-open`, `x-tui` with an image viewer, `xeet` with a browser + clipboard fallback | LOW-MEDIUM | One interface, one v1 implementation (`xdg-open`, which reaches `mpv` and `ffmpeg` on this machine). Keep the seam honest: no image code in the model layer |
| **Copy link / open in browser** | `y` in `x-tui` uses OSC 52; `tuir` pipes to `clipboard_cmd`; `xeet` ships X11/Wayland clipboard with a fallback | MEDIUM | Two mechanisms, one key: OSC 52 for the terminal clipboard and a fallback to `wl-copy`/`xclip`; `xdg-open` for the browser. Must degrade quietly over SSH (`xeet` documents exactly this) |
| **Borderless, unbranded, low-colour rendering** | `PROJECT.md` core value; the office setting is the point | LOW-MEDIUM | A design constraint expressed in code: no logo, no banner, no brand hues, no borders, no emoji icons, and colour used only for structure (author/handle/time/quote) with a mono fallback. Golden frames as the spec |
| **Startup and scroll feel instant** | `PROJECT.md` constraint; a text feed has no excuse for lag | LOW | No network on cold start (show the last-known-nothing state and wait for `R`), no synchronous IO in `Update`, no allocation in the scroll path. Architectural, not a feature — but it is the thing users notice first |
| **A visible way to leave and to check who you are** | Users must be able to confirm which account the cookie belongs to before blaming the app | LOW | `whoami`-style line in the status/help area: handle + session expiry if known. `xeet whoami` and `omaX cli whoami` both exist for this reason |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---|---|---|---|
| **Timeline never persisted, enforced structurally** | The office/privacy promise is only credible if it is architectural. No timeline cache, no read-state DB, no scroll history file, no logging of post bodies | LOW (if designed in) / HIGH (if retrofitted) | Enforce with a package boundary + test: only the bookmarks/likes store may open the SQLite file; timeline types never reach it. `x-tui` and `mastui` both cache to disk by default — the opposite choice is the point |
| **Bookmarks and likes mirrored to local SQLite** | Offline reading of the two lists worth keeping, instant list open, local keyword filtering, and a durable record that survives endpoint churn | MEDIUM | Requires auth + list views + a stable post identity. Depends on the same store the anti-features forbid for the timeline — so the store must be *named* for its scope (e.g. `saved.db`) and tested to prove it never sees a Following page |
| **Cookie-file auth with no browser, keyring or extension** | Works over SSH, in tmux, with the browser closed. `xeet`/`omaX`/`x-tui` all need the browser; `XTUI` needs an extension + native host. Only `twit` documents this route, and it tells users to burn an account | MEDIUM | `0600` file + env override + redaction + symlink refusal. The differentiator is legitimacy: here it is the documented, deliberate mechanism, not a hazard note |
| **Read-only by construction** | Nothing to mis-fire; no write scopes; no accidental engagement; least ToS surface for a personal tool | LOW upfront, compounding | Cost: it can never grow into a client that posts. Accept it. Note that `XTUI` proves read-only is a *sellable* posture — it is the only competitor that advertises it |
| **Log-viewer aesthetics: no counts, no avatars, no inline media** | The stated reason this project exists. Reads as generic tool output; zero social-app signalling | LOW-MEDIUM | The hard part is restraint, not code: engagement counts exist in the data and must be *dropped* at render time. Golden frames should fail if an image or a heart glyph appears |
| **Keys + help + shell completion from one table** | The three surfaces cannot drift; a new key is one row. No competitor does this (each documents keys in three places) | MEDIUM | Shell completion for a TUI is thin but real: subcommands (`doctor`, `whoami`), profile/config paths, and a `--help` that lists the same keys |
| **"N new posts" refresh instead of silent replacement** | Makes manual refresh a *decision* rather than an event that rewrites what you were reading. Directly supports the "no surprise timeline mutations" requirement | LOW-MEDIUM | Requires keeping the fetched page anchored and comparing IDs against the visible head. Pairs with refusing to auto-refresh |
| **Visible request budget** | A one-line, honest account of upstream calls this session ("12 requests, 0 rate-limited") — proof of restraint, and the first diagnostic when something 429s | LOW-MEDIUM | Needs the request manager to be a single choke point from day one. `xeet`/`twscrape` do this internally for bots; nothing surfaces it to a human reader |
| **Renderer interface with a Kitty-graphics implementation staged** | Ghostty 1.3.1 supports the Kitty graphics protocol (<https://ghostty.org/docs/features>, <https://sw.kovidgoyal.net/kitty/graphics-protocol.html>); enabling it later must not require touching the feed | MEDIUM | v1: external opener only. v1.x: implement the second renderer behind the existing seam and verify empirically in Ghostty — including through tmux, where `xeet` needed DCS passthrough for tmux 3.3+. `x-tui` and `omaX` show the payoff (inline previews) and the cost (protocol auto-detect, cache dir, motion/reduced-motion settings) |
| **Degradation ladder: truecolor → 256 → 16 → mono, with meaning never carried by colour** | Survives `NO_COLOR`, `TERM=dumb`, piped output, and a locked-down colour scheme. `hackernews-tui` exposes per-role colours; `newsboat`/`neomutt` have decades of colour-scheme handling to copy | MEDIUM | One palette table with role names; the mono frame is an assertion, not an afterthought (it is also what makes the "unbranded" claim testable) |
| **`doctor`-style self-diagnosis** | Tells the user *why* it is broken (cookie expired vs query ID stale vs rate limited vs offline) with a one-line remedy each. `xeet doctor` and `omaX doctor` both ship; theirs print fingerprints, never cookie values | MEDIUM | Depends on error surfacing + the query-ID cache. Must print structural facts only — never the cookie, never post bodies |
| **Query-ID cache with manual override** | The survival feature of the whole project: X rotates operation IDs, and the app should self-heal, then let the user paste an ID from devtools when self-healing fails | MEDIUM-HIGH | `xeet`: "discovers and caches current ids… if discovery fails, explains how to set the id manually from browser devtools"; `twscrape`: `scripts/update-gql-ops.py` + mocked fixtures. Design the cache as data, not code |

### Anti-Features (Commonly Requested, Often Problematic)

| Anti-Feature | Why Requested | Why Problematic | Alternative |
|---|---|---|---|
| **Background / periodic auto-refresh, streaming, or tailing** | "Why doesn't it just update?" — every other feed reader seems to | Generates unbounded upstream requests with no human trigger, maximises bot-detection surface, and can silently rewrite what you are reading; directly contradicts the manual-refresh decision in `PROJECT.md` | Explicit refresh key + a non-destructive "N new posts" banner. If the user wants liveness, let `watch` or a shell loop call the binary |
| **Write actions (post, reply, like, repost, follow, bookmark)** | They are the obvious next request, and `xeet`/`x-tui`/`omaX` all have them | Requires write-scope auth and irreversible side effects from a terminal tool, plus the largest step up in ToS exposure. `rtv`'s author pointed out the same class of risk in a different form; `xeet` ships a whole "never retry an ambiguous CreateTweet" protocol because of it | Stay read-only. Open the post in the browser (one key) for anything that mutates the account |
| **In-terminal video/audio playback** | "You already open images — why not play the video?" | Pulls in decoders, animation timing, focus stealing, and audio in a shared office; `twit` links ALSA/audio playback to build flags and cross-compilation pain | Hand to `mpv` via the renderer interface. `ffmpeg`/`mpv` are already installed on this machine |
| **Inline images in the timeline by default** | The most visible difference from `xeet`/`x-tui`/`omaX`, which all render previews | Breaks the plain-text/unbranded promise, costs protocol detection + cache + motion settings, and makes the feed conspicuous in exactly the setting this tool is for | Images only on demand, through the renderer interface; a Kitty-protocol implementation can land later behind the same seam and stay off by default |
| **Notification badges, unread counts, terminal bell** | "I want to know when someone replies" | Turns a quiet reader into an attention machine; a bell or a flashing title is visible to everyone around the desk; `xeet` had to build an entire notification panel to do this properly, with unread state | No counters at all. If notification is ever wanted, make it one explicit command (`--count-new` printed to stdout) that a user can wire up themselves |
| **Algorithmic "For You" as the default, or infinite scroll** | It is what x.com opens to, and infinite scroll is what a feed "should" feel like | Algorithmic ranking plus endless scroll is the exact behaviour the project exists to escape; it also maximises upstream traffic. No terminal client in the landscape paginates indefinitely — every paginator is keyed | Following by default; the algorithmic feed reachable only by an explicit toggle; pagination always keyed and always finite-feeling |
| **Multi-account switching / account pool** | `twscrape` rotates accounts and `tut`/`mastui` support multiple profiles | Single-user, single-session tool. Account rotation exists to dodge rate limits at scale — the opposite of the restraint this project promises, and it multiplies the secret-handling problem | One session. `logout` + a new cookie is the "switch accounts" story |
| **Timeline cache, read-state, or history on disk** | Offline reading, instant reopen, "mark as read", resume-where-I-left-off — all standard in `newsboat` and `mastui` | Leaves a browsing trace on the machine, explicitly rejected in `PROJECT.md`; it is also what makes "no timeline on disk" unverifiable | In-memory only for the timeline; disk only for bookmarks/likes. Say so in `--help` and assert it in tests |
| **Engagement metrics in the feed (likes, reposts, views as first-class UI)** | They are in the data, and omitting them looks like a missing feature next to `x-tui`'s count row | They are the social-app signal — the thing that makes a feed feel like a feed. Rendering them also invites styling them | Drop them in the list view; if a post detail needs a number, print it as plain text with no glyph and no colour |
| **Mouse-first interaction (click targets, drag select, wheel-to-focus)** | `x-tui`, `XTUI`, `tut` and `mastui` all ship it; it looks like polish | SGR mouse tracking adds hit-testing, selection artefacts and terminal-compat branches, and it competes with the "this is just text you can copy" property; the target user is keyboard-first in a tmux pane | Keyboard only in v1. Keep OSC 8 hyperlinks so terminal-native click-to-open still works for free |
| **AI summarisation / read-aloud / translation** | `twit` ships multi-provider summarisation and TTS; it is an easy differentiator to bolt on | Exfiltrates post text to third parties (a direct violation of the privacy posture), requires key management, and adds a network dependency that can fail in the feed path | Not built. The user can pipe a copied permalink into whatever they like |
| **Browser-cookie extraction (keyring decryption, DPAPI) and browser-extension bridges** | `xeet`, `omaX`, `XTUI` all do it; "just read my browser" is more convenient than pasting cookies | Keyring/DPAPI decryption is a per-platform, per-browser matrix (`omaX` ships separate Chrome/Chromium/Windows/macOS/Linux cookie providers; `xeet` needed GNOME Keyring, KDE Wallet, Secret Service and Windows DPAPI paths); a browser extension adds an install step and an OS-specific native host. Both are large surfaces for a personal tool | A `0600` cookie file + env override, with documentation pointing at `unjar`-style tools (<https://github.com/vladkens/unjar>) if the user wants automation |
| **Themes, skins, plugin systems, live palette watching** | `mastui` and `omaX` do theme switching live; users like it | Unbounded configuration surface for a single-user tool, and theme-following is exactly the "looks like a social app" failure mode | Three colour roles + a `NO_COLOR`/16-colour/mono ladder in the config file. No plugin API |
| **Desktop notifications (`notify-send`) / terminal title flashing** | Cheap way to feel alive | Conspicuous in a shared office; the same reason the notification badge is rejected | Nothing. Print to stdout if scripted |
| **In-app media download/manager (thumbnails, cache dir, GIF animation)** | `x-tui` has a media cache dir, `plume-tui` downloads; `xeet` plans GIF animation | Storage management, cache eviction, format handling and animation timing — all for a feature the project delegates | External program. If a media file is ever needed on disk, the external opener (or a shell command) can save it |

## Feature Dependencies

```
[Secret store: 0600 cookie file + env override] ──> [Auth/session load]
[Config file] ─────────────────────────────────────> [Media renderer choice]
                                                 └──> [Colour roles / keymap]

[Auth/session load] ──> [Fetch seam: GraphQL client]
[Query-ID cache + discovery] ──> [Fetch seam]        (ids are data, not code)
[Request manager: single-flight, dedupe, 429 budget] ──> [Fetch seam]

[Fetch seam] ──> [Post/User/Thread models] ──> [Text renderer: wrap, quote, links, unicode]
[Post/User/Thread models] ──> [Entity-agnostic list view: cursor + selection + scroll]
[Text renderer] ──> [List view]
[Cursor / keyset pagination primitive] ──> [List view] ──> [Following timeline]
                                                       ├──> [Search results (posts)]
                                                       ├──> [User search results]
                                                       ├──> [Profile recent posts]
                                                       ├──> [Bookmarks list]
                                                       └──> [Likes list]

[Navigation stack (restore list + selection + scroll)] ──> [Post detail + replies/thread]
[Post detail] ──> [Inline quoted post] ──enhances──> [Navigation stack]

[Post detail] ──> [Media attachments discovered] ──> [Renderer interface] ──> [External opener (v1)]
                                                                          └──> [Kitty-graphics renderer (v1.x)]

[Local store (saved.db): bookmarks + likes] <──requires── [Auth/session load]
                                            <──requires── [List view]      (to render what it stored)
[Local store] ──enhances──> [Bookmarks list], [Likes list]   (offline open + local filter)

[Keymap table (single source of truth)] ──> [Help overlay]
                                        ├──> [Shell completion]
                                        └──> [Input handling in every view]

[Error surfacing: expired / 429 / offline / shape change] <──requires── [Fetch seam]
[Error surfacing] ──> [doctor-style self-diagnosis] ──enhances──> [Query-ID cache override]

[Request budget counter] <──requires── [Request manager]
[Colour degradation ladder] <──requires── [Text renderer]

CONFLICTS
[Timeline in memory only] ──conflicts──> [Timeline persistence / read-state / history]
[Manual refresh only]     ──conflicts──> [Background polling / streaming]
[Borderless minimal UI]   ──conflicts──> [Mouse-first interaction]  (hit-test artefacts vs pure text)
[Read-only by construction] ──conflicts──> [Write actions]
```

### Dependency Notes

- **Fetch seam requires the query-ID cache.** X rotates internal operation IDs, so IDs must be resolved (discovered or cached) *before* any request can be built (`xeet` discovers and caches them, and documents a manual devtools override; `twscrape` ships `scripts/update-gql-ops.py`). Build the ID resolver as its own package with a data file, not a constant block in the client.
- **Every read feature requires the list view + cursor primitive.** Timeline, search, profile posts, bookmarks and likes are the same widget over different fetchers. If the list view is written per-feature, the project is five times its size. This is the first thing to extract, and it is why "entity-agnostic model" appears in the table-stakes notes.
- **Post detail requires the navigation stack.** Returning from a thread to the exact scroll position is a stack property, not a view property; `xeet` markets it explicitly ("`esc` restores the previous screen and reading position"), and retrofitting it after the views exist means rewriting every view's state.
- **Error surfacing requires the fetch seam and must land before pagination.** Pagination is the first feature that can genuinely hit a rate limit, so the 429/expired-session path has to exist before the "load older" key is bound. `xeet` treats offline/timeout/outage/rate-limit/expired-session recovery as required behaviour, not polish.
- **The local store requires auth, and must not require the timeline.** `saved.db` is opened only by the bookmarks/likes paths; the timeline model never reaches it. This dependency is what makes the "no timeline on disk" claim testable rather than aspirational — write the boundary test at the same time as the store.
- **Likes listing requires its own verification before it is scheduled.** `bookmarks()` is a documented operation in `twscrape`; no likes operation is listed. Sequence the likes feature after the fetch spike, and design the local mirror so it stays useful with bookmarks alone.
- **The help overlay, completion and input handling all require the keymap table.** Deriving all three from one table is cheap only if the table exists first; bolting it on later means three sources of truth drift silently.
- **The Kitty renderer enhances the media feature and conflicts with nothing** — because v1 puts it behind the renderer interface. If media rendering is called directly from the feed, the v1.x upgrade becomes a rewrite. Verify protocol behaviour in Ghostty empirically, including through tmux, where `xeet` needed DCS passthrough for tmux 3.3+.
- **The request-budget counter requires a single choke point.** If any fetch bypasses the request manager, the counter lies — and a lying counter is worse than none, because it is the thing that tells the user the tool is behaving.
- **`doctor` enhances the query-ID cache.** Self-diagnosis is only actionable if there is a place to record "this ID is stale" and a way to accept a replacement.
- **Colour degradation conflicts with nothing but constrains the renderer:** role names must resolve before glyphs are chosen, and the mono frame must be a tested output, not a fallback path.

## MVP Definition

This project has exactly one user, and that user's stated job is "read the feed quietly". The MVP is therefore *the whole read surface with none of the conveniences*.

### Launch With (v1)

- [ ] **Following timeline + keyset pagination** — the reason the tool exists; nothing else matters if this lags or loses position.
- [ ] **Post detail with reply/thread expansion and inline quotes** — a timeline without threads is a headline list.
- [ ] **Search: posts and users** — explicit requirement, and the fallback when the timeline is stale.
- [ ] **Profile: bio + recent posts** — the third drill-down target, sharing the same list view.
- [ ] **Bookmarks and likes read from the session** — explicit requirement; the only account-owned lists worth keeping.
- [ ] **Cookie auth from a `0600` file with env override, plus `whoami`** — required before any read works; `whoami` because a wrong-account session is the most confusing failure mode.
- [ ] **Manual refresh preserving position** — required by the no-surprise-mutation decision.
- [ ] **Error surfacing: expired session, rate limit, offline, upstream shape change** — without it, a rotated query ID looks like a hang, and the user cannot tell whether to re-auth or wait.
- [ ] **External media opener behind the renderer interface** — cheapest complete answer to "what about images/video".
- [ ] **Config file + keymap table + help overlay** — the config is small; the keymap table is what makes help, input and completion cheap.
- [ ] **Borderless, low-colour, count-free rendering** — the core value; shipping a decorated v1 would invalidate the point.

### Add After Validation (v1.x)

- [ ] **Local SQLite mirror for bookmarks/likes + local filter** — trigger: the bookmark list is slow to open, or you catch yourself wanting to search it.
- [ ] **Kitty-graphics image renderer behind the existing interface** — trigger: you actually open media often enough to care.
- [ ] **`doctor` self-diagnosis page** — trigger: the first time a query ID rotates.
- [ ] **Colour degradation ladder + `NO_COLOR` handling** — trigger: the tool is used on a second terminal or piped.
- [ ] **Clipboard fallbacks (OSC 52 → `wl-copy`/`xclip`) and open-in-browser** — trigger: link handling becomes a daily action.
- [ ] **Shell completion** — trigger: the subcommand set and keymap table stop moving.
- [ ] **Reply/thread pagination depth as a config knob** — `x-tui` caps replies at one page for latency; make that a setting rather than a constant.
- [ ] **Request budget line in the status area** — trigger: rate limits are hit in normal use.

### Future Consideration (v2+)

- [ ] **Export / pipeline integration (markdown, JSON, `grep`-friendly)** — deliberately out of v1 in `PROJECT.md`; revisit only if the real motivation turns out to be "get posts somewhere else".
- [ ] **Digest mode** — one screen, once a day, in the spirit of `xummary`; attractive but a different product.
- [ ] **List and Community timelines** — `twscrape` exposes both; pure scope growth with no new mechanics.
- [ ] **A second source behind the same seam** — the `FetchFunc` pattern in `rss-tm` already proves a foreign source can share the scheduler; only worth doing if another network is ever wanted.
- [ ] **Kitty keyboard protocol / progressive key enhancement** — Ghostty supports it; a nicety, not a need.
- [ ] **Media cache with eviction** — only if the Kitty renderer lands and the cache becomes a real directory to manage.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Auth: `0600` cookie file + env override | HIGH | MEDIUM | P1 |
| Fetch seam (GraphQL client, headers, query-ID resolution) | HIGH | HIGH | P1 |
| Post/User/Thread model + text renderer | HIGH | MEDIUM | P1 |
| Entity-agnostic list view + cursor/selection/scroll | HIGH | MEDIUM | P1 |
| Following timeline | HIGH | HIGH | P1 |
| Keyset pagination ("load older") | HIGH | MEDIUM | P1 |
| Error surfacing (expired / 429 / offline / shape) | HIGH | MEDIUM | P1 |
| Manual refresh preserving position | HIGH | LOW | P1 |
| Post detail + reply thread | HIGH | MEDIUM | P1 |
| Search (posts + users) | HIGH | MEDIUM | P1 |
| Profile (bio + recent posts) | MEDIUM | MEDIUM | P1 |
| Bookmarks list | MEDIUM | MEDIUM | P1 |
| Likes list | MEDIUM | MEDIUM (HIGH uncertainty) | P1, gated on spike |
| Navigation stack with state restore | HIGH | LOW-MEDIUM | P1 |
| Inline quoted post | MEDIUM | LOW-MEDIUM | P1 |
| Borderless / low-colour / count-free rendering | HIGH | LOW-MEDIUM | P1 |
| External media opener behind interface | MEDIUM | LOW-MEDIUM | P1 |
| Config file | MEDIUM | MEDIUM | P1 |
| Keymap table + help overlay | MEDIUM | MEDIUM | P1 |
| `whoami` / session status line | MEDIUM | LOW | P2 |
| "N new posts" refresh banner | MEDIUM | LOW-MEDIUM | P2 |
| Colour degradation ladder + `NO_COLOR` | MEDIUM | MEDIUM | P2 |
| Clipboard fallbacks + open in browser | MEDIUM | MEDIUM | P2 |
| Request budget / status accounting | LOW-MEDIUM | LOW-MEDIUM | P2 |
| Local SQLite mirror for bookmarks/likes | MEDIUM | MEDIUM | P2 |
| `doctor` self-diagnosis | MEDIUM | MEDIUM | P2 |
| Shell completion (from keymap table) | LOW | LOW | P2 |
| Kitty-graphics renderer | MEDIUM | MEDIUM | P3 |
| Reply-depth config knob | LOW | LOW | P3 |
| List / Community timelines | LOW | MEDIUM | P3 |
| Digest mode | LOW | MEDIUM | P3 |
| Export to markdown/JSON | LOW (revisit) | LOW-MEDIUM | P3 |

**Priority key:** P1 = must exist for v1 to be a browser replacement. P2 = add the first time the need bites. P3 = future consideration.

## Competitor Feature Analysis

Compared against the live terminal X clients plus the two design references that matter most for interaction decisions. Cells describe what each tool actually ships, per its own documentation at the URLs in "Sources".

| Feature | xeet | XTUI | omaX | x-tui | twit | tuir (Reddit) | Our approach |
|---------|------|------|------|-------|------|---------------|--------------|
| Data source | X internal GraphQL via browser session | X internal GraphQL via extension + native host (or official API) | X internal GraphQL via browser profile cookies | vendored Python `twitter-cli` daemon | X internal GraphQL from `config.yaml` cookies | Reddit official API (bundled PRAW 3) | X internal GraphQL from a `0600` cookie file or env var |
| Auth mechanism | Browser picker → keychain / Secret Service; atomic `0600` writes; symlink refusal | Load-unpacked extension + native host | Profiles found via `secret-tool`; secrets in memory only | Separate `twitter auth login`, then a CLI boundary | Paste `auth_token` + `ct0` into YAML (docs say: burner account, 2FA off) | OAuth via browser | Paste the cookie once into a `0600` file, or set the env var; never round-tripped through the general config surface |
| Read-only? | No | **Yes (advertised)** | No | No | Unclear/partial | No | **Yes, by construction** |
| Timeline surfaces | Following + For You, per-tab session position; bookmarks tab | Home / Explore / Mentions / Bookmarks / Lists; `f` toggles Following↔For You | Sidebar across For You/Following/… (`1`–`6`) | Feed / Search / Bookmarks / Profile | CLI verbs: home, user timeline, search, replies | `/front`, `/r/sub`, multis, domains | Following by default; algorithmic feed behind an explicit toggle; one view per surface with a real back stack |
| Pagination | Loaded posts kept per tab for the session; explicit fresh-load key | Documented paging keys | `n` load more | PgDn/Space paging, `gg`/`G`; replies capped at one page | CLI `--limit` / `--pages` | `Space`/`J`/`K`, comment folding | Keyset cursor from the last entry, anchored so a late page cannot move the viewport |
| Post detail / thread | Replies screen; `esc` restores reading position | Open + back | Threads via `Enter`/`q` | Two-stage load (post, then page 1 of replies) | CLI `-read URL` | Submission + comment tree with folding | Two-stage load, paginated replies, quotes inlined, all through the shared list view |
| Search | Posts + users | `/` search | `/` search; agent JSON CLI | Search tab | `-s` keyword | Navigation prompt | Posts (Latest/Top toggle) + users; `Esc` returns to the prior view |
| Bookmarks / likes | `--bookmarks` tab | Bookmarks tab | Bookmarks + likes | Bookmarks tab | — | Saved posts | Bookmarks + likes, read-only, mirrored to `saved.db` for offline open |
| Images in terminal | Lazy truecolor previews, collages, native Kitty/iTerm2, tmux DCS passthrough, ANSI/off modes | `M` preview / `O` open | Kitty graphics with text fallback | Halfblock in-feed + native viewer on `i` | AI image summarisation + TTS | mailcap → external program | **None in v1.** External opener behind one interface; a second renderer can land later without touching the feed |
| Engagement counts | Shown | Shown | Shown | Shown (replies/reposts/likes/bookmarks/views) | n/a | Scores/comments | Dropped at render time; plain text only if a detail view needs a number |
| Notifications | Reply/mention popups, unread state, notification panel | — | Notifications view | — | TTS read-aloud | — | None. No badges, no bell, no counters |
| Mouse | Keyboard-first | Keyboard-first | Keyboard-first | Mouse first-class | — | Optional | Keyboard only; OSC 8 hyperlinks still let the terminal handle clicks |
| Config | YAML + theme command; secrets excluded | `config.json` (XDG/AppData), `XTUI_MOTION` | Omarchy plugin layout + `colors.toml` | Several `X_TUI_*` env vars | `config.yaml` with cookie + AI + TTS keys | `tuir.cfg` + themes + mailcap, `--copy-config` | One documented config file: paths, keymap, colour roles, media opener, reply depth. Cookie explicitly excluded |
| Diagnostics | `doctor`, `whoami`, `logout`, `inspect-har` (structure only) | `extension status` | `doctor` | `twitter status` | — | `--help` | `whoami` in the status line; a `doctor` page in v1.x driven by the same error enum |
| Failure handling | Bounded retries, rate-limit errors, session-expiry detection, cookie-leak tests, never retries an ambiguous write | Extension bridge failure modes | — | — | — | Documented dependency-rot stall | Four named states with remedies; single-flight + request budget; each endpoint isolated so one break degrades one view |
| What it is for | Replace the whole X app in the shell, posting included | Read X read-only, B/W, with a browser extension installed | X in an Omarchy-themed terminal, with an agent CLI | X in a mouse-and-vim terminal app | One-shot reads and AI summaries | Browse Reddit from a terminal | Read the Following feed, threads, search, profiles and saved lists as quiet plain text in a pane, with nothing on disk |

## Sources

All URLs fetched 2026-10-14 unless noted. Activity dates come from the GitHub REST API (anonymous), a repository `commits/<branch>.atom` feed, the GitLab API, the Codeberg API, or the project's own releases feed — stated per item.

**Live X terminal clients**

- <https://github.com/melqtx/xeet> — Go X TUI, browser-session auth, write-capable; 73★, pushed 2026-09-17. Its `TODO.md` (`https://raw.githubusercontent.com/melqtx/xeet/master/TODO.md`) enumerates the shipped feature set and the deliberately out-of-scope list
- <https://github.com/Panther114/XTUI> — read-only Rust X TUI via browser extension + native messaging host; 1★, pushed 2026-08-13
- <https://github.com/MoizIbnYousaf/omaX> — Bun/TypeScript X client for Omarchy with Kitty graphics and an agent JSON CLI; 0★, pushed 2026-08-31
- <https://github.com/iola1999/x-tui> — Bun + Ink X TUI with inline images and optimistic writes; 0★, pushed 2026-04-24
- <https://github.com/blacktop/twit> — Rust X TUI, cookies in `config.yaml`, AI/TTS extras, burner-account warning; last commit 2026-09-05 (commits feed)
- <https://github.com/InstaZDLL/plume-tui> — Python X/Instagram terminal client; pushed 2026-04-28
- <https://github.com/kernoeb/xummary> — daily-digest reader; pushed 2026-10-01
- <https://github.com/k41-vibe/X-CLI> — reverse-engineered Python X CLI; pushed 2026-07-19

**Dead or dormant X terminal clients**

- <https://github.com/ryota-ka/twterm> — 244★, pushed 2023-12-15; <https://rubygems.org/gems/twterm> — 43 versions, latest 2021-05-05
- <https://github.com/orakaro/rainbowstream> — 3.5k★, pushed 2022-09-29; polling fallback documented at <https://github.com/abakely/twix>; background at <https://opensource.com/article/20/1/tweet-terminal-rainbow-stream>
- <https://github.com/oysttyer/oysttyer> — 337★, archived, pushed 2023-02-02
- <https://github.com/arrow2nd/nekome> — 119★, archived 2023-04-14; development ended 2023-02-09 (free API tier abolished)
- <https://github.com/arrow2nd/twnyan> — archived, pushed 2022-12-31
- <https://github.com/jugyo/earthquake> — 661★, pushed 2018-11-29
- <https://github.com/Nic0/tyrs> — pushed 2021-03-01
- <https://github.com/louipc/turses> — pushed 2023-05-16; <https://pypi.org/project/turses/>
- <https://github.com/akrennmair/gockel> — archived, 2014-03-09
- <https://github.com/Setheck/tweetstreem> — pushed 2023-01-07
- <https://github.com/maaslalani/twttr> — 13★, toy-scale

**Live X CLIs that depend on the official API (for contrast)**

- <https://github.com/sferik/x-cli> (formerly <https://github.com/sferik/t>) — 5.6k★, pushed 2026-10-05
- <https://github.com/mattn/twty> — 291★, OAuth 2.0 + PKCE against X API v2
- <https://github.com/sferik/twitter-ruby> — the v1.1-era library behind `t`

**Fetch-layer prior art**

- <https://github.com/vladkens/twscrape> — 2.9k★, pushed 2026-10-05; cookies, account pool, rate-limit rotation, mocked fixtures, `update-gql-ops.py`, optional `curl-cffi` TLS fingerprinting
- <https://github.com/vladkens/unjar> — browser-profile cookie extraction integrated with twscrape
- <https://github.com/prinsss/twitter-web-exporter> — userscript exporter; releases through 2026-08-31
- <https://pypi.org/project/tweetxvault/> — bookmarks/likes/authored-tweet archive CLI, released 2026-07-20
- <https://twitterapi.io/pricing> and <https://twitterapi.io/pricing/details> — third-party per-call aggregator pricing

**Platform policy and pricing (the forces that shaped the field)**

- <https://techcrunch.com/2023/01/19/twitterrific-tweetbot-app-store-removal-twitter-api/> — third-party client cutoff and the developer-terms rewrite, January 2023
- <https://docs.x.com/x-api/getting-started/pricing> — pay-per-usage pricing (Posts: Read $0.005 per resource; 3M Post reads/month cap). Direct fetch from this host failed (ECONNRESET); figures quoted as indexed and corroborated by the two sources below
- <https://devcommunity.x.com/t/announcing-the-x-api-pay-per-use-pricing-pilot/250253> — "Post (Read): $0.005 per Post fetched"
- <https://devcommunity.x.com/t/x-api-pricing-update-owned-reads-now-0-001-other-changes-effective-april-20-2026/263025> — April 2026 pricing revision
- <https://console.x.com/api/credits/pricing> — machine-readable event pricing (`OwnedRead: 0.001`, `Post: 0.005`)
- <https://github.com/zedeus/nitter> — 14.5k★, archived 2026-09-11; README carries the 24 Aug 2026 C&D notice
- <https://techcrunch.com/2026/09/15/nitter-and-xcancel-are-dead-again-after-xs-latest-legal-actions/> — Nitter archived, XCancel suspended
- <https://www.theverge.com/tech/984819/nitter-which-let-you-read-x-posts-without-using-x-is-offline> — corroborating report on the shutdown

**Adjacent-network TUIs used as design references**

- <https://github.com/RasmusLindroth/tut> — 503★ Go/Bubble Tea Mastodon TUI, pushed 2023-12-18
- <https://codeberg.org/ihabunek/tooi> — Go Mastodon TUI, release 0.27.0 on 2026-07-04
- <https://github.com/ihabunek/toot> — 1.3k★ Mastodon CLI, pushed 2026-09-12; TUI discontinued in favour of `tooi`
- <https://github.com/kimusan/mastui> — Python/Textual Mastodon TUI, release 2026-09-03; multi-column layout, ANSI/Sixel/TGP renderers, SQLite cache, per-profile config
- <https://github.com/sugyan/tuisky>, <https://github.com/ksk001100/bsky_tui>, <https://github.com/hdb/skyter>, <https://pypi.org/project/bskytui> — Bluesky TUIs, pushed 2025-12-28 / 2026-08-30 / 2026-08-19 / published on PyPI
- <https://gitlab.com/ajak/tuir> — TUIR, 180★, last commit 2020-04-26 (via `https://gitlab.com/api/v4/projects/ajak%2Ftuir/repository/commits`); mailcap/env/clipboard/themes spec at <https://gitlab.com/ajak/tuir/-/raw/master/README.md>
- <https://github.com/michael-lazar/rtv> — 4.6k★ Reddit TUI, archived 2023-02-20; shutdown announcement <https://github.com/michael-lazar/rtv/issues/696>
- <https://github.com/newsboat/newsboat> — 3.9k★ C++ RSS reader, pushed 2026-10-10; macros and query feeds in <https://github.com/newsboat/newsboat/blob/master/README.md>
- <https://github.com/NeoMutt/neomutt> — 3.8k★, pushed 2026-10-13; <https://neomutt.org/guide/gettingstarted.html>, <https://neomutt.org/feature/notmuch>
- <https://git.sr.ht/~rjarry/aerc> / <https://aerc-mail.org/> — Go terminal mail client, active
- <https://github.com/isamert/scli> — 538★ urwid terminal client for Signal; vim keys, command entry, optional history
- <https://github.com/aome510/hackernews-TUI> — 722★ read-only Rust Hacker News TUI, pushed 2026-03-29
- <https://github.com/ulyssa/iamb> — Rust Matrix TUI, pushed 2026-10-13
- `/home/user/repos/rss-tm` — local sibling: Go 1.27 + Bubble Tea v2 feed reader (patterns only, no shared code)

**Terminal capability**

- <https://ghostty.org/docs/features> — Ghostty supports the Kitty graphics protocol, grapheme-cluster rendering, and RTL rendering for LTR-only text
- <https://sw.kovidgoyal.net/kitty/graphics-protocol.html> — the Kitty graphics protocol specification

**Confidence notes.** Competitor existence, licence, read-only status and activity dates: HIGH (fetched directly on 2026-10-14, dates machine-read). Claims about *what each tool does well* are taken from that tool's own README and are therefore vendor-sourced: HIGH that the feature exists, MEDIUM about quality. The likes endpoint's availability is `[UNVERIFIED]` and is the one table-stakes row that must be spiked before it is scheduled. The reading that X's enforcement targets public alternative front ends rather than single-user personal clients is `[INFERENCE]` from the Nitter and XCancel events.

---
*Feature research for: a read-only terminal x.com client (single user, personal tool)*
*Researched: 2026-10-14*
