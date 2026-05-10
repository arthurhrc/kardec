package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestParagraphReturnsRefThatChainsBackToDocument(t *testing.T) {
	// The unified API: doc.Paragraph(...) returns a *ParagraphRef but
	// embeds *Document so subsequent doc methods continue to chain.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("first")).
		Heading(2, kardec.Text("section")).
		Paragraph(kardec.Text("second"))

	// doc here is *kardec.ParagraphRef; the embedded *Document gives
	// access to the section list.
	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks (paragraph, heading, paragraph), got %d", len(blocks))
	}
	if _, ok := blocks[0].(kardec.Paragraph); !ok {
		t.Errorf("first block should be Paragraph, got %T", blocks[0])
	}
	if _, ok := blocks[1].(kardec.Heading); !ok {
		t.Errorf("second block should be Heading, got %T", blocks[1])
	}
}

func TestParagraphRefAppliesStyleOverrideRetroactively(t *testing.T) {
	custom := kardec.Style{Color: kardec.HexColor("#cc0000")}
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("body")).
		WithStyle(custom)

	p, ok := doc.Sections()[0].Blocks[0].(kardec.Paragraph)
	if !ok {
		t.Fatalf("expected Paragraph, got %T", doc.Sections()[0].Blocks[0])
	}
	resolved := doc.ResolveBlockStyle(p)
	if resolved.Color != custom.Color {
		t.Errorf("WithStyle override lost: resolved color = %+v, want %+v", resolved.Color, custom.Color)
	}
}

func TestParagraphRefAlignAndLineHeightAndJustify(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("center")).Align(kardec.AlignCenter).
		Paragraph(kardec.Text("just")).Justify().LineHeight(1.6)

	blocks := doc.Sections()[0].Blocks
	if blocks[0].(kardec.Paragraph).Alignment() != kardec.AlignCenter {
		t.Errorf("first paragraph should be center-aligned")
	}
	second := blocks[1].(kardec.Paragraph)
	if second.Alignment() != kardec.AlignJustify {
		t.Errorf("second paragraph should be justified, got %v", second.Alignment())
	}
	if second.LineHeight() != 1.6 {
		t.Errorf("LineHeight = %v, want 1.6", second.LineHeight())
	}
}

