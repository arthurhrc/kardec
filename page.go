package kardec

// Orientation defines whether a page is taller than wide (Portrait) or
// wider than tall (Landscape). Applied at the PageSetup level.
type Orientation uint8

const (
	Portrait Orientation = iota
	Landscape
)

// PageSize describes the printable medium dimensions before margins are
// subtracted. Width is the short side in Portrait, the long side in
// Landscape; the rendering layer swaps them based on Orientation.
type PageSize struct {
	Name          string
	Width, Height Length
}

// Standard page presets. Dimensions follow ISO 216 for A* and ANSI for the
// US sizes, both expressed exactly in millimeters.
var (
	PageA3     = PageSize{Name: "A3", Width: Mm(297), Height: Mm(420)}
	PageA4     = PageSize{Name: "A4", Width: Mm(210), Height: Mm(297)}
	PageA5     = PageSize{Name: "A5", Width: Mm(148), Height: Mm(210)}
	PageLetter = PageSize{Name: "Letter", Width: In(8.5), Height: In(11)}
	PageLegal  = PageSize{Name: "Legal", Width: In(8.5), Height: In(14)}
)

// CustomPage builds a non-standard PageSize. The caller is responsible for
// passing a sensible (width, height) pair; Kardec does not enforce minimums
// since some use cases (labels, badges) rely on small media.
func CustomPage(name string, w, h Length) PageSize {
	return PageSize{Name: name, Width: w, Height: h}
}

// Margins describes the four printable insets of a page. All four sides are
// independent; the symmetric presets (MarginsNarrow / Normal / Wide) are
// expressed in centimeters following the values Word uses by default.
type Margins struct {
	Top, Right, Bottom, Left Length
}

// Symmetric returns Margins where all four sides equal the same value.
func Symmetric(v Length) Margins {
	return Margins{Top: v, Right: v, Bottom: v, Left: v}
}

var (
	MarginsNarrow = Symmetric(Cm(1.27))
	MarginsNormal = Symmetric(Cm(2.54))
	MarginsWide   = Symmetric(Cm(5.08))
)

// PageSetup couples a page size, orientation and margins. A Document holds
// one PageSetup per Section; the first Section inherits the values passed
// to New.
type PageSetup struct {
	Size        PageSize
	Orientation Orientation
	Margins     Margins
}
