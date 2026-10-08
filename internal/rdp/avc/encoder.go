// Package avc provides the optional OpenH264 encoder used by the RDP graphics
// pipeline. Build with -tags openh264 and cgo to enable the native codec.
package avc

import (
	"bytes"
	"errors"
	"image"
)

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
	if width < 16 || height < 16 || width > 4096 || height > 4096 || width%2 != 0 || height%2 != 0 {
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
	data, err := e.codec.encode(e.i420, force, e.frame*1000/30)
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
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x += 2 {
			var red, green, blue int
			for dy := range 2 {
				sy := (y + dy) * img.Rect.Dy() / height
				for dx := range 2 {
					sx := (x + dx) * img.Rect.Dx() / width
					offset := sy*img.Stride + sx*4
					r, g, b := int(img.Pix[offset]), int(img.Pix[offset+1]), int(img.Pix[offset+2])
					dst[(y+dy)*width+x+dx] = byte((54*r + 183*g + 18*b) >> 8)
					red, green, blue = red+r, green+g, blue+b
				}
			}
			red, green, blue = red/4, green/4, blue/4
			pos := (y/2)*(width/2) + x/2
			dst[luma+pos] = byte(((-29*red - 99*green + 128*blue) >> 8) + 128)
			dst[luma+chroma+pos] = byte(((128*red - 116*green - 12*blue) >> 8) + 128)
		}
	}
}
