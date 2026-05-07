package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// loadTestFont returns a TrueType byte slice usable as an EmbeddedFont
// payload, or skips the test when no candidate is available on the host.
// The CI runs Linux without bundled fonts; locally on Windows we use
// Arial. The test corpus deliberately stays out of the repo to avoid
// shipping non-Kardec font binaries.
func loadTestFont(t *testing.T) []byte {
	t.Helper()
	candidates := []string{
		`C:\Windows\Fonts\Arial.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil && len(data) > 0 {
			return data
		}
	}
	t.Skip("no TrueType font found on host; skipping font-dependent test")
	return nil
}

func TestWriteEmptyDocumentHasValidStructure(t *testing.T) {
	// A document with one blank page and no fonts must still produce a
	// well-formed PDF: header, catalog, pages, page, MediaBox, xref, EOF.
	doc := Document{
		Title:  "blank",
		Author: "Kardec test suite",
		Pages: []Page{{
			Width:  595, // A4 width in points
			Height: 842,
		}},
	}

	var buf bytes.Buffer
	if err := (Writer{}).Write(&buf, doc); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	out := buf.Bytes()
	mustContain(t, out, "%PDF-1.7", "/Catalog", "/Pages", "/Type /Page", "/MediaBox", "%%EOF")
	if !bytes.HasPrefix(out, []byte("%PDF-1.7")) {
		t.Errorf("output should start with %%PDF-1.7, got %q", firstLine(out))
	}
	if !bytes.HasSuffix(bytes.TrimRight(out, "\r\n"), []byte("%%EOF")) {
		t.Errorf("output should end with %%%%EOF; tail = %q", lastLine(out))
	}
}

func TestWriteSingleTextItem(t *testing.T) {
	ttf := loadTestFont(t)
	doc := Document{
		Title:  "hello",
		Author: "Kardec",
		Fonts:  []EmbeddedFont{{Name: "TestFont", TTFData: ttf}},
		Pages: []Page{{
			Width:  595,
			Height: 842,
			Items: []TextItem{{
				X: 72, Y: 770,
				Text: "Hello, Kardec.", FontID: 0, FontSize: 14,
				Color: Color{R: 40, G: 40, B: 40},
			}},
		}},
	}

	var buf bytes.Buffer
	if err := (Writer{}).Write(&buf, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.Bytes()
	mustContain(t, out,
		"%PDF-1.7",
		"/Type /Catalog",
		"/Type /Pages",
		"/Type /Page",
		"/Type /Font",
		"/Type /FontDescriptor",
		"/Subtype /TrueType",
		"/Encoding /WinAnsiEncoding",
		"/MediaBox",
		"BT", "ET", "Tj", "Tf",
		"%%EOF",
	)

	// xref offsets must point at the start of "<id> 0 obj" entries inside
	// the body.
	verifyXrefOffsets(t, out)
}

func TestRoundTripStructuralMarkers(t *testing.T) {
	doc := Document{
		Pages: []Page{{Width: 612, Height: 792}}, // US Letter
	}
	var buf bytes.Buffer
	if err := (Writer{}).Write(&buf, doc); err != nil {
		t.Fatal(err)
	}
	bs := buf.Bytes()
	for _, marker := range []string{"%PDF-1.7", "xref", "trailer", "startxref", "%%EOF"} {
		if !bytes.Contains(bs, []byte(marker)) {
			t.Errorf("output missing %q", marker)
		}
	}
}

func TestPdftotextSmoke(t *testing.T) {
	// Optional: if pdftotext is on PATH, write a real PDF and verify the
	// text we asked for actually round-trips through a real PDF parser.
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not available; skipping text round-trip smoke test")
	}
	ttf := loadTestFont(t)

	const wantText = "Kardec writer smoke test"
	doc := Document{
		Title: "smoke",
		Fonts: []EmbeddedFont{{Name: "Smoke", TTFData: ttf}},
		Pages: []Page{{
			Width:  595,
			Height: 842,
			Items: []TextItem{{
				X: 72, Y: 770, Text: wantText, FontID: 0, FontSize: 12,
				Color: Color{R: 0, G: 0, B: 0},
			}},
		}},
	}
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "smoke.pdf")
	f, err := os.Create(pdfPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := (Writer{}).Write(f, doc); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	out, err := exec.Command("pdftotext", pdfPath, "-").Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			t.Fatalf("pdftotext failed: %v\n%s", err, ee.Stderr)
		}
		t.Fatalf("pdftotext: %v", err)
	}
	if !strings.Contains(string(out), wantText) {
		t.Errorf("pdftotext output %q does not contain %q", string(out), wantText)
	}
}

// verifyXrefOffsets parses the cross-reference table and checks that every
// "in use" entry's offset points at a "<id> 0 obj" header in the body.
// This is a structural check — it does not validate object syntax.
func verifyXrefOffsets(t *testing.T, pdf []byte) {
	t.Helper()
	xrefIdx := bytes.LastIndex(pdf, []byte("\nxref\n"))
	if xrefIdx < 0 {
		t.Fatal("xref keyword not found")
	}
	xrefIdx++ // skip the leading newline
	tail := pdf[xrefIdx:]

	// xref header: "xref\n0 <count>\n"
	scanner := bytes.NewBuffer(tail)
	if line, _ := scanner.ReadString('\n'); strings.TrimSpace(line) != "xref" {
		t.Fatalf("expected 'xref', got %q", line)
	}
	rangeLine, _ := scanner.ReadString('\n')
	parts := strings.Fields(rangeLine)
	if len(parts) != 2 || parts[0] != "0" {
		t.Fatalf("expected '0 <count>' subsection header, got %q", rangeLine)
	}
	count, err := strconv.Atoi(parts[1])
	if err != nil || count < 1 {
		t.Fatalf("invalid xref count: %q", rangeLine)
	}

	entryRE := regexp.MustCompile(`^(\d{10}) \d{5} (n|f) $`)
	for i := 0; i < count; i++ {
		entry := make([]byte, 20)
		n, _ := scanner.Read(entry)
		if n != 20 {
			t.Fatalf("xref entry %d truncated: read %d bytes", i, n)
		}
		match := entryRE.FindStringSubmatch(strings.TrimRight(string(entry), "\n") + " ")
		// We loosen the regex by appending a space because the real entries
		// end with "\n"; the regex above expects a trailing space + end.
		if match == nil {
			// Fallback: accept exact 20-byte entries with " \n" terminator.
			if !regexp.MustCompile(`^\d{10} \d{5} [nf] \n$`).Match(entry) {
				t.Fatalf("xref entry %d malformed: %q", i, entry)
			}
		}
		off, _ := strconv.ParseInt(strings.TrimLeft(string(entry[0:10]), "0"+" "), 10, 64)
		flag := entry[17]
		if flag == 'f' {
			continue // free entry; offset is the next-free pointer, not a body location
		}
		if off <= 0 || off >= int64(len(pdf)) {
			t.Errorf("xref entry %d offset %d out of file bounds (len=%d)", i, off, len(pdf))
			continue
		}
		header := pdf[off : min(off+24, int64(len(pdf)))]
		want := fmt.Sprintf("%d 0 obj", i)
		if !bytes.HasPrefix(header, []byte(want)) {
			t.Errorf("xref entry %d points at %q, expected %q", i, string(header), want)
		}
	}
}

func mustContain(t *testing.T, b []byte, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !bytes.Contains(b, []byte(n)) {
			t.Errorf("output missing required marker %q", n)
		}
	}
}

func firstLine(b []byte) string {
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

func lastLine(b []byte) string {
	b = bytes.TrimRight(b, "\r\n")
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		return string(b[i+1:])
	}
	return string(b)
}
