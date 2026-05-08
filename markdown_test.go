package kardec

import "testing"

// firstParagraphRuns extracts the runs of the first paragraph block in
// the document's first section. Returns nil if no paragraph is present.
func firstParagraphRuns(t *testing.T, d *Document) []Run {
	t.Helper()
	for _, sec := range d.Sections() {
		for _, b := range sec.Blocks {
			if p, ok := b.(Paragraph); ok {
				return p.Runs()
			}
		}
	}
	return nil
}

func TestAppendMarkdownHeadingAndParagraph(t *testing.T) {
	doc := New(PageA4, MarginsNormal).AppendMarkdown(`# Title

Body text here.`)

	if err := doc.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}
	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("want 2 blocks (heading + paragraph), got %d", len(blocks))
	}
	h, ok := blocks[0].(Heading)
	if !ok {
		t.Fatalf("first block should be Heading, got %T", blocks[0])
	}
	if h.Level() != 1 {
		t.Errorf("heading level = %d, want 1", h.Level())
	}
	if got := h.Runs()[0].Text(); got != "Title" {
		t.Errorf("heading text = %q, want %q", got, "Title")
	}
	if got := blocks[1].(Paragraph).Runs()[0].Text(); got != "Body text here." {
		t.Errorf("paragraph text = %q, want %q", got, "Body text here.")
	}
}

func TestAppendMarkdownEmphasis(t *testing.T) {
	doc := New(PageA4, MarginsNormal).AppendMarkdown(`Sales grew **12%** vs *last year*.`)

	runs := firstParagraphRuns(t, doc)
	if len(runs) < 4 {
		t.Fatalf("want at least 4 runs, got %d", len(runs))
	}
	// Find the bold and italic runs.
	var sawBold, sawItalic bool
	for _, r := range runs {
		if r.Text() == "12%" && r.Bold() && !r.Italic() {
			sawBold = true
		}
		if r.Text() == "last year" && r.Italic() && !r.Bold() {
			sawItalic = true
		}
	}
	if !sawBold {
		t.Errorf("expected a bold run with text %q", "12%")
	}
	if !sawItalic {
		t.Errorf("expected an italic run with text %q", "last year")
	}
}

func TestAppendMarkdownThematicBreakBecomesPageBreak(t *testing.T) {
	doc := New(PageA4, MarginsNormal).AppendMarkdown(`first

---

second`)
	blocks := doc.Sections()[0].Blocks
	var pageBreaks int
	for _, b := range blocks {
		if _, ok := b.(PageBreak); ok {
			pageBreaks++
		}
	}
	if pageBreaks != 1 {
		t.Errorf("want 1 PageBreak, got %d (%+v)", pageBreaks, blocks)
	}
}

func TestAppendMarkdownUnorderedList(t *testing.T) {
	doc := New(PageA4, MarginsNormal).AppendMarkdown(`- Alpha
- Beta
- Gamma`)
	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("want 1 List block, got %d (%+v)", len(blocks), blocks)
	}
	list, ok := blocks[0].(List)
	if !ok {
		t.Fatalf("want List, got %T", blocks[0])
	}
	if list.Style() != ListUnordered {
		t.Errorf("want ListUnordered, got %v", list.Style())
	}
	items := list.Items()
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}
	for i, expected := range []string{"Alpha", "Beta", "Gamma"} {
		if len(items[i].Runs) == 0 || items[i].Runs[0].Text() != expected {
			t.Errorf("item %d runs = %+v, want first run %q", i, items[i].Runs, expected)
		}
	}
}

func TestAppendMarkdownOrderedList(t *testing.T) {
	doc := New(PageA4, MarginsNormal).AppendMarkdown(`1. First
2. Second`)
	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("want 1 List block, got %d", len(blocks))
	}
	list := blocks[0].(List)
	if list.Style() != ListOrdered {
		t.Errorf("want ListOrdered, got %v", list.Style())
	}
	if got := list.Items(); len(got) != 2 {
		t.Errorf("want 2 items, got %d", len(got))
	}
}

func TestAppendMarkdownPreservesDeferredError(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.fail(errInternalForTest())
	doc.AppendMarkdown(`# ignored`)
	if got := doc.Sections()[0].Blocks; len(got) != 0 {
		t.Errorf("AppendMarkdown should be inert after a captured error, got %d blocks", len(got))
	}
}

func errInternalForTest() error {
	return &simpleErr{msg: "synthetic"}
}

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }
