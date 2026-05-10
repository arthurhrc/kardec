package render_test

import (
	"bytes"
	"testing"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

func TestEncryptionEmitsEncryptDictAndIDArray(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetEncryption(kardec.EncryptionOptions{
			UserPassword:  "open-me",
			OwnerPassword: "owner",
			Permissions:   kardec.ReadOnlyPermissions(),
		}).
		Paragraph(kardec.Text("encrypted body"))

	out, err := render.Bytes(doc.Document)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	for _, want := range []string{
		"/Filter /Standard", // Standard Security Handler
		"/V 4",              // version 4
		"/R 4",              // revision 4
		"/Length 128",       // AES-128
		"/CFM /AESV2",       // AES-V2 crypt filter method
		"/StmF /StdCF",      // streams use the StdCF filter
		"/StrF /StdCF",      // strings encrypted via AESV2 (v0.22+)
		"/Encrypt ",         // trailer entry
		"/ID [<",            // /ID array required by the security handler
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("encryption marker %q missing from PDF byte stream", want)
		}
	}
}

func TestEncryptionStreamLengthsDifferFromPlain(t *testing.T) {
	plain, err := render.Bytes(kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Paragraph(kardec.Text("Quick brown fox jumps over the lazy dog.")).Document)
	if err != nil {
		t.Fatalf("plain Bytes: %v", err)
	}
	enc, err := render.Bytes(kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetEncryption(kardec.EncryptionOptions{UserPassword: "p"}).
		Paragraph(kardec.Text("Quick brown fox jumps over the lazy dog.")).Document)
	if err != nil {
		t.Fatalf("encrypted Bytes: %v", err)
	}
	// Encryption adds the /Encrypt indirect object + AES IV (16
	// bytes per encrypted stream) + PKCS#7 padding (1-16 bytes).
	// Encrypted output must therefore be larger than plain.
	if len(enc) <= len(plain) {
		t.Errorf("encrypted output (%d bytes) should be larger than plain (%d) — IV + padding overhead",
			len(enc), len(plain))
	}
	// /Encrypt entry only appears when encryption is on.
	if bytes.Contains(plain, []byte("/Encrypt ")) {
		t.Errorf("plain output should not have /Encrypt")
	}
	if !bytes.Contains(enc, []byte("/Encrypt ")) {
		t.Errorf("encrypted output should have /Encrypt")
	}
}

func TestPermissionsHelpersProduceExpectedBits(t *testing.T) {
	all := kardec.AllPermissions()
	if !all.Print || !all.Modify || !all.Copy {
		t.Errorf("AllPermissions should grant everything")
	}
	ro := kardec.ReadOnlyPermissions()
	if ro.Print || ro.Modify || ro.Copy || ro.Annotate {
		t.Errorf("ReadOnlyPermissions should disallow Print/Modify/Copy/Annotate")
	}
	if !ro.AccessibilityCopy {
		t.Errorf("ReadOnlyPermissions should leave AccessibilityCopy on for screen readers")
	}
}

func TestEncryptionRoundtripViaAccessor(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetEncryption(kardec.EncryptionOptions{
			UserPassword:  "u",
			OwnerPassword: "o",
			Permissions:   kardec.Permissions{Print: true, Copy: true},
		})
	got, ok := doc.Encryption()
	if !ok {
		t.Fatalf("Encryption() should report enabled after SetEncryption")
	}
	if got.UserPassword != "u" || got.OwnerPassword != "o" {
		t.Errorf("password roundtrip lost: got %+v", got)
	}
	if !got.Permissions.Print || !got.Permissions.Copy {
		t.Errorf("permissions roundtrip lost: %+v", got.Permissions)
	}
	if got.Permissions.Modify {
		t.Errorf("Modify should be false")
	}
}
