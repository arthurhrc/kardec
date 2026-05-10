// Encryption demonstrates Kardec's PDF Standard Security Handler
// integration. SetEncryption opts the document into V=4 / R=4 /
// AES-128 with strings AND streams encrypted (post-v0.22).
//
//	go run ./examples/encryption
//
// Produces encryption.pdf, openable with the user password "open-me".
// Try copying the title or content with the wrong password — Acrobat
// will refuse.
package main

import (
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		SetTitle("Confidential Q4 Memo").
		SetAuthor("Internal Audit").
		SetEncryption(kardec.EncryptionOptions{
			UserPassword:  "open-me",
			OwnerPassword: "owner-secret",
			// Read-only: viewable + accessibility-extractable, but no
			// printing, copying, or modification without the owner pwd.
			Permissions: kardec.ReadOnlyPermissions(),
		}).
		Heading(1, kardec.Text("Confidential Q4 Memo")).
		Paragraph(
			kardec.Text("This document demonstrates encryption. "),
			kardec.Text("Title, author, and link URLs are AES-128 encrypted "),
			kardec.Text("alongside the page content streams."),
		).
		Paragraph(
			kardec.Text("External link to a (fictional) staging host: "),
			kardec.Link("staging dashboard", "https://staging.internal.example.com/q4"),
			kardec.Text("."),
		)

	if err := doc.Render("encryption.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered encryption.pdf — open with password 'open-me'")
}
