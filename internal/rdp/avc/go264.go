package avc

import (
	"fmt"

	go264 "github.com/oops1/go.264"
)

const (
	codecGOPSize = 150
	codecQP      = 20
)

type go264Encoder struct {
	codec *go264.Encoder
}

func newCodecEncoder(width, height, threads int) (codecEncoder, int, error) {
	slices := min(threads, (height+15)/16)
	codec, err := go264.NewEncoder(go264.EncoderConfig{
		Width: width, Height: height,
		FPSNum: FrameRate, FPSDen: 1,
		GOPSize:   codecGOPSize,
		QP:        codecQP,
		RefFrames: 1,
		BFrames:   0,
		Slices:    slices,
		// The I420 samples already use AVC420's required full-range BT.709
		// matrix. go.264 v1.12 does not expose SPS VUI color fields.
		// The optional hardware backends do not all honor forced refreshes.
		// Keep the RDP reference chain on the deterministic pure-Go encoder.
		ForceSoftware: true,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: initialize: %w", ErrCodec, err)
	}

	return &go264Encoder{codec: codec}, slices, nil
}

func (e *go264Encoder) encode(i420 []byte, force bool, _ int64) ([]byte, error) {
	if force {
		e.codec.ForceKeyFrame()
	}
	data, err := e.codec.Encode(i420)
	if err != nil {
		return nil, fmt.Errorf("%w: encode: %w", ErrCodec, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: empty access unit", ErrCodec)
	}

	return data, nil
}

func (e *go264Encoder) close() { _ = e.codec.Close() }
