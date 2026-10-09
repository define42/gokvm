// Port of codec/api/wels/codec_def.h.

package api

import "math"

// Integer is the set of integer types accepted by the generic macro ports.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// EVideoFormatType enumerates the type of video format.
//
// The C enum needs an unsigned underlying type because of videoFormatVFlip
// (0x80000000). The Go type is int32 so that it mixes with the int32
// iColorFormat/iFormat fields; VideoFormatVFlip therefore has the value
// math.MinInt32, which has the same bit pattern (0x80000000). Use it only in
// bitwise operations, as the C code does.
type EVideoFormatType int32

const (
	VideoFormatRGB    EVideoFormatType = 1 ///< rgb color formats
	VideoFormatRGBA   EVideoFormatType = 2
	VideoFormatRGB555 EVideoFormatType = 3
	VideoFormatRGB565 EVideoFormatType = 4
	VideoFormatBGR    EVideoFormatType = 5
	VideoFormatBGRA   EVideoFormatType = 6
	VideoFormatABGR   EVideoFormatType = 7
	VideoFormatARGB   EVideoFormatType = 8

	VideoFormatYUY2     EVideoFormatType = 20 ///< yuv color formats
	VideoFormatYVYU     EVideoFormatType = 21
	VideoFormatUYVY     EVideoFormatType = 22
	VideoFormatI420     EVideoFormatType = 23 ///< the same as IYUV
	VideoFormatYV12     EVideoFormatType = 24
	VideoFormatInternal EVideoFormatType = 25 ///< only used in SVC decoder testbed

	VideoFormatNV12 EVideoFormatType = 26 ///< new format for output by DXVA decoding

	// VideoFormatVFlip is 0x80000000 in C; stored as math.MinInt32 (same bits).
	VideoFormatVFlip EVideoFormatType = math.MinInt32
)

// EVideoFrameType enumerates video frame types.
type EVideoFrameType int32

const (
	VideoFrameTypeInvalid EVideoFrameType = iota ///< encoder not ready or parameters are invalidate
	VideoFrameTypeIDR                            ///< IDR frame in H.264
	VideoFrameTypeI                              ///< I frame type
	VideoFrameTypeP                              ///< P frame type
	VideoFrameTypeSkip                           ///< skip the frame based encoder kernel
	VideoFrameTypeIPMixed                        ///< a frame where I and P slices are mixing, not supported yet
)

// CM_RETURN enumerates return types.
type CM_RETURN int32

const (
	CmResultSuccess CM_RETURN = iota ///< successful
	CmInitParaError                  ///< parameters are invalid
	CmUnknownReason
	CmMallocMemeError ///< malloc a memory error
	CmInitExpected    ///< initial action is expected
	CmUnsupportedData
)

// ENalUnitType enumerates the nal unit type.
type ENalUnitType int32

const (
	NAL_UNKNOWN   ENalUnitType = 0
	NAL_SLICE     ENalUnitType = 1
	NAL_SLICE_DPA ENalUnitType = 2
	NAL_SLICE_DPB ENalUnitType = 3
	NAL_SLICE_DPC ENalUnitType = 4
	NAL_SLICE_IDR ENalUnitType = 5 ///< ref_idc != 0
	NAL_SEI       ENalUnitType = 6 ///< ref_idc == 0
	NAL_SPS       ENalUnitType = 7
	NAL_PPS       ENalUnitType = 8
	///< ref_idc == 0 for 6,9,10,11,12
)

// ENalPriority is NRI: eNalRefIdc.
type ENalPriority int32

const (
	NAL_PRIORITY_DISPOSABLE ENalPriority = 0
	NAL_PRIORITY_LOW        ENalPriority = 1
	NAL_PRIORITY_HIGH       ENalPriority = 2
	NAL_PRIORITY_HIGHEST    ENalPriority = 3
)

// IS_PARAMETER_SET_NAL ports the macro of the same name.
func IS_PARAMETER_SET_NAL[T, U Integer](eNalRefIdc T, eNalType U) bool {
	return (int64(eNalRefIdc) == int64(NAL_PRIORITY_HIGHEST)) &&
		(int64(eNalType) == int64(NAL_SPS|NAL_PPS) || int64(eNalType) == int64(NAL_SPS))
}

// IS_IDR_NAL ports the macro of the same name.
func IS_IDR_NAL[T, U Integer](eNalRefIdc T, eNalType U) bool {
	return (int64(eNalRefIdc) == int64(NAL_PRIORITY_HIGHEST)) && (int64(eNalType) == int64(NAL_SLICE_IDR))
}

const (
	FRAME_NUM_PARAM_SET = -1
	FRAME_NUM_IDR       = 0
)

// eDeblockingIdc (anonymous enum).
const (
	DEBLOCKING_IDC_0 = 0
	DEBLOCKING_IDC_1 = 1
	DEBLOCKING_IDC_2 = 2
)

const (
	DEBLOCKING_OFFSET       = 6
	DEBLOCKING_OFFSET_MINUS = -6
)

// ERR_TOOL is the Error Tools type (unsigned short).
type ERR_TOOL = uint16

// Error tools (anonymous enum).
const (
	ET_NONE     = 0x00 ///< NONE Error Tools
	ET_IP_SCALE = 0x01 ///< IP Scalable
	ET_FMO      = 0x02 ///< Flexible Macroblock Ordering
	ET_IR_R1    = 0x04 ///< Intra Refresh in predifined 2% MB
	ET_IR_R2    = 0x08 ///< Intra Refresh in predifined 5% MB
	ET_IR_R3    = 0x10 ///< Intra Refresh in predifined 10% MB
	ET_FEC_HALF = 0x20 ///< Forward Error Correction in 50% redundency mode
	ET_FEC_FULL = 0x40 ///< Forward Error Correction in 100% redundency mode
	ET_RFS      = 0x80 ///< Reference Frame Selection
)

// SliceInfo is the information of coded Slice(=NAL)(s) (struct SliceInformation).
type SliceInfo struct {
	PBufferOfSlices      []uint8  ///< base buffer of coded slice(s) (unsigned char* -> slice starting at the pointed-to byte)
	ICodedSliceCount     int32    ///< number of coded slices
	PLengthOfSlices      []uint32 ///< array of slices length accordingly by number of slice (unsigned int* -> slice)
	IFecType             int32    ///< FEC type[0, 50%FEC, 100%FEC]
	UiSliceIdx           uint8    ///< index of slice in frame [FMO: 0,..,uiSliceCount-1; No FMO: 0]
	UiSliceCount         uint8    ///< count number of slice in frame [FMO: 2-8; No FMO: 1]
	IFrameIndex          int8     ///< index of frame[-1, .., idr_interval-1] (char -> int8)
	UiNalRefIdc          uint8    ///< NRI, priority level of slice(NAL)
	UiNalType            uint8    ///< NAL type
	UiContainingFinalNal uint8    ///< whether final NAL is involved in buffer of coded slices, flag used in Pause feature in T27
}

// PSliceInfo is the pointer alias of SliceInfo.
type PSliceInfo = *SliceInfo

// SRateThresholds holds thresholds of the initial, maximal and minimal rate.
type SRateThresholds struct {
	IWidth                 int32 ///< frame width
	IHeight                int32 ///< frame height
	IThresholdOfInitRate   int32 ///< threshold of initial rate
	IThresholdOfMaxRate    int32 ///< threshold of maximal rate
	IThresholdOfMinRate    int32 ///< threshold of minimal rate
	IMinThresholdFrameRate int32 ///< min frame rate min
	ISkipFrameRate         int32 ///< skip to frame rate min
	ISkipFrameStep         int32 ///< how many frames to skip
}

// PRateThresholds is the pointer alias of SRateThresholds.
type PRateThresholds = *SRateThresholds

// SSysMEMBuffer is the structure for decoder memory.
type SSysMEMBuffer struct {
	IWidth  int32    ///< width of decoded pic for display
	IHeight int32    ///< height of decoded pic for display
	IFormat int32    ///< type is "EVideoFormatType"
	IStride [2]int32 ///< stride of 2 component
}

// SBufferInfo is the decoder output buffer info.
type SBufferInfo struct {
	IBufferStatus     int32  ///< 0: one frame data is not ready; 1: one frame data is ready
	UiInBsTimeStamp   uint64 ///< input BS timestamp
	UiOutYuvTimeStamp uint64 ///< output YUV timestamp, when bufferstatus is 1
	// UsrData is the C union UsrData (it has a single member).
	UsrData struct {
		SSystemBuffer SSysMEMBuffer ///<  memory info for one picture
	} ///<  output buffer info
	// PDst points to picture YUV data. unsigned char* pDst[3] -> one slice
	// per plane, each starting at that plane's origin (top-left visible
	// sample); rows are UsrData.SSystemBuffer.IStride[0] (luma) /
	// IStride[1] (chroma) bytes apart.
	PDst [3][]uint8
}

// KiKeyNumMultiple: in a GOP, multiple of the key frame number, derived from
// the number of layers (index or array below). (static const char -> int8)
var KiKeyNumMultiple = [...]int8{
	1, 1, 2, 4, 8, 16,
}
