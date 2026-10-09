package processing

// Port of codec/processing/src/scenechangedetection/SceneChangeDetection.cpp
// and of the CSceneChangeDetection<T> member functions from
// SceneChangeDetection.h.

func newCSceneChangeDetection(eMethod EMethods, iCpuFlag int32) *CSceneChangeDetection {
	s := &CSceneChangeDetection{}
	s.initIStrategyBase()
	switch eMethod {
	case METHOD_SCENE_CHANGE_DETECTION_VIDEO:
		s.m_cDetector = NewCSceneChangeDetectorVideo(&s.m_sSceneChangeParam, iCpuFlag)
	default:
		s.m_cDetector = NewCSceneChangeDetectorScreen(&s.m_sSceneChangeParam, iCpuFlag)
	}
	s.m_eMethod = eMethod
	s.m_sSceneChangeParam = SSceneChangeResult{}
	return s
}

func BuildSceneChangeDetection(eMethod EMethods, iCpuFlag int32) IStrategy {
	switch eMethod {
	case METHOD_SCENE_CHANGE_DETECTION_VIDEO:
		return newCSceneChangeDetection(eMethod, iCpuFlag)
	case METHOD_SCENE_CHANGE_DETECTION_SCREEN:
		return newCSceneChangeDetection(eMethod, iCpuFlag)
	default:
		// not support yet
		return nil
	}
}

func (s *CSceneChangeDetection) Process(iType int32, pSrcPixMap *SPixMap, pRefPixMap *SPixMap) EResult {
	eReturn := RET_INVALIDPARAM

	s.m_sLocalParam.iWidth = pSrcPixMap.SRect.IRectWidth
	s.m_sLocalParam.iHeight = pSrcPixMap.SRect.IRectHeight
	s.m_sLocalParam.iBlock8x8Width = s.m_sLocalParam.iWidth >> 3
	s.m_sLocalParam.iBlock8x8Height = s.m_sLocalParam.iHeight >> 3
	s.m_sLocalParam.pRefY = pRefPixMap.PPixel[0]
	s.m_sLocalParam.iRefYOff = pRefPixMap.IPixelOff[0]
	s.m_sLocalParam.pCurY = pSrcPixMap.PPixel[0]
	s.m_sLocalParam.iCurYOff = pSrcPixMap.IPixelOff[0]
	s.m_sLocalParam.iRefStride = pRefPixMap.IStride[0]
	s.m_sLocalParam.iCurStride = pSrcPixMap.IStride[0]
	s.m_sLocalParam.pStaticBlockIdc = s.m_sSceneChangeParam.PStaticBlockIdc

	iBlock8x8Num := s.m_sLocalParam.iBlock8x8Width * s.m_sLocalParam.iBlock8x8Height
	// (int32_t)(float * int + 0.5f + PESN): float product and float sum, then
	// the double PESN addition. Explicit conversions prevent FMA fusion.
	iSceneChangeThresholdLarge := int32(float64(float32(s.m_cDetector.GetSceneChangeMotionRatioLarge()*float32(iBlock8x8Num))+0.5) + PESN)
	iSceneChangeThresholdMedium := int32(float64(float32(s.m_cDetector.GetSceneChangeMotionRatioMedium()*float32(iBlock8x8Num))+0.5) + PESN)

	s.m_sSceneChangeParam.IMotionBlockNum = 0
	s.m_sSceneChangeParam.IFrameComplexity = 0
	s.m_sSceneChangeParam.ESceneChangeIdc = SIMILAR_SCENE

	s.m_cDetector.detect(&s.m_sLocalParam)

	if s.m_sSceneChangeParam.IMotionBlockNum >= iSceneChangeThresholdLarge {
		s.m_sSceneChangeParam.ESceneChangeIdc = LARGE_CHANGED_SCENE
	} else if s.m_sSceneChangeParam.IMotionBlockNum >= iSceneChangeThresholdMedium {
		s.m_sSceneChangeParam.ESceneChangeIdc = MEDIUM_CHANGED_SCENE
	}

	eReturn = RET_SUCCESS

	return eReturn
}

// Get takes a *SSceneChangeResult.
func (s *CSceneChangeDetection) Get(iType int32, pParam any) EResult {
	p, ok := pParam.(*SSceneChangeResult)
	if !ok || p == nil {
		return RET_INVALIDPARAM
	}
	*p = s.m_sSceneChangeParam
	return RET_SUCCESS
}

// Set takes a *SSceneChangeResult.
func (s *CSceneChangeDetection) Set(iType int32, pParam any) EResult {
	p, ok := pParam.(*SSceneChangeResult)
	if !ok || p == nil {
		return RET_INVALIDPARAM
	}
	s.m_sSceneChangeParam = *p
	return RET_SUCCESS
}
