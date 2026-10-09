package decoder

import "testing"

// Port of TEST (DecoderFmoSecurityTest, RejectsOversizedRunLengthBeforeIndexWrap)
// from test/decoder/DecUT_ParseSyntax.cpp.
func TestFmoRejectsOversizedRunLengthBeforeIndexWrap(t *testing.T) {
	var sFmo SFmo
	var sPps SPps

	sPps.uiNumSliceGroups = 2
	sPps.uiSliceGroupMapType = 0
	sPps.uiRunLength[0] = 0xffffffff
	sPps.uiRunLength[1] = 1

	iRet := InitFmo(&sFmo, &sPps, 120, 68)
	if iRet == ERR_NONE {
		t.Fatal("InitFmo accepted an oversized run length")
	}
	if sFmo.pMbAllocMap != nil {
		t.Fatal("allocation map not released on the rejection path")
	}
}

func TestFmoType0Type1AndNextMb(t *testing.T) {
	var sFmo SFmo
	var sPps SPps
	sFmo.iSliceGroupType = -1
	sPps.uiNumSliceGroups = 2
	sPps.uiSliceGroupMapType = 0
	sPps.uiRunLength[0] = 3
	sPps.uiRunLength[1] = 2
	if r := InitFmo(&sFmo, &sPps, 4, 3); r != ERR_NONE {
		t.Fatalf("InitFmo = %d", r)
	}
	want0 := []uint8{0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0}
	for i, v := range want0 {
		if sFmo.pMbAllocMap[i] != v {
			t.Fatalf("type0 map[%d] = %d want %d", i, sFmo.pMbAllocMap[i], v)
		}
	}
	if FmoNextMb(&sFmo, 2) != 5 || FmoNextMb(&sFmo, 4) != 8 || FmoNextMb(&sFmo, 9) != -1 || FmoNextMb(&sFmo, 12) != -1 {
		t.Fatal("FmoNextMb")
	}
	if FmoMbToSliceGroup(&sFmo, 3) != 1 || FmoMbToSliceGroup(&sFmo, -1) != -1 {
		t.Fatal("FmoMbToSliceGroup")
	}

	// dispersed (type 1)
	var sFmo1 SFmo
	sFmo1.iSliceGroupType = -1
	var iActive int32
	sPps.uiSliceGroupMapType = 1
	sPps.uiNumSliceGroups = 3
	sps := SSps{iMbWidth: 4, iMbHeight: 2}
	if r := FmoParamUpdate(&sFmo1, &sps, &sPps, &iActive); r != ERR_NONE {
		t.Fatalf("FmoParamUpdate = %d", r)
	}
	if iActive != 1 || !sFmo1.bActiveFlag || sFmo1.iSliceGroupCount != 3 || sFmo1.iSliceGroupType != 1 {
		t.Fatalf("unexpected fmo state %+v active %d", sFmo1, iActive)
	}
	// ((i % w) + (((i / w) * n) >> 1)) % n
	want1 := []uint8{0, 1, 2, 0, 1, 2, 0, 1}
	for i, v := range want1 {
		if sFmo1.pMbAllocMap[i] != v {
			t.Fatalf("type1 map[%d] = %d want %d", i, sFmo1.pMbAllocMap[i], v)
		}
	}
	if FmoParamSetsChanged(&sFmo1, 8, 1, 3) {
		t.Fatal("FmoParamSetsChanged should be false")
	}
	list := []SFmo{sFmo1, {}}
	UninitFmoList(list, 2, 1)
	if list[0].bActiveFlag || list[0].pMbAllocMap != nil || list[0].iSliceGroupType != -1 {
		t.Fatal("UninitFmoList")
	}
}
