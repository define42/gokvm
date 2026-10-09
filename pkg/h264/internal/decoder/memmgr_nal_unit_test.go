package decoder

import "testing"

func TestMemNalListExpand(t *testing.T) {
	var pAu *SAccessUnit
	if MemInitNalList(&pAu, 0) != ERR_INFO_INVALID_PARAM {
		t.Fatal("size 0 accepted")
	}
	if MemInitNalList(&pAu, 2) != ERR_NONE || pAu.uiCountUnitsNum != 2 || len(pAu.pNalUnitsList) != 2 {
		t.Fatal("MemInitNalList")
	}
	n0 := MemGetNextNal(&pAu)
	n0.sNalData.sVclNal.iNalLength = 7
	_ = MemGetNextNal(&pAu)
	n2 := MemGetNextNal(&pAu) // expands the list
	if n2 == nil || pAu.uiCountUnitsNum != 2+MAX_NAL_UNIT_NUM_IN_AU/2 || pAu.uiAvailUnitsNum != 3 {
		t.Fatalf("expand failed: %+v", pAu)
	}
	if pAu.pNalUnitsList[0].sNalData.sVclNal.iNalLength != 7 {
		t.Fatal("NAL content not copied on expansion")
	}
	MemFreeNalList(&pAu)
	if pAu != nil {
		t.Fatal("MemFreeNalList")
	}
}
