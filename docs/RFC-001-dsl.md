# RFC-001 — Kardec DSL specification

| | |
|--|--|
| Status | Draft |
| Author | Arthur Carvalho |
| Created | 2026-05-07 |
| Supersedes | — |
| Target version | v1.0 |

## 1. Goal

A Go library that produces **document-like PDFs** (Word feel, not report grid) via a fluent, style-driven DSL — with **no external runtime dependency**: no container, no LibreOffice, no system fonts required.

## 2. Non-goals

| | Why it's out | Could be promoted later? |
|--|--|--|
| Convert existing `.docx` → PDF | Requires reimplementing LibreOffice's layout engine (~10M lines C++); years of work | No — fundamentally wrong scope for a generation DSL |
| Edit existing PDFs | Inverse problem (parsing fixed-layout content streams + inferring semantics); different domain | No — separate library territory |
| LaTeX-quality math typesetting | TeX is itself a complete layout engine | **Yes — subset MathML targeted for v1.x** |
| Office SmartArt / native ChartML / DrawingML | Each is its own substantial Office sub-spec | **Yes for charts, via raster image rendering** (e.g. wrap `gonum/plot`, `go-echarts`) targeted for v1.x |
| Write `.docx` files | Mixes concerns; covered by `unioffice` already | No — different project |

## 3. Design principles

1. **Style-first, not grid-first** — layout flows from text, not from a matrix. Distinguishes Kardec from Maroto.
2. **Fluent imperative builder** — idiomatic Go, like `bytes.Buffer` / `strings.Builder` / `http.ServeMux`.
3. **Composable styles with inheritance** — base styles → named styles → instance overrides.
4. **Beautiful defaults** — zero configuration must already produce a tasteful document.
5. **Embedded fonts** — three or four open, metric-compatible families ship with the binary; users can register more via `RegisterFont`.
6. **No global state** — everything passes through `*Document` or `*Section`. No singletons, no init magic.
7. **Explicit, deferred errors** — chain methods accumulate errors into a single `*Document.Err()`; only `Render` returns the failure if any.
8. **English everywhere** — code, comments, docs, error messages. Universal library distribution.

## 4. API surface (sketch)

```go
import "github.com/arthurhrc/kardec"

doc := kardec.New(
    kardec.PageA4,
    kardec.MarginsNormal,           // 2.5cm uniform; presets: Narrow / Normal / Wide
)

// Define a named style; inherits from "Default" unless ParentStyle is set
doc.DefineStyle("h1", kardec.Style{
    Family:      kardec.FontLiberationSans,
    Size:        24,
    Weight:      kardec.Bold,
    Color:       kardec.HexColor("#2E74B5"),
    SpaceBefore: kardec.Pt(16),
    SpaceAfter:  kardec.Pt(8),
    KeepWithNext: true,
})

// Content
doc.Heading(1, "Monthly Report")

doc.Paragraph(
    kardec.Text("Sales grew "),
    kardec.Bold("12%"),
    kardec.Text(" this quarter — vs. "),
    kardec.Italic("8%"),
    kardec.Text(" last year."),
)

doc.Table().
    Columns(
        kardec.Col("Month",   kardec.Width(0.4)),
        kardec.Col("Revenue", kardec.Width(0.3), kardec.AlignRight),
        kardec.Col("Δ",       kardec.Width(0.3), kardec.AlignRight),
    ).
    Row("January",  "R$ 12,000", "+5%").
    Row("February", "R$ 14,500", "+12%").
    Build()                           // validates; chain-error captured via doc.Err()

doc.PageBreak()
doc.Heading(2, "Analysis")
doc.Paragraph(/* long body */).Justify().LineHeight(1.4)

// Output — single I/O boundary
if err := doc.Render("report.pdf"); err != nil {
    log.Fatal(err)
}
// alternatives:
//   doc.RenderTo(io.Writer) error
//   doc.Bytes() ([]byte, error)
```

### 4.1 Markdown input alternative

```go
doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
doc.AppendMarkdown(`
# Monthly Report

Sales grew **12%** this quarter — vs. *8%* last year.

| Month    | Revenue   | Δ    |
|----------|-----------|------|
| January  | R$ 12,000 | +5%  |
| February | R$ 14,500 | +12% |
`)
doc.Render("report.pdf")
```

CommonMark + GFM tables. Kardec maps Markdown nodes onto the same internal block model as the DSL — both paths share the layout engine.

### 4.2 Templating

```go
tpl := kardec.NewTemplate(kardec.MustReadFile("invoice.tpl.md"))
for _, customer := range customers {
    doc, err := tpl.Render(customer)
    if err != nil { return err }
    doc.RenderToFile(fmt.Sprintf("invoice-%d.pdf", customer.ID))
}
```

Built on `text/template` (stdlib). Template files contain Markdown with `{{.Field}}` placeholders.

## 5. Document model

```
Document
  └── Section[]                       page setup may change between sections
        ├── PageSetup (size, margins, orientation, columns)
        ├── Header / Footer (optional, with page-number tokens)
        └── Block[]                   ordered flow
              ├── Paragraph(Run[])
              ├── Heading(level, Run[])
              ├── Table(Column[], Row[])
              ├── List(Item[], ordered/unordered, nested)
              ├── Image(src, width, height, align)
              ├── PageBreak
              └── Spacer(height)

Run                                   inline text fragment
  ├── text         string
  ├── styleOverride Style             merges over parent
  └── features     OpenTypeFeatures   ligatures, kerning toggle, etc.
```

A `Run` carries text plus inline style overrides. A `Block` is a positioned chunk. The layout engine walks `Section.Blocks`, asks each for measured dimensions, and assigns it to a page.

## 6. Style system

- **Built-in named styles**: `Default`, `H1`–`H6`, `Caption`, `Quote`, `Code`, `TableHeader`, `TableCell`, `Footer`, `Header`, `ListItem`, `Link`.
- **Inheritance chain** (from highest priority to lowest):
  1. Run-level inline overrides
  2. Block-level explicit `WithStyle(...)`
  3. Named style (`doc.DefineStyle("h1", ...)`)
  4. Parent block style
  5. Document `Default` style
- `Style` is a value type (not pointer); cloning is cheap and functional-update friendly.

## 7. Typography

### 7.1 Bundled fonts (via `embed.FS`)

| Family | Role | License | Substitutes |
|--|--|--|--|
| Liberation Sans | Default sans | OFL | Arial-metric-compatible |
| Carlito | Word-style sans | OFL | Calibri-metric-compatible |
| Liberation Serif | Default serif | OFL | Times-metric-compatible |
| JetBrains Mono | Code / monospace | OFL | — |

Estimated size cost: ~3–5 MB total. Significantly under the 200 MB embedded-LibreOffice path discussed earlier.

### 7.2 Custom fonts

```go
doc.RegisterFont("Inter", interTTFBytes)
doc.SetDefaultFont("Inter")
```

Users supply their own font bytes (`os.ReadFile`, `embed.FS`, network) — Kardec never reaches into system font directories.

### 7.3 Shaping

Backed by `github.com/tdewolff/canvas` for OpenType shaping (ligatures, kerning, contextual alternates). Fallback chain for missing glyphs: declared font → declared fallback → bundled default.

## 8. Layout engine (contract, not implementation)

- **Line breaking**: simplified Knuth–Plass; hyphenation deferred to v1.x.
- **Page breaks**: natural flow, with `KeepWithNext`, `KeepTogether`, `PageBreakBefore` honored.
- **Justification**: word-spacing first, then letter-spacing within tolerance.
- **Tables**: split across pages with optional repeated header row; cells exceeding remaining page space trigger an early break.
- **Images**: inline (in a Run) or block (centered/aligned). Floating images with text wrap deferred to v1.x.
- **Lists**: nested up to 6 levels; ordered/unordered; numbering format configurable per level.

## 9. Page setup

| | |
|--|--|
| Sizes | `A3`, `A4`, `A5`, `Letter`, `Legal`, `kardec.Custom(width, height Length)` |
| Orientation | `Portrait`, `Landscape` |
| Columns | 1 (v1.0); 2–3 (v1.x) |
| Margins | `MarginsNarrow` (1.27 cm), `MarginsNormal` (2.54 cm), `MarginsWide` (5.08 cm), or per-side |
| Headers/footers | Plain text or `Run`-based, with tokens: `{{page}}`, `{{totalPages}}`, `{{section}}`, `{{date}}` |

## 10. Output

```go
Render(path string) error              // most common
RenderTo(w io.Writer) error
Bytes() ([]byte, error)
```

PDF target: **1.7** (`/MediaBox` per page, font subsets embedded, ZIP-compressed content streams, ICC-based RGB color profile).

## 11. Error handling

- Builder methods accumulate errors; `Document.Err() error` returns the first error encountered.
- `Render` returns the deferred error before attempting I/O.
- Distinct error types: `*kardec.ErrUnknownStyle`, `*kardec.ErrFontNotRegistered`, `*kardec.ErrInvalidTable`, `*kardec.ErrLayoutOverflow`, etc., to enable `errors.As`.

## 12. Concurrency

A `*Document` is **not safe for concurrent use** by multiple goroutines, in line with `bytes.Buffer` and `strings.Builder`. Different `*Document` instances may be used concurrently. This is documented at the package level.

## 13. Versioning

- Semantic versioning, strict.
- `v0.x` may break the API freely.
- `v1.0` freezes the public API surface listed in this RFC. Additions are minor; removals require a major.

## 14. Distribution

| Module | `github.com/arthurhrc/kardec` |
| Go version | 1.22+ (uses `range over int`, `slices`/`maps` stdlib) |
| External dependencies | `github.com/tdewolff/canvas` (typography). Possibly `github.com/yuin/goldmark` for Markdown parsing. |
| Approximate binary impact | ~5 MB of embedded fonts + a few hundred KB of code |

## 15. Open questions

- **OQ-1**: Color management — embed a single sRGB ICC profile, or expose as configurable? Default sRGB seems sufficient for v1.0.
- **OQ-2**: Hyphenation in v1.x — `golang.org/x/text/language` has no hyphenation tables. Adopt `github.com/speedata/hyphenation` (CC0 patterns) or skip for v1.x as well?
- **OQ-3**: Should `AppendMarkdown` accept an optional style mapping (`map[string]string`, e.g. `"h1" -> "MyH1Style"`) so the same Markdown can be re-themed?
- **OQ-4**: PDF/A compliance for archival use cases — significant additional spec work; defer to v2 unless strongly required.

## 16. Roadmap snapshot

| Version | Scope |
|--|--|
| v0.1 | Core DSL, paragraphs, headings, basic tables, single section, A4/Letter, embedded fonts |
| v0.2 | Lists, images, multiple sections, headers/footers, page-number tokens |
| v0.3 | Markdown input (`AppendMarkdown`), templating |
| v0.4 | Style overrides, alignment options, justification, hyphenation experiments |
| v1.0 | API freeze of everything above; full doc + 5+ realistic examples; >85% test coverage |
| v1.1 | Subset MathML rendering |
| v1.2 | Charts via raster image rendering helper (`kardec.ChartImage(...)`) |
| v1.3 | Floating images with text wrap; multi-column sections |
| v2.0 | TBD — possibly PDF/A, full Knuth–Plass, or DOCX-export companion |

## 17. Implementation strategy

The work splits cleanly into four parallel tracks. Each track has a well-defined interface, allowing parallel development by specialist agents:

| Agent | Owns | Interfaces with |
|--|--|--|
| **Layout** | Line breaking, page breaks, table row-split logic | Typography (for measurement), DSL (consumes blocks) |
| **Typography** | Font registry, OpenType shaping, glyph metrics | Layout (provides measurement), Renderer (provides glyph runs) |
| **DSL / API** | Public surface, builder pattern, validation | Layout (emits blocks), Markdown parser (emits blocks) |
| **Renderer** | PDF 1.7 writer, content streams, font embedding | Layout (consumes positioned glyph runs) |

Cross-cutting concerns: testing corpus (a set of golden documents rendered by both Kardec and LibreOffice for visual diffing) and CI (lint, `go test`, golden image comparison).
