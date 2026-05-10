package pdf

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// objectWriter is a write-only PDF body builder. It hands out indirect
// object IDs in monotonic order and tracks the byte offset of each object
// so the cross-reference table (xref) can be emitted at the end.
//
// All PDF spec section references in this file cite ISO 32000-1:2008 (the
// PDF 1.7 standard).
type objectWriter struct {
	buf     bytes.Buffer
	offsets map[int]int64 // 1-based object ID -> byte offset
	nextID  int
	// fileKey is the AES-128 file encryption key derived from the
	// caller-supplied passwords + permissions + /ID. Non-nil opts
	// the writer into Standard Security Handler V=4/R=4: every
	// stream's payload is wrapped in AES-CBC with a per-object key
	// before being written. Strings stay plaintext (StrF=Identity)
	// — full string encryption is the v0.16.x follow-up.
	fileKey []byte
}

func newObjectWriter() *objectWriter {
	return &objectWriter{
		offsets: make(map[int]int64),
		nextID:  1,
	}
}

// allocID reserves an object ID without writing a body for it. Useful when
// an object's body needs to reference an ID that will be written later
// (e.g. /Pages forward-references its kids).
func (w *objectWriter) allocID() int {
	id := w.nextID
	w.nextID++
	return id
}

// writeObject emits "id 0 obj\n<body>\nendobj\n" and records the offset.
// Body is the literal serialized object content (dictionary, array, stream,
// etc.) without the surrounding obj/endobj markers.
func (w *objectWriter) writeObject(id int, body string) {
	w.offsets[id] = int64(w.buf.Len())
	fmt.Fprintf(&w.buf, "%d 0 obj\n%s\nendobj\n", id, body)
}

// writeStreamObject emits a stream object: "id 0 obj\n<<dict>>\nstream\n
// <data>\nendstream\nendobj\n". The dict is written as-is between the
// double angle brackets. Callers must include /Length in the dict to match
// len(data) — the writer does not patch it after the fact.
//
// When the writer carries a file key (Standard Security Handler is on),
// the stream payload is wrapped in AES-128-CBC with a per-object key
// derived from the file key + this object's number/generation. The dict
// must be patched ahead of time to declare the post-encryption /Length;
// the helper handles that automatically when fileKey is set.
func (w *objectWriter) writeStreamObject(id int, dict string, data []byte) {
	w.offsets[id] = int64(w.buf.Len())
	if w.fileKey != nil {
		ciphertext := aesEncryptObject(w.fileKey, id, 0, data)
		// Override /Length in the supplied dict — encrypted bytes
		// include a 16-byte IV plus PKCS#7 padding, so the
		// post-encryption length differs from the plaintext.
		dict = patchLength(dict, len(ciphertext))
		data = ciphertext
	}
	fmt.Fprintf(&w.buf, "%d 0 obj\n<<%s>>\nstream\n", id, dict)
	w.buf.Write(data)
	w.buf.WriteString("\nendstream\nendobj\n")
}

// patchLength replaces "/Length N" in a stream dict body with the
// supplied n. The helper is used when encryption changes the
// post-encryption byte length and the caller-built dict needs to
// reflect the new size.
func patchLength(dict string, n int) string {
	prefix := "/Length "
	idx := strings.Index(dict, prefix)
	if idx < 0 {
		return dict
	}
	end := idx + len(prefix)
	for end < len(dict) && dict[end] >= '0' && dict[end] <= '9' {
		end++
	}
	return dict[:idx+len(prefix)] + fmt.Sprintf("%d", n) + dict[end:]
}

// allocAndWrite is a convenience for the common "I need an ID and I have
// the body now" case.
func (w *objectWriter) allocAndWrite(body string) int {
	id := w.allocID()
	w.writeObject(id, body)
	return id
}

// ref formats an indirect reference: "id 0 R" (PDF 7.3.10).
func ref(id int) string {
	return fmt.Sprintf("%d 0 R", id)
}

// escapeLiteralString quotes s for use as a PDF literal string per
// PDF 7.3.4.2 — wrapping in parentheses and escaping (, ), and \.
//
// Non-ASCII bytes are passed through unchanged; under WinAnsiEncoding the
// font handles the byte->glyph mapping. Callers using Identity-H (CID)
// fonts must format their text as hex strings instead.
func escapeLiteralString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('(')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte(')')
	return b.String()
}

// writeTo flushes the body to w, returning the total bytes written so the
// caller knows where the body ended (the offset of "xref" in the final
// PDF). Header bytes are not part of the objectWriter — the top-level
// Writer prepends them.
func (w *objectWriter) writeTo(out io.Writer) (int64, error) {
	n, err := out.Write(w.buf.Bytes())
	return int64(n), err
}

// bodyLen returns the running size of the object body in bytes.
func (w *objectWriter) bodyLen() int64 {
	return int64(w.buf.Len())
}
