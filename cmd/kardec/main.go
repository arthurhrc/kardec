// Command kardec is a CLI front-end for the Kardec library. It wraps
// AppendMarkdown and Template.Render into a one-line invocation:
//
//	kardec render input.md -o out.pdf
//	kardec render -t invoice.md -d invoice.json -o invoice.pdf
//
// The CLI is a thin shim over the library — every flag maps directly
// to a public Kardec call. Anything not exposed here can be reached
// by importing the library directly.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

const usageText = `kardec — render Markdown (or a templated Markdown source) to PDF.

Usage:
  kardec render <input.md> [-o output.pdf]
  kardec render -t template.md [-d data.json] [-o output.pdf]

Flags:
  -o   output PDF path (default "out.pdf")
  -t   template Markdown path; when set the positional input is ignored
  -d   JSON data path passed to the template's Execute step

Run "kardec render -h" for full flag reference.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "render":
		if err := runRender(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "kardec:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		fmt.Print(usageText)
	case "version", "-v", "--version":
		fmt.Println("kardec dev")
	default:
		fmt.Fprintf(os.Stderr, "kardec: unknown subcommand %q\n\n%s", os.Args[1], usageText)
		os.Exit(2)
	}
}

// runRender wires the render subcommand: parse flags, choose the
// markdown vs template path, and write a PDF to the resolved output.
//
// Positional arguments are pulled out before flag parsing so the
// canonical "kardec render input.md -o out.pdf" form works as the
// user types it. Go's stdlib flag package stops at the first
// non-flag token, which makes "input.md -o out.pdf" silently drop
// the -o without this pre-scan.
func runRender(args []string) error {
	flagsOnly, positional := splitFlagsAndPositional(args)

	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	out := fs.String("o", "out.pdf", "output PDF path")
	tplPath := fs.String("t", "", "template Markdown path (optional)")
	dataPath := fs.String("d", "", "JSON data path (used with -t)")
	if err := fs.Parse(flagsOnly); err != nil {
		return err
	}

	if *tplPath != "" {
		return renderTemplate(*tplPath, *dataPath, *out)
	}
	if len(positional) < 1 {
		return errors.New("render: missing input markdown path (or -t template)")
	}
	return renderMarkdown(positional[0], *out)
}

// splitFlagsAndPositional partitions args into flag tokens and
// positional tokens so flag.Parse can run on the flags alone. A
// flag token starts with "-" and may pull the next token along as
// its value when no equals sign is present (e.g. "-o foo").
func splitFlagsAndPositional(args []string) (flags, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 0 && a[0] == '-' {
			flags = append(flags, a)
			// Multi-token flags ("-o out.pdf"): pull the next
			// argument unless this token already carries "=".
			if i+1 < len(args) && needsValue(a) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, a)
	}
	return flags, positional
}

// needsValue reports whether a flag token of the form "-x" expects a
// following value token. Boolean flags (none today, but adding one
// would land here) do not. Tokens carrying "=" already have their
// value baked in.
func needsValue(token string) bool {
	for i := 0; i < len(token); i++ {
		if token[i] == '=' {
			return false
		}
	}
	switch token {
	case "-h", "--help":
		return false
	}
	return true
}

// renderMarkdown reads input.md, builds a fresh Document, appends the
// markdown, and renders to the supplied output path.
func renderMarkdown(input, output string) error {
	src, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read %s: %w", input, err)
	}
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		AppendMarkdown(string(src))
	if err := doc.Err(); err != nil {
		return fmt.Errorf("parse markdown: %w", err)
	}
	return doc.Render(output)
}

// renderTemplate parses tpl.md, optionally executes it against the
// JSON in dataPath, and renders the resulting document. An empty
// dataPath substitutes a nil data context — useful when the template
// is data-free but the caller still wants the markdown pipeline.
func renderTemplate(tplPath, dataPath, output string) error {
	tplSrc, err := os.ReadFile(tplPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", tplPath, err)
	}
	tpl, err := kardec.NewTemplate(string(tplSrc))
	if err != nil {
		return err
	}
	var data any
	if dataPath != "" {
		raw, err := os.ReadFile(dataPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", dataPath, err)
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("parse %s: %w", dataPath, err)
		}
	}
	doc, err := tpl.Render(data)
	if err != nil {
		return err
	}
	return doc.Render(output)
}
