// Port of codec/decoder/core/src/fmo.cpp.
//
// Flexible Macroblock Ordering implementation.

package decoder

// FmoGenerateMbAllocMapType0 generates the MB allocated map for interleaved slice group (TYPE 0).
//
// return 0 - successful; none 0 - failed
func FmoGenerateMbAllocMapType0(pFmo *SFmo, pPps *SPps) int32 {
	var uiNumSliceGroups uint32
	var iMbNum int32
	var i int32

	if pFmo == nil || pPps == nil {
		return ERR_INFO_INVALID_PARAM
	}
	uiNumSliceGroups = pPps.uiNumSliceGroups
	iMbNum = pFmo.iCountMbNum
	if pFmo.pMbAllocMap == nil || iMbNum <= 0 || uiNumSliceGroups > MAX_SLICEGROUP_IDS {
		return ERR_INFO_INVALID_PARAM
	}

	for {
		var uiGroup uint8
		for {
			kuiRunIdx := pPps.uiRunLength[uiGroup]
			// Reject zero or oversized run lengths before advancing i. A single run
			// cannot legitimately exceed the total MB count; this also prevents any
			// integer wrap of the map index with attacker-influenced values.
			if kuiRunIdx == 0 || kuiRunIdx > uint32(iMbNum) {
				return ERR_INFO_INVALID_PARAM
			}
			var j int32
			for {
				pFmo.pMbAllocMap[i+j] = uiGroup
				j++
				if !(j < int32(kuiRunIdx) && i+j < iMbNum) {
					break
				}
			}
			i += int32(kuiRunIdx)
			uiGroup++
			if !(uint32(uiGroup) < uiNumSliceGroups && i < iMbNum) {
				break
			}
		}
		if !(i < iMbNum) {
			break
		}
	}

	return ERR_NONE // well here
}

// FmoGenerateMbAllocMapType1 generates the MB allocated map for dispersed slice group (TYPE 1).
//
// return 0 - successful; none 0 - failed
func FmoGenerateMbAllocMapType1(pFmo *SFmo, pPps *SPps, kiMbWidth int32) int32 {
	var uiNumSliceGroups uint32
	var iMbNum int32
	var i int32
	if pFmo == nil || pPps == nil {
		return ERR_INFO_INVALID_PARAM
	}
	uiNumSliceGroups = pPps.uiNumSliceGroups
	iMbNum = pFmo.iCountMbNum
	if pFmo.pMbAllocMap == nil || iMbNum <= 0 || kiMbWidth == 0 || uiNumSliceGroups > MAX_SLICEGROUP_IDS {
		return ERR_INFO_INVALID_PARAM
	}

	for {
		// C: int32 operands mixed with uint32_t uiNumSliceGroups -> unsigned arithmetic.
		pFmo.pMbAllocMap[i] = uint8((uint32(i%kiMbWidth) + ((uint32(i/kiMbWidth) * uiNumSliceGroups) >> 1)) % uiNumSliceGroups)
		i++
		if !(i < iMbNum) {
			break
		}
	}

	return ERR_NONE // well here
}

// FmoGenerateSliceGroup generates the MB allocated map for various type of slice group cases
// (TYPE 0, .., 6).
//
// return 0 - successful; none 0 - failed
func FmoGenerateSliceGroup(pFmo *SFmo, kpPps *SPps, kiMbWidth int32, kiMbHeight int32) int32 {
	var iNumMb int32
	var iErr int32
	bResolutionChanged := false

	// the cases we would not like
	if pFmo == nil || kpPps == nil {
		return ERR_INFO_INVALID_PARAM
	}

	iNumMb = kiMbWidth * kiMbHeight

	if iNumMb == 0 {
		return ERR_INFO_INVALID_PARAM
	}

	pFmo.pMbAllocMap = nil
	if iNumMb < 0 { // C: WelsMallocz of a huge size fails
		return ERR_INFO_OUT_OF_MEMORY
	}
	pFmo.pMbAllocMap = make([]uint8, iNumMb)

	pFmo.iCountMbNum = iNumMb

	if kpPps.uiNumSliceGroups < 2 && iNumMb > 0 { // only one slice group, exactly it is single slice based
		clear(pFmo.pMbAllocMap) // for safe

		pFmo.iSliceGroupCount = 1

		return ERR_NONE
	}

	if bResolutionChanged || (int32(kpPps.uiSliceGroupMapType) != pFmo.iSliceGroupType) ||
		(int32(kpPps.uiNumSliceGroups) != pFmo.iSliceGroupCount) {
		switch kpPps.uiSliceGroupMapType {
		case 0:
			iErr = FmoGenerateMbAllocMapType0(pFmo, kpPps)
		case 1:
			iErr = FmoGenerateMbAllocMapType1(pFmo, kpPps, kiMbWidth)
		case 2, 3, 4, 5, 6:
			// Reserve for others slice group type
			iErr = 1
		default:
			return ERR_INFO_UNSUPPORTED_FMOTYPE
		}
	}

	if iErr == 0 { // well now
		pFmo.iSliceGroupCount = int32(kpPps.uiNumSliceGroups)
		pFmo.iSliceGroupType = int32(kpPps.uiSliceGroupMapType)
	} else {
		// Map generation failed (e.g. a rejected/oversized run length). The map was
		// allocated above but this FMO is not marked active, so UninitFmoList would
		// never free it. Release it here to avoid a leak on the rejection path.
		pFmo.pMbAllocMap = nil
		pFmo.iCountMbNum = 0
	}

	return iErr
}

// InitFmo ports int32_t InitFmo (PFmo pFmo, PPps pPps, const int32_t kiMbWidth, const int32_t
// kiMbHeight, CMemoryAlign* pMa): initialize Wels Flexible Macroblock Ordering (FMO).
//
// The C CMemoryAlign* pMa parameter is dropped.
func InitFmo(pFmo *SFmo, pPps *SPps, kiMbWidth int32, kiMbHeight int32) int32 {
	return FmoGenerateSliceGroup(pFmo, pPps, kiMbWidth, kiMbHeight)
}

// UninitFmoList ports void UninitFmoList (PFmo pFmo, const int32_t kiCnt, const int32_t kiAvail,
// CMemoryAlign* pMa): uninitialize Wels Flexible Macroblock Ordering (FMO) list.
//
// pFmo: the FMO list (C: PFmo pointing at sFmoList[0]). The C CMemoryAlign* pMa parameter is dropped.
func UninitFmoList(pFmo []SFmo, kiCnt int32, kiAvail int32) {
	var i int32
	var iFreeNodes int32

	if pFmo == nil || kiAvail <= 0 || kiCnt < kiAvail {
		return
	}

	for i < kiCnt {
		pIter := &pFmo[i]
		if pIter.bActiveFlag {
			if pIter.pMbAllocMap != nil {
				pIter.pMbAllocMap = nil
			}
			pIter.iSliceGroupCount = 0
			pIter.iSliceGroupType = -1
			pIter.iCountMbNum = 0
			pIter.bActiveFlag = false
			iFreeNodes++
			if iFreeNodes >= kiAvail {
				break
			}
		}
		i++
	}
}

// FmoParamSetsChanged ports bool FmoParamSetsChanged (PFmo pFmo, const int32_t kiCountNumMb, const
// int32_t kiSliceGroupType, const int32_t kiSliceGroupCount): detect parameter sets are changed or
// not.
//
// return true - changed or not initialized yet; false - not change at all
func FmoParamSetsChanged(pFmo *SFmo, kiCountNumMb int32, kiSliceGroupType int32, kiSliceGroupCount int32) bool {
	if pFmo == nil {
		return false
	}

	return (!pFmo.bActiveFlag) ||
		(kiCountNumMb != pFmo.iCountMbNum) ||
		(kiSliceGroupType != pFmo.iSliceGroupType) ||
		(kiSliceGroupCount != pFmo.iSliceGroupCount)
}

// FmoParamUpdate ports int32_t FmoParamUpdate (PFmo pFmo, PSps pSps, PPps pPps, int32_t*
// pActiveFmoNum, CMemoryAlign* pMa): update/insert FMO parameter unit.
//
// The C CMemoryAlign* pMa parameter is dropped.
func FmoParamUpdate(pFmo *SFmo, pSps *SSps, pPps *SPps, pActiveFmoNum *int32) int32 {
	kuiMbWidth := pSps.iMbWidth
	kuiMbHeight := pSps.iMbHeight
	var iRet int32 = ERR_NONE
	if FmoParamSetsChanged(pFmo, int32(kuiMbWidth*kuiMbHeight), int32(pPps.uiSliceGroupMapType), int32(pPps.uiNumSliceGroups)) {
		iRet = InitFmo(pFmo, pPps, int32(kuiMbWidth), int32(kuiMbHeight))
		if iRet != 0 {
			return iRet
		}

		if !pFmo.bActiveFlag && *pActiveFmoNum < MAX_PPS_COUNT {
			*pActiveFmoNum++
			pFmo.bActiveFlag = true
		}
	}
	return iRet
}

// FmoMbToSliceGroup ports int32_t FmoMbToSliceGroup (PFmo pFmo, const MB_XY_T kiMbXy): convert kMbXy
// to slice group idc correspondingly.
//
// return slice group idc - successful; -1 - failed;
func FmoMbToSliceGroup(pFmo *SFmo, kiMbXy MB_XY_T) int32 {
	kiMbNum := pFmo.iCountMbNum
	kpMbMap := pFmo.pMbAllocMap

	if kiMbXy < 0 || kiMbXy >= kiMbNum || kpMbMap == nil {
		return -1
	}

	return int32(kpMbMap[kiMbXy])
}

// FmoNextMb ports MB_XY_T FmoNextMb (PFmo pFmo, const MB_XY_T kiMbXy): get successive mb to be
// processed with given current kMbXy.
//
// return iNextMb - successful; -1 - failed;
func FmoNextMb(pFmo *SFmo, kiMbXy MB_XY_T) MB_XY_T {
	kiTotalMb := pFmo.iCountMbNum
	kpMbMap := pFmo.pMbAllocMap
	iNextMb := kiMbXy
	kuiSliceGroupIdc := uint8(FmoMbToSliceGroup(pFmo, kiMbXy))

	if kuiSliceGroupIdc == 0xff {
		return -1
	}

	for {
		iNextMb++
		if iNextMb >= kiTotalMb {
			iNextMb = -1
			break
		}
		if kpMbMap[iNextMb] == kuiSliceGroupIdc {
			break
		}
	}

	// -1: No further MB in this slice (could be end of picture)
	return iNextMb
}
