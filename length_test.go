package kardec

import (
	"math"
	"testing"
)

func TestLengthConstructors(t *testing.T) {
	cases := []struct {
		name   string
		got    Length
		wantPt float64
	}{
		{"Pt direct", Pt(72), 72},
		{"In to Pt", In(1), 72},
		{"Mm A4 width", Mm(210), 595.275590},
		{"Cm 2.54 = 1in", Cm(2.54), 72},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if math.Abs(c.got.Points()-c.wantPt) > 1e-3 {
				t.Errorf("got %v points, want %v", c.got.Points(), c.wantPt)
			}
		})
	}
}

func TestLengthRoundTrip(t *testing.T) {
	l := Mm(50)
	if math.Abs(l.Millimeters()-50) > 1e-6 {
		t.Errorf("mm round-trip lost precision: %v", l.Millimeters())
	}
}
