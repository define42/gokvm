package common

// Port of codec/common/inc/cpu_core.h (WELS CPU feature flags). The flags
// are kept for completeness; WelsCPUFeatureDetect always returns 0.
const (
	WELS_CPU_MMX      = 0x00000001 /* mmx */
	WELS_CPU_MMXEXT   = 0x00000002 /* mmx-ext*/
	WELS_CPU_SSE      = 0x00000004 /* sse */
	WELS_CPU_SSE2     = 0x00000008 /* sse 2 */
	WELS_CPU_SSE3     = 0x00000010 /* sse 3 */
	WELS_CPU_SSE41    = 0x00000020 /* sse 4.1 */
	WELS_CPU_3DNOW    = 0x00000040 /* 3dnow! */
	WELS_CPU_3DNOWEXT = 0x00000080 /* 3dnow! ext */
	WELS_CPU_ALTIVEC  = 0x00000100 /* altivec */
	WELS_CPU_SSSE3    = 0x00000200 /* ssse3 */
	WELS_CPU_SSE42    = 0x00000400 /* sse 4.2 */

	/* CPU features application extensive */
	WELS_CPU_FPU   = 0x00001000 /* x87-FPU on chip */
	WELS_CPU_HTT   = 0x00002000 /* Hyper-Threading Technology (HTT) */
	WELS_CPU_CMOV  = 0x00004000 /* Conditional Move Instructions */
	WELS_CPU_MOVBE = 0x00008000 /* MOVBE instruction */
	WELS_CPU_AES   = 0x00010000 /* AES instruction extensions */
	WELS_CPU_FMA   = 0x00020000 /* AVX VEX FMA instruction sets */
	WELS_CPU_AVX   = 0x00000800 /* Advanced Vector eXtentions */
	WELS_CPU_AVX2  = 0x00000000 /* !AVX2 (HAVE_AVX2 is never defined in the Go port) */

	WELS_CPU_AVX512F  = 0x00080000 /* AVX512F */
	WELS_CPU_AVX512CD = 0x00100000 /* AVX512CD */
	WELS_CPU_AVX512DQ = 0x00200000 /* AVX512DQ */
	WELS_CPU_AVX512BW = 0x00400000 /* AVX512BW */
	WELS_CPU_AVX512VL = 0x00800000 /* AVX512VL */

	WELS_CPU_CACHELINE_16  = 0x10000000 /* CacheLine Size 16 */
	WELS_CPU_CACHELINE_32  = 0x20000000 /* CacheLine Size 32 */
	WELS_CPU_CACHELINE_64  = 0x40000000 /* CacheLine Size 64 */
	WELS_CPU_CACHELINE_128 = 0x80000000 /* CacheLine Size 128 */

	/* For the android OS */
	WELS_CPU_ARMv7 = 0x000001 /* ARMv7 */
	WELS_CPU_VFPv3 = 0x000002 /* VFPv3 */
	WELS_CPU_NEON  = 0x000004 /* NEON */

	/* For loongson */
	WELS_CPU_MMI  = 0x00000001 /* mmi */
	WELS_CPU_MSA  = 0x00000002 /* msa */
	WELS_CPU_LSX  = 0x00000003 /* lsx */
	WELS_CPU_LASX = 0x00000004 /* lasx */
)
