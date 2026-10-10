//go:build !amd64

package encoder

// WelsDctT4_sse2 falls back to the scalar transform on platforms without
// SSE2.
func WelsDctT4_sse2(
	dct []int16,
	pixel1 []uint8,
	pixel1Off int,
	stride1 int32,
	pixel2 []uint8,
	pixel2Off int,
	stride2 int32,
) {
	WelsDctT4_c(dct, pixel1, pixel1Off, stride1, pixel2, pixel2Off, stride2)
}

// WelsDctFourT4_sse2 falls back to the scalar transform on platforms without
// SSE2.
func WelsDctFourT4_sse2(
	dct []int16,
	pixel1 []uint8,
	pixel1Off int,
	stride1 int32,
	pixel2 []uint8,
	pixel2Off int,
	stride2 int32,
) {
	WelsDctFourT4_c(dct, pixel1, pixel1Off, stride1, pixel2, pixel2Off, stride2)
}
