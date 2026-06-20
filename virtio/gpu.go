package virtio

import (
	"encoding/binary"
	"errors"
	"image"
	"log"
	"sync"
	"time"

	"github.com/bobuhiro11/gokvm/pci"
)

// virtio-gpu is a 2D display device on the modern (virtio 1.0) PCI transport.
//
// The driver drives two virtqueues:
//
//	controlq (0) : resource lifecycle and 2D blits (the commands below)
//	cursorq  (1) : hardware cursor updates (accepted, not rendered)
//
// A typical frame goes: RESOURCE_CREATE_2D to allocate a host resource,
// RESOURCE_ATTACH_BACKING to point it at guest framebuffer pages, SET_SCANOUT
// to bind it to a display, then per update TRANSFER_TO_HOST_2D (copy guest
// pixels into the host resource) followed by RESOURCE_FLUSH (present it). On
// flush we hand the resource's pixels to a Display sink.
//
// We advertise no device features, so the driver operates the device in plain
// 2D mode (no virgl/3D, no EDID, no blob resources).
//
// refs https://docs.oasis-open.org/virtio/virtio/v1.1/csprd01/virtio-v1.1-csprd01.html#x1-3130007

const (
	// Virtqueue indices.
	gpuControlQueue = 0
	gpuCursorQueue  = 1
	gpuNumQueues    = 2

	// GPUMMIOBase is the guest-physical base of the GPU device's memory BAR.
	// It sits in the 32-bit MMIO hole, distinct from the net (0xd000_0000)
	// and blk (0xd001_0000) regions; the pci layer follows any reassignment
	// the guest performs.
	GPUMMIOBase = 0xd002_0000

	// Device-config geometry and defaults.
	gpuConfigLen     = 16 // sizeof(struct virtio_gpu_config)
	gpuNumScanouts   = 1
	gpuBytesPerPixel = 4
	gpuDefaultWidth  = 1024
	gpuDefaultHeight = 768

	// Wire-format struct sizes.
	gpuCtrlHdrLen    = 24 // struct virtio_gpu_ctrl_hdr
	gpuRectLen       = 16 // struct virtio_gpu_rect
	gpuDisplayOneLen = 24 // struct virtio_gpu_display_one (rect + enabled + flags)
	gpuMaxScanouts   = 16 // VIRTIO_GPU_MAX_SCANOUTS

	// virtio_gpu_ctrl_hdr.flags.
	gpuFlagFence = 0x1 // VIRTIO_GPU_FLAG_FENCE

	// Control commands (virtio_gpu_ctrl_type, 2D subset).
	gpuCmdGetDisplayInfo        = 0x0100
	gpuCmdResourceCreate2D      = 0x0101
	gpuCmdResourceUnref         = 0x0102
	gpuCmdSetScanout            = 0x0103
	gpuCmdResourceFlush         = 0x0104
	gpuCmdTransferToHost2D      = 0x0105
	gpuCmdResourceAttachBacking = 0x0106
	gpuCmdResourceDetachBacking = 0x0107

	// Responses.
	gpuRespOKNoData             = 0x1100
	gpuRespOKDisplayInfo        = 0x1101
	gpuRespErrUnspec            = 0x1200
	gpuRespErrInvalidScanoutID  = 0x1202
	gpuRespErrInvalidResourceID = 0x1203

	// virtio_gpu_formats. Bytes are listed most-significant-first, e.g.
	// B8G8R8A8 stores B at byte 0, A at byte 3.
	gpuFormatB8G8R8A8 = 1
	gpuFormatB8G8R8X8 = 2
	gpuFormatA8R8G8B8 = 3
	gpuFormatX8R8G8B8 = 4
	gpuFormatR8G8B8A8 = 67
	gpuFormatX8B8G8R8 = 68
	gpuFormatA8B8G8R8 = 121
	gpuFormatR8G8B8X8 = 134
)

// ErrNoGPUReq is returned by the queue processors when no request is pending.
var ErrNoGPUReq = errors.New("no virtio-gpu request")

// GPU is a modern PCI device exposing capabilities and a memory BAR.
var _ pci.CapsAndMMIO = (*GPU)(nil)

// gpuMemEntry is one guest-physical region of a resource's backing store
// (struct virtio_gpu_mem_entry).
type gpuMemEntry struct {
	addr   uint64
	length uint32
}

// gpuResource is a host-side 2D resource. data holds the host copy of the
// pixels (width*height*4), filled from the guest backing on transfer.
type gpuResource struct {
	width   uint32
	height  uint32
	format  uint32
	data    []byte
	backing []gpuMemEntry
}

// GPU is a modern (virtio 1.0) 2D display device.
type GPU struct {
	*ModernTransport

	width   uint32
	height  uint32
	display Display

	resources map[uint32]*gpuResource
	scanout   [gpuNumScanouts]uint32

	VirtQueue    [gpuNumQueues]*SplitQueue
	LastAvailIdx [gpuNumQueues]uint16

	kick      chan int
	done      chan struct{}
	closeOnce sync.Once

	irq         uint8
	IRQInjector IRQInjector
}

func (g *GPU) GetDeviceHeader() pci.DeviceHeader {
	return pci.DeviceHeader{
		// 0x1040 + virtio device id (16 = GPU). The 0x1050 id marks a
		// non-transitional device, so the driver uses the modern
		// interface and requires VIRTIO_F_VERSION_1.
		DeviceID:    0x1050,
		VendorID:    0x1AF4,
		ClassCode:   0x03, // Display controller
		Subclass:    0x00, // VGA-compatible controller
		HeaderType:  0,
		SubsystemID: 16, // GPU
		// Memory space enable | bus master.
		Command: 0x6,
		// Bit 4: capabilities list present.
		Status:              0x10,
		CapabilitiesPointer: capCommonAt,
		BAR: [6]uint32{
			// BAR0: 32-bit non-prefetchable memory BAR (low nibble 0).
			uint32(GPUMMIOBase),
		},
		InterruptPin:  1,
		InterruptLine: g.irq,
	}
}

// ModernDevice implementation.

// DeviceFeatures advertises no device-specific features; the device therefore
// operates in plain 2D mode. VIRTIO_F_VERSION_1 is added by the transport.
func (g *GPU) DeviceFeatures() uint64 { return 0 }

func (g *GPU) NumQueues() int { return gpuNumQueues }

// DeviceConfigLen covers struct virtio_gpu_config.
func (g *GPU) DeviceConfigLen() int { return gpuConfigLen }

// ReadDeviceConfig serves struct virtio_gpu_config: events_read (0),
// events_clear (0), num_scanouts and num_capsets (0).
func (g *GPU) ReadDeviceConfig(offset uint64, data []byte) {
	var cfg [gpuConfigLen]byte

	binary.LittleEndian.PutUint32(cfg[8:], gpuNumScanouts)
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

// WriteDeviceConfig handles writes to events_clear, which we have nothing to
// clear for (no config-change events are ever raised).
func (g *GPU) WriteDeviceConfig(offset uint64, data []byte) {}

func (g *GPU) QueueReady(idx int, q *SplitQueue) {
	if idx >= 0 && idx < gpuNumQueues {
		g.VirtQueue[idx] = q
	}
}

func (g *GPU) Notify(idx int) {
	// Non-blocking kick; the IO thread also polls on a ticker.
	select {
	case g.kick <- idx:
	default:
	}
}

func (g *GPU) IOThreadEntry() {
	log.Println("virtio-gpu: IOThreadEntry started")

	// The ticker is a safety net for a missed kick and for re-injecting an
	// unacknowledged interrupt; display updates are otherwise kick-driven.
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-g.done:
			log.Println("virtio-gpu: IOThreadEntry received done signal")

			return
		case <-g.kick:
			g.drain()
		case <-ticker.C:
			g.drain()
		}
	}
}

func (g *GPU) drain() {
	for g.ProcessControlQueue() == nil {
	}

	for g.ProcessCursorQueue() == nil {
	}

	_ = g.ReinjectIfPending()
}

// ProcessControlQueue services all pending controlq requests, writing a
// response for each. It returns nil when it serviced at least one request,
// or ErrNoGPUReq / ErrVQNotInit otherwise.
func (g *GPU) ProcessControlQueue() error { return g.process(gpuControlQueue, true) }

// ProcessCursorQueue drains the cursorq. Cursor commands are accepted but not
// rendered, and (per spec) are completed with a zero-length used entry.
func (g *GPU) ProcessCursorQueue() error { return g.process(gpuCursorQueue, false) }

func (g *GPU) process(sel int, control bool) error {
	q := g.VirtQueue[sel]
	if q == nil {
		return ErrVQNotInit
	}

	if g.LastAvailIdx[sel] == LoadU16(&q.Avail.Idx) {
		return ErrNoGPUReq
	}

	for g.LastAvailIdx[sel] != LoadU16(&q.Avail.Idx) {
		head := q.Avail.Ring[g.LastAvailIdx[sel]%QueueSize]

		var used uint32

		if control {
			req, wr := g.collectChain(q, head)
			resp := g.handleControl(req)
			used = g.writeResponse(wr, resp)
		}

		uidx := LoadU16(&q.Used.Idx)
		q.Used.Ring[uidx%QueueSize].ID = uint32(head)
		q.Used.Ring[uidx%QueueSize].Len = used

		StoreAddU16(&q.Used.Idx, 1)
		g.LastAvailIdx[sel]++
	}

	return g.Interrupt()
}

// writableSeg is a device-writable descriptor segment (response buffer).
type writableSeg struct {
	addr   uint64
	length uint32
}

// collectChain walks the descriptor chain from head, concatenating the
// device-readable descriptors into the request buffer and recording the
// device-writable descriptors for the response.
func (g *GPU) collectChain(q *SplitQueue, head uint16) ([]byte, []writableSeg) {
	var (
		req []byte
		wr  []writableSeg
	)

	descID := head

	for {
		desc := q.Desc[descID]

		if desc.Flags&descFWrite != 0 {
			wr = append(wr, writableSeg{addr: desc.Addr, length: desc.Len})
		} else {
			end := desc.Addr + uint64(desc.Len)
			if desc.Addr < uint64(len(g.Mem)) && end <= uint64(len(g.Mem)) {
				req = append(req, g.Mem[desc.Addr:end]...)
			}
		}

		if desc.Flags&descFNext == 0 {
			break
		}

		descID = desc.Next
	}

	return req, wr
}

// writeResponse spreads resp across the writable segments and returns the
// number of bytes written (for the used-ring length).
func (g *GPU) writeResponse(wr []writableSeg, resp []byte) uint32 {
	written := uint32(0)

	for _, seg := range wr {
		if len(resp) == 0 {
			break
		}

		n := uint64(seg.length)
		if n > uint64(len(resp)) {
			n = uint64(len(resp))
		}

		end := seg.addr + n
		if seg.addr < uint64(len(g.Mem)) && end <= uint64(len(g.Mem)) {
			copy(g.Mem[seg.addr:end], resp[:n])
		}

		resp = resp[n:]
		written += uint32(n)
	}

	return written
}

// handleControl dispatches one control command and returns its response. If
// the request carried a fence, the response echoes it so the driver's fence
// tracking completes.
func (g *GPU) handleControl(req []byte) []byte {
	if len(req) < gpuCtrlHdrLen {
		return g.respNoData(gpuRespErrUnspec)
	}

	var resp []byte

	switch binary.LittleEndian.Uint32(req[0:]) {
	case gpuCmdGetDisplayInfo:
		resp = g.cmdGetDisplayInfo()
	case gpuCmdResourceCreate2D:
		resp = g.cmdResourceCreate2D(req)
	case gpuCmdResourceUnref:
		resp = g.cmdResourceUnref(req)
	case gpuCmdSetScanout:
		resp = g.cmdSetScanout(req)
	case gpuCmdResourceFlush:
		resp = g.cmdResourceFlush(req)
	case gpuCmdTransferToHost2D:
		resp = g.cmdTransferToHost2D(req)
	case gpuCmdResourceAttachBacking:
		resp = g.cmdResourceAttachBacking(req)
	case gpuCmdResourceDetachBacking:
		resp = g.cmdResourceDetachBacking(req)
	default:
		log.Printf("virtio-gpu: unhandled control command 0x%x",
			binary.LittleEndian.Uint32(req[0:]))

		resp = g.respNoData(gpuRespErrUnspec)
	}

	g.echoFence(req, resp)

	return resp
}

// echoFence copies the request's fence into the response and sets the fence
// flag when the request asked for fencing.
func (g *GPU) echoFence(req, resp []byte) {
	if len(req) < gpuCtrlHdrLen || len(resp) < gpuCtrlHdrLen {
		return
	}

	if binary.LittleEndian.Uint32(req[4:])&gpuFlagFence == 0 {
		return
	}

	binary.LittleEndian.PutUint32(resp[4:], gpuFlagFence)
	copy(resp[8:24], req[8:24]) // fence_id, ctx_id, ring_idx
}

// respNoData builds a bare ctrl_hdr response carrying respType.
func (g *GPU) respNoData(respType uint32) []byte {
	b := make([]byte, gpuCtrlHdrLen)
	binary.LittleEndian.PutUint32(b[0:], respType)

	return b
}

// cmdGetDisplayInfo reports a single enabled scanout at the device's geometry
// (struct virtio_gpu_resp_display_info).
func (g *GPU) cmdGetDisplayInfo() []byte {
	b := make([]byte, gpuCtrlHdrLen+gpuMaxScanouts*gpuDisplayOneLen)
	le := binary.LittleEndian
	le.PutUint32(b[0:], gpuRespOKDisplayInfo)

	// pmodes[0]: rect {0, 0, width, height}, enabled = 1.
	base := gpuCtrlHdrLen
	le.PutUint32(b[base+8:], g.width)
	le.PutUint32(b[base+12:], g.height)
	le.PutUint32(b[base+16:], 1)

	return b
}

// cmdResourceCreate2D allocates a host resource (struct
// virtio_gpu_resource_create_2d).
func (g *GPU) cmdResourceCreate2D(req []byte) []byte {
	if len(req) < gpuCtrlHdrLen+16 {
		return g.respNoData(gpuRespErrUnspec)
	}

	le := binary.LittleEndian
	id := le.Uint32(req[gpuCtrlHdrLen+0:])
	format := le.Uint32(req[gpuCtrlHdrLen+4:])
	width := le.Uint32(req[gpuCtrlHdrLen+8:])
	height := le.Uint32(req[gpuCtrlHdrLen+12:])

	if id == 0 {
		return g.respNoData(gpuRespErrInvalidResourceID)
	}

	g.resources[id] = &gpuResource{
		width:  width,
		height: height,
		format: format,
		data:   make([]byte, int(width)*int(height)*gpuBytesPerPixel),
	}

	return g.respNoData(gpuRespOKNoData)
}

// cmdResourceUnref destroys a resource (struct virtio_gpu_resource_unref).
func (g *GPU) cmdResourceUnref(req []byte) []byte {
	if len(req) < gpuCtrlHdrLen+8 {
		return g.respNoData(gpuRespErrUnspec)
	}

	delete(g.resources, binary.LittleEndian.Uint32(req[gpuCtrlHdrLen:]))

	return g.respNoData(gpuRespOKNoData)
}

// cmdSetScanout binds a resource to a scanout (struct virtio_gpu_set_scanout).
func (g *GPU) cmdSetScanout(req []byte) []byte {
	if len(req) < gpuCtrlHdrLen+gpuRectLen+8 {
		return g.respNoData(gpuRespErrUnspec)
	}

	le := binary.LittleEndian
	base := gpuCtrlHdrLen + gpuRectLen
	scanoutID := le.Uint32(req[base:])
	resourceID := le.Uint32(req[base+4:])

	if scanoutID >= gpuNumScanouts {
		return g.respNoData(gpuRespErrInvalidScanoutID)
	}

	g.scanout[scanoutID] = resourceID

	return g.respNoData(gpuRespOKNoData)
}

// cmdResourceFlush presents a resource to the display (struct
// virtio_gpu_resource_flush).
func (g *GPU) cmdResourceFlush(req []byte) []byte {
	if len(req) < gpuCtrlHdrLen+gpuRectLen+8 {
		return g.respNoData(gpuRespErrUnspec)
	}

	resourceID := binary.LittleEndian.Uint32(req[gpuCtrlHdrLen+gpuRectLen:])

	res := g.resources[resourceID]
	if res == nil {
		return g.respNoData(gpuRespErrInvalidResourceID)
	}

	g.flush(res)

	return g.respNoData(gpuRespOKNoData)
}

// cmdTransferToHost2D copies guest pixels into a host resource (struct
// virtio_gpu_transfer_to_host_2d).
func (g *GPU) cmdTransferToHost2D(req []byte) []byte {
	if len(req) < gpuCtrlHdrLen+gpuRectLen+16 {
		return g.respNoData(gpuRespErrUnspec)
	}

	le := binary.LittleEndian
	x := le.Uint32(req[gpuCtrlHdrLen+0:])
	y := le.Uint32(req[gpuCtrlHdrLen+4:])
	w := le.Uint32(req[gpuCtrlHdrLen+8:])
	h := le.Uint32(req[gpuCtrlHdrLen+12:])
	offset := le.Uint64(req[gpuCtrlHdrLen+gpuRectLen:])
	resourceID := le.Uint32(req[gpuCtrlHdrLen+gpuRectLen+8:])

	res := g.resources[resourceID]
	if res == nil {
		return g.respNoData(gpuRespErrInvalidResourceID)
	}

	g.transferToHost2D(res, x, y, w, h, offset)

	return g.respNoData(gpuRespOKNoData)
}

// transferToHost2D fills res.data from the guest backing for the rectangle
// (x, y, w, h) starting at the given backing offset, following the same row
// arithmetic QEMU's virtio-gpu uses.
func (g *GPU) transferToHost2D(res *gpuResource, x, y, w, h uint32, offset uint64) {
	bpp := uint32(gpuBytesPerPixel)
	stride := res.width * bpp

	// Full-frame fast path: a single contiguous copy.
	if offset == 0 && x == 0 && y == 0 && w == res.width {
		total := int(stride) * int(res.height)
		if total > len(res.data) {
			total = len(res.data)
		}

		g.backingRead(res, 0, res.data[:total])

		return
	}

	for row := uint32(0); row < h; row++ {
		src := offset + uint64(stride)*uint64(row)
		dst := int((y+row)*stride + x*bpp)

		if dst >= len(res.data) {
			break
		}

		n := int(w * bpp)
		if dst+n > len(res.data) {
			n = len(res.data) - dst
		}

		g.backingRead(res, src, res.data[dst:dst+n])
	}
}

// backingRead copies len(dst) bytes from the resource's scatter-gather guest
// backing, starting at logical offset off, into dst.
func (g *GPU) backingRead(res *gpuResource, off uint64, dst []byte) {
	pos := uint64(0) // logical offset of the current entry's first byte

	for _, e := range res.backing {
		if len(dst) == 0 {
			return
		}

		entryEnd := pos + uint64(e.length)

		if off < entryEnd {
			skip := uint64(0)
			if off > pos {
				skip = off - pos
			}

			n := uint64(e.length) - skip
			if n > uint64(len(dst)) {
				n = uint64(len(dst))
			}

			src := e.addr + skip
			if src+n <= uint64(len(g.Mem)) {
				copy(dst[:n], g.Mem[src:src+n])
			}

			dst = dst[n:]
			off += n
		}

		pos = entryEnd
	}
}

// cmdResourceAttachBacking records a resource's guest backing pages (struct
// virtio_gpu_resource_attach_backing followed by virtio_gpu_mem_entry array).
func (g *GPU) cmdResourceAttachBacking(req []byte) []byte {
	if len(req) < gpuCtrlHdrLen+8 {
		return g.respNoData(gpuRespErrUnspec)
	}

	le := binary.LittleEndian
	resourceID := le.Uint32(req[gpuCtrlHdrLen:])
	nr := le.Uint32(req[gpuCtrlHdrLen+4:])

	res := g.resources[resourceID]
	if res == nil {
		return g.respNoData(gpuRespErrInvalidResourceID)
	}

	entries := make([]gpuMemEntry, 0, nr)
	off := gpuCtrlHdrLen + 8

	for i := uint32(0); i < nr; i++ {
		if off+16 > len(req) {
			break
		}

		entries = append(entries, gpuMemEntry{
			addr:   le.Uint64(req[off:]),
			length: le.Uint32(req[off+8:]),
		})
		off += 16
	}

	res.backing = entries

	return g.respNoData(gpuRespOKNoData)
}

// cmdResourceDetachBacking drops a resource's backing (struct
// virtio_gpu_resource_detach_backing).
func (g *GPU) cmdResourceDetachBacking(req []byte) []byte {
	if len(req) < gpuCtrlHdrLen+8 {
		return g.respNoData(gpuRespErrUnspec)
	}

	if res := g.resources[binary.LittleEndian.Uint32(req[gpuCtrlHdrLen:])]; res != nil {
		res.backing = nil
	}

	return g.respNoData(gpuRespOKNoData)
}

// flush converts a resource's pixels to RGBA and hands them to the display.
func (g *GPU) flush(res *gpuResource) {
	if g.display == nil || res.width == 0 || res.height == 0 {
		return
	}

	rOff, gOff, bOff, aOff, hasAlpha := formatOffsets(res.format)
	img := image.NewRGBA(image.Rect(0, 0, int(res.width), int(res.height)))

	pixels := int(res.width) * int(res.height)
	for i := 0; i < pixels; i++ {
		si := i * gpuBytesPerPixel
		if si+gpuBytesPerPixel > len(res.data) {
			break
		}

		di := i * 4
		img.Pix[di+0] = res.data[si+rOff]
		img.Pix[di+1] = res.data[si+gOff]
		img.Pix[di+2] = res.data[si+bOff]

		if hasAlpha {
			img.Pix[di+3] = res.data[si+aOff]
		} else {
			img.Pix[di+3] = 0xff
		}
	}

	if err := g.display.Flush(int(res.width), int(res.height), img); err != nil {
		log.Printf("virtio-gpu: display flush: %v", err)
	}
}

// formatOffsets returns the byte position of each channel within a 4-byte
// pixel for a virtio_gpu_formats value. Unknown formats are treated as BGRX.
func formatOffsets(format uint32) (rOff, gOff, bOff, aOff int, hasAlpha bool) {
	switch format {
	case gpuFormatB8G8R8A8:
		return 2, 1, 0, 3, true
	case gpuFormatB8G8R8X8:
		return 2, 1, 0, 0, false
	case gpuFormatA8R8G8B8:
		return 1, 2, 3, 0, true
	case gpuFormatX8R8G8B8:
		return 1, 2, 3, 0, false
	case gpuFormatR8G8B8A8:
		return 0, 1, 2, 3, true
	case gpuFormatX8B8G8R8:
		return 3, 2, 1, 0, false
	case gpuFormatA8B8G8R8:
		return 3, 2, 1, 0, true
	case gpuFormatR8G8B8X8:
		return 0, 1, 2, 0, false
	default:
		return 2, 1, 0, 0, false
	}
}

// Read and Write satisfy pci.Device. A modern device has no IO-port BAR, so
// these are no-ops.
func (g *GPU) Read(port uint64, bytes []byte) error { return nil }

func (g *GPU) Write(port uint64, bytes []byte) error { return nil }

func (g *GPU) IOPort() uint64 { return 0 }

func (g *GPU) Size() uint64 { return 0 }

func (g *GPU) Close() error {
	log.Println("virtio-gpu: Close called")
	g.closeOnce.Do(func() { close(g.done) })

	if g.display != nil {
		return g.display.Close()
	}

	return nil
}

func NewGPU(irq uint8, irqInjector IRQInjector, mem []byte, display Display) *GPU {
	g := &GPU{
		width:       gpuDefaultWidth,
		height:      gpuDefaultHeight,
		display:     display,
		resources:   map[uint32]*gpuResource{},
		kick:        make(chan int, QueueSize),
		done:        make(chan struct{}),
		irq:         irq,
		IRQInjector: irqInjector,
	}

	g.ModernTransport = NewModernTransport(g, mem, func() error {
		return irqInjector.InjectVirtioGPUIRQ()
	})

	return g
}
