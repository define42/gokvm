//go:build amd64

package encoder

// welsDctT4SSE2 computes one 4x4 forward integer transform. All amd64
// processors supported by Go provide SSE2.
//
//go:noescape
func welsDctT4SSE2(dct []int16, pixel1 []byte, stride1 int, pixel2 []byte, stride2 int)

//go:noescape
func welsDctFourT4SSE2(dct []int16, pixel1 []byte, stride1 int, pixel2 []byte, stride2 int)

// WelsDctT4_sse2 is the SSE2 implementation of WelsDctT4_c.
func WelsDctT4_sse2(
	dct []int16,
	pixel1 []uint8,
	pixel1Off int,
	stride1 int32,
	pixel2 []uint8,
	pixel2Off int,
	stride2 int32,
) {
	s1 := int(stride1)
	s2 := int(stride2)
	_ = dct[15]
	checkSampleSIMDBlock(pixel1, pixel1Off, stride1, 4, 4)
	checkSampleSIMDBlock(pixel2, pixel2Off, stride2, 4, 4)
	welsDctT4SSE2(dct, pixel1[pixel1Off:], s1, pixel2[pixel2Off:], s2)
}

// WelsDctFourT4_sse2 computes the four 4x4 transforms in an 8x8 block.
// Coefficients use OpenH264's block order: top-left, top-right,
// bottom-left, bottom-right.
func WelsDctFourT4_sse2(
	dct []int16,
	pixel1 []uint8,
	pixel1Off int,
	stride1 int32,
	pixel2 []uint8,
	pixel2Off int,
	stride2 int32,
) {
	s1 := int(stride1)
	s2 := int(stride2)
	_ = dct[63]
	checkSampleSIMDBlock(pixel1, pixel1Off, stride1, 8, 8)
	checkSampleSIMDBlock(pixel2, pixel2Off, stride2, 8, 8)
	welsDctFourT4SSE2(dct, pixel1[pixel1Off:], s1, pixel2[pixel2Off:], s2)
}
