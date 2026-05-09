package httpx_test

import (
	"bytes"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/httpx"
	_ "github.com/arthurhrc/kardec/render"
)

func helloDoc() *kardec.Document {
	return kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("hello")).
		Paragraph(kardec.Text("body")).Document
}

func TestWriteResponseSetsCorrectHeadersAndAttachment(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := httpx.WriteResponse(rec, helloDoc(), "report.pdf"); err != nil {
		t.Fatalf("WriteResponse: %v", err)
	}
	resp := rec.Result()
	if got := resp.Header.Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", got)
	}
	if got := resp.Header.Get("Content-Disposition"); got != `attachment; filename="report.pdf"` {
		t.Errorf("Content-Disposition = %q", got)
	}
	body := rec.Body.Bytes()
	if !bytes.HasPrefix(body, []byte("%PDF-")) {
		t.Errorf("body should start with %%PDF-, got first 8 bytes: %x", body[:8])
	}
	cl, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		t.Fatalf("Content-Length not numeric: %v", err)
	}
	if cl != len(body) {
		t.Errorf("Content-Length=%d but body has %d bytes", cl, len(body))
	}
}

func TestWriteResponseInlineUsesInlineDisposition(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := httpx.WriteResponseInline(rec, helloDoc(), "view.pdf"); err != nil {
		t.Fatalf("WriteResponseInline: %v", err)
	}
	if got := rec.Result().Header.Get("Content-Disposition"); got != `inline; filename="view.pdf"` {
		t.Errorf("Content-Disposition = %q, want inline", got)
	}
}

func TestWriteResponseEmptyFilenameOmitsDisposition(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := httpx.WriteResponse(rec, helloDoc(), ""); err != nil {
		t.Fatalf("WriteResponse: %v", err)
	}
	if got := rec.Result().Header.Get("Content-Disposition"); got != "" {
		t.Errorf("Content-Disposition should be unset, got %q", got)
	}
}

func TestWriteResponseFilenameQuotesAreStripped(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := httpx.WriteResponse(rec, helloDoc(), `weird"name.pdf`); err != nil {
		t.Fatalf("WriteResponse: %v", err)
	}
	if got := rec.Result().Header.Get("Content-Disposition"); got != `attachment; filename="weirdname.pdf"` {
		t.Errorf("Content-Disposition = %q", got)
	}
}

func TestWriteResponseRejectsNilArguments(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := httpx.WriteResponse(rec, nil, "x.pdf"); err == nil {
		t.Errorf("expected error for nil Document")
	}
	if err := httpx.WriteResponse(nil, helloDoc(), "x.pdf"); err == nil {
		t.Errorf("expected error for nil ResponseWriter")
	}
}
