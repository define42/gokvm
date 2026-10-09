// Package damage maps source image changes to scaled output regions.
package damage

import "image"

// Map returns exactly the destination pixels whose nearest-neighbor samples
// fall inside rect. Both rectangles use image coordinates, including nonzero
// origins. A source pixel skipped by downscaling maps to an empty rectangle.
func Map(rect, source, output image.Rectangle) image.Rectangle {
	if source.Empty() || output.Empty() {
		return image.Rectangle{}
	}
	rect = rect.Intersect(source)
	if rect.Empty() {
		return image.Rectangle{}
	}
	ceil := func(value, scale, divisor int) int {
		return int((int64(value)*int64(scale) + int64(divisor) - 1) / int64(divisor))
	}

	return image.Rect(
		output.Min.X+ceil(rect.Min.X-source.Min.X, output.Dx(), source.Dx()),
		output.Min.Y+ceil(rect.Min.Y-source.Min.Y, output.Dy(), source.Dy()),
		output.Min.X+ceil(rect.Max.X-source.Min.X, output.Dx(), source.Dx()),
		output.Min.Y+ceil(rect.Max.Y-source.Min.Y, output.Dy(), source.Dy()),
	)
}

// Align expands rect to blocks measured from bounds.Min, then clips to bounds.
// Chroma conversion uses two-pixel blocks and bitmap updates use 64-pixel tiles.
func Align(rect image.Rectangle, block int, bounds image.Rectangle) image.Rectangle {
	rect = rect.Intersect(bounds)
	if rect.Empty() || block < 1 {
		return image.Rectangle{}
	}
	rect = rect.Sub(bounds.Min)
	rect.Min.X = rect.Min.X / block * block
	rect.Min.Y = rect.Min.Y / block * block
	rect.Max.X = (rect.Max.X + block - 1) / block * block
	rect.Max.Y = (rect.Max.Y + block - 1) / block * block

	return rect.Add(bounds.Min).Intersect(bounds)
}

// ValidRGBA checks image metadata before conversion reads its pixel buffer.
func ValidRGBA(img *image.RGBA) bool {
	if img == nil || img.Rect.Empty() {
		return false
	}
	w, h := img.Rect.Dx(), img.Rect.Dy()
	if w <= 0 || h <= 0 {
		return false
	}
	// Divide before multiplying so malformed metadata cannot wrap an offset.
	return w <= len(img.Pix)/4 && img.Stride >= w*4 &&
		h-1 <= (len(img.Pix)-w*4)/img.Stride
}
