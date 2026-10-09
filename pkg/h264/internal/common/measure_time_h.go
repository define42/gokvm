package common

import "time"

// Port of codec/common/inc/measure_time.h.

// WelsTime returns the time elapsed since the Unix epoch in microseconds.
func WelsTime() int64 {
	return time.Now().UnixMicro()
}
