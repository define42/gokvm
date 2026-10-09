// Port of codec/decoder/core/inc/wels_decoder_thread.h.
//
// The Go port is single-threaded (see PORTING.md): the semaphore, event and
// thread types are inert placeholders that only exist so that the
// sequential code paths and struct layouts can be translated verbatim. The
// functions in wels_decoder_thread.go are no-ops.

package decoder

const (
	WELS_DEC_MAX_NUM_CPU            = 16
	WELS_DEC_MAX_THREAD_STACK_SIZE  = 4096
	WELS_DEC_THREAD_COMMAND_RUN     = 0
	WELS_DEC_THREAD_COMMAND_ABORT   = 1
	WELS_DEC_THREAD_WAIT_TIMEDOUT   = 110 // ETIMEDOUT
	WELS_DEC_THREAD_WAIT_SIGNALED   = 4   // EINTR
	WELS_DEC_THREAD_WAIT_INFINITE   = -1
	WELS_DEC_THREAD_WAIT_TIMEOUT_MS = 3000 // Bound inter-thread waits so a lost reference cannot hang the decoder.
)

// SWelsDecSemphore is a placeholder (no threading in the Go port).
type SWelsDecSemphore struct {
	max int32
	v   int32
}

// SWelsDecEvent is a placeholder (no threading in the Go port).
type SWelsDecEvent struct {
	manualReset int32
	isSignaled  int32
}

// SWelsDecThread is a placeholder (no threading in the Go port).
type SWelsDecThread struct{}

// LPWELS_THREAD_ROUTINE is the thread routine type (placeholder).
type LPWELS_THREAD_ROUTINE func(p any) uint32

// The CREATE_EVENT / SET_EVENT / WAIT_EVENT / ... macros map to the
// functions of wels_decoder_thread.go:
//
//	CREATE_THREAD(ph, threadproc, argument)          -> ThreadCreate(ph, threadproc, argument)
//	CREATE_EVENT(ph, manualreset, initial_state, _)  -> EventCreate(ph, manualreset, initial_state)
//	CREATE_SEMAPHORE(ph, initial_count, max_count, _) -> SemCreate(ph, initial_count, max_count)
//	CLOSE_EVENT(ph)                                   -> EventDestroy(ph)
//	CLOSE_SEMAPHORE(ph)                               -> SemDestroy(ph)
//	SET_EVENT(ph)                                     -> EventPost(ph)
//	RESET_EVENT(ph)                                   -> EventReset(ph)
//	RELEASE_SEMAPHORE(ph)                             -> SemRelease(ph, nil)
//	WAIT_EVENT(ph, timeout)                           -> EventWait(ph, timeout)
//	WAIT_THREAD(ph)                                   -> ThreadWait(ph)
//	WAIT_SEMAPHORE(ph, timeout)                       -> SemWait(ph, timeout)
