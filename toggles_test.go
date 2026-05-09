package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestEnablePDFATurnsFlagOn(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).EnablePDFA()
	if !doc.PDFAEnabled() {
		t.Errorf("EnablePDFA should set the flag")
	}
}

func TestDisablePDFAClearsFlag(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		EnablePDFA().
		DisablePDFA()
	if doc.PDFAEnabled() {
		t.Errorf("DisablePDFA should clear the flag")
	}
}

func TestEnableFontSubsettingTurnsFlagOn(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).EnableFontSubsetting()
	if !doc.FontSubsetEnabled() {
		t.Errorf("EnableFontSubsetting should set the flag")
	}
}

func TestDisableFontSubsettingClearsFlag(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		EnableFontSubsetting().
		DisableFontSubsetting()
	if doc.FontSubsetEnabled() {
		t.Errorf("DisableFontSubsetting should clear the flag")
	}
}

// Deprecated forms still work for v0.x consumers — drop these
// expectations when the legacy methods are removed at v1.0.
func TestDeprecatedPDFAStillToggles(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).PDFA()
	if !doc.PDFAEnabled() {
		t.Errorf("legacy PDFA() should still enable for v0.x callers")
	}
	doc.PDFA(false)
	if doc.PDFAEnabled() {
		t.Errorf("legacy PDFA(false) should still disable for v0.x callers")
	}
}

func TestDeprecatedSubsetFontsStillToggles(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).SubsetFonts()
	if !doc.FontSubsetEnabled() {
		t.Errorf("legacy SubsetFonts() should still enable for v0.x callers")
	}
	doc.SubsetFonts(false)
	if doc.FontSubsetEnabled() {
		t.Errorf("legacy SubsetFonts(false) should still disable")
	}
}
