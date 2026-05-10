package templates

import (
	"testing"
	"time"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestInvoiceRendersWithoutError(t *testing.T) {
	doc := Invoice(InvoiceData{
		Number:       "INV-001",
		IssueDate:    time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		DueDate:      time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
		VendorName:   "Acme Corp",
		VendorTaxID:  "12.345.678/0001-99",
		CustomerName: "Initech",
		CustomerCity: "São Paulo, SP",
		Items: []InvoiceItem{
			{Description: "Cloud subscription", Quantity: 1, UnitPrice: 1200.00},
			{Description: "Support hours", Quantity: 10, UnitPrice: 80.00},
		},
		TaxRate:  0.10,
		Currency: "R$",
		Notes:    "Pay via PIX or wire transfer.",
	})
	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if len(out) < 1000 {
		t.Errorf("invoice PDF suspiciously small: %d bytes", len(out))
	}
}

func TestCertificateRendersWithoutError(t *testing.T) {
	doc := Certificate(CertificateData{
		Title:     "Certificate of Completion",
		Awardee:   "Arthur Carvalho",
		Reason:    "completed the Advanced Go workshop",
		IssueDate: time.Now(),
		Signature: "Director of Education",
	})
	if _, err := render.Bytes(doc); err != nil {
		t.Fatalf("Bytes: %v", err)
	}
}

func TestReportRendersWithoutError(t *testing.T) {
	doc := Report(ReportData{
		Title:    "Q4 2025 Report",
		Subtitle: "Engineering metrics",
		Author:   "Arthur",
		Date:     time.Now(),
		WithTOC:  true,
		Sections: []ReportSection{
			{Title: "Summary", Body: "Lorem ipsum.\n\nDolor sit amet."},
			{Title: "Details", Body: "More body text here."},
		},
		Confidence: "INTERNAL",
	})
	if _, err := render.Bytes(doc); err != nil {
		t.Fatalf("Bytes: %v", err)
	}
}

// TestInvoiceCustomisable demonstrates the chain-extension idiom:
// take the template's doc, add tagging or encryption, then render.
func TestInvoiceCustomisable(t *testing.T) {
	doc := Invoice(InvoiceData{
		Number:       "INV-002",
		VendorName:   "V",
		CustomerName: "C",
		Items:        []InvoiceItem{{Description: "x", Quantity: 1, UnitPrice: 10}},
		Currency:     "$",
	}).
		SetTagged("en").
		SetEncryption(kardec.EncryptionOptions{
			UserPassword: "open-me",
		})
	if _, err := render.Bytes(doc); err != nil {
		t.Fatalf("Bytes: %v", err)
	}
}
