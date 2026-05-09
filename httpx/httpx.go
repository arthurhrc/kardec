// Package httpx is the three-line ergonomic helper every Go shop pastes
// into their first PDF endpoint. WriteResponse renders a Document and
// writes it to an http.ResponseWriter with the correct Content-Type,
// Content-Disposition, and (when the underlying writer supports it)
// Content-Length headers.
//
// Importing this package adds no behaviour to the rest of Kardec; it is
// a pure consumer of the public render package and exists only so that
// callers do not need to write the boilerplate themselves.
//
//	import (
//	    "net/http"
//
//	    "github.com/arthurhrc/kardec"
//	    _ "github.com/arthurhrc/kardec/render"
//	    "github.com/arthurhrc/kardec/httpx"
//	)
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
//	        Heading(1, kardec.Text("Invoice"))
//	    if err := httpx.WriteResponse(w, doc, "invoice.pdf"); err != nil {
//	        http.Error(w, err.Error(), http.StatusInternalServerError)
//	    }
//	}
package httpx

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

// DispositionInline lets browsers render the PDF inline (PDF.js, the
// built-in Chrome/Firefox viewers). Default is attachment, matching
// the most common "download a report" use case.
type Disposition int

const (
	// Attachment forces the browser to offer the PDF as a download.
	Attachment Disposition = iota
	// Inline asks the browser to display the PDF in-page when the
	// runtime supports a PDF viewer.
	Inline
)

// WriteResponse renders d and writes the resulting PDF bytes to w with
// `Content-Type: application/pdf` and an attachment disposition naming
// the file. Pass an empty filename to omit the disposition header.
//
// The render is buffered so the writer can emit Content-Length up front;
// callers that prefer a streaming render should use WriteResponseTo
// directly against the underlying http.ResponseWriter.
func WriteResponse(w http.ResponseWriter, d *kardec.Document, filename string) error {
	return writeWithDisposition(w, d, filename, Attachment)
}

// WriteResponseInline is the inline variant of WriteResponse: it sets
// `Content-Disposition: inline; filename="..."` so a browser-side PDF
// viewer renders the document in place rather than offering it for
// download.
func WriteResponseInline(w http.ResponseWriter, d *kardec.Document, filename string) error {
	return writeWithDisposition(w, d, filename, Inline)
}

func writeWithDisposition(w http.ResponseWriter, d *kardec.Document, filename string, disp Disposition) error {
	if w == nil {
		return fmt.Errorf("httpx: nil ResponseWriter")
	}
	if d == nil {
		return fmt.Errorf("httpx: nil Document")
	}
	pdfBytes, err := render.Bytes(d)
	if err != nil {
		return err
	}
	h := w.Header()
	h.Set("Content-Type", "application/pdf")
	h.Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	if filename != "" {
		h.Set("Content-Disposition", contentDisposition(disp, filename))
	}
	if _, err := w.Write(pdfBytes); err != nil {
		return err
	}
	return nil
}

// contentDisposition formats the Content-Disposition header value.
// RFC 6266 says the filename should be quoted; non-ASCII filenames
// would also use the `filename*=UTF-8''<percent-encoded>` form, but
// the ASCII path covers the overwhelming majority of report names.
// Quotes inside the filename are stripped to keep the header valid.
func contentDisposition(disp Disposition, filename string) string {
	prefix := "attachment"
	if disp == Inline {
		prefix = "inline"
	}
	clean := strings.ReplaceAll(filename, `"`, "")
	return fmt.Sprintf(`%s; filename="%s"`, prefix, clean)
}
