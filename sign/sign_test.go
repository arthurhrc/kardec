package sign

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

// generateTestKey produces a self-signed RSA-2048 certificate
// suitable for exercising the sign path. Real callers load their
// ICP-Brasil cert + key from a file / HSM; tests use an ephemeral
// pair so CI doesn't need pre-baked secrets.
func generateTestKey(t *testing.T) (*rsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "Kardec Test Signer",
			Organization: []string{"Kardec CI"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("x509.ParseCertificate: %v", err)
	}
	return priv, cert
}

func renderSamplePDF(t *testing.T) []byte {
	t.Helper()
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("Contract to be signed").
		Heading(1, kardec.Text("Service contract"))
	doc.Paragraph(kardec.Text("Body of the contract — terms and conditions."))
	out, err := render.Bytes(doc)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return out
}

func TestApplyProducesValidPDFWithSignatureMarkers(t *testing.T) {
	pdf := renderSamplePDF(t)
	priv, cert := generateTestKey(t)
	signed, err := Apply(pdf, Options{
		PrivateKey:  priv,
		Certificate: cert,
		Reason:      "I agree",
		Location:    "São Paulo, BR",
		SignerName:  "Test User",
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Output must still be a valid PDF.
	if !bytes.HasPrefix(signed, []byte("%PDF-1.7")) {
		t.Errorf("signed output missing %%PDF-1.7 header")
	}
	if !bytes.HasSuffix(signed, []byte("%%EOF\n")) {
		t.Errorf("signed output missing trailing %%EOF")
	}
	// Markers Adobe Acrobat looks for.
	for _, want := range []string{
		"/Type /Sig",
		"/SubFilter /adbe.pkcs7.detached",
		"/Filter /Adobe.PPKLite",
		"/ByteRange [",
		"/AcroForm",
		"/SigFlags 3",
		"/FT /Sig",
	} {
		if !bytes.Contains(signed, []byte(want)) {
			t.Errorf("signed PDF missing required marker %q", want)
		}
	}
	// /Contents zeros placeholder must be replaced — search for
	// a hex digit-only run between < and > that contains at
	// least some non-zero bytes (the signature).
	idx := bytes.Index(signed, []byte("/Contents <"))
	if idx < 0 {
		t.Fatalf("/Contents not found in signed PDF")
	}
	end := bytes.Index(signed[idx:], []byte(">"))
	if end < 0 {
		t.Fatalf("missing > after /Contents")
	}
	body := string(signed[idx+len("/Contents <") : idx+end])
	if strings.Trim(body, "0") == "" {
		t.Errorf("/Contents stayed all-zeros — signature was not patched in")
	}
}

func TestApplyRejectsMissingKey(t *testing.T) {
	_, cert := generateTestKey(t)
	_, err := Apply(renderSamplePDF(t), Options{Certificate: cert})
	if err == nil {
		t.Errorf("Apply without PrivateKey should fail")
	}
}

func TestApplyRejectsMissingCertificate(t *testing.T) {
	priv, _ := generateTestKey(t)
	_, err := Apply(renderSamplePDF(t), Options{PrivateKey: priv})
	if err == nil {
		t.Errorf("Apply without Certificate should fail")
	}
}

func TestApplyRejectsNonPDFInput(t *testing.T) {
	priv, cert := generateTestKey(t)
	_, err := Apply([]byte("not a pdf"), Options{PrivateKey: priv, Certificate: cert})
	if err == nil {
		t.Errorf("Apply with non-PDF input should fail")
	}
}

func TestSignedPDFByteRangeCoversWholeFileMinusContents(t *testing.T) {
	pdf := renderSamplePDF(t)
	priv, cert := generateTestKey(t)
	signed, err := Apply(pdf, Options{
		PrivateKey:  priv,
		Certificate: cert,
		SignerName:  "test",
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	// Find the /ByteRange entry — it's emitted as
	// `/ByteRange [DDDDDDDDDD DDDDDDDDDD DDDDDDDDDD DDDDDDDDDD]`
	// with ten-decimal-digit slots.
	idx := bytes.Index(signed, []byte("/ByteRange ["))
	if idx < 0 {
		t.Fatalf("/ByteRange not found")
	}
	end := bytes.Index(signed[idx:], []byte("]"))
	if end < 0 {
		t.Fatalf("/ByteRange not closed")
	}
	parts := strings.Fields(string(signed[idx+len("/ByteRange [") : idx+end]))
	if len(parts) != 4 {
		t.Fatalf("/ByteRange expected 4 numbers, got %d: %v", len(parts), parts)
	}
	// br[0]+br[1]+br[2]+br[3] should equal the PDF length only
	// if br[2] points right at /Contents > closer and br[1]
	// ends right at /Contents <. Verify the boundary: the byte
	// before signed[br[1]] should be `<`.
	br := make([]int, 4)
	for i, p := range parts {
		n := 0
		for _, c := range p {
			n = n*10 + int(c-'0')
		}
		br[i] = n
	}
	if br[0] != 0 {
		t.Errorf("ByteRange[0] = %d, want 0", br[0])
	}
	if br[1] <= 0 || br[1] >= len(signed) {
		t.Errorf("ByteRange[1] = %d out of bounds (len %d)", br[1], len(signed))
	}
	if br[2] <= br[1] {
		t.Errorf("ByteRange[2] = %d should be > ByteRange[1] = %d", br[2], br[1])
	}
	total := br[2] + br[3]
	if total != len(signed) {
		t.Errorf("ByteRange[2]+ByteRange[3] = %d, file length = %d", total, len(signed))
	}
}
