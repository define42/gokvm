package virtio

import (
	"image"
	"image/png"
	"os"
	"sync"
)

// Display is the sink virtio-gpu presents flushed frames to. Implementations
// must be safe for the single GPU IO goroutine to call; a future VNC backend
// can slot in behind the same interface.
type Display interface {
	// Flush is called on RESOURCE_FLUSH with the scanout's current frame.
	// The image is owned by the caller only for the duration of the call.
	Flush(width, height int, img *image.RGBA) error

	// Close releases any resources held by the display.
	Close() error
}

// PNGDisplay writes each flushed frame to a PNG file, replacing it in place.
// It is dependency-free (stdlib image/png) and serves as the default backend.
type PNGDisplay struct {
	path string
	mu   sync.Mutex
}

// NewPNGDisplay returns a Display that writes frames to path.
func NewPNGDisplay(path string) *PNGDisplay {
	return &PNGDisplay{path: path}
}

// Flush encodes img to a temporary file and atomically renames it over the
// target, so a reader never observes a half-written PNG.
func (d *PNGDisplay) Flush(width, height int, img *image.RGBA) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tmp := d.path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if err := png.Encode(f, img); err != nil {
		f.Close()

		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(tmp, d.path)
}

func (d *PNGDisplay) Close() error { return nil }
