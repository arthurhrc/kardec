# Changelog

All notable changes to Kardec are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Until
`v1.0.0`, the public API is allowed to break between minor releases.

## [Unreleased]

### Added

- **`Document.AppendMarkdown`** — feeds raw CommonMark to a document and
  appends the resulting blocks to the current section. Supports headings
  (1–6), paragraphs, bold / italic / bold-italic emphasis, inline code,
  unordered and ordered lists (flattened to bullets in v0.1), thematic
  breaks (mapped to PageBreak), code blocks (rendered with StyleCode),
  blockquotes (rendered with StyleQuote). Backed by `goldmark` v1.8.
- **`Template`** — `kardec.NewTemplate(src, opts...)` compiles a Markdown
  template with `text/template` placeholders. `Template.Render(data)`
  produces a fresh `*Document`. Options: `WithPageSize`, `WithMargins`,
  `WithName`. `MustNewTemplate` provides a panicking variant for init
  blocks. Useful for invoice / certificate / report-per-record flows.
- **`examples/markdown`** — end-to-end CommonMark → PDF demo.
- **`examples/invoice`** — three invoices generated from a single
  Markdown template, demonstrating per-record templating.

## [0.1.0] — 2026-05-07

The first usable release. Kardec produces real PDFs (`%PDF-1.7`) from a
fluent Go DSL with no container, no LibreOffice and no system-font
dependency.

### Added

- **DSL primitives** — `Length` (Pt/Mm/Cm/In), `Color` (`HexColor`, named
  primaries, sRGB), `PageSize` presets (A3 / A4 / A5 / Letter / Legal),
  `Margins` (`MarginsNarrow` / `Normal` / `Wide`).
- **Document builder** — `kardec.New(size, margins).Heading(...).Paragraph(...)`
  with deferred-error chain (`Document.Err`).
- **Style system** — `Style` value type, `Weight` enum, 16 built-in named
  styles (Default, H1..H6, Caption, Quote, Code, TableHeader, TableCell,
  Footer, Header, ListItem, Link), `DefineStyle` / `ResolveStyle` /
  `ResolveBlockStyle` with full inheritance chain and cycle detection.
- **Style-aware builders** — `AddParagraph` / `AddHeading` returning
  fluent builders with `WithStyle`, `WithNamedStyle`, `Justify`,
  `LineHeight`, `Done` rejoining the document chain.
- **Typography** — `internal/typography.Registry` backed by
  `tdewolff/canvas` for OpenType shaping. Bundled fonts via `embed.FS`:
  Liberation Sans, Liberation Serif, Carlito, JetBrains Mono — Regular,
  Bold, Italic, BoldItalic for each (16 faces, ~7 MB, OFL).
- **Layout engine** — `internal/layout` with greedy line breaking,
  page-break logic honoring `Spacer`, `PageBreak`, `KeepWithNext`,
  per-block style resolution via `Document.ResolveBlockStyle`.
- **PDF writer** — `internal/pdf` emits PDF 1.7 (FlateDecode-compressed
  content streams, FontFile2 TrueType embedding, WinAnsiEncoding, xref
  table + trailer).
- **Render orchestrator** — `github.com/arthurhrc/kardec/render` wires
  layout + typography + PDF together. Importing it (blank or otherwise)
  installs `Document.Render` / `RenderTo` / `Bytes`. Public functions
  `render.ToFile`, `render.ToWriter`, `render.Bytes` are also available.
- **Examples** — `examples/hello/` produces a real PDF (~216 KB).
- **CI** — GitHub Actions running `go vet`, `go test -race`, `tidy` check
  and `go build` of every example on `ubuntu-latest`.

### Limitations (intentional, lifted later)

- **Single-font embedding.** Every text run renders in Liberation Sans
  Regular even when the resolved style asks for bold or italic.
  Measurement is correct (line breaks are computed against the right
  face), only the embedded glyphs are uniform. Promoted to the
  `0.2.0` roadmap.
- **Stub blocks.** `Table` and `Image` block types accept content but
  emit a `"TODO ..."` placeholder during rendering. Real layout for
  both is `0.2.0`.
- **No hyphenation, no full Knuth–Plass.** The line breaker is greedy.
  Justified paragraphs distribute extra inter-word space; the last
  line falls back to AlignLeft.
- **Single section per document.** Multi-section page-setup changes
  are queued for `0.2.0`.

### Notes

- Module path: `github.com/arthurhrc/kardec`.
- Go: 1.22+ (the project tracks `go.mod`'s declared toolchain version).
- License: MIT for the source, OFL 1.1 for the bundled TTFs.

[0.1.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.1.0
