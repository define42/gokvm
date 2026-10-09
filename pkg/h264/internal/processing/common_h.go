package processing

// Port of codec/processing/src/common/common.h.
//
// The C header pulls WelsSampleSad8x8_c / WelsSampleSad16x16_c from
// codec/common/src/sad_common.cpp and WelsI16x16LumaPredV_c /
// WelsI16x16LumaPredH_c from codec/common/src/intra_pred_common.cpp; local
// copies of those _c functions live at the bottom of this file so the package
// stays independent of internal/common.
//
// Pixel pointers are (slice, offset) pairs, see PORTING.md.

// GetIntraPred: pPred is a plain 16x16 (stride 16) buffer; pRef is a picture
// plane pointer (reads above / left of it).
type GetIntraPred func(pPred []uint8, pRef []uint8, iRefOff int, kiStride int32)

type GetIntraPredPtr = GetIntraPred

type SadFunc func(pSrcY []uint8, iSrcYOff int, iSrcStrideY int32, pRefY []uint8, iRefYOff int, iRefStrideY int32) int32

type SadFuncPtr = SadFunc

type Sad16x16Func func(pSrcY []uint8, iSrcYOff int, iSrcStrideY int32, pRefY []uint8, iRefYOff int, iRefStrideY int32) int32

type PSad16x16Func = Sad16x16Func

// WelsSampleSad8x8_c (codec/common/src/sad_common.cpp)
func WelsSampleSad8x8_c(pSample1 []uint8, iOff1 int, iStride1 int32, pSample2 []uint8, iOff2 int, iStride2 int32) int32 {
	var iSadSum int32
	pSrc1 := iOff1
	pSrc2 := iOff2
	for i := 0; i < 8; i++ {
		s1 := pSample1[pSrc1 : pSrc1+8]
		s2 := pSample2[pSrc2 : pSrc2+8]
		for k := 0; k < 8; k++ {
			iSadSum += WELS_ABS(int32(s1[k]) - int32(s2[k]))
		}
		pSrc1 += int(iStride1)
		pSrc2 += int(iStride2)
	}
	return iSadSum
}

// WelsSampleSad16x16_c (codec/common/src/sad_common.cpp)
func WelsSampleSad16x16_c(pSample1 []uint8, iOff1 int, iStride1 int32, pSample2 []uint8, iOff2 int, iStride2 int32) int32 {
	var iSadSum int32
	iSadSum += WelsSampleSad8x8_c(pSample1, iOff1, iStride1, pSample2, iOff2, iStride2)
	iSadSum += WelsSampleSad8x8_c(pSample1, iOff1+8, iStride1, pSample2, iOff2+8, iStride2)
	iSadSum += WelsSampleSad8x8_c(pSample1, iOff1+int(iStride1<<3), iStride1, pSample2, iOff2+int(iStride2<<3), iStride2)
	iSadSum += WelsSampleSad8x8_c(pSample1, iOff1+int(iStride1<<3)+8, iStride1, pSample2, iOff2+int(iStride2<<3)+8, iStride2)
	return iSadSum
}

// WelsI16x16LumaPredV_c (codec/common/src/intra_pred_common.cpp)
func WelsI16x16LumaPredV_c(pPred []uint8, pRef []uint8, iRefOff int, kiStride int32) {
	kpSrc := pRef[iRefOff-int(kiStride) : iRefOff-int(kiStride)+16]
	for i := 0; i < 16; i++ {
		copy(pPred[i*16:i*16+16], kpSrc)
	}
}

// WelsI16x16LumaPredH_c (codec/common/src/intra_pred_common.cpp)
func WelsI16x16LumaPredH_c(pPred []uint8, pRef []uint8, iRefOff int, kiStride int32) {
	iStridex15 := (kiStride << 4) - kiStride
	iPredStridex15 := 240 //(iPredStride<<4)-iPredStride;
	const iPredStride = 16
	for i := 0; i < 16; i++ {
		kuiSrc8 := pRef[iRefOff+int(iStridex15)-1]
		row := pPred[iPredStridex15 : iPredStridex15+16]
		for k := range row {
			row[k] = kuiSrc8
		}
		iStridex15 -= kiStride
		iPredStridex15 -= iPredStride
	}
}
