package render_test

import (
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestLineBreakAlgorithmRoundtrip(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetLineBreakAlgorithm(kardec.LineBreakOptimal)
	if got := doc.LineBreakAlgorithm(); got != kardec.LineBreakOptimal {
		t.Errorf("LineBreakAlgorithm roundtrip failed: got %v want LineBreakOptimal", got)
	}
}

// TestOptimalAndGreedyBothProduceValidPDFs verifies that toggling
// the algorithm doesn't break the renderer. Output bytes will
// usually differ in line break positions; this test only asserts
// both runs succeed and produce non-empty PDFs.
func TestOptimalAndGreedyBothProduceValidPDFs(t *testing.T) {
	body := strings.Repeat("the quick brown fox jumps over the lazy dog ", 20)

	greedy, err := render.Bytes(kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text(body)).Document)
	if err != nil {
		t.Fatalf("greedy render: %v", err)
	}
	optimal, err := render.Bytes(kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetLineBreakAlgorithm(kardec.LineBreakOptimal).
		Paragraph(kardec.Text(body)).Document)
	if err != nil {
		t.Fatalf("optimal render: %v", err)
	}
	if len(greedy) == 0 || len(optimal) == 0 {
		t.Fatalf("both outputs must be non-empty (greedy=%d, optimal=%d)", len(greedy), len(optimal))
	}
}
