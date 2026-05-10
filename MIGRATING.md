# Migrating Kardec

This guide tracks every breaking-or-renamed identifier across the v0.x
line so callers can update mechanically. Each section names the
release that introduced the change, lists the old API, the new API,
and a sed-style codemod hint where one applies.

The `// Deprecated:` doc-comment on each forwarder also points at the
new name; gopls surfaces the migration target in your editor.

## v0.10 — API rename sweep (the big one)

v0.10 normalised the public surface ahead of the v1.0 freeze. Every
rename here ships with a deprecated forwarder that keeps the old
name working through the v0.x line; the forwarders are removed at
v1.0. Two changes are immediate removals (no forwarder).

### Removed at v0.10 (no forwarder)

| Old API | New API | Codemod |
|---|---|---|
| `Document.AddParagraph(runs...) *ParagraphBuilder` | `Document.Paragraph(runs...) *ParagraphRef` | `s/\.AddParagraph(/\.Paragraph(/g; s/\.Done()//g` |
| `kardec.ParagraphBuilder` (type alias) | `kardec.ParagraphRef` | type already absorbed; rename only |
| `Document.MathInline(src) *Document` | `Document.Math(src)` (display math); inline math will land as a Run constructor in a future release | drop the call or replace with `Math(src)` |

### Renamed with deprecated forwarders (removed at v1.0)

| Old API | New API | Codemod |
|---|---|---|
| `Document.PDFA(on ...bool)` | `Document.EnablePDFA()` / `Document.DisablePDFA()` | `s/\.PDFA()/\.EnablePDFA()/g; s/\.PDFA(true)/\.EnablePDFA()/g; s/\.PDFA(false)/\.DisablePDFA()/g` |
| `Document.SubsetFonts(on ...bool)` | `Document.EnableFontSubsetting()` / `Document.DisableFontSubsetting()` | `s/\.SubsetFonts()/\.EnableFontSubsetting()/g; s/\.SubsetFonts(true)/\.EnableFontSubsetting()/g; s/\.SubsetFonts(false)/\.DisableFontSubsetting()/g` |
| `kardec.Footnote(d, body)` | `d.Footnote(body)` | `s/kardec\.Footnote(\([a-zA-Z_]*\), /\1.Footnote(/g` |
| `kardec.FootnoteWithMarker(d, marker, body...)` | `d.FootnoteWith(marker, body...)` | `s/kardec\.FootnoteWithMarker(\([a-zA-Z_]*\), /\1.FootnoteWith(/g` |
| `Document.NewSection(size, margins)` | `Document.NewSection(setup PageSetup)` plus `kardec.SetupOf(size, margins)` helper | `s/\.NewSection(\(.*\), \(.*\))/\.NewSection(kardec.SetupOf(\1, \2))/g` |
| `Document.NewSectionWithSetup(setup)` | `Document.NewSection(setup)` | `s/\.NewSectionWithSetup(/\.NewSection(/g` |
| `kardec.Cells(runs...)` | `kardec.NewCell(runs...)` | `s/kardec\.Cells(/kardec.NewCell(/g` |
| `kardec.AlignLeftCol()` | `kardec.WithAlignment(kardec.AlignLeft)` | `s/kardec\.AlignLeftCol()/kardec.WithAlignment(kardec.AlignLeft)/g` |
| `kardec.AlignCenterCol()` | `kardec.WithAlignment(kardec.AlignCenter)` | `s/kardec\.AlignCenterCol()/kardec.WithAlignment(kardec.AlignCenter)/g` |
| `kardec.AlignRightCol()` | `kardec.WithAlignment(kardec.AlignRight)` | `s/kardec\.AlignRightCol()/kardec.WithAlignment(kardec.AlignRight)/g` |
| `kardec.AlignDecimalCol()` | `kardec.WithAlignment(kardec.AlignDecimal)` | `s/kardec\.AlignDecimalCol()/kardec.WithAlignment(kardec.AlignDecimal)/g` |
| `kardec.BordersNone` | `kardec.TableBordersNone` | `s/kardec\.BordersNone/kardec.TableBordersNone/g` |
| `kardec.BordersHorizontal` | `kardec.TableBordersHorizontal` | `s/kardec\.BordersHorizontal/kardec.TableBordersHorizontal/g` |
| `kardec.BordersAll` | `kardec.TableBordersAll` | `s/kardec\.BordersAll/kardec.TableBordersAll/g` |

### Marked Deprecated (still callable; will become internal at v1.0)

These three exist on the public surface only because Go has no
friend-package mechanism. They shipped without `// Deprecated:` until
v0.10; the comment was added so gopls steers users away. The
identifiers stay reachable for the v0.x line and become internal at
v1.0.

| API | Why Deprecated | Replacement |
|---|---|---|
| `kardec.SetRenderImpl` | render-injection seam; only the `kardec/render` `init()` should call it | none — user code should never have called it |
| `Document.FontRegistry()` | leaks internal `*typography.Registry` | `Document.RegisteredFamilies() []string` |
| `(*Run).SetLink(url)` | markdown-bridge mutation of an otherwise-immutable Run | `kardec.Link(text, url)` for new content |

## v0.9 — Paragraph / AddParagraph unified

v0.9.0 changed the return type of `Document.Paragraph` from
`*Document` to `*ParagraphRef`. The ref embeds `*Document` so chained
methods (`Heading`, `Image`, etc.) keep flowing; tests / call sites
that captured the result in a `*Document` variable and passed it to
a function expecting `*Document` need `.Document` once:

```go
// before
out, err := render.Bytes(doc)

// after (when doc came from a chain ending in .Paragraph(...))
out, err := render.Bytes(doc.Document)
```

`AddParagraph` lingered as a deprecated alias through v0.9; v0.10
removed it (see above).

## v0.7 — Cross-references introduced (no removals)

No breaking changes. `ImageBuilder.Label(name)` and
`TableBuilder.Label(name)` opt into the figure / table counter;
`doc.Ref(label)` and `doc.RefPage(label)` resolve references.
Existing code without labels keeps the v0.6 behaviour.

## v0.5 → v0.7 (skipped v0.6)

v0.6.0 has no release; the planned scope split between v0.7
(cross-references, KeepTogether), v0.10 (API rename sweep), and v0.11
(font embedding, ICC, metadata, hyphenation). The CHANGELOG entry
for [0.6.0](CHANGELOG.md#060) carries the marker; the tag history
jumps from v0.5.0 to v0.7.0 directly.

## Earlier than v0.5

No public breaking changes. The v0.x line up to v0.5 was strictly
additive — every block type / builder method that landed in v0.1–v0.5
remains callable.
