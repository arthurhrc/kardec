# Kardec

Document-style PDFs in pure Go. Embedded fonts, no Docker, no system dependencies.

[![Go Reference](https://pkg.go.dev/badge/github.com/arthurhrc/kardec.svg)](https://pkg.go.dev/github.com/arthurhrc/kardec)
[![CI](https://github.com/arthurhrc/kardec/actions/workflows/ci.yml/badge.svg)](https://github.com/arthurhrc/kardec/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/arthurhrc/kardec)](https://goreportcard.com/report/github.com/arthurhrc/kardec)
[![Latest release](https://img.shields.io/github/v/release/arthurhrc/kardec)](https://github.com/arthurhrc/kardec/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/arthurhrc/kardec)](go.mod)
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
# Core document features
go run ./examples/hello              # smallest end-to-end PDF
go run ./examples/report             # multi-section, header/footer, table with shading
go run ./examples/markdown           # CommonMark + GFM tables
go run ./examples/invoice            # text/template + Markdown -> one PDF per record
go run ./examples/image              # PNG embedded into the PDF
go run ./examples/math               # LaTeX math subset (\frac, \sqrt, \sum, \int, greek)

# Recent features (v0.19+)
go run ./examples/svg                # SVG embed as vector Form XObject
go run ./examples/encryption         # AES-128 with read-only permissions
go run ./examples/watermark          # diagonal "DRAFT" overlay with alpha
go run ./examples/tagged             # PDF/UA tagged structure (H1/H2/P/Figure)
go run ./examples/inlinemath         # math expression inside a paragraph
go run ./examples/qrcode             # QR codes at L/M/Q/H error tiers
go run ./examples/bookstyle          # background letterhead + verso/recto headers
go run ./examples/templates_invoice  # one-line invoice from kardec/templates
go run ./examples/chart              # bar / line / pie charts via kardec/chart
go run ./examples/signature          # PKCS#7 detached signature via kardec/sign
```

## Features

Document structure

- Headings, paragraphs, lists, tables (borders + shading + colspan), images (JPEG / PNG / SVG),
  page breaks, spacers, horizontal rules
- `KeepTogether(blocks...)` binds groups to a single page
- Two-column section layout via `PageSetup.Columns`
- Multi-section page setups (mix portrait, landscape, custom margins)
- Section background image; first-page / even-page header & footer variants

Inline content

- Run decorations: `Bold`, `Italic`, `Underline`, `Strikethrough`, colored, sized
- Hyperlinks, named anchors, automatic PDF outline (sidebar bookmarks)
- LaTeX math subset (`\frac`, `\sqrt`, `\sum`, `\int`, greek) — `Document.Math` (display) and `InlineMath` (Run-level)
- CommonMark + GFM ingest with Markdown image embed

Cross-references and references

- Auto figure / table numbering with `Label(name)` + `doc.Ref(label)` / `doc.RefPage(label)`
- Numeric citations: `doc.Cite(key)` + `doc.Bibliography(entries...)`
- Auto table of contents (clickable, resolved in a post-pass)
- Footnotes with auto-numbering or custom markers
- `Leader(left, right)` dotted rows; `SignatureBlock(name, role)` for contracts
- `Clause(level, runs...)` hierarchical numbering for legal documents

Styling and layout

- Style system with inheritance and per-block overrides
- Section headers and footers with `{{page}}` / `{{totalPages}}` / `{{section}}` / `{{date}}` tokens
- Decimal-point column alignment for currency / measurement tables
- `Image.Caption(...)` auto-prefixed with the figure marker, kept-together with its image
- Knuth-Liang hyphenation with bundled pattern sets: `en`, `pt-BR`, `es`, `fr`
- Optional Knuth-Plass optimum-fit line breaker

Output

- Byte-reproducible output via `Document.SetCreationDate(t)` (unsigned rendering)
- Optional TTF font subsetting (~70 % size reduction)
- PDF/A-2b conformance markers; OutputIntent + ICC profile when supplied
- PDF/UA-1 strict tagging: per-block H1–H6 / P / Figure roles, `Table > TR > TD/TH` hierarchy, `Sect` groupings around H1 boundaries
- AES-128 encryption + permissions (Standard Security Handler V=4 / R=4); strings + streams both encrypted (`/StrF /StdCF`)
- Per-page diagonal watermark with alpha blending
- QR codes via `Document.QRCode` (vector Form XObject)
- Unicode text via Type 0 / Identity-H font embedding — Δ, Σ, Cyrillic, CJK render correctly

Companion subpackages

- `kardec/render` — registers the renderer (blank-imported once at app entry)
- `kardec/httpx` — `WriteResponse(w, doc, filename)` for `net/http` handlers
- `kardec/templates` — ready-made `Invoice` / `Certificate` / `Report` scaffolds
- `kardec/chart` — pure-Go bar / line / pie chart renderer (SVG out)
- `kardec/sign` — PKCS#7 detached signatures (Adobe-compatible, ICP-Brasil-ready)

CLI

- `cmd/kardec` ships a `kardec` binary for rendering Markdown + Go templates without writing Go.

## What's NOT in Kardec

Honest non-goals — if you need these, reach for another tool or
contribute the support:

- **Interactive form fields** (AcroForm beyond signatures). Use a dedicated signing flow.
- **Multi-script text shaping** for Arabic, Hebrew (RTL), Devanagari, Thai, Burmese. Kardec
  handles every Unicode codepoint the source TTF covers but does not run a shaping engine;
  ligatures, complex marks, and bidi reordering are out of scope. CJK / Cyrillic /
  Greek + Latin scripts render correctly.
- **PDF/X (print-industry colour management)**. PDF/A-2b is the closest cousin and ships.
- **Browser-style HTML/CSS rendering**. Use Gotenberg + LibreOffice when input is HTML.
- **Live editing / WYSIWYG**. Kardec is one-pass programmatic generation, not a layout tool.

## Status

**v1.0 — API frozen.** Every public type, function, and method
exposed at v1.0 stays signature-compatible through the v1.x line.
Breaking changes need a v2.0 major bump. Internal packages
(`internal/...`) remain unstable; callers reaching into them get
whatever they deserve.

| Range | What it brought |
|---|---|
| 0.1–0.5 | Core document model, multi-face fonts, tables, images, Markdown ingest, math subset, byte-reproducible output, font subsetting |
| 0.6–0.11 | Cross-references, KeepTogether, footnotes, TOC, hyphenation, two-column layout, metadata setters, OutputIntent |
| 0.12–0.14 | CI quality gates, reproducibility check, ToUnicode CMaps, clickable TOC |
| 0.15–0.21 | OTF/CFF math glyphs, AES-128 encryption, PDF/UA-1 lite tagging, Knuth-Plass breaker, SVG embed, watermark, inline math |
| 0.22 | API cleanup pass; Identity-H Unicode body text; per-block PDF/UA roles; encrypted strings |
| 0.23 | Hyphenation pt/es/fr; background image; first-page + even-page header / footer; QR codes; table cell roles; `kardec/templates` |
| 0.24 | PDF/UA strict: `Table > TR > TD/TH` nesting + `Sect` groupings |
| 0.25 | `kardec/chart` — pure-Go bar / line / pie |
| 0.26 | `kardec/sign` — PKCS#7 detached signatures |
| **1.0 (current)** | **API frozen.** No code change from v0.26 — the surface is committed-stable. |

Full release notes in [CHANGELOG.md](CHANGELOG.md).
Design spec in [docs/RFC-001-dsl.md](docs/RFC-001-dsl.md).
Migration guide in [MIGRATING.md](MIGRATING.md).

## Contributing

PRs welcome. Each feature lands on its own branch with granular commits and a
PR before merging to `main`. Run `go vet ./... && go test ./...` before pushing.

## License

MIT. See [LICENSE](LICENSE) for the source license and [NOTICE.md](NOTICE.md)
for bundled-font and dependency attributions.
