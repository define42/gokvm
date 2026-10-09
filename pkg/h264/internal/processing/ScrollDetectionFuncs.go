package processing

// Port of codec/processing/src/scrolldetection/ScrollDetectionFuncs.cpp.

import "bytes"

func CheckLine(pData []uint8, iDataOff int, iWidth int32) int32 {
	var iQualified int32
	var iColorMap [8]int32
	var iChangedTimes int32
	var iColorCounts int32

	RECORD_COLOR(pData[iDataOff], &iColorMap)

	for i := int32(1); i < iWidth; i++ {
		RECORD_COLOR(pData[iDataOff+int(i)], &iColorMap)
		if pData[iDataOff+int(i)] != pData[iDataOff+int(i)-1] {
			iChangedTimes++
		}
	}
	for i := 0; i < 8; i++ {
		for j := 0; j < 32; j++ {
			iColorCounts += ((iColorMap[i] >> j) & 1)
		}
	}

	switch iColorCounts {
	case 1:
		iQualified = 0
	case 2, 3:
		if iChangedTimes > 3 {
			iQualified = 1
		} else {
			iQualified = 0
		}
	default:
		iQualified = 1
	}
	return iQualified
}

func SelectTestLine(pY []uint8, iYOff int, iWidth int32, iHeight int32, iPicHeight int32,
	iStride int32, iOffsetX int32, iOffsetY int32) int32 {
	kiHalfHeight := iHeight >> 1
	kiMidPos := iOffsetY + kiHalfHeight
	TestPos := kiMidPos
	var iOffsetAbs int32
	var pTmp int

	for iOffsetAbs = 0; iOffsetAbs < kiHalfHeight; iOffsetAbs++ {
		TestPos = kiMidPos + iOffsetAbs
		if TestPos < iPicHeight {
			pTmp = iYOff + int(TestPos*iStride+iOffsetX)
			if CheckLine(pY, pTmp, iWidth) != 0 {
				break
			}
		}
		TestPos = kiMidPos - iOffsetAbs
		if TestPos >= 0 {
			pTmp = iYOff + int(TestPos*iStride+iOffsetX)
			if CheckLine(pY, pTmp, iWidth) != 0 {
				break
			}
		}
	}
	if iOffsetAbs == kiHalfHeight {
		TestPos = -1
	}
	return TestPos
}

/*
 * compare pixel line between previous and current one
 * return: 0 for totally equal, otherwise 1
 */
func CompareLine(pYSrc []uint8, iSrcOff int, pYRef []uint8, iRefOff int, kiWidth int32) int32 {
	var iCmp int32 = 1

	if !bytes.Equal(pYSrc[iSrcOff:iSrcOff+4], pYRef[iRefOff:iRefOff+4]) {
		return 1
	}
	if !bytes.Equal(pYSrc[iSrcOff+4:iSrcOff+8], pYRef[iRefOff+4:iRefOff+8]) {
		return 1
	}
	if !bytes.Equal(pYSrc[iSrcOff+8:iSrcOff+12], pYRef[iRefOff+8:iRefOff+12]) {
		return 1
	}
	if kiWidth > 12 {
		n := int(kiWidth - 12)
		iCmp = int32(bytes.Compare(pYSrc[iSrcOff+12:iSrcOff+12+n], pYRef[iRefOff+12:iRefOff+12+n]))
	}
	return iCmp
}

func ScrollDetectionCore(pSrcPixMap *SPixMap, pRefPixMap *SPixMap, iWidth int32, iHeight int32,
	iOffsetX int32, iOffsetY int32, sScrollDetectionParam *SScrollDetectionParam) {
	bScrollDetected := false
	var pYLine int
	var pYTmp int
	var iTestPos, iSearchPos, iOffsetAbs, iMaxAbs int32
	iPicHeight := pRefPixMap.SRect.IRectHeight
	iMinHeight := WELS_MAX(iOffsetY, 0)
	iMaxHeight := WELS_MIN(iOffsetY+iHeight-1, iPicHeight-1) //offset_y + height - 1;//
	var pYRefBuf, pYSrcBuf []uint8
	var pYRef, pYSrc int
	var iYStride int32

	pYRefBuf, pYRef = pRefPixMap.PPixel[0], pRefPixMap.IPixelOff[0]
	pYSrcBuf, pYSrc = pSrcPixMap.PPixel[0], pSrcPixMap.IPixelOff[0]
	iYStride = pRefPixMap.IStride[0]

	iTestPos = SelectTestLine(pYSrcBuf, pYSrc, iWidth, iHeight, iPicHeight, iYStride, iOffsetX, iOffsetY)

	if iTestPos == -1 {
		sScrollDetectionParam.BScrollDetectFlag = false
		return
	}
	pYLine = pYSrc + int(iYStride*iTestPos+iOffsetX)
	iMaxAbs = WELS_MIN(WELS_MAX(iTestPos-iMinHeight-1, iMaxHeight-iTestPos), MAX_SCROLL_MV_Y)
	iSearchPos = iTestPos
	for iOffsetAbs = 0; iOffsetAbs <= iMaxAbs; iOffsetAbs++ {
		iSearchPos = iTestPos + iOffsetAbs
		if iSearchPos <= iMaxHeight {
			pYTmp = pYRef + int(iSearchPos*iYStride+iOffsetX)
			if CompareLine(pYSrcBuf, pYLine, pYRefBuf, pYTmp, iWidth) == 0 {
				var pYUpper, pYLineUpper int
				var iCheckedLines int32
				iLowOffset := WELS_MIN(iMaxHeight-iSearchPos, CHECK_OFFSET)
				var i int32

				iCheckedLines = WELS_MIN(iTestPos-iMinHeight+iLowOffset, 2*CHECK_OFFSET)
				pYUpper = pYTmp - int((iCheckedLines-iLowOffset)*iYStride)
				pYLineUpper = pYLine - int((iCheckedLines-iLowOffset)*iYStride)

				for i = 0; i < iCheckedLines; i++ {
					if CompareLine(pYSrcBuf, pYLineUpper, pYRefBuf, pYUpper, iWidth) != 0 {
						break
					}
					pYUpper += int(iYStride)
					pYLineUpper += int(iYStride)
				}
				if i == iCheckedLines {
					bScrollDetected = true
					break
				}
			}
		}

		iSearchPos = iTestPos - iOffsetAbs - 1
		if iSearchPos >= iMinHeight {
			pYTmp = pYRef + int(iSearchPos*iYStride+iOffsetX)
			if CompareLine(pYSrcBuf, pYLine, pYRefBuf, pYTmp, iWidth) == 0 {
				var pYUpper, pYLineUpper int
				var iCheckedLines int32
				iUpOffset := WELS_MIN(iSearchPos-iMinHeight, CHECK_OFFSET)
				var i int32

				pYUpper = pYTmp - int(iUpOffset*iYStride)
				pYLineUpper = pYLine - int(iUpOffset*iYStride)
				iCheckedLines = WELS_MIN(iMaxHeight-iTestPos+iUpOffset, 2*CHECK_OFFSET)

				for i = 0; i < iCheckedLines; i++ {
					if CompareLine(pYSrcBuf, pYLineUpper, pYRefBuf, pYUpper, iWidth) != 0 {
						break
					}
					pYUpper += int(iYStride)
					pYLineUpper += int(iYStride)
				}
				if i == iCheckedLines {
					bScrollDetected = true
					break
				}
			}
		}
	}

	if !bScrollDetected {
		sScrollDetectionParam.BScrollDetectFlag = false
	} else {
		sScrollDetectionParam.BScrollDetectFlag = true
		sScrollDetectionParam.IScrollMvY = iSearchPos - iTestPos // pre_pos - cur_pos, change to mv
		sScrollDetectionParam.IScrollMvX = 0
	}
}
