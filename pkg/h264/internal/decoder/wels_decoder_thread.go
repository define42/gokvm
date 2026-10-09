// Port of codec/decoder/core/src/wels_decoder_thread.cpp.
//
// The Go port is single-threaded (see PORTING.md and
// wels_decoder_thread_h.go): these functions only keep the inert
// placeholder state consistent. No thread is ever created.

package decoder

import "runtime"

func GetCPUCount() int32 {
	return int32(runtime.NumCPU())
}

// ThreadCreate never starts a thread; it returns 0 (success).
func ThreadCreate(t *SWelsDecThread, tf LPWELS_THREAD_ROUTINE, ta any) int32 {
	return 0
}

func ThreadWait(t *SWelsDecThread) int32 {
	return 0
}

// Event

func EventCreate(e *SWelsDecEvent, manualReset int32, initialState int32) int32 {
	e.manualReset = manualReset
	e.isSignaled = initialState
	return 0
}

func EventPost(e *SWelsDecEvent) {
	e.isSignaled = 1
}

// EventWait never blocks: it reports WELS_DEC_THREAD_WAIT_SIGNALED if the
// event is signaled and WELS_DEC_THREAD_WAIT_TIMEDOUT otherwise.
func EventWait(e *SWelsDecEvent, timeout int32) int32 {
	if e.isSignaled != 0 {
		if e.manualReset == 0 {
			e.isSignaled = 0
		}
		return WELS_DEC_THREAD_WAIT_SIGNALED
	}
	return WELS_DEC_THREAD_WAIT_TIMEDOUT
}

func EventReset(e *SWelsDecEvent) {
	e.isSignaled = 0
}

func EventDestroy(e *SWelsDecEvent) {
}

// Semaphore

func SemCreate(s *SWelsDecSemphore, value int32, max int32) int32 {
	s.v = value
	s.max = max
	return 0
}

// SemWait never blocks: it reports WELS_DEC_THREAD_WAIT_SIGNALED if the count
// is positive (and decrements it) and WELS_DEC_THREAD_WAIT_TIMEDOUT otherwise.
func SemWait(s *SWelsDecSemphore, timeout int32) int32 {
	if s.v > 0 {
		s.v--
		return WELS_DEC_THREAD_WAIT_SIGNALED
	}
	return WELS_DEC_THREAD_WAIT_TIMEDOUT
}

func SemRelease(s *SWelsDecSemphore, prev_count *int32) {
	if prev_count != nil {
		*prev_count = s.v
	}
	if s.v < s.max {
		s.v++
	}
}

func SemDestroy(s *SWelsDecSemphore) {
}
