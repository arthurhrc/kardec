# Changelog

All notable changes to Kardec are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Until
`v1.0.0`, the public API is allowed to break between minor releases.

## [Unreleased]

### Added

- **LaTeX math subset.** New `Math` block plus `Document.Math(src)` and
  `Document.MathInline(src)`. Source is parsed from the LaTeX subset
  documented in `internal/math` (greek lowercase + uppercase, fractions
  via `\frac` / `\dfrac`, square roots, nth roots, sub/superscripts,
  big operators `\sum` / `\int` / `\prod` with optional limits, named
  operators / relations / arrows). Layout follows TeXbook conventions:
  display style for the standalone block, inline style with side
  scripts on big operators when `MathInline` is used.
- **`internal/math`** — hand-rolled lexer + recursive-descent parser
  producing a sealed AST (`Atom`, `Op`, `Number`, `Identifier`, `Group`,
  `Frac`, `Sqrt`, `NthRoot`, `SubSup`, `BigOp`) plus a canonical symbol
  table mapping LaTeX commands to Unicode runes and categories.
- **`internal/mathlayout`** — TeXbook-style box layout: atom/group/op
  spacing, sub/superscript scaling (70 % size, 0.30 × down / 0.45 × up),
  fraction (numerator and denominator with rule between), square-root
  with overline, big-op inline / display modes.
- **Math typography subsystem.** New `typography.MathFont` interface
  (`GlyphFor` / `Measure` / `AscentDescent`) plus a Latin Modern Math
  implementation served via `typography.LatinModernMath`. The OTF
  comes from `github.com/go-fonts/latin-modern/lmmath` — no shadow
  copy. Public entry point: `(*Document).MathFont() typography.MathFont`,
  lazy-loaded and memoised per Document.
- **`internal/mathadapter`** — bridges the parser's AST onto the layout
  engine's interfaces and `typography.MathFont` onto `mathlayout.Font`,
  isolating the seam between the three independently-built tracks.
- **`examples/math`** — five display equations plus a greek-letters
  formula.

### Limitations (intentional, lifted later)

- **Fraction bars and square-root overlines are not yet rendered.**
  The math layout engine produces `Box.Rules` for them, but the PDF
  writer has no rectangle primitive yet — adding one is queued for
  v0.3.x. Until then frac / sqrt show the glyphs without the bar.
- **Math font embedding deferred.** Latin Modern Math ships as
  OpenType/CFF (sfnt header `OTTO`); the current writer only embeds
  TrueType (`0x00010000`). v0.3 routes math glyphs to the default body
  font (Liberation Sans) so PDFs remain valid. Greek letters render
  through Liberation Sans's coverage; large math operators (`∑`, `∫`,
  `∏`, `√`) fall back to the default font's glyph table. CFF support
  lands in v0.3.x.

## [0.2.0] — 2026-05-07

### Added

- **Image embedding.** New `Image` block + `ImageBuilder` fluent API
  (`doc.Image(bytes).Width(...).Center().Build()`, plus `ImageFile(path)`
  for the common case). JPEG payloads pass through into the PDF via
  `/Filter /DCTDecode` — no decode, no recompression. PNG payloads are
  decoded with stdlib `image/png`, alpha is composited over white, and
  the result is written as packed 8-bit RGB through `/Filter /FlateDecode`.
  Layout chooses target dimensions by combining `Width()` / `Height()` /
  natural aspect ratio, scales down to fit the available width, and
  paginates when the image does not fit on the remaining page.
- **`examples/image`** — generates a gradient PNG in-process and embeds
  it into the rendered PDF, demonstrating the pipeline end-to-end.
- **Markdown table support.** GFM-style pipe tables in source passed to
  `AppendMarkdown` now produce real `Table` blocks (instead of being
  rendered as inline text). Header cells become bold runs, the column
  alignment syntax (`:---`, `:---:`, `---:`) is honoured, and the
  resulting table opts in to `RepeatHeader` so multi-page tables keep
  their column titles visible. Powered by the upstream `extension.Table`
  parser; the bridge lives in `markdown.go`.
- **Real tables.** New `Table` block, `TableBuilder` fluent API
  (`doc.Table().Columns(...).Row(...).Build()`), `Col` / `Width` /
  `AlignLeftCol` / `AlignCenterCol` / `AlignRightCol` column-option
  helpers, and a `Cells` helper for rich-content cells. Layout supports
  fractional / fixed / auto column widths, multi-line cells, page split
  with optional `RepeatHeader`. Replaces the v0.1 `"TODO table"` stub.
- **Multi-face font embedding.** PlacedItems now flow with their resolved
  family (Style.Family), weight (bold flag) and italic flag through a new
  `*measureAdapter` that the render package reads back when assembling
  the embedded-font table. Bold and italic glyphs land in the PDF as the
  actual TrueType faces from the registry (e.g. LiberationSans-Bold,
  LiberationSerif-Italic), not flat regulars. Only faces actually
  referenced by the document are embedded; the rest of the registry is
  left out so size growth stays proportional to what is used.
- **`typography.Registry.Faces`** + new `FaceRecord` type — exposes the
  registered faces (with their TTF bytes) so the renderer can embed them
  in the PDF without re-reading the bundled FS.
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

### Changed

- `internal/layout` linebreaker now passes `Style.Family` and the
  block-style bold/italic flags through to `FontProvider.Resolve`,
  rather than always asking for the empty-string family.

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

[0.2.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.2.0
[0.1.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.1.0
