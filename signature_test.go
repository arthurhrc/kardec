package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestSignatureBlockEmitsRuleNameRole(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Signature("Jane Doe", "Lead Engineer")

	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("expected 1 wrapper block, got %d", len(blocks))
	}
	group, ok := blocks[0].(kardec.KeepTogether)
	if !ok {
		t.Fatalf("expected KeepTogether wrapper, got %T", blocks[0])
	}
	inner := group.Blocks()
	if len(inner) != 3 {
		t.Fatalf("expected rule + name + role, got %d blocks", len(inner))
	}
	if _, ok := inner[0].(kardec.HorizontalRule); !ok {
		t.Errorf("expected HorizontalRule first, got %T", inner[0])
	}
	name, ok := inner[1].(kardec.Paragraph)
	if !ok {
		t.Fatalf("expected name Paragraph, got %T", inner[1])
	}
	if name.Runs()[0].Text() != "Jane Doe" {
		t.Errorf("name = %q, want %q", name.Runs()[0].Text(), "Jane Doe")
	}
	role := inner[2].(kardec.Paragraph)
	if !role.Runs()[0].Italic() {
		t.Errorf("role run should be italic")
	}
}

func TestSignatureBlockOmitsRoleWhenEmpty(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Signature("Jane Doe", "")

	group := doc.Sections()[0].Blocks[0].(kardec.KeepTogether)
	if len(group.Blocks()) != 2 {
		t.Errorf("empty role should produce 2 inner blocks, got %d", len(group.Blocks()))
	}
}
