// Port of codec/decoder/core/inc/pic_queue.h.

package decoder

const PICTURE_RESOLUTION_ALIGNMENT = 32

// SPicBuff (TagPicBuff). The C member `CMemoryAlign* pMa` is dropped
// (allocation is done with make/new in Go).
type SPicBuff struct {
	ppPic       []*SPicture // C: PPicture* (array of iCapacity picture pointers)
	iCapacity   int32       // capacity size of queue
	iCurrentIdx int32
}

type PPicBuff = *SPicBuff

/*
 *  Interfaces (implemented in pic_queue.go):
 *
 *  func PrefetchPic(pPicBuf *SPicBuff) *SPicture
 *  func PrefetchPicForThread(pPicBuf *SPicBuff) *SPicture
 *  func PrefetchLastPicForThread(pPicBuf *SPicBuff, iLastPicBuffIdx int32) *SPicture
 */
