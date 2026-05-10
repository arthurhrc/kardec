package kardec

import "testing"

func TestNewSectionAppendsAndSwitchesCurrent(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		Paragraph(Text("page 1")).
		NewSection(SetupOf(PageA5, MarginsNarrow)).
		Paragraph(Text("section 2"))

	secs := doc.Sections()
	if len(secs) != 2 {
		t.Fatalf("want 2 sections, got %d", len(secs))
	}
	if secs[0].Setup.Size.Name != "A4" {
		t.Errorf("section 0 size = %q, want A4", secs[0].Setup.Size.Name)
	}
	if secs[1].Setup.Size.Name != "A5" {
		t.Errorf("section 1 size = %q, want A5", secs[1].Setup.Size.Name)
	}
	if doc.CurrentSection() != secs[1] {
		t.Errorf("CurrentSection should point at the new section after NewSection")
	}
	if len(secs[0].Blocks) != 1 {
		t.Errorf("section 0 should keep its single block, got %d", len(secs[0].Blocks))
	}
	if len(secs[1].Blocks) != 1 {
		t.Errorf("section 1 should hold the post-NewSection paragraph, got %d", len(secs[1].Blocks))
	}
}

func TestNewSectionAppliesSetupVerbatim(t *testing.T) {
	// The v0.10 NewSection takes a PageSetup directly; orientation
	// no longer auto-inherits from the previous section. Callers
	// who want to preserve orientation pass it explicitly via the
	// PageSetup struct.
	first := PageSetup{Size: PageA4, Orientation: Landscape, Margins: MarginsNormal}
	doc := New(PageA4, MarginsNormal)
	doc.sections[0].Setup = first
	doc.cur.Setup = first

	doc.NewSection(PageSetup{Size: PageLetter, Orientation: Landscape, Margins: MarginsNarrow})
	if got := doc.CurrentSection().Setup.Orientation; got != Landscape {
		t.Errorf("NewSection should apply explicit Orientation; got %v, want Landscape", got)
	}
}

func TestNewSectionWithSetupAppliesVerbatim(t *testing.T) {
	custom := PageSetup{
		Size:        PageLegal,
		Orientation: Landscape,
		Margins:     Symmetric(Cm(3)),
	}
	doc := New(PageA4, MarginsNormal).NewSectionWithSetup(custom)
	got := doc.CurrentSection().Setup
	if got.Size.Name != "Legal" || got.Orientation != Landscape {
		t.Errorf("NewSectionWithSetup did not apply setup verbatim: %+v", got)
	}
}

func TestNewSectionPreservesEarlierHeaderFooter(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		Header(Text("first header")).
		NewSection(SetupOf(PageA4, MarginsWide)).
		Header(Text("second header"))

	secs := doc.Sections()
	if got := secs[0].Header[0].Text(); got != "first header" {
		t.Errorf("first section header = %q", got)
	}
	if got := secs[1].Header[0].Text(); got != "second header" {
		t.Errorf("second section header = %q", got)
	}
}

func TestNewSectionInertAfterDeferredError(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.fail(errInternalForSectionTest())
	doc.NewSection(SetupOf(PageA5, MarginsNarrow))
	if len(doc.Sections()) != 1 {
		t.Errorf("NewSection should be inert once an error is captured, got %d sections", len(doc.Sections()))
	}
}
