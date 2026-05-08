package kardec

import (
	"strings"
	"testing"
)

func TestWarningsEmptyByDefault(t *testing.T) {
	doc := New(PageA4, MarginsNormal).Paragraph(Text("body"))
	if got := doc.Warnings(); len(got) != 0 {
		t.Errorf("fresh document should carry no warnings, got %d (%+v)", len(got), got)
	}
}

func TestAppendMarkdownWarnsOnInlineImage(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		AppendMarkdown(`See ![logo](https://example.com/logo.png) here.`)
	if !hasWarningContaining(doc, "inline image") {
		t.Errorf("expected inline-image warning, got %+v", doc.Warnings())
	}
}

func TestAppendMarkdownWarnsOnAutolink(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		AppendMarkdown(`Visit <https://example.com> please.`)
	if !hasWarningContaining(doc, "autolink") {
		t.Errorf("expected autolink warning, got %+v", doc.Warnings())
	}
}

func TestAppendMarkdownWarnsOnRawHTML(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		AppendMarkdown(`<div class="banner">html block</div>

paragraph`)
	if !hasWarningContaining(doc, "HTML") {
		t.Errorf("expected raw-HTML warning, got %+v", doc.Warnings())
	}
}

func TestAppendMarkdownWarnsOnEmptyLinkDestination(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		AppendMarkdown(`See [the docs]() here.`)
	if !hasWarningContaining(doc, "empty destination") {
		t.Errorf("expected empty-destination warning, got %+v", doc.Warnings())
	}
}

func TestAppendMarkdownNoWarningsForCleanInput(t *testing.T) {
	doc := New(PageA4, MarginsNormal).
		AppendMarkdown(`# Title

A simple paragraph with **bold** and *italic*.

- item one
- item two
`)
	if got := doc.Warnings(); len(got) != 0 {
		t.Errorf("clean Markdown should produce no warnings, got %+v", got)
	}
}

func hasWarningContaining(doc *Document, needle string) bool {
	for _, w := range doc.Warnings() {
		if strings.Contains(w, needle) {
			return true
		}
	}
	return false
}
