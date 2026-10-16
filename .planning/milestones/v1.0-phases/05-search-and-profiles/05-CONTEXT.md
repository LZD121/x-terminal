# Phase 5: Search and Profiles - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — proposals accepted in batch under the run-wide policy

<domain>
## Phase Boundary

The user searches posts by keyword, finds people, and reads a profile's bio and recent posts
through the same list they already know.

Delivers: keyword search over posts with an explicit Latest/Top toggle; user lookup by handle or
display name, returning rows the same pane renders; the profile frame (bio, counts, join date) with
that account's recent posts paging on the same keys as the timeline; the entity-blind extraction of
the list machinery that makes all of it one list — a user result is not a post, and the walk does
not care; and the proof that the extraction is real rather than named.

Does NOT deliver: the `whoami` / session-status line (it moved to Phase 1 — see that phase's notes);
profile tabs beyond recent posts (replies, media); list or community timelines (v2); advanced search
operators; saved searches or any search history on disk; the profile's pinned post; or any new
persistence of any kind.

</domain>

<decisions>
## Implementation Decisions

### The walk, proved entity-agnostic

- Phase 2 shipped `screen.List` as a post list. **This phase performs the entity-blind extraction**
  the roadmap's note claims: the page cache, id-keyed dedupe, stall detection, single-flight latch,
  selection and anchor become one implementation over an entity-blind row — a row's identity and its
  kind — so a People result list, a post list and a profile's posts all run on the same machinery
  with no second caching, dedupe or stall path. Naming that value, its fields and the kind are this
  phase's design task; Phase 4 has already widened the selection to a target (id plus quote path),
  which is what this extraction generalizes.
- **Entity-agnostic means shared machinery, distinct rendering.** A user row is not a post block and
  must not be forced into one: `internal/render` gets a user-row form (handle, display name) and the
  pane dispatches on the row's kind. What is *not* allowed is a second pagination path, a second
  dedupe set or a second stall rule — the requirement's whole point is that a "user result" flows
  through the list view, not through a parallel one.
- **Entity-agnostic means shared machinery, distinct rendering.** A user row is not a post block
  and must not be forced into one: `internal/render` gets a user-row form (handle, display name)
  and the pane dispatches on the row's kind. What is *not* allowed is a second pagination path, a
  second dedupe set or a second stall rule — the requirement's whole point is that a "user result"
  flows through the list view, not through a parallel one.
- **Row identity stays one string per element type.** The walk keys on a string supplied by the
  element (the post id, the user id); no new id type is introduced, and nothing above the port
  parses an id.
- A user row prints the handle and the display name — both, because a display name is what a person
  is looked up by and a handle is what they are — and nothing else: no avatar (`AvatarURL` stays a
  content field nothing renders, per REND-01), no follower counts, no verification glyph. Phase 2's
  rule that the list view's identity is the handle and display names are absent applies to **post**
  rows, where a byline column has one job; a user row's job is to be findable by the name the person
  goes by (`DISC-02`), which is why this row shows both and the post block does not change.

### Search

- **One operation, the products the requirement names.** `SearchTimeline` (already registered:
  `irezafrnzav4Ulf1vDkEBg`) with product `Latest` / `Top` for posts, and product `People` for users.
  The `Media` product is not exposed (see *Deferred*): it would put pictures in a list, which is the
  one thing this project's plain-text promise refuses.
- **One entry point, one prompt.** `/` — `search posts or accounts` — opens a query prompt. The
  prompt is a **modal overlay** with the mechanics Phase 2 gave the help overlay (it replaces the
  pane, the status line stays, no frame is pushed until the query is submitted); `enter` submits and
  pushes a results frame, `esc` cancels and restores the pane with no request. It renders the
  product it will search, so the user never discovers the product after paying for the request.
- **The product is a cycle on one key, not a second prompt and not a setting.** `t` — the row Phase
  4 rewrote in its final form (`switch-list`, `switch what this list shows`, screens `home search`)
  — cycles `latest → top → people → latest` **in place**: one request, no new frame on the stack,
  and the status field naming the product at every instant. That is `DISC-01`'s explicit Latest/Top
  toggle with a second value for the People product, and it is why this phase adds no row of its
  own beyond `/`.
- *Deviation from the refresh path, stated:* switching the product **clears the list**, because
  keeping the old product's rows on screen under the new product's name would be a lie about what
  is shown — this is the one case where the pane discards posts it holds, and the frame's cold-start
  failure rules apply to the new product's first page.
- **The Latest/Top toggle is a key, not a setting** (the requirement's own wording), and `latest` is
  the product a `/` search starts in: reverse-chronological is the project's posture, while `top` is
  engagement-ranked. Recorded because the default is a product choice, not a parser detail; changing
  it is one constant.
- **An empty result set is `no results` in the notice, in the successful state.** It is never
  rendered as a failure, and no failure is ever rendered as an empty list (NET-03's rule; the
  classifier already supplies the failure's own words).
- **The query is never written anywhere.** It lives in the frame and in the status line's `list`
  field (`search latest: {query}`, `search top: {query}`, `search people: {query}`, `profile
  {handle}`), so the pane always says what it is showing, and there is no search history file, no
  cache and no config key.
- Search results are posts and render as the timeline's blocks do — same grammar, same width
  model, same cover — which is exactly what makes the cover (Phase 3) apply to them for free.

### Profiles

- **The profile fields are fields `domain.User` gains** — bio, the three counts, the join instant,
  and the verification fact — not a second profile type: the same `user_results.result` object is
  embedded in every post, so the decoder fills them for authors as it meets them, and the timeline's
  list grammar still prints none of them. A separate value would mean decoding the same object twice.
  The verification fact is carried and printed by no grammar this phase ships.
- **Two requests to open a profile, in this order: the header (`UserByScreenName`), then page one of
  the posts (`UserTweets`).** There is no one-request form of this: the header carries no posts, and
  `UserTweets` addresses the account by its numeric id, which is what the header returns. The header
  renders as soon as it arrives, with the post list below it in the loading state; a failure in
  either request is the ordinary one-line failure state, and a header that arrived is kept.
- **The profile frame is a header block plus an ordinary post walk.** The header scrolls with the
  content — this pane is a log viewer, and a pinned header is layout work for one screen that every
  other frame does not have. Pages page with the same keys, the same trigger and the same
  single-flight latch as the timeline: reaching the last loaded post asks for the next page.
- **Counts print plainly, in the header only.** `DISC-03` names bio, counts and join date, so the
  profile's follower/following/post counts are required content; `REND-01`'s prohibition is about a
  post's engagement numbers, and Phase 4 keeps those off the detail screen. The header is three rows
  — the identity row (`{handle:16} {name}`, exactly the user row's head row), the bio wrapped at the
  body indent, then one counts-and-join row in a fixed clause order, separated by two spaces. No
  glyph, no colour, no styling, never animated. **Absent values are omitted, never zeroed**: a
  clause the answer did not carry is not printed, and if none was carried the row is not printed at
  all — `0 followers` would be a claim the answer never made. A missing bio prints nothing.
- **`UserTweets` only.** The replies and media tabs (`UserTweetsAndReplies`, `UserMedia`) are out of
  scope, and the pinned post is deferred rather than rendered: it is a row that is not part of the
  walk and can also appear inside it, which would make the walk's id dedupe produce a row the user
  cannot explain.
- **A profile is reached from a user row (`enter`) and from the query prompt** (`/`, the handle, and
  `t` to `people`). No second route is added: opening an account from a mention inside a post's body
  stays out of scope, and the three-keystroke path through search is the documented one.

### Keys and rows

- One new row, from the table in `internal/keys` (so input, `?` help and the `keys` subcommand
  follow for free): `/` — `search` — `search posts or accounts`.
- `enter` is Phase 4's row, and in this phase it opens whatever the selected row *is*: a post's
  conversation or a user's profile. That dispatch on the row's kind is the entity-blind list paying
  off in one line of code.
- Phase 3 lands first; a collision with a cover key is resolved in the cover's favour (Phase 4's
  rule), at plan time, against the keymap as it actually landed.

### Parallel safety (the files this phase touches, and how)

- New files only: `internal/xsource/search.go` and `profile.go` (the search decoder for both entity
  kinds, and the profile header and `UserTweets` decoders, with their operation names),
  `internal/screen/search.go` and `profile.go` (the two frame reducers over the extracted list),
  `internal/render/user.go` and `profile.go` (the user-row form and the header block),
  `internal/fake/search.go` and `profile.go` (fixture-backed sources and the fake's three new port
  methods, in their own files), and this phase's fixtures under `internal/xsource/testdata/` and
  `internal/fake/testdata/`.
- Shared edits, append-only and named: the `Source` port grows three methods (`Search`, `Profile`,
  `UserPosts`, in Phase 2's `Home` shape) in its existing file, with the fake growing the matching
  methods; `internal/domain` gains the profile fields on `User`, the entity-blind row value, the
  result page and `SearchProduct ∈ {Latest, Top, People}`; the keymap table gains one row; the
  shell's frame dispatch gains two frame kinds.
- `Source.User` is left in place even though this phase's `Profile` read supersedes it: removing it
  would touch Phase 2's port, the fake and the parity test while three other phases are building
  against them. Phase 8's audit can report it as unused; deleting it is that phase's call, not this
  one's.
- The capture harness and the redaction rule are Phase 4's; this phase adds its own fixtures
  (a search page, a People page, a profile header, a `UserTweets` page) through the same gate.

### Claude's Discretion

- The prompt's copy and its cursor behaviour; the exact user-row layout; the header's field order
  and labelling; whether the query is echoed in the frame's first row or only in the status line;
  whether `SearchTimeline`'s `querySource` and other variables come from the registry's entry alone;
  the fixture set's size and split; and whether the People walk shares one `Request` case with user
  posts or has its own.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/screen`'s list machinery (Phase 2's reducer, Phase 4's `Target` selection) plus its
  `Effect`/`Request` values and the sealed `Event` set — this phase generalizes it to an
  entity-blind row and adds no second walk.
- `internal/xsource` — `ParseTimeline` reads the entries of a timeline response; a search page and
  a `UserTweets` page are the same shape (the research's endpoint table lists `SearchTimeline` and
  `UserTweets` under the same "bottom cursor" semantics as the home timelines), so the search and
  profile walks reuse the timeline decoder and add only their operation names and variable shapes.
  `ParseUser` already reads a profile container (`data.user.result`, then
  `data.viewer.user_results.result`); the header parser extends the same `userNode` reading with the
  bio/counts/join-date fields this screen prints.
- `internal/xweb/volatile/registry.json` already holds `SearchTimeline`, `UserByScreenName`,
  `UserTweets`, `UserTweetsAndReplies` and `UserMedia` with their feature switches — no registry
  work and no pin is needed for this phase.
- `internal/render` (Phase 2, in flight at the time this was written) — block, grammar, status,
  frame, help; the search-results and profile-post frames reuse the post block unchanged, and the
  user row is a new block form beside it.
- `internal/fake` — the pattern (embedded JSON, call counter, knobs for failure and stall) that the
  two new fixture-backed sources copy in their own files.

### Established Patterns

- The port is declared by the consumer and satisfied at the composition root; the three methods this
  phase's screens consume (`Search`, `Profile`, `UserPosts`, in Phase 2's `Home` shape) are added to
  the existing `Source` port beside `Home` and `User`, and implemented by `xsource.Adapter` in new
  files of its package.
- Cursors are opaque, dedupe is id-keyed, stall detection is Phase 2's, and no screen sorts
  anything: the answer's order is the list's order — which matters here twice, because a `Top`
  search is ranked and a `People` page is ranked, and the pane must present them as they came.
- Failures are values in the model, never errors from `Update`; the classifier's words are the only
  vocabulary for a failure state.
- Every request is one `tea.Cmd` through the single choke point; opening a search frame costs
  exactly one request, and the budget line is the proof.

### Integration Points

- `internal/tui` — two new frame kinds (search results, profile) and their dispatch; the frozen
  frame, anchor and pop semantics are inherited unchanged, so "back from a profile returns to the
  search results you came from, at the row you left" is Phase 2's mechanism, not new code.
- `internal/cli` — the composition root wires the new reducer effects to the adapter methods; the
  command table does not change (both screens live inside the TUI, no new verb).
- `internal/keys` — four rows, and the help overlay prints them without a second source of truth.
- `internal/render` — a user-row form and the profile header; both must render under every cover
  grammar (Phase 3's `Grammar`), which means their literals belong in `MarkerSet` rather than in
  the block code.
- `internal/tui`'s import boundary — the new screens import `domain` and `screen` only; the AST
  scan that keeps `net/http`, `os/exec`, `database/sql` and `internal/xsource` out of the shell
  applies to them unchanged.

</code_context>

<specifics>
## Specific Ideas

- A search result is a post: same block, same width behaviour, same keys. Nothing about the frame
  should tell the reader they are in a "search mode" except the words in the status line.
- A user row should read like a directory entry, not a profile card: `handle` first, display name
  beside it, one row per person, no counts, no avatar, no badge.
- The profile header reads like a database record — one fact per line, aligned, plain — and the
  posts below it are indistinguishable from the timeline.
- The prompt is where the product is chosen: pressing `/` and seeing the product it will search —
  `latest` by default — before typing is what keeps `top` and `people` from being surprises the
  reader discovers after paying for a request.
- The profile's bio is read, not admired: it wraps at the measure and prints **all** its lines, and
  the counts-and-join row is one plain line beneath it. A value the answer did not carry is absent,
  never a zero and never a placeholder.
</specifics>

<deferred>
## Deferred Ideas

- The `Media` search product: it would render attachments in a list, which the plain-text promise
  refuses; if media search is ever wanted it belongs behind the Phase 7 renderer and a deliberate
  key, not in a results list.
- Profile tabs — `UserTweetsAndReplies`, `UserMedia`, `UserOriginalsTimeline` (all registered, all
  reachable): each is a walk this phase's machinery could carry, and none is in the requirement.
- The profile's pinned post: a row that is not part of the walk and can duplicate one that is.
- Advanced search operators (`from:`, `since:`, `filter:`): a syntax to teach, an unstable upstream
  contract, and a parser to maintain for a personal tool.
- A "go to handle" direct jump (`:@name`) and a "search this thread's author" action: a second
  prompt path for a case the profile key already covers.
- Search history, saved searches, or any filter persisted to disk — never: the same reason the
  timeline is not persisted.
- Relationship badges and "follows you" markers: the social-app signal REND-01 exists to keep out.
</deferred>
