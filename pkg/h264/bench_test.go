package openh264

import (
	"os"
	"testing"
)

func BenchmarkDecode720p(b *testing.B) {
	data, err := os.ReadFile("../../res/Zhling_1280x720.264")
	if err != nil {
		b.Skip(err)
	}
	for i := 0; i < b.N; i++ {
		dec, _ := NewDecoder(nil)
		start := 0
		for j := 4; j+4 <= len(data); j++ {
			if data[j] == 0 && data[j+1] == 0 && data[j+2] == 0 && data[j+3] == 1 {
				dec.Decode(data[start:j])
				start = j
			}
		}
		dec.Decode(data[start:])
		dec.Flush()
		dec.Close()
	}
}
