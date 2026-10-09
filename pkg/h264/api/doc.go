// Package api is a Go port of the OpenH264 public API headers
// (codec/api/wels/codec_api.h, codec_app_def.h, codec_def.h and codec_ver.h).
//
// Every identifier is the C identifier with its first letter upper-cased
// (struct field iPicWidth -> IPicWidth, enum constant dsErrorFree ->
// DsErrorFree, g_stCodecVersion -> G_stCodecVersion); identifiers that are
// already upper-case are unchanged (EVideoFormatType, DECODER_OPTION_END_OF_STREAM).
//
// Enum types are named integer types with typed constants (int32 for plain
// enums). Anonymous C enums and #defines become untyped constants.
//
// Pointer fields are translated as follows (each struct field comment repeats
// the choice):
//   - byte buffers (unsigned char*) -> []uint8 slices that start at the
//     position the C pointer points to (for picture planes: the plane origin,
//     with the row stride given by the matching iStride entry);
//   - int / unsigned int arrays -> []int32 / []uint32;
//   - char* strings -> string;
//   - void* option values -> any (callers pass a pointer of the type the C API
//     expects, e.g. *int32, *bool, *SDecoderStatistics, *SEncParamExt).
//
// The C++ abstract classes ISVCEncoder and ISVCDecoder become the Go
// interfaces of the same names. The C-only vtable structs (ISVCEncoderVtbl,
// ISVCDecoderVtbl) are not ported. The factory functions (WelsCreateDecoder,
// WelsCreateSVCEncoder, ...) live in the root package.
//
// # Option values
//
// Option values (void* pOption in C) are passed as `any` holding a pointer of
// the type the C implementation casts pOption to. nil corresponds to NULL.
//
// Encoder (ISVCEncoder.SetOption / GetOption):
//
//	ENCODER_OPTION_DATAFORMAT              *int32 (EVideoFormatType value)
//	ENCODER_OPTION_IDR_INTERVAL            *int32
//	ENCODER_OPTION_SVC_ENCODE_PARAM_BASE   *SEncParamBase
//	ENCODER_OPTION_SVC_ENCODE_PARAM_EXT    *SEncParamExt
//	ENCODER_OPTION_FRAME_RATE              *float32
//	ENCODER_OPTION_BITRATE                 *SBitrateInfo
//	ENCODER_OPTION_MAX_BITRATE             *SBitrateInfo
//	ENCODER_OPTION_RC_MODE                 *int32 (RC_MODES value)
//	ENCODER_OPTION_RC_FRAME_SKIP           *bool
//	ENCODER_PADDING_PADDING                *int32
//	ENCODER_OPTION_PROFILE                 *SProfileInfo
//	ENCODER_OPTION_LEVEL                   *SLevelInfo
//	ENCODER_OPTION_NUMBER_REF              *int32
//	ENCODER_OPTION_DELIVERY_STATUS         *SDeliveryStatus
//	ENCODER_LTR_RECOVERY_REQUEST           *SLTRRecoverRequest
//	ENCODER_LTR_MARKING_FEEDBACK           *SLTRMarkingFeedback
//	ENCODER_LTR_MARKING_PERIOD             *uint32
//	ENCODER_OPTION_LTR                     *SLTRConfig
//	ENCODER_OPTION_COMPLEXITY              *int32 (ECOMPLEXITY_MODE value)
//	ENCODER_OPTION_ENABLE_SSEI             *bool
//	ENCODER_OPTION_ENABLE_PREFIX_NAL_ADDING *bool
//	ENCODER_OPTION_SPS_PPS_ID_STRATEGY     *int32 (EParameterSetStrategy value)
//	ENCODER_OPTION_CURRENT_PATH            *string (char* in C)
//	ENCODER_OPTION_DUMP_FILE               *SDumpLayer
//	ENCODER_OPTION_TRACE_LEVEL             *uint32 (implementations should also accept *int32)
//	ENCODER_OPTION_TRACE_CALLBACK          *WelsTraceCallback
//	ENCODER_OPTION_TRACE_CALLBACK_CONTEXT  *any (C: void** -> the context value is *pOption)
//	ENCODER_OPTION_GET_STATISTICS          *SEncoderStatistics
//	ENCODER_OPTION_STATISTICS_LOG_INTERVAL *int32
//	ENCODER_OPTION_IS_LOSSLESS_LINK        *bool
//	ENCODER_OPTION_BITS_VARY_PERCENTAGE    *int32
//
// Decoder (ISVCDecoder.SetOption / GetOption):
//
//	DECODER_OPTION_END_OF_STREAM           *int32 (boolean as int)
//	DECODER_OPTION_ERROR_CON_IDC           *int32 (ERROR_CON_IDC value)
//	DECODER_OPTION_NUM_OF_THREADS          *int32
//	DECODER_OPTION_TRACE_LEVEL             *uint32 (implementations should also accept *int32)
//	DECODER_OPTION_TRACE_CALLBACK          *WelsTraceCallback
//	DECODER_OPTION_TRACE_CALLBACK_CONTEXT  *any (C: void** -> the context value is *pOption)
//	DECODER_OPTION_STATISTICS_LOG_INTERVAL *uint32
//	DECODER_OPTION_GET_STATISTICS          *SDecoderStatistics
//	DECODER_OPTION_GET_SAR_INFO            *SVuiSarInfo
//	DECODER_OPTION_VCL_NAL, _TEMPORAL_ID, _FRAME_NUM, _IDR_PIC_ID,
//	_LTR_MARKING_FLAG, _LTR_MARKED_FRAME_NUM, _IS_REF_PIC, _PROFILE, _LEVEL,
//	_NUM_OF_FRAMES_REMAINING_IN_BUFFER    *int32 (GetOption only)
package api
