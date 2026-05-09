package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestLeaderAppendsBlock(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Leader([]kardec.Run{kardec.Text("Skill")}, []kardec.Run{kardec.Text("80%")})

	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	l, ok := blocks[0].(kardec.Leader)
	if !ok {
		t.Fatalf("expected Leader, got %T", blocks[0])
	}
	if got := l.Left()[0].Text(); got != "Skill" {
		t.Errorf("left = %q, want %q", got, "Skill")
	}
	if got := l.Right()[0].Text(); got != "80%" {
		t.Errorf("right = %q, want %q", got, "80%")
	}
}

func TestLeaderAcceptsEmptySides(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Leader([]kardec.Run{kardec.Text("only-left")}, nil)
	l := doc.Sections()[0].Blocks[0].(kardec.Leader)
	if len(l.Right()) != 0 {
		t.Errorf("expected empty right side, got %v", l.Right())
	}
}
