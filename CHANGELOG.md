# Changelog

All notable changes to Kardec are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Until
`v1.0.0`, the public API is allowed to break between minor releases.

## [Unreleased]

## [0.14.0]

### Added

- **TOC text clickable.** Every Heading placement also drops an
  auto-anchor named after the slugified title (prefix
  `kardec-toc-`); the TOC's title tokens emit a hyperlink to that
  anchor. Click "Introduction" inside the TOC and the reader jumps
  to the heading — closes the bug the v1.0 readiness recap
  flagged in "open additions" (only the sidebar outline links
  worked through v0.13). Limitation: duplicate-title docs produce
  duplicate slugs; first anchor wins on link resolution.

### Deferred to dedicated minor releases (post-v0.14)

Each of the items below is large enough (600 – 2000 LoC of
low-level binary-format work) that bundling dilutes the attention
each one needs:

- **OTF/CFF font embedding** (Type 0 + CIDFontType0 + Identity-H)
- **Encryption + permissions** (RC4-128 + AES-128 Standard Security
  Handler)
- **PDF/UA tagging** (MarkInfo + StructTreeRoot + per-block
  classification + alt text)
- **Knuth-Plass total-fit line breaker** behind a feature flag
- **SVG image embed** (vector → PDF native)
- **Watermarks** (DRAFT / CONFIDENTIAL diagonal overlay)
- **Run-level inline math** constructor (the proper inline form,
  distinct from the display-mode `Math` block)

The freeze-readiness recap treats these as the v0.14 → v1.0
backlog. v1.0 is held until at least the encryption +
PDF/UA + OTF/CFF trio ships, since those three are what
distinguish "PDF library nice for personal projects" from
"library shippable to corporate / regulated / accessibility-
required environments".

## [0.13.0]

### Added

- **ToUnicode CMap** on every embedded font. Each TTF /Font dict
  now references a `/ToUnicode` stream mapping the WinAnsi byte
  range back to Unicode codepoints, so text extraction (copy/
  paste, find-in-page, accessibility tooling) preserves fidelity
  for ligatures, smart quotes, em dashes, the euro sign, and the
  whole 0x80-0xFF range. Format follows PDF 1.7 §9.10.3 + Adobe
  Tech Note #5411 with chunked `beginbfchar` blocks. Closes the
  PDF/A-2u promise that lit up in the v1.0 readiness recap.

### Deferred

- **Encryption + permissions (RC4-128 + AES-128)** stays scheduled
  for v0.13.x as a dedicated release. The Standard Security Handler
  in PDF 7.6 needs ~600-900 LoC of careful key-derivation + per-
  stream content wrapping that benefits from undivided attention.
  Confidential-document users wait one more minor cycle; the lite
  output continues to be what every consumer reader accepts today.

## [0.12.0]

### Added

- **CI quality gates.** New jobs alongside the existing test +
  build-examples flow: a coverage gate (parses `go tool cover`,
  fails when total falls below `COVERAGE_FLOOR=60` — current actual
  ~66 %), direct `staticcheck` + `ineffassign` + `misspell` lint
  (golangci-lint binaries are still built with Go 1.24 and refuse
  to lint a 1.26 go.mod), and `govulncheck` against the full module
  graph.
- **Cross-OS test matrix.** Test job now fans out to
  `ubuntu-latest`, `macos-latest`, `windows-latest` with
  `defaults.run.shell: bash` for portability. The byte-
  reproducibility claim Kardec ships on is meaningless if only
  Linux is exercised; the matrix lets a Windows-only race surface
  in CI.
- **`cmd/reprocheck` + reproducibility CI step.** Standalone binary
  that builds a canonical document twice with a pinned creation
  date, sha256s each render, and exits 1 on mismatch. Runs in every
  matrix entry — cross-OS nondeterminism now fails the matrix
  rather than corrupting bytes downstream.
- **`.goreleaser.yml` + release workflow with cosign keyless
  signing.** Tag push (`vX.Y.Z`) triggers goreleaser to build
  `cmd/kardec` for linux / darwin / windows × amd64 / arm64,
  package each archive with the canonical docs, emit `SHA256SUMS`,
  and sign the checksums file with cosign keyless OIDC. Releases
  land as drafts for human review. Reproducibility flags
  (`-trimpath`, pinned `mod_timestamp`, static linker flags) keep
  binaries byte-identical across runs.

### Changed

- Eight unused-code findings cleared (dead types / fields / funcs
  in `internal/layout`, `internal/math`, `internal/typography`,
  and the table builder). Friend-package `// Deprecated` calls in
  `render` (`SetRenderImpl`, `FontRegistry`) annotated with
  `//lint:ignore SA1019` documenting their legitimate-caller
  status.

## [0.11.0]

### Added

- **`Document.SetTitle` / `SetAuthor` / `SetSubject` / `SetKeywords`.**
  Public knobs for the PDF /Info dictionary entries; XMP packet
  mirrors them as dc:title / dc:creator / dc:description /
  pdf:Keywords when PDFA is on. Empty fields are omitted.
- **`Document.SetICCProfile(profile, components)` + OutputIntent
  emission.** New `pdf.Document.ICCProfile` plumbing writes a
  `/GTS_PDFA1` OutputIntent referencing the embedded ICC stream
  when both PDFA is enabled and the caller supplied a profile —
  the missing piece between PDF/A "lite" and strict veraPDF-passing
  PDF/A-2b. No bundled profile yet; callers fetch sRGB IEC 61966-2.1
  from color.org / W3C.
- **Knuth-Liang hyphenation.** New `internal/hyphenation/liang.go`
  implements Liang's pattern-based algorithm; ships ~120 curated
  English (en-US) patterns covering high-frequency prefixes,
  suffixes, and consonant-pair splits. New `Register(lang, patterns)`
  hook merges caller-supplied additional patterns (e.g. the full
  hyph-en-us standard set, ~4400 patterns). The v0.4 heuristic
  stays as a structural fallback so behaviour is strictly
  equal-or-better than v0.4 — never worse.

### Deferred

- **OTF/CFF font embedding** stays scheduled for v0.11.x — the
  code path is large enough (~1.5–2.5 KLoC of low-level PDF font-
  encoding work) to justify a dedicated release rather than
  bundling into v0.11.0. Math glyphs continue to fall back to
  TTF rendering until the CFF embed lands.

## [0.10.0]

### Added

- **`Document.EnablePDFA` / `DisablePDFA` and `EnableFontSubsetting`
  / `DisableFontSubsetting`.** Idiomatic toggle pairs replacing the
  variadic-bool `PDFA(on ...bool)` and `SubsetFonts(on ...bool)`
  forms. Old methods stay as `// Deprecated` forwarders.
- **`Document.Footnote(body)` / `Document.FootnoteWith(marker, body...)`.**
  Method form replaces `kardec.Footnote(d, body)` and
  `kardec.FootnoteWithMarker(d, marker, body...)`. Function forms
  forward and are deprecated.
- **`Document.NewSection(setup PageSetup)` + `kardec.SetupOf(size, margins)`.**
  Single section constructor accepting a full PageSetup; common
  path uses `SetupOf` for size + margins with Portrait default.
- **`kardec.NewCell(runs...)`.** Replaces the plural-singular-mismatched
  `Cells(runs...)` (which produced one `Cell`).
- **`kardec.WithAlignment(a Alignment)`.** Single ColumnOption taking
  the Alignment enum, replacing four `AlignXxxCol()` helpers.
- **`kardec.TableBordersNone` / `TableBordersHorizontal` /
  `TableBordersAll`.** Type-prefixed border constants replacing the
  bare `BordersXxx` names.
- **`Document.RegisteredFamilies() []string`.** Introspection
  alternative to `FontRegistry()` that doesn't leak the internal
  `*typography.Registry`.
- **`MIGRATING.md`.** Three-column tables (Old API / New API /
  Codemod hint) covering every rename and removal in this release
  plus the v0.9 ParagraphRef return-type change.

### Removed

- **`Document.AddParagraph` and `kardec.ParagraphBuilder` alias.**
  Deprecated since v0.9.0 when `Paragraph` absorbed the builder via
  `*ParagraphRef`. Removed after one full minor cycle.
- **`Document.MathInline(src)`.** Always a placeholder for proper
  Run-level inline math; the block-flag form is gone. Use `Math(src)`
  for display math; inline math will land as a Run constructor in
  a future release.

### Deprecated

- **`Document.PDFA` / `Document.SubsetFonts`** (variadic-bool toggles)
  → use `Enable*` / `Disable*`.
- **`kardec.Footnote` / `kardec.FootnoteWithMarker`** (package-level
  functions) → use `Document.Footnote` / `Document.FootnoteWith`.
- **`Document.NewSectionWithSetup`** → use `Document.NewSection`.
- **`kardec.Cells`** → use `kardec.NewCell`.
- **`kardec.AlignLeftCol` / `AlignCenterCol` / `AlignRightCol` /
  `AlignDecimalCol`** → use `kardec.WithAlignment(a)`.
- **`kardec.BordersNone` / `BordersHorizontal` / `BordersAll`** →
  use `kardec.TableBordersXxx`.
- **`kardec.SetRenderImpl`** — render-injection seam; only the
  `kardec/render` `init()` should call it. Becomes internal at v1.0.
- **`Document.FontRegistry`** — leaks internal `*typography.Registry`.
  Use `RegisteredFamilies()` for introspection.
- **`(*Run).SetLink`** — markdown-bridge mutation seam. Use
  `kardec.Link(text, url)` for new content.

## [0.9.1]

### Changed

- **`gofmt -s` clean across the tree.** Apply Go's
  simplification pass to all 21 files that carried column-aligned
  struct fields and inline comments gofmt would normalise; rephrase
  the RFC 6266 reference in `httpx/contentDisposition` to dodge Go
  1.19+ doc-comment canonicalisation of `''` into `”`. Single
  source of truth for code formatting now matches the goreportcard
  scoring rule, so the gofmt sub-score moves from 86% to 100%.
  Pure cosmetic; zero behavioural change.

## [0.9.0]

### Added

- **Render benchmarks.** New `render/bench_test.go` ships
  `BenchmarkRenderHello` (single-paragraph baseline),
  `BenchmarkRender100PageReport` (4000-paragraph multi-page sweep),
  `BenchmarkRenderTable100Rows` (table layout cost), and
  `BenchmarkRenderMarkdownIngest` (goldmark + layout end-to-end).
  All four use `b.ReportAllocs()` so allocation regressions
  surface in `benchstat`.
- **Runnable godoc `Example` functions.** New `example_test.go`,
  `httpx/example_test.go`, and `render/example_test.go` cover the
  high-traffic entry points (`New`, `NewWithSetup`, `Paragraph`,
  `Heading`, `Table`, `AppendMarkdown`, `Cite`, `Clause`,
  `KeepTogether`, `Leader`, `SignatureBlock`, `Ref`,
  `httpx.WriteResponse`, `render.Bytes`, `render.ToFile`). Examples
  show up on `pkg.go.dev` next to the symbol they document; the
  runnable `ExampleBytes` asserts the `%PDF-1.7` magic header so
  the doc-side compile checks doubly verify that the renderer
  produces real bytes.
- **`cmd/kardec` CLI.** `go install github.com/arthurhrc/kardec/cmd/kardec`
  ships a one-binary front-end. `kardec render input.md -o out.pdf`
  renders any CommonMark+GFM source through `AppendMarkdown`;
  `kardec render -t tpl.md -d data.json -o out.pdf` executes a Go
  text/template against a JSON data file before rendering. The flag
  parser tolerates flags before *or* after the positional input, so
  the canonical form reads naturally either way.

### Changed

- **`Document.Paragraph` now returns `*ParagraphRef`.** The single
  builder unifies the bare `Paragraph(...)` chain and the deprecated
  `AddParagraph(...)` builder. The ref embeds `*Document` so chained
  doc methods (`Heading`, `Image`, etc.) keep flowing without `Done()`.
  Style overrides — `WithStyle`, `WithNamedStyle`, `Align`, `Justify`,
  `LineHeight` — mutate the just-appended paragraph in place. Callers
  that previously passed the chain return value into a function
  expecting `*Document` (e.g. `render.Bytes(doc)`) need to use
  `doc.Document` once.

### Deprecated

- **`Document.AddParagraph` and `ParagraphBuilder`.** Use
  `Document.Paragraph` and the unified `*ParagraphRef`; the symbols
  remain as a one-line alias / forwarder for source compatibility
  through the v0.x line and will be removed at v1.0.

## [0.8.0]

### Added

- **Two-column section layout.** New `PageSetup.Columns` and
  `PageSetup.ColumnGap` fields. Body content flows top-to-bottom in
  the first column, advances to the next column on overflow, and
  flushes the page only after the last column fills. `PageBreak`
  still forces a new page (not the next column). Header / footer /
  section chrome continue to span the full content width — the
  column setting affects only the body. New `kardec.NewWithSetup`
  constructor accepts a fully-populated `PageSetup` so the multi-
  column layout works from the first section without going through
  `NewSectionWithSetup`. Default gap is 12pt when not specified.
- **Decimal-point column alignment.** New `kardec.AlignDecimalCol()`
  ColumnOption + `kardec.AlignDecimal` Alignment value. Layout splits
  each cell on the first `.` and right-aligns the integer part
  against a pivot positioned at 60% of the column width; cells
  without a `.` fall back to right alignment so an integer row sits
  at the same pivot as its dotted neighbours. Currency and
  measurement columns now read at a glance.
- **Table column groups (colspan).** New `kardec.SpanCell(span, runs...)`
  builds a Cell that absorbs the next `span-1` column widths so a
  single merged header can label a group of underlying columns.
  Layout walks the row's Cell slice with a column cursor that
  advances by Span; BordersAll vertical rules emit at cell
  boundaries instead of column boundaries so spanned cells render
  as one merged region. Span values larger than the remaining
  columns are clamped at the table edge.
- **Bibliography + numeric citations.** `doc.Cite(key)` returns a Run
  carrying `[N]` plus an internal hyperlink to the matching entry;
  numbers are assigned on first reference and reused on repeats.
  `doc.Bibliography(entries...)` emits a "References" heading plus
  one paragraph per entry, sorted in citation order with uncited
  entries appended at the end. Each entry is preceded by a
  `kardec-bib-<N>` anchor so Cite's link resolves correctly.
  `BibEntry` carries Author / Title / Year / Journal / Volume /
  Pages / URL — empty fields drop out of the rendered line.
  `Document.CitedKeys()` reports citation order so callers can
  audit which entries went unreferenced.
- **`Clause(level, runs...)` and `ClauseAt(number, runs...)`.** Auto-
  numbered hierarchical clauses for legal / contract documents.
  `Clause(level, ...)` increments the per-level counter and resets
  any deeper levels (1, 1.1, 1.2, 2, 2.1, ...). `ClauseAt(label, ...)`
  bypasses the auto-counter for explicit numbering schemes. Top-level
  numbers carry a trailing dot (`1. Definitions`); deeper levels do
  not (`1.2 Term`), matching Word's convention.
- **`SignatureBlock(name, role)` and `doc.Signature(name, role)`.** A
  contract-shaped composite: thin horizontal rule, centered name
  paragraph, and an optional italic role line below. Wrapped in
  `KeepTogether` so the rule and the name never split across pages.
  Empty role is supported and skips the second line.
- **`Leader` block.** New `kardec.NewLeader(left, right)` and the
  `doc.Leader(left, right)` builder render a one-line "left ........
  right" row with a dotted fill between the two sides. Reuses the
  TOC's dot-leader emitter for visual consistency. Canonical use
  cases: CV skills bars, financial line items, contract signatories.


## [0.7.0]

### Added

- **`ImageBuilder.Caption(text)` and `CaptionRuns(runs...)`.** Attach a
  centered caption (using `StyleCaption`) below the image. When a label
  is also set, Build prepends the canonical "Figure N: " marker so the
  on-page label matches what `doc.Ref(label)` resolves to. Captioned
  images automatically wrap in a `KeepTogether` group so the figure
  and its caption never split across pages. Captionless, label-less
  images keep emitting as a bare `Image` block, preserving v0.2 layout
  behaviour.
- **Auto figure / table numbering + cross-references.** New
  `ImageBuilder.Label(name)` and `TableBuilder.Label(name)` opt a
  block into the figure / table counter; counters are independent
  and 1-based. `doc.Ref(label)` returns a Run resolving to the
  canonical "Figure 3" / "Table 2" text with an internal hyperlink
  to the auto-anchor placed before the labeled block.
  `doc.RefPage(label)` returns a Run carrying a `{{refpage:label}}`
  placeholder that the layout post-pass replaces with the page
  number on which the matching anchor landed (mirrors the TOC's
  `{{tocpage:hN}}` resolution). Unknown labels resolve to a visible
  `[?ref:<label>]` / `?` so missing references stand out without
  breaking layout.
- **`KeepTogether` block.** New `doc.KeepTogether(blocks...)` and the
  package-level `kardec.NewKeepTogether(blocks...)` group bind a slice
  of inner blocks to a single page. Canonical use: a heading and the
  first paragraph after it never split across pages. The engine uses a
  speculative-place + rollback strategy: snapshot the page state,
  attempt placement, and if a flush fired during the attempt, roll
  back, flush the original page, then re-place on the fresh one.
  Groups that exceed a single page degrade gracefully — they overflow
  naturally instead of looping.
- **Standalone `NewParagraph` / `NewHeading` constructors.** Build a
  Paragraph or Heading without going through the Document chain, the
  ergonomic prerequisite for supplying blocks to `KeepTogether`.
- **`kardec/httpx` subpackage.** New `httpx.WriteResponse(w, doc,
  filename)` and `WriteResponseInline` helpers wire a Kardec
  Document into an `http.ResponseWriter` with the right
  Content-Type, Content-Length, and RFC-6266-shaped
  Content-Disposition. Pure consumer of the public render package;
  importing it adds no behaviour to the rest of Kardec. Pass an
  empty filename to omit the disposition header.
- **`HorizontalRule` block.** A real divider primitive instead of a
  `PageBreak` masquerading as one when Markdown ingests `---`. Defaults
  to a 0.5pt gray line with 6pt of padding above and below; the public
  struct exposes `Thickness`, `Color`, and `Padding` for explicit
  overrides. Rendered as a thin `PlacedRect` so the PDF writer needs no
  changes.
- **Run decorations: `Underline`, `Strikethrough`.** New package-level
  constructors and the `WithUnderline` / `WithStrikethrough` helpers
  for stacking decorations onto an existing `Bold` / `Italic` run.
  Layout emits a thin rect per decorated token: underline below the
  baseline, strikethrough through the x-height. Thickness scales with
  point size with a 0.4pt floor so 8pt body text still reads.

  ## [0.6.0]

Skipped — scope rolled into v0.10 (API rename sweep) and v0.11
(font embedding, ICC, metadata, Liang hyphenation). See
[docs/ROADMAP_TO_V1.md](docs/ROADMAP_TO_V1.md).

## [0.5.0]

### Added

- **PDF/A-2b conformance markers (lite).** New `Document.PDFA()`
  attaches an XMP metadata stream declaring `pdfaid:part=2` and
  `pdfaid:conformance=B`, references it from `/Catalog /Metadata`,
  and writes a stable `/ID` array in the trailer derived from
  Title + Author + the pinned creation date. Two renders with the
  same `SetCreationDate` produce byte-identical output. Strict
  validators (veraPDF) still flag the missing `OutputIntent` with
  an embedded sRGB ICC profile — that lands in v0.6 — but Acrobat,
  Foxit and Chrome honor the marker as-is.
- **Font subsetting (opt-in).** `Document.SubsetFonts()` enables a
  glyf-table zero-out pass: every glyph not actually referenced by
  the document is wiped from the embedded TTF before the FontFile2
  stream is FlateDecode-compressed. Composite glyphs are recursively
  expanded so accented characters keep their components. Real
  documents drop ~70 % of their PDF size in measured tests
  (433 KB → 126 KB on a one-paragraph hello). The structural tables
  (cmap, loca, hmtx, maxp.numGlyphs) stay intact so the writer
  contract does not change. Off by default — turning it on is a
  single fluent call. Closes recommendation #5 from the strategic
  audit's "table stakes pending" list.

## [0.4.0]

### Added

- **Markdown image embed.** When a paragraph contains only a single
  inline image, `AppendMarkdown` now produces a real `Image` block.
  Callers must opt in via `Document.SetMarkdownBaseDir(dir)` so the
  bridge knows which directory relative paths resolve against;
  remote (`http://`, `https://`, `data:`) sources still warn and
  drop, keeping the bridge network-free. Previously every
  Markdown image was silently dropped with a warning.
- **Word hyphenation.** New `internal/hyphenation` package returns
  candidate break points for English words using a heuristic
  combining known prefixes (`un-`, `re-`, `pre-`, `inter-`, ...) and
  the vowel-consonant-consonant-vowel rule. The line breaker now
  tries hyphenation when a word would otherwise overflow, emitting
  a soft-hyphen split that fits the remaining line. Knuth-Liang
  pattern data is a v0.5 swap — the public surface stays the same.
- **Auto table of contents.** New
  `Document.TableOfContents(maxLevel)` reserves space for one entry
  per heading and patches the page numbers in a post-pass once the
  document is fully laid out. Each entry shows the title, a dotted
  leader, and the resolved page number; nesting indents with depth.
  `maxLevel=0` indexes every heading; pass 1 / 2 / 3 to cap depth.
- **Footnotes.** New `kardec.Footnote(doc, body)` and
  `kardec.FootnoteWithMarker(doc, marker, runs...)` helpers register
  a numbered footnote whose marker shows inline and whose body
  surfaces at the bottom of the page where the marker landed. The
  layout engine tracks markers per page and stamps a thin separator
  followed by the footnote text into the chrome area beneath the
  body. Custom markers (`*`, `†`, etc.) bypass the auto-numbered
  decimals. `Document.Footnotes()` exposes the registered set for
  introspection. Closes the audit DX recommendation that named
  footnotes as a top-3 missing primitive for "real documents".
- **Multi-section page setups.** New `Document.NewSection(size,
  margins)` / `NewSectionWithSetup(setup)` start a new section that
  receives subsequent block / header / footer calls. Each section
  carries its own PageSetup, Header and Footer, so a document may
  interleave a portrait cover, landscape charts and tighter-margin
  appendices in the same `*Document`. `layout.Page` now exposes
  orientation-applied `Width` / `Height` so the renderer emits
  per-page `/MediaBox` values that reflect each section's
  orientation; `render/destinations.go` and `render/outline.go`
  read those fields too so anchors and outline destinations resolve
  through the correct page geometry.
- **Table borders and shading.** New `TableBorderStyle`
  (`BordersNone` / `BordersHorizontal` / `BordersAll`) plus
  `TableBuilder.Borders`, `HeaderShading(color)` and
  `AlternateRowShading(color)` setters. Layout emits the lines and
  backgrounds as `PlacedItem.Rect` entries before the cell glyphs so
  the renderer paints them under text. Reuses the rectangle primitive
  already used by math fraction bars — no PDF-writer change needed.
  `examples/report` adopts the new API.
- **Internal links + anchors.** New `Document.Anchor(name)` block
  registers a named destination at the current flow position. A
  hyperlink whose URL begins with `"#"` (`kardec.Link("Chapter 2",
  "#chapter-2")`) now emits a `/GoTo /D` action targeting that
  destination instead of an external `/URI`. PDF catalogs gain a
  `/Dests` dictionary mapping each anchor name to a
  `[pageRef /XYZ null y null]` array. Closes the "TOC linking to
  sections" use case the DX audit flagged.

## [0.3.0]

### Added

- **`Document.Warnings()` for Markdown silent failures.** The
  `AppendMarkdown` bridge previously dropped unsupported nodes
  silently (autolinks, inline images, raw-HTML blocks, empty link
  destinations). It now records a human-readable advisory through a
  new `Document.warn` helper; callers retrieve them via
  `doc.Warnings() []string`. Clean Markdown produces an empty slice;
  CI pipelines that demand strict fidelity assert
  `len(doc.Warnings()) == 0`. Closes recommendation #5 from the
  strategic audit.
- **Byte-reproducible output.** `Document.SetCreationDate(t)` pins the
  `/Info /CreationDate` entry the renderer writes; two renders of the
  same Document with the same fixed timestamp now produce
  byte-identical PDFs. Without an explicit value the renderer falls
  back to `time.Now()` at emission time, matching the pre-v0.3
  behaviour. `pdf.Writer` gains a `Clock func() time.Time` seam the
  render package wires through. Closes recommendation #4 from the
  strategic audit and concretises the "deterministic Go documents"
  unique-angle play the competitive analysis flagged.
- **Hyperlinks + PDF outline (sidebar bookmarks).**
  - `kardec.Link(text, url)` produces a Run that becomes a clickable
    `/URI` annotation in the rendered PDF. Markdown source links —
    previously rendered as plain text — now flow their destination
    through the same annotation path.
  - Heading blocks build a PDF `/Outlines` tree automatically. H1
    becomes a top-level entry, H2 nests under the closest preceding
    H1, and so on. Catalog `/PageMode /UseOutlines` opens the
    sidebar by default in Acrobat / Chrome / Firefox.
  - Layout's `PlacedItem` gains a `Link` field; render coalesces
    consecutive same-target items into one rectangular annotation
    per page so multi-word links remain a single click target.
  - Closes recommendation #3 from the strategic audit.
- **Section headers and footers + page-number tokens.** New
  `Document.Header(runs...)` / `Document.Footer(runs...)` setters
  attach inline content reprinted at the top and bottom of every
  page in the current section. Token substitution at render time:
  `{{page}}` (1-based), `{{totalPages}}` (resolved in a final
  post-pass), `{{section}}` (1-based), `{{date}}` (UTC YYYY-MM-DD).
  `examples/report` adopts the chrome to demonstrate. Closes
  recommendation #2 from the strategic audit.
- **List block.** New `List` / `ListItem` types plus
  `Document.List()` / `Document.OrderedList()` builders carrying
  fluent `Item(...)` / `Nested(runs, children...)` / `Build()` calls
  and a top-level `SubList` helper for rich nested construction.
  Layout indents nested levels and rotates the unordered marker
  (• / ◦ / ▪) by depth so the level is visually obvious; ordered
  lists carry decimal numerals. `AppendMarkdown` now produces real
  `List` blocks instead of flattening to bullet-prefixed paragraphs
  — the v0.2 limitation explicitly called out in the prior CHANGELOG
  entry is gone.
- **Rectangle primitive in the PDF writer.** `pdf.RectDraw` plus
  content-stream `re`/`f` ops. Layout's `PlacedItem` gains a `Rect`
  field carrying width/thickness/color; the renderer translates it
  to a filled rectangle in PDF user space. Wired through math layout
  so fraction bars and square-root overlines now appear in the
  output PDF (previously a documented limitation in the v0.3 first
  cut).
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

- **Math font embedding deferred.** Latin Modern Math ships as
  OpenType/CFF (sfnt header `OTTO`); the current writer only embeds
  TrueType (`0x00010000`). v0.3 routes math glyphs to the default body
  font (Liberation Sans) so PDFs remain valid. Greek letters render
  through Liberation Sans's coverage; large math operators (`∑`, `∫`,
  `∏`, `√`) fall back to the default font's glyph table. CFF support
  lands in v0.3.x.

## [0.2.0]

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

## [0.1.0]

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

[Unreleased]: https://github.com/arthurhrc/kardec/compare/v0.14.0...HEAD
[0.14.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.14.0
[0.13.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.13.0
[0.12.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.12.0
[0.11.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.11.0
[0.10.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.10.0
[0.9.1]: https://github.com/arthurhrc/kardec/releases/tag/v0.9.1
[0.9.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.9.0
[0.8.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.8.0
[0.7.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.7.0
[0.6.0]: #060
[0.5.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.5.0
[0.4.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.4.0
[0.3.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.3.0
[0.2.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.2.0
[0.1.0]: https://github.com/arthurhrc/kardec/releases/tag/v0.1.0
