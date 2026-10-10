//go:build amd64

package common

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
)

func TestFilterInput8bitWithStrideSSE2MatchesC(t *testing.T) {
	rng := rand.New(rand.NewSource(0x264))
	for _, stride := range []int32{1, 3, 17, 37} {
		buf := make([]byte, 1024)
		for i := range buf {
			buf[i] = byte(rng.Intn(256))
		}
		first := int(2 * stride)
		last := len(buf) - int(3*stride)
		for n := 0; n < 200; n++ {
			off := first + rng.Intn(last-first)
			want := FilterInput8bitWithStride_c(buf, off, stride)
			got := FilterInput8bitWithStride_sse2(buf, off, stride)
			if got != want {
				t.Fatalf("stride=%d off=%d: got %d, want %d", stride, off, got, want)
			}
		}
	}

	for _, value := range []byte{0, 255} {
		buf := bytes.Repeat([]byte{value}, 64)
		if got, want := FilterInput8bitWithStride_sse2(buf, 24, 7),
			FilterInput8bitWithStride_c(buf, 24, 7); got != want {
			t.Fatalf("constant %d: got %d, want %d", value, got, want)
		}
	}
}

func TestPixelAvgSSE2MatchesC(t *testing.T) {
	rng := rand.New(rand.NewSource(0xA66))
	widths := []int32{1, 2, 3, 4, 5, 7, 8, 9, 15, 16, 17, 31}
	heights := []int32{1, 3, 4, 5, 8, 15, 17}
	for _, width := range widths {
		for _, height := range heights {
			aStride := int(width) + 11
			bStride := int(width) + 13
			dstStride := int(width) + 9
			aOff, bOff, dstOff := 5, 7, 3
			a := make([]byte, aOff+aStride*int(height)+16)
			b := make([]byte, bOff+bStride*int(height)+16)
			for i := range a {
				a[i] = byte(rng.Intn(256))
			}
			for i := range b {
				b[i] = byte(rng.Intn(256))
			}
			aBefore := bytes.Clone(a)
			bBefore := bytes.Clone(b)
			want := bytes.Repeat([]byte{0xA5}, dstOff+dstStride*int(height)+16)
			got := bytes.Clone(want)

			PixelAvg_c(want, dstOff, int32(dstStride), a, aOff, int32(aStride),
				b, bOff, int32(bStride), width, height)
			PixelAvg_sse2(got, dstOff, int32(dstStride), a, aOff, int32(aStride),
				b, bOff, int32(bStride), width, height)

			if !bytes.Equal(got, want) {
				t.Fatalf("%dx%d output differs", width, height)
			}
			if !bytes.Equal(a, aBefore) || !bytes.Equal(b, bBefore) {
				t.Fatalf("%dx%d modified a source", width, height)
			}
		}
	}
}

func TestMcHalfpelSSE2MatchesC(t *testing.T) {
	type kernel struct {
		name string
		c    PWelsLumaHalfpelMcFunc
		simd PWelsLumaHalfpelMcFunc
	}
	kernels := []kernel{
		{name: "horizontal", c: McHorVer20_c, simd: McHorVer20_sse2},
		{name: "vertical", c: McHorVer02_c, simd: McHorVer02_sse2},
		{name: "center", c: McHorVer22_c, simd: McHorVer22_sse2},
	}
	rng := rand.New(rand.NewSource(0x6A7))
	widths := []int32{1, 4, 5, 8, 9, 16, 17}
	heights := []int32{1, 4, 5, 8, 9, 16, 17}
	for _, k := range kernels {
		t.Run(k.name, func(t *testing.T) {
			for _, width := range widths {
				for _, height := range heights {
					const pad = 8
					srcStride := int(width) + 2*pad + 7
					srcRows := int(height) + 2*pad
					src := make([]byte, srcStride*srcRows)
					for i := range src {
						src[i] = byte(rng.Intn(256))
					}
					// Exercise the filter's clipping paths deterministically.
					for i := 0; i < len(src); i += 11 {
						if i&1 == 0 {
							src[i] = 0
						} else {
							src[i] = 255
						}
					}
					srcBefore := bytes.Clone(src)
					srcOff := pad*srcStride + pad + 1
					dstStride := int(width) + 9
					dstOff := 3
					want := bytes.Repeat([]byte{0xD3}, dstOff+dstStride*int(height)+16)
					got := bytes.Clone(want)

					k.c(src, srcOff, int32(srcStride), want, dstOff, int32(dstStride), width, height)
					k.simd(src, srcOff, int32(srcStride), got, dstOff, int32(dstStride), width, height)

					if !bytes.Equal(got, want) {
						t.Fatalf("%dx%d output differs", width, height)
					}
					if !bytes.Equal(src, srcBefore) {
						t.Fatalf("%dx%d modified source", width, height)
					}
				}
			}
		})
	}
}

func TestMcSIMDDispatch(t *testing.T) {
	pointer := func(fn any) uintptr { return reflect.ValueOf(fn).Pointer() }
	var funcs SMcFunc

	InitMcFunc(&funcs, 0)
	if pointer(funcs.PfLumaHalfpelHor) != pointer(McHorVer20_c) ||
		pointer(funcs.PfLumaHalfpelVer) != pointer(McHorVer02_c) ||
		pointer(funcs.PfLumaHalfpelCen) != pointer(McHorVer22_c) ||
		pointer(funcs.PfSampleAveraging) != pointer(PixelAvg_c) ||
		pointer(funcs.PMcLumaFunc) != pointer(McLuma_c) {
		t.Fatal("scalar dispatch did not retain scalar motion-compensation kernels")
	}

	InitMcFunc(&funcs, WELS_CPU_SSE2)
	if pointer(funcs.PfLumaHalfpelHor) != pointer(McHorVer20_sse2) ||
		pointer(funcs.PfLumaHalfpelVer) != pointer(McHorVer02_sse2) ||
		pointer(funcs.PfLumaHalfpelCen) != pointer(McHorVer22_sse2) ||
		pointer(funcs.PfSampleAveraging) != pointer(PixelAvg_sse2) ||
		pointer(funcs.PMcLumaFunc) != pointer(McLuma_sse2) {
		t.Fatal("SSE2 dispatch did not install SIMD motion-compensation kernels")
	}
}

func TestMcSIMDZeroDimensions(t *testing.T) {
	src := make([]byte, 64)
	dst := bytes.Repeat([]byte{0x7E}, 64)
	want := bytes.Clone(dst)
	PixelAvg_sse2(dst, 0, 8, src, 0, 8, src, 0, 8, 0, 8)
	McHorVer20_sse2(src, 16, 8, dst, 0, 8, -1, 8)
	McHorVer02_sse2(src, 16, 8, dst, 0, 8, 8, -1)
	if !bytes.Equal(dst, want) {
		t.Fatal("zero or negative dimensions modified destination")
	}
}

func TestMcLumaSSE2MatchesC(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1A2B3C))
	plane := newRandPlane(rng, 64, 64, 8)
	sizes := [][2]int32{{16, 16}, {16, 8}, {8, 16}, {8, 8}, {8, 4}, {4, 8}, {4, 4}}
	const dstStride = 24
	for iteration := 0; iteration < 8; iteration++ {
		for _, size := range sizes {
			for mvX := -4; mvX < 4; mvX++ {
				for mvY := -4; mvY < 4; mvY++ {
					x := 2 + rng.Intn(40)
					y := 2 + rng.Intn(40)
					srcOff := plane.org + y*plane.stride + x
					dstOff := 3
					want := bytes.Repeat([]byte{0xB7}, dstStride*16+8)
					got := bytes.Clone(want)
					McLuma_c(plane.buf, srcOff, int32(plane.stride), want, dstOff, dstStride,
						int16(mvX), int16(mvY), size[0], size[1])
					McLuma_sse2(plane.buf, srcOff, int32(plane.stride), got, dstOff, dstStride,
						int16(mvX), int16(mvY), size[0], size[1])
					if !bytes.Equal(got, want) {
						t.Fatalf("%dx%d mv(%d,%d) output differs", size[0], size[1], mvX, mvY)
					}
				}
			}
		}
	}
}
