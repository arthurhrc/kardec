package layout

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestLayout_Leader_PlacesLeftRightAndDotFill(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Leader([]kardec.Run{kardec.Text("Skill")}, []kardec.Run{kardec.Text("80%")})
	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	var leftItem, rightItem *PlacedItem
	dotCount := 0
	for i := range pages[0].Items {
		it := &pages[0].Items[i]
		switch it.Text {
		case "Skill":
			leftItem = it
		case "80%":
			rightItem = it
		}
		if it.Rect != nil {
			dotCount++
		}
	}
	if leftItem == nil || rightItem == nil {
		t.Fatalf("expected both sides placed; left=%v right=%v", leftItem, rightItem)
	}
	if leftItem.X.Points() >= rightItem.X.Points() {
		t.Errorf("left X=%v should be less than right X=%v",
			leftItem.X.Points(), rightItem.X.Points())
	}
	if dotCount == 0 {
		t.Errorf("expected at least one dot rect filling the gap")
	}
}
