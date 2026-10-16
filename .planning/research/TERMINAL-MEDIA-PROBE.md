# Terminal media capability — empirical probe (not documentation-derived)

Date: 2026-10-14. Host: Omarchy/Arch, Hyprland (Wayland), Ghostty 1.3.1-arch2, tmux 3.7c.
Method: emit a real Kitty graphics protocol payload from a script, capture the window
with `grim`, inspect the pixels. Screenshots: `evidence/probe-{A,B,C}.png`.

Test image: 320x180 PNG, left half blue, right half red, centered yellow box
(`ffmpeg -f lavfi -i color=c=red:s=320x180 -vf "drawbox=...,drawbox=..."`).

## Probe A — direct Ghostty, Kitty graphics protocol

Payload: `ESC _ G a=T,f=100,q=2 ; <base64 png> ESC \` (single chunk, <4096 bytes).
Environment: `TERM=xterm-ghostty`, `TERM_PROGRAM=ghostty`, `KITTY_WINDOW_ID` unset.

**Result: WORKS.** The image rendered in the expected place and orientation, colors
correct. Two observed behaviours:

- The image was **scaled and interpolated** (soft edges at the blue/red boundary and
  around the yellow box) rather than drawn 1:1 at its native pixel size — Ghostty
  applies smoothing when the placement size is not requested explicitly.
- `KITTY_WINDOW_ID` is **not** exported by Ghostty, so capability detection must not
  rely on it; `TERM_PROGRAM=ghostty` / `TERM=xterm-ghostty` are the reliable signals.

## Probe B — inside tmux 3.7c, bare APC sequence

Payload: same as A, unwrapped. `allow-passthrough on` (set in
`~/.config/tmux/tmux.conf:75`). Environment: `TERM=tmux-256color`, `TMUX` set.

**Result: FAILS.** No image. tmux consumes the APC string; `allow-passthrough on`
alone does not forward a bare Kitty graphics sequence.

## Probe C — inside tmux 3.7c, DCS-wrapped sequence

Payload: `ESC P tmux ; ESC ESC _ G a=T,f=100,q=2 ; <base64> ESC ESC \ ESC \`
(every ESC inside the payload doubled, per tmux's passthrough rule).

**Result: WORKS — with a layout hazard.** The image rendered, and the same run shows
the hazard directly: the text printed *after* the payload (`--- end of probe C ...`)
was drawn **across** the image. tmux does not model image cells — the outer terminal
paints pixels at the cursor position while tmux's own grid believes those cells are
empty and advances normally. Consequences: overlap after any image, smear on scroll,
and no pane-split awareness.

## Implications for this project

- The v1 decision (media opens in an external program, behind an abstraction) is the
  right default and is now empirically justified, not assumed.
- A future in-terminal renderer must, at minimum: detect a graphics-capable outer
  terminal (`TERM_PROGRAM`/`TERM`, never `KITTY_WINDOW_ID`); DCS-wrap when `$TMUX` is
  set; reserve the cells the image occupies and force a full repaint afterwards; and
  confine images to a dedicated detail/pane that redraws on demand, because scrolling
  a list of images will smear under tmux.
- Scaling is not free of artifacts: request an explicit placement size (native pixels
  or a cell-aligned size) instead of relying on defaults if legibility matters.
- Sixel and iTerm2 inline images were **not** tested; the Kitty protocol is available
  on this host, so it is the only path worth building.
- `chafa`/`viu`/`timg` are not installed here; `mpv` and `ffmpeg` are. External
  opening is therefore already fully functional for images and video.

## Probe D/E — panicking away the scrollback (DSGT-02), 2026-10-14

Question: when the user hits the panic key, does clearing the screen also remove the feed
from anything reachable by scrolling up? Method: print 200 uniquely-labelled lines, then
paint a cover, then inject six Shift+Page_Up (`wtype`) into the focused window and capture
it (`grim`). Screenshots: `evidence/panic-scrollback-{D,E}.png`.

- **Direct Ghostty, with `ESC [ 3 J` (ED 3): clean.** After six pages up, only cover text is
  visible — no labelled line survives (`panic-scrollback-D.png`).
- **Direct Ghostty, same script without ED 3 (control): leaked.** Scrolling up immediately
  reveals the labelled lines (`panic-scrollback-E.png`). This control is what makes the
  previous result a measurement rather than a no-op: it proves the injected scroll works.
- **Inside tmux 3.7c, ED 3 does nothing to tmux's own history.** Measured with
  `tmux capture-pane -p -S -5000 | grep -c` on a 400-line flood: 401 matching lines before
  the escape, **401 after** it. `tmux clear-history -t <pane>` then takes it to 17 — the
  lines still on the visible screen — and nothing else.

Design consequences for panic mode:

1. Paint the cover **first**, then clear history. Clearing history while the feed is still
   on screen preserves that screen's content as the only remaining history.
2. Direct terminal: `ESC [ 3 J` is sufficient and verified.
3. Under tmux: ED 3 is not enough — the app must invoke `tmux clear-history -t $TMUX_PANE`
   (a subprocess, so it belongs behind the same effect/`Cmd` boundary as any other I/O).
4. The verification harness itself is reusable: `wtype` for terminal input and `grim` for
   capture make scrollback behaviour testable without a human watching.
