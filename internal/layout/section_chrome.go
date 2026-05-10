// Package-internal helpers responsible for the recurring page chrome:
// the header above the body and the footer below it. v0.3 keeps both
// to a single line of inline runs; multi-line headers and per-edge
// alignment are queued for v0.4.

package layout

import (
	"strings"
	"time"

	"github.com/arthurhrc/kardec"
)

// emitSectionChrome paints the header and footer of a section's page
// using positions derived from the page setup's margins. The header
// baseline sits at half the top margin; the footer at the page bottom
// minus half the bottom margin. Token substitution runs at emission
// time except for {{totalPages}}, which is patched after layout
// finishes.
func emitSectionChrome(
	cur *pageCursor,
	header, footer []kardec.Run,
	style blockStyle,
	fonts FontProvider,
	pageNumber, sectionNumber int,
) {
	if len(header) > 0 {
		emitChromeRow(cur, header, style, fonts, chromeHeaderY(cur), pageNumber, sectionNumber)
	}
	if len(footer) > 0 {
		emitChromeRow(cur, footer, style, fonts, chromeFooterY(cur), pageNumber, sectionNumber)
	}
}

// emitChromeRow shapes the runs into tokens and emits them along the
// content area's left edge at the supplied baseline Y. Centering /
// right-alignment of header lines is queued for v0.4 once the public
// surface settles around per-edge configuration.
func emitChromeRow(
	cur *pageCursor,
	runs []kardec.Run,
	style blockStyle,
	fonts FontProvider,
	y float64,
	pageNumber, sectionNumber int,
) {
	substituted := substituteRunTokens(runs, pageNumber, sectionNumber)
	tokens := shapeRuns(substituted, fonts, style, kardec.Pt(style.sizePt), style.color, nil)
	x := cur.x0
	for _, t := range tokens {
		if t.isSpace {
			x += t.width
			continue
		}
		cur.items = append(cur.items, PlacedItem{
			X:     kardec.Pt(x),
			Y:     kardec.Pt(y),
			Text:  t.text,
			Font:  t.font,
			Size:  kardec.Pt(t.sizePt),
			Color: t.color,
		})
		x += t.width
	}
}

// chromeHeaderY returns the Y coordinate (top-left origin) at which the
// header baseline should sit. The header rests above cur.y0 (the body's
// top edge) within the top margin band.
func chromeHeaderY(cur *pageCursor) float64 {
	margin := cur.y0 // top margin extent in points
	return margin / 2
}

// chromeFooterY mirrors chromeHeaderY for the bottom margin.
func chromeFooterY(cur *pageCursor) float64 {
	_, pageHeight := pageDimensions(cur.setup)
	margin := pageHeight - cur.y1 // bottom margin extent in points
	return cur.y1 + margin/2
}

// substituteRunTokens walks the runs and replaces {{page}}, {{section}}
// and {{date}} with their resolved values. {{totalPages}} is left in
// place — patched in a post-pass once the page count is final.
func substituteRunTokens(runs []kardec.Run, page, section int) []kardec.Run {
	if len(runs) == 0 {
		return runs
	}
	today := time.Now().UTC().Format("2006-01-02")
	out := make([]kardec.Run, 0, len(runs))
	for _, r := range runs {
		text := r.Text()
		text = strings.ReplaceAll(text, "{{page}}", itoa(page))
		text = strings.ReplaceAll(text, "{{section}}", itoa(section))
		text = strings.ReplaceAll(text, "{{date}}", today)
		out = append(out, copyRunWithText(r, text))
	}
	return out
}

// SubstituteTotalPages walks every PlacedItem in pages and replaces
// any remaining {{totalPages}} marker with the supplied count. Called
// from the public Layout entry once the page slice is final, so
// header / footer placeholders pick up the correct grand total.
//
// Exported because the renderer track also calls it after appending a
// final page in the rare case the layout engine does so without a
// trailing flush — keeps the helper available outside the package
// without exposing the per-page emission internals.
func SubstituteTotalPages(pages []Page, total int) {
	marker := "{{totalPages}}"
	value := itoa(total)
	for i := range pages {
		for j := range pages[i].Items {
			if pages[i].Items[j].Text == "" {
				continue
			}
			if strings.Contains(pages[i].Items[j].Text, marker) {
				pages[i].Items[j].Text = strings.ReplaceAll(pages[i].Items[j].Text, marker, value)
			}
		}
	}
}

// copyRunWithText rebuilds a kardec.Run preserving its bold/italic
// flags but swapping the text payload. Without an exported setter on
// Run, the layout track defers to the kardec.Text/Bold/Italic helpers
// re-emitted with the substituted string.
func copyRunWithText(r kardec.Run, text string) kardec.Run {
	switch {
	case r.Bold() && r.Italic():
		return kardec.BoldItalic(text)
	case r.Bold():
		return kardec.Bold(text)
	case r.Italic():
		return kardec.Italic(text)
	default:
		return kardec.Text(text)
	}
}

// itoa converts a non-negative int to its decimal string form without
// pulling in strconv (keeps the chrome helper free of an extra import
// for what is otherwise a local detail).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := make([]byte, 0, 4)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
