//go:build amd64

package encoder

//go:noescape
func satd4x4(src []byte, srcStride int, ref []byte, refStride int) int

//go:noescape
func satd8x8(src []byte, srcStride int, ref []byte, refStride int) int

//go:noescape
func satd8x8AVX2(src []byte, srcStride int, ref []byte, refStride int) int

func sampleSIMDBlockSlice(buf []byte, off int, stride int32, width, height int) []byte {
	if stride < 0 {
		panic("h264: SIMD sample buffer span out of range")
	}
	end := off + (height-1)*int(stride) + width
	return buf[off:end]
}

func checkSampleSIMDBlock(buf []byte, off int, stride int32, width, height int) {
	first := int64(off)
	last := first + int64(height-1)*int64(stride)
	if last < first {
		first, last = last, first
	}
	last += int64(width - 1)
	if first < 0 || last < first || last >= int64(len(buf)) {
		panic("h264: SIMD sample buffer span out of range")
	}
}

func WelsSampleSatd4x4_sse41(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	aBlock := sampleSIMDBlockSlice(a, aOff, aStride, 4, 4)
	refBlock := sampleSIMDBlockSlice(ref, refOff, refStride, 4, 4)
	return int32(satd4x4(aBlock, int(aStride), refBlock, int(refStride)))
}

func WelsSampleSatd8x8_sse41(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	aBlock := sampleSIMDBlockSlice(a, aOff, aStride, 8, 8)
	refBlock := sampleSIMDBlockSlice(ref, refOff, refStride, 8, 8)
	return int32(satd8x8(aBlock, int(aStride), refBlock, int(refStride)))
}

func WelsSampleSatd8x8_avx2(a []uint8, aOff int, aStride int32, ref []uint8, refOff int, refStride int32) int32 {
	aBlock := sampleSIMDBlockSlice(a, aOff, aStride, 8, 8)
	refBlock := sampleSIMDBlockSlice(ref, refOff, refStride, 8, 8)
	return int32(satd8x8AVX2(aBlock, int(aStride), refBlock, int(refStride)))
}
