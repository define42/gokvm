// Port of codec/decoder/core/inc/wels_const.h.
// (wels_const_common.h lives in package common.)

package decoder

/* Some list size */
const (
	MB_SUB_PARTITION_SIZE    = 4   // Sub partition size in a 8x8 sub-block
	NAL_UNIT_HEADER_EXT_SIZE = 3   // Size of NAL unit header for extension in byte
	MAX_PPS_COUNT            = 256 // Count number of PPS

	MAX_REF_PIC_COUNT   = 16                    // MAX Short + Long reference pictures
	MIN_REF_PIC_COUNT   = 1                     // minimal count number of reference pictures, 1 short + 2 key reference based?
	MAX_SHORT_REF_COUNT = 16                    // maximal count number of short reference pictures
	MAX_LONG_REF_COUNT  = 16                    // maximal count number of long reference pictures
	MAX_DPB_COUNT       = MAX_REF_PIC_COUNT + 1 // 1 additional position for re-order and other process

	MAX_MMCO_COUNT = 66

	MAX_SLICEGROUP_IDS = 8 // Count number of Slice Groups

	MAX_LAYER_NUM = 8

	LAYER_NUM_EXCHANGEABLE = 1

	MAX_NAL_UNIT_NUM_IN_AU   = 32      // predefined maximal number of NAL Units in an access unit
	MIN_ACCESS_UNIT_CAPACITY = 1048576 // Min AU capacity in bytes: (1<<20) = 1024 KB predefined
	MAX_BUFFERED_NUM         = 3       //mamixum stored number of AU|packet to prevent overwrite
	MAX_ACCESS_UNIT_CAPACITY = 7077888 //Maximum AU size in bytes for level 5.2 for single frame
	MAX_MACROBLOCK_CAPACITY  = 5000    //Maximal legal MB capacity, 15000 bits is enough
)
