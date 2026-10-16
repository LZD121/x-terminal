# Upstream reachability and request-shape probe — measured

Date: 2026-10-14, same host, after the user started their proxy client. ~8 requests total.
This file is evidence for Phase 1's premise; it is not documentation-derived.

## Proxy (as actually configured)

- **Clash Verge Rev / verge-mihomo listening on `127.0.0.1:7897`** — a *mixed* port: both
  `http://127.0.0.1:7897` and `socks5h://127.0.0.1:7897` returned 200 for `https://x.com/`.
  Note the port is **7897, not the commonly assumed 7890** — this is why the client must take
  its proxy from configuration rather than a hard-coded default.
- The GNOME system proxy is configured (host `127.0.0.1`, port `7897`) but its `mode` is
  `none`, and no `*_PROXY` variables were present in a non-interactive shell — so nothing
  picks the proxy up implicitly. The app must be told, or must read the standard variables
  when a user exports them.
- Egress appears to be Hong Kong: `cf-ray: a4a3a0bf990186e2-HKG`.

## Requests (plain curl through the proxy, no browser involved)

| Request | Result |
|---|---|
| `GET https://x.com/` via HTTP proxy | `200`, 41,434 bytes, 0.39 s |
| `GET https://x.com/` via SOCKS5 on the same port | `200`, 0.41 s |
| `POST https://api.x.com/1.1/guest/activate.json` with the web bearer | `200`, `{"guest_token":"2110216993752408238"}` |
| `GET https://x.com/i/api/graphql/<id>/UserByScreenName`, bearer only | **`403`, 0 bytes** |
| Same + a bogus `ct0` cookie and `x-csrf-token`, still no transaction-id | **`403`, 0 bytes** |

## What this settles, and what it changes

1. **The Phase-1 premise holds.** One real request path exists from this host, and it is
   proxy-configured rather than environment-magic. `NET-01`/`NET-02` are testable as written.
2. **Plain (non-browser) TLS is not challenged on the guest-token endpoint** — a vanilla
   client got a 200 with no Cloudflare interstitial. This is *partial* evidence for the open
   question in the roadmap ("whether plain Go TLS is challenged on authenticated reads"): the
   unauthenticated path passes, the authenticated path still needs the session cookie to test.
3. **Failure shapes are empty-bodied, and status code is the only signal.** Both
   unauthenticated GraphQL attempts returned `403` with a zero-byte body — no cf challenge
   page, no JSON error object. Research predicted a *stale query id* returns `404`-empty; the
   measured *unauthenticated/missing-signer* shape is `403`-empty. The classifier must key on
   status plus context, and an empty body must never be rendered as "no new posts"
   (`XAPI-06`, `NET-03`).

   **Caveat on attribution.** The query id used in those two GraphQL probes was *not* taken
   from a live snapshot, so the `403` proves only that the request was rejected *before* the
   operation was routed. It does **not** by itself separate "missing transaction-id /
   incoherent auth headers" from "unknown operation". Phase 1 must re-measure with an
   operation id from the pinned snapshot and a real session before the classifier's
   status-code table is treated as settled.
4. **Operation-id discovery is impossible unauthenticated — so the pinned snapshot is the
   primary source, not a fallback.** The shell HTML references
   `https://abs.twimg.com/x-web/x-web/entry-client-logged-out-BJDlp_4g.js`; that bundle is
   30,364 bytes and contains **zero** `queryId` pairs and no `graphql` reference at all — only
   lazily-imported locale and utility chunks (`assets/en-*.js`, `assets/fade-in-*.js`, …).
   x.com serves the *logged-out* entry to an unauthenticated client, and the operation map
   lives in chunks that only the authenticated app loads. Runtime scraping can therefore only
   ever be an increment on top of a pinned snapshot (`XAPI-02`, `XAPI-03`), which is exactly
   the shape the architecture research chose — now with evidence for why.
5. Media tooling is unchanged: `chafa`/`viu`/`timg` absent, `mpv`/`ffmpeg` present.

## Authenticated path — measured end to end, 2026-10-14 (later the same day)

The user's proxy was up and they asked for the session cookie to be taken from the browser
directly. Roughly thirteen requests in total; the whole path below was exercised with plain
`curl` — no browser automation, no TLS spoofing, no `x-client-transaction-id`.

### 1. Where the cookie lives, and what kind it is

Zen Browser's profile at `~/.config/zen/j5q0rheq.Default (release)/cookies.sqlite`
(Firefox schema, `moz_cookies`, values unencrypted). `.x.com` carries `auth_token` (40 chars),
`ct0` (160 chars) and `twid`.

**Both `auth_token` and `ct0` are *session* cookies — their `expiry` column is 0.** They die
with the browser session, so "cookie expired / not present" is a *routine* state this tool
must render calmly, not an error. That closes the roadmap's open question about `auth_token`
lifetime with a stronger answer than expected: there is no long-lived token to rely on.

Operational detail for anyone re-deriving the cookie: a `cookies.sqlite-wal` file existed and
was *newer* than the database, so a read with `immutable=1` is not safe in general. The two
reads happened to agree here, but the extraction method must be WAL-aware. (The application
itself never touches the browser's database — this is an operator-side step, as
`AUTH-01` specifies. It also does not change the Out-of-Scope ruling on browser-cookie
extraction *inside the product*.)

### 2. Request shape matters, and the failure is silent

| Request | Result |
|---|---|
| `GET https://x.com/home` + cookie **and** `authorization: Bearer …` | **`401`, 0 bytes** |
| Same, cookie only, with browser navigation headers | **`200`, 308,786 bytes** |
| `GET /i/api/1.1/account/verify_credentials.json` + full header set | `404` **with a JSON body** — the v1.1 API is retired |

A bearer header on a *navigation* request is actively rejected, while the identical cookies
on an API request are required. The header set has to match the request class, and the
failure is an empty 401 — the same silent-wrongness family as the rest of this upstream.

### 3. One real read, twice — and no signer needed

Both operations below used a `queryId` mined from the live bundle (below), the browser cookie,
`x-csrf-token` mirroring `ct0`, and the standard web bearer. **Neither sent
`x-client-transaction-id`.**

| Call | Result |
|---|---|
| `UserByScreenName` for `x` | `200`, 2,227 bytes, 0.38 s |
| `UserTweets` for `userId=783214`, `count=20` | `200`, 130,245 bytes, 0.81 s, **26 posts with `full_text`** |

Two consequences:

- **The signer is not universally required.** The research named `x-client-transaction-id`
  fragility as the project's top technical risk, and on this host, for these operations, from
  this client, it was not needed at all. That does not make it unnecessary everywhere — the
  per-endpoint requirement in the research stands as a hypothesis to re-test per operation —
  but it is no longer a prerequisite for the whole project, and the transport layer should not
  be built around it up front.
- **The response schema is the new one.** `UserByScreenName` returned `core` / `avatar` /
  `banner` / `id` and no `legacy.followers_count`; the research-era parsers that read `legacy`
  would silently return nothing useful. The fixture for the post list is checked in as
  `internal/xsource/testdata/fixture-usertweets.json` (with its provenance row).

### 4. The operation map is discoverable *only* when authenticated — and it ships its own feature switches

The authenticated shell references
`https://abs.twimg.com/responsive-web/client-web/main.a9b169a91ce3b55ba.js` (1,157,073 bytes).
It contains **104** records shaped like:

```
{queryId:"…",operationName:"…",operationType:"query",
 metadata:{featureSwitches:[…], fieldToggles:[…]}}
```

So one artifact supplies both the operation ids **and** the exact feature switches and field
toggles each operation wants — which is the "features blob drift / `(336)`" failure mode from
the pitfalls research, answered by the same file. Extracted to `x-ops-snapshot.json`
(104 operations, with the note that this is evidence of the mechanism, not the project's
registry — Phase 1 generates the registry by this method and records provenance).

**Still unreachable through this path:** `HomeTimeline`, `HomeLatestTimeline` and `Bookmarks`
are absent from both `main.js` and `vendor.js`; the timeline operations live in a further
lazy-loaded chunk. Locating that chunk is a genuine Phase-1 task — the method is now proven,
the target is not yet located — and until it is, the Following timeline's operation id has to
come from an external pinned source instead.

## Request volume

Eight requests in total for this probe, all read-only and unauthenticated, spaced by hand —
deliberately inside the discipline the project promises (`XAPI-07`): single-flight, no retry
loop, no background repetition.

The authenticated section above brings the running total for this file to roughly thirteen
read-only requests, all spaced by hand; the five listed earlier were unauthenticated and
came first.

## Following timeline: the operation id is not in the web bundle, and the feature bag turns out to be optional

Measured 2026-10-14 with the real session, through the proxy. This section post-dates the
authenticated-path section above.

- The authenticated app bundle contains
  `fetchGenericTimelineById:({count, cursor, timelineId}) => graphQL(..., {timelineId, count, cursor, withQuickPromoteEligibilityTweetFields:true, ...features})`,
  and its own string table maps `forYou -> {id:"home", type:"home"}` and
  `following -> {id:"home_latest", type:"home_latest", sort:"Popular"}`. So the current client
  reaches both home timelines through **one** operation, `GenericTimelineById`, keyed by
  `timelineId` — not through the historic `HomeTimeline`/`HomeLatestTimeline` operations.
- **That path does not work from a raw client.** `GenericTimelineById`
  (`cW93hZ3nlNYbYcUROF5r5A`) with `timelineId` = `home_latest` and then `home` answered
  **`200` with `{"errors":[{"message":"Internal server error"}]}`, 114 bytes** — tried with the
  full 40-switch feature bag, with two switches, with `{}`, and with the exact variable shape the
  bundle uses. It is not a features problem.
- The home-timeline operation ids are therefore **absent from the web bundle by design** and have
  to come from an external pinned source, which is what the architecture research predicted.
  `fa0311/TwitterInternalAPIDocument` (`docs/json/API.json`, master) supplied, on 2026-10-14:
  `HomeTimeline` → `og4a4SdSF3WiQkkwaPCdPg`, `HomeLatestTimeline` → `OQPHTgwczzp9RMAPt6BH9A`,
  `Bookmarks` → `-dgKZ58Dr9YSJYrcgEb5KA`.
- **`HomeLatestTimeline` works: the Following timeline is readable.** `200`,
  **957,866 bytes, 157 posts** in 6.9 s, with real authors and text. This is FEED-01's data path,
  proven before Phase 2 was planned.
- **The feature bag is optional for this operation.** Repeating the identical call with **no
  `features` parameter at all** returned `200`, 952,271 bytes, **155 posts**. The research's
  "features blob drifts and the request fails with `(336)`" risk is much smaller than modelled, at
  least for reads: the registry can carry an empty feature list here and the request still
  succeeds.
- One more shape for the classifier's table: **`200` carrying `errors[]`** (`Internal server
  error`) — a successful HTTP status that is nonetheless a failure. The classifier already has a
  bucket for it; this is the measured instance.

The honest gap this leaves: `UserTweets` and `HomeLatestTimeline` both work, but the *discovery*
path for the home-timeline ids is external, not local. A future bundle refresh will not supply
them, so they must be pinned in the registry and never pruned.

## `TweetDetail`: the thread shape, captured 2026-10-14

One authenticated call, `TweetDetail` (`3BlMtudy4N-d78ch7zTLZQ`) for the focal post
`2106019976381689871` (the most-replied post in the recorded `UserTweets` fixture):

- `200`, **202,462 bytes, 1.63 s**.
- **42 posts** inside the thread and **84 entries**; the entry kinds are
  `conversationthread` ×82, `tweet` ×1 (the focal post) and `cursor` ×1 (the paging cursor).
- So a thread is one paginated `TweetDetail` call, and the entry grammar is dominated by
  `conversationthread-*` entries — which is exactly the decoder Phase 4 was told to drive from a
  recorded fixture. The fixture is checked in as `internal/xsource/testdata/fixture-tweetdetail.json`
  (~202 KB) with the query id, the focal id, the variables and the response size recorded in its
  provenance row.

Also measured while capturing it: the reactions in that thread run from the focal account's own
reply down to plain spam, which is a useful reminder for the thread screen — the list is ranked,
not chronological, and the screen must not imply otherwise.

## Two request-shape drifts, measured 2026-10-15

Same host, same session, through the proxy at `http://127.0.0.1:7897`, one request at a time with
four seconds between them. Nothing here is derived from documentation or from the research files:
every row is a status, a byte count and, where the upstream sent one, its own sentence.

### Drift 1: required variables the client was not sending

The upstream answers a request that omits a required variable with `422` and
`{"errors":[{"code":"GRAPHQL_VALIDATION_FAILED","message":"must be defined","path":["variable","<name>"]}]}`
— 172 bytes. **The name of the missing variable is in `path`, not in `message`.** The message
alone (`must be defined`) names nothing; a client that printed only the message, or that failed to
decode an `errors[]` array whose `code` arrives as a *string* rather than a number, would have no
way to learn what the upstream wanted. Both of those were true of this client before this
measurement, which is why the probe was made to print the path before the variables were written.

| operation | registry query id | variables sent | live answer |
|---|---|---|---|
| `HomeLatestTimeline` | `OQPHTgwczzp9RMAPt6BH9A` | `count,cursor,seenTweetIds` | `422`, `path:["variable","includePromotedContent"]` |
| `HomeLatestTimeline` | same | `…,includePromotedContent:false` | `200`, 983,277 bytes (and `true` was measured answering `200` too) |
| `HomeTimeline` | `og4a4SdSF3WiQkkwaPCdPg` | `count,cursor,seenTweetIds` | `422`, `path:["variable","includePromotedContent"]` |
| `HomeTimeline` | same | `…,includePromotedContent:false` | `200` |
| `UserTweets` | `7_KBXBpG0zBPRN6UCIV0-w` | `userId,count,cursor` | `422`, `path:["variable","includePromotedContent"]` |
| `UserTweets` | same | `…,includePromotedContent:false` | `422`, `path:["variable","withVoice"]` |
| `UserTweets` | same | `…,includePromotedContent:false,withVoice:false` | `200` |
| `TweetDetail` | `3BlMtudy4N-d78ch7zTLZQ` | `focalTweetId,cursor` | `422`, `path:["variable","includePromotedContent"]` |
| `TweetDetail` | same | `…,includePromotedContent:false` | `422`, `path:["variable","withBirdwatchNotes"]` |
| `TweetDetail` | same | `…,withBirdwatchNotes:false` | `422`, `path:["variable","withVoice"]` |
| `TweetDetail` | same | `…,withVoice:false` | `200` |
| `Bookmarks` | `-dgKZ58Dr9YSJYrcgEb5KA` | unchanged | `200`, unchanged — the control |
| `Likes`, `Viewer`, `UserByScreenName` | unchanged | unchanged | `200`, unchanged |

So the complete required sets this build reads are:

- `HomeLatestTimeline`, `HomeTimeline`: `count`, `cursor`, `seenTweetIds`, `includePromotedContent`
- `UserTweets`: `userId`, `count`, `cursor`, `includePromotedContent`, `withVoice`
- `TweetDetail`: `focalTweetId`, `cursor`, `includePromotedContent`, `withBirdwatchNotes`, `withVoice`

The promotion switches are sent as `false`. Both `true` and `false` were measured answering `200`
on the home timelines, so `false` is not a value the upstream demanded — it is the value this
client chooses, because its renderer marks no advertisement anywhere in a timeline and asking for
promoted content would put posts in the pane that are indistinguishable from the ones the reader
asked for.

### Drift 2: the wire method is now a per-operation property

`SearchTimeline` (`irezafrnzav4Ulf1vDkEBg`) answers **`404` with a zero-byte body over `GET`** and
**`200` with real results (123,860 bytes) over `POST`** — with the same query id, the same
variables, the same feature switches, the same field toggles and the same headers. The wire method
is the only difference between the two answers, so it is a property of the operation rather than of
the client.

It cannot live in the registry: the bundle `cmd/x-opsgen` mines carries
`{queryId, operationName, operationType, metadata}` and no method, `Registry.Merge` replaces a
whole entry with what the bundle carries, and `Write` marshals the `Operation` struct — so a method
stored there would be dropped silently by the next `x-terminal doctor --refresh-ids`. It lives in
`internal/xweb/graphql.go`'s `postOperations` table, where every row's comment carries the
measurement that put it there.

### The command that reads a missing variable's name out of the upstream

```
X_TERMINAL_PROXY=http://127.0.0.1:7897 x-terminal doctor --probe <operation>
```

`--probe` prints the upstream's own sentence, which is where the variable name is:

```
probe      TweetDetail  bad request  remaining 295  reset 2026-10-15T12:13:24Z  must be defined at variable.includePromotedContent
```

`--json` carries the same sentence as `detail`. This is the instrument the repair path in
`RUNBOOK.md` now names first, and it is permanent rather than a throwaway probe.

### Two things that rule out the nearby wrong fixes

- **An operation-id refresh changes nothing here.** `doctor --refresh-ids` reports
  `+0 added, ~0 updated, 104 unchanged` for exactly this drift: the ids were current, and the
  drift was in the variables and the wire method. A refresh cannot see either.
- **It is not the session.** A rapid burst of requests occasionally answers
  `HTTP 401 {"code":89,"message":"Invalid or expired token"}`, and the identical request succeeds
  once it is spaced out again. Every measurement above therefore waited four seconds between
  requests, and one request that came back inside that window was retried once after the wait —
  never a loop.

## The translation endpoint, measured 2026-10-16

Same host, same session file, same client discipline as the two drifts above: one request at a time
with four seconds between them, and **every request went through the product's own client**
(`internal/xweb`'s `Client.Do`, so through the limiter) rather than through curl. That is not a
preference. Curl with hand-assembled headers produced spurious `401 {"code":89}` on requests this
client answered `200`, so a curl result that disagrees with a client result is the curl's problem and
never the upstream's.

### The request

```
GET https://api.x.com/1.1/strato/column/None/tweetId=<ID>,destinationLanguage=Some(<LANG>),translationSource=Some(X),feature=None,timeout=None,onlyCached=None/translation/service/translateTweet
```

It is built in `internal/xweb/translate.go` (`translateURL`, and `Client.Translate` above it), which
is where the prefix, the middle and the suffix live as constants. Every segment that is not the post
id or the language is the upstream's own vocabulary copied from the request the web client's bundle
builds: `strato/column/None`, `translationSource=Some(X)`, and `feature`/`timeout`/`onlyCached` sent
as `None`. None of it was chosen here.

**There is no `.json` suffix, and that is the one thing to get wrong.** The bundle's own call site is
`apiClient.get(path, {}, headers, "")`, and `get(e,t,r,i=".json",n)` builds `/1.1/${path}${i}` — the
empty fourth argument is what removes the suffix. Observed in this run, both forms in the same
process, four seconds apart:

| request | answer |
|---|---|
| the path above, `zh-CN` | `200`, `translationState:"Success"`, the post in Chinese (117 runes) |
| the same path with `.json` appended | `404`, 68 bytes, `{"code":34,"message":"Sorry, that page does not exist"}` — classified `endpoint retired` |

The suffixed `404` is a false negative and it is indistinguishable from "this endpoint does not
exist": a reader who trusted it would report a dead endpoint for a live one.

### The header set

The eight headers `graphQLRequest` already sends, now set by one function (`(*Client).apiHeaders`,
`internal/xweb/graphql.go`) because this endpoint is the same request class: `authorization: Bearer
<WebBearer>`, the `auth_token`+`ct0` cookie pair, `x-csrf-token` mirroring `ct0`,
`x-twitter-auth-type: OAuth2Session`, `x-twitter-active-user: yes`,
`x-twitter-client-language`, a browser `user-agent` and `content-type: application/json`.

**No `x-client-transaction-id` was needed**, and that is mechanical rather than lucky: the bundle's
own host allowlist for that signer is `^(https://)?((jf|grok|api|mobile)\.)?(twitter|x)\.com`, and it
guards the GraphQL path it is computed for. The signer seam in `graphql.go` is where a per-operation
requirement goes when one is measured; this endpoint gets one only if a measurement says so.

### Per-language answers, and the one that is not an answer

`destinationLanguage` really is the target: the same post came back in each language asked for.

| `destinationLanguage` | answer |
|---|---|
| `zh-CN` | `200`, Chinese |
| `de` | `200`, German |
| `ja` | `200`, Japanese |
| `en` on an English post | `200`, `"X is for everyone"`, `translationState:"Success"` |
| `zh-CN` for a post id that does not exist | **`200`** with `"translation":""` and `"translationState":"Missing"` |

**So a `200` does not mean a translation exists.** A nonexistent id answered `200` with nothing in
it, which is why the response carries its own `translationState` and why this build reads it: a pane
that rendered the `translation` field on any successful read would present an empty body as a
translation of the reader's post. `internal/xsource/ParseTranslation` is where that is refused —
`Success` with non-empty text is `Available: true`; every other state, and `Success` with an empty
body, is `Available: false` with no error, because that answer *is* the service answering; a body
whose `id_str` is not the post that was asked for is a typed error, because an answer about a
different post is an answer to a different question.

Of the response's fields (`id`, `id_str`, `translation`, `translationState`, `sourceLanguage`,
`localizedSourceLanguage`, `translationSource`, `destinationLanguage`, `entities`) this build reads
three and nothing else.

### Where each row was observed

The two rows of the first table are this executor's own run, printed verbatim by the gated test:

```
translate_test.go:326: the language this host resolves: zh-CN
translate_test.go:344: product path: Available=true, translation runes=117 (never the text)
translate_test.go:355: suffixed form: HTTP 404, classified "endpoint retired", body 68 bytes
```

`X_TERMINAL_CAPTURE=1 HTTPS_PROXY=http://127.0.0.1:7897 CGO_ENABLED=0 go test ./internal/xsource/
-run TestCaptureTranslation -count=1 -v` is the command, and the test skips without the gate, so a
normal `go test ./...` opens no socket. The per-language rows and the `Missing` row are the
measurement the plan records from the same host and the same day, taken with the same client and the
same spacing; they are carried here because the code's comments cite them, and this file is where a
claim about the upstream has to be able to point.

### Why the spacing, and the one retry

A rapid burst of requests on this host is occasionally answered
`HTTP 401 {"code":89,"message":"Invalid or expired token"}`, and the identical request succeeds once
it is spaced out again (first recorded here for the 2026-10-15 drifts). The gated test therefore
waits four seconds between its two requests and retries a `401` **once**, after another wait, and
never more than once — no loop, in the harness or in the product.

**The shipped path has no wait and no retry at all.** The 4-second spacing is a property of the
measurement, not of the product: the product spends exactly one request per translated post and the
limiter's own floor is the only wait it ever takes.

Nothing of a post's text lands in this file: the run printed the translation's rune count and never
its text, and no cookie is recorded anywhere here.
