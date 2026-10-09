package common

import (
	"bytes"
	"math/rand"
	"testing"
)

// Port of test/common/ExpandPicture.cpp.

const (
	EXPAND_PIC_TEST_NUM        = 10
	H264_PADDING_LENGTH_LUMA   = PADDING_LENGTH
	H264_PADDING_LENGTH_CHROMA = PADDING_LENGTH >> 1
)

func h264ExpandPictureAnchor_c(pDst []uint8, iDstOff int, iStride, iPicWidth, iPicHeight int32, pad int) {
	stride := int(iStride)
	w := int(iPicWidth)
	pTmp := iDstOff
	pDstLastLine := pTmp + int(iPicHeight-1)*stride
	pTL := pDst[pTmp]
	pTR := pDst[pTmp+w-1]
	pBL := pDst[pDstLastLine]
	pBR := pDst[pDstLastLine+w-1]
	set := func(off int, v uint8, n int) {
		for k := 0; k < n; k++ {
			pDst[off+k] = v
		}
	}
	for i := 0; i < pad; i++ {
		kStrides := (1 + i) * stride
		pTop := pTmp - kStrides
		pBottom := pDstLastLine + kStrides
		copy(pDst[pTop:pTop+w], pDst[pTmp:pTmp+w])
		copy(pDst[pBottom:pBottom+w], pDst[pDstLastLine:pDstLastLine+w])
		set(pTop-pad, pTL, pad)
		set(pTop+w, pTR, pad)
		set(pBottom-pad, pBL, pad)
		set(pBottom+w, pBR, pad)
	}
	for i := 0; i < int(iPicHeight); i++ {
		set(pTmp-pad, pDst[pTmp], pad)
		set(pTmp+w, pDst[pTmp+w-1], pad)
		pTmp += stride
	}
}

func H264ExpandPictureLumaAnchor_c(pDst []uint8, iDstOff int, iStride, iPicWidth, iPicHeight int32) {
	h264ExpandPictureAnchor_c(pDst, iDstOff, iStride, iPicWidth, iPicHeight, H264_PADDING_LENGTH_LUMA)
}

func H264ExpandPictureChromaAnchor_c(pDst []uint8, iDstOff int, iStride, iPicWidth, iPicHeight int32) {
	h264ExpandPictureAnchor_c(pDst, iDstOff, iStride, iPicWidth, iPicHeight, H264_PADDING_LENGTH_CHROMA)
}

func CompareBuff(pSrc0, pSrc1 []uint8, iStride, iWidth, iHeight int32) bool {
	for j := int32(0); j < iHeight; j++ {
		for i := int32(0); i < iWidth; i++ {
			if pSrc0[i+j*iStride] != pSrc1[i+j*iStride] {
				return false
			}
		}
	}
	return true
}

func TestExpandPictureLuma(t *testing.T) {
	var sExpandPicFunc SExpandPicFunc
	var iCpuCores int32 = 1
	rnd := rand.New(rand.NewSource(1))
	for k := 0; k < 2; k++ {
		var uiCpuFlag uint32
		if k != 0 {
			uiCpuFlag = WelsCPUFeatureDetect(&iCpuCores)
		}
		InitExpandPictureFunc(&sExpandPicFunc, uiCpuFlag)
		for iTestIdx := 0; iTestIdx < EXPAND_PIC_TEST_NUM; iTestIdx++ {
			iPicWidth := int32(16 + rnd.Intn(200)*16)
			iPicHeight := int32(16 + rnd.Intn(100)*16)

			iStride := iPicWidth + H264_PADDING_LENGTH_LUMA*2
			iBuffHeight := iPicHeight + H264_PADDING_LENGTH_LUMA*2
			iBuffSize := iBuffHeight * iStride
			pAnchorDstBuff := WelsMallocz(uint32(iBuffSize), "pAnchorDstBuff")
			pAnchorDst := int(H264_PADDING_LENGTH_LUMA*iStride + H264_PADDING_LENGTH_LUMA)
			pTestDstBuff := WelsMallocz(uint32(iBuffSize), "pTestDstBuff")
			pTestDst := pAnchorDst

			for j := int32(0); j < iPicHeight; j++ {
				for i := int32(0); i < iPicWidth; i++ {
					v := uint8(rnd.Intn(256))
					pAnchorDstBuff[pAnchorDst+int(i+j*iStride)] = v
					pTestDstBuff[pTestDst+int(i+j*iStride)] = v
				}
			}
			H264ExpandPictureLumaAnchor_c(pAnchorDstBuff, pAnchorDst, iStride, iPicWidth, iPicHeight)
			sExpandPicFunc.PfExpandLumaPicture(pTestDstBuff, pTestDst, iStride, iPicWidth, iPicHeight)
			if !CompareBuff(pAnchorDstBuff, pTestDstBuff, iStride, iPicWidth+H264_PADDING_LENGTH_LUMA*2,
				iPicHeight+H264_PADDING_LENGTH_LUMA*2) {
				t.Fatalf("luma mismatch %dx%d", iPicWidth, iPicHeight)
			}
		}
	}
}

func TestExpandPictureChroma(t *testing.T) {
	var sExpandPicFunc SExpandPicFunc
	var iCpuCores int32 = 1
	rnd := rand.New(rand.NewSource(2))
	for k := 0; k < 2; k++ {
		var uiCpuFlag uint32
		if k != 0 {
			uiCpuFlag = WelsCPUFeatureDetect(&iCpuCores)
		}
		InitExpandPictureFunc(&sExpandPicFunc, uiCpuFlag)
		for iTestIdx := 0; iTestIdx < EXPAND_PIC_TEST_NUM; iTestIdx++ {
			iPicWidth := int32(8 + rnd.Intn(200)*8)
			if uiCpuFlag&WELS_CPU_SSE2 != 0 {
				iPicWidth = WELS_MAX(iPicWidth, 16)
			}
			iPicHeight := int32(8 + rnd.Intn(100)*8)

			iStride := (iPicWidth + H264_PADDING_LENGTH_CHROMA*2 + 8) >> 4 << 4
			iBuffHeight := iPicHeight + H264_PADDING_LENGTH_CHROMA*2
			iBuffSize := iBuffHeight * iStride
			pAnchorDstBuff := WelsMallocz(uint32(iBuffSize), "pAnchorDstBuff")
			pAnchorDst := int(H264_PADDING_LENGTH_CHROMA*iStride + H264_PADDING_LENGTH_CHROMA)
			pTestDstBuff := WelsMallocz(uint32(iBuffSize), "pTestDstBuff")
			pTestDst := pAnchorDst

			for j := int32(0); j < iPicHeight; j++ {
				for i := int32(0); i < iPicWidth; i++ {
					v := uint8(rnd.Intn(256))
					pAnchorDstBuff[pAnchorDst+int(i+j*iStride)] = v
					pTestDstBuff[pTestDst+int(i+j*iStride)] = v
				}
			}
			H264ExpandPictureChromaAnchor_c(pAnchorDstBuff, pAnchorDst, iStride, iPicWidth, iPicHeight)
			sExpandPicFunc.PfExpandChromaPicture[0](pTestDstBuff, pTestDst, iStride, iPicWidth, iPicHeight)
			if !CompareBuff(pAnchorDstBuff, pTestDstBuff, iStride, iPicWidth+H264_PADDING_LENGTH_CHROMA*2,
				iPicHeight+H264_PADDING_LENGTH_CHROMA*2) {
				t.Fatalf("chroma mismatch %dx%d", iPicWidth, iPicHeight)
			}
		}
	}
}

func TestExpandPicForMotion(t *testing.T) {
	var sExpandPicFunc SExpandPicFunc
	var iCpuCores int32 = 1
	rnd := rand.New(rand.NewSource(3))
	for k := 0; k < 2; k++ {
		var uiCpuFlag uint32
		if k != 0 {
			uiCpuFlag = WelsCPUFeatureDetect(&iCpuCores)
		}
		InitExpandPictureFunc(&sExpandPicFunc, uiCpuFlag)
		var iStride [3]int32
		for iTestIdx := 0; iTestIdx < EXPAND_PIC_TEST_NUM; iTestIdx++ {
			iPicWidth := int32(16 + rnd.Intn(200)*16)
			iPicHeight := int32(16 + rnd.Intn(100)*16)
			if uiCpuFlag&WELS_CPU_SSE2 != 0 {
				iPicWidth = WELS_ALIGN(iPicWidth, 32)
			}
			iStride[0] = WELS_ALIGN(iPicWidth, MB_WIDTH_LUMA) + (PADDING_LENGTH << 1)       // with width of horizon
			iPicHeightExt := WELS_ALIGN(iPicHeight, MB_HEIGHT_LUMA) + (PADDING_LENGTH << 1) // with height of vertical
			iStride[1] = iStride[0] >> 1
			iPicChromaHeightExt := iPicHeightExt >> 1
			iStride[2] = iStride[1]
			iLumaSize := iStride[0] * iPicHeightExt
			iChromaSize := iStride[1] * iPicChromaHeightExt

			pPicAnchorBuffer := WelsMallocz(uint32(iLumaSize+(iChromaSize<<1)), "pPicAnchor")
			pPicTestBuffer := WelsMallocz(uint32(iLumaSize+(iChromaSize<<1)), "pPicTest")
			var pOff [3]int
			pOff[0] = int((1 + iStride[0]) * PADDING_LENGTH)
			pOff[1] = int(iLumaSize + (((1 + iStride[1]) * PADDING_LENGTH) >> 1))
			pOff[2] = int(iLumaSize + iChromaSize + (((1 + iStride[2]) * PADDING_LENGTH) >> 1))

			for j := int32(0); j < iPicHeight; j++ {
				for i := int32(0); i < iPicWidth; i++ {
					v := uint8(rnd.Intn(256))
					pPicAnchorBuffer[pOff[0]+int(i+j*iStride[0])] = v
					pPicTestBuffer[pOff[0]+int(i+j*iStride[0])] = v
				}
			}
			for j := int32(0); j < iPicHeight/2; j++ {
				for i := int32(0); i < iPicWidth/2; i++ {
					v := uint8(rnd.Intn(256))
					pPicAnchorBuffer[pOff[1]+int(i+j*iStride[1])] = v
					pPicTestBuffer[pOff[1]+int(i+j*iStride[1])] = v
					v = uint8(rnd.Intn(256))
					pPicAnchorBuffer[pOff[2]+int(i+j*iStride[2])] = v
					pPicTestBuffer[pOff[2]+int(i+j*iStride[2])] = v
				}
			}
			H264ExpandPictureLumaAnchor_c(pPicAnchorBuffer, pOff[0], iStride[0], iPicWidth, iPicHeight)
			H264ExpandPictureChromaAnchor_c(pPicAnchorBuffer, pOff[1], iStride[1], iPicWidth/2, iPicHeight/2)
			H264ExpandPictureChromaAnchor_c(pPicAnchorBuffer, pOff[2], iStride[2], iPicWidth/2, iPicHeight/2)
			pPicTest := [][]uint8{pPicTestBuffer, pPicTestBuffer, pPicTestBuffer}
			ExpandReferencingPicture(pPicTest, pOff[:], iPicWidth, iPicHeight, iStride[:],
				sExpandPicFunc.PfExpandLumaPicture, sExpandPicFunc.PfExpandChromaPicture)
			if !bytes.Equal(pPicAnchorBuffer, pPicTestBuffer) {
				t.Fatalf("picture mismatch %dx%d", iPicWidth, iPicHeight)
			}
		}
	}
}

// TestPadMB checks that padding every MB with PadMBLuma_c / PadMBChroma_c
// gives the same result as expanding the whole picture.
func TestPadMB(t *testing.T) {
	rnd := rand.New(rand.NewSource(4))
	for iTest := 0; iTest < 5; iTest++ {
		iMbW := int32(2 + rnd.Intn(6))
		iMbH := int32(2 + rnd.Intn(6))
		for _, luma := range []bool{true, false} {
			mbSize, pad := int32(16), int32(PADDING_LENGTH)
			if !luma {
				mbSize, pad = 8, CHROMA_PADDING_LENGTH
			}
			w, h := iMbW*mbSize, iMbH*mbSize
			stride := w + 2*pad
			size := stride * (h + 2*pad)
			a := make([]uint8, size)
			b := make([]uint8, size)
			off := int(pad*stride + pad)
			for j := int32(0); j < h; j++ {
				for i := int32(0); i < w; i++ {
					v := uint8(rnd.Intn(256))
					a[off+int(j*stride+i)] = v
					b[off+int(j*stride+i)] = v
				}
			}
			for y := int32(0); y < iMbH; y++ {
				for x := int32(0); x < iMbW; x++ {
					if luma {
						PadMBLuma_c(a, off, stride, w, h, x, y, iMbW, iMbH)
					} else {
						PadMBChroma_c(a, off, stride, w, h, x, y, iMbW, iMbH)
					}
				}
			}
			if luma {
				ExpandPictureLuma_c(b, off, stride, w, h)
			} else {
				ExpandPictureChroma_c(b, off, stride, w, h)
			}
			if !bytes.Equal(a, b) {
				t.Fatalf("PadMB mismatch luma=%v %dx%d MBs", luma, iMbW, iMbH)
			}
		}
	}
}
