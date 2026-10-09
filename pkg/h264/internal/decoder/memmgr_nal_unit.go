// Port of codec/decoder/core/src/memmgr_nal_unit.cpp.
//
// Memory manager utils for NAL Unit list available.

package decoder

// MemInitNalList ports int32_t MemInitNalList (PAccessUnit* ppAu, const uint32_t kuiSize,
// CMemoryAlign* pMa).
//
// The C CMemoryAlign* pMa parameter is dropped. The C single-block allocation becomes a SAccessUnit
// plus kuiSize separately allocated SNalUnit values referenced from pNalUnitsList.
func MemInitNalList(ppAu **SAccessUnit, kuiSize uint32) int32 {
	if kuiSize == 0 {
		return ERR_INFO_INVALID_PARAM
	}

	if *ppAu != nil {
		MemFreeNalList(ppAu)
	}

	pAu := &SAccessUnit{}
	*ppAu = pAu
	pAu.pNalUnitsList = make([]*SNalUnit, kuiSize)
	pNalUnits := make([]SNalUnit, kuiSize)
	for uiIdx := uint32(0); uiIdx < kuiSize; uiIdx++ {
		pAu.pNalUnitsList[uiIdx] = &pNalUnits[uiIdx]
	}

	pAu.uiCountUnitsNum = kuiSize
	pAu.uiAvailUnitsNum = 0
	pAu.uiActualUnitsNum = 0
	pAu.uiStartPos = 0
	pAu.uiEndPos = 0
	pAu.bCompletedAuFlag = false

	return ERR_NONE
}

// MemFreeNalList ports int32_t MemFreeNalList (PAccessUnit* ppAu, CMemoryAlign* pMa).
//
// The C CMemoryAlign* pMa parameter is dropped.
func MemFreeNalList(ppAu **SAccessUnit) int32 {
	if ppAu != nil {
		pAu := *ppAu
		if pAu != nil {
			*ppAu = nil
		}
	}
	return ERR_NONE
}

// ExpandNalUnitList ports int32_t ExpandNalUnitList (PAccessUnit* ppAu, const int32_t kiOrgSize,
// const int32_t kiExpSize, CMemoryAlign* pMa).
//
// The C CMemoryAlign* pMa parameter is dropped.
func ExpandNalUnitList(ppAu **SAccessUnit, kiOrgSize int32, kiExpSize int32) int32 {
	if kiExpSize <= kiOrgSize {
		return ERR_INFO_INVALID_PARAM
	}
	var pTmp *SAccessUnit
	var iIdx int32
	if iRet := MemInitNalList(&pTmp, uint32(kiExpSize)); iRet != ERR_NONE { // request new list with expanding
		return iRet
	}

	for {
		*pTmp.pNalUnitsList[iIdx] = *(*ppAu).pNalUnitsList[iIdx]
		iIdx++
		if iIdx >= kiOrgSize {
			break
		}
	}

	pTmp.uiCountUnitsNum = uint32(kiExpSize)
	pTmp.uiAvailUnitsNum = (*ppAu).uiAvailUnitsNum
	pTmp.uiActualUnitsNum = (*ppAu).uiActualUnitsNum
	pTmp.uiEndPos = (*ppAu).uiEndPos
	pTmp.bCompletedAuFlag = (*ppAu).bCompletedAuFlag

	MemFreeNalList(ppAu) // free old list
	*ppAu = pTmp
	return ERR_NONE
}

// MemGetNextNal ports PNalUnit MemGetNextNal (PAccessUnit* ppAu, CMemoryAlign* pMa): get next NAL
// Unit for using. Need expand NAL Unit list if exceeding count number of available NAL Units within
// an Access Unit.
//
// The C CMemoryAlign* pMa parameter is dropped.
func MemGetNextNal(ppAu **SAccessUnit) *SNalUnit {
	pAu := *ppAu

	if pAu.uiAvailUnitsNum >= pAu.uiCountUnitsNum { // need expand list
		kuiExpandingSize := pAu.uiCountUnitsNum + (MAX_NAL_UNIT_NUM_IN_AU >> 1)
		if ExpandNalUnitList(ppAu, int32(pAu.uiCountUnitsNum), int32(kuiExpandingSize)) != 0 {
			return nil // out of memory
		}
		pAu = *ppAu
	}

	pNu := pAu.pNalUnitsList[pAu.uiAvailUnitsNum] // ready for next nal position
	pAu.uiAvailUnitsNum++

	*pNu = SNalUnit{} // Please do not remove this for cache intend!!

	return pNu
}
