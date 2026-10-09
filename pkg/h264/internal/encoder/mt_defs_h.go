// Port of codec/encoder/core/inc/mt_defs.h.
//
// The port is single-threaded: thread handles, events and mutexes are
// dropped; only the per-thread data the sequential path uses is kept.

package encoder

import "github.com/define42/gokvm/pkg/h264/api"

const (
	THRESHOLD_RMSE_CORE8 = float32(0.0320) // v1.1: 0.0320f; v1.0: 0.02f
	THRESHOLD_RMSE_CORE4 = float32(0.0215) // v1.1: 0.0215f; v1.0: 0.03f
	THRESHOLD_RMSE_CORE2 = float32(0.0200) // v1.1: 0.0200f; v1.0: 0.04f
)

// SSliceThreadPrivateData is the per thread context.
type SSliceThreadPrivateData struct {
	pWelsPEncCtx *sWelsEncCtx // C void*
	pFrameBsInfo *api.SFrameBSInfo
	iSliceIndex  int32 // slice index, zero based
	iThreadIndex int32 // thread index, zero based
}

// SSliceThreading holds the slice threading resources (sequential placeholder).
// Dropped: eventNamespace, pThreadHandles, all WELS_EVENT / WELS_MUTEX members.
type SSliceThreading struct {
	pThreadPEncCtx []SSliceThreadPrivateData // C SSliceThreadPrivateData*: thread context, [iThreadIdx]

	pThreadBsBuffer      [MAX_THREADS_NUM][]uint8 // actual memory for slice buffer
	bThreadBsBufferUsage [MAX_THREADS_NUM]bool
}
