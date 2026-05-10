package svg

import (
	"bytes"
	"strings"
	"testing"
)

func TestConvertEmptySVGReturnsCanvasOnly(t *testing.T) {
	src := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="50" height="40"></svg>`)
	w, h, stream, err := Convert(src)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if w != 50 || h != 40 {
		t.Errorf("canvas size: got %vx%v, want 50x40", w, h)
	}
	if len(stream) == 0 {
		t.Errorf("expected at least the cm matrix, got empty")
	}
	if !bytes.Contains(stream, []byte(" cm")) {
		t.Errorf("missing CTM operator in: %s", stream)
	}
}

func TestConvertRectEmitsRectAndFillOps(t *testing.T) {
	src := []byte(`<svg width="100" height="100" viewBox="0 0 100 100">
		<rect x="10" y="20" width="30" height="40" fill="#ff0000" />
	</svg>`)
	_, _, stream, err := Convert(src)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	s := string(stream)
	for _, want := range []string{
		" re",       // rectangle path operator
		" rg",       // nonstroking fill RGB
		"1.000000",  // red channel = 1.0
		"\nf\n",     // fill-only paint
	} {
		if !strings.Contains(s, want) {
			t.Errorf("rect output missing %q\nGot: %s", want, s)
		}
	}
}

func TestConvertCircleEmitsFourCubicBeziers(t *testing.T) {
	src := []byte(`<svg width="100" height="100"><circle cx="50" cy="50" r="20" fill="black" /></svg>`)
	_, _, stream, err := Convert(src)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	s := string(stream)
	count := strings.Count(s, " c\n")
	if count != 4 {
		t.Errorf("expected 4 cubic Beziers for a circle, got %d\nGot: %s", count, s)
	}
}

func TestConvertPathHandlesAbsoluteAndRelativeCommands(t *testing.T) {
	// "M 10 10 l 20 0 l 0 20 z" → triangle with last side via Z.
	src := []byte(`<svg width="100" height="100"><path d="M 10 10 l 20 0 l 0 20 z" stroke="blue" /></svg>`)
	_, _, stream, err := Convert(src)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	s := string(stream)
	for _, want := range []string{" m\n", " l\n", "h\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("path output missing %q\nGot: %s", want, s)
		}
	}
}

func TestConvertGroupInheritsFill(t *testing.T) {
	src := []byte(`<svg width="100" height="100">
		<g fill="red">
			<rect x="0" y="0" width="10" height="10" />
		</g>
	</svg>`)
	_, _, stream, err := Convert(src)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.Contains(string(stream), "1.000000 0.000000 0.000000 rg") {
		t.Errorf("group's fill should propagate to child rect; got %s", stream)
	}
}

func TestConvertMalformedXMLFails(t *testing.T) {
	if _, _, _, err := Convert([]byte("<svg <not-valid>")); err == nil {
		t.Errorf("expected error for malformed XML")
	}
}
