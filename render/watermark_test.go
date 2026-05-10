package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestWatermarkEmitsTextAndRotation(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetWatermark("DRAFT").
		Paragraph(kardec.Text("body")).Document

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	s := string(out)
	// Watermark text is hex glyph IDs (Identity-H) post-v0.22, so
	// instead of grep'ing for "(DRAFT) Tj" we verify the structural
	// markers that distinguish a watermark page from an unmarked
	// one: rotation cm + ExtGState alpha entry + ca opacity value.
	for _, want := range []string{
		" cm\n",        // CTM operator (rotation matrix)
		"/ExtGState",   // page resources include the alpha entry
		"/ca 0.3000",   // configured 30 % opacity (rendered with %.4f)
	} {
		if !strings.Contains(s, want) {
			t.Errorf("watermark marker %q missing", want)
		}
	}
}

func TestWatermarkOpaqueOmitsExtGState(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetWatermark("FINAL", kardec.WatermarkOptions{Opacity: 1.0}).
		Paragraph(kardec.Text("body")).Document

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if bytes.Contains(out, []byte("/ExtGState")) {
		t.Errorf("opaque watermark should not declare /ExtGState")
	}
	// Opaque watermark still emits the rotation cm and a Tj op.
	// The body paragraph contributes 1 Tj; the watermark adds 1
	// more, so a properly rendered doc has ≥ 2 Tj ops.
	tjCount := bytes.Count(out, []byte(" Tj"))
	if tjCount < 2 {
		t.Errorf("expected ≥ 2 Tj ops (body + watermark), got %d", tjCount)
	}
}

func TestWatermarkAccessorRoundtrip(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetWatermark("CONFIDENTIAL")
	got, ok := doc.Watermark()
	if !ok {
		t.Fatalf("Watermark() should report enabled after SetWatermark")
	}
	if got != "CONFIDENTIAL" {
		t.Errorf("watermark text roundtrip: got %q", got)
	}

	doc.SetWatermark("")
	if _, ok := doc.Watermark(); ok {
		t.Errorf("SetWatermark(\"\") should clear the watermark")
	}
}

func TestWatermarkEmittedOnEveryPage(t *testing.T) {
	// Force a multi-page document via a tall paragraph chain.
	docB := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetWatermark("DRAFT")
	for i := 0; i < 80; i++ {
		docB = docB.Paragraph(kardec.Text("filler line for pagination")).Document
	}
	out, err := render.Bytes(docB)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	// Multi-page content streams compress through FlateDecode so
	// the literal "(DRAFT) Tj" disappears from the byte stream.
	// Use the page-level /ExtGState entry instead — one per page,
	// uncompressed.
	hits := strings.Count(string(out), "/ExtGState")
	if hits < 2 {
		t.Errorf("expected watermark resource on every page (≥2 /ExtGState entries), got %d", hits)
	}
}

func TestUntaggedDocumentHasNoWatermarkObjects(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("plain")).Document
	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	for _, leak := range []string{
		"/ExtGState",
		"/ca ",
	} {
		if bytes.Contains(out, []byte(leak)) {
			t.Errorf("unwatermarked output should not contain %q", leak)
		}
	}
}
