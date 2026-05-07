# Kardec

> A Go DSL for generating document-like PDFs — without containers, without LibreOffice, without external runtime dependencies.

Kardec is a pure-Go library for producing professional-looking PDFs (think Word, not financial report) through a fluent, style-driven API. It ships with embedded fonts and a layout engine tuned for body-text documents — paragraphs that flow, headings that breathe, tables that split across pages cleanly.

## Why

The existing Go landscape forces a choice:

- **Container-based** (Gotenberg, etc.) — wraps LibreOffice, perfect fidelity but requires Docker in your pipeline.
- **Grid-based PDF generators** (Maroto, gofpdf) — pure Go, but output looks like a report, not a document.

Kardec aims at the middle: **document-quality output, zero external runtime dependencies**.

## Status

`v0.x` — experimental. API is unstable until `v1.0`.

See [docs/RFC-001-dsl.md](docs/RFC-001-dsl.md) for the design spec.

## License

TBD (likely MIT or Apache 2.0).
