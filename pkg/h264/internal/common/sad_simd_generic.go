//go:build !amd64

package common

func WelsSampleSad16x16_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32) int32 {
	return WelsSampleSad16x16_c(a, aOff, aStride, b, bOff, bStride)
}

func WelsSampleSad16x8_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32) int32 {
	return WelsSampleSad16x8_c(a, aOff, aStride, b, bOff, bStride)
}

func WelsSampleSad8x16_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32) int32 {
	return WelsSampleSad8x16_c(a, aOff, aStride, b, bOff, bStride)
}

func WelsSampleSad8x8_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32) int32 {
	return WelsSampleSad8x8_c(a, aOff, aStride, b, bOff, bStride)
}

func WelsSampleSad8x4_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32) int32 {
	return WelsSampleSad8x4_c(a, aOff, aStride, b, bOff, bStride)
}

func WelsSampleSad4x8_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32) int32 {
	return WelsSampleSad4x8_c(a, aOff, aStride, b, bOff, bStride)
}

func WelsSampleSad4x4_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32) int32 {
	return WelsSampleSad4x4_c(a, aOff, aStride, b, bOff, bStride)
}
