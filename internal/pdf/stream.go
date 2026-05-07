package pdf

import (
	"bytes"
	"compress/zlib"
)

// compressThreshold is the size above which the writer FlateDecode-encodes
// content streams. Below it the payload stays uncompressed for easier
// debugging — small streams compress poorly in absolute terms anyway.
const compressThreshold = 1024

// maybeFlate compresses raw with zlib (FlateDecode is zlib per PDF 7.4.4)
// when the payload exceeds compressThreshold. It returns the bytes to
// write into the stream and a boolean indicating whether /Filter
// /FlateDecode must be added to the stream dictionary.
//
// On any zlib error the original bytes are returned uncompressed; PDFs
// without compression are still valid, just larger.
func maybeFlate(raw []byte) (data []byte, compressed bool) {
	if len(raw) < compressThreshold {
		return raw, false
	}
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return raw, false
	}
	if err := zw.Close(); err != nil {
		return raw, false
	}
	return buf.Bytes(), true
}

// flateAlways unconditionally zlib-compresses raw. Used for FontFile2
// streams where compression is essentially free in CPU terms versus the
// 100+ KB savings on a typical TTF.
func flateAlways(raw []byte) []byte {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return raw
	}
	if err := zw.Close(); err != nil {
		return raw
	}
	return buf.Bytes()
}
