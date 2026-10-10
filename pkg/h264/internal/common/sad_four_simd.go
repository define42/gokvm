package common

type sampleSadFunc func([]uint8, int, int32, []uint8, int, int32) int32

func sampleSadFourSSE2(fn sampleSadFunc, a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32, sad []int32) {
	sad[0] = fn(a, aOff, aStride, b, bOff-int(bStride), bStride)
	sad[1] = fn(a, aOff, aStride, b, bOff+int(bStride), bStride)
	sad[2] = fn(a, aOff, aStride, b, bOff-1, bStride)
	sad[3] = fn(a, aOff, aStride, b, bOff+1, bStride)
}

func WelsSampleSadFour16x16_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32, sad []int32) {
	sampleSadFourSSE2(WelsSampleSad16x16_sse2, a, aOff, aStride, b, bOff, bStride, sad)
}

func WelsSampleSadFour16x8_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32, sad []int32) {
	sampleSadFourSSE2(WelsSampleSad16x8_sse2, a, aOff, aStride, b, bOff, bStride, sad)
}

func WelsSampleSadFour8x16_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32, sad []int32) {
	sampleSadFourSSE2(WelsSampleSad8x16_sse2, a, aOff, aStride, b, bOff, bStride, sad)
}

func WelsSampleSadFour8x8_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32, sad []int32) {
	sampleSadFourSSE2(WelsSampleSad8x8_sse2, a, aOff, aStride, b, bOff, bStride, sad)
}

func WelsSampleSadFour8x4_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32, sad []int32) {
	sampleSadFourSSE2(WelsSampleSad8x4_sse2, a, aOff, aStride, b, bOff, bStride, sad)
}

func WelsSampleSadFour4x8_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32, sad []int32) {
	sampleSadFourSSE2(WelsSampleSad4x8_sse2, a, aOff, aStride, b, bOff, bStride, sad)
}

func WelsSampleSadFour4x4_sse2(a []uint8, aOff int, aStride int32, b []uint8, bOff int, bStride int32, sad []int32) {
	sampleSadFourSSE2(WelsSampleSad4x4_sse2, a, aOff, aStride, b, bOff, bStride, sad)
}
