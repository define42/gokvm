package processing

// Port of codec/processing/src/downsample/downsample.h.

type HalveDownsampleFunc func(pDst []uint8, iDstOff int, kiDstStride int32,
	pSrc []uint8, iSrcOff int, kiSrcStride int32,
	kiSrcWidth int32, kiSrcHeight int32)

type SpecificDownsampleFunc func(pDst []uint8, iDstOff int, kiDstStride int32,
	pSrc []uint8, iSrcOff int, kiSrcStride int32,
	kiSrcWidth int32, kiHeight int32)

type GeneralDownsampleFunc func(pDst []uint8, iDstOff int, kiDstStride int32, kiDstWidth int32,
	kiDstHeight int32,
	pSrc []uint8, iSrcOff int, kiSrcStride int32, kiSrcWidth int32, kiSrcHeight int32)

type PHalveDownsampleFunc = HalveDownsampleFunc
type PSpecificDownsampleFunc = SpecificDownsampleFunc
type PGeneralDownsampleFunc = GeneralDownsampleFunc

type SDownsampleFuncs struct {
	pfHalfAverageWidthx32 PHalveDownsampleFunc
	pfHalfAverageWidthx16 PHalveDownsampleFunc
	pfOneThirdDownsampler PSpecificDownsampleFunc
	pfQuarterDownsampler  PSpecificDownsampleFunc
	pfGeneralRatioLuma    PGeneralDownsampleFunc
	pfGeneralRatioChroma  PGeneralDownsampleFunc
}

type CDownsampling struct {
	IStrategyBase
	m_pfDownsample    SDownsampleFuncs
	m_iCPUFlag        int32
	m_pSampleBuffer   [2][3][]uint8
	m_bNoSampleBuffer bool
}
