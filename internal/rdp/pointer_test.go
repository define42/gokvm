package rdp

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"sync"
	"testing"
)

func TestPointerNegotiation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		caps []byte
		want pointerCapabilities
	}{
		{"color", []byte{0, 0, 7, 0}, pointerCapabilities{colors: 7}},
		{"alpha", []byte{1, 0, 9, 0, 3, 0}, pointerCapabilities{colors: 9, alpha: 3}},
		{"zero-alpha", []byte{1, 0, 5, 0, 0, 0}, pointerCapabilities{colors: 5}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			s := new(Session)
			if err := s.confirmActive(pointerConfirmFixture(test.caps)); err != nil {
				t.Fatal(err)
			}
			if s.pointer.caps != test.want {
				t.Fatalf("pointer capabilities = %+v, want %+v", s.pointer.caps, test.want)
			}
		})
	}
	for _, caps := range [][]byte{{}, {1, 0, 2}, {1, 0, 2, 0, 7}} {
		if err := new(Session).confirmActive(pointerConfirmFixture(caps)); err == nil {
			t.Fatalf("accepted malformed pointer capabilities: %x", caps)
		}
	}
	s := &Session{Width: 800, Height: 600}
	demand := s.demandActive()
	offset := 14 + int(binary.LittleEndian.Uint16(demand[10:])) + 4
	found := false
	for offset < len(demand)-4 {
		length := int(binary.LittleEndian.Uint16(demand[offset+2:]))
		if binary.LittleEndian.Uint16(demand[offset:]) == 8 {
			found = true
			if !bytes.Equal(demand[offset+4:offset+length], []byte{1, 0, 16, 0, 16, 0}) {
				t.Fatalf("incorrect server pointer capability: %x", demand[offset:offset+length])
			}
		}
		offset += length
	}
	if !found {
		t.Fatal("server omitted pointer capability")
	}
}

func TestPointerAlphaAndLegacyWirePixels(t *testing.T) {
	t.Parallel()
	shape := image.NewRGBA(image.Rect(5, 7, 8, 9))
	copy(shape.Pix, []byte{
		10, 20, 30, 255, 40, 50, 60, 255, 0, 0, 0, 0,
		64, 32, 16, 128, 0, 0, 0, 0, 4, 5, 6, 255,
	})
	for _, test := range []struct {
		name string
		caps pointerCapabilities
		want []byte
	}{
		{"alpha", pointerCapabilities{alpha: 2}, []byte{
			8, 0, 0, 0, 32, 0, 0, 0, 2, 0, 1, 0, 3, 0, 2, 0, 4, 0, 24, 0,
			16, 32, 64, 128, 0, 0, 0, 0, 6, 5, 4, 255,
			30, 20, 10, 255, 60, 50, 40, 255, 0, 0, 0, 0,
			0, 0, 0, 0, 0,
		}},
		{"legacy", pointerCapabilities{colors: 2}, []byte{
			6, 0, 0, 0, 0, 0, 2, 0, 1, 0, 3, 0, 2, 0, 4, 0, 20, 0,
			31, 63, 127, 0, 0, 0, 6, 5, 4, 0,
			30, 20, 10, 60, 50, 40, 0, 0, 0, 0,
			0x40, 0, 0x20, 0, 0,
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			conn := &graphicsTestConn{}
			s := &Session{conn: conn}
			s.pointer.caps = test.caps
			if err := s.WritePointer(shape, 2, 1); err != nil {
				t.Fatal(err)
			}
			output := pointerOutputFixture(t, conn.outgoing.Bytes())
			if len(output) != 1 || !bytes.Equal(output[0], test.want) {
				t.Fatalf("pointer wire = %x, want %x", output, test.want)
			}
		})
	}
}

func TestPointerCacheHideDefaultAndReset(t *testing.T) {
	t.Parallel()
	conn := &graphicsTestConn{}
	s := &Session{conn: conn}
	s.pointer.caps.alpha = 2
	first, second := pointerShapeFixture(12), pointerShapeFixture(24)
	for _, shape := range []*image.RGBA{first, first, second, first, nil, nil, first} {
		if err := s.WritePointer(shape, 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := s.WritePointerDefault(); err != nil {
			t.Fatal(err)
		}
	}
	s.ResetPointerCache()
	if err := s.WritePointer(first, 0, 0); err != nil {
		t.Fatal(err)
	}
	output := pointerOutputFixture(t, conn.outgoing.Bytes())
	if len(output) != 7 {
		t.Fatalf("emitted %d pointer updates, want 7", len(output))
	}
	for i, kind := range []byte{8, 8, 7, 1, 7, 1, 8} {
		if output[i][0] != kind {
			t.Fatalf("update %d = %x, want kind %d", i, output[i], kind)
		}
	}
	if binary.LittleEndian.Uint16(output[1][6:]) != 1 ||
		!bytes.Equal(output[2], []byte{7, 0, 0, 0, 0, 0}) ||
		!bytes.Equal(output[3], []byte{1, 0, 0, 0, 0, 0, 0, 0}) ||
		!bytes.Equal(output[5], []byte{1, 0, 0, 0, 0, 0x7f, 0, 0}) {
		t.Fatalf("incorrect cache/system fields: %x", output)
	}
}

func TestPointerFitsLegacySizeAndPreservesHotspot(t *testing.T) {
	t.Parallel()
	shape := image.NewRGBA(image.Rect(4, 8, 68, 72))
	shape.SetRGBA(14, 28, color.RGBA{R: 255, A: 255})
	shape.SetRGBA(33, 51, color.RGBA{G: 255, A: 255})
	cropped, x, y := fitPointer(shape, 13, 21)
	if cropped.Rect != image.Rect(0, 0, 20, 24) || x != 3 || y != 1 ||
		cropped.RGBAAt(0, 0).R != 255 || cropped.RGBAAt(19, 23).G != 255 {
		t.Fatalf("cropped padded cursor: %v, hotspot(%d,%d)", cropped.Rect, x, y)
	}
	shape.SetRGBA(67, 71, color.RGBA{B: 255, A: 255})
	fitted, x, y := fitPointer(shape, 0, 0)
	if fitted.Rect != image.Rect(0, 0, 32, 32) || x != 0 || y != 0 {
		t.Fatalf("oversized fitted cursor: %v, hotspot(%d,%d)", fitted.Rect, x, y)
	}
	if invisible, _, _ := fitPointer(image.NewRGBA(image.Rect(0, 0, 64, 64)), 0, 0); invisible != nil {
		t.Fatal("clear cursor was not hidden")
	}
}

func TestPointerCacheHonorsClientSlotLimit(t *testing.T) {
	t.Parallel()
	conn := &graphicsTestConn{}
	s := &Session{conn: conn}
	s.pointer.caps.colors = 1
	for _, red := range []byte{10, 20, 10} {
		if err := s.WritePointer(pointerShapeFixture(red), 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	output := pointerOutputFixture(t, conn.outgoing.Bytes())
	if len(output) != 3 {
		t.Fatalf("cache eviction generated %d messages, want3", len(output))
	}
	for _, message := range output {
		if message[0] != pointerColor || binary.LittleEndian.Uint16(message[4:]) != 0 {
			t.Fatalf("one-slot client received an invalid cache reference: %x", message)
		}
	}
}

func TestPointerPositionAndReactivation(t *testing.T) {
	t.Parallel()
	conn := &graphicsTestConn{}
	s := &Session{conn: conn, Width: 800, Height: 600}
	s.pointer.caps.alpha = 1
	for range 2 {
		if err := s.WritePointerPosition(-4, 9999); err != nil {
			t.Fatal(err)
		}
	}
	s.display.reactivation = &activation{}
	if err := s.WritePointerPosition(100, 200); err != nil {
		t.Fatal(err)
	}
	if err := s.WritePointer(pointerShapeFixture(20), 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.WritePointerDefault(); err != nil {
		t.Fatal(err)
	}
	output := pointerOutputFixture(t, conn.outgoing.Bytes())
	if len(output) != 2 || !bytes.Equal(output[0], []byte{3, 0, 0, 0, 0, 0, 0x57, 2}) ||
		!bytes.Equal(output[0], output[1]) {
		t.Fatalf("position clamp/repeated warp/reactivation handling = %x", output)
	}
	s.display.reactivation = nil
	s.pointer.caps = pointerCapabilities{}
	if err := s.WritePointer(pointerShapeFixture(20), 0, 0); err != nil {
		t.Fatal(err)
	}
	output = pointerOutputFixture(t, conn.outgoing.Bytes())
	if len(output) != 3 || !bytes.Equal(output[2], []byte{1, 0, 0, 0, 0, 0x7f, 0, 0}) {
		t.Fatal("client without cache entries did not receive a visible default cursor")
	}
}

func TestPointerRejectsMalformedImages(t *testing.T) {
	t.Parallel()
	s := &Session{conn: &graphicsTestConn{}}
	for _, shape := range []*image.RGBA{
		{},
		{Rect: image.Rect(0, 0, 2, 2), Stride: 8, Pix: make([]byte, 8)},
		{Rect: image.Rect(0, 0, 2, 2), Stride: int(^uint(0) >> 1), Pix: make([]byte, 16)},
		{Rect: image.Rect(0, 0, 513, 513)},
	} {
		if err := s.WritePointer(shape, 0, 0); err == nil {
			t.Fatalf("accepted malformed pointer: %+v", shape)
		}
	}
	if err := s.WritePointer(pointerShapeFixture(1), 2, 0); err == nil {
		t.Fatal("accepted out-of-bounds hotspot")
	}
}

func TestPointerConcurrentStateAndReactivation(t *testing.T) {
	t.Parallel()
	s := &Session{conn: &graphicsTestConn{}, Width: 800, Height: 600, BitsPerPixel: 24}
	shape := pointerShapeFixture(20)
	var workers sync.WaitGroup
	for worker := range 4 {
		workers.Go(func() {
			for index := range 50 {
				var err error
				switch worker {
				case 0:
					err = s.WritePointer(shape, 0, 0)
				case 1:
					err = s.WritePointerPosition(index, index)
				case 2:
					s.ResetPointerCache()
				default:
					s.display.mu.Lock()
					err = s.confirmActiveDepth(pointerConfirmFixture([]byte{1, 0, 1, 0, 1, 0}), false)
					s.display.mu.Unlock()
				}
				if err != nil {
					t.Error(err)
				}
			}
		})
	}
	workers.Wait()
}

func pointerConfirmFixture(caps []byte) []byte {
	p := make([]byte, 20+28+4+len(caps))
	put16(p, 0, len(p))
	put16(p, 2, 0x13)
	binary.LittleEndian.PutUint32(p[6:], shareID)
	put16(p, 14, len(p)-16)
	put16(p, 16, 2)
	put16(p, 20, 2)
	put16(p, 22, 28)
	put16(p, 24, 24)
	put16(p, 48, 8)
	put16(p, 50, len(caps)+4)
	copy(p[52:], caps)

	return p
}

func pointerShapeFixture(red byte) *image.RGBA {
	shape := image.NewRGBA(image.Rect(0, 0, 2, 2))
	shape.SetRGBA(0, 0, color.RGBA{R: red, A: 255})

	return shape
}

func pointerOutputFixture(t *testing.T, wire []byte) [][]byte {
	t.Helper()
	reader := bytes.NewReader(wire)
	var output [][]byte
	for reader.Len() > 0 {
		mcs, err := readMCS(reader)
		if err != nil || len(mcs) < 7 || !bytes.Equal(mcs[:6], []byte{0x68, 0, 1, 3, 0xeb, 0x70}) {
			t.Fatalf("invalid pointer transport %x: %v", mcs, err)
		}
		data, err := takePER(mcs[6:])
		if err != nil || len(data) < 22 || data[14] != 27 {
			t.Fatalf("invalid pointer share header %x: %v", data, err)
		}
		output = append(output, data[18:])
	}

	return output
}
