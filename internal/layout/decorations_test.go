package layout

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestLayout_Underline_EmitsRectBelowBaseline(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Underline("done"))

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	var textItem, rectItem *PlacedItem
	for i := range pages[0].Items {
		it := &pages[0].Items[i]
		switch {
		case it.Text == "done":
			textItem = it
		case it.Rect != nil:
			rectItem = it
		}
	}
	if textItem == nil {
		t.Fatalf("text item missing")
	}
	if rectItem == nil {
		t.Fatalf("underline rect missing")
	}
	if rectItem.Y.Points() <= textItem.Y.Points() {
		t.Errorf("underline Y=%v must be below text baseline Y=%v",
			rectItem.Y.Points(), textItem.Y.Points())
	}
	if rectItem.Rect.Width.Points() <= 0 {
		t.Errorf("underline width must be positive, got %v", rectItem.Rect.Width.Points())
	}
}

func TestLayout_Strikethrough_EmitsRectAboveBaseline(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Strikethrough("retracted"))

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	var textItem, rectItem *PlacedItem
	for i := range pages[0].Items {
		it := &pages[0].Items[i]
		switch {
		case it.Text == "retracted":
			textItem = it
		case it.Rect != nil:
			rectItem = it
		}
	}
	if textItem == nil {
		t.Fatalf("text item missing")
	}
	if rectItem == nil {
		t.Fatalf("strikethrough rect missing")
	}
	if rectItem.Y.Points() >= textItem.Y.Points() {
		t.Errorf("strikethrough Y=%v must be above baseline Y=%v",
			rectItem.Y.Points(), textItem.Y.Points())
	}
}

func TestLayout_PlainRun_EmitsNoDecorationRect(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("plain"))

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	for _, it := range pages[0].Items {
		if it.Rect != nil {
			t.Errorf("plain text should not produce decoration rects, got %+v", it.Rect)
		}
	}
}
