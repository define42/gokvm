package processing

// Framework-level tests: every method is driven through IWelsVP on pictures
// laid out like the encoder's (one allocation, 32 pixel padding, plane origin
// given by IPixelOff).

import (
	"math/rand"
	"testing"
)

const testPadding = 32

type testPic struct {
	buf    []uint8
	off    [3]int
	stride [3]int32
	w, h   int32
}

// newTestPic mirrors codec/encoder/core/src/picture_handle.cpp AllocPicture.
func newTestPic(w, h int32) *testPic {
	p := &testPic{w: w, h: h}
	iPicWidth := WELS_ALIGN(w, 16) + (testPadding << 1)
	iPicHeight := WELS_ALIGN(h, 16) + (testPadding << 1)
	iPicChromaWidth := iPicWidth >> 1
	iPicChromaHeight := iPicHeight >> 1
	iPicWidth = WELS_ALIGN(iPicWidth, 32)
	iPicChromaWidth = WELS_ALIGN(iPicChromaWidth, 16)
	iLumaSize := int(iPicWidth * iPicHeight)
	iChromaSize := int(iPicChromaWidth * iPicChromaHeight)
	p.buf = make([]uint8, iLumaSize+2*iChromaSize)
	p.stride = [3]int32{iPicWidth, iPicChromaWidth, iPicChromaWidth}
	p.off[0] = int((1 + iPicWidth) * testPadding)
	p.off[1] = iLumaSize + int(((1+iPicChromaWidth)*testPadding)>>1)
	p.off[2] = iLumaSize + iChromaSize + int(((1+iPicChromaWidth)*testPadding)>>1)
	return p
}

func (p *testPic) pixMap() SPixMap {
	var m SPixMap
	for i := 0; i < 3; i++ {
		m.PPixel[i] = p.buf
		m.IPixelOff[i] = p.off[i]
		m.IStride[i] = p.stride[i]
	}
	m.ISizeInBits = 8
	m.SRect.IRectWidth = p.w
	m.SRect.IRectHeight = p.h
	m.EFormat = VIDEO_FORMAT_I420
	return m
}

func (p *testPic) fillRandom(r *rand.Rand) {
	for i := range p.buf {
		p.buf[i] = uint8(r.Intn(256))
	}
}

// fillScrolled makes p a copy of src shifted vertically by mv lines with some noise blocks.
func (p *testPic) fillShifted(src *testPic, mv int32, r *rand.Rand) {
	copy(p.buf, src.buf)
	for y := int32(0); y < p.h; y++ {
		sy := y + mv
		if sy < 0 || sy >= p.h {
			continue
		}
		copy(p.buf[p.off[0]+int(y*p.stride[0]):][:p.w], src.buf[src.off[0]+int(sy*src.stride[0]):][:p.w])
	}
	for k := 0; k < 20; k++ {
		x, y := r.Int31n(p.w), r.Int31n(p.h)
		p.buf[p.off[0]+int(y*p.stride[0]+x)] ^= 0x55
	}
}

func newVaaResult(iMbNum int) *SVAACalcResult {
	return &SVAACalcResult{
		PSad8x8:           make([][4]int32, iMbNum),
		PSsd16x16:         make([]int32, iMbNum),
		PSum16x16:         make([]int32, iMbNum),
		PSumOfSquare16x16: make([]int32, iMbNum),
		PSumOfDiff8x8:     make([][4]int32, iMbNum),
		PMad8x8:           make([][4]uint8, iMbNum),
	}
}

func TestVpFrameworkAllMethods(t *testing.T) {
	r := rand.New(rand.NewSource(20))
	var vp IWelsVP
	if ret := WelsCreateVpInterface(&vp, WELSVP_INTERFACE_VERION); ret != RET_SUCCESS || vp == nil {
		t.Fatalf("create: %d", ret)
	}
	defer WelsDestroyVpInterface(vp, WELSVP_INTERFACE_VERION)

	const w, h = 320, 192
	iMbW, iMbH := w/16, h/16
	iMbNum := iMbW * iMbH
	cur := newTestPic(w, h)
	ref := newTestPic(w, h)
	ref.fillRandom(r)
	cur.fillShifted(ref, 7, r)
	sCur, sRef := cur.pixMap(), ref.pixMap()

	// VAA statistics with everything enabled.
	vaa := newVaaResult(iMbNum)
	calc := SVAACalcParam{ICalcVar: 1, ICalcBgd: 1, ICalcSsd: 1, PCalcResult: vaa}
	if ret := vp.Set(METHOD_VAA_STATISTICS, &calc); ret != RET_SUCCESS {
		t.Fatalf("vaa set: %d", ret)
	}
	if ret := vp.Process(METHOD_VAA_STATISTICS, &sCur, &sRef); ret != RET_SUCCESS {
		t.Fatalf("vaa process: %d", ret)
	}
	if &vaa.PCurY[0] != &cur.buf[cur.off[0]] || &vaa.PRefY[0] != &ref.buf[ref.off[0]] {
		t.Fatal("PCurY/PRefY not set to plane origins")
	}

	// Background detection.
	bgdFlags := make([]int8, iMbNum)
	bgd := SBGDInterface{PBackgroundMbFlag: bgdFlags, PCalcRes: vaa}
	if ret := vp.Set(METHOD_BACKGROUND_DETECTION, &bgd); ret != RET_SUCCESS {
		t.Fatalf("bgd set: %d", ret)
	}
	if ret := vp.Process(METHOD_BACKGROUND_DETECTION, &sCur, &sRef); ret != RET_SUCCESS {
		t.Fatalf("bgd process: %d", ret)
	}

	// Adaptive quantization: the VAA-result path and the variance path must agree.
	aqRun := func(useVaa bool) ([]SMotionTextureUnit, []int8, int32) {
		mt := make([]SMotionTextureUnit, iMbNum)
		dq := make([]int8, iMbNum)
		aq := SAdaptiveQuantizationParam{IAdaptiveQuantMode: AQ_BITRATE_MODE, PCalcResult: vaa,
			PMotionTextureUnit: mt, PMotionTextureIndexToDeltaQp: dq}
		if !useVaa {
			aq.PCalcResult = &SVAACalcResult{}
		}
		vp.Set(METHOD_ADAPTIVE_QUANT, &aq)
		if ret := vp.Process(METHOD_ADAPTIVE_QUANT, &sCur, &sRef); ret != RET_SUCCESS {
			t.Fatalf("aq process: %d", ret)
		}
		vp.Get(METHOD_ADAPTIVE_QUANT, &aq)
		return mt, dq, aq.IAverMotionTextureIndexToDeltaQp
	}
	mt1, dq1, av1 := aqRun(true)
	mt2, dq2, av2 := aqRun(false)
	for i := range mt1 {
		if mt1[i] != mt2[i] || dq1[i] != dq2[i] {
			t.Fatalf("aq paths differ at mb %d: %+v/%d vs %+v/%d", i, mt1[i], dq1[i], mt2[i], dq2[i])
		}
	}
	if av1 != av2 {
		t.Fatalf("aq average differs: %d vs %d", av1, av2)
	}

	// Complexity analysis (all three modes).
	gom := make([]int32, iMbH)
	fg := make([]int32, iMbH)
	refMbType := make([]uint32, iMbNum)
	for _, mode := range []int32{FRAME_SAD, GOM_SAD, GOM_VAR} {
		ca := SComplexityAnalysisParam{IComplexityAnalysisMode: mode, ICalcBgd: 1, IMbNumInGom: int32(iMbW),
			PGomComplexity: gom, PGomForegroundBlockNum: fg, PBackgroundMbFlag: bgdFlags, UiRefMbType: refMbType,
			PCalcResult: vaa}
		vp.Set(METHOD_COMPLEXITY_ANALYSIS, &ca)
		if ret := vp.Process(METHOD_COMPLEXITY_ANALYSIS, &sCur, &sRef); ret != RET_SUCCESS {
			t.Fatalf("ca mode %d: %d", mode, ret)
		}
		vp.Get(METHOD_COMPLEXITY_ANALYSIS, &ca)
		if ca.IFrameComplexity <= 0 {
			t.Errorf("ca mode %d: complexity %d", mode, ca.IFrameComplexity)
		}
	}

	// Scroll detection, then screen scene change and screen complexity with scroll.
	var scroll SScrollDetectionParam
	vp.Set(METHOD_SCROLL_DETECTION, &scroll)
	if ret := vp.Process(METHOD_SCROLL_DETECTION, &sCur, &sRef); ret != RET_SUCCESS {
		t.Fatalf("scroll: %d", ret)
	}
	vp.Get(METHOD_SCROLL_DETECTION, &scroll)
	if !scroll.BScrollDetectFlag || scroll.IScrollMvY != 7 {
		t.Fatalf("scroll result %+v", scroll)
	}

	staticIdc := make([]uint8, (w/8)*(h/8))
	scd := SSceneChangeResult{PStaticBlockIdc: staticIdc, SScrollResult: scroll}
	vp.Set(METHOD_SCENE_CHANGE_DETECTION_SCREEN, &scd)
	if ret := vp.Process(METHOD_SCENE_CHANGE_DETECTION_SCREEN, &sCur, &sRef); ret != RET_SUCCESS {
		t.Fatalf("scd screen: %d", ret)
	}
	vp.Get(METHOD_SCENE_CHANGE_DETECTION_SCREEN, &scd)
	nScrolled := 0
	for _, v := range staticIdc {
		if v == uint8(SCROLLED_STATIC) {
			nScrolled++
		}
	}
	if nScrolled == 0 || scd.ESceneChangeIdc != SIMILAR_SCENE {
		t.Errorf("screen scd: scrolled blocks %d, idc %d", nScrolled, scd.ESceneChangeIdc)
	}

	// Scroll MV pointing upwards: CComplexityAnalysisScreen reads above the origin.
	cas := SComplexityAnalysisScreenParam{IMbRowInGom: 2, PGomComplexity: make([]int32, iMbH),
		SScrollResult: SScrollDetectionParam{BScrollDetectFlag: true, IScrollMvY: 24, IScrollMvX: 3}}
	vp.Set(METHOD_COMPLEXITY_ANALYSIS_SCREEN, &cas)
	if ret := vp.Process(METHOD_COMPLEXITY_ANALYSIS_SCREEN, &sCur, &sRef); ret != RET_SUCCESS {
		t.Fatalf("cas: %d", ret)
	}
	vp.Get(METHOD_COMPLEXITY_ANALYSIS_SCREEN, &cas)
	if cas.IGomNumInFrame != int32(iMbH/2) || cas.IFrameComplexity <= 0 {
		t.Errorf("cas result %+v", cas)
	}
	cas.IIdrFlag = 1
	vp.Set(METHOD_COMPLEXITY_ANALYSIS_SCREEN, &cas)
	if ret := vp.Process(METHOD_COMPLEXITY_ANALYSIS_SCREEN, &sCur, nil); ret != RET_SUCCESS {
		t.Fatalf("cas intra: %d", ret)
	}

	// Video scene change: unrelated pictures are a large change.
	other := newTestPic(w, h)
	other.fillRandom(r)
	sOther := other.pixMap()
	var scv SSceneChangeResult
	vp.Set(METHOD_SCENE_CHANGE_DETECTION_VIDEO, &scv)
	vp.Process(METHOD_SCENE_CHANGE_DETECTION_VIDEO, &sCur, &sOther)
	vp.Get(METHOD_SCENE_CHANGE_DETECTION_VIDEO, &scv)
	if scv.ESceneChangeIdc != LARGE_CHANGED_SCENE {
		t.Errorf("video scd idc %d (motion blocks %d)", scv.ESceneChangeIdc, scv.IMotionBlockNum)
	}

	// Denoise must not touch the padding or the picture border rows.
	before := append([]uint8(nil), cur.buf...)
	if ret := vp.Process(METHOD_DENOISE, &sCur, nil); ret != RET_SUCCESS {
		t.Fatalf("denoise: %d", ret)
	}
	for i := 0; i < cur.off[0]; i++ {
		if before[i] != cur.buf[i] {
			t.Fatalf("denoise wrote into top padding at %d", i)
		}
	}

	// Downsampling: dyadic (multi stage), quarter, generic ratio.
	for _, d := range [][2]int32{{w / 2, h / 2}, {w / 4, h / 4}, {200, 120}, {w / 3, h / 3}} {
		dst := newTestPic(d[0], d[1])
		sDst := dst.pixMap()
		if ret := vp.Process(METHOD_DOWNSAMPLE, &sCur, &sDst); ret != RET_SUCCESS {
			t.Fatalf("downsample to %v: %d", d, ret)
		}
	}
	// half average result check
	dst := newTestPic(w/2, h/2)
	sDst := dst.pixMap()
	vp.Process(METHOD_DOWNSAMPLE, &sCur, &sDst)
	for y := int32(0); y < h/2; y++ {
		for x := int32(0); x < w/2; x++ {
			s := cur.buf[cur.off[0]+int(2*y*cur.stride[0]+2*x):]
			st := int(cur.stride[0])
			r1 := (int(s[0]) + int(s[1]) + 1) >> 1
			r2 := (int(s[st]) + int(s[st+1]) + 1) >> 1
			if int(dst.buf[dst.off[0]+int(y*dst.stride[0]+x)]) != (r1+r2+1)>>1 {
				t.Fatalf("half downsample mismatch at %d,%d", x, y)
			}
		}
	}
}

func TestVpCInterface(t *testing.T) {
	var pVPc *IWelsVPc
	if ret := WelsCreateVpInterface(&pVPc, WELSVP_INTERFACE_VERION_C); ret != RET_SUCCESS || pVPc == nil {
		t.Fatalf("create C: %d", ret)
	}
	if ret := pVPc.Init(pVPc.PCtx, METHOD_DENOISE, nil); ret != RET_SUCCESS {
		t.Fatalf("init: %d", ret)
	}
	if ret := pVPc.Get(pVPc.PCtx, METHOD_SCENE_CHANGE_DETECTION_VIDEO, nil); ret != RET_INVALIDPARAM {
		t.Fatalf("get nil: %d", ret)
	}
	if ret := WelsDestroyVpInterface(pVPc, WELSVP_INTERFACE_VERION_C); ret != RET_SUCCESS {
		t.Fatalf("destroy C: %d", ret)
	}
	if ret := WelsCreateVpInterface(&pVPc, 0); ret != RET_INVALIDPARAM {
		t.Fatalf("version 0: %d", ret)
	}
}
