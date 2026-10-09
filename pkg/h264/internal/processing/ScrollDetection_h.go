package processing

// Port of codec/processing/src/scrolldetection/ScrollDetection.h.

type CScrollDetection struct {
	IStrategyBase
	m_sScrollDetectionParam SScrollDetectionParam
}

func NewCScrollDetection(iCpuFlag int32) *CScrollDetection {
	s := &CScrollDetection{}
	s.initIStrategyBase()
	s.m_eMethod = METHOD_SCROLL_DETECTION
	s.m_sScrollDetectionParam = SScrollDetectionParam{}
	return s
}
