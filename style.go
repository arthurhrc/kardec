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

// Built-in named style identifiers. Strings (not typed constants) are kept
// because users define their own styles by arbitrary names and the same
// lookup path serves both built-in and custom styles.
const (
	StyleDefault     = "Default"
	StyleH1          = "H1"
	StyleH2          = "H2"
	StyleH3          = "H3"
	StyleH4          = "H4"
	StyleH5          = "H5"
	StyleH6          = "H6"
	StyleCaption     = "Caption"
	StyleQuote       = "Quote"
	StyleCode        = "Code"
	StyleTableHeader = "TableHeader"
	StyleTableCell   = "TableCell"
	StyleFooter      = "Footer"
	StyleHeader      = "Header"
	StyleListItem    = "ListItem"
	StyleLink        = "Link"
)

// FontLiberationSans is the family name of the bundled default sans face.
// Mirrors the typography track; declaring it here lets style consumers refer
// to it without importing the future font registry package.
const (
	FontLiberationSans  = "Liberation Sans"
	FontLiberationSerif = "Liberation Serif"
	FontJetBrainsMono   = "JetBrains Mono"
)

// DefaultStyle is the root of the inheritance tree. Every other built-in
// style — and every user-defined style without an explicit ParentStyle —
// resolves through it. Picked to match the Word "Normal" body-text feel.
var DefaultStyle = Style{
	Family:      FontLiberationSans,
	Size:        Pt(11),
	Weight:      WeightRegular,
	Italic:      false,
	Color:       ColorBlack,
	SpaceBefore: Pt(0),
	SpaceAfter:  Pt(6),
	LineHeight:  1.15,
	Alignment:   AlignLeft,
}

// BuiltinStyles returns a fresh map containing every built-in named style.
// The returned map is owned by the caller; mutating it does not affect
// future calls. Document.New copies these into the document's style table
// during construction.
func BuiltinStyles() map[string]Style {
	headingBlue := HexColor("#2E74B5")
	headingBlueLight := HexColor("#5B9BD5")
	codeColor := HexColor("#333333")
	quoteColor := HexColor("#666666")
	linkColor := HexColor("#0563C1")

	return map[string]Style{
		StyleDefault: DefaultStyle,

		StyleH1: {
			Size:         Pt(24),
			Weight:       WeightBold,
			Color:        headingBlue,
			SpaceBefore:  Pt(18),
			SpaceAfter:   Pt(8),
			LineHeight:   1.2,
			KeepWithNext: true,
		},
		StyleH2: {
			Size:         Pt(18),
			Weight:       WeightBold,
			Color:        headingBlueLight,
			SpaceBefore:  Pt(14),
			SpaceAfter:   Pt(6),
			LineHeight:   1.2,
			KeepWithNext: true,
		},
		StyleH3: {
			Size:         Pt(14),
			Weight:       WeightBold,
			Color:        headingBlueLight,
			SpaceBefore:  Pt(12),
			SpaceAfter:   Pt(6),
			LineHeight:   1.2,
			KeepWithNext: true,
		},
		StyleH4: {
			Size:         Pt(12),
			Weight:       WeightBold,
			Color:        headingBlueLight,
			SpaceBefore:  Pt(10),
			SpaceAfter:   Pt(4),
			LineHeight:   1.2,
			KeepWithNext: true,
			Italic:       true,
		},
		StyleH5: {
			Size:         Pt(11),
			Weight:       WeightSemiBold,
			Color:        headingBlueLight,
			SpaceBefore:  Pt(8),
			SpaceAfter:   Pt(4),
			LineHeight:   1.2,
			KeepWithNext: true,
		},
		StyleH6: {
			Size:         Pt(11),
			Weight:       WeightSemiBold,
			Italic:       true,
			Color:        headingBlueLight,
			SpaceBefore:  Pt(8),
			SpaceAfter:   Pt(4),
			LineHeight:   1.2,
			KeepWithNext: true,
		},

		StyleCaption: {
			Size:       Pt(9),
			Italic:     true,
			Color:      quoteColor,
			SpaceAfter: Pt(8),
			LineHeight: 1.1,
			Alignment:  AlignCenter,
		},
		StyleQuote: {
			Size:        Pt(11),
			Italic:      true,
			Color:       quoteColor,
			SpaceBefore: Pt(8),
			SpaceAfter:  Pt(8),
			LineHeight:  1.3,
		},
		StyleCode: {
			Family:      FontJetBrainsMono,
			Size:        Pt(10),
			Color:       codeColor,
			SpaceBefore: Pt(6),
			SpaceAfter:  Pt(6),
			LineHeight:  1.2,
		},
		StyleTableHeader: {
			Weight:    WeightBold,
			Alignment: AlignCenter,
		},
		StyleTableCell: {
			// Identity over Default; named so users can override globally.
		},
		StyleHeader: {
			Size:       Pt(9),
			Color:      quoteColor,
			Alignment:  AlignCenter,
			SpaceAfter: Pt(0),
		},
		StyleFooter: {
			Size:       Pt(9),
			Color:      quoteColor,
			Alignment:  AlignCenter,
			SpaceAfter: Pt(0),
		},
		StyleListItem: {
			SpaceAfter: Pt(2),
			LineHeight: 1.15,
		},
		StyleLink: {
			Color: linkColor,
		},
	}
}

// HeadingStyleName returns the built-in named style that corresponds to a
// heading level (1..6). Levels outside the range are clamped, matching the
// Document.Heading constructor.
func HeadingStyleName(level int) string {
	switch {
	case level <= 1:
		return StyleH1
	case level == 2:
		return StyleH2
	case level == 3:
		return StyleH3
	case level == 4:
		return StyleH4
	case level == 5:
		return StyleH5
	default:
		return StyleH6
	}
}
