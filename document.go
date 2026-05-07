package kardec

import (
	"bytes"
	"errors"
	"io"
	"os"

	pdfwriter "github.com/arthurhrc/kardec/internal/pdf"
)

// Document is the central builder a caller composes content onto. It holds
// one or more sections (a section owns a PageSetup, an optional header /
// footer, and a list of blocks).
//
// A *Document is not safe for concurrent use; see the package overview.
type Document struct {
	sections []*Section
	cur      *Section // pointer into sections; the section currently receiving blocks

	err error // first error encountered during builder usage; surfaced by Err and Render
}

// Section couples a PageSetup with the ordered list of blocks that flow
// inside it. Most documents use a single Section; multi-section support
// (different page setups within one document) lands later in v0.x.
type Section struct {
	Setup  PageSetup
	Blocks []Block
}

// New creates an empty Document with a single Section configured from the
// supplied page size and margins. Default orientation is Portrait.
func New(size PageSize, margins Margins) *Document {
	first := &Section{
		Setup: PageSetup{
			Size:        size,
			Orientation: Portrait,
			Margins:     margins,
		},
	}
	return &Document{
		sections: []*Section{first},
		cur:      first,
	}
}

// Err returns the first error captured during builder usage, or nil. Render
// returns the same error before attempting any I/O, so most callers can
// wait until Render to handle failures.
func (d *Document) Err() error { return d.err }

// fail records err as the document's first error if no error was set yet,
// then returns d for fluent chaining. Subsequent calls remain inert until
// the document is recreated.
func (d *Document) fail(err error) *Document {
	if d.err == nil {
		d.err = err
	}
	return d
}

// append adds a block to the current section, unless a prior error has
// already invalidated the document.
func (d *Document) append(b Block) *Document {
	if d.err != nil {
		return d
	}
	d.cur.Blocks = append(d.cur.Blocks, b)
	return d
}

// Heading appends a heading block at the requested level. Levels outside
// 1..6 are clamped to the nearest valid value.
func (d *Document) Heading(level int, runs ...Run) *Document {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return d.append(Heading{level: level, runs: runs})
}

// Paragraph appends a body paragraph composed of inline runs.
func (d *Document) Paragraph(runs ...Run) *Document {
	return d.append(Paragraph{runs: runs})
}

// PageBreak appends a forced page break.
func (d *Document) PageBreak() *Document { return d.append(PageBreak{}) }

// Spacer appends vertical whitespace of the given height.
func (d *Document) Spacer(h Length) *Document { return d.append(Spacer{Height: h}) }

// ErrNotImplemented is retained as a sentinel for callers who wrote
// against the skeleton phase. Once the layout and typography tracks land
// it will be unreferenced from the render path and may be removed in a
// future minor version.
var ErrNotImplemented = errors.New("kardec: render path not implemented yet")

// Render produces a PDF and writes it to the named file. The file is
// created (or truncated) with default permissions and closed before
// Render returns; callers don't manage the handle.
func (d *Document) Render(path string) error {
	if d.err != nil {
		return d.err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return d.RenderTo(f)
}

// RenderTo produces a PDF and writes it to the supplied io.Writer. The
// pipeline is: builder model -> stub layout (until the Layout track
// lands) -> internal/pdf writer. Errors from the writer propagate
// unchanged.
func (d *Document) RenderTo(w io.Writer) error {
	if d.err != nil {
		return d.err
	}
	model := d.toPDFModel()
	return pdfwriter.Writer{}.Write(w, model)
}

// Bytes returns the rendered PDF as a byte slice. Convenient for tests
// and for HTTP handlers that buffer responses; for large documents
// callers should prefer RenderTo + io.Pipe.
func (d *Document) Bytes() ([]byte, error) {
	if d.err != nil {
		return nil, d.err
	}
	var buf bytes.Buffer
	if err := d.RenderTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// toPDFModel converts the builder state into the renderer's input model.
//
// LAYOUT-TRACK STUB: until the Layout agent lands, this function emits a
// single blank page sized to the first section's PageSetup with a
// placeholder TextItem that references font ID 0. Because Fonts is left
// empty, the content-stream builder drops the item — the rendered PDF
// shows a blank page (still a valid PDF that opens in viewers, which is
// the v0.1 acceptance criterion). When Layout integrates, this body is
// replaced with the real walk over Sections/Blocks/Runs and the embedded
// font registry; the public Render/RenderTo/Bytes signatures stay
// unchanged.
func (d *Document) toPDFModel() pdfwriter.Document {
	if len(d.sections) == 0 {
		return pdfwriter.Document{}
	}
	setup := d.sections[0].Setup
	w, h := setup.Size.Width.Points(), setup.Size.Height.Points()
	if setup.Orientation == Landscape {
		w, h = h, w
	}
	return pdfwriter.Document{
		Title: "",
		Pages: []pdfwriter.Page{{
			Width:  w,
			Height: h,
			Items: []pdfwriter.TextItem{{
				X: 72, Y: h - 72,
				Text:     "Render placeholder — Layout track integrates next",
				FontID:   0,
				FontSize: 12,
				Color:    pdfwriter.Color{R: 0, G: 0, B: 0},
			}},
		}},
	}
}
