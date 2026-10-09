package common

// Port of codec/common/inc/wels_const_common.h.

// Miscellaneous sizing infos
const (
	MAX_FNAME_LEN      = 256            // maximal length of file name in char size
	WELS_LOG_BUF_SIZE  = 4096           //
	MAX_TRACE_LOG_SIZE = 50 * (1 << 20) // max trace log size: 50 MB
	MB_WIDTH_LUMA      = 16             // MB width in pixels for I420 luma
	MB_WIDTH_CHROMA    = MB_WIDTH_LUMA >> 1
	MB_HEIGHT_LUMA     = 16
	MB_HEIGHT_CHROMA   = MB_HEIGHT_LUMA >> 1
	MB_COEFF_LIST_SIZE = 256 + ((MB_WIDTH_CHROMA * MB_HEIGHT_CHROMA) << 1)
	MB_PARTITION_SIZE  = 4  // Macroblock partition size in 8x8 sub-blocks
	MB_BLOCK4x4_NUM    = 16 //
	MB_BLOCK8x8_NUM    = 4  //
	MAX_SPS_COUNT      = 32 // Count number of SPS
	BASE_QUALITY_ID    = 0  //
)
