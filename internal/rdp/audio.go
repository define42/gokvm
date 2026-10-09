//nolint:err113 // Protocol errors identify invalid audio data for diagnostics.
package rdp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"slices"
	"sync"
	"time"
)

const (
	audioMaxPCM      = 3840 // 20 ms of 48 kHz, stereo, signed 16-bit PCM.
	audioMaxIncoming = 65539
	audioMaxBlocks   = 8

	audioQualityTimeout   = time.Second
	audioHandshakeTimeout = 5 * time.Second

	audioFormats  = 7
	audioQuality  = 12
	audioTraining = 6
	audioConfirm  = 5
)

const (
	audioIdle byte = iota
	audioWaitFormats
	audioWaitQuality
	audioWaitTraining
	audioPlaying
)

// audioState implements the playback-only, reliable-channel subset of
// MS-RDPEA. State is per connection: reconnecting always renegotiates formats.
// mu protects negotiation, acknowledgments, and inbound reassembly.
type audioState struct {
	mu        sync.Mutex
	channel   uint16
	disabled  bool
	phase     byte
	version   uint16
	format    uint16
	block     byte
	inFlight  []byte
	changed   chan struct{}
	fragments fragmentBuffer
	deadline  time.Time
}

// EnableAudio starts optional RDPSND playback negotiation. A false result
// means that the client did not request redirected audio. ReadPacket must keep
// running to process negotiation and playback acknowledgments.
func (s *Session) EnableAudio() (bool, error) {
	a := s.audio
	if a == nil {
		return false, nil
	}
	a.mu.Lock()
	if a.disabled || a.channel == 0 || !s.joined[a.channel] {
		a.mu.Unlock()

		return false, nil
	}
	if a.phase != audioIdle {
		a.mu.Unlock()

		return true, nil
	}
	a.phase = audioWaitFormats
	a.deadline = time.Now().Add(audioHandshakeTimeout)
	a.mu.Unlock()

	return true, s.writeAudioMessage(audioPDU(audioFormats, serverAudioFormats()))
}

// AudioReady reports whether the client accepted the PCM format and completed
// training. Unsupported audio never prevents bitmap or graphics output.
func (s *Session) AudioReady() bool {
	if s.audio == nil {
		return false
	}
	s.audio.mu.Lock()
	defer s.audio.mu.Unlock()

	return s.audio.phase == audioPlaying
}

// AudioCanSend reports whether negotiation and the bounded acknowledgment
// window permit another packet. There must be only one audio writer.
func (s *Session) AudioCanSend() bool {
	if s.audio == nil {
		return false
	}
	s.audio.mu.Lock()
	defer s.audio.mu.Unlock()

	return s.audio.canSend()
}

// AudioChanged wakes the audio writer after negotiation or acknowledgments.
// Notifications coalesce; always check readiness after receiving a notification.
func (s *Session) AudioChanged() <-chan struct{} {
	if s.audio == nil {
		return nil
	}
	a := s.audio
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.changed == nil {
		a.changed = make(chan struct{}, 1)
	}

	return a.changed
}

// CheckAudioTimeout advances optional audio negotiation without creating any
// session-owned goroutines. The audio writer should call it periodically,
// including when the guest is silent. Missing formats/training disable only
// audio; an absent Quality Mode PDU falls back to training after one second.
func (s *Session) CheckAudioTimeout() error {
	return s.checkAudioTimeout(time.Now())
}

func (s *Session) checkAudioTimeout(now time.Time) error {
	a := s.audio
	if a == nil {
		return nil
	}
	a.mu.Lock()
	var response []byte
	if !a.disabled && !a.deadline.IsZero() && !now.Before(a.deadline) {
		switch a.phase {
		case audioWaitQuality:
			response = a.beginTraining(now)
		case audioWaitFormats, audioWaitTraining:
			a.disable()
		}
	}
	a.mu.Unlock()
	if response != nil {
		return s.writeAudioMessage(response)
	}

	return nil
}

func (a *audioState) disable() {
	a.disabled = true
	a.phase = audioIdle
	a.deadline = time.Time{}
	a.fragments = fragmentBuffer{}
	a.notifyChanged()
}

func (a *audioState) notifyChanged() {
	select {
	case a.changed <- struct{}{}:
	default:
	}
}

func (a *audioState) canSend() bool {
	return a.phase == audioPlaying && len(a.inFlight) < audioMaxBlocks
}

// WriteAudio sends up to 20 ms of interleaved signed 16-bit little-endian
// stereo PCM at 48 kHz. timestamp is the source's monotonic millisecond clock.
// The caller must pace playback and drop old queued samples when necessary.
// A false result means that negotiation or acknowledgments are pending; this
// method never waits for acknowledgment and never retains the caller's PCM.
func (s *Session) WriteAudio(pcm []byte, timestamp uint32) (bool, error) {
	if len(pcm) == 0 || len(pcm) > audioMaxPCM || len(pcm)%4 != 0 {
		return false, errors.New("rdp: invalid PCM audio length")
	}
	a := s.audio
	if a == nil {
		return false, nil
	}
	a.mu.Lock()
	if !a.canSend() {
		a.mu.Unlock()

		return false, nil
	}
	block, format, version := a.block, a.format, a.version
	a.block++
	a.inFlight = append(a.inFlight, block)
	a.mu.Unlock()

	// Do not hold audio state while writing: the reader can process display
	// input and ACKs concurrently with a slow network write.
	if version >= 8 {
		payload := make([]byte, 12+len(pcm))
		binary.LittleEndian.PutUint16(payload, uint16(timestamp))
		binary.LittleEndian.PutUint16(payload[2:], format)
		payload[4] = block
		binary.LittleEndian.PutUint32(payload[8:], timestamp)
		copy(payload[12:], pcm)

		return true, s.writeAudioMessage(audioPDU(13, payload))
	}

	// Older clients require WaveInfo with the first four sample bytes, then
	// a separate Wave PDU whose four-byte padding replaces those sample bytes.
	info := make([]byte, 16)
	info[0] = 2
	binary.LittleEndian.PutUint16(info[2:], uint16(8+len(pcm)))
	binary.LittleEndian.PutUint16(info[4:], uint16(timestamp))
	binary.LittleEndian.PutUint16(info[6:], format)
	info[8] = block
	copy(info[12:], pcm[:4])
	if err := s.writeAudioMessage(info); err != nil {
		return false, err
	}
	wave := make([]byte, len(pcm))
	copy(wave[4:], pcm[4:])

	return true, s.writeAudioMessage(wave)
}

func audioPDU(kind byte, data []byte) []byte {
	pdu := make([]byte, 4+len(data))
	pdu[0] = kind
	binary.LittleEndian.PutUint16(pdu[2:], uint16(len(data)))
	copy(pdu[4:], data)

	return pdu
}

func pcmAudioFormat() []byte {
	format := make([]byte, 18)
	binary.LittleEndian.PutUint16(format, 1) // WAVE_FORMAT_PCM
	binary.LittleEndian.PutUint16(format[2:], 2)
	binary.LittleEndian.PutUint32(format[4:], 48000)
	binary.LittleEndian.PutUint32(format[8:], 192000)
	binary.LittleEndian.PutUint16(format[12:], 4)
	binary.LittleEndian.PutUint16(format[14:], 16)

	return format
}

func serverAudioFormats() []byte {
	payload := make([]byte, 38)
	binary.LittleEndian.PutUint16(payload[14:], 1)
	binary.LittleEndian.PutUint16(payload[17:], 8)

	copy(payload[20:], pcmAudioFormat())

	return payload
}

func (s *Session) readAudioChannel(channel uint16, data []byte) error {
	a := s.audio
	if a == nil || channel != a.channel {
		return nil
	}
	a.mu.Lock()
	if a.phase == audioIdle || a.disabled {
		a.mu.Unlock()

		return nil
	}
	// RDPSND bodies have a 16-bit length, much smaller than graphics PDUs.
	// Bound before fragmentBuffer allocates the advertised static length.
	if len(data) < 8 || binary.LittleEndian.Uint32(data) > audioMaxIncoming {
		a.fragments = fragmentBuffer{}
		a.mu.Unlock()

		return nil
	}
	message, err := a.fragments.static(data)
	if err != nil {
		a.fragments = fragmentBuffer{}
	}
	var response []byte
	if err == nil && message != nil {
		response = a.readPDU(message)
	}
	a.mu.Unlock()
	if response != nil {
		return s.writeAudioMessage(response)
	}

	return nil
}

// MS-RDPEA 3.1.5 requires ignoring malformed, unknown, or out-of-sequence
// messages. Such messages must not disconnect an otherwise usable desktop.
func (a *audioState) readPDU(data []byte) []byte {
	return a.readPDUAt(data, time.Now())
}

func (a *audioState) readPDUAt(data []byte, now time.Time) []byte {
	if a.disabled || len(data) < 4 || int(binary.LittleEndian.Uint16(data[2:])) != len(data)-4 {
		return nil
	}
	kind, payload := data[0], data[4:]
	switch kind {
	case audioFormats:
		if a.phase != audioWaitFormats {
			return nil
		}
		format, version, supported, err := selectAudioFormat(payload)
		if err != nil {
			return nil
		}
		if !supported {
			a.disable()

			return nil
		}
		a.format, a.version = format, version
		if version >= 6 {
			a.phase = audioWaitQuality
			a.deadline = now.Add(audioQualityTimeout)

			return nil
		}

		return a.beginTraining(now)
	case audioQuality:
		if a.phase != audioWaitQuality || len(payload) != 4 || binary.LittleEndian.Uint16(payload) > 2 {
			return nil
		}

		return a.beginTraining(now)
	case audioTraining:
		if a.phase == audioWaitTraining && bytes.Equal(payload, []byte{1, 0, 0, 0}) {
			a.phase = audioPlaying
			a.deadline = time.Time{}
			a.notifyChanged()
		}
	case audioConfirm:
		if a.phase != audioPlaying || len(payload) != 4 {
			return nil
		}
		if index := slices.Index(a.inFlight, payload[2]); index >= 0 {
			a.inFlight = slices.Delete(a.inFlight, index, index+1)
			a.notifyChanged()
		}
	}

	return nil
}

func (a *audioState) beginTraining(now time.Time) []byte {
	a.phase = audioWaitTraining
	a.deadline = now.Add(audioHandshakeTimeout)
	// Reliable-channel training needs no bandwidth probe payload. The client
	// echoes this timestamp and zero-byte payload length in Training Confirm.
	return audioPDU(audioTraining, []byte{1, 0, 0, 0})
}

func selectAudioFormat(data []byte) (uint16, uint16, bool, error) {
	if len(data) < 20 {
		return 0, 0, false, errors.New("rdp: truncated client audio formats")
	}
	flags, count := binary.LittleEndian.Uint32(data), int(binary.LittleEndian.Uint16(data[14:]))
	version := binary.LittleEndian.Uint16(data[17:])
	if count > 256 {
		return 0, 0, false, errors.New("rdp: excessive client audio formats")
	}
	data = data[20:]
	var selected uint16
	found := false
	for i := range count {
		if len(data) < 18 {
			return 0, 0, false, errors.New("rdp: truncated client audio format")
		}
		size := 18 + int(binary.LittleEndian.Uint16(data[16:]))
		if size > len(data) {
			return 0, 0, false, errors.New("rdp: truncated audio format extension")
		}
		if !found && bytes.Equal(data[:size], pcmAudioFormat()) {
			selected, found = uint16(i), true
		}
		data = data[size:]
	}
	if len(data) != 0 {
		return 0, 0, false, errors.New("rdp: trailing client audio formats")
	}

	return selected, version, found && flags&1 != 0 && version >= 2, nil
}

// Audio messages exceed the default 1600-byte static-channel chunk. Fragment
// each message independently, carrying its full length on every fragment.
// Fragments from another static channel may safely interleave between writes.
func (s *Session) writeAudioMessage(data []byte) error {
	total := len(data)
	first := true
	for len(data) > 0 {
		size := min(1600, len(data))
		fragment := make([]byte, 8+size)
		binary.LittleEndian.PutUint32(fragment, uint32(total))
		var flags uint32
		if first {
			flags |= 1
		}
		if size == len(data) {
			flags |= 2
		}
		binary.LittleEndian.PutUint32(fragment[4:], flags)
		copy(fragment[8:], data[:size])
		if err := s.writeChannel(s.audio.channel, fragment); err != nil {
			return err
		}
		first = false
		data = data[size:]
	}

	return nil
}
