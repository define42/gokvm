// Port of codec/decoder/core/inc/rec_mb.h.
//
// Functions declared in this header are implemented in rec_mb.go.

package decoder

// WELS_B_MB_REC_VERIFY(uiRet) contains a `return` and must be expanded
// inline:
//
//	if uiRetTmp := uint32(uiRet); uiRetTmp != ERR_NONE {
//		return int32(uiRetTmp)
//	}

// sMCRefMember (TagMCRefMember). Every plane pointer is a (slice, offset)
// pair: pFoo is the whole picture allocation (SPicture.pData[i]) and iFooOff
// the position the C pointer points to.
type sMCRefMember struct {
	pDstY    []uint8
	iDstYOff int
	pDstU    []uint8
	iDstUOff int
	pDstV    []uint8
	iDstVOff int

	pSrcY    []uint8
	iSrcYOff int
	pSrcU    []uint8
	iSrcUOff int
	pSrcV    []uint8
	iSrcVOff int

	iSrcLineLuma   int32
	iSrcLineChroma int32

	iDstLineLuma   int32
	iDstLineChroma int32

	iPicWidth  int32
	iPicHeight int32
}
