//nolint:err113 // RFB parsing returns contextual protocol errors for clients.
package virtio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	rfbProtocolVersion = "RFB 003.008\n"
	rfbSecurityNone    = 1

	rfbMsgSetPixelFormat           = 0
	rfbMsgSetEncodings             = 2
	rfbMsgFramebufferUpdateRequest = 3
	rfbMsgKeyEvent                 = 4
	rfbMsgPointerEvent             = 5
	rfbMsgClientCutText            = 6

	rfbEncodingRaw    = 0
	rfbEncodingCursor = 0xffffff11 // int32(-239) on the wire.

	vncDefaultWidth  = 1024
	vncDefaultHeight = 768
	vncTextCols      = 100
	vncTextRows      = 40
	vncTextCellW     = 7
	vncTextCellH     = 13

	vgaTextBase = 0xb8000
	vgaTextCols = 80
	vgaTextRows = 25
)

type rfbPixelFormat struct {
	bitsPerPixel uint8
	depth        uint8
	bigEndian    bool
	trueColor    bool
	redMax       uint16
	greenMax     uint16
	blueMax      uint16
	redShift     uint8
	greenShift   uint8
	blueShift    uint8
}

type vncFrame struct {
	width  int
	height int
	pix    []byte
	seq    uint64
}

// VNCInput receives input events decoded from RFB client messages.
type VNCInput interface {
	KeyEvent(down bool, keysym uint32)
	PointerEvent(buttonMask uint8, x, y uint16)
}

// VNCDisplay exposes flushed virtio-gpu frames over the RFB/VNC protocol.
type VNCDisplay struct {
	listener net.Listener

	mu        sync.Mutex
	cond      *sync.Cond
	width     int
	height    int
	frame     []byte
	seq       uint64
	done      chan struct{}
	closeOnce sync.Once

	conns map[net.Conn]struct{}
	wg    sync.WaitGroup
	input VNCInput

	textMu       sync.Mutex
	textConsole  *vncTextConsole
	textDisabled bool
	serialMuted  bool
}

// NewVNCDisplay starts a VNC server listening on addr, such as ":5900".
func NewVNCDisplay(addr string) (*VNCDisplay, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	d := &VNCDisplay{
		listener: ln,
		width:    vncDefaultWidth,
		height:   vncDefaultHeight,
		done:     make(chan struct{}),
		conns:    make(map[net.Conn]struct{}),
	}
	d.cond = sync.NewCond(&d.mu)

	d.wg.Add(1)
	go d.acceptLoop()

	return d, nil
}

// Addr returns the address the VNC server is listening on.
func (d *VNCDisplay) Addr() string {
	return d.listener.Addr().String()
}

// SetInput attaches an input sink for VNC keyboard and pointer events.
func (d *VNCDisplay) SetInput(input VNCInput) {
	d.mu.Lock()
	d.input = input
	d.mu.Unlock()
}

func (d *VNCDisplay) Flush(width, height int, img *image.RGBA) error {
	d.textMu.Lock()
	d.textDisabled = true
	d.textMu.Unlock()

	return d.flush(width, height, img)
}

func (d *VNCDisplay) flush(width, height int, img *image.RGBA) error {
	if width <= 0 || height <= 0 {
		return nil
	}

	frame := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		src := img.PixOffset(0, y)
		dst := y * width * 4
		copy(frame[dst:dst+width*4], img.Pix[src:src+width*4])
	}

	d.mu.Lock()
	d.width = width
	d.height = height
	d.frame = frame
	d.seq++
	d.cond.Broadcast()
	d.mu.Unlock()

	return nil
}

// Write mirrors serial console bytes into VNC until virtio-gpu flushes a frame.
func (d *VNCDisplay) Write(p []byte) (int, error) {
	d.textMu.Lock()
	if d.textDisabled || d.serialMuted {
		d.textMu.Unlock()

		return len(p), nil
	}

	if d.textConsole == nil {
		d.textConsole = newVNCTextConsole(vncTextCols, vncTextRows)
	}

	d.textConsole.write(p)
	img := d.textConsole.render()
	d.textMu.Unlock()

	if err := d.flush(img.Bounds().Dx(), img.Bounds().Dy(), img); err != nil {
		return 0, err
	}

	return len(p), nil
}

// StartVGATextFallback renders legacy VGA text memory into VNC until a real
// virtio-gpu frame is flushed.
func (d *VNCDisplay) StartVGATextFallback(mem []byte) {
	const textSize = vgaTextCols * vgaTextRows * 2
	if len(mem) < vgaTextBase+textSize {
		return
	}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		var last []byte
		rendered := false

		for {
			select {
			case <-d.done:
				return
			case <-ticker.C:
			}

			d.textMu.Lock()
			disabled := d.textDisabled
			d.textMu.Unlock()
			if disabled {
				return
			}

			snap := make([]byte, textSize)
			copy(snap, mem[vgaTextBase:vgaTextBase+textSize])
			if bytes.Equal(snap, last) {
				continue
			}

			last = snap
			if !rendered && vgaTextBlank(snap) {
				continue
			}

			rendered = true
			d.muteSerialFallback()
			img := renderVGAText(snap)
			_ = d.flush(img.Bounds().Dx(), img.Bounds().Dy(), img)
		}
	}()
}

// StartLinearFramebufferFallback renders a guest linear BGRX framebuffer into
// VNC until a real virtio-gpu frame is flushed.
func (d *VNCDisplay) StartLinearFramebufferFallback(mem []byte, base, width, height, stride int) {
	if base < 0 || width <= 0 || height <= 0 || stride < width*4 {
		return
	}

	size := stride * height
	if len(mem) < base+size {
		return
	}

	go func() {
		ticker := time.NewTicker(33 * time.Millisecond)
		defer ticker.Stop()

		var last []byte
		rendered := false

		for {
			select {
			case <-d.done:
				return
			case <-ticker.C:
			}

			d.textMu.Lock()
			disabled := d.textDisabled
			d.textMu.Unlock()
			if disabled {
				return
			}

			snap := make([]byte, size)
			copy(snap, mem[base:base+size])
			if bytes.Equal(snap, last) {
				continue
			}

			last = snap
			if !rendered && framebufferBlank(snap) {
				continue
			}

			rendered = true
			d.muteSerialFallback()
			img := renderLinearFramebuffer(snap, width, height, stride)
			_ = d.flush(width, height, img)
		}
	}()
}

func (d *VNCDisplay) muteSerialFallback() {
	d.textMu.Lock()
	d.serialMuted = true
	d.textMu.Unlock()
}

func (d *VNCDisplay) Close() error {
	var err error

	d.closeOnce.Do(func() {
		close(d.done)
		err = d.listener.Close()

		d.mu.Lock()
		for conn := range d.conns {
			_ = conn.Close()
		}
		d.cond.Broadcast()
		d.mu.Unlock()

		d.wg.Wait()
	})

	return err
}

func (d *VNCDisplay) acceptLoop() {
	defer d.wg.Done()

	for {
		conn, err := d.listener.Accept()
		if err != nil {
			select {
			case <-d.done:
				return
			default:
				log.Printf("vnc: accept: %v", err)

				continue
			}
		}

		d.trackConn(conn)
		d.wg.Add(1)
		go d.handleConn(conn)
	}
}

func (d *VNCDisplay) trackConn(conn net.Conn) {
	d.mu.Lock()
	d.conns[conn] = struct{}{}
	d.mu.Unlock()
}

func (d *VNCDisplay) untrackConn(conn net.Conn) {
	d.mu.Lock()
	delete(d.conns, conn)
	d.mu.Unlock()
}

func (d *VNCDisplay) handleConn(conn net.Conn) {
	defer d.wg.Done()
	defer d.untrackConn(conn)
	defer conn.Close()

	if err := d.serveConn(conn); err != nil {
		log.Printf("vnc: client %s: %v", conn.RemoteAddr(), err)
	}
}

func (d *VNCDisplay) serveConn(conn net.Conn) error {
	if err := d.handshake(conn); err != nil {
		return err
	}

	pf := defaultRFBPixelFormat()
	cursorEnabled := false
	frame := d.snapshot()

	if err := writeServerInit(conn, frame.width, frame.height, pf); err != nil {
		return err
	}

	updateReqs := make(chan vncUpdateRequest, 1)
	updateErrs := make(chan error, 1)
	done := make(chan struct{})
	defer func() {
		close(done)
		d.wakeFrameWaiters()
	}()

	go d.writeFramebufferUpdates(conn, updateReqs, updateErrs, done)

	for {
		select {
		case err := <-updateErrs:
			return err
		default:
		}

		var typ [1]byte
		if _, err := io.ReadFull(conn, typ[:]); err != nil {
			return err
		}

		switch typ[0] {
		case rfbMsgSetPixelFormat:
			next, err := readSetPixelFormat(conn)
			if err != nil {
				return err
			}

			pf = next
		case rfbMsgSetEncodings:
			encodings, err := readSetEncodings(conn)
			if err != nil {
				return err
			}

			cursorEnabled = encodings.cursor
		case rfbMsgFramebufferUpdateRequest:
			req, err := readFramebufferUpdateRequest(conn)
			if err != nil {
				return err
			}

			select {
			case updateReqs <- vncUpdateRequest{req: req, pf: pf, cursor: cursorEnabled}:
			default:
			}
		case rfbMsgKeyEvent:
			event, err := readKeyEvent(conn)
			if err != nil {
				return err
			}

			d.sendKeyEvent(event.down, event.keysym)
		case rfbMsgPointerEvent:
			event, err := readPointerEvent(conn)
			if err != nil {
				return err
			}

			d.sendPointerEvent(event.buttonMask, event.x, event.y)
		case rfbMsgClientCutText:
			if err := readClientCutText(conn); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown client message %d", typ[0])
		}
	}
}

type vncUpdateRequest struct {
	req    framebufferUpdateRequest
	pf     rfbPixelFormat
	cursor bool
}

func (d *VNCDisplay) writeFramebufferUpdates(
	conn net.Conn,
	reqs <-chan vncUpdateRequest,
	errs chan<- error,
	done <-chan struct{},
) {
	lastSeq := uint64(0)

	for {
		select {
		case <-done:
			return
		case req := <-reqs:
			frame, ok := d.frameForRequestUntil(req.req.incremental, lastSeq, done)
			if !ok {
				return
			}

			if err := writeFramebufferUpdate(
				conn,
				frame,
				req.pf,
				req.cursor,
				req.req.x,
				req.req.y,
				req.req.width,
				req.req.height,
			); err != nil {
				select {
				case errs <- err:
				default:
				}
				_ = conn.Close()

				return
			}

			lastSeq = frame.seq
		}
	}
}

func (d *VNCDisplay) handshake(conn net.Conn) error {
	if _, err := conn.Write([]byte(rfbProtocolVersion)); err != nil {
		return err
	}

	var version [12]byte
	if _, err := io.ReadFull(conn, version[:]); err != nil {
		return err
	}

	if _, err := conn.Write([]byte{1, rfbSecurityNone}); err != nil {
		return err
	}

	var security [1]byte
	if _, err := io.ReadFull(conn, security[:]); err != nil {
		return err
	}

	if security[0] != rfbSecurityNone {
		return fmt.Errorf("unsupported security type %d", security[0])
	}

	var result [4]byte
	if _, err := conn.Write(result[:]); err != nil {
		return err
	}

	var clientInit [1]byte
	_, err := io.ReadFull(conn, clientInit[:])

	return err
}

func (d *VNCDisplay) sendKeyEvent(down bool, keysym uint32) {
	d.mu.Lock()
	input := d.input
	d.mu.Unlock()

	if input != nil {
		input.KeyEvent(down, keysym)
	}
}

func (d *VNCDisplay) sendPointerEvent(buttonMask uint8, x, y uint16) {
	d.mu.Lock()
	input := d.input
	d.mu.Unlock()

	if input != nil {
		input.PointerEvent(buttonMask, x, y)
	}
}

func (d *VNCDisplay) snapshot() vncFrame {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.snapshotLocked()
}

func (d *VNCDisplay) frameForRequestUntil(incremental bool, lastSeq uint64, done <-chan struct{}) (vncFrame, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if incremental {
		for d.seq == lastSeq {
			if d.frameWaitCanceled(done) {
				return d.snapshotLocked(), false
			}

			d.cond.Wait()
		}
	}

	return d.snapshotLocked(), true
}

func (d *VNCDisplay) frameWaitCanceled(done <-chan struct{}) bool {
	select {
	case <-d.done:
		return true
	default:
	}

	if done != nil {
		select {
		case <-done:
			return true
		default:
		}
	}

	return false
}

func (d *VNCDisplay) wakeFrameWaiters() {
	d.mu.Lock()
	d.cond.Broadcast()
	d.mu.Unlock()
}

func (d *VNCDisplay) snapshotLocked() vncFrame {
	frame := make([]byte, len(d.frame))
	copy(frame, d.frame)

	return vncFrame{
		width:  d.width,
		height: d.height,
		pix:    frame,
		seq:    d.seq,
	}
}

type framebufferUpdateRequest struct {
	incremental bool
	x           uint16
	y           uint16
	width       uint16
	height      uint16
}

type keyEvent struct {
	down   bool
	keysym uint32
}

type pointerEvent struct {
	buttonMask uint8
	x          uint16
	y          uint16
}

type clientEncodings struct {
	cursor bool
}

func readFramebufferUpdateRequest(r io.Reader) (framebufferUpdateRequest, error) {
	var b [9]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return framebufferUpdateRequest{}, err
	}

	return framebufferUpdateRequest{
		incremental: b[0] != 0,
		x:           binary.BigEndian.Uint16(b[1:3]),
		y:           binary.BigEndian.Uint16(b[3:5]),
		width:       binary.BigEndian.Uint16(b[5:7]),
		height:      binary.BigEndian.Uint16(b[7:9]),
	}, nil
}

func readKeyEvent(r io.Reader) (keyEvent, error) {
	var b [7]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return keyEvent{}, err
	}

	return keyEvent{
		down:   b[0] != 0,
		keysym: binary.BigEndian.Uint32(b[3:7]),
	}, nil
}

func readPointerEvent(r io.Reader) (pointerEvent, error) {
	var b [5]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return pointerEvent{}, err
	}

	return pointerEvent{
		buttonMask: b[0],
		x:          binary.BigEndian.Uint16(b[1:3]),
		y:          binary.BigEndian.Uint16(b[3:5]),
	}, nil
}

func readSetPixelFormat(r io.Reader) (rfbPixelFormat, error) {
	var b [19]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return rfbPixelFormat{}, err
	}

	pf := parsePixelFormat(b[3:])
	if err := validatePixelFormat(pf); err != nil {
		return rfbPixelFormat{}, err
	}

	return pf, nil
}

func readSetEncodings(r io.Reader) (clientEncodings, error) {
	var hdr [3]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return clientEncodings{}, err
	}

	n := binary.BigEndian.Uint16(hdr[1:3])
	var enc clientEncodings
	var raw [4]byte

	for range n {
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return clientEncodings{}, err
		}

		if binary.BigEndian.Uint32(raw[:]) == rfbEncodingCursor {
			enc.cursor = true
		}
	}

	return enc, nil
}

func readClientCutText(r io.Reader) error {
	var hdr [7]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}

	n := binary.BigEndian.Uint32(hdr[3:7])

	return discardFull(r, int(n))
}

func discardFull(r io.Reader, n int) error {
	if n == 0 {
		return nil
	}

	_, err := io.CopyN(io.Discard, r, int64(n))

	return err
}

func writeServerInit(w io.Writer, width, height int, pf rfbPixelFormat) error {
	name := []byte("gokvm")
	b := make([]byte, 24+len(name))

	binary.BigEndian.PutUint16(b[0:2], uint16(width))
	binary.BigEndian.PutUint16(b[2:4], uint16(height))
	putPixelFormat(b[4:20], pf)
	binary.BigEndian.PutUint32(b[20:24], uint32(len(name)))
	copy(b[24:], name)

	_, err := w.Write(b)

	return err
}

func writeFramebufferUpdate(
	w io.Writer,
	frame vncFrame,
	pf rfbPixelFormat,
	cursor bool,
	xReq, yReq, wReq, hReq uint16,
) error {
	x := int(xReq)
	y := int(yReq)
	width := int(wReq)
	height := int(hReq)

	if x >= frame.width || y >= frame.height || width == 0 || height == 0 {
		if !cursor {
			_, err := w.Write([]byte{0, 0, 0, 0})

			return err
		}

		b := append([]byte{0, 0, 0, 1}, encodeCursorPseudoRect(pf)...)
		_, err := w.Write(b)

		return err
	}

	if x+width > frame.width {
		width = frame.width - x
	}

	if y+height > frame.height {
		height = frame.height - y
	}

	pixels := encodeRawPixels(frame, pf, x, y, width, height)
	cursorRect := []byte(nil)
	rects := uint16(1)
	if cursor {
		cursorRect = encodeCursorPseudoRect(pf)
		rects++
	}

	b := make([]byte, 16+len(pixels)+len(cursorRect))

	b[0] = 0 // FramebufferUpdate
	binary.BigEndian.PutUint16(b[2:4], rects)
	binary.BigEndian.PutUint16(b[4:6], uint16(x))
	binary.BigEndian.PutUint16(b[6:8], uint16(y))
	binary.BigEndian.PutUint16(b[8:10], uint16(width))
	binary.BigEndian.PutUint16(b[10:12], uint16(height))
	binary.BigEndian.PutUint32(b[12:16], rfbEncodingRaw)
	copy(b[16:], pixels)
	copy(b[16+len(pixels):], cursorRect)

	_, err := w.Write(b)

	return err
}

func encodeCursorPseudoRect(pf rfbPixelFormat) []byte {
	const (
		width  = 1
		height = 1
	)

	bytesPerPixel := int(pf.bitsPerPixel) / 8
	maskStride := (width + 7) / 8
	pixels := make([]byte, width*height*bytesPerPixel)
	mask := make([]byte, height*maskStride)
	b := make([]byte, 12+len(pixels)+len(mask))
	binary.BigEndian.PutUint16(b[4:6], width)
	binary.BigEndian.PutUint16(b[6:8], height)
	binary.BigEndian.PutUint32(b[8:12], rfbEncodingCursor)
	copy(b[12:], pixels)
	copy(b[12+len(pixels):], mask)

	return b
}

func encodeRawPixels(frame vncFrame, pf rfbPixelFormat, x, y, width, height int) []byte {
	bytesPerPixel := int(pf.bitsPerPixel) / 8
	dst := make([]byte, width*height*bytesPerPixel)

	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			src := ((y+row)*frame.width + x + col) * 4
			dstOff := (row*width + col) * bytesPerPixel

			var r, g, b uint8
			if src+3 <= len(frame.pix) {
				r = frame.pix[src]
				g = frame.pix[src+1]
				b = frame.pix[src+2]
			}

			pixel := scaledChannel(r, pf.redMax)<<pf.redShift |
				scaledChannel(g, pf.greenMax)<<pf.greenShift |
				scaledChannel(b, pf.blueMax)<<pf.blueShift
			putPixel(dst[dstOff:dstOff+bytesPerPixel], pixel, pf.bigEndian)
		}
	}

	return dst
}

func defaultRFBPixelFormat() rfbPixelFormat {
	return rfbPixelFormat{
		bitsPerPixel: 32,
		depth:        24,
		bigEndian:    false,
		trueColor:    true,
		redMax:       255,
		greenMax:     255,
		blueMax:      255,
		redShift:     16,
		greenShift:   8,
		blueShift:    0,
	}
}

func parsePixelFormat(b []byte) rfbPixelFormat {
	return rfbPixelFormat{
		bitsPerPixel: b[0],
		depth:        b[1],
		bigEndian:    b[2] != 0,
		trueColor:    b[3] != 0,
		redMax:       binary.BigEndian.Uint16(b[4:6]),
		greenMax:     binary.BigEndian.Uint16(b[6:8]),
		blueMax:      binary.BigEndian.Uint16(b[8:10]),
		redShift:     b[10],
		greenShift:   b[11],
		blueShift:    b[12],
	}
}

func putPixelFormat(b []byte, pf rfbPixelFormat) {
	b[0] = pf.bitsPerPixel
	b[1] = pf.depth
	if pf.bigEndian {
		b[2] = 1
	}
	if pf.trueColor {
		b[3] = 1
	}
	binary.BigEndian.PutUint16(b[4:6], pf.redMax)
	binary.BigEndian.PutUint16(b[6:8], pf.greenMax)
	binary.BigEndian.PutUint16(b[8:10], pf.blueMax)
	b[10] = pf.redShift
	b[11] = pf.greenShift
	b[12] = pf.blueShift
}

func validatePixelFormat(pf rfbPixelFormat) error {
	if !pf.trueColor {
		return fmt.Errorf("unsupported color-map pixel format")
	}

	if pf.bitsPerPixel != 8 && pf.bitsPerPixel != 16 && pf.bitsPerPixel != 32 {
		return fmt.Errorf("unsupported bits-per-pixel %d", pf.bitsPerPixel)
	}

	if pf.bitsPerPixel < pf.depth {
		return fmt.Errorf("depth %d exceeds bits-per-pixel %d", pf.depth, pf.bitsPerPixel)
	}

	return nil
}

func scaledChannel(v uint8, max uint16) uint32 {
	return uint32(v) * uint32(max) / 255
}

func putPixel(dst []byte, pixel uint32, bigEndian bool) {
	switch len(dst) {
	case 1:
		dst[0] = byte(pixel)
	case 2:
		if bigEndian {
			binary.BigEndian.PutUint16(dst, uint16(pixel))
		} else {
			binary.LittleEndian.PutUint16(dst, uint16(pixel))
		}
	case 4:
		if bigEndian {
			binary.BigEndian.PutUint32(dst, pixel)
		} else {
			binary.LittleEndian.PutUint32(dst, pixel)
		}
	}
}

type vncTextConsole struct {
	cols int
	rows int

	cells [][]rune
	row   int
	col   int

	escape []byte
}

func newVNCTextConsole(cols, rows int) *vncTextConsole {
	c := &vncTextConsole{
		cols:  cols,
		rows:  rows,
		cells: make([][]rune, rows),
	}
	for y := range c.cells {
		c.cells[y] = make([]rune, cols)
		for x := range c.cells[y] {
			c.cells[y][x] = ' '
		}
	}

	return c
}

func (c *vncTextConsole) write(p []byte) {
	for _, b := range p {
		c.putByte(b)
	}
}

func (c *vncTextConsole) putByte(b byte) {
	if len(c.escape) > 0 {
		c.putEscapeByte(b)

		return
	}

	switch b {
	case 0x1b:
		c.escape = []byte{b}
	case '\r':
		c.col = 0
	case '\n':
		c.newline()
	case '\b':
		if c.col > 0 {
			c.col--
		}
	case '\t':
		for {
			c.putRune(' ')
			if c.col%8 == 0 {
				break
			}
		}
	default:
		if b >= 0x20 && b < 0x7f {
			c.putRune(rune(b))
		}
	}
}

func (c *vncTextConsole) putEscapeByte(b byte) {
	c.escape = append(c.escape, b)
	if len(c.escape) == 2 && b != '[' {
		c.escape = nil

		return
	}

	if len(c.escape) < 3 {
		return
	}

	if b < 0x40 || b > 0x7e {
		return
	}

	c.handleCSI(string(c.escape[2:len(c.escape)-1]), b)
	c.escape = nil
}

func (c *vncTextConsole) handleCSI(params string, final byte) {
	switch final {
	case 'H', 'f':
		c.row, c.col = 0, 0
	case 'J':
		if params == "" || params == "2" {
			c.clear()
		}
	case 'K':
		for x := c.col; x < c.cols; x++ {
			c.cells[c.row][x] = ' '
		}
	case 'm', 'h', 'l':
		// Styling and terminal mode toggles are ignored by the fallback.
	default:
	}
}

func (c *vncTextConsole) putRune(r rune) {
	if c.col >= c.cols {
		c.newline()
	}

	c.cells[c.row][c.col] = r
	c.col++
}

func (c *vncTextConsole) newline() {
	c.col = 0
	c.row++
	if c.row < c.rows {
		return
	}

	copy(c.cells, c.cells[1:])
	c.cells[c.rows-1] = make([]rune, c.cols)
	for x := range c.cells[c.rows-1] {
		c.cells[c.rows-1][x] = ' '
	}

	c.row = c.rows - 1
}

func (c *vncTextConsole) clear() {
	for y := range c.cells {
		for x := range c.cells[y] {
			c.cells[y][x] = ' '
		}
	}

	c.row, c.col = 0, 0
}

func (c *vncTextConsole) render() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, c.cols*vncTextCellW, c.rows*vncTextCellH))
	drawer := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 0xe8, G: 0xea, B: 0xed, A: 0xff}),
		Face: basicfont.Face7x13,
	}

	for y, row := range c.cells {
		drawer.Dot = fixed.P(0, y*vncTextCellH+11)
		drawer.DrawString(string(row))
	}

	return img
}

func renderVGAText(text []byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, vgaTextCols*vncTextCellW, vgaTextRows*vncTextCellH))
	drawer := font.Drawer{
		Dst:  img,
		Face: basicfont.Face7x13,
	}

	for y := 0; y < vgaTextRows; y++ {
		for x := 0; x < vgaTextCols; x++ {
			off := (y*vgaTextCols + x) * 2
			ch := text[off]
			if ch < 0x20 || ch >= 0x7f {
				ch = ' '
			}

			if ch == ' ' {
				continue
			}

			drawer.Dot = fixed.P(x*vncTextCellW, y*vncTextCellH+11)
			drawer.Src = image.NewUniform(vgaColor(text[off+1] & 0x0f))
			drawer.DrawString(string([]byte{ch}))
		}
	}

	return img
}

func vgaTextBlank(text []byte) bool {
	for i := 0; i+1 < len(text); i += 2 {
		ch := text[i]
		if ch != 0 && ch != ' ' {
			return false
		}
	}

	return true
}

func vgaColor(idx byte) color.Color {
	palette := [...]color.RGBA{
		{R: 0x00, G: 0x00, B: 0x00, A: 0xff},
		{R: 0x00, G: 0x00, B: 0xaa, A: 0xff},
		{R: 0x00, G: 0xaa, B: 0x00, A: 0xff},
		{R: 0x00, G: 0xaa, B: 0xaa, A: 0xff},
		{R: 0xaa, G: 0x00, B: 0x00, A: 0xff},
		{R: 0xaa, G: 0x00, B: 0xaa, A: 0xff},
		{R: 0xaa, G: 0x55, B: 0x00, A: 0xff},
		{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff},
		{R: 0x55, G: 0x55, B: 0x55, A: 0xff},
		{R: 0x55, G: 0x55, B: 0xff, A: 0xff},
		{R: 0x55, G: 0xff, B: 0x55, A: 0xff},
		{R: 0x55, G: 0xff, B: 0xff, A: 0xff},
		{R: 0xff, G: 0x55, B: 0x55, A: 0xff},
		{R: 0xff, G: 0x55, B: 0xff, A: 0xff},
		{R: 0xff, G: 0xff, B: 0x55, A: 0xff},
		{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}

	return palette[idx&0x0f]
}

func renderLinearFramebuffer(frame []byte, width, height, stride int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			src := y*stride + x*4
			dst := img.PixOffset(x, y)
			img.Pix[dst+0] = frame[src+2]
			img.Pix[dst+1] = frame[src+1]
			img.Pix[dst+2] = frame[src+0]
			img.Pix[dst+3] = 0xff
		}
	}

	return img
}

func framebufferBlank(frame []byte) bool {
	for _, b := range frame {
		if b != 0 {
			return false
		}
	}

	return true
}
