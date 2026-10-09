//nolint:err113 // Pointer validation reports malformed caller-provided images.
package rdp

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"image"
	"sync"
)

const (
	pointerCacheEntries = 16
	pointerMaxSize      = 32
	pointerSystem       = 1
	pointerPosition     = 3
	pointerColor        = 6
	pointerCached       = 7
	pointerNew          = 8
)

type pointerCapabilities struct {
	colors, alpha int
}

type pointerCacheEntry struct {
	key   [sha256.Size]byte
	valid bool
}

type pointerState struct {
	mu     sync.Mutex
	caps   pointerCapabilities
	cache  [pointerCacheEntries]pointerCacheEntry
	next   int
	active int // Zero: unknown; one: hidden; two: default; three and up: cache index.
}

func (p *pointerState) reset() {
	p.cache = [pointerCacheEntries]pointerCacheEntry{}
	p.next, p.active = 0, 0
}

// ResetPointerCache invalidates locally tracked cursor state. Resend the shape
// and position after a display refresh or resize; the next shape repopulates
// its client cache entry. Capabilities survive the reset.
func (s *Session) ResetPointerCache() {
	s.pointer.mu.Lock()
	s.pointer.reset()
	s.pointer.mu.Unlock()
}

// WritePointer updates the client-rendered cursor, or hides it for a nil image.
// Pixels must use image.RGBA's premultiplied alpha format. The hotspot is relative
// to the image's upper-left corner. Clear padding is trimmed, preserving the
// hotspot, and oversized shapes are fitted to the negotiated 32-pixel maximum.
// Clients without New Pointer support receive a 24-bit color/AND-mask cursor.
// Call this between frames; updates are suppressed during bitmap reactivation.
func (s *Session) WritePointer(shape *image.RGBA, hotX, hotY int) error {
	if shape != nil && !validPointerImage(shape, hotX, hotY) {
		return errors.New("rdp: invalid pointer image or hotspot")
	}
	s.display.mu.Lock()
	defer s.display.mu.Unlock()
	if s.display.reactivation != nil {
		return nil
	}
	p := &s.pointer
	p.mu.Lock()
	defer p.mu.Unlock()
	if shape == nil {
		return s.writeSystemPointer(0, 1)
	}
	shape, hotX, hotY = fitPointer(shape, hotX, hotY)
	if shape == nil {
		return s.writeSystemPointer(0, 1)
	}
	slots, alpha := min(p.caps.colors, pointerCacheEntries), false
	if p.caps.alpha > 0 {
		slots, alpha = min(p.caps.alpha, pointerCacheEntries), true
	}
	if slots == 0 {
		// Malformed/minimal clients can omit usable cache slots. They still
		// receive a visible system cursor rather than an invalid cache index.
		return s.writeSystemPointer(0x7f00, 2)
	}
	payload := encodePointer(shape, hotX, hotY, alpha)
	key := sha256.Sum256(payload)
	for index := range slots {
		if !p.cache[index].valid || p.cache[index].key != key {
			continue
		}
		if p.active == index+3 {
			return nil
		}
		cached := []byte{byte(index), 0}
		if err := s.writePointerUpdate(pointerCached, cached); err != nil {
			return err
		}
		p.active = index + 3

		return nil
	}
	index := p.next % slots
	kind, offset := pointerColor, 0
	if alpha {
		kind, offset = pointerNew, 2
	}
	put16(payload, offset, index)
	if err := s.writePointerUpdate(kind, payload); err != nil {
		return err
	}
	p.cache[index] = pointerCacheEntry{key: key, valid: true}
	p.next, p.active = (index+1)%slots, index+3

	return nil
}

// WritePointerDefault restores the client's standard arrow cursor.
func (s *Session) WritePointerDefault() error {
	s.display.mu.Lock()
	defer s.display.mu.Unlock()
	if s.display.reactivation != nil {
		return nil
	}
	s.pointer.mu.Lock()
	defer s.pointer.mu.Unlock()

	return s.writeSystemPointer(0x7f00, 2)
}

func (s *Session) writeSystemPointer(system uint32, active int) error {
	if s.pointer.active == active {
		return nil
	}
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, system)
	if err := s.writePointerUpdate(pointerSystem, payload); err != nil {
		return err
	}
	s.pointer.active = active

	return nil
}

// WritePointerPosition moves the client cursor to a desktop pixel, clamped to
// the active desktop bounds. Only use it for guest-initiated pointer movement;
// reflecting every client mouse event can pull its locally drawn cursor back.
func (s *Session) WritePointerPosition(x, y int) error {
	s.display.mu.Lock()
	defer s.display.mu.Unlock()
	if s.display.reactivation != nil {
		return nil
	}
	p := &s.pointer
	p.mu.Lock()
	defer p.mu.Unlock()
	width, height := s.Size()
	if width <= 0 || height <= 0 {
		return errors.New("rdp: pointer position requires an active desktop")
	}
	point := image.Pt(max(0, min(x, width-1)), max(0, min(y, height-1)))
	payload := make([]byte, 4)
	put16(payload, 0, point.X)
	put16(payload, 2, point.Y)
	if err := s.writePointerUpdate(pointerPosition, payload); err != nil {
		return err
	}

	return nil
}

func (s *Session) writePointerUpdate(kind int, payload []byte) error {
	data := make([]byte, 4+len(payload))
	put16(data, 0, kind)
	copy(data[4:], payload)

	return s.WriteDataPDU(27, data)
}

func validPointerImage(shape *image.RGBA, hotX, hotY int) bool {
	w, h := shape.Rect.Dx(), shape.Rect.Dy()

	return w > 0 && h > 0 && w <= 512 && h <= 512 && hotX >= 0 && hotY >= 0 &&
		hotX < w && hotY < h && shape.Stride >= w*4 && len(shape.Pix) >= w*4 &&
		(h == 1 || shape.Stride <= (len(shape.Pix)-w*4)/(h-1))
}

func fitPointer(shape *image.RGBA, hotX, hotY int) (*image.RGBA, int, int) {
	box := image.Rect(hotX, hotY, hotX+1, hotY+1)
	visible := false
	for y := range shape.Rect.Dy() {
		for x := range shape.Rect.Dx() {
			if shape.Pix[y*shape.Stride+x*4+3] != 0 {
				visible = true
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if !visible {
		return nil, 0, 0
	}
	w, h := box.Dx(), box.Dy()
	largest := max(w, h)
	if largest > pointerMaxSize {
		w = max(1, w*pointerMaxSize/largest)
		h = max(1, h*pointerMaxSize/largest)
	}
	result := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			source := (box.Min.Y+y*box.Dy()/h)*shape.Stride + (box.Min.X+x*box.Dx()/w)*4
			destination := y*result.Stride + x*4
			copy(result.Pix[destination:destination+4], shape.Pix[source:source+4])
		}
	}

	return result, (hotX - box.Min.X) * w / box.Dx(), (hotY - box.Min.Y) * h / box.Dy()
}

// MS-RDPBCGR 2.2.9.1.1.4.4/.5: both masks are bottom-up, with each scanline
// padded to two bytes. New Pointer uses BGRA32; Color Pointer uses BGR24 and
// the AND mask for transparency. Alpha cursors retain premultiplied channels.
func encodePointer(shape *image.RGBA, hotX, hotY int, alpha bool) []byte {
	w, h := shape.Rect.Dx(), shape.Rect.Dy()
	bytesPerPixel, header := 3, 14
	if alpha {
		bytesPerPixel, header = 4, 16
	}
	xorStride, andStride := (w*bytesPerPixel+1)&^1, ((w+7)/8+1)&^1
	xorSize, andSize := xorStride*h, andStride*h
	data := make([]byte, header+xorSize+andSize+1)
	attributes := data
	if alpha {
		put16(data, 0, 32)
		attributes = data[2:]
	}
	put16(attributes, 2, hotX)
	put16(attributes, 4, hotY)
	put16(attributes, 6, w)
	put16(attributes, 8, h)
	put16(attributes, 10, andSize)
	put16(attributes, 12, xorSize)
	for y := range h {
		for x := range w {
			source := y*shape.Stride + x*4
			pixel := shape.Pix[source : source+4]
			destination := header + (h-y-1)*xorStride + x*bytesPerPixel
			switch {
			case alpha:
				data[destination], data[destination+1] = pixel[2], pixel[1]
				data[destination+2], data[destination+3] = pixel[0], pixel[3]
			case pixel[3] >= 128:
				for channel := range 3 {
					data[destination+channel] = byte(min(255, int(pixel[2-channel])*255/int(pixel[3])))
				}
			default:
				data[header+xorSize+(h-y-1)*andStride+x/8] |= 0x80 >> (x % 8)
			}
		}
	}

	return data
}
