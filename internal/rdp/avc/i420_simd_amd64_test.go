//go:build amd64

package avc

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"testing"

	"golang.org/x/sys/cpu"
)

const i420SIMDGuardSize = 37

type i420SIMDGuardedBytes struct {
	storage []byte
	value   []byte
	prefix  int
	canary  byte
}

func newI420SIMDGuardedBytes(length, prefix int, canary byte) i420SIMDGuardedBytes {
	storage := bytes.Repeat([]byte{canary}, prefix+length+i420SIMDGuardSize)

	return i420SIMDGuardedBytes{
		storage: storage,
		value:   storage[prefix : prefix+length],
		prefix:  prefix,
		canary:  canary,
	}
}

func (b i420SIMDGuardedBytes) checkGuards(t *testing.T) {
	t.Helper()
	for index, value := range b.storage[:b.prefix] {
		if value != b.canary {
			t.Fatalf("prefix guard byte %d = %#x, want %#x", index, value, b.canary)
		}
	}
	for index, value := range b.storage[b.prefix+len(b.value):] {
		if value != b.canary {
			t.Fatalf("suffix guard byte %d = %#x, want %#x", index, value, b.canary)
		}
	}
}

func TestRGBAToI420AVX2MatchesScalar(t *testing.T) {
	t.Parallel()
	if !cpu.X86.HasAVX2 {
		t.Skip("AVX2 is unavailable")
	}

	widths := []int{2, 4, 6, 8, 10, 14, 16, 18, 30, 32, 34, 62, 64, 66, 130, 1920}
	sourceOffsets := []int{0, 4, 12, 28}
	destinationOffsets := []int{0, 1, 3, 7, 15, 31}
	for caseIndex, width := range widths {
		t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
			t.Parallel()
			random := rand.New(rand.NewPCG(uint64(width), uint64(width*37+11))) //nolint:gosec // Deterministic test data.
			src0 := newI420SIMDGuardedBytes(width*4+11, sourceOffsets[caseIndex%len(sourceOffsets)], 0xa1)
			src1 := newI420SIMDGuardedBytes(width*4+13, sourceOffsets[(caseIndex+1)%len(sourceOffsets)], 0xb2)
			dstY0 := newI420SIMDGuardedBytes(width, destinationOffsets[caseIndex%len(destinationOffsets)], 0xc3)
			dstY1 := newI420SIMDGuardedBytes(width+5, destinationOffsets[(caseIndex+1)%len(destinationOffsets)], 0xd4)
			dstU := newI420SIMDGuardedBytes(width/2+3, destinationOffsets[(caseIndex+2)%len(destinationOffsets)], 0xe5)
			dstV := newI420SIMDGuardedBytes(width/2+7, destinationOffsets[(caseIndex+3)%len(destinationOffsets)], 0xf6)

			for _, buffer := range [][]byte{src0.value, src1.value, dstY0.value, dstY1.value, dstU.value, dstV.value} {
				for index := range buffer {
					buffer[index] = byte(random.Uint32())
				}
			}
			source0Before := bytes.Clone(src0.storage)
			source1Before := bytes.Clone(src1.storage)
			y1ExtraBefore := bytes.Clone(dstY1.value[width:])
			uExtraBefore := bytes.Clone(dstU.value[width/2:])
			vExtraBefore := bytes.Clone(dstV.value[width/2:])

			wantY0, wantY1, wantU, wantV := referenceRGBAI420Rows(src0.value, src1.value, width)
			wantChanged := !bytes.Equal(dstY0.value, wantY0) ||
				!bytes.Equal(dstY1.value[:width], wantY1) ||
				!bytes.Equal(dstU.value[:width/2], wantU) ||
				!bytes.Equal(dstV.value[:width/2], wantV)
			changed, ok := rgbaToI420AVX2(
				dstY0.value,
				dstY1.value[:width],
				dstU.value[:width/2],
				dstV.value[:width/2],
				src0.value[:width*4],
				src1.value[:width*4],
			)
			if !ok {
				t.Fatal("valid input was rejected on an AVX2 CPU")
			}
			if changed != wantChanged {
				t.Fatalf("changed = %t, want %t", changed, wantChanged)
			}
			checkI420SIMDPlanes(t, dstY0.value, dstY1.value[:width], dstU.value[:width/2], dstV.value[:width/2],
				wantY0, wantY1, wantU, wantV)

			if !bytes.Equal(src0.storage, source0Before) || !bytes.Equal(src1.storage, source1Before) {
				t.Fatal("conversion modified a source row")
			}
			if !bytes.Equal(dstY1.value[width:], y1ExtraBefore) ||
				!bytes.Equal(dstU.value[width/2:], uExtraBefore) ||
				!bytes.Equal(dstV.value[width/2:], vExtraBefore) {
				t.Fatal("conversion modified bytes beyond the requested span")
			}
			for _, buffer := range []i420SIMDGuardedBytes{src0, src1, dstY0, dstY1, dstU, dstV} {
				buffer.checkGuards(t)
			}
		})
	}
}

func TestRGBAToI420AVX2ChangedAndRepair(t *testing.T) {
	t.Parallel()
	if !cpu.X86.HasAVX2 {
		t.Skip("AVX2 is unavailable")
	}

	const width = 30 // Three vector blocks and a six-pixel Go tail.
	src0 := make([]byte, width*4)
	src1 := make([]byte, width*4)
	for index := range src0 {
		src0[index] = byte(index*29 + 17)
		src1[index] = byte(index*43 + 91)
	}
	wantY0, wantY1, wantU, wantV := referenceRGBAI420Rows(src0, src1, width)

	t.Run("unchanged", func(t *testing.T) {
		t.Parallel()
		y0, y1 := bytes.Clone(wantY0), bytes.Clone(wantY1)
		u, v := bytes.Clone(wantU), bytes.Clone(wantV)
		changed, ok := rgbaToI420AVX2(y0, y1, u, v, src0, src1)
		if !ok || changed {
			t.Fatalf("unchanged conversion = changed %t, ok %t", changed, ok)
		}
	})

	t.Run("alpha ignored", func(t *testing.T) {
		t.Parallel()
		alpha0, alpha1 := bytes.Clone(src0), bytes.Clone(src1)
		for x := range width {
			alpha0[x*4+3] ^= 0xff
			alpha1[x*4+3] ^= byte(x*13 + 1)
		}
		y0, y1 := bytes.Clone(wantY0), bytes.Clone(wantY1)
		u, v := bytes.Clone(wantU), bytes.Clone(wantV)
		changed, ok := rgbaToI420AVX2(y0, y1, u, v, alpha0, alpha1)
		if !ok || changed {
			t.Fatalf("alpha-only change = changed %t, ok %t", changed, ok)
		}
	})

	for _, test := range []struct {
		name  string
		plane int
		index int
	}{
		{"Y0 first vector", 0, 0},
		{"Y0 last tail", 0, width - 1},
		{"Y1 first vector", 1, 0},
		{"Y1 last tail", 1, width - 1},
		{"U first vector", 2, 0},
		{"U last tail", 2, width/2 - 1},
		{"V first vector", 3, 0},
		{"V last tail", 3, width/2 - 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			planes := [][]byte{bytes.Clone(wantY0), bytes.Clone(wantY1), bytes.Clone(wantU), bytes.Clone(wantV)}
			planes[test.plane][test.index] ^= 1
			changed, ok := rgbaToI420AVX2(planes[0], planes[1], planes[2], planes[3], src0, src1)
			if !ok || !changed {
				t.Fatalf("mismatched destination = changed %t, ok %t", changed, ok)
			}
			checkI420SIMDPlanes(t, planes[0], planes[1], planes[2], planes[3], wantY0, wantY1, wantU, wantV)

			changed, ok = rgbaToI420AVX2(planes[0], planes[1], planes[2], planes[3], src0, src1)
			if !ok || changed {
				t.Fatalf("repaired destination = changed %t, ok %t", changed, ok)
			}
		})
	}
}

func TestRGBAToI420AVX2RejectsUnsupportedShapes(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name                     string
		y0, y1, u, v, src0, src1 int
	}{
		{"empty", 0, 0, 0, 0, 0, 0},
		{"odd width", 9, 9, 4, 4, 36, 36},
		{"short Y1", 16, 15, 8, 8, 64, 64},
		{"long Y1", 16, 17, 8, 8, 64, 64},
		{"short U", 16, 16, 7, 8, 64, 64},
		{"long U", 16, 16, 9, 8, 64, 64},
		{"short V", 16, 16, 8, 7, 64, 64},
		{"long V", 16, 16, 8, 9, 64, 64},
		{"short source row 0", 16, 16, 8, 8, 63, 64},
		{"long source row 0", 16, 16, 8, 8, 65, 64},
		{"short source row 1", 16, 16, 8, 8, 64, 63},
		{"long source row 1", 16, 16, 8, 8, 64, 65},
		{
			"over maximum width",
			maxDimension + 2, maxDimension + 2,
			maxDimension/2 + 1, maxDimension/2 + 1,
			(maxDimension + 2) * 4, (maxDimension + 2) * 4,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			buffers := [][]byte{
				make([]byte, test.y0), make([]byte, test.y1),
				make([]byte, test.u), make([]byte, test.v),
				make([]byte, test.src0), make([]byte, test.src1),
			}
			for bufferIndex, buffer := range buffers {
				for index := range buffer {
					buffer[index] = byte(bufferIndex*31 + index*17 + 1)
				}
			}
			before := make([][]byte, len(buffers))
			for index := range buffers {
				before[index] = bytes.Clone(buffers[index])
			}

			changed, ok := rgbaToI420AVX2(buffers[0], buffers[1], buffers[2], buffers[3], buffers[4], buffers[5])
			if changed || ok {
				t.Fatalf("unsupported shape = changed %t, ok %t", changed, ok)
			}
			for index := range buffers {
				if !bytes.Equal(buffers[index], before[index]) {
					t.Fatalf("unsupported shape modified buffer %d", index)
				}
			}
		})
	}
}

func referenceRGBAI420Rows(src0, src1 []byte, width int) (y0, y1, u, v []byte) {
	y0 = make([]byte, width)
	y1 = make([]byte, width)
	u = make([]byte, width/2)
	v = make([]byte, width/2)
	for x := 0; x < width; x += 2 {
		offset := x * 4
		r00, g00, b00 := int(src0[offset]), int(src0[offset+1]), int(src0[offset+2])
		r01, g01, b01 := int(src0[offset+4]), int(src0[offset+5]), int(src0[offset+6])
		r10, g10, b10 := int(src1[offset]), int(src1[offset+1]), int(src1[offset+2])
		r11, g11, b11 := int(src1[offset+4]), int(src1[offset+5]), int(src1[offset+6])

		y0[x] = byte((54*r00 + 183*g00 + 18*b00) >> 8)
		y0[x+1] = byte((54*r01 + 183*g01 + 18*b01) >> 8)
		y1[x] = byte((54*r10 + 183*g10 + 18*b10) >> 8)
		y1[x+1] = byte((54*r11 + 183*g11 + 18*b11) >> 8)
		red := (r00 + r01 + r10 + r11) / 4
		green := (g00 + g01 + g10 + g11) / 4
		blue := (b00 + b01 + b10 + b11) / 4
		u[x/2] = byte(((-29*red - 99*green + 128*blue) >> 8) + 128)
		v[x/2] = byte(((128*red - 116*green - 12*blue) >> 8) + 128)
	}

	return y0, y1, u, v
}

func checkI420SIMDPlanes(t *testing.T, y0, y1, u, v, wantY0, wantY1, wantU, wantV []byte) {
	t.Helper()
	for _, plane := range []struct {
		name      string
		got, want []byte
	}{
		{"Y0", y0, wantY0},
		{"Y1", y1, wantY1},
		{"U", u, wantU},
		{"V", v, wantV},
	} {
		if !bytes.Equal(plane.got, plane.want) {
			for index, want := range plane.want {
				if plane.got[index] != want {
					t.Fatalf("%s byte %d = %d, want %d", plane.name, index, plane.got[index], want)
				}
			}
		}
	}
}
