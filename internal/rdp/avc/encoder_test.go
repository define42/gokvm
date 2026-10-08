package avc

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"testing"
)

func TestGeometry(t *testing.T) {
	t.Parallel()
	for _, size := range [][2]int{{0, 32}, {32, -1}, {15, 32}, {32, 17}, {4098, 32}, {32, 4098}} {
		if encoder, err := NewEncoder(size[0], size[1]); !errors.Is(err, ErrGeometry) || encoder != nil {
			t.Fatalf("NewEncoder(%v) = %v, %v", size, encoder, err)
		}
	}
}

func TestI420ColorConversion(t *testing.T) {
	t.Parallel()
	// The expected values come directly from the MS-RDPEGFX color matrix.
	for _, test := range []struct {
		name string
		rgb  color.RGBA
		yuv  [3]byte
	}{
		{"black", color.RGBA{A: 255}, [3]byte{0, 128, 128}},
		{"white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, [3]byte{254, 128, 128}},
		{"red", color.RGBA{R: 255, A: 255}, [3]byte{53, 99, 255}},
		{"green", color.RGBA{G: 255, A: 255}, [3]byte{182, 29, 12}},
		{"blue", color.RGBA{B: 255, A: 255}, [3]byte{17, 255, 116}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			img := image.NewRGBA(image.Rect(0, 0, 2, 2))
			for y := range 2 {
				for x := range 2 {
					img.SetRGBA(x, y, test.rgb)
				}
			}
			got := make([]byte, 6)
			scaleI420(got, 2, 2, img)
			want := []byte{test.yuv[0], test.yuv[0], test.yuv[0], test.yuv[0], test.yuv[1], test.yuv[2]}
			if !bytes.Equal(got, want) {
				t.Fatalf("I420 = %v, want %v", got, want)
			}
		})
	}
}

func TestI420SubimageAndScaling(t *testing.T) {
	t.Parallel()
	img := image.NewRGBA(image.Rect(-5, -7, 5, 3))
	img.SetRGBA(-3, -5, color.RGBA{R: 255, A: 255})
	img.SetRGBA(-2, -5, color.RGBA{G: 255, A: 255})
	img.SetRGBA(-3, -4, color.RGBA{B: 255, A: 255})
	img.SetRGBA(-2, -4, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	sub := img.SubImage(image.Rect(-3, -5, -1, -3)).(*image.RGBA)
	got := make([]byte, 24)
	scaleI420(got, 4, 4, sub)
	want := []byte{
		53, 53, 182, 182, 53, 53, 182, 182,
		17, 17, 254, 254, 17, 17, 254, 254,
		99, 29, 255, 128, 255, 12, 116, 128,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("scaled I420 = %v, want %v", got, want)
	}

	got = make([]byte, 6)
	scaleI420(got, 2, 2, sub)
	if !bytes.Equal(got, []byte{53, 182, 17, 254, 128, 128}) {
		t.Fatalf("averaged chroma = %v", got)
	}
}

type recordingCodec struct {
	forces     []bool
	timestamps []int64
	closed     int
}

func (c *recordingCodec) encode(_ []byte, force bool, timestamp int64) ([]byte, error) {
	c.forces = append(c.forces, force)
	c.timestamps = append(c.timestamps, timestamp)

	return []byte{1}, nil
}

func (c *recordingCodec) close() { c.closed++ }

func TestEncoderRefreshAndClose(t *testing.T) {
	t.Parallel()
	codec := &recordingCodec{}
	encoder := &Encoder{codec: codec, width: 16, height: 16, i420: make([]byte, 16*16*3/2)}
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for _, force := range []bool{false, false, true} {
		if _, err := encoder.Encode(img, force); err != nil {
			t.Fatal(err)
		}
	}
	if len(codec.forces) != 2 || codec.forces[0] || !codec.forces[1] || codec.timestamps[1] != 33 {
		t.Fatalf("encode calls: force=%v timestamp=%v", codec.forces, codec.timestamps)
	}

	img.SetRGBA(0, 0, color.RGBA{R: 255})
	if _, err := encoder.Encode(img, false); err != nil || len(codec.forces) != 3 {
		t.Fatalf("changed frame = %v, %d calls", err, len(codec.forces))
	}
	encoder.Close()
	encoder.Close()
	if codec.closed != 1 {
		t.Fatalf("native close called %d times", codec.closed)
	}
	if _, err := encoder.Encode(img, false); !errors.Is(err, ErrClosed) {
		t.Fatalf("Encode after Close = %v", err)
	}
}

func TestRejectMalformedRGBA(t *testing.T) {
	t.Parallel()
	codec := &recordingCodec{}
	encoder := &Encoder{codec: codec, width: 16, height: 16, i420: make([]byte, 16*16*3/2)}
	defer encoder.Close()
	for _, img := range []*image.RGBA{
		nil,
		{},
		{Pix: make([]byte, 4), Stride: 8, Rect: image.Rect(0, 0, 2, 2)},
		{Pix: make([]byte, 16), Stride: 0, Rect: image.Rect(0, 0, 2, 2)},
		{Pix: make([]byte, 16), Stride: -1, Rect: image.Rect(0, 0, 2, 2)},
		{Pix: make([]byte, 16), Stride: 8, Rect: image.Rect(0, 0, 2, 3)},
	} {
		if _, err := encoder.Encode(img, false); !errors.Is(err, ErrGeometry) {
			t.Fatalf("malformed RGBA returned %v", err)
		}
	}
	if len(codec.forces) != 0 {
		t.Fatal("malformed image reached native encoder")
	}
}
