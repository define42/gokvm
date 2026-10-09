package processing

// Port of codec/processing/src/complexityanalysis/ComplexityAnalysis.cpp.

///////////////////////////////////////////////////////////////////////////////////////////////////////////////

func NewCComplexityAnalysis(iCpuFlag int32) *CComplexityAnalysis {
	c := &CComplexityAnalysis{}
	c.initIStrategyBase()
	c.m_eMethod = METHOD_COMPLEXITY_ANALYSIS
	c.m_pfGomSad = nil
	c.m_sComplexityAnalysisParam = SComplexityAnalysisParam{}
	return c
}

func (c *CComplexityAnalysis) Process(iType int32, pSrcPixMap *SPixMap, pRefPixMap *SPixMap) EResult {
	eReturn := RET_SUCCESS

	switch c.m_sComplexityAnalysisParam.IComplexityAnalysisMode {
	case FRAME_SAD:
		c.AnalyzeFrameComplexityViaSad(pSrcPixMap, pRefPixMap)
	case GOM_SAD:
		c.AnalyzeGomComplexityViaSad(pSrcPixMap, pRefPixMap)
	case GOM_VAR:
		c.AnalyzeGomComplexityViaVar(pSrcPixMap, pRefPixMap)
	default:
		eReturn = RET_INVALIDPARAM
	}

	return eReturn
}

// Set takes a *SComplexityAnalysisParam.
func (c *CComplexityAnalysis) Set(iType int32, pParam any) EResult {
	p, ok := pParam.(*SComplexityAnalysisParam)
	if !ok || p == nil {
		return RET_INVALIDPARAM
	}

	c.m_sComplexityAnalysisParam = *p

	return RET_SUCCESS
}

// Get takes a *SComplexityAnalysisParam; only IFrameComplexity is written.
func (c *CComplexityAnalysis) Get(iType int32, pParam any) EResult {
	sComplexityAnalysisParam, ok := pParam.(*SComplexityAnalysisParam)
	if !ok || sComplexityAnalysisParam == nil {
		return RET_INVALIDPARAM
	}

	sComplexityAnalysisParam.IFrameComplexity = c.m_sComplexityAnalysisParam.IFrameComplexity

	return RET_SUCCESS
}

// /////////////////////////////////////////////////////////////////////////////////////////////
func (c *CComplexityAnalysis) AnalyzeFrameComplexityViaSad(pSrcPixMap *SPixMap, pRefPixMap *SPixMap) {
	pVaaCalcResults := c.m_sComplexityAnalysisParam.PCalcResult

	c.m_sComplexityAnalysisParam.IFrameComplexity = int64(pVaaCalcResults.IFrameSad)

	if c.m_sComplexityAnalysisParam.ICalcBgd != 0 { //BGD control
		c.m_sComplexityAnalysisParam.IFrameComplexity = int64(c.GetFrameSadExcludeBackground(pSrcPixMap, pRefPixMap))
	}
}

func (c *CComplexityAnalysis) GetFrameSadExcludeBackground(pSrcPixMap *SPixMap, pRefPixMap *SPixMap) int32 {
	iWidth := pSrcPixMap.SRect.IRectWidth
	iHeight := pSrcPixMap.SRect.IRectHeight
	iMbWidth := iWidth >> 4
	iMbHeight := iHeight >> 4
	iMbNum := iMbWidth * iMbHeight

	iMbNumInGom := c.m_sComplexityAnalysisParam.IMbNumInGom
	iGomMbNum := (iMbNum + iMbNumInGom - 1) / iMbNumInGom
	var iGomMbStartIndex, iGomMbEndIndex int32

	pBackgroundMbFlag := c.m_sComplexityAnalysisParam.PBackgroundMbFlag
	uiRefMbType := c.m_sComplexityAnalysisParam.UiRefMbType
	pVaaCalcResults := c.m_sComplexityAnalysisParam.PCalcResult
	pGomForegroundBlockNum := c.m_sComplexityAnalysisParam.PGomForegroundBlockNum

	var uiFrameSad uint32
	for j := int32(0); j < iGomMbNum; j++ {
		iGomMbStartIndex = j * iMbNumInGom
		iGomMbEndIndex = WELS_MIN((j+1)*iMbNumInGom, iMbNum)

		for i := iGomMbStartIndex; i < iGomMbEndIndex; i++ {
			if pBackgroundMbFlag[i] == 0 || IS_INTRA(uiRefMbType[i]) {
				pGomForegroundBlockNum[j]++
				uiFrameSad += uint32(pVaaCalcResults.PSad8x8[i][0])
				uiFrameSad += uint32(pVaaCalcResults.PSad8x8[i][1])
				uiFrameSad += uint32(pVaaCalcResults.PSad8x8[i][2])
				uiFrameSad += uint32(pVaaCalcResults.PSad8x8[i][3])
			}
		}
	}

	return int32(uiFrameSad)
}

func InitGomSadFunc(pfGomSad *PGOMSadFunc, iCalcBgd uint8) {
	*pfGomSad = GomSampleSad

	if iCalcBgd != 0 {
		*pfGomSad = GomSampleSadExceptBackground
	}
}

func GomSampleSad(pGomSad *uint32, pGomForegroundBlockNum *int32, pSad8x8 *[4]int32, pBackgroundMbFlag uint8) {
	(*pGomForegroundBlockNum)++
	*pGomSad += uint32(pSad8x8[0])
	*pGomSad += uint32(pSad8x8[1])
	*pGomSad += uint32(pSad8x8[2])
	*pGomSad += uint32(pSad8x8[3])
}

func GomSampleSadExceptBackground(pGomSad *uint32, pGomForegroundBlockNum *int32, pSad8x8 *[4]int32,
	pBackgroundMbFlag uint8) {
	if pBackgroundMbFlag == 0 {
		(*pGomForegroundBlockNum)++
		*pGomSad += uint32(pSad8x8[0])
		*pGomSad += uint32(pSad8x8[1])
		*pGomSad += uint32(pSad8x8[2])
		*pGomSad += uint32(pSad8x8[3])
	}
}

func (c *CComplexityAnalysis) AnalyzeGomComplexityViaSad(pSrcPixMap *SPixMap, pRefPixMap *SPixMap) {
	iWidth := pSrcPixMap.SRect.IRectWidth
	iHeight := pSrcPixMap.SRect.IRectHeight
	iMbWidth := iWidth >> 4
	iMbHeight := iHeight >> 4
	iMbNum := iMbWidth * iMbHeight

	iMbNumInGom := c.m_sComplexityAnalysisParam.IMbNumInGom
	iGomMbNum := (iMbNum + iMbNumInGom - 1) / iMbNumInGom

	var iGomMbStartIndex, iGomMbEndIndex, iGomMbRowNum int32
	var iMbStartIndex, iMbEndIndex int32

	pBackgroundMbFlag := c.m_sComplexityAnalysisParam.PBackgroundMbFlag
	uiRefMbType := c.m_sComplexityAnalysisParam.UiRefMbType
	pVaaCalcResults := c.m_sComplexityAnalysisParam.PCalcResult
	pGomForegroundBlockNum := c.m_sComplexityAnalysisParam.PGomForegroundBlockNum
	pGomComplexity := c.m_sComplexityAnalysisParam.PGomComplexity

	var uiGomSad, uiFrameSad uint32
	InitGomSadFunc(&c.m_pfGomSad, uint8(c.m_sComplexityAnalysisParam.ICalcBgd))

	for j := int32(0); j < iGomMbNum; j++ {
		uiGomSad = 0

		iGomMbStartIndex = j * iMbNumInGom
		iGomMbEndIndex = WELS_MIN((j+1)*iMbNumInGom, iMbNum)
		iGomMbRowNum = (iGomMbEndIndex+iMbWidth-1)/iMbWidth - iGomMbStartIndex/iMbWidth

		iMbStartIndex = iGomMbStartIndex
		iMbEndIndex = WELS_MIN((iMbStartIndex/iMbWidth+1)*iMbWidth, iGomMbEndIndex)

		for {
			for i := iMbStartIndex; i < iMbEndIndex; i++ {
				var uiFlag uint8
				if pBackgroundMbFlag[i] != 0 && !IS_INTRA(uiRefMbType[i]) {
					uiFlag = 1
				}
				c.m_pfGomSad(&uiGomSad, &pGomForegroundBlockNum[j], &pVaaCalcResults.PSad8x8[i], uiFlag)
			}

			iMbStartIndex = iMbEndIndex
			iMbEndIndex = WELS_MIN(iMbEndIndex+iMbWidth, iGomMbEndIndex)

			iGomMbRowNum--
			if iGomMbRowNum == 0 {
				break
			}
		}
		pGomComplexity[j] = int32(uiGomSad)
		uiFrameSad += uint32(pGomComplexity[j])
	}
	c.m_sComplexityAnalysisParam.IFrameComplexity = int64(uiFrameSad)
}

func (c *CComplexityAnalysis) AnalyzeGomComplexityViaVar(pSrcPixMap *SPixMap, pRefPixMap *SPixMap) {
	iWidth := pSrcPixMap.SRect.IRectWidth
	iHeight := pSrcPixMap.SRect.IRectHeight
	iMbWidth := iWidth >> 4
	iMbHeight := iHeight >> 4
	iMbNum := iMbWidth * iMbHeight

	iMbNumInGom := c.m_sComplexityAnalysisParam.IMbNumInGom
	iGomMbNum := (iMbNum + iMbNumInGom - 1) / iMbNumInGom
	var iGomSampleNum int32

	var iGomMbStartIndex, iGomMbEndIndex, iGomMbRowNum int32
	var iMbStartIndex, iMbEndIndex int32

	pVaaCalcResults := c.m_sComplexityAnalysisParam.PCalcResult
	pGomComplexity := c.m_sComplexityAnalysisParam.PGomComplexity
	var uiFrameSad uint32

	var uiSampleSum, uiSquareSum uint32

	for j := int32(0); j < iGomMbNum; j++ {
		uiSampleSum = 0
		uiSquareSum = 0

		iGomMbStartIndex = j * iMbNumInGom
		iGomMbEndIndex = WELS_MIN((j+1)*iMbNumInGom, iMbNum)
		iGomMbRowNum = (iGomMbEndIndex+iMbWidth-1)/iMbWidth - iGomMbStartIndex/iMbWidth

		iMbStartIndex = iGomMbStartIndex
		iMbEndIndex = WELS_MIN((iMbStartIndex/iMbWidth+1)*iMbWidth, iGomMbEndIndex)

		iGomSampleNum = (iMbEndIndex - iMbStartIndex) * MB_WIDTH_LUMA * MB_WIDTH_LUMA

		for {
			for i := iMbStartIndex; i < iMbEndIndex; i++ {
				uiSampleSum += uint32(pVaaCalcResults.PSum16x16[i])
				uiSquareSum += uint32(pVaaCalcResults.PSumOfSquare16x16[i])
			}

			iMbStartIndex = iMbEndIndex
			iMbEndIndex = WELS_MIN(iMbEndIndex+iMbWidth, iGomMbEndIndex)

			iGomMbRowNum--
			if iGomMbRowNum == 0 {
				break
			}
		}

		pGomComplexity[j] = int32(uiSquareSum - (uiSampleSum * uiSampleSum / uint32(iGomSampleNum)))
		uiFrameSad += uint32(pGomComplexity[j])
	}
	c.m_sComplexityAnalysisParam.IFrameComplexity = int64(uiFrameSad)
}

func NewCComplexityAnalysisScreen(iCpuFlag int32) *CComplexityAnalysisScreen {
	c := &CComplexityAnalysisScreen{}
	c.initIStrategyBase()
	c.m_eMethod = METHOD_COMPLEXITY_ANALYSIS_SCREEN
	c.m_ComplexityAnalysisParam = SComplexityAnalysisScreenParam{}

	c.m_pSadFunc = WelsSampleSad16x16_c
	c.m_pIntraFunc[0] = WelsI16x16LumaPredV_c
	c.m_pIntraFunc[1] = WelsI16x16LumaPredH_c
	return c
}

func (c *CComplexityAnalysisScreen) Process(nType int32, pSrc *SPixMap, pRef *SPixMap) EResult {
	bScrollFlag := c.m_ComplexityAnalysisParam.SScrollResult.BScrollDetectFlag
	iIdrFlag := c.m_ComplexityAnalysisParam.IIdrFlag
	iScrollMvX := c.m_ComplexityAnalysisParam.SScrollResult.IScrollMvX
	iScrollMvY := c.m_ComplexityAnalysisParam.SScrollResult.IScrollMvY

	if c.m_ComplexityAnalysisParam.IMbRowInGom <= 0 {
		return RET_INVALIDPARAM
	}
	if iIdrFlag == 0 && pRef == nil {
		return RET_INVALIDPARAM
	}

	if iIdrFlag != 0 || pRef == nil {
		c.GomComplexityAnalysisIntra(pSrc)
	} else if !bScrollFlag || ((iScrollMvX == 0) && (iScrollMvY == 0)) {
		c.GomComplexityAnalysisInter(pSrc, pRef, false)
	} else {
		c.GomComplexityAnalysisInter(pSrc, pRef, true)
	}

	return RET_SUCCESS
}

// Set takes a *SComplexityAnalysisScreenParam.
func (c *CComplexityAnalysisScreen) Set(nType int32, pParam any) EResult {
	p, ok := pParam.(*SComplexityAnalysisScreenParam)
	if !ok || p == nil {
		return RET_INVALIDPARAM
	}

	c.m_ComplexityAnalysisParam = *p

	return RET_SUCCESS
}

// Get takes a *SComplexityAnalysisScreenParam.
func (c *CComplexityAnalysisScreen) Get(nType int32, pParam any) EResult {
	p, ok := pParam.(*SComplexityAnalysisScreenParam)
	if !ok || p == nil {
		return RET_INVALIDPARAM
	}

	*p = c.m_ComplexityAnalysisParam

	return RET_SUCCESS
}

func (c *CComplexityAnalysisScreen) GomComplexityAnalysisIntra(pSrc *SPixMap) {
	iWidth := pSrc.SRect.IRectWidth
	iHeight := pSrc.SRect.IRectHeight
	iBlockWidth := iWidth >> 4
	iBlockHeight := iHeight >> 4

	var iBlockSadH, iBlockSadV, iGomSad int32
	var iIdx int32

	var pPtrY int
	var iStrideY int32
	var iRowStrideY int32

	var pTmpCur int

	var iMemPredMb [256]uint8

	pPtrYBuf := pSrc.PPixel[0]
	pPtrY = pSrc.IPixelOff[0]

	iStrideY = pSrc.IStride[0]
	iRowStrideY = iStrideY << 4

	c.m_ComplexityAnalysisParam.IFrameComplexity = 0

	for j := int32(0); j < iBlockHeight; j++ {
		pTmpCur = pPtrY

		for i := int32(0); i < iBlockWidth; i++ {
			iBlockSadH = 0x7fffffff // INT_MAX
			iBlockSadV = 0x7fffffff
			if j > 0 {
				c.m_pIntraFunc[0](iMemPredMb[:], pPtrYBuf, pTmpCur, iStrideY)
				iBlockSadH = c.m_pSadFunc(pPtrYBuf, pTmpCur, iStrideY, iMemPredMb[:], 0, 16)
			}
			if i > 0 {
				c.m_pIntraFunc[1](iMemPredMb[:], pPtrYBuf, pTmpCur, iStrideY)
				iBlockSadV = c.m_pSadFunc(pPtrYBuf, pTmpCur, iStrideY, iMemPredMb[:], 0, 16)
			}
			if i != 0 || j != 0 {
				iGomSad += WELS_MIN(iBlockSadH, iBlockSadV)
			}

			pTmpCur += 16

			if i == iBlockWidth-1 && ((j+1)%c.m_ComplexityAnalysisParam.IMbRowInGom == 0 || j == iBlockHeight-1) {
				c.m_ComplexityAnalysisParam.PGomComplexity[iIdx] = iGomSad
				c.m_ComplexityAnalysisParam.IFrameComplexity += int64(iGomSad)
				iIdx++
				iGomSad = 0
			}
		}

		pPtrY += int(iRowStrideY)
	}
	c.m_ComplexityAnalysisParam.IGomNumInFrame = iIdx
}

// GomComplexityAnalysisInter: note that with a scroll motion vector the
// reference is read at pTmpRef - iScrollMvY * iStrideX + iScrollMvX, which can
// lie above the plane origin (the bounds check uses +iScrollMvY). That is why
// SPixMap carries IPixelOff.
func (c *CComplexityAnalysisScreen) GomComplexityAnalysisInter(pSrc *SPixMap, pRef *SPixMap, bScrollFlag bool) {
	iWidth := pSrc.SRect.IRectWidth
	iHeight := pSrc.SRect.IRectHeight
	iBlockWidth := iWidth >> 4
	iBlockHeight := iHeight >> 4

	var iInterSad, iScrollSad, iBlockSadH, iBlockSadV, iGomSad int32
	var iIdx int32

	iScrollMvX := c.m_ComplexityAnalysisParam.SScrollResult.IScrollMvX
	iScrollMvY := c.m_ComplexityAnalysisParam.SScrollResult.IScrollMvY

	var pPtrX, pPtrY int
	var iStrideX, iStrideY int32
	var iRowStrideX, iRowStrideY int32

	var pTmpRef, pTmpCur, pTmpRefScroll int

	var iMemPredMb [256]uint8

	pPtrXBuf := pRef.PPixel[0]
	pPtrX = pRef.IPixelOff[0]
	pPtrYBuf := pSrc.PPixel[0]
	pPtrY = pSrc.IPixelOff[0]

	iStrideX = pRef.IStride[0]
	iStrideY = pSrc.IStride[0]

	iRowStrideX = pRef.IStride[0] << 4
	iRowStrideY = pSrc.IStride[0] << 4

	c.m_ComplexityAnalysisParam.IFrameComplexity = 0

	for j := int32(0); j < iBlockHeight; j++ {
		pTmpRef = pPtrX
		pTmpCur = pPtrY

		for i := int32(0); i < iBlockWidth; i++ {
			iBlockPointX := i << 4
			iBlockPointY := j << 4

			iInterSad = c.m_pSadFunc(pPtrYBuf, pTmpCur, iStrideY, pPtrXBuf, pTmpRef, iStrideX)
			if bScrollFlag {
				if (iInterSad != 0) &&
					(iBlockPointX+iScrollMvX >= 0) && (iBlockPointX+iScrollMvX <= iWidth-8) &&
					(iBlockPointY+iScrollMvY >= 0) && (iBlockPointY+iScrollMvY <= iHeight-8) {
					pTmpRefScroll = pTmpRef - int(iScrollMvY*iStrideX) + int(iScrollMvX)
					iScrollSad = c.m_pSadFunc(pPtrYBuf, pTmpCur, iStrideY, pPtrXBuf, pTmpRefScroll, iStrideX)

					if iScrollSad < iInterSad {
						iInterSad = iScrollSad
					}
				}

			}

			iBlockSadH = 0x7fffffff // INT_MAX
			iBlockSadV = 0x7fffffff

			if j > 0 {
				c.m_pIntraFunc[0](iMemPredMb[:], pPtrYBuf, pTmpCur, iStrideY)
				iBlockSadH = c.m_pSadFunc(pPtrYBuf, pTmpCur, iStrideY, iMemPredMb[:], 0, 16)
			}
			if i > 0 {
				c.m_pIntraFunc[1](iMemPredMb[:], pPtrYBuf, pTmpCur, iStrideY)
				iBlockSadV = c.m_pSadFunc(pPtrYBuf, pTmpCur, iStrideY, iMemPredMb[:], 0, 16)
			}

			iGomSad += WELS_MIN(WELS_MIN(iBlockSadH, iBlockSadV), iInterSad)

			if i == iBlockWidth-1 && ((j+1)%c.m_ComplexityAnalysisParam.IMbRowInGom == 0 || j == iBlockHeight-1) {
				c.m_ComplexityAnalysisParam.PGomComplexity[iIdx] = iGomSad
				c.m_ComplexityAnalysisParam.IFrameComplexity += int64(iGomSad)
				iIdx++
				iGomSad = 0
			}

			pTmpRef += 16
			pTmpCur += 16
		}
		pPtrX += int(iRowStrideX)
		pPtrY += int(iRowStrideY)
	}
	c.m_ComplexityAnalysisParam.IGomNumInFrame = iIdx
}
