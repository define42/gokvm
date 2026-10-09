package encoder

import "testing"

func TestWelsMoveMemoryAndPadding(t *testing.T) {
	const w, h, stride = 6, 4, 8
	src := make([]uint8, stride*h+2*(stride/2)*(h/2))
	for i := range src {
		src[i] = uint8(i + 1)
	}
	dst := make([]uint8, 2*stride*8+2*stride*4)
	dY, dU, dV := 3, 3+2*stride*8, 3+2*stride*8+stride*4
	sU, sV := stride*h, stride*h+(stride/2)*(h/2)
	WelsMoveMemory_c(dst, dY, dst, dU, dst, dV, 2*stride, stride, stride,
		src, 0, src, sU, src, sV, stride, stride/2, stride/2, w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if dst[dY+y*2*stride+x] != src[y*stride+x] {
				t.Fatalf("Y mismatch at %d,%d", x, y)
			}
		}
	}
	for y := 0; y < h/2; y++ {
		for x := 0; x < w/2; x++ {
			if dst[dU+y*stride+x] != src[sU+y*stride/2+x] || dst[dV+y*stride+x] != src[sV+y*stride/2+x] {
				t.Fatalf("UV mismatch at %d,%d", x, y)
			}
		}
	}
	var p CWelsPreProcess
	p.Padding(dst, dY, dst, dU, dst, dV, 2*stride, stride, w, w+4, h, h+2)
	for y := 0; y < h+2; y++ {
		for x := 0; x < w+4; x++ {
			v := dst[dY+y*2*stride+x]
			if (y >= h || x >= w) && v != 0 {
				t.Fatalf("Y padding at %d,%d = %d", x, y, v)
			}
		}
	}
	for y := 0; y < (h+2)/2; y++ {
		for x := 0; x < (w+4)/2; x++ {
			if (y >= h/2 || x >= w/2) && (dst[dU+y*stride+x] != 0x80 || dst[dV+y*stride+x] != 0x80) {
				t.Fatalf("UV padding at %d,%d", x, y)
			}
		}
	}
}

func TestClearEndOfLinePadding(t *testing.T) {
	buf := make([]uint8, 40)
	for i := range buf {
		buf[i] = 0xff
	}
	ClearEndOfLinePadding(buf, 4, 8, 5, 3)
	for y := 0; y < 3; y++ {
		for x := 0; x < 8; x++ {
			v := buf[4+y*8+x]
			if x < 5 && v != 0xff || x >= 5 && v != 0 {
				t.Fatalf("at %d,%d = %d", x, y, v)
			}
		}
	}
}

func TestJudgeNeedOfScaling(t *testing.T) {
	p := NewSWelsSvcCodingParam()
	p.ISpatialLayerNum = 2
	p.SUsedPicRect.iWidth, p.SUsedPicRect.iHeight = 640, 480
	p.sDependencyLayers[0].iActualWidth, p.sDependencyLayers[0].iActualHeight = 160, 90
	p.sDependencyLayers[1].iActualWidth, p.sDependencyLayers[1].iActualHeight = 320, 180
	var s Scaled_Picture
	if !JudgeNeedOfScaling(p, &s) {
		t.Fatal("expected downsampling")
	}
	if s.iScaledWidth[1] != 240 || s.iScaledHeight[1] != 180 || s.iScaledWidth[0] != 120 || s.iScaledHeight[0] != 90 {
		t.Fatalf("scaled sizes %v %v", s.iScaledWidth, s.iScaledHeight)
	}
}
