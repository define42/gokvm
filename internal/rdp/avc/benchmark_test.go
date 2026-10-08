package avc

import (
	"image"
	"testing"
)

func benchmarkFrames(width, height int) []*image.RGBA {
	frames := make([]*image.RGBA, 8)
	for frame := range frames {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := range height {
			for x := range width {
				offset := y*img.Stride + x*4
				img.Pix[offset] = byte((x + frame*4) / 4)
				img.Pix[offset+1] = byte((y + frame*4) / 4)
				img.Pix[offset+2] = byte((x + y + frame*8) / 8)
				img.Pix[offset+3] = 255
				if x >= 100+frame*16 && x < 300+frame*16 && y >= 150 && y < 350 {
					img.Pix[offset], img.Pix[offset+1], img.Pix[offset+2] = 230, 40, 80
				}
			}
		}
		frames[frame] = img
	}

	return frames
}

func BenchmarkI420(b *testing.B) {
	for _, size := range []struct {
		name          string
		width, height int
	}{
		{"Native1024x768", 1024, 768},
		{"Scaled800x600", 800, 600},
	} {
		b.Run(size.name, func(b *testing.B) {
			frames := benchmarkFrames(size.width, size.height)
			dst := make([]byte, 1024*768*3/2)
			b.SetBytes(1024 * 768 * 4)
			b.ReportAllocs()
			var frame int
			for b.Loop() {
				scaleI420(dst, 1024, 768, frames[frame%len(frames)])
				frame++
			}
		})
	}
}

func BenchmarkEncode(b *testing.B) {
	if !Available() {
		b.Skip("requires OpenH264")
	}
	frames := benchmarkFrames(1024, 768)
	encoder, err := NewEncoder(1024, 768)
	if err != nil {
		b.Fatal(err)
	}
	defer encoder.Close()
	b.ReportAllocs()
	var frame int
	for b.Loop() {
		if _, err := encoder.Encode(frames[frame%len(frames)], false); err != nil {
			b.Fatal(err)
		}
		frame++
	}
}
