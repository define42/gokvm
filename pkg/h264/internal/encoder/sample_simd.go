package encoder

type sampleSatdFunc func([]uint8, int, int32, []uint8, int, int32) int32

func sampleSatd8x4SIMD(fn sampleSatdFunc, a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return fn(a, aOff, aStride, ref, refOff, refStride) +
		fn(a, aOff+4, aStride, ref, refOff+4, refStride)
}

func sampleSatd4x8SIMD(fn sampleSatdFunc, a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return fn(a, aOff, aStride, ref, refOff, refStride) +
		fn(a, aOff+int(aStride)*4, aStride, ref, refOff+int(refStride)*4, refStride)
}

func sampleSatd16x8SIMD(fn sampleSatdFunc, a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return fn(a, aOff, aStride, ref, refOff, refStride) +
		fn(a, aOff+8, aStride, ref, refOff+8, refStride)
}

func sampleSatd8x16SIMD(fn sampleSatdFunc, a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return fn(a, aOff, aStride, ref, refOff, refStride) +
		fn(a, aOff+int(aStride)*8, aStride, ref, refOff+int(refStride)*8, refStride)
}

func sampleSatd16x16SIMD(fn sampleSatdFunc, a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	a8, ref8 := int(aStride)*8, int(refStride)*8
	return fn(a, aOff, aStride, ref, refOff, refStride) +
		fn(a, aOff+8, aStride, ref, refOff+8, refStride) +
		fn(a, aOff+a8, aStride, ref, refOff+ref8, refStride) +
		fn(a, aOff+a8+8, aStride, ref, refOff+ref8+8, refStride)
}

func WelsSampleSatd8x4_sse41(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return sampleSatd8x4SIMD(WelsSampleSatd4x4_sse41, a, aOff, aStride, ref, refOff, refStride)
}

func WelsSampleSatd4x8_sse41(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return sampleSatd4x8SIMD(WelsSampleSatd4x4_sse41, a, aOff, aStride, ref, refOff, refStride)
}

func WelsSampleSatd16x8_sse41(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return sampleSatd16x8SIMD(WelsSampleSatd8x8_sse41, a, aOff, aStride, ref, refOff, refStride)
}

func WelsSampleSatd8x16_sse41(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return sampleSatd8x16SIMD(WelsSampleSatd8x8_sse41, a, aOff, aStride, ref, refOff, refStride)
}

func WelsSampleSatd16x16_sse41(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return sampleSatd16x16SIMD(WelsSampleSatd8x8_sse41, a, aOff, aStride, ref, refOff, refStride)
}

func WelsSampleSatd16x8_avx2(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return sampleSatd16x8SIMD(WelsSampleSatd8x8_avx2, a, aOff, aStride, ref, refOff, refStride)
}

func WelsSampleSatd8x16_avx2(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return sampleSatd8x16SIMD(WelsSampleSatd8x8_avx2, a, aOff, aStride, ref, refOff, refStride)
}

func WelsSampleSatd16x16_avx2(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	return sampleSatd16x16SIMD(WelsSampleSatd8x8_avx2, a, aOff, aStride, ref, refOff, refStride)
}
