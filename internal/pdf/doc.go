// Package pdf is Kardec's PDF 1.7 writer. It consumes a positioned glyph
// model (Document/Page/TextItem) and emits a real PDF file readable by
// Adobe Acrobat, Chrome, and pdftk.
//
// This package is intentionally minimal: it implements only the spec subset
// needed to place styled text on a page. Tables, images, links, encryption
// and PDF/A compliance are out of scope. ISO 32000-1 (PDF 1.7) is the
// reference; section numbers in code comments cite that spec.
//
// The writer is internal to Kardec; the public surface stays in the root
// package via Document.Render / RenderTo / Bytes.
package pdf
