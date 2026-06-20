package pci

import (
	"bytes"
	"encoding/binary"
)

// Configuration Space Access Mechanism #1
//
// refs
// https://wiki.osdev.org/PCI
// http://www2.comp.ufscar.br/~helio/boot-int/pci.html
type address uint32

func (a address) getRegisterOffset() uint32 {
	return uint32(a) & 0xfc
}

func (a address) getFunctionNumber() uint32 {
	return (uint32(a) >> 8) & 0x7
}

func (a address) getDeviceNumber() uint32 {
	return (uint32(a) >> 11) & 0x1f
}

func (a address) getBusNumber() uint32 {
	return (uint32(a) >> 16) & 0xff
}

func (a address) isEnable() bool {
	return ((uint32(a) >> 31) | 0x1) == 0x1
}

// interface for a PCI device.
type Device interface {
	GetDeviceHeader() DeviceHeader
	Read(uint64, []byte) error
	Write(uint64, []byte) error

	// IO port range for this PCI device.
	// This range corresponds to IO Range in BAR0.
	IOPort() uint64
	Size() uint64
}

// CapsAndMMIO is implemented by PCI devices that expose a capabilities list
// and decode a memory BAR (i.e. modern virtio devices). The base address of
// the BAR is owned by the pci package; the device works purely in
// BAR-relative offsets and never needs to know where the BAR was mapped.
type CapsAndMMIO interface {
	Device

	// Capabilities returns config-space bytes starting at offset 0x40,
	// i.e. the PCI capabilities list pointed to by CapabilitiesPointer.
	Capabilities() []byte

	// MMIOBARIndex is the index (0-5) of the memory BAR carrying the
	// device's MMIO structures.
	MMIOBARIndex() int

	// MMIOSize is the size in bytes of that memory BAR.
	MMIOSize() uint64

	// MMIO services a guest memory access within the BAR. offset is
	// relative to the start of the BAR. For reads (isWrite false) the
	// handler fills data; for writes it consumes data.
	MMIO(offset uint64, data []byte, isWrite bool)
}

type DeviceHeader struct {
	VendorID uint16
	DeviceID uint16
	Command  uint16
	// Status bit 4 (0x10) advertises a PCI capabilities list; modern
	// virtio devices set it together with CapabilitiesPointer.
	Status              uint16
	RevisionID          uint8
	ProgIF              uint8
	Subclass            uint8
	ClassCode           uint8
	_                   uint8 // cacheLineSize
	_                   uint8 // latencyTimer
	HeaderType          uint8
	_                   uint8 // bist
	BAR                 [6]uint32
	_                   uint32 // cardbusCISPointer
	SubsystemVendorID   uint16
	SubsystemID         uint16
	_                   uint32 // expansionROMBaseAddress
	CapabilitiesPointer uint8  // offset of the first PCI capability
	_                   [7]uint8
	InterruptLine       uint8
	InterruptPin        uint8
	_                   uint8 // minGnt
	_                   uint8 // maxLat
}

func (h DeviceHeader) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.LittleEndian, h); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

type PCI struct {
	addr    address
	Devices []Device

	// A single size-probe can be in flight at a time: the guest writes
	// 0xffffffff to a BAR then immediately reads it back. probeSlot is -1
	// when no probe is pending.
	probeSlot int
	probeBar  int

	// barOverride records guest-assigned base addresses for the memory
	// BARs of modern (CapsAndMMIO) devices, keyed by slot then BAR index.
	barOverride map[int]map[int]uint32

	// configOverride records writable configuration-space bytes outside BARs.
	// Firmware uses this for chipset registers such as i440FX PAM shadow RAM
	// controls, and later reads expect to see the values it wrote.
	configOverride map[int]map[int]byte
}

func New(devices ...Device) *PCI {
	return &PCI{
		Devices:        devices,
		probeSlot:      -1,
		barOverride:    map[int]map[int]uint32{},
		configOverride: map[int]map[int]byte{},
	}
}

// barAtOffset returns the BAR index (0-5) addressed by a config-space byte
// offset, or -1 if the offset is not within the BAR registers (0x10-0x27).
func barAtOffset(offset int) int {
	if offset < 0x10 || offset >= 0x28 {
		return -1
	}

	return offset/4 - 4
}

// barSize returns the size of a device's BAR. Modern devices report the
// size of their memory BAR; every device reports its legacy IO BAR as BAR0.
func (p *PCI) barSize(slot, bar int) uint64 {
	if cm, ok := p.Devices[slot].(CapsAndMMIO); ok && bar == cm.MMIOBARIndex() {
		return cm.MMIOSize()
	}

	if bar == 0 {
		return p.Devices[slot].Size()
	}

	return 0
}

// barBase returns the guest-physical base address a device's BAR currently
// decodes at, with the low type bits masked off. It honors a guest-assigned
// override when present, otherwise the value baked into the device header.
func (p *PCI) barBase(slot, bar int) uint64 {
	raw := p.Devices[slot].GetDeviceHeader().BAR[bar]
	if v, ok := p.barOverride[slot][bar]; ok {
		raw = v
	}

	if raw&0x1 == 0x1 {
		return uint64(raw &^ 0x3) // IO space BAR
	}

	return uint64(raw &^ 0xf) // memory space BAR
}

// configSpace builds the 256-byte PCI configuration space image for a slot:
// the 64-byte header followed by the device's capability list, with any
// guest-assigned BAR overrides applied.
func (p *PCI) configSpace(slot int) ([]byte, error) {
	b, err := p.Devices[slot].GetDeviceHeader().Bytes()
	if err != nil {
		return nil, err
	}

	cfg := make([]byte, 256)
	copy(cfg, b)

	if cm, ok := p.Devices[slot].(CapsAndMMIO); ok {
		copy(cfg[0x40:], cm.Capabilities())
	}

	for bar, v := range p.barOverride[slot] {
		copy(cfg[0x10+bar*4:], NumToBytes(v))
	}

	for off, v := range p.configOverride[slot] {
		if off >= 0 && off < len(cfg) {
			cfg[off] = v
		}
	}

	return cfg, nil
}

// LookupMMIO finds the modern device whose memory BAR decodes addr and
// returns it together with the BAR-relative offset.
func (p *PCI) LookupMMIO(addr uint64) (CapsAndMMIO, uint64, bool) {
	for slot, dev := range p.Devices {
		cm, ok := dev.(CapsAndMMIO)
		if !ok {
			continue
		}

		base := p.barBase(slot, cm.MMIOBARIndex())
		size := cm.MMIOSize()

		if size > 0 && addr >= base && addr < base+size {
			return cm, addr - base, true
		}
	}

	return nil, 0, false
}

func (p *PCI) PciConfDataIn(port uint64, values []byte) error {
	fillAbsent := func() {
		for i := range values {
			values[i] = 0xff
		}
	}

	// offset can be obtained from many source as below:
	//        (address from IO port 0xcf8) & 0xfc + (IO port address for Data) - 0xCFC
	// see pci_conf1_read in linux/arch/x86/pci/direct.c for more detail.
	offset := int(p.addr.getRegisterOffset() + uint32(port-0xCFC))

	if !p.addr.isEnable() {
		return nil
	}

	if p.addr.getBusNumber() != 0 {
		fillAbsent()

		return nil
	}

	if p.addr.getFunctionNumber() != 0 {
		fillAbsent()

		return nil
	}

	slot := int(p.addr.getDeviceNumber())

	if slot >= len(p.Devices) {
		fillAbsent()

		return nil
	}

	// Reply to a pending BAR size probe with the size mask.
	if bar := barAtOffset(offset); bar >= 0 && p.probeSlot == slot && p.probeBar == bar {
		barOffset := 0x10 + bar*4
		mask := NumToBytes(SizeToBits(p.barSize(slot, bar)))
		copy(values, mask[offset-barOffset:])

		if offset+len(values) >= barOffset+4 {
			p.probeSlot, p.probeBar = -1, 0
		}

		return nil
	}

	b, err := p.configSpace(slot)
	if err != nil {
		return err
	}

	if offset >= len(b) {
		fillAbsent()

		return nil
	}

	end := offset + len(values)
	if end > len(b) {
		end = len(b)
	}

	copy(values, b[offset:end])
	for i := end - offset; i < len(values); i++ {
		values[i] = 0xff
	}

	return nil
}

func (p *PCI) PciConfDataOut(port uint64, values []byte) error {
	offset := int(p.addr.getRegisterOffset() + uint32(port-0xCFC))

	if !p.addr.isEnable() {
		return nil
	}

	if p.addr.getBusNumber() != 0 {
		return nil
	}

	if p.addr.getFunctionNumber() != 0 {
		return nil
	}

	slot := int(p.addr.getDeviceNumber())

	if slot >= len(p.Devices) {
		return nil
	}

	bar := barAtOffset(offset)
	if bar < 0 {
		if p.configOverride[slot] == nil {
			p.configOverride[slot] = map[int]byte{}
		}
		for i, v := range values {
			if off := offset + i; off >= 0 && off < 256 {
				p.configOverride[slot][off] = v
			}
		}

		return nil
	}
	barOffset := 0x10 + bar*4

	// 0xffffffff arms a size probe; the next read of this BAR returns the
	// size mask instead of the address.
	if offset == barOffset && len(values) == 4 && BytesToNum(values) == 0xffffffff {
		p.probeSlot, p.probeBar = slot, bar

		return nil
	}

	// Capture guest-assigned base addresses for modern devices' memory
	// BARs so reads return the assigned value and MMIO decoding follows
	// the driver. Legacy IO BARs keep their header-baked address.
	if cm, ok := p.Devices[slot].(CapsAndMMIO); ok && bar == cm.MMIOBARIndex() {
		if p.barOverride[slot] == nil {
			p.barOverride[slot] = map[int]uint32{}
		}

		cur := p.Devices[slot].GetDeviceHeader().BAR[bar]
		if v, ok := p.barOverride[slot][bar]; ok {
			cur = v
		}

		raw := NumToBytes(cur)
		copy(raw[offset-barOffset:], values)
		p.barOverride[slot][bar] = uint32(BytesToNum(raw))
	}

	return nil
}

func (p *PCI) PciConfAddrIn(port uint64, values []byte) error {
	if len(values) != 4 {
		return nil
	}

	copy(values[:4], NumToBytes(uint32(p.addr)))

	return nil
}

func (p *PCI) PciConfAddrOut(port uint64, values []byte) error {
	if len(values) != 4 {
		return nil
	}

	p.addr = address(BytesToNum(values))

	return nil
}

func SizeToBits(size uint64) uint32 {
	if size == 0 {
		return 0
	}

	return ^uint32(1) - uint32(size-2)
}

func BytesToNum(bytes []byte) uint64 {
	res := uint64(0)

	for i, x := range bytes {
		res |= uint64(x) << (i * 8)
	}

	return res
}

func NumToBytes(x interface{}) []byte {
	res := []byte{}
	l := 0
	y := uint64(0)

	switch v := x.(type) {
	case uint8:
		l = 1
		y = uint64(v)
	case uint16:
		l = 2
		y = uint64(v)
	case uint32:
		l = 4
		y = uint64(v)
	case uint64:
		l = 8
		y = v
	default:
		return []byte{}
	}

	for i := 0; i < l; i++ {
		res = append(res, uint8(y))
		y >>= 8
	}

	return res
}
