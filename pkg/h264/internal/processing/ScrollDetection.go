package processing

// Port of codec/processing/src/scrolldetection/ScrollDetection.cpp.

func (s *CScrollDetection) Process(iType int32, pSrcPixMap *SPixMap, pRefPixMap *SPixMap) EResult {
	if pRefPixMap.PPixel[0] == nil || pSrcPixMap.PPixel[0] == nil ||
		pRefPixMap.SRect.IRectWidth != pSrcPixMap.SRect.IRectWidth ||
		pRefPixMap.SRect.IRectHeight != pSrcPixMap.SRect.IRectHeight {
		return RET_INVALIDPARAM
	}

	if !s.m_sScrollDetectionParam.BMaskInfoAvailable {
		s.ScrollDetectionWithoutMask(pSrcPixMap, pRefPixMap)
	} else {
		s.ScrollDetectionWithMask(pSrcPixMap, pRefPixMap)
	}

	return RET_SUCCESS
}

// Set takes a *SScrollDetectionParam.
func (s *CScrollDetection) Set(iType int32, pParam any) EResult {
	p, ok := pParam.(*SScrollDetectionParam)
	if !ok || p == nil {
		return RET_INVALIDPARAM
	}
	s.m_sScrollDetectionParam = *p
	return RET_SUCCESS
}

// Get takes a *SScrollDetectionParam.
func (s *CScrollDetection) Get(iType int32, pParam any) EResult {
	p, ok := pParam.(*SScrollDetectionParam)
	if !ok || p == nil {
		return RET_INVALIDPARAM
	}
	*p = s.m_sScrollDetectionParam
	return RET_SUCCESS
}

func (s *CScrollDetection) ScrollDetectionWithMask(pSrcPixMap *SPixMap, pRefPixMap *SPixMap) {
	var iStartX, iStartY, iWidth, iHeight int32

	iStartX = s.m_sScrollDetectionParam.SMaskRect.IRectLeft
	iStartY = s.m_sScrollDetectionParam.SMaskRect.IRectTop
	iWidth = s.m_sScrollDetectionParam.SMaskRect.IRectWidth
	iHeight = s.m_sScrollDetectionParam.SMaskRect.IRectHeight

	iWidth /= 2
	iStartX += iWidth / 2

	s.m_sScrollDetectionParam.IScrollMvX = 0
	s.m_sScrollDetectionParam.IScrollMvY = 0
	s.m_sScrollDetectionParam.BScrollDetectFlag = false

	if iStartX >= 0 && iWidth > MINIMUM_DETECT_WIDTH && iHeight > 2*CHECK_OFFSET {
		ScrollDetectionCore(pSrcPixMap, pRefPixMap, iWidth, iHeight, iStartX, iStartY, &s.m_sScrollDetectionParam)
	}
}

func (s *CScrollDetection) ScrollDetectionWithoutMask(pSrcPixMap *SPixMap, pRefPixMap *SPixMap) {
	var iStartX, iStartY, iWidth, iHeight int32

	kiPicBorderWidth := pSrcPixMap.SRect.IRectHeight >> 4
	kiRegionWidth := (pSrcPixMap.SRect.IRectWidth - (kiPicBorderWidth << 1)) / 3
	kiRegionHeight := (pSrcPixMap.SRect.IRectHeight * 7) >> 3
	kiHieghtStride := pSrcPixMap.SRect.IRectHeight * 5 / 24

	for i := int32(0); i < REGION_NUMBER; i++ {
		iStartX = kiPicBorderWidth + (i%3)*kiRegionWidth
		iStartY = -pSrcPixMap.SRect.IRectHeight*7/48 + (i/3)*(kiHieghtStride)
		iWidth = kiRegionWidth
		iHeight = kiRegionHeight

		iWidth /= 2
		iStartX += iWidth / 2

		ScrollDetectionCore(pSrcPixMap, pRefPixMap, iWidth, iHeight, iStartX, iStartY, &s.m_sScrollDetectionParam)

		if s.m_sScrollDetectionParam.BScrollDetectFlag && s.m_sScrollDetectionParam.IScrollMvY != 0 {
			break
		}
	}
}
