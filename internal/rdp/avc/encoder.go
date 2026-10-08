// Package avc provides the optional OpenH264 encoder used by the RDP graphics
// pipeline. Build with -tags openh264 and cgo to enable the native codec.
package avc

import (
	"bytes"
	"errors"
	"image"
)

// FrameRate is the maximum frame rate of the RDP AVC420 encoder.
const FrameRate = 60

const maxDimension = 4096

var (
	ErrUnavailable = errors.New("OpenH264 support requires rebuilding with CGO_ENABLED=1 and -tags openh264")
	ErrGeometry    = errors.New("invalid OpenH264 frame geometry")
	ErrCodec       = errors.New("OpenH264 operation failed")
	ErrClosed      = errors.New("OpenH264 encoder is closed")
)

type nativeEncoder interface {
	encode([]byte, bool, int64) ([]byte, error)
	close()
}

// Encoder produces Annex-B AVC access units, including parameter sets with IDR
// frames. One goroutine must own the encoder, including calls to Close.
type Encoder struct {
	codec         nativeEncoder
	width, height int
	i420          []byte
	previous      []byte
	frame         int64
}

// NewEncoder fixes the output size for the lifetime of the encoder. Dimensions
// must be even and between 16 and 4096 pixels, inclusive.
func NewEncoder(width, height int) (*Encoder, error) {
	if width < 16 || height < 16 || width > maxDimension || height > maxDimension || width%2 != 0 || height%2 != 0 {
		return nil, ErrGeometry
	}

	codec, err := newNativeEncoder(width, height)
	if err != nil {
		return nil, err
	}

	return &Encoder{
		codec: codec, width: width, height: height,
		i420: make([]byte, width*height*3/2),
	}, nil
}

// Encode scales img to the session dimensions and encodes it. Force requests an
// independent IDR frame, for example when a client asks for a complete refresh.
// The returned slice is owned by the caller and survives the next Encode call.
// Identical I420 frames return nil unless force is true.
func (e *Encoder) Encode(img *image.RGBA, force bool) ([]byte, error) {
	if e.codec == nil {
		return nil, ErrClosed
	}
	if !validRGBA(img) {
		return nil, ErrGeometry
	}

	scaleI420(e.i420, e.width, e.height, img)
	if !force && bytes.Equal(e.i420, e.previous) {
		return nil, nil
	}
	data, err := e.codec.encode(e.i420, force, e.frame*1000/FrameRate)
	e.frame++
	if err == nil {
		e.previous, e.i420 = e.i420, e.previous
		if e.i420 == nil {
			e.i420 = make([]byte, len(e.previous))
		}
	}

	return data, err
}

// Close releases all native buffers and is safe to call more than once.
func (e *Encoder) Close() {
	if e.codec != nil {
		e.codec.close()
		e.codec = nil
		e.i420 = nil
		e.previous = nil
	}
}

func validRGBA(img *image.RGBA) bool {
	if img == nil || img.Rect.Empty() {
		return false
	}

	w, h := img.Rect.Dx(), img.Rect.Dy()
	if w <= 0 || h <= 0 {
		return false
	}
	// Check using division before multiplication so malformed image metadata
	// cannot overflow an offset and panic during conversion.
	return w <= len(img.Pix)/4 && img.Stride >= w*4 &&
		h-1 <= (len(img.Pix)-w*4)/img.Stride
}

// MS-RDPEGFX 3.3.8.3.1 requires full-range BT.709, rather than the studio-range
// BT.601 used by many general-purpose I420 conversion routines. Average each
// 2x2 source block for the subsampled chroma planes.
func scaleI420(dst []byte, width, height int, img *image.RGBA) {
	luma := width * height
	chroma := luma / 4
	// Mapping columns once avoids repeated integer divisions for every pixel.
	// NewEncoder bounds width, so this fixed-size scratch array stays on the
	// stack without allocating a lookup table for every frame.
	var columns [maxDimension]int
	for x := range width {
		columns[x] = x * img.Rect.Dx() / width * 4
	}
	var pos int
	for y := 0; y < height; y += 2 {
		row0 := img.Pix[(y*img.Rect.Dy()/height)*img.Stride:]
		row1 := img.Pix[((y+1)*img.Rect.Dy()/height)*img.Stride:]
		luma0 := dst[y*width : (y+1)*width]
		luma1 := dst[(y+1)*width : (y+2)*width]
		for x := 0; x+1 < width; x += 2 {
			p00 := row0[columns[x]:][:3]
			p01 := row0[columns[x+1]:][:3]
			p10 := row1[columns[x]:][:3]
			p11 := row1[columns[x+1]:][:3]
			r00, g00, b00 := int(p00[0]), int(p00[1]), int(p00[2])
			r01, g01, b01 := int(p01[0]), int(p01[1]), int(p01[2])
			r10, g10, b10 := int(p10[0]), int(p10[1]), int(p10[2])
			r11, g11, b11 := int(p11[0]), int(p11[1]), int(p11[2])
			luma0[x] = byte((54*r00 + 183*g00 + 18*b00) >> 8)
			luma0[x+1] = byte((54*r01 + 183*g01 + 18*b01) >> 8)
			luma1[x] = byte((54*r10 + 183*g10 + 18*b10) >> 8)
			luma1[x+1] = byte((54*r11 + 183*g11 + 18*b11) >> 8)
			red := (r00 + r01 + r10 + r11) / 4
			green := (g00 + g01 + g10 + g11) / 4
			blue := (b00 + b01 + b10 + b11) / 4
			dst[luma+pos] = byte(((-29*red - 99*green + 128*blue) >> 8) + 128)
			dst[luma+chroma+pos] = byte(((128*red - 116*green - 12*blue) >> 8) + 128)
			pos++
		}
	}
}
