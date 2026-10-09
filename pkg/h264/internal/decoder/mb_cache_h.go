// Port of codec/decoder/core/inc/mb_cache.h.

package decoder

const (
	REF_NOT_AVAIL   = -2
	REF_NOT_IN_LIST = -1 //intra
)

/*
 *  MB Cache information, such one cache should be defined within a slice
 */
/*
 * Cache for Luma               Cache for Chroma(Cb, Cr)
 *
 *  TL T T T T                  TL T T
 *   L - - - -                   L - -
 *   L - - - -                   L - - TR
 *   L - - - -
 *   L - - - - TR
 *
 */

////////////////////////mapping scan index////////////////////////

// g_kuiScan4 is defined in decoder_data_tables.go.

type SWelsNeighAvail struct {
	iTopAvail      int32
	iLeftAvail     int32
	iRightTopAvail int32
	iLeftTopAvail  int32 //used for check intra_pred_mode avail or not   //1: avail; 0: unavail

	iLeftType     int32
	iTopType      int32
	iLeftTopType  int32
	iRightTopType int32

	iTopCbp  int8
	iLeftCbp int8
	iDummy   [2]int8 //for align
}

type PWelsNeighAvail = *SWelsNeighAvail
