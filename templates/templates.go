// Package templates ships ready-made document scaffolds — Invoice,
// Certificate, Report, Contract — that cover the most common
// real-world "I need a PDF for X" cases. Each constructor takes a
// strongly-typed input struct and returns a *kardec.Document ready
// to Render; callers can chain additional blocks before rendering
// when they need to extend the template.
//
// Templates here are intentionally simple. The goal is "5 lines of
// code from input to PDF", not "every conceivable variation
// configurable via options". Callers who outgrow a template copy
// it into their own code and customise from there — the source is
// a few hundred lines of plain kardec API calls.
package templates

import (
	"fmt"
	"strings"
	"time"

	"github.com/arthurhrc/kardec"
)

// InvoiceData drives the Invoice template. Customer / vendor are
// required; everything else is optional and renders only when
// non-empty.
type InvoiceData struct {
	Number       string    // "A-1001", "INV-2025-007"
	IssueDate    time.Time // when the invoice was generated
	DueDate      time.Time // payment deadline (zero = same as IssueDate)
	VendorName   string    // your company
	VendorTaxID  string    // CNPJ / VAT / EIN
	CustomerName string    // billed party
	CustomerCity string    // optional address fragment
	Items        []InvoiceItem
	TaxRate      float64 // e.g. 0.10 for 10 %
	Currency     string  // "R$", "$", "€" — symbol only
	Notes        string  // payment instructions, late-fee policy, etc.
}

// InvoiceItem is one line on the invoice.
type InvoiceItem struct {
	Description string
	Quantity    float64
	UnitPrice   float64
}

// Total returns the line's quantity × unit price.
func (it InvoiceItem) Total() float64 { return it.Quantity * it.UnitPrice }

// Invoice produces a Document for the supplied InvoiceData. The
// returned doc carries no encryption / watermark / tagging by
// default — callers chain those on after construction:
//
//	doc := templates.Invoice(data).SetEncryption(...)
//	doc.Render("invoice.pdf")
func Invoice(data InvoiceData) *kardec.Document {
	cur := data.Currency
	if cur == "" {
		cur = "$"
	}
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("Invoice " + data.Number)

	// Header band.
	doc.Heading(1, kardec.Text("Invoice "+data.Number))
	if !data.IssueDate.IsZero() {
		doc.Paragraph(
			kardec.Bold("Issue date: "),
			kardec.Text(data.IssueDate.Format("2006-01-02")),
		)
	}
	if !data.DueDate.IsZero() {
		doc.Paragraph(
			kardec.Bold("Due date: "),
			kardec.Text(data.DueDate.Format("2006-01-02")),
		)
	}

	// Parties.
	doc.Heading(2, kardec.Text("Parties"))
	if data.VendorName != "" {
		doc.Paragraph(kardec.Bold("From: "), kardec.Text(data.VendorName))
		if data.VendorTaxID != "" {
			doc.Paragraph(kardec.Text("Tax ID: "+data.VendorTaxID))
		}
	}
	if data.CustomerName != "" {
		doc.Paragraph(kardec.Bold("To: "), kardec.Text(data.CustomerName))
		if data.CustomerCity != "" {
			doc.Paragraph(kardec.Text(data.CustomerCity))
		}
	}

	// Items as a table.
	if len(data.Items) > 0 {
		doc.Heading(2, kardec.Text("Items"))
		var subtotal float64
		tb := doc.Table().Columns(
			kardec.Col("Description", kardec.Width(60)),
			kardec.Col("Qty", kardec.Width(15), kardec.WithAlignment(kardec.AlignRight)),
			kardec.Col("Unit", kardec.Width(25), kardec.WithAlignment(kardec.AlignRight)),
			kardec.Col("Total", kardec.Width(25), kardec.WithAlignment(kardec.AlignRight)),
		).RepeatHeader().Borders(kardec.TableBordersHorizontal)
		// Header row.
		tb.Row("Description", "Qty", "Unit", "Total")
		for _, it := range data.Items {
			tb.Row(
				it.Description,
				strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", it.Quantity), "0"), "."),
				fmt.Sprintf("%s %.2f", cur, it.UnitPrice),
				fmt.Sprintf("%s %.2f", cur, it.Total()),
			)
			subtotal += it.Total()
		}
		tb.Build()
		// Totals row underneath, plain paragraphs.
		doc.Paragraph(kardec.Bold("Subtotal: "), kardec.Text(fmt.Sprintf("%s %.2f", cur, subtotal)))
		if data.TaxRate > 0 {
			tax := subtotal * data.TaxRate
			doc.Paragraph(
				kardec.Bold(fmt.Sprintf("Tax (%.0f%%): ", data.TaxRate*100)),
				kardec.Text(fmt.Sprintf("%s %.2f", cur, tax)),
			)
			doc.Paragraph(kardec.Bold("Total: "), kardec.Text(fmt.Sprintf("%s %.2f", cur, subtotal+tax)))
		} else {
			doc.Paragraph(kardec.Bold("Total: "), kardec.Text(fmt.Sprintf("%s %.2f", cur, subtotal)))
		}
	}

	if data.Notes != "" {
		doc.Heading(2, kardec.Text("Notes"))
		doc.Paragraph(kardec.Italic(data.Notes))
	}
	return doc
}

// CertificateData drives the Certificate template — a one-page
// document acknowledging that someone completed something. The
// Awardee name renders large and centered; the Reason runs as a
// shorter centered paragraph below.
type CertificateData struct {
	Title     string    // "Certificate of Completion"
	Awardee   string    // honoree's name
	Reason    string    // "completed the Advanced Go course"
	IssueDate time.Time // optional date stamp
	Signature string    // "Director of Education" or similar
}

// Certificate produces a one-page certificate centered around the
// awardee's name. Pair with SetBackgroundImage for the decorative
// border most certificates carry.
func Certificate(data CertificateData) *kardec.Document {
	title := data.Title
	if title == "" {
		title = "Certificate"
	}
	doc := kardec.New(kardec.PageA4, kardec.MarginsWide).
		SetTitle(title)
	doc.Spacer(kardec.Cm(2))
	doc.Paragraph(kardec.Bold(title)).Align(kardec.AlignCenter)
	doc.Spacer(kardec.Cm(2))
	doc.Paragraph(kardec.Italic("This certifies that")).Align(kardec.AlignCenter)
	doc.Spacer(kardec.Cm(1))
	doc.Paragraph(kardec.Bold(data.Awardee)).Align(kardec.AlignCenter)
	doc.Spacer(kardec.Cm(1))
	if data.Reason != "" {
		doc.Paragraph(kardec.Text(data.Reason)).Align(kardec.AlignCenter)
	}
	doc.Spacer(kardec.Cm(3))
	if !data.IssueDate.IsZero() {
		doc.Paragraph(kardec.Text("Issued " + data.IssueDate.Format("January 2, 2006"))).Align(kardec.AlignCenter)
	}
	if data.Signature != "" {
		doc.Spacer(kardec.Cm(1))
		doc.Paragraph(kardec.Italic(data.Signature)).Align(kardec.AlignCenter)
	}
	return doc
}

// ReportData drives the simple Report template — title page +
// table of contents + body. Sections is the document body; each
// section becomes an H1.
type ReportData struct {
	Title      string
	Subtitle   string
	Author     string
	Date       time.Time
	WithTOC    bool
	Sections   []ReportSection
	Confidence string // "CONFIDENTIAL" / "DRAFT" / "INTERNAL" — watermark text
}

// ReportSection is one top-level chapter in the report. Body is
// rendered as a sequence of paragraphs separated by blank lines
// from the source string — keeps the input ergonomic for
// hand-authored content.
type ReportSection struct {
	Title string
	Body  string
}

// Report produces a multi-page Document with cover + optional TOC
// + body. Confidence becomes the page watermark.
func Report(data ReportData) *kardec.Document {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle(data.Title).
		SetAuthor(data.Author)
	if data.Confidence != "" {
		doc.SetWatermark(data.Confidence)
	}

	// Cover page.
	doc.Spacer(kardec.Cm(6))
	doc.Heading(1, kardec.Text(data.Title))
	if data.Subtitle != "" {
		doc.Paragraph(kardec.Italic(data.Subtitle))
	}
	doc.Spacer(kardec.Cm(2))
	if data.Author != "" {
		doc.Paragraph(kardec.Bold("Author: "), kardec.Text(data.Author))
	}
	if !data.Date.IsZero() {
		doc.Paragraph(kardec.Bold("Date: "), kardec.Text(data.Date.Format("2006-01-02")))
	}
	doc.PageBreak()

	// TOC.
	if data.WithTOC && len(data.Sections) > 0 {
		doc.Heading(1, kardec.Text("Table of contents"))
		doc.TableOfContents(2)
		doc.PageBreak()
	}

	// Body.
	for _, sec := range data.Sections {
		doc.Heading(1, kardec.Text(sec.Title))
		for _, para := range strings.Split(sec.Body, "\n\n") {
			para = strings.TrimSpace(para)
			if para == "" {
				continue
			}
			doc.Paragraph(kardec.Text(para))
		}
	}
	return doc
}
