package kardec

// Anchor is a named position inside the document. It occupies no
// vertical space on its own; the layout engine simply records the
// page and Y coordinate at which it was inserted, and the PDF writer
// turns the result into a named destination linkable from anywhere
// else in the document via a "#<name>" URL.
//
// Pair Anchor with Link("text", "#name") to build clickable cross-
// references — the canonical use case is a TOC at the top of the
// document linking to each section heading.
type Anchor struct {
	name string
}

// blockKind implements Block.
func (Anchor) blockKind() blockKind { return kindAnchor }

// Name returns the anchor's identifier. Read-only access for layout
// and renderer integrations.
func (a Anchor) Name() string { return a.name }

// Anchor appends a named destination at the current flow position.
// The name is matched against the "#<name>" target of internal Link
// runs — see Link / SetLink — and is also exposed in the PDF's named
// destinations table so external tools can jump directly via URL
// fragments.
//
// Names should be ASCII identifiers without spaces; the writer does
// not escape them. An empty name is a no-op so callers can guard
// programmatic anchor generation without special-casing.
func (d *Document) Anchor(name string) *Document {
	if d.err != nil || name == "" {
		return d
	}
	return d.append(Anchor{name: name})
}
