package processing

// Port of codec/processing/src/complexityanalysis/ComplexityAnalysis.h.

// GOMSadFunc: pSad8x8 is the [4] SAD group of one macroblock
// (C: pVaaCalcResults->pSad8x8[i] decayed to int32_t*).
type GOMSadFunc func(pGomSad *uint32, pGomForegroundBlockNum *int32, pSad8x8 *[4]int32,
	pBackgroundMbFlag uint8)

type PGOMSadFunc = GOMSadFunc

type CComplexityAnalysis struct {
	IStrategyBase
	m_pfGomSad                 PGOMSadFunc
	m_sComplexityAnalysisParam SComplexityAnalysisParam
}

//for screen content

type CComplexityAnalysisScreen struct {
	IStrategyBase
	m_pSadFunc                PSad16x16Func
	m_pIntraFunc              [2]GetIntraPredPtr
	m_ComplexityAnalysisParam SComplexityAnalysisScreenParam
}
