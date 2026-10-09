package common

// Port of codec/common/inc/memory_align.h.
//
// Go allocations cannot fail and need no manual alignment, so CMemoryAlign
// only keeps the MEMORY_MONITOR accounting (WelsGetMemoryUsage), which the
// C codec uses for log messages and a debug assert. Allocate typed memory
// with make()/new() and, when the accounting matters, report the byte size
// through WelsMallocAccount / WelsFreeAccount. WelsMalloc / WelsMallocz /
// WelsFree on []uint8 do both in one step.
//
// WELS_SAFE_FREE, WELS_NEW_OP and WELS_DELETE_OP map to `p = nil` /
// `&T{}` and are not provided.

// CMemoryAlign mirrors WelsCommon::CMemoryAlign.
type CMemoryAlign struct {
	m_nCacheLineSize      uint32
	m_nMemoryUsageInBytes uint32
}
