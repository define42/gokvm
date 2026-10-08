package virtio

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"sync"
	"testing"
	"time"
)

var _ ConsoleDisplay = (*VNCDisplay)(nil)

func TestFramebufferTabletUsesCurrentDimensions(t *testing.T) {
	t.Parallel()
	d := newFramebuffer()
	t.Cleanup(d.shutdown)
	tablet := NewInputTablet(6, nil, nil)
	t.Cleanup(func() { _ = tablet.Close() })
	d.SetInput(NewInputPair(nil, tablet))
	for _, tt := range []struct {
		size int
		want int32
	}{{4, 12287}, {8, 6143}} {
		if err := d.Flush(tt.size, tt.size, image.NewRGBA(image.Rect(0, 0, tt.size, tt.size))); err != nil {
			t.Fatal(err)
		}
		d.sendPointerEvent(0, 1, 1)
		for _, want := range []inputEvent{
			{typ: evAbs, code: absX, value: tt.want},
			{typ: evAbs, code: absY, value: tt.want},
			synEvent(),
		} {
			if got, ok := tablet.popPending(); !ok || got != want {
				t.Fatalf("size %d: tablet event = %+v, %v; want %+v", tt.size, got, ok, want)
			}
		}
	}
}

func TestFramebufferCopiesSubimage(t *testing.T) {
	t.Parallel()

	d := newFramebuffer()
	t.Cleanup(d.shutdown)
	img := image.NewRGBA(image.Rect(4, 7, 8, 10))
	img.SetRGBA(5, 8, color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff})
	img.SetRGBA(6, 9, color.RGBA{R: 0x78, G: 0x9a, B: 0xbc, A: 0xff})
	sub := img.SubImage(image.Rect(5, 8, 7, 10)).(*image.RGBA)
	if err := d.Flush(2, 2, sub); err != nil {
		t.Fatal(err)
	}

	clear(img.Pix)
	frame := d.snapshot()
	want := []byte{
		0x12, 0x34, 0x56, 0xff, 0, 0, 0, 0,
		0, 0, 0, 0, 0x78, 0x9a, 0xbc, 0xff,
	}
	if frame.width != 2 || frame.height != 2 || !bytes.Equal(frame.pix, want) {
		t.Fatalf("snapshot: got %+v, want a packed copy of the subimage", frame)
	}

	clear(frame.pix)
	if !bytes.Equal(d.snapshot().pix, want) {
		t.Fatal("snapshot pixels alias the stored framebuffer")
	}
}

func TestFramebufferRejectsInvalidImages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		height int
		img    *image.RGBA
	}{
		{name: "nil", width: 1, height: 1},
		{name: "bounds", width: 2, height: 1, img: image.NewRGBA(image.Rect(0, 0, 1, 1))},
		{name: "overflow", width: int(^uint(0) >> 1), height: 2, img: image.NewRGBA(image.Rect(0, 0, 1, 1))},
		{name: "short pixels", width: 1, height: 2, img: &image.RGBA{
			Pix: make([]byte, 4), Stride: 4, Rect: image.Rect(0, 0, 1, 2),
		}},
		{name: "short stride", width: 1, height: 1, img: &image.RGBA{
			Pix: make([]byte, 4), Stride: 0, Rect: image.Rect(0, 0, 1, 1),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := newFramebuffer()
			t.Cleanup(d.shutdown)
			if err := d.Flush(tt.width, tt.height, tt.img); !errors.Is(err, errInvalidFramebuffer) {
				t.Fatalf("Flush: got %v, want invalid-framebuffer error", err)
			}
		})
	}
}

func TestFramebufferGPUFrameSupersedesFallback(t *testing.T) {
	t.Parallel()

	d := newFramebuffer()
	t.Cleanup(d.shutdown)
	entered := make(chan struct{})
	proceed := make(chan struct{})
	fallbackDone := make(chan struct{})
	go func() {
		defer close(fallbackDone)
		var last []byte
		rendered := false
		d.refreshFallback([]byte{1}, &last, &rendered, framebufferBlank, func([]byte) *image.RGBA {
			close(entered)
			<-proceed

			return image.NewRGBA(image.Rect(0, 0, 1, 1))
		})
	}()
	<-entered

	gpuDone := make(chan error, 1)
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 0xff, A: 0xff})
	go func() { gpuDone <- d.Flush(1, 1, img) }()
	close(proceed)
	<-fallbackDone
	if err := <-gpuDone; err != nil {
		t.Fatal(err)
	}

	if _, err := d.Write([]byte("later serial output")); err != nil {
		t.Fatal(err)
	}
	var last []byte
	rendered := false
	if d.refreshFallback([]byte{2}, &last, &rendered, framebufferBlank, func([]byte) *image.RGBA {
		t.Error("fallback rendered after the GPU took over")

		return img
	}) {
		t.Fatal("fallback remains enabled after the GPU took over")
	}

	if frame := d.snapshot(); !bytes.Equal(frame.pix, img.Pix) {
		t.Fatalf("GPU framebuffer was overwritten: %v", frame.pix)
	}
}

func TestFramebufferShutdownWakesWaiters(t *testing.T) {
	t.Parallel()

	d := newFramebuffer()
	finished := make(chan bool, 1)
	go func() {
		_, ok := d.frameForRequestUntil(true, 0, nil)
		finished <- ok
	}()

	d.shutdown()
	d.shutdown()
	select {
	case ok := <-finished:
		if ok {
			t.Fatal("frame waiter reported an update after shutdown")
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown left a frame waiter blocked")
	}
}

func TestFramebufferConcurrentFallbackStartAndShutdown(t *testing.T) {
	t.Parallel()

	for range 100 {
		d := newFramebuffer()
		mem := []byte{1, 2, 3, 0}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			d.StartLinearFramebufferFallback(mem, 0, 1, 1, 4)
		}()
		go func() {
			defer wg.Done()
			d.shutdown()
		}()
		wg.Wait()

		d.StartLinearFramebufferFallback(mem, 0, 1, 1, 4)
		before := d.snapshot().seq
		if _, err := d.Write([]byte("after shutdown")); err != nil {
			t.Fatal(err)
		}
		if err := d.Flush(1, 1, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
			t.Fatal(err)
		}
		if got := d.snapshot().seq; got != before {
			t.Fatalf("frame changed after shutdown: sequence %d -> %d", before, got)
		}
	}
}
