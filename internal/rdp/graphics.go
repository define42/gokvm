//nolint:err113 // Protocol errors identify malformed wire structures for diagnostic logging.
package rdp

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"
)

const (
	graphicsChannelName = "Microsoft::Windows::RDS::Graphics\x00"
	graphicsChannelID   = 1
	graphicsMaxMessage  = 8 << 20
	graphicsMaxIncoming = 1 << 20
	graphicsMaxFrames   = 2
	graphicsVersion81   = 0x00080105
)

// graphicsState implements the AVC420 subset of MS-RDPEGFX over MS-RDPEDYC.
// mu serializes negotiation, frame acknowledgments, and complete DVC messages.
// A single reader owns both inbound reassembly buffers.
type graphicsState struct {
	mu          sync.Mutex
	channel     uint16
	supported   bool
	phase       byte
	ready       bool
	ackOff      bool
	frameID     uint32
	inFlight    []uint32
	changed     chan struct{}
	staticData  fragmentBuffer
	dynamicData fragmentBuffer
}

// BeginGraphics starts optional AVC420 negotiation. A false result means the
// client did not offer the graphics channel; the bitmap path remains available.
// ReadPacket must continue running to process the client's negotiation replies.
func (s *Session) BeginGraphics() (bool, error) {
	g := s.graphics
	if g == nil {
		return false, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.supported || g.channel == 0 || !s.joined[g.channel] {
		return false, nil
	}
	if g.phase != 0 {
		return g.phase != 4, nil
	}
	g.phase = 1

	return true, s.writeStatic(g.channel, []byte{0x50, 0, 1, 0})
}

// GraphicsReady reports whether AVC420 capabilities and the output surface
// have been established. Bitmap output should stop once this returns true.
func (s *Session) GraphicsReady() bool {
	if s.graphics == nil {
		return false
	}
	s.graphics.mu.Lock()
	defer s.graphics.mu.Unlock()

	return s.graphics.ready
}

// GraphicsCanSend applies a bounded frame acknowledgment window. Call this
// before encoding a predicted frame: dropping an encoded frame can invalidate
// the following frame's references. There must be only one video writer.
func (s *Session) GraphicsCanSend() bool {
	if s.graphics == nil {
		return false
	}
	s.graphics.mu.Lock()
	defer s.graphics.mu.Unlock()

	return s.graphics.canSend()
}

// GraphicsChanged wakes the single video writer when negotiation or frame
// acknowledgments change graphics readiness. Notifications are coalesced; the
// writer must check the current state after subscribing and after each wake.
func (s *Session) GraphicsChanged() <-chan struct{} {
	if s.graphics == nil {
		return nil
	}
	g := s.graphics
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.changed == nil {
		g.changed = make(chan struct{}, 1)
	}

	return g.changed
}

// notifyChanged is called with mu held and never blocks the protocol reader.
func (g *graphicsState) notifyChanged() {
	select {
	case g.changed <- struct{}{}:
	default:
	}
}

func (g *graphicsState) canSend() bool {
	return g.ready && (g.ackOff || len(g.inFlight) < graphicsMaxFrames)
}

// WriteAVC420 sends one Annex B H.264 frame covering the entire desktop. The
// encoder must use YUV420 and dimensions rounded up to multiples of 16. Returns
// false without writing if negotiation or frame acknowledgments are pending.
func (s *Session) WriteAVC420(annexB []byte) (bool, error) {
	if len(annexB) == 0 || len(annexB) > graphicsMaxMessage-128 {
		return false, errors.New("rdp: invalid AVC420 frame length")
	}
	g := s.graphics
	if g == nil {
		return false, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.canSend() {
		return false, nil
	}
	g.frameID++
	frame := avc420Frame(s.Width, s.Height, g.frameID, graphicsTimestamp(time.Now()), annexB)
	if err := s.writeGraphics(frame); err != nil {
		return false, err
	}
	if !g.ackOff {
		g.inFlight = append(g.inFlight, g.frameID)
	}

	return true, nil
}

// MS-RDPEGFX encodes UTC time of day, not Unix milliseconds, in START_FRAME.
func graphicsTimestamp(now time.Time) uint32 {
	now = now.UTC()

	return uint32(now.Hour())<<22 | uint32(now.Minute())<<16 | uint32(now.Second())<<10 |
		uint32(now.Nanosecond()/int(time.Millisecond))
}

func (s *Session) readGraphicsChannel(channel uint16, data []byte) error {
	g := s.graphics
	if g == nil || channel != g.channel {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase == 0 || g.phase == 4 {
		return nil
	}
	message, err := g.staticData.static(data)
	if err != nil || message == nil {
		return err
	}

	return s.readDynamic(message)
}

func (s *Session) readDynamic(data []byte) error {
	if len(data) == 0 {
		return errors.New("rdp: empty dynamic channel PDU")
	}
	g := s.graphics
	command := data[0] >> 4
	if command == 5 {
		if g.phase != 1 || len(data) != 4 || data[0] != 0x50 || binary.LittleEndian.Uint16(data[2:]) != 1 {
			return errors.New("rdp: invalid dynamic channel capability response")
		}
		g.phase = 2

		return s.writeStatic(g.channel, append([]byte{0x10, graphicsChannelID}, graphicsChannelName...))
	}
	header := data[0]
	channel, payload, err := readDynamicInteger(data[1:], header&3)
	if err != nil {
		return err
	}
	if channel != graphicsChannelID {
		return errors.New("rdp: unknown dynamic channel")
	}
	switch command {
	case 1: // DYNVC_CREATE_RSP
		if g.phase != 2 || len(payload) != 4 {
			return errors.New("rdp: invalid dynamic channel create response")
		}
		if binary.LittleEndian.Uint32(payload) != 0 {
			g.phase = 4
		} else {
			g.phase = 3
		}
	case 2, 3: // DYNVC_DATA_FIRST / DYNVC_DATA
		if g.phase != 3 {
			return errors.New("rdp: graphics data before channel creation")
		}
		var message []byte
		if command == 2 {
			total, fragment, err := readDynamicInteger(payload, (header>>2)&3)
			if err != nil {
				return err
			}
			message, err = g.dynamicData.first(total, fragment)
			if err != nil {
				return err
			}
		} else {
			message, err = g.dynamicData.next(payload)
			if err != nil {
				return err
			}
		}
		if message != nil {
			return s.readGraphics(message)
		}
	case 4: // DYNVC_CLOSE
		if len(payload) != 0 {
			return errors.New("rdp: invalid dynamic channel close")
		}
		g.ready = false
		g.phase = 4
		g.inFlight = nil
		g.notifyChanged()

		return s.writeStatic(g.channel, []byte{0x40, graphicsChannelID})
	default:
		return errors.New("rdp: unsupported dynamic channel command")
	}

	return nil
}

func (s *Session) readGraphics(data []byte) error {
	for len(data) > 0 {
		if len(data) < 8 {
			return errors.New("rdp: truncated graphics header")
		}
		size := binary.LittleEndian.Uint32(data[4:])
		if size < 8 || uint64(size) > uint64(len(data)) || binary.LittleEndian.Uint16(data[2:]) != 0 {
			return errors.New("rdp: invalid graphics PDU size or flags")
		}
		if err := s.readGraphicsPDU(binary.LittleEndian.Uint16(data), data[8:size]); err != nil {
			return err
		}
		data = data[size:]
	}

	return nil
}

func (s *Session) readGraphicsPDU(command uint16, data []byte) error {
	g := s.graphics
	switch command {
	case 0x12: // RDPGFX_CAPS_ADVERTISE
		if g.ready {
			return errors.New("rdp: duplicate graphics capabilities")
		}
		version, flags, err := selectGraphicsCapabilities(data)
		if err != nil {
			return err
		}
		if version == 0 {
			g.phase = 4

			return s.writeStatic(g.channel, []byte{0x40, graphicsChannelID})
		}
		if err := s.writeGraphics(graphicsInitialization(s.Width, s.Height, version, flags)); err != nil {
			return err
		}
		g.ready = true
		g.notifyChanged()
	case 0x0d: // RDPGFX_FRAME_ACKNOWLEDGE
		if !g.ready || len(data) != 12 {
			return errors.New("rdp: invalid graphics frame acknowledgment")
		}
		if binary.LittleEndian.Uint32(data) == ^uint32(0) {
			g.ackOff = true
			g.inFlight = nil
			g.notifyChanged()

			return nil
		}
		g.ackOff = false
		frameID := binary.LittleEndian.Uint32(data[4:])
		for i, id := range g.inFlight {
			if id == frameID {
				g.inFlight = g.inFlight[i+1:]
				g.notifyChanged()

				return nil
			}
		}
	case 0x10: // Optional persistent cache offer; we import no cache entries.
		if len(data) < 2 || len(data) != 2+int(binary.LittleEndian.Uint16(data))*12 {
			return errors.New("rdp: invalid graphics cache offer")
		}

		return s.writeGraphics(graphicsPDU(0x11, []byte{0, 0}))
	case 0x16: // RDPGFX_QOE_FRAME_ACKNOWLEDGE: no quality adaptation yet.
		if len(data) != 12 {
			return errors.New("rdp: invalid graphics quality acknowledgment")
		}
	default:
		return errors.New("rdp: unsupported client graphics command")
	}

	return nil
}

func selectGraphicsCapabilities(data []byte) (uint32, uint32, error) {
	if len(data) < 2 {
		return 0, 0, errors.New("rdp: truncated graphics capabilities")
	}
	count := int(binary.LittleEndian.Uint16(data))
	if count == 0 || count > 64 {
		return 0, 0, errors.New("rdp: invalid graphics capability count")
	}
	data = data[2:]
	var selected, selectedFlags uint32
	for range count {
		if len(data) < 8 {
			return 0, 0, errors.New("rdp: truncated graphics capability")
		}
		version, size := binary.LittleEndian.Uint32(data), binary.LittleEndian.Uint32(data[4:])
		if uint64(size) > uint64(len(data)-8) {
			return 0, 0, errors.New("rdp: invalid graphics capability size")
		}
		if size == 4 {
			flags := binary.LittleEndian.Uint32(data[8:])
			if version == graphicsVersion81 && flags&0x10 != 0 {
				selected, selectedFlags = version, flags&3|0x10
			} else if selected != graphicsVersion81 && graphicsVersionSupportsAVC(version) && flags&0x20 == 0 {
				selected, selectedFlags = version, flags&0x43
			}
		}
		data = data[8+size:]
	}
	if len(data) != 0 {
		return 0, 0, errors.New("rdp: trailing graphics capabilities")
	}

	return selected, selectedFlags, nil
}

func graphicsVersionSupportsAVC(version uint32) bool {
	switch version {
	case 0x000a0002, 0x000a0200, 0x000a0301, 0x000a0400, 0x000a0502, 0x000a0600, 0x000a0701:
		return true
	default:
		return false
	}
}

func graphicsPDU(command uint16, payload []byte) []byte {
	data := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint16(data, command)
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)))
	copy(data[8:], payload)

	return data
}

func graphicsInitialization(width, height int, version, flags uint32) []byte {
	caps := make([]byte, 12)
	binary.LittleEndian.PutUint32(caps, version)
	binary.LittleEndian.PutUint32(caps[4:], 4)
	binary.LittleEndian.PutUint32(caps[8:], flags)
	data := graphicsPDU(0x13, caps)
	reset := make([]byte, 332)
	binary.LittleEndian.PutUint32(reset, uint32(width))
	binary.LittleEndian.PutUint32(reset[4:], uint32(height))
	binary.LittleEndian.PutUint32(reset[8:], 1)
	binary.LittleEndian.PutUint32(reset[20:], uint32(width-1))
	binary.LittleEndian.PutUint32(reset[24:], uint32(height-1))
	binary.LittleEndian.PutUint32(reset[28:], 1)
	data = append(data, graphicsPDU(0x0e, reset)...)
	create := make([]byte, 7)
	put16(create, 2, width)
	put16(create, 4, height)
	create[6] = 0x20
	data = append(data, graphicsPDU(9, create)...)

	return append(data, graphicsPDU(0x0f, make([]byte, 12))...)
}

func avc420Frame(width, height int, frameID, timestamp uint32, annexB []byte) []byte {
	start := make([]byte, 8)
	binary.LittleEndian.PutUint32(start, timestamp)
	binary.LittleEndian.PutUint32(start[4:], frameID)
	data := graphicsPDU(0x0b, start)
	// WIRE_TO_SURFACE_1 followed by one full-screen AVC420 metadata rectangle.
	wire := make([]byte, 31+len(annexB))
	put16(wire, 2, 0x0b)
	wire[4] = 0x20
	put16(wire, 9, width)
	put16(wire, 11, height)
	binary.LittleEndian.PutUint32(wire[13:], uint32(14+len(annexB)))
	binary.LittleEndian.PutUint32(wire[17:], 1)
	put16(wire, 25, width)
	put16(wire, 27, height)
	// Quantization and quality are advisory metadata for progressive rendering.
	wire[29], wire[30] = 22, 100
	copy(wire[31:], annexB)
	data = append(data, graphicsPDU(1, wire)...)
	end := make([]byte, 4)
	binary.LittleEndian.PutUint32(end, frameID)

	return append(data, graphicsPDU(0x0c, end)...)
}
