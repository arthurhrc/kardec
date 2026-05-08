package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
)

// imageHandle is the writer-internal record produced when an
// EmbeddedImage is written as an XObject. It mirrors fontHandle: a
// stable resource name plus the indirect-object ID page resources
// reference.
type imageHandle struct {
	Name   string
	ID     int
	Width  int // pixel width, used by callers that need the natural size
	Height int
}

// emitImage writes one EmbeddedImage as a PDF XObject and returns the
// handle the page resources will refer to. The resource name is
// derived from the image's index in Document.Images so it is stable
// regardless of which pages reference it.
//
// JPEGs pass through with /Filter /DCTDecode (PDF 7.4.8) — no decode,
// no recompression. Raw-RGB payloads are zlib-compressed and emitted
// with /Filter /FlateDecode.
func emitImage(ow *objectWriter, idx int, img EmbeddedImage) (*imageHandle, error) {
	if img.WidthPx <= 0 || img.HeightPx <= 0 {
		return nil, fmt.Errorf("pdf: image %d has non-positive dimensions (%dx%d)",
			idx, img.WidthPx, img.HeightPx)
	}
	if len(img.Data) == 0 {
		return nil, fmt.Errorf("pdf: image %d has empty payload", idx)
	}

	var (
		filter string
		body   []byte
	)
	switch img.Encoding {
	case ImageJPEG:
		filter = "/DCTDecode"
		body = img.Data
	case ImageRawRGB:
		expected := img.WidthPx * img.HeightPx * 3
		if len(img.Data) != expected {
			return nil, fmt.Errorf("pdf: raw RGB image %d size mismatch — got %d bytes, want %d (%dx%d × 3)",
				idx, len(img.Data), expected, img.WidthPx, img.HeightPx)
		}
		filter = "/FlateDecode"
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		if _, err := zw.Write(img.Data); err != nil {
			return nil, fmt.Errorf("pdf: compress image %d: %w", idx, err)
		}
		if err := zw.Close(); err != nil {
			return nil, fmt.Errorf("pdf: close zlib for image %d: %w", idx, err)
		}
		body = buf.Bytes()
	default:
		return nil, fmt.Errorf("pdf: image %d has unknown encoding %d", idx, img.Encoding)
	}

	id := ow.allocID()
	dict := fmt.Sprintf(
		"/Type /XObject /Subtype /Image /Width %d /Height %d "+
			"/ColorSpace /DeviceRGB /BitsPerComponent 8 "+
			"/Filter %s /Length %d",
		img.WidthPx, img.HeightPx, filter, len(body),
	)
	ow.writeStreamObject(id, dict, body)
	return &imageHandle{
		Name:   fmt.Sprintf("Im%d", idx),
		ID:     id,
		Width:  img.WidthPx,
		Height: img.HeightPx,
	}, nil
}
