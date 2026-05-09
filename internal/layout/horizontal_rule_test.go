package layout

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestLayout_HorizontalRule_EmitsRectAcrossContentWidth(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("before")).
		HorizontalRule().
		Paragraph(kardec.Text("after")).Document

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	var rect *PlacedRect
	for _, p := range pages {
		for _, it := range p.Items {
			if it.Rect != nil {
				rect = it.Rect
				break
			}
		}
	}
	if rect == nil {
		t.Fatalf("expected a PlacedRect for the horizontal rule, found none")
	}
	if rect.Thickness.Points() != defaultRuleThicknessPt {
		t.Errorf("rule thickness = %v, want %v", rect.Thickness.Points(), defaultRuleThicknessPt)
	}
	if rect.Color != kardec.ColorGray {
		t.Errorf("rule color = %+v, want gray default", rect.Color)
	}
	if rect.Width.Points() <= 0 {
		t.Errorf("rule width must be positive, got %v", rect.Width.Points())
	}
}

func TestLayout_HorizontalRule_RespectsCustomConfig(t *testing.T) {
	custom := kardec.HorizontalRule{
		Thickness: kardec.Pt(3),
		Color:     kardec.HexColor("#cc0000"),
		Padding:   kardec.Pt(20),
	}
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).HorizontalRule(custom)

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	var rect *PlacedRect
	for _, it := range pages[0].Items {
		if it.Rect != nil {
			rect = it.Rect
		}
	}
	if rect == nil {
		t.Fatalf("expected a PlacedRect")
	}
	if rect.Thickness.Points() != 3 {
		t.Errorf("custom thickness = %v, want 3pt", rect.Thickness.Points())
	}
	if rect.Color != kardec.HexColor("#cc0000") {
		t.Errorf("custom color = %+v, want #cc0000", rect.Color)
	}
}
