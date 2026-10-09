// Port of codec/decoder/core/inc/fmo.h.

package decoder

// MB_XY_T is the type of MB addresses.
type MB_XY_T = int32

/*!
 * \brief   Wels Flexible Macroblock Ordering (FMO)
 */
type SFmo struct {
	pMbAllocMap      []uint8 // C: uint8_t* (iCountMbNum entries)
	iCountMbNum      int32
	iSliceGroupCount int32
	iSliceGroupType  int32
	bActiveFlag      bool
	uiReserved       [3]uint8 // reserved padding bytes
}

type PFmo = *SFmo

// The C functions take a trailing `CMemoryAlign* pMa` parameter; it is
// dropped in Go. Implemented in fmo.go:
//
//	func InitFmo(pFmo *SFmo, pPps *SPps, kiMbWidth, kiMbHeight int32) int32
//	func UninitFmoList(pFmo []SFmo, kiCnt, kiAvail int32)
//	func FmoParamUpdate(pFmo *SFmo, pSps *SSps, pPps *SPps, pActiveFmoNum *int32) int32
//	func FmoNextMb(pFmo *SFmo, kiMbXy MB_XY_T) MB_XY_T
