package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

// TestStrictTableHierarchy guards the v0.24 PDF/UA strict tagging:
// a tagged table must produce /Table > /TR > /TD/TH nesting in
// the structure tree, not flat /TD/TH leaves under the root.
func TestStrictTableHierarchy(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTagged("en")
	doc.Heading(1, kardec.Text("Numbers"))
	doc.Table().Columns(
		kardec.Col("Month"),
		kardec.Col("Revenue"),
	).Row("October", "1200").Row("November", "1500").Build()

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	s := string(out)
	expectations := map[string]int{
		"/S /Table": 1,
		"/S /TR":    2, // header row + body row
		"/S /TH":    2, // 2 header cells
		"/S /TD":    2, // 2 body cells in the second row
	}
	for marker, want := range expectations {
		got := strings.Count(s, marker)
		if got != want {
			t.Errorf("strict-table marker %q count: got %d, want %d", marker, got, want)
		}
	}
}

// TestStrictSectGroupings guards the H1-boundary Sect wrapping:
// every H1 starts a new /Sect that absorbs its following P / H2
// blocks, so screen readers can collapse / expand entire chapters.
func TestStrictSectGroupings(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTagged("en").
		Heading(1, kardec.Text("Chapter A")).
		Paragraph(kardec.Text("body a")).
		Heading(2, kardec.Text("A.1 sub")).
		Paragraph(kardec.Text("sub body")).
		Heading(1, kardec.Text("Chapter B")).
		Paragraph(kardec.Text("body b"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	s := string(out)
	if got := strings.Count(s, "/S /Sect"); got != 2 {
		t.Errorf("expected 2 Sect containers (one per H1), got %d", got)
	}
	if got := strings.Count(s, "/S /H1"); got != 2 {
		t.Errorf("expected 2 H1 leaves, got %d", got)
	}
	if got := strings.Count(s, "/S /H2"); got != 1 {
		t.Errorf("expected 1 H2 leaf, got %d", got)
	}
	if got := strings.Count(s, "/S /P"); got != 3 {
		t.Errorf("expected 3 P leaves, got %d", got)
	}
}

// TestStrictUntaggedDocumentEmitsNoStructure mirrors the older
// untagged-output test but ensures the new Sect / Table elements
// don't leak into untagged output.
func TestStrictUntaggedDocumentEmitsNoStructure(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("plain")).Document
	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	for _, leak := range []string{
		"/S /Table",
		"/S /TR",
		"/S /TH",
		"/S /TD",
		"/S /Sect",
	} {
		if bytes.Contains(out, []byte(leak)) {
			t.Errorf("untagged output should not carry %q", leak)
		}
	}
}
