// Port of codec/decoder/core/inc/nal_prefix.h.

package decoder

/* Prefix NAL Unix syntax, refer to Page 392 in JVT X201wcm */
type SPrefixNalUnit struct {
	sRefPicBaseMarking              SRefBasePicMarking
	bStoreRefBasePicFlag            bool
	bPrefixNalUnitAdditionalExtFlag bool
	bPrefixNalUnitExtFlag           bool
	bPrefixNalCorrectFlag           bool
}

type PPrefixNalUnit = *SPrefixNalUnit
