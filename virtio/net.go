package virtio

import (
	"errors"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bobuhiro11/gokvm/pci"
)

var (
	ErrIONotPermit = errors.New("IO is not permitted for virtio device")
	ErrNoTxPacket  = errors.New("no packet for tx")
	ErrNoRxPacket  = errors.New("no packet for rx")
	ErrVQNotInit   = errors.New("vq not initialized")
	ErrNoRxBuf     = errors.New("no buffer found for rx")
)

const (
	// netHdrLen is sizeof(struct virtio_net_hdr_v1). Under
	// VIRTIO_F_VERSION_1 the header is always 12 bytes (it carries
	// num_buffers unconditionally), unlike the 10-byte legacy header.
	netHdrLen = 12

	// Virtqueue indices.
	netRxQueue   = 0
	netTxQueue   = 1
	netNumQueues = 2

	// NetMMIOBase is the guest-physical base of the net device's memory
	// BAR. It sits in the 32-bit MMIO hole above the (small) guest RAM;
	// the pci layer follows any reassignment the guest performs.
	NetMMIOBase = 0xd000_0000
)

// Net is a modern PCI device exposing capabilities and a memory BAR.
var _ pci.CapsAndMMIO = (*Net)(nil)

// Net is a modern (virtio 1.0) network device.
type Net struct {
	*ModernTransport

	tap io.ReadWriter

	// VirtQueue holds the split virtqueues once the driver enables them.
	VirtQueue    [netNumQueues]*SplitQueue
	LastAvailIdx [netNumQueues]uint16

	txKick    chan interface{}
	rxKick    chan os.Signal
	done      chan struct{}
	closeOnce sync.Once

	irq         uint8
	IRQInjector IRQInjector
}

func (v *Net) GetDeviceHeader() pci.DeviceHeader {
	return pci.DeviceHeader{
		// 0x1040 + virtio device id (1 = net). The 0x1041 id marks a
		// non-transitional device, so the driver uses the modern
		// interface and requires VIRTIO_F_VERSION_1.
		DeviceID:    0x1041,
		VendorID:    0x1AF4,
		ClassCode:   0x02, // Network controller
		Subclass:    0x00, // Ethernet controller
		HeaderType:  0,
		SubsystemID: 1, // Network Card
		// Memory space enable | bus master.
		Command: 0x6,
		// Bit 4: capabilities list present.
		Status:              0x10,
		CapabilitiesPointer: capCommonAt,
		BAR: [6]uint32{
			// BAR0: 32-bit non-prefetchable memory BAR (low nibble 0).
			uint32(NetMMIOBase),
		},
		// https://github.com/torvalds/linux/blob/fb3b0673b7d5b477ed104949450cd511337ba3c6/drivers/pci/setup-irq.c#L30-L55
		InterruptPin: 1,
		// https://www.webopedia.com/reference/irqnumbers/
		InterruptLine: v.irq,
	}
}

// ModernDevice implementation.

// DeviceFeatures advertises no device-specific features; the guest assigns a
// random MAC and assumes the link is always up. VIRTIO_F_VERSION_1 is added by
// the transport.
func (v *Net) DeviceFeatures() uint64 { return 0 }

func (v *Net) NumQueues() int { return netNumQueues }

// DeviceConfigLen covers struct virtio_net_config (mac, status,
// max_virtqueue_pairs). The fields are unused since no features expose them,
// but the region is still described by the device-cfg capability.
func (v *Net) DeviceConfigLen() int { return 12 }

func (v *Net) ReadDeviceConfig(offset uint64, data []byte) { zero(data) }

func (v *Net) WriteDeviceConfig(offset uint64, data []byte) {}

func (v *Net) QueueReady(idx int, q *SplitQueue) {
	if idx < 0 || idx >= netNumQueues {
		return
	}

	v.VirtQueue[idx] = q
}

func (v *Net) Notify(idx int) {
	switch idx {
	case netRxQueue:
		// RX queue kick: silently drop. RX is driven by SIGIO.
	case netTxQueue:
		// TX queue kick: non-blocking send.
		select {
		case v.txKick <- true:
		default:
		}
	default:
		log.Printf("virtio-net: unexpected queue %d", idx)
	}
}

func (v *Net) RxThreadEntry() {
	log.Println("virtio-net: RxThreadEntry started")

	for {
		select {
		case <-v.done:
			log.Println("virtio-net: RxThreadEntry " +
				"received done signal")

			return
		case <-v.rxKick:
			for v.Rx() == nil {
			}
		}
	}
}

func (v *Net) Rx() error {
	// read raw packet from tap device
	packet := make([]byte, 4096)

	n, err := v.tap.Read(packet)
	if err != nil {
		return ErrNoRxPacket
	}

	// Prepend struct virtio_net_hdr_v1. With VIRTIO_NET_F_MRG_RXBUF not
	// negotiated, num_buffers (bytes 10:12) must be 1.
	frame := make([]byte, netHdrLen+n)
	frame[10] = 1
	copy(frame[netHdrLen:], packet[:n])
	packet = frame

	const sel = netRxQueue

	q := v.VirtQueue[sel]
	if q == nil {
		return ErrVQNotInit
	}

	avail := q.Avail
	used := q.Used

	if v.LastAvailIdx[sel] == LoadU16(&avail.Idx) {
		return ErrNoRxBuf
	}

	const NONE = uint16(256)
	headDescID := NONE
	prevDescID := NONE
	uidx := LoadU16(&used.Idx)

	for len(packet) > 0 {
		descID := avail.Ring[v.LastAvailIdx[sel]%QueueSize]

		// head of vring chain
		if headDescID == NONE {
			headDescID = descID

			// This structure is holding both the
			// index of the descriptor chain and the
			// number of bytes that were written to
			// memory as part of serving the request.
			used.Ring[uidx%QueueSize].ID = uint32(headDescID)
			used.Ring[uidx%QueueSize].Len = 0
		}

		desc := &q.Desc[descID]
		l := uint32(len(packet))

		if l > desc.Len {
			l = desc.Len
		}

		copy(v.Mem[desc.Addr:desc.Addr+uint64(l)], packet[:l])

		packet = packet[l:]
		desc.Len = l

		used.Ring[uidx%QueueSize].Len += l

		if prevDescID != NONE {
			q.Desc[prevDescID].Flags |= descFNext
			q.Desc[prevDescID].Next = descID
		}

		prevDescID = descID
		v.LastAvailIdx[sel]++
	}

	StoreAddU16(&used.Idx, 1)

	return v.Interrupt()
}

func (v *Net) TxThreadEntry() {
	log.Println("virtio-net: TxThreadEntry started")

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-v.done:
			log.Println("virtio-net: TxThreadEntry " +
				"received done signal")

			return
		case <-v.txKick:
			for v.Tx() == nil {
			}

			_ = v.ReinjectIfPending()
		case <-ticker.C:
			for v.Tx() == nil {
			}

			_ = v.ReinjectIfPending()
		}
	}
}

func (v *Net) Tx() error {
	const sel = netTxQueue

	q := v.VirtQueue[sel]
	if q == nil {
		return ErrVQNotInit
	}

	avail := q.Avail
	used := q.Used

	if v.LastAvailIdx[sel] == LoadU16(&avail.Idx) {
		return ErrNoTxPacket
	}

	for v.LastAvailIdx[sel] != LoadU16(&avail.Idx) {
		buf := []byte{}
		descID := avail.Ring[v.LastAvailIdx[sel]%QueueSize]

		uidx := LoadU16(&used.Idx)
		used.Ring[uidx%QueueSize].ID = uint32(descID)
		used.Ring[uidx%QueueSize].Len = 0

		for {
			desc := q.Desc[descID]

			b := make([]byte, desc.Len)
			copy(b, v.Mem[desc.Addr:desc.Addr+uint64(desc.Len)])

			buf = append(buf, b...)

			used.Ring[uidx%QueueSize].Len += desc.Len

			if desc.Flags&descFNext != 0 {
				descID = desc.Next
			} else {
				break
			}
		}

		// Skip struct virtio_net_hdr_v1.
		// refs https://github.com/torvalds/linux/blob/38f80f42/include/uapi/linux/virtio_net.h#L178-L191
		buf = buf[netHdrLen:]

		if _, err := v.tap.Write(buf); err != nil {
			return err
		}

		StoreAddU16(&used.Idx, 1)
		v.LastAvailIdx[sel]++
	}

	return v.Interrupt()
}

// Read and Write satisfy pci.Device. A modern device has no IO-port BAR, so
// these are no-ops.
func (v *Net) Read(port uint64, bytes []byte) error { return nil }

func (v *Net) Write(port uint64, bytes []byte) error { return nil }

func (v *Net) IOPort() uint64 { return 0 }

func (v *Net) Size() uint64 { return 0 }

func (v *Net) Close() error {
	log.Println("virtio-net: Close called")
	signal.Stop(v.rxKick)

	v.closeOnce.Do(func() { close(v.done) })

	if c, ok := v.tap.(io.Closer); ok {
		return c.Close()
	}

	return nil
}

func NewNet(irq uint8, irqInjector IRQInjector, tap io.ReadWriter, mem []byte) *Net {
	res := &Net{
		irq:         irq,
		IRQInjector: irqInjector,
		txKick:      make(chan interface{}, QueueSize),
		rxKick:      make(chan os.Signal, 1),
		done:        make(chan struct{}),
		tap:         tap,
	}

	res.ModernTransport = NewModernTransport(res, mem, func() error {
		return irqInjector.InjectVirtioNetIRQ()
	})

	signal.Notify(res.rxKick, syscall.SIGIO)

	return res
}
