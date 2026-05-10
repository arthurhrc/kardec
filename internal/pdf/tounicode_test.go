package pdf

import (
	"bytes"
	"testing"
)

func TestBuildToUnicodeCMapShape(t *testing.T) {
	got := buildToUnicodeCMap()
	checks := []string{
		"/CIDInit /ProcSet findresource begin",
		"/CMapName /Adobe-Identity-UCS def",
		"/CMapType 2 def",
		"1 begincodespacerange",
		"<00> <FF>",
		"endcodespacerange",
		"beginbfchar",
		"endbfchar",
		"endcmap",
		// 'A' should round-trip to U+0041.
		"<41> <0041>",
		// Smart quote at WinAnsi 0x91 → U+2018.
		"<91> <2018>",
		// Euro at WinAnsi 0x80 → U+20AC.
		"<80> <20AC>",
	}
	for _, c := range checks {
		if !bytes.Contains(got, []byte(c)) {
			t.Errorf("CMap missing %q", c)
		}
	}
}

func TestBuildToUnicodeCMapChunksAt100(t *testing.T) {
	// PDF spec caps each beginbfchar block at 100 entries. The CMap
	// table has well over 100 assigned WinAnsi slots so the body
	// should split into at least two blocks.
	got := buildToUnicodeCMap()
	count := bytes.Count(got, []byte("beginbfchar"))
	if count < 2 {
		t.Errorf("expected >= 2 beginbfchar blocks for 100-entry chunking, got %d", count)
	}
}
