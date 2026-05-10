package sign

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
)

// findLastStartXref returns the byte offset of the last xref
// table in pdfBytes — the value of the last `startxref\nNNNN\n`
// line. Incremental signature update points its /Prev at this so
// the reader's chain reaches the original catalog and pages.
func findLastStartXref(pdfBytes []byte) (int, error) {
	// Scan backward for `startxref`. The PDF is well-formed
	// kardec output so there's exactly one — but a defensive
	// search-from-end still picks the *latest* one if some other
	// tool already appended an update.
	idx := bytes.LastIndex(pdfBytes, []byte("startxref"))
	if idx < 0 {
		return 0, fmt.Errorf("startxref not found")
	}
	// The offset value follows on the next line.
	rest := pdfBytes[idx+len("startxref"):]
	// Skip whitespace.
	i := 0
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\r' || rest[i] == '\n') {
		i++
	}
	j := i
	for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
		j++
	}
	if j == i {
		return 0, fmt.Errorf("no digits after startxref")
	}
	v, err := strconv.Atoi(string(rest[i:j]))
	if err != nil {
		return 0, err
	}
	return v, nil
}

// findNextObjID scans pdfBytes for the highest indirect-object ID
// in use and returns highest+1. Used by the signature appendage
// to allocate fresh IDs that don't collide with existing objects.
func findNextObjID(pdfBytes []byte) (int, error) {
	re := regexp.MustCompile(`(?m)^(\d+) \d+ obj`)
	matches := re.FindAllSubmatch(pdfBytes, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("no objects found")
	}
	maxID := 0
	for _, m := range matches {
		id, err := strconv.Atoi(string(m[1]))
		if err != nil {
			continue
		}
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1, nil
}

// findCatalogID locates the /Root entry in the trailer and
// returns its indirect-object ID. The trailer dict has the form
// `<< ... /Root N 0 R ... >>` (PDF 7.5.5); we grep for /Root
// followed by an object reference.
func findCatalogID(pdfBytes []byte) (int, error) {
	re := regexp.MustCompile(`/Root\s+(\d+)\s+\d+\s+R`)
	m := re.FindSubmatch(pdfBytes)
	if len(m) < 2 {
		return 0, fmt.Errorf("/Root not found in trailer")
	}
	return strconv.Atoi(string(m[1]))
}

// extractCatalogBody reads the body of the catalog indirect
// object (everything between `<<` and `>>` immediately after
// `<catalogID> 0 obj`). Used by the incremental signature update
// so the replacement catalog preserves the original entries
// (/Pages, /Outlines, /Metadata, /StructTreeRoot, …).
func extractCatalogBody(pdfBytes []byte, catalogID int) (string, error) {
	header := []byte(fmt.Sprintf("%d 0 obj", catalogID))
	idx := bytes.Index(pdfBytes, header)
	if idx < 0 {
		return "", fmt.Errorf("catalog object %d not found", catalogID)
	}
	rest := pdfBytes[idx+len(header):]
	open := bytes.Index(rest, []byte("<<"))
	if open < 0 {
		return "", fmt.Errorf("catalog body opening << not found")
	}
	// Find the matching closing `>>`. Catalogs in kardec output
	// don't nest dicts at the top level (the inner << values
	// have their own >> closers), so we need a balanced scan.
	depth := 0
	i := open
	for i < len(rest)-1 {
		if rest[i] == '<' && rest[i+1] == '<' {
			depth++
			i += 2
			continue
		}
		if rest[i] == '>' && rest[i+1] == '>' {
			depth--
			i += 2
			if depth == 0 {
				return string(rest[open:i]), nil
			}
			continue
		}
		i++
	}
	return "", fmt.Errorf("catalog body closing >> not found")
}

// injectAcroFormIntoCatalog returns body with an /AcroForm entry
// pointing at acroFormID appended just before the final `>>`.
// If body already has /AcroForm, it is REPLACED (Kardec doesn't
// emit /AcroForm itself, so this case only triggers when the
// caller re-signs a previously signed PDF).
func injectAcroFormIntoCatalog(body string, acroFormID int) string {
	// Drop any existing /AcroForm. Match `/AcroForm <id> <gen> R`.
	body = regexp.MustCompile(`/AcroForm\s+\d+\s+\d+\s+R`).ReplaceAllString(body, "")
	// Insert the new /AcroForm just before the closing `>>` of
	// the outermost catalog dict.
	insert := fmt.Sprintf(" /AcroForm %d 0 R", acroFormID)
	last := bytes.LastIndex([]byte(body), []byte(">>"))
	if last < 0 {
		return body
	}
	return body[:last] + insert + " " + body[last:]
}
