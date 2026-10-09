//go:build !openh264 || !cgo

package avc

// Available reports whether this binary includes the OpenH264 encoder.
func Available() bool { return false }

func newNativeEncoder(_, _, _ int) (nativeEncoder, int, error) {
	return nil, 0, ErrUnavailable
}
