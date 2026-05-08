package kardec

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// makePNG returns a minimal w×h PNG payload with a solid blue fill,
// suitable for exercising the image pipeline without bundling a fixture.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	blue := color.RGBA{R: 30, G: 80, B: 170, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, blue)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

// makeJPEG returns a minimal w×h JPEG payload (red fill).
func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	red := color.RGBA{R: 200, G: 30, B: 30, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, red)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatalf("jpeg.Encode: %v", err)
	}
	return buf.Bytes()
}

func TestImageDetectsPNGAndJPEG(t *testing.T) {
	if got, err := detectImageFormat(makePNG(t, 10, 10)); err != nil || got != ImageFormatPNG {
		t.Errorf("detect PNG = (%v, %v), want (PNG, nil)", got, err)
	}
	if got, err := detectImageFormat(makeJPEG(t, 10, 10)); err != nil || got != ImageFormatJPEG {
		t.Errorf("detect JPEG = (%v, %v), want (JPEG, nil)", got, err)
	}
}

func TestImageDetectsUnknown(t *testing.T) {
	if _, err := detectImageFormat([]byte("not an image at all")); err == nil {
		t.Error("detectImageFormat should reject random bytes")
	}
}

func TestDocumentImageBuilderProducesBlock(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.Image(makePNG(t, 50, 50)).Width(Pt(200)).Center().Build()

	if err := doc.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}
	blocks := doc.Sections()[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("want 1 block, got %d", len(blocks))
	}
	img, ok := blocks[0].(Image)
	if !ok {
		t.Fatalf("first block should be Image, got %T", blocks[0])
	}
	if img.Format() != ImageFormatPNG {
		t.Errorf("Format = %v, want PNG", img.Format())
	}
	if img.Width() != Pt(200) {
		t.Errorf("Width = %v, want %v", img.Width(), Pt(200))
	}
	if img.Alignment() != AlignCenter {
		t.Errorf("Alignment = %v, want AlignCenter", img.Alignment())
	}
}

func TestDocumentImageBuilderEmptyDataFails(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.Image(nil).Build()
	if doc.Err() == nil {
		t.Error("nil image data should record a deferred error")
	}
}

func TestDocumentImageBuilderUnknownFormatFails(t *testing.T) {
	doc := New(PageA4, MarginsNormal)
	doc.Image([]byte("xxxxxxxxxxxx")).Build()
	if doc.Err() == nil {
		t.Error("unknown image format should record a deferred error")
	}
}
