//go:build openh264 && cgo

package avc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

//nolint:paralleltest,tparallel // Sequential subtests bound native decoder/reference-picture memory.
func TestNativeThreadedRoundTrip(t *testing.T) {
	t.Parallel()
	flags, err := exec.Command("pkg-config", "--cflags", "--libs", "openh264").Output()
	if err != nil {
		t.Fatal(err)
	}
	decoder := filepath.Join(t.TempDir(), "decode")
	args := append([]string{"testdata/decode.c", "-o", decoder}, strings.Fields(string(flags))...)
	if output, err := exec.Command("cc", args...).CombinedOutput(); err != nil {
		t.Fatalf("build decoder: %v\n%s", err, output)
	}
	// Sequential subtests bound native reference-picture memory. Each stream
	// changes resolution through a fresh encoder, just as an RDP resize does.
	for _, threads := range []int{1, 2, 4, MaxThreads} {
		t.Run(fmt.Sprint(threads), func(t *testing.T) {
			var stream bytes.Buffer
			var frames []decodedReference
			for _, size := range [][2]int{{16, 16}, {64, 48}, {200, 200}, {320, 240}, {1024, 768}} {
				frames = append(frames, threadedTestFrames(t, &stream, threads, size[0], size[1])...)
			}
			command := exec.Command(decoder)
			command.Stdin = &stream
			decoded, err := command.Output()
			if err != nil {
				t.Fatalf("decode %d-worker stream: %v", threads, err)
			}
			for index, frame := range frames {
				if len(decoded) < 8+len(frame.pixels) {
					t.Fatalf("frame %d truncated at %d bytes", index, len(decoded))
				}
				width, height := binary.LittleEndian.Uint32(decoded), binary.LittleEndian.Uint32(decoded[4:])
				if width != uint32(frame.width) || height != uint32(frame.height) {
					t.Fatalf("frame %d dimensions %dx%d, want %dx%d", index, width, height, frame.width, frame.height)
				}
				var difference int
				for i, value := range frame.pixels {
					got := int(decoded[8+i])
					difference += max(int(value)-got, got-int(value))
				}
				if average := float64(difference) / float64(len(frame.pixels)); average > 3 {
					t.Fatalf("frame %d mean sample error %.3f", index, average)
				}
				decoded = decoded[8+len(frame.pixels):]
			}
			if len(decoded) != 0 {
				t.Fatalf("unexpected trailing output: %d bytes", len(decoded))
			}
		})
	}
}

type decodedReference struct {
	width, height int
	pixels        []byte
}

func threadedTestFrames(t *testing.T, stream *bytes.Buffer, threads, width, height int) []decodedReference {
	t.Helper()
	encoder, err := NewEncoderWithOptions(width, height, Options{Threads: threads, Measure: true})
	if err != nil {
		t.Fatalf("initialize %dx%d/%d workers: %v", width, height, threads, err)
	}
	defer encoder.Close()
	actual := encoder.Threads()
	if actual < 1 || actual > threads {
		t.Fatalf("%dx%d requested %d workers, initialized %d", width, height, threads, actual)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Rect, image.NewUniform(color.RGBA{R: 210, G: 225, B: 230, A: 255}), image.Point{}, draw.Src)
	patch := image.Rect(width/4, height/4, width/2, height/2)
	var frames []decodedReference
	for index := range 3 {
		shade := color.RGBA{R: 40, G: 70, B: byte(100 + index*30), A: 255}
		draw.Draw(img, patch, image.NewUniform(shade), image.Point{}, draw.Src)
		data, err := encoder.EncodeDamage(img, []image.Rectangle{patch}, index == 2)
		if err != nil {
			t.Fatal(err)
		}
		kind := byte(5)
		if index == 1 {
			kind = 1
		}
		if slices := countNAL(data, kind); slices < actual {
			t.Fatalf("%dx%d frame %d: got %d slices for %d workers", width, height, index, slices, actual)
		}
		if index != 1 && (!hasNAL(data, 7) || !hasNAL(data, 8)) {
			t.Fatalf("frame %d lacks parameter sets", index)
		}
		stats := encoder.LastStats()
		if stats.Conversion <= 0 || stats.Encoding <= 0 {
			t.Fatalf("missing measurements: %+v", stats)
		}
		stream.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(data))))
		stream.Write(data)
		frames = append(frames, decodedReference{width, height, referenceI420(width, height, img)})
	}

	return frames
}

func countNAL(data []byte, kind byte) int {
	count := 0
	for i := 0; i+4 < len(data); i++ {
		if data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 && data[i+3]&31 == kind {
			count++
		}
	}

	return count
}
