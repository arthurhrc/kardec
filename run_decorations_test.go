package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestUnderlineRun(t *testing.T) {
	r := kardec.Underline("link-ish")
	if r.Text() != "link-ish" {
		t.Errorf("Text = %q, want %q", r.Text(), "link-ish")
	}
	if !r.Underline() {
		t.Errorf("Underline() should be true")
	}
	if r.Strikethrough() {
		t.Errorf("Strikethrough() should be false")
	}
}

func TestStrikethroughRun(t *testing.T) {
	r := kardec.Strikethrough("retracted")
	if !r.Strikethrough() {
		t.Errorf("Strikethrough() should be true")
	}
	if r.Underline() {
		t.Errorf("Underline() should be false")
	}
}

func TestWithUnderlineDecoratesExistingRun(t *testing.T) {
	r := kardec.WithUnderline(kardec.Bold("emphasis"))
	if !r.Bold() {
		t.Errorf("WithUnderline must preserve Bold")
	}
	if !r.Underline() {
		t.Errorf("WithUnderline must set Underline")
	}
}

func TestWithStrikethroughDecoratesExistingRun(t *testing.T) {
	r := kardec.WithStrikethrough(kardec.Italic("retracted"))
	if !r.Italic() {
		t.Errorf("WithStrikethrough must preserve Italic")
	}
	if !r.Strikethrough() {
		t.Errorf("WithStrikethrough must set Strikethrough")
	}
}

func TestPlainRunHasNoDecorations(t *testing.T) {
	r := kardec.Text("plain")
	if r.Underline() || r.Strikethrough() {
		t.Errorf("plain Run should not carry decorations: %+v", r)
	}
}
