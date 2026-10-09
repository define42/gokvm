package common

// Port of codec/common/inc/deblocking_common.h and
// codec/common/src/deblocking_common.cpp (C code only).
//
// Pixel pointers are (slice, offset) pairs pointing at the first q0 sample
// of the edge; the filters reach backwards (p samples) through negative
// offsets. pTc is the 4 entry tc0 table (int8).

//  C code only

func DeblockLumaLt4_c(pPix []uint8, iPixOff int, iStrideX, iStrideY int32, iAlpha, iBeta int32, pTc []int8) {
	sx := int(iStrideX)
	for i := 0; i < 16; i++ {
		iTc0 := int32(pTc[i>>2])
		if iTc0 >= 0 {
			p0 := int32(pPix[iPixOff-sx])
			p1 := int32(pPix[iPixOff-2*sx])
			p2 := int32(pPix[iPixOff-3*sx])
			q0 := int32(pPix[iPixOff])
			q1 := int32(pPix[iPixOff+sx])
			q2 := int32(pPix[iPixOff+2*sx])
			bDetaP0Q0 := WELS_ABS(p0-q0) < iAlpha
			bDetaP1P0 := WELS_ABS(p1-p0) < iBeta
			bDetaQ1Q0 := WELS_ABS(q1-q0) < iBeta
			iTc := iTc0
			if bDetaP0Q0 && bDetaP1P0 && bDetaQ1Q0 {
				bDetaP2P0 := WELS_ABS(p2-p0) < iBeta
				bDetaQ2Q0 := WELS_ABS(q2-q0) < iBeta
				if bDetaP2P0 {
					pPix[iPixOff-2*sx] = uint8(p1 + WELS_CLIP3((p2+((p0+q0+1)>>1)-(p1*(1<<1)))>>1, -iTc0, iTc0))
					iTc++
				}
				if bDetaQ2Q0 {
					pPix[iPixOff+sx] = uint8(q1 + WELS_CLIP3((q2+((p0+q0+1)>>1)-(q1*(1<<1)))>>1, -iTc0, iTc0))
					iTc++
				}
				iDeta := WELS_CLIP3((((q0-p0)*(1<<2))+(p1-q1)+4)>>3, -iTc, iTc)
				pPix[iPixOff-sx] = WelsClip1(p0 + iDeta) /* p0' */
				pPix[iPixOff] = WelsClip1(q0 - iDeta)    /* q0' */
			}
		}
		iPixOff += int(iStrideY)
	}
}

func DeblockLumaEq4_c(pPix []uint8, iPixOff int, iStrideX, iStrideY int32, iAlpha, iBeta int32) {
	var p0, p1, p2, q0, q1, q2 int32
	var iDetaP0Q0 int32
	var bDetaP1P0, bDetaQ1Q0 bool
	sx := int(iStrideX)
	for i := 0; i < 16; i++ {
		p0 = int32(pPix[iPixOff-sx])
		p1 = int32(pPix[iPixOff-2*sx])
		p2 = int32(pPix[iPixOff-3*sx])
		q0 = int32(pPix[iPixOff])
		q1 = int32(pPix[iPixOff+sx])
		q2 = int32(pPix[iPixOff+2*sx])
		iDetaP0Q0 = WELS_ABS(p0 - q0)
		bDetaP1P0 = WELS_ABS(p1-p0) < iBeta
		bDetaQ1Q0 = WELS_ABS(q1-q0) < iBeta
		if (iDetaP0Q0 < iAlpha) && bDetaP1P0 && bDetaQ1Q0 {
			if iDetaP0Q0 < ((iAlpha >> 2) + 2) {
				bDetaP2P0 := WELS_ABS(p2-p0) < iBeta
				bDetaQ2Q0 := WELS_ABS(q2-q0) < iBeta
				if bDetaP2P0 {
					p3 := int32(pPix[iPixOff-4*sx])
					pPix[iPixOff-sx] = uint8((p2 + (p1 * (1 << 1)) + (p0 * (1 << 1)) + (q0 * (1 << 1)) + q1 + 4) >> 3) //p0
					pPix[iPixOff-2*sx] = uint8((p2 + p1 + p0 + q0 + 2) >> 2)                                           //p1
					pPix[iPixOff-3*sx] = uint8(((p3 * (1 << 1)) + p2 + (p2 * (1 << 1)) + p1 + p0 + q0 + 4) >> 3)       //p2
				} else {
					pPix[iPixOff-1*sx] = uint8(((p1 * (1 << 1)) + p0 + q1 + 2) >> 2) //p0
				}
				if bDetaQ2Q0 {
					q3 := int32(pPix[iPixOff+3*sx])
					pPix[iPixOff] = uint8((p1 + (p0 * (1 << 1)) + (q0 * (1 << 1)) + (q1 * (1 << 1)) + q2 + 4) >> 3) //q0
					pPix[iPixOff+sx] = uint8((p0 + q0 + q1 + q2 + 2) >> 2)                                          //q1
					pPix[iPixOff+2*sx] = uint8(((q3 * (1 << 1)) + q2 + (q2 * (1 << 1)) + q1 + q0 + p0 + 4) >> 3)    //q2
				} else {
					pPix[iPixOff] = uint8(((q1 * (1 << 1)) + q0 + p1 + 2) >> 2) //q0
				}
			} else {
				pPix[iPixOff-sx] = uint8(((p1 * (1 << 1)) + p0 + q1 + 2) >> 2) //p0
				pPix[iPixOff] = uint8(((q1 * (1 << 1)) + q0 + p1 + 2) >> 2)    //q0
			}
		}
		iPixOff += int(iStrideY)
	}
}

func DeblockLumaLt4V_c(pPix []uint8, iPixOff int, iStride int32, iAlpha, iBeta int32, tc []int8) {
	DeblockLumaLt4_c(pPix, iPixOff, iStride, 1, iAlpha, iBeta, tc)
}

func DeblockLumaLt4H_c(pPix []uint8, iPixOff int, iStride int32, iAlpha, iBeta int32, tc []int8) {
	DeblockLumaLt4_c(pPix, iPixOff, 1, iStride, iAlpha, iBeta, tc)
}

func DeblockLumaEq4V_c(pPix []uint8, iPixOff int, iStride int32, iAlpha, iBeta int32) {
	DeblockLumaEq4_c(pPix, iPixOff, iStride, 1, iAlpha, iBeta)
}

func DeblockLumaEq4H_c(pPix []uint8, iPixOff int, iStride int32, iAlpha, iBeta int32) {
	DeblockLumaEq4_c(pPix, iPixOff, 1, iStride, iAlpha, iBeta)
}

func DeblockChromaLt4_c(pPixCb []uint8, iPixCbOff int, pPixCr []uint8, iPixCrOff int, iStrideX, iStrideY int32,
	iAlpha, iBeta int32, pTc []int8) {
	var p0, p1, q0, q1, iDeta int32
	var bDetaP0Q0, bDetaP1P0, bDetaQ1Q0 bool
	sx := int(iStrideX)

	for i := 0; i < 8; i++ {
		iTc0 := int32(pTc[i>>1])
		if iTc0 > 0 {
			p0 = int32(pPixCb[iPixCbOff-sx])
			p1 = int32(pPixCb[iPixCbOff-2*sx])
			q0 = int32(pPixCb[iPixCbOff])
			q1 = int32(pPixCb[iPixCbOff+sx])

			bDetaP0Q0 = WELS_ABS(p0-q0) < iAlpha
			bDetaP1P0 = WELS_ABS(p1-p0) < iBeta
			bDetaQ1Q0 = WELS_ABS(q1-q0) < iBeta
			if bDetaP0Q0 && bDetaP1P0 && bDetaQ1Q0 {
				iDeta = WELS_CLIP3((((q0-p0)*(1<<2))+(p1-q1)+4)>>3, -iTc0, iTc0)
				pPixCb[iPixCbOff-sx] = WelsClip1(p0 + iDeta) /* p0' */
				pPixCb[iPixCbOff] = WelsClip1(q0 - iDeta)    /* q0' */
			}

			p0 = int32(pPixCr[iPixCrOff-sx])
			p1 = int32(pPixCr[iPixCrOff-2*sx])
			q0 = int32(pPixCr[iPixCrOff])
			q1 = int32(pPixCr[iPixCrOff+sx])

			bDetaP0Q0 = WELS_ABS(p0-q0) < iAlpha
			bDetaP1P0 = WELS_ABS(p1-p0) < iBeta
			bDetaQ1Q0 = WELS_ABS(q1-q0) < iBeta

			if bDetaP0Q0 && bDetaP1P0 && bDetaQ1Q0 {
				iDeta = WELS_CLIP3((((q0-p0)*(1<<2))+(p1-q1)+4)>>3, -iTc0, iTc0)
				pPixCr[iPixCrOff-sx] = WelsClip1(p0 + iDeta) /* p0' */
				pPixCr[iPixCrOff] = WelsClip1(q0 - iDeta)    /* q0' */
			}
		}
		iPixCbOff += int(iStrideY)
		iPixCrOff += int(iStrideY)
	}
}

func DeblockChromaEq4_c(pPixCb []uint8, iPixCbOff int, pPixCr []uint8, iPixCrOff int, iStrideX, iStrideY int32,
	iAlpha, iBeta int32) {
	var p0, p1, q0, q1 int32
	var bDetaP0Q0, bDetaP1P0, bDetaQ1Q0 bool
	sx := int(iStrideX)
	for i := 0; i < 8; i++ {
		//cb
		p0 = int32(pPixCb[iPixCbOff-sx])
		p1 = int32(pPixCb[iPixCbOff-2*sx])
		q0 = int32(pPixCb[iPixCbOff])
		q1 = int32(pPixCb[iPixCbOff+sx])
		bDetaP0Q0 = WELS_ABS(p0-q0) < iAlpha
		bDetaP1P0 = WELS_ABS(p1-p0) < iBeta
		bDetaQ1Q0 = WELS_ABS(q1-q0) < iBeta
		if bDetaP0Q0 && bDetaP1P0 && bDetaQ1Q0 {
			pPixCb[iPixCbOff-sx] = uint8(((p1 * (1 << 1)) + p0 + q1 + 2) >> 2) /* p0' */
			pPixCb[iPixCbOff] = uint8(((q1 * (1 << 1)) + q0 + p1 + 2) >> 2)    /* q0' */
		}

		//cr
		p0 = int32(pPixCr[iPixCrOff-sx])
		p1 = int32(pPixCr[iPixCrOff-2*sx])
		q0 = int32(pPixCr[iPixCrOff])
		q1 = int32(pPixCr[iPixCrOff+sx])
		bDetaP0Q0 = WELS_ABS(p0-q0) < iAlpha
		bDetaP1P0 = WELS_ABS(p1-p0) < iBeta
		bDetaQ1Q0 = WELS_ABS(q1-q0) < iBeta
		if bDetaP0Q0 && bDetaP1P0 && bDetaQ1Q0 {
			pPixCr[iPixCrOff-sx] = uint8(((p1 * (1 << 1)) + p0 + q1 + 2) >> 2) /* p0' */
			pPixCr[iPixCrOff] = uint8(((q1 * (1 << 1)) + q0 + p1 + 2) >> 2)    /* q0' */
		}
		iPixCrOff += int(iStrideY)
		iPixCbOff += int(iStrideY)
	}
}

func DeblockChromaLt4V_c(pPixCb []uint8, iPixCbOff int, pPixCr []uint8, iPixCrOff int, iStride int32,
	iAlpha, iBeta int32, tc []int8) {
	DeblockChromaLt4_c(pPixCb, iPixCbOff, pPixCr, iPixCrOff, iStride, 1, iAlpha, iBeta, tc)
}

func DeblockChromaLt4H_c(pPixCb []uint8, iPixCbOff int, pPixCr []uint8, iPixCrOff int, iStride int32,
	iAlpha, iBeta int32, tc []int8) {
	DeblockChromaLt4_c(pPixCb, iPixCbOff, pPixCr, iPixCrOff, 1, iStride, iAlpha, iBeta, tc)
}

func DeblockChromaEq4V_c(pPixCb []uint8, iPixCbOff int, pPixCr []uint8, iPixCrOff int, iStride int32,
	iAlpha, iBeta int32) {
	DeblockChromaEq4_c(pPixCb, iPixCbOff, pPixCr, iPixCrOff, iStride, 1, iAlpha, iBeta)
}

func DeblockChromaEq4H_c(pPixCb []uint8, iPixCbOff int, pPixCr []uint8, iPixCrOff int, iStride int32,
	iAlpha, iBeta int32) {
	DeblockChromaEq4_c(pPixCb, iPixCbOff, pPixCr, iPixCrOff, 1, iStride, iAlpha, iBeta)
}

func DeblockChromaLt42_c(pPixCbCr []uint8, iPixOff int, iStrideX, iStrideY int32, iAlpha, iBeta int32, pTc []int8) {
	var p0, p1, q0, q1, iDeta int32
	var bDetaP0Q0, bDetaP1P0, bDetaQ1Q0 bool
	sx := int(iStrideX)

	for i := 0; i < 8; i++ {
		iTc0 := int32(pTc[i>>1])
		if iTc0 > 0 {
			p0 = int32(pPixCbCr[iPixOff-sx])
			p1 = int32(pPixCbCr[iPixOff-2*sx])
			q0 = int32(pPixCbCr[iPixOff])
			q1 = int32(pPixCbCr[iPixOff+sx])

			bDetaP0Q0 = WELS_ABS(p0-q0) < iAlpha
			bDetaP1P0 = WELS_ABS(p1-p0) < iBeta
			bDetaQ1Q0 = WELS_ABS(q1-q0) < iBeta
			if bDetaP0Q0 && bDetaP1P0 && bDetaQ1Q0 {
				iDeta = WELS_CLIP3((((q0-p0)*(1<<2))+(p1-q1)+4)>>3, -iTc0, iTc0)
				pPixCbCr[iPixOff-sx] = WelsClip1(p0 + iDeta) /* p0' */
				pPixCbCr[iPixOff] = WelsClip1(q0 - iDeta)    /* q0' */
			}
		}
		iPixOff += int(iStrideY)
	}
}

func DeblockChromaEq42_c(pPixCbCr []uint8, iPixOff int, iStrideX, iStrideY int32, iAlpha, iBeta int32) {
	var p0, p1, q0, q1 int32
	var bDetaP0Q0, bDetaP1P0, bDetaQ1Q0 bool
	sx := int(iStrideX)
	for i := 0; i < 8; i++ {
		p0 = int32(pPixCbCr[iPixOff-sx])
		p1 = int32(pPixCbCr[iPixOff-2*sx])
		q0 = int32(pPixCbCr[iPixOff])
		q1 = int32(pPixCbCr[iPixOff+sx])
		bDetaP0Q0 = WELS_ABS(p0-q0) < iAlpha
		bDetaP1P0 = WELS_ABS(p1-p0) < iBeta
		bDetaQ1Q0 = WELS_ABS(q1-q0) < iBeta
		if bDetaP0Q0 && bDetaP1P0 && bDetaQ1Q0 {
			pPixCbCr[iPixOff-sx] = uint8(((p1 * (1 << 1)) + p0 + q1 + 2) >> 2) /* p0' */
			pPixCbCr[iPixOff] = uint8(((q1 * (1 << 1)) + q0 + p1 + 2) >> 2)    /* q0' */
		}

		iPixOff += int(iStrideY)
	}
}

func DeblockChromaLt4V2_c(pPixCbCr []uint8, iPixOff int, iStride int32, iAlpha, iBeta int32, tc []int8) {
	DeblockChromaLt42_c(pPixCbCr, iPixOff, iStride, 1, iAlpha, iBeta, tc)
}

func DeblockChromaLt4H2_c(pPixCbCr []uint8, iPixOff int, iStride int32, iAlpha, iBeta int32, tc []int8) {
	DeblockChromaLt42_c(pPixCbCr, iPixOff, 1, iStride, iAlpha, iBeta, tc)
}

func DeblockChromaEq4V2_c(pPixCbCr []uint8, iPixOff int, iStride int32, iAlpha, iBeta int32) {
	DeblockChromaEq42_c(pPixCbCr, iPixOff, iStride, 1, iAlpha, iBeta)
}

func DeblockChromaEq4H2_c(pPixCbCr []uint8, iPixOff int, iStride int32, iAlpha, iBeta int32) {
	DeblockChromaEq42_c(pPixCbCr, iPixOff, 1, iStride, iAlpha, iBeta)
}

// WelsNonZeroCount_c normalizes the 24 non-zero counts to 0/1.
func WelsNonZeroCount_c(pNonZeroCount []int8) {
	for i := 0; i < 24; i++ {
		if pNonZeroCount[i] != 0 {
			pNonZeroCount[i] = 1
		} else {
			pNonZeroCount[i] = 0
		}
	}
}
