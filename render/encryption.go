package render

import "github.com/arthurhrc/kardec"

// encryptionPermissionBits packs the public Permissions struct
// into the signed-int32 /P value the PDF writer expects. The base
// value sets the reserved-must-be-1 bits per spec §7.6.3.2 Table
// 22; individual fields OR the corresponding bit. Mirrors the
// helper inside the kardec package; duplicated here so the render
// bridge has no reason to reach back into kardec internals.
func encryptionPermissionBits(p kardec.Permissions) int32 {
	bits := int32(-3904)
	if p.Print {
		bits |= 1 << 2
	}
	if p.Modify {
		bits |= 1 << 3
	}
	if p.Copy {
		bits |= 1 << 4
	}
	if p.Annotate {
		bits |= 1 << 5
	}
	if p.FillForms {
		bits |= 1 << 8
	}
	if p.AccessibilityCopy {
		bits |= 1 << 9
	}
	if p.AssembleDocument {
		bits |= 1 << 10
	}
	if p.PrintHighRes {
		bits |= 1 << 11
	}
	return bits
}
