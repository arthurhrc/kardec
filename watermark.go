package kardec

// WatermarkOptions tunes the appearance of a document watermark.
// The zero value renders a sensible default (gray, 30 % opacity,
// 45° diagonal, 60 pt) so callers needing the canonical "DRAFT"
// stamp pass an empty struct.
//
// Color is the device-RGB fill colour of the rendered glyphs.
// When zero (R=G=B=0 — pure black), the renderer substitutes a
// neutral gray so the stamp doesn't look like body text.
//
// Opacity is in [0, 1]. Values outside that range are clamped;
// opacity == 1 omits the /ExtGState alpha entry (fully opaque).
//
// AngleDeg is the counter-clockwise rotation in degrees applied
// around the page centre. 0 renders the watermark horizontally
// across the middle; 45 (the default) gives the canonical
// diagonal layout.
//
// FontSize is the rendered point size. Scales with page size —
// landscape A3 documents may want 100+ pt for the stamp to read
// at thumbnail scale.
type WatermarkOptions struct {
	Color    Color
	Opacity  float64
	AngleDeg float64
	FontSize float64
}

// watermarkConfig is the document-local storage form of the
// configured watermark. Mirrors the fields the internal pdf
// package's Watermark struct expects but does not depend on
// internal/pdf so the public surface stays free of internal
// imports.
type watermarkConfig struct {
	text     string
	color    Color
	opacity  float64
	angleDeg float64
	fontSize float64
}

// SetWatermark stamps text diagonally across every page in the
// document. The renderer paints the stamp after primary content so
// it sits on top of body text and images; opacity blending keeps
// the underlying content readable.
//
// Calling with an empty text disables any prior watermark.
//
// Defaults applied when the corresponding WatermarkOptions field
// is zero:
//
//	Color    → mid-gray (#888888) so stamp reads as auxiliary
//	Opacity  → 0.30
//	AngleDeg → 45 (canonical diagonal)
//	FontSize → 60 pt
func (d *Document) SetWatermark(text string, opts ...WatermarkOptions) *Document {
	if d.err != nil {
		return d
	}
	if text == "" {
		d.watermark = nil
		return d
	}
	cfg := watermarkConfig{
		text:     text,
		color:    Color{R: 0x88, G: 0x88, B: 0x88},
		opacity:  0.30,
		angleDeg: 45,
		fontSize: 60,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.Color != (Color{}) {
			cfg.color = o.Color
		}
		if o.Opacity > 0 && o.Opacity <= 1 {
			cfg.opacity = o.Opacity
		}
		if o.AngleDeg != 0 {
			cfg.angleDeg = o.AngleDeg
		}
		if o.FontSize > 0 {
			cfg.fontSize = o.FontSize
		}
	}
	d.watermark = &cfg
	return d
}

// Watermark reports the configured watermark text plus a boolean
// indicating whether SetWatermark was called. The render bridge
// consults this to populate pdf.Document.Watermark.
func (d *Document) Watermark() (text string, ok bool) {
	if d.watermark == nil {
		return "", false
	}
	return d.watermark.text, true
}

// WatermarkConfig is the resolved per-document watermark
// configuration the render bridge consumes. Its fields mirror the
// inputs SetWatermark accepted, with defaults already filled in so
// callers do not need to re-derive them.
//
// This type is the friend-package seam: the render package reads
// WatermarkConfig to populate the internal pdf.Watermark struct.
// User code rarely constructs one directly — use SetWatermark and
// the public WatermarkOptions instead.
type WatermarkConfig struct {
	Text     string
	Color    Color
	Opacity  float64
	AngleDeg float64
	FontSize float64
}

// WatermarkResolved returns the configured watermark fields with
// defaults applied, plus a boolean indicating whether SetWatermark
// was called. Used by render to populate pdf.Document.Watermark.
//
// Deprecated: friend-package seam. Stable while exported but the
// surface is expected to move internal at v1.0.
func (d *Document) WatermarkResolved() (WatermarkConfig, bool) {
	if d.watermark == nil {
		return WatermarkConfig{}, false
	}
	w := d.watermark
	return WatermarkConfig{
		Text:     w.text,
		Color:    w.color,
		Opacity:  w.opacity,
		AngleDeg: w.angleDeg,
		FontSize: w.fontSize,
	}, true
}
