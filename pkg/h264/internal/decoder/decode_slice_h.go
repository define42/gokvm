// Port of codec/decoder/core/inc/decode_slice.h.
//
// Functions declared in this header are implemented in decode_slice.go.

package decoder

// Raw byte size of one 4:2:0 8-bit I_PCM macroblock copied verbatim from the
// bitstream: 16x16 luma + 2 x (8x8) chroma.
const I_PCM_MB_SIZE_IN_BYTE = 16*16 + 2*(8*8)

type PWelsDecMbFunc func(pCtx *SWelsDecoderContext, pNalCur *SNalUnit, uiEosFlag *uint32) int32
