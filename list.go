package kardec

// ListStyle selects the marker used for items in a List.
type ListStyle uint8

const (
	// ListUnordered prefixes each item with a bullet (•). Nested
	// levels rotate through bullet, hollow circle, square.
	ListUnordered ListStyle = iota
	// ListOrdered prefixes each item with its 1-based decimal index
	// followed by a period (1., 2., 3.). Nested levels reuse the
	// same numeral by default; per-level markers may be added later.
	ListOrdered
)

// List is a vertical sequence of bulleted or numbered items, each of
// which can carry inline runs and optionally nested sub-lists. The
// layout engine indents nested lists and rotates the bullet shape so
// the level is visually obvious.
type List struct {
	style ListStyle
	items []ListItem
}

// blockKind implements Block.
func (List) blockKind() blockKind { return kindList }

// Style returns the marker style for the list.
func (l List) Style() ListStyle { return l.style }

// Items returns the list's items in source order.
func (l List) Items() []ListItem { return l.items }

// ListItem is a single entry in a List. Runs is the inline content
// shown to the right of the marker; Children optionally nests another
// List one level deeper.
type ListItem struct {
	Runs     []Run
	Children []List
}

// ListBuilder accumulates items before the List block is appended to
// the parent document. Build returns the document so callers can resume
// the top-level chain.
type ListBuilder struct {
	doc   *Document
	style ListStyle
	items []ListItem
}

// List starts an unordered ListBuilder anchored to the document.
// The built block lands on Build.
func (d *Document) List() *ListBuilder {
	return &ListBuilder{doc: d, style: ListUnordered}
}

// OrderedList starts an ordered ListBuilder. Same shape as List with a
// different marker; useful when callers need a numbered list without
// having to tap a setter after construction.
func (d *Document) OrderedList() *ListBuilder {
	return &ListBuilder{doc: d, style: ListOrdered}
}

// Item appends one entry whose content is the supplied inline runs.
// For nested items, callers use Nested.
func (b *ListBuilder) Item(runs ...Run) *ListBuilder {
	b.items = append(b.items, ListItem{Runs: runs})
	return b
}

// Nested appends an item carrying inline runs plus one or more nested
// sub-lists rendered indented below the item's own content. Nested
// lists may themselves contain further nesting; the layout engine
// indents one step per level.
func (b *ListBuilder) Nested(runs []Run, children ...List) *ListBuilder {
	b.items = append(b.items, ListItem{Runs: runs, Children: children})
	return b
}

// Build appends the constructed List to the parent document and
// returns the document for chained subsequent calls. An empty list
// (no items) appends nothing — empty lists are not an error, simply
// a no-op so callers can guard with conditional `Item` calls without
// special-casing.
func (b *ListBuilder) Build() *Document {
	if b.doc.err != nil {
		return b.doc
	}
	if len(b.items) == 0 {
		return b.doc
	}
	return b.doc.append(List{style: b.style, items: b.items})
}

// SubList constructs a nested List value (without committing to a
// document) so it can be passed to ListBuilder.Nested. The same Style
// rules as the top-level builder apply.
func SubList(style ListStyle, items ...ListItem) List {
	return List{style: style, items: items}
}
