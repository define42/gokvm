//go:build !openh264 || !cgo

package avc

// Available reports whether this binary includes the OpenH264 encoder.
func Available() bool { return false }

func newNativeEncoder(_, _ int) (nativeEncoder, error) {
	return nil, ErrUnavailable
}
