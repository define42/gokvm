//go:build amd64

package common

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestMcLumaQuarterpelSSE2MatchesC(t *testing.T) {
	rng := rand.New(rand.NewSource(0x51A7E))
	plane := newRandPlane(rng, 64, 64, 8)
	sizes := [][2]int32{{16, 16}, {16, 8}, {8, 16}, {8, 8}, {8, 4}, {4, 8}, {4, 4}}
	phases := [][2]int16{
		{0, 1}, {0, 3},
		{1, 0}, {1, 1}, {1, 2}, {1, 3},
		{2, 1}, {2, 3},
		{3, 0}, {3, 1}, {3, 2}, {3, 3},
	}
	const dstStride = int32(24)
	for iteration := 0; iteration < 8; iteration++ {
		for _, size := range sizes {
			for _, phase := range phases {
				x := 2 + rng.Intn(40)
				y := 2 + rng.Intn(40)
				srcOff := plane.org + y*plane.stride + x
				dstOff := 3
				want := bytes.Repeat([]byte{0xB7}, int(dstStride)*16+8)
				got := bytes.Clone(want)

				McLuma_c(plane.buf, srcOff, int32(plane.stride), want, dstOff, dstStride,
					phase[0], phase[1], size[0], size[1])
				mcLumaQuarterpel_sse2(plane.buf, srcOff, int32(plane.stride), got, dstOff, dstStride,
					phase[0], phase[1], size[0], size[1])
				if !bytes.Equal(got, want) {
					t.Fatalf("%dx%d phase(%d,%d) output differs", size[0], size[1], phase[0], phase[1])
				}
			}
		}
	}
}

func BenchmarkMcLumaQuarterpel16x16(b *testing.B) {
	const (
		width     = int32(16)
		height    = int32(16)
		srcStride = int32(64)
		dstStride = int32(32)
		pad       = 8
	)
	src := make([]byte, int(srcStride)*(int(height)+2*pad))
	dst := make([]byte, int(dstStride)*int(height))
	for i := range src {
		src[i] = byte(i*29 + 41)
	}
	srcOff := pad*int(srcStride) + pad
	benchmarks := []struct {
		name string
		fn   func([]uint8, int, int32, []uint8, int, int32, int16, int16, int32, int32)
	}{
		{name: "scalar/phase11", fn: McLuma_c},
		{name: "sse2/phase11", fn: mcLumaQuarterpel_sse2},
		{name: "scalar/phase12", fn: McLuma_c},
		{name: "sse2/phase12", fn: mcLumaQuarterpel_sse2},
	}
	for _, bm := range benchmarks {
		mvX, mvY := int16(1), int16(1)
		if bm.name[len(bm.name)-1] == '2' {
			mvY = 2
		}
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(width * height))
			for b.Loop() {
				bm.fn(src, srcOff, srcStride, dst, 0, dstStride, mvX, mvY, width, height)
			}
			mcSIMDBenchmarkSink = dst[0]
		})
	}
}
