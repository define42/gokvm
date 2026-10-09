package common

// Port of codec/common/inc/utils.h.

const (
	MAX_LOG_SIZE      = 1024
	MAX_MBS_PER_FRAME = 36864 //in accordance with max level support in Rec
)

// PWelsLogCallbackFunc is the wels log output callback. The C va_list is
// passed as a []any holding the format arguments.
type PWelsLogCallbackFunc func(pCtx any, iLevel int32, kpFmt string, argv []any)

// SLogContext mirrors TagLogContext.
type SLogContext struct {
	PfLog          PWelsLogCallbackFunc
	PLogCtx        any
	PCodecInstance any
}
