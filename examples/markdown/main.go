// Markdown demonstrates AppendMarkdown — feeding raw CommonMark to a
// Document and rendering the result. Useful for converting documentation
// or generated reports without manually emitting block calls.
//
// Run from the repository root:
//
//	go run ./examples/markdown
//
// The input is intentionally varied: H1 / H2, paragraphs with **bold**
// and *italic*, an unordered list, a thematic break that becomes a page
// break, and a final summary paragraph.
package main

import (
	"fmt"
	"log"

	"github.com/arthurhrc/kardec"
	_ "github.com/arthurhrc/kardec/render"
)

const source = `# Kardec — Markdown ingest

Kardec accepts raw **CommonMark** as an alternative to the fluent builder.
Body text flows through the same layout engine, so headings honor the
**named-style table** and paragraphs benefit from line breaking.

## What is supported

- Headings 1 through 6
- Bold, italic, and combined ` + "`bold-italic`" + ` runs
- Unordered and ordered lists *(flattened to bullets in v0.1)*
- Horizontal rules become page breaks
- Code blocks render with the StyleCode named style

## What is not (yet)

- Tables and images render as TODO stubs (v0.2)
- Inline links keep the visible text only — URL is dropped (v0.2)

---

# Closing

The same Document can mix DSL calls and AppendMarkdown — for instance,
seed a frame with the builder, then append a CHANGELOG file's contents
verbatim. The resulting PDF is a single, coherent document.
`

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).AppendMarkdown(source)
	if err := doc.Err(); err != nil {
		log.Fatalf("builder error: %v", err)
	}
	if err := doc.Render("markdown.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	fmt.Println("rendered markdown.pdf")
}
