package kardec

// Weight selects a font face within a family. Concrete glyph mapping happens
// in the typography track; here we only model the abstract weight ordinal so
// styles can declare intent without depending on a font registry.
//
// The "Weight" prefix on each constant disambiguates them from same-named
// run helpers (Bold, Italic) that already exist in the inline content API.
type Weight uint8

// Weight constants follow the common five-step subset of the OpenType
// usWeightClass scale. Additional intermediate weights may be added later
// without breaking the existing names.
const (
	WeightRegular  Weight = iota // 400 in OpenType terms
	WeightMedium                 // 500
	WeightSemiBold               // 600
	WeightBold                   // 700
	WeightBlack                  // 900
)

// Style is the value-type carrier for every text-formatting attribute Kardec
// understands. Styles compose by inheritance (see Document.ResolveStyle):
// a child Style fills in only the fields it cares about, and missing fields
// fall through to the parent identified by ParentStyle.
//
// "Missing" is encoded as the Go zero value for each field. This works
// because all defaults flow from the Default style, which sets every field
// to a sensible non-zero value; an explicit zero on a child therefore means
// "do not override", not "force zero". The lone exception is bool fields
// (Italic, KeepWithNext, KeepTogether, PageBreakBefore), which can only be
// true to override — they are additive.
type Style struct {
	// Family is the font family name (e.g. "Liberation Sans"). Resolution
	// against an actual font registry is the typography track's concern.
	Family string

	// Size is the glyph em size; zero means "inherit".
	Size Length

	// Weight selects a face within the family.
	Weight Weight

	// Italic toggles italic/oblique style. Additive across the chain.
	Italic bool

	// Color is the glyph color.
	Color Color

	// SpaceBefore and SpaceAfter add vertical whitespace around the block.
	SpaceBefore Length
	SpaceAfter  Length

	// LineHeight is a multiplier of the font size: 1.0 = single-spaced,
	// 1.5 = one-and-a-half. Zero means "inherit".
	LineHeight float64

	// Alignment controls horizontal text arrangement.
	Alignment Alignment

	// KeepWithNext asks the layout engine not to break a page between this
	// block and the following one. Additive across the chain.
	KeepWithNext bool

	// KeepTogether asks the layout engine to render this block on a single
	// page if at all possible. Additive.
	KeepTogether bool

	// PageBreakBefore forces a page break before this block. Additive.
	PageBreakBefore bool

	// ParentStyle names the style this one inherits from. The empty string
	// means "inherit from Default" (unless this Style is itself Default).
	ParentStyle string
}
