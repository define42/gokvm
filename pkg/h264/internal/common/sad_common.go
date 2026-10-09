package common

// Port of codec/common/inc/sad_common.h and codec/common/src/sad_common.cpp
// (C code only).
//
// Signature of the WelsSampleSad*_c functions:
//
//	func(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32) int32
//
// and of the WelsSampleSadFour*_c functions (pSad receives 4 values: SAD
// against pSample2 shifted up, down, left and right by one sample):
//
//	func(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32, pSad []int32)

func WelsSampleSad4x4_c(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32) int32 {
	var iSadSum int32
	pSrc1 := iSample1Off
	pSrc2 := iSample2Off
	for i := 0; i < 4; i++ {
		iSadSum += WELS_ABS(int32(pSample1[pSrc1+0]) - int32(pSample2[pSrc2+0]))
		iSadSum += WELS_ABS(int32(pSample1[pSrc1+1]) - int32(pSample2[pSrc2+1]))
		iSadSum += WELS_ABS(int32(pSample1[pSrc1+2]) - int32(pSample2[pSrc2+2]))
		iSadSum += WELS_ABS(int32(pSample1[pSrc1+3]) - int32(pSample2[pSrc2+3]))

		pSrc1 += int(iStride1)
		pSrc2 += int(iStride2)
	}

	return iSadSum
}

func WelsSampleSad8x4_c(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32) int32 {
	var iSadSum int32
	iSadSum += WelsSampleSad4x4_c(pSample1, iSample1Off, iStride1, pSample2, iSample2Off, iStride2)
	iSadSum += WelsSampleSad4x4_c(pSample1, iSample1Off+4, iStride1, pSample2, iSample2Off+4, iStride2)
	return iSadSum
}

func WelsSampleSad4x8_c(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32) int32 {
	var iSadSum int32
	iSadSum += WelsSampleSad4x4_c(pSample1, iSample1Off, iStride1, pSample2, iSample2Off, iStride2)
	iSadSum += WelsSampleSad4x4_c(pSample1, iSample1Off+int(iStride1<<2), iStride1, pSample2, iSample2Off+int(iStride2<<2), iStride2)
	return iSadSum
}

func WelsSampleSad8x8_c(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32) int32 {
	var iSadSum int32
	pSrc1 := iSample1Off
	pSrc2 := iSample2Off
	for i := 0; i < 8; i++ {
		for k := 0; k < 8; k++ {
			iSadSum += WELS_ABS(int32(pSample1[pSrc1+k]) - int32(pSample2[pSrc2+k]))
		}

		pSrc1 += int(iStride1)
		pSrc2 += int(iStride2)
	}

	return iSadSum
}

func WelsSampleSad16x8_c(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32) int32 {
	var iSadSum int32

	iSadSum += WelsSampleSad8x8_c(pSample1, iSample1Off, iStride1, pSample2, iSample2Off, iStride2)
	iSadSum += WelsSampleSad8x8_c(pSample1, iSample1Off+8, iStride1, pSample2, iSample2Off+8, iStride2)

	return iSadSum
}

func WelsSampleSad8x16_c(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32) int32 {
	var iSadSum int32
	iSadSum += WelsSampleSad8x8_c(pSample1, iSample1Off, iStride1, pSample2, iSample2Off, iStride2)
	iSadSum += WelsSampleSad8x8_c(pSample1, iSample1Off+int(iStride1<<3), iStride1, pSample2, iSample2Off+int(iStride2<<3), iStride2)

	return iSadSum
}

func WelsSampleSad16x16_c(pSample1 []uint8, iSample1Off int, iStride1 int32, pSample2 []uint8, iSample2Off int, iStride2 int32) int32 {
	var iSadSum int32
	s1 := int(iStride1 << 3)
	s2 := int(iStride2 << 3)
	iSadSum += WelsSampleSad8x8_c(pSample1, iSample1Off, iStride1, pSample2, iSample2Off, iStride2)
	iSadSum += WelsSampleSad8x8_c(pSample1, iSample1Off+8, iStride1, pSample2, iSample2Off+8, iStride2)
	iSadSum += WelsSampleSad8x8_c(pSample1, iSample1Off+s1, iStride1, pSample2, iSample2Off+s2, iStride2)
	iSadSum += WelsSampleSad8x8_c(pSample1, iSample1Off+s1+8, iStride1, pSample2, iSample2Off+s2+8, iStride2)

	return iSadSum
}

func WelsSampleSadFour16x16_c(iSample1 []uint8, iSample1Off int, iStride1 int32, iSample2 []uint8, iSample2Off int, iStride2 int32, pSad []int32) {
	pSad[0] = WelsSampleSad16x16_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-int(iStride2), iStride2)
	pSad[1] = WelsSampleSad16x16_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+int(iStride2), iStride2)
	pSad[2] = WelsSampleSad16x16_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-1, iStride2)
	pSad[3] = WelsSampleSad16x16_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+1, iStride2)
}

func WelsSampleSadFour16x8_c(iSample1 []uint8, iSample1Off int, iStride1 int32, iSample2 []uint8, iSample2Off int, iStride2 int32, pSad []int32) {
	pSad[0] = WelsSampleSad16x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-int(iStride2), iStride2)
	pSad[1] = WelsSampleSad16x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+int(iStride2), iStride2)
	pSad[2] = WelsSampleSad16x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-1, iStride2)
	pSad[3] = WelsSampleSad16x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+1, iStride2)
}

func WelsSampleSadFour8x16_c(iSample1 []uint8, iSample1Off int, iStride1 int32, iSample2 []uint8, iSample2Off int, iStride2 int32, pSad []int32) {
	pSad[0] = WelsSampleSad8x16_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-int(iStride2), iStride2)
	pSad[1] = WelsSampleSad8x16_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+int(iStride2), iStride2)
	pSad[2] = WelsSampleSad8x16_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-1, iStride2)
	pSad[3] = WelsSampleSad8x16_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+1, iStride2)
}

func WelsSampleSadFour8x8_c(iSample1 []uint8, iSample1Off int, iStride1 int32, iSample2 []uint8, iSample2Off int, iStride2 int32, pSad []int32) {
	pSad[0] = WelsSampleSad8x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-int(iStride2), iStride2)
	pSad[1] = WelsSampleSad8x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+int(iStride2), iStride2)
	pSad[2] = WelsSampleSad8x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-1, iStride2)
	pSad[3] = WelsSampleSad8x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+1, iStride2)
}

func WelsSampleSadFour4x4_c(iSample1 []uint8, iSample1Off int, iStride1 int32, iSample2 []uint8, iSample2Off int, iStride2 int32, pSad []int32) {
	pSad[0] = WelsSampleSad4x4_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-int(iStride2), iStride2)
	pSad[1] = WelsSampleSad4x4_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+int(iStride2), iStride2)
	pSad[2] = WelsSampleSad4x4_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-1, iStride2)
	pSad[3] = WelsSampleSad4x4_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+1, iStride2)
}

func WelsSampleSadFour8x4_c(iSample1 []uint8, iSample1Off int, iStride1 int32, iSample2 []uint8, iSample2Off int, iStride2 int32, pSad []int32) {
	pSad[0] = WelsSampleSad8x4_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-int(iStride2), iStride2)
	pSad[1] = WelsSampleSad8x4_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+int(iStride2), iStride2)
	pSad[2] = WelsSampleSad8x4_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-1, iStride2)
	pSad[3] = WelsSampleSad8x4_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+1, iStride2)
}

func WelsSampleSadFour4x8_c(iSample1 []uint8, iSample1Off int, iStride1 int32, iSample2 []uint8, iSample2Off int, iStride2 int32, pSad []int32) {
	pSad[0] = WelsSampleSad4x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-int(iStride2), iStride2)
	pSad[1] = WelsSampleSad4x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+int(iStride2), iStride2)
	pSad[2] = WelsSampleSad4x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off-1, iStride2)
	pSad[3] = WelsSampleSad4x8_c(iSample1, iSample1Off, iStride1, iSample2, iSample2Off+1, iStride2)
}
