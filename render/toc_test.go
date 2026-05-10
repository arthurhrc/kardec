package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestRenderTOCEmitsTitlesAndPageNumbers(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		TableOfContents(0).
		PageBreak().
		Heading(1, kardec.Text("Introduction")).
		Paragraph(kardec.Text("body")).
		PageBreak().
		Heading(1, kardec.Text("Methods")).
		Paragraph(kardec.Text("body"))

	out, err := Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := findContentStreams(out)
	// Body text is hex glyph IDs (Identity-H) post-v0.22 so we
	// can't grep for "Introduction" or "Methods" directly. Verify
	// instead that:
	//   1. the TOC tocpage placeholder was replaced (no leakage),
	//   2. the document has enough Tj ops to plausibly carry both
	//      headings AND a TOC entry for each.
	if strings.Contains(string(streams), "{{tocpage:") {
		t.Errorf("unresolved tocpage placeholder leaked into PDF")
	}
	tjCount := bytes.Count(streams, []byte(" Tj"))
	const minWithTOC = 6 // 2 TOC titles + 2 TOC page numbers + 2 body headings
	if tjCount < minWithTOC {
		t.Errorf("expected ≥ %d Tj ops with a 2-entry TOC, got %d", minWithTOC, tjCount)
	}
}

func TestRenderTOCMaxLevelExcludesDeeper(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		TableOfContents(1). // H1 only
		PageBreak().
		Heading(1, kardec.Text("Top")).
		Heading(2, kardec.Text("Sub-topic")).
		Paragraph(kardec.Text("body"))

	out, err := Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := findContentStreams(out)
	tjCount := bytes.Count(streams, []byte(" Tj"))
	// Compare against a doc with maxLevel=2 (which would include
	// "Sub-topic" as a TOC entry too): the maxLevel=1 doc should
	// have FEWER Tj ops because the Sub-topic isn't entered into
	// the TOC. Direct assertion of "Sub-topic absent" is no longer
	// possible since text bytes are hex glyph IDs.
	doc2 := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		TableOfContents(2).
		PageBreak().
		Heading(1, kardec.Text("Top")).
		Heading(2, kardec.Text("Sub-topic")).
		Paragraph(kardec.Text("body"))
	out2, err := Bytes(doc2.Document)
	if err != nil {
		t.Fatalf("Bytes (maxLevel 2): %v", err)
	}
	tj2 := bytes.Count(findContentStreams(out2), []byte(" Tj"))
	if tj2 <= tjCount {
		t.Errorf("expected maxLevel=2 doc to have MORE Tj ops than maxLevel=1 (TOC includes more entries); got %d vs %d", tj2, tjCount)
	}
}
