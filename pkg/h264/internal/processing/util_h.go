package processing

// Port of codec/processing/src/common/util.h, plus the few macros of
// codec/common/inc/macros.h that the processing code uses (kept local so this
// package does not depend on internal/common).

import "cmp"

const MAX_MBS_PER_FRAME = 36864 //in accordance with max level support in Rec

const (
	MB_WIDTH_LUMA         = 16
	PESN                  = 1e-6 // desired float precision
	AQ_INT_MULTIPLY       = 10000000
	AQ_TIME_INT_MULTIPLY  = 10000
	AQ_QSTEP_INT_MULTIPLY = 100
	AQ_PESN               = 10 // (1e-6)*AQ_INT_MULTIPLY
)

const (
	MB_TYPE_INTRA4x4   = 0x00000001
	MB_TYPE_INTRA16x16 = 0x00000002
	MB_TYPE_INTRA_PCM  = 0x00000004
	MB_TYPE_INTRA      = MB_TYPE_INTRA4x4 | MB_TYPE_INTRA16x16 | MB_TYPE_INTRA_PCM
)

func IS_INTRA(t uint32) bool {
	return (t & MB_TYPE_INTRA) != 0
}

func WELS_MAX[T cmp.Ordered](x, y T) T {
	if x > y {
		return x
	}
	return y
}

func WELS_MIN[T cmp.Ordered](x, y T) T {
	if x < y {
		return x
	}
	return y
}

func WELS_SIGN(a int32) int32 {
	return a >> 31
}

func WELS_ABS(a int32) int32 {
	return (WELS_SIGN(a) ^ a) - WELS_SIGN(a)
}

func WELS_CLAMP[T cmp.Ordered](x, minv, maxv T) T {
	return WELS_MIN(WELS_MAX(x, minv), maxv)
}

const ALIGNBYTES = 16 /* Worst case is requiring alignment to an 16 byte boundary */

func GET_METHOD(x int32) int32  { return x & 0xff }        // mask method as the lowest 8bits
func GET_SPECIAL(x int32) int32 { return (x >> 8) & 0xff } // mask special flag as 8bits

func WelsVpGetValidMethod(a int32) EMethods {
	iMethod := GET_METHOD(a)
	return EMethods(WELS_CLAMP(iMethod, METHOD_NULL+1, METHOD_MASK-1))
}

// ---- from codec/common/inc/macros.h ----

// WELS_ALIGN: (((x)+(n)-1)&~((n)-1))
func WELS_ALIGN(x, n int32) int32 {
	return (x + n - 1) &^ (n - 1)
}

// WELS_ROUND: ((int32_t)(0.5+(x))), x being a float expression promoted to double.
func WELS_ROUND(x float64) int32 {
	return int32(0.5 + x)
}

// WELS_DIV_ROUND64: ((int64_t)((y)==0?((x)/((y)+1)):(((y)/2+(x))/(y))))
func WELS_DIV_ROUND64(x, y int64) int64 {
	if y == 0 {
		return x / (y + 1)
	}
	return (y/2 + x) / y
}
