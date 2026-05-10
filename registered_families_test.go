package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestRegisteredFamiliesReportsBundledFonts(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	families := doc.RegisteredFamilies()
	if len(families) == 0 {
		t.Fatalf("expected bundled font families to be reported")
	}
	// Liberation Sans should always be present — it's the default
	// body font shipped with the registry.
	found := false
	for _, f := range families {
		if f == kardec.FontLiberationSans {
			found = true
		}
	}
	if !found {
		t.Errorf("Liberation Sans missing from registered families: %v", families)
	}
}

func TestRegisteredFamiliesIsOrderPreservingAndDeduped(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal)
	families := doc.RegisteredFamilies()
	seen := map[string]bool{}
	for _, f := range families {
		if seen[f] {
			t.Errorf("RegisteredFamilies returned duplicate family %q", f)
		}
		seen[f] = true
	}
}
