package kardec

import (
	"errors"
	"testing"
)

func TestNewDocumentHasOneSection(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	if len(doc.sections) != 1 {
		t.Fatalf("want 1 section, got %d", len(doc.sections))
	}
	if doc.cur == nil || doc.cur != doc.sections[0] {
		t.Fatal("cur should point at the first section")
	}
	if doc.cur.Setup.Size.Name != "A4" {
		t.Errorf("first section size = %q, want A4", doc.cur.Setup.Size.Name)
	}
	if doc.Err() != nil {
		t.Errorf("fresh document should have no error, got %v", doc.Err())
	}
}

func TestBuilderChainAccumulatesBlocks(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		Heading(1, Text("Title")).
		Paragraph(Text("Body text.")).
		PageBreak().
		Spacer(Pt(12))

	got := doc.cur.Blocks
	if len(got) != 4 {
		t.Fatalf("want 4 blocks, got %d", len(got))
	}
	wantKinds := []blockKind{kindHeading, kindParagraph, kindPageBreak, kindSpacer}
	for i, want := range wantKinds {
		if got[i].blockKind() != want {
			t.Errorf("block %d kind = %v, want %v", i, got[i].blockKind(), want)
		}
	}
}

func TestHeadingClampsLevel(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		Heading(0, Text("too low")).
		Heading(7, Text("too high"))

	h0 := doc.cur.Blocks[0].(Heading)
	h1 := doc.cur.Blocks[1].(Heading)
	if h0.level != 1 {
		t.Errorf("level 0 should clamp to 1, got %d", h0.level)
	}
	if h1.level != 6 {
		t.Errorf("level 7 should clamp to 6, got %d", h1.level)
	}
}

func TestBytesProducesValidPDF17(t *testing.T) {
	// With the renderer track wired, Document.Bytes must produce a real
	// PDF 1.7 stream — even the layout-track stub yields a single blank
	// page that opens in Acrobat / Chrome.
	doc := New(PageA4, MarginsNormal).Paragraph(Text("hi"))
	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if len(out) < 100 {
		t.Fatalf("rendered PDF unexpectedly short: %d bytes", len(out))
	}
	if string(out[:8]) != "%PDF-1.7" {
		t.Errorf("expected %%PDF-1.7 prefix, got %q", string(out[:8]))
	}
	for _, marker := range []string{"/Catalog", "/Pages", "/Type /Page", "%%EOF"} {
		if !bytesContains(out, marker) {
			t.Errorf("rendered PDF missing %q", marker)
		}
	}
}

func TestRenderPropagatesBuilderError(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.fail(errors.New("synthetic"))
	if _, err := doc.Bytes(); err == nil {
		t.Error("Bytes should propagate the captured builder error")
	}
}

func bytesContains(haystack []byte, needle string) bool {
	n := []byte(needle)
	for i := 0; i+len(n) <= len(haystack); i++ {
		match := true
		for j := range n {
			if haystack[i+j] != n[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func TestErrPropagationStopsFurtherAppends(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.fail(errors.New("synthetic"))
	doc.Paragraph(Text("after failure"))

	if len(doc.cur.Blocks) != 0 {
		t.Errorf("blocks added after fail should be inert, got %d", len(doc.cur.Blocks))
	}
}
