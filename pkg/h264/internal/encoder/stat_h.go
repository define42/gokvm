// Port of codec/encoder/core/inc/stat.h.
//
// These statistics structures are only used with STAT_OUTPUT (not ported);
// they are kept for completeness.

package encoder

// SStatQuality is the quality statistic.
type SStatQuality struct {
	rYPsnr [5]float32
	rUPsnr [5]float32
	rVPsnr [5]float32
}

// SComplexityStat is the complexity statistic (only FME_TEST fields in C).
type SComplexityStat struct {
}

// SStatSliceInfo is the per slice statistic.
type SStatSliceInfo struct {
	/* per slice info */
	iSliceCount [5]int32
	iSliceSize  [5]int32
	iMbCount    [5][18]int32
}

// SStatData is the overall statistic.
type SStatData struct {
	// Quality
	sQualityStat SStatQuality
	// Complexity
	sComplexityStat SComplexityStat
	// SSlice information output
	sSliceData SStatSliceInfo
}
