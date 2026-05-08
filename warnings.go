package kardec

// Warnings returns the document's accumulated non-fatal advisories
// in source order. Each entry is a human-readable sentence the
// builder pipeline produced when it could not render a piece of
// input as expected — typically Markdown nodes that the bridge does
// not yet honor (HTML blocks, autolinks, footnotes), or links whose
// destination was empty.
//
// Warnings are non-fatal: the document still renders successfully
// even when this slice is non-empty. CI pipelines that want strict
// fidelity check `len(doc.Warnings()) == 0` after AppendMarkdown.
//
// The returned slice is the document's own backing storage; callers
// must not mutate it.
func (d *Document) Warnings() []string { return d.warnings }

// warn appends a non-fatal advisory to the document. Used by the
// internal Markdown bridge and any future ingest layer that wants to
// flag dropped or downgraded input. End-user code should not need to
// call this directly.
func (d *Document) warn(msg string) {
	d.warnings = append(d.warnings, msg)
}
