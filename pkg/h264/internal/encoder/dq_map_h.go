// Port of codec/encoder/core/inc/dq_map.h.

package encoder

// SDqIdc is the DQ layer idc map entry for svc encoding.
type SDqIdc struct {
	iPpsId      uint16 // pPps id
	iSpsId      uint8  // pSps id
	uiSpatialId int8   // spatial id
}
