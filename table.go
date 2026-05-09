package kardec

import "errors"

// errEmptyTable is captured on a Document when TableBuilder.Build runs
// without any columns having been declared. The deferred-error chain
// then surfaces this through Render / Bytes.
var errEmptyTable = errors.New("kardec: table requires at least one column")

// TableBorderStyle selects which lines the layout engine draws around
// and between table cells. The default zero value (BordersNone) keeps
// tables visually quiet — useful when the surrounding document
// already separates rows through whitespace.
type TableBorderStyle uint8

const (
	// BordersNone draws no lines. Cells flow as text blocks.
	BordersNone TableBorderStyle = iota
	// BordersHorizontal draws only the rules between rows plus the
	// outer top and bottom borders. Common for typographically clean
	// tables (no vertical dividers).
	BordersHorizontal
	// BordersAll draws a full grid: outer rectangle plus every row
	// and column boundary. Closest analogue to Word's "All Borders".
	BordersAll
)

// Table is a block of tabular data composed of column descriptors and
// rows of cells. Each cell is a slice of inline Runs, so callers can
// mix styled fragments inside a single cell ("R$ ", Bold("1.000")).
//
// Borders, header shading and alternate-row shading are all opt-in via
// the matching TableBuilder setters. The defaults produce the same
// visually quiet table v0.2 shipped.
type Table struct {
	columns             []Column
	rows                []Row
	repeatHeader        bool // when true, the first row is reprinted on each page break
	borderStyle         TableBorderStyle
	headerShading       *Color // non-nil paints a background rectangle behind row 0
	alternateRowShading *Color // non-nil paints behind every other body row (1, 3, 5, ...)
}

// blockKind implements Block. The layout engine dispatches on this kind
// to invoke the table-specific placement code.
func (Table) blockKind() blockKind { return kindTable }

// Columns returns the table's column descriptors. Layout code reads
// widths and alignments from this slice; callers should not mutate the
// returned values.
func (t Table) Columns() []Column { return t.columns }

// Rows returns the table's data rows in order.
func (t Table) Rows() []Row { return t.rows }

// RepeatHeader reports whether the first row should be reprinted at the
// top of every page the table spans. Only meaningful for tables wider
// than one page in height.
func (t Table) RepeatHeader() bool { return t.repeatHeader }

// BorderStyle returns the table's selected border configuration.
func (t Table) BorderStyle() TableBorderStyle { return t.borderStyle }

// HeaderShading returns the row-0 background color and a flag
// indicating whether one was set. Layout reads this to emit a
// shading rectangle behind the header row before placing text.
func (t Table) HeaderShading() (Color, bool) {
	if t.headerShading == nil {
		return Color{}, false
	}
	return *t.headerShading, true
}

// AlternateRowShading returns the alternate-row background color and
// a flag indicating whether one was set. The body rows at odd indices
// (1, 3, 5, ...) receive the shading; the header row is never shaded
// through this knob, even when AlternateRowShading is set.
func (t Table) AlternateRowShading() (Color, bool) {
	if t.alternateRowShading == nil {
		return Color{}, false
	}
	return *t.alternateRowShading, true
}

// Column describes a single table column: its visible header, target
// width and horizontal alignment for the contained cells.
//
// Width carries a dual meaning by convention: values in (0, 1] are
// interpreted as fractions of the available content width; values
// strictly greater than 1 are treated as fixed point sizes. Negative or
// zero widths are normalised by the layout engine into equal shares of
// the remaining space.
type Column struct {
	Header    string
	Width     float64
	Alignment Alignment
}

// Row is one tabular line. Each entry in Cells corresponds positionally
// to the column slice declared on the Table.
type Row struct {
	Cells []Cell
}

// Cell carries the inline runs that fill a single (row, column)
// intersection. Plain string cells become a single text Run; richer
// content can be supplied via the Cells helper.
type Cell struct {
	Runs []Run
	// Span is the number of columns this cell occupies. 0 and 1 are
	// equivalent (single column); values larger than the remaining
	// columns are clamped at layout time. Cells consumed by a
	// preceding span are simply absent from the row's slice — the
	// layout engine advances the column index by Span as it walks
	// the row.
	Span int
}

// ColumnOption customises a Column when constructed through the Col
// helper. The functional-option pattern keeps the constructor surface
// stable while leaving room for future per-column knobs.
type ColumnOption func(*Column)

// Col returns a Column with the supplied header and any number of
// options applied. Without options, a Column inherits Width=0 (which
// the layout engine resolves to "an equal share of the remainder")
// and Alignment=AlignLeft.
func Col(header string, opts ...ColumnOption) Column {
	c := Column{Header: header}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// Width sets the column's target width. See the Column doc for the
// fraction-vs-fixed convention.
func Width(v float64) ColumnOption { return func(c *Column) { c.Width = v } }

// AlignLeftCol forces a column's cells to left-align. AlignLeft is the
// default; this option exists for documentary symmetry alongside the
// right and center variants.
func AlignLeftCol() ColumnOption { return func(c *Column) { c.Alignment = AlignLeft } }

// AlignCenterCol centers the column's cells horizontally.
func AlignCenterCol() ColumnOption { return func(c *Column) { c.Alignment = AlignCenter } }

// AlignRightCol right-aligns the column's cells, useful for currency
// or numeric data.
func AlignRightCol() ColumnOption { return func(c *Column) { c.Alignment = AlignRight } }

// AlignDecimalCol aligns numeric cells on the decimal point. The
// integer part (everything up to the first ".") is right-aligned
// against a pivot positioned at 60% of the column width; the
// fractional part flows from the pivot. Cells without a "." pivot
// fall back to right alignment so an integer row matches the column
// of digits formed by the dotted neighbours. Recommended for
// currency or measurement columns where the eye benefits from a
// shared pivot.
func AlignDecimalCol() ColumnOption { return func(c *Column) { c.Alignment = AlignDecimal } }

// TableBuilder accumulates column descriptors, rows and rendering hints
// before the Table block is appended to the parent document. Build
// returns the document so callers can resume the top-level chain.
type TableBuilder struct {
	doc                 *Document
	columns             []Column
	rows                []Row
	repeatHeader        bool
	borderStyle         TableBorderStyle
	headerShading       *Color
	alternateRowShading *Color
	label               string
	err                 error
}

// Table starts a new TableBuilder anchored to the document. The Table
// block is not committed until Build is called.
func (d *Document) Table() *TableBuilder { return &TableBuilder{doc: d} }

// Columns sets the table's column descriptors. Subsequent calls
// override; the last call wins. Provide at least one column or Build
// will record a deferred error on the document.
func (b *TableBuilder) Columns(cols ...Column) *TableBuilder {
	b.columns = cols
	return b
}

// RepeatHeader marks the table's first row as a header that the layout
// engine reprints on every page the table spans.
func (b *TableBuilder) RepeatHeader() *TableBuilder {
	b.repeatHeader = true
	return b
}

// Borders selects the table's border configuration. The default zero
// value (BordersNone) leaves cells without lines; BordersHorizontal
// draws between rows plus top/bottom; BordersAll draws a full grid.
func (b *TableBuilder) Borders(style TableBorderStyle) *TableBuilder {
	b.borderStyle = style
	return b
}

// HeaderShading paints a colored rectangle behind the table's first
// row before the cell text is placed. Pair with RepeatHeader to keep
// the highlight visible on every continuation page.
func (b *TableBuilder) HeaderShading(c Color) *TableBuilder {
	color := c
	b.headerShading = &color
	return b
}

// AlternateRowShading paints a colored rectangle behind every other
// body row (indices 1, 3, 5, ...). The header row is never shaded by
// this knob — set HeaderShading separately if both are wanted.
func (b *TableBuilder) AlternateRowShading(c Color) *TableBuilder {
	color := c
	b.alternateRowShading = &color
	return b
}

// Label tags the table with a cross-reference label. Build registers
// the label, increments the table counter, and emits an invisible
// anchor immediately before the table so doc.Ref(label) hyperlinks
// resolve to its position in the rendered PDF. An empty label is
// ignored.
func (b *TableBuilder) Label(name string) *TableBuilder {
	b.label = name
	return b
}

// Row appends one row of cells. Each argument is the plain text for the
// corresponding column; richer cell content goes through RowCells.
func (b *TableBuilder) Row(cells ...string) *TableBuilder {
	row := Row{Cells: make([]Cell, len(cells))}
	for i, s := range cells {
		row.Cells[i] = Cell{Runs: []Run{Text(s)}}
	}
	b.rows = append(b.rows, row)
	return b
}

// RowCells appends a row composed of pre-built Cell values, allowing
// per-cell rich runs (bold, italic, color overrides). Cells beyond the
// configured column count are kept; cells short of the column count are
// padded with empty cells at render time.
func (b *TableBuilder) RowCells(cells ...Cell) *TableBuilder {
	b.rows = append(b.rows, Row{Cells: cells})
	return b
}

// Cells builds a Cell from the supplied runs — the rich-content
// counterpart to the plain string accepted by Row.
func Cells(runs ...Run) Cell { return Cell{Runs: runs} }

// SpanCell builds a Cell that occupies span columns. The most common
// use case is a merged header that labels a group of underlying
// columns ("Q1 vs Q2" spanning two single-month columns).
//
// span values <= 1 collapse to a normal one-column cell. The next
// span-1 cells in the row's slice should be omitted: layout absorbs
// the column budget into this Cell's width and advances the column
// pointer by span.
func SpanCell(span int, runs ...Run) Cell {
	if span < 1 {
		span = 1
	}
	return Cell{Runs: runs, Span: span}
}

// Build appends the constructed Table to the parent document and
// returns the document for chained subsequent calls. If the builder has
// no columns, the document captures a deferred error and the table is
// not appended.
func (b *TableBuilder) Build() *Document {
	if b.doc.err != nil {
		return b.doc
	}
	if len(b.columns) == 0 {
		return b.doc.fail(errEmptyTable)
	}
	tbl := Table{
		columns:             b.columns,
		rows:                b.rows,
		repeatHeader:        b.repeatHeader,
		borderStyle:         b.borderStyle,
		headerShading:       b.headerShading,
		alternateRowShading: b.alternateRowShading,
	}
	if b.label != "" {
		b.doc.registerTableLabel(b.label)
		b.doc.append(Anchor{name: RefAnchorName(b.label)})
	}
	return b.doc.append(tbl)
}
