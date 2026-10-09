package api

import "testing"

func TestConstants(t *testing.T) {
	if SAVED_NALUNIT_NUM_TMP != 21 {
		t.Errorf("SAVED_NALUNIT_NUM_TMP = %d, want 21", SAVED_NALUNIT_NUM_TMP)
	}
	if MAX_SLICES_NUM_TMP != 35 {
		t.Errorf("MAX_SLICES_NUM_TMP = %d, want 35", MAX_SLICES_NUM_TMP)
	}
	vflip := VideoFormatVFlip
	if uint32(vflip) != 0x80000000 {
		t.Errorf("VideoFormatVFlip bits = %#x", uint32(vflip))
	}
	if int32(VideoFormatI420)&^int32(VideoFormatVFlip) != 23 {
		t.Error("VFlip mask")
	}
	if DECODER_OPTION_END_OF_STREAM != 1 || DECODER_OPTION_NUM_OF_THREADS != 19 {
		t.Error("DECODER_OPTION values")
	}
	if ENCODER_OPTION_DATAFORMAT != 0 || ENCODER_OPTION_BITS_VARY_PERCENTAGE != 31 {
		t.Error("ENCODER_OPTION values")
	}
	if ENCODER_OPTION_TRACE_CALLBACK != 26 || ENCODER_OPTION_GET_STATISTICS != 28 {
		t.Error("ENCODER_OPTION middle values")
	}
	if WELS_LOG_DEFAULT != WELS_LOG_WARNING || WELS_LOG_DETAIL != 16 {
		t.Error("WELS_LOG values")
	}
	if VIDEO_BITSTREAM_DEFAULT != VIDEO_BITSTREAM_SVC || RC_OFF_MODE != -1 {
		t.Error("enum values")
	}
	if VideoFrameTypeIPMixed != 5 || CmUnsupportedData != 5 || ERROR_CON_SLICE_MV_COPY_CROSS_IDR_FREEZE_RES_CHANGE != 7 {
		t.Error("iota enums")
	}
	if !IS_PARAMETER_SET_NAL(NAL_PRIORITY_HIGHEST, NAL_SPS) || !IS_PARAMETER_SET_NAL(uint8(3), uint8(15)) ||
		IS_PARAMETER_SET_NAL(NAL_PRIORITY_HIGHEST, NAL_PPS) || !IS_IDR_NAL(int32(3), NAL_SLICE_IDR) {
		t.Error("NAL macros")
	}
	v := G_stCodecVersion
	if v.UMajor != OPENH264_MAJOR || v.UMinor != OPENH264_MINOR || v.URevision != OPENH264_REVISION || v.UReserved != OPENH264_RESERVED {
		t.Errorf("version %+v", v)
	}
	if len(KiKeyNumMultiple) != 6 || KiKeyNumMultiple[5] != 16 {
		t.Error("KiKeyNumMultiple")
	}
}
