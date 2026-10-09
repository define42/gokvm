package common

import "testing"

func TestWelsClip1(t *testing.T) {
	for _, c := range []struct {
		in   int32
		want uint8
	}{{-1, 0}, {-1 << 31, 255}, {0, 0}, {128, 128}, {255, 255}, {256, 255}, {1 << 30, 255}, {-300, 0}} {
		if got := WelsClip1(c.in); got != c.want {
			t.Errorf("WelsClip1(%d) = %d want %d", c.in, got, c.want)
		}
	}
}

func TestMacros(t *testing.T) {
	if WELS_ALIGN(int32(17), 16) != 32 || WELS_ALIGN(uint32(32), 16) != 32 {
		t.Error("WELS_ALIGN")
	}
	if WELS_CLIP3(int32(-5), -2, 2) != -2 || WELS_CLIP3(int16(7), 0, 5) != 5 || WELS_CLIP3(3.5, 0, 5) != 3.5 {
		t.Error("WELS_CLIP3")
	}
	if WELS_ABS(int32(-7)) != 7 || WELS_ABS(int8(3)) != 3 {
		t.Error("WELS_ABS")
	}
	if WELS_DIV_ROUND(int32(7), 2) != 4 || WELS_DIV_ROUND(int32(7), 0) != 7 || WELS_DIV_ROUND(int64(-7), 2) != -3 {
		t.Error("WELS_DIV_ROUND")
	}
	if WELS_ROUND(2.5) != 3 || WELS_ROUND(float32(-0.7)) != 0 || WELS_ROUND(-1.7) != -1 {
		t.Error("WELS_ROUND")
	}
	if WelsMedian(3, 9, 5) != 5 || WelsMedian(-1, -1, 4) != -1 {
		t.Error("WelsMedian")
	}
	if CeilLog2(1) != 0 || CeilLog2(5) != 3 || CeilLog2(8) != 3 || WELS_LOG2(1) != 0 || WELS_LOG2(9) != 3 {
		t.Error("log2")
	}
	if NEG_NUM(int32(5)) != -5 || WELS_SIGN(int32(-3)) != -1 || WELS_SIGN(int32(3)) != 0 {
		t.Error("NEG_NUM / WELS_SIGN")
	}
	if WELS_MIN_POSITIVE(int8(-1), 3) != 3 || WELS_MIN_POSITIVE(int8(2), 3) != 2 {
		t.Error("WELS_MIN_POSITIVE")
	}
	// WELS_NON_ZERO_COUNT_AVERAGE
	for _, c := range []struct{ a, b, want int8 }{{-1, -1, 0}, {-1, 4, 4}, {3, -1, 3}, {3, 4, 4}, {0, 0, 0}, {16, 16, 16}} {
		if got := WELS_NON_ZERO_COUNT_AVERAGE(c.a, c.b); got != c.want {
			t.Errorf("WELS_NON_ZERO_COUNT_AVERAGE(%d,%d)=%d want %d", c.a, c.b, got, c.want)
		}
	}
	if !IS_INTRA(uint32(MB_TYPE_INTRA16x16)) || IS_INTRA(uint32(MB_TYPE_16x16)) || !IS_DIR(uint32(MB_TYPE_P1L0), 1, 0) {
		t.Error("IS_*")
	}
	if !IS_VCL_NAL(NAL_UNIT_CODED_SLICE_EXT, 1) || IS_VCL_NAL(NAL_UNIT_CODED_SLICE_EXT, 0) {
		t.Error("IS_VCL_NAL")
	}
	buf := make([]uint8, 8)
	ST32(buf, 1, 0x11223344)
	if LD32(buf, 1) != 0x11223344 || LD16(buf, 1) != 0x3344 {
		t.Error("LD/ST")
	}
	m := make([]uint16, 6)
	WelsSetMemMultiplebytes_c(m, 0x12345, 4, 2)
	if m[3] != 0x2345 || m[4] != 0 {
		t.Error("WelsSetMemMultiplebytes_c")
	}
	WelsSetMemMultiplebytes_c(m, 0, 6, 2)
	if m[0] != 0 {
		t.Error("WelsSetMemMultiplebytes_c zero")
	}
}
