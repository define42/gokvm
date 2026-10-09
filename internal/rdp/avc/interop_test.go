package avc

import (
	"bytes"
	"image"
	"os/exec"
	"testing"
)

// TestFFmpegCompatibility keeps an independent decoder in CI so matching bugs
// in go.264's encoder and decoder cannot make all interoperability tests pass.
func TestFFmpegCompatibility(t *testing.T) {
	t.Parallel()
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}

	const width, height = 64, 48
	encoder, err := NewEncoder(width, height)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var stream bytes.Buffer
	expected := make([][]byte, 0, 3)
	for frame := range 3 {
		fillColorBars(img, frame)
		data, err := encoder.Encode(img, frame == 2)
		if err != nil {
			t.Fatalf("encode frame %d: %v", frame, err)
		}
		stream.Write(data)
		want := make([]byte, width*height*3/2)
		scaleI420(want, width, height, img)
		expected = append(expected, want)
	}

	command := exec.Command(ffmpeg, //nolint:gosec // Executable came from the system PATH; all arguments are fixed.
		"-hide_banner", "-loglevel", "error", "-f", "h264", "-i", "pipe:0",
		"-map", "0:v:0", "-f", "rawvideo", "-pix_fmt", "yuv420p", "pipe:1")
	command.Stdin = bytes.NewReader(stream.Bytes())
	var stderr bytes.Buffer
	command.Stderr = &stderr
	decoded, err := command.Output()
	if err != nil {
		t.Fatalf("ffmpeg decode: %v: %s", err, stderr.Bytes())
	}

	frameSize := width * height * 3 / 2
	if len(decoded) != len(expected)*frameSize {
		t.Fatalf("ffmpeg returned %d bytes, want %d", len(decoded), len(expected)*frameSize)
	}
	for frame, want := range expected {
		var difference int
		for index, value := range want {
			got := int(decoded[frame*frameSize+index])
			difference += max(int(value)-got, got-int(value))
		}
		if average := float64(difference) / float64(frameSize); average > 3 {
			t.Fatalf("frame %d average sample error = %f", frame, average)
		}
	}
}
