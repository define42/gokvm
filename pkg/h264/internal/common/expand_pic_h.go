package common

// Port of codec/common/inc/expand_pic.h.

const (
	PADDING_LENGTH        = 32 // reference extension
	CHROMA_PADDING_LENGTH = 16 // chroma reference extension
)

// PExpandPictureFunc pads a picture plane whose top-left sample is
// pDst[iDstOff] (the slice must cover the padding area).
type PExpandPictureFunc func(pDst []uint8, iDstOff int, kiStride int32, kiPicW int32, kiPicH int32)

// SExpandPicFunc mirrors TagExpandPicFunc.
type SExpandPicFunc struct {
	PfExpandLumaPicture   PExpandPictureFunc
	PfExpandChromaPicture [2]PExpandPictureFunc
}
