# Vivid mode - design

Status: draft for approval. Date: 2026-10-17. Nothing is built.

"Vivid mode" is the working name for the second display mode the operator asked for as "酷炫模式".
The name is the author's call; rename freely.

## Intent and why

x-terminal is today one thing: a plain, log-viewer-looking reader, designed so a feed can be read
under a work-software cover and hidden with one key. The operator also uses it where no disguise is
needed, and there it still falls short of the browser: they keep opening x.com to **refresh, scan for
news, and open a post to read the whole thread**, and the plain reader's look and image handling are
the reason they go back.

Vivid mode is a second, deliberately different presentation of the same data. The operator's words:
it is no longer minimal, and each feature may be designed as beautifully as it needs. The reason it
exists is the browser-opening habit; success is that the operator can do the refresh / scan / read-a-
thread loop, plus search, profiles and saved lists, without opening x.com.

## Goals

- A second mode covering **every screen the plain reader has**: home timeline, thread (post detail),
  search, profile, bookmarks, likes. (Operator: "一起做进去".)
- **Layout B:** a left navigation column, a main reading area to its right, a bottom status bar.
  High information density, in the manner of lazygit or btop.
- **The core loop is first class:** refresh, see what is new, open a post, read the whole thread with
  the list still visible beside it, go back, continue.
- **New-post marking:** after a refresh, posts that arrived since the previous load are visibly
  distinct from ones already seen; once a post has been opened it dims. (Author's proposal, not yet
  confirmed by the operator.)
- **Images drawn in-terminal (C)**, in the reading area only, via the Kitty graphics protocol with
  Unicode placeholders; inside tmux the commands are DCS-wrapped.
- Hidden entry: the Konami sequence **Up Up Down Down Left Right Left Right B A**.
- Free to use colour, borders and a richer layout; the plain mode's "reads like a log" rule does not
  apply inside vivid mode.

## Non-goals

- **No writing of any kind.** No post, reply, like, repost, bookmark or follow. The operator asked about
  replying and composing and then chose not to do it for now. `internal/audit`'s read-only checks are
  unchanged, and vivid mode must pass them as they stand.
- No change to the plain mode's look or behaviour.
- No images inside the scrolling list. Only the reading area.
- No video or animated-image playback in the terminal; those keep opening in the external player.
- No background polling and no timeline cache on disk. Refresh stays manual; this is an existing
  product constraint, not a vivid-mode choice.
- No configuration option, flag or environment variable that starts the program in vivid mode.

## Anti-goals (failure even if it technically works)

- Vivid mode becoming reachable by accident: a stray key, a paste, a mistyped command.
- Any vivid-mode state surviving a panic: feed text, or an image, still on screen or still stored by
  the terminal.
- Vivid mode weakening a promise the project already makes (read-only, no persistence of timeline
  data, secrets handling, request budgets).
- Images that smear, overlap text, or corrupt the layout under tmux.

## Entry, exit and panic (decided)

1. **Start state is always plain mode.** There is no flag, config key or env var for vivid mode, and
   nothing on screen, in `--help`, or in the keymap listing (`x-terminal keys`) mentions it. [Author's
   call: hidden means hidden in `--help` too. Whether `keys` should list it is left to the builder to
   weigh against the existing test that joins the keymap to the dispatch.]
2. **Entry** is the Konami sequence, typed in the plain mode's main screen. A wrong key resets the
   sequence and produces no visible reaction. Entering vivid mode is itself the confirmation; no extra
   message. [Author's call: which plain-mode states accept the sequence - at minimum the feed list; the
   builder decides about the others, defaulting to "only the screens where no text input is focused".]
3. **Panic behaves exactly as it does today in both modes**: paint the cover first, then clear the
   scrollback (and `tmux clear-history` under tmux). **Additionally**, in vivid mode, delete every
   image the program placed (Kitty `a=d`), DCS-wrapped under tmux, and drop its in-memory image data.
4. **After panic the mode is plain.** Re-entering vivid mode requires typing the sequence again.
   (Operator delegated this; this is the author's call and the reason is that a stray key after a panic
   must not bring the feed back in its vivid form.)
5. Leaving vivid mode without panic is an open design point: a normal "back to plain" key is
   reasonable. [Left to the builder, subject to: it must not be discoverable from the plain mode.]

## Layout B (shape, not pixels)

- Left column: destinations (Home, Search, Bookmarks, Likes, Profile) with the current one marked.
- Main area: list on the left half, reading pane on the right half when the terminal is wide enough;
  a single-column fallback below a width floor the builder picks. Home and Search show a list; opening
  a post fills the reading pane with the thread; profile and saved lists follow the same pattern.
- Bottom status bar: refresh state, count of new posts, and the **remaining request budget** (author's
  proposal, not confirmed by the operator - it is one line and the budget already exists in the code).
- Sizing and colour palette: left to the builder, inside the constraint that the whole thing renders
  correctly at the sizes the plain mode already supports and has a defined fallback below them.

## Images (decided shape, unproven corners)

- Protocol: Kitty graphics, transmit once, place with Unicode placeholders (`U+10EEEE`, diacritics for
  row/column, image id in the foreground colour), so the image occupies real cells.
- Environments: Ghostty directly, and Ghostty inside tmux (DCS passthrough, every ESC doubled;
  requires `allow-passthrough on`, which the operator's tmux config already sets). Detection by
  `TERM_PROGRAM`/`TERM`, never `KITTY_WINDOW_ID` (Ghostty does not export it). Anything else: no
  images, show a one-line placeholder with the existing "open externally" affordance.
- Where: the reading pane only.
- **Evidence so far (2026-10-17, measured on this host, Ghostty 1.3.1 + tmux 3.7c):** a 320x180 PNG
  placed with placeholders in a 16x5 cell block rendered correctly inside tmux and the text before and
  after it was not overdrawn.
- **Not yet proven:** (a) the image when it is scrolled partly or fully out of the pane, (b) large or
  many images against Ghostty's 320 MB image-storage limit, (c) the first run showed stray `h`/`on`
  characters that did not recur and were not explained, (d) pane resize and tmux pane splits. The first
  implementation step for images is a spike that settles (a), (c) and (d) before the rest is built on
  them; if (a) fails the reading pane must not scroll over an image (page by whole blocks) rather than
  the design changing.
- Image bytes are held in memory only and **never written to disk**, consistent with the no-persistence
  rule. `internal/media` today only hands a URL to `xdg-open`/`mpv`; nothing in the program fetches
  image bytes, so in-process fetching is new work (see "Calls made after review").

## Constraints (existing, binding)

- Read-only: the `audit` read-only and transport checks pass unmodified.
- No timeline data on disk; secrets rules unchanged; the cookie never appears in any output.
- All x.com traffic goes through the existing proxy-aware transport and its budgets; vivid mode adds
  no new endpoint. Images come from `*.twimg.com`, which already requires the proxy; the fetch must use
  the same transport and fail loudly the same way.
- Layout: the program already renders through Bubble Tea v2, with golden-frame tests as the spec of
  every screen. Vivid mode's screens get the same kind of tests.
- English-only interface text.
- The rule that nothing on screen looks like social media is a project principle (`.planning/PROJECT.md`)
  enforced by banned-vocabulary tests in `internal/render` (cover, corpus and status tests), not by one
  of the `audit` checks. Vivid mode is a **declared, narrow exception**: those tests and the principle's
  text must say so explicitly and keep applying in full to plain mode, the cover and the panic frame.
  The exception must not be reached by weakening or skipping them.
- The repository is public and its history is not being rewritten. Fixtures and docs for this work must
  carry no real account names, addresses or machine paths (use `$HOME`).

## What is left to the builder

- Package layout (a new `internal/vivid`-style package versus extending `tui`/`render`), the mode
  state machine, and how the Konami matcher shares the key pipeline.
- Palette, borders, widths, the narrow-terminal fallback.
- How "new since last load" is computed without persisting timeline data (in-memory per run is the
  natural reading of the constraints).
- The exact image-placement bookkeeping and the redraw strategy under tmux.
- Whether the status bar's budget line is kept.
- The test plan, including a golden corpus for each vivid screen and a panic test proving no image
  survives.

## Open items for the operator (not decided)

- Confirm the new-post marking and dimming behaviour.
- Confirm the status-bar budget line.
- A name other than "vivid mode", if wanted.

## Calls made after review

A reviewer read this document cold and asked the questions below; these are the author's answers,
none of them confirmed by the operator. The operator can overrule any.

1. **Entry sequence vs. plain-mode keys.** Plain mode binds `b` (bookmarks, a request) and `a` (open
   author). Up/Down/Left/Right are harmless motion and pass through normally. The matcher holds back
   only the final two keys: once the eight arrow keys have been typed in order, `B` is held, not
   dispatched; if `A` follows, vivid mode opens, otherwise the held `B` is replayed as an ordinary
   key. So no request or navigation happens because of the sequence. Vivid mode opens on the screen
   the reader was already on, not on a fresh Home.
2. **Image fetching.** Built inside `xweb` (the only place the transport audit allows request
   builders), through the existing proxy-aware transport. The session cookie is never sent to
   `*.twimg.com`. Images load on demand for the thread being read, not eagerly, with a per-image size
   cap and a total in-memory cap well under Ghostty's 320 MB; the builder picks the numbers. Image
   requests do **not** draw down the GraphQL request budget the status bar shows (different host,
   different limit); if they get their own limiter, that is the builder's call. Decoding JPEG/PNG with
   the standard library and re-encoding as PNG for Kitty is acceptable; a new dependency is not
   without justification.
3. **Which images.** Post attachments in the reading pane only. Avatars, banners and quoted-post
   media are out; video/GIF posters show as a one-line placeholder with the external-open hint.
4. **Vocabulary.** Inside vivid mode real product wording (handles, "x.com", destination names) is
   allowed. Plain mode, cover and panic keep their current vocabulary; vivid mode reuses `render`'s
   failure/notice grammar where it fits and rewrites it where it does not.
5. **Panic.** Panic leaves the plain mode's frame stack and loaded data exactly as today. Everything
   vivid-only (the mode flag, new/opened marks, image bookkeeping, in-memory image bytes) is
   discarded. Leaving vivid mode normally also clears the marks.
