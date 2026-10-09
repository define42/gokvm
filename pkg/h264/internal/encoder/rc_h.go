// Port of codec/encoder/core/inc/rc.h.

package encoder

// trace
const (
	GOM_TRACE_FLAG = 0
	GOM_H_SCC      = 8
)

const (
	BITS_NORMAL = iota
	BITS_LIMITED
	BITS_EXCEEDED
)

const (
	// virtual gop size
	VGOP_SIZE = 8

	// qp information
	GOM_MIN_QP_MODE  = 12
	GOM_MAX_QP_MODE  = 36
	MAX_LOW_BR_QP    = 42
	MIN_IDR_QP       = 26
	MAX_IDR_QP       = 32
	MIN_SCREEN_QP    = 26
	MAX_SCREEN_QP    = 35
	DELTA_QP         = 2
	DELTA_QP_BGD_THD = 3
	QP_MIN_VALUE     = 0
	QP_MAX_VALUE     = 51

	// frame skip constants
	SKIP_QP_90P                     = 24
	SKIP_QP_180P                    = 24
	SKIP_QP_360P                    = 31
	SKIP_QP_720P                    = 31
	LAST_FRAME_QP_RANGE_UPPER_MODE0 = 3
	LAST_FRAME_QP_RANGE_LOWER_MODE0 = 2
	LAST_FRAME_QP_RANGE_UPPER_MODE1 = 5
	LAST_FRAME_QP_RANGE_LOWER_MODE1 = 3

	MB_WIDTH_THRESHOLD_90P  = 15
	MB_WIDTH_THRESHOLD_180P = 30
	MB_WIDTH_THRESHOLD_360P = 60

	// Mode 0 parameter
	GOM_ROW_MODE0_90P  = 2
	GOM_ROW_MODE0_180P = 2
	GOM_ROW_MODE0_360P = 4
	GOM_ROW_MODE0_720P = 4
	QP_RANGE_MODE0     = 3

	// Mode 1 parameter
	GOM_ROW_MODE1_90P    = 1
	GOM_ROW_MODE1_180P   = 1
	GOM_ROW_MODE1_360P   = 2
	GOM_ROW_MODE1_720P   = 2
	QP_RANGE_UPPER_MODE1 = 9
	QP_RANGE_LOWER_MODE1 = 4
	QP_RANGE_INTRA_MODE1 = 3
)

// bits allocation
const (
	MAX_BITS_VARY_PERCENTAGE      = 100  // bits vary range in percentage
	MAX_BITS_VARY_PERCENTAGE_x3d2 = 150  // bits vary range in percentage * 3/2
	INT_MULTIPLY                  = 100  // use to multiply in Double to Int Conversion, should be same as AQ_QSTEP_INT_MULTIPLY in WelsVP
	WEIGHT_MULTIPLY               = 2000 //
	REMAIN_BITS_TH                = 1
	VGOP_BITS_PERCENTAGE_DIFF     = 5
	IDR_BITRATE_RATIO             = 4
	FRAME_iTargetBits_VARY_RANGE  = 50 // *INT_MULTIPLY
	// R-Q Model
	LINEAR_MODEL_DECAY_FACTOR = 80 // *INT_MULTIPLY
	FRAME_CMPLX_RATIO_RANGE   = 20 // *INT_MULTIPLY
	SMOOTH_FACTOR_MIN_VALUE   = 2  // *INT_MULTIPLY
	// skip and padding
	TIME_CHECK_WINDOW         = 5000 // ms
	SKIP_RATIO                = 50   // *INT_MULTIPLY
	LAST_FRAME_PREDICT_WEIGHT = 0.5  // (double in C)
	PADDING_BUFFER_RATIO      = 50   // *INT_MULTIPLY
	PADDING_THRESHOLD         = 5    // *INT_MULTIPLY

	VIRTUAL_BUFFER_LOW_TH  = 120 // *INT_MULTIPLY
	VIRTUAL_BUFFER_HIGH_TH = 180 // *INT_MULTIPLY

	_BITS_RANGE = 0
)

const (
	EVEN_TIME_WINDOW  = 0
	ODD_TIME_WINDOW   = 1
	TIME_WINDOW_TOTAL = 2
)

// SRCTemporal holds the per temporal layer rc state.
type SRCTemporal struct {
	iMinBitsTl    int32
	iMaxBitsTl    int32
	iTlayerWeight int32
	iGopBitsDq    int32
	// P frame level R-Q Model
	iLinearCmplx    int64 // *INT_MULTIPLY
	iPFrameNum      int32
	iFrameCmplxMean int64
	iMaxQp          int32
	iMinQp          int32
}

// SWelsSvcRc holds the per dependency layer rc state.
type SWelsSvcRc struct {
	iRcVaryPercentage int32
	iRcVaryRatio      int32

	iInitialQp       int32 // initial qp
	iBitRate         int64 // Note: although the max bit rate is 240000*1200 which can be represented by int32, but there are many multipler of this iBitRate in the calculation of RC, so use int64 to avoid type conversion at all such places
	iPreviousBitrate int32
	iPreviousGopSize int32
	fFrameRate       float64
	iBitsPerFrame    int32
	iMaxBitsPerFrame int32
	dPreviousFps     float64

	// bits allocation and status
	iLastAllocatedBits int32
	iRemainingBits     int32
	iBitsPerMb         int32
	iTargetBits        int32
	iCurrentBitsLevel  int32 // 0:normal; 1:limited; 2:exceeded.

	iIdrNum          int32
	iIntraComplexity int64 // 255*255(MaxMbSAD)*36864(MaxFS) make the highest bit of 32-bit integer 1
	iIntraMbCount    int32
	iIntraComplxMean int64

	iTlOfFrames       [VGOP_SIZE]int8
	iRemainingWeights int32
	iFrameDqBits      int32

	bGomRC                 bool
	pGomComplexity         []float64 // C double*: per GOM
	pGomForegroundBlockNum []int32   // C int32_t*: per GOM
	pCurrentFrameGomSad    []int32   // C int32_t*: per GOM
	pGomCost               []int32   // C int32_t*: per GOM

	bEnableGomQp    int32
	iAverageFrameQp int32
	iMinFrameQp     int32
	iMaxFrameQp     int32
	iNumberMbFrame  int32
	iNumberMbGom    int32
	iGomSize        int32

	iSkipFrameNum     int32
	iFrameCodedInVGop int32
	iSkipFrameInVGop  int32
	iGopNumberInVGop  int32
	iGopIndexInVGop   int32

	iSkipQpValue         int32
	iQpRangeUpperInFrame int32
	iQpRangeLowerInFrame int32
	iMinQp               int32
	iMaxQp               int32
	iSkipBufferRatio     int32

	iQStep                int32 // *INT_MULTIPLY
	iFrameDeltaQpUpper    int32
	iFrameDeltaQpLower    int32
	iLastCalculatedQScale int32

	// for skip frame and padding
	iBufferSizeSkip        int32
	iBufferFullnessSkip    int64
	iBufferMaxBRFullness   [TIME_WINDOW_TOTAL]int64 // 0: EVEN_TIME_WINDOW; 1: ODD_TIME_WINDOW
	iPredFrameBit          int32
	bNeedShiftWindowCheck  [TIME_WINDOW_TOTAL]bool
	iBufferSizePadding     int32
	iBufferFullnessPadding int32
	iPaddingSize           int32
	iPaddingBitrateStat    int32
	bSkipFlag              bool
	iContinualSkipFrames   int32

	pTemporalOverRc []SRCTemporal // C SRCTemporal*: per temporal layer

	// for scc
	iAvgCost2Bits   int64
	iCost2BitsIntra int64
	iBaseQp         int32
	uiLastTimeStamp int64 // long long

	// for statistics and online adjustments
	iActualBitRate   int32   // TODO: to complete later
	fLatestFrameRate float32 // TODO: to complete later
}

type PWelsRCPictureInitFunc func(pCtx *sWelsEncCtx, uiTimeStamp int64)
type PWelsRCPictureDelayJudgeFunc func(pCtx *sWelsEncCtx, uiTimeStamp int64, iDidIdx int32)
type PWelsRCPictureInfoUpdateFunc func(pCtx *sWelsEncCtx, iLayerSize int32)
type PWelsRCMBInfoUpdateFunc func(pCtx *sWelsEncCtx, pCurMb *SMB, iCostLuma int32, pSlice *SSlice)
type PWelsRCMBInitFunc func(pCtx *sWelsEncCtx, pCurMb *SMB, pSlice *SSlice)
type PWelsCheckFrameSkipBasedMaxbrFunc func(pCtx *sWelsEncCtx, uiTimeStamp int64, iDidIdx int32)
type PWelsUpdateBufferWhenFrameSkippedFunc func(pCtx *sWelsEncCtx, iSpatialNum int32)
type PWelsUpdateMaxBrCheckWindowStatusFunc func(pCtx *sWelsEncCtx, iSpatialNum int32, uiTimeStamp int64)
type PWelsRCPostFrameSkippingFunc func(pCtx *sWelsEncCtx, iDid int32, uiTimeStamp int64) bool

// SWelsRcFunc is the rate control function table.
type SWelsRcFunc struct {
	pfWelsRcPictureInit           PWelsRCPictureInitFunc
	pfWelsRcPicDelayJudge         PWelsRCPictureDelayJudgeFunc
	pfWelsRcPictureInfoUpdate     PWelsRCPictureInfoUpdateFunc
	pfWelsRcMbInit                PWelsRCMBInitFunc
	pfWelsRcMbInfoUpdate          PWelsRCMBInfoUpdateFunc
	pfWelsCheckSkipBasedMaxbr     PWelsCheckFrameSkipBasedMaxbrFunc
	pfWelsUpdateBufferWhenSkip    PWelsUpdateBufferWhenFrameSkippedFunc
	pfWelsUpdateMaxBrWindowStatus PWelsUpdateMaxBrCheckWindowStatusFunc
	pfWelsRcPostFrameSkipping     PWelsRCPostFrameSkippingFunc
}
