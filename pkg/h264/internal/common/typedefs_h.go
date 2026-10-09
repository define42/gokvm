package common

// Port of codec/common/inc/typedefs.h.
//
// The fixed width C integer types map 1:1 onto Go's int8 … uint64. intX_t
// (pointer sized signed integer) maps onto Go's int.

// IntX_t mirrors intX_t: a pointer sized signed integer.
type IntX_t = int

// EPSN is the desired float precision (1e-6).
const EPSN float32 = 0.000001
