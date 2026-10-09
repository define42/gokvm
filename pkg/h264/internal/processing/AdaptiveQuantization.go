package processing

// Port of codec/processing/src/adaptivequantization/AdaptiveQuantization.cpp.

const (
	AVERAGE_TIME_MOTION              = 3000  //0.3046875 // 1/4 + 1/16 - 1/128 ~ 0.3 *AQ_TIME_INT_MULTIPLY
	AVERAGE_TIME_TEXTURE_QUALITYMODE = 10000 //0.5 // 1/2 *AQ_TIME_INT_MULTIPLY
	AVERAGE_TIME_TEXTURE_BITRATEMODE = 8750  //0.5 // 1/2 *AQ_TIME_INT_MULTIPLY
	MODEL_ALPHA                      = 9910  //1.5 //1.1102 *AQ_TIME_INT_MULTIPLY
	MODEL_TIME                       = 58185 //9.0 //5.9842 *AQ_TIME_INT_MULTIPLY
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////

func NewCAdaptiveQuantization(iCpuFlag int32) *CAdaptiveQuantization {
	a := &CAdaptiveQuantization{}
	a.initIStrategyBase()
	a.m_CPUFlag = iCpuFlag
	a.m_eMethod = METHOD_ADAPTIVE_QUANT
	a.m_pfVar = nil
	a.m_sAdaptiveQuantParam = SAdaptiveQuantizationParam{}
	a.WelsInitVarFunc(&a.m_pfVar, a.m_CPUFlag)
	return a
}

// samePlaneOrigin reports whether the C pointers p and (q + iQOff) are equal;
// p is a slice that starts at its pointer value.
func samePlaneOrigin(p []uint8, q []uint8, iQOff int) bool {
	if len(p) == 0 || iQOff < 0 || iQOff >= len(q) {
		return false
	}
	return &p[0] == &q[iQOff]
}

func (q *CAdaptiveQuantization) Process(iType int32, pSrcPixMap *SPixMap, pRefPixMap *SPixMap) EResult {
	eReturn := RET_INVALIDPARAM

	iWidth := pSrcPixMap.SRect.IRectWidth
	iHeight := pSrcPixMap.SRect.IRectHeight
	iMbWidth := iWidth >> 4
	iMbHeight := iHeight >> 4
	iMbTotalNum := iMbWidth * iMbHeight

	var pMotionTexture int // index into m_sAdaptiveQuantParam.PMotionTextureUnit
	var pVaaCalcResults *SVAACalcResult
	var iMotionTextureIndexToDeltaQp int32
	var iAverMotionTextureIndexToDeltaQp int32 // double to uint32
	var iAverageMotionIndex int64              // double to float
	var iAverageTextureIndex int64

	var iQStep int64
	var iLumaMotionDeltaQp int64
	var iLumaTextureDeltaQp int64

	var pRefFrameY, pCurFrameY int
	var iRefStride, iCurStride int32

	var pRefFrameTmp, pCurFrameTmp int
	var i, j int32

	pRefFrameYBuf := pRefPixMap.PPixel[0]
	pCurFrameYBuf := pSrcPixMap.PPixel[0]
	pRefFrameY = pRefPixMap.IPixelOff[0]
	pCurFrameY = pSrcPixMap.IPixelOff[0]

	iRefStride = pRefPixMap.IStride[0]
	iCurStride = pSrcPixMap.IStride[0]

	pMotionTextureUnit := q.m_sAdaptiveQuantParam.PMotionTextureUnit

	/////////////////////////////////////// motion //////////////////////////////////
	//  motion MB residual variance
	iAverageMotionIndex = 0
	iAverageTextureIndex = 0
	pMotionTexture = 0
	pVaaCalcResults = q.m_sAdaptiveQuantParam.PCalcResult

	if samePlaneOrigin(pVaaCalcResults.PRefY, pRefFrameYBuf, pRefFrameY) &&
		samePlaneOrigin(pVaaCalcResults.PCurY, pCurFrameYBuf, pCurFrameY) {
		iMbIndex := 0
		var iSumDiff, iSQDiff, uiSum, iSQSum int32
		for j = 0; j < iMbHeight; j++ {
			pRefFrameTmp = pRefFrameY
			pCurFrameTmp = pCurFrameY
			for i = 0; i < iMbWidth; i++ {
				iSumDiff = pVaaCalcResults.PSad8x8[iMbIndex][0]
				iSumDiff += pVaaCalcResults.PSad8x8[iMbIndex][1]
				iSumDiff += pVaaCalcResults.PSad8x8[iMbIndex][2]
				iSumDiff += pVaaCalcResults.PSad8x8[iMbIndex][3]

				iSQDiff = pVaaCalcResults.PSsd16x16[iMbIndex]
				uiSum = pVaaCalcResults.PSum16x16[iMbIndex]
				iSQSum = pVaaCalcResults.PSumOfSquare16x16[iMbIndex]

				iSumDiff = iSumDiff >> 8
				pMotionTextureUnit[pMotionTexture].UiMotionIndex = uint16((iSQDiff >> 8) - (iSumDiff * iSumDiff))

				uiSum = uiSum >> 8
				pMotionTextureUnit[pMotionTexture].UiTextureIndex = uint16((iSQSum >> 8) - (uiSum * uiSum))

				iAverageMotionIndex += int64(pMotionTextureUnit[pMotionTexture].UiMotionIndex)
				iAverageTextureIndex += int64(pMotionTextureUnit[pMotionTexture].UiTextureIndex)
				pMotionTexture++
				iMbIndex++
				pRefFrameTmp += MB_WIDTH_LUMA
				pCurFrameTmp += MB_WIDTH_LUMA
			}
			pRefFrameY += int(iRefStride << 4)
			pCurFrameY += int(iCurStride << 4)
		}
	} else {
		for j = 0; j < iMbHeight; j++ {
			pRefFrameTmp = pRefFrameY
			pCurFrameTmp = pCurFrameY
			for i = 0; i < iMbWidth; i++ {
				q.m_pfVar(pRefFrameYBuf, pRefFrameTmp, iRefStride, pCurFrameYBuf, pCurFrameTmp, iCurStride,
					&pMotionTextureUnit[pMotionTexture])
				iAverageMotionIndex += int64(pMotionTextureUnit[pMotionTexture].UiMotionIndex)
				iAverageTextureIndex += int64(pMotionTextureUnit[pMotionTexture].UiTextureIndex)
				pMotionTexture++
				pRefFrameTmp += MB_WIDTH_LUMA
				pCurFrameTmp += MB_WIDTH_LUMA

			}
			pRefFrameY += int(iRefStride << 4)
			pCurFrameY += int(iCurStride << 4)
		}
	}
	iAverageMotionIndex = WELS_DIV_ROUND64(iAverageMotionIndex*AQ_INT_MULTIPLY, int64(iMbTotalNum))
	iAverageTextureIndex = WELS_DIV_ROUND64(iAverageTextureIndex*AQ_INT_MULTIPLY, int64(iMbTotalNum))
	if (iAverageMotionIndex <= AQ_PESN) && (iAverageMotionIndex >= -AQ_PESN) {
		iAverageMotionIndex = AQ_INT_MULTIPLY
	}
	if (iAverageTextureIndex <= AQ_PESN) && (iAverageTextureIndex >= -AQ_PESN) {
		iAverageTextureIndex = AQ_INT_MULTIPLY
	}
	//  motion mb residual map to QP
	//  texture mb original map to QP
	iAverMotionTextureIndexToDeltaQp = 0
	iAverageMotionIndex = WELS_DIV_ROUND64(AVERAGE_TIME_MOTION*iAverageMotionIndex, AQ_TIME_INT_MULTIPLY)

	if q.m_sAdaptiveQuantParam.IAdaptiveQuantMode == AQ_QUALITY_MODE {
		iAverageTextureIndex = WELS_DIV_ROUND64(AVERAGE_TIME_TEXTURE_QUALITYMODE*iAverageTextureIndex, AQ_TIME_INT_MULTIPLY)
	} else {
		iAverageTextureIndex = WELS_DIV_ROUND64(AVERAGE_TIME_TEXTURE_BITRATEMODE*iAverageTextureIndex, AQ_TIME_INT_MULTIPLY)
	}

	iAQ_EPSN := -(int64(AQ_PESN) * AQ_TIME_INT_MULTIPLY * AQ_QSTEP_INT_MULTIPLY / AQ_INT_MULTIPLY)
	pMotionTexture = 0
	for j = 0; j < iMbHeight; j++ {
		for i = 0; i < iMbWidth; i++ {
			pMT := &pMotionTextureUnit[pMotionTexture]
			a := WELS_DIV_ROUND64(int64(pMT.UiTextureIndex)*AQ_INT_MULTIPLY*AQ_TIME_INT_MULTIPLY,
				iAverageTextureIndex)
			iQStep = WELS_DIV_ROUND64((a-AQ_TIME_INT_MULTIPLY)*AQ_QSTEP_INT_MULTIPLY, (a + MODEL_ALPHA))
			iLumaTextureDeltaQp = MODEL_TIME * iQStep // range +- 6

			iMotionTextureIndexToDeltaQp = int32(iLumaTextureDeltaQp / (AQ_TIME_INT_MULTIPLY))

			a = WELS_DIV_ROUND64(int64(pMT.UiMotionIndex)*AQ_INT_MULTIPLY*AQ_TIME_INT_MULTIPLY,
				iAverageMotionIndex)
			iQStep = WELS_DIV_ROUND64((a-AQ_TIME_INT_MULTIPLY)*AQ_QSTEP_INT_MULTIPLY, (a + MODEL_ALPHA))
			iLumaMotionDeltaQp = MODEL_TIME * iQStep // range +- 6

			if (q.m_sAdaptiveQuantParam.IAdaptiveQuantMode == AQ_QUALITY_MODE && iLumaMotionDeltaQp < iAQ_EPSN) ||
				(q.m_sAdaptiveQuantParam.IAdaptiveQuantMode == AQ_BITRATE_MODE) {
				iMotionTextureIndexToDeltaQp += int32(iLumaMotionDeltaQp / (AQ_TIME_INT_MULTIPLY))
			}

			q.m_sAdaptiveQuantParam.PMotionTextureIndexToDeltaQp[j*iMbWidth+i] = int8(iMotionTextureIndexToDeltaQp /
				AQ_QSTEP_INT_MULTIPLY)
			iAverMotionTextureIndexToDeltaQp += iMotionTextureIndexToDeltaQp
			pMotionTexture++
		}
	}

	q.m_sAdaptiveQuantParam.IAverMotionTextureIndexToDeltaQp = iAverMotionTextureIndexToDeltaQp / iMbTotalNum

	eReturn = RET_SUCCESS

	return eReturn
}

// Set takes a *SAdaptiveQuantizationParam.
func (q *CAdaptiveQuantization) Set(iType int32, pParam any) EResult {
	p, ok := pParam.(*SAdaptiveQuantizationParam)
	if !ok || p == nil {
		return RET_INVALIDPARAM
	}

	q.m_sAdaptiveQuantParam = *p

	return RET_SUCCESS
}

// Get takes a *SAdaptiveQuantizationParam; only IAverMotionTextureIndexToDeltaQp is written.
func (q *CAdaptiveQuantization) Get(iType int32, pParam any) EResult {
	sAdaptiveQuantParam, ok := pParam.(*SAdaptiveQuantizationParam)
	if !ok || sAdaptiveQuantParam == nil {
		return RET_INVALIDPARAM
	}

	sAdaptiveQuantParam.IAverMotionTextureIndexToDeltaQp = q.m_sAdaptiveQuantParam.IAverMotionTextureIndexToDeltaQp

	return RET_SUCCESS
}

///////////////////////////////////////////////////////////////////////////////////////////////

func (q *CAdaptiveQuantization) WelsInitVarFunc(pfVar *PVarFunc, iCpuFlag int32) {
	*pfVar = SampleVariance16x16_c
}

func SampleVariance16x16_c(pRefY []uint8, iRefYOff int, iRefStride int32, pSrcY []uint8, iSrcYOff int, iSrcStride int32,
	pMotionTexture *SMotionTextureUnit) {
	var uiCurSquare, uiSquare uint32
	var uiCurSum, uiSum uint16

	for y := 0; y < MB_WIDTH_LUMA; y++ {
		for x := 0; x < MB_WIDTH_LUMA; x++ {
			s := uint32(pSrcY[iSrcYOff+x])
			uiDiff := uint32(WELS_ABS(int32(pRefY[iRefYOff+x]) - int32(s)))
			uiSum += uint16(uiDiff)
			uiSquare += uiDiff * uiDiff

			uiCurSum += uint16(s)
			uiCurSquare += s * s
		}
		iRefYOff += int(iRefStride)
		iSrcYOff += int(iSrcStride)
	}

	uiSum = uiSum >> 8
	pMotionTexture.UiMotionIndex = uint16((uiSquare >> 8) - (uint32(uiSum) * uint32(uiSum)))

	uiCurSum = uiCurSum >> 8
	pMotionTexture.UiTextureIndex = uint16((uiCurSquare >> 8) - (uint32(uiCurSum) * uint32(uiCurSum)))
}
