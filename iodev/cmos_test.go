package iodev

import (
	"encoding/binary"
	"testing"
)

func TestCMOSReportsConfiguredMemoryBelow4G(t *testing.T) {
	t.Parallel()

	cmos := NewCMOS(512*mib, 0)

	if got, want := binary.LittleEndian.Uint16(cmos.Data[0x34:0x36]), uint16(0x1f00); got != want {
		t.Fatalf("CMOS 0x34 memory = %#x, want %#x", got, want)
	}
}

func TestToBCD(t *testing.T) {
	t.Parallel()

	if got, want := toBCD(42), uint8(0x42); got != want {
		t.Fatalf("toBCD(42) = %#x, want %#x", got, want)
	}
}
