package processing

// Port of codec/processing/src/imagerotate/imagerotate.cpp.

///////////////////////////////////////////////////////////////////////////////////////////////////////////////

func NewCImageRotating(iCpuFlag int32) *CImageRotating {
	r := &CImageRotating{}
	r.initIStrategyBase()
	r.m_iCPUFlag = iCpuFlag
	r.m_eMethod = METHOD_IMAGE_ROTATE
	r.m_pfRotateImage = SImageRotateFuncs{}
	r.InitImageRotateFuncs(&r.m_pfRotateImage, r.m_iCPUFlag)
	return r
}

func (r *CImageRotating) InitImageRotateFuncs(sImageRotateFuncs *SImageRotateFuncs, iCpuFlag int32) {
	sImageRotateFuncs.pfImageRotate90D = ImageRotate90D_c
	sImageRotateFuncs.pfImageRotate180D = ImageRotate180D_c
	sImageRotateFuncs.pfImageRotate270D = ImageRotate270D_c
}

func (r *CImageRotating) ProcessImageRotate(iType int32, pSrc []uint8, uiBytesPerPixel uint32, iWidth uint32,
	iHeight uint32, pDst []uint8) EResult {
	if iType == 90 {
		r.m_pfRotateImage.pfImageRotate90D(pSrc, uiBytesPerPixel, iWidth, iHeight, pDst)
	} else if iType == 180 {
		r.m_pfRotateImage.pfImageRotate180D(pSrc, uiBytesPerPixel, iWidth, iHeight, pDst)
	} else if iType == 270 {
		r.m_pfRotateImage.pfImageRotate270D(pSrc, uiBytesPerPixel, iWidth, iHeight, pDst)
	} else {
		return RET_NOTSUPPORTED
	}
	return RET_SUCCESS
}

// plane returns the C pointer pPixel[i] as a slice starting at the origin.
func (p *SPixMap) plane(i int) []uint8 {
	if p.PPixel[i] == nil {
		return nil
	}
	return p.PPixel[i][p.IPixelOff[i]:]
}

func (r *CImageRotating) Process(iType int32, pSrc *SPixMap, pDst *SPixMap) EResult {
	eReturn := RET_INVALIDPARAM

	if (pSrc.EFormat == VIDEO_FORMAT_RGBA) ||
		(pSrc.EFormat == VIDEO_FORMAT_BGRA) ||
		(pSrc.EFormat == VIDEO_FORMAT_ABGR) ||
		(pSrc.EFormat == VIDEO_FORMAT_ARGB) {
		eReturn = r.ProcessImageRotate(iType, pSrc.plane(0), uint32(pSrc.ISizeInBits*8), uint32(pSrc.SRect.IRectWidth),
			uint32(pSrc.SRect.IRectHeight), pDst.plane(0))
	} else if pSrc.EFormat == VIDEO_FORMAT_I420 {
		r.ProcessImageRotate(iType, pSrc.plane(0), uint32(pSrc.ISizeInBits*8), uint32(pSrc.SRect.IRectWidth),
			uint32(pSrc.SRect.IRectHeight), pDst.plane(0))
		r.ProcessImageRotate(iType, pSrc.plane(1), uint32(pSrc.ISizeInBits*8), uint32(pSrc.SRect.IRectWidth>>1),
			uint32(pSrc.SRect.IRectHeight>>1), pDst.plane(1))
		eReturn = r.ProcessImageRotate(iType, pSrc.plane(2), uint32(pSrc.ISizeInBits*8), uint32(pSrc.SRect.IRectWidth>>1),
			uint32(pSrc.SRect.IRectHeight>>1), pDst.plane(2))
	} else {
		eReturn = RET_NOTSUPPORTED
	}

	return eReturn
}
