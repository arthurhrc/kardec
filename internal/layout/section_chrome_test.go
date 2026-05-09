package layout

import (
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
)

// chromeFontProvider returns predictable metrics so chrome tests can
// assert on PlacedItem text without depending on actual font measure.
type chromeFontProvider struct{}

func (chromeFontProvider) Resolve(string, bool, bool) Font { return chromeFontFace{} }

type chromeFontFace struct{}

func (chromeFontFace) Measure(text string, sizePt float64) (float64, float64, float64) {
	return float64(len(text)) * sizePt * 0.5, sizePt * 0.7, sizePt * 0.2
}

func TestLayoutEmitsHeaderTokensSubstituted(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Header(kardec.Text("Page {{page}} of {{totalPages}}")).
		Paragraph(kardec.Text("body"))

	pages, err := Layout(doc.Document, chromeFontProvider{})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(pages))
	}
	header := joinPageText(pages[0])
	if !strings.Contains(header, "Page 1 of 1") {
		t.Errorf("expected substituted 'Page 1 of 1' on first page, items text = %q", header)
	}
}

func TestLayoutEmitsFooterTokensSubstituted(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Footer(kardec.Text("Section {{section}} — {{date}}")).
		Paragraph(kardec.Text("body"))

	pages, err := Layout(doc.Document, chromeFontProvider{})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	text := joinPageText(pages[0])
	if !strings.Contains(text, "Section 1") {
		t.Errorf("expected 'Section 1' in footer, got %q", text)
	}
	// {{date}} — exact value depends on the day; assert the marker
	// itself has been removed from the output.
	if strings.Contains(text, "{{date}}") {
		t.Errorf("{{date}} should be substituted, got %q", text)
	}
}

func TestLayoutTotalPagesAcrossMultiPage(t *testing.T) {
	doc := kardec.New(kardec.PageA5, kardec.MarginsNormal).
		Header(kardec.Text("p {{page}}/{{totalPages}}"))

	// Force several pages by feeding many forced page breaks.
	for i := 0; i < 4; i++ {
		doc.Paragraph(kardec.Text("body"))
		doc.PageBreak()
	}

	pages, err := Layout(doc, chromeFontProvider{})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(pages) < 4 {
		t.Fatalf("expected >= 4 pages, got %d", len(pages))
	}
	last := joinPageText(pages[len(pages)-1])
	if !strings.Contains(last, "p "+itoa(len(pages))+"/"+itoa(len(pages))) {
		t.Errorf("expected last page to read p N/N where N=%d; items = %q", len(pages), last)
	}
}

// joinPageText collects every PlacedItem.Text on a page into a single
// space-separated string for substring assertions.
func joinPageText(p Page) string {
	parts := make([]string, 0, len(p.Items))
	for _, it := range p.Items {
		if it.Text != "" {
			parts = append(parts, it.Text)
		}
	}
	return strings.Join(parts, " ")
}
