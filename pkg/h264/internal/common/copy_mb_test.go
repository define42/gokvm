package common

import (
	"math/rand"
	"testing"
)

func TestWelsCopy(t *testing.T) {
	type copyFn func([]uint8, int, int32, []uint8, int, int32)
	cases := []struct {
		w, h int
		fn   copyFn
	}{
		{4, 4, WelsCopy4x4_c}, {8, 4, WelsCopy8x4_c}, {4, 8, WelsCopy4x8_c}, {8, 8, WelsCopy8x8_c},
		{8, 16, WelsCopy8x16_c}, {16, 8, WelsCopy16x8_c}, {16, 16, WelsCopy16x16_c},
	}
	rnd := rand.New(rand.NewSource(40))
	const ss, ds = 37, 21
	src := make([]uint8, ss*20)
	for i := range src {
		src[i] = uint8(rnd.Intn(256))
	}
	for _, c := range cases {
		dst := make([]uint8, ds*20)
		for i := range dst {
			dst[i] = 0xAA
		}
		c.fn(dst, ds+1, ds, src, 2*ss+3, ss)
		for j := 0; j < 20; j++ {
			for i := 0; i < ds; i++ {
				k := j*ds + i
				inside := k >= ds+1 && j-1 < c.h && j >= 1 && i-1 >= 0 && i-1 < c.w
				want := uint8(0xAA)
				if inside {
					want = src[2*ss+3+(j-1)*ss+(i-1)]
				}
				if dst[k] != want {
					t.Fatalf("WelsCopy%dx%d mismatch at (%d,%d)", c.w, c.h, i, j)
				}
			}
		}
	}
}

func TestI16x16LumaPred(t *testing.T) {
	rnd := rand.New(rand.NewSource(41))
	const stride = 40
	ref := make([]uint8, stride*20)
	for i := range ref {
		ref[i] = uint8(rnd.Intn(256))
	}
	refOff := 2*stride + 4
	pred := make([]uint8, 256+5)
	WelsI16x16LumaPredV_c(pred, 5, ref, refOff, stride)
	for j := 0; j < 16; j++ {
		for i := 0; i < 16; i++ {
			if pred[5+j*16+i] != ref[refOff-stride+i] {
				t.Fatalf("PredV mismatch at (%d,%d)", i, j)
			}
		}
	}
	WelsI16x16LumaPredH_c(pred, 5, ref, refOff, stride)
	for j := 0; j < 16; j++ {
		for i := 0; i < 16; i++ {
			if pred[5+j*16+i] != ref[refOff+j*stride-1] {
				t.Fatalf("PredH mismatch at (%d,%d)", i, j)
			}
		}
	}
}
