package kardec_test

import (
	"testing"

	"github.com/arthurhrc/kardec"
)

func TestImageCaptionWrapsInKeepTogether(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image(crossrefPNG(t)).
		Caption("Quarterly growth").
		Build()

	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("expected one wrapper block, got %d", len(blocks))
	}
	group, ok := blocks[0].(kardec.KeepTogether)
	if !ok {
		t.Fatalf("expected KeepTogether wrapper, got %T", blocks[0])
	}
	inner := group.Blocks()
	if len(inner) != 2 {
		t.Fatalf("expected image + caption inside group, got %d blocks", len(inner))
	}
	if _, ok := inner[0].(kardec.Image); !ok {
		t.Errorf("expected first inner block to be Image, got %T", inner[0])
	}
	if _, ok := inner[1].(kardec.Paragraph); !ok {
		t.Errorf("expected second inner block to be caption Paragraph, got %T", inner[1])
	}
}

func TestImageCaptionAutoPrefixesLabelMarker(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image(crossrefPNG(t)).
		Label("growth-2024").
		Caption("Quarterly growth").
		Build()

	blocks := doc.Sections()[0].Blocks
	group, ok := blocks[0].(kardec.KeepTogether)
	if !ok {
		t.Fatalf("expected KeepTogether wrapper, got %T", blocks[0])
	}
	inner := group.Blocks()
	if len(inner) != 3 {
		t.Fatalf("expected anchor + image + caption, got %d blocks", len(inner))
	}
	if a, ok := inner[0].(kardec.Anchor); !ok {
		t.Errorf("expected first inner block to be Anchor, got %T", inner[0])
	} else if a.Name() != kardec.RefAnchorName("growth-2024") {
		t.Errorf("anchor name = %q, want %q", a.Name(), kardec.RefAnchorName("growth-2024"))
	}
	caption, ok := inner[2].(kardec.Paragraph)
	if !ok {
		t.Fatalf("expected caption Paragraph, got %T", inner[2])
	}
	runs := caption.Runs()
	if len(runs) == 0 || runs[0].Text() != "Figure 1: " {
		t.Errorf("expected first caption run to be %q, got runs=%+v", "Figure 1: ", runs)
	}
}

func TestImageWithoutCaptionStaysBareForBackwardCompat(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image(crossrefPNG(t)).
		Build()

	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("expected one block, got %d", len(blocks))
	}
	if _, ok := blocks[0].(kardec.Image); !ok {
		t.Errorf("expected bare Image, got %T", blocks[0])
	}
}

func TestImageCaptionRunsPreservesRichContent(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Image(crossrefPNG(t)).
		Label("rev").
		CaptionRuns(kardec.Italic("Revenue"), kardec.Text(" — Q4")).
		Build()

	blocks := doc.Sections()[0].Blocks
	group := blocks[0].(kardec.KeepTogether)
	caption := group.Blocks()[2].(kardec.Paragraph)
	runs := caption.Runs()
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs (marker + 2 user runs), got %d: %+v", len(runs), runs)
	}
	if runs[0].Text() != "Figure 1: " {
		t.Errorf("marker run = %q", runs[0].Text())
	}
	if !runs[1].Italic() {
		t.Errorf("italic preservation lost on first user run")
	}
}
