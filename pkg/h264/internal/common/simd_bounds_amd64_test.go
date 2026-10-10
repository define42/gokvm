//go:build amd64

package common

import "testing"

func requireSIMDBoundsPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("SIMD wrapper accepted an out-of-range buffer span")
		}
	}()
	fn()
}

func TestSIMDWrappersRejectOutOfRangeSpans(t *testing.T) {
	t.Run("SAD source tail", func(t *testing.T) {
		requireSIMDBoundsPanic(t, func() {
			WelsSampleSad8x8_sse2(make([]byte, 63), 0, 8, make([]byte, 64), 0, 8)
		})
	})
	t.Run("SAD reference tail", func(t *testing.T) {
		requireSIMDBoundsPanic(t, func() {
			WelsSampleSad8x8_sse2(make([]byte, 64), 0, 8, make([]byte, 63), 0, 8)
		})
	})
	t.Run("pixel average destination", func(t *testing.T) {
		requireSIMDBoundsPanic(t, func() {
			PixelAvg_sse2(make([]byte, 63), 0, 8, make([]byte, 64), 0, 8,
				make([]byte, 64), 0, 8, 8, 8)
		})
	})
	t.Run("horizontal left padding", func(t *testing.T) {
		requireSIMDBoundsPanic(t, func() {
			McHorVer20_sse2(make([]byte, 32), 1, 16, make([]byte, 8), 0, 8, 4, 1)
		})
	})
	t.Run("horizontal right padding", func(t *testing.T) {
		requireSIMDBoundsPanic(t, func() {
			McHorVer20_sse2(make([]byte, 8), 2, 8, make([]byte, 8), 0, 8, 4, 1)
		})
	})
	t.Run("vertical top padding", func(t *testing.T) {
		requireSIMDBoundsPanic(t, func() {
			McHorVer02_sse2(make([]byte, 64), 8, 8, make([]byte, 8), 0, 8, 4, 1)
		})
	})
	t.Run("center bottom padding", func(t *testing.T) {
		requireSIMDBoundsPanic(t, func() {
			McHorVer22_sse2(make([]byte, 54), 24, 8, make([]byte, 8), 0, 8, 4, 1)
		})
	})
}
