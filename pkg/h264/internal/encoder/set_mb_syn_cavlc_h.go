// Port of codec/encoder/core/inc/set_mb_syn_cavlc.h.

package encoder

// ECtxBlockCat is the residual block category.
type ECtxBlockCat int32

const (
	LUMA_DC   ECtxBlockCat = 0
	LUMA_AC   ECtxBlockCat = 1
	LUMA_4x4  ECtxBlockCat = 2
	CHROMA_DC ECtxBlockCat = 3
	CHROMA_AC ECtxBlockCat = 4
)

const LUMA_DC_AC = 0x04

type SCavlcTableItem struct {
	uiBits         uint16
	uiLen          uint8
	uiSuffixLength uint8
}
