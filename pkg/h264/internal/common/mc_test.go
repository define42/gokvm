package common

import (
	"math/rand"
	"testing"
)

// Reference H.264 luma/chroma interpolation written directly from the
// specification (8.4.2.2.1 / 8.4.2.2.2), independent of mc.go.

func refClip1(x int) uint8 {
	if x < 0 {
		return 0
	}
	if x > 255 {
		return 255
	}
	return uint8(x)
}

type refPlane struct {
	buf    []uint8
	stride int
	org    int // offset of sample (0,0)
}

func (p *refPlane) at(x, y int) int { return int(p.buf[p.org+y*p.stride+x]) }

func tap6(e, f, g, h, i, j int) int { return e - 5*f + 20*g + 20*h - 5*i + j }

// b1: horizontal intermediate at half position between (x,y) and (x+1,y).
func (p *refPlane) b1(x, y int) int {
	return tap6(p.at(x-2, y), p.at(x-1, y), p.at(x, y), p.at(x+1, y), p.at(x+2, y), p.at(x+3, y))
}

// h1: vertical intermediate at half position between (x,y) and (x,y+1).
func (p *refPlane) h1(x, y int) int {
	return tap6(p.at(x, y-2), p.at(x, y-1), p.at(x, y), p.at(x, y+1), p.at(x, y+2), p.at(x, y+3))
}

// j1: center intermediate, computed vertically from the b1 values.
func (p *refPlane) j1(x, y int) int {
	return tap6(p.b1(x, y-2), p.b1(x, y-1), p.b1(x, y), p.b1(x, y+1), p.b1(x, y+2), p.b1(x, y+3))
}

func (p *refPlane) lumaSample(x, y, xFrac, yFrac int) uint8 {
	G := p.at(x, y)
	H := p.at(x+1, y)
	M := p.at(x, y+1)
	b := int(refClip1((p.b1(x, y) + 16) >> 5))
	h := int(refClip1((p.h1(x, y) + 16) >> 5))
	s := int(refClip1((p.b1(x, y+1) + 16) >> 5))
	m := int(refClip1((p.h1(x+1, y) + 16) >> 5))
	j := int(refClip1((p.j1(x, y) + 512) >> 10))
	avg := func(a, b int) uint8 { return uint8((a + b + 1) >> 1) }
	switch xFrac<<2 | yFrac {
	case 0<<2 | 0:
		return uint8(G)
	case 1<<2 | 0:
		return avg(G, b) // a
	case 2<<2 | 0:
		return uint8(b)
	case 3<<2 | 0:
		return avg(H, b) // c
	case 0<<2 | 1:
		return avg(G, h) // d
	case 1<<2 | 1:
		return avg(b, h) // e
	case 2<<2 | 1:
		return avg(b, j) // f
	case 3<<2 | 1:
		return avg(b, m) // g
	case 0<<2 | 2:
		return uint8(h)
	case 1<<2 | 2:
		return avg(h, j) // i
	case 2<<2 | 2:
		return uint8(j)
	case 3<<2 | 2:
		return avg(j, m) // k
	case 0<<2 | 3:
		return avg(M, h) // n
	case 1<<2 | 3:
		return avg(h, s) // p
	case 2<<2 | 3:
		return avg(j, s) // q
	default:
		return avg(m, s) // r
	}
}

func (p *refPlane) chromaSample(x, y, xFrac, yFrac int) uint8 {
	A, B := p.at(x, y), p.at(x+1, y)
	C, D := p.at(x, y+1), p.at(x+1, y+1)
	return uint8(((8-xFrac)*(8-yFrac)*A + xFrac*(8-yFrac)*B + (8-xFrac)*yFrac*C + xFrac*yFrac*D + 32) >> 6)
}

func newRandPlane(rnd *rand.Rand, w, h, pad int) *refPlane {
	stride := w + 2*pad
	buf := make([]uint8, stride*(h+2*pad))
	for i := range buf {
		buf[i] = uint8(rnd.Intn(256))
	}
	// add some extreme values to exercise clipping
	for i := 0; i < len(buf)/8; i++ {
		if rnd.Intn(2) == 0 {
			buf[rnd.Intn(len(buf))] = 0
		} else {
			buf[rnd.Intn(len(buf))] = 255
		}
	}
	return &refPlane{buf: buf, stride: stride, org: pad*stride + pad}
}

func TestMcLuma(t *testing.T) {
	rnd := rand.New(rand.NewSource(10))
	var sMcFunc SMcFunc
	InitMcFunc(&sMcFunc, 0)
	p := newRandPlane(rnd, 64, 64, 8)
	sizes := [][2]int32{{16, 16}, {16, 8}, {8, 16}, {8, 8}, {8, 4}, {4, 8}, {4, 4}}
	const dstStride = 24
	for iter := 0; iter < 20; iter++ {
		for _, sz := range sizes {
			for mvx := -4; mvx < 4; mvx++ {
				for mvy := -4; mvy < 4; mvy++ {
					x0 := 2 + rnd.Intn(40)
					y0 := 2 + rnd.Intn(40)
					dst := make([]uint8, dstStride*16+8)
					dstOff := 3
					iSrcOff := p.org + y0*p.stride + x0
					sMcFunc.PMcLumaFunc(p.buf, iSrcOff, int32(p.stride), dst, dstOff, dstStride, int16(mvx), int16(mvy), sz[0], sz[1])
					for j := 0; j < int(sz[1]); j++ {
						for i := 0; i < int(sz[0]); i++ {
							want := p.lumaSample(x0+i, y0+j, mvx&3, mvy&3)
							if got := dst[dstOff+j*dstStride+i]; got != want {
								t.Fatalf("luma %dx%d mv(%d,%d) at (%d,%d): got %d want %d", sz[0], sz[1], mvx, mvy, i, j, got, want)
							}
						}
					}
				}
			}
		}
	}
}

func TestMcChroma(t *testing.T) {
	rnd := rand.New(rand.NewSource(11))
	var sMcFunc SMcFunc
	InitMcFunc(&sMcFunc, 0)
	p := newRandPlane(rnd, 32, 32, 4)
	sizes := [][2]int32{{8, 8}, {8, 4}, {4, 8}, {4, 4}, {4, 2}, {2, 4}, {2, 2}}
	const dstStride = 16
	for iter := 0; iter < 20; iter++ {
		for _, sz := range sizes {
			for mvx := -8; mvx < 8; mvx++ {
				for mvy := -8; mvy < 8; mvy++ {
					x0 := rnd.Intn(20)
					y0 := rnd.Intn(20)
					dst := make([]uint8, dstStride*8+4)
					dstOff := 1
					iSrcOff := p.org + y0*p.stride + x0
					sMcFunc.PMcChromaFunc(p.buf, iSrcOff, int32(p.stride), dst, dstOff, dstStride, int16(mvx), int16(mvy), sz[0], sz[1])
					for j := 0; j < int(sz[1]); j++ {
						for i := 0; i < int(sz[0]); i++ {
							want := p.chromaSample(x0+i, y0+j, mvx&7, mvy&7)
							if got := dst[dstOff+j*dstStride+i]; got != want {
								t.Fatalf("chroma %dx%d mv(%d,%d) at (%d,%d): got %d want %d", sz[0], sz[1], mvx, mvy, i, j, got, want)
							}
						}
					}
				}
			}
		}
	}
	// G_kuiABCD must match the formula
	for dy := 0; dy < 8; dy++ {
		for dx := 0; dx < 8; dx++ {
			w := G_kuiABCD[dy][dx]
			if int(w[0]) != (8-dx)*(8-dy) || int(w[1]) != dx*(8-dy) || int(w[2]) != (8-dx)*dy || int(w[3]) != dx*dy {
				t.Fatalf("G_kuiABCD[%d][%d] = %v", dy, dx, w)
			}
		}
	}
}

// The half-pel functions are called by the encoder with widths/heights of
// 5, 9 and 17 (block size + 1).
func TestMcLumaHalfpel(t *testing.T) {
	rnd := rand.New(rand.NewSource(12))
	var sMcFunc SMcFunc
	InitMcFunc(&sMcFunc, 0)
	p := newRandPlane(rnd, 48, 48, 8)
	const dstStride = 32
	type hp struct {
		fn           PWelsLumaHalfpelMcFunc
		xFrac, yFrac int
	}
	fns := []hp{{sMcFunc.PfLumaHalfpelHor, 2, 0}, {sMcFunc.PfLumaHalfpelVer, 0, 2}, {sMcFunc.PfLumaHalfpelCen, 2, 2}}
	for iter := 0; iter < 20; iter++ {
		for _, f := range fns {
			for _, w := range []int32{4, 5, 8, 9, 16, 17} {
				for _, h := range []int32{4, 5, 8, 9, 16, 17} {
					x0 := rnd.Intn(28)
					y0 := rnd.Intn(28)
					dst := make([]uint8, dstStride*17)
					f.fn(p.buf, p.org+y0*p.stride+x0, int32(p.stride), dst, 0, dstStride, w, h)
					for j := 0; j < int(h); j++ {
						for i := 0; i < int(w); i++ {
							want := p.lumaSample(x0+i, y0+j, f.xFrac, f.yFrac)
							if got := dst[j*dstStride+i]; got != want {
								t.Fatalf("halfpel (%d,%d) %dx%d at (%d,%d): got %d want %d", f.xFrac, f.yFrac, w, h, i, j, got, want)
							}
						}
					}
				}
			}
		}
	}
}

func TestPixelAvg(t *testing.T) {
	rnd := rand.New(rand.NewSource(13))
	var sMcFunc SMcFunc
	InitMcFunc(&sMcFunc, 0)
	a := make([]uint8, 32*16)
	b := make([]uint8, 24*16)
	for i := range a {
		a[i] = uint8(rnd.Intn(256))
	}
	for i := range b {
		b[i] = uint8(rnd.Intn(256))
	}
	dst := make([]uint8, 20*16)
	sMcFunc.PfSampleAveraging(dst, 2, 20, a, 1, 32, b, 3, 24, 16, 15)
	for j := 0; j < 15; j++ {
		for i := 0; i < 16; i++ {
			want := uint8((int(a[1+j*32+i]) + int(b[3+j*24+i]) + 1) >> 1)
			if got := dst[2+j*20+i]; got != want {
				t.Fatalf("PixelAvg at (%d,%d): got %d want %d", i, j, got, want)
			}
		}
	}
}
