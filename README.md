# Kardec

Document-style PDFs in pure Go. Embedded fonts, no Docker, no system dependencies.

[![Go Reference](https://pkg.go.dev/badge/github.com/arthurhrc/kardec.svg)](https://pkg.go.dev/github.com/arthurhrc/kardec)
[![Go Report Card](https://goreportcard.com/badge/github.com/arthurhrc/kardec)](https://goreportcard.com/report/github.com/arthurhrc/kardec)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

```go
package main

import (
    "log"

    "github.com/arthurhrc/kardec"
    _ "github.com/arthurhrc/kardec/render"
)

func main() {
    doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
        Heading(1, kardec.Text("Monthly Report")).
        Paragraph(
            kardec.Text("Sales grew "),
            kardec.Bold("12%"),
            kardec.Text(" this quarter."),
        )
    if err := doc.Render("report.pdf"); err != nil {
        log.Fatal(err)
    }
}
```

## Install

```sh
go get github.com/arthurhrc/kardec
```

Go 1.22+. The blank import `_ "github.com/arthurhrc/kardec/render"` wires
`Document.Render` via `init()`. Without it, calls to `Render` return
`ErrRendererUnregistered`.

## Why Kardec

Allan Kardec (1804-1869) was a French educator who took a body of
scattered oral traditions and codified them into a structured written
doctrine. The library does the equivalent for documents: a program
assembles a flowing structure through a fluent Go API, and Kardec
freezes it into a portable PDF that opens identically on every reader.
The name is a nod to the act of codification.

What that buys you in practice:

- Flowing prose, headings, footnotes, tables, math, lists. Not a grid.
- Single static binary. Liberation Sans / Serif, Carlito and JetBrains Mono ship inside (~7 MB, OFL).
- Byte-reproducible output: same input plus a fixed clock, identical PDFs.

## Comparison

| Tool | Pure Go | Container needed | Document-flow | Markdown ingest | Templating |
|---|---|---|---|---|---|
| Kardec | yes | no | yes | yes | yes |
| [Maroto](https://github.com/johnfercher/maroto) | yes | no | grid only | no | no |
| [gofpdf / fpdf](https://github.com/go-pdf/fpdf) | yes | no | low-level only | no | no |
| [Gotenberg](https://github.com/gotenberg/gotenberg) | no (Docker) | yes | yes (via LibreOffice) | yes | no |

Kardec covers the seam: pure Go like Maroto, document-shaped like Gotenberg.

## Examples

```sh
go run ./examples/hello       # smallest end-to-end PDF
go run ./examples/report      # multi-section, header/footer, table with shading
go run ./examples/markdown    # CommonMark + GFM tables
go run ./examples/invoice     # text/template + Markdown -> one PDF per record
go run ./examples/image       # PNG embedded into the PDF
go run ./examples/math        # LaTeX math subset (\frac, \sqrt, \sum, \int, greek)
```

## Features

Document structure

- Headings, paragraphs, lists, tables (borders + shading + colspan), images (JPEG / PNG),
  page breaks, spacers, horizontal rules
- `KeepTogether(blocks...)` binds groups to a single page
- Two-column section layout via `PageSetup.Columns`
- Multi-section page setups (mix portrait, landscape, custom margins)

Inline content

- Run decorations: `Bold`, `Italic`, `Underline`, `Strikethrough`, colored, sized
- Hyperlinks, named anchors, automatic PDF outline (sidebar bookmarks)
- LaTeX math subset (`\frac`, `\sqrt`, `\sum`, `\int`, greek)
- CommonMark + GFM ingest with Markdown image embed

Cross-references and references

- Auto figure / table numbering with `Label(name)` + `doc.Ref(label)` / `doc.RefPage(label)`
- Numeric citations: `doc.Cite(key)` + `doc.Bibliography(entries...)`
- Auto table of contents resolved in a post-pass
- Footnotes with auto-numbering or custom markers
- `Leader(left, right)` dotted rows; `SignatureBlock(name, role)` for contracts
- `Clause(level, runs...)` hierarchical numbering for legal documents

Styling and layout

- Style system with inheritance and per-block overrides
- Section headers and footers with `{{page}}` / `{{totalPages}}` / `{{section}}` / `{{date}}` tokens
- Decimal-point column alignment for currency / measurement tables
- `Image.Caption(...)` auto-prefixed with the figure marker, kept-together with its image
- Heuristic English word hyphenation

Output

- Byte-reproducible output via `Document.SetCreationDate(t)`
- Optional TTF font subsetting (~70 % size reduction)
- Optional PDF/A-2b conformance markers (lite)
- `kardec/httpx.WriteResponse` helper for `net/http` handlers
- `text/template` companion for per-record generation

## Status

| Version | Notes |
|---|---|
| 0.1 | Paragraphs, headings, page breaks, styles, embedded fonts |
| 0.2 | Multi-face fonts, tables, images, Markdown, templating |
| 0.3 | Math subset, lists, headers/footers, hyperlinks + outline, byte-reproducible output |
| 0.4 | Anchors, table borders, multi-section, footnotes, auto-TOC, hyphenation, Markdown images |
| 0.5 | TTF font subsetting, PDF/A-2b lite |
| 0.6 | Skipped — feature batch folded forward into 0.7; see [CHANGELOG](CHANGELOG.md#060) |
| 0.7 | HorizontalRule, run decorations, KeepTogether, cross-references, image captions, `kardec/httpx` |
| 0.8 | Leader, SignatureBlock, Clause numbering, Bibliography + Cite, table colspan, decimal alignment, two-column layout |
| 0.9 | Paragraph builder unified, `cmd/kardec` CLI, runnable godoc Examples, render benchmarks |
| 0.10 | API rename sweep (Enable*/Disable*, `WithAlignment`, `TableBorders*`, `NewSection(setup)`, `NewCell`, `Document.Footnote`); `RegisteredFamilies()`; `MIGRATING.md` |
| 0.11 (current) | Metadata setters (Title/Author/Subject/Keywords); OutputIntent + ICC profile infrastructure (strict PDF/A-2b ready); Knuth-Liang hyphenation |
| 1.0 (planned) | API freeze; see [docs/ROADMAP_TO_V1.md](docs/ROADMAP_TO_V1.md) |

Full release notes in [CHANGELOG.md](CHANGELOG.md).
Design spec in [docs/RFC-001-dsl.md](docs/RFC-001-dsl.md).

## Contributing

PRs welcome. Each feature lands on its own branch with granular commits and a
PR before merging to `main`. Run `go vet ./... && go test ./...` before pushing.

## License

MIT. See [LICENSE](LICENSE) for the source license and [NOTICE.md](NOTICE.md)
for bundled-font and dependency attributions.
