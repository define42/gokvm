# OpenH264 → Go porting conventions

This directory holds a pure Go (no cgo, no assembly) port of OpenH264.
The goal is **bit-exact** behaviour against the C++ reference (`_c`) code paths:
the decoder must reproduce the SHA1 hashes in `test/api/decoder_test.cpp`, and the
encoder must reproduce the bitstream hashes in `test/api/encoder_test.cpp`.

The port is deliberately a close, mechanical translation so that each Go file can
be diffed against its C++ source. Idiomatic wrappers live at the `pkg/h264` package root.

## Layout

| Go package                | C/C++ source                                   |
|---------------------------|------------------------------------------------|
| `api`                     | `codec/api/wels/*.h` (public types/constants)  |
| `internal/common`         | `codec/common`                                 |
| `internal/decoder`        | `codec/decoder/core` + `codec/decoder/plus`    |
| `internal/encoder`        | `codec/encoder/core` + `codec/encoder/plus`    |
| `internal/processing`     | `codec/processing`                             |
| `.` (package `openh264`)  | factory functions, idiomatic wrappers          |
| `cmd/h264dec`, `cmd/h264enc` | `codec/console/dec`, `codec/console/enc`    |
| `internal/console`        | `codec/console/common` (CReadConfig, atoi/atof) |

File mapping: `foo.cpp` → `foo.go`; `foo.h` → `foo_h.go` (types, constants,
inline functions, macros, function-pointer types). Tests: `foo_test.go`.

Not ported: SIMD assembly and every `#if defined(X86_ASM|HAVE_NEON|HAVE_MMI|...)`
branch, CPU detection (cpu flags are always 0), Windows DLL entry, trace-to-file,
the memory checker, platform thread/lock/event wrappers.

## Naming

* Inside a package keep the C identifiers **verbatim** (`WelsI4x4LumaPredV_c`,
  `SPicture`, `iWidthInPixel`, `g_kuiMbCountScan4Idx`). This keeps every symbol
  greppable across languages and lets independent files agree on names.
* Identifiers that must be used from another package (everything in `api` and
  `internal/common`, plus anything another package needs) are exported by
  **upper-casing the first letter only**: `g_kuiCache48CountScan4Idx` →
  `G_kuiCache48CountScan4Idx`, struct field `iPicWidth` → `IPicWidth`,
  `pfLumaHalfpelHor` → `PfLumaHalfpelHor`. Names that already start upper-case
  are unchanged (`WelsLog`, `WELS_CLIP3`, `SMcFunc`).
* C++ namespaces (`WelsCommon`, `WelsDec`, `WelsEnc`, `WelsVP`) map to packages.
* `typedef struct TagFoo {...} SFoo, *PFoo;` → `type SFoo struct {...}` plus
  `type PFoo = *SFoo` when the `P` alias is used.
* C++ classes → Go struct + methods; virtual interfaces → Go interfaces.

## Types

| C                                   | Go        |
|-------------------------------------|-----------|
| `int8_t … uint64_t`                 | `int8 … uint64` (same width!) |
| `int`, `long` (as used here)        | `int32`   |
| `unsigned int`                      | `uint32`  |
| `long long` / `unsigned long long`  | `int64` / `uint64` |
| `bool` / `BOOL`                     | `bool`    |
| `float` / `double`                  | `float32` / `float64` |
| `size_t`, array index/offset        | `int`     |

Keep the C width for every value that takes part in arithmetic so overflow and
truncation match. Use Go `int` only for slice indices, offsets and lengths.

### Integer semantics — the main source of mismatches

* **Promotion:** C promotes `uint8_t`/`int16_t` operands to `int` before any
  arithmetic. Go does not: `a+b` on `uint8` wraps. Convert to `int32` first:
  `int32(p[0]) + int32(p[1])`.
* **Mixed signedness:** in C, `int32_t < uint32_t` converts to unsigned. Make the
  conversion explicit in Go so it stays the same.
* **Shifts:** `>>` on signed values is arithmetic in both. A shift count must be
  non-negative in Go (it panics otherwise). `x << n` on a negative `x` is fine in Go.
* **Truncating stores:** `uint8_t v = expr;` → `v := uint8(expr)` (wraps the same way).
* `/` and `%` truncate toward zero in both languages.
* **Floats:** Go may fuse `x*y + z` into an FMA on arm64. Where the encoder's rate
  control has to be exact, force rounding with an explicit conversion:
  `float64(x*y) + z`.

## Pointers and memory

* `SFoo*` (one object) → `*SFoo`. A `malloc`ed array of structs → `[]SFoo`, using
  indices or `&s[i]` instead of pointer arithmetic.
* **Pixel/sample plane pointers** (anything that comes with a stride, or that is
  indexed negatively, e.g. `pPred[-1-stride]`, `pSrc - 2`) become a
  **(slice, offset) pair**: `pDst []uint8, iDstOff int` (plus the stride), where
  the slice is the *whole* underlying buffer. `pDst[k]` → `pDst[iDstOff+k]`, and
  `pDst += n` → `iDstOff += n`.
* Pointers that are only indexed with non-negative offsets (coefficient blocks,
  tables, bitstream buffers) may simply become a sub-slice `buf[off:]`.
* Picture planes: `pBuffer[i]` → the whole allocation (`[]uint8`), and `pData[i]`
  → `pData[i] []uint8` **holding the same whole allocation** plus
  `iDataOff[i] int` for where the plane origin lies inside it (so MC and
  deblocking can reach into the padding). Struct definitions in `*_h.go` say
  which representation each field uses; the `.cpp` ports must follow them.
* `WelsMalloc`/`WelsMallocz`/`CMemoryAlign` → `make`/`new` (allocations never
  fail in Go, but keep the C error-path structure where it is cheap to do so).
  `WelsFree` → assign `nil`. `memset(p,0,n)` → `clear()` or a zero value.
  `memcpy` → `copy()`. Struct `memcpy` → assignment.
* Fixed-size C arrays in structs stay Go arrays (they copy by value, as in C).
* Unaligned loads and stores (`LD32`, `ST32`, `*(uint32_t*)p`) used to copy memory → `copy()`.

## Functions

* Keep function-pointer tables (`SMcFunc`, `pGetI4x4LumaPredFunc[...]`, …) as Go
  `func` types/fields, and fill them in the `Init*` functions with the `_c`
  implementations only.
* Macros → functions in `internal/common/macros_h.go` (`WELS_CLIP3`, `WELS_MAX`,
  `WELS_MIN`, `WELS_ABS`, `WelsClip1`, `WELS_ROUND`, `WELS_DIV_ROUND`, …), as
  generic functions where several types are used.
* Return codes stay `int32` error codes exactly as in C (no Go `error` inside
  internal packages).
* Logging: `common.WelsLog(pLogCtx *common.SLogContext, iLevel int32, format string, args ...any)`;
  convert printf verbs to Go (`%lld`→`%d`, `%u`→`%d`, `%p`→`%p`).

## Threads

The decoder remains single-threaded. The encoder runs the audited fixed-slice,
CAVLC camera path with disabled rate control and load balancing on bounded Go
workers; other modes stay sequential. Per-slice buffers must have stable worker
ownership, shared results must be reduced after a `sync.WaitGroup` barrier, and
new concurrent paths require race and bit-exactness tests. OS thread handles,
events, and semaphores remain omitted.

## Checking your work

```
cd go && gofmt -l . && go vet ./... && go build ./... && go test ./...
```
