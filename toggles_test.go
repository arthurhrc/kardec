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

