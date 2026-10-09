package avc

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"

	openh264 "github.com/define42/gokvm/pkg/h264"
)

//nolint:paralleltest,tparallel // Sequential subtests bound codec reference-picture memory.
func TestOpenH264SlicesRoundTrip(t *testing.T) {
	t.Parallel()
	// Sequential subtests bound codec reference-picture memory. Each stream
	// changes resolution through a fresh encoder, just as an RDP resize does.
	for _, threads := range []int{1, 2, 4, MaxThreads} {
		t.Run(fmt.Sprint(threads), func(t *testing.T) {
			var frames []decodedReference
			for _, size := range [][2]int{{16, 16}, {64, 48}, {200, 200}, {320, 240}, {1024, 768}} {
				frames = append(frames, slicedTestFrames(t, threads, size[0], size[1])...)
			}
			decodeSlicedFrames(t, threads, frames)
		})
	}
}

type decodedReference struct {
	width, height int
	data          []byte
	pixels        []byte
}

func decodeSlicedFrames(t *testing.T, threads int, frames []decodedReference) {
	t.Helper()
	decoder, err := openh264.NewDecoder(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	var decoded []*openh264.Frame
	for index, frame := range frames {
		picture, err := decoder.Decode(frame.data)
		if err != nil {
			t.Fatalf("decode %d-slice frame %d: %v", threads, index, err)
		}
		if picture != nil {
			decoded = append(decoded, picture)
		}
	}
	rest, err := decoder.Flush()
	if err != nil {
		t.Fatalf("flush %d-slice stream: %v", threads, err)
	}
	decoded = append(decoded, rest...)
	if len(decoded) != len(frames) {
		t.Fatalf("decode %d-slice stream returned %d pictures, want %d", threads, len(decoded), len(frames))
	}
	for index, picture := range decoded {
		frame := frames[index]
		if picture.Width != frame.width || picture.Height != frame.height {
			t.Fatalf("frame %d dimensions %dx%d, want %dx%d",
				index, picture.Width, picture.Height, frame.width, frame.height)
		}
		decoded := appendFrameI420(nil, picture)
		if len(decoded) != len(frame.pixels) {
			t.Fatalf("frame %d decoded length = %d, want %d", index, len(decoded), len(frame.pixels))
		}
		var difference int
		for i, value := range frame.pixels {
			got := int(decoded[i])
			difference += max(int(value)-got, got-int(value))
		}
		if average := float64(difference) / float64(len(frame.pixels)); average > 3 {
			t.Fatalf("frame %d mean sample error %.3f", index, average)
		}
	}
}

func slicedTestFrames(t *testing.T, threads, width, height int) []decodedReference {
	t.Helper()
	encoder, err := NewEncoderWithOptions(width, height, Options{Threads: threads, Measure: true})
	if err != nil {
		t.Fatalf("initialize %dx%d/%d slices: %v", width, height, threads, err)
	}
	defer encoder.Close()
	actual := encoder.Threads()
	if actual < 1 || actual > threads {
		t.Fatalf("%dx%d requested %d slices, initialized %d", width, height, threads, actual)
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
			t.Fatalf("%dx%d frame %d: got %d slices, want at least %d", width, height, index, slices, actual)
		}
		if index != 1 && (!hasNAL(data, 7) || !hasNAL(data, 8)) {
			t.Fatalf("frame %d lacks parameter sets", index)
		}
		stats := encoder.LastStats()
		if stats.Conversion <= 0 || stats.Encoding <= 0 {
			t.Fatalf("missing measurements: %+v", stats)
		}
		frames = append(frames, decodedReference{
			width: width, height: height, data: bytes.Clone(data),
			pixels: referenceI420(width, height, img),
		})
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
