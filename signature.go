package kardec

// SignatureLine and friends are contract-shaped composites: a thin
// horizontal rule followed by a name (and optionally a role) centered
// below. The whole bundle is wrapped in a KeepTogether so the line
// and the name never split across pages.
//
// The composite uses existing primitives (HorizontalRule + two
// Paragraphs), so the layout engine needs no special case for it —
// the construction simply emits the right block sequence at build
// time.

// SignatureBlock returns a KeepTogether containing a horizontal rule
// (the signature line itself), a centered name paragraph, and an
// optional centered role paragraph styled smaller. The returned
// value is a Block; pass it to doc.KeepTogether or use the
// Document.Signature builder for fluent appending.
func SignatureBlock(name, role string) Block {
	parts := make([]Block, 0, 3)
	parts = append(parts, HorizontalRule{Padding: Pt(2)})
	parts = append(parts, Paragraph{
		runs:      []Run{Text(name)},
		alignment: AlignCenter,
	})
	if role != "" {
		parts = append(parts, Paragraph{
			runs:      []Run{Italic(role)},
			styleName: StyleCaption,
			alignment: AlignCenter,
		})
	}
	return NewKeepTogether(parts...)
}

// Signature is the fluent builder shortcut for SignatureBlock. Pass
// an empty role to omit the second line.
func (d *Document) Signature(name, role string) *Document {
	return d.append(SignatureBlock(name, role))
}
