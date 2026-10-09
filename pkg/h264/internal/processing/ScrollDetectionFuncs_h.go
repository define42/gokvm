package processing

// Port of codec/processing/src/scrolldetection/ScrollDetectionFuncs.h.

const (
	MINIMUM_DETECT_WIDTH = 50 // no less than 16
	CHECK_OFFSET         = 25
	MAX_SCROLL_MV_Y      = 511
	REGION_NUMBER        = 9
)

func RECORD_COLOR(a uint8, x *[8]int32) {
	_t := int32(a)
	x[_t>>5] |= (int32(1) << (_t & 31))
}
