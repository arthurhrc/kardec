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

// emitStructTree writes the StructTreeRoot, ParentTree, and per-page
// StructElem objects. pageIDs are the indirect IDs of the already-
// emitted /Page dicts; pageElemIDs were pre-allocated by Write before
// pages were emitted so each page dict could carry a forward-reference
// to its parent struct element via /StructParents.
//
// Returns the StructTreeRoot ID. The catalog references it through
// /StructTreeRoot N.
func emitStructTree(ow *objectWriter, pageIDs, pageElemIDs []int) int {
	rootID := ow.allocID()
	parentTreeID := ow.allocID()

	// Per-page StructElem objects. Each says "I am a /P, my parent
	// is the StructTreeRoot, my graphical content lives on page
	// /Pg, and my one kid is the marked-content sequence with
	// MCID 0 inside that page."
	for i, elemID := range pageElemIDs {
		body := fmt.Sprintf(
			"<< /Type /StructElem /S /P /P %s /Pg %s /K [0] >>",
			ref(rootID),
			ref(pageIDs[i]),
		)
		ow.writeObject(elemID, body)
	}

	// ParentTree — a number tree whose /Nums array pairs each
	// page's StructParents index with the StructElem that owns
	// the page's MCIDs. Single MCID per page in lite mode means
	// the value is a direct StructElem reference rather than an
	// array of references (PDF 14.7.4.4 case 2).
	var nums bytes.Buffer
	nums.WriteString("[")
	for i, elemID := range pageElemIDs {
		if i > 0 {
			nums.WriteByte(' ')
		}
		fmt.Fprintf(&nums, "%d %s", i, ref(elemID))
	}
	nums.WriteString("]")
	ow.writeObject(parentTreeID, fmt.Sprintf("<< /Nums %s >>", nums.String()))

	// StructTreeRoot. /K is the array of top-level structure
	// elements; here, one per page. /ParentTree resolves
	// /StructParents lookups during reading. /ParentTreeNextKey
	// is one past the highest used index — the same number as
	// len(pageElemIDs) because we used 0..N-1.
	var kids bytes.Buffer
	kids.WriteString("[")
	for i, elemID := range pageElemIDs {
		if i > 0 {
			kids.WriteByte(' ')
		}
		kids.WriteString(ref(elemID))
	}
	kids.WriteString("]")
	rootBody := fmt.Sprintf(
		"<< /Type /StructTreeRoot /K %s /ParentTree %s /ParentTreeNextKey %d >>",
		kids.String(),
		ref(parentTreeID),
		len(pageElemIDs),
	)
	ow.writeObject(rootID, rootBody)
	return rootID
}
