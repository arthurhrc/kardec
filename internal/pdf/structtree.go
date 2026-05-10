package pdf

import (
	"bytes"
	"fmt"
)

// PDF/UA tagging — strict per-block hierarchy.
//
// This file emits the structure-tag object graph that PDF/UA-1
// validators expect when the writer is in tagged mode (Document.Tagged
// == true):
//
//   /Catalog        — adds /MarkInfo /Marked true, /Lang, /StructTreeRoot
//   /Page           — adds /StructParents N (per-page index into ParentTree)
//   StructTreeRoot  — root of the structure hierarchy
//   ParentTree      — number tree mapping per-page MCIDs to their owning elems
//   StructElem (×N) — recursive: leaves carry MCID, inner nodes carry /K [child refs]
//
// Each page's content stream is wrapped in N marked-content
// sequences (one per leaf block) — see wrapMarkedContentByBlocks
// in content.go. Inner blocks (Table, TR, Sect, …) own no MCIDs;
// they appear in the structure tree purely as container parents.

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
// StructElem objects. pageBlockElemIDs is per-page slice of slices:
// for page i, pageBlockElemIDs[i] holds the IDs of every StructElem
// (leaf or inner) the page tree contains, in pre-order traversal.
//
// The writer pre-allocates these IDs in Write() so each page dict can
// forward-reference them through /StructParents N. The actual objects
// are emitted here.
//
// The structure tree honours nested Children, so /Table > /TR > /TD
// hierarchies and cross-page /Sect groupings appear as real
// container elements in the PDF — what strict PDF/UA-1 validators
// (veraPDF, PAC) expect.
func emitStructTree(ow *objectWriter, pageIDs []int, pages []Page, pageBlockElemIDs [][]int) int {
	rootID := ow.allocID()
	parentTreeID := ow.allocID()

	// Walk every page's block tree, assigning IDs in pre-order so
	// pageBlockElemIDs[pageIdx][k] is the k-th element in pre-order
	// traversal of pages[pageIdx].StructBlocks. Track each leaf's
	// MCID (just its position among leaves on the page) so the
	// per-leaf StructElem references the matching marked-content
	// sequence in the content stream.
	for pageIdx, blockIDs := range pageBlockElemIDs {
		page := pages[pageIdx]
		ctx := &emitCtx{
			pageRef:    ref(pageIDs[pageIdx]),
			rootRef:    ref(rootID),
			elemIDs:    blockIDs,
			elemCursor: 0,
			leafMCID:   0,
			ow:         ow,
		}
		// Determine the parent of every top-level page block —
		// each is parented to the StructTreeRoot.
		topLevel := page.StructBlocks
		if len(topLevel) == 0 {
			// Lite-mode placeholder: synthesize a single P element
			// covering the whole page so the legacy v0.17 path
			// stays alive.
			if len(blockIDs) > 0 {
				ow.writeObject(blockIDs[0], fmt.Sprintf(
					"<< /Type /StructElem /S /P /P %s /Pg %s /K [0] >>",
					ctx.rootRef, ctx.pageRef,
				))
			}
			continue
		}
		for i := range topLevel {
			emitStructElemTree(ctx, topLevel[i], ctx.rootRef)
		}
	}

	// ParentTree — a number tree whose /Nums array pairs each
	// page's StructParents index with the array of StructElems
	// owning that page's MCIDs. PDF 14.7.4.4 case 1: when a page
	// has multiple MCIDs each owned by a different element,
	// /Nums maps to an array of refs in MCID order.
	var nums bytes.Buffer
	nums.WriteString("[")
	first := true
	for pageIdx, blockIDs := range pageBlockElemIDs {
		if !first {
			nums.WriteByte(' ')
		}
		first = false
		// Array entry: one ref per MCID on the page, in MCID
		// order. Walk leaves in pre-order to build the list.
		leafIDs := collectLeafElemIDs(pages[pageIdx].StructBlocks, blockIDs)
		if len(leafIDs) == 0 && len(blockIDs) > 0 {
			// Lite-mode: single synthesized P element.
			leafIDs = []int{blockIDs[0]}
		}
		var arr bytes.Buffer
		arr.WriteString("[")
		for j, elemID := range leafIDs {
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

	// StructTreeRoot. /K is the array of every TOP-LEVEL element
	// (the immediate children of the root): typically per-page
	// top-level blocks. ParentTreeNextKey is len(pages).
	var kids bytes.Buffer
	kids.WriteString("[")
	totalTop := 0
	for pageIdx, blockIDs := range pageBlockElemIDs {
		// Emit refs to top-level elements only — the first
		// `len(StructBlocks)` IDs in pre-order, since
		// emitStructElemTree assigns IDs in DFS / pre-order and
		// top-level blocks are at depth 0. For lite-mode,
		// blockIDs[0] is the synthetic P.
		topCount := len(pages[pageIdx].StructBlocks)
		if topCount == 0 {
			topCount = 1
		}
		for k := 0; k < topCount && k < len(blockIDs); k++ {
			if totalTop > 0 {
				kids.WriteByte(' ')
			}
			kids.WriteString(ref(blockIDs[k]))
			totalTop++
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

// emitCtx carries the per-page state emitStructElemTree needs to
// emit one StructElem per block in pre-order.
type emitCtx struct {
	pageRef    string
	rootRef    string
	elemIDs    []int
	elemCursor int // next index into elemIDs to consume
	leafMCID   int // next MCID to assign to a leaf
	ow         *objectWriter
}

// emitStructElemTree recursively emits one StructElem per block in
// the tree rooted at b, walking children in pre-order. parentRef is
// the /P value the emitted element references — root for top-level
// blocks, the parent's ref for nested ones.
//
// Leaves (no children) carry /K [mcid] — a single integer wrapping
// the marked-content sequence they own. Inner nodes carry /K
// [childRef1 childRef2 …] — forward references to children's
// StructElem objects.
func emitStructElemTree(ctx *emitCtx, b StructBlock, parentRef string) int {
	// Pre-order: assign this block its ID first, recurse for
	// children after. Children's IDs are immediately after this
	// one in pre-order.
	if ctx.elemCursor >= len(ctx.elemIDs) {
		// Defensive: shouldn't happen if pre-allocation walked
		// the same tree.
		return 0
	}
	myID := ctx.elemIDs[ctx.elemCursor]
	ctx.elemCursor++

	role := b.Role
	if role == "" {
		role = "P"
	}

	if b.IsLeaf() {
		// Leaf: own one MCID. Use it and advance.
		mcid := ctx.leafMCID
		ctx.leafMCID++
		ctx.ow.writeObject(myID, fmt.Sprintf(
			"<< /Type /StructElem /S /%s /P %s /Pg %s /K [%d] >>",
			role, parentRef, ctx.pageRef, mcid,
		))
		return myID
	}

	// Inner node: emit children first so we know their IDs, then
	// emit ourselves with /K = [childRef1 childRef2 …]. Pre-order
	// emission consumes elemCursor in DFS-walk order.
	myRef := ref(myID)
	childRefs := make([]int, 0, len(b.Children))
	for i := range b.Children {
		childID := emitStructElemTree(ctx, b.Children[i], myRef)
		childRefs = append(childRefs, childID)
	}
	var kBuf bytes.Buffer
	kBuf.WriteString("[")
	for i, cid := range childRefs {
		if i > 0 {
			kBuf.WriteByte(' ')
		}
		kBuf.WriteString(ref(cid))
	}
	kBuf.WriteString("]")
	ctx.ow.writeObject(myID, fmt.Sprintf(
		"<< /Type /StructElem /S /%s /P %s /Pg %s /K %s >>",
		role, parentRef, ctx.pageRef, kBuf.String(),
	))
	return myID
}

// collectLeafElemIDs walks the per-page block tree and returns the
// IDs of leaf blocks in pre-order — same order leaves consume MCIDs
// during content-stream emission. Used to populate ParentTree's
// /Nums entry for the page.
func collectLeafElemIDs(blocks []StructBlock, allIDs []int) []int {
	var leaves []int
	cursor := 0
	var walk func(b StructBlock)
	walk = func(b StructBlock) {
		if cursor >= len(allIDs) {
			return
		}
		myID := allIDs[cursor]
		cursor++
		if b.IsLeaf() {
			leaves = append(leaves, myID)
			return
		}
		for i := range b.Children {
			walk(b.Children[i])
		}
	}
	for i := range blocks {
		walk(blocks[i])
	}
	return leaves
}

// countBlocksInTree returns the total StructElem count for the
// block tree (this block + all descendants). Used by the writer
// to pre-allocate enough indirect object IDs.
func countBlocksInTree(b StructBlock) int {
	n := 1
	for i := range b.Children {
		n += countBlocksInTree(b.Children[i])
	}
	return n
}
