package avc

import (
	"errors"
	"image"
	"testing"
)

func TestThreadOptions(t *testing.T) {
	t.Parallel()
	for _, threads := range []int{-1, 0, MaxThreads + 1} {
		encoder, err := NewEncoderWithOptions(64, 48, Options{Threads: threads})
		if !errors.Is(err, ErrThreads) || encoder != nil {
			t.Fatalf("threads %d: encoder %v, error %v", threads, encoder, err)
		}
	}
}

func TestEncodeMeasurements(t *testing.T) {
	t.Parallel()
	for _, measure := range []bool{false, true} {
		codec := &recordingCodec{}
		encoder := &Encoder{
			codec: codec, width: 16, height: 16, i420: make([]byte, 16*16*3/2), measure: measure,
		}
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))
		if _, err := encoder.Encode(img, false); err != nil {
			t.Fatal(err)
		}
		stats := encoder.LastStats()
		if measure && (stats.Conversion <= 0 || stats.Encoding <= 0) {
			t.Fatalf("measured frame stats = %+v", stats)
		}
		if !measure && stats != (EncodeStats{}) {
			t.Fatalf("disabled frame stats = %+v", stats)
		}
		if _, err := encoder.Encode(img, false); err != nil {
			t.Fatal(err)
		}
		if encoder.LastStats().Encoding != 0 {
			t.Fatal("unchanged frame retained previous encoding measurement")
		}
		encoder.Close()
		if _, err := encoder.Encode(img, false); !errors.Is(err, ErrClosed) {
			t.Fatalf("closed encoder returned %v", err)
		}
		if encoder.LastStats() != (EncodeStats{}) {
			t.Fatal("failed frame retained measurements")
		}
	}
}
