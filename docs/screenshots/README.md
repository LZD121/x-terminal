# Screenshots

Five states of the reader, captured from a live run on 2026-10-17 and rendered from the terminal's own
grid.

**How these were made.** `tmux capture-pane -p` printed the pane's exact cells at 150x44 (and 62x22 for
the narrow one); a small script drew those cells with a monospace face (Noto Sans Mono, Noto Sans CJK
for the wide glyphs, which is why the CJK rows read correctly). They are **not** photographs of a
window: screen capture (`grim`) stopped answering on this host mid-session — even a 437x480 region
timed out — so the pane's text was rendered instead. That means the images show the frame's content
and column geometry exactly, and do not show the terminal's own colours, font or window chrome.

| File | What it shows |
|---|---|
| `01-plain.png` | The plain reader: the surface the program starts in and the one the disguise covers. |
| `02-vivid.png` | Vivid mode on the home list: the navigation column, the post list, and the reading pane before anything is opened. |
| `03-vivid-reading.png` | A post opened in the reading pane. The list on the left has not moved. |
| `04-vivid-narrow.png` | Vivid mode at 62 columns: one column, the presentation's own heading, no reading pane. |
| `05-panic.png` | The panic frame: one key replaces everything, including any image the presentation had placed. |

The content is the operator's own timeline at the time of capture. Nothing here is a fixture, and
nothing was edited — including the posts, which are public posts by their authors.

Everything visible was reached by keys alone: the plain reader is where a run starts; vivid mode is
entered by a sequence this repository does not document; `ctrl+p` is panic.

Known gap at the time of capture: the reading pane shows the post's text and its first attachment as a
line of text (`[1] video`). The in-terminal image renderer is built and tested
(`internal/media/kitty_payload.go`, `internal/xweb/image.go`) but is not yet wired to the pane, so no
picture is drawn in these screenshots.
