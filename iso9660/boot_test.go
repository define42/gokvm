package iso9660

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestLoadBootFilesFromTinyCoreStyleISO(t *testing.T) {
	t.Parallel()

	const (
		rootSector     = 20
		bootSector     = 21
		isolinuxSector = 22
		kernelSector   = 23
		initrdSector   = 24
		configSector   = 25
	)

	kernel := []byte("tinycore kernel")
	initrd := []byte("tinycore initrd")
	config := []byte("DEFAULT tinycore\n" +
		"LABEL tinycore\n" +
		"KERNEL /boot/vmlinuz\n" +
		"APPEND initrd=/boot/core.gz loglevel=3 cde\n")

	isolinuxDir := joinRecords(
		dirRecord([]byte{0}, isolinuxSector, 0, true),
		dirRecord([]byte{1}, bootSector, 0, true),
		dirRecord([]byte("ISOLINUX.CFG;1"), configSector, len(config), false),
	)
	isolinuxDir = joinRecords(
		dirRecord([]byte{0}, isolinuxSector, len(isolinuxDir), true),
		dirRecord([]byte{1}, bootSector, 0, true),
		dirRecord([]byte("ISOLINUX.CFG;1"), configSector, len(config), false),
	)

	bootDir := joinRecords(
		dirRecord([]byte{0}, bootSector, 0, true),
		dirRecord([]byte{1}, rootSector, 0, true),
		dirRecord([]byte("VMLINUZ.;1"), kernelSector, len(kernel), false),
		dirRecord([]byte("CORE.GZ;1"), initrdSector, len(initrd), false),
		dirRecord([]byte("ISOLINUX"), isolinuxSector, len(isolinuxDir), true),
	)
	bootDir = joinRecords(
		dirRecord([]byte{0}, bootSector, len(bootDir), true),
		dirRecord([]byte{1}, rootSector, 0, true),
		dirRecord([]byte("VMLINUZ.;1"), kernelSector, len(kernel), false),
		dirRecord([]byte("CORE.GZ;1"), initrdSector, len(initrd), false),
		dirRecord([]byte("ISOLINUX"), isolinuxSector, len(isolinuxDir), true),
	)

	rootDir := joinRecords(
		dirRecord([]byte{0}, rootSector, 0, true),
		dirRecord([]byte{1}, rootSector, 0, true),
		dirRecord([]byte("BOOT"), bootSector, len(bootDir), true),
	)
	rootDir = joinRecords(
		dirRecord([]byte{0}, rootSector, len(rootDir), true),
		dirRecord([]byte{1}, rootSector, len(rootDir), true),
		dirRecord([]byte("BOOT"), bootSector, len(bootDir), true),
	)

	iso := make([]byte, sectorSize*32)
	writePrimaryVolumeDescriptor(iso[sectorSize*16:sectorSize*17], rootSector, len(rootDir))
	writeVolumeDescriptorTerminator(iso[sectorSize*17 : sectorSize*18])
	copySector(iso, rootSector, rootDir)
	copySector(iso, bootSector, bootDir)
	copySector(iso, isolinuxSector, isolinuxDir)
	copySector(iso, kernelSector, kernel)
	copySector(iso, initrdSector, initrd)
	copySector(iso, configSector, config)

	reader, err := NewReader(bytes.NewReader(iso), int64(len(iso)))
	if err != nil {
		t.Fatal(err)
	}

	files, err := LoadBootFiles(reader)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(files.Kernel, kernel) {
		t.Fatalf("kernel: got %q, want %q", files.Kernel, kernel)
	}

	if !bytes.Equal(files.Initrd, initrd) {
		t.Fatalf("initrd: got %q, want %q", files.Initrd, initrd)
	}

	if files.KernelPath != "/boot/vmlinuz" {
		t.Fatalf("kernel path: got %q", files.KernelPath)
	}

	if files.InitrdPath != "/boot/core.gz" {
		t.Fatalf("initrd path: got %q", files.InitrdPath)
	}

	if files.Cmdline != "loglevel=3 cde" {
		t.Fatalf("cmdline: got %q", files.Cmdline)
	}
}

func TestLoadBootFilesFromExternalISO(t *testing.T) {
	t.Parallel()

	name := os.Getenv("GOKVM_TEST_ISO")
	if name == "" {
		t.Skip("set GOKVM_TEST_ISO to an ISO image path")
	}

	file, err := os.Open(name) //nolint:gosec // Opt-in test path from GOKVM_TEST_ISO.
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	reader, err := NewReader(file, info.Size())
	if err != nil {
		t.Fatal(err)
	}

	files, err := LoadBootFiles(reader)
	if err != nil {
		t.Fatal(err)
	}

	if len(files.Kernel) == 0 {
		t.Fatal("kernel is empty")
	}

	t.Logf("kernel=%s initrd=%s cmdline=%q", files.KernelPath, files.InitrdPath, files.Cmdline)
}

func writePrimaryVolumeDescriptor(dst []byte, rootExtent, rootSize int) {
	dst[0] = 1
	copy(dst[1:6], "CD001")
	dst[6] = 1
	copy(dst[156:], dirRecord([]byte{0}, rootExtent, rootSize, true))
}

func writeVolumeDescriptorTerminator(dst []byte) {
	dst[0] = 255
	copy(dst[1:6], "CD001")
	dst[6] = 1
}

func dirRecord(name []byte, extent, size int, isDir bool) []byte {
	recLen := 33 + len(name)
	if recLen%2 != 0 {
		recLen++
	}

	rec := make([]byte, recLen)
	rec[0] = byte(recLen)
	binary.LittleEndian.PutUint32(rec[2:6], uint32(extent))
	binary.BigEndian.PutUint32(rec[6:10], uint32(extent))
	binary.LittleEndian.PutUint32(rec[10:14], uint32(size))
	binary.BigEndian.PutUint32(rec[14:18], uint32(size))
	if isDir {
		rec[25] = 0x02
	}

	rec[28] = 1
	binary.BigEndian.PutUint16(rec[30:32], 1)
	rec[32] = byte(len(name))
	copy(rec[33:], name)

	return rec
}

func joinRecords(records ...[]byte) []byte {
	var out []byte
	for _, record := range records {
		out = append(out, record...)
	}

	return out
}

func copySector(iso []byte, sector int, data []byte) {
	copy(iso[sector*sectorSize:], data)
}
