// Package render is the orchestrator that turns a kardec.Document into a PDF
// byte stream. It plugs three internal subsystems together:
//
//   - internal/layout    walks the Document tree, breaks lines, places blocks
//   - internal/typography resolves font faces and provides text measurement
//   - internal/pdf        writes the final PDF 1.7 byte stream
//
// Importing this package wires Document.Render / RenderTo / Bytes via an
// init() hook in kardec; the public surface here is therefore optional —
// users can call render.ToFile / ToWriter / Bytes directly, or rely on the
// method API after a blank import:
//
//	import (
//	    "github.com/arthurhrc/kardec"
//	    _ "github.com/arthurhrc/kardec/render"
//	)
//
// The indirection avoids an import cycle: kardec cannot import internal/layout
// because layout already imports kardec to walk the document tree.
package render

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/internal/layout"
	"github.com/arthurhrc/kardec/internal/pdf"
	"github.com/arthurhrc/kardec/internal/typography"
)

func init() {
	kardec.SetRenderImpl(renderImpl)
}

// ToFile renders d as a PDF and writes it to path. The file is created
// (or truncated) and closed before ToFile returns.
func ToFile(d *kardec.Document, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return ToWriter(d, f)
}

// ToWriter renders d as a PDF to the supplied io.Writer.
func ToWriter(d *kardec.Document, w io.Writer) error {
	if err := d.Err(); err != nil {
		return err
	}
	return renderImpl(d, w)
}

// Bytes renders d as a PDF and returns the bytes. Convenient for tests and
// HTTP handlers that buffer responses.
func Bytes(d *kardec.Document) ([]byte, error) {
	if err := d.Err(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := renderImpl(d, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// defaultFontFile is the single TTF the v0.1 PDF writer embeds. Every text
// item in the rendered document references this font, regardless of the
// (family, weight, italic) tuple the layout engine resolved. Multi-font
// fidelity (bold runs really bold, italic runs really italic) lands in v0.2
// once the typography registry exposes ttf bytes per face.
const defaultFontFile = "embedded/LiberationSans-Regular.ttf"

// renderImpl is the canonical implementation registered with kardec at init.
// It runs the layout engine over the document, converts the layout pages to
// the PDF writer's input model, and emits PDF 1.7 bytes.
func renderImpl(d *kardec.Document, w io.Writer) error {
	registry := d.FontRegistry()
	provider := newLayoutFontProvider(registry)

	pages, err := layout.Layout(d, provider)
	if err != nil {
		return fmt.Errorf("render: layout: %w", err)
	}

	model, err := buildPDFModel(pages)
	if err != nil {
		return fmt.Errorf("render: build pdf model: %w", err)
	}
	if err := (pdf.Writer{}).Write(w, model); err != nil {
		return fmt.Errorf("render: pdf write: %w", err)
	}
	return nil
}

// buildPDFModel converts layout output into the pdf package's input shape.
// Coordinates are flipped from layout's top-left origin to PDF's bottom-left.
//
// v0.1 limitation: only Liberation Sans Regular is embedded. Every text item
// uses font ID 0 even if the source style requested bold or italic. Layout
// still measures correctly via the typography registry because measurement
// and embedding are decoupled — the visual difference is glyph weight only.
func buildPDFModel(pages []layout.Page) (pdf.Document, error) {
	ttf, err := typography.FontsFS.ReadFile(defaultFontFile)
	if err != nil {
		return pdf.Document{}, fmt.Errorf("read default font %s: %w", defaultFontFile, err)
	}
	out := pdf.Document{
		Fonts: []pdf.EmbeddedFont{{
			Name:    "LiberationSans",
			TTFData: ttf,
		}},
	}

	for _, lp := range pages {
		pdfPage := pdf.Page{
			Width:  lp.Size.Width.Points(),
			Height: lp.Size.Height.Points(),
		}
		for _, item := range lp.Items {
			pdfPage.Items = append(pdfPage.Items, pdf.TextItem{
				X:        item.X.Points(),
				Y:        pdfPage.Height - item.Y.Points(),
				Text:     item.Text,
				FontID:   0,
				FontSize: item.Size.Points(),
				Color:    pdf.Color{R: item.Color.R, G: item.Color.G, B: item.Color.B},
			})
		}
		out.Pages = append(out.Pages, pdfPage)
	}
	return out, nil
}
