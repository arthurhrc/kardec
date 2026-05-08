package layout

import (
	"bytes"
	"image"
	_ "image/jpeg" // register JPEG decoder for image.DecodeConfig
	_ "image/png"  // register PNG decoder for image.DecodeConfig
	"math"

	"github.com/arthurhrc/kardec"
)

// placeImage emits the PlacedItem for a kardec.Image block, paginating
// when the image does not fit on the remaining page.
//
// Sizing rules:
//   - Both Width and Height set → use both verbatim.
//   - Only one set         → derive the other from the natural aspect ratio.
//   - Neither set          → fall back to the natural pixel size assumed
//     to be at 72 DPI (1 px = 1 pt). Rare in practice; users normally
//     constrain at least one dimension.
//
// Images wider than the available content area are scaled down
// uniformly so they fit; images taller than the page receive a forced
// page break and are placed at the top of the new page (clipping is
// avoided by also scaling height down to the page if necessary).
func (e Engine) placeImage(cur *pageCursor, flush func(), img kardec.Image) error {
	natW, natH, err := imageNaturalSize(img.Data())
	if err != nil {
		return err
	}
	w, h := resolveImageDimensions(img, natW, natH)

	available := cur.availableWidth()
	if w > available && available > 0 {
		scale := available / w
		w *= scale
		h *= scale
	}
	pageH := cur.y1 - cur.y0
	if h > pageH && pageH > 0 {
		scale := pageH / h
		w *= scale
		h *= scale
	}

	if cur.remainingHeight() < h {
		flush()
	}

	x := cur.x0
	switch img.Alignment() {
	case kardec.AlignCenter:
		x = cur.x0 + (available-w)/2
	case kardec.AlignRight:
		x = cur.x0 + available - w
	}

	cur.items = append(cur.items, PlacedItem{
		X: kardec.Pt(x),
		Y: kardec.Pt(cur.cursorY),
		Image: &PlacedImage{
			Data:   img.Data(),
			Format: img.Format(),
			Width:  kardec.Pt(w),
			Height: kardec.Pt(h),
		},
	})
	cur.cursorY += h
	return nil
}

// imageNaturalSize reads only the header of an image payload to recover
// its pixel dimensions. Backed by the standard library's image.DecodeConfig
// which dispatches by registered format (jpeg, png).
func imageNaturalSize(data []byte) (float64, float64, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return float64(cfg.Width), float64(cfg.Height), nil
}

// resolveImageDimensions applies the sizing rules described on
// placeImage. Inputs come from the kardec.Image block (in points where
// non-zero) plus the natural pixel size (interpreted as points).
func resolveImageDimensions(img kardec.Image, natWPx, natHPx float64) (float64, float64) {
	w := img.Width().Points()
	h := img.Height().Points()
	switch {
	case w > 0 && h > 0:
		return w, h
	case w > 0:
		ratio := natHPx / natWPx
		if math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			ratio = 1
		}
		return w, w * ratio
	case h > 0:
		ratio := natWPx / natHPx
		if math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			ratio = 1
		}
		return h * ratio, h
	default:
		return natWPx, natHPx
	}
}
