package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSplitFlagsAndPositional_FlagsAfterPositional(t *testing.T) {
	flags, pos := splitFlagsAndPositional([]string{"input.md", "-o", "out.pdf"})
	if len(pos) != 1 || pos[0] != "input.md" {
		t.Errorf("positional = %v, want [input.md]", pos)
	}
	if len(flags) != 2 || flags[0] != "-o" || flags[1] != "out.pdf" {
		t.Errorf("flags = %v, want [-o out.pdf]", flags)
	}
}

func TestSplitFlagsAndPositional_FlagsFirst(t *testing.T) {
	flags, pos := splitFlagsAndPositional([]string{"-o", "out.pdf", "input.md"})
	if len(pos) != 1 || pos[0] != "input.md" {
		t.Errorf("positional = %v, want [input.md]", pos)
	}
	if len(flags) != 2 {
		t.Errorf("flags = %v, want 2 entries", flags)
	}
}

func TestSplitFlagsAndPositional_EqualsForm(t *testing.T) {
	flags, pos := splitFlagsAndPositional([]string{"-o=out.pdf", "input.md"})
	if len(flags) != 1 || flags[0] != "-o=out.pdf" {
		t.Errorf("equals-form should land as one token, got %v", flags)
	}
	if len(pos) != 1 || pos[0] != "input.md" {
		t.Errorf("positional = %v, want [input.md]", pos)
	}
}

func TestRenderMarkdown_WritesPDF(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.md")
	out := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(in, []byte("# Hello\n\nBody."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := renderMarkdown(in, out); err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("expected output file, got %v", err)
	}
	if info.Size() < 1000 {
		t.Errorf("PDF suspiciously small: %d bytes", info.Size())
	}
}

func TestRenderTemplate_WithJSONData(t *testing.T) {
	dir := t.TempDir()
	tpl := filepath.Join(dir, "tpl.md")
	data := filepath.Join(dir, "data.json")
	out := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(tpl, []byte("# Invoice {{.ID}}\n\n**Total:** R$ {{.Total}}"), 0o644); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"ID": 42, "Total": "1234.56"}
	raw, _ := json.Marshal(payload)
	if err := os.WriteFile(data, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := renderTemplate(tpl, data, out); err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected templated output, got %v", err)
	}
}

func TestRunRender_RejectsMissingInput(t *testing.T) {
	err := runRender([]string{}) // no positional, no -t
	if err == nil {
		t.Errorf("expected an error when no input is provided")
	}
}
