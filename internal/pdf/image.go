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
	Width  int  // natural width: pixels for raster, BBox width for Form
	Height int  // natural height: pixels for raster, BBox height for Form
	IsForm bool // true for SVG-derived Form XObjects (different cm math)
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

	// SVG images take a separate path: they are Form XObjects
	// rather than Image XObjects, so the dictionary keys / required
	// resources are different. Emission stays in this file so a
	// page's resource lookup still funnels through imageHandle.
	if img.Encoding == ImageSVGForm {
		return emitFormXObject(ow, idx, img)
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
		IsForm: false,
	}, nil
}

// emitFormXObject writes an SVG-derived vector graphic as a PDF
// Form XObject (PDF 8.10). The page references it through the same
// /Im0 Do operator as a raster image, but the underlying object's
// content stream is graphics operators (m, l, c, re, f, S, …)
// rather than image samples.
//
// /BBox is set to (0, 0, WidthPx, HeightPx) — same as the natural
// canvas size the SVG converter resolved. /Matrix is omitted
// (identity) so the page's "cm" operator (the same one used for
// raster images) places and scales the form.
//
// /Resources is empty: the v0.19.0 converter only uses inline
// graphics-state operators (rg, RG, w, …) plus path operators that
// don't need indirect resources. v0.19.x will populate
// /Resources /Font when the converter starts handling <text>.
func emitFormXObject(ow *objectWriter, idx int, img EmbeddedImage) (*imageHandle, error) {
	body := img.Data
	id := ow.allocID()
	dict := fmt.Sprintf(
		"/Type /XObject /Subtype /Form /FormType 1 "+
			"/BBox [0 0 %d %d] /Resources << /ProcSet [/PDF] >> "+
			"/Length %d",
		img.WidthPx, img.HeightPx, len(body),
	)
	ow.writeStreamObject(id, dict, body)
	return &imageHandle{
		Name:   fmt.Sprintf("Im%d", idx),
		ID:     id,
		Width:  img.WidthPx,
		Height: img.HeightPx,
		IsForm: true,
	}, nil
}
