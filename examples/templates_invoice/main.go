// Templates demonstrates the ready-made document scaffolds shipped
// under kardec/templates. Five lines of code, one PDF.
//
//	go run ./examples/templates_invoice
//
// Produces invoice-from-template.pdf — same output shape as
// examples/invoice but without the manual Markdown template.
package main

import (
	"log"
	"time"

	_ "github.com/arthurhrc/kardec/render"
	"github.com/arthurhrc/kardec/templates"
)

func main() {
	doc := templates.Invoice(templates.InvoiceData{
		Number:       "INV-2026-001",
		IssueDate:    time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		DueDate:      time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
		VendorName:   "Kardec Tipografia Ltda.",
		VendorTaxID:  "12.345.678/0001-99",
		CustomerName: "Cliente Exemplo S.A.",
		CustomerCity: "São Paulo, SP",
		Items: []templates.InvoiceItem{
			{Description: "Cloud subscription (annual)", Quantity: 1, UnitPrice: 12000.00},
			{Description: "Premium support hours", Quantity: 10, UnitPrice: 320.00},
			{Description: "Onboarding workshop", Quantity: 1, UnitPrice: 2500.00},
		},
		TaxRate:  0.10,
		Currency: "R$",
		Notes:    "Pagamento via PIX para chave: financeiro@example.com",
	})

	if err := doc.Render("invoice-from-template.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered invoice-from-template.pdf")
}
