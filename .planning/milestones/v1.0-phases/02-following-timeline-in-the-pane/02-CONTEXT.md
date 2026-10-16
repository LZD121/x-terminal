# Phase 2: Following Timeline in the Pane - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — proposals accepted in batch under the run-wide policy

<domain>
## Phase Boundary

The user reads their Following timeline in a terminal pane — scrolling back through older posts
and refreshing deliberately — and the pane never lies about what it is showing.

Delivers: the Bubble Tea v2 application shell, the domain model and its consumer-side `Source`
port, the paginated list screen, the pure-`Update`/`View` discipline with I/O expressed as
command values, the single width model, the colour ladder, golden-frame tests at 80 and 120
columns inside and outside tmux, the keymap table that feeds input + help + a `keys` subcommand,
the navigation shell with a real back stack, the four named failure states rendered in one
status line, and the request-budget line wired to the existing limiter.

Does NOT deliver: post detail or threads, search, profiles, bookmarks/likes, the store, media
opening, or the disguise modes. Those are later phases; the timeline is the slice that proves the
rendering, pagination and request-discipline machinery every later surface reuses.
</domain>

<decisions>
## Implementation Decisions

### Data access

- `internal/domain` **declares** the `Source` port (consumer-side, as `rss-tm` does with
  `FetchFunc`); `internal/xsource` implements it. The port's shape: cursor in, page out, plus
  `User(ctx, handle)` — nothing screen-specific.
- Cursors are opaque strings; identity is the post id. Dedupe is an id set **plus**
  `(cursor, entry-id)` stall detection, because both "infinite loop on a repeated cursor" and
  "silently stops on an empty page" are real failure modes of this upstream.
- Pagination has **no key of its own**: reaching the last loaded post requests the next page.
  This keeps the keymap small and matches how the reader actually travels.
- Refresh is `r`: fetch page one, diff its ids against the visible head, and show
  "`{n}` new posts" instead of replacing what is on screen. Selection and scroll do not move.
- A fake `Source` implementation backed by the recorded fixtures
  (`internal/xsource/testdata/`) ships with the phase, so every screen is testable with no
  network and no session.

### Rendering and width

- `internal/text` owns `Width`/`Wrap`/`Pad`/`Truncate` and is the **only** package allowed to
  measure text; an AST import-boundary test forbids `len`, `utf8` and fmt width verbs elsewhere
  in the render path. Width dispatches on `$TMUX`: wcwidth semantics under tmux, grapheme
  semantics in a bare Ghostty — the split the research measured.
- Measure is capped at 96 columns so a wide terminal does not produce unreadably long lines.
- Colour is a role table with a degradation ladder truecolor → 256 → 16 → mono, honouring
  `NO_COLOR` and `TERM=dumb`. **Meaning never rides on colour alone** is asserted by tests, not
  by style.
- Golden frames at 80 and 120 columns, captured inside and outside tmux, are the view
  specification (`charmbracelet/x/exp/golden`), with the adversarial content list from the
  UI-SPEC (15-cell handle, CJK, emoji, ZWJ family, over-long URL, exact-fill line, depth-2 quote,
  tombstone, selection not at row 0, three-digit budget).

### Shell and keys

- `internal/tui` is pure `View`/`Update`; every read is a `tea.Cmd`. An import-boundary test
  forbids `net/http`, `os/exec`, `database/sql`, `internal/xsource` and `internal/store` from
  reaching it.
- Keys, the `?` help overlay and a `keys` subcommand are all derived from **one** keymap table,
  so a new key is one row and the three surfaces cannot drift.
- The navigation shell and the back stack land in this phase, not with the first drill-down:
  Phases 4–7 run in parallel and all push and pop the same stack.
- Exactly one text attribute exists in the whole surface (reverse video on the selected row);
  the `>` column carries selection when the colour rung would not.

### State and honesty

- The status line is a single row of `{state} {remedy} {notice} {list} {position} {budget}`,
  never wrapped, with a defined drop order under width pressure: `notice → position → list`,
  never `state`, `remedy` or `budget`.
- The four failure states reuse `internal/xweb`'s own `Outcome.Error()`/`Remedy()` strings, so
  the words on screen and the words in the classifier cannot drift. No failure may render as
  nothing.
- The budget line is `{n} requests, {m} rate-limited`, read from `Limiter.Budget()` — which is
  why the limiter is the only path to the upstream.

### Dependencies and build

- Bubble Tea v2 `v2.1.0` (`charm.land/bubbletea/v2`), plus `lipgloss/v2`, `bubbles/v2` and
  `charmbracelet/x/ansi` for width. **Verified 2026-10-14:** the Go module proxy is unreachable
  from this host directly, but works through the Clash proxy, so module-fetching commands must
  run with `HTTPS_PROXY=http://127.0.0.1:7897` (and `HTTP_PROXY` likewise). The module cache
  makes this a once-per-module cost.

### Claude's Discretion

Package-internal layout, test naming, and the exact split of `internal/screen` reducers vs
`internal/tui` glue are at the implementer's discretion within the decisions above.
</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

- Phase 1 delivered and verified: `internal/config`, `internal/session` (redacting credential),
  `internal/xweb` (proxied transport, measured classifier, single request choke point with a
  1.5s floor and a 200-request cap), `internal/netcheck`, and `internal/cli` with `doctor`,
  `whoami` and `logout`.
- `internal/xweb/classify.go` is the source of the failure-state copy; read it rather than
  inventing wording.
- The UI-SPEC for this phase (`.planning/phases/02-following-timeline-in-the-pane/02-UI-SPEC.md`)
  is the design authority: post block layout, status line fields and drop order, the 13-row
  keymap table, colour roles, the width-model properties A4–A7, and the golden-frame checklist.
  It also draws the content/presentation split that Phase 3's covers will ride on.
- `/home/user/repos/rss-tm` remains a pattern reference only (no shared code): pure `Update`
  returning effects, the AST boundary test with anti-vacuity arms, golden frames as spec,
  consumer-side ports.

### Established Patterns

- Live response shapes: the **new** schema (`core`/`avatar`/`id`), never `legacy`. Fixtures for
  the parser live in `internal/xsource/testdata/`.
- Request mechanics: bearer + cookie + `x-csrf-token` on API calls; cookie-only on navigation
  requests. `Authorization` on a navigation request is rejected with `401`.
- The limiter is the only send path; a second concurrent request waits rather than failing.

### Integration Points

- `internal/cli` — the new `feed` subcommand registers in the existing command table.
- `internal/xweb` — the timeline screen reads through the existing `Client`, so the budget line
  and the classifier come for free.
- `internal/domain` will grow the `Post`/`User`/`Page` types Phase 1 created in minimal form;
  later phases extend them rather than defining a second model.
</code_context>

<specifics>
## Specific Ideas

- The selection marker is a literal `>` in column 1, plus reverse video where the colour rung
  allows — the marker alone must be sufficient to find the selected row in a monochrome frame.
- A repost shows the original author in the identity column and `reposted by X` as provenance;
  a quote is a nested block, never elided.
- Display names are absent from the list view; the handle is the identity. Phase 4 owns the
  detail header.
</specifics>

<deferred>
## Deferred Ideas

- Display names in the list view — deferred to Phase 4's detail header.
- Relative timestamps ("2h ago") — rejected: the stamp is absolute and fixed-width so a frame is
  reproducible.
- Any inline media or printed URL — rejected; media opens externally from Phase 7.
</deferred>
