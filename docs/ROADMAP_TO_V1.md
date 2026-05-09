# Roadmap to v1.0

Synthesis of three independent audits run against the v0.5.0 surface:
API completeness, README/docs voice, release engineering. The plan
below is what stands between today and a v1.0 freeze.

## Why "Kardec"

Allan Kardec (1804-1869) was a French educator who took a body of
scattered oral traditions and codified them into a structured written
doctrine. Whatever one makes of the subject matter, the work itself
was an act of structuring: turning loose, transient material into a
coherent, portable record.

The library does the equivalent for documents. A program assembles a
flowing structure (headings, paragraphs, tables, citations) through a
fluent Go API, and Kardec freezes that structure into a portable PDF
that opens identically on every reader. The name is a nod to the act
of codification, nothing more.

## Version progression

### v0.6 — close the v0.5 deferrals and lock the API shape

The two strands that need to land before v0.7 introduces frozen
primitives: finishing what v0.5 deferred, and renaming everything
that would otherwise be the #1 reviewer complaint on `pkg.go.dev`
once frozen.

**Deferrals from v0.5:**

- **OTF / CFF font embedding** via Type 0 + CIDFontType0 + Identity-H.
  Lifts the math-glyph fallback so Latin Modern Math actually renders
  `∑`, `∫`, `√`. Adds `FontFile3` with `/Subtype /CIDFontType0C`.
- **OutputIntent + sRGB ICC profile** so `Document.PDFA()` becomes
  strict PDF/A-2b (passes veraPDF). Bundle a 3 KB sRGB IEC 61966-2.1
  profile next to the embedded fonts.
- **Document metadata setters**: `SetTitle`, `SetAuthor`, `SetSubject`,
  `SetKeywords`. Already partly land via `/Info` but no public knobs.
- **Liang hyphenation patterns** (English first). Drop-in replacement
  for the v0.4 heuristic; same `BreakPoints` surface.
- **Knuth-Plass total-fit line breaker** behind a feature flag while
  the greedy default keeps shipping.

**Idiomatic API sweep:**

- `Document.PDFA(on ...bool)` → `EnablePDFA()` / `DisablePDFA()`.
  Variadic-bool is unidiomatic Go; will be the #1 reviewer complaint
  on `pkg.go.dev`.
- `Document.SubsetFonts(on ...bool)` → `EnableFontSubsetting()` /
  `DisableFontSubsetting()`. Same fix.
- `kardec.Footnote(d, body)` and `FootnoteWithMarker(d, marker, runs...)`
  → methods on `*Document`: `doc.Footnote(body)`, `doc.FootnoteWith(marker, runs...)`.
  Passing the document into a Run constructor is a smell.
- `Run.SetLink(url)` → package-private. Public Runs should stay
  immutable; users get hyperlinks via `kardec.Link(text, url)`.
- `MathInline(src)` → remove. Always was a placeholder; inline math
  needs a Run-level constructor whenever it ships, not a block flag.
- `NewSectionWithSetup(setup)` → fold into `NewSection(setup PageSetup)`;
  add `kardec.SetupOf(size, margins)` helper for the common path.
- `kardec.Cells(runs...)` → rename `NewCell` (avoids the plural-singular
  mismatch with `RowCells`).
- `AlignLeftCol` / `AlignCenterCol` / `AlignRightCol` → drop the `Col`
  suffix; they only exist as `ColumnOption` and the type already
  namespaces them.
- `BordersHorizontal` / `BordersAll` / `BordersNone` → `TableBordersNone`
  / `TableBordersHorizontal` / `TableBordersAll` so the global namespace
  carries the type prefix.
- `Document.SetRenderImpl` → unexported (move to `internal_access.go`).
  Public seam for one internal consumer; freezing it locks us to a
  renderer-injection ABI we do not want to support.
- `Document.FontRegistry()` → unexported. Leaks `*typography.Registry`
  (an internal type) through the public API. Replace with
  `Document.RegisteredFamilies() []string` if introspection is needed.

Every renamed identifier ships a `// Deprecated:` doc-comment pointing
at the new name. `gopls` surfaces those automatically.

### v0.7 — cross-references and structural primitives

- **Auto figure / table numbering** + `doc.Ref(label)` resolving to
  "Figure 3" plus the destination page. Single biggest hole; without
  it Kardec loses to LaTeX/Typst the moment a buyer says "academic"
  or "regulated".
- **`KeepTogether(blocks...)` group** so a heading-and-its-paragraph
  pair never splits across pages. Small.
- **`HorizontalRule()` block** (real rule, not a `PageBreak`
  masquerading as one from `---`). Small.
- **`Image.WithCaption(...)`** travelling as a keep-together unit
  with auto figure numbering. Small.
- **`Run` decorations**: underline, strikethrough, letter-spacing.
  Small.
- **`kardec/httpx` subpackage** with `WriteResponse(w, doc, filename)`.
  Three-line ergonomic helper every Go shop pastes into their first
  PDF endpoint.

### v0.8 — academic-grade primitives

- **Bibliography + citation**: `doc.Cite("Knuth1984")` and
  `doc.Bibliography(entries...)`. Medium.
- **Leader-dots primitive**: `Leader(left, right)` for "Skills........80%"
  layouts. Reused by the existing TOC. Small.
- **`SignatureBlock(name, role)`** for contracts.
- **Numbered-clause helper** (`Clause("1.2.3", text)`) with auto
  numbering.
- **Column groups + merged header cells (colspan)** in tables.
- **Decimal-point alignment** for numeric columns.
- **Two-column section layout** (CV / résumé use case).

### v0.9 — release-candidate polish

- **Unify `Paragraph` and `AddParagraph`** behind one returned-ref
  builder. Today every example uses the bare path; the builder is
  dead weight that locks two ways to do everything. Cannot be
  removed once frozen at v1.0.
- **`cmd/kardec` CLI**: `kardec render input.md -o out.pdf` and
  `kardec render -t tpl.md -d data.json`. Demoable without writing
  Go code.
- **Public golden-document corpus**: a separate repo with reference
  PDFs visually diffed against LibreOffice in CI (RFC §17 already
  names this).
- **`kardec/chart`** adapter accepting `gonum/plot` / `go-echarts`
  output as `Image` blocks. Ship as a separate module so the v1.0
  surface stays small.
- **`kardec/goldmarkx`** extension exposing Kardec as a goldmark
  renderer. Separate module.
- **godoc with runnable `Example*`** for every exported symbol in
  `style.go`, `block.go`, `run.go`, `document.go`, `typography.go`.
- **Benchmarks**: `BenchmarkRenderHello`, `BenchmarkRender100PageReport`,
  tracked with `benchstat` and gated in CI.

### v1.0 — freeze

The unsuffixed import path (`github.com/arthurhrc/kardec`) is correct
for v1.0 per [Go's module rules](https://go.dev/ref/mod#major-version-suffixes).
The `/v2` migration only triggers on the next breaking redesign.

Final freeze gates:

- API audit signed off against `go doc -all` diff vs. v0.9-rc.
- Every public block type appears in at least one `examples/*` main.
- `pkg.go.dev` lint pass (zero broken `[link]` references in godoc).
- Deterministic-output guarantee asserted in CI (render twice,
  `sha256sum`, fail on diff).
- `goreleaser` config emitting `SHA256SUMS` plus a `cosign`-signed
  attestation.

## Release-engineering checklist (pre-v1.0)

Replaces and tightens what `.github/workflows/ci.yml` ships today.

- **Coverage gating** (currently the report is printed, not gated).
  Threshold ≥ 85 %.
- **`golangci-lint`** running `govet`, `staticcheck`, `errcheck`,
  `revive`. Separate job.
- **`govulncheck ./...`** required for any "production" claim.
- **Cross-OS matrix**: Ubuntu, macOS, Windows. The deterministic
  byte-output claim is meaningless if it is Linux-only.
- **Reproducibility CI step**: render `examples/hello` twice with
  pinned `SetCreationDate`, `sha256sum` both, fail on diff.
- **Example PDFs as artifacts** (`actions/upload-artifact@v4`) so a
  reviewer can eyeball the rendering without checking out.
- **`pkg.go.dev` lint**: short script that runs `go doc` over every
  package and greps for unresolved `[Link]` references.
- **`MIGRATING.md`** keyed by version with three columns: Old API,
  New API, Codemod hint.
- **Signed release artifacts**: `goreleaser` + `cosign` keyless via
  GitHub OIDC. `SHA256SUMS` and `SHA256SUMS.sig` published with each
  tagged release.

## Honest risks

1. **Variadic-bool toggles will be the #1 review complaint.** Two
   appear today (`PDFA`, `SubsetFonts`); the pattern leaks if v1.0
   freezes them. Mitigation: the v0.6.1 sweep above.
2. **`Paragraph` + `AddParagraph` is an architectural fork.** Once
   both are public at v1.0, neither can be removed without v2.0.
   Mitigation: unify in v0.9.
3. **PDF/A-2b "lite" without OutputIntent + sRGB ICC will embarrass
   any user who submits to a strict regulator.** Mitigation: v0.6
   ships the ICC + OutputIntent before the v0.6.1 rename to
   `EnablePDFA()` cements the API contract; if v0.6 slips, the doc
   comment must promote "lite" from caveat to error message.
4. **Deterministic-output is unverified in CI.** Twelve indirect
   dependencies any of which could silently introduce
   map-iteration nondeterminism. A v1.0.1 patch breaking byte
   stability is a SemVer violation on the load-bearing
   differentiator. Mitigation: 20-line CI step described above,
   shipped before v1.0.

## Open additions (not yet scheduled)

These came up across the audits and deserve thought before v1.0
freezes the slot:

- ToUnicode CMap (PDF/A-2u + correct text extraction).
- PDF tagging for accessibility / PDF/UA.
- Forms / fillable fields.
- Encryption / permissions.
- Vector image embed (SVG → PDF native).
- Internal-anchor TOC entries clickable (currently the outline links,
  the TOC text does not).
