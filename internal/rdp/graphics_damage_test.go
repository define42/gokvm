package rdp

import (
	"bytes"
	"encoding/binary"
	"image"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestGraphicsAVC420DamageMetadata(t *testing.T) {
	t.Parallel()
	annexB := []byte{0, 0, 0, 1, 0x41, 0x37, 0x53}
	regions := []image.Rectangle{image.Rect(32, 48, 64, 80), image.Rect(140, 150, 160, 180)}
	for _, size := range []image.Point{image.Pt(640, 480), image.Pt(202, 206)} {
		frame := avc420FrameDamage(size.X, size.Y, 37, 53, annexB, regions)
		decoded, payload := decodeAVC420DamageFixture(t, frame, size.X, size.Y, 37)
		if !slices.Equal(decoded, regions) || !bytes.Equal(payload, annexB) {
			t.Fatalf("metadata = %v, payload = %x", decoded, payload)
		}
	}
}

func TestGraphicsAVC420DamageBounds(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		regions []image.Rectangle
		want    []image.Rectangle
	}{
		{"nil", nil, []image.Rectangle{image.Rect(0, 0, 202, 206)}},
		{"empty", []image.Rectangle{}, []image.Rectangle{image.Rect(0, 0, 202, 206)}},
		{"outside", []image.Rectangle{image.Rect(500, 500, 600, 600)}, []image.Rectangle{image.Rect(0, 0, 202, 206)}},
		{"zero-area", []image.Rectangle{image.Rect(20, 30, 20, 50)}, []image.Rectangle{image.Rect(0, 0, 202, 206)}},
		{"odd-edge", []image.Rectangle{image.Rect(35, 51, 39, 57)}, []image.Rectangle{image.Rect(32, 48, 48, 64)}},
		{"block-edge", []image.Rectangle{image.Rect(32, 32, 64, 64)}, []image.Rectangle{image.Rect(16, 16, 80, 80)}},
		{"negative", []image.Rectangle{image.Rect(-100, -100, 2, 4)}, []image.Rectangle{image.Rect(0, 0, 16, 16)}},
		{
			"right-bottom",
			[]image.Rectangle{image.Rect(200, 204, 300, 400)},
			[]image.Rectangle{image.Rect(192, 192, 202, 206)},
		},
		{"whole", []image.Rectangle{image.Rect(-100, -100, 500, 500)}, []image.Rectangle{image.Rect(0, 0, 202, 206)}},
		{"too-many", make([]image.Rectangle, graphicsMaxRegions+1), []image.Rectangle{image.Rect(0, 0, 202, 206)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := avc420Regions(202, 206, test.regions); !slices.Equal(got, test.want) {
				t.Fatalf("regions = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGraphicsAVC420DamageKeyframesAndPacing(t *testing.T) {
	t.Parallel()
	conn := &graphicsTestConn{}
	session := &Session{
		Width: 640, Height: 480, conn: conn,
		graphics: &graphicsState{ready: true}, dynamic: graphicsDynamicFixture(3),
	}
	regions := []image.Rectangle{image.Rect(35, 51, 39, 57), image.Rect(401, 301, 403, 307)}
	predicted := []byte{0, 0, 1, 0x41, 0x55}
	if sent, err := session.WriteAVC420Damage(predicted, regions); err != nil || !sent {
		t.Fatalf("send partial frame: %t, %v", sent, err)
	}
	messages := graphicsOutputFixture(t, conn.outgoing.Bytes())
	if len(messages) != 1 {
		t.Fatalf("messages = %d", len(messages))
	}
	got, payload := decodeAVC420DamageFixture(t, decodeZGFXFixture(t, messages[0][2:]), 640, 480, 1)
	want := []image.Rectangle{image.Rect(32, 48, 48, 64), image.Rect(400, 288, 416, 320)}
	if !slices.Equal(got, want) || !bytes.Equal(payload, predicted) {
		t.Fatalf("partial frame regions = %v, payload = %x", got, payload)
	}
	conn.outgoing.Reset()
	keyframe := []byte{0, 0, 0, 1, 0x67, 0x42, 0, 0, 1, 0x65, 0x11}
	if sent, err := session.WriteAVC420Damage(keyframe, regions); err != nil || !sent {
		t.Fatalf("send keyframe: %t, %v", sent, err)
	}
	messages = graphicsOutputFixture(t, conn.outgoing.Bytes())
	got, payload = decodeAVC420DamageFixture(t, decodeZGFXFixture(t, messages[0][2:]), 640, 480, 2)
	if !slices.Equal(got, []image.Rectangle{image.Rect(0, 0, 640, 480)}) || !bytes.Equal(payload, keyframe) {
		t.Fatalf("keyframe regions = %v, payload = %x", got, payload)
	}
	conn.outgoing.Reset()
	if sent, err := session.WriteAVC420Damage(predicted, regions); err != nil || sent || conn.outgoing.Len() != 0 {
		t.Fatalf("over-window damage frame = %t, %v", sent, err)
	}
	ack := make([]byte, 12)
	binary.LittleEndian.PutUint32(ack[4:], 1)
	if err := session.readGraphicsPDU(0x0d, ack); err != nil {
		t.Fatal(err)
	}
	if sent, err := session.WriteAVC420Damage(predicted, []image.Rectangle{}); err != nil || !sent {
		t.Fatalf("empty repaint mask dropped encoded interframe: %t, %v", sent, err)
	}
	messages = graphicsOutputFixture(t, conn.outgoing.Bytes())
	got, _ = decodeAVC420DamageFixture(t, decodeZGFXFixture(t, messages[0][2:]), 640, 480, 3)
	if !slices.Equal(got, []image.Rectangle{image.Rect(0, 0, 640, 480)}) {
		t.Fatalf("empty-mask fallback = %v", got)
	}
}

func TestGraphicsAVC420DamageMessageLimit(t *testing.T) {
	t.Parallel()
	regions := make([]image.Rectangle, graphicsMaxRegions)
	for index := range regions {
		x, y := 35+index%16*32, 35+index/16*32
		regions[index] = image.Rect(x, y, x+2, y+2)
	}
	normalized := avc420Regions(640, 480, regions)
	if len(normalized) != graphicsMaxRegions {
		t.Fatalf("region limit changed: %d", len(normalized))
	}
	annexB := make([]byte, avc420MaxPayload(len(normalized))+1)
	frame := avc420FrameDamage(640, 480, 1, 0, annexB[:len(annexB)-1], normalized)
	if len(segmentGraphics(frame)) > graphicsMaxMessage {
		t.Fatal("maximum accepted frame exceeds the complete DVC message limit")
	}
	conn := &graphicsTestConn{}
	session := &Session{
		Width: 640, Height: 480, conn: conn,
		graphics: &graphicsState{ready: true}, dynamic: graphicsDynamicFixture(3),
	}
	if sent, err := session.WriteAVC420Damage(annexB, regions); err == nil || sent {
		t.Fatalf("oversized metadata+payload accepted: %t, %v", sent, err)
	}
	if conn.outgoing.Len() != 0 || session.graphics.frameID != 0 || len(session.graphics.inFlight) != 0 {
		t.Fatal("rejected frame changed sender state")
	}
	if sent, err := session.WriteAVC420Damage(nil, nil); err == nil || sent {
		t.Fatalf("empty encoded frame accepted: %t, %v", sent, err)
	}
}

func TestGraphicsAVC420DamageDoesNotOverlap(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		regions []image.Rectangle
		want    []image.Rectangle
	}{
		{
			"adjacent-before-expansion",
			[]image.Rectangle{image.Rect(0, 64, 128, 128), image.Rect(64, 128, 192, 192)},
			[]image.Rectangle{image.Rect(0, 48, 208, 208)},
		},
		{
			"duplicate-and-contained",
			[]image.Rectangle{image.Rect(32, 32, 96, 96), image.Rect(32, 32, 96, 96), image.Rect(48, 48, 64, 64)},
			[]image.Rectangle{image.Rect(16, 16, 112, 112)},
		},
		{
			// The third region merges with the second; its enlarged bounding
			// box then overlaps the already-visited first region.
			"unsorted-transitive-merge",
			[]image.Rectangle{image.Rect(80, 80, 96, 96), image.Rect(0, 0, 48, 160), image.Rect(32, 0, 160, 48)},
			[]image.Rectangle{image.Rect(0, 0, 176, 176)},
		},
		{
			"touching-edges-are-disjoint",
			[]image.Rectangle{image.Rect(18, 18, 30, 30), image.Rect(34, 18, 46, 30)},
			[]image.Rectangle{image.Rect(16, 16, 32, 32), image.Rect(32, 16, 48, 32)},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			conn := &graphicsTestConn{}
			session := &Session{
				Width: 640, Height: 480, conn: conn,
				graphics: &graphicsState{ready: true}, dynamic: graphicsDynamicFixture(3),
			}
			payload := []byte{0, 0, 1, 0x41, 0x55}
			if sent, err := session.WriteAVC420Damage(payload, test.regions); err != nil || !sent {
				t.Fatalf("send damage: %t, %v", sent, err)
			}
			messages := graphicsOutputFixture(t, conn.outgoing.Bytes())
			got, encoded := decodeAVC420DamageFixture(t, decodeZGFXFixture(t, messages[0][2:]), 640, 480, 1)
			if !slices.Equal(got, test.want) || !bytes.Equal(encoded, payload) {
				t.Fatalf("wire damage=%v want=%v payload=%x", got, test.want, encoded)
			}
			checkAVCDamageCoverage(t, 640, 480, test.regions, got)
		})
	}
}

func TestGraphicsAVC420DamageRandomCoverage(t *testing.T) {
	t.Parallel()
	random := rand.New(rand.NewPCG(17, 31)) //nolint:gosec // Deterministic test geometry, not security randomness.
	for range 200 {
		regions := make([]image.Rectangle, 1+random.IntN(graphicsMaxRegions))
		for index := range regions {
			x, y := random.IntN(400)-50, random.IntN(300)-50
			regions[index] = image.Rect(x, y, x+random.IntN(80), y+random.IntN(80))
		}
		checkAVCDamageCoverage(t, 302, 206, regions, avc420Regions(302, 206, regions))
	}
}

func checkAVCDamageCoverage(t *testing.T, width, height int, input, output []image.Rectangle) {
	t.Helper()
	bounds := image.Rect(0, 0, width, height)
	if len(output) == 0 || len(output) > graphicsMaxRegions {
		t.Fatalf("invalid output region count: %d", len(output))
	}
	for i, rect := range output {
		if rect.Empty() || !rect.In(bounds) {
			t.Fatalf("invalid output rectangle: %v", rect)
		}
		for _, other := range output[i+1:] {
			if rect.Overlaps(other) {
				t.Fatalf("FreeRDP would skip %v because it overlaps %v", rect, other)
			}
		}
	}
	for _, rect := range input {
		rect = rect.Intersect(bounds)
		if rect.Empty() {
			continue
		}
		covered := false
		for _, candidate := range output {
			if rect.In(candidate) {
				covered = true

				break
			}
		}
		if !covered {
			t.Fatalf("dirty pixels lost: %v absent from %v", rect, output)
		}
	}
}

func TestGraphicsAVC420KeyframeDetection(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		data []byte
		want bool
	}{
		{nil, false},
		{[]byte{0, 0, 1}, false},
		{[]byte{0, 0, 0, 1, 0x65}, true},
		{[]byte{0, 0, 1, 0x65}, true},
		{[]byte{0, 0, 1, 0x41, 0, 0, 3, 1, 0x65}, false},
		{[]byte{0, 0, 1, 0x67, 0x11, 0, 0, 0, 1, 0x65}, true},
	} {
		if got := avc420Keyframe(test.data); got != test.want {
			t.Fatalf("keyframe %x = %t, want %t", test.data, got, test.want)
		}
	}
}

// Decode the metadata independently in the order used by FreeRDP's
// rdpgfx_read_h264_metablock: count, all rectangles, then all quality pairs.
func decodeAVC420DamageFixture(
	t *testing.T, frame []byte, width, height int, frameID uint32,
) ([]image.Rectangle, []byte) {
	t.Helper()
	if len(frame) < 67 || binary.LittleEndian.Uint16(frame) != 0x0b ||
		binary.LittleEndian.Uint32(frame[4:]) != 16 || binary.LittleEndian.Uint32(frame[12:]) != frameID {
		t.Fatal("invalid START_FRAME")
	}
	data := frame[16:]
	size := int(binary.LittleEndian.Uint32(data[4:]))
	if size < 39 || size > len(data)-12 || binary.LittleEndian.Uint16(data) != 1 {
		t.Fatal("invalid WIRE_TO_SURFACE_1")
	}
	wire := data[8:size]
	if binary.LittleEndian.Uint16(wire[2:]) != 0x0b || wire[4] != 0x20 ||
		int(binary.LittleEndian.Uint16(wire[9:])) != width || int(binary.LittleEndian.Uint16(wire[11:])) != height ||
		int(binary.LittleEndian.Uint32(wire[13:])) != len(wire)-17 {
		t.Fatal("invalid surface geometry or bitmapDataLength")
	}
	count := int(binary.LittleEndian.Uint32(wire[17:]))
	if count < 1 || count > graphicsMaxRegions || len(wire) <= 21+count*10 {
		t.Fatalf("invalid region count %d or truncated payload", count)
	}
	regions := make([]image.Rectangle, count)
	for index := range regions {
		offset := 21 + index*8
		regions[index] = image.Rect(int(binary.LittleEndian.Uint16(wire[offset:])),
			int(binary.LittleEndian.Uint16(wire[offset+2:])), int(binary.LittleEndian.Uint16(wire[offset+4:])),
			int(binary.LittleEndian.Uint16(wire[offset+6:])))
		quality := wire[21+count*8+index*2:]
		if quality[0] != 22 || quality[1] != 100 {
			t.Fatalf("quality entry %d = %x", index, quality[:2])
		}
	}
	end := data[size:]
	if len(end) != 12 || binary.LittleEndian.Uint16(end) != 0x0c || binary.LittleEndian.Uint32(end[8:]) != frameID {
		t.Fatal("invalid END_FRAME")
	}

	return regions, wire[21+count*10:]
}
