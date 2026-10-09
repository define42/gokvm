// Port of codec/api/wels/codec_api.h.

package api

// ISVCEncoder is the encoder definition (C++ abstract class ISVCEncoder).
//
// Return values are CM_RETURN codes as int32: 0 - success; otherwise - failed.
type ISVCEncoder interface {
	// Initialize the encoder with the basic encoder parameter.
	Initialize(pParam *SEncParamBase) int32

	// InitializeExt initializes the encoder by using extension parameters.
	InitializeExt(pParam *SEncParamExt) int32

	// GetDefaultParams gets the default extension parameters.
	// If you want to change some parameters of encoder, firstly you need to
	// get the default encoding parameters, after that you can change part of
	// parameters you want to.
	GetDefaultParams(pParam *SEncParamExt) int32

	// Uninitialize the encoder.
	Uninitialize() int32

	// EncodeFrame encodes one frame. kpSrcPic is the source picture,
	// pBsInfo receives the output bit stream.
	EncodeFrame(kpSrcPic *SSourcePicture, pBsInfo *SFrameBSInfo) int32

	// EncodeParameterSets encodes the parameters into the output bit stream.
	EncodeParameterSets(pBsInfo *SFrameBSInfo) int32

	// ForceIntraFrame forces the encoder to encode the frame as IDR if bIDR is
	// true (false: return 1 and nothing to do). The C++ default argument
	// iLayerId = -1 cannot be expressed in Go: callers must pass -1 for the
	// default (all layers).
	ForceIntraFrame(bIDR bool, iLayerId int32) int32

	// SetOption sets an option for the encoder; see ENCODER_OPTION and the
	// "Option values" table in the package documentation.
	SetOption(eOptionId ENCODER_OPTION, pOption any) int32

	// GetOption gets an option of the encoder; see ENCODER_OPTION.
	GetOption(eOptionId ENCODER_OPTION, pOption any) int32
}

// ISVCDecoder is the decoder definition (C++ abstract class ISVCDecoder).
//
// C `long` return values are int32: 0 - success; otherwise - failed.
// `const unsigned char* pSrc, const int iSrcLen` is ported as
// `pSrc []byte, iSrcLen int32` (pSrc may be nil with iSrcLen 0 where the C
// API accepts NULL); `unsigned char** ppDst` is ported as `ppDst *[3][]byte`,
// which receives one slice per output plane starting at the plane origin.
type ISVCDecoder interface {
	// Initialize the decoder.
	Initialize(pParam *SDecodingParam) int32

	// Uninitialize the decoder.
	Uninitialize() int32

	// DecodeFrame decodes one frame.
	// pStride (int* in C, 2 entries) receives the luma and chroma strides;
	// iWidth / iHeight (int& in C) receive the output size.
	DecodeFrame(pSrc []byte, iSrcLen int32, ppDst *[3][]byte, pStride *[2]int32,
		iWidth *int32, iHeight *int32) DECODING_STATE

	// DecodeFrameNoDelay is the slice level decode (4 parameters input).
	// Whatever the function return value is, the output data of I420 format
	// will only be available when pDstInfo.IBufferStatus == 1. This function
	// parses and reconstructs the input frame immediately if it is complete.
	// It is recommended as the main decoding function for H.264/AVC input.
	DecodeFrameNoDelay(pSrc []byte, iSrcLen int32, ppDst *[3][]byte, pDstInfo *SBufferInfo) DECODING_STATE

	// DecodeFrame2 is the slice level decode (4 parameters input).
	// Whatever the function return value is, the output data of I420 format
	// will only be available when pDstInfo.IBufferStatus == 1 (e.g., in
	// multi-slice cases, only when the whole picture is completely
	// reconstructed, this variable would be set equal to 1).
	DecodeFrame2(pSrc []byte, iSrcLen int32, ppDst *[3][]byte, pDstInfo *SBufferInfo) DECODING_STATE

	// FlushFrame gets a decoded ready frame remaining in buffers after the
	// last frame has been decoded. Use GetOption with
	// DECODER_OPTION_NUM_OF_FRAMES_REMAINING_IN_BUFFER to get the number of
	// frames remaining in buffers. Only applicable for profile_idc != 66.
	FlushFrame(ppDst *[3][]byte, pDstInfo *SBufferInfo) DECODING_STATE

	// DecodeParser parses the input bitstream only, and rewrites possible SVC
	// syntax to AVC syntax.
	DecodeParser(pSrc []byte, iSrcLen int32, pDstInfo *SParserBsInfo) DECODING_STATE

	// DecodeFrameEx does not work for now (future use to support non-I420
	// color format output). pDst (unsigned char*) -> []byte; the int&
	// parameters become *int32.
	DecodeFrameEx(pSrc []byte, iSrcLen int32, pDst []byte, iDstStride int32,
		iDstLen *int32, iWidth *int32, iHeight *int32, iColorFormat *int32) DECODING_STATE

	// SetOption sets an option for the decoder; see DECODER_OPTION and the
	// "Option values" table in the package documentation.
	SetOption(eOptionId DECODER_OPTION, pOption any) int32

	// GetOption gets an option of the decoder; see DECODER_OPTION.
	GetOption(eOptionId DECODER_OPTION, pOption any) int32
}

// WelsTraceCallback receives log messages
// (typedef void (*WelsTraceCallback) (void* ctx, int level, const char* string)).
// ctx is the value set with *_OPTION_TRACE_CALLBACK_CONTEXT.
type WelsTraceCallback func(ctx any, level int32, msg string)
