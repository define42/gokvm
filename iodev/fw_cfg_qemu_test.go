package iodev

import (
	"bytes"
	"testing"
)

func TestFWCfgSignatureAndDirectory(t *testing.T) {
	t.Parallel()

	dev := NewFWCfg(512<<20, 2)

	if err := dev.Write(fwCfgPortSelector, []byte{0x00, 0x00}); err != nil {
		t.Fatal(err)
	}

	sig := make([]byte, 4)
	if err := dev.Read(fwCfgPortData, sig); err != nil {
		t.Fatal(err)
	}
	if string(sig) != "QEMU" {
		t.Fatalf("signature: got %q", sig)
	}

	if err := dev.Write(fwCfgPortSelector, []byte{fwCfgFileDir, 0x00}); err != nil {
		t.Fatal(err)
	}

	dir := make([]byte, 4)
	if err := dev.Read(fwCfgPortData, dir); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dir, []byte{0, 0, 0, 0}) {
		t.Fatalf("file directory count: got %#v, want zero", dir)
	}
}
