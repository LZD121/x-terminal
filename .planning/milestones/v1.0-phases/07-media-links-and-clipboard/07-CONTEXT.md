# Phase 7: Media, Links and Clipboard - Context

**Gathered:** 2026-10-14
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — proposals accepted in batch under the run-wide policy

<domain>
## Phase Boundary

The user opens a post's images and video in external programs and copies or opens links — with
nothing silently unreachable, and cover-breaking actions gated in immersive mode.

Delivers: `internal/media` (the `Renderer` interface and its external-program implementation), the
three actions — open media, open the post in a browser, copy the post's link — with their fallbacks,
the DSGT-10 gate in the action dispatcher, and the
tests that prove the gate and the fallback rather than asserting them.

Does NOT deliver: any in-terminal image code (settled by measurement, not preference — see
`TERMINAL-MEDIA-PROBE.md`); a media cache, thumbnail store or download manager; any fetch of media
bytes by this client; OSC 8 hyperlinks on every post; player configuration beyond one program name;
or cover-breaking actions in panic mode, which is not interactive at all.

</domain>

<decisions>
## Implementation Decisions

### The renderer seam (MEDIA-02)

- The interface lives in a new package, `internal/media`:
  `Open(ctx context.Context, ref domain.MediaRef) (Result, error)`, where `Result` carries a
  one-line notice and **optionally** rows for the pane to print in a detail frame.
- The `Rows` half is why the interface has that shape. A renderer that draws in the terminal (the
  v1.x Kitty implementation) cannot be expressed as "spawn a program with a URL"; if v1 defined the
  seam that way, the later addition would need new pane code, which is precisely what MEDIA-02
  forbids. So the frame knows how to print rows a renderer supplies — and **only in a detail frame,
  never while a scrollable list is on screen**, because the probe measured what tmux does to a
  scrolled image (text drawn across it, smeared pixels, no pane-split awareness).
- v1 ships one implementation, `external.go`, which spawns the configured program and returns a
  notice, never rows. The v1.x addition is one new file (`kitty.go`) plus one config value and one
  wiring line at the composition root; no list or detail code changes. That is the honest reading
  of "a one-file addition", and it is stated here so the plan can hold the line.
- **Dispatch by media kind, through the programs the config already names**: `config.MediaOpener`
  (default `xdg-open`) for an image's URL and for a permalink; `mpv` for video and animated media.
  `mpv` is fixed for this phase rather than a second config key — the configuration surface stays at
  the one opener the project already ships (`MediaOpener`), and the scope-drift pitfall's rule is
  that a new key must be about credentials or safety. The URL is one argument and **no shell is
  involved**: nothing in a post's text can become part of a command line.
- **A video opens a playable variant when the response carries one, and its poster frame when it
  does not** — `domain.MediaRef.URL` is documented as the poster, not the stream, so this is a
  content question the plan settles against the recorded fixture: if the fixture carries no variant
  list, v1 opens the poster frame and says so, rather than pretending to play a video.
- **The pane never blocks and never waits for the child.** The call is a `tea.Cmd`; the status line
  reports the spawn result; a goroutine reaps the process so no zombie accumulates.
- **The client never fetches media bytes.** The URL handed to the opener is the one the payload
  carried, and the whole media path is one spawn plus one line of output.

### Nothing silently unreachable (MEDIA-03)

- **The fallback is a row in the pane, not a sentence in the status line.** When the opener cannot
  take the target, the post's block carries a `url: {url}` row — its own marker, its own column, the
  URL wrapped to the measure with its continuation rows indented to that same column so it reads as
  one run of text, and **never truncated**. It must not be the status line alone: that line drops
  fields under width pressure by design, and a URL cut to the pane's width is an unreachable URL.
  This is what `MEDIA-03`'s "never silently unreachable" means concretely, and the phase's golden
  frames store it at 80 and 120 columns.
- The fallback URL carries an **OSC 8 hyperlink** on the colour rungs where escapes are emitted at
  all, and the mono rung carries none — the escape is part of the same ladder that decides whether
  any escape is written. It carries no styling either (the URL is not underlined, not recoloured),
  so Phase 2's "exactly one text attribute in the whole surface" still holds.
- **A missing opener is a state, not a notice**: `no opener`, with the remedy `copy the url by hand`,
  in the status line's existing two slots, and it is *recoverable* (never the stopping accent — a
  missing program is repaired locally and the session is not over). The URL row above is what makes
  the remedy true rather than a suggestion.

### Links and the clipboard (REND-07)

- **Permalink form**: `https://x.com/<handle>/status/<id>` when the post's handle is known,
  `https://x.com/i/status/<id>` otherwise — a tombstone or a withheld post has no handle, and the
  `/i/status/` form still resolves.
- **`y` copies the permalink through a chain, and the notice reports what actually happened.**
  Link one: the OSC 52 escape is written to the terminal unless `$TERM` is `dumb` or empty — the same
  condition under which this program emits no other escape, and deliberately **not** `NO_COLOR`,
  which is honoured for exactly what it names. Link two: when the escape was not written, `wl-copy`
  is run with the URL on stdin. Link three: when neither is available, nothing was copied — the state
  is `no clipboard`, and the URL row prints so the value is not lost. The confirmation is
  `link copied` in the first two cases; which link ran is not the user's problem, and naming
  `wl-copy` on the line would be chrome. `xclip`/`xsel` are not used (absent on this host,
  X11-only) and there is no clipboard command in the config.
- Both writes are I/O, so both are effects: the OSC 52 escape goes to the tty through the same
  `tea.Cmd` boundary as Phase 3's tmux and title escapes, never from `Update`.
- **`o` opens the permalink in the browser** through the link opener.
- The clipboard is not private, and the notice says what was put there (a link). Nothing else ever
  goes to the clipboard: no post body, no handle list, no session material. A clipboard manager may
  keep what this writes, and the "no trace" promise covers the timeline, not the user's own
  clipboard.

### The gate (DSGT-10)

- **One gate, in the action dispatcher, not in three key handlers.** In the immersive state the
  three cover-breaking actions — opening in the browser, opening media, copying a post link — are
  **refused**: no request, no process, no escape, nothing handed anywhere, and one line in the slot
  the state word occupies, written with the **active grammar's `Gated` literal** — every cover grammar
  must have a non-empty one (a test asserts it), because a refused key with no words on screen is
  indistinguishable from a broken key, and a covered pane must not name the product or the purpose it
  is declining to reveal.
- Refusing rather than unbinding is deliberate. DSGT-08 requires the whole read surface to work with
  unchanged keys under the cover; a key that exists in normal mode and vanishes in immersive mode is
  a screen that behaves differently, which is what "unchanged keys" forbids.
- **Copying is gated even in the handle-free `/i/status/` form.** The rule is what the action
  reveals — any post link points at the post — so an exception for a URL without a handle would be a
  loophole rather than a rule.
- **Panic is not interactive**: only the key that leaves panic is honoured and no action key does
  anything from the panic frame. Phase 3 owns the state model; this phase only states what its
  actions do in each state, and that statement is a test.
- **The gate is proven, not asserted.** With a recording renderer and a recording effect sink, a
  covered press of each of the three keys must produce zero renderer calls, zero effects, and a
  frame that differs from the pre-keypress frame only in the refusal slot; the covered help frame
  must contain no product name, no handle and no purpose-naming copy; and the refusal may not tell
  the user to leave the cover, because doing so would name what the cover exists to hide. The way
  out is the documented key, not a sentence on the screen.

### Keys and rows

- **Three rows, and one of them opens the whole set.** `m` — `media` — `open the post's media in an
  external viewer` (every attachment of the selected post; the status line reports `opened {n}
  media`); `y` — `copy-link` — `copy a link to the post`; `o` — `open-post` — `open the post in your
  browser`. No digit keys and no selection mode inside a block: a post with several attachments
  opens them as a set, which is what the viewer is for.
- `enter` (Phase 4), `b`/`l` (Phase 6) and Phase 2's rows are untouched. Phase 3 lands first and a
  collision with a cover key is resolved in the cover's favour (Phase 4's rule).
- The "does not apply" rule (Phase 4) covers the empty cases: a post with no attachments reports
  `no media in this post` and spawns nothing; `y` on a post with no handle builds the handle-free
  `/i/status/` permalink rather than refusing.

### Configuration

- **No new configuration key.** `config.MediaOpener` already names the program images and links go
  to, and it is the only opener this phase reads; `mpv` for video is fixed in the dispatch. The
  configuration surface therefore does not grow at all — no clipboard command, no player flags, no
  image mode, no auto-open-on-select, no per-terminal overrides — which is the scope-drift
  pitfall's rule met by subtraction rather than by argument.

### Parallel safety (the files this phase touches, and how)

- New files: `internal/media/{renderer.go,external.go}` and their tests (a new package),
  `internal/render/media.go` (the attachment line, the `url:` fallback row and the help literals),
  `internal/fake/media.go` (a recording renderer for the tests), plus this phase's rows and dispatch
  arms.
- Shared edits, small and named: `internal/keys` gains three rows; `internal/tui` gains the action
  dispatch, the state check and the refusal; the boundary test's allowlist gains `internal/media` as
  the package permitted to run `os/exec`. **`internal/config` and `go.mod` are untouched** — this
  phase adds no dependency and no configuration key.
- Phase 3 owns the display state, the cover's marker set and the alias function; this phase reads the
  state and adds none. The three new rows' help text and the refusal are presentation literals, so
  the cover's marker set carries its own wording for them — one entry per row plus the `Gated`
  literal, in Phase 3's grammar rather than a second table here.

### Claude's Discretion

- The attachment line's exact wording and how it names a GIF versus a video; the `url:` row's
  marker text and the exact column arithmetic at each width; the OSC 8 parameters; whether `mpv`'s
  arguments live in code or in a small table next to the dispatch; and the `no media in this post`
  notice's exact placement among the status line's fields.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets

- `internal/domain` — `MediaRef{Kind, URL, AltText}` and `MediaKind{Image, Video, GIF}` are already
  parsed and carried by the timeline and thread decoders, and deliberately unrendered until now;
  `Post.Media` is nil-versus-empty on purpose, which is how the screen tells "no attachment" from
  "attachments that did not parse". Nothing new is needed in this package.
- `internal/config` — `MediaOpener` with its default, the `KEY = "value"` parser, and the
  warning-not-failure rule for a bad value; `Config.Warnings` is already surfaced where the tool
  starts.
- `internal/render` — the `Grammar`/`MarkerSet` seam, the block composer and the status line: the
  attachment line is a marker template today, and the `url:` row and the refusal slot are new
  variants of the line and the status line, not a new block type.
- `internal/text` — the one measurement; the fallback URL is wrapped with it like any other text.
- `internal/keys` — the single table feeding input, `?` and the `keys` subcommand.
- `internal/tui` — the effect boundary (`tea.Cmd` per I/O), the frame stack, and Phase 3's display
  state and cover grammar, which the gate reads.
- `internal/cli` — the composition root, where the concrete `media.Renderer` is chosen and where the
  config is resolved; no new verb is needed (every action lives inside the TUI).
- Evidence to read rather than re-derive: `.planning/research/TERMINAL-MEDIA-PROBE.md` (Probes A–C:
  bare APC works in Ghostty, fails silently in tmux, DCS-wrapped renders and then text draws across
  it; `KITTY_WINDOW_ID` is not exported) and the media rows of the pitfalls checklist.

### Established Patterns

- Every read *and* every side effect is a value returned from `Update` and performed by the shell;
  `internal/tui` may not import `os/exec`, and `internal/media` becomes the named exception rather
  than a new habit.
- No shell interpolation anywhere: a program name from config plus one argument.
- One width model, so a wrapped URL is wrapped by the same code that wraps post text.
- Golden frames are the view specification: the `url:` fallback row (wrapped at 80, whole at 120),
  the `no media in this post` notice and the covered refusal line each need a stored frame.
- Failure is a value the frame renders, never an error from `Update`; the media outcome is a value
  (a notice, optionally rows) for the same reason.
- The help overlay's rows are data from the keymap table, so the covered wording of a row belongs to
  the cover's grammar rather than to a second string table.

### Integration Points

- `internal/cli` — the composition root builds the renderer from the resolved config and passes it
  to the shell as the effect runner's dependency; the command table does not change.
- `internal/tui` — three actions, one gate, one dispatch arm; the state check reads Phase 3's state
  value and writes only the status line's notice field.
- `internal/render` — the attachment line gains its `url:` fallback variant, and the cover's
  `MarkerSet` gains the covered wording of the three new help rows plus the `Gated` literal
  (Phase 3's grammar file).
- `internal/config` — untouched: the phase reads `MediaOpener` and adds no key, and the doctor's
  config line (Phase 8) already reports the configuration it resolved.
- Phase 8 re-checks the whole surface under the cover, so the covered refusal and the covered help
  rows must be in the golden corpus, not only in this phase's tests.

</code_context>

<specifics>
## Specific Ideas

- The media affordance must be legible without looking clickable: the attachment line states what is
  there and which key reaches it. No underlined "link", no icon, no arrow.
- Opening media must never block or flicker the pane: the frame stays responsive, and the status
  line reports the spawn's outcome when it returns.
- The fallback is the whole point of MEDIA-03: a URL on screen beats a "failed" line, and it beats a
  silent no-op by a mile.
- Copy must report what it did: "copied" with an unchanged clipboard is the failure mode the
  pitfalls research names, and it is why the notice distinguishes the mechanisms that ran.
- These three actions are the only ways anything leaves the pane, so they are also the only place
  the cover has to be defended — one gate, tested once, in three keys.

</specifics>

<deferred>
## Deferred Ideas

- The Kitty graphics renderer (IMG-01, v1.x, trigger: "media is opened often enough to care"). The
  seam is shaped for it: one new file, one config value, one wiring line. That file must detect
  capability from `TERM_PROGRAM`/`TERM` (never `KITTY_WINDOW_ID`), DCS-wrap under `$TMUX`, request an
  explicit placement size, reserve the cells it occupies, force a repaint, and live in a dedicated
  non-scrolling pane.
- A media cache with eviction (IMG-02) — after IMG-01, and only then.
- OSC 8 hyperlinks on every URL and on post rows: one function to add, deferred because it changes
  every stored golden frame for a mouse affordance nobody asked for.
- A configurable clipboard command (for SSH or a non-Wayland session): the OSC 52 path covers remote
  sessions by design; a command key would be configuration surface for one user's convenience.
- Downloading media to a directory, thumbnails, GIF animation: delegated to the external program,
  which can save a file when the user needs one.
- Opening a thread, a profile or a search in the browser: v1 opens the selected post's permalink; a
  general "open what I am looking at in the browser" mode is the signer-break recovery path, which
  Phase 8's runbook describes as an operator step rather than a feature.
- mpv flags, playlists or a per-kind player table in the config file.
</deferred>
