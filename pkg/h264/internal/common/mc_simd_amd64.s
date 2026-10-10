//go:build amd64

// The six-tap and pixel-average kernels follow OpenH264's x86 implementation
// in codec/common/x86/mc_luma.asm.
// Copyright (c) 2009-2013, Cisco Systems. BSD-2-Clause; see
// LICENSE-OpenH264 in the repository root.

#include "textflag.h"

// Coefficients for (p[-2]+p[3])-5*(p[-1]+p[2])+20*(p[0]+p[1]).
DATA ·mcFilterCoefficients+0(SB)/2, $1
DATA ·mcFilterCoefficients+2(SB)/2, $1
DATA ·mcFilterCoefficients+4(SB)/2, $-5
DATA ·mcFilterCoefficients+6(SB)/2, $-5
DATA ·mcFilterCoefficients+8(SB)/2, $20
DATA ·mcFilterCoefficients+10(SB)/2, $20
DATA ·mcFilterCoefficients+12(SB)/2, $0
DATA ·mcFilterCoefficients+14(SB)/2, $0
GLOBL ·mcFilterCoefficients(SB), RODATA|NOPTR, $16

// func filterInput8bitWithStrideSSE2(src []byte, stride int) int
// Requires: SSE2.
TEXT ·filterInput8bitWithStrideSSE2(SB), NOSPLIT, $0-40
	MOVQ src_base+0(FP), AX
	MOVQ stride+24(FP), CX

	// Gather the six unsigned samples into three coefficient pairs.
	PXOR X0, X0
	MOVQ AX, DX
	MOVQ CX, BX
	SHLQ $1, BX
	SUBQ BX, DX
	MOVBLZX (DX), R8
	PINSRW $0, R8, X0

	MOVQ AX, DX
	MOVQ CX, BX
	SHLQ $1, BX
	ADDQ CX, BX
	ADDQ BX, DX
	MOVBLZX (DX), R8
	PINSRW $1, R8, X0

	MOVQ AX, DX
	SUBQ CX, DX
	MOVBLZX (DX), R8
	PINSRW $2, R8, X0

	MOVQ AX, DX
	MOVQ CX, BX
	SHLQ $1, BX
	ADDQ BX, DX
	MOVBLZX (DX), R8
	PINSRW $3, R8, X0

	MOVBLZX (AX), R8
	PINSRW $4, R8, X0
	MOVQ AX, DX
	ADDQ CX, DX
	MOVBLZX (DX), R8
	PINSRW $5, R8, X0

	// PMADDWD forms the three signed pair sums in 32-bit lanes.
	MOVOU ·mcFilterCoefficients(SB), X1
	PMADDWL X1, X0
	MOVOU X0, X1
	PSRLDQ $8, X1
	PADDD X1, X0
	MOVOU X0, X1
	PSRLDQ $4, X1
	PADDD X1, X0
	MOVD X0, AX
	MOVQ AX, ret+32(FP)
	RET

// func pixelAvgSSE2(dst []byte, dstStride int, srcA []byte, srcAStride int,
//     srcB []byte, srcBStride, width, height int)
// Requires: SSE2.
TEXT ·pixelAvgSSE2(SB), NOSPLIT, $0-112
	MOVQ dst_base+0(FP), AX
	MOVQ dstStride+24(FP), R8
	MOVQ srcA_base+32(FP), BX
	MOVQ srcAStride+56(FP), R9
	MOVQ srcB_base+64(FP), CX
	MOVQ srcBStride+88(FP), R10
	MOVQ width+96(FP), DI
	MOVQ height+104(FP), SI

pixel_avg_row:
	MOVQ DI, DX

pixel_avg_16:
	CMPQ DX, $16
	JB pixel_avg_8
	MOVOU (BX), X0
	MOVOU (CX), X1
	PAVGB X1, X0
	MOVOU X0, (AX)
	ADDQ $16, AX
	ADDQ $16, BX
	ADDQ $16, CX
	SUBQ $16, DX
	JMP pixel_avg_16

pixel_avg_8:
	CMPQ DX, $8
	JB pixel_avg_4
	MOVQ (BX), X0
	MOVQ (CX), X1
	PAVGB X1, X0
	MOVQ X0, (AX)
	ADDQ $8, AX
	ADDQ $8, BX
	ADDQ $8, CX
	SUBQ $8, DX

pixel_avg_4:
	CMPQ DX, $4
	JB pixel_avg_tail
	MOVL (BX), R11
	MOVD R11, X0
	MOVL (CX), R12
	MOVD R12, X1
	PAVGB X1, X0
	MOVD X0, R11
	MOVL R11, (AX)
	ADDQ $4, AX
	ADDQ $4, BX
	ADDQ $4, CX
	SUBQ $4, DX

pixel_avg_tail:
	CMPQ DX, $0
	JE pixel_avg_row_done
	MOVBLZX (BX), R11
	MOVBLZX (CX), R12
	ADDQ R12, R11
	ADDQ $1, R11
	SHRQ $1, R11
	MOVB R11, (AX)
	INCQ AX
	INCQ BX
	INCQ CX
	DECQ DX
	JMP pixel_avg_tail

pixel_avg_row_done:
	ADDQ R8, AX
	SUBQ DI, AX
	ADDQ R9, BX
	SUBQ DI, BX
	ADDQ R10, CX
	SUBQ DI, CX
	DECQ SI
	JNZ pixel_avg_row
	RET

// func mcHorVer20SSE2(dst []byte, dstStride int, src []byte, srcStride,
//     width, height int)
// Requires: SSE2.
TEXT ·mcHorVer20SSE2(SB), NOSPLIT, $0-80
	MOVQ dst_base+0(FP), AX
	MOVQ dstStride+24(FP), CX
	MOVQ src_base+32(FP), DX
	MOVQ srcStride+56(FP), BX
	MOVQ width+64(FP), SI
	MOVQ height+72(FP), DI

	PXOR X7, X7
	PCMPEQW X6, X6
	PSRLW $15, X6
	PSLLW $4, X6

mc_hor_row:
	MOVQ SI, R8

mc_hor_8:
	CMPQ R8, $8
	JB mc_hor_4
	MOVQ -2(DX), X0
	PUNPCKLBW X7, X0
	MOVQ 3(DX), X1
	PUNPCKLBW X7, X1
	PADDW X1, X0
	MOVQ -1(DX), X2
	PUNPCKLBW X7, X2
	MOVQ 2(DX), X3
	PUNPCKLBW X7, X3
	PADDW X3, X2
	MOVQ (DX), X4
	PUNPCKLBW X7, X4
	MOVQ 1(DX), X5
	PUNPCKLBW X7, X5
	PADDW X5, X4
	MOVOU X4, X5
	PSLLW $2, X5
	PADDW X5, X4
	PSLLW $2, X4
	MOVOU X2, X3
	PSLLW $2, X3
	PADDW X2, X3
	PADDW X4, X0
	PSUBW X3, X0
	PADDW X6, X0
	PSRAW $5, X0
	PACKUSWB X0, X0
	MOVQ X0, (AX)
	ADDQ $8, AX
	ADDQ $8, DX
	SUBQ $8, R8
	JMP mc_hor_8

mc_hor_4:
	CMPQ R8, $4
	JB mc_hor_tail
	MOVL -2(DX), R9
	MOVD R9, X0
	PUNPCKLBW X7, X0
	MOVL 3(DX), R9
	MOVD R9, X1
	PUNPCKLBW X7, X1
	PADDW X1, X0
	MOVL -1(DX), R9
	MOVD R9, X2
	PUNPCKLBW X7, X2
	MOVL 2(DX), R9
	MOVD R9, X3
	PUNPCKLBW X7, X3
	PADDW X3, X2
	MOVL (DX), R9
	MOVD R9, X4
	PUNPCKLBW X7, X4
	MOVL 1(DX), R9
	MOVD R9, X5
	PUNPCKLBW X7, X5
	PADDW X5, X4
	MOVOU X4, X5
	PSLLW $2, X5
	PADDW X5, X4
	PSLLW $2, X4
	MOVOU X2, X3
	PSLLW $2, X3
	PADDW X2, X3
	PADDW X4, X0
	PSUBW X3, X0
	PADDW X6, X0
	PSRAW $5, X0
	PACKUSWB X0, X0
	MOVD X0, R9
	MOVL R9, (AX)
	ADDQ $4, AX
	ADDQ $4, DX
	SUBQ $4, R8

mc_hor_tail:
	CMPQ R8, $0
	JE mc_hor_row_done
	MOVBLZX -2(DX), R9
	MOVBLZX 3(DX), R10
	ADDQ R10, R9
	MOVBLZX -1(DX), R11
	MOVBLZX 2(DX), R10
	ADDQ R10, R11
	IMULQ $5, R11
	MOVBLZX (DX), R12
	MOVBLZX 1(DX), R10
	ADDQ R10, R12
	IMULQ $20, R12
	ADDQ R12, R9
	SUBQ R11, R9
	ADDQ $16, R9
	SARQ $5, R9
	CMPQ R9, $0
	JGE mc_hor_tail_high
	XORQ R9, R9
	JMP mc_hor_tail_store
mc_hor_tail_high:
	CMPQ R9, $255
	JLE mc_hor_tail_store
	MOVQ $255, R9
mc_hor_tail_store:
	MOVB R9, (AX)
	INCQ AX
	INCQ DX
	DECQ R8
	JMP mc_hor_tail

mc_hor_row_done:
	ADDQ CX, AX
	SUBQ SI, AX
	ADDQ BX, DX
	SUBQ SI, DX
	DECQ DI
	JNZ mc_hor_row
	RET

// func mcHorVer02SSE2(dst []byte, dstStride int, src []byte, srcStride,
//     width, height int)
// Requires: SSE2.
TEXT ·mcHorVer02SSE2(SB), NOSPLIT, $0-80
	MOVQ dst_base+0(FP), AX
	MOVQ dstStride+24(FP), CX
	MOVQ src_base+32(FP), DX
	MOVQ srcStride+56(FP), BX
	MOVQ width+64(FP), SI
	MOVQ height+72(FP), DI

	MOVQ BX, R10
	SHLQ $1, R10
	MOVQ R10, R12
	ADDQ BX, R12
	PXOR X7, X7
	PCMPEQW X6, X6
	PSRLW $15, X6
	PSLLW $4, X6

mc_ver_row:
	MOVQ SI, R8

mc_ver_8:
	CMPQ R8, $8
	JB mc_ver_4
	MOVQ DX, R9
	SUBQ R10, R9
	MOVQ (R9), X0
	PUNPCKLBW X7, X0
	MOVQ DX, R11
	ADDQ R12, R11
	MOVQ (R11), X1
	PUNPCKLBW X7, X1
	PADDW X1, X0
	MOVQ DX, R9
	SUBQ BX, R9
	MOVQ (R9), X2
	PUNPCKLBW X7, X2
	MOVQ DX, R11
	ADDQ R10, R11
	MOVQ (R11), X3
	PUNPCKLBW X7, X3
	PADDW X3, X2
	MOVQ (DX), X4
	PUNPCKLBW X7, X4
	MOVQ (DX)(BX*1), X5
	PUNPCKLBW X7, X5
	PADDW X5, X4
	MOVOU X4, X5
	PSLLW $2, X5
	PADDW X5, X4
	PSLLW $2, X4
	MOVOU X2, X3
	PSLLW $2, X3
	PADDW X2, X3
	PADDW X4, X0
	PSUBW X3, X0
	PADDW X6, X0
	PSRAW $5, X0
	PACKUSWB X0, X0
	MOVQ X0, (AX)
	ADDQ $8, AX
	ADDQ $8, DX
	SUBQ $8, R8
	JMP mc_ver_8

mc_ver_4:
	CMPQ R8, $4
	JB mc_ver_tail
	MOVQ DX, R9
	SUBQ R10, R9
	MOVL (R9), R13
	MOVD R13, X0
	PUNPCKLBW X7, X0
	MOVQ DX, R11
	ADDQ R12, R11
	MOVL (R11), R13
	MOVD R13, X1
	PUNPCKLBW X7, X1
	PADDW X1, X0
	MOVQ DX, R9
	SUBQ BX, R9
	MOVL (R9), R13
	MOVD R13, X2
	PUNPCKLBW X7, X2
	MOVQ DX, R11
	ADDQ R10, R11
	MOVL (R11), R13
	MOVD R13, X3
	PUNPCKLBW X7, X3
	PADDW X3, X2
	MOVL (DX), R13
	MOVD R13, X4
	PUNPCKLBW X7, X4
	MOVL (DX)(BX*1), R13
	MOVD R13, X5
	PUNPCKLBW X7, X5
	PADDW X5, X4
	MOVOU X4, X5
	PSLLW $2, X5
	PADDW X5, X4
	PSLLW $2, X4
	MOVOU X2, X3
	PSLLW $2, X3
	PADDW X2, X3
	PADDW X4, X0
	PSUBW X3, X0
	PADDW X6, X0
	PSRAW $5, X0
	PACKUSWB X0, X0
	MOVD X0, R13
	MOVL R13, (AX)
	ADDQ $4, AX
	ADDQ $4, DX
	SUBQ $4, R8

mc_ver_tail:
	CMPQ R8, $0
	JE mc_ver_row_done
	MOVQ DX, R11
	SUBQ R10, R11
	MOVBLZX (R11), R9
	MOVQ DX, R11
	ADDQ R12, R11
	MOVBLZX (R11), R13
	ADDQ R13, R9
	MOVQ DX, R11
	SUBQ BX, R11
	MOVBLZX (R11), R13
	MOVQ DX, R11
	ADDQ R10, R11
	MOVBLZX (R11), R14
	ADDQ R14, R13
	IMULQ $5, R13
	MOVBLZX (DX), R14
	MOVBLZX (DX)(BX*1), R11
	ADDQ R11, R14
	IMULQ $20, R14
	ADDQ R14, R9
	SUBQ R13, R9
	ADDQ $16, R9
	SARQ $5, R9
	CMPQ R9, $0
	JGE mc_ver_tail_high
	XORQ R9, R9
	JMP mc_ver_tail_store
mc_ver_tail_high:
	CMPQ R9, $255
	JLE mc_ver_tail_store
	MOVQ $255, R9
mc_ver_tail_store:
	MOVB R9, (AX)
	INCQ AX
	INCQ DX
	DECQ R8
	JMP mc_ver_tail

mc_ver_row_done:
	ADDQ CX, AX
	SUBQ SI, AX
	ADDQ BX, DX
	SUBQ SI, DX
	DECQ DI
	JNZ mc_ver_row
	RET

// func mcVerticalRawSSE2(dst []int16, dstStride int, src []byte,
//     srcStride, width, height int)
// Requires: SSE2. The source points at the leftmost tap column.
TEXT ·mcVerticalRawSSE2(SB), NOSPLIT, $0-80
	MOVQ dst_base+0(FP), AX
	MOVQ dstStride+24(FP), CX
	SHLQ $1, CX
	MOVQ src_base+32(FP), DX
	MOVQ srcStride+56(FP), BX
	MOVQ width+64(FP), SI
	MOVQ height+72(FP), DI

	MOVQ BX, R10
	SHLQ $1, R10
	MOVQ R10, R12
	ADDQ BX, R12
	PXOR X7, X7

mc_raw_ver_row:
	MOVQ SI, R8

mc_raw_ver_8:
	CMPQ R8, $8
	JB mc_raw_ver_4
	MOVQ DX, R9
	SUBQ R10, R9
	MOVQ (R9), X0
	PUNPCKLBW X7, X0
	MOVQ DX, R11
	ADDQ R12, R11
	MOVQ (R11), X1
	PUNPCKLBW X7, X1
	PADDW X1, X0
	MOVQ DX, R9
	SUBQ BX, R9
	MOVQ (R9), X2
	PUNPCKLBW X7, X2
	MOVQ DX, R11
	ADDQ R10, R11
	MOVQ (R11), X3
	PUNPCKLBW X7, X3
	PADDW X3, X2
	MOVQ (DX), X4
	PUNPCKLBW X7, X4
	MOVQ (DX)(BX*1), X5
	PUNPCKLBW X7, X5
	PADDW X5, X4
	MOVOU X4, X5
	PSLLW $2, X5
	PADDW X5, X4
	PSLLW $2, X4
	MOVOU X2, X3
	PSLLW $2, X3
	PADDW X2, X3
	PADDW X4, X0
	PSUBW X3, X0
	MOVOU X0, (AX)
	ADDQ $16, AX
	ADDQ $8, DX
	SUBQ $8, R8
	JMP mc_raw_ver_8

mc_raw_ver_4:
	CMPQ R8, $4
	JB mc_raw_ver_tail
	MOVQ DX, R9
	SUBQ R10, R9
	MOVL (R9), R13
	MOVD R13, X0
	PUNPCKLBW X7, X0
	MOVQ DX, R11
	ADDQ R12, R11
	MOVL (R11), R13
	MOVD R13, X1
	PUNPCKLBW X7, X1
	PADDW X1, X0
	MOVQ DX, R9
	SUBQ BX, R9
	MOVL (R9), R13
	MOVD R13, X2
	PUNPCKLBW X7, X2
	MOVQ DX, R11
	ADDQ R10, R11
	MOVL (R11), R13
	MOVD R13, X3
	PUNPCKLBW X7, X3
	PADDW X3, X2
	MOVL (DX), R13
	MOVD R13, X4
	PUNPCKLBW X7, X4
	MOVL (DX)(BX*1), R13
	MOVD R13, X5
	PUNPCKLBW X7, X5
	PADDW X5, X4
	MOVOU X4, X5
	PSLLW $2, X5
	PADDW X5, X4
	PSLLW $2, X4
	MOVOU X2, X3
	PSLLW $2, X3
	PADDW X2, X3
	PADDW X4, X0
	PSUBW X3, X0
	MOVQ X0, (AX)
	ADDQ $8, AX
	ADDQ $4, DX
	SUBQ $4, R8

mc_raw_ver_tail:
	CMPQ R8, $0
	JE mc_raw_ver_row_done
	MOVQ DX, R11
	SUBQ R10, R11
	MOVBLZX (R11), R9
	MOVQ DX, R11
	ADDQ R12, R11
	MOVBLZX (R11), R13
	ADDQ R13, R9
	MOVQ DX, R11
	SUBQ BX, R11
	MOVBLZX (R11), R13
	MOVQ DX, R11
	ADDQ R10, R11
	MOVBLZX (R11), R14
	ADDQ R14, R13
	IMULQ $5, R13
	MOVBLZX (DX), R14
	MOVBLZX (DX)(BX*1), R11
	ADDQ R11, R14
	IMULQ $20, R14
	ADDQ R14, R9
	SUBQ R13, R9
	MOVW R9, (AX)
	ADDQ $2, AX
	INCQ DX
	DECQ R8
	JMP mc_raw_ver_tail

mc_raw_ver_row_done:
	ADDQ CX, AX
	MOVQ SI, R9
	SHLQ $1, R9
	SUBQ R9, AX
	ADDQ BX, DX
	SUBQ SI, DX
	DECQ DI
	JNZ mc_raw_ver_row
	RET

// func mcHorizontalRawToU8SSE2(dst []byte, dstStride int, src []int16,
//     srcStride, width, height int)
// Requires: SSE2. Intermediate samples are kept signed until the final clip.
TEXT ·mcHorizontalRawToU8SSE2(SB), NOSPLIT, $0-80
	MOVQ dst_base+0(FP), AX
	MOVQ dstStride+24(FP), CX
	MOVQ src_base+32(FP), DX
	MOVQ srcStride+56(FP), BX
	SHLQ $1, BX
	MOVQ width+64(FP), SI
	MOVQ height+72(FP), DI

	PCMPEQL X7, X7
	PSRLL $31, X7
	PSLLL $9, X7

mc_raw_hor_row:
	MOVQ SI, R8

mc_raw_hor_4:
	CMPQ R8, $4
	JB mc_raw_hor_tail

	MOVQ 0(DX), X0
	MOVOU X0, X6
	PSRAW $15, X6
	PUNPCKLWL X6, X0
	MOVQ 10(DX), X1
	MOVOU X1, X6
	PSRAW $15, X6
	PUNPCKLWL X6, X1
	PADDD X1, X0

	MOVQ 2(DX), X2
	MOVOU X2, X6
	PSRAW $15, X6
	PUNPCKLWL X6, X2
	MOVQ 8(DX), X3
	MOVOU X3, X6
	PSRAW $15, X6
	PUNPCKLWL X6, X3
	PADDD X3, X2

	MOVQ 4(DX), X4
	MOVOU X4, X6
	PSRAW $15, X6
	PUNPCKLWL X6, X4
	MOVQ 6(DX), X5
	MOVOU X5, X6
	PSRAW $15, X6
	PUNPCKLWL X6, X5
	PADDD X5, X4

	MOVOU X4, X5
	PSLLL $2, X5
	PADDD X5, X4
	PSLLL $2, X4
	MOVOU X2, X3
	PSLLL $2, X3
	PADDD X2, X3
	PADDD X4, X0
	PSUBL X3, X0
	PADDD X7, X0
	PSRAL $10, X0
	PACKSSLW X0, X0
	PACKUSWB X0, X0
	MOVD X0, R9
	MOVL R9, (AX)

	ADDQ $4, AX
	ADDQ $8, DX
	SUBQ $4, R8
	JMP mc_raw_hor_4

mc_raw_hor_tail:
	CMPQ R8, $0
	JE mc_raw_hor_row_done
	MOVWQSX 0(DX), R9
	MOVWQSX 10(DX), R10
	ADDQ R10, R9
	MOVWQSX 2(DX), R11
	MOVWQSX 8(DX), R10
	ADDQ R10, R11
	IMULQ $5, R11
	MOVWQSX 4(DX), R12
	MOVWQSX 6(DX), R10
	ADDQ R10, R12
	IMULQ $20, R12
	ADDQ R12, R9
	SUBQ R11, R9
	ADDQ $512, R9
	SARQ $10, R9
	CMPQ R9, $0
	JGE mc_raw_hor_tail_high
	XORQ R9, R9
	JMP mc_raw_hor_tail_store
mc_raw_hor_tail_high:
	CMPQ R9, $255
	JLE mc_raw_hor_tail_store
	MOVQ $255, R9
mc_raw_hor_tail_store:
	MOVB R9, (AX)
	INCQ AX
	ADDQ $2, DX
	DECQ R8
	JMP mc_raw_hor_tail

mc_raw_hor_row_done:
	ADDQ CX, AX
	SUBQ SI, AX
	ADDQ BX, DX
	MOVQ SI, R9
	SHLQ $1, R9
	SUBQ R9, DX
	DECQ DI
	JNZ mc_raw_hor_row
	RET
