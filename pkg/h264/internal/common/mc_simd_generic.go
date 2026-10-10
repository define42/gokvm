//go:build !amd64

package common

func FilterInput8bitWithStride_sse2(pSrc []uint8, iOff int, kiOffset int32) int32 {
	return FilterInput8bitWithStride_c(pSrc, iOff, kiOffset)
}

func PixelAvg_sse2(pDst []uint8, iDstOff int, iDstStride int32, pSrcA []uint8, iSrcAOff int,
	iSrcAStride int32, pSrcB []uint8, iSrcBOff int, iSrcBStride int32, iWidth, iHeight int32) {
	PixelAvg_c(pDst, iDstOff, iDstStride, pSrcA, iSrcAOff, iSrcAStride,
		pSrcB, iSrcBOff, iSrcBStride, iWidth, iHeight)
}

func PixelAvgWidthEq16_sse2(pDst []uint8, iDstOff int, iDstStride int32, pSrcA []uint8, iSrcAOff int,
	iSrcAStride int32, pSrcB []uint8, iSrcBOff int, iSrcBStride int32, iHeight int32) {
	PixelAvg_c(pDst, iDstOff, iDstStride, pSrcA, iSrcAOff, iSrcAStride,
		pSrcB, iSrcBOff, iSrcBStride, 16, iHeight)
}

func PixelAvgWidthEq8_sse2(pDst []uint8, iDstOff int, iDstStride int32, pSrcA []uint8, iSrcAOff int,
	iSrcAStride int32, pSrcB []uint8, iSrcBOff int, iSrcBStride int32, iHeight int32) {
	PixelAvg_c(pDst, iDstOff, iDstStride, pSrcA, iSrcAOff, iSrcAStride,
		pSrcB, iSrcBOff, iSrcBStride, 8, iHeight)
}

func PixelAvgWidthEq4_sse2(pDst []uint8, iDstOff int, iDstStride int32, pSrcA []uint8, iSrcAOff int,
	iSrcAStride int32, pSrcB []uint8, iSrcBOff int, iSrcBStride int32, iHeight int32) {
	PixelAvg_c(pDst, iDstOff, iDstStride, pSrcA, iSrcAOff, iSrcAStride,
		pSrcB, iSrcBOff, iSrcBStride, 4, iHeight)
}

func McHorVer20_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32, pDst []uint8, iDstOff int,
	iDstStride, iWidth, iHeight int32) {
	McHorVer20_c(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
}

func McHorVer02_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32, pDst []uint8, iDstOff int,
	iDstStride, iWidth, iHeight int32) {
	McHorVer02_c(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
}

func McHorVer22_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32, pDst []uint8, iDstOff int,
	iDstStride, iWidth, iHeight int32) {
	McHorVer22_c(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
}

func McLuma_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32, pDst []uint8, iDstOff int, iDstStride int32,
	iMvX, iMvY int16, iWidth, iHeight int32) {
	McLuma_c(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride,
		iMvX, iMvY, iWidth, iHeight)
}
