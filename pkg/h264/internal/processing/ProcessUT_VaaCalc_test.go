package processing

// Port of test/processing/ProcessUT_VaaCalc.cpp: the _c functions are checked
// against an independent straightforward reference (flat C-style result
// arrays) on random data, including result entries the functions must leave
// untouched.

import (
	"math/rand"
	"testing"
)

const BUFFER_SIZE = 320 * 320

type vaaRefOut struct {
	frameSad                 int32
	sad8x8, sd8x8            []int32 // BUFFER_SIZE/64, (mb<<2)+k
	mad8x8                   []uint8 // BUFFER_SIZE/64
	sum16, sqsum16, sqdiff16 []int32 // BUFFER_SIZE/256
}

// vaaRef computes the requested statistics like the C reference functions do.
func vaaRef(cur, ref []uint8, w, h, stride int32, o *vaaRefOut, bSum, bSsd, bBgd bool) {
	o.frameSad = 0
	mbw, mbh := w>>4, h>>4
	for my := int32(0); my < mbh; my++ {
		for mx := int32(0); mx < mbw; mx++ {
			mb := my*mbw + mx
			var sum, sqsum, sqdiff int32
			for k := int32(0); k < 4; k++ {
				x0 := mx*16 + (k&1)*8
				y0 := my*16 + (k>>1)*8
				var sad, sd, mad int32
				for y := y0; y < y0+8; y++ {
					for x := x0; x < x0+8; x++ {
						c := int32(cur[y*stride+x])
						d := c - int32(ref[y*stride+x])
						ad := d
						if ad < 0 {
							ad = -ad
						}
						sad += ad
						sd += d
						if ad > mad {
							mad = ad
						}
						sqdiff += ad * ad
						sum += c
						sqsum += c * c
					}
				}
				o.frameSad += sad
				o.sad8x8[mb*4+k] = sad
				if bBgd {
					o.sd8x8[mb*4+k] = sd
					o.mad8x8[mb*4+k] = uint8(mad)
				}
			}
			if bSum {
				o.sum16[mb] = sum
				o.sqsum16[mb] = sqsum
			}
			if bSsd {
				o.sqdiff16[mb] = sqdiff
			}
		}
	}
}

type vaaGoOut struct {
	frameSad                 int32
	sad8x8, sd8x8            [][4]int32
	mad8x8                   [][4]uint8
	sum16, sqsum16, sqdiff16 []int32
}

func runVaaTest(t *testing.T, seed int64, bSum, bSsd, bBgd bool, call func(cur, ref []uint8, w, h, stride int32, o *vaaGoOut)) {
	r := rand.New(rand.NewSource(seed))
	cur := make([]uint8, BUFFER_SIZE)
	ref := make([]uint8, BUFFER_SIZE)
	for i := 0; i < 4; i++ {
		w := int32(320 - 16*i)
		const h, stride = 320, 320
		ro := &vaaRefOut{sad8x8: make([]int32, BUFFER_SIZE/64), sd8x8: make([]int32, BUFFER_SIZE/64),
			mad8x8: make([]uint8, BUFFER_SIZE/64), sum16: make([]int32, BUFFER_SIZE/256),
			sqsum16: make([]int32, BUFFER_SIZE/256), sqdiff16: make([]int32, BUFFER_SIZE/256)}
		go_ := &vaaGoOut{sad8x8: make([][4]int32, BUFFER_SIZE/256), sd8x8: make([][4]int32, BUFFER_SIZE/256),
			mad8x8: make([][4]uint8, BUFFER_SIZE/256), sum16: make([]int32, BUFFER_SIZE/256),
			sqsum16: make([]int32, BUFFER_SIZE/256), sqdiff16: make([]int32, BUFFER_SIZE/256)}
		for j := 0; j < BUFFER_SIZE; j++ {
			cur[j] = uint8(r.Intn(256))
			ref[j] = uint8(r.Intn(256))
			k := j % (BUFFER_SIZE / 64)
			v := int32(r.Intn(256))
			ro.sad8x8[k], go_.sad8x8[k/4][k%4] = v, v
			v = int32(r.Intn(256))
			ro.sd8x8[k], go_.sd8x8[k/4][k%4] = v, v
			v = int32(r.Intn(256))
			ro.mad8x8[k], go_.mad8x8[k/4][k%4] = uint8(v), uint8(v)
			m := j % (BUFFER_SIZE / 256)
			v = int32(r.Intn(256))
			ro.sum16[m], go_.sum16[m] = v, v
			v = int32(r.Intn(256))
			ro.sqsum16[m], go_.sqsum16[m] = v, v
			v = int32(r.Intn(256))
			ro.sqdiff16[m], go_.sqdiff16[m] = v, v
		}
		vaaRef(cur, ref, w, h, stride, ro, bSum, bSsd, bBgd)
		call(cur, ref, w, h, stride, go_)
		if ro.frameSad != go_.frameSad {
			t.Fatalf("w=%d: frame sad %d != %d", w, go_.frameSad, ro.frameSad)
		}
		for k := 0; k < BUFFER_SIZE/64; k++ {
			if go_.sad8x8[k/4][k%4] != ro.sad8x8[k] || go_.sd8x8[k/4][k%4] != ro.sd8x8[k] || go_.mad8x8[k/4][k%4] != ro.mad8x8[k] {
				t.Fatalf("w=%d: 8x8 mismatch at %d", w, k)
			}
		}
		for m := 0; m < BUFFER_SIZE/256; m++ {
			if go_.sum16[m] != ro.sum16[m] || go_.sqsum16[m] != ro.sqsum16[m] || go_.sqdiff16[m] != ro.sqdiff16[m] {
				t.Fatalf("w=%d: 16x16 mismatch at %d", w, m)
			}
		}
	}
}

func TestVAACalcFuncTest_VAACalcSad_c(t *testing.T) {
	runVaaTest(t, 10, false, false, false, func(cur, ref []uint8, w, h, stride int32, o *vaaGoOut) {
		VAACalcSad_c(cur, 0, ref, 0, w, h, stride, &o.frameSad, o.sad8x8)
	})
}

func TestVAACalcFuncTest_VAACalcSadBgd_c(t *testing.T) {
	runVaaTest(t, 11, false, false, true, func(cur, ref []uint8, w, h, stride int32, o *vaaGoOut) {
		VAACalcSadBgd_c(cur, 0, ref, 0, w, h, stride, &o.frameSad, o.sad8x8, o.sd8x8, o.mad8x8)
	})
}

func TestVAACalcFuncTest_VAACalcSadSsdBgd_c(t *testing.T) {
	runVaaTest(t, 12, true, true, true, func(cur, ref []uint8, w, h, stride int32, o *vaaGoOut) {
		VAACalcSadSsdBgd_c(cur, 0, ref, 0, w, h, stride, &o.frameSad, o.sad8x8, o.sum16, o.sqsum16, o.sqdiff16, o.sd8x8, o.mad8x8)
	})
}

func TestVAACalcFuncTest_VAACalcSadSsd_c(t *testing.T) {
	runVaaTest(t, 13, true, true, false, func(cur, ref []uint8, w, h, stride int32, o *vaaGoOut) {
		VAACalcSadSsd_c(cur, 0, ref, 0, w, h, stride, &o.frameSad, o.sad8x8, o.sum16, o.sqsum16, o.sqdiff16)
	})
}

func TestVAACalcFuncTest_VAACalcSadVar_c(t *testing.T) {
	runVaaTest(t, 14, true, false, false, func(cur, ref []uint8, w, h, stride int32, o *vaaGoOut) {
		VAACalcSadVar_c(cur, 0, ref, 0, w, h, stride, &o.frameSad, o.sad8x8, o.sum16, o.sqsum16)
	})
}
