//go:build amd64

// Derived from github.com/oops1/go.264 v1.12.0/internal/simd.
// Copyright (c) 2026 Valery Chukalin. BSD-2-Clause; see LICENSE-go.264.

#include "textflag.h"
// func satd4x4(src []byte, srcStride int, ref []byte, refStride int) int
// Requires: SSE2, SSE4.1, SSSE3
TEXT ·satd4x4(SB), NOSPLIT, $0-72
	MOVQ       src_base+0(FP), AX
	MOVQ       srcStride+24(FP), CX
	MOVQ       ref_base+32(FP), DX
	MOVQ       refStride+56(FP), BX
	PMOVZXBD   (AX), X0
	PMOVZXBD   (DX), X1
	MOVOU      X0, X0
	PSUBL      X1, X0
	ADDQ       CX, AX
	ADDQ       BX, DX
	PMOVZXBD   (AX), X1
	PMOVZXBD   (DX), X2
	MOVOU      X1, X1
	PSUBL      X2, X1
	ADDQ       CX, AX
	ADDQ       BX, DX
	PMOVZXBD   (AX), X2
	PMOVZXBD   (DX), X3
	MOVOU      X2, X2
	PSUBL      X3, X2
	ADDQ       CX, AX
	ADDQ       BX, DX
	PMOVZXBD   (AX), X3
	PMOVZXBD   (DX), X4
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X0, X4
	PUNPCKLLQ  X1, X4
	MOVOU      X0, X0
	PUNPCKHLQ  X1, X0
	MOVOU      X2, X1
	PUNPCKLLQ  X3, X1
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X4, X3
	PUNPCKLQDQ X1, X3
	MOVOU      X4, X4
	PUNPCKHQDQ X1, X4
	MOVOU      X0, X1
	PUNPCKLQDQ X2, X1
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X4, X2
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X1, X4
	PADDD      X0, X4
	MOVOU      X1, X1
	PSUBL      X0, X1
	MOVOU      X2, X0
	PADDD      X4, X0
	MOVOU      X3, X5
	PADDD      X1, X5
	MOVOU      X2, X2
	PSUBL      X4, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X0, X1
	PUNPCKLLQ  X5, X1
	MOVOU      X0, X0
	PUNPCKHLQ  X5, X0
	MOVOU      X2, X4
	PUNPCKLLQ  X3, X4
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X1, X3
	PUNPCKLQDQ X4, X3
	MOVOU      X1, X1
	PUNPCKHQDQ X4, X1
	MOVOU      X0, X4
	PUNPCKLQDQ X2, X4
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X1, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X4, X1
	PADDD      X0, X1
	MOVOU      X4, X4
	PSUBL      X0, X4
	MOVOU      X2, X0
	PADDD      X1, X0
	MOVOU      X3, X5
	PADDD      X4, X5
	MOVOU      X2, X2
	PSUBL      X1, X2
	MOVOU      X3, X1
	PSUBL      X4, X1
	PABSD      X0, X0
	PABSD      X5, X3
	PABSD      X2, X2
	PABSD      X1, X1
	MOVOU      X0, X0
	PADDD      X3, X0
	MOVOU      X2, X2
	PADDD      X1, X2
	MOVOU      X0, X0
	PADDD      X2, X0
	MOVOU      X0, X0
	MOVOU      X0, X1
	PSRLDQ     $0x08, X1
	PADDD      X1, X0
	MOVOU      X0, X1
	PSRLDQ     $0x04, X1
	PADDD      X1, X0
	MOVD       X0, AX
	ADDL       $0x01, AX
	SARL       $0x01, AX
	MOVQ       AX, ret+64(FP)
	RET
// func satd8x8(src []byte, srcStride int, ref []byte, refStride int) int
// Requires: SSE2, SSE4.1, SSSE3
TEXT ·satd8x8(SB), NOSPLIT, $0-72
	MOVQ       src_base+0(FP), AX
	MOVQ       srcStride+24(FP), CX
	MOVQ       ref_base+32(FP), DX
	MOVQ       refStride+56(FP), BX
	MOVQ       CX, SI
	SHLQ       $0x02, SI
	MOVQ       BX, DI
	SHLQ       $0x02, DI
	MOVQ       AX, R8
	MOVQ       DX, R9
	PMOVZXBD   (R8), X0
	PMOVZXBD   (R9), X1
	MOVOU      X0, X0
	PSUBL      X1, X0
	ADDQ       CX, R8
	ADDQ       BX, R9
	PMOVZXBD   (R8), X1
	PMOVZXBD   (R9), X2
	MOVOU      X1, X1
	PSUBL      X2, X1
	ADDQ       CX, R8
	ADDQ       BX, R9
	PMOVZXBD   (R8), X2
	PMOVZXBD   (R9), X3
	MOVOU      X2, X2
	PSUBL      X3, X2
	ADDQ       CX, R8
	ADDQ       BX, R9
	PMOVZXBD   (R8), X3
	PMOVZXBD   (R9), X4
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X0, X4
	PUNPCKLLQ  X1, X4
	MOVOU      X0, X0
	PUNPCKHLQ  X1, X0
	MOVOU      X2, X1
	PUNPCKLLQ  X3, X1
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X4, X3
	PUNPCKLQDQ X1, X3
	MOVOU      X4, X4
	PUNPCKHQDQ X1, X4
	MOVOU      X0, X1
	PUNPCKLQDQ X2, X1
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X4, X2
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X1, X4
	PADDD      X0, X4
	MOVOU      X1, X1
	PSUBL      X0, X1
	MOVOU      X2, X0
	PADDD      X4, X0
	MOVOU      X3, X5
	PADDD      X1, X5
	MOVOU      X2, X2
	PSUBL      X4, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X0, X1
	PUNPCKLLQ  X5, X1
	MOVOU      X0, X0
	PUNPCKHLQ  X5, X0
	MOVOU      X2, X4
	PUNPCKLLQ  X3, X4
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X1, X3
	PUNPCKLQDQ X4, X3
	MOVOU      X1, X1
	PUNPCKHQDQ X4, X1
	MOVOU      X0, X4
	PUNPCKLQDQ X2, X4
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X1, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X4, X1
	PADDD      X0, X1
	MOVOU      X4, X4
	PSUBL      X0, X4
	MOVOU      X2, X0
	PADDD      X1, X0
	MOVOU      X3, X5
	PADDD      X4, X5
	MOVOU      X2, X2
	PSUBL      X1, X2
	MOVOU      X3, X1
	PSUBL      X4, X1
	PABSD      X0, X0
	PABSD      X5, X3
	PABSD      X2, X2
	PABSD      X1, X1
	MOVOU      X0, X0
	PADDD      X3, X0
	MOVOU      X2, X2
	PADDD      X1, X2
	MOVOU      X0, X0
	PADDD      X2, X0
	MOVOU      X0, X0
	MOVOU      X0, X1
	PSRLDQ     $0x08, X1
	PADDD      X1, X0
	MOVOU      X0, X1
	PSRLDQ     $0x04, X1
	PADDD      X1, X0
	MOVD       X0, R8
	ADDL       $0x01, R8
	SARL       $0x01, R8
	MOVQ       AX, R9
	ADDQ       $0x04, R9
	MOVQ       DX, R10
	ADDQ       $0x04, R10
	PMOVZXBD   (R9), X0
	PMOVZXBD   (R10), X1
	MOVOU      X0, X0
	PSUBL      X1, X0
	ADDQ       CX, R9
	ADDQ       BX, R10
	PMOVZXBD   (R9), X1
	PMOVZXBD   (R10), X2
	MOVOU      X1, X1
	PSUBL      X2, X1
	ADDQ       CX, R9
	ADDQ       BX, R10
	PMOVZXBD   (R9), X2
	PMOVZXBD   (R10), X3
	MOVOU      X2, X2
	PSUBL      X3, X2
	ADDQ       CX, R9
	ADDQ       BX, R10
	PMOVZXBD   (R9), X3
	PMOVZXBD   (R10), X4
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X0, X4
	PUNPCKLLQ  X1, X4
	MOVOU      X0, X0
	PUNPCKHLQ  X1, X0
	MOVOU      X2, X1
	PUNPCKLLQ  X3, X1
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X4, X3
	PUNPCKLQDQ X1, X3
	MOVOU      X4, X4
	PUNPCKHQDQ X1, X4
	MOVOU      X0, X1
	PUNPCKLQDQ X2, X1
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X4, X2
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X1, X4
	PADDD      X0, X4
	MOVOU      X1, X1
	PSUBL      X0, X1
	MOVOU      X2, X0
	PADDD      X4, X0
	MOVOU      X3, X5
	PADDD      X1, X5
	MOVOU      X2, X2
	PSUBL      X4, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X0, X1
	PUNPCKLLQ  X5, X1
	MOVOU      X0, X0
	PUNPCKHLQ  X5, X0
	MOVOU      X2, X4
	PUNPCKLLQ  X3, X4
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X1, X3
	PUNPCKLQDQ X4, X3
	MOVOU      X1, X1
	PUNPCKHQDQ X4, X1
	MOVOU      X0, X4
	PUNPCKLQDQ X2, X4
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X1, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X4, X1
	PADDD      X0, X1
	MOVOU      X4, X4
	PSUBL      X0, X4
	MOVOU      X2, X0
	PADDD      X1, X0
	MOVOU      X3, X5
	PADDD      X4, X5
	MOVOU      X2, X2
	PSUBL      X1, X2
	MOVOU      X3, X1
	PSUBL      X4, X1
	PABSD      X0, X0
	PABSD      X5, X3
	PABSD      X2, X2
	PABSD      X1, X1
	MOVOU      X0, X0
	PADDD      X3, X0
	MOVOU      X2, X2
	PADDD      X1, X2
	MOVOU      X0, X0
	PADDD      X2, X0
	MOVOU      X0, X0
	MOVOU      X0, X1
	PSRLDQ     $0x08, X1
	PADDD      X1, X0
	MOVOU      X0, X1
	PSRLDQ     $0x04, X1
	PADDD      X1, X0
	MOVD       X0, R9
	ADDL       $0x01, R9
	SARL       $0x01, R9
	ADDQ       R9, R8
	MOVQ       AX, R9
	ADDQ       SI, R9
	MOVQ       DX, R10
	ADDQ       DI, R10
	PMOVZXBD   (R9), X0
	PMOVZXBD   (R10), X1
	MOVOU      X0, X0
	PSUBL      X1, X0
	ADDQ       CX, R9
	ADDQ       BX, R10
	PMOVZXBD   (R9), X1
	PMOVZXBD   (R10), X2
	MOVOU      X1, X1
	PSUBL      X2, X1
	ADDQ       CX, R9
	ADDQ       BX, R10
	PMOVZXBD   (R9), X2
	PMOVZXBD   (R10), X3
	MOVOU      X2, X2
	PSUBL      X3, X2
	ADDQ       CX, R9
	ADDQ       BX, R10
	PMOVZXBD   (R9), X3
	PMOVZXBD   (R10), X4
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X0, X4
	PUNPCKLLQ  X1, X4
	MOVOU      X0, X0
	PUNPCKHLQ  X1, X0
	MOVOU      X2, X1
	PUNPCKLLQ  X3, X1
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X4, X3
	PUNPCKLQDQ X1, X3
	MOVOU      X4, X4
	PUNPCKHQDQ X1, X4
	MOVOU      X0, X1
	PUNPCKLQDQ X2, X1
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X4, X2
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X1, X4
	PADDD      X0, X4
	MOVOU      X1, X1
	PSUBL      X0, X1
	MOVOU      X2, X0
	PADDD      X4, X0
	MOVOU      X3, X5
	PADDD      X1, X5
	MOVOU      X2, X2
	PSUBL      X4, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X0, X1
	PUNPCKLLQ  X5, X1
	MOVOU      X0, X0
	PUNPCKHLQ  X5, X0
	MOVOU      X2, X4
	PUNPCKLLQ  X3, X4
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X1, X3
	PUNPCKLQDQ X4, X3
	MOVOU      X1, X1
	PUNPCKHQDQ X4, X1
	MOVOU      X0, X4
	PUNPCKLQDQ X2, X4
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X1, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X4, X1
	PADDD      X0, X1
	MOVOU      X4, X4
	PSUBL      X0, X4
	MOVOU      X2, X0
	PADDD      X1, X0
	MOVOU      X3, X5
	PADDD      X4, X5
	MOVOU      X2, X2
	PSUBL      X1, X2
	MOVOU      X3, X1
	PSUBL      X4, X1
	PABSD      X0, X0
	PABSD      X5, X3
	PABSD      X2, X2
	PABSD      X1, X1
	MOVOU      X0, X0
	PADDD      X3, X0
	MOVOU      X2, X2
	PADDD      X1, X2
	MOVOU      X0, X0
	PADDD      X2, X0
	MOVOU      X0, X0
	MOVOU      X0, X1
	PSRLDQ     $0x08, X1
	PADDD      X1, X0
	MOVOU      X0, X1
	PSRLDQ     $0x04, X1
	PADDD      X1, X0
	MOVD       X0, R9
	ADDL       $0x01, R9
	SARL       $0x01, R9
	ADDQ       R9, R8
	ADDQ       $0x04, AX
	ADDQ       SI, AX
	ADDQ       $0x04, DX
	ADDQ       DI, DX
	PMOVZXBD   (AX), X0
	PMOVZXBD   (DX), X1
	MOVOU      X0, X0
	PSUBL      X1, X0
	ADDQ       CX, AX
	ADDQ       BX, DX
	PMOVZXBD   (AX), X1
	PMOVZXBD   (DX), X2
	MOVOU      X1, X1
	PSUBL      X2, X1
	ADDQ       CX, AX
	ADDQ       BX, DX
	PMOVZXBD   (AX), X2
	PMOVZXBD   (DX), X3
	MOVOU      X2, X2
	PSUBL      X3, X2
	ADDQ       CX, AX
	ADDQ       BX, DX
	PMOVZXBD   (AX), X3
	PMOVZXBD   (DX), X4
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X0, X4
	PUNPCKLLQ  X1, X4
	MOVOU      X0, X0
	PUNPCKHLQ  X1, X0
	MOVOU      X2, X1
	PUNPCKLLQ  X3, X1
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X4, X3
	PUNPCKLQDQ X1, X3
	MOVOU      X4, X4
	PUNPCKHQDQ X1, X4
	MOVOU      X0, X1
	PUNPCKLQDQ X2, X1
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X4, X2
	MOVOU      X3, X3
	PSUBL      X4, X3
	MOVOU      X1, X4
	PADDD      X0, X4
	MOVOU      X1, X1
	PSUBL      X0, X1
	MOVOU      X2, X0
	PADDD      X4, X0
	MOVOU      X3, X5
	PADDD      X1, X5
	MOVOU      X2, X2
	PSUBL      X4, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X0, X1
	PUNPCKLLQ  X5, X1
	MOVOU      X0, X0
	PUNPCKHLQ  X5, X0
	MOVOU      X2, X4
	PUNPCKLLQ  X3, X4
	MOVOU      X2, X2
	PUNPCKHLQ  X3, X2
	MOVOU      X1, X3
	PUNPCKLQDQ X4, X3
	MOVOU      X1, X1
	PUNPCKHQDQ X4, X1
	MOVOU      X0, X4
	PUNPCKLQDQ X2, X4
	MOVOU      X0, X0
	PUNPCKHQDQ X2, X0
	MOVOU      X3, X2
	PADDD      X1, X2
	MOVOU      X3, X3
	PSUBL      X1, X3
	MOVOU      X4, X1
	PADDD      X0, X1
	MOVOU      X4, X4
	PSUBL      X0, X4
	MOVOU      X2, X0
	PADDD      X1, X0
	MOVOU      X3, X5
	PADDD      X4, X5
	MOVOU      X2, X2
	PSUBL      X1, X2
	MOVOU      X3, X1
	PSUBL      X4, X1
	PABSD      X0, X0
	PABSD      X5, X3
	PABSD      X2, X2
	PABSD      X1, X1
	MOVOU      X0, X0
	PADDD      X3, X0
	MOVOU      X2, X2
	PADDD      X1, X2
	MOVOU      X0, X0
	PADDD      X2, X0
	MOVOU      X0, X0
	MOVOU      X0, X1
	PSRLDQ     $0x08, X1
	PADDD      X1, X0
	MOVOU      X0, X1
	PSRLDQ     $0x04, X1
	PADDD      X1, X0
	MOVD       X0, AX
	ADDL       $0x01, AX
	SARL       $0x01, AX
	ADDQ       AX, R8
	MOVQ       R8, ret+64(FP)
	RET

// func satd8x8AVX2(src []byte, srcStride int, ref []byte, refStride int) int
// Requires: AVX, AVX2
TEXT ·satd8x8AVX2(SB), NOSPLIT, $0-72
	MOVQ         src_base+0(FP), AX
	MOVQ         srcStride+24(FP), CX
	MOVQ         ref_base+32(FP), DX
	MOVQ         refStride+56(FP), BX
	MOVQ         CX, SI
	SHLQ         $0x02, SI
	MOVQ         BX, DI
	SHLQ         $0x02, DI
	MOVL         $0x00010001, R8
	VMOVD        R8, X0
	VPBROADCASTD X0, Y0
	MOVQ         AX, R8
	MOVQ         DX, R9
	VMOVD        (R8), X1
	ADDQ         CX, R8
	VMOVD        (R8), X2
	ADDQ         CX, R8
	VMOVD        (R8), X3
	ADDQ         CX, R8
	VMOVD        (R8), X4
	VPUNPCKLDQ   X2, X1, X1
	VPUNPCKLDQ   X4, X3, X2
	VPUNPCKLQDQ  X2, X1, X1
	MOVQ         R9, R8
	VMOVD        (R8), X2
	ADDQ         BX, R8
	VMOVD        (R8), X3
	ADDQ         BX, R8
	VMOVD        (R8), X4
	ADDQ         BX, R8
	VMOVD        (R8), X5
	VPUNPCKLDQ   X3, X2, X2
	VPUNPCKLDQ   X5, X4, X3
	VPUNPCKLQDQ  X3, X2, X2
	VPMOVZXBW    X1, Y1
	VPMOVZXBW    X2, Y2
	VPSUBW       Y2, Y1, Y1
	VPSHUFLW     $0xb1, Y1, Y2
	VPSHUFHW     $0xb1, Y2, Y2
	VPADDW       Y2, Y1, Y3
	VPSUBW       Y2, Y1, Y1
	VPBLENDW     $0xaa, Y1, Y3, Y1
	VPSHUFD      $0xb1, Y1, Y2
	VPADDW       Y2, Y1, Y3
	VPSUBW       Y2, Y1, Y1
	VPBLENDD     $0xaa, Y1, Y3, Y1
	VPSHUFD      $0x4e, Y1, Y2
	VPADDW       Y2, Y1, Y3
	VPSUBW       Y2, Y1, Y1
	VPBLENDD     $0xcc, Y1, Y3, Y1
	VPERMQ       $0x4e, Y1, Y2
	VPADDW       Y2, Y1, Y3
	VPSUBW       Y2, Y1, Y1
	VPABSW       Y3, Y2
	VPABSW       Y1, Y1
	VPADDW       Y1, Y2, Y1
	VPMADDWD     Y0, Y1, Y1
	VEXTRACTI128 $0x01, Y1, X2
	VPADDD       X2, X1, X1
	VPSHUFD      $0x0e, X1, X2
	VPADDD       X2, X1, X1
	VPSHUFD      $0x01, X1, X2
	VPADDD       X2, X1, X1
	MOVQ         AX, R8
	ADDQ         $0x04, R8
	MOVQ         DX, R9
	ADDQ         $0x04, R9
	VMOVD        (R8), X2
	ADDQ         CX, R8
	VMOVD        (R8), X3
	ADDQ         CX, R8
	VMOVD        (R8), X4
	ADDQ         CX, R8
	VMOVD        (R8), X5
	VPUNPCKLDQ   X3, X2, X2
	VPUNPCKLDQ   X5, X4, X3
	VPUNPCKLQDQ  X3, X2, X2
	MOVQ         R9, R8
	VMOVD        (R8), X3
	ADDQ         BX, R8
	VMOVD        (R8), X4
	ADDQ         BX, R8
	VMOVD        (R8), X5
	ADDQ         BX, R8
	VMOVD        (R8), X6
	VPUNPCKLDQ   X4, X3, X3
	VPUNPCKLDQ   X6, X5, X4
	VPUNPCKLQDQ  X4, X3, X3
	VPMOVZXBW    X2, Y2
	VPMOVZXBW    X3, Y3
	VPSUBW       Y3, Y2, Y2
	VPSHUFLW     $0xb1, Y2, Y3
	VPSHUFHW     $0xb1, Y3, Y3
	VPADDW       Y3, Y2, Y4
	VPSUBW       Y3, Y2, Y2
	VPBLENDW     $0xaa, Y2, Y4, Y2
	VPSHUFD      $0xb1, Y2, Y3
	VPADDW       Y3, Y2, Y4
	VPSUBW       Y3, Y2, Y2
	VPBLENDD     $0xaa, Y2, Y4, Y2
	VPSHUFD      $0x4e, Y2, Y3
	VPADDW       Y3, Y2, Y4
	VPSUBW       Y3, Y2, Y2
	VPBLENDD     $0xcc, Y2, Y4, Y2
	VPERMQ       $0x4e, Y2, Y3
	VPADDW       Y3, Y2, Y4
	VPSUBW       Y3, Y2, Y2
	VPABSW       Y4, Y3
	VPABSW       Y2, Y2
	VPADDW       Y2, Y3, Y2
	VPMADDWD     Y0, Y2, Y2
	VEXTRACTI128 $0x01, Y2, X3
	VPADDD       X3, X2, X2
	VPSHUFD      $0x0e, X2, X3
	VPADDD       X3, X2, X2
	VPSHUFD      $0x01, X2, X3
	VPADDD       X3, X2, X2
	MOVQ         AX, R8
	ADDQ         SI, R8
	MOVQ         DX, R9
	ADDQ         DI, R9
	VMOVD        (R8), X3
	ADDQ         CX, R8
	VMOVD        (R8), X4
	ADDQ         CX, R8
	VMOVD        (R8), X5
	ADDQ         CX, R8
	VMOVD        (R8), X6
	VPUNPCKLDQ   X4, X3, X3
	VPUNPCKLDQ   X6, X5, X4
	VPUNPCKLQDQ  X4, X3, X3
	MOVQ         R9, R8
	VMOVD        (R8), X4
	ADDQ         BX, R8
	VMOVD        (R8), X5
	ADDQ         BX, R8
	VMOVD        (R8), X6
	ADDQ         BX, R8
	VMOVD        (R8), X7
	VPUNPCKLDQ   X5, X4, X4
	VPUNPCKLDQ   X7, X6, X5
	VPUNPCKLQDQ  X5, X4, X4
	VPMOVZXBW    X3, Y3
	VPMOVZXBW    X4, Y4
	VPSUBW       Y4, Y3, Y3
	VPSHUFLW     $0xb1, Y3, Y4
	VPSHUFHW     $0xb1, Y4, Y4
	VPADDW       Y4, Y3, Y5
	VPSUBW       Y4, Y3, Y3
	VPBLENDW     $0xaa, Y3, Y5, Y3
	VPSHUFD      $0xb1, Y3, Y4
	VPADDW       Y4, Y3, Y5
	VPSUBW       Y4, Y3, Y3
	VPBLENDD     $0xaa, Y3, Y5, Y3
	VPSHUFD      $0x4e, Y3, Y4
	VPADDW       Y4, Y3, Y5
	VPSUBW       Y4, Y3, Y3
	VPBLENDD     $0xcc, Y3, Y5, Y3
	VPERMQ       $0x4e, Y3, Y4
	VPADDW       Y4, Y3, Y5
	VPSUBW       Y4, Y3, Y3
	VPABSW       Y5, Y4
	VPABSW       Y3, Y3
	VPADDW       Y3, Y4, Y3
	VPMADDWD     Y0, Y3, Y3
	VEXTRACTI128 $0x01, Y3, X4
	VPADDD       X4, X3, X3
	VPSHUFD      $0x0e, X3, X4
	VPADDD       X4, X3, X3
	VPSHUFD      $0x01, X3, X4
	VPADDD       X4, X3, X3
	ADDQ         $0x04, AX
	ADDQ         SI, AX
	ADDQ         $0x04, DX
	ADDQ         DI, DX
	VMOVD        (AX), X4
	ADDQ         CX, AX
	VMOVD        (AX), X5
	ADDQ         CX, AX
	VMOVD        (AX), X6
	ADDQ         CX, AX
	VMOVD        (AX), X7
	VPUNPCKLDQ   X5, X4, X4
	VPUNPCKLDQ   X7, X6, X5
	VPUNPCKLQDQ  X5, X4, X4
	MOVQ         DX, AX
	VMOVD        (AX), X5
	ADDQ         BX, AX
	VMOVD        (AX), X6
	ADDQ         BX, AX
	VMOVD        (AX), X7
	ADDQ         BX, AX
	VMOVD        (AX), X8
	VPUNPCKLDQ   X6, X5, X5
	VPUNPCKLDQ   X8, X7, X6
	VPUNPCKLQDQ  X6, X5, X5
	VPMOVZXBW    X4, Y4
	VPMOVZXBW    X5, Y5
	VPSUBW       Y5, Y4, Y4
	VPSHUFLW     $0xb1, Y4, Y5
	VPSHUFHW     $0xb1, Y5, Y5
	VPADDW       Y5, Y4, Y6
	VPSUBW       Y5, Y4, Y4
	VPBLENDW     $0xaa, Y4, Y6, Y4
	VPSHUFD      $0xb1, Y4, Y5
	VPADDW       Y5, Y4, Y6
	VPSUBW       Y5, Y4, Y4
	VPBLENDD     $0xaa, Y4, Y6, Y4
	VPSHUFD      $0x4e, Y4, Y5
	VPADDW       Y5, Y4, Y6
	VPSUBW       Y5, Y4, Y4
	VPBLENDD     $0xcc, Y4, Y6, Y4
	VPERMQ       $0x4e, Y4, Y5
	VPADDW       Y5, Y4, Y6
	VPSUBW       Y5, Y4, Y4
	VPABSW       Y6, Y5
	VPABSW       Y4, Y4
	VPADDW       Y4, Y5, Y4
	VPMADDWD     Y0, Y4, Y0
	VEXTRACTI128 $0x01, Y0, X4
	VPADDD       X4, X0, X0
	VPSHUFD      $0x0e, X0, X4
	VPADDD       X4, X0, X0
	VPSHUFD      $0x01, X0, X4
	VPADDD       X4, X0, X0
	VMOVD        X1, AX
	SHRL         $0x01, AX
	ADDL         $0x01, AX
	SARL         $0x01, AX
	VMOVD        X2, CX
	SHRL         $0x01, CX
	ADDL         $0x01, CX
	SARL         $0x01, CX
	ADDQ         CX, AX
	VMOVD        X3, CX
	SHRL         $0x01, CX
	ADDL         $0x01, CX
	SARL         $0x01, CX
	ADDQ         CX, AX
	VMOVD        X0, CX
	SHRL         $0x01, CX
	ADDL         $0x01, CX
	SARL         $0x01, CX
	ADDQ         CX, AX
	VZEROUPPER
	MOVQ         AX, ret+64(FP)
	RET
