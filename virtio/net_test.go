package virtio_test

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bobuhiro11/gokvm/virtio"
)

type mockInjector struct {
	called bool
}

func (m *mockInjector) InjectVirtioNetIRQ() error {
	m.called = true

	return nil
}

func (m *mockInjector) InjectVirtioBlkIRQ() error {
	m.called = true

	return nil
}

func (m *mockInjector) InjectVirtioGPUIRQ() error {
	m.called = true

	return nil
}

// newSplitQueue builds a stand-alone split virtqueue for tests. The rings live
// in ordinary Go memory; only descriptor data addresses point into mem.
func newSplitQueue() *virtio.SplitQueue {
	return &virtio.SplitQueue{
		Desc:  &[virtio.QueueSize]virtio.SplitDesc{},
		Avail: &virtio.SplitAvail{},
		Used:  &virtio.SplitUsed{},
	}
}

func TestNetGetDeviceHeader(t *testing.T) {
	t.Parallel()

	v := virtio.NewNet(9, &mockInjector{}, bytes.NewBuffer([]byte{}), []byte{})

	// 0x1041 is the non-transitional (modern) virtio-net device id.
	if id := v.GetDeviceHeader().DeviceID; id != 0x1041 {
		t.Fatalf("DeviceID: expected 0x1041, actual 0x%x", id)
	}

	hdr := v.GetDeviceHeader()
	if hdr.ClassCode != 0x02 || hdr.Subclass != 0x00 {
		t.Fatalf("class: got %#x/%#x, want 0x02/0x00", hdr.ClassCode, hdr.Subclass)
	}

	if hdr.Status&0x10 == 0 {
		t.Fatal("capabilities-list status bit not set")
	}

	if hdr.CapabilitiesPointer != 0x40 {
		t.Fatalf("CapabilitiesPointer: expected 0x40, actual 0x%x", hdr.CapabilitiesPointer)
	}
}

func TestNetMMIOSize(t *testing.T) {
	t.Parallel()

	v := virtio.NewNet(9, &mockInjector{}, bytes.NewBuffer([]byte{}), []byte{})

	if sz := v.MMIOSize(); sz != 0x4000 {
		t.Fatalf("MMIOSize: expected 0x4000, actual 0x%x", sz)
	}

	if idx := v.MMIOBARIndex(); idx != 0 {
		t.Fatalf("MMIOBARIndex: expected 0, actual %d", idx)
	}
}

func TestTx(t *testing.T) {
	t.Parallel()

	expected := []byte{0xaa, 0xbb, 0xcc, 0xdd}
	b := bytes.NewBuffer([]byte{})

	mem := make([]byte, 0x1000000)
	v := virtio.NewNet(9, &mockInjector{}, b, mem)

	// Size of struct virtio_net_hdr_v1.
	const K = 12

	copy(mem[0x100+K:0x100+K+2], []byte{0xaa, 0xbb})
	copy(mem[0x200:0x200+2], []byte{0xcc, 0xdd})

	q := newSplitQueue()
	q.Desc[0].Addr = 0x100
	q.Desc[0].Len = K + 2
	q.Desc[0].Flags = 0x1 // VRING_DESC_F_NEXT
	q.Desc[0].Next = 0x1

	q.Desc[1].Addr = 0x200
	q.Desc[1].Len = 2

	q.Avail.Idx = 1
	v.VirtQueue[1] = q

	if err := v.Tx(); err != nil {
		t.Fatalf("err: %v\n", err)
	}

	if !v.IRQInjector.(*mockInjector).called {
		t.Fatalf("irqInjected = false\n")
	}

	if !bytes.Equal(expected, b.Bytes()) {
		t.Fatalf("expected: %v, actual: %v", expected, b.Bytes())
	}
}

func TestRx(t *testing.T) {
	t.Parallel()

	expected := []byte{0xaa, 0xbb}
	mem := make([]byte, 0x1000000)
	v := virtio.NewNet(9, &mockInjector{}, bytes.NewBuffer(expected), mem)

	q := newSplitQueue()
	q.Avail.Idx = 1
	q.Desc[0].Addr = 0x100
	q.Desc[0].Len = 0x200
	v.VirtQueue[0] = q

	// Size of struct virtio_net_hdr_v1.
	const K = 12

	if err := v.Rx(); err != nil {
		t.Fatalf("err: %v\n", err)
	}

	if !v.IRQInjector.(*mockInjector).called {
		t.Fatalf("irqInjected = false\n")
	}

	actual := mem[0x100+K : 0x100+K+2]
	if !bytes.Equal(expected, actual) {
		t.Fatalf("expected: %v, actual: %v", expected, actual)
	}

	// num_buffers (header bytes 10:12) must be 1 when MRG_RXBUF is off.
	if mem[0x100+10] != 1 || mem[0x100+11] != 0 {
		t.Fatalf("num_buffers: expected 1, actual %d",
			uint16(mem[0x100+10])|uint16(mem[0x100+11])<<8)
	}
}

func TestNetNotifyTxKick(t *testing.T) {
	t.Parallel()

	tap := &mockTapCloser{}
	mem := make([]byte, 0x10000)
	v := virtio.NewNet(9, &mockInjector{}, tap, mem)

	defer v.Close()

	// Notifying the TX queue twice must never block.
	for i := 0; i < 2; i++ {
		done := make(chan struct{})

		go func() {
			v.Notify(1) // TX
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatalf("Notify(TX) #%d blocked", i)
		}
	}
}

func TestNetNotifyRxDropped(t *testing.T) {
	t.Parallel()

	tap := &mockTapCloser{}
	mem := make([]byte, 0x10000)
	v := virtio.NewNet(9, &mockInjector{}, tap, mem)

	defer v.Close()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		v.TxThreadEntry()
	}()

	// Notifying the RX queue must be silently dropped, never reaching the
	// TX path.
	v.Notify(0) // RX

	time.Sleep(50 * time.Millisecond)

	v.Close()
	wg.Wait()

	if tap.Len() != 0 {
		t.Fatalf("tap had %d bytes; want 0", tap.Len())
	}
}

// mockTapCloser implements io.ReadWriteCloser for testing Net.Close().
type mockTapCloser struct {
	bytes.Buffer
	closed bool
}

func (m *mockTapCloser) Close() error {
	m.closed = true

	return nil
}

func TestNetClose(t *testing.T) {
	t.Parallel()

	tap := &mockTapCloser{}
	v := virtio.NewNet(9, &mockInjector{}, tap, []byte{})

	if err := v.Close(); err != nil {
		t.Fatalf("Close: got %v, want nil", err)
	}

	if !tap.closed {
		t.Fatal("tap was not closed")
	}
}

func TestNetCloseNonCloser(t *testing.T) {
	t.Parallel()

	// Use a plain io.ReadWriter (no Close method).
	var buf bytes.Buffer
	v := virtio.NewNet(9, &mockInjector{}, io.ReadWriter(&buf), []byte{})

	if err := v.Close(); err != nil {
		t.Fatalf("Close: got %v, want nil", err)
	}
}

func TestNetThreadsExitOnClose(t *testing.T) {
	t.Parallel()

	tap := &mockTapCloser{}
	mem := make([]byte, 0x10000)
	v := virtio.NewNet(9, &mockInjector{}, tap, mem)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		v.TxThreadEntry()
	}()

	go func() {
		defer wg.Done()
		v.RxThreadEntry()
	}()

	if err := v.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("thread entries did not exit after Close")
	}
}

func TestNetNotifyAfterClose(t *testing.T) {
	t.Parallel()

	tap := &mockTapCloser{}
	mem := make([]byte, 0x10000)
	v := virtio.NewNet(9, &mockInjector{}, tap, mem)

	if err := v.Close(); err != nil {
		t.Fatal(err)
	}

	// TX kick after Close must not panic.
	v.Notify(1)
}

func TestNetConcurrentCloseAndNotify(t *testing.T) {
	t.Parallel()

	tap := &mockTapCloser{}
	mem := make([]byte, 0x10000)
	v := virtio.NewNet(9, &mockInjector{}, tap, mem)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		v.Close()
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < 100; i++ {
			v.Notify(1)
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent Close+Notify deadlocked")
	}
}
