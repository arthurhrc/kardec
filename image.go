package kardec

import (
	"errors"
	"fmt"
	"os"
)

// ImageFormat identifies the supported on-disk encodings. JPEG carries
// straight through into the PDF; PNG is decoded and re-encoded as raw
// RGB + FlateDecode (transparency is flattened against white in v0.2).
type ImageFormat uint8

const (
	// ImageFormatUnknown is the zero value. Builders treat it as an error.
	ImageFormatUnknown ImageFormat = iota
	ImageFormatJPEG
	ImageFormatPNG
)

// String returns a human-readable label, mainly for error messages.
func (f ImageFormat) String() string {
	switch f {
	case ImageFormatJPEG:
		return "JPEG"
	case ImageFormatPNG:
		return "PNG"
	default:
		return "unknown"
	}
}

// Image is the block carrying a raster image. Width and Height are
// expressed in PDF points; if both are zero the layout engine substitutes
// the image's natural pixel dimensions converted to points at 72 DPI.
// If exactly one is set, the other is derived from the source aspect
// ratio so the image is never distorted.
type Image struct {
	data      []byte
	format    ImageFormat
	width     Length
	height    Length
	alignment Alignment
}

// blockKind implements Block.
func (Image) blockKind() blockKind { return kindImage }

// Data returns the raw bytes that the renderer will embed. Read-only:
// callers must not mutate the slice.
func (i Image) Data() []byte { return i.data }

// Format reports the on-disk encoding of Data.
func (i Image) Format() ImageFormat { return i.format }

// Width returns the requested target width in points (zero means
// "derive from height or natural size").
func (i Image) Width() Length { return i.width }

// Height returns the requested target height in points.
func (i Image) Height() Length { return i.height }

// Alignment returns the horizontal placement of the image inside the
// page's content area.
func (i Image) Alignment() Alignment { return i.alignment }

// detectImageFormat inspects the first few bytes of data and returns
// the matching ImageFormat. Empty or unrecognised payloads return
// ImageFormatUnknown plus an error.
func detectImageFormat(data []byte) (ImageFormat, error) {
	if len(data) < 8 {
		return ImageFormatUnknown, errors.New("kardec: image data too short to identify")
	}
	switch {
	case data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return ImageFormatJPEG, nil
	case data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G':
		return ImageFormatPNG, nil
	default:
		return ImageFormatUnknown, fmt.Errorf("kardec: unsupported image format (header %x)", data[:4])
	}
}

// ImageBuilder accumulates customisation for an image before the
// caller commits it to the document via Build. Functional helpers
// (Width / Height / Center / etc.) keep the builder fluent.
type ImageBuilder struct {
	doc     *Document
	img     Image
	label   string // optional cross-reference label set via Label(name)
	caption []Run  // optional caption runs set via Caption / CaptionRuns
	err     error
}

// Image starts an ImageBuilder from an in-memory image payload. The
// format is auto-detected from the leading bytes; callers needing
// explicit control can use AddImage with a fully-built Image value.
func (d *Document) Image(data []byte) *ImageBuilder {
	b := &ImageBuilder{doc: d, img: Image{data: data}}
	if d.err != nil {
		return b
	}
	format, err := detectImageFormat(data)
	if err != nil {
		b.err = err
		return b
	}
	b.img.format = format
	return b
}

// ImageFile is a convenience that reads path and forwards to Image.
// Errors from the read are captured in the builder and surface from
// Build through the document's deferred-error chain.
func (d *Document) ImageFile(path string) *ImageBuilder {
	b := &ImageBuilder{doc: d}
	if d.err != nil {
		return b
	}
	data, err := os.ReadFile(path)
	if err != nil {
		b.err = fmt.Errorf("kardec: read image %s: %w", path, err)
		return b
	}
	return d.Image(data)
}

// Width sets the image's target width. Pair with Height for explicit
// sizing or call alone to derive the height from the source aspect ratio.
func (b *ImageBuilder) Width(w Length) *ImageBuilder {
	b.img.width = w
	return b
}

// Height sets the image's target height. Pair with Width for explicit
// sizing or call alone to derive the width from the source aspect ratio.
func (b *ImageBuilder) Height(h Length) *ImageBuilder {
	b.img.height = h
	return b
}

// Center horizontally centers the image in the page's content area.
func (b *ImageBuilder) Center() *ImageBuilder {
	b.img.alignment = AlignCenter
	return b
}

// AlignRight right-aligns the image in the page's content area.
func (b *ImageBuilder) AlignRight() *ImageBuilder {
	b.img.alignment = AlignRight
	return b
}

// Label tags the image with a cross-reference label. Build will
// register the label, increment the figure counter, and emit an
// invisible anchor immediately before the image so doc.Ref(label)
// hyperlinks resolve to its position in the rendered PDF. An empty
// label is ignored.
func (b *ImageBuilder) Label(name string) *ImageBuilder {
	b.label = name
	return b
}

// Caption attaches a single-string caption that renders as a
// centered paragraph immediately below the image. When the image
// also carries a label, Build prefixes the caption with the
// canonical "Figure N: " marker so the on-page label matches what
// doc.Ref(label) resolves to.
//
// Callers needing rich runs (italics, bold) inside the caption use
// CaptionRuns instead.
func (b *ImageBuilder) Caption(text string) *ImageBuilder {
	b.caption = []Run{Text(text)}
	return b
}

// CaptionRuns is the rich-content variant of Caption: callers
// supply a fully-styled run sequence and Build keeps the runs
// intact, only prepending "Figure N: " when a label is also set.
func (b *ImageBuilder) CaptionRuns(runs ...Run) *ImageBuilder {
	b.caption = runs
	return b
}

// Build appends the image to the parent document and returns the
// document so the caller can resume the top-level chain. Builder-side
// errors (read failure, unrecognised format) are folded into the
// document's deferred error.
//
// When the image carries a caption (or a label), Build wraps the
// anchor + image + caption sequence inside a KeepTogether group so
// the figure and its caption never split across pages. Plain
// captionless, label-less images still emit as a bare Image block
// so existing layouts stay untouched.
func (b *ImageBuilder) Build() *Document {
	if b.doc.err != nil {
		return b.doc
	}
	if b.err != nil {
		return b.doc.fail(b.err)
	}
	if len(b.img.data) == 0 {
		return b.doc.fail(errors.New("kardec: image with no data"))
	}
	figureNumber := 0
	if b.label != "" {
		figureNumber = b.doc.registerFigureLabel(b.label)
	}
	if len(b.caption) == 0 {
		// Plain image (with optional anchor) — preserve the v0.2
		// emission order so existing tests + layouts are stable.
		if b.label != "" {
			b.doc.append(Anchor{name: RefAnchorName(b.label)})
		}
		return b.doc.append(b.img)
	}

	// Captioned image: bind anchor + image + caption together so a
	// page break never separates the figure from its label.
	captionRuns := b.caption
	if figureNumber > 0 {
		marker := "Figure " + itoaSmall(figureNumber) + ": "
		captionRuns = append([]Run{Text(marker)}, captionRuns...)
	}
	captionPara := Paragraph{
		runs:      captionRuns,
		styleName: StyleCaption,
		alignment: AlignCenter,
	}
	parts := make([]Block, 0, 3)
	if b.label != "" {
		parts = append(parts, Anchor{name: RefAnchorName(b.label)})
	}
	parts = append(parts, b.img, captionPara)
	return b.doc.append(NewKeepTogether(parts...))
}
