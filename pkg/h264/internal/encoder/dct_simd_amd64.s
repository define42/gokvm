// This forward transform follows OpenH264's SSE2 implementation in
// codec/common/x86/dct.asm.
// Copyright (c) 2009-2013, Cisco Systems.
// All rights reserved.
// BSD-2-Clause; see LICENSE-OpenH264 in the repository root.

#include "textflag.h"

// func welsDctT4SSE2(dct []int16, pixel1 []byte, stride1 int,
//     pixel2 []byte, stride2 int)
TEXT ·welsDctT4SSE2(SB), NOSPLIT, $0-88
	MOVQ dct_base+0(FP), AX
	MOVQ pixel1_base+24(FP), BX
	MOVQ stride1+48(FP), CX
	MOVQ pixel2_base+56(FP), DX
	MOVQ stride2+80(FP), SI
	JMP dctT4SSE2Kernel<>(SB)

// func welsDctFourT4SSE2(dct []int16, pixel1 []byte, stride1 int,
//     pixel2 []byte, stride2 int)
TEXT ·welsDctFourT4SSE2(SB), NOSPLIT, $0-88
	MOVQ dct_base+0(FP), R8
	MOVQ pixel1_base+24(FP), R9
	MOVQ stride1+48(FP), CX
	MOVQ pixel2_base+56(FP), R10
	MOVQ stride2+80(FP), SI

	// Top-left block.
	MOVQ R8, AX
	MOVQ R9, BX
	MOVQ R10, DX
	CALL dctT4SSE2Kernel<>(SB)

	// Top-right block.
	LEAQ 32(R8), AX
	LEAQ 4(R9), BX
	LEAQ 4(R10), DX
	CALL dctT4SSE2Kernel<>(SB)

	// Bottom-left block.
	LEAQ 64(R8), AX
	LEAQ (R9)(CX*4), BX
	LEAQ (R10)(SI*4), DX
	CALL dctT4SSE2Kernel<>(SB)

	// Bottom-right block.
	LEAQ 96(R8), AX
	LEAQ 4(R9)(CX*4), BX
	LEAQ 4(R10)(SI*4), DX
	CALL dctT4SSE2Kernel<>(SB)
	RET

// dctT4SSE2Kernel uses a private register ABI:
// AX=destination, BX=pixel1, CX=stride1, DX=pixel2, SI=stride2.
TEXT dctT4SSE2Kernel<>(SB), NOSPLIT|NOFRAME, $0-0

	// Load four rows of four unsigned pixels and form signed word differences.
	PXOR X10, X10
	MOVD (BX), X0
	MOVD (DX), X4
	PUNPCKLBW X10, X0
	PUNPCKLBW X10, X4
	PSUBW X4, X0
	ADDQ CX, BX
	ADDQ SI, DX

	MOVD (BX), X1
	MOVD (DX), X4
	PUNPCKLBW X10, X1
	PUNPCKLBW X10, X4
	PSUBW X4, X1
	ADDQ CX, BX
	ADDQ SI, DX

	MOVD (BX), X2
	MOVD (DX), X4
	PUNPCKLBW X10, X2
	PUNPCKLBW X10, X4
	PSUBW X4, X2
	ADDQ CX, BX
	ADDQ SI, DX

	MOVD (BX), X3
	MOVD (DX), X4
	PUNPCKLBW X10, X3
	PUNPCKLBW X10, X4
	PSUBW X4, X3

	// Apply the vertical 4-point transform in parallel to all four columns.
	// A=x0+x3, B=x1+x2, C=x1-x2, D=x0-x3.
	MOVOU X0, X4
	PADDW X3, X4
	MOVOU X1, X5
	PADDW X2, X5
	MOVOU X1, X6
	PSUBW X2, X6
	MOVOU X0, X7
	PSUBW X3, X7

	MOVOU X4, X0
	PADDW X5, X0
	MOVOU X7, X1
	PADDW X7, X1
	PADDW X6, X1
	MOVOU X4, X2
	PSUBW X5, X2
	MOVOU X6, X8
	PADDW X6, X8
	MOVOU X7, X3
	PSUBW X8, X3

	// Transpose the low 4x4 words so the same packed transform computes the
	// horizontal dimension.
	MOVOU X0, X4
	PUNPCKLWL X1, X4
	MOVOU X2, X5
	PUNPCKLWL X3, X5
	MOVOU X4, X6
	PUNPCKLLQ X5, X6
	MOVOU X4, X7
	PUNPCKHLQ X5, X7
	MOVOU X6, X0
	MOVOU X6, X1
	PSRLDQ $8, X1
	MOVOU X7, X2
	MOVOU X7, X3
	PSRLDQ $8, X3

	// Apply the horizontal 4-point transform.
	MOVOU X0, X4
	PADDW X3, X4
	MOVOU X1, X5
	PADDW X2, X5
	MOVOU X1, X6
	PSUBW X2, X6
	MOVOU X0, X7
	PSUBW X3, X7

	MOVOU X4, X0
	PADDW X5, X0
	MOVOU X7, X1
	PADDW X7, X1
	PADDW X6, X1
	MOVOU X4, X2
	PSUBW X5, X2
	MOVOU X6, X8
	PADDW X6, X8
	MOVOU X7, X3
	PSUBW X8, X3

	// Restore row-major coefficient order.
	MOVOU X0, X4
	PUNPCKLWL X1, X4
	MOVOU X2, X5
	PUNPCKLWL X3, X5
	MOVOU X4, X6
	PUNPCKLLQ X5, X6
	MOVOU X4, X7
	PUNPCKHLQ X5, X7
	MOVOU X6, X0
	MOVOU X6, X1
	PSRLDQ $8, X1
	MOVOU X7, X2
	MOVOU X7, X3
	PSRLDQ $8, X3

	MOVQ X0, 0(AX)
	MOVQ X1, 8(AX)
	MOVQ X2, 16(AX)
	MOVQ X3, 24(AX)
	RET
