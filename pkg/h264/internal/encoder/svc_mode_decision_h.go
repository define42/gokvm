// Port of codec/encoder/core/inc/svc_mode_decision.h.

package encoder

const DELTA_QP_SCD_THD = 5

// ESkipModes
type ESkipModes int32

const (
	STATIC ESkipModes = iota
	SCROLLED
)

type pJudgeSkipFun func(pEncCtx *sWelsEncCtx, pCurMb *SMB, pMbCache *SMbCache, pWelsMd *SWelsMD) bool
