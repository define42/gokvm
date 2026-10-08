//go:build openh264 && cgo

package avc

/*
#cgo pkg-config: openh264
#include "native.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type openH264Encoder struct{ handle *C.gokvm_avc_encoder }

// Available reports whether this binary includes the OpenH264 encoder.
func Available() bool { return true }

//nolint:gocritic,nlreturn // cgo-generated pointer checks trigger these style checks.
func newNativeEncoder(width, height int) (nativeEncoder, error) {
	var handle *C.gokvm_avc_encoder
	if status := C.gokvm_avc_create(C.int(width), C.int(height), &handle); status != 0 {
		return nil, fmt.Errorf("%w: initialize (status %d)", ErrCodec, status)
	}

	return &openH264Encoder{handle: handle}, nil
}

//nolint:gocritic,nlreturn // cgo-generated pointer checks trigger these style checks.
func (e *openH264Encoder) encode(i420 []byte, force bool, timestamp int64) ([]byte, error) {
	var (
		data   *C.uchar
		length C.int
		idr    C.int
	)
	if force {
		idr = 1
	}
	status := C.gokvm_avc_encode(e.handle, (*C.uchar)(unsafe.Pointer(&i420[0])), idr,
		C.longlong(timestamp), &data, &length)
	if status != 0 {
		return nil, fmt.Errorf("%w: encode (status %d)", ErrCodec, status)
	}

	return C.GoBytes(unsafe.Pointer(data), length), nil
}

func (e *openH264Encoder) close() { C.gokvm_avc_destroy(e.handle) }
