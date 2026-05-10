// Signature demonstrates the kardec/sign subpackage: render a
// PDF, then apply a PKCS#7 detached signature so Acrobat
// (and ICP-Brasil-aware viewers) report it as "Signed by X".
//
//	go run ./examples/signature
//
// Produces signature.pdf signed with an ephemeral self-signed
// certificate. Acrobat will validate the cryptographic signature
// but warn that the issuer is untrusted — production callers
// swap in an ICP-Brasil cert + private key (loaded from a smart
// card / HSM / encrypted file).
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
	"github.com/arthurhrc/kardec/sign"
)

func main() {
	// Render the unsigned PDF.
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("Service Contract").
		Heading(1, kardec.Text("Service contract"))
	doc.Paragraph(kardec.Text(
		"This is a sample contract demonstrating Kardec's PDF signature " +
			"integration. Adobe Acrobat will display this document as " +
			"signed once the kardec/sign step runs."))
	doc.Paragraph(kardec.Text(
		"Production usage replaces the ephemeral key + cert below with a " +
			"real ICP-Brasil certificate loaded from a smart card or HSM."))

	pdfBytes, err := docBytes(doc)
	if err != nil {
		log.Fatalf("render: %v", err)
	}

	// Generate an ephemeral RSA key + self-signed cert. Real use
	// loads these from a secure store.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "Kardec Demo Signer",
			Organization: []string{"Kardec"},
			Country:      []string{"BR"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("x509.CreateCertificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		log.Fatalf("x509.ParseCertificate: %v", err)
	}

	// Apply the signature.
	signed, err := sign.Apply(pdfBytes, sign.Options{
		PrivateKey:  priv,
		Certificate: cert,
		Reason:      "I agree with the terms",
		Location:    "São Paulo, BR",
		SignerName:  "Kardec Demo Signer",
	})
	if err != nil {
		log.Fatalf("sign.Apply: %v", err)
	}

	if err := os.WriteFile("signature.pdf", signed, 0o644); err != nil {
		log.Fatalf("write: %v", err)
	}
	log.Printf("rendered signature.pdf (%d bytes, PKCS#7-signed with ephemeral cert)\n", len(signed))
}

// docBytes is a small helper because Document.Bytes is the
// "render to memory" convenience — kept inline so the example
// stays a single file.
func docBytes(doc interface {
	Render(path string) error
}) ([]byte, error) {
	// Render to a temp file, then read it back. Avoids depending
	// on kardec.Document.Bytes for the example.
	tmp, err := os.CreateTemp("", "kardec-sign-*.pdf")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	if err := doc.Render(tmp.Name()); err != nil {
		return nil, err
	}
	return os.ReadFile(tmp.Name())
}
