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
	// kardec.SetRenderImpl is the friend-package seam this file
	// exists to consume. The Deprecated comment on the kardec
	// side warns user code away; this package is the only
	// legitimate caller and stays so until the seam moves
	// internal at v1.0.
	//lint:ignore SA1019 friend-package seam; render is the legitimate caller
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
	// FontRegistry is the friend-package seam exposing the
	// internal *typography.Registry to render. User code should
	// use Document.RegisteredFamilies; render needs the full
	// registry to construct a FontProvider.
	//lint:ignore SA1019 friend-package seam; render needs the registry
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
	if profile, n := d.ICCProfile(); len(profile) > 0 {
		model.ICCProfile = profile
		model.ICCProfileN = n
	}
	if opts, ok := d.Encryption(); ok {
		model.Encryption = &pdf.Encryption{
			UserPwd:     opts.UserPassword,
			OwnerPwd:    opts.OwnerPassword,
			Permissions: encryptionPermissionBits(opts.Permissions),
		}
	}
	if lang, ok := d.Tagged(); ok {
		model.Tagged = true
		model.Lang = lang
	}
	//lint:ignore SA1019 friend-package seam; render is the legitimate caller
	if cfg, ok := d.WatermarkResolved(); ok {
		// Watermark uses the registry's default (regular, non-italic)
		// face. Body text on each page may not reference this font;
		// ensureFontIncluded in the writer guarantees the watermark
		// face still lands in the page's /Font dict.
		model.Watermark = &pdf.Watermark{
			Text:     cfg.Text,
			FontID:   0,
			FontSize: cfg.FontSize,
			Color:    pdf.Color{R: cfg.Color.R, G: cfg.Color.G, B: cfg.Color.B},
			Opacity:  cfg.Opacity,
			AngleDeg: cfg.AngleDeg,
		}
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

	images, imageIndex, bgIndex, err := buildEmbeddedImages(pages)
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
		// Page background: covers the full MediaBox underneath all
		// other draws. The image-embed table keyed by raw bytes
		// resolves repeated backgrounds (every page in a section
		// shares one) to a single XObject.
		if id, ok := bgIndex[string(lp.BackgroundImage)]; ok && len(lp.BackgroundImage) > 0 {
			pdfPage.Images = append(pdfPage.Images, pdf.ImageDraw{
				X:       0,
				Y:       0,
				W:       pdfPage.Width,
				H:       pdfPage.Height,
				ImageID: id,
			})
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
		// Per-block PDF/UA structure groups: walk the layout
		// items in order and start a new StructBlock every time
		// the role changes. Items with empty role are folded
		// into the surrounding block.
		pdfPage.StructBlocks = buildStructBlocks(lp)
		out.Pages = append(out.Pages, pdfPage)
	}
	return out, index, nil
}

// buildStructBlocks groups consecutive PlacedItems into a tree
// of pdf.StructBlock entries. Three passes: leaf-block extraction,
// table-cell hierarchy folding, and Sect grouping around H1
// boundaries.
//
// The Sect pass is per-page (it doesn't span pages today — that
// would require model awareness across page boundaries the writer
// doesn't yet wire through). Within a page, every H1 starts a new
// Sect and absorbs the following P / H2-H6 / Figure / Table blocks
// into its Children. Pages with no H1 stay flat.
func buildStructBlocks(p layout.Page) []pdf.StructBlock {
	leaves := buildLeafBlocks(p)
	if len(leaves) == 0 {
		return nil
	}
	flat := foldTableHierarchy(leaves)
	return foldSectHierarchy(flat)
}

// foldSectHierarchy walks the flat top-level block sequence and
// wraps each H1 + its following non-H1 siblings in a /Sect
// container. The H1 stays inside the Sect as the first child so
// screen readers walking the tree announce the section title
// before its body. H2-H6 don't open new Sect levels here — they
// stay siblings of the surrounding P. Nested-Sect support (H1
// containing H2 sub-sections) lands once the layout engine
// surfaces multi-level outlines.
func foldSectHierarchy(blocks []pdf.StructBlock) []pdf.StructBlock {
	var out []pdf.StructBlock
	i := 0
	// Skip leading non-H1 blocks — they stay at the top of the
	// page tree (e.g., a header strip or pre-section paragraph).
	for i < len(blocks) && blocks[i].Role != "H1" {
		out = append(out, blocks[i])
		i++
	}
	// From the first H1, every block joins the running Sect
	// until the next H1 starts a new one.
	for i < len(blocks) {
		if blocks[i].Role != "H1" {
			// Defensive: an H1 was always supposed to come first
			// here, but handle non-H1 by appending at the top
			// level just in case.
			out = append(out, blocks[i])
			i++
			continue
		}
		sect := pdf.StructBlock{Role: "Sect", Children: []pdf.StructBlock{blocks[i]}}
		i++
		for i < len(blocks) && blocks[i].Role != "H1" {
			sect.Children = append(sect.Children, blocks[i])
			i++
		}
		out = append(out, sect)
	}
	return out
}

// buildLeafBlocks produces the flat sequence of leaf blocks: one
// per role / cell transition. Each leaf carries its (role,
// tableID, rowIdx, colIdx) so foldTableHierarchy can group
// table cells into TR + Table parents in a second pass.
func buildLeafBlocks(p layout.Page) []leafBlock {
	if len(p.Items) == 0 {
		return nil
	}
	var leaves []leafBlock
	cur := leafBlock{role: "", tableID: -1}
	pdfTextIdx := 0
	pdfImageIdx := 0
	flush := func(at int, atImg int) {
		if cur.role == "" {
			return
		}
		cur.itemEnd = at
		cur.imageEnd = atImg
		leaves = append(leaves, cur)
	}
	for _, it := range p.Items {
		role := string(it.Role)
		if role == "" {
			role = "P"
		}
		// Open a new leaf when role OR table-cell coordinates
		// change. Cells inside the same table change rowIdx /
		// colIdx; transitioning rowIdx + colIdx within the same
		// tableID still requires a new leaf because each cell
		// owns its own MCID.
		newLeaf := role != cur.role ||
			it.TableID != cur.tableID ||
			(it.TableID != 0 && (it.RowIdx != cur.rowIdx || it.ColIdx != cur.colIdx))
		if newLeaf {
			flush(pdfTextIdx, pdfImageIdx)
			cur = leafBlock{
				role:       role,
				itemStart:  pdfTextIdx,
				imageStart: pdfImageIdx,
				tableID:    it.TableID,
				rowIdx:     it.RowIdx,
				colIdx:     it.ColIdx,
			}
		}
		switch {
		case it.Image != nil:
			pdfImageIdx++
		case it.Rect != nil:
			// Rects don't contribute to MCIDs.
		default:
			pdfTextIdx++
		}
	}
	flush(pdfTextIdx, pdfImageIdx)
	return leaves
}

// leafBlock is render's intermediate flat representation. It
// carries enough info for foldTableHierarchy to produce a tree.
type leafBlock struct {
	role       string
	itemStart  int
	itemEnd    int
	imageStart int
	imageEnd   int
	tableID    int
	rowIdx     int
	colIdx     int
}

// foldTableHierarchy walks the flat leaf sequence and groups
// consecutive cells (TableID == same, non-zero) into a /Table
// container, with one /TR child per row, one /TD or /TH child per
// cell. Non-table leaves pass through unchanged at the same level.
func foldTableHierarchy(leaves []leafBlock) []pdf.StructBlock {
	var out []pdf.StructBlock
	i := 0
	for i < len(leaves) {
		L := leaves[i]
		if L.tableID == 0 {
			out = append(out, pdf.StructBlock{
				Role:       L.role,
				ItemStart:  L.itemStart,
				ItemEnd:    L.itemEnd,
				ImageStart: L.imageStart,
				ImageEnd:   L.imageEnd,
			})
			i++
			continue
		}
		// Found the first cell of a table. Greedily consume all
		// leaves with the same TableID, grouping them by RowIdx
		// to build TR children, with one TD/TH leaf inside each.
		tID := L.tableID
		var rows []pdf.StructBlock
		var curRow []pdf.StructBlock
		curRowIdx := L.rowIdx
		for i < len(leaves) && leaves[i].tableID == tID {
			c := leaves[i]
			if c.rowIdx != curRowIdx && len(curRow) > 0 {
				rows = append(rows, pdf.StructBlock{Role: "TR", Children: curRow})
				curRow = nil
				curRowIdx = c.rowIdx
			}
			curRow = append(curRow, pdf.StructBlock{
				Role:       c.role, // TD or TH
				ItemStart:  c.itemStart,
				ItemEnd:    c.itemEnd,
				ImageStart: c.imageStart,
				ImageEnd:   c.imageEnd,
			})
			i++
		}
		if len(curRow) > 0 {
			rows = append(rows, pdf.StructBlock{Role: "TR", Children: curRow})
		}
		out = append(out, pdf.StructBlock{Role: "Table", Children: rows})
	}
	return out
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
