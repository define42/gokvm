// Port of the C-reference parts of test/decoder/DecUT_IntraPrediction.cpp.
//
// The SIMD comparisons are not ported. The C test's reference functions are
// either generic macros (PREDV/PREDH/PREDDC...) or copies of the production
// code; the latter are written here directly from the H.264 equations
// (8.3.1.2.x / 8.3.3 / 8.3.4) so they are an independent check.

package decoder

import (
	"math/rand"
	"testing"
)

type intraPredFunc func(pPred []uint8, iPredOff int, kiStride int32)

// ---- generic references (PREDV / PREDH / PREDDC* macros) ----

func refPredV(size int) intraPredFunc {
	return func(p []uint8, o int, kiStride int32) {
		s := int(kiStride)
		for i := 0; i < size; i++ {
			copy(p[o+i*s:o+i*s+size], p[o-s:o-s+size])
		}
	}
}

func refPredH(size int) intraPredFunc {
	return func(p []uint8, o int, kiStride int32) {
		s := int(kiStride)
		for i := 0; i < size; i++ {
			v := p[o+i*s-1]
			for j := 0; j < size; j++ {
				p[o+i*s+j] = v
			}
		}
	}
}

func refFill(p []uint8, o int, s int, size int, v uint8) {
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			p[o+i*s+j] = v
		}
	}
}

func refPredDC(size, log int) intraPredFunc {
	return func(p []uint8, o int, kiStride int32) {
		s := int(kiStride)
		iSum := size
		for i := 0; i < size; i++ {
			iSum += int(p[o-1+i*s]) + int(p[o+i-s])
		}
		refFill(p, o, s, size, uint8(iSum>>(log+1)))
	}
}

func refPredDCLeft(size, log int) intraPredFunc {
	return func(p []uint8, o int, kiStride int32) {
		s := int(kiStride)
		iSum := size / 2
		for i := 0; i < size; i++ {
			iSum += int(p[o-1+i*s])
		}
		refFill(p, o, s, size, uint8(iSum>>log))
	}
}

func refPredDCTop(size, log int) intraPredFunc {
	return func(p []uint8, o int, kiStride int32) {
		s := int(kiStride)
		iSum := size / 2
		for i := 0; i < size; i++ {
			iSum += int(p[o+i-s])
		}
		refFill(p, o, s, size, uint8(iSum>>log))
	}
}

func refPredDCNone(size int) intraPredFunc {
	return func(p []uint8, o int, kiStride int32) {
		refFill(p, o, int(kiStride), size, 128)
	}
}

// ---- 4x4 directional references from the standard ----

// ref4x4 builds a 4x4 reference from a per-sample function. top(x) is
// p[x,-1] (x = -1 is the top-left sample), left(y) is p[-1,y] (y = -1 is the
// top-left sample).
func ref4x4(f func(x, y int, top, left func(int) int) int, topRightAsT3 bool) intraPredFunc {
	return func(p []uint8, o int, kiStride int32) {
		s := int(kiStride)
		top := func(x int) int {
			if topRightAsT3 && x > 3 {
				x = 3
			}
			return int(p[o-s+x])
		}
		left := func(y int) int {
			if y < 0 {
				return int(p[o-s-1])
			}
			return int(p[o+y*s-1])
		}
		var out [4][4]uint8
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				out[y][x] = uint8(f(x, y, top, left))
			}
		}
		for y := 0; y < 4; y++ {
			copy(p[o+y*s:o+y*s+4], out[y][:])
		}
	}
}

func predDDL4x4(x, y int, top, left func(int) int) int {
	if x == 3 && y == 3 {
		return (top(6) + 3*top(7) + 2) >> 2
	}
	return (top(x+y) + 2*top(x+y+1) + top(x+y+2) + 2) >> 2
}

func predDDR4x4(x, y int, top, left func(int) int) int {
	switch {
	case x > y:
		return (top(x-y-2) + 2*top(x-y-1) + top(x-y) + 2) >> 2
	case x < y:
		return (left(y-x-2) + 2*left(y-x-1) + left(y-x) + 2) >> 2
	default:
		return (top(0) + 2*top(-1) + left(0) + 2) >> 2
	}
}

func predVR4x4(x, y int, top, left func(int) int) int {
	zVR := 2*x - y
	switch {
	case zVR >= 0 && zVR&1 == 0:
		return (top(x-(y>>1)-1) + top(x-(y>>1)) + 1) >> 1
	case zVR >= 0:
		return (top(x-(y>>1)-2) + 2*top(x-(y>>1)-1) + top(x-(y>>1)) + 2) >> 2
	case zVR == -1:
		return (left(0) + 2*left(-1) + top(0) + 2) >> 2
	default:
		return (left(y-1) + 2*left(y-2) + left(y-3) + 2) >> 2
	}
}

func predVL4x4(x, y int, top, left func(int) int) int {
	if y&1 == 0 {
		return (top(x+(y>>1)) + top(x+(y>>1)+1) + 1) >> 1
	}
	return (top(x+(y>>1)) + 2*top(x+(y>>1)+1) + top(x+(y>>1)+2) + 2) >> 2
}

func predHU4x4(x, y int, top, left func(int) int) int {
	zHU := x + 2*y
	switch {
	case zHU < 5 && zHU&1 == 0:
		return (left(y+(x>>1)) + left(y+(x>>1)+1) + 1) >> 1
	case zHU < 5:
		return (left(y+(x>>1)) + 2*left(y+(x>>1)+1) + left(y+(x>>1)+2) + 2) >> 2
	case zHU == 5:
		return (left(2) + 3*left(3) + 2) >> 2
	default:
		return left(3)
	}
}

func predHD4x4(x, y int, top, left func(int) int) int {
	zHD := 2*y - x
	switch {
	case zHD >= 0 && zHD&1 == 0:
		return (left(y-(x>>1)-1) + left(y-(x>>1)) + 1) >> 1
	case zHD >= 0:
		return (left(y-(x>>1)-2) + 2*left(y-(x>>1)-1) + left(y-(x>>1)) + 2) >> 2
	case zHD == -1:
		return (left(0) + 2*left(-1) + top(0) + 2) >> 2
	default:
		return (top(x-1) + 2*top(x-2) + top(x-3) + 2) >> 2
	}
}

// ---- chroma / 16x16 references ----

func WelsIChromaPredPlane_ref(p []uint8, o int, kiStride int32) {
	s := int(kiStride)
	var a, b, c, H, V int32
	pTop := o - s
	pLeft := o - 1
	for i := 0; i < 4; i++ {
		H += int32(i+1) * (int32(p[pTop+4+i]) - int32(p[pTop+2-i]))
		V += int32(i+1) * (int32(p[pLeft+(4+i)*s]) - int32(p[pLeft+(2-i)*s]))
	}
	a = (int32(p[pLeft+7*s]) + int32(p[pTop+7])) << 4
	b = (17*H + 16) >> 5
	c = (17*V + 16) >> 5
	for i := int32(0); i < 8; i++ {
		for j := int32(0); j < 8; j++ {
			iTmp := (a + b*(j-3) + c*(i-3) + 16) >> 5
			if iTmp < 0 {
				iTmp = 0
			} else if iTmp > 255 {
				iTmp = 255
			}
			p[o+int(i)*s+int(j)] = uint8(iTmp)
		}
	}
}

// chroma DC per 4x4 block (8.3.4.1-3): blocks (0,0) and (1,1) use top and
// left, block (1,0) the top only and block (0,1) the left only.
func WelsIChromaPredDc_ref(p []uint8, o int, kiStride int32) {
	s := int(kiStride)
	sumT := func(x0 int) int {
		v := 0
		for x := x0; x < x0+4; x++ {
			v += int(p[o-s+x])
		}
		return v
	}
	sumL := func(y0 int) int {
		v := 0
		for y := y0; y < y0+4; y++ {
			v += int(p[o+y*s-1])
		}
		return v
	}
	m := [2][2]uint8{
		{uint8((sumT(0) + sumL(0) + 4) >> 3), uint8((sumT(4) + 2) >> 2)},
		{uint8((sumL(4) + 2) >> 2), uint8((sumT(4) + sumL(4) + 4) >> 3)},
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			p[o+y*s+x] = m[y>>2][x>>2]
		}
	}
}

func WelsIChromaPredDcLeft_ref(p []uint8, o int, kiStride int32) {
	s := int(kiStride)
	for blk := 0; blk < 2; blk++ {
		v := 2
		for y := blk * 4; y < blk*4+4; y++ {
			v += int(p[o+y*s-1])
		}
		for y := blk * 4; y < blk*4+4; y++ {
			for x := 0; x < 8; x++ {
				p[o+y*s+x] = uint8(v >> 2)
			}
		}
	}
}

func WelsIChromaPredDcTop_ref(p []uint8, o int, kiStride int32) {
	s := int(kiStride)
	var m [2]uint8
	for blk := 0; blk < 2; blk++ {
		v := 2
		for x := blk * 4; x < blk*4+4; x++ {
			v += int(p[o-s+x])
		}
		m[blk] = uint8(v >> 2)
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			p[o+y*s+x] = m[x>>2]
		}
	}
}

func WelsI16x16LumaPredPlane_ref(p []uint8, o int, kiStride int32) {
	s := int(kiStride)
	var a, b, c, H, V int32
	pTop := o - s
	pLeft := o - 1
	for i := 0; i < 8; i++ {
		H += int32(i+1) * (int32(p[pTop+8+i]) - int32(p[pTop+6-i]))
		V += int32(i+1) * (int32(p[pLeft+(8+i)*s]) - int32(p[pLeft+(6-i)*s]))
	}
	a = (int32(p[pLeft+15*s]) + int32(p[pTop+15])) << 4
	b = (5*H + 32) >> 6
	c = (5*V + 32) >> 6
	for i := int32(0); i < 16; i++ {
		for j := int32(0); j < 16; j++ {
			iTmp := (a + b*(j-7) + c*(i-7) + 16) >> 5
			if iTmp < 0 {
				iTmp = 0
			} else if iTmp > 255 {
				iTmp = 255
			}
			p[o+int(i)*s+int(j)] = uint8(iTmp)
		}
	}
}

// ---- test drivers (GENERATE_4x4_UT / GENERATE_8x8_UT / GENERATE_16x16_UT) ----

func run4x4UT(t *testing.T, name string, pred, ref intraPredFunc) {
	const kiStride = 32
	pPredBuffer := make([]uint8, 12*kiStride)
	pRefBuffer := make([]uint8, 12*kiStride)
	for iRunTimes := 0; iRunTimes < 1000; iRunTimes++ {
		for i := 0; i < 12; i++ {
			v := uint8(rand.Intn(256))
			pRefBuffer[kiStride*3+i], pPredBuffer[kiStride*3+i] = v, v
			v = uint8(rand.Intn(256))
			pRefBuffer[i*kiStride+3], pPredBuffer[i*kiStride+3] = v, v
		}
		pred(pPredBuffer, kiStride*4+4, kiStride)
		ref(pRefBuffer, kiStride*4+4, kiStride)
		for i := 0; i < 4; i++ {
			for j := 0; j < 4; j++ {
				k := (i+4)*kiStride + j + 4
				if pPredBuffer[k] != pRefBuffer[k] {
					t.Fatalf("%s: mismatch at (%d,%d): %d != %d", name, j, i, pPredBuffer[k], pRefBuffer[k])
				}
			}
		}
	}
}

func runNxNUT(t *testing.T, name string, size int, pred, ref intraPredFunc) {
	const kiStride = 32
	pPredBuffer := make([]uint8, 18*kiStride)
	pRefBuffer := make([]uint8, 18*kiStride)
	for iRunTimes := 0; iRunTimes < 1000; iRunTimes++ {
		for i := 0; i < 17; i++ {
			v := uint8(rand.Intn(256))
			pRefBuffer[kiStride+i], pPredBuffer[kiStride+i] = v, v
			v = uint8(rand.Intn(256))
			pRefBuffer[(i+1)*kiStride-1], pPredBuffer[(i+1)*kiStride-1] = v, v
		}
		pred(pPredBuffer, 2*kiStride, kiStride)
		ref(pRefBuffer, 2*kiStride, kiStride)
		for i := 0; i < size; i++ {
			for j := 0; j < size; j++ {
				k := (i+2)*kiStride + j
				if pPredBuffer[k] != pRefBuffer[k] {
					t.Fatalf("%s: mismatch at (%d,%d): %d != %d", name, j, i, pPredBuffer[k], pRefBuffer[k])
				}
			}
		}
	}
}

func TestDecoderIntraPrediction4x4(t *testing.T) {
	cases := []struct {
		name      string
		pred, ref intraPredFunc
	}{
		{"WelsI4x4LumaPredV_c", WelsI4x4LumaPredV_c, refPredV(4)},
		{"WelsI4x4LumaPredH_c", WelsI4x4LumaPredH_c, refPredH(4)},
		{"WelsI4x4LumaPredDc_c", WelsI4x4LumaPredDc_c, refPredDC(4, 2)},
		{"WelsI4x4LumaPredDcLeft_c", WelsI4x4LumaPredDcLeft_c, refPredDCLeft(4, 2)},
		{"WelsI4x4LumaPredDcTop_c", WelsI4x4LumaPredDcTop_c, refPredDCTop(4, 2)},
		{"WelsI4x4LumaPredDcNA_c", WelsI4x4LumaPredDcNA_c, refPredDCNone(4)},
		{"WelsI4x4LumaPredDDL_c", WelsI4x4LumaPredDDL_c, ref4x4(predDDL4x4, false)},
		{"WelsI4x4LumaPredDDLTop_c", WelsI4x4LumaPredDDLTop_c, ref4x4(predDDL4x4, true)},
		{"WelsI4x4LumaPredDDR_c", WelsI4x4LumaPredDDR_c, ref4x4(predDDR4x4, false)},
		{"WelsI4x4LumaPredVR_c", WelsI4x4LumaPredVR_c, ref4x4(predVR4x4, false)},
		{"WelsI4x4LumaPredVL_c", WelsI4x4LumaPredVL_c, ref4x4(predVL4x4, false)},
		{"WelsI4x4LumaPredVLTop_c", WelsI4x4LumaPredVLTop_c, ref4x4(predVL4x4, true)},
		{"WelsI4x4LumaPredHU_c", WelsI4x4LumaPredHU_c, ref4x4(predHU4x4, false)},
		{"WelsI4x4LumaPredHD_c", WelsI4x4LumaPredHD_c, ref4x4(predHD4x4, false)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { run4x4UT(t, c.name, c.pred, c.ref) })
	}
}

func TestDecoderIntraPredictionChroma(t *testing.T) {
	cases := []struct {
		name      string
		pred, ref intraPredFunc
	}{
		{"WelsIChromaPredDcNA_c", WelsIChromaPredDcNA_c, refPredDCNone(8)},
		{"WelsIChromaPredPlane_c", WelsIChromaPredPlane_c, WelsIChromaPredPlane_ref},
		{"WelsIChromaPredDc_c", WelsIChromaPredDc_c, WelsIChromaPredDc_ref},
		{"WelsIChromaPredDcTop_c", WelsIChromaPredDcTop_c, WelsIChromaPredDcTop_ref},
		{"WelsIChromaPredDcLeft_c", WelsIChromaPredDcLeft_c, WelsIChromaPredDcLeft_ref},
		{"WelsIChromaPredH_c", WelsIChromaPredH_c, refPredH(8)},
		{"WelsIChromaPredV_c", WelsIChromaPredV_c, refPredV(8)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { runNxNUT(t, c.name, 8, c.pred, c.ref) })
	}
}

func TestDecoderIntraPrediction16x16(t *testing.T) {
	cases := []struct {
		name      string
		pred, ref intraPredFunc
	}{
		{"WelsI16x16LumaPredDcNA_c", WelsI16x16LumaPredDcNA_c, refPredDCNone(16)},
		{"WelsI16x16LumaPredPlane_c", WelsI16x16LumaPredPlane_c, WelsI16x16LumaPredPlane_ref},
		{"WelsI16x16LumaPredDcLeft_c", WelsI16x16LumaPredDcLeft_c, refPredDCLeft(16, 4)},
		{"WelsI16x16LumaPredDcTop_c", WelsI16x16LumaPredDcTop_c, refPredDCTop(16, 4)},
		{"WelsI16x16LumaPredDc_c", WelsI16x16LumaPredDc_c, refPredDC(16, 4)},
		{"WelsI16x16LumaPredH_c", WelsI16x16LumaPredH_c, refPredH(16)},
		{"WelsI16x16LumaPredV_c", WelsI16x16LumaPredV_c, refPredV(16)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { runNxNUT(t, c.name, 16, c.pred, c.ref) })
	}
}

// ---- 8x8 luma (no C unit test exists; checked against the H.264 equations
// 8.3.2.2.1 - 8.3.2.2.10) ----

type intraPred8x8Func func(pPred []uint8, iPredOff int, kiStride int32, bTLAvail bool, bTRAvail bool)

// ref8x8Filtered returns the filtered reference samples p'[x,-1] (top[x+1],
// x = -1..15) and p'[-1,y] (left[y+1], y = -1..7).
func ref8x8Filtered(p []uint8, o, s int, bTLAvail, bTRAvail bool) (top [17]int, left [9]int) {
	raw := func(x, y int) int {
		if y == -1 && x > 7 && !bTRAvail {
			x = 7
		}
		return int(p[o+y*s+x])
	}
	// top
	if bTLAvail {
		top[1] = (raw(-1, -1) + 2*raw(0, -1) + raw(1, -1) + 2) >> 2
	} else {
		top[1] = (3*raw(0, -1) + raw(1, -1) + 2) >> 2
	}
	for x := 1; x < 15; x++ {
		top[x+1] = (raw(x-1, -1) + 2*raw(x, -1) + raw(x+1, -1) + 2) >> 2
	}
	top[16] = (raw(14, -1) + 3*raw(15, -1) + 2) >> 2
	// left
	if bTLAvail {
		left[1] = (raw(-1, -1) + 2*raw(-1, 0) + raw(-1, 1) + 2) >> 2
	} else {
		left[1] = (3*raw(-1, 0) + raw(-1, 1) + 2) >> 2
	}
	for y := 1; y < 7; y++ {
		left[y+1] = (raw(-1, y-1) + 2*raw(-1, y) + raw(-1, y+1) + 2) >> 2
	}
	left[8] = (raw(-1, 6) + 3*raw(-1, 7) + 2) >> 2
	// top-left (only used by modes that require all neighbours)
	tl := (raw(0, -1) + 2*raw(-1, -1) + raw(-1, 0) + 2) >> 2
	top[0], left[0] = tl, tl
	return
}

func ref8x8(mode string) intraPred8x8Func {
	return func(p []uint8, o int, kiStride int32, bTLAvail, bTRAvail bool) {
		s := int(kiStride)
		topF, leftF := ref8x8Filtered(p, o, s, bTLAvail, bTRAvail)
		T := func(x int) int { return topF[x+1] }
		L := func(y int) int { return leftF[y+1] }
		var out [8][8]int
		sumT, sumL := 0, 0
		for i := 0; i < 8; i++ {
			sumT += T(i)
			sumL += L(i)
		}
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				var v int
				switch mode {
				case "V":
					v = T(x)
				case "H":
					v = L(y)
				case "DC":
					v = (sumT + sumL + 8) >> 4
				case "DCLeft":
					v = (sumL + 4) >> 3
				case "DCTop":
					v = (sumT + 4) >> 3
				case "DCNA":
					v = 128
				case "DDL":
					if x == 7 && y == 7 {
						v = (T(14) + 3*T(15) + 2) >> 2
					} else {
						v = (T(x+y) + 2*T(x+y+1) + T(x+y+2) + 2) >> 2
					}
				case "DDR":
					switch {
					case x > y:
						v = (T(x-y-2) + 2*T(x-y-1) + T(x-y) + 2) >> 2
					case x < y:
						v = (L(y-x-2) + 2*L(y-x-1) + L(y-x) + 2) >> 2
					default:
						v = (T(0) + 2*T(-1) + L(0) + 2) >> 2
					}
				case "VR":
					zVR := 2*x - y
					switch {
					case zVR >= 0 && zVR&1 == 0:
						v = (T(x-(y>>1)-1) + T(x-(y>>1)) + 1) >> 1
					case zVR >= 0:
						v = (T(x-(y>>1)-2) + 2*T(x-(y>>1)-1) + T(x-(y>>1)) + 2) >> 2
					case zVR == -1:
						v = (L(0) + 2*L(-1) + T(0) + 2) >> 2
					default:
						v = (L(y-2*x-1) + 2*L(y-2*x-2) + L(y-2*x-3) + 2) >> 2
					}
				case "HD":
					zHD := 2*y - x
					switch {
					case zHD >= 0 && zHD&1 == 0:
						v = (L(y-(x>>1)-1) + L(y-(x>>1)) + 1) >> 1
					case zHD >= 0:
						v = (L(y-(x>>1)-2) + 2*L(y-(x>>1)-1) + L(y-(x>>1)) + 2) >> 2
					case zHD == -1:
						v = (L(0) + 2*L(-1) + T(0) + 2) >> 2
					default:
						v = (T(x-2*y-1) + 2*T(x-2*y-2) + T(x-2*y-3) + 2) >> 2
					}
				case "VL":
					if y&1 == 0 {
						v = (T(x+(y>>1)) + T(x+(y>>1)+1) + 1) >> 1
					} else {
						v = (T(x+(y>>1)) + 2*T(x+(y>>1)+1) + T(x+(y>>1)+2) + 2) >> 2
					}
				case "HU":
					zHU := x + 2*y
					switch {
					case zHU < 13 && zHU&1 == 0:
						v = (L(y+(x>>1)) + L(y+(x>>1)+1) + 1) >> 1
					case zHU < 13:
						v = (L(y+(x>>1)) + 2*L(y+(x>>1)+1) + L(y+(x>>1)+2) + 2) >> 2
					case zHU == 13:
						v = (L(6) + 3*L(7) + 2) >> 2
					default:
						v = L(7)
					}
				}
				out[y][x] = v
			}
		}
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				p[o+y*s+x] = uint8(out[y][x])
			}
		}
	}
}

func TestDecoderIntraPrediction8x8Luma(t *testing.T) {
	type avail struct{ tl, tr bool }
	all := []avail{{false, false}, {false, true}, {true, false}, {true, true}}
	tlAll := []avail{{true, false}, {true, true}}   // modes needing the top-left sample
	trOn := []avail{{false, true}, {true, true}}    // modes needing top-right
	trOff := []avail{{false, false}, {true, false}} // "Top" variants: top-right unavailable
	cases := []struct {
		name  string
		pred  intraPred8x8Func
		mode  string
		avail []avail
	}{
		{"WelsI8x8LumaPredV_c", WelsI8x8LumaPredV_c, "V", all},
		{"WelsI8x8LumaPredH_c", WelsI8x8LumaPredH_c, "H", all},
		{"WelsI8x8LumaPredDc_c", WelsI8x8LumaPredDc_c, "DC", all},
		{"WelsI8x8LumaPredDcLeft_c", WelsI8x8LumaPredDcLeft_c, "DCLeft", all},
		{"WelsI8x8LumaPredDcTop_c", WelsI8x8LumaPredDcTop_c, "DCTop", all},
		{"WelsI8x8LumaPredDcNA_c", WelsI8x8LumaPredDcNA_c, "DCNA", all},
		{"WelsI8x8LumaPredDDL_c", WelsI8x8LumaPredDDL_c, "DDL", trOn},
		{"WelsI8x8LumaPredDDLTop_c", WelsI8x8LumaPredDDLTop_c, "DDL", trOff},
		{"WelsI8x8LumaPredDDR_c", WelsI8x8LumaPredDDR_c, "DDR", tlAll},
		{"WelsI8x8LumaPredVR_c", WelsI8x8LumaPredVR_c, "VR", tlAll},
		{"WelsI8x8LumaPredHU_c", WelsI8x8LumaPredHU_c, "HU", all},
		{"WelsI8x8LumaPredHD_c", WelsI8x8LumaPredHD_c, "HD", tlAll},
		{"WelsI8x8LumaPredVL_c", WelsI8x8LumaPredVL_c, "VL", trOn},
		{"WelsI8x8LumaPredVLTop_c", WelsI8x8LumaPredVLTop_c, "VL", trOff},
	}
	const kiStride = 32
	for _, c := range cases {
		ref := ref8x8(c.mode)
		for _, av := range c.avail {
			pPredBuffer := make([]uint8, 18*kiStride)
			pRefBuffer := make([]uint8, 18*kiStride)
			for iRunTimes := 0; iRunTimes < 500; iRunTimes++ {
				for i := 0; i < 17; i++ {
					v := uint8(rand.Intn(256))
					pRefBuffer[kiStride+i], pPredBuffer[kiStride+i] = v, v
					v = uint8(rand.Intn(256))
					pRefBuffer[(i+1)*kiStride-1], pPredBuffer[(i+1)*kiStride-1] = v, v
				}
				c.pred(pPredBuffer, 2*kiStride, kiStride, av.tl, av.tr)
				ref(pRefBuffer, 2*kiStride, kiStride, av.tl, av.tr)
				for i := 0; i < 8; i++ {
					for j := 0; j < 8; j++ {
						k := (i+2)*kiStride + j
						if pPredBuffer[k] != pRefBuffer[k] {
							t.Fatalf("%s (TL %v TR %v): mismatch at (%d,%d): %d != %d", c.name, av.tl, av.tr,
								j, i, pPredBuffer[k], pRefBuffer[k])
						}
					}
				}
			}
		}
	}
}
