package render_test

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestRenderTOCTextEmitsClickableLink(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		TableOfContents(2).
		PageBreak().
		Heading(1, kardec.Text("Introduction")).
		Paragraph(kardec.Text("Body of the introduction.")).
		PageBreak().
		Heading(2, kardec.Text("Subsection")).
		Paragraph(kardec.Text("Body of the subsection.")).Document

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	// The catalog's named-destinations table should now contain
	// kardec-toc-introduction and kardec-toc-subsection so the
	// TOC text annotations resolve.
	for _, name := range []string{"kardec-toc-introduction", "kardec-toc-subsection"} {
		// /Dests entries are keyed by Name objects (PDF spec
		// 7.3.5); the writer emits each as `/<name> [pageRef ...]`.
		if !bytes.Contains(out, []byte("/"+name+" [")) {
			t.Errorf("named destination %q missing — TOC link cannot resolve", name)
		}
	}
	// The TOC title tokens should also carry an /A << /Type /Action
	// /S /GoTo /D /kardec-toc-... >> link annotation.
	if !bytes.Contains(out, []byte("/A << /Type /Action /S /GoTo /D /kardec-toc-introduction")) {
		t.Errorf("TOC title link annotation missing for Introduction")
	}
}
