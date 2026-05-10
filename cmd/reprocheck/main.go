// Command reprocheck renders a canonical Kardec document twice with a
// pinned creation date and asserts both renders produce identical
// bytes. Exits 0 on match, 1 on mismatch.
//
// Used by CI as the explicit reproducibility gate: byte-for-byte
// stability is the load-bearing differentiator vs Maroto / fpdf, and
// a regression in any indirect dependency that introduces map-
// iteration nondeterminism would silently break it. This program
// makes the check fail-loud in every CI run on every OS.
//
//	go run ./cmd/reprocheck
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/render"
)

const fixedDate = "2026-05-10T12:00:00Z"

func main() {
	t, err := time.Parse(time.RFC3339, fixedDate)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reprocheck: parse fixed date:", err)
		os.Exit(2)
	}
	build := func() ([]byte, error) {
		doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
			SetCreationDate(t).
			SetTitle("Reproducibility canonical").
			SetAuthor("Kardec CI").
			Heading(1, kardec.Text("Hello, deterministic world.")).
			Paragraph(
				kardec.Text("Two renders, "),
				kardec.Bold("identical bytes"),
				kardec.Text("."),
			).
			Math(`a^2 + b^2 = c^2`)
		return render.Bytes(doc)
	}
	a, err := build()
	if err != nil {
		fmt.Fprintln(os.Stderr, "reprocheck: first render:", err)
		os.Exit(2)
	}
	b, err := build()
	if err != nil {
		fmt.Fprintln(os.Stderr, "reprocheck: second render:", err)
		os.Exit(2)
	}
	hashA := sha256.Sum256(a)
	hashB := sha256.Sum256(b)
	hexA := hex.EncodeToString(hashA[:])
	hexB := hex.EncodeToString(hashB[:])

	fmt.Printf("render 1 sha256: %s (%d bytes)\n", hexA, len(a))
	fmt.Printf("render 2 sha256: %s (%d bytes)\n", hexB, len(b))

	if hexA != hexB {
		fmt.Fprintln(os.Stderr, "reprocheck: HASH MISMATCH — output is not byte-reproducible")
		os.Exit(1)
	}
	fmt.Println("reprocheck: OK (byte-reproducible)")
}
