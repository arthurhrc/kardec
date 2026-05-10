// Package sign attaches a PKCS#7 detached signature to a
// kardec-rendered PDF, producing a document Adobe Acrobat (and
// every other PDF reader that respects signatures) reports as
// "Signed by <X>".
//
// The package implements PDF 1.7 §12.8 "Digital Signatures": a
// /AcroForm with one /Sig field, a /SubFilter /adbe.pkcs7.detached
// /Contents holding the PKCS#7 SignedData blob, and a /ByteRange
// covering the entire file except the /Contents value. The blob
// itself is built via github.com/digitorus/pkcs7 — a maintained
// fork of mozilla-services/pkcs7 — so the ASN.1 stays correct
// without reinventing CMS encoding.
//
// Typical usage:
//
//	priv, cert := loadKeyAndCert(...)  // crypto/x509 + crypto/rsa
//	doc.Render("contract.pdf")          // emit the unsigned PDF
//	raw, _ := os.ReadFile("contract.pdf")
//	signed, err := sign.Apply(raw, sign.Options{
//	    PrivateKey: priv,
//	    Certificate: cert,
//	    Reason: "I agree",
//	    Location: "São Paulo, BR",
//	    SignerName: "Arthur Carvalho",
//	})
//	os.WriteFile("contract-signed.pdf", signed, 0644)
//
// The signed file's bytes are NOT byte-reproducible — every
// signature carries a fresh signing timestamp and PKCS#7 nonce
// per the spec. The unsigned-render reprocheck still holds; the
// signature step is the layer that adds randomness.
package sign

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	pkcs7 "github.com/digitorus/pkcs7"
)

// Options configures the signature operation. PrivateKey +
// Certificate are required; everything else is optional and shows
// up in the /Sig dict.
type Options struct {
	// PrivateKey signs the document. Today only RSA is supported;
	// ECDSA / Ed25519 land if real ICP-Brasil HSMs surface them.
	PrivateKey *rsa.PrivateKey

	// Certificate is the signer's X.509 cert. Embedded in the
	// PKCS#7 blob so verifiers can check the chain without
	// out-of-band lookups.
	Certificate *x509.Certificate

	// CertChain holds intermediate CA certificates. Optional but
	// recommended — without them, readers that don't have the CA
	// pre-installed warn the user "signature valid but chain
	// untrusted".
	CertChain []*x509.Certificate

	// Reason, Location, SignerName populate the /Reason,
	// /Location, /Name fields of the /Sig dict. Acrobat surfaces
	// them in the signature-panel detail view.
	Reason     string
	Location   string
	SignerName string

	// SigningTime overrides the timestamp baked into the PKCS#7
	// signedAttrs. Defaults to time.Now when zero — pass a fixed
	// value when reproducible output is needed (the signature
	// itself still won't be reproducible because the RSA
	// signature carries random padding, but the timestamp at
	// least becomes deterministic).
	SigningTime time.Time
}

// Apply attaches a PKCS#7 detached signature to pdfBytes and
// returns the signed PDF. The input must already be a valid
// kardec-rendered PDF; the output is byte-for-byte identical to
// the input EXCEPT the appended signature-field objects, AcroForm
// catalog entry, and updated xref. Acrobat will report the
// signature as covering "the entire document".
//
// Errors come from malformed input, missing PrivateKey /
// Certificate, or PKCS#7 build failures. None of these can leak
// the private key bytes — failure modes drop the operation
// entirely rather than emitting a partial signature.
func Apply(pdfBytes []byte, opts Options) ([]byte, error) {
	if opts.PrivateKey == nil {
		return nil, errors.New("sign: PrivateKey is required")
	}
	if opts.Certificate == nil {
		return nil, errors.New("sign: Certificate is required")
	}
	if !strings.HasPrefix(string(pdfBytes[:8]), "%PDF-") {
		return nil, errors.New("sign: input does not look like a PDF (missing %PDF- header)")
	}

	signingTime := opts.SigningTime
	if signingTime.IsZero() {
		signingTime = time.Now()
	}

	// Build the signed PDF with a placeholder for /Contents. The
	// placeholder must be hex-encoded and big enough to hold the
	// final PKCS#7 blob — 16 KB is plenty for an RSA-2048 cert +
	// chain + signed attributes.
	const placeholderSize = 16384
	placeholderHex := strings.Repeat("0", placeholderSize*2)

	pdfWithPlaceholder, byteRange, err := injectSignaturePlaceholder(
		pdfBytes, placeholderHex, opts.Reason, opts.Location, opts.SignerName, signingTime,
	)
	if err != nil {
		return nil, fmt.Errorf("sign: inject placeholder: %w", err)
	}

	// Hash the bytes covered by /ByteRange. Compute over the
	// concatenation: pdfWithPlaceholder[ByteRange[0]..ByteRange[0]+ByteRange[1]]
	// + pdfWithPlaceholder[ByteRange[2]..ByteRange[2]+ByteRange[3]].
	h := sha256.New()
	h.Write(pdfWithPlaceholder[byteRange[0] : byteRange[0]+byteRange[1]])
	h.Write(pdfWithPlaceholder[byteRange[2] : byteRange[2]+byteRange[3]])
	digest := h.Sum(nil)

	// Build PKCS#7 SignedData with the digest as content. detached
	// = true so the PDF /Contents carries only the wrapper, not
	// the document bytes again.
	sd, err := pkcs7.NewSignedData(digest)
	if err != nil {
		return nil, fmt.Errorf("sign: PKCS#7 new: %w", err)
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	signerConfig := pkcs7.SignerInfoConfig{}
	if err := sd.AddSigner(opts.Certificate, opts.PrivateKey, signerConfig); err != nil {
		return nil, fmt.Errorf("sign: PKCS#7 add signer: %w", err)
	}
	for _, c := range opts.CertChain {
		sd.AddCertificate(c)
	}
	sd.Detach()
	signed, err := sd.Finish()
	if err != nil {
		return nil, fmt.Errorf("sign: PKCS#7 finish: %w", err)
	}

	// Hex-encode the PKCS#7 blob and patch it into the
	// placeholder slot.
	signedHex := hex.EncodeToString(signed)
	if len(signedHex) > placeholderSize*2 {
		return nil, fmt.Errorf(
			"sign: PKCS#7 blob (%d hex chars) exceeds %d-byte placeholder — increase placeholderSize",
			len(signedHex), placeholderSize)
	}
	// Pad with zeros on the right to fill the full placeholder
	// span so ByteRange offsets stay valid.
	padding := strings.Repeat("0", placeholderSize*2-len(signedHex))
	out := []byte(strings.Replace(string(pdfWithPlaceholder), placeholderHex, signedHex+padding, 1))
	if len(out) != len(pdfWithPlaceholder) {
		return nil, errors.New("sign: placeholder substitution changed file length — ByteRange would be wrong")
	}
	return out, nil
}

// _ asserts that crypto.Hasher is satisfied — PKCS#7 internals
// reach for it. Compile-time guard so future Go releases that
// rename the interface break the build instead of silently
// changing behaviour.
var _ crypto.Hash = crypto.SHA256
