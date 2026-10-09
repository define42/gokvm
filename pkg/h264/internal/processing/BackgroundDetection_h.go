package processing

// Port of codec/processing/src/backgrounddetection/BackgroundDetection.h.

type SBackgroundOU struct {
	iBackgroundFlag int32
	iSAD            int32
	iSD             int32
	iMAD            int32
	iMinSubMad      int32
	iMaxDiffSubSd   int32
}

// vBGDParam: plane pointers are (slice, offset) pairs; pOU_array is the
// OU array (C pointer arithmetic on it becomes indices).
type vBGDParam struct {
	pCur              [3][]uint8
	iCurOff           [3]int
	pRef              [3][]uint8
	iRefOff           [3]int
	iBgdWidth         int32
	iBgdHeight        int32
	iStride           [3]int32
	pOU_array         []SBackgroundOU
	pBackgroundMbFlag []int8
	pCalcRes          *SVAACalcResult
}

type CBackgroundDetection struct {
	IStrategyBase
	m_BgdParam vBGDParam

	m_iLargestFrameSize int32
}
