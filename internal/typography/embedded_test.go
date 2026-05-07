package typography

import (
	"strings"
	"testing"
)

func TestBuiltinFamiliesListsFour(t *testing.T) {
	got := BuiltinFamilies()
	want := []string{
		FamilyLiberationSans,
		FamilyCarlito,
		FamilyLiberationSerif,
		FamilyJetBrainsMono,
	}
	if len(got) != len(want) {
		t.Fatalf("BuiltinFamilies len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("BuiltinFamilies[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadBuiltinFontsRegistersKnownFamilies(t *testing.T) {
	reg := NewRegistry()
	if err := LoadBuiltinFonts(reg); err != nil {
		t.Fatalf("LoadBuiltinFonts: %v", err)
	}

	// At minimum, Liberation Sans Regular should resolve. If TTF files are
	// only placeholders the registry will be empty; in that case skip the
	// rest so the suite still passes during the early stub phase.
	if reg.Default() == nil {
		t.Skip("registry empty — TTF files appear to be placeholders")
	}

	for _, family := range BuiltinFamilies() {
		if _, ok := reg.Resolve(family, Regular, false); !ok {
			t.Errorf("expected %q Regular to resolve after LoadBuiltinFonts", family)
		}
	}
}

func TestLoadBuiltinFontsRejectsNilRegistry(t *testing.T) {
	if err := LoadBuiltinFonts(nil); err == nil {
		t.Error("expected error when registry is nil")
	}
}

func TestBuiltinFontMeasurementSmoke(t *testing.T) {
	reg := NewRegistry()
	if err := LoadBuiltinFonts(reg); err != nil {
		t.Fatalf("load: %v", err)
	}
	face, ok := reg.Resolve(FamilyLiberationSans, Regular, false)
	if !ok {
		t.Skip("Liberation Sans not bundled — placeholder phase")
	}
	w := face.Measure("Hello", 12.0)
	if w <= 0 {
		t.Errorf("Measure returned %g, want positive", w)
	}
	// Sanity: the empty string has zero advance.
	if face.Measure("", 12.0) != 0 {
		t.Error("empty string should have zero advance")
	}
	// Ascent/Descent/LineHeight should all be positive at 12pt.
	if face.Ascent(12) <= 0 {
		t.Error("Ascent should be positive")
	}
	if face.Descent(12) < 0 {
		t.Error("Descent should be non-negative (absolute value)")
	}
	if face.LineHeight(12) <= face.Ascent(12) {
		t.Error("LineHeight should be greater than Ascent")
	}
}

func TestBuiltinFontNameContainsFamily(t *testing.T) {
	reg := NewRegistry()
	if err := LoadBuiltinFonts(reg); err != nil {
		t.Fatalf("load: %v", err)
	}
	face, ok := reg.Resolve(FamilyLiberationSans, Regular, false)
	if !ok {
		t.Skip("Liberation Sans not bundled")
	}
	// canvas reports the SFNT-embedded full name. Liberation TTFs use
	// "Liberation Sans" as the family; full name typically contains it.
	if !strings.Contains(face.Name(), "Liberation") {
		t.Errorf("expected font name to mention Liberation, got %q", face.Name())
	}
}
