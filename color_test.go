package kardec

import "testing"

func TestHexColorParsing(t *testing.T) {
	cases := []struct {
		in   string
		want Color
	}{
		{"#2E74B5", Color{0x2E, 0x74, 0xB5}},
		{"2E74B5", Color{0x2E, 0x74, 0xB5}},
		{"#fff", Color{0xFF, 0xFF, 0xFF}},
		{"  #000  ", Color{0, 0, 0}},
	}
	for _, c := range cases {
		got := HexColor(c.in)
		if got != c.want {
			t.Errorf("HexColor(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestHexColorInvalidFallsBackToBlack(t *testing.T) {
	got := HexColor("not-a-color")
	if got != ColorBlack {
		t.Errorf("invalid hex should fall back to black, got %+v", got)
	}
}

func TestColorHexRoundTrip(t *testing.T) {
	c := Color{0x2E, 0x74, 0xB5}
	if c.Hex() != "#2E74B5" {
		t.Errorf("Hex() = %s, want #2E74B5", c.Hex())
	}
}
