package processing

// Port of codec/processing/src/downsample/downsamplefuncs.cpp (_c functions only;
// the SIMD wrappers are not ported).

func DyadicBilinearDownsampler_c(pDst []uint8, iDstOff int, kiDstStride int32,
	pSrc []uint8, iSrcOff int, kiSrcStride int32,
	kiSrcWidth int32, kiSrcHeight int32) {
	pDstLine := iDstOff
	pSrcLine := iSrcOff
	kiSrcStridex2 := int(kiSrcStride << 1)
	kiDstWidth := kiSrcWidth >> 1
	kiDstHeight := kiSrcHeight >> 1

	for j := int32(0); j < kiDstHeight; j++ {
		for i := int32(0); i < kiDstWidth; i++ {
			kiSrcX := pSrcLine + int(i<<1)
			kiTempRow1 := (int32(pSrc[kiSrcX]) + int32(pSrc[kiSrcX+1]) + 1) >> 1
			kiTempRow2 := (int32(pSrc[kiSrcX+int(kiSrcStride)]) + int32(pSrc[kiSrcX+int(kiSrcStride)+1]) + 1) >> 1

			pDst[pDstLine+int(i)] = uint8((kiTempRow1 + kiTempRow2 + 1) >> 1)
		}
		pDstLine += int(kiDstStride)
		pSrcLine += kiSrcStridex2
	}
}

func DyadicBilinearQuarterDownsampler_c(pDst []uint8, iDstOff int, kiDstStride int32,
	pSrc []uint8, iSrcOff int, kiSrcStride int32,
	kiSrcWidth int32, kiSrcHeight int32) {
	pDstLine := iDstOff
	pSrcLine := iSrcOff
	kiSrcStridex4 := int(kiSrcStride << 2)
	kiDstWidth := kiSrcWidth >> 2
	kiDstHeight := kiSrcHeight >> 2

	for j := int32(0); j < kiDstHeight; j++ {
		for i := int32(0); i < kiDstWidth; i++ {
			kiSrcX := pSrcLine + int(i<<2)
			kiTempRow1 := (int32(pSrc[kiSrcX]) + int32(pSrc[kiSrcX+1]) + 1) >> 1
			kiTempRow2 := (int32(pSrc[kiSrcX+int(kiSrcStride)]) + int32(pSrc[kiSrcX+int(kiSrcStride)+1]) + 1) >> 1

			pDst[pDstLine+int(i)] = uint8((kiTempRow1 + kiTempRow2 + 1) >> 1)
		}
		pDstLine += int(kiDstStride)
		pSrcLine += kiSrcStridex4
	}
}

func DyadicBilinearOneThirdDownsampler_c(pDst []uint8, iDstOff int, kiDstStride int32,
	pSrc []uint8, iSrcOff int, kiSrcStride int32,
	kiSrcWidth int32, kiDstHeight int32) {
	pDstLine := iDstOff
	pSrcLine := iSrcOff
	kiSrcStridex3 := int(kiSrcStride * 3)
	kiDstWidth := kiSrcWidth / 3

	for j := int32(0); j < kiDstHeight; j++ {
		for i := int32(0); i < kiDstWidth; i++ {
			kiSrcX := pSrcLine + int(i*3)
			kiTempRow1 := (int32(pSrc[kiSrcX]) + int32(pSrc[kiSrcX+1]) + 1) >> 1
			kiTempRow2 := (int32(pSrc[kiSrcX+int(kiSrcStride)]) + int32(pSrc[kiSrcX+int(kiSrcStride)+1]) + 1) >> 1

			pDst[pDstLine+int(i)] = uint8((kiTempRow1 + kiTempRow2 + 1) >> 1)
		}
		pDstLine += int(kiDstStride)
		pSrcLine += kiSrcStridex3
	}
}

// downsampleScale computes WELS_ROUND ((float)kiSrc / (float)kiDst * kuiScale)
// with C float semantics (single precision quotient and product, then the
// 0.5 + x addition in double precision).
func downsampleScale(kiSrc, kiDst int32, kuiScale uint32) int32 {
	fRatio := float32(kiSrc) / float32(kiDst)
	fScaled := float32(fRatio * float32(kuiScale))
	return WELS_ROUND(float64(fScaled))
}

func GeneralBilinearFastDownsampler_c(pDst []uint8, iDstOff int, kiDstStride int32, kiDstWidth int32,
	kiDstHeight int32,
	pSrc []uint8, iSrcOff int, kiSrcStride int32, kiSrcWidth int32, kiSrcHeight int32) {
	const kuiScaleBitWidth, kuiScaleBitHeight uint32 = 16, 15
	const kuiScaleWidth, kuiScaleHeight uint32 = (1 << kuiScaleBitWidth), (1 << kuiScaleBitHeight)
	fScalex := downsampleScale(kiSrcWidth, kiDstWidth, kuiScaleWidth)
	fScaley := downsampleScale(kiSrcHeight, kiDstHeight, kuiScaleHeight)
	var x uint32
	var iYInverse, iXInverse int32

	pByDst := iDstOff
	pByLineDst := iDstOff

	iYInverse = 1 << (kuiScaleBitHeight - 1)
	for i := int32(0); i < kiDstHeight-1; i++ {
		iYy := iYInverse >> kuiScaleBitHeight
		fv := iYInverse & int32(kuiScaleHeight-1)

		pBySrc := iSrcOff + int(iYy*kiSrcStride)

		pByDst = pByLineDst
		iXInverse = 1 << (kuiScaleBitWidth - 1)
		for j := int32(0); j < kiDstWidth-1; j++ {
			iXx := iXInverse >> kuiScaleBitWidth
			iFu := iXInverse & int32(kuiScaleWidth-1)

			pByCurrent := pBySrc + int(iXx)
			var a, b, c, d uint8

			a = pSrc[pByCurrent]
			b = pSrc[pByCurrent+1]
			c = pSrc[pByCurrent+int(kiSrcStride)]
			d = pSrc[pByCurrent+int(kiSrcStride)+1]

			x = ((kuiScaleWidth - 1 - uint32(iFu)) * (kuiScaleHeight - 1 - uint32(fv)) >> kuiScaleBitWidth) * uint32(a)
			x += (uint32(iFu) * (kuiScaleHeight - 1 - uint32(fv)) >> kuiScaleBitWidth) * uint32(b)
			x += ((kuiScaleWidth - 1 - uint32(iFu)) * uint32(fv) >> kuiScaleBitWidth) * uint32(c)
			x += (uint32(iFu) * uint32(fv) >> kuiScaleBitWidth) * uint32(d)
			x >>= (kuiScaleBitHeight - 1)
			x += 1
			x >>= 1
			x = WELS_CLAMP(x, 0, 255)
			pDst[pByDst] = uint8(x)
			pByDst++

			iXInverse += fScalex
		}
		pDst[pByDst] = pSrc[pBySrc+int(iXInverse>>kuiScaleBitWidth)]
		pByLineDst += int(kiDstStride)
		iYInverse += fScaley
	}

	// last row special
	{
		iYy := iYInverse >> kuiScaleBitHeight
		pBySrc := iSrcOff + int(iYy*kiSrcStride)

		pByDst = pByLineDst
		iXInverse = 1 << (kuiScaleBitWidth - 1)
		for j := int32(0); j < kiDstWidth; j++ {
			iXx := iXInverse >> kuiScaleBitWidth
			pDst[pByDst] = pSrc[pBySrc+int(iXx)]
			pByDst++

			iXInverse += fScalex
		}
	}
}

func GeneralBilinearAccurateDownsampler_c(pDst []uint8, iDstOff int, kiDstStride int32, kiDstWidth int32,
	kiDstHeight int32,
	pSrc []uint8, iSrcOff int, kiSrcStride int32, kiSrcWidth int32, kiSrcHeight int32) {
	const kiScaleBit int32 = 15
	const kiScale int32 = (1 << kiScaleBit)
	iScalex := downsampleScale(kiSrcWidth, kiDstWidth, uint32(kiScale))
	iScaley := downsampleScale(kiSrcHeight, kiDstHeight, uint32(kiScale))
	var x int64
	var iYInverse, iXInverse int32

	pByDst := iDstOff
	pByLineDst := iDstOff

	iYInverse = 1 << (kiScaleBit - 1)
	for i := int32(0); i < kiDstHeight-1; i++ {
		iYy := iYInverse >> kiScaleBit
		iFv := iYInverse & (kiScale - 1)

		pBySrc := iSrcOff + int(iYy*kiSrcStride)

		pByDst = pByLineDst
		iXInverse = 1 << (kiScaleBit - 1)
		for j := int32(0); j < kiDstWidth-1; j++ {
			iXx := iXInverse >> kiScaleBit
			iFu := iXInverse & (kiScale - 1)

			pByCurrent := pBySrc + int(iXx)
			var a, b, c, d uint8

			a = pSrc[pByCurrent]
			b = pSrc[pByCurrent+1]
			c = pSrc[pByCurrent+int(kiSrcStride)]
			d = pSrc[pByCurrent+int(kiSrcStride)+1]

			x = (int64(kiScale-1-iFu)*int64(kiScale-1-iFv)*int64(a) + int64(iFu)*int64(kiScale-1-iFv)*int64(b) +
				int64(kiScale-1-iFu)*int64(iFv)*int64(c) +
				int64(iFu)*int64(iFv)*int64(d) + int64(int32(1)<<(2*kiScaleBit-1))) >> (2 * kiScaleBit)
			x = WELS_CLAMP(x, 0, 255)
			pDst[pByDst] = uint8(x)
			pByDst++

			iXInverse += iScalex
		}
		pDst[pByDst] = pSrc[pBySrc+int(iXInverse>>kiScaleBit)]
		pByLineDst += int(kiDstStride)
		iYInverse += iScaley
	}

	// last row special
	{
		iYy := iYInverse >> kiScaleBit
		pBySrc := iSrcOff + int(iYy*kiSrcStride)

		pByDst = pByLineDst
		iXInverse = 1 << (kiScaleBit - 1)
		for j := int32(0); j < kiDstWidth; j++ {
			iXx := iXInverse >> kiScaleBit
			pDst[pByDst] = pSrc[pBySrc+int(iXx)]
			pByDst++

			iXInverse += iScalex
		}
	}
}
