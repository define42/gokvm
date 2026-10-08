package rdp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
)

var ErrBitmapFormat = errors.New("unsupported RDP bitmap geometry or color depth")

const bitmapTileSize = 64

// BitmapEncoder scales the console into a fixed session desktop and sends only
// changed tiles. Each update contains one uncompressed, bottom-up bitmap, as
// specified by MS-RDPBCGR 2.2.9.1.1.3.1.2. Small tiles fit the MCS PER envelope.
type BitmapEncoder struct {
	width, height, bytesPerPixel int
	previous                     []byte
}

func NewBitmapEncoder(width, height, bitsPerPixel int) (*BitmapEncoder, error) {
	if width <= 0 || height <= 0 || width > 4096 || height > 4096 ||
		(bitsPerPixel != 16 && bitsPerPixel != 24 && bitsPerPixel != 32) {
		return nil, ErrBitmapFormat
	}

	return &BitmapEncoder{width: width, height: height, bytesPerPixel: bitsPerPixel / 8}, nil
}

// WriteFrame calls send with TS_UPDATE_BITMAP_DATA payloads (including the
// updateType field). A failed send leaves the previous frame unchanged, so a
// retry retransmits every tile that might not have reached the client.
func (e *BitmapEncoder) WriteFrame(img *image.RGBA, force bool, send func([]byte) error) error {
	if img == nil || img.Bounds().Empty() {
		return nil
	}

	pixels := e.scale(img)
	for top := 0; top < e.height; top += bitmapTileSize {
		for left := 0; left < e.width; left += bitmapTileSize {
			width := min(bitmapTileSize, e.width-left)
			height := min(bitmapTileSize, e.height-top)
			if !force && e.unchanged(pixels, left, top, width, height) {
				continue
			}

			if err := send(e.tile(pixels, left, top, width, height)); err != nil {
				return err
			}
		}
	}

	e.previous = pixels

	return nil
}

func (e *BitmapEncoder) scale(img *image.RGBA) []byte {
	packed := make([]byte, e.width*e.height*e.bytesPerPixel)
	bounds := img.Bounds()
	for y := 0; y < e.height; y++ {
		srcY := bounds.Min.Y + y*bounds.Dy()/e.height
		for x := 0; x < e.width; x++ {
			srcX := bounds.Min.X + x*bounds.Dx()/e.width
			src := img.PixOffset(srcX, srcY)
			dst := (y*e.width + x) * e.bytesPerPixel
			r, g, b := img.Pix[src], img.Pix[src+1], img.Pix[src+2]
			if e.bytesPerPixel == 2 {
				value := uint16(r>>3)<<11 | uint16(g>>2)<<5 | uint16(b>>3)
				binary.LittleEndian.PutUint16(packed[dst:], value)
			} else {
				packed[dst], packed[dst+1], packed[dst+2] = b, g, r
			}
		}
	}

	return packed
}

func (e *BitmapEncoder) unchanged(pixels []byte, left, top, width, height int) bool {
	if len(e.previous) != len(pixels) {
		return false
	}

	for y := top; y < top+height; y++ {
		start := (y*e.width + left) * e.bytesPerPixel
		end := start + width*e.bytesPerPixel
		if !bytes.Equal(pixels[start:end], e.previous[start:end]) {
			return false
		}
	}

	return true
}

func (e *BitmapEncoder) tile(pixels []byte, left, top, width, height int) []byte {
	stride := (width*e.bytesPerPixel + 3) &^ 3
	data := make([]byte, 22+stride*height)
	fields := []uint16{
		1, 1, // Bitmap update; one rectangle.
		uint16(left), uint16(top), uint16(left + width - 1), uint16(top + height - 1),
		uint16(width), uint16(height), uint16(e.bytesPerPixel * 8), 0, uint16(stride * height),
	}
	for i, field := range fields {
		binary.LittleEndian.PutUint16(data[i*2:], field)
	}

	for row := 0; row < height; row++ {
		src := ((top+height-1-row)*e.width + left) * e.bytesPerPixel
		copy(data[22+row*stride:], pixels[src:src+width*e.bytesPerPixel])
	}

	return data
}
