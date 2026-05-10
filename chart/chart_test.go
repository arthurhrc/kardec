package chart

import (
	"bytes"
	"strings"
	"testing"
)

func TestBarRendersExpectedBars(t *testing.T) {
	b := Bar(BarChart{
		Title: "Test",
		Series: []BarSeries{
			{Label: "A", Value: 10},
			{Label: "B", Value: 20},
			{Label: "C", Value: 30},
		},
	})
	out := b.Render(400, 250)
	s := string(out)
	if !strings.HasPrefix(s, `<svg`) {
		t.Errorf("not an SVG: %.100q", s)
	}
	if !bytes.HasSuffix(out, []byte("</svg>")) {
		t.Errorf("missing </svg> at end")
	}
	// One <rect> per bar + one background rect + 0..N for value
	// labels. Count is at least 4 (background + 3 bars).
	if got := strings.Count(s, "<rect"); got < 4 {
		t.Errorf("expected ≥ 4 rects (background + 3 bars), got %d", got)
	}
	// Each series label shows up as an X tick.
	for _, label := range []string{"A", "B", "C"} {
		if !strings.Contains(s, ">"+label+"</text>") {
			t.Errorf("missing X tick %q", label)
		}
	}
}

func TestLineRendersExpectedSeries(t *testing.T) {
	l := Line(LineChart{
		Title: "T",
		Series: []LineSeries{
			{Label: "p50", Points: []Point{{1, 10}, {2, 20}, {3, 15}}},
			{Label: "p99", Points: []Point{{1, 30}, {2, 50}, {3, 40}}},
		},
	})
	out := l.Render(400, 250)
	s := string(out)
	if got := strings.Count(s, "<polyline"); got != 2 {
		t.Errorf("expected 2 polylines (one per series), got %d", got)
	}
	// Legend circles plus point markers — 2 series × (1 legend + 3 points) = 8 circles.
	if got := strings.Count(s, "<circle"); got != 8 {
		t.Errorf("expected 8 circles (legend + point markers), got %d", got)
	}
}

func TestPieRendersExpectedSectors(t *testing.T) {
	p := Pie(PieChart{
		Title: "Share",
		Slices: []PieSlice{
			{Label: "A", Value: 50},
			{Label: "B", Value: 30},
			{Label: "C", Value: 20},
		},
	})
	out := p.Render(400, 250)
	s := string(out)
	if got := strings.Count(s, "<path"); got != 3 {
		t.Errorf("expected 3 sector paths, got %d", got)
	}
	// Legend percentages should sum visually (50% + 30% + 20%).
	for _, want := range []string{"50%", "30%", "20%"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing legend %q", want)
		}
	}
}

func TestBarEmptySeriesProducesValidSVG(t *testing.T) {
	out := Bar(BarChart{}).Render(200, 100)
	if !strings.HasPrefix(string(out), "<svg") || !bytes.HasSuffix(out, []byte("</svg>")) {
		t.Errorf("empty bar should still produce a wrapping SVG")
	}
}

func TestPieZeroTotalProducesValidSVG(t *testing.T) {
	out := Pie(PieChart{Slices: []PieSlice{{Label: "x", Value: 0}}}).Render(200, 100)
	if !strings.HasPrefix(string(out), "<svg") || !bytes.HasSuffix(out, []byte("</svg>")) {
		t.Errorf("zero-total pie should still produce a wrapping SVG")
	}
}
