package layout

import (
	"math"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestTable_DecimalAlignment_AlignsOnDot(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Table().
		Columns(kardec.Col("Region"), kardec.Col("Revenue", kardec.WithAlignment(kardec.AlignDecimal))).
		Row("NA", "1234.56").
		Row("EMEA", "78.9").
		Build()

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}

	// Find the X positions of the integer-part tokens whose text
	// straddles the decimal point. With our stub font (6pt per
	// char at 12pt size), "1234" is 24pt wide and "78" is 12pt
	// wide; their right edges (X + measured-prefix-width) should
	// match — that *is* decimal alignment.
	var firstRowDotX, secondRowDotX float64
	firstRowDotX, secondRowDotX = math.NaN(), math.NaN()
	for _, p := range pages {
		for _, it := range p.Items {
			switch it.Text {
			case "1234.56":
				w, _, _ := it.Font.Measure("1234", it.Size.Points())
				firstRowDotX = it.X.Points() + w
			case "78.9":
				w, _, _ := it.Font.Measure("78", it.Size.Points())
				secondRowDotX = it.X.Points() + w
			}
		}
	}
	if math.IsNaN(firstRowDotX) || math.IsNaN(secondRowDotX) {
		t.Fatalf("decimal-aligned cells not found; first=%v second=%v", firstRowDotX, secondRowDotX)
	}
	if math.Abs(firstRowDotX-secondRowDotX) > 0.5 {
		t.Errorf("decimal pivots should match: 1234.56 ends int at %v, 78.9 ends int at %v",
			firstRowDotX, secondRowDotX)
	}
}

func TestTable_DecimalAlignment_IntegerOnlyFallsBackToRightAlign(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Table().
		Columns(kardec.Col("Region"), kardec.Col("Revenue", kardec.WithAlignment(kardec.AlignDecimal))).
		Row("NA", "100").
		Row("EMEA", "1.5").
		Build()

	pages, err := NewEngine().Layout(doc, stubProvider{})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	var integerRight, dotIntRight float64
	integerRight, dotIntRight = math.NaN(), math.NaN()
	for _, p := range pages {
		for _, it := range p.Items {
			switch it.Text {
			case "100":
				w, _, _ := it.Font.Measure("100", it.Size.Points())
				integerRight = it.X.Points() + w
			case "1.5":
				w, _, _ := it.Font.Measure("1", it.Size.Points())
				dotIntRight = it.X.Points() + w
			}
		}
	}
	if math.IsNaN(integerRight) || math.IsNaN(dotIntRight) {
		t.Fatalf("expected both rows to render; integer=%v dot=%v", integerRight, dotIntRight)
	}
	// "100" with no dot falls back to right-align (whole 100 ends
	// at column right edge); the dotted row's int part ends at the
	// pivot which is at 60% of column width. So integerRight must
	// be > dotIntRight (right edge is past the pivot).
	if integerRight <= dotIntRight {
		t.Errorf("integer-only cell should right-align past the pivot; integerRight=%v pivot=%v",
			integerRight, dotIntRight)
	}
}
