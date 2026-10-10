package common

const mcLumaSIMDTempStride int32 = 16

// mcLumaQuarterpel_sse2 composes the twelve quarter-pel luma positions from
// the SIMD half-pel filters and packed averaging kernel. The codec limits luma
// motion-compensation blocks to 16 by 16, matching the fixed scratch buffers.
// Direct integer and half-pel positions are handled by McLuma_sse2.
func mcLumaQuarterpel_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride int32, iMvX, iMvY int16,
	iWidth, iHeight int32) {
	switch (uint16(iMvX)&3)<<2 | (uint16(iMvY) & 3) {
	case 1:
		mcHorVer01_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 3:
		mcHorVer03_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 1 << 2:
		mcHorVer10_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 1<<2 | 1:
		mcHorVer11_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 1<<2 | 2:
		mcHorVer12_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 1<<2 | 3:
		mcHorVer13_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 2<<2 | 1:
		mcHorVer21_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 2<<2 | 3:
		mcHorVer23_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 3 << 2:
		mcHorVer30_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 3<<2 | 1:
		mcHorVer31_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 3<<2 | 2:
		mcHorVer32_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	case 3<<2 | 3:
		mcHorVer33_sse2(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride, iWidth, iHeight)
	default:
		McLuma_c(pSrc, iSrcOff, iSrcStride, pDst, iDstOff, iDstStride,
			iMvX, iMvY, iWidth, iHeight)
	}
}

func mcHorVer01_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var vertical [16 * 16]uint8
	McHorVer02_sse2(pSrc, iSrcOff, iSrcStride, vertical[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, pSrc, iSrcOff, iSrcStride,
		vertical[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer03_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var vertical [16 * 16]uint8
	McHorVer02_sse2(pSrc, iSrcOff, iSrcStride, vertical[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, pSrc, iSrcOff+int(iSrcStride), iSrcStride,
		vertical[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer10_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var horizontal [16 * 16]uint8
	McHorVer20_sse2(pSrc, iSrcOff, iSrcStride, horizontal[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, pSrc, iSrcOff, iSrcStride,
		horizontal[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer11_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var horizontal, vertical [16 * 16]uint8
	McHorVer20_sse2(pSrc, iSrcOff, iSrcStride, horizontal[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	McHorVer02_sse2(pSrc, iSrcOff, iSrcStride, vertical[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, horizontal[:], 0, mcLumaSIMDTempStride,
		vertical[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer12_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var vertical, center [16 * 16]uint8
	McHorVer02_sse2(pSrc, iSrcOff, iSrcStride, vertical[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	McHorVer22_sse2(pSrc, iSrcOff, iSrcStride, center[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, vertical[:], 0, mcLumaSIMDTempStride,
		center[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer13_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var horizontal, vertical [16 * 16]uint8
	McHorVer20_sse2(pSrc, iSrcOff+int(iSrcStride), iSrcStride, horizontal[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	McHorVer02_sse2(pSrc, iSrcOff, iSrcStride, vertical[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, horizontal[:], 0, mcLumaSIMDTempStride,
		vertical[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer21_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var horizontal, center [16 * 16]uint8
	McHorVer20_sse2(pSrc, iSrcOff, iSrcStride, horizontal[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	McHorVer22_sse2(pSrc, iSrcOff, iSrcStride, center[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, horizontal[:], 0, mcLumaSIMDTempStride,
		center[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer23_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var horizontal, center [16 * 16]uint8
	McHorVer20_sse2(pSrc, iSrcOff+int(iSrcStride), iSrcStride, horizontal[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	McHorVer22_sse2(pSrc, iSrcOff, iSrcStride, center[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, horizontal[:], 0, mcLumaSIMDTempStride,
		center[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer30_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var horizontal [16 * 16]uint8
	McHorVer20_sse2(pSrc, iSrcOff, iSrcStride, horizontal[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, pSrc, iSrcOff+1, iSrcStride,
		horizontal[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer31_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var horizontal, vertical [16 * 16]uint8
	McHorVer20_sse2(pSrc, iSrcOff, iSrcStride, horizontal[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	McHorVer02_sse2(pSrc, iSrcOff+1, iSrcStride, vertical[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, horizontal[:], 0, mcLumaSIMDTempStride,
		vertical[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer32_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var vertical, center [16 * 16]uint8
	McHorVer02_sse2(pSrc, iSrcOff+1, iSrcStride, vertical[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	McHorVer22_sse2(pSrc, iSrcOff, iSrcStride, center[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, vertical[:], 0, mcLumaSIMDTempStride,
		center[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}

func mcHorVer33_sse2(pSrc []uint8, iSrcOff int, iSrcStride int32,
	pDst []uint8, iDstOff int, iDstStride, iWidth, iHeight int32) {
	var horizontal, vertical [16 * 16]uint8
	McHorVer20_sse2(pSrc, iSrcOff+int(iSrcStride), iSrcStride, horizontal[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	McHorVer02_sse2(pSrc, iSrcOff+1, iSrcStride, vertical[:], 0,
		mcLumaSIMDTempStride, iWidth, iHeight)
	PixelAvg_sse2(pDst, iDstOff, iDstStride, horizontal[:], 0, mcLumaSIMDTempStride,
		vertical[:], 0, mcLumaSIMDTempStride, iWidth, iHeight)
}
