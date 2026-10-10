//go:build amd64

package common

import (
	"runtime"
	"testing"

	"golang.org/x/sys/cpu"
)

func TestWelsCPUFeatureDetectAMD64(t *testing.T) {
	var logicalProcessors int32
	flags := WelsCPUFeatureDetect(&logicalProcessors)

	if want := int32(runtime.NumCPU()); logicalProcessors != want {
		t.Fatalf("logical processor count = %d, want %d", logicalProcessors, want)
	}

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"SSE2", flags&WELS_CPU_SSE2 != 0, cpu.X86.HasSSE2},
		{"SSE3", flags&WELS_CPU_SSE3 != 0, cpu.X86.HasSSE3},
		{"SSSE3", flags&WELS_CPU_SSSE3 != 0, cpu.X86.HasSSSE3},
		{"SSE4.1", flags&WELS_CPU_SSE41 != 0, cpu.X86.HasSSE41},
		{"SSE4.2", flags&WELS_CPU_SSE42 != 0, cpu.X86.HasSSE42},
		{"AVX", flags&WELS_CPU_AVX != 0, cpu.X86.HasAVX},
		{"FMA", flags&WELS_CPU_FMA != 0, cpu.X86.HasFMA && cpu.X86.HasAVX},
		{"AES", flags&WELS_CPU_AES != 0, cpu.X86.HasAES},
		{"AVX2", flags&WELS_CPU_AVX2 != 0, cpu.X86.HasAVX2},
		{"AVX512F", flags&WELS_CPU_AVX512F != 0, cpu.X86.HasAVX512F},
		{"AVX512CD", flags&WELS_CPU_AVX512CD != 0, cpu.X86.HasAVX512CD},
		{"AVX512DQ", flags&WELS_CPU_AVX512DQ != 0, cpu.X86.HasAVX512DQ},
		{"AVX512BW", flags&WELS_CPU_AVX512BW != 0, cpu.X86.HasAVX512BW},
		{"AVX512VL", flags&WELS_CPU_AVX512VL != 0, cpu.X86.HasAVX512VL},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("detected = %t, want %t", test.got, test.want)
			}
		})
	}

	if cpu.X86.HasSSE2 {
		const baseline = WELS_CPU_MMX | WELS_CPU_MMXEXT | WELS_CPU_SSE |
			WELS_CPU_FPU | WELS_CPU_CMOV
		if missing := uint32(baseline) &^ flags; missing != 0 {
			t.Fatalf("amd64 baseline flags missing: %#x", missing)
		}
	}
}

func TestWelsCPUFeatureDetectNilCountAMD64(t *testing.T) {
	_ = WelsCPUFeatureDetect(nil)
}

func TestWelsCPUAVX2FlagValue(t *testing.T) {
	if WELS_CPU_AVX2 != 0x00040000 {
		t.Fatalf("WELS_CPU_AVX2 = %#x, want %#x", WELS_CPU_AVX2, uint32(0x00040000))
	}
}
