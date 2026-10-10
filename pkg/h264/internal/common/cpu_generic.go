//go:build !amd64

package common

import "runtime"

// WelsCPUFeatureDetect reports no SIMD features on architectures for which
// optimized kernels have not been ported yet.
func WelsCPUFeatureDetect(pNumberOfLogicProcessors *int32) uint32 {
	if pNumberOfLogicProcessors != nil {
		*pNumberOfLogicProcessors = int32(runtime.NumCPU())
	}
	return 0
}
