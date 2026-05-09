# Kardec

> A Go DSL for producing document-like PDFs — pure Go, embedded fonts, no container required.

```go
import (
    "github.com/arthurhrc/kardec"
    _ "github.com/arthurhrc/kardec/render"
)

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
```

## What it is

Kardec generates PDFs that read like a document, not a printout. Body text flows; headings have rhythm; named styles compose with inheritance; bundled fonts ship with the binary so the output looks the same on every machine.

Under the hood:

- **`internal/typography`** — font registry, OpenType shaping (via `tdewolff/canvas`), Liberation Sans / Liberation Serif / Carlito / JetBrains Mono embedded in the binary (~7 MB, OFL-licensed).
- **`internal/layout`** — line breaking, page break logic, block-level placement. Style-driven through `Document.ResolveBlockStyle`.
- **`internal/pdf`** — minimal PDF 1.7 writer with TrueType font embedding (FontFile2 / WinAnsiEncoding).
- **`render`** — orchestrator that wires the three together. Importing it (even blank) installs `Document.Render`.

## Why this name

[Allan Kardec](https://en.wikipedia.org/wiki/Allan_Kardec) was the 19th-century French author known as **the codifier** — he organized a sprawling body of ideas into a structured form readable by anyone. The metaphor fits: Kardec the library codifies a fluent builder API into a portable document. The name is short, unique on `pkg.go.dev`, and idiomatic to import:

```go
import "github.com/arthurhrc/kardec"
```

## Why this project

The Go ecosystem has two well-established families of PDF tooling — both excellent at what they do, both leaving a specific gap:

| Tool | Strength | Constraint that Kardec is filling |
|------|----------|-----------------------------------|
| **[Gotenberg](https://gotenberg.dev/)** | Renders DOCX / HTML / Markdown via LibreOffice with full fidelity. | Runs as a service in a container; adds a network hop and a Docker image to your pipeline. |
| **[Maroto](https://github.com/johnfercher/maroto)** | Pure Go, ergonomic API, fast. Production-ready for invoices, reports, dashboards. | Output is grid-first by design — perfect for tabular layouts, less natural for flowing prose with headings, footnotes and document-style typography. |
| **[gofpdf](https://github.com/jung-kurt/gofpdf)** / **[gopdf](https://github.com/signintech/gopdf)** | Low-level PDF primitives in pure Go. | Layout, line breaking, style inheritance, and font management are caller's problem. |

Kardec sits between these. It is **pure Go like Maroto**, **document-flavored like Gotenberg**, and **opinionated about typography**: by default the output already looks like a document instead of a form.

That is the whole goal — nothing more. If you need pixel-perfect DOCX conversion, run Gotenberg; if you need a grid of cells with deterministic placement, Maroto is faster to wire up; if you need fine PDF control, the lower-level libraries are the right tool. Kardec is for the case where you want a document that reads like one and you would rather not ship Docker to do it.

## Status

`v0.x` — experimental, breaking changes allowed. The published surface freezes at `v1.0`.

| Version | Scope |
|---------|-------|
| 0.1.0 | Paragraphs, headings, page breaks, style system, embedded fonts, Liberation Sans output |
| 0.2.0 | Multi-face embedding (real bold / italic glyphs), real tables, raster images, Markdown ingest with GFM tables, Markdown templating |
| 0.3.0 | LaTeX math subset (`\frac`, `\sqrt`, `\sum`, `\int`, greek), real lists, section header/footer + page tokens, hyperlinks + PDF outline, byte-reproducible output, `Document.Warnings()` |
| 0.4.0 | Internal links + anchors, table borders + shading, multi-section page setups, footnotes, auto table of contents, heuristic word hyphenation, Markdown image embed |
| **[0.5.0](CHANGELOG.md#050--2026-05-08)** _(current)_ | TTF font subsetting (~70 % PDF size reduction, opt-in), PDF/A-2b conformance markers (lite) |
| 0.6 | OTF/CFF font embedding (lifts the math-glyph fallback), full Knuth–Plass, Liang hyphenation patterns, OutputIntent + sRGB ICC for strict PDF/A-2 |
| 1.0 | API freeze + comprehensive examples + ≥85 % test coverage |

Full design spec: [docs/RFC-001-dsl.md](docs/RFC-001-dsl.md).
Release notes: [CHANGELOG.md](CHANGELOG.md).

## Project layout

```text
kardec/
├── doc.go, length.go, color.go, page.go    primitives (units, color, page setup)
├── style.go, run.go, block.go              the DSL surface
├── document.go                             builder + deferred-error chain
├── typography.go                           public font-registry surface
├── render/                                 orchestrator (layout + pdf + typography)
├── internal/typography/                    font registry + bundled OFL TTFs
├── internal/layout/                        line break + page break engine
├── internal/pdf/                           PDF 1.7 byte writer
├── docs/RFC-001-dsl.md                     design spec
└── examples/hello/                         smallest end-to-end example
```

## Running

```sh
go test ./...
go run ./examples/hello     # smallest end-to-end PDF
go run ./examples/report    # multi-section showcase with DefineStyle, table, header/footer
go run ./examples/markdown  # CommonMark + GFM tables → PDF
go run ./examples/invoice   # text/template + Markdown → one PDF per record
go run ./examples/image     # PNG embedded into the rendered PDF
go run ./examples/math      # LaTeX math subset (\frac, \sqrt, \sum, \int, greek)
```

## Contributing

Pull requests are welcome. The development workflow uses feature branches with `--no-ff` merges so the history of each piece of work is preserved on `main`. Granular commits are encouraged; the existing log is a good guide.

## License

MIT — see [LICENSE](LICENSE).

The bundled fonts are distributed under the SIL Open Font License 1.1; their notices live in [`internal/typography/embedded/README.md`](internal/typography/embedded/README.md).
