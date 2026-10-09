package processing

// Port of test/processing/ProcessUT_DownSample.cpp (C reference tests only).

import (
	"math/rand"
	"testing"
)

func DyadicBilinearDownsampler_ref(pDst []uint8, kiDstStride int32, pSrc []uint8, kiSrcStride int32,
	kiSrcWidth, kiSrcHeight int32) {
	kiDstWidth := kiSrcWidth >> 1
	kiDstHeight := kiSrcHeight >> 1
	for j := int32(0); j < kiDstHeight; j++ {
		src := pSrc[2*j*kiSrcStride:]
		for i := int32(0); i < kiDstWidth; i++ {
			x := i << 1
			r1 := (int32(src[x]) + int32(src[x+1]) + 1) >> 1
			r2 := (int32(src[x+kiSrcStride]) + int32(src[x+kiSrcStride+1]) + 1) >> 1
			pDst[j*kiDstStride+i] = uint8((r1 + r2 + 1) >> 1)
		}
	}
}

// refScale mirrors WELS_ROUND ((float)a / (float)b * scale).
func refScale(a, b int32, scale float32) int32 {
	f := float32(a) / float32(b)
	f = float32(f * scale)
	return int32(0.5 + float64(f))
}

func GeneralBilinearFastDownsampler_ref(pDst []uint8, kiDstStride, kiDstWidth, kiDstHeight int32,
	pSrc []uint8, kiSrcStride, kiSrcWidth, kiSrcHeight int32) {
	const bw, bh = 16, 15
	const sw, sh = uint32(1) << bw, uint32(1) << bh
	sx := refScale(kiSrcWidth, kiDstWidth, float32(sw))
	sy := refScale(kiSrcHeight, kiDstHeight, float32(sh))
	yInv := int32(1 << (bh - 1))
	for i := int32(0); i < kiDstHeight-1; i++ {
		yy := yInv >> bh
		fv := uint32(yInv) & (sh - 1)
		srcRow := yy * kiSrcStride
		dst := pDst[i*kiDstStride:]
		xInv := int32(1 << (bw - 1))
		for j := int32(0); j < kiDstWidth-1; j++ {
			xx := xInv >> bw
			fu := uint32(xInv) & (sw - 1)
			p := srcRow + xx
			a, b := uint32(pSrc[p]), uint32(pSrc[p+1])
			c, d := uint32(pSrc[p+kiSrcStride]), uint32(pSrc[p+kiSrcStride+1])
			x := ((sw - 1 - fu) * (sh - 1 - fv) >> bw) * a
			x += (fu * (sh - 1 - fv) >> bw) * b
			x += ((sw - 1 - fu) * fv >> bw) * c
			x += (fu * fv >> bw) * d
			x >>= bh - 1
			x += 1
			x >>= 1
			if x > 255 {
				x = 255
			}
			dst[j] = uint8(x)
			xInv += sx
		}
		dst[kiDstWidth-1] = pSrc[srcRow+(xInv>>bw)]
		yInv += sy
	}
	yy := yInv >> bh
	dst := pDst[(kiDstHeight-1)*kiDstStride:]
	xInv := int32(1 << (bw - 1))
	for j := int32(0); j < kiDstWidth; j++ {
		dst[j] = pSrc[yy*kiSrcStride+(xInv>>bw)]
		xInv += sx
	}
}

func GeneralBilinearAccurateDownsampler_ref(pDst []uint8, kiDstStride, kiDstWidth, kiDstHeight int32,
	pSrc []uint8, kiSrcStride, kiSrcWidth, kiSrcHeight int32) {
	const sb = 15
	const s = int64(1) << sb
	sx := refScale(kiSrcWidth, kiDstWidth, float32(s))
	sy := refScale(kiSrcHeight, kiDstHeight, float32(s))
	yInv := int32(1 << (sb - 1))
	for i := int32(0); i < kiDstHeight-1; i++ {
		yy := yInv >> sb
		fv := int64(yInv) & (s - 1)
		srcRow := yy * kiSrcStride
		dst := pDst[i*kiDstStride:]
		xInv := int32(1 << (sb - 1))
		for j := int32(0); j < kiDstWidth-1; j++ {
			xx := xInv >> sb
			fu := int64(xInv) & (s - 1)
			p := srcRow + xx
			a, b := int64(pSrc[p]), int64(pSrc[p+1])
			c, d := int64(pSrc[p+kiSrcStride]), int64(pSrc[p+kiSrcStride+1])
			x := ((s-1-fu)*(s-1-fv)*a + fu*(s-1-fv)*b + (s-1-fu)*fv*c + fu*fv*d + (1 << (2*sb - 1))) >> (2 * sb)
			x = max(0, min(x, 255))
			dst[j] = uint8(x)
			xInv += sx
		}
		dst[kiDstWidth-1] = pSrc[srcRow+(xInv>>sb)]
		yInv += sy
	}
	yy := yInv >> sb
	dst := pDst[(kiDstHeight-1)*kiDstStride:]
	xInv := int32(1 << (sb - 1))
	for j := int32(0); j < kiDstWidth; j++ {
		dst[j] = pSrc[yy*kiSrcStride+(xInv>>sb)]
		xInv += sx
	}
}

func fillRand2(r *rand.Rand, a, b []uint8) {
	for j := range a {
		v := uint8(r.Intn(256))
		a[j], b[j] = v, v
	}
}

func TestDownSampleTest_DyadicBilinearDownsampler_c(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	dst_c, src_c := make([]uint8, 50000), make([]uint8, 50000)
	dst_a, src_a := make([]uint8, 50000), make([]uint8, 50000)
	const stride, width, height = 560, 512, 80
	fillRand2(r, dst_c, dst_a)
	fillRand2(r, src_c, src_a)
	DyadicBilinearDownsampler_ref(dst_c, stride, src_c, stride, width, height)
	DyadicBilinearDownsampler_c(dst_a, 0, stride, src_a, 0, stride, width, height)
	for j := 0; j < height>>1; j++ {
		for m := 0; m < width>>1; m++ {
			if dst_c[m+j*stride] != dst_a[m+j*stride] {
				t.Fatalf("mismatch at %d,%d", m, j)
			}
		}
	}
}

func testGeneralBilinear(t *testing.T, fn GeneralDownsampleFunc,
	ref func([]uint8, int32, int32, int32, []uint8, int32, int32, int32)) {
	r := rand.New(rand.NewSource(4))
	dst_c, src_c := make([]uint8, 70000), make([]uint8, 70000)
	dst_a, src_a := make([]uint8, 70000), make([]uint8, 70000)
	for i := 0; i < 5; i++ {
		const stride, srcW, srcH = 320, 320, 180
		dstW := int32(srcW>>(i+1)) + int32(r.Intn(srcW>>(i+1)))
		dstH := int32(srcH>>(i+1)) + int32(r.Intn(srcH>>(i+1)))
		fillRand2(r, dst_c, dst_a)
		fillRand2(r, src_c, src_a)
		ref(dst_c, stride, dstW, dstH, src_c, stride, srcW, srcH)
		fn(dst_a, 0, stride, dstW, dstH, src_a, 0, stride, srcW, srcH)
		for j := int32(0); j < dstH; j++ {
			for m := int32(0); m < dstW; m++ {
				if dst_c[m+j*stride] != dst_a[m+j*stride] {
					t.Fatalf("iter %d (%dx%d): mismatch at %d,%d", i, dstW, dstH, m, j)
				}
			}
		}
	}
}

func TestDownSampleTest_GeneralBilinearFastDownsampler_c(t *testing.T) {
	testGeneralBilinear(t, GeneralBilinearFastDownsampler_c, GeneralBilinearFastDownsampler_ref)
}

func TestDownSampleTest_GeneralBilinearAccurateDownsampler_c(t *testing.T) {
	testGeneralBilinear(t, GeneralBilinearAccurateDownsampler_c, GeneralBilinearAccurateDownsampler_ref)
}

func TestDownSampleTest_OneThirdAndQuarter_c(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	src := make([]uint8, 50000)
	dst := make([]uint8, 50000)
	fillWithRandomData(r, src)
	const stride = 560
	// one third: 480x30 -> 160x10
	DyadicBilinearOneThirdDownsampler_c(dst, 0, stride, src, 0, stride, 480, 30/3)
	for j := 0; j < 10; j++ {
		for i := 0; i < 160; i++ {
			s := src[j*3*stride+i*3:]
			r1 := (int(s[0]) + int(s[1]) + 1) >> 1
			r2 := (int(s[stride]) + int(s[stride+1]) + 1) >> 1
			if int(dst[j*stride+i]) != (r1+r2+1)>>1 {
				t.Fatalf("one third mismatch at %d,%d", i, j)
			}
		}
	}
	// quarter: 640x80 -> 160x20
	DyadicBilinearQuarterDownsampler_c(dst, 0, stride, src, 0, stride, 640, 80)
	for j := 0; j < 20; j++ {
		for i := 0; i < 160; i++ {
			s := src[j*4*stride+i*4:]
			r1 := (int(s[0]) + int(s[1]) + 1) >> 1
			r2 := (int(s[stride]) + int(s[stride+1]) + 1) >> 1
			if int(dst[j*stride+i]) != (r1+r2+1)>>1 {
				t.Fatalf("quarter mismatch at %d,%d", i, j)
			}
		}
	}
}

func TestDownSampleTest_VpFrameworkRejectsInvalidChromaStride(t *testing.T) {
	var pVp IWelsVP
	if WelsCreateVpInterface(&pVp, WELSVP_INTERFACE_VERION) != RET_SUCCESS {
		t.Fatal("create failed")
	}
	if pVp == nil {
		t.Fatal("nil interface")
	}
	if pVp.Init(METHOD_DOWNSAMPLE, nil) != RET_SUCCESS {
		t.Fatal("init failed")
	}

	srcY := make([]uint8, 16*16)
	srcU := make([]uint8, 8*8)
	srcV := make([]uint8, 8*8)
	dstY := make([]uint8, 8*8)
	dstU := make([]uint8, 4*4)
	dstV := make([]uint8, 4*4)

	var src, dst SPixMap
	src.EFormat = VIDEO_FORMAT_I420
	src.PPixel[0] = srcY
	src.PPixel[1] = srcU
	src.PPixel[2] = srcV
	src.IStride[0] = 16
	src.IStride[1] = 0
	src.IStride[2] = 8
	src.SRect.IRectWidth = 16
	src.SRect.IRectHeight = 16

	dst.EFormat = VIDEO_FORMAT_I420
	dst.PPixel[0] = dstY
	dst.PPixel[1] = dstU
	dst.PPixel[2] = dstV
	dst.IStride[0] = 8
	dst.IStride[1] = 4
	dst.IStride[2] = 4
	dst.SRect.IRectWidth = 8
	dst.SRect.IRectHeight = 8

	expect := func(line int) {
		t.Helper()
		if ret := pVp.Process(METHOD_DOWNSAMPLE, &src, &dst); ret != RET_INVALIDPARAM {
			t.Errorf("case %d: got %d, want RET_INVALIDPARAM", line, ret)
		}
	}
	expect(1)

	src.IStride[1] = 8
	src.IStride[2] = 0
	expect(2)

	src.IStride[1] = 8
	src.IStride[2] = 8
	src.PPixel[1] = nil
	expect(3)

	src.PPixel[1] = srcU
	src.PPixel[2] = nil
	expect(4)

	src.PPixel[2] = srcV
	dst.PPixel[1] = nil
	expect(5)

	dst.PPixel[1] = dstU
	dst.PPixel[2] = nil
	expect(6)

	if WelsDestroyVpInterface(pVp, WELSVP_INTERFACE_VERION) != RET_SUCCESS {
		t.Fatal("destroy failed")
	}
}
