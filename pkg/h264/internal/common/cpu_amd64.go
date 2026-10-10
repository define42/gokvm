//go:build amd64

package common

import (
	"runtime"

	"golang.org/x/sys/cpu"
)

// WelsCPUFeatureDetect returns the OpenH264 feature flags supported by the
// current amd64 CPU and operating system. x/sys/cpu only reports AVX-family
// features when the OS also supports saving their extended register state.
func WelsCPUFeatureDetect(pNumberOfLogicProcessors *int32) uint32 {
	if pNumberOfLogicProcessors != nil {
		*pNumberOfLogicProcessors = int32(runtime.NumCPU())
	}

	var flags uint32
	if cpu.X86.HasSSE2 {
		// SSE2 is part of the amd64 baseline. SSE also implies the MMX
		// extensions in OpenH264's feature model.
		flags |= WELS_CPU_MMX | WELS_CPU_MMXEXT | WELS_CPU_SSE | WELS_CPU_SSE2 |
			WELS_CPU_FPU | WELS_CPU_CMOV
	}
	if cpu.X86.HasSSE3 {
		flags |= WELS_CPU_SSE3
	}
	if cpu.X86.HasSSSE3 {
		flags |= WELS_CPU_SSSE3
	}
	if cpu.X86.HasSSE41 {
		flags |= WELS_CPU_SSE41
	}
	if cpu.X86.HasSSE42 {
		flags |= WELS_CPU_SSE42
	}
	if cpu.X86.HasAVX {
		flags |= WELS_CPU_AVX
	}
	if cpu.X86.HasFMA && cpu.X86.HasAVX {
		flags |= WELS_CPU_FMA
	}
	if cpu.X86.HasAES {
		flags |= WELS_CPU_AES
	}
	if cpu.X86.HasAVX2 {
		flags |= WELS_CPU_AVX2
	}
	if cpu.X86.HasAVX512F {
		flags |= WELS_CPU_AVX512F
	}
	if cpu.X86.HasAVX512CD {
		flags |= WELS_CPU_AVX512CD
	}
	if cpu.X86.HasAVX512DQ {
		flags |= WELS_CPU_AVX512DQ
	}
	if cpu.X86.HasAVX512BW {
		flags |= WELS_CPU_AVX512BW
	}
	if cpu.X86.HasAVX512VL {
		flags |= WELS_CPU_AVX512VL
	}

	return flags
}
