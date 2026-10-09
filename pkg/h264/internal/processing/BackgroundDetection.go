package processing

// Port of codec/processing/src/backgrounddetection/BackgroundDetection.cpp.

const (
	LOG2_BGD_OU_SIZE    = 4
	LOG2_BGD_OU_SIZE_UV = LOG2_BGD_OU_SIZE - 1
	BGD_OU_SIZE         = 1 << LOG2_BGD_OU_SIZE
	BGD_OU_SIZE_UV      = BGD_OU_SIZE >> 1
	BGD_THD_SAD         = 2 * BGD_OU_SIZE * BGD_OU_SIZE
	BGD_THD_ASD_UV      = 4 * BGD_OU_SIZE_UV
	LOG2_MB_SIZE        = 4
	OU_SIZE_IN_MB       = BGD_OU_SIZE >> 4
	Q_FACTOR            = 8
	BGD_DELTA_QP_THD    = 3

	OU_LEFT   = 0x01
	OU_RIGHT  = 0x02
	OU_TOP    = 0x04
	OU_BOTTOM = 0x08
)

func b2i(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// lnot is the C logical not `!x` on an int.
func lnot(x int32) int32 {
	if x == 0 {
		return 1
	}
	return 0
}

func NewCBackgroundDetection(iCpuFlag int32) *CBackgroundDetection {
	b := &CBackgroundDetection{}
	b.initIStrategyBase()
	b.m_eMethod = METHOD_BACKGROUND_DETECTION
	b.m_BgdParam = vBGDParam{}
	b.m_iLargestFrameSize = 0
	return b
}

func (b *CBackgroundDetection) Process(iType int32, pSrcPixMap *SPixMap, pRefPixMap *SPixMap) EResult {
	eReturn := RET_INVALIDPARAM

	if pSrcPixMap == nil || pRefPixMap == nil {
		return eReturn
	}

	b.m_BgdParam.pCur = pSrcPixMap.PPixel
	b.m_BgdParam.iCurOff = pSrcPixMap.IPixelOff
	b.m_BgdParam.pRef = pRefPixMap.PPixel
	b.m_BgdParam.iRefOff = pRefPixMap.IPixelOff
	b.m_BgdParam.iBgdWidth = pSrcPixMap.SRect.IRectWidth
	b.m_BgdParam.iBgdHeight = pSrcPixMap.SRect.IRectHeight
	b.m_BgdParam.iStride[0] = pSrcPixMap.IStride[0]
	b.m_BgdParam.iStride[1] = pSrcPixMap.IStride[1]
	b.m_BgdParam.iStride[2] = pSrcPixMap.IStride[2]

	iCurFrameSize := b.m_BgdParam.iBgdWidth * b.m_BgdParam.iBgdHeight
	if b.m_BgdParam.pOU_array == nil || iCurFrameSize > b.m_iLargestFrameSize {
		b.m_BgdParam.pOU_array = b.AllocateOUArrayMemory(b.m_BgdParam.iBgdWidth, b.m_BgdParam.iBgdHeight)
		b.m_iLargestFrameSize = iCurFrameSize
	}

	if b.m_BgdParam.pOU_array == nil {
		return eReturn
	}

	b.BackgroundDetection(&b.m_BgdParam)

	return RET_SUCCESS
}

// Set takes a *SBGDInterface.
func (b *CBackgroundDetection) Set(iType int32, pParam any) EResult {
	pInterface, ok := pParam.(*SBGDInterface)
	if !ok || pInterface == nil {
		return RET_INVALIDPARAM
	}

	b.m_BgdParam.pBackgroundMbFlag = pInterface.PBackgroundMbFlag
	b.m_BgdParam.pCalcRes = pInterface.PCalcRes

	return RET_SUCCESS
}

func (b *CBackgroundDetection) AllocateOUArrayMemory(iWidth int32, iHeight int32) []SBackgroundOU {
	iMaxOUWidth := (BGD_OU_SIZE - 1 + iWidth) >> LOG2_BGD_OU_SIZE
	iMaxOUHeight := (BGD_OU_SIZE - 1 + iHeight) >> LOG2_BGD_OU_SIZE
	return make([]SBackgroundOU, iMaxOUWidth*iMaxOUHeight)
}

func (b *CBackgroundDetection) GetOUParameters(sVaaCalcInfo *SVAACalcResult, iMbIndex int32, iMbWidth int32,
	pBgdOU *SBackgroundOU) {
	var iSubSD [4]int32
	var iSubMAD [4]uint8
	var iSubSAD [4]int32

	pSad8x8 := sVaaCalcInfo.PSad8x8
	pMad8x8 := sVaaCalcInfo.PMad8x8
	pSd8x8 := sVaaCalcInfo.PSumOfDiff8x8

	iSubSAD[0] = pSad8x8[iMbIndex][0]
	iSubSAD[1] = pSad8x8[iMbIndex][1]
	iSubSAD[2] = pSad8x8[iMbIndex][2]
	iSubSAD[3] = pSad8x8[iMbIndex][3]

	iSubSD[0] = pSd8x8[iMbIndex][0]
	iSubSD[1] = pSd8x8[iMbIndex][1]
	iSubSD[2] = pSd8x8[iMbIndex][2]
	iSubSD[3] = pSd8x8[iMbIndex][3]

	iSubMAD[0] = pMad8x8[iMbIndex][0]
	iSubMAD[1] = pMad8x8[iMbIndex][1]
	iSubMAD[2] = pMad8x8[iMbIndex][2]
	iSubMAD[3] = pMad8x8[iMbIndex][3]

	pBgdOU.iSD = iSubSD[0] + iSubSD[1] + iSubSD[2] + iSubSD[3]
	pBgdOU.iSAD = iSubSAD[0] + iSubSAD[1] + iSubSAD[2] + iSubSAD[3]
	pBgdOU.iSD = WELS_ABS(pBgdOU.iSD)

	// get the max absolute difference (MAD) of OU and min value of the MAD of sub-blocks of OU
	pBgdOU.iMAD = int32(WELS_MAX(WELS_MAX(iSubMAD[0], iSubMAD[1]), WELS_MAX(iSubMAD[2], iSubMAD[3])))
	pBgdOU.iMinSubMad = int32(WELS_MIN(WELS_MIN(iSubMAD[0], iSubMAD[1]), WELS_MIN(iSubMAD[2], iSubMAD[3])))

	// get difference between the max and min SD of the SDs of sub-blocks of OU
	pBgdOU.iMaxDiffSubSd = WELS_MAX(WELS_MAX(iSubSD[0], iSubSD[1]), WELS_MAX(iSubSD[2], iSubSD[3])) -
		WELS_MIN(WELS_MIN(iSubSD[0], iSubSD[1]), WELS_MIN(iSubSD[2], iSubSD[3]))
}

func (b *CBackgroundDetection) ForegroundBackgroundDivision(pBgdParam *vBGDParam) {
	iPicWidthInOU := pBgdParam.iBgdWidth >> LOG2_BGD_OU_SIZE
	iPicHeightInOU := pBgdParam.iBgdHeight >> LOG2_BGD_OU_SIZE
	iPicWidthInMb := (15 + pBgdParam.iBgdWidth) >> 4

	pBackgroundOU := 0
	pOU := pBgdParam.pOU_array

	for j := int32(0); j < iPicHeightInOU; j++ {
		for i := int32(0); i < iPicWidthInOU; i++ {
			pCur := &pOU[pBackgroundOU]
			b.GetOUParameters(pBgdParam.pCalcRes, (j*iPicWidthInMb+i)<<(LOG2_BGD_OU_SIZE-LOG2_MB_SIZE), iPicWidthInMb,
				pCur)

			pCur.iBackgroundFlag = 0
			if pCur.iMAD > 63 {
				pBackgroundOU++
				continue
			}
			if (pCur.iMaxDiffSubSd <= pCur.iSAD>>3 ||
				pCur.iMaxDiffSubSd <= (BGD_OU_SIZE*Q_FACTOR)) &&
				pCur.iSAD < (BGD_THD_SAD<<1) { //BGD_OU_SIZE*BGD_OU_SIZE>>2
				if pCur.iSAD <= BGD_OU_SIZE*Q_FACTOR {
					pCur.iBackgroundFlag = 1
				} else {
					if pCur.iSAD < BGD_THD_SAD {
						pCur.iBackgroundFlag = b2i(pCur.iSD < (pCur.iSAD*3)>>2)
					} else {
						pCur.iBackgroundFlag = b2i(pCur.iSD<<1 < pCur.iSAD)
					}
				}
			}
			pBackgroundOU++
		}
	}
}

func (b *CBackgroundDetection) CalculateAsdChromaEdge(pOriRef []uint8, iRefOff int, pOriCur []uint8, iCurOff int, iStride int32) int32 {
	var ASD int32
	for idx := 0; idx < BGD_OU_SIZE_UV; idx++ {
		ASD += int32(pOriCur[iCurOff]) - int32(pOriRef[iRefOff])
		iRefOff += int(iStride)
		iCurOff += int(iStride)
	}
	return WELS_ABS(ASD)
}

func (b *CBackgroundDetection) ForegroundDilation23Luma(pBackgroundOU *SBackgroundOU,
	pOUNeighbours *[4]*SBackgroundOU) bool {
	pOU_L := pOUNeighbours[0]
	pOU_R := pOUNeighbours[1]
	pOU_U := pOUNeighbours[2]
	pOU_D := pOUNeighbours[3]

	if pBackgroundOU.iMAD > pBackgroundOU.iMinSubMad<<1 {
		var iMaxNbrForegroundMad int32
		var iMaxNbrBackgroundMad int32
		var aBackgroundMad [4]int32
		var aForegroundMad [4]int32

		aForegroundMad[0] = (pOU_L.iBackgroundFlag - 1) & pOU_L.iMAD
		aForegroundMad[1] = (pOU_R.iBackgroundFlag - 1) & pOU_R.iMAD
		aForegroundMad[2] = (pOU_U.iBackgroundFlag - 1) & pOU_U.iMAD
		aForegroundMad[3] = (pOU_D.iBackgroundFlag - 1) & pOU_D.iMAD
		iMaxNbrForegroundMad = WELS_MAX(WELS_MAX(aForegroundMad[0], aForegroundMad[1]), WELS_MAX(aForegroundMad[2],
			aForegroundMad[3]))

		aBackgroundMad[0] = (lnot(pOU_L.iBackgroundFlag) - 1) & pOU_L.iMAD
		aBackgroundMad[1] = (lnot(pOU_R.iBackgroundFlag) - 1) & pOU_R.iMAD
		aBackgroundMad[2] = (lnot(pOU_U.iBackgroundFlag) - 1) & pOU_U.iMAD
		aBackgroundMad[3] = (lnot(pOU_D.iBackgroundFlag) - 1) & pOU_D.iMAD
		iMaxNbrBackgroundMad = WELS_MAX(WELS_MAX(aBackgroundMad[0], aBackgroundMad[1]), WELS_MAX(aBackgroundMad[2],
			aBackgroundMad[3]))

		return (iMaxNbrForegroundMad > pBackgroundOU.iMinSubMad<<2) || (pBackgroundOU.iMAD > iMaxNbrBackgroundMad<<1 &&
			pBackgroundOU.iMAD <= (iMaxNbrForegroundMad*3)>>1)
	}
	return false
}

var kaOUPos = [4]int8{OU_LEFT, OU_RIGHT, OU_TOP, OU_BOTTOM}

func (b *CBackgroundDetection) ForegroundDilation23Chroma(iNeighbourForegroundFlags int8,
	iStartSamplePos int32, iPicStrideUV int32, pBgdParam *vBGDParam) bool {
	aEdgeOffset := [4]int32{0, BGD_OU_SIZE_UV - 1, 0, iPicStrideUV * (BGD_OU_SIZE_UV - 1)}
	iStride := [4]int32{iPicStrideUV, iPicStrideUV, 1, 1}

	// V component first, high probability because V stands for red color and human skin colors have more weight on this component
	for i := 0; i < 4; i++ {
		if iNeighbourForegroundFlags&kaOUPos[i] != 0 {
			pRefC := pBgdParam.iRefOff[2] + int(iStartSamplePos+aEdgeOffset[i])
			pCurC := pBgdParam.iCurOff[2] + int(iStartSamplePos+aEdgeOffset[i])
			if b.CalculateAsdChromaEdge(pBgdParam.pRef[2], pRefC, pBgdParam.pCur[2], pCurC, iStride[i]) > BGD_THD_ASD_UV {
				return true
			}
		}
	}
	// U component, which stands for blue color, low probability
	for i := 0; i < 4; i++ {
		if iNeighbourForegroundFlags&kaOUPos[i] != 0 {
			pRefC := pBgdParam.iRefOff[1] + int(iStartSamplePos+aEdgeOffset[i])
			pCurC := pBgdParam.iCurOff[1] + int(iStartSamplePos+aEdgeOffset[i])
			if b.CalculateAsdChromaEdge(pBgdParam.pRef[1], pRefC, pBgdParam.pCur[1], pCurC, iStride[i]) > BGD_THD_ASD_UV {
				return true
			}
		}
	}

	return false
}

func (b *CBackgroundDetection) ForegroundDilation(pBackgroundOU *SBackgroundOU, pOUNeighbours *[4]*SBackgroundOU,
	pBgdParam *vBGDParam, iChromaSampleStartPos int32) {
	iPicStrideUV := pBgdParam.iStride[1]
	iSumNeighBackgroundFlags := pOUNeighbours[0].iBackgroundFlag + pOUNeighbours[1].iBackgroundFlag +
		pOUNeighbours[2].iBackgroundFlag + pOUNeighbours[3].iBackgroundFlag

	if pBackgroundOU.iSAD > BGD_OU_SIZE*Q_FACTOR {
		switch iSumNeighBackgroundFlags {
		case 0, 1:
			pBackgroundOU.iBackgroundFlag = 0
		case 2, 3:
			pBackgroundOU.iBackgroundFlag = b2i(!b.ForegroundDilation23Luma(pBackgroundOU, pOUNeighbours))

			// chroma component check
			if pBackgroundOU.iBackgroundFlag == 1 {
				iNeighbourForegroundFlags := int8(lnot(pOUNeighbours[0].iBackgroundFlag) | (lnot(pOUNeighbours[1].iBackgroundFlag) << 1) |
					(lnot(pOUNeighbours[2].iBackgroundFlag) << 2) | (lnot(pOUNeighbours[3].iBackgroundFlag) << 3))
				pBackgroundOU.iBackgroundFlag = b2i(!b.ForegroundDilation23Chroma(iNeighbourForegroundFlags, iChromaSampleStartPos,
					iPicStrideUV, pBgdParam))
			}
		default:
		}
	}
}

func (b *CBackgroundDetection) BackgroundErosion(pBackgroundOU *SBackgroundOU, pOUNeighbours *[4]*SBackgroundOU) {
	if pBackgroundOU.iMaxDiffSubSd <= (BGD_OU_SIZE * Q_FACTOR) { //BGD_OU_SIZE*BGD_OU_SIZE>>2
		iSumNeighBackgroundFlags := pOUNeighbours[0].iBackgroundFlag + pOUNeighbours[1].iBackgroundFlag +
			pOUNeighbours[2].iBackgroundFlag + pOUNeighbours[3].iBackgroundFlag
		sumNbrBGsad := (pOUNeighbours[0].iSAD & (-pOUNeighbours[0].iBackgroundFlag)) + (pOUNeighbours[2].iSAD &
			(-pOUNeighbours[2].iBackgroundFlag)) +
			(pOUNeighbours[1].iSAD & (-pOUNeighbours[1].iBackgroundFlag)) + (pOUNeighbours[3].iSAD &
			(-pOUNeighbours[3].iBackgroundFlag))
		if pBackgroundOU.iSAD*iSumNeighBackgroundFlags <= (3*sumNbrBGsad)>>1 {
			if iSumNeighBackgroundFlags == 4 {
				pBackgroundOU.iBackgroundFlag = 1
			} else {
				if (pOUNeighbours[0].iBackgroundFlag&pOUNeighbours[1].iBackgroundFlag) != 0 ||
					(pOUNeighbours[2].iBackgroundFlag&pOUNeighbours[3].iBackgroundFlag) != 0 {
					pBackgroundOU.iBackgroundFlag = b2i(!b.ForegroundDilation23Luma(pBackgroundOU, pOUNeighbours))
				}
			}
		}
	}
}

func (b *CBackgroundDetection) SetBackgroundMbFlag(pBackgroundMbFlag []int8, iFlagOff int, iPicWidthInMb int32,
	iBackgroundMbFlag int32) {
	pBackgroundMbFlag[iFlagOff] = int8(iBackgroundMbFlag)
}

func (b *CBackgroundDetection) UpperOUForegroundCheck(pOU []SBackgroundOU, iCurOU int, pBackgroundMbFlag []int8, iFlagOff int,
	iPicWidthInOU int32, iPicWidthInMb int32) {
	pCurOU := &pOU[iCurOU]
	if pCurOU.iSAD > BGD_OU_SIZE*Q_FACTOR {
		pOU_L := &pOU[iCurOU-1]
		pOU_R := &pOU[iCurOU+1]
		pOU_U := &pOU[iCurOU-int(iPicWidthInOU)]
		pOU_D := &pOU[iCurOU+int(iPicWidthInOU)]
		if pOU_L.iBackgroundFlag+pOU_R.iBackgroundFlag+pOU_U.iBackgroundFlag+pOU_D.iBackgroundFlag <= 1 {
			b.SetBackgroundMbFlag(pBackgroundMbFlag, iFlagOff, iPicWidthInMb, 0)
			pCurOU.iBackgroundFlag = 0
		}
	}
}

func (b *CBackgroundDetection) ForegroundDilationAndBackgroundErosion(pBgdParam *vBGDParam) {
	iPicStrideUV := pBgdParam.iStride[1]
	iPicWidthInOU := pBgdParam.iBgdWidth >> LOG2_BGD_OU_SIZE
	iPicHeightInOU := pBgdParam.iBgdHeight >> LOG2_BGD_OU_SIZE
	iOUStrideUV := iPicStrideUV << (LOG2_BGD_OU_SIZE - 1)
	iPicWidthInMb := (15 + pBgdParam.iBgdWidth) >> 4

	pOU := pBgdParam.pOU_array
	pBackgroundOU := 0
	pVaaBackgroundMbFlag := pBgdParam.pBackgroundMbFlag
	iVaaBackgroundMbFlag := 0
	var pOUNeighbours [4]int //0: left; 1: right; 2: top; 3: bottom (indices into pOU)
	var pNbr [4]*SBackgroundOU

	pOUNeighbours[2] = pBackgroundOU //top OU
	for j := int32(0); j < iPicHeightInOU; j++ {
		pRowSkipFlag := iVaaBackgroundMbFlag
		pOUNeighbours[0] = pBackgroundOU                                                     //left OU
		pOUNeighbours[3] = pBackgroundOU + int(iPicWidthInOU&(b2i(j == iPicHeightInOU-1)-1)) //bottom OU
		for i := int32(0); i < iPicWidthInOU; i++ {
			pOUNeighbours[1] = pBackgroundOU + int(b2i(i < iPicWidthInOU-1)) //right OU

			for k := 0; k < 4; k++ {
				pNbr[k] = &pOU[pOUNeighbours[k]]
			}
			if pOU[pBackgroundOU].iBackgroundFlag != 0 {
				b.ForegroundDilation(&pOU[pBackgroundOU], &pNbr, pBgdParam, j*iOUStrideUV+(i<<LOG2_BGD_OU_SIZE_UV))
			} else {
				b.BackgroundErosion(&pOU[pBackgroundOU], &pNbr)
			}

			// check the up OU
			if j > 1 && i > 0 && i < iPicWidthInOU-1 && pOU[pOUNeighbours[2]].iBackgroundFlag == 1 {
				b.UpperOUForegroundCheck(pOU, pOUNeighbours[2], pVaaBackgroundMbFlag, pRowSkipFlag-OU_SIZE_IN_MB*int(iPicWidthInMb),
					iPicWidthInOU, iPicWidthInMb)
			}

			b.SetBackgroundMbFlag(pVaaBackgroundMbFlag, pRowSkipFlag, iPicWidthInMb, pOU[pBackgroundOU].iBackgroundFlag)

			// preparation for the next OU
			pRowSkipFlag += OU_SIZE_IN_MB
			pOUNeighbours[0] = pBackgroundOU
			pOUNeighbours[2]++
			pOUNeighbours[3]++
			pBackgroundOU++
		}
		pOUNeighbours[2] = pBackgroundOU - int(iPicWidthInOU)
		iVaaBackgroundMbFlag += OU_SIZE_IN_MB * int(iPicWidthInMb)
	}
}

func (b *CBackgroundDetection) BackgroundDetection(pBgdParam *vBGDParam) {
	// 1st step: foreground/background coarse division
	b.ForegroundBackgroundDivision(pBgdParam)

	// 2nd step: foreground dilation and background erosion
	b.ForegroundDilationAndBackgroundErosion(pBgdParam)
}
