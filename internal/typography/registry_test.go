package typography

import (
	"testing"
)

// loadSampleTTF returns a small TTF byte slice from the embedded bundle so
// registry tests do not need to fetch from disk. Liberation Sans Regular is
// guaranteed to exist by embedded.go's go:embed directive.
func loadSampleTTF(t *testing.T) []byte {
	t.Helper()
	data, err := FontsFS.ReadFile("embedded/LiberationSans-Regular.ttf")
	if err != nil {
		t.Skipf("sample TTF not bundled: %v", err)
	}
	if len(data) == 0 {
		t.Skip("sample TTF is a placeholder; skipping measurement-based assertions")
	}
	return data
}

func TestRegistryRegisterAndResolveRoundTrip(t *testing.T) {
	data := loadSampleTTF(t)
	reg := NewRegistry()
	if err := reg.Register("Sample", Regular, false, data); err != nil {
		t.Fatalf("register: %v", err)
	}
	got, ok := reg.Resolve("Sample", Regular, false)
	if !ok {
		t.Fatal("expected resolve to succeed for registered face")
	}
	if got.Name() == "" {
		t.Error("font Name should not be empty")
	}
}

func TestRegistryRejectsEmptyInputs(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register("", Regular, false, []byte{0x01}); err == nil {
		t.Error("expected error for empty family")
	}
	if err := reg.Register("X", Regular, false, nil); err == nil {
		t.Error("expected error for nil font bytes")
	}
}

func TestRegistryWeightFallback(t *testing.T) {
	data := loadSampleTTF(t)
	reg := NewRegistry()
	// Register only Regular; resolving Medium should fall back to it.
	if err := reg.Register("Sample", Regular, false, data); err != nil {
		t.Fatalf("register regular: %v", err)
	}
	got, ok := reg.Resolve("Sample", Medium, false)
	if !ok {
		t.Fatal("expected medium to fall back to regular")
	}
	want, _ := reg.Resolve("Sample", Regular, false)
	if got != want {
		t.Error("medium-fallback should return the registered Regular face")
	}
}

func TestRegistryItalicFallback(t *testing.T) {
	data := loadSampleTTF(t)
	reg := NewRegistry()
	if err := reg.Register("Sample", Regular, false, data); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, ok := reg.Resolve("Sample", Regular, true); !ok {
		t.Error("italic request should fall back to upright when italic missing")
	}
}

func TestRegistryUnknownFamilyMisses(t *testing.T) {
	reg := NewRegistry()
	if _, ok := reg.Resolve("Nope", Regular, false); ok {
		t.Error("unknown family should not resolve")
	}
}

func TestRegistryFirstRegisteredBecomesDefault(t *testing.T) {
	data := loadSampleTTF(t)
	reg := NewRegistry()
	if reg.Default() != nil {
		t.Fatal("empty registry must have nil default")
	}
	if err := reg.Register("First", Regular, false, data); err != nil {
		t.Fatalf("register: %v", err)
	}
	if reg.Default() == nil {
		t.Error("expected first registered face to become default")
	}
}

func TestRegistrySetDefault(t *testing.T) {
	data := loadSampleTTF(t)
	reg := NewRegistry()
	if err := reg.Register("A", Regular, false, data); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register("B", Bold, false, data); err != nil {
		t.Fatal(err)
	}
	if !reg.SetDefault("B", Bold, false) {
		t.Fatal("SetDefault should accept a registered face")
	}
	if reg.SetDefault("C", Regular, false) {
		t.Error("SetDefault should reject an unregistered face")
	}
}

func TestRegistryFamiliesPreservesInsertionOrder(t *testing.T) {
	data := loadSampleTTF(t)
	reg := NewRegistry()
	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		if err := reg.Register(name, Regular, false, data); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}
	got := reg.Families()
	want := []string{"Alpha", "Beta", "Gamma"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Families[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWeightCSSAndString(t *testing.T) {
	cases := []struct {
		w   Weight
		css int
		s   string
	}{
		{Regular, 400, "Regular"},
		{Medium, 500, "Medium"},
		{SemiBold, 600, "SemiBold"},
		{Bold, 700, "Bold"},
		{Black, 900, "Black"},
	}
	for _, c := range cases {
		if got := c.w.CSS(); got != c.css {
			t.Errorf("%s.CSS() = %d, want %d", c.s, got, c.css)
		}
		if got := c.w.String(); got != c.s {
			t.Errorf("Weight String mismatch: got %q want %q", got, c.s)
		}
	}
}
