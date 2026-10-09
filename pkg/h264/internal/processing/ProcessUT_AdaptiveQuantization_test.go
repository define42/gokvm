package processing

// Port of test/processing/ProcessUT_AdaptiveQuantization.cpp.

import (
	"math/rand"
	"testing"
)

func fillWithRandomData(r *rand.Rand, p []uint8) {
	for i := range p {
		p[i] = uint8(r.Intn(256))
	}
}

func SampleVariance16x16_ref(pRefY []uint8, iRefStride int32, pSrcY []uint8, iSrcStride int32,
	pMotionTexture *SMotionTextureUnit) {
	var uiCurSquare, uiSquare uint32
	var uiCurSum, uiSum uint16

	for y := int32(0); y < MB_WIDTH_LUMA; y++ {
		for x := int32(0); x < MB_WIDTH_LUMA; x++ {
			r := int32(pRefY[y*iRefStride+x])
			s := int32(pSrcY[y*iSrcStride+x])
			d := r - s
			if d < 0 {
				d = -d
			}
			uiDiff := uint32(d)
			uiSum += uint16(uiDiff)
			uiSquare += uiDiff * uiDiff

			uiCurSum += uint16(s)
			uiCurSquare += uint32(s * s)
		}
	}

	uiSum = uiSum >> 8
	pMotionTexture.UiMotionIndex = uint16((uiSquare >> 8) - uint32(uiSum)*uint32(uiSum))

	uiCurSum = uiCurSum >> 8
	pMotionTexture.UiTextureIndex = uint16((uiCurSquare >> 8) - uint32(uiCurSum)*uint32(uiCurSum))
}

func TestAdaptiveQuantization_SampleVariance16x16_c(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	pRefY := make([]uint8, 32*16)
	pSrcY := make([]uint8, 48*16)
	var pMotionTexture [2]SMotionTextureUnit
	for iter := 0; iter < 100; iter++ {
		fillWithRandomData(r, pRefY)
		fillWithRandomData(r, pSrcY)
		SampleVariance16x16_ref(pRefY, 32, pSrcY, 48, &pMotionTexture[0])
		SampleVariance16x16_c(pRefY, 0, 32, pSrcY, 0, 48, &pMotionTexture[1])
		if pMotionTexture[0] != pMotionTexture[1] {
			t.Fatalf("random: ref %+v != c %+v", pMotionTexture[0], pMotionTexture[1])
		}
	}
	clear(pRefY)
	for i := range pSrcY {
		pSrcY[i] = 255
	}
	SampleVariance16x16_ref(pRefY, 32, pSrcY, 48, &pMotionTexture[0])
	SampleVariance16x16_c(pRefY, 0, 32, pSrcY, 0, 48, &pMotionTexture[1])
	if pMotionTexture[0] != pMotionTexture[1] {
		t.Fatalf("extreme: ref %+v != c %+v", pMotionTexture[0], pMotionTexture[1])
	}
}
