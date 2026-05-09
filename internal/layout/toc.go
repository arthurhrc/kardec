package layout

import (
	"strings"

	"github.com/arthurhrc/kardec"
)

// tocPagePlaceholder is the literal string the layout emits in place
// of the page number while the TOC is being laid out for the first
// time. The post-layout pass recognises the tag and replaces it with
// the real page number once pagination is final.
const tocPagePlaceholder = "{{tocpage:"

// placeTOC reserves exact vertical space for one TOC entry per
// heading the document carries (filtered by maxLevel) and emits each
// entry as `Title . . . . . . {{tocpage:N}}`. The trailing
// placeholder is patched by patchTOCPlaceholders after every
// section has been laid out and we know the page each heading
// landed on.
func (e Engine) placeTOC(cur *pageCursor, flush func(), doc *kardec.Document, toc kardec.TableOfContents, style blockStyle, fonts FontProvider) {
	headings := collectIndexableHeadings(doc, toc.MaxLevel())
	if len(headings) == 0 {
		return
	}
	applySpaceBefore(cur, flush, style.spaceBeforePt)

	available := cur.availableWidth()
	lineHeight := style.lineHeight * style.sizePt

	for n, h := range headings {
		if cur.remainingHeight() < lineHeight {
			flush()
		}
		// Indent deeper headings so the TOC visually mirrors the
		// outline structure.
		indent := float64(maxInt(0, h.level-1)) * style.sizePt * 1.4
		// Reserve room on the right for the page number — three
		// digits at the body size keeps the dotted leader proportional.
		pagePlaceholder := tocPagePlaceholder + h.id + "}}"
		pageNumWidth := style.sizePt * 1.8
		// Body uses the available area minus the indent and the
		// reserved page-number column.
		titleX := cur.x0 + indent
		titleMaxWidth := available - indent - pageNumWidth - style.sizePt

		// Emit the title.
		titleRuns := []kardec.Run{kardec.Text(h.title)}
		titleTokens := shapeRuns(titleRuns, fonts, style, kardec.Pt(style.sizePt), style.color)
		emitInlineTokens(cur, titleTokens, titleX, cur.cursorY)

		// Compute how wide the title actually drew so we can stretch
		// a dot leader between it and the page number.
		titleWidth := tokensWidth(titleTokens)
		if titleWidth > titleMaxWidth {
			titleWidth = titleMaxWidth
		}
		dotStart := titleX + titleWidth + style.sizePt*0.5
		dotEnd := cur.x0 + available - pageNumWidth
		emitDotLeader(cur, dotStart, cur.cursorY, dotEnd-dotStart, style)

		// Emit the page-number placeholder right-aligned within the
		// reserved column.
		pageRuns := []kardec.Run{kardec.Text(pagePlaceholder)}
		pageTokens := shapeRuns(pageRuns, fonts, style, kardec.Pt(style.sizePt), style.color)
		emitInlineTokens(cur, pageTokens, cur.x0+available-tokensWidth(pageTokens), cur.cursorY)

		_ = n
		cur.cursorY += lineHeight
	}
	cur.cursorY += style.spaceAfterPt
}

// emitInlineTokens places non-whitespace tokens left-to-right at
// (startX, baselineY). Used by the TOC and chrome paths that don't
// need the line-break logic.
func emitInlineTokens(cur *pageCursor, tokens []token, startX, baselineY float64) {
	x := startX
	for _, t := range tokens {
		if t.isSpace {
			x += t.width
			continue
		}
		cur.items = append(cur.items, PlacedItem{
			X:     kardec.Pt(x),
			Y:     kardec.Pt(baselineY),
			Text:  t.text,
			Font:  t.font,
			Size:  kardec.Pt(t.sizePt),
			Color: t.color,
		})
		x += t.width
	}
}

// tokensWidth sums the widths of a token slice.
func tokensWidth(tokens []token) float64 {
	var w float64
	for _, t := range tokens {
		w += t.width
	}
	return w
}

// emitDotLeader draws a row of small dots between the title and the
// page number to guide the eye across the line. Width is the
// horizontal extent of the leader region.
func emitDotLeader(cur *pageCursor, x, y, width float64, style blockStyle) {
	if width <= 0 {
		return
	}
	dotSpacing := style.sizePt * 0.5
	cursor := x
	for cursor < x+width {
		cur.items = append(cur.items, PlacedItem{
			X: kardec.Pt(cursor),
			Y: kardec.Pt(y - style.sizePt*0.25),
			Rect: &PlacedRect{
				Width:     kardec.Pt(0.6),
				Thickness: kardec.Pt(0.6),
				Color:     style.color,
			},
		})
		cursor += dotSpacing
	}
}

// indexableHeading is an internal record of a heading the TOC will
// surface. Each carries a unique ID so the post-pass patcher can
// match the placeholder back to the right page number.
type indexableHeading struct {
	id    string
	title string
	level int
}

// collectIndexableHeadings walks every section's blocks and pulls
// out the headings whose level is within maxLevel. maxLevel of 0
// means "all levels". Each heading receives a sequential ID so the
// page-number post-pass can correlate the placeholder back to the
// matching HeadingMark on the laid-out pages.
func collectIndexableHeadings(doc *kardec.Document, maxLevel int) []indexableHeading {
	var out []indexableHeading
	idx := 0
	for _, sec := range doc.Sections() {
		for _, b := range sec.Blocks {
			h, ok := b.(kardec.Heading)
			if !ok {
				continue
			}
			if maxLevel > 0 && h.Level() > maxLevel {
				continue
			}
			out = append(out, indexableHeading{
				id:    "h" + itoa(idx),
				title: headingTitle(h),
				level: h.Level(),
			})
			idx++
		}
	}
	return out
}

// patchTOCPlaceholders walks every laid-out page and replaces each
// `{{tocpage:hN}}` placeholder text with the page number on which
// the matching heading was placed. The match walks pages in source
// order and assigns a 1-based page number to each indexable
// heading.
func patchTOCPlaceholders(pages []Page, doc *kardec.Document, maxLevel int) {
	headings := collectIndexableHeadings(doc, maxLevel)
	if len(headings) == 0 {
		return
	}
	pageOf := mapHeadingToPage(pages, maxLevel)
	for i := range pages {
		for j := range pages[i].Items {
			text := pages[i].Items[j].Text
			if !strings.Contains(text, tocPagePlaceholder) {
				continue
			}
			for hi, h := range headings {
				marker := tocPagePlaceholder + h.id + "}}"
				if !strings.Contains(text, marker) {
					continue
				}
				pn, ok := pageOf[hi]
				replacement := "?"
				if ok {
					replacement = itoa(pn)
				}
				pages[i].Items[j].Text = strings.ReplaceAll(text, marker, replacement)
				break
			}
		}
	}
}

// mapHeadingToPage produces a slice-index → 1-based-page-number map
// by walking pages.Headings in source order, filtered by maxLevel.
func mapHeadingToPage(pages []Page, maxLevel int) map[int]int {
	out := make(map[int]int)
	idx := 0
	for pageIdx, p := range pages {
		for _, h := range p.Headings {
			if maxLevel > 0 && h.Level > maxLevel {
				continue
			}
			out[idx] = pageIdx + 1
			idx++
		}
	}
	return out
}

// maxInt returns the larger of two ints.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
