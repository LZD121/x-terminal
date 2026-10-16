# Architecture Research

**Domain:** Single-user read-only terminal (TUI) client for x.com, backed by x.com's internal web GraphQL endpoints and session-cookie auth
**Researched:** 2026-10-14
**Confidence:** MEDIUM-HIGH (HIGH for everything read out of primary code on this machine or in upstream repos; MEDIUM where the only evidence is third-party write-ups or version-snapshot drift; LOW items flagged inline)

**Feasibility verdict (this dimension):** **YES, with one condition** — the whole design rests on a single thin, isolated, replaceable layer. Every other part of the system is ordinary, well-trodden terminal-application work (proven in `/home/user/repos/rss-tm`). The condition is that the up-front spike (Build Order Phase 0) demonstrates a live cookie → GraphQL → parsed-posts path on the day it is attempted; if it cannot, nothing about the rest of the plan changes except that the *adapter* becomes the whole project.

---

## Standard Architecture

### System Overview

Data flows **down** to the wire; effects flow **up** as values, never as calls from the render path.

```
┌──────────────────────────────────────────────────────────────────────────┐
│ L0  PRESENTATION            internal/tui        (pure View + Update)     │
│      ┌──────────┬──────────┬──────────┬──────────┬──────────┐           │
│      │ timeline │ thread   │ search   │ profile  │ saved    │           │
│      └────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬─────┘           │
│           │          │          │          │          │                 │
├───────────┴──────────┴──────────┴──────────┴──────────┴─────────────────┤
│ L1  SCREEN STATE          internal/screen   (reducers, no I/O, no ctx)   │
│      pageCache[cursor] · dedupeSet[postID] · inflight flag · mode       │
├──────────────────────────────────────────────────────────────────────────┤
│ L2  DOMAIN                internal/domain   (Post, Author, Thread, Page) │
│      ── port declared HERE, consumed HERE, implemented in internal/cli ─│
├──────────────────────────────────────────────────────────────────────────┤
│ L3  SOURCE ADAPTER        internal/xsource  (x.com GraphQL *semantics*)  │
│      operation choice · variables · response→domain · cursor extraction   │
│      thread assembly · page de-dup · stall detection                      │
├──────────────────────────────────────────────────────────────────────────┤
│ L4  TRANSPORT             internal/xweb     (HTTP *mechanics*)            │
│      cookie jar · headers · request build · status/error classification   │
│      retry+backoff · rate-limit headers · single-flight · tid retry-on-404│
├──────────────────────────────────────────────────────────────────────────┤
│ L5  VOLATILE SURFACE      internal/xweb/volatile   ◄── THE CHURN SEAM      │
│      opid map (queryId/OpName) · features+fieldToggles bag ·             │
│      x-client-transaction-id generator · bundle/homepage fetchers        │
├───────────────┬─────────────────────────────┬────────────────────────────┤
│ L6 SESSION    │ L6 STORE                    │ L6 CONFIG / MEDIA          │
│ cookie 0600   │ SQLite mirror               │ non-secret config;         │
│ + env override│ bookmarks/likes ONLY        │ Renderer interface         │
└───────────────┴─────────────────────────────┴────────────────────────────┘
                                                          │
                                        ┌─────────────────┴───────────────┐
                                        │ external processes: xdg-open,   │
                                        │ mpv, (later) kitty-graphics tty │
                                        └─────────────────────────────────┘
```

### The churn-containment seam (the answer to "what absorbs upstream churn")

**Name:** the **Volatile Surface** — package `internal/xweb/volatile`, consumed only by `internal/xweb`, and never by `internal/xsource`, `internal/domain`, or `internal/tui`.

Everything upstream can change without warning, and all of it is exactly three things:

| Volatile thing | What changes | Where it is isolated |
|---|---|---|
| **Operation map** — `{queryId}/OperationName` per logical endpoint | queryId rotates; operation renames | `volatile/opmap.go` (+ cached JSON on disk, TTL) |
| **Feature / fieldToggles bag** — ~39 booleans per operation | X adds a required flag; responses start failing with error `(336)` | `volatile/features.go` (+ self-heal from the 336 message) |
| **Transaction-id generator** — `x-client-transaction-id` value | X changes the homepage key material / JS bundle chunking; some endpoints *require* the header at all | `volatile/transactionid.go` (+ home-page fetcher) |

Concrete evidence that all three rotate independently and often:

- The same operation carries **different queryIds in three public snapshots**: `HomeTimeline` is `-X_hcgQzmHGl29-UXxz4sw` in twikit's source ([twikit/client/gql.py:54](https://github.com/d60/twikit/blob/main/twikit/client/gql.py)), `HCosKfLNW1AcOo3la3mMgg` in an xfeed issue ([xfeed#5](https://github.com/ainergiz/xfeed/issues/5)) and `og4a4SdSF3WiQkkwaPCdPg` in fa0311's current capture ([TwitterInternalAPIDocument API.json](https://github.com/fa0311/TwitterInternalAPIDocument/blob/master/docs/json/API.json)). `TweetDetail` likewise differs across all three (`U0HTv-bAWTBYylwEMT7x5A` vs `XMOz5h24KAZ86qKffKTLdQ` vs `zoF7_t363wZyzylk-BLfZQ`).
- twscrape's own release notes carry **"Updated GraphQL operation IDs for current X API compatibility"** in four consecutive releases (v0.19.1 2026-06-26, v0.19.2 2026-07-21, v0.20.0 2026-08-07, v0.20.1 2026-08-25) — roughly monthly ([changelog.md](https://github.com/vladkens/twscrape/blob/main/changelog.md)).
- The transaction-id generator broke repeatedly: **"Fixed X client transaction ID generation for builds using 16-character chunk hashes"** (#327/#328, v0.20.1), **"Fixed JS bundle parsing for `x-client-transaction-id` generation"** (#303, v0.18.0), **"Fixed generation of `x-client-transaction-id` header required by the X API"** (#245/#248, v0.17.0 2025-04-29).
- The transaction-id header went from optional to **required per-endpoint**: FxEmbed had to add it specifically for `TweetDetail` in April 2025, with the author noting the endpoint "usually returns 404 now" ([FxEmbed#1283](https://github.com/FxEmbed/FxEmbed/issues/1283)). twscrape's client now retries a 404 **once** with a freshly generated transaction id ([queue_client.py:89-113](https://github.com/vladkens/twscrape/blob/main/twscrape/queue_client.py)).
- The feature bag drifts: twscrape parses `"(336) The following features cannot be null: a, b, c"` out of the error body and flips those flags to `true` on retry, and raises `GqlFeaturesOutdatedError` (replacing a process kill, #324) when it cannot ([queue_client.py:139-146, 285-287](https://github.com/vladkens/twscrape/blob/main/twscrape/queue_client.py)).

**Contract of the seam (written so work can be planned against it):**

```go
// internal/xweb/volatile — the ONLY package permitted to know about
// x.com's JS bundles, query IDs, feature flags, and header signing.

// Ops resolves a logical endpoint to today's {queryId}/{operationName}.
// It is a pure map lookup over an in-memory snapshot refreshed elsewhere.
type Ops interface {
    URL(op Operation) (string, bool)   // op = HomeLatestTimeline, TweetDetail, ...
    Age() time.Duration                // how stale the snapshot is
}

// Features returns the flag bag required for op, plus its fieldToggles.
type Features interface {
    Features(op Operation) map[string]bool
    FieldToggles(op Operation) map[string]any
    // Absorb336 records the flags X named as missing and makes them default-on
    // from now on. Never returns an error: a failure to learn is not fatal.
    Absorb336(op Operation, msg string)
}

// Signer produces the per-request x-client-transaction-id.
type Signer interface {
    Header(method, path string) (string, error)
    // Reset drops cached key material after a 404 or bundle change.
    Reset()
}
```

Rules the seam enforces by construction:

1. `Ops.URL` returns a `bool`: an unknown operation is a **typed, expected** condition (`xsource` maps it to `ErrUpstreamSchema`), never a panic and never a silent fallback to a stale id.
2. No package above `internal/xweb` may name a queryId, a feature flag, a header name, or `abs.twimg.com`. That is checkable mechanically (see Pattern 1 and Integration Points).
3. A snapshot may be **pinned** (a checked-in JSON file) *and* refreshed at runtime; the pinned file is the fallback that keeps the tool alive on the day X rotates everything, which is the difference between "update the tool" and "the tool is dead".
4. The refresh mechanism is deliberately dumb and matches how existing clients do it: fetch x.com's home page HTML, collect `responsive-web/client-web/*.js` (and `/x-web/*.js`) bundle URLs, then regex the bundles for `queryId:"…"` + `operationName:"…"` pairs, preferring `/responsive-web/client-web/` over `/x-web/` ([update-gql-ops.py:32-36, 154-203](https://github.com/vladkens/twscrape/blob/main/scripts/update-gql-ops.py); twscrape ships the map as a code-generated block between `GQL_OPS_CODEGEN` markers at [api.py:28-50](https://github.com/vladkens/twscrape/blob/main/twscrape/api.py)).

### Domain Model

The vocabulary the UI is written against. Upstream field names appear **only** in the `decode` column of this table and nowhere else.

| Domain value | Fields that matter here | Decoded from | Notes |
|---|---|---|---|
| `Author` | `UserID` (rest id string), `Handle`, `DisplayName`, `Bio`, `PinnedPostID`, counts, `Verified`, `Blue` | `core.user_results.result` / `author_results.result`; profile header from `UserByScreenName` | Upstream does **not** normalize authors: the same user object is repeated inside every post. De-duplicate into an `Author` map keyed by `UserID` at absorb time; it is what makes a timeline render cheap |
| `Post` | `ID`, `Author`, `Text`, `CreatedAt`, `Lang`, `ConversationID`, `ReplyTo *PostID`, `Repost *PostRef`, `Quote *PostRef`, `Media []Media`, `Counts`, `Visibility` | `legacy.*`, `note_tweet.*`, `retweeted_status_result.*`, `quoted_status_result.*` | `Text` resolution order: `note_tweet.note_tweet_results.result.text`, else `legacy.full_text`, then trim by `display_text_range`; expand `entities.urls[].expanded_url` and `entities.media[].url`, keep `alt_text`. Retweet and quote id resolution has nested paths (`…result.rest_id` **and** `…result.tweet.rest_id`) — [twscrape models.py:447-484](https://github.com/vladkens/twscrape/blob/main/twscrape/models.py) |
| **(repost / quote / plain)** — a three-valued relationship, not three types | `Post.Repost != nil` ⇒ this row is a repost of another post; `Post.Quote != nil` ⇒ this post quotes another; both nil ⇒ plain | as above | **A repost must be collapsed into the post it reposts at the page level** — upstream has returned the reposted original *as well*, producing duplicate top-level rows (twscrape #310/#311). Quotes are rendered inline as an indented block, never expanded into a second top-level entry |
| `Media` | `Kind` (photo/video/animated/gif), `URL`, `PreviewURL`, `AltText`, `Width`/`Height`, `DurationMS` | `legacy.entities.media[]`, `legacy.extended_entities.media[]` | The UI never fetches media itself; it hands the URL to `media.Renderer` |
| `PostRef` | `ID`, `AuthorID` (may be unknown) | same | A *reference*, not an embedded post: what a quote shows when the quoted post is unreachable |
| `Visibility` | `OK` / `Limited` (wrapper) / `Unavailable` (tombstone) / `Withheld` | `TweetWithVisibilityResults`, `TweetTombstone`, `__typename` | Drives one placeholder line; never a blank row |
| `Thread` | `Ancestors []Post` (root→parent), `Focal Post`, `Groups []Group`, `Next Cursor` | one `TweetDetail` family | `Group{Kind, Posts, More Cursor}` with `Kind ∈ {Replies, SelfThread, Related}` |
| `Page` | `Posts []Post`, `Next Cursor`, `FetchedAt`, `Requested` | every timeline response | The unit every screen consumes; `Next == ""` means end-of-list |
| `Cursor` | opaque string; **no exported constructor, no parsing, no arithmetic** | `content.value` on entries whose `cursorType` matches the operation | The UI can compare a cursor for equality and store it; that is all |
| `Profile` | `Author` + `PinnedPostID` | `UserByScreenName` | A profile view is one header request plus the ordinary `UserTweets` page flow |
| `SavedItem` | `Kind ∈ {Bookmark, Like}`, `Post`, `SortIndex`, `FirstSeen`, `Present` | `Bookmarks` / `Likes` pages | The **only** value the store accepts |
| Enums | `HomeKind ∈ {Following, ForYou}`, `SearchProduct ∈ {Top, Latest, People, Media}`, `UserTab ∈ {Posts, Replies, Media}`, `SavedKind` | — | Each enum exists to keep a raw upstream string out of the UI; `SearchProduct` maps 1:1 onto upstream's `product` variable |

Two consequences worth stating because they shape every screen:

1. **Identity is the post id.** Everything the UI does (de-dup, refresh diffing, thread ordering, mirror keys) is keyed on `PostID`; nothing is keyed on a position or an index.
2. **Unavailability is a first-class value.** Deleted, protected, limited-visibility and withheld posts are normal outcomes of clicking a link in a timeline; the domain models them so no screen has to invent "what if the payload is empty".

### Component Responsibilities

| Component | Responsibility | Typical implementation |
|---|---|---|
| `internal/tui` (L0) | Terminal I/O loop, key grammar, focus/scroll, View rendering. Owns the `tea.Model`. **Cannot import** `net/http`, `os/exec`, `database/sql`, `xweb`, `xsource`, `store`. | Go + Bubble Tea v2; `Update` returns `(Model, tea.Cmd)`; `View` is a pure function of fields |
| `internal/screen` (L1) | Per-screen reducers: timeline, thread, search, profile, saved. Page cache, dedupe set, inflight flag, mode/error state. Pure; no `context.Context` in any signature. | Plain structs + `func (m TimelineModel) Apply(ev Event) (TimelineModel, Cmd)` |
| `internal/domain` (L2) | `Post`, `Author`, `Thread`, `Page`, `Cursor`, `Profile`, `SavedItem`; error taxonomy; **the `Source` port declared here**. No HTTP, no SQL, no tea. | Value structs; ~250 lines |
| `internal/xsource` (L3) | x.com **semantics**: pick operation, assemble `variables`/`features`, decode `instructions`/`entries`, map to domain, extract cursors, de-dup pages, detect stalls, assemble threads. | Consumes `xweb.Fetcher`; pure mapping functions + a thin orchestrator |
| `internal/xweb` (L4) | HTTP **mechanics**: cookie jar, canonical header set, request build, status/error classification, bounded retry + jitter, `x-rate-limit-*` interpretation, single-flight, 404→new-tid retry. | `net/http` with a shared `*http.Client`; injected clock; injected `Signer`/`Ops` |
| `internal/xweb/volatile` (L5) | **Only** place that knows queryIds, feature bags, transaction-id derivation, JS-bundle URLs. Pinned snapshot + runtime refresh. | Regex over bundles; base64/hash arithmetic; on-disk cache with TTL |
| `internal/session` (L6) | Load/validate the cookie; `0600` enforcement; redaction-by-type; diagnostics values. | `Cookie` struct with unexported fields and redacting `String`/`MarshalJSON` |
| `internal/store` (L6) | SQLite mirror for **bookmarks and likes only**; migrations; incremental + full sync. Insert API accepts only `SavedItem` values. | `database/sql` + a pure-Go SQLite driver; single writer, `query_only` reader pool |
| `internal/config` (L6) | Non-secret config: paths, theme, external opener command, image mode, page caps. Round-trips safely through an editor. | TOML or JSON file; env overrides |
| `internal/media` (L6) | `Renderer` interface: `Open(ctx, Media) error`. External-program implementation today; Kitty-graphics implementation later. | `os/exec` behind the interface, only reachable from `internal/cli`/`internal/media` |
| `internal/cli` | Composition root: wires concrete `xweb`, `xsource`, `store`, `media` into the ports the UI consumes; owns `tea.Program`. | ~200 lines of wiring, the only place that names everything |

### Recommended Project Structure

```
cmd/x-terminal/              # main(): flag parsing, config load, calls cli.Run
internal/
├── tui/                     # L0  bubbletea Model/Update/View, keys, golden frames
│   └── boundary_test.go     #     AST import scan: this package may not reach I/O
├── screen/                  # L1  timeline.go thread.go search.go profile.go saved.go
│   └── cache.go             #     page cache + dedupe + stall detection (pure)
├── domain/                  # L2  post.go author.go thread.go page.go errors.go port.go
├── xsource/                 # L3  adapter: home.go detail.go search.go profile.go saved.go
│   ├── decode/              #     response → domain mapping, one file per response shape
│   └── testdata/            #     recorded JSON fixtures (redacted, checked in)
├── xweb/                    # L4  client.go headers.go errors.go retry.go ratelimit.go
│   ├── volatile/            # L5  opmap.go features.go transactionid.go bundle.go
│   │   └── pinned.json      #     last-known-good op map + feature bag
│   └── testdata/
├── session/                 # L6  cookie.go redact.go doctor.go
├── store/                   # L6  schema.go migrate.go sync.go queries.go
├── config/                  # L6  config.go paths.go
├── media/                   # L6  renderer.go external.go kitty.go(capability-gated)
└── cli/                     #     composition root + commands (run, sync, doctor)
```

### Structure Rationale

- **`domain` owns the port, not the adapter.** The interface lives with its consumer; the implementation is wired in `cli`. This is the pattern already load-bearing in the sibling repo: `tui.Source` (`rss-tm/internal/tui/model.go:154`) and `tui.Subscriber` (`:272`) are declared inside `internal/tui`, `daemon.FetchFunc` is declared inside `internal/daemon` (`scheduler.go:79-82`), and every one of them is satisfied by `internal/cli` at start-up. No shared code is reused here — the *pattern* is the evidence, not the package.
- **`xweb` split from `xsource` split from `volatile`.** Three different rates of change: HTTP mechanics (rarely changes), x.com response semantics (changes when the payload shape changes), volatile signatures (changes monthly). One package for all three would mean a queryId rotation forces a re-read of the parser.
- **`volatile` has no dependencies.** It cannot import `xsource` or `domain`, so it also cannot accidentally become the place where parsing logic accumulates. Its tests use checked-in HTML/JS fixtures.
- **`store` is a leaf with a one-way API.** Nothing about timelines or threads can be written because no insert function accepts those types — the constraint is expressed in the type system, not in a comment.
- **`session` is separate from `config`.** The project decision is explicit that the cookie must not live on the generally-editable config surface; two packages is the cheapest way to make that structural.


## Architectural Patterns

### Pattern 1: Pure `Update`/`View` with every read expressed as a `Cmd`, enforced by an AST import scan

**What:** The render/update layer must be incapable of blocking. Every network read, disk read, and process spawn is a value returned from `Update` (a `tea.Cmd` closure created in the composition root or by a thin adapter-aware wrapper), never a call made inside `Update` or `View`.

**When to use:** Always, in a terminal application that must feel instant and must never wedge on a hung socket. This project has manual refresh only, so the temptation to "just fetch here" is at its highest.

**Trade-offs:** A little ceremony (an effect enum + a `Cmd` builder per effect) buys a model that can be driven to any state in a test with zero servers, zero sleeps, and no terminal. The cost is real for throwaway code; this is not throwaway code.

**Example — the state, then the gate:**

```go
// internal/screen/timeline.go — no context.Context, no client, no clock.
type TimelineModel struct {
    Pages    []domain.PostPage      // ordered, newest page first
    Cursor   domain.Cursor          // adapter-owned opaque value
    Seen     map[domain.PostID]bool // de-dup across refreshes
    Inflight bool                   // single-flight latch
    Err      domain.Error           // taxonomy value, never a raw string
}

func (m TimelineModel) Apply(ev Event, p domain.Source) (TimelineModel, Cmd) {
    switch e := ev.(type) {
    case ScrollToEnd:
        if m.Inflight || m.Cursor == "" { return m, nil }   // latch, not a queue
        return m.setInflight(), fetchPage(p, domain.HomeLatest, m.Cursor)
    case PageFetched:
        return m.absorb(e.Page), nil                        // pure; de-dup happens here
    }
    return m, nil
}
```

The gate that keeps this honest is an import-boundary test in `internal/tui`, copying the hermetic AST-scan shape already proven in the sibling repo (`rss-tm/internal/tui/boundary_test.go:43-63`): parse the package's own non-test sources with `go/parser`, collect import paths, fail if any forbidden path appears, and **fail if the scan found zero files or zero imports** — the anti-vacuity arms are load-bearing, because a scan that sees nothing passes for the wrong reason. A second arm asserts that a package which *must* import bubbletea actually does, so the test cannot pass while looking at the wrong directory. Extend the same file (or its sibling) to assert the *inverse* for `internal/xweb/volatile`: it must not import `domain` or `xsource`, so churn containment cannot be eroded by a convenient import.

### Pattern 2: The port is declared by the consumer and implemented at the composition root

**What:** `internal/domain` declares the small read-only interface the screens need; `internal/xsource` satisfies it structurally; `internal/cli` is the only package that names both. There is no "interfaces" package and no `x.com` type escaping `xsource`.

**When to use:** Whenever one component is expected to be replaced. Here that is not hypothetical — the adapter *will* be rewritten, at least once, probably more than once.

**Trade-offs:** Go's implicit interfaces make this nearly free. The one discipline required is that the interface must be phrased in domain terms (`HomeLatest`, `Cursor`) and never in upstream terms (`focalTweetId`, `cursor_type`).

**Example:**

```go
// internal/domain/port.go
type Source interface {
    Home(ctx context.Context, kind HomeKind, c Cursor) (PostPage, error)
    Post(ctx context.Context, id PostID) (Post, error)
    Thread(ctx context.Context, id PostID, c Cursor) (ThreadPage, error)
    SearchPosts(ctx context.Context, q string, p SearchProduct, c Cursor) (PostPage, error)
    SearchUsers(ctx context.Context, q string, c Cursor) (UserPage, error)
    User(ctx context.Context, h Handle) (Profile, error)
    UserPosts(ctx context.Context, id UserID, tab UserTab, c Cursor) (PostPage, error)
    Saved(ctx context.Context, kind SavedKind, c Cursor) (PostPage, error)
}

type PostPage struct {
    Posts      []Post
    Next       Cursor    // "" means end of list
    FetchedAt  time.Time
    Requested  int       // how many entries upstream returned (for staleness UI)
}
```

Note what is *absent*: no `map[string]any`, no `json.RawMessage`, no HTTP status, no retry count. Those live below the port. The error taxonomy that crosses it is closed and small — `ErrSessionExpired`, `RateLimited{Reset time.Time}`, `ErrUpstreamSchema{Op string}`, `ErrUnavailable`, `ErrTransport` — so the UI can render a specific line per class and never print an upstream message (which may echo user content or a URL).

### Pattern 3: Volatile surface with a pinned snapshot + self-healing

**What:** The three rotating artifacts live in one leaf package behind three tiny interfaces (see *System Overview*). The package ships a pinned snapshot; at runtime it may refresh from the JS bundles; on the specific upstream signals it repairs itself in-place rather than failing the request.

**When to use:** Any time the upstream contract changes without notice. The alternative — hardcoding `queryId`s in the adapter — is how these tools die: it makes a rotation a code change plus a rebuild instead of a cached-file update.

**Trade-offs:** Refresh machinery is more code than a constant, and the discovery path is itself fragile (it parses bundle URLs out of homepage HTML — and twscrape has had to fix that path repeatedly, e.g. #303 and #327/#328). Mitigation: **the pinned snapshot is authoritative**; refresh only *adds* to it, and a failed refresh never degrades a working map.

**Self-healing rules, each traceable to a real failure mode:**

| Signal | Response | Evidence |
|---|---|---|
| HTTP 404 with empty body on a GraphQL op | Regenerate the transaction id **once**, retry once, then abort the request | twscrape `queue_client.py:89-113` (its comment cites its own issue #248); FxEmbed#1283 for the endpoint-scoped requirement |
| Error `(336) The following features cannot be null: …` | Parse the named flags, set them true in the bag, retry once | twscrape `queue_client.py:139-146, 285-287` |
| Error `(-1) LoadShed` | Retry with backoff; **do not** treat as account problem | twscrape `queue_client.py:326-331` |
| Error `(32) Could not authenticate you`, or 403 with no structured error | Terminal: session expired → `ErrSessionExpired` (see Pattern 7 and Data Flow) | twscrape `queue_client.py:306-314` |
| Error `(88) Rate limit exceeded` while `x-rate-limit-remaining > 0` | Treat as account-level block, stop for the session | twscrape `queue_client.py:296-299` |
| `x-rate-limit-remaining == 0` | Park until `x-rate-limit-reset` | twscrape `queue_client.py:290-293` |

### Pattern 4: Opaque cursors, per-screen page cache, and stall detection

**What:** Cursors are opaque domain values the UI only stores and hands back. Each screen keeps its pages in memory; pagination is keyset-based (the server's cursor encodes position), never offset-based. De-duplication happens on absorb: an id already in `Seen` is dropped before it reaches the view.

**When to use:** All timelines here — home, user, search, bookmarks, likes. The home timeline is *ranked* (For You) or reverse-chronological (Following); search has Top/Latest; none of them is stable enough for offsets.

**Trade-offs:** An in-memory cache dies with the process, which is the point (the timeline must never be persisted). The price is that every launch is a cold start — acceptable for a manual-refresh reader.

**Endpoint → screen → pagination semantics (the table the adapter implements):**

| Screen | Operation(s) | Variables that matter | Cursor semantics |
|---|---|---|---|
| Home / **Following** | `HomeLatestTimeline` | `count`, `cursor`, `seenTweetIds` | The **last** entry's `content.value` where `content.cursorType == "Bottom"` is the next cursor; `"Top"` pages toward newer posts |
| Home / For You | `HomeTimeline` | same | same shape; responses are ranked, so ordering is not chronological |
| Post detail + thread | `TweetDetail` | `focalTweetId`, `cursor`, `referrer` | Bottom cursor for the next page of the conversation; a **per-module** cursor of type `ShowMoreThreads` is the "show more replies" handle and is passed back as the top-level `cursor` |
| Search posts | `SearchTimeline` | `rawQuery`, `count`, `product: Top\|Latest\|Media`, `querySource` | Bottom cursor; dedupe is mandatory — repeated cursors/pages are a known upstream behaviour |
| Search users | `SearchTimeline` with `product: "People"` | `rawQuery`, `count`, `cursor` | Bottom cursor; pages return users, not posts |
| Profile header | `UserByScreenName` | `screen_name`, `withSafetyModeUserFields` | No pagination (single object) |
| Profile posts | `UserTweets` (or `UserTweetsAndReplies`) | `userId`, `count`, `cursor` | Bottom cursor |
| Bookmarks | `Bookmarks` (or `BookmarkFolderTimeline` with a folder id) | `count`, `cursor` | **Two** cursors: `items[-1].content.value` is next, `items[-2].content.value` is previous (used when the folder variant is requested) |
| Likes | `Likes` with the account's own `userId` | `userId`, `count`, `cursor` | Same shape as `UserTweets` |

Evidence: operation names and variable shapes from [twikit/client/gql.py:34-101, 262-374, 406-435](https://github.com/d60/twikit/blob/main/twikit/client/gql.py) and [twscrape/api.py](https://github.com/vladkens/twscrape/blob/main/twscrape/api.py); next-cursor extraction as `items[-1]['content']['value']` from [twikit/client/client.py:2003-2004, 2063-2064](https://github.com/d60/twikit/blob/main/twikit/client/client.py); bookmarks' next/previous pair from [client.py:2284-2290](https://github.com/d60/twikit/blob/main/twikit/client/client.py); the `ShowMoreThreads` cursor type from [twscrape `tweet_replies_raw`](https://github.com/vladkens/twscrape/blob/main/twscrape/api.py); instruction/cursor/`sortIndex` structure from [Trekhleb's typed walkthrough of the home timeline API](https://dev.to/trekhleb/an-exploration-of-how-xs-home-timeline-api-is-designed-3n6o) (`TimelineAddEntries` / `TimelineTerminateTimeline`, `cursorType: 'Top' | 'Bottom'`, `seenTweetIds`).

**Two de-duplication layers, both required:**

1. **Client-side id set.** Every absorbed post id goes into `Seen`; a repeat is dropped (and is the signal that pagination is not advancing). This also fixes a real upstream artefact: retweeted originals have appeared as duplicate top-level timeline entries (twscrape #310/#311, fixed in v0.19.0).
2. **Stall detection.** Terminate a scroll-run when (a) `Next` is empty, (b) `Next` equals the cursor just used, or (c) a page adds zero unseen ids. twscrape implements exactly this as `_is_stalled` and keeps a per-request id set ([api.py:109-122, 172-196](https://github.com/vladkens/twscrape/blob/main/twscrape/api.py)); its changelog records both "Fixed pagination loops caused by repeated cursors or pages" and "Fixed pagination to continue past empty pages" (#247, #265) — i.e. both failure modes are real and opposite.

**Also pass `seenTweetIds`.** Filling `variables.seenTweetIds` with the ids currently on screen asks the server not to repeat them; it is the only server-side help available and it is cheap (Trekhleb documents it as the mechanism that keeps consecutive pages distinct).

### Pattern 5: Thread assembly from one `TweetDetail` stream, not from N reply requests

**What:** A conversation is reconstructed from a single paginated `TweetDetail` call family plus local ordering — never by fanning out one request per reply.

**When to use:** Always for post detail. The N+1 alternative both burns the request budget and reads as bot traffic.

**Trade-offs:** The response's entry grammar is the fiddliest thing in the project; it needs its own decoder package and fixtures. In exchange, "show more replies" is one more page, not a new subsystem.

**Assembly algorithm (each step traceable to twikit's decoder, [client.py:1552-1650](https://github.com/d60/twikit/blob/main/twikit/client/client.py)):**

1. Request `TweetDetail(focalTweetId=id)`. Iterate `data.threaded_conversation_with_injections_v2.instructions[].entries[]` — skipping entries whose `entryId` starts with `cursor` (those are the page's own cursors, not content).
2. `entryId == "tweet-<id>"` is the **focal post**. Entries that appear *before* it in the stream are the **ancestor chain** (what the UI shows above the focal post as context); twikit collects them as `reply_to`.
3. Entries whose id begins with `tweetdetailrelatedtweets` are *related/similar* posts — a separate rail, not part of the thread; classify them separately or drop them.
4. Conversation entries carry `content.items[]`; skip the composer (`entryId` containing `tweetcomposer`), take tweets, and note any nested `cursor` inside the module: that value is the **"show more replies"** handle. Emit it as a per-post affordance, not as the page cursor.
5. `tweetDisplayType == "SelfThread"` marks a self-thread: the focal post and its immediate replies are one authored block and should render as one block, not as a post plus unrelated replies.
6. Take the page cursor from the trailing `cursor-*` entry. Fetch it with the same operation to extend.

**Where the honest limits are** (the UI must say these out loud rather than imply completeness):

- **Replies are ranked and truncated server-side.** The page is not the full reply set; "show more" is a bounded walk, and the walk can end before the conversation does.
- **Some replies are structurally unreachable.** Replies from blocked/muted accounts, replies to protected accounts, deleted posts, and replies hidden by the author's reply controls simply do not appear in the payload. There is no authorised endpoint for this client to ask differently.
- **Author-side limits on the reply set will be reached before the client's own caps on large threads**, so the client cap is the fallback, not the primary limit.
- **`TweetWithVisibilityResults` wrappers** mean a post's fields are one level deeper than usual (`result.tweet.legacy…`); the decoder must unwrap before reading, and a "tombstone" result means a post that exists in position but has no content — render a placeholder line, not an empty post.
- **Promoted/interstitial entries** appear inline in some timelines; the adapter must classify and skip them rather than render them, and must never count them toward the visible-post budget (a count=20 page can return well over 20 entries — Trekhleb observed 37).

### Pattern 6: Mirror-only persistence — two tables, an upsert, an incremental sync

**What:** A SQLite file that contains **only** the account's bookmarks and likes. The insert API accepts `domain.SavedItem` and nothing else; there is no code path from a timeline, thread, search result or profile to disk.

**When to use:** Immediately, in the same phase that adds the bookmarks/likes screens — not "later". Retrofitting a persistence boundary after the UI already holds timeline pages is where leaks appear.

**Trade-offs:** A schema for two lists is over-engineering in the abstract; here it pays for itself because the sync story ("what did I read before?") is the only reason the mirror exists at all.

**Minimum schema (v0):**

```sql
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);      -- schema_version
CREATE TABLE accounts (id INTEGER PRIMARY KEY, handle TEXT NOT NULL UNIQUE, user_id TEXT);
CREATE TABLE saved_items (
  kind         TEXT    NOT NULL CHECK (kind IN ('bookmark','like')),
  post_id      TEXT    NOT NULL,
  sort_index   TEXT    NOT NULL,        -- upstream sortIndex; orders without re-parsing
  first_seen   INTEGER NOT NULL,        -- unix seconds
  last_seen_at INTEGER NOT NULL,        -- last sync that still contained it
  present      INTEGER NOT NULL DEFAULT 1,  -- 0 = no longer on x.com
  json         TEXT    NOT NULL,        -- render-ready snapshot of the post
  PRIMARY KEY (kind, post_id)
);
CREATE INDEX saved_items_order ON saved_items(kind, sort_index DESC);
CREATE TABLE sync_runs (
  id INTEGER PRIMARY KEY, kind TEXT NOT NULL, started_at INTEGER NOT NULL,
  finished_at INTEGER, pages INTEGER, items INTEGER, stop_reason TEXT NOT NULL
);
```

- **Upsert semantics:** `INSERT … ON CONFLICT(kind, post_id) DO UPDATE SET last_seen_at=excluded.last_seen_at, sort_index=excluded.sort_index, present=1, json=excluded.json` ([SQLite UPSERT](https://sqlite.org/lang_upsert.html) — verified live).
- **Incremental sync (the default):** page newest→older; stop when a page contains **no id that is missing from `saved_items` for that kind**. Because both lists are append-only in practice, this is one request in the common case. Persist the last synced cursor so a resumed sync does not re-walk.
- **Full re-list (`sync --full`):** walk to the end (bounded by a page cap), then `UPDATE saved_items SET present=0 WHERE kind=? AND last_seen_at < :run_start`. Rows are **never deleted**: an un-bookmark should surface as "gone from x.com", not as history silently disappearing.
- **Migrations:** ordered `[]migration{version, ddl}`, each applied in its own transaction together with the `meta.schema_version` bump, refusing to run against a file newer than the build knows. This mirrors the sibling repo's approach exactly (`rss-tm/internal/store/migrate.go:16-60`, with the version row written inside the same transaction as the DDL) and avoids depending on `PRAGMA user_version`.
- **Pragmas:** `journal_mode(WAL)`, `busy_timeout`, `synchronous(NORMAL)`, `foreign_keys(1)`; a single writer connection and a reader connection opened `query_only` (the UI only reads). WAL is read back and verified rather than assumed — the sibling repo does exactly this, and the read-back is the difference between "database is locked" and not (`rss-tm/internal/store/store.go:30-68`, DSN pragmas at `:155-163`; background on the mode itself at [SQLite WAL](https://sqlite.org/wal.html), pragmas at [sqlite.org/pragma.html](https://sqlite.org/pragma.html)).
- **No timeline, ever.** Enforced three ways: the store's insert functions only accept `SavedItem`; a test asserts the table list via `PRAGMA table_info`; and the import-boundary test asserts `internal/tui` cannot reach `internal/store` at all.

### Pattern 7: The secret is a type, and the diagnostics path is additive

**What:** The session cookie is a struct whose default formatting is redacted. Reading the real value is an unexported method that only `internal/xweb` can call. There is no configuration surface, log call, error path, or CLI dump that can print it by accident.

**When to use:** From the first commit. This project has no precedent to copy (the sibling repo has no secret mechanism at all), and the cookie is a bearer credential for the user's entire x.com session — not a token for this app.

**Trade-offs:** Slightly awkward when debugging (you must use the diagnostics command to see *anything*), which is precisely the intent.

**Rules:**

- **Location:** `$XDG_CONFIG_HOME/x-terminal/session`, mode `0600`, created by the app if absent (template body with two `key=value` lines). Loading order: `XTERMINAL_AUTH_TOKEN` + `XTERMINAL_CT0` (env) → `XTERMINAL_SESSION_FILE` (env path) → default path. `auth_token` and `ct0` together are what a working session requires — twscrape's own session predicate is exactly those two cookie names ([account.py:18-19](https://github.com/vladkens/twscrape/blob/main/twscrape/account.py)).
- **Mode check:** `stat` the file; if `mode & 0o077 != 0`, refuse to start and print the exact `chmod 600 <path>` command. (Design choice, `[INFERENCE]` from general practice; the value is that it fails loudly rather than silently reading a world-readable credential.)
- **Never reaches:** `internal/config` (which must round-trip through an editor), argv (visible in `/proc/*/cmdline` to any local process), log lines, wrapped error strings, the SQLite file, or an HTTP error message echoed to the terminal. CWE-532 names the log-file case directly, and its stated mitigation is to not write secrets into logs ([CWE-532](https://cwe.mitre.org/data/definitions/532.html)).
- **Enforcement by construction:**

```go
// internal/session/cookie.go
type Cookie struct{ authToken, ct0 string }   // unexported: unreachable from other packages

func (c Cookie) String() string  { return "session{auth_token:REDACTED, ct0:REDACTED}" }
func (c Cookie) GoString() string { return c.String() }
func (c Cookie) MarshalJSON() ([]byte, error) { return []byte(`"REDACTED"`), nil }  // a struct cannot contain one

// headerValue is the ONLY way the raw values leave this package, and it is
// unexported, so the compiler enforces that only internal/session's package
// and its embedding site (internal/xweb) can call it.
func (c Cookie) headerValue() (auth, csrf string) { … }
```

- **Leak test:** a fixture cookie whose value is a unique random string; a test drives every error path (transport error, 401, 403, 429, 404, 336, parse failure, store failure, doctor output, `--verbose` logging into a buffer) and asserts the fixture string appears **nowhere** in the captured output. This is a golden test with a single, unmissable assertion, and it is the reason `String()`/`MarshalJSON` are implemented rather than trusted to discipline.
- **Redacted diagnostics (`x-terminal doctor`)** prints only: the resolved session-file path; its mode; presence booleans (`auth_token present (len 40)`, `ct0 present`); the op-map snapshot's age and how many operations resolved; whether the transaction-id signer initialised; the last request's operation name, status, and `x-rate-limit-remaining/reset`; and the store's schema version. It never prints a header dump, a response body, or a token prefix/suffix.

## Data Flow

### Request Flow

```
[key: j / scroll to end]
        ↓
tui.Update  ── pure ──▶ screen.Apply(Event)            (no I/O; sets Inflight)
        ↓
returns (Model, tea.Cmd)                               (Cmd = closure value, built in cli)
        ↓
cli's effect runner ──▶ domain.Source.HomeLatest(ctx, Cursor)
        ↓
xsource: choose op ──▶ variables + features ──▶ xweb.Fetcher.Do
        ↓
xweb.volatile.Ops.URL(TweetDetail)      ← pinned snapshot, maybe refreshed
xweb.volatile.Features.Features(op)     ← pinned bag, maybe healed by 336
xweb.volatile.Signer.Header(GET, path)  ← derived from today's bundle key material
        ↓
net/http  ──▶ x.com /i/api/graphql/{queryId}/{OperationName}
        ↓
status + x-rate-limit-*  ──▶ classified: ok | Ratelimited | Session | Schema | Transport
        ↓
xsource decode: instructions → entries → posts → domain.PostPage{Posts, Next}
        ↓
screen absorbs page: drop seen ids, append, store Next cursor, clear Inflight
        ↓
tui.View renders from the model only  ──▶ terminal
```

Nothing in this chain writes the timeline anywhere. The only disk writes in the whole system are: (a) the session file, when the user writes it; (b) the bookmarks/likes mirror, in the `Saved` flow only; (c) the volatile snapshot's cache file.

### State Management

```
                    ┌──────────────────────────────────────┐
   key event ──────▶│ tui.Model                            │
                    │  ├─ screen.TimelineModel             │
                    │  │    Pages[] · Cursor · Seen{} · Inflight
                    │  ├─ screen.ThreadModel                │
                    │  │    Ancestors[] · Focal · Groups[] · Cursor
                    │  └─ screen.Search/Profile/SavedModel  │
                    └───────────────┬──────────────────────┘
                                    │ Update returns (Model, Cmd)
                                    ▼
                    ┌──────────────────────────────────────┐
                    │ effect runner (internal/cli)         │
                    │  maps Cmd → one port call + one event │
                    └───────────────┬──────────────────────┘
                                    │ msg: PageFetched / FetchFailed / SessionExpired
                                    ▼
                    ┌──────────────────────────────────────┐
                    │ tui.Update(msg) → new Model (pure)    │
                    └──────────────────────────────────────┘
```

Three invariants worth stating as testable properties:

1. **One in-flight request per screen instance.** The latch is set in `Apply` and cleared only by a terminal message for that request (success, failure, or cancellation). A scroll key pressed while `Inflight` is true is a no-op, not a queued fetch.
2. **Cursors are never computed by the UI.** The model stores an opaque `domain.Cursor` copied from the page; there is no arithmetic on it anywhere above `internal/xsource`.
3. **Every failure is a value in the model.** `Update` never returns an error; the error is data the `View` renders, which is what makes "session expired" a screen state rather than a crash.

### Key Data Flows

1. **Cold start (one page, then stop).** Load session (env override → file, mode-checked) → `Ops` snapshot from the pinned file (age reported in the footer, not blocking) → first `HomeLatestTimeline` page (count 20). The signer initialises lazily on first request: it fetches x.com's home page once, derives the key material, and caches it — the same lazy, cookie-preserving init twikit performs ([client.py:134-146](https://github.com/d60/twikit/blob/main/twikit/client/client.py)). No second page is fetched until the user asks.
2. **Scroll older.** Next cursor from the last page → one request → absorb with de-dup → repeat until the cursor stalls or is empty. All completed pages stay in memory, keyed by the cursor that produced them, so scrolling back up costs zero requests.
3. **Manual refresh (the only "new content" path).** Drop the top-of-stack entry, re-request the first page *without* a cursor but *with* `seenTweetIds` = ids currently rendered, and prepend only previously-unseen ids. Never refetch history; never reorder what the user is reading; never fire on a timer — refresh is one explicit key, per the project decision.
4. **Open a post.** From a rendered post, `Post(id)` (cheap single-object read) → immediately render the post and its ancestor chain, with replies marked as loading → `Thread(id)` pages fill the reply groups → the "show more replies" affordance carries the `ShowMoreThreads` cursor. The screen labels partial threads honestly (see Pattern 5).
5. **Bookmarks / likes.** `Saved(kind)` pages newest→older; each page is upserted into the mirror; the screen renders from the adapter, and the mirror is what makes the list readable when the network is unavailable. This is the *only* place a `store` write exists, and the store's API cannot express anything else.
6. **Rate limited.** `RateLimited{Reset}` → the footer shows "rate-limited, retrying after HH:MM", the scroll latch stays closed, and no automatic retry happens before `Reset`.
7. **Session expired.** `ErrSessionExpired` → a single line replaces the footer: the cookie was rejected by x.com. The already-rendered timeline stays on screen (nothing to protect; nothing was persisted), refresh and pagination disable, and the message names the session file path and the `XTERMINAL_AUTH_TOKEN` override — never a byte of the cookie, and never the upstream error text verbatim.
8. **Open media.** `View` renders a stable, click-free affordance (e.g. `[1] image · [2] video`) that `Update` maps to a `Cmd`; the `media.Renderer` implementation decides what "open" means. In v1 that is an external process (`xdg-open`, `mpv`); the Kitty implementation is added behind the same interface only after the empirical terminal test (Build Order Phase 0b) passes.

## Scaling Considerations

The template's user-count axis does not apply — this is a single-user personal tool by decision, and it must never be distributed. The honest axes are **request budget** (the scarce resource; a ban or a 429 ruins the tool), **terminal responsiveness**, and **local file size**.

| Concern | One session, one screen | A long scroll (≈10k posts in memory) | Five years of bookmarks |
|---|---|---|---|
| Upstream requests | 1 per explicit action; single-flight; no polling | Bounded by the cursor walk; page cache means scrolling back up is free | Incremental sync stops after the first page that contains only known ids |
| Latency felt | Startup must render before the first request completes (skeleton rows, not a blank screen) | Re-render only the visible window; never re-layout the whole slice on keypress | Read from SQLite; no network on open |
| Memory | A few dozen posts | `[]Post` for 10k posts with ~1–4 KB each is tens of MB — cap retained pages per screen (e.g. 10 pages) and drop the oldest, since re-fetching is one request | `json` per saved row dominates; a few thousand rows is a few MB — fine |
| Disk | Nothing | Nothing (the timeline is never persisted) | Single file, WAL; bounded growth |
| Failure mode that bites first | Session expiry mid-scroll | Repeated cursors causing an infinite walk (stall detection is what prevents this) | Schema change in the stored snapshot → version the snapshot blob, not the whole row |

**Scaling priorities (what breaks first):**

1. **The upstream request budget.** Fixed by single-flight, the page cache, manual refresh, and treating a stalled cursor as termination. Nothing else in this system should be optimized before this.
2. **Memory from unbounded page retention.** Fixed with a per-screen page cap and a "drop the oldest non-visible page" rule; there is no reason to hold an unbounded scrollback.
3. **Not a bottleneck, by decision:** concurrency, multi-user, distributed anything. Adding any of them reintroduces the ToS exposure the project explicitly refuses.

## Anti-Patterns

### Anti-Pattern 1: The view model owns pagination
**What people do:** Store raw cursors in the widget, decide "is there more" in `View`, and re-issue fetches from three different screens.
**Why it's wrong:** Cursor semantics differ per endpoint (home has Top/Bottom, bookmarks have next *and* previous, the thread has a per-module `ShowMoreThreads` handle). Re-implementing that in the render layer duplicates a subtle rule three times and gets it wrong at least once — and it puts the de-dup set where the renderer can accidentally iterate it.
**Do this instead:** The adapter returns `PostPage{Posts, Next}`; screens store the opaque cursor and nothing else.

### Anti-Pattern 2: Walking raw JSON in the UI or the domain
**What people do:** Thread `map[string]any` up the stack and reach for `obj["legacy"]["full_text"]` inside a renderer.
**Why it's wrong:** It moves the churn boundary *upward* — the exact opposite of the project's one explicit constraint. Every payload change then touches every screen.
**Do this instead:** One decode boundary in `internal/xsource/decode`, producing `domain` values. If a field is missing, the decoder substitutes a domain default and the UI never learns the difference.

### Anti-Pattern 3: Query IDs as constants in the adapter
**What people do:** Copy today's `{queryId}/{OpName}` pairs into `xsource` and move on.
**Why it's wrong:** Rotation is monthly at times. Constant-in-adapter means a rotation requires editing the parser package — the one place that must stay still.
**Do this instead:** Everything volatile lives in `internal/xweb/volatile` behind `Ops`/`Features`/`Signer`, with a pinned snapshot that refresh adds to.

### Anti-Pattern 4: Persisting the timeline "just for scroll performance"
**What people do:** Cache fetched pages to SQLite so relaunch feels instant.
**Why it's wrong:** It directly contradicts the project's privacy constraint (no browsing traces on disk in the office environment) and it is unnecessary: the page cache + one request per page makes relaunch cheap anyway.
**Do this instead:** In-memory pages only; the mirror exists solely for bookmarks/likes, and no store function accepts a post page.

### Anti-Pattern 5: Logging headers or bodies while debugging
**What people do:** Dump the request headers on a 403, or print the response body on a parse failure.
**Why it's wrong:** The headers contain the session bearer and the CSRF token; the body contains whatever the account can see. One `--debug` flag turns the terminal into a credential leak, and CWE-532 exists because this is a classic ([CWE-532](https://cwe.mitre.org/data/definitions/532.html)).
**Do this instead:** Log the operation name, status, and rate-limit numbers; implement redaction on the credential type; keep a leak test that greps captured output for a fixture secret.

### Anti-Pattern 6: Retrying a 404 forever
**What people do:** Treat every non-2xx as transient and retry with backoff.
**Why it's wrong:** On these endpoints a 404 with an empty body usually means the transaction-id/op-id pair is wrong — it is a schema signal, not a network blip. Retrying it is load without information and raises the odds of tripping bot defences (FxEmbed#1283, twscrape #248).
**Do this instead:** One regeneration-and-retry for the tid case, then `ErrUpstreamSchema{Op}` and stop. Never retry a 404 more than once.

### Anti-Pattern 7: N+1 reply fetching
**What people do:** Fetch the thread, then one request per reply to get their sub-replies.
**Why it's wrong:** A 200-reply thread becomes 200 requests: the request budget is gone, the account looks abusive, and the server-side conversation module already delivered most of it.
**Do this instead:** One `TweetDetail` family with `ShowMoreThreads` paging (Pattern 5).

### Anti-Pattern 8: "Just for search, use the official API"
**What we would be doing:** Falling back to X API v2 or a paid aggregator for reads that the web endpoints do not serve cleanly.
**Why it's wrong:** Cost is the reason this project exists; per-post read pricing puts a normal week of browsing in the dollars, and it needs developer credentials the user does not have.
**Do this instead:** Accept the honest limits of the web endpoints and label them in the UI.

### Anti-Pattern 9: Media as an in-app concern
**What we would be doing:** Downloading images/video to disk, transcoding, thumbnailing, and rendering them in-app.
**Why it's wrong:** It creates a disk footprint from timeline content (privacy), pulls codecs and image pipelines into a text tool, and blocks the UI on I/O.
**Do this instead:** A `Renderer` interface with an external-program implementation first; optional Kitty graphics only after Phase 0b's empirical check.

### Anti-Pattern 10: Background polling / auto-refresh
**What we would be doing:** A timer that refetches the first page every N minutes.
**Why it's wrong:** The project decision is explicit manual refresh; polling multiplies requests ~100×, reorders the user's reading position, and is the fastest route to a 429 or a session block.
**Do this instead:** One key, one request, and an explicit "newer posts available" affordance only if refresh finds any.

## Recommended Build Order (added section — the template has no equivalent, but roadmap creation needs it)

Two spikes, then phases whose dependency edges are stated as "must be proven before". `∥` marks work that can proceed concurrently with the item before it.

| # | Work | Depends on | Proves / produces |
|---|---|---|---|
| **0** | **Live request path, end to end.** A `cmd/spike` (throwaway) that: reads the cookie → issues one authenticated GraphQL request → prints parsed post text/author to stdout. | — | That the whole premise works *today*: cookie accepted, header set accepted, an op id resolves, the features bag satisfies the server, the transaction id is generated correctly, the response parses. Nothing else can be honestly planned until this is green; it also fixes the concrete `Cursor` and `Post` shapes that Phase A and B both need as an interface. **Throw this code away** — it exists to answer questions, and its fixture output becomes `xsource/testdata`. |
| **0b** | ∥ **Empirical terminal-image test.** Render a static image in the actual environment: (a) plain Ghostty; (b) inside tmux with `allow-passthrough off` then `on`; (c) after a pane resize and a scroll. Record what survives. | — (independent of 0; pure terminal work) | Whether the Kitty-graphics upgrade path is viable at all in this user's setup. Ghostty documents Kitty graphics support ([ghostty.org/docs/features](https://ghostty.org/docs/features)); tmux grew `allow-passthrough` in 3.3 with default off and a third state `all` in 3.4, and its image handling is SIXEL-oriented ([tmux CHANGES](https://github.com/tmux/tmux/blob/master/CHANGES), local tmux 3.7c). **Do not design around images before this test.** |
| **A** | Domain model + `Source` port + `xsource` adapter + `xweb` transport + `volatile` surface, with recorded fixtures and unit tests. | 0 | The load-bearing half: everything the UI will ever need, behind the port, testable offline. |
| **B** | ∥ TUI skeleton: Model/Update/View, key grammar, golden frames, screens against an in-memory fake `Source`. | Shape of the port from **0** (not A's implementation) | That the render/update split is enforced and the app feels instant with a fake. Can and should run concurrently with A — the two meet only at the interface. |
| **C** | Home timeline: Following + For You, scroll/pagination, page cache, de-dup, stall detection, manual refresh, single-flight, rate-limit and session-expiry UI. | A + B | The primary screen and every request-discipline rule. |
| **D** | Post detail + thread assembly (ancestors, reply modules, `SelfThread`, show-more, honest partial labelling). | A + C (post rendering) | The hardest decoder; benefits from fixtures captured in 0. |
| **E** | ∥ Search (posts + people) and profile (header + recent posts). | A + C | Reuses pagination and post rendering wholesale; only new decode paths are search-response and user-response. Independent of D. |
| **F** | ∥ Bookmarks + likes mirror: SQLite schema, migrations, incremental + full sync, `Saved` screen reading from the mirror. | A (+ B for the screen) | The only persistence in the system, and the only place where "never persist the timeline" has to be proven rather than asserted. Independent of D and E. |
| **G** | ∥ Media `Renderer` interface + external-program implementation; Kitty implementation **only if** 0b passed. | B | Keeps v1 working everywhere and makes the later upgrade a one-file change. |
| **H** | ∥ Session-expiry/renewal UX, `doctor` command, redaction leak test, doctor's diagnostics. | C (needs a state to expire *from*) | Closes the security and operability story before the tool is relied on daily. |

**Ordering rationale (the dependency argument, not a preference):**

- **0 before everything.** Every phase's design assumes "the upstream answers". If it does not, the plan is still correct but the *first* work item becomes the volatile surface, and that is worth knowing on day one rather than day fifteen. It is also the cheapest possible way to discover the real fixture shapes.
- **0b before any media design, not before coding.** It is a two-hour experiment that decides whether a whole subsystem exists in v2. Running it in parallel with 0 costs nothing.
- **A and B in parallel.** They touch different packages and share only the port shape, which 0 fixed. Serializing them would add latency for no information.
- **C before D/E/F.** Pagination, de-dup and request discipline are written once and reused; writing the thread decoder first would tempt a second paging implementation into the post-detail screen.
- **D, E, F, G, H are mutually independent** (given A, B, C) and can be fanned out — this is the natural place to parallelize agents, one package boundary per slice.
- **H is last only because it needs a failure to render**, not because it is optional. If the tool is used daily before H lands, a cookie expiry will look like a crash.

**What must be true before each later phase is worth starting** (the "prove it" checklist):

1. Before **A**: one request works (0) and the response shape of `HomeLatestTimeline` is captured as a fixture.
2. Before **C**: the port compiles against a fake and against the real adapter, with the import-boundary test passing for both directions.
3. Before **F**: the store's insert API has been reviewed to accept only `SavedItem`, and the `PRAGMA table_info` allowlist test exists.
4. Before shipping daily use: the leak test passes and `doctor` answers "is my session healthy?" without printing a secret.

## Integration Points

### External Services

| Service | Integration pattern | Notes |
|---|---|---|
| **x.com internal GraphQL** — `https://x.com/i/api/graphql/{queryId}/{OperationName}` | `GET` (query-string params) or `POST` (JSON body) with `variables` + `features`; both are used by existing clients ([twikit gql_get/gql_post](https://github.com/d60/twikit/blob/main/twikit/client/gql.py), [twscrape](https://github.com/vladkens/twscrape/blob/main/twscrape/api.py)) | The churn hotspot. `{queryId}` rotates; see the seam contract. Undocumented and unsupported by X; the project accepts that (personal, non-distributed). |
| **Required request headers** | `authorization: Bearer <public web client token>`, `x-csrf-token: <ct0>`, `content-type: application/json`, `x-twitter-active-user: yes`, `x-twitter-auth-type: OAuth2Session`, `x-twitter-client-language`, `referer: https://x.com/`, realistic `user-agent`, and `x-client-transaction-id` for the operations that enforce it | The bearer at [twscrape/account.py:14](https://github.com/vladkens/twscrape/blob/main/twscrape/account.py) is a **public constant baked into x.com's web bundle** — it is not a user credential and may be checked in. The *cookie* is the credential. Header set mirrors [twikit `_base_headers`](https://github.com/d60/twikit/blob/main/twikit/client/client.py). |
| **`abs.twimg.com/responsive-web/client-web/*.js`** and x.com's home page HTML | Fetched by `volatile` only, to (a) resolve queryIds and (b) derive transaction-id key material | The discovery path is itself brittle (twscrape fixed it in #303, #327/#328). It is why the pinned snapshot, not the refresh, is authoritative. |
| **`https://api.x.com/1.1/guest/activate.json`** | Guest token activation, `x-guest-token` header ([twikit v11.py](https://github.com/d60/twikit/blob/main/twikit/client/v11.py)) | **Not needed for this design** — every screen requires an authenticated session. Kept as a documented escape hatch only (e.g. if a future feature wants anonymous reads); do not build it in v1. |
| **TLS fingerprinting** | Plain `net/http`; optionally a browser-impersonating transport later | twscrape added a `curl_cffi` backend for browser-like TLS fingerprints (#269/#308) because Cloudflare-style blocks care about it; twscrape also detects Cloudflare and HTML-block pages by `cf-ray`/content-type when a request fails ([queue_client.py:257-260](https://github.com/vladkens/twscrape/blob/main/twscrape/queue_client.py)). v1 uses `net/http` and classifies a block as a plain transport failure; upgrading the transport is a one-file change behind `xweb`. |
| **External media programs** (`xdg-open`, `mpv`) | `media.Renderer.Open(ctx, Media)` → `os/exec` with an allowlisted command from config | The only process spawn in the system, and it is behind an interface so `internal/tui` cannot name `os/exec` (import-boundary test). |
| **The terminal itself** | Plain text; Kitty graphics later | Ghostty supports the Kitty graphics protocol ([Ghostty docs](https://ghostty.org/docs/features), [libghostty Kitty graphics API](https://libghostty.tip.ghostty.org/group__kitty__graphics.html)). Inside tmux it requires `allow-passthrough` (added 3.3, `all` state in 3.4, [tmux CHANGES](https://github.com/tmux/tmux/blob/master/CHANGES)); with passthrough the outer terminal draws the image while tmux does not track its cells, so redraw/scroll behaviour must be observed rather than assumed `[INFERENCE]`. |

### Internal Boundaries

| Boundary | Communication | Notes |
|---|---|---|
| `tui` ↔ `screen` | Direct function calls on value types; `Event` in, `(Model, Cmd)` out | Pure both sides; no `context.Context` appears in either package's signatures |
| `screen`/`tui` ↔ `domain.Source` | Method call on an interface value held by the model | The only way the UI reaches the network. Injected in `cli`; replaced by a fake in every UI test |
| `tui`/`screen` ⊥ `xweb`, `xsource`, `store`, `os/exec`, `net/http`, `database/sql` | **Forbidden imports**, verified by an AST import scan per package with anti-vacuity arms | Mirrors `rss-tm/internal/tui/boundary_test.go:43-63`. This is the mechanism that makes "pure update" a fact instead of a convention |
| `xsource` ↔ `xweb` | Method call `Fetcher.Do(ctx, Request) (Response, error)` where `Request` names an **operation**, not a URL | `xsource` never builds a URL, never sees a header, never sees a status code it must interpret beyond the taxonomy |
| `xweb` ↔ `volatile` | `Ops.URL`, `Features.Features`, `Signer.Header` | The churn seam. `volatile` imports nothing from the project (not even `domain`), so it cannot grow parsing logic |
| `session` ↔ `xweb` | `Cookie.headerValue()` (unexported) called once when the client is constructed | Raw token values exist in exactly one function's scope for the life of one header build |
| `store` ↔ everything else | `UpsertSaved`, `ListSaved`, `SyncState` — value types only | Insert functions accept `domain.SavedItem`; no function accepts a page, a thread, or raw upstream JSON |
| `media` ↔ `tui` | Interface method via a `Cmd` | Keeps `os/exec` out of the render package and keeps the Kitty upgrade a same-interface swap |

## Sources

**Primary — upstream client code read directly (highest confidence):**

- twikit — `twikit/client/gql.py` (endpoint constants, variable shapes, `get_query_id`) — https://github.com/d60/twikit/blob/main/twikit/client/gql.py
- twikit — `twikit/client/client.py` (headers, transaction-id init, status/error mapping, timeline and bookmarks cursor extraction, `TweetDetail` thread decoding) — https://github.com/d60/twikit/blob/main/twikit/client/client.py
- twikit — `twikit/client/v11.py` (guest activation endpoint) — https://github.com/d60/twikit/blob/main/twikit/client/v11.py
- twikit — `twikit/x_client_transaction/transaction.py` (transaction-id derivation needs the home page + `loading-x-anim` frames) — https://github.com/d60/twikit/blob/main/twikit/x_client_transaction/transaction.py
- twscrape — `twscrape/api.py` (operation map with `GQL_OPS_CODEGEN`, `GQL_FEATURES`, cursor/entry extraction, stall detection, `ShowMoreThreads`) — https://github.com/vladkens/twscrape/blob/main/twscrape/api.py
- twscrape — `twscrape/queue_client.py` (404→new-tid retry, 336 self-heal, rate-limit headers, 32/88/326/403 classification, LoadShed) — https://github.com/vladkens/twscrape/blob/main/twscrape/queue_client.py
- twscrape — `twscrape/account.py` (public web bearer, `auth_token`+`ct0` session predicate, header set) — https://github.com/vladkens/twscrape/blob/main/twscrape/account.py
- twscrape — `scripts/update-gql-ops.py` (queryId discovery by scraping webpack bundles, source priority) — https://github.com/vladkens/twscrape/blob/main/scripts/update-gql-ops.py
- twscrape — `twscrape/models.py` (`retweeted_status_result` / `quoted_status_result` / `note_tweet` unwrapping) — https://github.com/vladkens/twscrape/blob/main/twscrape/models.py
- twscrape — `changelog.md` (dated evidence of op-id rotation, transaction-id breakage, pagination-loop bugs, duplicate-repost bugs, feature-bag breakage) — https://github.com/vladkens/twscrape/blob/main/changelog.md
- fa0311/TwitterInternalAPIDocument — `docs/json/API.json` (current queryIds and per-operation feature sets; a third independent snapshot) — https://github.com/fa0311/TwitterInternalAPIDocument/blob/master/docs/json/API.json
- Local sibling repo `/home/user/repos/rss-tm` — `internal/tui/boundary_test.go:3-63` (hermetic import scan with anti-vacuity arms), `internal/tui/model.go:154,272` (consumer-side ports), `internal/daemon/scheduler.go:79-82` (`FetchFunc` seam), `internal/store/migrate.go:16-60` and `internal/store/schema.go:44-238` (ordered migrations + `meta.schema_version`), `internal/store/store.go:30-68,155-163` (WAL verified by reading it back; DSN pragmas) — read locally on 2026-10-14

**Secondary — issue threads and write-ups (medium confidence, used for failure-mode evidence):**

- FxEmbed #1283 — `TweetDetail` returning 404 until a valid `x-client-transaction-id` was supplied, April 2025 — https://github.com/FxEmbed/FxEmbed/issues/1283
- xfeed #5 — `HomeTimeline` / `HomeLatestTimeline` operation ids and the Following/chronological mapping — https://github.com/ainergiz/xfeed/issues/5
- Trekhleb — typed walkthrough of the home timeline API (`TimelineAddEntries`, `TimelineTerminateTimeline`, `cursorType: Top|Bottom`, `sortIndex`, `seenTweetIds`, page sizes exceeding `count`) — https://dev.to/trekhleb/an-exploration-of-how-xs-home-timeline-api-is-designed-3n6o
- X developer community — likes became private/authenticated-only in June 2024, which is why *own* likes are the only likes this client can read — https://devcommunity.x.com/t/updates-to-support-private-likes-on-the-x-api/221104

**Standards / platform documentation (verified live 2026-10-14):**

- SQLite UPSERT — https://sqlite.org/lang_upsert.html
- SQLite WAL mode — https://sqlite.org/wal.html
- SQLite PRAGMA reference — https://sqlite.org/pragma.html
- CWE-532: Insertion of Sensitive Information into Log File — https://cwe.mitre.org/data/definitions/532.html
- Ghostty feature list (Kitty graphics protocol support) — https://ghostty.org/docs/features ; libghostty Kitty graphics API — https://libghostty.tip.ghostty.org/group__kitty__graphics.html
- Kitty terminal graphics protocol specification — https://sw.kovidgoyal.net/kitty/graphics-protocol
- tmux `CHANGES` (allow-passthrough introduced with default off in 3.3; `all` state added in 3.4; SIXEL — not Kitty — is tmux's own image path) — https://github.com/tmux/tmux/blob/master/CHANGES

**Explicitly unverified / to be settled by experiment:**

- Per-endpoint rate-limit *quotas* for these internal endpoints: no published source exists. The design therefore reads `x-rate-limit-limit/remaining/reset` from responses and never assumes a number. `[INFERENCE]`
- Whether the account's own `Likes` list is served to a cookie-authenticated web client today (the endpoint exists in both twikit and the API capture; twscrape *removed* its `liked_tweets` method in v0.16 after the 2024 privacy change). **Treat as a Phase-0 question with a real possibility of "no"**, with the fallback that the likes screen is dropped or rendered read-only-but-empty with an honest message. `[UNVERIFIED]`
- Whether tmux's passthrough of Kitty graphics sequences yields correct placement/redraw in this configuration — Phase 0b answers it empirically. `[UNVERIFIED]`
- Whether `go/parser`-based import scans plus golden frames carry over unchanged to the chosen TUI framework's model type (they did for Bubble Tea v2 in the sibling repo, but this project chooses its stack on its own merits). `[INFERENCE]`

---
*Architecture research for: a read-only terminal x.com client on internal GraphQL endpoints*
*Researched: 2026-10-14*
