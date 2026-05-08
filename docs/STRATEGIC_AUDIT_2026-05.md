# Strategic audit — May 2026

Snapshot consolidating three independent agent audits run against Kardec
v0.2.0 + v0.3-in-flight: a product strategist, a developer-experience
auditor, and a competitive analyst. Each ran read-only against the
public surface; this doc keeps the high-signal items so the next
release planning session has them at hand.

## Where the three audits agreed

1. **The credibility gap is missing primitives, not missing math.** All
   three flagged the same hole: the RFC and `style.go` reference
   `StyleListItem`, `StyleHeader`, `StyleFooter`, `StyleLink` — but
   there is no `List` block, no `Header` / `Footer` API on `Section`,
   no link runs. A "document-feel" library that cannot do bullet
   lists, page numbers, or clickable links is positioned for
   adoption-blocking gaps.

2. **Math is a flashy distraction from the core mission.** Strategy
   and competitive both said the same: the math subset (v0.3) is
   visually impressive but does not advance the README's promise of
   *document quality*. The strategy agent recommends pausing math
   after v0.3 ships and going back to core primitives.

3. **The README oversells one axis (HTML/CSS workflows) and undersells
   another (deterministic, byte-reproducible Go output).** The
   competitive agent argued the real differentiator is not "between
   Gotenberg and Maroto" but "deterministic, hash-stable PDFs from a
   typed Go AST" — an angle no Go competitor owns today.

## Adoption-blocking gaps (ranked)

The strategy and DX audits independently surfaced this priority list:

1. **`List` block** with ordered / unordered / nested. Today
   `markdown.go:19` flattens to `"• "`-prefixed paragraphs, which is
   honest but lossy. **Effort: small.**
2. **`Section.Header(...)` / `Section.Footer(...)` + page-number
   tokens** (`{{page}}`, `{{totalPages}}`, `{{section}}`, `{{date}}`).
   The PageSetup model already has the slot. **Effort: medium.**
3. **Hyperlink runs + heading-derived PDF outline (bookmarks).** One
   feature, two payoffs: clickable links in the body and a sidebar
   TOC that every PDF reader exposes. **Effort: medium.**
4. **Table borders / shading / row-and-column spans.** Maroto wins
   today on this axis; closing it makes the niche claim stronger.
   **Effort: medium-large.**
5. **TTF subsetting.** The default ~7 MB embedded payload undermines
   the "pure Go and small" pitch; subsetting trims this an order of
   magnitude. **Effort: medium.**

## DX fixes the README and godoc need

- `doc.go:11` snippet does not compile (`Heading(1, "string")` —
  signature is `...Run`). **Fix: small.**
- The blank-import-wires-Render contract is buried; the example
  `examples/hello/main.go:21` carries it but the README's snippet at
  `README.md:5-22` does not call it out. New users hit
  `ErrRendererUnregistered` and bounce. **Fix: small** — README
  callout box.
- Markdown silent failures (links dropped, images dropped, footnotes
  dropped, autolinks dropped). **Fix: medium** — add
  `Document.Warnings() []string` and document the supported subset.
- Style precedence is correct in code but undocumented in `doc.go`.
  Two reasonable users would disagree on
  `WithStyle(Style{Alignment:AlignRight}).Justify()`. **Fix: medium**
  — `# Style precedence` H2 in `doc.go` with a worked example.
- "Custom fonts" section absent from the README — `RegisterFont`
  exists in `typography.go:14-25` but is invisible. **Fix: small.**
- "Error handling" rationale absent from `doc.go`. **Fix: small.**

## Positioning honesty (competitive)

- The "between Gotenberg and Maroto" framing is *axiomatically*
  honest but the *real* fork in the road for most teams is `chromedp`
  or WeasyPrint, because HTML/CSS is more familiar than a new DSL.
  Kardec wins where airgapped / regulated / high-volume per-record
  matter; loses where designer-supplied HTML is the input.
- README claims a "real layout engine" — credible only after
  Knuth–Plass + hyphenation land. Today the line breaker is greedy.
- README's niche claim needs three proof points it currently lacks:
  (a) screenshots vs. Maroto / gofpdf for the same source;
  (b) PDF-size + render-time benchmarks;
  (c) a public golden-document corpus visually diffed against
  LibreOffice (RFC §17 already names this as the right CI artefact).

## The unique-angle play

The competitive audit's most useful new framing: Kardec's combination
of bundled fonts + init-hook orchestrator + Go-typed AST points at
**deterministic, byte-reproducible PDFs** — same input, same bytes, no
font-fallback drift, no Chromium nondeterminism. Typst owns this
conceptually in Rust; nothing owns it in Go. The action item is concrete:

- Make `Info.CreationDate` injectable through a clock seam (currently
  `time.Now()` in `internal/pdf/writer.go:160`).
- Publish a reproducibility-hash test in CI.
- Add a "deterministic invoice" scenario to the README — "same input
  → same bytes" is the headline benchmark.

## Ecosystem hooks the audits all agreed on

- `kardec/httpx` helper:
  `WriteResponse(w http.ResponseWriter, doc *Document, filename string)`
  — three lines, every Go shop's first handler.
- `cmd/kardec` CLI: `kardec render input.md -o out.pdf`
  + `kardec render -t tpl.md -d data.json` — wraps `Template`, makes
  the library demoable without writing Go.
- `kardec/goldmarkx` extension: expose Kardec as a goldmark renderer
  so anyone using goldmark elsewhere can target PDF.
- `kardec/chart` adapter accepting `gonum/plot` or `go-echarts`
  output as `Image` blocks — RFC §2 already names it as a v1.x non-goal
  worth promoting.

## Recommended next moves (synthesised)

In order of expected adoption-impact-per-line-of-code:

1. **`feat/list-block`** — closes the largest credibility gap.
2. **`feat/section-headers-footers`** with page-number tokens.
3. **`feat/hyperlinks-and-outline`** (one PR, two visible payoffs).
4. **`feat/deterministic-output`** — clock seam + repro test + README
   "same input, same bytes" headline.
5. **`feat/markdown-warnings`** — `Document.Warnings` + supported
   subset table in `markdown.go` godoc.
6. Pause math after v0.3 cuts; revisit only after #1–#3 ship.
7. **`feat/ttf-subsetting`** — table-stakes payload trim.

The first three would land a real-world invoice example in the README
that alone closes the visible gap between RFC promises and exposed
public surface — and gives the niche claim its missing screenshot.
