package typography

import (
	"embed"
	"fmt"
)

// FontsFS exposes the bundled TTF files as an embed.FS so callers and tests
// can enumerate or read individual files. The actual font registration is
// performed by LoadBuiltinFonts.
//
// All bundled fonts are licensed under the SIL Open Font License (OFL) 1.1.
// See internal/typography/embedded/README.md for sources and license terms.
//
//go:embed embedded/*.ttf
var FontsFS embed.FS

// Family names for the bundled fonts. These string values are stable and
// part of the public API surface (callers compare against them).
const (
	FamilyLiberationSans  = "Liberation Sans"
	FamilyLiberationSerif = "Liberation Serif"
	FamilyCarlito         = "Carlito"
	FamilyJetBrainsMono   = "JetBrains Mono"
)

// builtinFace describes one bundled (family, weight, italic, file) tuple.
type builtinFace struct {
	family string
	weight Weight
	italic bool
	file   string
}

// builtinFaces is the canonical list of bundled font files. The order is
// stable so the first registered face becomes a deterministic default
// (Liberation Sans Regular).
var builtinFaces = []builtinFace{
	{FamilyLiberationSans, Regular, false, "embedded/LiberationSans-Regular.ttf"},
	{FamilyLiberationSans, Bold, false, "embedded/LiberationSans-Bold.ttf"},
	{FamilyLiberationSans, Regular, true, "embedded/LiberationSans-Italic.ttf"},
	{FamilyLiberationSans, Bold, true, "embedded/LiberationSans-BoldItalic.ttf"},

	{FamilyCarlito, Regular, false, "embedded/Carlito-Regular.ttf"},
	{FamilyCarlito, Bold, false, "embedded/Carlito-Bold.ttf"},
	{FamilyCarlito, Regular, true, "embedded/Carlito-Italic.ttf"},
	{FamilyCarlito, Bold, true, "embedded/Carlito-BoldItalic.ttf"},

	{FamilyLiberationSerif, Regular, false, "embedded/LiberationSerif-Regular.ttf"},
	{FamilyLiberationSerif, Bold, false, "embedded/LiberationSerif-Bold.ttf"},
	{FamilyLiberationSerif, Regular, true, "embedded/LiberationSerif-Italic.ttf"},
	{FamilyLiberationSerif, Bold, true, "embedded/LiberationSerif-BoldItalic.ttf"},

	{FamilyJetBrainsMono, Regular, false, "embedded/JetBrainsMono-Regular.ttf"},
	{FamilyJetBrainsMono, Bold, false, "embedded/JetBrainsMono-Bold.ttf"},
	{FamilyJetBrainsMono, Regular, true, "embedded/JetBrainsMono-Italic.ttf"},
	{FamilyJetBrainsMono, Bold, true, "embedded/JetBrainsMono-BoldItalic.ttf"},
}

// LoadBuiltinFonts registers every bundled TTF face into reg. Faces whose
// embedded payload is empty (a placeholder used while the real TTFs are
// pending) are skipped silently so callers can run on a half-stubbed bundle.
//
// On any other error (parse failure, missing file) it returns the first
// failure with a wrapping description naming the offending family.
func LoadBuiltinFonts(reg *Registry) error {
	if reg == nil {
		return fmt.Errorf("typography: nil registry")
	}
	for _, face := range builtinFaces {
		data, err := FontsFS.ReadFile(face.file)
		if err != nil {
			// Missing file: treat as not-yet-bundled. Continue so the rest
			// of the registry stays populated with what we do have.
			continue
		}
		if len(data) == 0 {
			// Placeholder file; skip until real TTF lands.
			continue
		}
		if err := reg.Register(face.family, face.weight, face.italic, data); err != nil {
			return fmt.Errorf("typography: register builtin %s: %w", face.family, err)
		}
	}
	return nil
}

// BuiltinFamilies returns the family names of every bundled font, in
// registration order. Useful for tests and diagnostics.
func BuiltinFamilies() []string {
	seen := make(map[string]bool)
	out := make([]string, 0, 4)
	for _, f := range builtinFaces {
		if seen[f.family] {
			continue
		}
		seen[f.family] = true
		out = append(out, f.family)
	}
	return out
}
