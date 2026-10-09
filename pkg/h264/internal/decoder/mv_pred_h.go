// Port of codec/decoder/core/inc/mv_pred.h.
//
// Functions declared in this header are implemented in mv_pred.go.

package decoder

// RETURN_ERR_IF_NULL(pRefPic0) contains a `return` and must be expanded
// inline:
//
//	if pRefPic0 == nil {
//		return GENERATE_ERROR_NO(ERR_LEVEL_MB_DATA, ERR_INFO_INVALID_REF_INDEX)
//	}

// GetMbType returns the per-MB mb type array, taken from the picture when
// available.
func GetMbType(pCurDqLayer *SDqLayer) []uint32 {
	if pCurDqLayer.pDec != nil {
		return pCurDqLayer.pDec.pMbType
	}
	return pCurDqLayer.pMbType
}
