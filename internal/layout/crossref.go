package layout

import (
	"strings"

	"github.com/arthurhrc/kardec"
)

// patchRefPagesAcrossSections walks every laid-out page and replaces
// each `{{refpage:<label>}}` placeholder with the page number on which
// the matching `kardec-ref-<label>` anchor landed. Mirrors the TOC's
// `{{tocpage:hN}}` post-pass.
//
// Anchors that the document never declared resolve to "?" so the
// missing reference is visually conspicuous without breaking layout.
func patchRefPagesAcrossSections(pages []Page) {
	if len(pages) == 0 {
		return
	}
	pageOf := mapRefAnchorsToPage(pages)
	for i := range pages {
		for j := range pages[i].Items {
			text := pages[i].Items[j].Text
			if !strings.Contains(text, kardec.RefPagePlaceholder) {
				continue
			}
			pages[i].Items[j].Text = substituteRefPages(text, pageOf)
		}
	}
}

// mapRefAnchorsToPage builds a label → 1-based-page-number map by
// walking every page's AnchorMark slice. Only anchors carrying the
// kardec.RefAnchorPrefix name participate; user-supplied anchors are
// left out so they cannot collide with cross-reference labels.
func mapRefAnchorsToPage(pages []Page) map[string]int {
	out := make(map[string]int)
	for i, p := range pages {
		for _, a := range p.Anchors {
			if !strings.HasPrefix(a.Name, kardec.RefAnchorPrefix) {
				continue
			}
			label := strings.TrimPrefix(a.Name, kardec.RefAnchorPrefix)
			if _, exists := out[label]; exists {
				continue
			}
			out[label] = i + 1
		}
	}
	return out
}

// substituteRefPages replaces every `{{refpage:<label>}}` occurrence
// in text with the matching page number from pageOf, or "?" when the
// label has no recorded anchor.
func substituteRefPages(text string, pageOf map[string]int) string {
	for {
		start := strings.Index(text, kardec.RefPagePlaceholder)
		if start < 0 {
			return text
		}
		end := strings.Index(text[start:], "}}")
		if end < 0 {
			return text
		}
		end += start
		label := text[start+len(kardec.RefPagePlaceholder) : end]
		replacement := "?"
		if pn, ok := pageOf[label]; ok {
			replacement = itoa(pn)
		}
		text = text[:start] + replacement + text[end+2:]
	}
}
