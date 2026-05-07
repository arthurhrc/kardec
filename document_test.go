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

func TestBytesWithoutRenderImplReturnsSentinel(t *testing.T) {
	// Without the render package imported, Document.Bytes must surface
	// ErrRendererUnregistered so callers learn to wire the orchestrator.
	// The matching positive test lives in render/render_test.go where the
	// init hook is loaded.
	doc := New(PageA4, MarginsNormal).Paragraph(Text("hi"))
	if _, err := doc.Bytes(); !errors.Is(err, ErrRendererUnregistered) {
		t.Errorf("want ErrRendererUnregistered, got %v", err)
	}
}

func TestRenderPropagatesBuilderError(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.fail(errors.New("synthetic"))
	if _, err := doc.Bytes(); err == nil {
		t.Error("Bytes should propagate the captured builder error")
	}
}


func TestErrPropagationStopsFurtherAppends(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.fail(errors.New("synthetic"))
	doc.Paragraph(Text("after failure"))

	if len(doc.cur.Blocks) != 0 {
		t.Errorf("blocks added after fail should be inert, got %d", len(doc.cur.Blocks))
	}
}
