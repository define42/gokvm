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

// Package decoder is a pure Go port of the OpenH264 decoder
// (codec/decoder/core and codec/decoder/plus, C++ namespace WelsDec).
//
// The port is a close, mechanical translation; see go/PORTING.md for the
// general conventions. File mapping: foo.cpp -> foo.go, foo.h -> foo_h.go.
// C identifiers are kept verbatim.
//
// Decoder-specific representation rules (the *_h.go struct definitions
// document the representation of every pointer field):
//
//   - Picture planes (SPicture): pBuffer[i] and pData[i] both hold the whole
//     picture allocation; iDataOff[i] is the offset of plane i's origin inside
//     it, so negative offsets (padding) stay reachable.
//   - Other pixel pointers are (slice, offset) pairs: a field or parameter
//     pFoo becomes pFoo []uint8 plus iFooOff int.
//   - Per-macroblock arrays such as int16_t (*pMv[LIST_A])[16][2] become
//     slices indexed by MB: [LIST_A][][MB_BLOCK4x4_NUM][MV_A]int16.
//   - Fixed-size C array parameters (int16_t iMvArray[LIST_A][30][MV_A],
//     int16_t iMVs[2], ...) become pointers to Go arrays (*[LIST_A][30][MV_A]int16,
//     *[2]int16) so writes are visible to the caller, as in C.
//   - Non-negatively indexed buffers (coefficients, caches, tables) become
//     sub-slices (buf[off:]).
//   - C++ reference parameters (T&) become *T when the callee writes them and
//     plain values when they are const.
//   - Byte-stream positions inside the decoder's raw buffers (SDataBuffer)
//     are offsets into the whole buffer so that buffer reallocation and the
//     pointer comparisons of the C code keep working.
//   - The decoder is single-threaded: thread contexts, events and semaphores
//     are kept only as inert placeholders (pThreadCtx is always nil).
//
// The original code is licensed under the BSD 2-Clause license reproduced
// above; this port is distributed under the same terms.
package decoder
