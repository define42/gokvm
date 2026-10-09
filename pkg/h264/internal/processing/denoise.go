package processing

// Port of codec/processing/src/denoise/denoise.cpp.

func CALC_BI_STRIDE(iWidth, iBitcount int32) int32 {
	return (((iWidth * iBitcount) + 31) &^ 31) >> 3
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////

func NewCDenoiser(iCpuFlag int32) *CDenoiser {
	d := &CDenoiser{}
	d.initIStrategyBase()
	d.m_CPUFlag = iCpuFlag
	d.m_eMethod = METHOD_DENOISE
	d.m_pfDenoise = SDenoiseFuncs{}

	d.m_uiSpaceRadius = DENOISE_GRAY_RADIUS
	d.m_fSigmaGrey = DENOISE_GRAY_SIGMA
	d.m_uiType = DENOISE_ALL_COMPONENT
	d.InitDenoiseFunc(&d.m_pfDenoise, d.m_CPUFlag)
	return d
}

func (d *CDenoiser) InitDenoiseFunc(denoiser *SDenoiseFuncs, iCpuFlag int32) {
	denoiser.pfBilateralLumaFilter8 = BilateralLumaFilter8_c
	denoiser.pfWaverageChromaFilter8 = WaverageChromaFilter8_c
}

func (d *CDenoiser) Process(iType int32, pSrc *SPixMap, dst *SPixMap) EResult {
	pSrcY := pSrc.PPixel[0]
	pSrcU := pSrc.PPixel[1]
	pSrcV := pSrc.PPixel[2]
	if pSrcY == nil || pSrcU == nil || pSrcV == nil {
		return RET_INVALIDPARAM
	}

	iWidthY := pSrc.SRect.IRectWidth
	iHeightY := pSrc.SRect.IRectHeight
	iWidthUV := iWidthY >> 1
	iHeightUV := iHeightY >> 1

	if d.m_uiType&DENOISE_Y_COMPONENT != 0 {
		d.BilateralDenoiseLuma(pSrcY, pSrc.IPixelOff[0], iWidthY, iHeightY, pSrc.IStride[0])
	}

	if d.m_uiType&DENOISE_U_COMPONENT != 0 {
		d.WaverageDenoiseChroma(pSrcU, pSrc.IPixelOff[1], iWidthUV, iHeightUV, pSrc.IStride[1])
	}

	if d.m_uiType&DENOISE_V_COMPONENT != 0 {
		d.WaverageDenoiseChroma(pSrcV, pSrc.IPixelOff[2], iWidthUV, iHeightUV, pSrc.IStride[2])
	}

	return RET_SUCCESS
}

func (d *CDenoiser) BilateralDenoiseLuma(pSrcY []uint8, iSrcYOff int, iWidth int32, iHeight int32, iStride int32) {
	var w int32
	kiRadius := int32(d.m_uiSpaceRadius)

	iSrcYOff += int(kiRadius * iStride)
	for h := kiRadius; h < iHeight-kiRadius; h++ {
		for w = kiRadius; w < iWidth-kiRadius-TAIL_OF_LINE8; w += 8 {
			d.m_pfDenoise.pfBilateralLumaFilter8(pSrcY, iSrcYOff+int(w), iStride)
		}
		for ; w < iWidth-kiRadius; w++ {
			Gauss3x3Filter(pSrcY, iSrcYOff+int(w), iStride)
		}
		iSrcYOff += int(iStride)
	}
}

func (d *CDenoiser) WaverageDenoiseChroma(pSrcUV []uint8, iSrcUVOff int, iWidth int32, iHeight int32, iStride int32) {
	var w int32

	iSrcUVOff += int(UV_WINDOWS_RADIUS * iStride)
	for h := int32(UV_WINDOWS_RADIUS); h < iHeight-UV_WINDOWS_RADIUS; h++ {
		for w = UV_WINDOWS_RADIUS; w < iWidth-UV_WINDOWS_RADIUS-TAIL_OF_LINE8; w += 8 {
			d.m_pfDenoise.pfWaverageChromaFilter8(pSrcUV, iSrcUVOff+int(w), iStride)
		}

		for ; w < iWidth-UV_WINDOWS_RADIUS; w++ {
			Gauss3x3Filter(pSrcUV, iSrcUVOff+int(w), iStride)
		}
		iSrcUVOff += int(iStride)
	}
}
