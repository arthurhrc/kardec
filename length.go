package kardec

// Length is a typed dimension expressed internally in PDF user-space units
// (1/72 of an inch). Constructors Pt, Mm, Cm, In convert from common units.
//
// Using a distinct type prevents accidental mixing of raw float64 sizes from
// different unit systems at API boundaries.
type Length float64

const (
	pointsPerInch = 72.0
	mmPerInch     = 25.4
	cmPerInch     = 2.54
)

// Pt returns a Length expressed in PDF points (1/72 inch).
func Pt(v float64) Length { return Length(v) }

// Mm returns a Length converted from millimeters.
func Mm(v float64) Length { return Length(v / mmPerInch * pointsPerInch) }

// Cm returns a Length converted from centimeters.
func Cm(v float64) Length { return Length(v / cmPerInch * pointsPerInch) }

// In returns a Length converted from inches.
func In(v float64) Length { return Length(v * pointsPerInch) }

// Points returns the Length value in PDF points.
func (l Length) Points() float64 { return float64(l) }

// Millimeters returns the Length value in millimeters.
func (l Length) Millimeters() float64 { return float64(l) / pointsPerInch * mmPerInch }
