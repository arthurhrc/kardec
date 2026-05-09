package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestHorizontalRuleAppendsBlock(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("before")).
		HorizontalRule().
		Paragraph(kardec.Text("after"))
	if err := doc.Err(); err != nil {
		t.Fatalf("doc err: %v", err)
	}
	sections := doc.Sections()
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	blocks := sections[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if _, ok := blocks[1].(kardec.HorizontalRule); !ok {
		t.Errorf("expected blocks[1] to be HorizontalRule, got %T", blocks[1])
	}
}

func TestHorizontalRuleAcceptsCustomConfig(t *testing.T) {
	rule := kardec.HorizontalRule{
		Thickness: kardec.Pt(2),
		Color:     kardec.HexColor("#cc0000"),
		Padding:   kardec.Pt(12),
	}
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).HorizontalRule(rule)
	got, ok := doc.Sections()[0].Blocks[0].(kardec.HorizontalRule)
	if !ok {
		t.Fatalf("expected HorizontalRule, got %T", doc.Sections()[0].Blocks[0])
	}
	if got.Thickness != kardec.Pt(2) {
		t.Errorf("Thickness = %v, want 2pt", got.Thickness)
	}
	if got.Padding != kardec.Pt(12) {
		t.Errorf("Padding = %v, want 12pt", got.Padding)
	}
	if got.Color != kardec.HexColor("#cc0000") {
		t.Errorf("Color = %+v, want #cc0000", got.Color)
	}
}
