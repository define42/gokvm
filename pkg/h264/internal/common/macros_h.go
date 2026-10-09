package common

import "math"

// Port of codec/common/inc/macros.h.
//
// Function-like macros become (generic) functions. Macros whose expansion
// contains a `return` (WELS_VERIFY_RETURN_IF, WELS_VERIFY_RETURN_IFNEQ,
// WELS_VERIFY_RETURN_PROC_IF) cannot be expressed as functions and must be
// expanded inline by the caller:
//
//	if bCaseIf { return iResult }
//
// ENFORCE_STACK_ALIGN_1D/2D and ALIGNED_DECLARE become plain local arrays,
// DISALLOW_COPY_AND_ASSIGN and WELS_GCC_UNUSED have no equivalent.

// Signed is the set of signed integer types.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned is the set of unsigned integer types.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Integer is the set of all integer types.
type Integer interface {
	Signed | Unsigned
}

// Float is the set of floating point types.
type Float interface {
	~float32 | ~float64
}

// Number is the set of integer and floating point types.
type Number interface {
	Integer | Float
}

// WELS_ALIGN rounds x up to a multiple of n (n must be a power of two).
func WELS_ALIGN[T Integer](x, n T) T {
	return (x + n - 1) &^ (n - 1)
}

// WELS_MAX returns the larger of x and y: ((x) > (y) ? (x) : (y)).
func WELS_MAX[T Number](x, y T) T {
	if x > y {
		return x
	}
	return y
}

// WELS_MIN returns the smaller of x and y: ((x) < (y) ? (x) : (y)).
func WELS_MIN[T Number](x, y T) T {
	if x < y {
		return x
	}
	return y
}

// WELS_MIN_POSITIVE: (x >= 0 && y >= 0) ? WELS_MIN(x, y) : WELS_MAX(x, y).
func WELS_MIN_POSITIVE[T Signed | Float](x, y T) T {
	if x >= 0 && y >= 0 {
		return WELS_MIN(x, y)
	}
	return WELS_MAX(x, y)
}

// WELS_CEIL is ceil(x).
func WELS_CEIL(x float64) float64 { return math.Ceil(x) }

// WELS_FLOOR is floor(x).
func WELS_FLOOR(x float64) float64 { return math.Floor(x) }

// WELS_ROUND is ((int32_t)(0.5+(x))). The addition is done in double
// precision exactly as in C (0.5 is a double literal).
func WELS_ROUND[T Number](x T) int32 {
	return int32(0.5 + float64(x))
}

// WELS_ROUND64 is ((int64_t)(0.5+(x))).
func WELS_ROUND64[T Number](x T) int64 {
	return int64(0.5 + float64(x))
}

// WELS_DIV_ROUND is ((int32_t)((y)==0?((x)/((y)+1)):(((y)/2+(x))/(y)))).
// Both operands must have the same (integer) type, which is the type the C
// expression is evaluated in after the usual arithmetic conversions.
func WELS_DIV_ROUND[T Integer](x, y T) int32 {
	if y == 0 {
		return int32(x / (y + 1))
	}
	return int32((y/2 + x) / y)
}

// WELS_DIV_ROUND64 is ((int64_t)((y)==0?((x)/((y)+1)):(((y)/2+(x))/(y)))).
func WELS_DIV_ROUND64[T Integer](x, y T) int64 {
	if y == 0 {
		return int64(x / (y + 1))
	}
	return int64((y/2 + x) / y)
}

// WELS_NON_ZERO_COUNT_AVERAGE computes nC from nA and nB:
//
//	nC = nA + nB + 1;
//	nC >>= (uint8_t)( nA != -1 && nB != -1);
//	nC += (uint8_t)(nA == -1 && nB == -1);
//
// The macro assigns to nC; the Go version returns the value (of nC's type T).
func WELS_NON_ZERO_COUNT_AVERAGE[T Integer](nA, nB T) T {
	a, b := int64(nA), int64(nB)
	nC := T(a + b + 1)
	if a != -1 && b != -1 {
		nC >>= 1
	}
	if a == -1 && b == -1 {
		nC++
	}
	return nC
}

// CeilLog2 returns ceil(log2(i)).
func CeilLog2(i int32) int32 {
	var s int32
	i--
	for i > 0 {
		s++
		i >>= 1
	}
	return s
}

// WelsMedian returns the median of three values.
func WelsMedian(iX, iY, iZ int32) int32 {
	iMin, iMax := iX, iX

	if iY < iMin {
		iMin = iY
	} else {
		iMax = iY
	}

	if iZ < iMin {
		iMin = iZ
	} else if iZ > iMax {
		iMax = iZ
	}

	return (iX + iY + iZ) - (iMin + iMax)
}

// NEG_NUM is (1+(~(iX))).
func NEG_NUM[T Integer](iX T) T {
	return 1 + (^iX)
}

// WelsClip1 clips iX to [0, 255].
func WelsClip1(iX int32) uint8 {
	if iX&^255 != 0 {
		return uint8((-iX) >> 31)
	}
	return uint8(iX)
}

// WELS_SIGN is ((int32_t)(iX) >> 31).
func WELS_SIGN[T Integer](iX T) int32 {
	return int32(iX) >> 31
}

// WELS_ABS is ((iX)>0 ? (iX) : -(iX)).
func WELS_ABS[T Number](iX T) T {
	if iX > 0 {
		return iX
	}
	return -iX
}

// WELS_CLIP3 is ((iX) < (iY) ? (iY) : ((iX) > (iZ) ? (iZ) : (iX))).
func WELS_CLIP3[T Number](iX, iY, iZ T) T {
	if iX < iY {
		return iY
	}
	if iX > iZ {
		return iZ
	}
	return iX
}

// WelsClip3 is the C++ template version of WELS_CLIP3.
func WelsClip3[T Number](iX, iY, iZ T) T {
	if iX < iY {
		return iY
	}
	if iX > iZ {
		return iZ
	}
	return iX
}

// WELS_LOG2 returns floor(log2(v)) (0 for v == 0).
func WELS_LOG2(v uint32) int32 {
	var r int32
	for {
		v >>= 1
		if v == 0 {
			break
		}
		r++
	}
	return r
}

// CLIP3_QP_0_51 is WELS_CLIP3(q, 0, 51).
func CLIP3_QP_0_51[T Number](q T) T {
	return WELS_CLIP3(q, 0, 51)
}

// CALC_BI_STRIDE is ((((width * bitcount) + 31) & ~31) >> 3).
func CALC_BI_STRIDE(width, bitcount int32) int32 {
	return (((width * bitcount) + 31) &^ 31) >> 3
}

// BUTTERFLY1x2 is (((b)<<8) | (b)).
func BUTTERFLY1x2(b uint8) uint16 {
	return uint16(b)<<8 | uint16(b)
}

// BUTTERFLY2x4 is (((uint32_t)(wd)<<16) |(wd)).
func BUTTERFLY2x4(wd uint16) uint32 {
	return uint32(wd)<<16 | uint32(wd)
}

// BUTTERFLY4x8 is (((uint64_t)(dw)<<32) | (dw)).
func BUTTERFLY4x8(dw uint32) uint64 {
	return uint64(dw)<<32 | uint64(dw)
}

// WELS_POWER2_IF reports whether v is a (non zero) power of two.
func WELS_POWER2_IF(v uint32) bool {
	return v != 0 && (v&(v-1)) == 0
}

// CheckInRangeCloseOpen reports kiMin <= kiCurrent < kiMax.
func CheckInRangeCloseOpen(kiCurrent, kiMin, kiMax int16) bool {
	return (kiCurrent >= kiMin) && (kiCurrent < kiMax)
}

// WelsSetMemUint32_c sets iSizeOfData uint32 values to iValue.
func WelsSetMemUint32_c(pDst []uint32, iValue uint32, iSizeOfData int32) {
	for i := int32(0); i < iSizeOfData; i++ {
		pDst[i] = iValue
	}
}

// WelsSetMemUint16_c sets iSizeOfData uint16 values to iValue.
func WelsSetMemUint16_c(pDst []uint16, iValue uint16, iSizeOfData int32) {
	for i := int32(0); i < iSizeOfData; i++ {
		pDst[i] = iValue
	}
}

// WelsSetMemMultiplebytes_c sets the first iSizeOfData elements of pDst to
// iValue (truncated to the element type). iDataLengthOfData is the element
// size in bytes (1, 2 or 4) and must match T; it is kept for signature
// fidelity with the C code.
func WelsSetMemMultiplebytes_c[T Integer](pDst []T, iValue uint32, iSizeOfData int32, iDataLengthOfData int32) {
	if iDataLengthOfData != 4 && iDataLengthOfData != 2 && iDataLengthOfData != 1 {
		panic("WelsSetMemMultiplebytes_c: invalid data length")
	}
	if iValue != 0 {
		v := T(iValue)
		for i := int32(0); i < iSizeOfData; i++ {
			pDst[i] = v
		}
	} else {
		clear(pDst[:iSizeOfData])
	}
}
