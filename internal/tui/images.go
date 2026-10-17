package tui

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LZD121/x-terminal/internal/domain"
	"github.com/LZD121/x-terminal/internal/media"
	"github.com/LZD121/x-terminal/internal/vivid"
)

// The reading column's image: one picture, fetched on demand, transmitted to the terminal once, and
// placed as a block of cells the frame prints.
//
// The division is the design's own. The pane knows cells; the terminal knows pixels. So the shell
// fetches bytes through the port, hands them to the terminal with Term.Draw, and puts an Image value
// in the frame — and the frame's placeholder cells are what make the picture occupy real cells, so
// the text under it is laid out rather than drawn across (the hazard probe C measured for a bare
// put, and which the placeholder form removes).

// errBadImage is a body that decoded to an empty rectangle: not a picture, whatever it claimed.
var errBadImage = errors.New("tui: the image decoded to nothing")

// jpegEncode and jpegDecode are the JPEG path, as variables for the same reason pngEncode is: a
// test states the encoder without a fixture, and the production code names the standard library's.
var jpegEncode = func(b *bytes.Buffer, img image.Image) error { return jpeg.Encode(b, img, nil) }

// pngEncode is the encoder, as a variable so the tests can hold the failure arm without a body that
// encodes badly (there is none: png.Encode accepts every image.Image).
var pngEncode = png.Encode

const (
	// imageCap bounds one picture, in bytes. It matches the fetcher's own cap: the fetcher refuses
	// more, and a shell that accepted more would be a second limit nobody measured.
	imageCap = 8 << 20
	// maxImageCells is the largest block on either axis: the protocol's diacritic table holds 16,
	// and a larger image would need a table this build does not carry.
	maxImageCells = 16
	// imageTimeout bounds the fetch. It is shorter than a GraphQL request's budget because a
	// picture is optional content: a slow edge must become the attachment line again, not a
	// spinner the reader waits behind.
	imageTimeout = 15 * time.Second
)

// imageLoadedMsg is one fetched picture, already sized in cells.
type imageLoadedMsg struct {
	post domain.PostID
	png  []byte
	cols int
	rows int
	err  error
}

// vividImageCmd fetches the opened post's first attachment and returns the bytes with the block
// size the reading column will reserve.
//
// One image per open, and only the first attachment: a reading column of a few rows, four fetches
// the reader did not ask for, and the rest stay the grammar's own `[2] image` line.
func (m Model) vividImageCmd(post domain.Post) tea.Cmd {
	if !m.deps.Images || m.deps.Image == nil {
		return nil
	}
	if len(post.Media) == 0 {
		return nil
	}
	ref := post.Media[0]
	if ref.Kind != domain.Image {
		// A video or a gif is the external player's, and the frame already says so.
		return nil
	}
	load := m.deps.Image
	cols, _ := m.guessAspect()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), imageTimeout)
		defer cancel()
		body, _, err := load(ctx, ref.URL)
		if err != nil {
			return imageLoadedMsg{post: post.ID, err: err}
		}
		// The protocol's PNG form is what this build transmits, and a twimg edge serves JPEG as
		// often as not, so the bytes are decoded and re-encoded before they are sent. That is also
		// what makes the block's aspect exact: the decoder knows the real dimensions, and the
		// reserve above is only what the frame draws before anything is known.
		png, w, h, err := toPNG(body)
		if err != nil {
			return imageLoadedMsg{post: post.ID, err: err}
		}
		c, r := imageBlockFor(w, h, cols)
		return imageLoadedMsg{post: post.ID, png: png, cols: c, rows: r}
	}
}

// guessAspect is the block the frame reserves before the bytes arrive: a 16:9-ish picture in the
// column's width. It is the same block the answer usually keeps, so the pane does not jump.
func (m Model) guessAspect() (int, int) {
	return m.imageBlock(9.0 / 16.0)
}

// imageBlock is a block for one aspect ratio (height/width) at this pane's reading column, capped
// on both axes. Cells are about twice as tall as they are wide, which is why the ratio is halved.
func (m Model) imageBlock(aspect float64) (int, int) {
	cols := m.readingWidth()
	if cols > maxImageCells {
		cols = maxImageCells
	}
	if cols < 1 {
		cols = 1
	}
	rows := int(float64(cols) * aspect / 2)
	if rows < 1 {
		rows = 1
	}
	if rows > maxImageCells {
		rows = maxImageCells
	}
	return cols, rows
}

// readingWidth is the reading column's content width, borders subtracted. It is the same arithmetic
// the vivid renderer does, asked once here so the block fits the column it is printed in.
func (m Model) readingWidth() int {
	g := vivid.GeometryOf(m.w)
	return max(1, g.Reading-2)
}

// toPNG decodes one image the fetcher accepted and re-encodes it as PNG, which is the only format
// the graphics protocol's `f=100` form carries. The dimensions come back with it so the block's
// aspect is the picture's own.
func toPNG(body []byte) (png []byte, w, h int, err error) {
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, 0, 0, err
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, 0, 0, errBadImage
	}
	var buf bytes.Buffer
	if err := pngEncode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

// imageBlockFor is the block the answer asks for: the aspect the bytes actually have, in the
// column's width.
func imageBlockFor(w, h, cols int) (int, int) {
	if w <= 0 || h <= 0 {
		// A body the decoder cannot measure keeps the reserved block: it is still an image the
		// terminal may draw, and a zero-height block would be a picture with nowhere to go.
		return cols, max(1, cols/2)
	}
	aspect := float64(h) / float64(w)
	rows := int(float64(cols) * aspect / 2)
	if rows < 1 {
		rows = 1
	}
	if rows > maxImageCells {
		rows = maxImageCells
	}
	return cols, rows
}

// imageDrawn places one picture: transmit, then place. It is a command because it writes to the
// terminal, and Update is pure.
func (m Model) imageDrawn(msg imageLoadedMsg) (Model, tea.Cmd) {
	if msg.err != nil || len(msg.png) > imageCap {
		// The frame keeps the attachment line it already had; nothing is drawn and nothing is
		// booked, so the delete on close has nothing to remove.
		return m, nil
	}
	if m.vivid.reading != msg.post {
		// The reader moved on while the bytes were in flight: placing it now would put a picture
		// of one post beside another.
		return m, nil
	}
	id := m.vivid.next()
	seq := media.KittyTransmit(id, msg.png) + media.KittyPlace(id, msg.cols, msg.rows)
	term := m.deps.Term
	m.vivid = m.vivid.withImage(placedImage{id: id, post: msg.post, rows: msg.rows, cols: msg.cols})
	return m, func() tea.Msg {
		term.Draw(seq)
		return nil
	}
}

// clearImage removes every picture this presentation placed. It is the delete-all rather than a
// per-id delete: the presentation holds at most one picture, and an id that was transmitted but
// never placed (a fetch that lost its race) is exactly the case a per-id delete would miss.
func (m Model) clearImage() (Model, tea.Cmd) {
	had := len(m.vivid.images)
	m.vivid.images = nil
	if had == 0 {
		return m, nil
	}
	term := m.deps.Term
	return m, func() tea.Msg {
		term.Draw(media.KittyDeleteAll())
		return nil
	}
}

// clearReading closes the reading column: the post, the opened mark's business is done, and the
// picture it placed is removed from the terminal's store.
func (m Model) clearReading() Model {
	m, _ = m.clearImage()
	m.vivid.reading = ""
	return m
}
