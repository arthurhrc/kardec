package render_test

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestTaggedEmitsStructureMarkers(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTagged("en").
		Heading(1, kardec.Text("Title")).
		Paragraph(kardec.Text("body text"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	for _, want := range []string{
		"/MarkInfo << /Marked true >>", // tagged opt-in
		"/StructTreeRoot ",             // catalog reference
		"/Lang (en)",                   // catalog language
		"/Type /StructTreeRoot",        // root object
		"/Type /StructElem",            // per-block element
		"/S /H1",                       // heading role
		"/S /P",                        // paragraph role
		"/ParentTree ",                 // number tree pointer
		"/StructParents 0",             // page parent-tree key
		"/Tabs /S",                     // logical tab order
		"/H1 << /MCID 0 >> BDC",        // heading marked-content open
		"/P << /MCID 1 >> BDC",         // paragraph marked-content open
		"EMC",                          // marked-content close
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("tagged marker %q missing from PDF byte stream", want)
		}
	}
}

func TestUntaggedHasNoStructureMarkers(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("plain"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	for _, leak := range []string{
		"/StructTreeRoot",
		"/MarkInfo",
		"/StructElem",
		"/Lang ",
		" BDC",
		"EMC",
	} {
		if bytes.Contains(out, []byte(leak)) {
			t.Errorf("untagged output should not contain %q", leak)
		}
	}
}

func TestTaggedAccessorRoundtrip(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTagged("pt-BR")
	lang, ok := doc.Tagged()
	if !ok {
		t.Fatalf("Tagged() should report enabled after SetTagged")
	}
	if lang != "pt-BR" {
		t.Errorf("language roundtrip lost: got %q", lang)
	}

	doc.DisableTagging()
	if _, ok := doc.Tagged(); ok {
		t.Errorf("DisableTagging should clear the tagged flag")
	}
}
