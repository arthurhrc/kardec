package render

import (
	"github.com/arthurhrc/kardec/internal/layout"
	"github.com/arthurhrc/kardec/internal/pdf"
)

// buildOutline walks every Page's HeadingMark slice and assembles the
// PDF outline tree. Heading levels translate into nesting depth: H1
// becomes a top-level entry, H2 nests under the closest preceding H1,
// and so on. Out-of-order jumps (e.g. H1 followed directly by H3) are
// tolerated — the missing levels are skipped silently rather than
// inventing intermediate placeholders.
//
// PageIndex on each entry indexes back into pdf.Document.Pages so the
// outline /Dest can resolve through pageIDs. Y is converted from
// layout's top-left to PDF's bottom-left at this point so the writer
// embeds it directly.
func buildOutline(pages []layout.Page) []pdf.OutlineEntry {
	type frame struct {
		level   int
		entries *[]pdf.OutlineEntry
	}
	root := []pdf.OutlineEntry{}
	stack := []frame{{level: 0, entries: &root}}

	for pageIdx, p := range pages {
		pageHeight := p.Size.Height.Points()
		for _, h := range p.Headings {
			entry := pdf.OutlineEntry{
				Title:     h.Title,
				PageIndex: pageIdx,
				Y:         pageHeight - h.Y.Points(),
			}
			// Pop until the top of the stack is at a strictly lower
			// level than the new heading. The new entry then attaches
			// as a child of whatever frame is on top.
			for len(stack) > 1 && stack[len(stack)-1].level >= h.Level {
				stack = stack[:len(stack)-1]
			}
			parent := stack[len(stack)-1].entries
			*parent = append(*parent, entry)
			// Push the slot the new entry now occupies so future
			// deeper headings nest under it. We push the address of
			// the freshly appended entry's Children field.
			top := &(*parent)[len(*parent)-1]
			stack = append(stack, frame{level: h.Level, entries: &top.Children})
		}
	}
	return root
}
