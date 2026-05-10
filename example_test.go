package kardec_test

import (
	"github.com/arthurhrc/kardec"
)

// Example shows the canonical "Hello report" pattern: build a Document
// fluently, then call Render to write a PDF. The Document method API
// stays compile-checked here; the runnable byte-stream variant lives
// in the render package's example_test so this file does not need to
// import render and break the package's "renderer-unregistered"
// sentinel test.
func Example() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Monthly Report")).
		Paragraph(
			kardec.Text("Sales grew "),
			kardec.Bold("12%"),
			kardec.Text(" this quarter."),
		)
	_ = doc.Render
}

// ExampleNew builds a minimal one-paragraph document.
func ExampleNew() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("Hello, world."))
	_ = doc // hand off to render.Bytes / render.ToFile in real code
}

// ExampleNewWithSetup configures a two-column section from the first
// page using a fully-populated PageSetup.
func ExampleNewWithSetup() {
	doc := kardec.NewWithSetup(kardec.PageSetup{
		Size:    kardec.PageA4,
		Margins: kardec.MarginsNormal,
		Columns: 2,
	}).Heading(1, kardec.Text("Reference Card"))
	_ = doc
}

// ExampleDocument_Paragraph layers a style override onto the just-
// appended paragraph using the unified ParagraphRef builder.
func ExampleDocument_Paragraph() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("Body text.")).
		WithStyle(kardec.Style{Color: kardec.HexColor("#444")}).
		LineHeight(1.5)
	_ = doc
}

// ExampleDocument_Heading shows the level argument and text-only
// shortcut. Levels outside 1..6 are clamped.
func ExampleDocument_Heading() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Top")).
		Heading(2, kardec.Text("Section")).
		Paragraph(kardec.Text("Body."))
	_ = doc
}

// ExampleDocument_Table builds a small bordered table with a header
// row that is reprinted on every continuation page.
func ExampleDocument_Table() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Table().
		Columns(kardec.Col("Region"), kardec.Col("Revenue", kardec.WithAlignment(kardec.AlignDecimal))).
		Borders(kardec.TableBordersAll).
		RepeatHeader().
		Row("Region", "Revenue").
		Row("NA", "1,234.56").
		Row("EMEA", "78.9").
		Build()
	_ = doc
}

// ExampleDocument_AppendMarkdown ingests CommonMark + GFM into a
// Document. Useful for content that arrives as Markdown from a CMS or
// authored by hand.
func ExampleDocument_AppendMarkdown() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		AppendMarkdown(`# Title

Plain paragraph with *emphasis* and **strong**.

| col | val |
|---|---|
| a | 1 |
| b | 2 |
`)
	_ = doc
}

// ExampleDocument_Cite shows numeric citations with a Bibliography.
// Numbers assign on first reference and reuse on repeats; the
// generated [N] runs are clickable hyperlinks to the matching entry.
func ExampleDocument_Cite() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	knuth := doc.Cite("Knuth1984")
	doc.Paragraph(
		kardec.Text("Following "),
		knuth,
		kardec.Text(", we adopt the literate model."),
	).Bibliography(
		kardec.BibEntry{
			Key:    "Knuth1984",
			Author: "Knuth, D.",
			Title:  "Literate Programming",
			Year:   1984,
		},
	)
	_ = doc
}

// ExampleDocument_Clause uses the hierarchical clause numbering
// helper. Clause(1) advances the top-level counter; Clause(2) opens a
// sub-level (1.1, 1.2, ...). Calling Clause(1) again resets the deeper
// counters.
func ExampleDocument_Clause() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Clause(1, kardec.Text("Definitions")).
		Clause(2, kardec.Text("Confidential Information")).
		Clause(2, kardec.Text("Term")).
		Clause(1, kardec.Text("Obligations"))
	_ = doc
	// The rendered output reads:
	//   1. Definitions
	//   1.1 Confidential Information
	//   1.2 Term
	//   2. Obligations
}

// ExampleDocument_KeepTogether binds a heading to the paragraph that
// follows so the two never split across pages. Use NewParagraph /
// NewHeading to build inner blocks outside the Document chain.
func ExampleDocument_KeepTogether() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		KeepTogether(
			kardec.NewHeading(2, kardec.Text("Section")),
			kardec.NewParagraph(kardec.Text("Body that must accompany the heading.")),
		)
	_ = doc
}

// ExampleDocument_Leader emits a one-line "left ........ right" row
// with a dotted fill between the two sides. Reused for CV skill bars,
// contract signatories, and financial line items.
func ExampleDocument_Leader() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Leader(
			[]kardec.Run{kardec.Text("Senior Engineer")},
			[]kardec.Run{kardec.Bold("Acme Corp")},
		)
	_ = doc
}

// ExampleSignatureBlock produces a contract-style signature line:
// horizontal rule, centered name, optional italic role.
func ExampleSignatureBlock() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("Signed and acknowledged below.")).
		Signature("Jane Doe", "Lead Engineer")
	_ = doc
}

// ExampleDocument_Ref resolves a label set via ImageBuilder.Label
// (or TableBuilder.Label) into the canonical "Figure N" / "Table N"
// reference, plus a hyperlink to the matching anchor.
func ExampleDocument_Ref() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	// Imagine an Image was added with .Label("growth-2024") above.
	// doc.Ref returns a Run with text "Figure N" + an internal
	// hyperlink to the auto-anchor placed before that block.
	ref := doc.Ref("growth-2024")
	doc.Paragraph(
		kardec.Text("As shown in "),
		ref,
		kardec.Text(", growth was strong."),
	)
}
