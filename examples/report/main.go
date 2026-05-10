// Report builds a multi-page document that exercises Kardec's style
// system end-to-end: a corporate-blue heading scale, a colored body
// run, page breaks between sections, and a code-style monospace block.
//
// Running this from the repository root produces report.pdf in the
// working directory:
//
//	go run ./examples/report
//
// Open it in Chrome / Acrobat / pdftk to confirm. The output is the
// closest thing v0.1 has to a "real" document — the next milestone
// (v0.2) lights up real bold and italic glyphs in addition to layout.
package main

import (
	"fmt"
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)

	// House style — three named styles overriding the built-ins.
	corporateBlue := kardec.HexColor("#1F4E79")
	mutedGray := kardec.HexColor("#595959")

	doc.DefineStyle(kardec.StyleH1, kardec.Style{
		ParentStyle: kardec.StyleH1,
		Family:      kardec.FontLiberationSans,
		Size:        kardec.Pt(28),
		Weight:      kardec.WeightBold,
		Color:       corporateBlue,
		SpaceBefore: kardec.Pt(0),
		SpaceAfter:  kardec.Pt(12),
	})
	doc.DefineStyle(kardec.StyleH2, kardec.Style{
		ParentStyle: kardec.StyleH2,
		Size:        kardec.Pt(18),
		Weight:      kardec.WeightSemiBold,
		Color:       corporateBlue,
		SpaceBefore: kardec.Pt(16),
		SpaceAfter:  kardec.Pt(6),
	})
	doc.DefineStyle(kardec.StyleDefault, kardec.Style{
		Family:     kardec.FontLiberationSerif,
		Size:       kardec.Pt(11),
		Color:      kardec.ColorBlack,
		LineHeight: 1.4,
		SpaceAfter: kardec.Pt(8),
	})
	doc.DefineStyle(kardec.StyleCaption, kardec.Style{
		ParentStyle: kardec.StyleDefault,
		Size:        kardec.Pt(9),
		Color:       mutedGray,
		Italic:      true,
	})

	// Page chrome — repeated on every page.
	doc.Header(kardec.Text("Kardec Quarterly Report — confidential"))
	doc.Footer(kardec.Text("Page {{page}} of {{totalPages}} · generated {{date}}"))

	// Cover.
	doc.Heading(1, kardec.Text("Kardec Quarterly Report"))
	doc.AddParagraph(kardec.Text("Q4 2025 — Generated end-to-end by Kardec v0.1.0")).
		WithNamedStyle(kardec.StyleCaption).
		Done()
	doc.PageBreak()

	// Section 1.
	doc.Heading(2, kardec.Text("1 — Executive summary"))
	doc.Paragraph(
		kardec.Text("Kardec turns a fluent Go DSL into a PDF that reads like a document. "),
		kardec.Text("It is style-driven: "),
		kardec.Bold("DefineStyle"),
		kardec.Text(" entries propagate through resolution and are honored by the layout engine, "),
		kardec.Text("which is what produces the corporate blue you see in this section's heading."),
	)
	doc.Paragraph(
		kardec.Text("Compared to grid-first generators, the output here flows: paragraphs justify, headings claim "),
		kardec.Text("vertical breathing room, and overflowing content moves to the next page automatically."),
	)

	// Section 2.
	doc.Heading(2, kardec.Text("2 — Footprint"))
	doc.Paragraph(
		kardec.Text("Kardec ships four bundled font families totaling roughly seven megabytes "),
		kardec.Text("(Liberation Sans, Liberation Serif, Carlito, JetBrains Mono — each in Regular, Bold, "),
		kardec.Text("Italic and BoldItalic). The PDF writer subsets nothing yet, so output files "),
		kardec.Text("are larger than they will be once subsetting lands."),
	)
	doc.Paragraph(kardec.Text("Bundled families and their roles:"))
	doc.Table().
		Columns(
			kardec.Col("Family", kardec.Width(0.35)),
			kardec.Col("Role", kardec.Width(0.45)),
			kardec.Col("Faces", kardec.Width(0.20), kardec.WithAlignment(kardec.AlignRight)),
		).
		RepeatHeader().
		Borders(kardec.TableBordersHorizontal).
		HeaderShading(kardec.HexColor("#E7EFF7")).
		AlternateRowShading(kardec.HexColor("#F8F8F8")).
		Row("Family", "Role", "Faces").
		Row("Liberation Sans", "Default sans (Arial-equivalent)", "4").
		Row("Liberation Serif", "Default serif (Times-equivalent)", "4").
		Row("Carlito", "Calibri-equivalent for Word-style docs", "4").
		Row("JetBrains Mono", "Monospace, code blocks", "4").
		Build()

	// Section 3 — example with named style.
	doc.Heading(2, kardec.Text("3 — Wiring the renderer"))
	doc.Paragraph(kardec.Text("To enable Document.Render, callers blank-import the orchestrator package:"))
	doc.AddParagraph(
		kardec.Text(`import _ "github.com/arthurhrc/kardec/render"`),
	).WithNamedStyle(kardec.StyleCode).Done()
	doc.Paragraph(
		kardec.Text("That single line installs the layout + typography + pdf wiring through an init hook, "),
		kardec.Text("which keeps the kardec package free of an internal/layout import that would otherwise "),
		kardec.Text("introduce a cycle."),
	)

	doc.Spacer(kardec.Pt(20))
	doc.AddParagraph(
		kardec.Text("Generated with Kardec — github.com/arthurhrc/kardec"),
	).WithNamedStyle(kardec.StyleCaption).Align(kardec.AlignCenter).Done()

	if err := doc.Err(); err != nil {
		log.Fatalf("builder error: %v", err)
	}
	if err := doc.Render("report.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Println("rendered report.pdf")
}
