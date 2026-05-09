package kardec_test

import (
	"strings"
	"testing"

	"github.com/arthurhrc/kardec"
)

func clausePrefixOf(b kardec.Block) string {
	p, ok := b.(kardec.Paragraph)
	if !ok {
		return ""
	}
	runs := p.Runs()
	if len(runs) == 0 {
		return ""
	}
	return strings.TrimSpace(runs[0].Text())
}

func TestClauseAutoNumberingTopLevel(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Clause(1, kardec.Text("Definitions")).
		Clause(1, kardec.Text("Obligations")).
		Clause(1, kardec.Text("Termination"))

	blocks := doc.Sections()[0].Blocks
	wants := []string{"1.", "2.", "3."}
	for i, want := range wants {
		if got := clausePrefixOf(blocks[i]); got != want {
			t.Errorf("blocks[%d] prefix = %q, want %q", i, got, want)
		}
	}
}

func TestClauseAutoNumberingNested(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Clause(1, kardec.Text("Definitions")).
		Clause(2, kardec.Text("Confidential Info")).
		Clause(2, kardec.Text("Term")).
		Clause(1, kardec.Text("Obligations")).
		Clause(2, kardec.Text("Disclosure"))

	blocks := doc.Sections()[0].Blocks
	wants := []string{"1.", "1.1", "1.2", "2.", "2.1"}
	for i, want := range wants {
		if got := clausePrefixOf(blocks[i]); got != want {
			t.Errorf("blocks[%d] prefix = %q, want %q", i, got, want)
		}
	}
}

func TestClauseLevelsAreClampedAndDeepReset(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Clause(0, kardec.Text("clamped")).        // becomes level 1
		Clause(2, kardec.Text("first sub")).      // 1.1
		Clause(3, kardec.Text("deeper sub")).     // 1.1.1
		Clause(2, kardec.Text("back up to two")). // 1.2 — depth 3 should reset
		Clause(3, kardec.Text("new deeper sub"))  // 1.2.1

	blocks := doc.Sections()[0].Blocks
	wants := []string{"1.", "1.1", "1.1.1", "1.2", "1.2.1"}
	for i, want := range wants {
		if got := clausePrefixOf(blocks[i]); got != want {
			t.Errorf("blocks[%d] prefix = %q, want %q", i, got, want)
		}
	}
}

func TestClauseAtUsesExplicitLabel(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		ClauseAt("4.7.1", kardec.Text("Force majeure"))
	if got := clausePrefixOf(doc.Sections()[0].Blocks[0]); got != "4.7.1" {
		t.Errorf("explicit label = %q, want %q", got, "4.7.1")
	}
}

func TestClausePrefixIsBold(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Clause(1, kardec.Text("Definitions"))
	p := doc.Sections()[0].Blocks[0].(kardec.Paragraph)
	if !p.Runs()[0].Bold() {
		t.Errorf("clause prefix should be bold for visual emphasis")
	}
}
