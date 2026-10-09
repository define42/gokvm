package common

import "encoding/binary"

// Port of codec/common/inc/ls_defines.h.
//
// The unaligned load/store macros operate on a (slice, offset) pair. Values
// use little endian byte order (the native order of every platform the C
// reference is tested on). When the macros are only used to copy memory
// (ST32(dst, LD32(src))) prefer a plain copy().

// LD16 loads a uint16 from p[off:].
func LD16(p []uint8, off int) uint16 { return binary.LittleEndian.Uint16(p[off:]) }

// LD32 loads a uint32 from p[off:].
func LD32(p []uint8, off int) uint32 { return binary.LittleEndian.Uint32(p[off:]) }

// LD64 loads a uint64 from p[off:].
func LD64(p []uint8, off int) uint64 { return binary.LittleEndian.Uint64(p[off:]) }

// ST16 stores b at p[off:].
func ST16(p []uint8, off int, b uint16) { binary.LittleEndian.PutUint16(p[off:], b) }

// ST32 stores b at p[off:].
func ST32(p []uint8, off int, b uint32) { binary.LittleEndian.PutUint32(p[off:], b) }

// ST64 stores b at p[off:].
func ST64(p []uint8, off int, b uint64) { binary.LittleEndian.PutUint64(p[off:], b) }

// Aligned variants (alignment is irrelevant in Go).

func LD16A2(p []uint8, off int) uint16 { return LD16(p, off) }
func LD32A2(p []uint8, off int) uint32 { return LD32(p, off) }
func LD32A4(p []uint8, off int) uint32 { return LD32(p, off) }
func LD64A2(p []uint8, off int) uint64 { return LD64(p, off) }
func LD64A4(p []uint8, off int) uint64 { return LD64(p, off) }
func LD64A8(p []uint8, off int) uint64 { return LD64(p, off) }

func ST16A2(p []uint8, off int, b uint16) { ST16(p, off, b) }
func ST32A2(p []uint8, off int, b uint32) { ST32(p, off, b) }
func ST32A4(p []uint8, off int, b uint32) { ST32(p, off, b) }
func ST64A2(p []uint8, off int, b uint64) { ST64(p, off, b) }
func ST64A4(p []uint8, off int, b uint64) { ST64(p, off, b) }
func ST64A8(p []uint8, off int, b uint64) { ST64(p, off, b) }

// INTD16 / INTD32 / INTD64 are aliases of LD16 / LD32 / LD64.

func INTD16(p []uint8, off int) uint16 { return LD16(p, off) }
func INTD32(p []uint8, off int) uint32 { return LD32(p, off) }
func INTD64(p []uint8, off int) uint64 { return LD64(p, off) }
