//go:build amd64

package processing

import (
	"fmt"
	"math/rand"
	"testing"
)

var vaaBenchmarkSink int32

func BenchmarkVAACalcSad(b *testing.B) {
	sizes := []struct {
		width, height int32
	}{
		{width: 320, height: 192},
		{width: 1920, height: 1088},
	}
	implementations := []struct {
		name string
		call VAACalcSadFunc
	}{
		{name: "Scalar", call: VAACalcSad_c},
		{name: "SSE2", call: VAACalcSad_sse2},
	}

	for _, size := range sizes {
		stride := (size.width + 95) &^ 31
		offset := int(32*stride + 32)
		bufferSize := offset + int((size.height+32)*stride) + 32
		cur := make([]uint8, bufferSize)
		ref := make([]uint8, bufferSize)
		rng := rand.New(rand.NewSource(int64(size.width)<<32 | int64(size.height)))
		rng.Read(cur)
		rng.Read(ref)
		mbCount := int(size.width>>4) * int(size.height>>4)

		for _, implementation := range implementations {
			b.Run(fmt.Sprintf("%dx%d/%s", size.width, size.height, implementation.name), func(b *testing.B) {
				sad := make([][4]int32, mbCount)
				var frameSad int32
				b.SetBytes(int64(size.width * size.height))
				b.ReportAllocs()
				for b.Loop() {
					implementation.call(cur, offset, ref, offset, size.width, size.height, stride, &frameSad, sad)
				}
				vaaBenchmarkSink = frameSad + sad[len(sad)-1][3]
			})
		}
	}
}
