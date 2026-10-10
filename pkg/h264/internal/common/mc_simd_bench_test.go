//go:build amd64

package common

import "testing"

var mcSIMDBenchmarkSink byte

func BenchmarkPixelAvg(b *testing.B) {
	const (
		width  = 16
		height = 16
		stride = 32
	)
	a := make([]byte, stride*height)
	c := make([]byte, stride*height)
	dst := make([]byte, stride*height)
	for i := range a {
		a[i] = byte(i*37 + 11)
		c[i] = byte(i*19 + 73)
	}
	benchmarks := []struct {
		name string
		fn   PWelsSampleAveragingFunc
	}{
		{name: "scalar", fn: PixelAvg_c},
		{name: "sse2", fn: PixelAvg_sse2},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(width * height)
			for b.Loop() {
				bm.fn(dst, 0, stride, a, 0, stride, c, 0, stride, width, height)
			}
			mcSIMDBenchmarkSink = dst[0]
		})
	}
}

func BenchmarkMcHalfpel17x17(b *testing.B) {
	const (
		width     = 17
		height    = 17
		srcStride = 48
		dstStride = 32
		pad       = 8
	)
	src := make([]byte, srcStride*(height+2*pad))
	dst := make([]byte, dstStride*height)
	for i := range src {
		src[i] = byte(i*29 + 41)
	}
	srcOff := pad*srcStride + pad
	benchmarks := []struct {
		name string
		fn   PWelsLumaHalfpelMcFunc
	}{
		{name: "horizontal/scalar", fn: McHorVer20_c},
		{name: "horizontal/sse2", fn: McHorVer20_sse2},
		{name: "vertical/scalar", fn: McHorVer02_c},
		{name: "vertical/sse2", fn: McHorVer02_sse2},
		{name: "center/scalar", fn: McHorVer22_c},
		{name: "center/sse2", fn: McHorVer22_sse2},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(width * height)
			for b.Loop() {
				bm.fn(src, srcOff, srcStride, dst, 0, dstStride, width, height)
			}
			mcSIMDBenchmarkSink = dst[0]
		})
	}
}

var mcFilterBenchmarkSink int32

func BenchmarkFilterInput8bitWithStride(b *testing.B) {
	const stride = 64
	src := make([]byte, stride*8)
	for i := range src {
		src[i] = byte(i*31 + 17)
	}
	off := 3*stride + 8
	benchmarks := []struct {
		name string
		fn   func([]uint8, int, int32) int32
	}{
		{name: "scalar", fn: FilterInput8bitWithStride_c},
		{name: "sse2-single", fn: FilterInput8bitWithStride_sse2},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			var result int32
			for b.Loop() {
				result = bm.fn(src, off, stride)
			}
			mcFilterBenchmarkSink = result
		})
	}
}
