package processing

// Port of codec/processing/src/denoise/denoise.h.

const (
	DENOISE_GRAY_RADIUS = 1
	DENOISE_GRAY_SIGMA  = 2

	UV_WINDOWS_RADIUS = 2
	TAIL_OF_LINE8     = 7

	DENOISE_Y_COMPONENT   = 1
	DENOISE_U_COMPONENT   = 2
	DENOISE_V_COMPONENT   = 4
	DENOISE_ALL_COMPONENT = 7
)

// DenoiseFilterFunc filters 8 samples starting at pixels[iOff]; reads the
// neighbouring rows/columns around them.
type DenoiseFilterFunc func(pixels []uint8, iOff int, stride int32)

type DenoiseFilterFuncPtr = DenoiseFilterFunc

type SDenoiseFuncs struct {
	pfBilateralLumaFilter8  DenoiseFilterFuncPtr //on 8 samples
	pfWaverageChromaFilter8 DenoiseFilterFuncPtr //on 8 samples
}

type CDenoiser struct {
	IStrategyBase
	m_fSigmaGrey    float32 //sigma for grey scale similarity, suggestion 2.5-3
	m_uiSpaceRadius uint16  //filter windows radius: 1-3x3, 2-5x5,3-7x7. Larger size, slower speed
	m_uiType        uint16  //do denoising on which component 1-Y, 2-U, 4-V; 7-YUV, 3-YU, 5-YV, 6-UV

	m_pfDenoise SDenoiseFuncs
	m_CPUFlag   int32
}
