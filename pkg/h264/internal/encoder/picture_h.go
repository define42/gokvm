// Port of codec/encoder/core/inc/picture.h.

package encoder

const LIST_SIZE = 0x10000 // (256*256)

// SScreenBlockFeatureStorage should be stored with RefPic, one for each frame.
type SScreenBlockFeatureStorage struct {
	// Input
	pFeatureOfBlockPointer []uint16 // C uint16_t*: whole feature-of-block buffer (owned by SFeatureSearchPreparation)
	iIs16x16               int32    // Feature block size
	uiFeatureStrategyIndex uint8    // index of hash strategy

	// Modify
	pTimesOfFeatureValue []uint32 // C uint32_t*: times of every value in Feature (kiListSize entries)
	// C uint16_t**: pLocationOfFeature[i] saves all the location(x,y) whose Feature = i;
	// each entry is a sub-slice of pLocationPointer.
	pLocationOfFeature         [][]uint16
	pLocationPointer           []uint16 // C uint16_t*: buffer of position array
	iActualListSize            int32    // actual list size
	uiSadCostThreshold         [BLOCK_SIZE_ALL]uint32
	bRefBlockFeatureCalculated bool // flag of whether pre-process is done
	// C uint16_t**: pFeatureValuePointerList[WELS_MAX (LIST_SIZE_SUM_16x16, LIST_SIZE_MSE_16x16)];
	// each entry is a sub-slice of pLocationPointer (moved forward while filling).
	pFeatureValuePointerList [][]uint16
}

// SPicture is the reconstructed picture definition. It is used to express
// reference picture, also consequent reconstruction picture for output.
type SPicture struct {
	/************************************payload pData*********************************/
	pBuffer []uint8 // C uint8_t*: the whole allocation (all three planes incl. padding)
	// C uint8_t* pData[3]: each entry holds the SAME whole allocation as pBuffer;
	// the plane origin lies at iDataOff[i] (negative offsets reach the padding).
	pData     [3][]uint8
	iDataOff  [3]int   // Go-only: offset of plane i's origin inside pData[i]
	iLineSize [3]int32 // iLineSize of picture planes respectively

	// picture information
	/*from pSps*/
	iWidthInPixel  int32   // picture width in pixel
	iHeightInPixel int32   // picture height in pixel
	iPictureType   int32   // got from sSliceHeader(): eSliceType
	iFramePoc      int32   // frame POC
	fFrameRate     float32 // MOVE
	iFrameNum      int32   // frame number      //for pRef pic management

	uiRefMbType []uint32    // C uint32_t*: per MB, iMbWidth*iMbHeight entries
	pRefMbQp    []uint8     // C uint8_t*: per MB
	pMbSkipSad  []int32     // C int32_t*: per MB
	sMvList     []SMVUnitXY // C SMVUnitXY*: per MB

	/*******************************sef_definition for misc use****************************/
	iMarkFrameNum      int32
	iLongTermPicNum    int32
	bUsedAsRef         bool // for pRef pic management
	bIsLongRef         bool // long term reference frame flag  //for pRef pic management
	bIsSceneLTR        bool // long term reference & large scene change
	uiRecieveConfirmed uint8
	uiTemporalId       uint8
	uiSpatialId        uint8
	iFrameAverageQp    int32

	/*******************************for screen reference frames****************************/
	pScreenBlockFeatureStorage *SScreenBlockFeatureStorage
}

// SetUnref sets the picture as unreferenced.
func (p *SPicture) SetUnref() {
	p.iFramePoc = -1
	p.iFrameNum = -1
	// uiTemporalId = uiSpatialId = iLongTermPicNum = -1 (chained assignment, truncated to uint8 for the ids)
	p.iLongTermPicNum = -1
	p.uiSpatialId = uint8(p.iLongTermPicNum & 0xff)
	p.uiTemporalId = p.uiSpatialId
	p.bIsLongRef = false
	p.uiRecieveConfirmed = uint8(RECIEVE_FAILED)
	p.iMarkFrameNum = -1
	p.bUsedAsRef = false
	if nil != p.pScreenBlockFeatureStorage {
		p.pScreenBlockFeatureStorage.bRefBlockFeatureCalculated = false
	}
}
