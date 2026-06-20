package virtio

import (
	"encoding/binary"
	"errors"
	"log"
	"sync"
	"time"
	"unsafe"

	"github.com/bobuhiro11/gokvm/disk"
	"github.com/bobuhiro11/gokvm/pci"
)

const (
	SectorSize = 512

	blkFeatureRO = 1 << 5

	blkNumQueues = 1
	blkQueue     = 0

	// BlkMMIOBase is the guest-physical base of the block device's memory
	// BAR. It sits in the 32-bit MMIO hole, distinct from the net device's
	// region; the pci layer follows any reassignment the guest performs.
	BlkMMIOBase = 0xd001_0000
)

// LoadU16 reads a uint16 through a non-inlined function
// call, preventing the compiler from caching the value
// across iterations. This is needed for shared memory
// fields (AvailRing.Idx, UsedRing.Idx) that are written
// by KVM vCPU threads via unsafe.Pointer.
//
//go:noinline
func LoadU16(p *uint16) uint16 { return *p }

// StoreAddU16 atomically-enough increments a uint16
// through a non-inlined function call, ensuring the
// write is visible to other threads.
//
//go:noinline
func StoreAddU16(p *uint16, delta uint16) {
	*p += delta
}

// Blk is a modern (virtio 1.0) block device.
var _ pci.CapsAndMMIO = (*Blk)(nil)

var errReadOnlyBlk = errors.New("virtio-blk: write to read-only device")

type Blk struct {
	*ModernTransport

	image disk.Image

	// capacity is the device size in 512-byte sectors.
	capacity uint64
	readOnly bool

	VirtQueue    [blkNumQueues]*SplitQueue
	LastAvailIdx [blkNumQueues]uint16

	kick      chan interface{}
	done      chan struct{}
	closeOnce sync.Once

	irq         uint8
	IRQInjector IRQInjector
}

func (v *Blk) GetDeviceHeader() pci.DeviceHeader {
	return pci.DeviceHeader{
		// 0x1040 + virtio device id (2 = block). The 0x1042 id marks a
		// non-transitional device, so the driver uses the modern
		// interface and requires VIRTIO_F_VERSION_1.
		DeviceID:    0x1042,
		VendorID:    0x1AF4,
		ClassCode:   0x01, // Mass storage controller
		Subclass:    0x00, // SCSI storage controller
		HeaderType:  0,
		SubsystemID: 2, // Block Device
		// Memory space enable | bus master.
		Command: 0x6,
		// Bit 4: capabilities list present.
		Status:              0x10,
		CapabilitiesPointer: capCommonAt,
		BAR: [6]uint32{
			// BAR0: 32-bit non-prefetchable memory BAR (low nibble 0).
			uint32(BlkMMIOBase),
		},
		// https://github.com/torvalds/linux/blob/fb3b0673b7d5b477ed104949450cd511337ba3c6/drivers/pci/setup-irq.c#L30-L55
		InterruptPin: 1,
		// https://www.webopedia.com/reference/irqnumbers/
		InterruptLine: v.irq,
	}
}

// ModernDevice implementation.

// DeviceFeatures advertises device-specific block features; VIRTIO_F_VERSION_1
// is added by the transport.
func (v *Blk) DeviceFeatures() uint64 {
	if v.readOnly {
		return blkFeatureRO
	}

	return 0
}

func (v *Blk) NumQueues() int { return blkNumQueues }

// DeviceConfigLen covers struct virtio_blk_config.capacity (the only field the
// driver reads without further feature negotiation).
func (v *Blk) DeviceConfigLen() int { return 8 }

// ReadDeviceConfig serves struct virtio_blk_config: capacity (le64, in
// 512-byte sectors) at offset 0.
func (v *Blk) ReadDeviceConfig(offset uint64, data []byte) {
	var cfg [8]byte

	binary.LittleEndian.PutUint64(cfg[:], v.capacity)
	zero(data)

	if offset >= uint64(len(cfg)) {
		return
	}

	end := offset + uint64(len(data))
	if end > uint64(len(cfg)) {
		end = uint64(len(cfg))
	}

	copy(data, cfg[offset:end])
}

func (v *Blk) WriteDeviceConfig(offset uint64, data []byte) {}

func (v *Blk) QueueReady(idx int, q *SplitQueue) {
	if idx == blkQueue {
		v.VirtQueue[idx] = q
	}
}

func (v *Blk) Notify(idx int) {
	// Non-blocking kick; the IO thread also polls on a ticker.
	select {
	case v.kick <- true:
	default:
	}
}

func (v *Blk) IOThreadEntry() {
	log.Println("virtio-blk: IOThreadEntry started")

	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-v.done:
			log.Println("virtio-blk: IOThreadEntry " +
				"received done signal")

			return
		case <-v.kick:
			for v.IO() == nil {
			}

			_ = v.ReinjectIfPending()
		case <-ticker.C:
			for v.IO() == nil {
			}

			_ = v.ReinjectIfPending()
		}
	}
}

type BlkReq struct {
	Type   uint32
	_      uint32
	Sector uint64
}

func (v *Blk) IO() error {
	const sel = blkQueue

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
		descID := avail.Ring[v.LastAvailIdx[sel]%QueueSize]

		// This structure holds both the index of the descriptor
		// chain and the number of bytes written to memory as part
		// of serving the request.
		uidx := LoadU16(&used.Idx)
		used.Ring[uidx%QueueSize].ID = uint32(descID)
		used.Ring[uidx%QueueSize].Len = 0

		var buf [3][]byte

		for i := 0; i < 3; i++ {
			desc := q.Desc[descID]
			buf[i] = v.Mem[desc.Addr : desc.Addr+uint64(desc.Len)]

			used.Ring[uidx%QueueSize].Len += desc.Len
			descID = desc.Next
		}

		// buf[0] contains type, reserved, and sector.
		// buf[1] contains raw io data.
		// buf[2] contains a status field.
		//
		// refs https://wiki.osdev.org/Virtio#Block_Device_Packets
		blkReq := *(*BlkReq)(unsafe.Pointer(&buf[0][0]))
		data := buf[1]

		var ioErr error

		isWrite := blkReq.Type&0x1 == 0x1
		switch {
		case isWrite && v.readOnly:
			ioErr = errReadOnlyBlk
		case isWrite:
			_, ioErr = v.image.WriteAt(
				data,
				int64(blkReq.Sector*SectorSize),
			)

			if ioErr == nil {
				ioErr = v.image.Sync()
			}
		default:
			_, ioErr = v.image.ReadAt(
				data,
				int64(blkReq.Sector*SectorSize),
			)
		}

		// Write status byte per virtio spec.
		if ioErr != nil {
			buf[2][0] = 1 // VIRTIO_BLK_S_IOERR
		} else {
			buf[2][0] = 0 // VIRTIO_BLK_S_OK
		}

		StoreAddU16(&used.Idx, 1)
		v.LastAvailIdx[sel]++
	}

	return v.Interrupt()
}

// Read and Write satisfy pci.Device. A modern device has no IO-port BAR, so
// these are no-ops.
func (v *Blk) Read(port uint64, bytes []byte) error { return nil }

func (v *Blk) Write(port uint64, bytes []byte) error { return nil }

func (v *Blk) IOPort() uint64 { return 0 }

func (v *Blk) Size() uint64 { return 0 }

func (v *Blk) Close() error {
	log.Println("virtio-blk: Close called")
	v.closeOnce.Do(func() { close(v.done) })

	return v.image.Close()
}

func NewBlk(path string, irq uint8, irqInjector IRQInjector, mem []byte) (*Blk, error) {
	image, err := disk.Open(path)
	if err != nil {
		return nil, err
	}

	return newBlk(image, false, irq, irqInjector, mem), nil
}

func NewReadOnlyBlk(path string, irq uint8, irqInjector IRQInjector, mem []byte) (*Blk, error) {
	image, err := disk.OpenReadOnly(path)
	if err != nil {
		return nil, err
	}

	return newBlk(image, true, irq, irqInjector, mem), nil
}

func newBlk(image disk.Image, readOnly bool, irq uint8, irqInjector IRQInjector, mem []byte) *Blk {
	res := &Blk{
		image:       image,
		capacity:    uint64(image.Size()) / SectorSize,
		readOnly:    readOnly,
		irq:         irq,
		IRQInjector: irqInjector,
		kick:        make(chan interface{}, QueueSize),
		done:        make(chan struct{}),
	}

	res.ModernTransport = NewModernTransport(res, mem, func() error {
		return irqInjector.InjectVirtioBlkIRQ()
	})

	return res
}
