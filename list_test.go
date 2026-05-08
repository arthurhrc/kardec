package kardec

import "testing"

func TestListBuilderAppendsBlock(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		List().
		Item(Text("Alpha")).
		Item(Text("Beta")).
		Build()

	if err := doc.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}
	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("want 1 block, got %d", len(blocks))
	}
	list, ok := blocks[0].(List)
	if !ok {
		t.Fatalf("first block should be List, got %T", blocks[0])
	}
	if list.Style() != ListUnordered {
		t.Errorf("Style = %v, want ListUnordered", list.Style())
	}
	if len(list.Items()) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(list.Items()))
	}
}

func TestOrderedListBuilderSetsStyle(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		OrderedList().
		Item(Text("First")).
		Item(Text("Second")).
		Build()

	list := doc.Sections()[0].Blocks[0].(List)
	if list.Style() != ListOrdered {
		t.Errorf("Style = %v, want ListOrdered", list.Style())
	}
}

func TestListBuilderEmptyIsNoOp(t *testing.T) {
	doc := New(PageA4, MarginsNormal).List().Build()
	if got := doc.Sections()[0].Blocks; len(got) != 0 {
		t.Errorf("empty list should append nothing, got %d blocks", len(got))
	}
}

func TestListBuilderNestedCarriesChildren(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		List().
		Nested(
			[]Run{Text("Top")},
			SubList(ListUnordered, ListItem{Runs: []Run{Text("Inner")}}),
		).
		Build()

	list := doc.Sections()[0].Blocks[0].(List)
	items := list.Items()
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if got := items[0].Runs[0].Text(); got != "Top" {
		t.Errorf("top runs[0] = %q, want %q", got, "Top")
	}
	if len(items[0].Children) != 1 {
		t.Fatalf("want 1 nested child, got %d", len(items[0].Children))
	}
	if got := items[0].Children[0].Items()[0].Runs[0].Text(); got != "Inner" {
		t.Errorf("nested child first run = %q, want %q", got, "Inner")
	}
}
