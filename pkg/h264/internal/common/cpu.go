package common

// Port of codec/common/inc/cpu.h and codec/common/src/cpu.cpp.
// Architecture-specific feature detection is implemented in cpu_*.go.

// WelsEmms is a no-op (it clears the MMX state on x86 in C).
func WelsEmms() {}

// WelsCPURestore is a no-op.
func WelsCPURestore(kuiCPU uint32) {}
