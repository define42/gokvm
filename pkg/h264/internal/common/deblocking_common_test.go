package common

import (
	"bytes"
	"math/rand"
	"testing"
)

// Reference deblocking filters written from the H.264 specification
// (8.7.2.3 / 8.7.2.4), independent of deblocking_common.go. Each call
// filters one line of samples across an edge: s(k) addresses p_k for k < 0
// as s(-1-k) style offsets (p0 = -1, p1 = -2, ... q0 = 0, q1 = 1, ...).

func refAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func refClip3(lo, hi, x int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// refFilterLine filters one line; get/set address sample at position k
// relative to the edge (k=-1 is p0, k=0 is q0).
func refFilterLine(get func(int) int, set func(int, int), alpha, beta int, bS4 bool, tc0 int, chroma bool) {
	p0, p1, p2 := get(-1), get(-2), get(-3)
	q0, q1, q2 := get(0), get(1), get(2)
	if !(refAbs(p0-q0) < alpha && refAbs(p1-p0) < beta && refAbs(q1-q0) < beta) {
		return
	}
	ap := refAbs(p2 - p0)
	aq := refAbs(q2 - q0)
	if !bS4 {
		var tc int
		if chroma {
			tc = tc0 // the C code receives tC0+1 for chroma already
		} else {
			tc = tc0
			if ap < beta {
				tc++
			}
			if aq < beta {
				tc++
			}
		}
		delta := refClip3(-tc, tc, (((q0-p0)<<2)+(p1-q1)+4)>>3)
		set(-1, int(refClip1(p0+delta)))
		set(0, int(refClip1(q0-delta)))
		if !chroma {
			if ap < beta {
				set(-2, p1+refClip3(-tc0, tc0, (p2+((p0+q0+1)>>1)-(p1<<1))>>1))
			}
			if aq < beta {
				set(1, q1+refClip3(-tc0, tc0, (q2+((p0+q0+1)>>1)-(q1<<1))>>1))
			}
		}
		return
	}
	if chroma {
		set(-1, (2*p1+p0+q1+2)>>2)
		set(0, (2*q1+q0+p1+2)>>2)
		return
	}
	strong := refAbs(p0-q0) < ((alpha >> 2) + 2)
	if strong && ap < beta {
		p3 := get(-4)
		set(-1, (p2+2*p1+2*p0+2*q0+q1+4)>>3)
		set(-2, (p2+p1+p0+q0+2)>>2)
		set(-3, (2*p3+3*p2+p1+p0+q0+4)>>3)
	} else {
		set(-1, (2*p1+p0+q1+2)>>2)
	}
	if strong && aq < beta {
		q3 := get(3)
		set(0, (p1+2*p0+2*q0+2*q1+q2+4)>>3)
		set(1, (p0+q0+q1+q2+2)>>2)
		set(2, (2*q3+3*q2+q1+q0+p0+4)>>3)
	} else {
		set(0, (2*q1+q0+p1+2)>>2)
	}
}

// refDeblock filters nLines lines starting at off; vertical selects a
// horizontal edge (samples across the edge are `stride` apart).
func refDeblock(buf []uint8, off, stride int, vertical bool, nLines int, alpha, beta int, bS4 bool, tc []int8, chroma bool) {
	orig := append([]uint8(nil), buf...)
	sx, sy := 1, stride
	if vertical {
		sx, sy = stride, 1
	}
	for l := 0; l < nLines; l++ {
		base := off + l*sy
		tc0 := 0
		if !bS4 {
			if chroma {
				tc0 = int(tc[l>>1])
				if tc0 <= 0 {
					continue
				}
			} else {
				tc0 = int(tc[l>>2])
				if tc0 < 0 {
					continue
				}
			}
		}
		get := func(k int) int { return int(orig[base+k*sx]) }
		set := func(k int, v int) { buf[base+k*sx] = uint8(v) }
		refFilterLine(get, set, alpha, beta, bS4, tc0, chroma)
	}
}

func randEdgeBlock(rnd *rand.Rand, n int) []uint8 {
	b := make([]uint8, n)
	base := rnd.Intn(256)
	spread := 1 + rnd.Intn(40)
	for i := range b {
		v := base + rnd.Intn(2*spread+1) - spread
		if rnd.Intn(50) == 0 {
			v = rnd.Intn(256)
		}
		b[i] = refClip1(v)
	}
	return b
}

func TestDeblocking(t *testing.T) {
	rnd := rand.New(rand.NewSource(20))
	const stride = 32
	const off = 8*stride + 8
	changed := 0
	defer func() {
		if !t.Failed() && changed < 500 {
			t.Fatalf("luma filters modified samples in only %d iterations", changed)
		}
	}()
	for iter := 0; iter < 2000; iter++ {
		alpha := int32(rnd.Intn(64))
		beta := int32(rnd.Intn(24))
		var tc [4]int8
		for i := range tc {
			tc[i] = int8(rnd.Intn(28) - 1)
		}
		for _, vertical := range []bool{true, false} {
			// luma bS < 4
			a := randEdgeBlock(rnd, stride*32)
			b := append([]uint8(nil), a...)
			if vertical {
				DeblockLumaLt4V_c(a, off, stride, alpha, beta, tc[:])
			} else {
				DeblockLumaLt4H_c(a, off, stride, alpha, beta, tc[:])
			}
			refDeblock(b, off, stride, vertical, 16, int(alpha), int(beta), false, tc[:], false)
			if !bytes.Equal(a, b) {
				t.Fatalf("DeblockLumaLt4 vertical=%v alpha=%d beta=%d tc=%v", vertical, alpha, beta, tc)
			}
			// luma bS == 4
			a = randEdgeBlock(rnd, stride*32)
			b = append([]uint8(nil), a...)
			if vertical {
				DeblockLumaEq4V_c(a, off, stride, alpha, beta)
			} else {
				DeblockLumaEq4H_c(a, off, stride, alpha, beta)
			}
			orig := append([]uint8(nil), b...)
			refDeblock(b, off, stride, vertical, 16, int(alpha), int(beta), true, nil, false)
			if !bytes.Equal(orig, b) {
				changed++
			}
			if !bytes.Equal(a, b) {
				t.Fatalf("DeblockLumaEq4 vertical=%v alpha=%d beta=%d", vertical, alpha, beta)
			}
			// chroma (separate planes) bS < 4 and == 4
			for _, bS4 := range []bool{false, true} {
				cb := randEdgeBlock(rnd, stride*16)
				cr := randEdgeBlock(rnd, stride*16)
				cbRef := append([]uint8(nil), cb...)
				crRef := append([]uint8(nil), cr...)
				const coff = 4*stride + 4
				switch {
				case !bS4 && vertical:
					DeblockChromaLt4V_c(cb, coff, cr, coff, stride, alpha, beta, tc[:])
				case !bS4 && !vertical:
					DeblockChromaLt4H_c(cb, coff, cr, coff, stride, alpha, beta, tc[:])
				case bS4 && vertical:
					DeblockChromaEq4V_c(cb, coff, cr, coff, stride, alpha, beta)
				default:
					DeblockChromaEq4H_c(cb, coff, cr, coff, stride, alpha, beta)
				}
				refDeblock(cbRef, coff, stride, vertical, 8, int(alpha), int(beta), bS4, tc[:], true)
				refDeblock(crRef, coff, stride, vertical, 8, int(alpha), int(beta), bS4, tc[:], true)
				if !bytes.Equal(cb, cbRef) || !bytes.Equal(cr, crRef) {
					t.Fatalf("DeblockChroma bS4=%v vertical=%v alpha=%d beta=%d tc=%v", bS4, vertical, alpha, beta, tc)
				}
				// single plane (2) variants
				c2 := randEdgeBlock(rnd, stride*16)
				c2Ref := append([]uint8(nil), c2...)
				switch {
				case !bS4 && vertical:
					DeblockChromaLt4V2_c(c2, coff, stride, alpha, beta, tc[:])
				case !bS4 && !vertical:
					DeblockChromaLt4H2_c(c2, coff, stride, alpha, beta, tc[:])
				case bS4 && vertical:
					DeblockChromaEq4V2_c(c2, coff, stride, alpha, beta)
				default:
					DeblockChromaEq4H2_c(c2, coff, stride, alpha, beta)
				}
				refDeblock(c2Ref, coff, stride, vertical, 8, int(alpha), int(beta), bS4, tc[:], true)
				if !bytes.Equal(c2, c2Ref) {
					t.Fatalf("DeblockChroma2 bS4=%v vertical=%v alpha=%d beta=%d tc=%v", bS4, vertical, alpha, beta, tc)
				}
			}
		}
	}
}

func TestWelsNonZeroCount(t *testing.T) {
	var nz [24]int8
	for i := range nz {
		nz[i] = int8(i%5 - 1)
	}
	WelsNonZeroCount_c(nz[:])
	for i, v := range nz {
		want := int8(0)
		if i%5-1 != 0 {
			want = 1
		}
		if v != want {
			t.Fatalf("nz[%d]=%d want %d", i, v, want)
		}
	}
}
