# Phase 3: Three-State Disguise on the Working Renderer - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — proposals accepted in batch under the run-wide policy

<domain>
## Phase Boundary

The user can wear the feed as work-software output or replace the pane with a one-key panic
frame — identities aliased, nothing on screen naming the real product — and step back without
losing their place.

Delivers: the three-state display model (normal / immersive / panic), four selectable immersive
cover styles with deterministic identity aliasing, one-key panic that clears what came before,
focus-loss auto-panic, persistence of the chosen state and cover, and the whole existing read
surface working unchanged under a cover.

Does NOT deliver: the gating of browser/media/clipboard actions (DSGT-10 lands in Phase 7, where
those actions exist), or any new reading capability. The cover is a **renderer mode**, not a new
screen: this phase must not add a second way to render posts.
</domain>

<decisions>
## Implementation Decisions

### The three states

- `normal` (the plain reader), `immersive` (the same feed through a cover), `panic` (a static
  innocuous frame that replaces everything instantly). Panic is an **overlay state**, not a
  screen: it is entered from either other state with one key and left with the same key.
- Restoring from panic restores the exact previous screen, selection and scroll and issues
  **no** request. This is a property to test, not an intention: the frames must be byte-identical
  before and after a panic round trip, and the request budget must not move.
- Losing window focus enters panic automatically. Regaining focus does **not** restore by
  itself — un-covering is only safe as a deliberate act. Focus detection uses the terminal's
  focus reporting through Bubble Tea; if the terminal does not report focus, the auto-panic
  silently does not fire, and that limitation is recorded rather than faked (the key still works).
- The active state and cover style persist in the config file, and the user chooses which state
  the app starts in.

### Panic: clearing what came before

Measured on this host (`.planning/research/TERMINAL-MEDIA-PROBE.md`, Probe D/E, with a control):

- Direct Ghostty 1.3.1: `ESC [ 3 J` (ED 3) **does** clear the terminal scrollback — after six
  injected Shift+Page_Up pages, no labelled feed line survived; the same run without ED 3 leaked
  immediately.
- Under tmux 3.7c, ED 3 does **nothing** to the multiplexer's own history (401 matching
  `capture-pane -S -5000` lines before and after), and only `tmux clear-history -t <pane>` clears
  it. After that, the 17 lines still visible on the pane remain — the screen, not the history.

Therefore:

- **Order matters: paint the cover first, then clear history.** Clearing while the feed is still
  on the visible pane leaves that pane as the only remaining history.
- Two mechanisms satisfy one requirement: ED 3 for a bare terminal, `tmux clear-history -t
  $TMUX_PANE` under a multiplexer. The latter is a subprocess, so it belongs behind the same
  effect/`Cmd` boundary as every other I/O — never called from `Update` directly.
- A missing or empty `$TMUX_PANE` means "no tmux history to clear", not an error.
- Panic also sets the terminal title (and the tmux window name where applicable) to the cover's
  own name, and restores the previous title on exit.

### Immersive covers

- At least four cover styles, selectable and switchable at runtime: timestamped service-log
  lines, git-log/commit entries, CI/build pipeline output, and a generic internal-tool layout.
- The cover extends to **identity**: author handles and display names are replaced throughout by
  cover-appropriate aliases. No screen in immersive mode may render a handle or display name that
  exists on x.com.
- Alias mapping is **deterministic** (a stable function of the account id), so an account keeps
  the same alias across screens, sessions and restarts, and **no alias table is written to
  disk** — a table would be a list of the accounts the user follows, which is exactly the kind of
  trace this project refuses to leave.
- The cover is a `Grammar` value over the existing render path (Name, HandleCols, Stamp,
  BodyIndent, MeasureCap, Separator, Markers, Identity, ShowCounts, palette) — the split Phase 2's
  UI-SPEC defines. No parallel rendering code, no per-screen copies: the same composer produces
  both the plain frame and the covered frame from the same content.
- Nothing in immersive or panic mode reveals the real product or its purpose: no product name, no
  status line in the product's own voice, no hint text, no engagement counts, avatars or icons.

### Verification

- The two terminal behaviours are testable without a human watching: `wtype` injects keys and
  `grim` captures the window for the bare-Ghostty leg, and `tmux capture-pane -S -N` reads tmux's
  history for the multiplexer leg. Phase 3's acceptance runs both.
- A test asserts that no rendered frame under a cover contains any real handle or display name
  from the content it was built from — scanning the frame, not trusting the mapping.

### Claude's Discretion

The exact key assignments, the visual detail of each cover beyond the constraints above, and the
alias function's shape are at the implementer's discretion within the UI-SPEC.
</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

- Phase 1: `internal/config` (the config file and the data dir), `internal/session`,
  `internal/xweb` (the only send path, with the budget line), `internal/cli`.
- Phase 2 (in flight at the time this was written): `internal/text` (the one width model),
  `internal/render` (copy, grammar, block, status, frame, palette, help), the keymap table, and
  the Bubble Tea shell with its back stack. Phase 3 changes the `Grammar` the composer is given,
  nothing else.
- `.planning/phases/02-following-timeline-in-the-pane/02-UI-SPEC.md` draws the content/presentation
  split explicitly for this phase, and carries two worked renderings of the same post (the plain
  reader and a timestamped service-log line) as proof the split is real.
- `.planning/research/TERMINAL-MEDIA-PROBE.md` — the measured scrollback behaviour, with the
  control that makes it a measurement.

### Established Patterns

- Every read is a `tea.Cmd`; `internal/tui` may not import `net/http`, `os/exec`, `database/sql`,
  `internal/xsource` or `internal/store`. `tmux clear-history` and the title-setting escape are
  I/O, so they live behind the same boundary — an AST boundary test will have to allow the new
  package and forbid it from `tui`.
- The classifier's outcomes are the only vocabulary for failure states; a cover must not invent
  new wording that could contradict what actually happened.

### Integration Points

- `internal/config` — the persisted display state and cover style; a new key in the existing
  reader.
- `internal/render` — the `Grammar` seam this phase fills.
- `internal/tui` — the state machine and the focus/panic transitions; the shell already owns the
  back stack that panic must restore through.
</code_context>

<specifics>
## Specific Ideas

- Panic should be **instant**: one frame, no transition, no animation, no flash of the real
  content on the way in.
- The cover should be convincing at a glance but honest to its reader: an alias is stable and
  human-readable (`svc-deploy`, not a hash), so the user can still follow a conversation.
- Immersive mode must keep every key working — if covering the feed breaks scrolling or the back
  stack, the feature has failed its purpose.
</specifics>

<deferred>
## Deferred Ideas

- Gating cover-breaking actions (browser open, media open, handle-bearing URL copy) — DSGT-10,
  Phase 7, where those actions exist.
- A cover for the *drill-down* screens beyond what Phase 2/4 render — revisit after Phase 4.
- Any AI-generated cover text: rejected outright — it would send post content to a third party.
</deferred>
