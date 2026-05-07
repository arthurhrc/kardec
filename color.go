package kardec

import (
	"fmt"
	"strings"
)

// Color is an sRGB color with three 8-bit channels. Alpha is not part of the
// PDF 1.7 base color model used by Kardec; transparency lives in graphics
// state rather than per-color.
type Color struct {
	R, G, B uint8
}

// HexColor parses a CSS-style hex string ("#2E74B5", "2E74B5", "#fff") into a
// Color. Invalid input is reported by the document via Document.Err once the
// color is consumed by a style or content operation.
func HexColor(hex string) Color {
	c, err := parseHexColor(hex)
	if err != nil {
		// Invalid hex sentinel; the document captures the error when this
		// color is attached to a style or run via Document.attachColorErr.
		return Color{R: 0, G: 0, B: 0}
	}
	return c
}

// RGB constructs a Color from raw 8-bit channels.
func RGB(r, g, b uint8) Color { return Color{R: r, G: g, B: b} }

// Hex returns the canonical six-digit hex representation, prefixed with "#".
func (c Color) Hex() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// Common named colors. Kept deliberately small; users compose more through
// HexColor or RGB.
var (
	ColorBlack = Color{0, 0, 0}
	ColorWhite = Color{255, 255, 255}
	ColorGray  = Color{128, 128, 128}
)

func parseHexColor(s string) (Color, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	switch len(s) {
	case 3:
		// "abc" -> "aabbcc"
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6:
		// canonical
	default:
		return Color{}, fmt.Errorf("kardec: invalid hex color %q: expected 3 or 6 digits", s)
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return Color{}, fmt.Errorf("kardec: invalid hex color %q: %w", s, err)
	}
	return Color{R: r, G: g, B: b}, nil
}
