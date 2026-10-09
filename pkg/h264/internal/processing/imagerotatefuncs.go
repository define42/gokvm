package processing

// Port of codec/processing/src/imagerotate/imagerotatefuncs.cpp.

func ImageRotate90D_c(pSrc []uint8, uiBytesPerPixel uint32, iWidth uint32, iHeight uint32, pDst []uint8) {
	for j := uint32(0); j < iHeight; j++ {
		for i := uint32(0); i < iWidth; i++ {
			for n := uint32(0); n < uiBytesPerPixel; n++ {
				pDst[(i*iHeight+iHeight-1-j)*uiBytesPerPixel+n] = pSrc[(iWidth*j+i)*uiBytesPerPixel+n]
			}
		}
	}
}
func ImageRotate180D_c(pSrc []uint8, uiBytesPerPixel uint32, iWidth uint32, iHeight uint32, pDst []uint8) {
	for j := uint32(0); j < iHeight; j++ {
		for i := uint32(0); i < iWidth; i++ {
			for n := uint32(0); n < uiBytesPerPixel; n++ {
				pDst[((iHeight-1-j)*iWidth+iWidth-1-i)*uiBytesPerPixel+n] = pSrc[(iWidth*j+i)*uiBytesPerPixel+n]
			}
		}
	}
}
func ImageRotate270D_c(pSrc []uint8, uiBytesPerPixel uint32, iWidth uint32, iHeight uint32, pDst []uint8) {
	for j := uint32(0); j < iWidth; j++ {
		for i := uint32(0); i < iHeight; i++ {
			for n := uint32(0); n < uiBytesPerPixel; n++ {
				pDst[((iWidth-1-j)*iHeight+i)*uiBytesPerPixel+n] = pSrc[(iWidth*i+j)*uiBytesPerPixel+n]
			}
		}
	}
}
