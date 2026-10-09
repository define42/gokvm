// Copyright (c) 2013, Cisco Systems
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
//   - Redistributions of source code must retain the above copyright
//     notice, this list of conditions and the following disclaimer.
//
//   - Redistributions in binary form must reproduce the above copyright
//     notice, this list of conditions and the following disclaimer in
//     the documentation and/or other materials provided with the
//     distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
// FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE
// COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
// INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
// BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
// LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN
// ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

// Package encoder is a pure Go port of the OpenH264 encoder
// (codec/encoder/core and codec/encoder/plus, C++ namespace WelsEnc).
//
// The port is a close, mechanical translation; see go/PORTING.md for the
// general conventions. File mapping: foo.cpp -> foo.go, foo.h -> foo_h.go.
// C identifiers are kept verbatim. Symbols of codec/common live in package
// common (common.SBitStringAux, common.WelsLog, common.NAL_UNIT_SPS, ...),
// codec/processing in package processing and the public API types in
// package api (api.SEncParamExt, ...).
//
// Encoder-specific representation rules. The *_h.go struct definitions
// document the representation of every pointer field; the .go ports of the
// .cpp files must follow them.
//
//   - Picture planes (SPicture): pBuffer and pData[i] all hold the whole
//     picture allocation; iDataOff[i] is the offset of plane i's origin
//     inside it, so negative offsets (padding) stay reachable.
//   - Other pixel/sample pointers into pictures (pEncMb, pDecMb, pRefMb,
//     pCsData, pEncData, ...) are (slice, offset) pairs: a field or
//     parameter pFoo becomes pFoo []uint8 plus iFooOff int. The same holds
//     for every other negatively indexed pointer (MVD cost tables:
//     pMvdCost + iMvdCostOff, background flags: pVaaBgMbFlag +
//     iVaaBgMbFlagOff, SAD maps: pEncSad + iEncSadOff, ...).
//   - Every uint8_t pixel pointer parameter of a function (or function
//     pointer type) that comes with a stride is a (slice, offset) pair, even
//     when it is only indexed non-negatively (prediction buffers): callers
//     pass (buf, 0) for scratch buffers.
//   - Scratch buffers inside structs that are only indexed non-negatively
//     (SMbCache.pMemPredLuma, SMeRefinePointer.pHalfPixH, ...) are plain
//     sub-slices of their allocation; pointer moves become re-slicing.
//   - Coefficient pointers (int16_t*) are sub-slices ([]int16).
//     SDCTCoeff.iLumaBlock / iChromaBlock are flattened ([256]int16,
//     [128]int16) because the C code walks a single pointer across the rows
//     of the 2-D arrays.
//   - Per-MB arrays are slices indexed by MB (SMB.sMv is the 16-entry
//     sub-slice of sWelsEncCtx.pMvUnitBlock4x4, ...). SMB carries a Go-only
//     back reference pMbList to the MB list of its layer so that the C
//     pointer arithmetic pCurMb + n / pCurMb - iMbWidth can be written as
//     pCurMb.Add(n) / pCurMb.Add(-iMbWidth).
//   - Bitstream writer: common.SBitStringAux. Byte pointers of the CABAC
//     engine (SCabacCtx.m_pBufStart/m_pBufEnd/m_pBufCur) are int offsets into
//     the shared buffer SCabacCtx.m_pBuf, which is the same slice as the
//     bitstream writer's buffer.
//   - A struct pointer that walks an array and is also indexed backwards
//     becomes (slice, index): SLayerBSInfo*& pLayerBsInfo in encoder_ext is
//     pLayerBsInfo []api.SLayerBSInfo (the whole SFrameBSInfo.SLayerInfo)
//     plus iLayerBsInfoIdx *int. A struct pointer that is only dereferenced
//     or compared stays a plain pointer (&array[i]).
//   - Fixed-size C array parameters that the callee writes (uint8_t
//     uiBS[2][4][4], ...) become pointers to Go arrays (*[2][4][4]uint8).
//   - C++ reference parameters (T&) become *T when the callee writes them and
//     plain values when they are const.
//   - void* parameters that are always a known struct (void* pEncCtx, void*
//     pSlice, ...) are typed with that struct.
//   - CMemoryAlign* parameters are dropped: WelsMalloc/WelsMallocz become
//     make/new (zeroed), WelsFree becomes assigning nil. sWelsEncCtx has no
//     pMemAlign field.
//   - C++ classes become structs with methods. Virtual dispatch from base
//     class code is done through a Go-only field holding the most-derived
//     object as an interface (pVirt / self); constructors become
//     New<Class>(...) functions plus ctor<Class>(...) methods that run the
//     constructor body (so derived constructors can chain), destructors
//     become Destruct() methods (call them where C++ uses delete). Static
//     member functions become plain functions (CreatePreProcess,
//     CreateParametersetStrategy, CreateReferenceStrategy,
//     CreateTaskManage); C++ overloads get distinct names
//     (SWelsSvcCodingParam::FillDefault (SEncParamExt&) -> FillDefaultParam,
//     CWelsPreProcess::GetBestRefPic (kiDidx, iRefTemporalIdx) ->
//     GetBestRefPic2). C++ default arguments must be passed explicitly.
//   - C unions: SWelsME.uSadPredISatd (uiSadPred / uiSatd) is one uint32.
//   - Static (file-local) helpers and file-local tables are owned by the
//     .go port of their .cpp file; cross-file tables live in the *_h.go file
//     of the header that declares them (or encoder_data_tables.go).
//   - The encoder is single-threaded: thread pools, events and mutexes are
//     dropped; the task classes (wels_task_*.h) keep a minimal shape and run
//     their tasks sequentially. SSliceThreading keeps only the per-thread
//     data the sequential path needs.
//
// The original code is licensed under the BSD 2-Clause license reproduced
// above; this port is distributed under the same terms.
package encoder
