package kardec_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
)

func crossrefPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 50, G: 50, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

func TestRef_FigureResolvesToVisibleTextAndAnchorLink(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image(crossrefPNG(t)).Label("growth-2024").Build()
	r := doc.Ref("growth-2024")
	if r.Text() != "Figure 1" {
		t.Errorf("Ref text = %q, want %q", r.Text(), "Figure 1")
	}
	if r.Link() != "#"+kardec.RefAnchorName("growth-2024") {
		t.Errorf("Ref link = %q, want %q", r.Link(), "#"+kardec.RefAnchorName("growth-2024"))
	}
}

func TestRef_TableResolvesWithSeparateCounter(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image(crossrefPNG(t)).Label("fig-a").Build().
		Table().Columns(kardec.Col("Col1")).Row("v").Label("tbl-revenue").Build()
	rfig := doc.Ref("fig-a")
	rtbl := doc.Ref("tbl-revenue")
	if rfig.Text() != "Figure 1" {
		t.Errorf("figure ref = %q, want Figure 1", rfig.Text())
	}
	if rtbl.Text() != "Table 1" {
		t.Errorf("table ref = %q, want Table 1", rtbl.Text())
	}
}

func TestRef_MultipleFiguresIncrementCounter(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image(crossrefPNG(t)).Label("a").Build().
		Image(crossrefPNG(t)).Label("b").Build().
		Image(crossrefPNG(t)).Label("c").Build()
	for label, want := range map[string]string{"a": "Figure 1", "b": "Figure 2", "c": "Figure 3"} {
		got := doc.Ref(label).Text()
		if got != want {
			t.Errorf("Ref(%q) = %q, want %q", label, got, want)
		}
	}
}

func TestRef_UnknownLabelFlagsMissingReference(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	r := doc.Ref("missing")
	if !strings.Contains(r.Text(), "missing") {
		t.Errorf("missing-label visible text should mention the label, got %q", r.Text())
	}
	if r.Link() != "" {
		t.Errorf("missing label should not produce a link, got %q", r.Link())
	}
}

func TestRefPage_EmitsPlaceholderForKnownLabel(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image(crossrefPNG(t)).Label("fig-x").Build()
	r := doc.RefPage("fig-x")
	want := kardec.RefPagePlaceholder + "fig-x}}"
	if r.Text() != want {
		t.Errorf("RefPage placeholder = %q, want %q", r.Text(), want)
	}
}

func TestRefPage_UnknownLabelEmitsQuestionMark(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	if got := doc.RefPage("unknown").Text(); got != "?" {
		t.Errorf("RefPage(unknown) = %q, want %q", got, "?")
	}
}
