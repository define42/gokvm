package processing

// Port of codec/processing/src/denoise/denoise_filter.cpp.

func BilateralLumaFilter8_c(pSample []uint8, iSampleOff int, iStride int32) {
	var nSum, nTotWeight int32
	var iCenterSample int32
	var pCurLine int
	var iCurSample, iCurWeight, iGreyDiff int32
	var aSample [8]uint8

	for i := 0; i < 8; i++ {
		nSum = 0
		nTotWeight = 0
		iCenterSample = int32(pSample[iSampleOff])
		pCurLine = iSampleOff - int(iStride) - DENOISE_GRAY_RADIUS
		for y := 0; y < 3; y++ {
			for x := 0; x < 3; x++ {
				if x == 1 && y == 1 {
					continue // except center point
				}
				iCurSample = int32(pSample[pCurLine+x])
				iCurWeight = WELS_ABS(iCurSample - iCenterSample)
				iGreyDiff = 32 - iCurWeight
				if iGreyDiff < 0 {
					continue
				} else {
					iCurWeight = (iGreyDiff * iGreyDiff) >> 5
				}
				nSum += iCurSample * iCurWeight
				nTotWeight += iCurWeight
			}
			pCurLine += int(iStride)
		}
		nTotWeight = 256 - nTotWeight
		nSum += iCenterSample * nTotWeight
		aSample[i] = uint8(nSum >> 8)
		iSampleOff++
	}
	copy(pSample[iSampleOff-8:iSampleOff], aSample[:])
}

/*
**************************************************************************
5x5 filter:
1   1   2   1   1
1   2   4   2   1
2   4   20  4   2
1   2   4   2   1
1   1   2   1   1
**************************************************************************
*/
func SUM_LINE1(p []uint8, o int) int32 {
	return int32(p[o]) + int32(p[o+1]) + (int32(p[o+2]) << 1) + int32(p[o+3]) + int32(p[o+4])
}
func SUM_LINE2(p []uint8, o int) int32 {
	return int32(p[o]) + (int32(p[o+1]) << 1) + (int32(p[o+2]) << 2) + (int32(p[o+3]) << 1) + int32(p[o+4])
}
func SUM_LINE3(p []uint8, o int) int32 {
	return (int32(p[o]) << 1) + (int32(p[o+1]) << 2) + (int32(p[o+2]) * 20) + (int32(p[o+3]) << 2) + (int32(p[o+4]) << 1)
}

func WaverageChromaFilter8_c(pSample []uint8, iSampleOff int, iStride int32) {
	var sum int32
	pStartPixels := iSampleOff - UV_WINDOWS_RADIUS*int(iStride) - UV_WINDOWS_RADIUS
	pCurLine1 := pStartPixels
	pCurLine2 := pCurLine1 + int(iStride)
	pCurLine3 := pCurLine2 + int(iStride)
	pCurLine4 := pCurLine3 + int(iStride)
	pCurLine5 := pCurLine4 + int(iStride)
	var aSample [8]uint8

	for i := 0; i < 8; i++ {
		sum = SUM_LINE1(pSample, pCurLine1+i) + SUM_LINE2(pSample, pCurLine2+i) + SUM_LINE3(pSample, pCurLine3+i) +
			SUM_LINE2(pSample, pCurLine4+i) + SUM_LINE1(pSample, pCurLine5+i)
		aSample[i] = uint8(sum >> 6)
		iSampleOff++
	}
	copy(pSample[iSampleOff-8:iSampleOff], aSample[:])
}

/*
**************************************************************************
edge of y/uv use a 3x3 Gauss filter, radius = 1:
1   2   1
2   4   2
1   2   1
**************************************************************************
*/
func Gauss3x3Filter(pSrc []uint8, iSrcOff int, iStride int32) {
	var nSum int32
	pCurLine1 := iSrcOff - int(iStride) - 1
	pCurLine2 := pCurLine1 + int(iStride)
	pCurLine3 := pCurLine2 + int(iStride)

	nSum = int32(pSrc[pCurLine1]) + (int32(pSrc[pCurLine1+1]) << 1) + int32(pSrc[pCurLine1+2]) +
		(int32(pSrc[pCurLine2]) << 1) + (int32(pSrc[pCurLine2+1]) << 2) + (int32(pSrc[pCurLine2+2]) << 1) +
		int32(pSrc[pCurLine3]) + (int32(pSrc[pCurLine3+1]) << 1) + int32(pSrc[pCurLine3+2])
	pSrc[iSrcOff] = uint8(nSum >> 4)
}
