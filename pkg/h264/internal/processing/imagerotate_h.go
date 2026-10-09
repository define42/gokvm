package processing

// Port of codec/processing/src/imagerotate/imagerotate.h.

// ImageRotateFunc: pSrc/pDst are packed planes (no stride) starting at the
// plane origin.
type ImageRotateFunc func(pSrc []uint8, uiBytesPerPixel uint32, iWidth uint32, iHeight uint32,
	pDst []uint8)

type ImageRotateFuncPtr = ImageRotateFunc

type SImageRotateFuncs struct {
	pfImageRotate90D  ImageRotateFuncPtr
	pfImageRotate180D ImageRotateFuncPtr
	pfImageRotate270D ImageRotateFuncPtr
}

type CImageRotating struct {
	IStrategyBase
	m_pfRotateImage SImageRotateFuncs
	m_iCPUFlag      int32
}
