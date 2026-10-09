package common

// Port of codec/common/inc/intra_pred_common.h and
// codec/common/src/intra_pred_common.cpp (C code only).
//
// pPred is the 16x16 prediction block (stride 16) at pPred[iPredOff:];
// pRef/iRefOff points at the top-left sample of the current MB in the
// reconstructed picture (the predictor reaches up/left through negative
// offsets).

// WelsI16x16LumaPredV_c: vertical 16x16 luma prediction.
func WelsI16x16LumaPredV_c(pPred []uint8, iPredOff int, pRef []uint8, iRefOff int, kiStride int32) {
	kpSrc := pRef[iRefOff-int(kiStride) : iRefOff-int(kiStride)+16]
	for i := 0; i < 16; i++ {
		copy(pPred[iPredOff+(i<<4):iPredOff+(i<<4)+16], kpSrc)
	}
}

// WelsI16x16LumaPredH_c: horizontal 16x16 luma prediction.
func WelsI16x16LumaPredH_c(pPred []uint8, iPredOff int, pRef []uint8, iRefOff int, kiStride int32) {
	iStridex15 := (kiStride << 4) - kiStride
	const iPredStride = 16
	iPredStridex15 := 240 //(iPredStride<<4)-iPredStride;
	i := uint8(15)

	for {
		kuiSrc8 := pRef[iRefOff+int(iStridex15)-1]
		row := pPred[iPredOff+iPredStridex15 : iPredOff+iPredStridex15+16]
		for k := range row {
			row[k] = kuiSrc8
		}

		iStridex15 -= kiStride
		iPredStridex15 -= iPredStride
		if i == 0 {
			break
		}
		i--
	}
}
