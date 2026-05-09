package kardec

// KeepTogether wraps a slice of inner blocks so the layout engine
// guarantees they all land on the same page. The canonical use case
// is binding a heading to the first paragraph that follows it: the
// engine never produces a page that ends with the heading and starts
// the next page with the paragraph.
//
// When the group is taller than a single page, KeepTogether degrades
// gracefully: the engine flushes once to start the group on a fresh
// page, then allows the inner blocks to overflow naturally onto
// further pages. This avoids an infinite-flush loop on oversized
// groups.
//
// KeepTogether may not be nested inside another KeepTogether. The
// engine treats nested groups as a flat block sequence.
type KeepTogether struct {
	blocks []Block
}

// NewKeepTogether returns a KeepTogether group containing the supplied
// blocks in document order. The slice is copied so further mutation by
// the caller does not affect the document.
func NewKeepTogether(blocks ...Block) KeepTogether {
	cp := make([]Block, len(blocks))
	copy(cp, blocks)
	return KeepTogether{blocks: cp}
}

func (KeepTogether) blockKind() blockKind { return kindKeepTogether }

// Blocks returns the inner blocks of the group. The returned slice is
// the group's own backing storage; callers must not mutate it.
func (k KeepTogether) Blocks() []Block { return k.blocks }
