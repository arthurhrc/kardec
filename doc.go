// Package kardec produces document-like PDFs through a fluent, style-driven DSL.
//
// Kardec targets the gap between report-grid PDF generators (such as Maroto)
// and container-based DOCX-to-PDF converters (such as Gotenberg). It runs
// pure Go, ships embedded fonts, and does not require LibreOffice, a system
// font directory, or a container at runtime.
//
// # Quick start
//
//	import (
//	    "github.com/arthurhrc/kardec"
//	    _ "github.com/arthurhrc/kardec/render"
//	)
//
//	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
//	doc.Heading(1, kardec.Text("Monthly Report"))
//	doc.Paragraph(kardec.Text("Sales grew "), kardec.Bold("12%"), kardec.Text(" this quarter."))
//	if err := doc.Render("report.pdf"); err != nil {
//	    log.Fatal(err)
//	}
//
// # Styles
//
// Every block resolves its visual attributes through the named style table
// the document carries. BuiltinStyles seeds the table with Default, H1..H6,
// Caption, Quote, Code, TableHeader, TableCell, Header, Footer, ListItem
// and Link; users add or override entries via Document.DefineStyle. Styles
// inherit from a parent (Style.ParentStyle), and per-block overrides via
// Paragraph(...).WithStyle(...) layer on top during resolution. See
// Document.ResolveStyle and Document.ResolveBlockStyle for the full chain.
//
// # Concurrency
//
// A *Document is not safe for concurrent use by multiple goroutines, in line
// with bytes.Buffer and strings.Builder. Different *Document values may be
// used concurrently.
//
// See docs/RFC-001-dsl.md in the repository for the full design specification.
package kardec
