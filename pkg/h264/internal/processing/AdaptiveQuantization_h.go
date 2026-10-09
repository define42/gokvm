package processing

// Port of codec/processing/src/adaptivequantization/AdaptiveQuantization.h.

type VarFunc func(pRefY []uint8, iRefYOff int, iRefStrideY int32, pSrc []uint8, iSrcOff int, iSrcStrideY int32,
	pMotionTexture *SMotionTextureUnit)

type PVarFunc = VarFunc

type CAdaptiveQuantization struct {
	IStrategyBase
	m_pfVar               PVarFunc
	m_CPUFlag             int32
	m_sAdaptiveQuantParam SAdaptiveQuantizationParam
}
