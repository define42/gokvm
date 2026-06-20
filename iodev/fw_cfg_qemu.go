package iodev

import (
	"encoding/binary"
)

const (
	fwCfgPortSelector = 0x510
	fwCfgPortData     = 0x511

	fwCfgSignature  = 0x00
	fwCfgID         = 0x01
	fwCfgUUID       = 0x02
	fwCfgRAMSize    = 0x03
	fwCfgNoGraphic  = 0x04
	fwCfgNBCPUs     = 0x05
	fwCfgBootDevice = 0x0c
	fwCfgBootMenu   = 0x0e
	fwCfgMaxCPUs    = 0x0f
	fwCfgFileDir    = 0x19
)

type FWCfg struct {
	items    map[uint16][]byte
	selected uint16
	offset   int
}

func NewFWCfg(memSize uint64, ncpus uint16, bootDeviceOpt ...string) *FWCfg {
	bootDevice := "c"
	if len(bootDeviceOpt) > 0 && bootDeviceOpt[0] != "" {
		bootDevice = bootDeviceOpt[0]
	}

	le16 := func(v uint16) []byte {
		b := make([]byte, 2)
		binary.LittleEndian.PutUint16(b, v)

		return b
	}
	le32 := func(v uint32) []byte {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, v)

		return b
	}
	le64 := func(v uint64) []byte {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, v)

		return b
	}

	return &FWCfg{
		items: map[uint16][]byte{
			fwCfgSignature:  []byte("QEMU"),
			fwCfgID:         le32(1),
			fwCfgUUID:       make([]byte, 16),
			fwCfgRAMSize:    le64(memSize),
			fwCfgNoGraphic:  le16(0),
			fwCfgNBCPUs:     le16(ncpus),
			fwCfgBootDevice: []byte(bootDevice),
			fwCfgBootMenu:   le16(0),
			fwCfgMaxCPUs:    le16(ncpus),
			// File directory count is big-endian.
			fwCfgFileDir: {0, 0, 0, 0},
		},
	}
}

func (f *FWCfg) Read(port uint64, data []byte) error {
	if port != fwCfgPortData {
		for i := range data {
			data[i] = 0
		}

		return nil
	}

	item := f.items[f.selected]
	for i := range data {
		if f.offset >= len(item) {
			data[i] = 0
		} else {
			data[i] = item[f.offset]
		}
		f.offset++
	}

	return nil
}

func (f *FWCfg) Write(port uint64, data []byte) error {
	if port != fwCfgPortSelector || len(data) < 2 {
		return nil
	}

	f.selected = binary.LittleEndian.Uint16(data[:2])
	f.offset = 0

	return nil
}

func (f *FWCfg) IOPort() uint64 { return fwCfgPortSelector }

func (f *FWCfg) Size() uint64 { return 2 }
