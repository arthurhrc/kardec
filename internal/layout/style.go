package layout

import "github.com/arthurhrc/kardec"

// blockStyle is the resolved style the engine consumes for a single
// block. It is intentionally tiny: only the fields that influence
// placement live here. The canonical source of these values is
// kardec.Document.ResolveBlockStyle; styleFromKardec converts the
// returned kardec.Style into the engine-internal shape.
type blockStyle struct {
	family        string
	sizePt        float64
	bold          bool
	italic        bool
	color         kardec.Color
	lineHeight    float64 // multiplier; 1.2 == 120% of size
	spaceBeforePt float64
	spaceAfterPt  float64
	alignment     kardec.Alignment
}

// styleFromKardec maps a fully-resolved kardec.Style onto the engine's
// internal representation. The conversion is straightforward: lengths
// become point floats, the Weight enum collapses to a bold flag (the
// only distinction the engine cares about for line breaking), and a
// zero LineHeight falls back to 1.2× — Kardec's documented body
// default.
func styleFromKardec(s kardec.Style) blockStyle {
	lh := s.LineHeight
	if lh <= 0 {
		lh = 1.2
	}
	return blockStyle{
		family:        s.Family,
		sizePt:        s.Size.Points(),
		bold:          s.Weight >= kardec.WeightSemiBold,
		italic:        s.Italic,
		color:         s.Color,
		lineHeight:    lh,
		spaceBeforePt: s.SpaceBefore.Points(),
		spaceAfterPt:  s.SpaceAfter.Points(),
		alignment:     s.Alignment,
	}
}

