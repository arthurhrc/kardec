package kardec_test

import (
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestCiteAssignsSequentialNumbers(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	if got := doc.Cite("A").Text(); got != "[1]" {
		t.Errorf("first cite = %q, want [1]", got)
	}
	if got := doc.Cite("B").Text(); got != "[2]" {
		t.Errorf("second cite = %q, want [2]", got)
	}
	if got := doc.Cite("A").Text(); got != "[1]" {
		t.Errorf("re-cited key should reuse number, got %q", got)
	}
}

func TestCiteCarriesAnchorLink(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	r := doc.Cite("Knuth1984")
	want := "#" + kardec.BibAnchorName(1)
	if r.Link() != want {
		t.Errorf("Cite link = %q, want %q", r.Link(), want)
	}
}

func TestBibliographyEmitsHeadingAndEntriesInCitationOrder(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	b := doc.Cite("B")
	a := doc.Cite("A")
	doc.Paragraph(b, a). // citation order: B, A
		Bibliography(
			kardec.BibEntry{Key: "A", Author: "Adams, A.", Title: "Alpha", Year: 2001},
			kardec.BibEntry{Key: "B", Author: "Brown, B.", Title: "Beta", Year: 2002},
		)

	blocks := doc.Sections()[0].Blocks
	// Expect: Paragraph(citations) + Heading + 2 * (Anchor + Paragraph)
	if len(blocks) < 6 {
		t.Fatalf("expected at least 6 blocks, got %d", len(blocks))
	}
	heading, ok := blocks[1].(kardec.Heading)
	if !ok {
		t.Fatalf("expected Heading at blocks[1], got %T", blocks[1])
	}
	if heading.Runs()[0].Text() != "References" {
		t.Errorf("heading text = %q", heading.Runs()[0].Text())
	}
	// First emitted entry should be B (cited first).
	a1 := blocks[2].(kardec.Anchor)
	if a1.Name() != kardec.BibAnchorName(1) {
		t.Errorf("first anchor = %q, want %q", a1.Name(), kardec.BibAnchorName(1))
	}
	p1 := blocks[3].(kardec.Paragraph)
	body := concatRunText(p1.Runs())
	if !strings.HasPrefix(body, "[1] ") || !strings.Contains(body, "Brown") {
		t.Errorf("first entry body = %q, expected to start with [1] and mention Brown", body)
	}
}

func TestBibliographyAppendsUncitedEntriesAtEnd(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	a := doc.Cite("A")
	doc.Paragraph(a).
		Bibliography(
			kardec.BibEntry{Key: "A", Author: "Adams, A."},
			kardec.BibEntry{Key: "Z", Author: "Zeta, Z."}, // uncited
		)

	blocks := doc.Sections()[0].Blocks
	// Find last paragraph (uncited Z entry).
	var lastPara kardec.Paragraph
	for _, b := range blocks {
		if p, ok := b.(kardec.Paragraph); ok {
			lastPara = p
		}
	}
	body := concatRunText(lastPara.Runs())
	if !strings.HasPrefix(body, "[2] ") || !strings.Contains(body, "Zeta") {
		t.Errorf("uncited entry should be numbered [2] and mention Zeta, got %q", body)
	}
}

func TestCitedKeysReportsOrder(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	doc.Cite("Z")
	doc.Cite("A")
	doc.Cite("Z") // re-cite, no new entry
	keys := doc.CitedKeys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
	}
	if keys[0] != "Z" || keys[1] != "A" {
		t.Errorf("citation order = %v, want [Z A]", keys)
	}
}

func TestBibliographyPlaceholderForCitedButMissingKey(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	g := doc.Cite("Ghost")
	doc.Paragraph(g).
		Bibliography() // no entries supplied — heading still suppressed when zero entries

	blocks := doc.Sections()[0].Blocks
	for _, b := range blocks {
		if h, ok := b.(kardec.Heading); ok {
			if h.Runs()[0].Text() == "References" {
				t.Errorf("Bibliography(no entries) should not emit a References heading")
			}
		}
	}
}

func concatRunText(runs []kardec.Run) string {
	var b strings.Builder
	for _, r := range runs {
		b.WriteString(r.Text())
	}
	return b.String()
}
