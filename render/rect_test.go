package render

import (
	"bytes"
	"compress/zlib"
	"io"
	"testing"

	"github.com/arthurhrc/kardec"
)

// findContentStreams scans data for `<<...>>\nstream\n...\nendstream`
// blocks, decompresses them when /Filter /FlateDecode is declared, and
// concatenates the raw operator bytes for inspection. Used by tests
// that need to assert which PDF operators were emitted.
func findContentStreams(data []byte) []byte {
	var out bytes.Buffer
	cursor := 0
	for cursor < len(data) {
		idx := bytes.Index(data[cursor:], []byte("stream\n"))
		if idx == -1 {
			break
		}
		streamStart := cursor + idx + len("stream\n")
		dictStart := bytes.LastIndex(data[:cursor+idx], []byte("<<"))
		if dictStart == -1 {
			cursor = streamStart
			continue
		}
		dict := data[dictStart : cursor+idx]
		end := bytes.Index(data[streamStart:], []byte("\nendstream"))
		if end == -1 {
			break
		}
		body := data[streamStart : streamStart+end]
		if bytes.Contains(dict, []byte("/Filter /FlateDecode")) {
			zr, err := zlib.NewReader(bytes.NewReader(body))
			if err == nil {
				decompressed, _ := io.ReadAll(zr)
				out.Write(decompressed)
				zr.Close()
			}
		} else {
			out.Write(body)
		}
		cursor = streamStart + end + len("\nendstream")
	}
	return out.Bytes()
}

func TestMathFractionEmitsRectangleForBar(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).Math(`\frac{a}{b}`)
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := findContentStreams(out)
	if !bytes.Contains(streams, []byte(" re\n")) {
		t.Errorf("expected an `re` operator (filled rectangle) in the content stream for a fraction, got %q", abbrev(streams))
	}
	if !bytes.Contains(streams, []byte("\nf\n")) {
		t.Errorf("expected an `f` operator (fill) in the content stream for a fraction")
	}
}

func TestMathSqrtEmitsRectangleForOverline(t *testing.T) {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).Math(`\sqrt{x}`)
	out, err := Bytes(doc)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	streams := findContentStreams(out)
	if !bytes.Contains(streams, []byte(" re\n")) {
		t.Errorf("expected an `re` operator for the sqrt overline")
	}
}

func abbrev(b []byte) string {
	const max = 200
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
