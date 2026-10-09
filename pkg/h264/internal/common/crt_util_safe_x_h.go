package common

import (
	"io"
	"os"
)

// Port of codec/common/inc/crt_util_safe_x.h (safe CRT like utilities).
//
// C character buffers become Go strings: functions that write into a
// `char* buffer` take a *string (may be nil) and keep the C return values.

const (
	WELS_FILE_SEEK_SET = io.SeekStart
	WELS_FILE_SEEK_CUR = io.SeekCurrent
	WESL_FILE_SEEK_END = io.SeekEnd // (sic) spelled as in the C header
)

// WelsFileHandle is the C FILE.
type WelsFileHandle = os.File

// SWelsTime mirrors TagWelsTime.
type SWelsTime struct {
	Time    int64 // time_t: seconds since the Unix epoch
	Millitm uint16
}
