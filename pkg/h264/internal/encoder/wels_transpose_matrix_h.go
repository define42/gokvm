// Port of codec/encoder/core/inc/wels_transpose_matrix.h.
// (Only SIMD implementations exist; the function types are kept for completeness.)

package encoder

type PTransposeMatrixBlockFunc func(pDst []uint8, iDstOff int, kiDstStride int32, pSrc []uint8, iSrcOff int,
	kiSrcStride int32)
type PTransposeMatrixBlocksFunc func(pDst []uint8, iDstOff int, kiDstStride int32, pSrc []uint8, iSrcOff int,
	kiSrcStride int32, kiBlocksNum int32)
