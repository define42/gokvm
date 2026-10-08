package rdp

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func slowInput(events ...[4]uint16) []byte {
	data := make([]byte, 22+12*len(events))
	binary.LittleEndian.PutUint16(data, uint16(len(data)))
	binary.LittleEndian.PutUint16(data[2:], 0x17)
	data[14] = 28
	binary.LittleEndian.PutUint16(data[18:], uint16(len(events)))
	for i, event := range events {
		for j, value := range event {
			binary.LittleEndian.PutUint16(data[26+12*i+2*j:], value)
		}
	}

	return data
}

func TestInputPaths(t *testing.T) {
	t.Parallel()
	// Shift+A, right Control, Up, left drag, right click, and a scroll notch.
	slow := slowInput(
		[4]uint16{4, 0, 0x2a}, [4]uint16{4, 0, 0x1e},
		[4]uint16{4, 0x8000, 0x1e}, [4]uint16{4, 0x8000, 0x2a},
		[4]uint16{4, 0x100, 0x1d}, [4]uint16{4, 0x100, 0x48},
		[4]uint16{0x8001, 0x9000, 100, 80}, [4]uint16{0x8001, 0x0800, 120, 90},
		[4]uint16{0x8001, 0xa000, 120, 90}, [4]uint16{0x8001, 0x0278},
	)
	fast := []byte{
		10 << 2,
		0, 0x2a, 0, 0x1e, 1, 0x1e, 1, 0x2a, 2, 0x1d, 2, 0x48,
		0x20, 0, 0x90, 100, 0, 80, 0, 0x20, 0, 0x08, 120, 0, 90, 0,
		0x20, 0, 0xa0, 120, 0, 90, 0, 0x20, 0x78, 2, 0, 0, 0, 0,
	}
	want := []InputEvent{
		{Kind: InputKey, Down: true, Key: 0xffe1},
		{Kind: InputKey, Down: true, Key: 'a'},
		{Kind: InputKey, Key: 'a'},
		{Kind: InputKey, Key: 0xffe1},
		{Kind: InputKey, Down: true, Key: 0xffe4},
		{Kind: InputKey, Down: true, Key: 0xff52},
		{Kind: InputPointer, X: 100, Y: 80, Buttons: 1},
		{Kind: InputPointer, X: 120, Y: 90, Buttons: 1},
		{Kind: InputPointer, X: 120, Y: 90, Buttons: 5},
		{Kind: InputPointer, X: 120, Y: 90, Buttons: 13},
		{Kind: InputPointer, X: 120, Y: 90, Buttons: 5},
	}
	for _, tc := range []struct {
		name string
		data []byte
		fast bool
	}{{"slow", slow, false}, {"fast", fast, true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var decoder InputDecoder
			got, err := decoder.Decode(tc.data, tc.fast)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("Decode = %#v, %v; want %#v", got, err, want)
			}
			releases := decoder.ReleaseAll()
			wantReleases := []InputEvent{
				{Kind: InputKey, Key: 0xff52},
				{Kind: InputKey, Key: 0xffe4},
				{Kind: InputPointer, X: 120, Y: 90},
			}
			if !reflect.DeepEqual(releases, wantReleases) {
				t.Fatalf("ReleaseAll = %#v; want %#v", releases, wantReleases)
			}
			if got := decoder.ReleaseAll(); len(got) != 0 {
				t.Fatalf("second ReleaseAll = %#v", got)
			}
		})
	}
}

func TestInputWheelAccumulation(t *testing.T) {
	t.Parallel()
	var decoder InputDecoder
	for _, flags := range []uint16{0x023c, 0x023c, 0x0388} {
		events, err := decoder.Decode(slowInput([4]uint16{0x8001, flags}), false)
		if err != nil {
			t.Fatal(err)
		}
		switch flags {
		case 0x023c:
			if decoder.wheel == 0 && (len(events) != 2 || events[0].Buttons != 8 || events[1].Buttons != 0) {
				t.Fatalf("positive wheel events = %#v", events)
			}
		case 0x0388:
			if len(events) != 2 || events[0].Buttons != 16 || events[1].Buttons != 0 {
				t.Fatalf("negative wheel events = %#v", events)
			}
		}
	}
}

func TestInputUnicodeSurrogates(t *testing.T) {
	t.Parallel()
	var decoder InputDecoder
	// Down/up surrogate halves can be interleaved and cross packet boundaries.
	packets := [][]byte{
		{4, 0x80, 0x3d, 0xd8},
		{4, 0x81, 0x3d, 0xd8},
		{4, 0x80, 0x42, 0xde},
		{4, 0x81, 0x42, 0xde},
		{4, 0x80, 0x42, 0xde}, // Unpaired low surrogate is ignored.
		{4, 0x80, 0x0d, 0},
	}
	var got []InputEvent
	for _, packet := range packets {
		events, err := decoder.Decode(packet, true)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, events...)
	}
	want := []InputEvent{
		{Kind: InputKey, Down: true, Key: 0x0101f642},
		{Kind: InputKey, Key: 0x0101f642},
		{Kind: InputKey, Down: true, Key: 0xff0d},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unicode events = %#v; want %#v", got, want)
	}
}

func TestInputExtendedCountAndPause(t *testing.T) {
	t.Parallel()
	var decoder InputDecoder
	events, err := decoder.Decode([]byte{0, 5, 4, 0x1d, 0, 0x45, 5, 0x1d, 1, 0x45, 0x67}, true)
	want := []InputEvent{{Kind: InputKey, Down: true, Key: 0xff13}, {Kind: InputKey, Key: 0xff13}}
	if err != nil || !reflect.DeepEqual(events, want) {
		t.Fatalf("Pause events = %#v, %v; want %#v", events, err, want)
	}
}

func TestInputRejectsTruncatedPacketsAtomically(t *testing.T) {
	t.Parallel()
	packets := []struct {
		data []byte
		fast bool
	}{
		{slowInput([4]uint16{4, 0, 0x1e}, [4]uint16{0x8001, 0x9000, 20, 30}), false},
		{[]byte{8, 0, 0x1e, 0x20, 0, 0x90, 20, 0, 30, 0}, true},
	}
	for _, packet := range packets {
		for end := 0; end < len(packet.data); end++ {
			var decoder InputDecoder
			if _, err := decoder.Decode(packet.data[:end], packet.fast); err == nil {
				t.Fatalf("accepted truncated packet fast=%v length=%d", packet.fast, end)
			}
			if got := decoder.ReleaseAll(); len(got) != 0 {
				t.Fatalf("malformed packet modified state: %#v", got)
			}
		}
	}
}

func TestInputRejectsUnknownAndTrailingData(t *testing.T) {
	t.Parallel()
	for _, packet := range [][]byte{{4, 0xe0}, {4, 0, 0x1e, 0}, {0x84, 0, 0x1e}, {5, 0, 0x1e}} {
		var decoder InputDecoder
		if _, err := decoder.Decode(packet, true); err == nil {
			t.Errorf("accepted invalid packet %x", packet)
		}
	}
	var decoder InputDecoder
	if _, err := decoder.Decode(slowInput([4]uint16{0x1234}), false); err == nil {
		t.Error("accepted unknown slow-path input event")
	}
}

func FuzzInputDecoder(f *testing.F) {
	f.Add([]byte{4, 0, 0x1e}, true)
	f.Add(slowInput([4]uint16{0x8001, 0x9000, 100, 80}), false)
	f.Add([]byte{0, 1, 0x80, 0x3d, 0xd8}, true)
	f.Fuzz(func(t *testing.T, data []byte, fast bool) {
		var decoder InputDecoder
		_, err := decoder.Decode(data, fast)
		releases := decoder.ReleaseAll()
		if err != nil && len(releases) != 0 {
			t.Fatalf("invalid packet changed input state: %#v", releases)
		}
	})
}
