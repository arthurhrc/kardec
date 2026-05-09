package kardec

// TableOfContents is the block that, after layout, expands into a
// list of every Heading in the document with its title, a dot
// leader and the page number on which it appears.
//
// The block must be placed before any heading the caller wants
// indexed; headings appearing before the block are skipped because
// their page number is already known when we reach the block, but
// the contract simplifies if the TOC sits at the top of the document.
//
// Levels selects the maximum heading depth indexed (1 = H1 only,
// 6 = every level). Zero means "all levels".
type TableOfContents struct {
	maxLevel int
}

// blockKind implements Block.
func (TableOfContents) blockKind() blockKind { return kindTOC }

// MaxLevel returns the deepest heading level included in the TOC.
// Zero means unlimited.
func (t TableOfContents) MaxLevel() int { return t.maxLevel }

// TableOfContents appends an automatic table-of-contents block.
// The maxLevel argument caps which heading depths land in the TOC:
// 1 indexes only H1, 2 indexes H1 and H2, and so on. Pass 0 to
// include every heading regardless of depth.
//
// The block reserves space proportional to the heading count at
// layout time; page numbers are patched in a post-pass once the
// final pagination is known.
func (d *Document) TableOfContents(maxLevel int) *Document {
	if d.err != nil {
		return d
	}
	return d.append(TableOfContents{maxLevel: maxLevel})
}
