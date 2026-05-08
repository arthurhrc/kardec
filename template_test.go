package kardec

import (
	"strings"
	"testing"
)

func TestNewTemplateParsesValidSource(t *testing.T) {
	if _, err := NewTemplate("# {{.Title}}"); err != nil {
		t.Fatalf("NewTemplate: %v", err)
	}
}

func TestNewTemplateRejectsSyntaxError(t *testing.T) {
	if _, err := NewTemplate("{{.MissingEnd"); err == nil {
		t.Error("NewTemplate should reject unterminated action")
	}
}

func TestTemplateRenderSubstitutes(t *testing.T) {
	tpl := MustNewTemplate(`# Invoice {{.ID}}

Total: R$ {{.Total}}`)
	doc, err := tpl.Render(struct {
		ID    string
		Total int
	}{"A-1234", 1500})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if err := doc.Err(); err != nil {
		t.Fatalf("doc.Err: %v", err)
	}
	blocks := doc.Sections()[0].Blocks
	if len(blocks) < 2 {
		t.Fatalf("want >=2 blocks, got %d", len(blocks))
	}
	heading := blocks[0].(Heading)
	if got := strings.Join(runTexts(heading.Runs()), ""); got != "Invoice A-1234" {
		t.Errorf("heading = %q, want %q", got, "Invoice A-1234")
	}
	body := blocks[1].(Paragraph)
	if got := strings.Join(runTexts(body.Runs()), ""); got != "Total: R$ 1500" {
		t.Errorf("body = %q, want %q", got, "Total: R$ 1500")
	}
}

func TestTemplateRenderRangeOverSlice(t *testing.T) {
	tpl := MustNewTemplate(`# Members
{{range .Names}}
- {{.}}
{{end}}`)
	doc, err := tpl.Render(struct{ Names []string }{[]string{"Alpha", "Beta", "Gamma"}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	blocks := doc.Sections()[0].Blocks
	// Heading + 3 list-item paragraphs.
	if len(blocks) != 4 {
		t.Fatalf("want 4 blocks, got %d", len(blocks))
	}
	for i, expected := range []string{"Alpha", "Beta", "Gamma"} {
		runs := blocks[i+1].(Paragraph).Runs()
		text := strings.Join(runTexts(runs), "")
		if !strings.Contains(text, expected) {
			t.Errorf("item %d (%q) does not contain %q", i, text, expected)
		}
	}
}

func TestTemplateRenderReportsExecutionError(t *testing.T) {
	tpl := MustNewTemplate(`{{.Missing.Field}}`)
	if _, err := tpl.Render(struct{}{}); err == nil {
		t.Error("Render should surface execution errors from missing fields")
	}
}

func TestTemplateOptionsApplyToProducedDocument(t *testing.T) {
	tpl := MustNewTemplate("# Title", WithPageSize(PageLetter), WithMargins(MarginsNarrow))
	doc, err := tpl.Render(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := doc.Sections()[0].Setup.Size.Name; got != "Letter" {
		t.Errorf("page size = %q, want Letter", got)
	}
	if got := doc.Sections()[0].Setup.Margins.Top; got != MarginsNarrow.Top {
		t.Errorf("margins.Top = %v, want %v", got, MarginsNarrow.Top)
	}
}

// runTexts returns the textual payload of each run, preserving order.
func runTexts(runs []Run) []string {
	out := make([]string, len(runs))
	for i, r := range runs {
		out[i] = r.Text()
	}
	return out
}
