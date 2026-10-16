# Phase 6: Bookmarks, Likes and the Mirror-Only Store - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — proposals accepted in batch under the run-wide policy

<domain>
## Phase Boundary

The user reads their own bookmarks and likes from the active session and can open that mirror on
disk with the network down — while the timeline stays structurally unable to reach disk.

Delivers: `internal/store` (schema, ordered migrations with the version bump in the same
transaction, an incremental sync that stops at the first page with no unknown ids, a full re-list
that marks rows `present = 0` instead of deleting them, a `query_only` reader, and a write API that
accepts nothing but a saved item); `saved.db` under the data directory; the bookmarks and likes
screens reading through the same walk and the same pane as every other list; the offline read; the
`sync` verb and the in-pane `r`; the schema-allowlist and import-boundary tests that make the
privacy promise a property rather than a statement; and the pin-override path exercised end to end
for the one operation the bundle cannot see.

Does NOT deliver: any other persistence (no timeline, no read-state, no cache, no body log); write
actions (un-bookmarking, liking, saving); a UI for pins; more than one account per mirror; a
background or scheduled sync; or an import of anything from outside the session.

</domain>

<decisions>
## Implementation Decisions

### The store is the only persistence in the system

- **Location and name.** `saved.db` under `config.DataDir` (the XDG data directory the pins and the
  notice marker already use), with SQLite's own `-wal` and `-shm` siblings. The name is the scope
  statement: this file is the saved lists and nothing else.
- **Driver.** `modernc.org/sqlite` (pure Go, BSD-3-Clause, v1.60.1 per the stack research) through
  `database/sql`, driver name `sqlite`, `CGO_ENABLED=0` preserved — the only module this project
  adds, and it must be fetched through the local proxy like every other module on this host.
- **Schema v1: three tables, and no fourth.** `meta` (schema version, the mirror's owner, the last
  sync's kind/time/stop reason), `saved_items` (`kind` checked to `bookmark`/`like`, `post_id`,
  `sort_index`, `first_seen`, `last_seen_at`, `present`, and the post's own render-ready snapshot),
  `sync_runs` (what each run walked and why it stopped). The research's separate `accounts` table is
  **not** built: this project decided on one account, and a table with exactly one row is dead
  weight — the owner is a `meta` row, which is also what the pane reads to say whose mirror this is
  when it is offline.
- **The stored item carries the author's id, not only the handle.** The cover aliases an account by a
  deterministic function of its account id (`DSGT-07`), so a mirror read offline under a cover can
  only produce the same alias if the id is on disk: with a handle alone the frame would have to
  print a real handle under a cover (`DSGT-06`) or invent a second alias for the same account. One
  extra field, and it is content.
- **The write API is sealed by type.** The package exposes no `Exec`, no raw statement, no
  `Put(key, value)` and no exported `*sql.DB`. Its mutating entry points take `domain.SavedItem`
  values (or a saved kind and a run record) and nothing else; the JSON column is produced *inside*
  the store from the item's own post. A timeline post cannot be written because no function accepts
  one (SEC-02), and that is a property of the type system rather than of a comment.
- **One writer, one reader pool.** A single write connection owned by the sync path, and a reader
  opened `query_only` for the pane. `journal_mode(WAL)`, `busy_timeout`, `synchronous(NORMAL)`,
  `foreign_keys(1)` are set in the DSN, and the WAL mode is **read back and verified** rather than
  assumed — the sibling project's precedent, and the difference between "database is locked" and
  not.
- **Migrations are ordered data.** `[]migration{version, ddl}`, each applied in its own transaction
  together with the `meta.schema_version` bump, refusing to run against a file newer than the build
  knows. `PRAGMA user_version` is not used.
- **The stored post is the post the user saved.** A bookmarked post's own body is on disk — that is
  what makes the offline read worth having — and it is the *only* kind of post that can ever reach
  the file. The timeline, a thread, a search result and a profile page have no write path at all;
  see *The proof* below.

### Sync: incremental by default, never destructive

- **The pane never fetches on open.** Opening the bookmarks or likes frame renders from `saved.db`
  and issues no request. This is what makes SAVE-03's "opens and reads with the network down" true
  on the first launch as well as the hundredth, and it is the same rule the rest of the app lives
  by: reads happen on an explicit key, and opening a screen is navigation, not a fetch. The
  consequence is stated plainly: on a fresh machine the saved frame shows an empty mirror until
  `r` (or `x-terminal sync`) fills it.
- **`r` inside a saved frame is the incremental sync**, using Phase 2's refresh key with its
  existing meaning ("fetch newer, keep the place"): walk newest→older, upsert each page's items,
  and **stop at the first page that contains no id missing from the mirror for that kind**. Page
  one always costs a request; the common case is exactly one. Selection and scroll do not move, and
  the posts already on screen are not re-ordered.
- **`x-terminal sync [--full] [--kind bookmarks|likes]` is the headless path**: first population,
  scripted refreshes, and the full re-list. It is one more row in the existing command table, it
  goes through the same choke point and the same budget, and it prints presence facts only — pages
  walked, items added, stop reason — never a post body.
- **A full re-list marks, never deletes:** after walking to the end (bounded by the page cap and the
  session budget), `UPDATE saved_items SET present = 0 WHERE kind = ? AND last_seen_at < <run
  start>`. An un-bookmark is "gone from x.com", not history quietly disappearing (SAVE-04).
- **The pane renders from the mirror after a sync**, not from the bytes that just passed through
  it: after the walk completes, the frame re-reads the store so that what is on screen is exactly
  what is on disk. This is testable and it is the difference between a mirror and a cache.
- **Rows marked `present = 0` never render**, and they are **not counted on screen** either: the
  store is where "gone from x.com" is recorded, and `doctor` is where it is reported. A frame that
  counted them would be inventing a second number the mirror's own reader does not have. A key to
  reveal them is deferred.
- **Each run is recorded**, so the status line (and, in Phase 8, `doctor`) can say when the mirror
  was last synced and why the walk stopped, without keeping a log of what it saw.

### The likes endpoint: build the screen, expect either answer

- `Likes` is in the live bundle with a captured id and 40 feature switches (measured in Phase 1),
  but whether the server answers for the **viewer's own** likes is the one `[UNVERIFIED]` surface in
  this project. The plan makes that call once, with the real session, and records the outcome in
  the plan's summary and in the likes fixture — it is not a decision to be argued in advance.
- **The screen exists either way and degrades through a capability state, not a guess.** An
  operation id the build does not hold is *not* a classifier outcome — no request was made and none
  can be — so it gets a state of its own in the same two status-line slots, with the remedy that
  repairs it: `likes unavailable` / `bookmarks unavailable`, remedy `doctor --pin, then restart`
  (the pin file is read at every start). The closed `domain.FailureKind` taxonomy is **not** widened
  for it: a "we never dialled" state posing as a "we dialled and this happened" one is exactly the
  confusion the taxonomy exists to prevent.
- **The app never editorialises about the unverified endpoint.** It states the presence fact it
  actually has and the repair, and nothing else — no "this endpoint may not exist", no hedge in the
  empty state, no speculation in the notice. An empty *success* read is the ordinary empty list
  (`no posts in likes` plus Phase 2's body line); a failed read over an empty mirror is the state
  block, never the empty copy. Both come from Phase 2's vocabulary; this phase writes no second
  sentence for either.
- **Nothing else in the app changes when likes are unavailable**, and that is a test rather than a
  hope: the bookmarks path, the timeline and every frame behave identically whether the operation is
  present, missing or refused, because the mirror must stay worth having with bookmarks alone (the
  roadmap's own sequencing requirement).
- The mirror's schema keeps both kinds from the first migration whatever the answer turns out to
  be: the availability of an endpoint is not a schema question, and a v2 migration to add a kind
  would be a repair in the middle of a working install.

### The pin override (XAPI-04)

- The mechanism is Phase 1's and is not rebuilt here: `doctor --pin <operation> <queryId>`, backed
  by `pins.json` under the data directory, with per-field overlay semantics (a field the pin names
  wins; a field it omits keeps the snapshot's), shape validation on both the operation name and the
  id, and an atomic write that refuses to lose a pin it did not understand.
- **This phase exercises it end to end for the operation it exists for.** `Bookmarks` is the
  operation the bundle cannot see — it is registered from an external pinned source with no feature
  switches — so the acceptance is: pin a deliberately wrong id and the failure must be the *named*
  stale-operation state (never an empty list); pin the correct id and the sync succeeds; no code
  changes at either step. `Likes` gets the same treatment only if the live measurement says its
  registry entry is missing or stale.
- What was pinned, when, and with which id is recorded in the runbook's operation-rotation entry
  (Phase 8) rather than in a second file this phase would have to maintain.

### The proof (SEC-02's three legs)

- **The type.** The store's write API accepts `domain.SavedItem` and nothing else; there is no
  generic write. Enforced by the package's exported surface, not by review.
- **The schema.** An allowlist test reads `sqlite_master` and `PRAGMA table_info` and asserts the
  file contains exactly the declared tables and columns — no extra table, no column added later, and
  nothing generic (`blob`, `payload`, `extra`, `data`) that could hold anything but a saved post.
- **The import boundary.** An AST scan asserts that no file which handles home, thread or search
  pages imports `internal/store`: not `internal/xsource`, not the timeline/thread/search files in
  `internal/screen`, not `internal/render`, and not `internal/tui` (which Phase 2 already forbids
  from `database/sql`). The scan carries anti-vacuity arms — it must find the files it inspects and
  must fail when the import is added to a fixture of its own.
- **The behavioural leg.** A test drives a full timeline session through the fakes, then inspects
  the data directory and asserts it contains only `saved.db` and the files that existed before it
  (the pins and the notice marker). The pitfalls checklist asks for this "by inspection after a full
  session, not by intent"; this is that inspection, mechanised.

### Keys, the frame, and the honesty rules

- Two new keymap rows, both from the single table: `b` — `open bookmarks`; `l` — `open likes`. Each
  pushes a saved frame, exactly like every other list, and `q`/`esc` returns to what the user was
  reading, at the row they left.
- Inside a saved frame, `r` is the sync (Phase 2's key, Phase 2's meaning), and the saved frame's
  posts render as the ordinary post blocks — a bookmarked post looks exactly as it does in the
  timeline, because it is the same post.
- **Where a frame's rows came from is a status-line fact, not a notice.** A saved frame that has
  read the mirror carries a `source` field (`from disk`), whose literal belongs to the display
  grammar (a cover must be able to say it in its own fiction); a frame that has only ever read the
  network carries nothing, and Phase 2's "an empty field is omitted entirely" makes that free. No
  frame renders a saved row as though it came from the network, and none claims the network's end of
  list at the mirror's end.
- No counts anywhere: a saved list is not a leaderboard (REND-01).
- **The "does not apply" rule (Phase 4) is how the empty and unavailable cases are handled**: an
  inapplicable key is a silent no-op; a key whose action has nothing to act on reports one honest
  line and issues no request — `r` with no session, `l` with no likes operation in the build.

### Parallel safety (the files this phase touches, and how)

- New files only: `internal/store/{schema.go,migrate.go,sync.go,read.go}` and their tests (a new
  package, so nothing else in the tree is touched to create it), `internal/domain/saved.go` (the
  `SavedItem` value), `internal/xsource/saved.go`,
  `internal/screen/saved.go`, `internal/render/saved.go`, `internal/fake/saved.go`,
  `internal/cli/sync.go` (plus one row in the existing command table), and this phase's fixtures
  under `internal/xsource/testdata/` and `internal/fake/testdata/`.
- Shared append-only: the `Source` port grows one method (`Saved`, in `Home`'s shape) in its existing
  file, with the fake growing the matching method in its own file; the keymap table gains two rows;
  the shell's frame dispatch gains one frame kind among the shared edits.
- `go.mod` gains `modernc.org/sqlite` — the only dependency this phase adds, fetched through the
  local proxy like every other module on this host.
- The capture harness and its redaction rule are Phase 4's; the fixtures for bookmarks and likes go
  through the same gate, and the redaction matters more here than anywhere else in the project:
  these responses carry **the viewer's own account** and the exact set of posts they saved. The
  checked-in fixture keeps the response's *shape* — entries, cursors, item kinds, the module
  structure — with the viewer's identity rewritten and the post content replaced by synthetic text
  of the same lengths, and a test asserts neither the real handle nor any captured post body appears
  under `testdata/`.

### Needs the user

- **The mirror's owner after an account change.** The requirements are silent, and the situation is
  likely rather than theoretical: `auth_token` and `ct0` are session cookies that die with the
  browser, so re-exporting a cookie (possibly for a different account) is a routine act, while the
  mirror outlives it.
  - Chosen: the mirror records the account it was synced from (`meta.owner`); a session for a
    different account makes `sync` refuse with a message naming the mirror's owner, while the
    offline read keeps working. Re-pointing the mirror is deleting `saved.db`, which the runbook
    states in one line.
  - Alternative: treat a new account as a fresh mirror — which either deletes rows (destroying a
    mirror the user may still want to read) or mixes two accounts' rows under one `present` flag (a
    lie about whose list is on screen).
  - Consequence of the choice: after an account change, one command (or a read-only mirror) rather
    than an automatic rebind. Consequence of the alternative: a destructive or quietly-mixed file
    that the pane cannot describe honestly. Flagged because the decision is visible to the user and
    only they can say which trade they want.

### Claude's Discretion

- The exact DDL, index set and pragma list; the shape of the sync's result value; whether `sync`
  grows `--json`/`--quiet` (consistency with `doctor` argues for both); the page cap for a full
  re-list; the wording of the absent-row hint; how much of the post snapshot is stored versus
  re-derived at read time; the fixture set's size and split.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/domain` — `Post`, `User`, `Cursor`, `Failure`; post identity is already the key for
  dedupe, selection and the refresh diff, which is exactly what a saved item's primary key needs.
  The new `SavedItem` (`Kind`, `Post`, `SortIndex`, `FirstSeen`, `LastSeen`, `Present`) is a new
  file in this package.
- `internal/screen` — the shared walk (Phase 4) and its `Event`/`Effect` values; the saved frame is
  a walk whose pages come from the mirror as well as from the network, which the walk already
  supports because it only ever asks for "the next page" as a value.
- `internal/xsource` — the timeline decoder reads a bookmarks or likes page unchanged: the research's
  endpoint table gives both the same "bottom cursor from the last item" semantics as the home
  timelines, and the items are ordinary tweet entries. Only the operation names and the variables
  differ, and the adapter carries no state.
- `internal/xweb` — the single choke point, the classifier, the budget line and the registry with
  its `pins.json` overlay: the sync path uses all four and adds none.
- `internal/config` — `DataDir` is already resolved and is where `pins.json` and the notice marker
  live; the store's path belongs beside them, built by this package so the sync, the pane and
  `doctor` cannot disagree about where the mirror is.
- `internal/cli` — the command table (one new row), the failure mapping (no new case), and
  `doctor`'s presence-facts shape, which Phase 8 extends with the store's line.
- `internal/fake` — the fixture-and-counter pattern for the new saved source; `internal/text`,
  `internal/render` and `internal/tui` as Phase 2 delivers them, unchanged.

### Established Patterns

- Every read is a `tea.Cmd`; the shell may not import `database/sql`, `internal/store`, `net/http`,
  `os/exec` or `internal/xsource` — Phase 2's boundary test already names the store, and this phase
  is where that line starts to matter.
- A page that fails keeps everything already rendered and puts the classifier's words on the status
  line; the mirror is not modified by a failed page.
- Ordered migrations with the version bump inside the same transaction, the writer/reader split and
  the read-back of `journal_mode` all come from the sibling project's store as a pattern (not as
  code, per the no-shared-code decision).
- Fixtures are recorded, redacted, checked in with provenance, and drive the decoder tests with no
  network and no session; the fake has no clock and no I/O.
- `doctor` prints presence facts and never a token; the store's line is a path, a schema version, row
  counts and a stop reason — never a post body.

### Integration Points

- `internal/store` is a new leaf package with a one-way API; nothing about timelines or threads can
  be written because no function accepts those types.
- `internal/cli` — `sync` is a new verb body beside `doctor`, `whoami`, `logout`, `keys` and `feed`,
  registered in the same table, and the composition root is the only place that opens the store.
- `internal/tui` — one new frame kind and its dispatch; the frame's rows are the walk's rows, so the
  cover (Phase 3) applies to a saved list exactly as it does to the timeline.
- `internal/xweb/volatile` — `Bookmarks` is the operation the refresh path cannot supply and the pin
  path exists for; the plan must show both halves of that (a wrong pin failing loudly, a right pin
  working) rather than only the happy path.
- Phase 8's `doctor` reads the store's schema version and the mirror's owner from `meta` and prints
  them; this phase must therefore record them where a second process can read them without opening
  the timeline (which does not exist).

</code_context>

<specifics>
## Specific Ideas

- The mirror's whole value is the offline read: opening bookmarks with the network down must work
  the first time, with no spinner, no request and no "reconnecting" state — the rows are simply
  there.
- `sync`'s output should be quotable and boring: one line per kind, pages walked, items added, stop
  reason, and the mirror's path. Never a post body, never a token.
- The store must stay inspectable with the installed `sqlite3` CLI (`sqlite3 saved.db '.tables'`,
  `.schema`): a debugging affordance that costs nothing and is part of why the stack research chose
  SQLite for this.
- A saved list should read like the timeline, because it *is* the timeline's rows: nothing in the
  frame should tell the reader they are looking at a mirror, except the status line telling them
  when it was last synced.
- `present = 0` must never be silent: hiding a row the user once saved without saying how many are
  hidden is the same class of lie as rendering an empty list for a failure.
</specifics>

<deferred>
## Deferred Ideas

- A key to reveal `present = 0` rows (the "gone from x.com" list): the mirror keeps them from day
  one; the key is a later decision.
- Local filtering and keyword search over the mirror — the research notes it as a benefit of having
  a mirror at all, and it is a second query surface (and a second thing to test) that v1 does not
  need.
- Export or pipe of the mirror (EXP-01 is v2): the file is already there and inspectable with
  `sqlite3`, which is the v1 answer.
- Any scheduled, background or on-open sync: the anti-feature list forbids it, and the explicit key
  is the product's promise.
- Un-bookmarking, liking, or saving from the terminal: a write path, out of scope by the project's
  first decision.
- More than one account per mirror, or a per-account mirror: multi-account is out of scope; the
  owner record exists to refuse mixing, not to support it.
- Likes for other accounts, or a likes tab in a profile: x.com withdrew the public surface; not a
  client feature.
- The replies and media tabs of a profile (`UserTweetsAndReplies`, `UserMedia`): deferred with
  Phase 5's profile work.
</deferred>
