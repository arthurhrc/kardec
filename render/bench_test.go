package render_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

// BenchmarkRenderHello measures the smallest end-to-end render: one
// heading, one short paragraph, no embedded images. Tracks regressions
// in the per-render fixed cost (font registry init, layout setup, PDF
// header, embedded font copy).
func BenchmarkRenderHello(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			Heading(1, kardec.Text("Hello")).
			Paragraph(kardec.Text("Body."))
		if _, err := render.Bytes(doc.Document); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRender100PageReport measures a multi-page report shape:
// a headed section followed by enough body paragraphs to span ~100
// pages at 12pt body / 1.2 line-height. Tracks regressions in the
// page-break path, line breaker, and PDF object emission cost. Each
// iteration rebuilds the Document from scratch so the benchmark
// captures both build and render time.
func BenchmarkRender100PageReport(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			Heading(1, kardec.Text("Quarterly Report"))
		// ~40 paragraphs per A4 page at body size; 100 pages
		// needs ~4000 paragraphs.
		for p := 0; p < 4000; p++ {
			doc.Paragraph(kardec.Text(
				"This is a body paragraph of moderate length. " +
					"It carries enough characters to wrap across two " +
					"or three lines depending on the column width.",
			))
		}
		if _, err := render.Bytes(doc); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRenderTable100Rows isolates table layout cost: a single
// 100-row table is the canonical "report" body shape and stresses
// the per-row breakLines pass + border/shading emission.
func BenchmarkRenderTable100Rows(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			Table().
			Columns(
				kardec.Col("Region"),
				kardec.Col("Q1", kardec.WithAlignment(kardec.AlignDecimal)),
				kardec.Col("Q2", kardec.WithAlignment(kardec.AlignDecimal)),
			).
			Borders(kardec.TableBordersAll).
			RepeatHeader().
			Row("Region", "Q1", "Q2")
		for r := 0; r < 100; r++ {
			tbl = tbl.Row("Region", "1234.56", "789.01")
		}
		doc := tbl.Build()
		if _, err := render.Bytes(doc); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRenderMarkdownIngest stresses the goldmark bridge plus
// the layout pipeline against a moderately rich CommonMark+GFM
// source: headings, lists, table, link, emphasis. The fixed source
// keeps iteration time deterministic.
func BenchmarkRenderMarkdownIngest(b *testing.B) {
	const src = `# Quarterly Highlights

Sales grew **12%** this quarter; see the [Q1 dashboard](https://example.com/q1)
for the full breakdown.

## Top regions

- North America — *strong*
- EMEA — flat
- APAC — slight decline

| Region | Q1   | Q2   |
|--------|------|------|
| NA     | 1234 | 1357 |
| EMEA   | 800  | 810  |
| APAC   | 510  | 495  |
`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			AppendMarkdown(src)
		if _, err := render.Bytes(doc); err != nil {
			b.Fatal(err)
		}
	}
}
