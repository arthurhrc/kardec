package kardec

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// makeTinyPNGFile writes a minimal PNG to a temp directory and
// returns the (dir, file-name). Used by tests that exercise the
// markdown-image-embed path.
func makeTinyPNGFile(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: 0x40, G: 0x80, B: 0xC0, A: 0xFF})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	fname := "logo.png"
	if err := os.WriteFile(filepath.Join(dir, fname), buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
	return dir, fname
}

func TestAppendMarkdownImageDropsByDefault(t *testing.T) {
	dir, fname := makeTinyPNGFile(t)
	doc := New(PageA4, MarginsNormal).
		AppendMarkdown("![logo](" + fname + ")")
	_ = dir // base dir intentionally not configured
	if !hasWarningContaining(doc, "SetMarkdownBaseDir not configured") {
		t.Errorf("expected drop-with-warning when base dir is unset, got %v", doc.Warnings())
	}
	for _, b := range doc.Sections()[0].Blocks {
		if _, ok := b.(Image); ok {
			t.Errorf("no Image block should be appended without a configured base dir")
		}
	}
}

func TestAppendMarkdownImageEmbedsWhenBaseDirConfigured(t *testing.T) {
	dir, fname := makeTinyPNGFile(t)
	doc := New(PageA4, MarginsNormal).
		SetMarkdownBaseDir(dir).
		AppendMarkdown("![logo](" + fname + ")")

	if err := doc.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}
	var seen bool
	for _, b := range doc.Sections()[0].Blocks {
		if _, ok := b.(Image); ok {
			seen = true
		}
	}
	if !seen {
		t.Errorf("expected Image block from markdown image, got blocks %v", doc.Sections()[0].Blocks)
	}
}

func TestAppendMarkdownImageRemoteWarnsAndSkips(t *testing.T) {
	dir, _ := makeTinyPNGFile(t)
	doc := New(PageA4, MarginsNormal).
		SetMarkdownBaseDir(dir).
		AppendMarkdown("![logo](https://example.com/logo.png)")
	if !hasWarningContaining(doc, "remote image") {
		t.Errorf("expected remote-image warning, got %v", doc.Warnings())
	}
}

func TestAppendMarkdownImageMissingFileWarnsButSucceeds(t *testing.T) {
	dir, _ := makeTinyPNGFile(t)
	doc := New(PageA4, MarginsNormal).
		SetMarkdownBaseDir(dir).
		AppendMarkdown("![missing](nonexistent.png)")
	// ImageFile will fail; failure flows through deferred-error
	// chain. Document.Err returns the read error.
	if doc.Err() == nil {
		t.Errorf("missing file should leave a deferred error, got nil")
	}
}
