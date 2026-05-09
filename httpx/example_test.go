package httpx_test

import (
	"net/http"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/httpx"
	_ "github.com/arthurhrc/kardec/render"
)

// ExampleWriteResponse is the canonical "PDF endpoint" pattern:
// build a Document, hand it to WriteResponse, and the helper sets
// the Content-Type, Content-Disposition, and Content-Length headers
// before flushing the bytes.
func ExampleWriteResponse() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			Heading(1, kardec.Text("Invoice 42")).
			Paragraph(kardec.Text("Total: R$ 1,234.56"))
		if err := httpx.WriteResponse(w, doc.Document, "invoice.pdf"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	_ = handler
}

// ExampleWriteResponseInline picks the inline disposition so a
// browser-side PDF viewer renders the document in place rather than
// offering it as a download.
func ExampleWriteResponseInline() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			Paragraph(kardec.Text("Inline preview."))
		_ = httpx.WriteResponseInline(w, doc.Document, "preview.pdf")
	}
	_ = handler
}
