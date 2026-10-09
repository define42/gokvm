package common

// Port of codec/common/inc/cpu.h and codec/common/src/cpu.cpp.
//
// No CPU specific code is ported, so feature detection always reports no
// SIMD capability.

// WelsCPUFeatureDetect returns the CPU feature flags: always 0.
// pNumberOfLogicProcessors is left untouched (as in the generic C branch).
func WelsCPUFeatureDetect(pNumberOfLogicProcessors *int32) uint32 {
	return 0
}

// WelsEmms is a no-op (it clears the MMX state on x86 in C).
func WelsEmms() {}

// WelsCPURestore is a no-op.
func WelsCPURestore(kuiCPU uint32) {}
