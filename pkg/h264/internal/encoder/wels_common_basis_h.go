// Port of codec/encoder/core/inc/wels_common_basis.h.

package encoder

// WelsErrorType is int32_t.
type WelsErrorType = int32

// SMVUnitXY holds one motion vector (each 4 Bytes).
type SMVUnitXY struct {
	iMvX int16
	iMvY int16
}

// sDeltaMv sets *p = _v0 - _v1 (component wise) and returns p.
func (p *SMVUnitXY) sDeltaMv(_v0, _v1 SMVUnitXY) *SMVUnitXY {
	p.iMvX = _v0.iMvX - _v1.iMvX
	p.iMvY = _v0.iMvY - _v1.iMvY
	return p
}

// sAssignMv sets *p = _v0 and returns p.
func (p *SMVUnitXY) sAssignMv(_v0 SMVUnitXY) *SMVUnitXY {
	p.iMvX = _v0.iMvX
	p.iMvY = _v0.iMvY
	return p
}

// SMVComponentUnit is the MV / ref-idx cache of one list (each LIST_0/LIST_1).
type SMVComponentUnit struct {
	sMotionVectorCache [5*6 - 1]SMVUnitXY // Luma only: 5 x 6 - 1 = 29 D-Words
	iRefIndexCache     [5 * 6]int8        // Luma only: 5 x 6 = 30 bytes
}

// PMVComponentUnit is a pointer alias.
type PMVComponentUnit = *SMVComponentUnit

type SParaSetOffsetVariable struct {
	iParaSetIdDelta          [MAX_DQ_LAYER_NUM]int32 // mark delta between SPS_ID_in_bs and sps_id_in_encoder, can be minus, for each dq-layer
	bUsedParaSetIdInBs       [MAX_PPS_COUNT]bool     // mark the used SPS_ID with 1
	uiNextParaSetIdToUseInBs uint32                  // mark the next SPS_ID_in_bs, for all layers
}

type SParaSetOffset struct {
	// in PS0 design, "sParaSetOffsetVariable" record the previous paras before current IDR, AND NEED to be stacked and recover across IDR
	sParaSetOffsetVariable [PARA_SET_TYPE]SParaSetOffsetVariable // PARA_SET_TYPE=3; paraset_type = 0: AVC_SPS; =1: Subset_SPS; =2: PPS
	// in PSO design, "bPpsIdMappingIntoSubsetsps" uses the current para of current IDR period
	bPpsIdMappingIntoSubsetsps [MAX_DQ_LAYER_NUM]bool
	iPpsIdList                 [MAX_DQ_LAYER_NUM][MAX_PPS_COUNT]int32 // index0: max pps types; index1: for differnt IDRs, if only index0=1, index1 can reach MAX_PPS_COUNT
	// (eSpsPpsIdStrategy only exists in _DEBUG builds and is not ported)
	uiNeededSpsNum       uint32
	uiNeededSubsetSpsNum uint32
	uiNeededPpsNum       uint32
	uiInUseSpsNum        uint32
	uiInUseSubsetSpsNum  uint32
	uiInUsePpsNum        uint32
}

// SCropOffset is the position offset structure.
type SCropOffset struct {
	iCropLeft   int16
	iCropRight  int16
	iCropTop    int16
	iCropBottom int16
}

// ETransType is the transform type.
type ETransType int32

const (
	T_4x4   ETransType = 0
	T_8x8   ETransType = 1
	T_16x16 ETransType = 2
	T_PCM   ETransType = 3
)

// EMbPosition
const (
	LEFT_MB_POS        = 0x01 // A
	TOP_MB_POS         = 0x02 // B
	TOPRIGHT_MB_POS    = 0x04 // C
	TOPLEFT_MB_POS     = 0x08 // D,
	RIGHT_MB_POS       = 0x10 //  add followed four case to reuse when intra up-sample
	BOTTOM_MB_POS      = 0x20 //
	BOTTOMRIGHT_MB_POS = 0x40 //
	BOTTOMLEFT_MB_POS  = 0x80 //
	MB_POS_A           = 0x100
)

// Mb_Type is the MB type & sub-MB type (uint32_t); values are the
// common.MB_TYPE_* constants.
type Mb_Type = uint32

const (
	MB_LEFT_BIT     = 0 // add to use in intra up-sample
	MB_TOP_BIT      = 1
	MB_TOPRIGHT_BIT = 2
	MB_TOPLEFT_BIT  = 3
	MB_RIGHT_BIT    = 4
	MB_BOTTOM_BIT   = 5
	MB_BTMRIGHT_BIT = 6
	MB_BTMLEFT_BIT  = 7

	MB_TYPE_BACKGROUND = 0x00010000 // conditional BG skip_mb
)

const (
	Intra4x4   = 0
	Intra16x16 = 1
	Inter16x16 = 2
	Inter16x8  = 3
	Inter8x16  = 4
	Inter8x8   = 5
	PSkip      = 6
)
