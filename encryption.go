package kardec

// encryptionConfig is the local-to-Document storage form of the
// configured encryption. It holds the same data the internal pdf
// package's Encryption struct expects, but exists here so the
// public surface does not transitively pull in internal/pdf.
type encryptionConfig struct {
	UserPwd     string
	OwnerPwd    string
	Permissions int32
}

// EncryptionOptions configures the PDF Standard Security Handler
// (V=4 / R=4 / AES-128). Pass to Document.SetEncryption to opt in.
//
// UserPassword is what readers must supply to OPEN the document.
// An empty user password produces a "permissions-only" PDF anyone
// can open but only the owner password can re-edit / re-print.
//
// OwnerPassword authorises bypassing the permissions. Empty
// OwnerPassword falls back to UserPassword (which is then the
// only credential — common for personal docs).
//
// Permissions controls which operations a non-owner reader is
// allowed to perform: print, copy, modify, fill forms, etc. The
// zero value disallows everything. Use the convenience
// AllPermissions / ReadOnlyPermissions helpers, or compose
// individual flags via the Permission constants.
type EncryptionOptions struct {
	UserPassword  string
	OwnerPassword string
	Permissions   Permissions
}

// Permissions enumerates the standard-security-handler permission
// bits per PDF 1.7 §7.6.3.2 Table 22. Each field maps to a single
// bit; the zero value denies everything.
type Permissions struct {
	Print              bool // bit 3 — basic print resolution
	Modify             bool // bit 4 — modify document contents
	Copy               bool // bit 5 — extract text / images
	Annotate           bool // bit 6 — add comments / annotations
	FillForms          bool // bit 9 — fill interactive form fields
	AccessibilityCopy  bool // bit 10 — extract for accessibility
	AssembleDocument   bool // bit 11 — rotate / insert / delete pages
	PrintHighRes       bool // bit 12 — high-resolution print
}

// AllPermissions grants every standard permission. Useful when the
// caller only wants password-protection without restricting ops.
func AllPermissions() Permissions {
	return Permissions{
		Print:             true,
		Modify:            true,
		Copy:              true,
		Annotate:          true,
		FillForms:         true,
		AccessibilityCopy: true,
		AssembleDocument:  true,
		PrintHighRes:      true,
	}
}

// ReadOnlyPermissions allows only viewing + accessibility. The
// canonical "no print, no copy, no modify" configuration for
// distribution-controlled documents.
func ReadOnlyPermissions() Permissions {
	return Permissions{AccessibilityCopy: true}
}

// SetEncryption opts the document into the PDF Standard Security
// Handler. The renderer wraps every stream payload (content
// streams, font data, image data, ToUnicode CMaps) in AES-128-CBC
// with a per-object key derived from the supplied passwords +
// permissions + document /ID.
//
// Strings (Title, Author, Subject, Keywords, link /URI) are
// encrypted alongside streams (post-v0.22). The /Encrypt dict
// declares /StmF /StdCF /StrF /StdCF so AESV2 covers both.
//
// Calling SetEncryption with the zero EncryptionOptions disables
// any prior encryption.
func (d *Document) SetEncryption(opts EncryptionOptions) *Document {
	if d.err != nil {
		return d
	}
	if opts == (EncryptionOptions{}) {
		d.encryption = nil
		return d
	}
	d.encryption = &encryptionConfig{
		UserPwd:     opts.UserPassword,
		OwnerPwd:    opts.OwnerPassword,
		Permissions: permissionBits(opts.Permissions),
	}
	return d
}


// permissionBits packs a Permissions struct into the signed int32
// the PDF /P entry expects. The base value sets the reserved-must-
// be-1 bits per spec; individual fields OR the corresponding bit.
func permissionBits(p Permissions) int32 {
	bits := int32(-3904) // bits 7-8, 13-32 set per spec
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

// Encryption returns the configured EncryptionOptions plus a
// boolean indicating whether SetEncryption was called. Read-only;
// the render bridge consults this to populate pdf.Document.
func (d *Document) Encryption() (EncryptionOptions, bool) {
	if d.encryption == nil {
		return EncryptionOptions{}, false
	}
	return EncryptionOptions{
		UserPassword:  d.encryption.UserPwd,
		OwnerPassword: d.encryption.OwnerPwd,
		Permissions:   permissionsFromBits(d.encryption.Permissions),
	}, true
}

// permissionsFromBits is the inverse of permissionBits. Used by
// the Encryption accessor so callers can introspect the configured
// permissions without re-deriving them.
func permissionsFromBits(bits int32) Permissions {
	return Permissions{
		Print:             bits&(1<<2) != 0,
		Modify:            bits&(1<<3) != 0,
		Copy:              bits&(1<<4) != 0,
		Annotate:          bits&(1<<5) != 0,
		FillForms:         bits&(1<<8) != 0,
		AccessibilityCopy: bits&(1<<9) != 0,
		AssembleDocument:  bits&(1<<10) != 0,
		PrintHighRes:      bits&(1<<11) != 0,
	}
}
