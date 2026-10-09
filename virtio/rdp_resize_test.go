package virtio

import (
	"image"
	"reflect"
	"testing"
	"time"

	"github.com/bobuhiro11/gokvm/internal/rdp"
)

func TestRDPResizeControllerReconnect(t *testing.T) {
	t.Parallel()
	d := &RDPDisplay{}
	first := &rdp.Session{Width: 1280, Height: 720}
	second := &rdp.Session{Width: 800, Height: 600}
	third := &rdp.Session{Width: 1920, Height: 1080}
	var modes []rdp.DesktopSize
	d.addResizeViewer(first)
	// The GPU can attach after a client has completed negotiation.
	d.SetResizeHandler(func(width, height int) error {
		modes = append(modes, rdp.DesktopSize{Width: width, Height: height})

		return nil
	})
	d.resizeGuest(second)
	d.removeResizeViewer(second)
	d.removeResizeViewer(first)
	d.resizeGuest(first) // A departed viewer cannot reclaim control.
	d.addResizeViewer(second)
	d.removeResizeViewer(first)
	if d.viewer != second {
		t.Fatal("departed viewer removed the new resize controller")
	}
	d.removeResizeViewer(second)
	d.removeResizeViewer(second)
	d.addResizeViewer(third)
	d.resizeGuest(second)
	want := []rdp.DesktopSize{{Width: 1280, Height: 720}, {Width: 800, Height: 600}, {Width: 1920, Height: 1080}}
	if !reflect.DeepEqual(modes, want) {
		t.Fatalf("GPU modes: got %v, want %v", modes, want)
	}
}

func TestRDPResizeResetsEncoderHistory(t *testing.T) {
	t.Parallel()
	bitmap, err := rdp.NewBitmapEncoder(1024, 768, 32)
	if err != nil {
		t.Fatal(err)
	}
	writer := &rdpFrameWriter{
		session: &rdp.Session{Width: 1280, Height: 720, BitsPerPixel: 32},
		bitmap:  bitmap, width: 1024, height: 768, nextFrame: time.Now().Add(time.Hour),
	}
	if err := writer.updateSize(); err != nil {
		t.Fatal(err)
	}
	if writer.bitmap == bitmap || !writer.force || !writer.nextFrame.IsZero() ||
		writer.width != 1280 || writer.height != 720 {
		t.Fatalf("resize did not discard old encoder history: %+v", writer)
	}
	bitmap = writer.bitmap
	writer.force = false
	if err := writer.updateSize(); err != nil {
		t.Fatal(err)
	}
	if writer.bitmap != bitmap || writer.force {
		t.Fatal("unchanged geometry unnecessarily reset the encoder")
	}
}

type resizePointerInput struct {
	x, y    uint16
	buttons uint8
}

func (p *resizePointerInput) KeyEvent(bool, uint32) {}

func (p *resizePointerInput) PointerEvent(buttons uint8, x, y uint16) {
	p.buttons, p.x, p.y = buttons, x, y
}

func TestRDPResizePointerTracksFramebuffer(t *testing.T) {
	t.Parallel()
	d := &RDPDisplay{framebuffer: newFramebuffer()}
	defer d.shutdown()
	input := &resizePointerInput{}
	d.SetInput(input)
	session := &rdp.Session{Width: 800, Height: 600}
	for _, size := range []image.Point{{X: 1024, Y: 768}, {X: 1280, Y: 720}, {X: 800, Y: 600}} {
		if err := d.Flush(size.X, size.Y, image.NewRGBA(image.Rect(0, 0, size.X, size.Y))); err != nil {
			t.Fatal(err)
		}
		d.dispatchInput(session, []rdp.InputEvent{{Kind: rdp.InputPointer, X: 400, Y: 300, Buttons: 1}})
		if int(input.x) != size.X/2 || int(input.y) != size.Y/2 || input.buttons != 1 {
			t.Fatalf("pointer at %v: got %+v", size, input)
		}
		d.dispatchInput(session, []rdp.InputEvent{{Kind: rdp.InputPointer, X: 65535, Y: 65535}})
		if int(input.x) >= size.X || int(input.y) >= size.Y {
			t.Fatalf("pointer escaped framebuffer %v: %+v", size, input)
		}
	}
}

func TestRDPMultiDisplayForwardsResizeHandler(t *testing.T) {
	t.Parallel()
	rdpDisplay := &RDPDisplay{}
	display := NewMultiDisplay(NewPNGDisplay("unused.png"), rdpDisplay)
	var width, height int
	display.SetResizeHandler(func(w, h int) error {
		width, height = w, h

		return nil
	})
	rdpDisplay.addResizeViewer(&rdp.Session{Width: 1280, Height: 720})
	if width != 1280 || height != 720 {
		t.Fatalf("combined displays did not attach resize handler: %dx%d", width, height)
	}
}
