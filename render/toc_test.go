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
	if !bytes.Contains(streams, []byte("Introduction")) {
		t.Errorf("expected 'Introduction' title in TOC")
	}
	if !bytes.Contains(streams, []byte("Methods")) {
		t.Errorf("expected 'Methods' title in TOC")
	}
	// Placeholders must be patched away.
	if strings.Contains(string(streams), "{{tocpage:") {
		t.Errorf("unresolved tocpage placeholder leaked into PDF")
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
	// Top is H1 — must be in TOC. Sub-topic is H2 and excluded.
	if !bytes.Contains(streams, []byte("Top")) {
		t.Errorf("H1 'Top' should be in TOC")
	}
	// "Sub-topic" still appears in the body heading; assert the
	// dotted-leader sequence doesn't pair it with a TOC page number.
	// Imperfect signal but a useful canary.
	occurrences := bytes.Count(streams, []byte("Sub-topic"))
	if occurrences > 1 {
		t.Errorf("H2 'Sub-topic' should appear once (body heading) when maxLevel=1, got %d occurrences", occurrences)
	}
}
