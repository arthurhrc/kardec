// Visual-regression tests. The unit suite was happy to ship every
// release through v0.21 with broken text rendering because asserting
// "/Subtype /Type0 is in the byte stream" never inspected the content
// stream's actual draw operators. These tests decompress page content
// streams, parse the relevant operators, and check that:
//
//   - adjacent words have a horizontal advance large enough to NOT
//     overlap (catches the canvas size-unit regression);
//   - SVG draws emit a cm matrix whose scale matches the requested
//     point size, not 60× over (catches the Form XObject regression);
//   - inline math superscripts sit ABOVE the base baseline (catches
//     the emitMathBox convention regression);
//   - inline math base glyph baseline aligns with surrounding text
//     baseline (catches the inline-math anchor regression).
//
// The tests are deliberately tolerant of small numeric noise — they
// assert directional / order-of-magnitude properties, not byte-exact
// positions — so they survive cosmetic spacing tweaks without
// becoming a snapshot graveyard.
package render_test

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

// pdfStreams returns every decompressed content stream in pdfBytes.
// Streams that fail to inflate (raw or different filter) come back
// as their original bytes so the regex pass that follows can still
// inspect them.
func pdfStreams(pdfBytes []byte) []string {
	re := regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	matches := re.FindAllSubmatch(pdfBytes, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		body := m[1]
		zr, err := zlib.NewReader(bytes.NewReader(body))
		if err == nil {
			dec, _ := io.ReadAll(zr)
			zr.Close()
			out = append(out, string(dec))
			continue
		}
		out = append(out, string(body))
	}
	return out
}

// pageStreams returns only the content streams that look like page
// content. Filters by ASCII-printability + presence of `Td` and
// `Tj`; that excludes font glyph data and ToUnicode CMap streams
// which would otherwise pass a naive substring check because the
// raw bytes happen to spell "Tj" / "Td" mid-stream.
func pageStreams(pdfBytes []byte) []string {
	all := pdfStreams(pdfBytes)
	var pages []string
	for _, s := range all {
		// Page content streams contain at least one of: text-show
		// (`Tj`), text-position (`Td`), or XObject invocation
		// (` Do`). Image-only pages have only `Do`; text-only
		// pages have `Tj`+`Td`. Excluding font/CMap streams is
		// done via the ASCII check below.
		hasText := strings.Contains(s, " Td") && strings.Contains(s, " Tj")
		hasDo := strings.Contains(s, " Do")
		if !hasText && !hasDo {
			continue
		}
		if !mostlyASCII(s) {
			continue
		}
		pages = append(pages, s)
	}
	return pages
}

// mostlyASCII checks that >95% of bytes are printable ASCII or
// whitespace. Page content streams are pure ASCII operator soup.
// Font glyph data has lots of high-bit bytes; hex-encoded glyph
// runs (`<0042>`) are still printable.
func mostlyASCII(s string) bool {
	if len(s) == 0 {
		return false
	}
	printable := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 0x20 && c < 0x7F) || c == '\n' || c == '\r' || c == '\t' {
			printable++
		}
	}
	return float64(printable)/float64(len(s)) > 0.95
}

// snippet returns up to maxLen characters of s, suitable for error
// messages. Larger streams have their middle elided.
func snippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	half := maxLen / 2
	return s[:half] + "\n…[truncated]…\n" + s[len(s)-half:]
}

// textShow holds one decoded Tj operator paired with its preceding
// Td position, Tf size, and Tf font name — the minimum needed to
// reason about word placement and font identity.
type textShow struct {
	X, Y     float64
	SizePt   float64
	Font     string // resource name from Tf (e.g. "F0", "F2")
	Operand  string // raw inside the parens or angle brackets
}

// parseTextShows pulls every (Tf, Td, Tj) triple out of a content
// stream. Lines like `/F0 11.0000 Tf`, `72.0000 769.89 Td`, `(Hello,)
// Tj` are matched in order; the latest Tf and Td set the position +
// size carried by each subsequent Tj.
func parseTextShows(stream string) []textShow {
	var (
		out        []textShow
		curSize    float64
		curFont    string
		curX, curY float64
	)
	tfRE := regexp.MustCompile(`/(F\d+) ([\d.\-]+) Tf`)
	tdRE := regexp.MustCompile(`([\d.\-]+) ([\d.\-]+) Td`)
	tjRE := regexp.MustCompile(`(?:\(([^)]*)\)|<([0-9A-Fa-f]+)>) Tj`)
	for _, ln := range strings.Split(stream, "\n") {
		if m := tfRE.FindStringSubmatch(ln); m != nil {
			curFont = m[1]
			curSize, _ = strconv.ParseFloat(m[2], 64)
			continue
		}
		if m := tdRE.FindStringSubmatch(ln); m != nil {
			curX, _ = strconv.ParseFloat(m[1], 64)
			curY, _ = strconv.ParseFloat(m[2], 64)
			continue
		}
		if m := tjRE.FindStringSubmatch(ln); m != nil {
			operand := m[1]
			if operand == "" {
				operand = "<" + m[2] + ">"
			}
			out = append(out, textShow{
				X: curX, Y: curY, SizePt: curSize, Font: curFont, Operand: operand,
			})
		}
	}
	return out
}

// TestVisualRegression_WordAdvance guards the bug that produced the
// v0.1–v0.21 word-overlap: canvas was being asked for a font in mm
// when it wanted points, so advance widths came out 1/2.83 the
// correct value. "Hello," at 24pt Liberation Sans Bold has an
// advance ≈ 65pt; a regression to the old behaviour would put the
// next word ≈ 25pt away, and the assert below would catch it.
func TestVisualRegression_WordAdvance(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Hello, Kardec"))

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := pageStreams(out)
	if len(streams) == 0 {
		t.Fatalf("no page content streams found")
	}
	shows := parseTextShows(streams[0])
	// Body text emits 2-byte hex glyph IDs (Identity-H) post-v0.22,
	// so identify the heading by font size: the heading is the
	// largest size in the stream, and its first two shows are the
	// "Hello," and "Kardec" runs (split on whitespace by the layout
	// engine).
	var bigSize float64
	for _, s := range shows {
		if s.SizePt > bigSize {
			bigSize = s.SizePt
		}
	}
	if bigSize < 12 {
		t.Fatalf("no large-size shows found (bigSize=%.2f); snippet:\n%s", bigSize, snippet(streams[0], 400))
	}
	var headingLine []textShow
	for _, s := range shows {
		if s.SizePt == bigSize {
			headingLine = append(headingLine, s)
		}
	}
	if len(headingLine) < 2 {
		t.Fatalf("need ≥ 2 heading shows, got %d", len(headingLine))
	}
	advance := headingLine[1].X - headingLine[0].X
	// Liberation Sans Bold's width for "Hello," at 24pt is ≈ 65pt.
	// The pre-fix output had ≈ 25pt. Threshold sits between bands.
	const minAdvance = 50.0
	if advance < minAdvance {
		t.Errorf("advance between first two heading words was %.2fpt at size %.2f, want ≥ %.2fpt — canvas-units regression",
			advance, bigSize, minAdvance)
	}
}

// TestVisualRegression_FormXObjectScale guards the SVG cm matrix
// regression: Form XObjects already declare their /BBox, so the
// page-level cm must divide the requested target size by the BBox
// dimensions. Pre-fix code used the raster-image formula and
// scaled SVG drawings 60× too big, putting them off-page.
func TestVisualRegression_FormXObjectScale(t *testing.T) {
	const sampleSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="60" height="60" viewBox="0 0 60 60">
  <rect x="5" y="5" width="50" height="50" fill="black" />
</svg>`
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image([]byte(sampleSVG)).Width(kardec.Pt(60)).Build()

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := pageStreams(out)
	// Look for the cm operator preceding `/Im0 Do` and parse its
	// scale factors. Format: `sx 0 0 sy tx ty cm`.
	cmRE := regexp.MustCompile(`([\d.\-]+) 0 0 ([\d.\-]+) [\d.\-]+ [\d.\-]+ cm\s*/Im\d+ Do`)
	var found bool
	for _, s := range streams {
		m := cmRE.FindStringSubmatch(s)
		if m == nil {
			continue
		}
		found = true
		sx, _ := strconv.ParseFloat(m[1], 64)
		sy, _ := strconv.ParseFloat(m[2], 64)
		// 60-pt target on a 60-unit BBox should yield sx, sy ≈ 1.
		// The old (buggy) emitter would yield sx, sy = 60 (the
		// raster-image formula). Reject anything > 5 because that
		// scale puts the drawing wildly off-page.
		if sx > 5 || sy > 5 {
			t.Errorf("SVG cm scale %.2f×%.2f looks like the raster formula — the Form XObject would render off-page",
				sx, sy)
		}
		// And it shouldn't be zero either.
		if sx <= 0 || sy <= 0 {
			t.Errorf("SVG cm scale %.2f×%.2f is non-positive", sx, sy)
		}
	}
	if !found {
		t.Fatalf("no `cm Do` invocation found in any page stream")
	}
}

// TestVisualRegression_InlineMathBaseline guards the inline-math
// vertical alignment fix: the math base glyph's baseline must align
// with the surrounding text's baseline (PlacedItem.Y maps to PDF Td
// baseline in this layout). A regression that re-introduces the
// `+ ln.ascent` term would put the math baseline ~10pt below the
// text on the same line, which this test rejects.
func TestVisualRegression_InlineMathBaseline(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(
			kardec.Text("a"),
			kardec.InlineMath("x"), // single-glyph math, easy to find
			kardec.Text("z"),
		).Document

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := pageStreams(out)
	if len(streams) == 0 {
		t.Fatalf("no page content streams")
	}
	shows := parseTextShows(streams[0])
	// Body text and math both use hex operands (Identity-H), so we
	// distinguish them by font resource. The body fonts are F0/F1
	// (registry-default + bold); the math font is the last font ID
	// added to the document (assigned during render). Group shows
	// by font and pick out the body row(s) and math row.
	fontCounts := map[string]int{}
	for _, s := range shows {
		fontCounts[s.Font]++
	}
	if len(fontCounts) < 2 {
		t.Fatalf("expected ≥ 2 distinct fonts (body + math), got %d: %v", len(fontCounts), fontCounts)
	}
	// Body fonts have many shows (carry the prose); math has few.
	// Identify the math font as the one with the FEWEST shows on
	// the line — the single inline glyph.
	var mathFont string
	minN := 1 << 30
	for f, n := range fontCounts {
		if n < minN {
			minN = n
			mathFont = f
		}
	}
	if mathFont == "" {
		t.Fatalf("could not identify math font in %v", fontCounts)
	}
	var mathShow textShow
	var bodyShows []textShow
	for _, s := range shows {
		if s.Font == mathFont {
			if mathShow.Y == 0 {
				mathShow = s
			}
		} else if s.SizePt > 0 && s.SizePt < 30 {
			// Skip tiny / huge outliers; keep body-size shows.
			bodyShows = append(bodyShows, s)
		}
	}
	if mathShow.Y == 0 || len(bodyShows) == 0 {
		t.Fatalf("missing math (%v) or body (%d shows) glyph", mathShow, len(bodyShows))
	}
	// Tolerance: math baseline within 3pt of surrounding text. A
	// regression to the +ln.ascent formula puts math 10+ pt below
	// the text baseline — well outside this band.
	const tol = 3.0
	delta := mathShow.Y - bodyShows[0].Y
	if delta < -tol || delta > tol {
		t.Errorf("math glyph Y=%.2f is %.2fpt off the body text Y=%.2f — inline-math baseline alignment regressing",
			mathShow.Y, delta, bodyShows[0].Y)
	}
}

// TestVisualRegression_MathSuperscriptAbove guards the emitMathBox
// convention fix: superscripts must render ABOVE the base glyph
// baseline (smaller PDF Y is lower; superscript should have LARGER
// PDF Y). Pre-fix, sub/superscripts collapsed onto the base row.
func TestVisualRegression_MathSuperscriptAbove(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.InlineMath("a^2")).Document

	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := pageStreams(out)
	if len(streams) == 0 {
		t.Fatalf("no page content streams")
	}
	shows := parseTextShows(snippet(streams[0], 600))
	// In `a^2`, two glyphs: the base "a" at sizePt and the sup "2"
	// at 0.7×sizePt. Identify the base by its larger size.
	var base, sup textShow
	var haveBase, haveSup bool
	for _, s := range shows {
		if !strings.HasPrefix(s.Operand, "<") {
			continue
		}
		if !haveBase || s.SizePt > base.SizePt {
			if haveBase {
				sup = base
				haveSup = true
			}
			base = s
			haveBase = true
			continue
		}
		if !haveSup {
			sup = s
			haveSup = true
		}
	}
	if !haveBase || !haveSup {
		t.Fatalf("did not find both base and sup glyphs in a^2\n%s", snippet(streams[0], 600))
	}
	// In PDF user space Y grows up; sup's baseline must be higher,
	// so sup.Y > base.Y. Demand at least 2pt of clearance to reject
	// the "collapsed-on-baseline" regression where the diff was
	// fractions of a point.
	const minRise = 2.0
	if sup.Y-base.Y < minRise {
		t.Errorf("superscript Y=%.2f is only %.2fpt above base Y=%.2f — sub/sup placement is regressing back to the emitMathBox bug",
			sup.Y, sup.Y-base.Y, base.Y)
	}
}
