package rdp

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"strconv"
	"testing"
)

func TestBitmapEncoderPixelLayout(t *testing.T) {
	t.Parallel()
	for _, depth := range []int{16, 24, 32} {
		t.Run(strconv.Itoa(depth), func(t *testing.T) {
			t.Parallel()
			// A subimage exercises nonzero origin and a stride wider than its rows.
			source := image.NewRGBA(image.Rect(0, 0, 12, 8)).SubImage(image.Rect(3, 2, 6, 4)).(*image.RGBA)
			colors := []color.RGBA{
				{255, 0, 0, 255},
				{0, 255, 0, 255},
				{0, 0, 255, 255},
				{255, 255, 255, 255},
				{0, 0, 0, 255},
				{248, 128, 64, 255},
			}
			for i, value := range colors {
				source.SetRGBA(3+i%3, 2+i/3, value)
			}
			encoder, err := NewBitmapEncoder(67, 65, depth)
			if err != nil {
				t.Fatal(err)
			}
			decoded := image.NewRGBA(image.Rect(0, 0, 67, 65))
			updates := 0
			err = encoder.WriteFrame(source, false, func(packet []byte) error {
				updates++
				decodeTestBitmap(t, decoded, packet, depth)

				return nil
			})
			if err != nil || updates != 4 {
				t.Fatalf("updates=%d err=%v", updates, err)
			}
			for y := range 65 {
				for x := range 67 {
					want := colors[(y*2/65)*3+x*3/67]
					if depth == 16 {
						want.R &= 0xf8
						want.G &= 0xfc
						want.B &= 0xf8
					}
					if got := decoded.RGBAAt(x, y); got != want {
						t.Fatalf("pixel (%d,%d): got %v want %v", x, y, got, want)
					}
				}
			}
		})
	}
}

func decodeTestBitmap(t *testing.T, dst *image.RGBA, packet []byte, depth int) {
	t.Helper()
	if len(packet) < 22 || len(packet) >= 32767 || binary.LittleEndian.Uint16(packet) != 1 ||
		binary.LittleEndian.Uint16(packet[2:]) != 1 {
		t.Fatalf("invalid bitmap update header/length: %d", len(packet))
	}
	x, y := int(binary.LittleEndian.Uint16(packet[4:])), int(binary.LittleEndian.Uint16(packet[6:]))
	w, h := int(binary.LittleEndian.Uint16(packet[12:])), int(binary.LittleEndian.Uint16(packet[14:]))
	if int(binary.LittleEndian.Uint16(packet[8:])) != x+w-1 || int(binary.LittleEndian.Uint16(packet[10:])) != y+h-1 ||
		int(binary.LittleEndian.Uint16(packet[16:])) != depth || binary.LittleEndian.Uint16(packet[18:]) != 0 ||
		int(binary.LittleEndian.Uint16(packet[20:])) != len(packet)-22 {
		t.Fatal("inconsistent bitmap rectangle")
	}
	stride := ((w*depth/8 + 3) / 4) * 4
	if len(packet) != 22+stride*h {
		t.Fatal("incorrect row padding")
	}
	for row := range h {
		for col := range w {
			pixel := packet[22+row*stride+col*depth/8:]
			c := color.RGBA{A: 255}
			if depth == 16 {
				value := binary.LittleEndian.Uint16(pixel)
				c.R, c.G, c.B = uint8(value>>11)<<3, uint8(value>>5&63)<<2, uint8(value&31)<<3
			} else {
				c.R, c.G, c.B = pixel[2], pixel[1], pixel[0]
			}
			dst.SetRGBA(x+col, y+h-1-row, c)
		}
	}
}

func TestBitmapEncoderDirtyTilesAndFailedWrite(t *testing.T) {
	t.Parallel()
	encoder, err := NewBitmapEncoder(128, 128, 24)
	if err != nil {
		t.Fatal(err)
	}
	frame := image.NewRGBA(image.Rect(0, 0, 128, 128))
	count := 0
	send := func([]byte) error {
		count++

		return nil
	}
	if err := encoder.WriteFrame(frame, false, send); err != nil || count != 4 {
		t.Fatalf("initial frame: updates=%d err=%v", count, err)
	}
	count = 0
	if err := encoder.WriteFrame(frame, false, send); err != nil || count != 0 {
		t.Fatalf("unchanged frame: updates=%d err=%v", count, err)
	}
	frame.SetRGBA(100, 100, color.RGBA{255, 0, 0, 255})
	failure := io.ErrClosedPipe
	if err := encoder.WriteFrame(frame, false, func([]byte) error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("write error: %v", err)
	}
	if err := encoder.WriteFrame(frame, false, send); err != nil || count != 1 {
		t.Fatalf("retry changed tile: updates=%d err=%v", count, err)
	}
	count = 0
	if err := encoder.WriteFrame(frame, true, send); err != nil || count != 4 {
		t.Fatalf("forced refresh: updates=%d err=%v", count, err)
	}
}
