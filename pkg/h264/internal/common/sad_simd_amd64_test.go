//go:build amd64

package common

import (
	"math/rand"
	"testing"
)

func TestSampleSadSSE2MatchesScalar(t *testing.T) {
	type sadFn func([]uint8, int, int32, []uint8, int, int32) int32
	type sadFourFn func([]uint8, int, int32, []uint8, int, int32, []int32)
	cases := []struct {
		name       string
		w, h       int
		scalar     sadFn
		simd       sadFn
		scalarFour sadFourFn
		simdFour   sadFourFn
	}{
		{"16x16", 16, 16, WelsSampleSad16x16_c, WelsSampleSad16x16_sse2, WelsSampleSadFour16x16_c, WelsSampleSadFour16x16_sse2},
		{"16x8", 16, 8, WelsSampleSad16x8_c, WelsSampleSad16x8_sse2, WelsSampleSadFour16x8_c, WelsSampleSadFour16x8_sse2},
		{"8x16", 8, 16, WelsSampleSad8x16_c, WelsSampleSad8x16_sse2, WelsSampleSadFour8x16_c, WelsSampleSadFour8x16_sse2},
		{"8x8", 8, 8, WelsSampleSad8x8_c, WelsSampleSad8x8_sse2, WelsSampleSadFour8x8_c, WelsSampleSadFour8x8_sse2},
		{"8x4", 8, 4, WelsSampleSad8x4_c, WelsSampleSad8x4_sse2, WelsSampleSadFour8x4_c, WelsSampleSadFour8x4_sse2},
		{"4x8", 4, 8, WelsSampleSad4x8_c, WelsSampleSad4x8_sse2, WelsSampleSadFour4x8_c, WelsSampleSadFour4x8_sse2},
		{"4x4", 4, 4, WelsSampleSad4x4_c, WelsSampleSad4x4_sse2, WelsSampleSadFour4x4_c, WelsSampleSadFour4x4_sse2},
	}

	rng := rand.New(rand.NewSource(0x264))
	for _, stride := range []int{16, 23, 32, 63} {
		for _, off := range []int{0, 1, 3, 7, 15, 31} {
			for iteration := 0; iteration < 20; iteration++ {
				a := make([]uint8, stride*32+128)
				b := make([]uint8, stride*32+128)
				for i := range a {
					a[i] = uint8(rng.Intn(256))
					b[i] = uint8(rng.Intn(256))
				}
				if iteration == 0 {
					for i := range a {
						a[i], b[i] = 255, 0
					}
				} else if iteration == 1 {
					copy(b, a)
				}

				aOff := 2*stride + off
				bOff := 3*stride + off + 1
				for _, tc := range cases {
					got := tc.simd(a, aOff, int32(stride), b, bOff, int32(stride))
					want := tc.scalar(a, aOff, int32(stride), b, bOff, int32(stride))
					if got != want {
						t.Fatalf("%s stride=%d off=%d iteration=%d: SIMD=%d scalar=%d", tc.name, stride, off, iteration, got, want)
					}

					var gotFour, wantFour [4]int32
					tc.simdFour(a, aOff, int32(stride), b, bOff, int32(stride), gotFour[:])
					tc.scalarFour(a, aOff, int32(stride), b, bOff, int32(stride), wantFour[:])
					if gotFour != wantFour {
						t.Fatalf("four %s stride=%d off=%d iteration=%d: SIMD=%v scalar=%v", tc.name, stride, off, iteration, gotFour, wantFour)
					}
				}
			}
		}
	}
}

func BenchmarkSampleSad8x8(b *testing.B) {
	benchmarkSampleSad(b, WelsSampleSad8x8_c, WelsSampleSad8x8_sse2)
}

func BenchmarkSampleSad16x16(b *testing.B) {
	benchmarkSampleSad(b, WelsSampleSad16x16_c, WelsSampleSad16x16_sse2)
}

func benchmarkSampleSad(b *testing.B, scalar, simd func([]uint8, int, int32, []uint8, int, int32) int32) {
	const stride = 32
	a := make([]uint8, stride*24)
	ref := make([]uint8, stride*24)
	for i := range a {
		a[i] = uint8(i*17 + 3)
		ref[i] = uint8(i*29 + 11)
	}
	for _, tc := range []struct {
		name string
		fn   func([]uint8, int, int32, []uint8, int, int32) int32
	}{{"scalar", scalar}, {"sse2", simd}} {
		b.Run(tc.name, func(b *testing.B) {
			var result int32
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result = tc.fn(a, 3, stride, ref, 5, stride)
			}
			if result == 0 {
				b.Fatal("unexpected zero SAD")
			}
		})
	}
}
