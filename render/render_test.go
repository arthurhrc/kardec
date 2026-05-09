package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
)

// containsAll reports whether every needle appears somewhere in haystack.
func containsAll(haystack []byte, needles ...string) (string, bool) {
	for _, n := range needles {
		if !bytes.Contains(haystack, []byte(n)) {
			return n, false
		}
	}
	return "", true
}

func TestBytesProducesValidPDF17(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Hello, Kardec")).
		Paragraph(kardec.Text("Sales grew 12% this quarter."))

	out, err := Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if len(out) < 1000 {
		t.Fatalf("PDF unexpectedly small: %d bytes (font should embed ~150KB)", len(out))
	}
	if !bytes.HasPrefix(out, []byte("%PDF-1.7")) {
		t.Errorf("missing %%PDF-1.7 header, got %q", string(out[:8]))
	}
	if !bytes.HasSuffix(bytes.TrimSpace(out), []byte("%%EOF")) {
		t.Errorf("missing %%EOF trailer")
	}
	if missing, ok := containsAll(out,
		"/Catalog", "/Pages", "/Type /Page", "/Font", "FontFile2", "/MediaBox",
	); !ok {
		t.Errorf("rendered PDF missing structural marker %q", missing)
	}
}

func TestRenderImplWiredOnImport(t *testing.T) {
	// Importing this package's init() must register a real implementation,
	// so Document.Bytes (the method API) succeeds without an explicit call
	// to render.Bytes.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("test"))
	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("Document.Bytes after import _ render: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-1.7")) {
		t.Errorf("Document.Bytes did not produce a valid PDF")
	}
}

func TestRenderPropagatesBuilderError(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	// Force an error via a raw style cycle.
	doc.DefineStyle("loop", kardec.Style{ParentStyle: "loop"})
	doc.ResolveStyle("loop") // populates d.err with the cycle error
	if _, err := Bytes(doc); err == nil {
		t.Error("Bytes should propagate a captured builder error")
	}
}

func TestEmbeddedFontFlows(t *testing.T) {
	// Sanity check that the bundled Liberation Sans Regular ends up in the
	// PDF byte stream — its presence is the difference between visible text
	// and a blank page.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("Quick brown fox."))
	out, err := Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	// The font name appears in the FontDescriptor and BaseFont entries.
	if !strings.Contains(string(out[:min(len(out), 4096)])+string(out[max(0, len(out)-4096):]),
		"LiberationSans") {
		// Search whole stream as fallback (compressed sections may move it)
		if !bytes.Contains(out, []byte("LiberationSans")) {
			t.Errorf("expected LiberationSans reference in PDF byte stream")
		}
	}
}
