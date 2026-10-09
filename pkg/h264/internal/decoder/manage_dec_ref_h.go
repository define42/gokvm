// Port of codec/decoder/core/inc/manage_dec_ref.h.
//
// This header only declares functions; they are implemented (with the Go
// signatures documented there) in manage_dec_ref.go. SIMD declarations are not ported.

package decoder

// Note: the C++ default argument of WelsMarkAsRef (PPicture pLastDec = NULL)
// cannot be expressed in Go; callers pass nil explicitly.
