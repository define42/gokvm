//go:build !openh264 || !cgo

package avc

import (
	"errors"
	"testing"
)

func TestUnavailable(t *testing.T) {
	t.Parallel()
	if Available() {
		t.Fatal("OpenH264 unexpectedly available")
	}
	if encoder, err := NewEncoder(1024, 768); !errors.Is(err, ErrUnavailable) || encoder != nil {
		t.Fatalf("NewEncoder = %v, %v", encoder, err)
	}
}
