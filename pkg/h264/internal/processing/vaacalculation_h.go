package processing

// Port of codec/processing/src/vaacalc/vaacalculation.h.
//
// pCurData/pRefData are (slice, offset) plane pointers. The int32_t* result
// arrays of C (pSad8x8, pSd8x8, pMad8x8 with "every 4 in the same 16x16 get
// together") are [][4] slices indexed by macroblock: C index
// (mb_index << 2) + k is Go [mb_index][k].

type VAACalcSadBgdFunc func(pCurData []uint8, iCurOff int, pRefData []uint8, iRefOff int, iPicWidth int32,
	iPicHeight int32, iPicStride int32,
	pFrameSad *int32, pSad8x8 [][4]int32, pSd8x8 [][4]int32, pMad8x8 [][4]uint8)

type VAACalcSadSsdBgdFunc func(pCurData []uint8, iCurOff int, pRefData []uint8, iRefOff int, iPicWidth int32,
	iPicHeight int32, iPicStride int32,
	pFrameSad *int32, pSad8x8 [][4]int32, pSum16x16 []int32, pSumSquare16x16 []int32,
	pSsd16x16 []int32, pSd8x8 [][4]int32, pMad8x8 [][4]uint8)

type VAACalcSadFunc func(pCurData []uint8, iCurOff int, pRefData []uint8, iRefOff int, iPicWidth int32, iPicHeight int32,
	iPicStride int32,
	pFrameSad *int32, pSad8x8 [][4]int32)

type VAACalcSadVarFunc func(pCurData []uint8, iCurOff int, pRefData []uint8, iRefOff int, iPicWidth int32,
	iPicHeight int32, iPicStride int32,
	pFrameSad *int32, pSad8x8 [][4]int32, pSum16x16 []int32, pSumSquare16x16 []int32)

type VAACalcSadSsdFunc func(pCurData []uint8, iCurOff int, pRefData []uint8, iRefOff int, iPicWidth int32,
	iPicHeight int32, iPicStride int32,
	pFrameSad *int32, pSad8x8 [][4]int32, pSum16x16 []int32, pSumSquare16x16 []int32, pSsd16x16 []int32)

type PVAACalcSadBgdFunc = VAACalcSadBgdFunc
type PVAACalcSadSsdBgdFunc = VAACalcSadSsdBgdFunc
type PVAACalcSadFunc = VAACalcSadFunc
type PVAACalcSadVarFunc = VAACalcSadVarFunc
type PVAACalcSadSsdFunc = VAACalcSadSsdFunc

type SVaaFuncs struct {
	pfVAACalcSadBgd    PVAACalcSadBgdFunc
	pfVAACalcSadSsdBgd PVAACalcSadSsdBgdFunc
	pfVAACalcSad       PVAACalcSadFunc
	pfVAACalcSadVar    PVAACalcSadVarFunc
	pfVAACalcSadSsd    PVAACalcSadSsdFunc
}

type CVAACalculation struct {
	IStrategyBase
	m_sVaaFuncs  SVaaFuncs
	m_iCPUFlag   int32
	m_sCalcParam SVAACalcParam
}
