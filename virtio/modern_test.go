package virtio

import (
	"testing"
	"unsafe"
)

// mockModernDev is a minimal ModernDevice for exercising the transport.
type mockModernDev struct {
	features uint64
	numQ     int
	cfgLen   int
	ready    map[int]*SplitQueue
	notified []int
}

func (d *mockModernDev) DeviceFeatures() uint64                 { return d.features }
func (d *mockModernDev) NumQueues() int                         { return d.numQ }
func (d *mockModernDev) DeviceConfigLen() int                   { return d.cfgLen }
func (d *mockModernDev) ReadDeviceConfig(_ uint64, data []byte) { zero(data) }
func (d *mockModernDev) WriteDeviceConfig(_ uint64, _ []byte)   {}

func (d *mockModernDev) QueueReady(idx int, q *SplitQueue) {
	if d.ready == nil {
		d.ready = map[int]*SplitQueue{}
	}

	d.ready[idx] = q
}

func (d *mockModernDev) Notify(idx int) { d.notified = append(d.notified, idx) }

func writeCfg(tr *ModernTransport, off, v uint64, n int) {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = byte(v >> (8 * i))
	}

	tr.MMIO(commonCfgOffset+off, b, true)
}

func readCfg(tr *ModernTransport, off uint64, n int) uint64 {
	b := make([]byte, n)
	tr.MMIO(commonCfgOffset+off, b, false)

	var v uint64
	for i := 0; i < n; i++ {
		v |= uint64(b[i]) << (8 * i)
	}

	return v
}

func newTestTransport(dev ModernDevice, mem []byte, injected *int) *ModernTransport {
	return NewModernTransport(dev, mem, func() error {
		*injected++

		return nil
	})
}

func TestModernFeatureNegotiation(t *testing.T) {
	t.Parallel()

	dev := &mockModernDev{numQ: 2, cfgLen: 12}
	injected := 0
	tr := newTestTransport(dev, make([]byte, 0x10000), &injected)

	// High dword must advertise VIRTIO_F_VERSION_1 (bit 32 -> bit 0).
	writeCfg(tr, 0, 1, 4) // device_feature_select = 1
	if f := readCfg(tr, 4, 4); f&0x1 == 0 {
		t.Fatalf("VIRTIO_F_VERSION_1 not advertised: high feature dword=0x%x", f)
	}

	// Low dword is the device's own features (none for this mock).
	writeCfg(tr, 0, 0, 4)
	if f := readCfg(tr, 4, 4); f != 0 {
		t.Fatalf("low feature dword: expected 0, got 0x%x", f)
	}

	if n := readCfg(tr, 18, 2); n != 2 {
		t.Fatalf("num_queues: expected 2, got %d", n)
	}
}

func TestModernQueueSetup(t *testing.T) {
	t.Parallel()

	mem := make([]byte, 0x10000)
	dev := &mockModernDev{numQ: 2, cfgLen: 12}
	injected := 0
	tr := newTestTransport(dev, mem, &injected)

	writeCfg(tr, 22, 1, 2) // queue_select = 1

	// Default reported queue size is the max.
	if sz := readCfg(tr, 24, 2); sz != QueueSize {
		t.Fatalf("queue_size: expected %d, got %d", QueueSize, sz)
	}

	const descAddr, availAddr, usedAddr = 0x1000, 0x2000, 0x3000

	writeCfg(tr, 32, descAddr, 4) // queue_desc lo
	writeCfg(tr, 36, 0, 4)        // queue_desc hi
	writeCfg(tr, 40, availAddr, 4)
	writeCfg(tr, 44, 0, 4)
	writeCfg(tr, 48, usedAddr, 4)
	writeCfg(tr, 52, 0, 4)

	// Addresses must read back.
	if a := readCfg(tr, 32, 4); a != descAddr {
		t.Fatalf("queue_desc readback: expected 0x%x, got 0x%x", descAddr, a)
	}

	writeCfg(tr, 28, 1, 2) // queue_enable = 1

	q := dev.ready[1]
	if q == nil {
		t.Fatal("QueueReady was not called for queue 1")
	}

	if got := uintptr(unsafe.Pointer(&q.Desc[0])); got != uintptr(unsafe.Pointer(&mem[descAddr])) {
		t.Fatalf("desc table not mapped at descAddr")
	}

	if got := uintptr(unsafe.Pointer(q.Avail)); got != uintptr(unsafe.Pointer(&mem[availAddr])) {
		t.Fatalf("avail ring not mapped at availAddr")
	}

	if got := uintptr(unsafe.Pointer(q.Used)); got != uintptr(unsafe.Pointer(&mem[usedAddr])) {
		t.Fatalf("used ring not mapped at usedAddr")
	}

	if e := readCfg(tr, 28, 2); e != 1 {
		t.Fatalf("queue_enable readback: expected 1, got %d", e)
	}
}

func TestModernISRReadClears(t *testing.T) {
	t.Parallel()

	dev := &mockModernDev{numQ: 1, cfgLen: 8}
	injected := 0
	tr := newTestTransport(dev, make([]byte, 0x1000), &injected)

	if err := tr.Interrupt(); err != nil {
		t.Fatal(err)
	}

	if injected != 1 {
		t.Fatalf("inject count: expected 1, got %d", injected)
	}

	b := make([]byte, 1)
	tr.MMIO(isrCfgOffset, b, false)

	if b[0] != 1 {
		t.Fatalf("first ISR read: expected 1, got %d", b[0])
	}

	tr.MMIO(isrCfgOffset, b, false)

	if b[0] != 0 {
		t.Fatalf("ISR not cleared on read: got %d", b[0])
	}
}

func TestModernNotifyDispatch(t *testing.T) {
	t.Parallel()

	dev := &mockModernDev{numQ: 2, cfgLen: 8}
	injected := 0
	tr := newTestTransport(dev, make([]byte, 0x1000), &injected)

	tr.MMIO(notifyCfgOffset, []byte{0x1, 0x0}, true)

	if len(dev.notified) != 1 || dev.notified[0] != 1 {
		t.Fatalf("notify dispatch: expected [1], got %v", dev.notified)
	}
}

func TestModernCapabilitiesChain(t *testing.T) {
	t.Parallel()

	dev := &mockModernDev{numQ: 2, cfgLen: 12}
	injected := 0
	tr := newTestTransport(dev, make([]byte, 0x1000), &injected)

	caps := tr.Capabilities()

	for _, tc := range []struct {
		name     string
		configAt int // absolute config-space offset of the capability
		cfgType  byte
		next     byte
	}{
		{"common", capCommonAt, cfgTypeCommon, capNotifyAt},
		{"notify", capNotifyAt, cfgTypeNotify, capISRAt},
		{"isr", capISRAt, cfgTypeISR, capDeviceAt},
		{"device", capDeviceAt, cfgTypeDevice, 0x00},
	} {
		// caps starts at config offset capCommonAt (0x40).
		at := tc.configAt - capCommonAt

		if caps[at] != capVendor {
			t.Fatalf("%s: cap_vndr expected 0x%x, got 0x%x", tc.name, capVendor, caps[at])
		}

		if caps[at+3] != tc.cfgType {
			t.Fatalf("%s: cfg_type expected %d, got %d", tc.name, tc.cfgType, caps[at+3])
		}

		if caps[at+1] != tc.next {
			t.Fatalf("%s: cap_next expected 0x%x, got 0x%x", tc.name, tc.next, caps[at+1])
		}
	}
}
