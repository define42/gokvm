//go:build amd64

// Ported from Cisco OpenH264 codec/encoder/core/x86/quant.asm.
// Copyright (c) 2009-2013, Cisco Systems.
// BSD-2-Clause; see LICENSE-OpenH264.

#include "textflag.h"

// Quantization follows OpenH264's valid coefficient domain: ff and mf are
// non-negative int16 values. PADDUSW is exact because abs(dct)+ff <= 65535.

// func quant4x4SSE2(dct []int16, ff []int16, mf []int16)
TEXT ·quant4x4SSE2(SB), NOSPLIT, $0-72
	MOVQ dct_base+0(FP), AX
	MOVQ ff_base+24(FP), BX
	MOVQ mf_base+48(FP), CX
	MOVOU (BX), X2
	MOVOU (CX), X3
	MOVQ $2, DX

quant4x4_loop:
	MOVOU (AX), X0
	PXOR X1, X1
	PCMPGTW X0, X1
	PXOR X1, X0
	PSUBW X1, X0
	PADDUSW X2, X0
	PMULHUW X3, X0
	PXOR X1, X0
	PSUBW X1, X0
	MOVOU X0, (AX)
	ADDQ $16, AX
	DECQ DX
	JNZ quant4x4_loop
	RET

// func quant4x4DcSSE2(dct []int16, ff int64, mf int64)
TEXT ·quant4x4DcSSE2(SB), NOSPLIT, $0-40
	MOVQ ff+24(FP), AX
	MOVD AX, X2
	PSHUFLW $0, X2, X2
	PSHUFD $0, X2, X2
	MOVQ mf+32(FP), AX
	MOVD AX, X3
	PSHUFLW $0, X3, X3
	PSHUFD $0, X3, X3
	MOVQ dct_base+0(FP), AX
	MOVQ $2, DX

quant_dc_loop:
	MOVOU (AX), X0
	PXOR X1, X1
	PCMPGTW X0, X1
	PXOR X1, X0
	PSUBW X1, X0
	PADDUSW X2, X0
	PMULHUW X3, X0
	PXOR X1, X0
	PSUBW X1, X0
	MOVOU X0, (AX)
	ADDQ $16, AX
	DECQ DX
	JNZ quant_dc_loop
	RET

// func quantFour4x4SSE2(dct []int16, ff []int16, mf []int16)
TEXT ·quantFour4x4SSE2(SB), NOSPLIT, $0-72
	MOVQ dct_base+0(FP), AX
	MOVQ ff_base+24(FP), BX
	MOVQ mf_base+48(FP), CX
	MOVOU (BX), X2
	MOVOU (CX), X3
	MOVQ $8, DX

quant_four_loop:
	MOVOU (AX), X0
	PXOR X1, X1
	PCMPGTW X0, X1
	PXOR X1, X0
	PSUBW X1, X0
	PADDUSW X2, X0
	PMULHUW X3, X0
	PXOR X1, X0
	PSUBW X1, X0
	MOVOU X0, (AX)
	ADDQ $16, AX
	DECQ DX
	JNZ quant_four_loop
	RET

// func quantFour4x4MaxSSE2(dct []int16, ff []int16, mf []int16, max []int16)
TEXT ·quantFour4x4MaxSSE2(SB), NOSPLIT, $0-96
	MOVQ dct_base+0(FP), AX
	MOVQ ff_base+24(FP), BX
	MOVQ mf_base+48(FP), CX
	MOVQ max_base+72(FP), DI
	MOVOU (BX), X2
	MOVOU (CX), X3
	MOVQ $4, R8

quant_max_block:
	PXOR X4, X4

	MOVOU (AX), X0
	PXOR X1, X1
	PCMPGTW X0, X1
	PXOR X1, X0
	PSUBW X1, X0
	PADDUSW X2, X0
	PMULHUW X3, X0
	PMAXSW X0, X4
	PXOR X1, X0
	PSUBW X1, X0
	MOVOU X0, (AX)

	MOVOU 16(AX), X0
	PXOR X1, X1
	PCMPGTW X0, X1
	PXOR X1, X0
	PSUBW X1, X0
	PADDUSW X2, X0
	PMULHUW X3, X0
	PMAXSW X0, X4
	PXOR X1, X0
	PSUBW X1, X0
	MOVOU X0, 16(AX)

	MOVOU X4, X5
	PSRLDQ $8, X5
	PMAXSW X5, X4
	MOVOU X4, X5
	PSRLDQ $4, X5
	PMAXSW X5, X4
	MOVOU X4, X5
	PSRLDQ $2, X5
	PMAXSW X5, X4
	PEXTRW $0, X4, R9
	MOVW R9, (DI)

	ADDQ $32, AX
	ADDQ $2, DI
	DECQ R8
	JNZ quant_max_block
	RET
