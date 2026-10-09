// Port of codec/encoder/core/inc/ref_list_mgr_svc.h.

package encoder

// LTR_MARKING_PROCESS_MODE
type LTR_MARKING_PROCESS_MODE int32

const (
	LTR_DIRECT_MARK LTR_MARKING_PROCESS_MODE = 0
	LTR_DELAY_MARK  LTR_MARKING_PROCESS_MODE = 1
)

// COMPARE_FRAME_NUM
type COMPARE_FRAME_NUM int32

const (
	FRAME_NUM_EQUAL    COMPARE_FRAME_NUM = 0x01
	FRAME_NUM_BIGGER   COMPARE_FRAME_NUM = 0x02
	FRAME_NUM_SMALLER  COMPARE_FRAME_NUM = 0x04
	FRAME_NUM_OVER_MAX COMPARE_FRAME_NUM = 0x08
)

// IWelsReferenceStrategy is the virtual interface of the reference list
// strategies (class IWelsReferenceStrategy). Create objects with
// CreateReferenceStrategy (ref_list_mgr_svc.go).
type IWelsReferenceStrategy interface {
	BuildRefList(iPOC int32, iBestLtrRefIdx int32) bool
	MarkPic()
	UpdateRefList() bool
	EndofUpdateRefList()
	AfterBuildRefList()

	// protected virtual:
	Init(pCtx *sWelsEncCtx)
}

// CWelsReference_TemporalLayer is the camera (temporal layer) strategy.
type CWelsReference_TemporalLayer struct {
	m_pEncoderCtx *sWelsEncCtx
}

// CWelsReference_Screen is the screen content strategy.
type CWelsReference_Screen struct {
	CWelsReference_TemporalLayer
}

// CWelsReference_LosslessWithLtr is the lossless screen content with LTR strategy.
type CWelsReference_LosslessWithLtr struct {
	CWelsReference_Screen
}

var (
	_ IWelsReferenceStrategy = (*CWelsReference_TemporalLayer)(nil)
	_ IWelsReferenceStrategy = (*CWelsReference_Screen)(nil)
	_ IWelsReferenceStrategy = (*CWelsReference_LosslessWithLtr)(nil)
)
