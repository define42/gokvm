// Port of codec/decoder/core/inc/deblocking.h.
//
// Functions declared in this header are implemented in deblocking.go.

package decoder

// GetPNzc returns the non-zero-count entry (24 values) of MB iMbXy, taken
// from the picture when it carries its own copy.
func GetPNzc(pCurDqLayer *SDqLayer, iMbXy int32) []int8 {
	if pCurDqLayer.pDec != nil && pCurDqLayer.pDec.pNzc != nil {
		return pCurDqLayer.pDec.pNzc[iMbXy][:]
	}
	return pCurDqLayer.pNzc[iMbXy][:]
}
