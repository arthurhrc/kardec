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
