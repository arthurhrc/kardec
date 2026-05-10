package render_test

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

const sampleSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="50" height="50" viewBox="0 0 50 50">
  <rect x="5" y="5" width="40" height="40" fill="#3366cc" stroke="black" stroke-width="2" />
  <circle cx="25" cy="25" r="10" fill="white" />
</svg>`

func TestSVGImageEmbedsAsFormXObject(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image([]byte(sampleSVG)).Width(kardec.Pt(80)).Build()

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	for _, want := range []string{
		"/Subtype /Form",   // Form XObject (vs /Subtype /Image)
		"/BBox [",          // bounding box
		"/Im0 Do",          // page invokes the form
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("SVG marker %q missing from PDF byte stream", want)
		}
	}
	for _, leak := range []string{
		"/Subtype /Image",
		"/Filter /DCTDecode",
	} {
		if bytes.Contains(out, []byte(leak)) {
			t.Errorf("SVG-only document should not contain raster marker %q", leak)
		}
	}
}

func TestImageFormatDetectsSVG(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image([]byte(sampleSVG)).Build()
	if err := doc.Err(); err != nil {
		t.Fatalf("doc.Err: %v", err)
	}
}



