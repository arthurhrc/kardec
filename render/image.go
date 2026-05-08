package render

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // image.Decode dispatch
	"image/png"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/internal/layout"
	"github.com/arthurhrc/kardec/internal/pdf"
)

// imageEntry caches the conversion of a single source image so multi-page
// documents that reference the same logo only pay decoding/embedding cost
// once. The map key is the address of the PlacedImage's payload header,
// which the layout engine reuses across PlacedItem instances.
type imageEntry struct {
	id    int           // index into pdf.Document.Images
	embed pdf.EmbeddedImage
}

// buildEmbeddedImages walks every PlacedImage on every page and assembles
// the pdf.Document.Images table. Returns the table plus an index keyed by
// the layout PlacedImage pointer so per-page draws can resolve their ID.
//
// JPEG payloads pass through verbatim. PNG payloads are decoded via
// stdlib image/png and re-encoded as packed 8-bit RGB (alpha is
// composited over white). Unknown formats are silently dropped — the
// kardec.Image builder rejects them earlier, so reaching this branch
// indicates a layout-level bug rather than user input.
func buildEmbeddedImages(pages []layout.Page) ([]pdf.EmbeddedImage, map[*layout.PlacedImage]int, error) {
	out := []pdf.EmbeddedImage{}
	index := map[*layout.PlacedImage]int{}
	for _, p := range pages {
		for _, it := range p.Items {
			if it.Image == nil {
				continue
			}
			if _, seen := index[it.Image]; seen {
				continue
			}
			embed, err := encodeImage(it.Image)
			if err != nil {
				return nil, nil, err
			}
			id := len(out)
			out = append(out, embed)
			index[it.Image] = id
			_ = imageEntry{id: id, embed: embed}
		}
	}
	return out, index, nil
}

// encodeImage maps a layout PlacedImage to the pdf.EmbeddedImage shape
// the writer consumes. Pixel dimensions are read from the source so the
// PDF's /Width and /Height match what was actually decoded.
func encodeImage(img *layout.PlacedImage) (pdf.EmbeddedImage, error) {
	switch img.Format {
	case kardec.ImageFormatJPEG:
		w, h, err := decodeNaturalSize(img.Data)
		if err != nil {
			return pdf.EmbeddedImage{}, fmt.Errorf("render: decode jpeg: %w", err)
		}
		return pdf.EmbeddedImage{
			WidthPx:  w,
			HeightPx: h,
			Encoding: pdf.ImageJPEG,
			Data:     img.Data,
		}, nil
	case kardec.ImageFormatPNG:
		decoded, err := png.Decode(bytes.NewReader(img.Data))
		if err != nil {
			return pdf.EmbeddedImage{}, fmt.Errorf("render: decode png: %w", err)
		}
		rgb, w, h := flattenRGB(decoded)
		return pdf.EmbeddedImage{
			WidthPx:  w,
			HeightPx: h,
			Encoding: pdf.ImageRawRGB,
			Data:     rgb,
		}, nil
	default:
		return pdf.EmbeddedImage{}, fmt.Errorf("render: unsupported image format %s", img.Format)
	}
}

// decodeNaturalSize reads only the header of an image to recover pixel
// dimensions, avoiding a full decode for JPEGs that pass through.
func decodeNaturalSize(data []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// flattenRGB walks the decoded image, composes any alpha against an
// opaque white background, and emits packed 8-bit RGB triples in the
// order PDF expects (top-to-bottom, left-to-right). Returns the bytes
// plus the pixel width and height.
func flattenRGB(img image.Image) ([]byte, int, int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]byte, 0, w*h*3)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			// RGBA() returns 16-bit channels in the alpha-premultiplied
			// space. Composite against white when alpha < 0xFFFF.
			if a < 0xFFFF {
				inv := 0xFFFF - a
				r = r + inv
				g = g + inv
				bl = bl + inv
			}
			out = append(out,
				byte(r>>8),
				byte(g>>8),
				byte(bl>>8),
			)
		}
	}
	return out, w, h
}
