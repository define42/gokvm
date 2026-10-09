package common

import (
	"math/rand"
	"testing"
)

func refSad(a []uint8, aOff, aStride int, b []uint8, bOff, bStride int, w, h int) int32 {
	var s int32
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			d := int32(a[aOff+j*aStride+i]) - int32(b[bOff+j*bStride+i])
			if d < 0 {
				d = -d
			}
			s += d
		}
	}
	return s
}

func TestSampleSad(t *testing.T) {
	type sadFn func([]uint8, int, int32, []uint8, int, int32) int32
	type sadFourFn func([]uint8, int, int32, []uint8, int, int32, []int32)
	cases := []struct {
		w, h int
		fn   sadFn
		four sadFourFn
	}{
		{16, 16, WelsSampleSad16x16_c, WelsSampleSadFour16x16_c},
		{16, 8, WelsSampleSad16x8_c, WelsSampleSadFour16x8_c},
		{8, 16, WelsSampleSad8x16_c, WelsSampleSadFour8x16_c},
		{8, 8, WelsSampleSad8x8_c, WelsSampleSadFour8x8_c},
		{8, 4, WelsSampleSad8x4_c, WelsSampleSadFour8x4_c},
		{4, 8, WelsSampleSad4x8_c, WelsSampleSadFour4x8_c},
		{4, 4, WelsSampleSad4x4_c, WelsSampleSadFour4x4_c},
	}
	rnd := rand.New(rand.NewSource(30))
	const s1, s2 = 40, 48
	a := make([]uint8, s1*24)
	b := make([]uint8, s2*24)
	for iter := 0; iter < 100; iter++ {
		for i := range a {
			a[i] = uint8(rnd.Intn(256))
		}
		for i := range b {
			b[i] = uint8(rnd.Intn(256))
		}
		if iter == 0 { // extremes
			for i := range a {
				a[i], b[i] = 255, 0
			}
		}
		aOff := 2*s1 + 3
		bOff := 2*s2 + 5
		for _, c := range cases {
			if got, want := c.fn(a, aOff, s1, b, bOff, s2), refSad(a, aOff, s1, b, bOff, s2, c.w, c.h); got != want {
				t.Fatalf("Sad%dx%d: got %d want %d", c.w, c.h, got, want)
			}
			var pSad [4]int32
			c.four(a, aOff, s1, b, bOff, s2, pSad[:])
			want := [4]int32{
				refSad(a, aOff, s1, b, bOff-s2, s2, c.w, c.h),
				refSad(a, aOff, s1, b, bOff+s2, s2, c.w, c.h),
				refSad(a, aOff, s1, b, bOff-1, s2, c.w, c.h),
				refSad(a, aOff, s1, b, bOff+1, s2, c.w, c.h),
			}
			if pSad != want {
				t.Fatalf("SadFour%dx%d: got %v want %v", c.w, c.h, pSad, want)
			}
		}
	}
}
