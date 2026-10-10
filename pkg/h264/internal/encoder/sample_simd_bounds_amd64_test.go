//go:build amd64

package encoder

import "testing"

func TestSATDSIMDRejectsOutOfRangeSpan(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("SIMD SATD wrapper accepted an out-of-range buffer span")
		}
	}()
	WelsSampleSatd8x8_sse41(make([]byte, 64), 0, 8, make([]byte, 63), 0, 8)
}
