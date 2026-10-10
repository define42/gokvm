package encoder

import (
	"math/rand"
	"slices"
	"testing"
)

func TestWelsDctT4SSE2MatchesScalar(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(0x264))
	for test := 0; test < 4000; test++ {
		stride1 := 4 + rng.Intn(61)
		stride2 := 4 + rng.Intn(61)
		off1 := rng.Intn(17)
		off2 := rng.Intn(17)
		pixel1 := make([]byte, off1+3*stride1+4)
		pixel2 := make([]byte, off2+3*stride2+4)
		_, _ = rng.Read(pixel1)
		_, _ = rng.Read(pixel2)

		want := make([]int16, 16)
		got := make([]int16, 16)
		WelsDctT4_c(want, pixel1, off1, int32(stride1), pixel2, off2, int32(stride2))
		WelsDctT4_sse2(got, pixel1, off1, int32(stride1), pixel2, off2, int32(stride2))
		if !slices.Equal(got, want) {
			t.Fatalf("test %d (strides %d/%d, offsets %d/%d):\n got %v\nwant %v", test, stride1, stride2, off1, off2, got, want)
		}
	}
}

func TestWelsDctT4SSE2ExtremePixels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    byte
		b    byte
	}{
		{name: "equal zero", a: 0, b: 0},
		{name: "equal max", a: 255, b: 255},
		{name: "positive max", a: 255, b: 0},
		{name: "negative max", a: 0, b: 255},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pixel1 := make([]byte, 4*19)
			pixel2 := make([]byte, 4*23)
			for i := range pixel1 {
				pixel1[i] = tc.a
			}
			for i := range pixel2 {
				pixel2[i] = tc.b
			}

			want := make([]int16, 16)
			got := make([]int16, 16)
			WelsDctT4_c(want, pixel1, 0, 19, pixel2, 0, 23)
			WelsDctT4_sse2(got, pixel1, 0, 19, pixel2, 0, 23)
			if !slices.Equal(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestWelsDctFourT4SSE2MatchesScalar(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(0x4264))
	for test := 0; test < 2000; test++ {
		stride1 := 8 + rng.Intn(57)
		stride2 := 8 + rng.Intn(57)
		off1 := rng.Intn(17)
		off2 := rng.Intn(17)
		pixel1 := make([]byte, off1+7*stride1+8)
		pixel2 := make([]byte, off2+7*stride2+8)
		_, _ = rng.Read(pixel1)
		_, _ = rng.Read(pixel2)

		want := make([]int16, 64)
		got := make([]int16, 64)
		WelsDctFourT4_c(want, pixel1, off1, int32(stride1), pixel2, off2, int32(stride2))
		WelsDctFourT4_sse2(got, pixel1, off1, int32(stride1), pixel2, off2, int32(stride2))
		if !slices.Equal(got, want) {
			t.Fatalf("test %d (strides %d/%d, offsets %d/%d):\n got %v\nwant %v", test, stride1, stride2, off1, off2, got, want)
		}
	}
}

var dctBenchmarkSink int16

func BenchmarkWelsDctT4(b *testing.B) {
	pixel1, pixel2 := dctBenchmarkPixels()
	dct := make([]int16, 16)
	b.SetBytes(16)

	b.Run("scalar", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			WelsDctT4_c(dct, pixel1, 3, 64, pixel2, 5, 64)
		}
		dctBenchmarkSink = dct[0]
	})
	b.Run("sse2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			WelsDctT4_sse2(dct, pixel1, 3, 64, pixel2, 5, 64)
		}
		dctBenchmarkSink = dct[0]
	})
}

func BenchmarkWelsDctFourT4(b *testing.B) {
	pixel1, pixel2 := dctBenchmarkPixels()
	dct := make([]int16, 64)
	b.SetBytes(64)

	b.Run("scalar", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			WelsDctFourT4_c(dct, pixel1, 3, 64, pixel2, 5, 64)
		}
		dctBenchmarkSink = dct[0]
	})
	b.Run("sse2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			WelsDctFourT4_sse2(dct, pixel1, 3, 64, pixel2, 5, 64)
		}
		dctBenchmarkSink = dct[0]
	})
}

func dctBenchmarkPixels() ([]byte, []byte) {
	pixel1 := make([]byte, 8*64+16)
	pixel2 := make([]byte, 8*64+16)
	for i := range pixel1 {
		pixel1[i] = byte(i*29 + 17)
		pixel2[i] = byte(i*13 + 71)
	}
	return pixel1, pixel2
}
