package kardec

import (
	"math"
	"testing"
)

func TestStandardPageSizes(t *testing.T) {
	// A4: 210 × 297 mm; convert to points and verify within ε.
	if math.Abs(PageA4.Width.Millimeters()-210) > 1e-6 {
		t.Errorf("A4 width = %v mm, want 210", PageA4.Width.Millimeters())
	}
	if math.Abs(PageA4.Height.Millimeters()-297) > 1e-6 {
		t.Errorf("A4 height = %v mm, want 297", PageA4.Height.Millimeters())
	}
	if math.Abs(PageLetter.Width.Points()-612) > 1e-3 {
		t.Errorf("Letter width = %v pt, want 612", PageLetter.Width.Points())
	}
}

func TestSymmetricMargins(t *testing.T) {
	m := Symmetric(Cm(2))
	if m.Top != m.Bottom || m.Left != m.Right || m.Top != m.Left {
		t.Errorf("Symmetric should produce equal sides, got %+v", m)
	}
}

func TestMarginPresetsMatchWordDefaults(t *testing.T) {
	cases := []struct {
		preset Margins
		wantCm float64
	}{
		{MarginsNarrow, 1.27},
		{MarginsNormal, 2.54},
		{MarginsWide, 5.08},
	}
	for _, c := range cases {
		gotMm := c.preset.Top.Millimeters()
		if math.Abs(gotMm-c.wantCm*10) > 1e-3 {
			t.Errorf("preset top = %v mm, want %v", gotMm, c.wantCm*10)
		}
	}
}
