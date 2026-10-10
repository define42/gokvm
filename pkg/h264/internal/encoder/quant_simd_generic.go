//go:build !amd64

package encoder

// Keep the architecture-selected entry points available to direct callers.
// Architectures without the amd64 kernels use the bit-exact scalar routines.
func WelsQuant4x4_sse2(dct, ff, mf []int16) {
	WelsQuant4x4_c(dct, ff, mf)
}

func WelsQuant4x4Dc_sse2(dct []int16, ff, mf int16) {
	WelsQuant4x4Dc_c(dct, ff, mf)
}

func WelsQuantFour4x4_sse2(dct, ff, mf []int16) {
	WelsQuantFour4x4_c(dct, ff, mf)
}

func WelsQuantFour4x4Max_sse2(dct, ff, mf []int16, max []int16) {
	WelsQuantFour4x4Max_c(dct, ff, mf, max)
}

// No quantization SIMD kernels have been ported for this architecture, so the
// function table keeps its scalar defaults.
func initQuantizationSIMD(funcs *SWelsFuncPtrList, cpuFlags uint32) {}
