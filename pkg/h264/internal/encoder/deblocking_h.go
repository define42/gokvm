// Port of codec/encoder/core/inc/deblocking.h.

package encoder

// SDeblockingFilter is the deblocking state of the current MB.
type SDeblockingFilter struct {
	// C uint8_t* pCsData[3]: pointer to reconstructed picture pData (current MB);
	// (slice, offset): pCsData[i] is the whole picture allocation.
	pCsData    [3][]uint8
	iCsDataOff [3]int
	iCsStride  [3]int32 // Cs iStride

	iMbStride           int16
	iSliceAlphaC0Offset int8
	iSliceBetaOffset    int8
	uiLumaQP            uint8
	uiChromaQP          uint8
	uiFilterIdc         uint8
	uiReserved          uint8
}
