package processing

// Port of codec/processing/src/scenechangedetection/SceneChangeDetection.h.

const (
	HIGH_MOTION_BLOCK_THRESHOLD            = 320
	SCENE_CHANGE_MOTION_RATIO_LARGE_VIDEO  = float32(0.85)
	SCENE_CHANGE_MOTION_RATIO_MEDIUM       = float32(0.50)
	SCENE_CHANGE_MOTION_RATIO_LARGE_SCREEN = float32(0.80)
)

type SLocalParam struct {
	iWidth          int32
	iHeight         int32
	iBlock8x8Width  int32
	iBlock8x8Height int32
	pRefY           []uint8
	iRefYOff        int
	pCurY           []uint8
	iCurYOff        int
	iRefStride      int32
	iCurStride      int32
	pStaticBlockIdc []uint8 // advanced (re-sliced) as the C pointer is incremented
}

// sceneChangeDetector is the template parameter T of CSceneChangeDetection
// (CSceneChangeDetectorVideo or CSceneChangeDetectorScreen).
type sceneChangeDetector interface {
	detect(sLocalParam *SLocalParam) // operator()
	GetSceneChangeMotionRatioLarge() float32
	GetSceneChangeMotionRatioMedium() float32
}

type CSceneChangeDetectorVideo struct {
	m_pfSad                         SadFuncPtr
	m_sParam                        *SSceneChangeResult
	m_fSceneChangeMotionRatioLarge  float32
	m_fSceneChangeMotionRatioMedium float32
}

func NewCSceneChangeDetectorVideo(sParam *SSceneChangeResult, iCpuFlag int32) *CSceneChangeDetectorVideo {
	v := &CSceneChangeDetectorVideo{}
	v.initCSceneChangeDetectorVideo(sParam, iCpuFlag)
	return v
}

func (v *CSceneChangeDetectorVideo) initCSceneChangeDetectorVideo(sParam *SSceneChangeResult, iCpuFlag int32) {
	v.m_sParam = sParam
	v.m_pfSad = WelsSampleSad8x8_c

	v.m_fSceneChangeMotionRatioLarge = SCENE_CHANGE_MOTION_RATIO_LARGE_VIDEO
	v.m_fSceneChangeMotionRatioMedium = SCENE_CHANGE_MOTION_RATIO_MEDIUM
}

func (v *CSceneChangeDetectorVideo) detect(sLocalParam *SLocalParam) {
	var iRefRowStride, iCurRowStride int32
	pRefY := sLocalParam.iRefYOff
	pCurY := sLocalParam.iCurYOff
	var pRefTmp, pCurTmp int

	iRefRowStride = sLocalParam.iRefStride << 3
	iCurRowStride = sLocalParam.iCurStride << 3

	for j := int32(0); j < sLocalParam.iBlock8x8Height; j++ {
		pRefTmp = pRefY
		pCurTmp = pCurY
		for i := int32(0); i < sLocalParam.iBlock8x8Width; i++ {
			iSad := v.m_pfSad(sLocalParam.pCurY, pCurTmp, sLocalParam.iCurStride, sLocalParam.pRefY, pRefTmp, sLocalParam.iRefStride)
			if iSad > HIGH_MOTION_BLOCK_THRESHOLD {
				v.m_sParam.IMotionBlockNum++
			}
			pRefTmp += 8
			pCurTmp += 8
		}
		pRefY += int(iRefRowStride)
		pCurY += int(iCurRowStride)
	}
}

func (v *CSceneChangeDetectorVideo) GetSceneChangeMotionRatioLarge() float32 {
	return v.m_fSceneChangeMotionRatioLarge
}
func (v *CSceneChangeDetectorVideo) GetSceneChangeMotionRatioMedium() float32 {
	return v.m_fSceneChangeMotionRatioMedium
}

type CSceneChangeDetectorScreen struct {
	CSceneChangeDetectorVideo
}

func NewCSceneChangeDetectorScreen(sParam *SSceneChangeResult, iCpuFlag int32) *CSceneChangeDetectorScreen {
	s := &CSceneChangeDetectorScreen{}
	s.initCSceneChangeDetectorVideo(sParam, iCpuFlag)
	s.m_fSceneChangeMotionRatioLarge = SCENE_CHANGE_MOTION_RATIO_LARGE_SCREEN
	s.m_fSceneChangeMotionRatioMedium = SCENE_CHANGE_MOTION_RATIO_MEDIUM
	return s
}

func (s *CSceneChangeDetectorScreen) detect(sLocalParam *SLocalParam) {
	bScrollDetectFlag := s.m_sParam.SScrollResult.BScrollDetectFlag
	iScrollMvX := s.m_sParam.SScrollResult.IScrollMvX
	iScrollMvY := s.m_sParam.SScrollResult.IScrollMvY

	var iRefRowStride, iCurRowStride int32
	pRefY := sLocalParam.iRefYOff
	pCurY := sLocalParam.iCurYOff
	var pRefTmp, pCurTmp int
	iWidth := sLocalParam.iWidth
	iHeight := sLocalParam.iHeight

	iRefRowStride = sLocalParam.iRefStride << 3
	iCurRowStride = sLocalParam.iCurStride << 3

	for j := int32(0); j < sLocalParam.iBlock8x8Height; j++ {
		pRefTmp = pRefY
		pCurTmp = pCurY
		for i := int32(0); i < sLocalParam.iBlock8x8Width; i++ {
			iBlockPointX := i << 3
			iBlockPointY := j << 3
			uiBlockIdcTmp := uint8(NO_STATIC)
			iSad := s.m_pfSad(sLocalParam.pCurY, pCurTmp, sLocalParam.iCurStride, sLocalParam.pRefY, pRefTmp, sLocalParam.iRefStride)
			if iSad == 0 {
				uiBlockIdcTmp = uint8(COLLOCATED_STATIC)
			} else if bScrollDetectFlag && (iScrollMvX == 0 || iScrollMvY == 0) && (iBlockPointX+iScrollMvX >= 0) &&
				(iBlockPointX+iScrollMvX <= iWidth-8) &&
				(iBlockPointY+iScrollMvY >= 0) && (iBlockPointY+iScrollMvY <= iHeight-8) {
				pRefTmpScroll := pRefTmp + int(iScrollMvY*sLocalParam.iRefStride+iScrollMvX)
				iSadScroll := s.m_pfSad(sLocalParam.pCurY, pCurTmp, sLocalParam.iCurStride, sLocalParam.pRefY, pRefTmpScroll, sLocalParam.iRefStride)

				if iSadScroll == 0 {
					uiBlockIdcTmp = uint8(SCROLLED_STATIC)
				} else {
					s.m_sParam.IFrameComplexity += int64(iSad)
					if iSad > HIGH_MOTION_BLOCK_THRESHOLD {
						s.m_sParam.IMotionBlockNum++
					}
				}
			} else {
				s.m_sParam.IFrameComplexity += int64(iSad)
				if iSad > HIGH_MOTION_BLOCK_THRESHOLD {
					s.m_sParam.IMotionBlockNum++
				}
			}
			sLocalParam.pStaticBlockIdc[0] = uiBlockIdcTmp
			sLocalParam.pStaticBlockIdc = sLocalParam.pStaticBlockIdc[1:]
			pRefTmp += 8
			pCurTmp += 8
		}
		pRefY += int(iRefRowStride)
		pCurY += int(iCurRowStride)
	}
}

// CSceneChangeDetection is the template class CSceneChangeDetection<T>; the
// detector T is held through the sceneChangeDetector interface.
type CSceneChangeDetection struct {
	IStrategyBase
	m_sSceneChangeParam SSceneChangeResult
	m_sLocalParam       SLocalParam
	m_cDetector         sceneChangeDetector
}
