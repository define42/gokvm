// Port of codec/decoder/core/inc/bit_stream.h.
//
// The bit-stream reader structure SBitStringAux is defined in package common
// (common.SBitStringAux: PBuf is the whole buffer, PStartBuf / PEndBuf /
// PCurBuf are int offsets into it).
//
// Declared functions (implemented in bit_stream.go):
//
//	func DecInitBits(pBitString *common.SBitStringAux, kpBuf []uint8, kiBufOff int, kiSize int32) int32
//	func InitReadBits(pBitString *common.SBitStringAux, iEndOffset int) int32
//	func RBSP2EBSP(pDstBuf []uint8, pSrcBuf []uint8, kiSize int32)

package decoder
