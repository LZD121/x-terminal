# Phase 4: Post Detail, Threads and Quotes - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — proposals accepted in batch under the run-wide policy

<domain>
## Phase Boundary

The user opens a post and reads its conversation — replies a page at a time, quotes inline,
tombstones and unavailable replies labelled honestly.

Delivers: the `TweetDetail` conversation decoder (the entry grammar: ancestors, the focal post,
reply modules with their own cursors, `SelfThread`, visibility wrappers, tombstones, promoted
entries), the `Thread`/`Group` domain values it produces, the thread frame on Phase 2's existing
navigation stack, the selection widened so a quoted post can be chosen and opened with the same one
key, inline quote rendering in the thread, the honest `partial thread` state, and the
recorded fixtures the decoder is driven from.

Does NOT deliver: search, profiles, bookmarks or likes (Phases 5 and 6); opening media or links
and their cover gate (Phase 7); engagement counts anywhere in this screen (REND-01 — see *Counts*);
the related-tweets rail, which is dropped deliberately; a reply-depth setting (CFG-01 is v1.x); or
a second walking implementation (the list machinery is shared — see *The walk*).

</domain>

<decisions>
## Implementation Decisions

### The conversation decoder

- **One operation, one response family.** `TweetDetail` is the only new operation this phase
  reads (`internal/xweb/volatile/registry.json` already carries it: `3BlMtudy4N-d78ch7zTLZQ`,
  mined from the bundle). The `Source` port grows one method — `Thread(ctx, focalID, cursor)
  (ThreadPage, error)` — and the variables it sends are the adapter's business below the port, as
  with `Home`.
- **The focal post is not fetched.** Opening a post renders from the copy the caller already
  holds — the list row, or the parent's nested quote — so opening is instant, costs no request and
  cannot fail. The first `TweetDetail` page then carries the same post in full and replaces that
  copy. There is therefore **no `Post()` / `TweetResultByRestId` read** in this phase and no
  single-object operation in the port; a second request for something already on screen would be
  the N+1 anti-pattern one level up.
- **The new code is the entry grammar, not a second tweet decoder.** `internal/xsource/tweets.go`
  already knows how to read a tweet: `unwrapTweet` (the `TweetWithVisibilityResults` /
  `TweetTombstone` wrappers, one level deeper than usual), `postFrom` (repost, quote with its
  bounded depth), `tombstone`, `mediaOf`, `countsOf`, `authorOf`. The detail walker calls those and
  adds only the classification of *entries*: `cursor-*` entries are the page's own cursors;
  entries before `tweet-<focalID>` are the **ancestor chain**; `tweetdetailrelatedtweets*` is a
  separate rail; `tweetcomposer`-bearing and `messageprompt-` entries are not content; conversation
  modules carry reply groups and, inside them, a cursor that is a **`ShowMoreThreads` handle**.
  A second tweet decoder would drift from the first one on the first payload change — the reason
  this package exists in one piece.
- **Unknown entry classes are skipped and counted; unknown fields inside a tweet are typed
  errors.** The two rules are different on purpose: the entry stream is a ranked, ad-laden
  container that ships new decorations without notice (a skipped decoration must not become a
  blank row, and `PostPage.Requested` already counts only entries that could become posts), while a
  tweet whose body, author or date moved is the silent-wrongness failure mode this package refuses
  to guess at (`ParseError` names the path). The one entry-level error is the **missing focal
  post**: a page one without it means the response answered a different question, and the page is
  refused rather than rendered as an empty conversation.
- **The related-tweets rail is dropped, not deferred to a key.** It is an algorithmic
  recommendation surface; rendering it would put "posts you did not ask for, chosen for you" on
  screen, which is the behaviour this project exists to escape, and it would also be a second list
  inside the thread frame with its own identity. The decoder classifies those entries so the count
  is honest, then drops them, and a test asserts no post from that rail reaches a frame.
- **`SelfThread` is a group kind, not a nested conversation.** The focal post and its own
  immediate replies are one authored block (the upstream says so with `tweetDisplayType`), and the
  renderer prints them as a continuation of the focal block using the `Continued` marker Phase 2
  already defines. No new layout primitive, and no pretending that a self-reply is a stranger's
  reply.
- **One cursor for the walk.** The adapter hands back the page's bottom cursor when the answer has
  one, and otherwise the module's own `ShowMoreThreads` handle; either way it is opaque and the
  screen only stores it. A group that still carries an unspent per-module cursor is *not* the end
  of the conversation, and the walk's stall rules are Phase 2's rules verbatim: a repeated cursor or
  a page that adds nothing unseen ends the walk and says so.
- **Fixture-driven, and the fixture does not exist yet.** Phase 1 recorded `UserTweets` only, so
  the first task of this phase is an operator-run capture, not an assumption: a gated test
  (`internal/xsource/capture_test.go`, opt-in through `X_TERMINAL_CAPTURE=1` with a live session)
  issues **one** request through the existing `xweb.Client` — so the limiter, the floor, the budget
  and the classifier all apply — and writes the body into `internal/xsource/testdata/`. The
  fixture set must contain, as a list that the decoder's tests name: a reply chain with ancestors
  above the focal post, a `SelfThread`, a module carrying its own cursor, a
  `TweetWithVisibilityResults` wrapper, a tombstone, a related-tweets rail, a composer entry, and a
  nested quote. A synthetic fixture covers the one shape no recording has (a withheld post), and
  `testdata/provenance.json` records per file the operation, the date, the query id at capture time
  and the redaction applied.
- **The capture is redacted before it is checked in.** The capture step rewrites the *viewer's own*
  identity (their id and handle) to fixed placeholders and leaves the public post content alone;
  a test asserts the real handle appears nowhere in `testdata/`. The cookie never enters the file
  for the same reason it enters no log (SEC-01). This matters more in Phase 6, where the captured
  list is the user's own; the rule and the test land here.

### The thread frame on Phase 2's stack

- **The thread is a frame, exactly like the feed is.** Opening a post pushes a frame whose content
  is `{ancestors, focal, groups, cursor, selected post id, anchor}`; the frame underneath is frozen
  (Phase 2's rule) and `q`/`esc` pops it. Restoring means Phase 2's four observables unchanged: the
  same post ids in the same order with **zero** new requests, the same post id selected (not the
  same row index), the selected block's head row back at the same pane row, and nothing else
  restored.
- **Opening a post from inside a thread pushes another thread frame** — a reply's own conversation,
  or the quoted post's. There is no artificial depth cap and no cycle guard: the user can only
  descend into something they selected, the stack is a slice, and the process request budget
  (XAPI-07) is what bounds the walk. Opening a post that is already the current frame's focal post
  is a no-op.
- **Ancestors, the focal post and the replies are rows in one pane, and the selection is a
  `Target`** — a row's identity plus the chain of containing posts for a post quoted inside another
  (the value Phase 4 defines; it is Phase 2's bare post id widened, and that widening is what makes
  a quote selectable). The selectable order is the frame's row order, and the pagination trigger
  stays implicit and unchanged: reaching the last loaded row of the *reply stream* asks for the next
  page, single-flight, and a press while a page is out is a no-op. Reaching an ancestor row is not a
  trigger — the ancestors arrived with page one and are not a walk.
- **Display names are printed by no grammar this phase ships.** 02-CONTEXT deferred them "to Phase
  4's detail header"; the phase's UI-SPEC resolves the other way and this file records that
  resolution: the thread's identity column stays the handle, and the display name's single home is
  Phase 5's profile header. A display name identifies a person rather than a post, a byline column
  in a conversation has one job, and a second identity form in every block would be a second thing
  every cover grammar has to alias.
- **The thread screen owns no new vocabulary for failure.** A failed page keeps everything already
  rendered and puts the classifier's own sentence on the status line (FEED-05's rule, same
  reducer shape). A thread that failed on its very first page is the same state, not an empty
  conversation.

### Honesty on the thread screen

- **Tombstones and limited posts render as themselves, in place, and stay selectable.** `Visibility`
  already models all four states and the timeline decoder already produces them; the thread screen
  renders Phase 2's `Unavailable`/`Limited`/`Withheld` markers and keeps the row's id and its place
  in the conversation. `enter` on a tombstone opens that post's own conversation — a deleted post
  can still have replies worth reading — and nothing about a tombstone is an error or raises a
  failure state.
- **A page that was ranked and truncated server-side is the normal state of this endpoint, not an
  error.** The walk ends when the cursor says so, and the pane's job is to not imply completeness it
  cannot see.
- **`partial thread` is printed only when the frame can name *why* the thread is incomplete**, never
  as a standing disclaimer. The phase's UI-SPEC fixes three conditions, and they are adopted as
  written:
  1. the reply walk stopped on a cursor the upstream would not advance (Phase 2's stall detection,
     reused verbatim) — the reply walk stopped here;
  2. the focal post answers a post the frame does not hold (`Focal.ReplyTo` names an id in neither
     the ancestors nor the focal post) — the post above is not reachable;
  3. the answer carried no replies, no reply cursor, **and** the focal post's own reply count is
     greater than zero — replies are not reachable. The count is *consulted as evidence and never
     printed* (`REND-01`); if the recorded fixture carries no such field, this condition is dropped
     rather than guessed.
- **Not conditions, deliberately:** the end of the reply list prints nothing (the upstream says
  there is nothing more, and the frame repeats that by saying nothing); a failed page prints only
  the failure's own line; a limited or withheld post inside a thread is a per-post state, not a
  statement about the thread. Replies filtered upstream are not distinguishable from replies that do
  not exist, and the pane does not pretend otherwise.
- **A failure raised while a thread frame is stacked stays on the status line when it is popped** —
  Phase 2's rule that notices, the in-flight latch and the failure state are per-process, not
  per-frame. Nothing in this phase changes that.

### Counts (a document conflict, resolved)

- **No engagement counts are printed on the thread screen** — not on the focal post, not on a
  reply, not in the ancestors. `REND-01` is explicit that list *and detail* screens carry no
  engagement counts, and `DISC-03`'s profile counts are a separate, required exception that Phase 5
  owns. The Out-of-Scope table's note that "in detail they print as plain text" is the other
  reading, and it is superseded by the requirement's own wording.
- The consequence is recorded rather than hidden: the `partial thread` label is derived from a
  count without printing it, and a reader who wants the numbers cannot get them here.
  *Flagged for the user below*, because the two documents disagree and the user wrote both.

### Keys and rows

- **One new row, and one new field on the row value.** `enter` — `open` — `open the selected post or
  account`, in the table in `internal/keys` that feeds input, the `?` overlay and the `keys`
  subcommand. There is no second quote key: the selection is a `Target`, so a quoted post *is* a
  selectable row and `enter` opens it as its own conversation (`THRD-02`). One key cannot carry two
  actions in one table — the matcher is a linear scan — so THRD-02 is satisfied by making the quote
  selectable rather than by adding a key.
- The row value gains a **`screens` field** (`all`, or a set drawn from `{home, search, thread,
  profile}`): a row that does not apply to the top frame is not dispatched, so no key is documented
  that does not work. This phase also rewrites the `t` row *in its final form* (`switch-list`,
  `switch what this list shows`, screens `home search`) so that Phase 4 and Phase 5, which run in
  parallel, never edit the same row twice; on this phase's frames it behaves as Phase 2's
  `toggle-feed` did.
- **The "does not apply" rule, stated once for every later phase.** An inapplicable *row* is a
  silent no-op (there is nothing to say); an applicable key whose action has nothing to act on
  reports one honest line and issues no request. This phase's instance is `enter` where there is
  nothing to open; Phase 6's and Phase 7's rows inherit the rule and their own UI-SPECs name their
  own copies.
- **Key collisions with Phase 3 are resolved in the cover's favour.** Phase 3 lands first and its
  state-switching keys must work in every state; if one of them is `enter`, the row here moves
  rather than the cover key. The allocation is settled at plan time, against the keymap table as it
  actually landed.

### The walk: one machinery, never a second one

- Phase 2's `screen.List` is a *post* list whose machinery is element-specific: it absorbs
  `[]domain.Post`, keys on `PostID`, and its `Effect` names a home feed. This phase, Phase 5 and
  Phase 6 all need that machinery for other content (this phase's reply pages, Phase 5's users and
  search results, Phase 6's saved items). Forked copies of dedupe, stall detection, single-flight
  and the refresh diff are exactly the duplication the roadmap's sequencing exists to prevent, so
  the rule is: **one implementation of the walk, extended in place, never a second one.**
- The split of that work is the one the phases' UI-SPECs draw, adopted here because it keeps each
  phase's diff to what that phase actually needs: **Phase 4 widens the selection to `Target`** (so a
  quote is openable from every list, at the stated cost of one reducer refactor plus its tests) and
  builds the thread frame on the same reducer; **Phase 5 performs the entity-blind extraction** (a
  row's id and kind, the shared list state) that makes a People result list and a post list the same
  machinery; **Phase 6 adds its rows to it.** Inventing a generic type in this file would be a third
  vocabulary for the same decision.
- The coordination requirement that follows: the files in `internal/screen` carrying the walk are
  the **only files in the fan-out that are not append-only**. Whichever of Phases 4–7 executes first
  lands its change and freezes that shape; the others build on what is there, and if the phases are
  ever run in parallel worktrees these files land in numeric order rather than concurrently.

### Parallel safety (the files this phase touches, and how)

- New files only, in every package this phase extends: `internal/xsource/detail.go` (+
  `detail_test.go`, `capture_test.go`, `testdata/fixture-tweetdetail-*.json`,
  `testdata/provenance.json`), `internal/domain/thread.go`, `internal/screen/thread.go`,
  `internal/fake/thread.go` (a fixture-backed thread source of its own, rather than growing the
  shared `fake.Source` struct), `internal/render/thread.go` (the thread frame's view).
- Shared files touched, append-only, one row each: the keymap table (`internal/keys`), the shell's
  frame dispatch (`internal/tui`), and the `Request` union inside `screen/walk.go`.
- `internal/domain/domain.go`, `internal/domain/source.go` and `internal/xsource/home.go` are **not
  edited**: new capabilities go in new files inside those packages (methods may live in any file
  of the package), which is what keeps Phases 4–7 from colliding.

### Needs the user

- **Engagement counts on the detail screen.** Two documents disagree: `REND-01` forbids counts on
  list and detail screens, while the Out-of-Scope table says "counts are dropped in the list view;
  in detail they print as plain text with no glyph and no colour".
  - Chosen: follow `REND-01` — no counts anywhere on the thread screen.
  - Alternative: one plain-text counts line on the focal post only (one `Grammar.ShowCounts` flag
    and one marker literal; the list stays count-free either way).
  - The consequence of the choice: a reader of a thread cannot see how many replies or likes a post
    has without leaving for the browser. The consequence of the alternative: the detail screen
    carries a number whose only purpose on x.com is engagement signalling, and the golden frames
    gain a line the requirement asked not to have.
  - Cost to change later: one flag plus one golden-frame regeneration. Recorded rather than guessed
    because the requirements are the user's own words and they conflict.

### Claude's Discretion

- The exact marker literals for a continuation, a tombstone and a limited post (Phase 2's
  `MarkerSet` already has slots for all three); whether ancestors are capped and at what count; the
  precise wording of the `partial thread` state and its remedy column; the internal shape of the
  group model beyond the fields named above; how the fixture set is split across files; and whether
  the capture harness writes one body per fixture or one body carrying several shapes.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/domain` — `Post` (with `Quote *Post`, `ConversationID`, `ReplyTo`, `Visibility`,
  `Media []MediaRef`, `Counts`), `User`, `PostPage`, `Cursor`, `Visibility{Visible, Limited,
  Unavailable, Withheld}`, `Failure`/`FailureKind`, and the `Source` port. The thread's new values
  (`Thread`, `ThreadPage` — the port's return — `Group`, `GroupKind`) belong in a new file in this
  package, and the port itself grows one method beside `Home` and `User`.
- `internal/xsource` — `ParseError` (path-naming), `GraphQLer` (the one-call seam), `Adapter` over
  the recorded fixtures, and the tweet decoder (`postFrom`, `unwrapTweet`, `tombstone`, `mediaOf`,
  `countsOf`, `authorOf`, `ParseTimeline`) which this phase's entry walker reuses rather than
  re-implements.
- `internal/xweb` — `Client.GraphQL(ctx, opName, variables)`, the single request choke point with
  the 1.5 s floor and the 200-request cap, the measured classifier's outcome vocabulary, and
  `volatile.Registry` (107 operations, `TweetDetail` among them) with the `pins.json` overlay.
- `internal/screen` — `List`, `Effect`/`EffectKind`, the sealed `Event` set and the stall/stop
  rules, with `internal/fake`'s fixture-backed source and its call counter as the test harness.
- `internal/text` — the only width measurement; `internal/render` and `internal/tui` as Phase 2
  delivers them (still in flight at the time this was written): copy, `Grammar`, block, status,
  frame, palette, help, and the shell's frame stack with its anchor.
- Phase 1's fixtures and evidence: `internal/xsource/testdata/fixture-usertweets.json`,
  `internal/fake/testdata/pages.json`, `.planning/research/UPSTREAM-PROBE.md`, and the registry's
  provenance record — the model for what `testdata/provenance.json` should say about this phase's
  capture.

### Established Patterns

- Every read is a value returned from `Update` and performed by the shell as a `tea.Cmd`; the
  reducer never touches `net/http`, a file or a clock it does not take as an argument.
- The port is declared by the consumer and implemented at the composition root; no upstream field
  name, operation name or query id may reach `screen`, `render` or `tui`.
- A response field that moved is a typed `ParseError` naming the path; a decoration the entry
  stream grew that the decoder does not know is skipped, and the count of what was skipped is
  visible in `Requested`.
- Golden frames at 80 and 120 columns, inside and outside tmux, are the view specification; a new
  frame in the product is a new golden frame, including its adversarial cases (a tombstone row, a
  limited row, a quote nested two deep, an ancestor chain, a `partial thread` status line).
- The fake source counts calls, so "popping a frame issues no request" and "opening a post issues no
  request of its own" are assertions, not intentions.

### Integration Points

- `internal/cli` — the composition root maps the thread screen's `Effect` to `Adapter.Thread` and
  feeds the answer back as an event; the failure mapping (`internal/cli/failure.go`) already turns
  every transport error class into a `domain.Failure` and needs no new case.
- `internal/xweb/volatile` — `TweetDetail` is already registered; if a bundle refresh ever drops
  it, the failure is the registry's typed `OperationStale`, never a silent empty thread.
- `internal/keys` — two rows; `internal/tui` — one frame kind and one dispatch line; these are the
  only shared edits this phase makes, and both are append-only.
- `internal/render` — the quote block's nested layout exists (Phase 2's depth-2 quote, `QuotePrefix`
  and `QuoteDepthCap`); the thread frame reuses the same composer, which is what makes the cover
  (Phase 3) apply to it for free.
- `.planning/config.json` and `go.mod` are untouched: no new dependency, no new configuration key.

</code_context>

<specifics>
## Specific Ideas

- Opening a post must feel instant: the focal block is on screen in the same frame as the keypress,
  with the conversation below it in a loading state — never a blank pane and never a spinner that
  means "hung".
- A quote is rendered in place, indented, never elided and never expanded into a second top-level
  row; it is a post the reader can step into with one key, and stepping back out returns to exactly
  the frame they left.
- The ancestor chain above the focal post is context, and it is worth one row of separation: the
  reader should be able to see at a glance where the conversation came from before the post they
  opened.
- `partial thread` is copy, not a warning: the tone is the status line's ("no older posts") — a
  quiet statement of what the pane knows, in the same words on every run.
</specifics>

<deferred>
## Deferred Ideas

- The related-tweets rail: dropped in this phase by decision, not deferred to a key. If it is ever
  wanted it is an anti-feature under this project's own Out-of-Scope list ("algorithmic feed as
  default").
- A per-group "show more replies" key (walking one module's cursor without walking the page): the
  page walk covers the case; a second trigger would be a second cursor surface and a second key.
- Reply/thread page depth in the config file (CFG-01) — v1.x, with its named trigger ("one-page
  replies feel too shallow or too slow"). This phase fixes the page size at X's own default.
- A "jump to the root of the conversation" key and a full-conversation view rendered from the root:
  the ancestor chain is already shown; the root view is a different walk.
- Opening media or links from a thread — Phase 7 owns the action and its cover gate; the attachment
  marker line is Phase 2's.
- Any rendering of engagement counts (see *Counts*), pending the user's reading of the conflict.
</deferred>
