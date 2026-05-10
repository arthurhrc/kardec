package pdf

import (
	"bytes"
	"fmt"
)

// PDF/UA tagging — lite scaffolding.
//
// This file emits the minimum object graph PDF/UA-1 readers expect
// when the writer is in tagged mode (Document.Tagged == true):
//
//   /Catalog        — adds /MarkInfo /Marked true, /Lang, /StructTreeRoot
//   /Page           — adds /StructParents N (index into ParentTree.Nums)
//   StructTreeRoot  — root of the structure hierarchy
//   ParentTree      — number tree mapping per-page StructParents indices
//                     to the StructElem(s) that own each MCID
//   StructElem (×N) — one per page, role /P, kid /K [0] referencing MCID 0
//
// Each page's content stream is wrapped in a single marked-content
// sequence by wrapMarkedContent: `/P << /MCID 0 >> BDC ... EMC`. v0.17.0
// uses one MCID per page (every glyph on the page is part of the same
// /P element). v0.17.x will refine the classification so headings,
// paragraphs, and figures get distinct elements with their own MCIDs.

// wrapMarkedContent surrounds a page's content-stream bytes with the
// `BDC ... EMC` operators that bind the glyphs underneath to the
// owning structure element. role is the structure type tag (`P`,
// `H1`, `Figure`, …); mcid is the marked-content identifier the
// StructElem references via its /K array.
func wrapMarkedContent(raw []byte, role string, mcid int) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "/%s << /MCID %d >> BDC\n", role, mcid)
	buf.Write(raw)
	buf.WriteString("EMC\n")
	return buf.Bytes()
}

// emitStructTree writes the StructTreeRoot, ParentTree, and per-block
// StructElem objects. pageIDs and pageBlockElemIDs are aligned: for
// page i, pageBlockElemIDs[i] holds one indirect ID per
// pages[i].StructBlocks entry (pre-allocated by Write so each /Page
// dict could forward-reference them through /StructParents).
//
// Pages with no StructBlocks fall back to a single "P" element
// covering the whole page (lite mode). The block-aware path
// honours per-block roles (H1..H6, P, Figure, …) so PDF/UA
// validators see real semantics instead of a flat sequence of P.
//
// Returns the StructTreeRoot ID; the catalog references it through
// /StructTreeRoot N.
func emitStructTree(ow *objectWriter, pageIDs []int, pages []Page, pageBlockElemIDs [][]int) int {
	rootID := ow.allocID()
	parentTreeID := ow.allocID()

	// Per-block StructElem objects. /K carries the MCID(s) the
	// element owns within the page; /Pg points at the page so the
	// reader knows where to look. Multi-MCID blocks are unusual at
	// this stage — every paragraph / heading is a single MCID —
	// so /K is `[mcid]` (single integer wrapped in an array).
	for pageIdx, blockIDs := range pageBlockElemIDs {
		page := pages[pageIdx]
		for blockIdx, elemID := range blockIDs {
			role := "P"
			if blockIdx < len(page.StructBlocks) && page.StructBlocks[blockIdx].Role != "" {
				role = page.StructBlocks[blockIdx].Role
			}
			body := fmt.Sprintf(
				"<< /Type /StructElem /S /%s /P %s /Pg %s /K [%d] >>",
				role,
				ref(rootID),
				ref(pageIDs[pageIdx]),
				blockIdx,
			)
			ow.writeObject(elemID, body)
		}
	}

	// ParentTree — a number tree whose /Nums array pairs each
	// page's StructParents index with the array of StructElems
	// owning that page's MCIDs (one per block in order). PDF
	// 14.7.4.4 case 1: when a page has multiple MCIDs each owned
	// by a different element, /Nums maps to an array of refs in
	// MCID order.
	var nums bytes.Buffer
	nums.WriteString("[")
	first := true
	for pageIdx, blockIDs := range pageBlockElemIDs {
		if !first {
			nums.WriteByte(' ')
		}
		first = false
		var arr bytes.Buffer
		arr.WriteString("[")
		for j, elemID := range blockIDs {
			if j > 0 {
				arr.WriteByte(' ')
			}
			arr.WriteString(ref(elemID))
		}
		arr.WriteString("]")
		fmt.Fprintf(&nums, "%d %s", pageIdx, arr.String())
	}
	nums.WriteString("]")
	ow.writeObject(parentTreeID, fmt.Sprintf("<< /Nums %s >>", nums.String()))

	// StructTreeRoot. /K is the array of top-level structure
	// elements — every block becomes a top-level child since we
	// don't yet emit nested groupings. PDF/UA conformance asks
	// for hierarchy in real documents (Sect > P, Sect > H1, …);
	// landed flat for v0.22 with the role tags right.
	var kids bytes.Buffer
	totalBlocks := 0
	kids.WriteString("[")
	for _, blockIDs := range pageBlockElemIDs {
		for _, elemID := range blockIDs {
			if totalBlocks > 0 {
				kids.WriteByte(' ')
			}
			kids.WriteString(ref(elemID))
			totalBlocks++
		}
	}
	kids.WriteString("]")
	rootBody := fmt.Sprintf(
		"<< /Type /StructTreeRoot /K %s /ParentTree %s /ParentTreeNextKey %d >>",
		kids.String(),
		ref(parentTreeID),
		len(pageBlockElemIDs),
	)
	ow.writeObject(rootID, rootBody)
	return rootID
}
