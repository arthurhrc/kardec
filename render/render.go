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
	"time"

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

// renderImpl is the canonical implementation registered with kardec at
// init. It runs the layout engine over the document, converts the
// layout pages to the PDF writer's input model, and emits PDF 1.7 bytes.
func renderImpl(d *kardec.Document, w io.Writer) error {
	registry := d.FontRegistry()
	provider := newLayoutFontProvider(registry)

	pages, err := layout.Layout(d, provider)
	if err != nil {
		return fmt.Errorf("render: layout: %w", err)
	}

	model, fontIdx, err := buildPDFModel(pages, registry)
	if err != nil {
		return fmt.Errorf("render: build pdf model: %w", err)
	}
	if d.FontSubsetEnabled() {
		applyFontSubset(model.Fonts, pages, fontIdx)
	}
	if d.PDFAEnabled() {
		model.PDFA = true
	}
	model.Title = d.Title()
	model.Author = d.Author()
	model.Subject = d.Subject()
	model.Keywords = d.Keywords()
	writer := pdf.Writer{}
	if t, ok := d.CreationDate(); ok {
		fixed := t
		writer.Clock = func() time.Time { return fixed }
	}
	if err := writer.Write(w, model); err != nil {
		return fmt.Errorf("render: pdf write: %w", err)
	}
	return nil
}

// fontKey identifies an (family, bold, italic) tuple within the
// embedded-font index. Mirrors the inputs to layout.FontProvider.Resolve
// so a PlacedItem's measureAdapter maps cleanly onto a registered face.
type fontKey struct {
	family string
	bold   bool
	italic bool
}

// buildPDFModel converts layout output into the pdf package's input
// shape. Coordinates are flipped from layout's top-left origin to PDF's
// bottom-left.
//
// Only faces actually referenced by a PlacedItem are embedded; the rest
// of the registry is left out so the resulting PDF stays close in size
// to the v0.1 single-font baseline. Subsetting (trimming individual
// glyphs within an embedded face) is a v0.3 feature.
func buildPDFModel(pages []layout.Page, registry *typography.Registry) (pdf.Document, map[fontKey]int, error) {
	used := collectUsedFontKeys(pages)
	embedded, index, defaultID := assembleEmbeddedFonts(registry, used)

	mathID, embeddedWithMath := appendMathFontIfUsed(embedded, pages)
	embedded = embeddedWithMath

	images, imageIndex, err := buildEmbeddedImages(pages)
	if err != nil {
		return pdf.Document{}, nil, err
	}

	out := pdf.Document{
		Fonts:        embedded,
		Images:       images,
		Outlines:     buildOutline(pages),
		Destinations: buildDestinations(pages),
	}
	for _, lp := range pages {
		pdfPage := pdf.Page{
			Width:  lp.Width.Points(),
			Height: lp.Height.Points(),
		}
		linkRanges := newLinkRangeAccumulator()
		for _, item := range lp.Items {
			if item.Rect != nil {
				w := item.Rect.Width.Points()
				h := item.Rect.Thickness.Points()
				pdfPage.Rects = append(pdfPage.Rects, pdf.RectDraw{
					X:     item.X.Points(),
					Y:     pdfPage.Height - item.Y.Points() - h,
					W:     w,
					H:     h,
					Color: pdf.Color{R: item.Rect.Color.R, G: item.Rect.Color.G, B: item.Rect.Color.B},
				})
				continue
			}
			if item.Image != nil {
				imgID, ok := imageIndex[item.Image]
				if !ok {
					continue
				}
				w := item.Image.Width.Points()
				h := item.Image.Height.Points()
				pdfPage.Images = append(pdfPage.Images, pdf.ImageDraw{
					X:       item.X.Points(),
					Y:       pdfPage.Height - item.Y.Points() - h,
					W:       w,
					H:       h,
					ImageID: imgID,
				})
				continue
			}
			id := defaultID
			if item.IsMath && mathID >= 0 {
				id = mathID
			} else if a, ok := item.Font.(*measureAdapter); ok {
				if mapped, found := index[fontKey{family: a.family, bold: a.bold, italic: a.italic}]; found {
					id = mapped
				}
			}
			pdfX := item.X.Points()
			pdfY := pdfPage.Height - item.Y.Points()
			pdfPage.Items = append(pdfPage.Items, pdf.TextItem{
				X:        pdfX,
				Y:        pdfY,
				Text:     item.Text,
				FontID:   id,
				FontSize: item.Size.Points(),
				Color:    pdf.Color{R: item.Color.R, G: item.Color.G, B: item.Color.B},
			})
			if item.Link != "" {
				// Approximate the visible glyph extent by font size:
				// width = len(text) × size × 0.55 keeps the click box
				// generous without TTF-precise measurement.
				w := float64(len(item.Text)) * item.Size.Points() * 0.55
				h := item.Size.Points() * 1.2
				linkRanges.add(item.Link, pdfX, pdfY-item.Size.Points()*0.2, w, h)
			}
		}
		pdfPage.Links = linkRanges.flush()
		out.Pages = append(out.Pages, pdfPage)
	}
	return out, index, nil
}

// collectUsedFontKeys walks every PlacedItem on every page and gathers
// the set of (family, bold, italic) tuples each measureAdapter carried.
// Items whose Font is not a *measureAdapter (stub items) are ignored.
func collectUsedFontKeys(pages []layout.Page) map[fontKey]struct{} {
	used := make(map[fontKey]struct{})
	for _, p := range pages {
		for _, it := range p.Items {
			a, ok := it.Font.(*measureAdapter)
			if !ok {
				continue
			}
			used[fontKey{family: a.family, bold: a.bold, italic: a.italic}] = struct{}{}
		}
	}
	return used
}

// assembleEmbeddedFonts builds the pdf.EmbeddedFont slice that includes
// only the faces referenced by used. The returned index maps each
// fontKey to its position in the slice. defaultID points at the first
// regular, non-italic face that made it in (or 0 when nothing did).
//
// At least one face is always embedded so the PDF writer has a font to
// reference even for documents that had no measurable runs.
func assembleEmbeddedFonts(registry *typography.Registry, used map[fontKey]struct{}) (
	[]pdf.EmbeddedFont, map[fontKey]int, int,
) {
	faces := registry.Faces()
	embedded := make([]pdf.EmbeddedFont, 0, len(faces))
	index := make(map[fontKey]int)
	defaultID := 0

	for _, f := range faces {
		bold := f.Weight >= typography.Bold
		key := fontKey{family: f.Family, bold: bold, italic: f.Italic}
		if _, hit := used[key]; !hit {
			continue
		}
		idx := len(embedded)
		embedded = append(embedded, pdf.EmbeddedFont{
			Name:    faceFontName(f.Family, f.Weight, f.Italic),
			TTFData: f.Bytes,
		})
		index[key] = idx
		if defaultID == 0 && f.Weight == typography.Regular && !f.Italic {
			defaultID = idx
		}
	}

	// Guarantee at least one embedded face so the PDF writer has
	// something to reference. Fall back to the registry default.
	if len(embedded) == 0 {
		def := registry.Default()
		if def != nil {
			for _, f := range faces {
				if f.Font == def {
					embedded = append(embedded, pdf.EmbeddedFont{
						Name:    faceFontName(f.Family, f.Weight, f.Italic),
						TTFData: f.Bytes,
					})
					index[fontKey{family: f.Family, bold: f.Weight >= typography.Bold, italic: f.Italic}] = 0
					break
				}
			}
		}
	}
	return embedded, index, defaultID
}

// faceFontName produces a PostScript-style identifier for an
// (family, weight, italic) tuple. Spaces in the family are removed; a
// dash plus the qualifying suffix is appended when the face is anything
// other than plain Regular.
func faceFontName(family string, weight typography.Weight, italic bool) string {
	base := strings_replaceAll(family, " ", "")
	suffix := ""
	switch {
	case weight >= typography.Bold && italic:
		suffix = "-BoldItalic"
	case weight >= typography.Bold:
		suffix = "-Bold"
	case italic:
		suffix = "-Italic"
	}
	return base + suffix
}

// strings_replaceAll inlines strings.ReplaceAll(s, old, new) — keeping
// this file out of the main strings dependency footprint while the
// helper is the only consumer. Drop in favor of strings.ReplaceAll if
// other helpers in this package need it later.
func strings_replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	out := []byte{}
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			out = append(out, new...)
			i += len(old)
			continue
		}
		out = append(out, s[i])
		i++
	}
	return string(out)
}
