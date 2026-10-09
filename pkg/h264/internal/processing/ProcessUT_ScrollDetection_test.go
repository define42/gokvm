package processing

// Port of test/processing/ProcessUT_ScrollDetection.cpp.

import (
	"math/rand"
	"testing"
)

func RandomPixelDataGenerator(r *rand.Rand, p []uint8, iWidth, iHeight, iStride int32) {
	for j := int32(0); j < iHeight; j++ {
		for i := int32(0); i < iWidth; i++ {
			p[j*iStride+i] = uint8(r.Intn(256))
		}
	}
}

func TestScrollDetectionTest_TestScroll(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	iWidthSets := [4]int32{640, 1024, 1280, 1980}
	iHeightSets := [4]int32{360, 768, 720, 1080}

	for rep := 0; rep < 5; rep++ {
		for i := 0; i < 4; i++ {
			iWidth := iWidthSets[i]
			iHeight := iHeightSets[i]
			iStride := iWidth + 16
			pSrc := make([]uint8, iHeight*iStride)
			pRef := make([]uint8, iHeight*iStride)
			RandomPixelDataGenerator(r, pRef, iWidth, iHeight, iStride)

			iMvRange := iHeight / 3
			iScrollMv := int32(r.Intn(int(iMvRange<<1))) - iMvRange

			for j := int32(0); j < iHeight; j++ {
				row := pSrc[j*iStride : j*iStride+iWidth]
				if (j+iScrollMv) >= 0 && (j+iScrollMv) < iHeight {
					copy(row, pRef[(j+iScrollMv)*iStride:(j+iScrollMv)*iStride+iWidth])
				} else {
					for k := range row {
						row[k] = uint8(r.Intn(256))
					}
				}
			}

			var sSrcMap, sRefMap SPixMap
			sSrcMap.PPixel[0] = pSrc
			sRefMap.PPixel[0] = pRef
			sSrcMap.IStride[0] = iStride
			sRefMap.IStride[0] = iStride
			sSrcMap.SRect.IRectWidth = iWidth
			sRefMap.SRect.IRectWidth = iWidth
			sSrcMap.SRect.IRectHeight = iHeight
			sRefMap.SRect.IRectHeight = iHeight

			var sScrollDetectionResult SScrollDetectionParam

			pTest := NewCScrollDetection(0)
			iMethodIdx := METHOD_SCROLL_DETECTION

			pTest.Set(iMethodIdx, &sScrollDetectionResult)
			ret := pTest.Process(iMethodIdx, &sSrcMap, &sRefMap)
			if ret != 0 {
				t.Fatalf("Process ret %d", ret)
			}
			pTest.Get(iMethodIdx, &sScrollDetectionResult)

			if !sScrollDetectionResult.BScrollDetectFlag {
				t.Errorf("%dx%d mv %d: scroll not detected", iWidth, iHeight, iScrollMv)
			}
			if sScrollDetectionResult.IScrollMvY != iScrollMv {
				t.Errorf("%dx%d: IScrollMvY %d, want %d", iWidth, iHeight, sScrollDetectionResult.IScrollMvY, iScrollMv)
			}
		}
	}
}
