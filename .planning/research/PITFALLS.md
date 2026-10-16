# Pitfalls Research

**Domain:** read-only terminal (TUI) client for x.com, reading x.com's internal web GraphQL
endpoints with the user's own session cookie (`auth_token` + `ct0`) — single user, not distributed
**Researched:** 2026-10-14
**Confidence:** MEDIUM overall. Mechanism-level claims (how the signer/query IDs/rate limits fail)
are HIGH — taken from live code and issue threads in maintained scrapers. X's own terms text and
any absolute rate-limit number are MEDIUM — X publishes neither, and the numbers below are the
best public measurements, not a contract.

**How to read confidence tags:** `[HIGH]` = primary source read (repo file, issue thread, host probe).
`[MEDIUM]` = search-index-level quotation of a primary document I could not fetch directly.
`[UNVERIFIED]` = asserted but not verified. `[INFERENCE]` = my reasoning from evidence, not stated
by a source.

---

## Phase labels used below (the roadmap will refine names; the mapping is what matters)

| Label | Phase |
|---|---|
| **P1 Transport spike & config** | Go module, config, `0600` secret file, cookie loading, one authenticated read proven end-to-end from a CLI |
| **P2 Read layer** | query-ID registry + codegen, `x-client-transaction-id` signer, session health probe, per-endpoint rate ledger |
| **P3 Timeline view** | Bubble Tea shell, width model, borderless renderer, manual refresh, keyset pagination + dedup, golden frames |
| **P4 Post detail & threads** | `TweetDetail`, reply-thread expansion (`ShowMoreThreads` cursor), quoted posts |
| **P5 Search & profiles** | `SearchTimeline`, `UserByScreenName`, user timeline |
| **P6 Bookmarks & likes** | bookmarks/likes reads + the only local persistence surface |
| **P7 Media & external openers** | renderer abstraction, `xdg-open`, clipboard, optional Kitty-protocol view |
| **P8 Hardening & operations** | error surfaces, soft-ban detection, repair/`doctor` path, log audit, release hygiene, runbook |

---

## Critical Pitfalls

### Pitfall 1: Cookie-authenticated reads put the user's real account in the enforcement path

**What goes wrong:**
A terminal client that authenticates with `auth_token` is not an anonymous scraper. Every read is
attributable to a real account, so the failure modes are account-shaped: throttling, a silent
"soft ban" where requests succeed but return less, a temporary read-only lock, and — in the tail —
suspension. The tool may appear to work perfectly while the account is being degraded.

**Why it happens:**
Anonymous scraping (guest tokens, public endpoints) is protected only by IP-level limits, so
operators think in terms of "getting blocked". Session-cookie access has an identity attached, so
X's heuristics have a subject to sanction, and the sanction is applied to the user's personal
account rather than a disposable one. Prior art says this plainly:

- `the-convocation/twitter-scraper`: "**While we are not aware of confirmed cases caused by this
  library, any account you log into with this library is subject to being banned at any time. You
  have been warned.**" ([README](https://github.com/the-convocation/twitter-scraper/blob/main/README.md)) `[HIGH]`
- `x-native` (same shape as this project — single account, cookie auth, read paths): "Use a
  **dedicated/secondary account** — scripted access can get accounts flagged."
  ([README](https://github.com/jpoindexter/x-native)) `[HIGH]`
- Twikit's account-protection page: "Twitter actively monitors for automated behavior. Following
  these guidelines reduces but doesn't eliminate the risk of account suspension." and the rule of
  thumb "never send more requests than a human reasonably could in the same timeframe."
  ([Twikit docs](https://d60-twikit.mintlify.app/advanced/account-protection)) `[HIGH]`

**Honest risk model for this project** (state this to the user, do not soften it):

1. Volume is the dominant variable, not intent. Nitter's maintainer reports that on the internal
   API *"even 1 request per second"* is enough to hit a secondary limiter that blocks the account
   **for 24 hours across endpoints** ([nitter#1107](https://github.com/zedeus/nitter/issues/1107)) `[HIGH]`.
2. The thresholds are undocumented and unstable: as of Dec 2025 the maintainer describes a
   *"completely opaque 24 hour rate limit"* ([nitter#1349](https://github.com/zedeus/nitter/issues/1349)) `[HIGH]`.
   So the risk cannot be *bounded* by design — only minimised.
3. A single user doing one manual refresh per session, one request at a time, is far below every
   reported threshold. A background poller is not.
4. The one mitigation prior art recommends — a burner account — **does not cover this project's
   core features**: bookmarks and likes are the authenticated account's own lists (PROJECT.md,
   Active requirements). A burner account cannot read *your* bookmarks. That tension must be stated
   as a decision, not hidden behind "use a throwaway account".
5. Multi-account rotation is itself risky and against X's posture: twscrape's README warns
   "X/Twitter's Terms of Service discourage using multiple accounts."
   ([twscrape readme](https://github.com/vladkens/twscrape)) `[HIGH]`

**How to avoid:**
- Manual refresh only (already decided) and **single-flight**: never issue two requests
  concurrently, ever — not for prefetch, not for media metadata, not for "while we're loading,
  check DMs". Concurrency is the behaviour that looks automated.
- One request per user action; page sizes at X's defaults (≈20 for timelines/search/bookmarks,
  ≈40 for user-tweets endpoints — twscrape uses exactly those, see
  [`api.py`](https://github.com/vladkens/twscrape/blob/main/twscrape/api.py) `[HIGH]`); never
  "prefetch the next page" in the background.
- No writes, no timers, no retry loops on failure. A retry loop after a 429 is the pattern that
  converts a 15-minute throttle into a 24-hour lock.
- Put the risk statement in the README **and** on first run (one line, e.g. `read-only personal
  client; uses undocumented endpoints against X's terms; the risk is to your own account`).
- Give the user a "safe mode" switch that hard-caps requests per session (e.g. 30) so a debugging
  session cannot accidentally become a crawl.

**Warning signs:**
- Any 429 on a *single* manual refresh (that means the account is already near a limiter, or a
  background process is still running).
- `x-rate-limit-remaining` decreasing faster than the user pressed keys.
- The web client itself starts showing "you're viewing too much" / a suspicious-activity prompt.
- The account gets logged out elsewhere, or the tool's session starts failing right after a
  browser session was established.

**Phase to address:** P1 (risk statement + safe-mode default), enforced in every phase; audited in P8.

---

### Pitfall 2: "Personal, not distributed" is not a legal shield

**What goes wrong:**
The project assumes that single-user, never-published use makes the ToS problem disappear. It does
not: it changes *who* bears the consequence (the user's account rather than a public service), and
it removes the specific liability shape that gets projects a cease-and-desist. It does not make the
access permitted.

**Why it happens:**
Two different risks get conflated. (a) *Terms breach* — X's acceptable-use terms prohibit exactly
this access pattern: **"You may not access the Services in any way other than through the currently
available, published interfaces that we provide. For example, this means that you cannot scrape the
Services…"** (X Terms of Service PDF dated 2026-04-10,
[legal.x.com](https://legal.x.com/content/dam/legal-twitter/site-assets/x-terms-of-service-2026-04-10/en/x-terms-of-service-2026-04-10.pdf)) `[MEDIUM — quoted from search-index of the official PDF; my direct fetch timed out]`.
There is no personal-use carve-out in the quoted clause. (b) *Enforcement pressure* — this scales
with visibility and scale. X's August 2026 cease-and-desist to Nitter alleged "unlawful use and
circumvention" of X's API and "access[ing] accounts and session tokens"; the repository was archived
(TechCrunch reported the archive on 2026-09-11; the banner on the repo's pages now reads
2026-10-09 — the date reported by the two sources disagrees, the outcome does not) and XCancel went dark
([TechCrunch, 2026-09-15](https://techcrunch.com/2026/09/15/nitter-and-xcancel-are-dead-again-after-xs-latest-legal-actions/)) `[HIGH]`.
That action targeted a *distributed service with tens of thousands of accounts*, not a private
binary.

**What "not distributed" does and does not change:**
- **Does:** removes the hosted-service/third-party-user dimension, removes the "circumvention at
  scale" narrative, and means there is nobody to send a takedown to. The practical worst case for a
  private tool is account-level enforcement, not litigation. `[INFERENCE, consistent with the
  shape of the Nitter action]`
- **Does not:** make the access permitted; does not protect the account; does not survive the tool
  becoming shareable (a public repo, a container image, an AUR package, a screenshot with a working
  cookie path, a "how I did it" blog post with the signer in it).

**How to avoid:**
- Write the contract down and keep it enforced: read-only, single account, no distribution
  artifacts, no hosted mode, cookie never leaves the machine.
- Add release hygiene to P8: no packaging for public registries, no Dockerfile that runs it, no
  published signer that turns "my script" into "infrastructure".
- Prefer paths that reduce exposure where they cost nothing: no write-auth path, no guest-token
  farming, no proxy pool, no account rotation — those are the Nitter ingredients.

**Warning signs:**
- Any step that makes the tool reproducible by a stranger: a public repo, a release tarball, a
  packaged image, a `--help` that explains how to extract cookies for *other people*.
- The project gaining any of Nitter's four ingredients: many accounts, proxies, a hosted endpoint,
  a public instance.
- Any write action appearing in the backlog "just to test".

**Phase to address:** P1 (write the contract into README + PROJECT.md out-of-scope), re-audited in P8.

---

### Pitfall 3: A naive HTTP client gets 403 or an empty 404 — and the developer blames the API

**What goes wrong:**
The first request from a fresh `net/http`/`reqwest`/`fetch` client with correct cookies returns
`403` (Cloudflare advanced bot detection) or a `404` with an empty body (request-signing layer).
The developer concludes the endpoint set is dead or that cookies are wrong, and burns days probing
payloads.

**Why it happens:**
Three independent layers must all line up, and each one fails with a *different, misleading*
status:

1. **TLS/HTTP fingerprint.** X's auth endpoints are Cloudflare-protected and reject non-browser
   ClientHellos. `the-convocation/twitter-scraper` ships a CycleTLS integration for exactly this:
   "Twitter's authentication endpoints may be protected by Cloudflare's advanced bot detection,
   resulting in `403 Forbidden` errors during login. This typically happens because standard
   Node.js TLS fingerprints are detected as non-browser clients." (`[HIGH]`,
   [README](https://github.com/the-convocation/twitter-scraper/blob/main/README.md))
2. **Request signing.** Every `/i/api/graphql/*` call now needs `x-client-transaction-id`; without
   it the WAF answers **`404` with `content-length: 0`** — while still decrementing the rate-limit
   counters, proving that auth was fine (`[HIGH]`,
   [twscrape#306](https://github.com/vladkens/twscrape/issues/306)).
3. **Header coherence.** `ct0` must be mirrored in `x-csrf-token`, the public bearer must be sent,
   and `x-twitter-auth-type: OAuth2Session` is required with `auth_token`. Hobenshield documents the
   conditions and concludes: "Any other combination or permutation of these headers will result in
   an error." (`[HIGH]`, [hobenshield.com](https://hobenshield.com/articles/web-api-re))

**Real vs cargo-culted mitigations:**

| Mitigation | Verdict | Evidence |
|---|---|---|
| Send `x-client-transaction-id` on every GraphQL call | **Real, mandatory** | 404-empty without it; still rate-counted ([#306](https://github.com/vladkens/twscrape/issues/306)) `[HIGH]` |
| Mirror `ct0` into `x-csrf-token`; use `x-twitter-auth-type: OAuth2Session` | **Real, mandatory** | [hobenshield](https://hobenshield.com/articles/web-api-re) `[HIGH]` |
| Browser-like TLS fingerprint (`curl_cffi`/CycleTLS/tls-client class) | **Real for 403/Cloudflare paths** | twscrape added a `curl-cffi` backend for "browser-like TLS fingerprinting" ([changelog v0.19.0](https://github.com/vladkens/twscrape/blob/main/changelog.md)); convocation ships CycleTLS `[HIGH]` |
| Set a normal browser `User-Agent` | **Real but insufficient alone** | hobenshield lists it as table stakes; TLS must match `[HIGH]` |
| Rotate proxies / IPs | **Mostly cargo cult here** | Nitter: "Using a proxy didn't fix it once the ratelimit had occurred" ([#1349](https://github.com/zedeus/nitter/issues/1349)); hobenshield: proxies are a last resort, find the root cause first `[HIGH]` |
| Rotate User-Agents per request | **Cargo cult** | The mismatch (browser UA, Go/JS TLS) is the tell; per-request UA churn adds nothing `[INFERENCE]` |
| Disable the transaction ID (`disableTid`-style) | **Cargo cult / temporary** | Nitter's owner: it "authenticates in a different less officially supported way, and only really counts as a temporary workaround" ([#1349](https://github.com/zedeus/nitter/issues/1349)) `[HIGH]` |
| Solve a CAPTCHA / drive a headless browser | **Last resort** | hobenshield: browser automation is "always a last resort … expensive and fragile"; x-native deliberately avoids it `[HIGH]` |

**How to avoid:**
- Treat the transport as one component with a *smoke test*: a `doctor` subcommand that performs one
  authenticated read and prints `status`, presence of `cf-ray`, `x-rate-limit-remaining`, and
  whether a `x-client-transaction-id` was attached.
- Decide the fingerprint strategy at P1, on evidence: either accept a Go `utls`/`tls-client` style
  impersonating client, or accept "works from my home IP, may 403 from datacenters" (x-native's
  explicit stance). Do not pretend a plain client is fine.
- Never log raw response bodies as the primary diagnosis — print the *classification* (Cloudflare /
  WAF-404 / auth / rate-limit / GraphQL error).

**Warning signs:**
- `403` with an HTML body or a `cf-ray` header → Cloudflare, not auth.
- `404` with `content-length: 0` on the GraphQL path → signing, not routing.
- Requests "succeeding" while `x-rate-limit-remaining` drops → they are being counted, so the
  problem is upstream of your URL.
- Login/guest endpoints 403 while reads work → fingerprint, specifically.

**Phase to address:** P1 (spike proves one authenticated read from the real stack), hardened in P2.

---

### Pitfall 4: The transaction-ID signer breaks silently on X's own malformed bundles

**What goes wrong:**
The signer is the most fragile dependency in the system, and when it breaks, *every* request
returns an empty 404. The failure looks identical to "X changed the API" and is easy to misdiagnose
as an endpoint problem.

**Why it happens:**
The header is derived from the web bundle: you fetch `x.com`, locate `ondemand.s`, extract
`KEY_BYTE` indices and animation data, and sign `(method, path)` with it.
([XClientTransaction](https://github.com/iSarabjitDhiman/XClientTransaction) `[HIGH]`;
[Lqm1/x-client-transaction-id](https://github.com/Lqm1/x-client-transaction-id) `[HIGH]`).
Every link in that chain is a churn surface, and the chain has failed in production repeatedly:

- **Dec 2025:** X shipped a manifest with **unquoted object keys** (e.g.
  `node_modules_pnpm_ws_8_18_0_node_modules_ws_browser_js: "475cbd6"`), so `json.loads` threw and
  every account was locked out with `Exception: Failed to parse scripts`. Users diagnosed it,
  patched it with a quote-fixing regex, and the workaround itself later broke because *more* keys
  became unquoted ([twscrape#280](https://github.com/vladkens/twscrape/issues/280)) `[HIGH]`.
- **Aug 2026:** "Fixed X client transaction ID generation for builds using **16-character chunk
  hashes**" ([changelog v0.20.1](https://github.com/vladkens/twscrape/blob/main/changelog.md)) `[HIGH]`
  — i.e. a hash-length assumption in the parser.
- The community's own README warns about an extra failure mode: "Getting `Couldn't get KEY_BYTE
  indices` error? Try passing the original response or the response text"
  ([XClientTransaction](https://github.com/iSarabjitDhiman/XClientTransaction)) `[HIGH]`.
- Even the code comments encode a workaround specific to this layer: "if code 404 on first try then
  generate new x-client-transaction-id and retry"
  ([twscrape `queue_client.py`](https://github.com/vladkens/twscrape/blob/main/twscrape/queue_client.py)) `[HIGH]`.

**How to avoid:**
- Isolate it: `type Signer interface { Sign(method, path string) (string, error) }`, one
  implementation, one entry point, no other code knows the header exists.
- Make it self-verifying: sign a known path at startup and assert non-empty + expected length +
  charset. A broken signer must fail *loudly* on start, with the parse stage named.
- Cache the last-known-good bundle/key material on disk (this is not timeline data; it is public
  web assets) so a parse regression does not take the whole client down, and add a TTL.
- Be tolerant of X's malformed output: parse defensively (the quote-fixing regex is the shipped
  precedent), never `json.Unmarshal`-and-panic on an X-authored blob, and report "bundle schema
  changed" as its own error class rather than "network error".
- Port a documented implementation instead of re-deriving it: the reverse-engineering write-up
  series is the reference (`antibot.blog` parts 1–3, linked from
  [XClientTransaction](https://github.com/iSarabjitDhiman/XClientTransaction)) `[MEDIUM — antibot.blog
  returned HTTP 522 when I tried; cited via the library that links it]`, and two MIT-licensed
  ports exist in TypeScript and Python to cross-check against.

**Warning signs:**
- All endpoints start 404-ing at once, with no query-ID change.
- `Failed to parse scripts` / "couldn't get KEY_BYTE indices" / empty signer output.
- Latency per request jumps by seconds (signer fetching bundles per call instead of once).
- The signer works from a fresh process but not after a long-running session (stale cached bundle).

**Phase to address:** P2 (signer + `doctor`), re-tested in P8.

---

### Pitfall 5: queryId, features blobs and response shapes rotate on different clocks

**What goes wrong:**
Three things churn independently — the `queryId` hash, the `features`/`fieldToggles` JSON blobs,
and the response nesting. A stale `queryId` does **not** produce a clear error: it produces
**HTTP 404 with an empty body**, which a naive client renders as "no posts". The user sees an empty
timeline and blames the client; the developer sees no error and blames the network.

**Why it happens:**
X rotates operation IDs continuously and publishes nothing. Evidence:

- `x-native`: "query IDs that **rotate every few weeks**", and its self-heal is explicitly
  partial — a scraped ID "can be a build behind the live API and return **HTTP 404**", and
  lazy-loaded operations (`Bookmarks`) "aren't in those bundles at all" but in per-route chunks
  behind an inlined manifest of ~940 opaque numeric chunks ([README](https://github.com/jpoindexter/x-native)) `[HIGH]`.
- twscrape ships operation-ID updates in **four consecutive releases** (v0.19.1 2026-06-26,
  v0.19.2 2026-07-21, v0.20.0 2026-08-07, v0.20.1 2026-08-25 — each with "Updated GraphQL operation
  IDs for current X API compatibility") ([changelog](https://github.com/vladkens/twscrape/blob/main/changelog.md)) `[HIGH]`.
- It also needed a *features* fix: `withSafetyModeUserFields` was replaced by `withGrokTranslatedBio`
  and a new `fieldToggles` parameter became required — "Browser uses a different features set than
  twscrape's defaults" ([#280](https://github.com/vladkens/twscrape/issues/280)) `[HIGH]`. v0.20.0
  added `GqlFeaturesOutdatedError` for this class of failure.
- Response *structure* also moves: "they've also restructured their API responses and nested
  screen_names differently… a confluence of a change in GraphQL endpoints, a diff in the response
  structure, and the [malformed-JSON] bug" (same thread) `[HIGH]`.
- Failure taxonomy observed in the wild (Bird CLI's troubleshooting table): `404` on search →
  stale query ID; `422 GraphQL validation` → wrong query ID/features; `401` → bad cookie
  ([bird-x-cli](https://github.com/0xEnc0der/bird-x-cli)) `[HIGH]`.

**How to avoid:** (the discipline that keeps the fix one line)
- **One registry, one file, machine-writable.** `operationName → {queryId, features, fieldToggles}`
  in a single generated file delimited by markers (twscrape's `# GQL_OPS_CODEGEN` block is the
  precedent), regenerated by a script; **no `queryId` string literal anywhere else in the repo.**
- **Auto-heal first, pin second.** Port the bundle-scraping refresher (two levels deep, per
  [twscrape's `update-gql-ops.py`](https://github.com/vladkens/twscrape/blob/main/scripts/update-gql-ops.py) `[HIGH]`)
  and expose `x-terminal doctor --refresh-ids`; *plus* a manual pin override per operation
  (`OP_HOME_TIMELINE=<id>`), because bookmarks/likes are exactly the ops the scraper cannot see
  `[HIGH]`.
- **Fail loudly and specifically.** Classify responses before parsing: `404+empty` → "query ID
  stale for op X"; `422`/`errors[]` → "features blob stale for op X"; `200` → check `errors[]`
  before `data`. Never let a missing `data` field degrade to an empty list.
- **Fixture every endpoint** and keep the parser tolerant: missing fields → zero values; unknown
  unions/`__typename` → keep a raw copy of the JSON for offline inspection. twscrape's
  `tests/mocked-data/_issue_*` convention is a good precedent for regression fixtures `[HIGH]`.

**Warning signs:**
- Any endpoint returning an empty list *without* an error, especially the timeline.
- A fix that requires touching more than one file.
- A `422`, or a `200` whose body has `errors` but no `data`.
- The web client working while the tool returns empty for the same endpoint (the browser is using
  a newer ID — that is the whole diagnosis).

**Phase to address:** P2 (registry + refresher + classifier); P5 and **P6 especially** for the
pin-override path.

---

### Pitfall 6: The session silently invalidates, and the failure looks like an API problem

**What goes wrong:**
`auth_token` expires (or the user re-logs-in on the web), `ct0` goes stale, or a mismatched pair is
assembled from two sessions. The client then gets a mix of 401/403/404-shaped errors and partial
results, and the developer re-examines the whole endpoint layer instead of the credential.

**Why it happens:**
The two cookies have different lifetimes and different jobs:

- `auth_token` is the long-lived session bearer; in practice it is a 40-character hex value, and
  anything that mangles it (see Pitfall 7) produces `HTTP 401` on reads and writes
  ([bird-x-cli troubleshooting](https://github.com/0xEnc0der/bird-x-cli)) `[HIGH]`.
- `ct0` is the CSRF token and **must equal the `x-csrf-token` header**; "ct0 is a short lived cookie,
  you can regenerate it as long as you've the long lived auth_token"
  ([twitter-api-client#102](https://github.com/trevorhobenshield/twitter-api-client/issues/102)) `[HIGH]`.
  Mismatched pairs are one of hobenshield's documented failure permutations `[HIGH]`.
- Libraries that *can* log in again tell you not to: "Repeated logins are closely monitored by
  Twitter and can quickly trigger security alerts. The act of calling the `login()` method
  repeatedly should be avoided at all costs. … especially from different IPs"
  ([Twikit](https://d60-twikit.mintlify.app/advanced/account-protection)) `[HIGH]`.
- Cookie expiry is expected, not exceptional: "Cookies expire over time. If authentication fails,
  you may need to export fresh cookies from your browser"
  ([convocation README](https://github.com/the-convocation/twitter-scraper/blob/main/README.md)) `[HIGH]`,
  and twscrape added `add_cookie` refresh support in v0.20.0 for the same reason `[HIGH]`.

**Multi-device:** X monitors repeated logins and logins from different IPs `[HIGH, Twikit]`.
Whether *logging out of the web client* immediately kills a separately-copied `auth_token` is
`[INFERENCE]` — I found no primary source stating it — so the design must not depend on either
answer: treat "the browser session changed" as a plausible cause of tool-side invalidation.

**How to avoid:**
- The tool has **no password path at all**. It cannot log in; it can only read cookies. This makes
  "repeated logins" impossible by construction and is worth stating as a design feature.
- Startup probe: one cheap authenticated read; distinguish and print the *specific* reason —
  `auth_token missing`, `ct0 missing`, `ct0 does not match x-csrf-token`, `session expired
  (auth_token rejected)`, `network unreachable`. Never collapse these into "auth failed".
- Keep the pair together: store/load `auth_token` and `ct0` as one atomic credential, and refuse to
  load a half-populated one.
- Treat an expired session as a first-class UI state with the one-line fix (re-copy cookies from
  the same browser profile) rather than a generic error toast.
- Do not add automatic re-login, session refresh via browser automation, or keyring decryption in
  v1 — PROJECT.md already excludes browser-cookie extraction, and each of those multiplies the
  auth surface.

**Warning signs:**
- 401/403 that disappears after re-copying cookies (i.e. it was never an API problem).
- A `ct0` copied from an older DevTools session than the `auth_token`.
- The browser showing a verification/challenge prompt while the tool still "works".
- The same error code appearing across unrelated endpoints simultaneously.

**Phase to address:** P1 (atomic secret load), P2 (session probe + error taxonomy), P8 (UI state).

---

### Pitfall 7: The cookie leaks into logs, error strings, config, shell history, or the wrong host

**What goes wrong:**
`auth_token` is a bearer credential for the user's entire x.com session. It ends up in a debug log
line, an `Error()` chain, a config round-trip, shell history, or — a real shipped bug in a widely
used scraper — attached to HTTP requests to hosts that are not x.com.

**Why it happens:**
The credential is a 40-hex opaque string that looks like every other id, so generic machinery
(`fmt.Sprintf("%+v", cfg)`, request dumps, HTTP client logging, `fmt.Errorf("cookie: %s", c)`)
prints it without anyone intending to. Concretely:

- twscrape had to fix "X client transaction ID **cookies being sent outside `x.com`**"
  ([changelog v0.20.0](https://github.com/vladkens/twscrape/blob/main/changelog.md)) `[HIGH]` — a
  shared cookie jar plus asset-host requests equals third-party disclosure.
- Writing the credential with a shell heredoc **corrupts tokens containing special characters**;
  bird-x-cli documents this and recommends writing the env file with Python, then *verifying the
  auth_token is exactly 40 characters*. ([bird-x-cli](https://github.com/0xEnc0der/bird-x-cli)) `[HIGH]`
- PROJECT.md already flags the local precedent: the sibling project's config "round-trips bytes
  through a panel-editable file and has no secret-handling mechanism at all" — so a naive config
  write path will happily persist a cookie `[HIGH, project document]`.
- twscrape's CLI avoids secrets on the command line and instead "prompts securely for cookies"
  for the `add_cookie` flow ([readme](https://github.com/vladkens/twscrape)) `[HIGH]` — the accepted
  practice.

**How to avoid:**
- A `Secret` type in the credential package: unexported bytes, `String()`/`Format()`/`GoString()`
  → `[redacted]`, no JSON marshalling, no `%v` passthrough, and a unit test that asserts the value
  does not appear in `fmt.Sprintf("%v", v)` or in error output.
- Secret file: `0600`, outside the general config surface (PROJECT.md decision), validated at read
  time (`stat` mode check → refuse to run if group/world-readable or if it is a symlink into a
  synced dotfile directory), and never written back by the app. Env override documented as
  per-shell only — explicitly *not* for `~/.bashrc`.
- Host-scoped cookie jar: allow only `x.com`/`.x.com`; assert in a test that a request to
  `abs.twimg.com` carries no cookies. This is exactly the twscrape bug, and it is cheap to prevent.
- No request-dump/verbose-request flag in the shipped binary; if one is needed for diagnosis, it
  prints method/host/status/length only.
- Never accept the cookie as a CLI argument (history + `ps`); if a CLI helper is needed, read
  stdin or prompt.
- Scrub the cookie from any error surfaced by the HTTP layer: wrap errors so that a transport error
  string containing headers is never emitted verbatim. The `doctor` command prints *presence*
  (`auth_token: present (40 chars)`), never a prefix.

**Warning signs:**
- `grep -rn "auth_token"` finds anything outside the secret loader and its test.
- Any log line containing a 40-char hex string.
- `ls -l` on the credential file showing anything but `-rw-------`.
- `.gitignore` missing the credential path (or a sample file with a real token in it).
- `history | grep -i cookie` on the user's shell showing the export command.

**Phase to address:** P1 (type + file permissions + jar scoping), P8 (log audit, redaction test,
`grep` sweep as an acceptance check).

---

### Pitfall 8: Rate limits, soft bans, and the invisible 24-hour lock

**What goes wrong:**
The client trips a limiter, and instead of a clean 429 the user sees one of: a successful response
with fewer posts, a random mix of 200 and 429, an instant failure with no body, or a 24-hour
account-wide block that looks like "X is broken". Repeated retries make it worse.

**Why it happens:**
The internal API has (at least) two limiters plus enforcement heuristics, and its accounting is
unreliable:

- **Per-endpoint 15-minute windows** (Nitter, from `auth.nim`): search **50**, `TweetDetail` **150**,
  photo rail 180, and ≈**500** for the big user/list endpoints; the window "start[s] from the first
  request made with a given account" ([nitter#1107](https://github.com/zedeus/nitter/issues/1107)) `[HIGH]`.
- **A second, faster limiter**: "most of these endpoints have a second rate limit which kicks in if
  you make requests too rapidly. **Even 1 request per second is enough to hit it.** … will 'block' a
  guest account for the following 24 hours, and it's applied **across endpoints**, not just the one
  that got you blocked." (same source) `[HIGH]`
- **The opaque 24-hour limit** (2025-12): "This is related to the completely opaque 24 hour rate
  limit introduced within the last year or two." ([nitter#1349](https://github.com/zedeus/nitter/issues/1349)) `[HIGH]`
  That thread also documents the honest bad news: a *random mix* of 429 and 200 responses while the
  reported budget still allows requests, from two different IPs, and the body of a failed call is
  exactly `{"errors":[{"code":88,"message":"Rate limit exceeded."}]}` with HTTP 429 `[HIGH]`.
- **Soft-ban detection by other clients:** twscrape treats `"(88) Rate limit exceeded"` **while
  `x-rate-limit-remaining > 0`** as a ban signal and deactivates the account for that endpoint
  ([queue_client.py](https://github.com/vladkens/twscrape/blob/main/twscrape/queue_client.py)) `[HIGH]`;
  it reads `x-rate-limit-remaining`/`x-rate-limit-reset` for the normal case and locks until reset `[HIGH]`.
- **Waiting is normal:** the JS client's default strategy "waits for the current rate-limiting
  period to expire before resuming requests. This has been known to take a very long time, in some
  cases (up to 13 minutes)." ([convocation README](https://github.com/the-convocation/twitter-scraper/blob/main/README.md)) `[HIGH]`
- **Load shedding is a different signal:** X also sheds load (retryable failures), which clients
  must not conflate with rate limiting — twscrape added "load-shedding retries to avoid unnecessary
  account rotation" and per-account retry caps in v0.20.0 `[HIGH]`.

For contrast, the *official* X API publishes per-endpoint limits and `x-rate-limit-*` headers
([docs.x.com](https://docs.x.com/x-api/fundamentals/rate-limits)) — the undocumented endpoint set
has no such contract, which is precisely why it must be treated conservatively `[HIGH]`.

**Practical budget for this app:** one manual refresh = 1–2 GraphQL calls; opening a post = 1 more;
expanding a thread = 1 page per "show more". A realistic heavy session is 20–60 calls. That is
inside the 15-minute buckets but *must* be serialised and spaced, because the second limiter
triggers on burstiness, not volume.

**How to avoid:**
- A single-flight request queue with a **minimum inter-request gap** (start at ~1 request per
  1–2 seconds; 1 req/s is already documented as enough to trip the fast limiter, so treat the gap as
  a floor, not a target).
- **Account for every response**, including failures: read `x-rate-limit-remaining`/`-reset` per
  endpoint and keep a small ledger (in memory; persisting it is not timeline data, so disk is
  acceptable if you want it to survive restarts).
- On 429: **do not retry**. Compute the wait from `x-rate-limit-reset`, show `rate limited until
  14:32`, and disable the refresh key until then.
- On `code 88` with budget remaining: surface the **soft-ban** message and stop issuing requests
  for the session — that is the account telling you to go away, and it is not a bug in the client.
- Treat 5xx/load-shedding with a bounded exponential backoff and a distinct message, never as a
  rate limit.
- Debounce the refresh key (ignore repeats while a request is in flight) so a key-repeat cannot
  produce a burst.
- Never parallelise media metadata, link expansion, or avatar fetches. One at a time, or don't.

**Warning signs:**
- Requests getting slower before they start failing.
- 429s from a *single* refresh press.
- Empty timelines that are not explained by a stale query ID.
- `x-rate-limit-remaining` dropping during failed calls (the WAF path still counts them).
- The web client degrading in parallel — that is account-level, not client-level.

**Phase to address:** P2 (ledger + accounting), P3 (refresh debounce/serialise), P8 (surfaces +
runbook + "stop" behaviour).

---

### Pitfall 9: Grapheme-cluster width mismatch corrupts the layout (emoji, ZWJ, CJK)

**What goes wrong:**
Posts and bios contain emoji, ZWJ sequences, CJK, flags, or combining marks. The client computes a
width that disagrees with what the terminal draws, so columns drift, borders smear, and — in the
worst case — writing "into" a wide character's spacer cell makes the terminal **erase the wide
character entirely**.

**Why it happens:**
"The width of a character" is not a single number the ecosystem agrees on:

- Mitchell Hashimoto's write-up (Ghostty's author) shows the farmer emoji `🧑‍🌾` advancing the
  cursor by **2, 4, 5, or 6 cells** depending on the terminal, and concludes: "**What you should not
  do is assume `wcwidth` behavior *or* grapheme clustering behavior.**"
  ([mitchellh.com](https://mitchellh.com/writing/grapheme-clusters-in-terminals)) `[HIGH]`
- Ghostty defaults to **grapheme clustering** (`grapheme-width-method = unicode`) while most
  applications use `wcswidth`; when the two disagree, "writes to the spacer tail is erasing the
  full grapheme". Ghostty's own issue documents the mechanism step by step and notes the workaround
  (`grapheme-width-method = wcswidth`) has other downsides: "Without mode 2027, there is really no
  guarantee in how anything will behave."
  ([ghostty#1695](https://github.com/ghostty-org/ghostty/issues/1695)) `[HIGH]`
- The classic Go width library cannot solve it: "I'm sure that you cannot simply add up the widths
  of individual runes ('🏳️‍🌈' would then have a width of 4 which is obviously wrong)"
  ([go-runewidth#59](https://github.com/mattn/go-runewidth/issues/59)) `[HIGH]`;
  and locale changes even box-drawing widths ([go-runewidth#64](https://github.com/mattn/go-runewidth/issues/64)) `[HIGH]`.
- Under a multiplexer the mismatch is *structural*: "tmux uses wcwidth … But tmux itself lives in a
  terminal emulator which also sees the value when printed. If tmux and the terminal don't match,
  the cursors become out of sync" ([mitchellh.com](https://mitchellh.com/writing/grapheme-clusters-in-terminals)) `[HIGH]`.
  This host runs **Ghostty 1.3.1** and `tmux` 3.7c — the exact combination that disagrees.

**How to avoid:**
- Pick **one** width model and route every wrap, pad, truncate, and cursor-move decision through it.
  In this stack, `charmbracelet/x/ansi` exposes both: `StringWidth` ("treats the text as a sequence
  of grapheme clusters") and `StringWidthWc` (wcwidth semantics), plus grapheme-aware `Truncate`,
  `Cut`, and `ByteToGraphemeRange`
  ([source](https://github.com/charmbracelet/x/blob/main/ansi/width.go),
  [truncate.go](https://github.com/charmbracelet/x/blob/main/ansi/truncate.go)) `[HIGH]`.
  Choose deliberately and document why.
- Under `$TMUX`, prefer the model tmux itself uses (wcwidth) so the inner and outer grids agree;
  standalone in Ghostty, grapheme semantics match the outer terminal. `[INFERENCE from the two
  sources above — verify empirically with golden frames in both contexts.]`
- **Avoid the problem where cheap**: this project's style is unbranded/plain text. Replace emoji with
  ASCII markers in the renderer by default (`[media]`, `[video]`, `->`), keep a toggle for raw text,
  and strip/replace ZWJ sequences and variation selectors in *display* only (never in the stored
  text used for search/copy).
- Truncate with the same function that measures, and truncate on grapheme boundaries (never by
  byte or rune count).
- Test with a fixture corpus: ZWJ emoji, flags, skin-tone modifiers, CJK, Arabic, combining
  diacritics, box-drawing — at 80 and 120 columns, inside tmux and outside, on both a dark and a
  light Ghostty theme. Golden frames (the sibling repo's convention) are the natural mechanism.

**Warning signs:**
- Any drift in a golden frame that contains non-ASCII text.
- Two different width helpers in the codebase (`len()`, `utf8.RuneCountInString`, `ansi.StringWidth`)
  used for the same job.
- Layout that is correct in the author's terminal but wrong in a tmux pane (or vice versa).
- Truncated posts ending mid-grapheme (a stray ZWJ or a broken flag).

**Phase to address:** P3 (renderer + golden frames), re-checked in P4/P5/P6 wherever new text arrives.

---

### Pitfall 10: In-terminal images under a multiplexer are a layout hazard, not a feature flag

**What goes wrong:**
Kitty-protocol image rendering "works" during development and then either (a) does nothing at all
inside tmux, with no error, or (b) renders and then destroys the layout: text printed afterwards is
drawn *across* the image, and scrolling/resizing/splitting smears pixels.

**Why it happens:**
Terminal multiplexers do not model image cells.

Empirical probe on this host (`[HIGH]`, local evidence:
[`TERMINAL-MEDIA-PROBE.md`](./TERMINAL-MEDIA-PROBE.md), screenshots in `evidence/probe-{A,B,C}.png`,
Ghostty 1.3.1 / tmux 3.7c with `allow-passthrough on`):

| Probe | Setup | Result |
|---|---|---|
| A | Direct Ghostty, bare Kitty APC | **Works**; image scaled/interpolated by default; `KITTY_WINDOW_ID` is **not** exported |
| B | Inside tmux, bare APC (`allow-passthrough on`) | **Silently fails** — no image, no error, nothing to catch |
| C | Inside tmux, DCS-wrapped APC | **Renders, then text is drawn across the image**; tmux's grid still believes those cells are empty |

The upstream maintainer's own warning matches: wrapping the escape codes in tmux passthrough gets
you "basic image display, though of course anything like changing the window layout in tmux or
resizing the terminal or scrolling will gum things up royally" — and the cell-aware placement
proposal that would fix it is still an unimplemented discussion from 2021
([kitty#4021](https://github.com/kovidgoyal/kitty/discussions/4021)) `[HIGH]`.

**How to avoid:**
- Keep v1 exactly as decided: media opens externally (Pitfall 13 covers the opener plumbing).
- Any future in-terminal renderer must, at minimum: detect capability from `TERM_PROGRAM`/`TERM`
  (never `KITTY_WINDOW_ID` — Ghostty does not export it `[HIGH]`); DCS-wrap when `$TMUX` is set;
  **reserve the exact cell rectangle and force a full repaint afterwards**; request an explicit
  placement size (Ghostty interpolates by default); and confine images to a **dedicated detail
  view** that never scrolls with a list.
- Always pair it with a user-visible fallback line ("graphics not available in this pane; opening
  in mpv"), because the primary failure mode is a silent no-op — there is nothing to catch.

**Warning signs:**
- An "image rendered" code path with no branch for `$TMUX`.
- Stale pixels after the next keypress; text drawn through a picture.
- Any graphics escape emitted while a scrollable list is on screen.
- A capability check that reads `KITTY_WINDOW_ID` or `TERM=xterm-kitty`.

**Phase to address:** P7 (media), with the v1 default already resolved in P1/P3 (external opener,
abstracted behind the renderer interface).

---

### Pitfall 11: Pagination, dedup and ordering bugs that make the feed feel broken

**What goes wrong:**
The same post appears twice after a refresh; "load older" stops early or loops forever; posts go
missing; the list shows a jumble that is neither chronological nor consistent. The user concludes
the client is unreliable and stops using it — the single most common way a personal feed reader
dies.

**Why it happens:**
X's timeline payloads are built for a ranked, ad-laden web client, not for an append-only reader:

- Cursors are opaque and easy to reuse wrongly: the standard key is
  `{cursorType: "Bottom", value: "..."}` for a timeline, with `ShowMoreThreads` for reply threads
  ([twscrape `api.py`](https://github.com/vladkens/twscrape/blob/main/twscrape/api.py)) `[HIGH]`.
- Repeated cursors cause loops: v0.20.0 fixed "pagination loops caused by repeated cursors or
  pages"; twscrape even carries a `_is_stalled()` guard that keys on the (cursor, entry-id) tuple `[HIGH]`.
- Empty pages are *normal* (promoted/filtered entries) and must not end the iteration: "Fixed
  pagination to continue past empty pages (#265, #247)", with a tolerance of a few empty pages
  before giving up `[HIGH]`.
- Retweets surface as standalone entries: "Fixed retweeted originals appearing as **duplicate
  top-level tweets** in timelines (#310, #311)" `[HIGH]`.
- The payload contains non-post entries that must be filtered by `entryId` prefix
  (`cursor-`, `messageprompt-`, `module-`, `who-to-follow-`) or they appear as blank rows `[HIGH]`.
- `count` is **not** a page size: "`limit` is the target number of parsed objects, not a page size.
  X/Twitter controls page size per endpoint, so a call can return fewer or more objects than
  requested." ([twscrape readme](https://github.com/vladkens/twscrape)) `[HIGH]`
- The Following timeline is **ranked**, so "ordering by recency" is a choice, not a property
  (PROJECT.md scopes "a way to reach the algorithmic feed" separately from Following) `[HIGH, project document]`.
- Errors silently dropping data: v0.20.0 also fixed "API error throttling dropping response data" `[HIGH]`.

**How to avoid:**
- Model the timeline as an append-only list keyed by `rest_id`, with a `seen` set and retweet
  normalisation to the original id; on refresh, **merge new items at the top** rather than replacing
  the list, so scroll position and identity survive.
- Store both cursors: top (for "new posts since") and bottom (for "older"). Refresh uses the top
  cursor; never re-use a bottom cursor that has already been consumed.
- Treat an empty page with a valid cursor as "keep going" up to a small budget (3 is the shipped
  precedent), then stop with a distinguishable "no more results" state.
- Detect stalls on the (cursor, first-entry-id) tuple, not on page count.
- Keep page sizes at X's defaults; iterate until the user's requested target is met or the cursor
  runs out.
- Never sort by id and call it recency; if X gives rank order, present it as-is and label it, so
  "missing posts" is not misdiagnosed.
- Do not discard the raw JSON on parse failure — keep it (in memory, or in a debug dump the user
  triggers) so a pagination bug is diagnosable the next day.

**Warning signs:**
- The same post visible twice after refresh.
- "End of timeline" while a bottom cursor still exists.
- Scroll position jumping after a refresh.
- Older posts appearing above newer ones, or a page that returns the same first entry as the
  previous page.
- Any place in the code that sorts posts.

**Phase to address:** P3 (timeline), reused verbatim in P4/P5/P6.

---

### Pitfall 12: Scope drift — the "while I'm in here" spiral that sinks personal tools

**What goes wrong:**
A read-only viewer accretes write actions, notifications, an offline cache, a plugin system, and
multi-account rotation. Each addition brings auth surface, state, and maintenance; the project
either rots (twterm) or becomes the next Nitter (distribution + scraping infrastructure + legal
target). Either way the *original* job — read the feed quietly — stops working.

**Why it happens:**
Personal tools have no external forcing function: no users demanding features, no release train,
no code review. The forcing function is missing *subtraction*, and every tempting addition looks
cheap because it is only one more endpoint. Prior art shows the two failure shapes:

- **Rot, from over-scoping the client:** twterm is the only real TUI precedent, is "full-featured"
  (including posting), requires a Ruby `< 3` toolchain plus ncurses/Readline/Libidn, and is dormant
  ([twterm](https://github.com/ryota-ka/twterm)) `[HIGH]`. n0madic's Go frontend-API scraper — the
  ancestor of the TypeScript port — was archived by its owner on 2023-11-30
  ([repo banner](https://github.com/n0madic/twitter-scraper)) `[HIGH]`.
- **Escalation, from over-scoping the access:** Nitter went from an anonymous reader to a service
  that needed "20k-50k guest accounts" ([nitter#1107](https://github.com/zedeus/nitter/issues/1107)) `[HIGH]`,
  and ended with a cease-and-desist that named scraped session tokens `[HIGH, TechCrunch]`.
- **The maintenance floor is real and permanent:** the JS client's README warns "Twitter's
  authentication requirements and frontend API endpoints change frequently, breaking this library.
  Fixes for these issues typically take at least a few days to go out."
  ([convocation](https://github.com/the-convocation/twitter-scraper/blob/main/README.md)) `[HIGH]`;
  twscrape's recent releases are dominated by operation-ID refreshes and signer repairs `[HIGH]`.

**How to avoid:**
- Hold PROJECT.md's Out-of-Scope list as a **contract with tests where cheap**: a test (or CI grep)
  asserting no `POST` to `/i/api/`, no `time.Ticker`/cron in the UI layer, exactly one persistence
  store, and no second-account config key. The sibling repo's import-boundary test is the in-repo
  precedent for mechanising this `[HIGH, project document]`.
- Budget the maintenance explicitly: assume a signer/ID repair is needed within weeks, not months,
  and make that repair a *one-line* change (Pitfall 5) plus a `doctor` run.
- Refuse the four Nitter ingredients by policy: many accounts, proxies, a hosted endpoint, a public
  instance. Refuse writes for the same reason the PROJECT.md gives — the write-auth path adds
  exposure for zero read value.
- Prefer deleting a feature to adding one; a personal tool's feature list should shrink toward the
  core value ("the feed without the browser").
- No plugin system, no config DSL, no theme engine. Config surface = cookie location, safe-mode cap,
  and (later) one media-opener override.

**Warning signs:**
- A new config key that is not about credentials or safety.
- Any background timer, scheduler, or "check for new posts while idle".
- A second storage file appearing.
- Backlog items phrased "while I'm in here…".
- Feature work outnumbering repair work in the commit log.

**Phase to address:** P1 (write the contract), enforced every phase, audited in P8.

---

### Pitfall 13: Terminal-assumption traps — dumb/non-tty, colour, clipboard, external openers

**What goes wrong:**
The TUI is launched somewhere it cannot render (a pipe, a script, an SSH-ish/non-tty context),
produces escape-sequence garbage or blocks waiting for input; the palette looks fine on the
author's dark theme and is illegible on the office's light theme; "copy link" and "open media"
silently do nothing because the wrong clipboard/opener tool was assumed.

**Why it happens:**
Environment variables are unreliable proxies for capability, and this host has specific facts that
differ from the common default assumptions. Verified on this machine `[HIGH, host probes this run]`:

- Ghostty 1.3.1 sets `TERM=xterm-ghostty` and `TERM_PROGRAM=ghostty`, and does **not** export
  `KITTY_WINDOW_ID` (`TERMINAL-MEDIA-PROBE.md`, probe A).
- A **non-interactive shell on this host reports `TERM=dumb`, `NO_COLOR=1`, `XDG_SESSION_TYPE=tty`** —
  so every automated run, test harness, or `ssh`-style invocation exercises the degraded path. If
  that path is accidental rather than designed, the degraded path is garbage.
- Clipboard/opener inventory: `wl-copy` and `xdg-open` are **present**; `xclip`/`xsel` are **absent**;
  `xdg-desktop-portal-hyprland` is installed (portals: `hyprland.portal`, `gtk.portal`, …); `mpv` and
  `ffmpeg` are present; `chafa`/`viu`/`timg` are **absent** `[HIGH, host probe]`.
- The environment is not a contract anyway: `NO_COLOR` is an explicit, widely adopted informal
  standard ("when present and not an empty string … prevents the addition of ANSI color",
  [no-color.org](https://no-color.org/), updated 2026-10-12) `[HIGH]`.

**How to avoid:**
- Refuse to start the interactive UI unless stdout (and stdin) are TTYs; in a pipe, print the
  plain-text rendering (or an error) and exit 0/1 cleanly. Never emit raw cursor-control sequences
  into a pipe.
- Honour `NO_COLOR` and `TERM=dumb` by emitting plain text; detect colour depth from terminfo /
  `COLORTERM` but design the palette to survive a 16-colour terminal, and to be legible on both
  light and dark backgrounds (the office-legibility requirement in PROJECT.md is a design
  constraint, not a colour-count problem). Prefer attributes (reverse/bold) over saturated hues.
- External open: `xdg-open` for URLs/media (portal-backed under Hyprland here), with the URL printed
  in the status line as the always-available fallback; never hard-code a single tool, and never
  require an uninstalled one (`chafa`/`viu`/`timg` are absent on this host).
- Clipboard: prefer OSC 52 (works regardless of session type and across tmux) and fall back to
  `wl-copy` under Wayland; do not shell out to `xclip`/`xsel` (absent here).
- Resize: treat `tea.WindowSizeMsg` as the single source of truth (Bubble Tea sends it once
  initially and on every resize: `WindowSizeMsg{Width,Height}`
  ([docs](https://github.com/charmbracelet/bubbletea/blob/main/screen.go)) `[HIGH]`), re-wrap from the
  model rather than the rendered string, and make the golden-frame tests parameterised by width so
  a tmux split (half-width) is a test case, not a discovery.

**Warning signs:**
- Escape sequences in pipe output; the program hanging when run from a script.
- A screenshot that looks right only on the author's theme.
- "Copy" that reports success but leaves the clipboard unchanged.
- Any resize/split producing stale rows, duplicated lines, or a frozen region.
- An external-open path that depends on a tool not in the host inventory.

**Phase to address:** P1 (tty/plumbing decisions), P3 (renderer honours `NO_COLOR`/`dumb`, resize),
P7 (openers and clipboard fallback chain).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|---|---|---|---|
| `queryId` string literals at call sites | First call shipped a day sooner | Churn becomes a multi-file edit; the 404-empty failure shape hides the cause | **Never** — one generated registry file |
| Skip the transaction-ID signer, "cookies are enough" | Fewer moving parts in the spike | Every request 404s within weeks; misdiagnosed as an API change | **Never** |
| Put the cookie in the normal config file | One file to edit; simple round-trip | Leaks via dotfile sync, panel editors, `git`, bug reports; it is a bearer credential | **Never** (PROJECT.md already forbids it) |
| Treat an empty list as "no results" | Simpler error handling | Makes query-ID churn invisible and undiagnosable | Only if the internal error type still carries the classification |
| Compute widths with `len()`/rune counts | No width dependency | Alignment breaks on emoji/CJK; wide-char erasure under Ghostty | **Never** for user-visible text |
| Persist the timeline "just for scrollback" | Instant scroll, restores session | Violates the office/privacy requirement; adds a second data-at-rest surface | **Never in v1** (explicit PROJECT.md decision) |
| Auto-refresh timer | Feels alive | Turns a manual reader into a crawler → the ban surface | **Never** |
| Multi-account rotation "for safety" | Spreads rate limits | X discourages multi-account; it is the Nitter pattern that attracted enforcement | **Not for this project** |
| Headless-browser fallback "when it breaks" | Guaranteed path that always works today | Heavy, fragile, conspicuous, and adds a browser-automation dependency to a terminal tool | Only as an explicitly out-of-scope escape hatch the user runs manually |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|---|---|---|
| x.com GraphQL | Assuming HTTP 200 means success | Check `errors[]` in the body **before** reading `data`; a 200 can carry an error and no data |
| x.com GraphQL | Treating `404` as "endpoint removed" | `404` + `content-length: 0` on `/i/api/graphql/*` means stale query ID or a missing/invalid `x-client-transaction-id`; log `content-length`, `cf-ray`, `x-rate-limit-remaining` alongside the status |
| Cloudflare | Rotating proxies/UAs to clear 403 and 404s | Fix TLS fingerprint + header coherence first; proxies did not clear Nitter's 24-hour lock |
| Headers | Sending the `ct0` cookie without mirroring it into `x-csrf-token` (or mixing values from two sessions) | Mirror the value exactly; include `x-twitter-auth-type: OAuth2Session` with `auth_token`; treat the pair as atomic |
| Cookies | One global cookie jar for every host | Scope to `x.com`/`.x.com` and assert (in a test) that asset-host requests carry no cookies — twscrape shipped a fix for exactly this |
| Bundle scraping | Assuming the top-level bundles contain every operation | Lazy-loaded ops (bookmarks, and friends) live in per-route chunks; always provide a manual pin override |
| Pagination | Treating `count` as the page size | It is a *target*; X chooses page size. Keep defaults and iterate until the user's target or the cursor runs out |
| Timeline entries | Rendering every `entryId` | Filter `cursor-`, `messageprompt-`, `module-`, `who-to-follow-` prefixed entries, and normalise retweets (`retweeted_status_id`) so they are not duplicates |
| Rate limiting | Retrying on 429 | Read `x-rate-limit-reset`, show the wait, disable refresh until then; distinguish 429 / `code 88`-with-budget / 5xx load-shedding |
| External tools | Hard-coding one opener/clipboard binary | `xdg-open` for URLs, OSC 52 → `wl-copy` for clipboard, print the URL as the ultimate fallback |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|---|---|---|---|
| Fetching the signer's bundle chain on every request | Multi-second first-byte latency; a burst of asset requests that looks like scraping | Fetch once per process, cache with a TTL, reuse the last-known-good bundle | Immediately (first refresh) |
| Re-parsing the timeline JSON on every keystroke | Visible jank in a tmux pane | Parse once per page, build a render model, and treat rendering as pure formatting | At ~20 posts per page on a slow pane |
| Re-wrapping the whole buffer per frame | Scroll lag that grows with the buffer | Wrap once per post per width; invalidate only on resize or content change | A few hundred posts in memory |
| Fan-out prefetch (avatars, link metadata, media probes) | Invisible: it *works*, and quietly converts the tool into a crawler | Concurrency cap of 1–2, prefetch only what is on screen, never in the background | Immediately, in the risk dimension |
| Unbounded in-memory timeline growth | Rising RSS after hours of scrolling | Cap retained pages (the timeline is explicitly not persisted) | After hours of use |
| Blocking the UI on the network call | Frozen panes, unresponsive quit | Keep the pure-`Update`/`tea.Cmd` discipline the sibling repo already demonstrates | Any slow request |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---|---|---|
| Cookie in the general config surface / round-tripped by an editor | Full account takeover from a synced dotfile or a stray paste | Separate `0600` file + env override; `Secret` type; never write it back |
| Cookie in logs, `Error()` chains, or a request dump | Account compromise; leak into bug reports and transcripts | Redacting `String()`/`Format()`; a unit test asserting redaction; no verbose-request mode in the shipped binary |
| Cookie passed as a CLI argument | Shell history and `ps` leakage to any local user | Read from file/stdin or prompt (twscrape's `add_cookie` precedent) |
| World/group-readable secret file, or a symlink into a synced directory | Any local process reads the session | `stat`-check mode at startup and refuse; resolve the path and refuse symlinks out of `$HOME` |
| Cookie jar shared across hosts | Cookies sent to third-party asset hosts (a real shipped bug in a major scraper) | Host-scoped jar + a regression test asserting no cookies to non-x.com hosts |
| Real cookies committed as fixtures/samples | Permanent public leak | Synthetic fixtures only; `.gitignore` the credential path; `grep` sweep in CI |
| Printing a token prefix/length in the UI for "debugging" | Partial disclosure and false reassurance | Show `session: valid/invalid` only |
| Treating clipboard contents as ephemeral | Clipboard managers may persist what is copied (the "no trace" promise covers the timeline, not the clipboard) `[INFERENCE]` | Don't put the cookie or account-identifying content on the clipboard; prefer OSC 52 with an explicit key action |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---|---|---|
| Silent no-op for media open, clipboard, or session failure | User believes the tool is broken and distrusts everything else | Always print one status line: `image opened in mpv` / `no opener — URL printed` / `session expired: auth_token rejected` |
| Refresh replaces the list and loses position | "The feed feels broken" — the most common reason a personal reader is abandoned | Merge-and-dedupe by `rest_id`, keep scroll anchored to a stable post id |
| Re-ordering or "improving" the ranked feed | User cannot follow their actual Following timeline | Show X's order as-is and label it (ranked); don't sort |
| Silent truncation of long posts and threads | Reader thinks the post ended; misses the payoff | Explicit `… [open post]` affordance; thread continuation is a visible action, not an invisible cut |
| Borderless UI with no state at all | Cannot distinguish loading / throttled / stale-id / empty | No app chrome, but exactly one quiet status line — always |
| Colour palette that assumes a dark background | Illegible in a light office theme; the core "inconspicuous" goal fails | Test both themes; prefer reverse-video/bold over saturated colour; degrade to 16 colours |
| Emoji-driven decoration in the UI | Width bugs *and* conspicuousness (both project goals violated) | ASCII markers (`[img]`, `>>`, `·`) with an opt-in raw-text toggle |
| Raw JSON or HTTP codes shown as errors | Unreadable and unactionable | Map to `rate limited until 14:32` / `session expired` / `query id stale (op: HomeTimeline)` / `offline` |
| Requiring an uninstalled helper (`chafa`/`viu`/`timg`) for media | Feature appears broken on the user's own machine | External openers only (`mpv`, browser) + printed URL fallback |

---

## "Looks Done But Isn't" Checklist

- [ ] **Auth:** "works with cookies" usually means "works with *my* cookies, right now". Verify: missing `auth_token` → named error; missing `ct0` → named error; `ct0`/`x-csrf-token` mismatch → named error; expired session → "session invalid", never a `404` or an empty timeline.
- [ ] **Query IDs:** deliberately corrupt one `queryId` and confirm the client says "stale query ID for op X", not "no posts".
- [ ] **Signer:** delete/corrupt the cached bundle and confirm the failure is a *loud, named* error and that a stale-but-valid cached bundle still allows reads.
- [ ] **Pagination:** two refreshes in a row produce no duplicates; an empty page mid-stream continues; end-of-timeline is distinguishable from stale-ID; a repeated cursor does not loop.
- [ ] **Rate limits:** a simulated 429 prints the reset time and disables refresh; `code 88` with budget remaining prints the soft-ban message and stops; 5xx is retried with backoff and a different message.
- [ ] **Width:** emoji/ZWJ/CJK/combining-mark fixtures in posts and bios do not shift columns; golden frames pass at 80 and 120 columns, inside tmux and outside.
- [ ] **Terminal:** behaves under `NO_COLOR=1` and `TERM=dumb`; refuses to run in a pipe without emitting escapes; a tmux pane split reflows cleanly.
- [ ] **Media:** with the opener missing, it prints the URL; no graphics escape is emitted unless the dedicated detail view is active; no image is rendered while a scrollable list is on screen.
- [ ] **Secrets:** `grep` for the cookie across the repo finds only the secret loader and its test; the credential file is `0600`; `.gitignore` covers it; no cookie is sent to a non-x.com host.
- [ ] **Persistence:** the data directory contains **only** the bookmarks/likes store; nothing writes timeline data (verify by inspection after a full session, not by intent).
- [ ] **Degradation:** with the network off, the app says "offline" and keeps working with what it has (bookmarks/likes), rather than crashing or hanging.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---|---|---|
| Query ID rotation | LOW (minutes) | Run the ID refresh, re-pin the failing operation, add/refresh the response fixture, note the date in the runbook |
| Features-blob change (`422`, `errors[]`) | MEDIUM (hours) | Capture the browser's current `features`/`fieldToggles` for that op, update the registry entry, add the fixture |
| Signer / bundle-parse break | MEDIUM (hours–days) | Check upstream ports for a same-week fix; apply the defensively-patched parser (quote-fixing precedent); fall back to the cached bundle; if unfixable, degrade to "open in browser" mode rather than shipping a broken client |
| Expired/invalid session | LOW | Re-export `ct0` + `auth_token` from the same browser profile, reload, confirm the startup probe; if the browser shows a verification prompt, resolve it there first and stop the tool until resolved |
| 429 | LOW | Wait out `x-rate-limit-reset` (minutes to 15); do not retry in a loop; resume at a lower request ceiling |
| Soft ban / opaque 24-hour lock | HIGH (a day, on the account) | Stop the tool entirely for 24h; verify the account is healthy in the web client; on return, cut the per-session request cap and confirm no background activity exists |
| Account suspension | HIGH | Appeal through X's own process (outside this project's scope); treat the cookie path as no longer safe on that account; the tool must still start and say so `[INFERENCE]` |
| Cookie leak | HIGH | Log out of all sessions / change the password (invalidates the session), re-export to the `0600` file, audit logs and shell history, then re-run the redaction test |
| Width regression | LOW | Add the exact post text as a golden-frame fixture; fix the single width helper, not the call sites |
| Image smear inside tmux | LOW | Force a full repaint / leave the image view; consider disabling image mode whenever `$TMUX` is set |
| Accidental distribution | MEDIUM | Unpublish/archive the artifact, rotate the credential that was exposed, re-read Pitfall 2 before re-publishing anything |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---|---|---|
| 1. Account risk from cookie auth | P1 (risk statement, safe-mode cap) | README/first-run states the risk; a request counter caps a session; no concurrent requests exist in the code |
| 2. "Personal ≠ permitted" legal model | P1 (contract), P8 (audit) | Out-of-scope list has tests/greps; no distribution artifacts (image, package, hosted mode) in the repo |
| 3. Naive HTTP → 403/empty-404 | P1 (transport spike), P2 | `doctor` performs one authenticated read and classifies the result (Cloudflare / WAF-404 / auth / rate / GraphQL error) |
| 4. Signer breaks on malformed bundles | P2 | Startup self-test on the signer; deleting the cached bundle yields a named error; last-known-good bundle path exercised |
| 5. queryId/features/schema churn | P2 (+ P5/P6 for pin overrides) | Corrupting a query ID yields "stale ID for op X"; the fix is provably a one-file change; bookmarks have a documented pin override |
| 6. Session silently invalidates | P1 (atomic load), P2 (probe) | Missing/mismatched/expired credentials each produce a distinct, named message |
| 7. Cookie leakage | P1 (type, perms, jar scope), P8 (audit) | Redaction unit test; `0600` check; `grep` sweep; no cookies to non-x.com hosts |
| 8. Rate limits and soft bans | P2 (ledger), P3 (serialise + debounce), P8 (surfaces) | Simulated 429 → reset time shown, refresh disabled; `code 88` + budget → soft-ban message and halt; keys cannot be repeated into a burst |
| 9. Grapheme/width corruption | P3 | Golden frames at 80/120 cols with emoji/ZWJ/CJK fixtures, inside and outside tmux |
| 10. Images under tmux | P7 (v1 = external only) | No graphics escape is emitted with a list on screen; media always degrades to a printed URL |
| 11. Feed integrity (dedup/cursor/order) | P3 (reused in P4/P5/P6) | Double refresh produces no duplicates; empty-page continuation works; stall detection uses (cursor, entry-id); no sort by id anywhere |
| 12. Scope drift | P1 (contract), every phase, P8 (audit) | CI greps: no `POST /i/api/`, no ticker, one persistence store, one account key |
| 13. Terminal assumptions | P1 (tty), P3 (colour/resize), P7 (openers) | Runs (or refuses cleanly) in a pipe and under `TERM=dumb`; `NO_COLOR` honoured; resize reflows; opener fallback prints the URL |

---

## Feasibility Verdict (2026)

**Verdict: MAYBE — a conditional yes.** Confidence: MEDIUM (mechanisms HIGH; X's thresholds and
terms text MEDIUM).

Read-only cookie-authenticated x.com reading is *demonstrably possible today*: multiple maintained
projects read exactly the endpoint set this project needs — timeline, tweet detail/threads, search,
profiles, **bookmarks** — with the same two cookies (`[HIGH]`:
[twscrape](https://github.com/vladkens/twscrape) is actively shipping; `x-native`
[implements search + bookmarks + tweet reads](https://github.com/jpoindexter/x-native);
[bird-x-cli](https://github.com/0xEnc0der/bird-x-cli) covers read paths plus writes). The risk is not
feasibility; it is **survivability and honesty about the cost**.

The honest cost model:

1. **The endpoints work, but they break on someone else's schedule.** A signer or ID repair is a
   weeks-not-months event; one outage in Dec 2025 ran for days and required community patches
   ([twscrape#280](https://github.com/vladkens/twscrape/issues/280)) `[HIGH]`, and four consecutive
   twscrape releases in mid-2026 were largely operation-ID refreshes `[HIGH]`.
2. **The failure modes are silent by construction** — empty 404s, empty pages, success-with-less —
   so the build must invest in *diagnosis* (a `doctor`, a classifier, fixtures) rather than only in
   fetching. This is a real, non-optional phase of work.
3. **The account risk is real, unquantifiable, and falls on the user's own account.** Prior art
   says so in plain language (`[HIGH]`: convocation's ban warning, x-native's "scripted access can
   get accounts flagged", Twikit's "reduces but doesn't eliminate"). It can be minimised
   (serial, manual, low-volume, read-only, no background anything) but not eliminated.
4. **Enforcement now visibly targets scraping tooling** — the Nitter cease-and-desist and archive in
   Aug–Oct 2026 `[HIGH]` — although that action was aimed at a large distributed service, not a
   private binary. This raises the value of the non-distribution contract and of staying "boring
   and small".
5. **The project's own scope contains the mitigation**: read-only + manual refresh + no
   distribution + no persistence is, accidentally, close to the lowest-risk configuration possible.
   Preserve it and the answer stays yes; add writes, a poller, or a second account and it degrades
   toward the pattern that gets enforced.

**Conditions that make this a yes:**
1. Read-only, forever (no write path, no write-auth code).
2. Manual refresh only; single-flight requests; a session-wide request cap; no timers, ever.
3. Personal and undistributed — no public repo/binary/image/instance, no "how to" artifacts.
4. A first-class repair path (`doctor`: signer self-test, ID refresh, pin override, session probe,
   response classifier) budgeted as a phase, not a nice-to-have.
5. Accept that features can break for days; the UI must degrade to "open in browser" rather than
   pretend.
6. The account risk is stated to the user in the README and on first run; the user owns that risk
   consciously.
7. The two features that *require* the account (bookmarks, likes) are understood as the reason a
   burner account cannot be the answer — and therefore as the reason the risk cannot be engineered
   away.

**Prior art snapshot (why this is not a solved problem):**

| Project | Status (2026-10) | Lesson for this build |
|---|---|---|
| [Nitter](https://github.com/zedeus/nitter) | Dead: X cease-and-desist Aug 2026, repo archived 2026-10-09, XCancel offline `[HIGH]` | Distribution + scraping infrastructure is the enforcement trigger; stay private and small |
| [twscrape](https://github.com/vladkens/twscrape) | Active, 2.8k stars, Python CLI/lib, multi-account pool, per-endpoint locks `[HIGH]` | Proves the endpoint set and the accounting discipline; far more machinery than one user needs |
| [the-convocation/twitter-scraper](https://github.com/the-convocation/twitter-scraper) | Active port of an [archived Go original](https://github.com/n0madic/twitter-scraper) (archived 2023-11-30) `[HIGH]` | The clearest published risk statement and the CycleTLS/TLS-fingerprint fix |
| [x-native](https://github.com/jpoindexter/x-native) | New, same shape as this project (single account, read-only, cookie auth, self-healing IDs) `[HIGH]` | Closest analogue: confirms bookmarks need manual pinning and that plain `fetch` can 403 |
| [bird-x-cli](https://github.com/0xEnc0der/bird-x-cli) | Small fork, cookie CLI, includes writes `[HIGH]` | Concrete failure taxonomy (401/404/422) and the "don't write the secret with a heredoc" lesson |
| [twterm](https://github.com/ryota-ka/twterm) | Dormant Ruby/ncurses "full-featured" TUI incl. posting `[HIGH]` | The over-scoped TUI precedent: heavy toolchain, no maintenance, no current value |
| [twitter-api-v2](https://www.npmjs.com/package/twitter-api-v2) | Active, 1.29.1, ~403k weekly downloads, official API only `[HIGH]` | Confirms the official path is healthy but pay-per-use; out of scope by cost |

---

## Sources

**Primary — code and issues in maintained scrapers**
- twscrape README, changelog, `api.py`, `http.py`, `queue_client.py`, `scripts/update-gql-ops.py` — https://github.com/vladkens/twscrape `[HIGH]`
- twscrape issue #306 — "SearchTimeline returns HTTP 404 (empty body) → IndexError, even with fresh queryId" (2026-05-10) — https://github.com/vladkens/twscrape/issues/306 `[HIGH]`
- twscrape issue #280 — "Unknown Error" / `Failed to parse scripts`, malformed unquoted JSON keys, `withGrokTranslatedBio` features change (Dec 2025 – Jan 2026) — https://github.com/vladkens/twscrape/issues/280 `[HIGH]`
- the-convocation/twitter-scraper README — ban warning, rate-limit wait times, CycleTLS/Cloudflare, cookie auth — https://github.com/the-convocation/twitter-scraper/blob/main/README.md `[HIGH]`
- n0madic/twitter-scraper — archived Go original (2023-11-30), guest-token/`LoginOpenAccount` era — https://github.com/n0madic/twitter-scraper `[HIGH]`
- XClientTransaction (Python) — `ondemand.s`, `KEY_BYTE` indices, links to the antibot.blog RE series — https://github.com/iSarabjitDhiman/XClientTransaction `[HIGH]`
- Lqm1/x-client-transaction-id (TS) — `fetchXDocument`, `generateTransactionId(method, path)` — https://github.com/Lqm1/x-client-transaction-id `[HIGH]`
- x-native — "self-healing query IDs", the honest boundary of bundle scraping, bookmarks pinning, "dedicated/secondary account" — https://github.com/jpoindexter/x-native `[HIGH]`
- bird-x-cli — troubleshooting table (401/404/422), 40-char `auth_token`, env-file corruption — https://github.com/0xEnc0der/bird-x-cli `[HIGH]`
- Trevor Hobenshield, "API Reverse Engineering and Web Scraping" (2024-04-16) — JA3, the `ct0` ↔ `x-csrf-token` conditions, bearer/`x-twitter-auth-type`, proxies as last resort — https://hobenshield.com/articles/web-api-re `[HIGH]`
- twitter-api-client #102 — "ct0 is a short lived cookie, you can regenerate it as long as you've the long lived auth_token" — https://github.com/trevorhobenshield/twitter-api-client/issues/102 `[HIGH]`
- Twikit, "Account protection" — suspension-risk language, request-pacing guidance, "repeated logins are closely monitored" — https://d60-twikit.mintlify.app/advanced/account-protection `[HIGH]`
- antibot.blog "Twitter Header" parts 1–3 (deobfuscation / RE / generation) — referenced by XClientTransaction; **HTTP 522 when I attempted to fetch** — https://antibot.blog/posts/1741552025433 `[MEDIUM — cited via the library that links it; not read directly]`

**Primary — prior art, legal and enforcement**
- TechCrunch, "Nitter and XCancel are dead (again) after X's latest legal actions" (2026-09-15) — C&D over "unlawful use and circumvention", account/session-token scraping; reports the repo archive (TechCrunch says 2026-09-11; GitHub's own banner reads 2026-10-09 — dates disagree, outcome does not) — https://techcrunch.com/2026/09/15/nitter-and-xcancel-are-dead-again-after-xs-latest-legal-actions/ `[HIGH]`
- X Terms of Service (PDF dated 2026-04-10) — "You may not access the Services in any way other than through the currently available, published interfaces that we provide. For example, this means that you cannot scrape the Services…" — https://legal.x.com/content/dam/legal-twitter/site-assets/x-terms-of-service-2026-04-10/en/x-terms-of-service-2026-04-10.pdf `[MEDIUM — quoted from search-indexed copies of the official PDF; direct fetch timed out twice, and https://x.com/en/tos is JS-rendered]`
- Nitter tools/get_session.py — the guest-token flow (`/1.1/guest/activate.json`, public bearer, `X-Guest-Token`) that predates login requirements — https://github.com/zedeus/nitter/blob/74f5ff8a/tools/get_session.py `[HIGH]`
- Nitter issue #878 "New Restriction Requires Log-in" (May 2023) — the end of anonymous timeline reads — https://github.com/zedeus/nitter/issues/878 `[MEDIUM — title/summary level]`
- twterm — the dormant full-featured TUI precedent — https://github.com/ryota-ka/twterm `[HIGH]`
- twitter-api-v2 (npm) — official-API client, 1.29.1, ~403k weekly downloads — https://www.npmjs.com/package/twitter-api-v2 `[HIGH]`
- X API rate limits (official, for contrast) — https://docs.x.com/x-api/fundamentals/rate-limits `[HIGH]`

**Primary — internal rate limits (measurements, not documentation)**
- nitter#1107 (2023-11) — per-endpoint 15-minute budgets (search 50, `TweetDetail` 150, ≈500 others), the second limiter at ~1 req/s, the 24-hour cross-endpoint block — https://github.com/zedeus/nitter/issues/1107 `[HIGH]`
- nitter#1349 (2025-12) — "completely opaque 24 hour rate limit", random 429/200 mix with budget remaining, `{"errors":[{"code":88,...}]}`, proxies did not help, `disableTid` as a temporary workaround — https://github.com/zedeus/nitter/issues/1349 `[HIGH]`

**Primary — terminals, width, images**
- Mitchell Hashimoto, "Grapheme Clusters and Terminal Emulators" (2023-10-02) — the 2/4/5/6-cell table, mode 2027, "do not assume `wcwidth` **or** grapheme clustering", tmux's mismatch — https://mitchellh.com/writing/grapheme-clusters-in-terminals `[HIGH]`
- ghostty-org/ghostty#1695 — Ghostty computes width by grapheme clustering by default; wcswidth apps erase wide characters; `grapheme-width-method`; "Without mode 2027, there is really no guarantee" — https://github.com/ghostty-org/ghostty/issues/1695 `[HIGH]`
- mattn/go-runewidth#59 — the maintainer on why per-rune widths are wrong (🏳️🌈 = 4) — https://github.com/mattn/go-runewidth/issues/59 `[HIGH]`; #64 — locale-dependent box-drawing width — https://github.com/mattn/go-runewidth/issues/64 `[HIGH]`
- rivo/uniseg — UAX #29 segmentation + width, the Go reference for correct grapheme handling — https://pkg.go.dev/github.com/rivo/uniseg `[HIGH]`
- charmbracelet/x/ansi — `StringWidth` (grapheme clusters) vs `StringWidthWc` (wcwidth), grapheme-aware `Truncate`/`Cut`/`ByteToGraphemeRange` — https://github.com/charmbracelet/x/blob/main/ansi/width.go `[HIGH]`
- kovidgoyal/kitty discussion #4021 (2021) — image placement and tmux; the maintainer's "resizing … or scrolling will gum things up royally"; cell-aware placement still unimplemented — https://github.com/kovidgoyal/kitty/discussions/4021 `[HIGH]`
- Bubble Tea v2 — `WindowSizeMsg` semantics, resize contract, `tea.View` — https://github.com/charmbracelet/bubbletea/blob/main/screen.go `[HIGH]`
- NO_COLOR — the informal standard (page last updated 2026-10-12) — https://no-color.org/ `[HIGH]`

**Local empirical evidence (this machine, 2026-10-14)**
- `.planning/research/TERMINAL-MEDIA-PROBE.md` + `evidence/probe-{A,B,C}.png` — Kitty graphics direct vs tmux (bare = silent fail; DCS-wrapped = renders but text overlaps), `KITTY_WINDOW_ID` not exported by Ghostty 1.3.1 `[HIGH]`
- Host tool probe (this run) — `xdg-open`, `wl-copy`, `mpv`, `ffmpeg`, `bat` present; `xclip`, `xsel`, `chafa`, `viu`, `timg` absent; `xdg-desktop-portal-hyprland` installed; a non-interactive shell reports `TERM=dumb`, `NO_COLOR=1`, `XDG_SESSION_TYPE=tty` `[HIGH]`
- `.planning/PROJECT.md` — the project's own decisions (read-only, manual refresh, no timeline persistence, cookie outside the config surface, not distributed, sibling repo conventions) `[HIGH]`

---

*Pitfalls research for: a read-only, cookie-authenticated x.com TUI client (single user, undisclosed internal GraphQL endpoints)*
*Researched: 2026-10-14*
