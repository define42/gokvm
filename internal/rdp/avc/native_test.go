//go:build openh264 && cgo

package avc

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeRoundTrip(t *testing.T) {
	t.Parallel()
	if !Available() {
		t.Fatal("OpenH264 unavailable in tagged build")
	}
	// Decode in a separate native process: this checks that emitted access units
	// are independently consumable, including a P-frame and a forced refresh.
	flags, err := exec.Command("pkg-config", "--cflags", "--libs", "openh264").Output()
	if err != nil {
		t.Fatal(err)
	}
	decoder := filepath.Join(t.TempDir(), "decode")
	args := append([]string{"testdata/decode.c", "-o", decoder}, strings.Fields(string(flags))...)
	if output, err := exec.Command("cc", args...).CombinedOutput(); err != nil {
		t.Fatalf("build decoder: %v\n%s", err, output)
	}

	encoder, err := NewEncoder(64, 48)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	var stream bytes.Buffer
	var expected [][]byte
	for index := range 3 {
		fillColorBars(img, index)
		data, err := encoder.Encode(img, index == 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 || !bytes.HasPrefix(data, []byte{0, 0, 0, 1}) {
			t.Fatalf("frame %d lacks Annex-B start code: %x", index, data)
		}
		if index != 1 && (!hasNAL(data, 7) || !hasNAL(data, 8) || !hasNAL(data, 5)) {
			t.Fatalf("frame %d lacks SPS/PPS/IDR NAL units", index)
		}
		if index == 1 && !hasNAL(data, 1) {
			t.Fatal("interframe did not produce a P-frame")
		}
		stream.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(data))))
		stream.Write(data)
		want := make([]byte, 64*48*3/2)
		scaleI420(want, 64, 48, img)
		expected = append(expected, want)
	}
	command := exec.Command(decoder)
	command.Stdin = &stream
	decoded, err := command.Output()
	if err != nil {
		t.Fatalf("decode access units: %v", err)
	}
	if len(decoded) != 3*(8+64*48*3/2) {
		t.Fatalf("decoded length = %d", len(decoded))
	}
	for index, want := range expected {
		if binary.LittleEndian.Uint32(decoded) != 64 || binary.LittleEndian.Uint32(decoded[4:]) != 48 {
			t.Fatalf("frame %d dimensions = %x", index, decoded[:8])
		}
		pixels := decoded[8 : 8+len(want)]
		var difference int
		for i, value := range want {
			difference += max(int(value)-int(pixels[i]), int(pixels[i])-int(value))
		}
		if float64(difference)/float64(len(want)) > 3 {
			t.Fatalf("frame %d average sample error = %f", index, float64(difference)/float64(len(want)))
		}
		decoded = decoded[8+len(want):]
	}
}

func fillColorBars(img *image.RGBA, frame int) {
	colors := []color.RGBA{
		{R: 230, G: 30, B: 40, A: 255},
		{R: 20, G: 220, B: 40, A: 255},
		{R: 30, G: 40, B: 230, A: 255},
		{R: 240, G: 240, B: 240, A: 255},
	}
	for y := range img.Rect.Dy() {
		for x := range img.Rect.Dx() {
			value := colors[x/16]
			if x >= 16 && x < 32 && y >= 16 && y < 32 {
				value.R += byte(frame * 15)
			}
			img.SetRGBA(x, y, value)
		}
	}
}

func hasNAL(data []byte, kind byte) bool {
	for i := 0; i+4 < len(data); i++ {
		if data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 && data[i+3]&31 == kind {
			return true
		}
	}

	return false
}
