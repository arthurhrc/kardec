// QRCode demonstrates Kardec's built-in QR encoder. Document.QRCode
// renders a QR code as a vector Form XObject so it stays sharp at
// any size — useful for tickets, vouchers, PIX strings, and any
// other "scan with your phone" workflow.
//
//	go run ./examples/qrcode
//
// Produces qrcode.pdf with the same URL encoded at three error-
// correction tiers + a longer PIX-style payload.
package main

import (
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("QR codes")).
		Paragraph(kardec.Text(
			"The same URL encoded at three error-correction tiers. " +
				"Higher tiers add redundancy: the High code stays scannable " +
				"with up to 30% of its pixels damaged."))

	url := "https://example.com/checkout/abc-123-456"
	for _, level := range []struct {
		name string
		ecl  kardec.QRErrorLevel
	}{
		{"Low (7%)", kardec.QRLow},
		{"Medium (15%)", kardec.QRMedium},
		{"Quartile (25%)", kardec.QRQuart},
		{"High (30%)", kardec.QRHigh},
	} {
		doc.Paragraph(kardec.Bold(level.name + ":"))
		doc.QRCode(url, level.ecl, kardec.Pt(80)).Build()
	}

	doc.Heading(2, kardec.Text("Longer payload — PIX-style"))
	pixPayload := "00020126360014BR.GOV.BCB.PIX0114+5511999999999520400005303986540510.00" +
		"5802BR5913Joao da Silva6008Sao Paulo62070503***6304ABCD"
	doc.QRCode(pixPayload, kardec.QRMedium, kardec.Pt(120)).Build()

	if err := doc.Render("qrcode.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered qrcode.pdf")
}
