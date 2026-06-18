package virtio_test

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/bobuhiro11/gokvm/virtio"
)

// TestPNGDisplayFlush checks the default backend writes a decodable PNG whose
// pixels round-trip.
func TestPNGDisplayFlush(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "fb.png")

	d := virtio.NewPNGDisplay(path)
	defer d.Close()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Pix[0], img.Pix[1], img.Pix[2], img.Pix[3] = 0x10, 0x20, 0x30, 0xff

	if err := d.Flush(2, 2, img); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer f.Close()

	got, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}

	if b := got.Bounds(); b.Dx() != 2 || b.Dy() != 2 {
		t.Fatalf("decoded dims: got %dx%d want 2x2", b.Dx(), b.Dy())
	}

	// RGBA() returns 16-bit channels; compare the high byte.
	r, g, b, a := got.At(0, 0).RGBA()
	if uint8(r>>8) != 0x10 || uint8(g>>8) != 0x20 || uint8(b>>8) != 0x30 || uint8(a>>8) != 0xff {
		t.Fatalf("pixel(0,0): got (%d,%d,%d,%d) want (16,32,48,255)", r>>8, g>>8, b>>8, a>>8)
	}
}
