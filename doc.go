// Package kardec produces document-like PDFs through a fluent, style-driven DSL.
//
// Kardec targets the gap between report-grid PDF generators (such as Maroto)
// and container-based DOCX-to-PDF converters (such as Gotenberg). It runs
// pure Go, ships embedded fonts, and does not require LibreOffice, a system
// font directory, or a container at runtime.
//
// # Quick start
//
//	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
//	doc.Heading(1, "Monthly Report")
//	doc.Paragraph(kardec.Text("Sales grew "), kardec.Bold("12%"), kardec.Text(" this quarter."))
//	if err := doc.Render("report.pdf"); err != nil {
//	    log.Fatal(err)
//	}
//
// # Concurrency
//
// A *Document is not safe for concurrent use by multiple goroutines, in line
// with bytes.Buffer and strings.Builder. Different *Document values may be
// used concurrently.
//
// See docs/RFC-001-dsl.md in the repository for the full design specification.
package kardec
