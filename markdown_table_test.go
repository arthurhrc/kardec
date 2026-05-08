package kardec

import "testing"

const markdownTableSource = `# Q4 figures

| Month    | Revenue   | Δ    |
|:---------|----------:|:----:|
| October  | R$ 12,000 | +5%  |
| November | R$ 14,500 | +12% |
| December | R$ 13,200 | -9%  |

End of report.`

func firstTable(t *testing.T, d *Document) Table {
	t.Helper()
	for _, sec := range d.Sections() {
		for _, b := range sec.Blocks {
			if tbl, ok := b.(Table); ok {
				return tbl
			}
		}
	}
	t.Fatalf("no Table block in document")
	return Table{}
}

func TestAppendMarkdownTableProducesTableBlock(t *testing.T) {
	doc := New(PageA4, MarginsNormal).AppendMarkdown(markdownTableSource)
	if err := doc.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}
	tbl := firstTable(t, doc)

	if cols := tbl.Columns(); len(cols) != 3 {
		t.Errorf("want 3 columns, got %d", len(cols))
	}
	rows := tbl.Rows()
	if len(rows) != 4 {
		t.Errorf("want 4 rows (header + 3 data), got %d", len(rows))
	}
	if !tbl.RepeatHeader() {
		t.Error("Markdown tables should default to RepeatHeader")
	}
}

func TestAppendMarkdownTableHonorsAlignmentSyntax(t *testing.T) {
	doc := New(PageA4, MarginsNormal).AppendMarkdown(markdownTableSource)
	tbl := firstTable(t, doc)
	cols := tbl.Columns()
	if got := cols[0].Alignment; got != AlignLeft {
		t.Errorf("col 0 alignment = %v, want AlignLeft", got)
	}
	if got := cols[1].Alignment; got != AlignRight {
		t.Errorf("col 1 alignment = %v, want AlignRight", got)
	}
	if got := cols[2].Alignment; got != AlignCenter {
		t.Errorf("col 2 alignment = %v, want AlignCenter", got)
	}
}

func TestAppendMarkdownTableHeaderCellsAreBold(t *testing.T) {
	doc := New(PageA4, MarginsNormal).AppendMarkdown(markdownTableSource)
	tbl := firstTable(t, doc)
	header := tbl.Rows()[0]
	for i, cell := range header.Cells {
		if len(cell.Runs) == 0 {
			t.Fatalf("header cell %d empty", i)
		}
		if !cell.Runs[0].Bold() {
			t.Errorf("header cell %d run is not bold (run = %+v)", i, cell.Runs[0])
		}
	}
}

func TestAppendMarkdownTableInlineEmphasisInDataRows(t *testing.T) {
	src := `| Field | Value |
|-------|-------|
| Note  | **important** insight |
`
	doc := New(PageA4, MarginsNormal).AppendMarkdown(src)
	tbl := firstTable(t, doc)
	dataRow := tbl.Rows()[1] // header is row 0
	cell := dataRow.Cells[1]
	var sawBold bool
	for _, r := range cell.Runs {
		if r.Bold() && r.Text() == "important" {
			sawBold = true
		}
	}
	if !sawBold {
		t.Errorf("expected a bold run carrying %q in cell, got %+v", "important", cell.Runs)
	}
}
