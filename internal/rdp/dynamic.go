//nolint:err113 // Protocol errors identify malformed wire structures for diagnostic logging.
package rdp

import (
	"encoding/binary"
	"errors"
)

// fragmentBuffer bounds both static-channel and dynamic-channel reassembly.
// The buffer is only allocated after its declared length has been validated.
type fragmentBuffer struct {
	total uint32
	data  []byte
}

func (b *fragmentBuffer) first(total uint32, data []byte) ([]byte, error) {
	if b.total != 0 || total == 0 || total > graphicsMaxIncoming || uint64(len(data)) > uint64(total) {
		return nil, errors.New("rdp: invalid channel fragment length or sequence")
	}
	if uint32(len(data)) == total {
		return data, nil
	}
	b.total = total
	b.data = make([]byte, 0, int(total))
	b.data = append(b.data, data...)

	return nil, nil
}

func (b *fragmentBuffer) next(data []byte) ([]byte, error) {
	if b.total == 0 {
		if len(data) == 0 || len(data) > graphicsMaxIncoming {
			return nil, errors.New("rdp: invalid dynamic channel data length")
		}

		return data, nil
	}
	if len(data) == 0 || uint64(len(b.data))+uint64(len(data)) > uint64(b.total) {
		return nil, errors.New("rdp: channel fragment exceeds declared length")
	}
	b.data = append(b.data, data...)
	if uint32(len(b.data)) != b.total {
		return nil, nil
	}
	message := b.data
	b.data, b.total = nil, 0

	return message, nil
}

func (b *fragmentBuffer) static(data []byte) ([]byte, error) {
	if len(data) < 8 {
		return nil, errors.New("rdp: truncated static channel header")
	}
	total, flags := binary.LittleEndian.Uint32(data), binary.LittleEndian.Uint32(data[4:])
	if flags&0x00ff0000 != 0 {
		return nil, errors.New("rdp: unsupported static channel compression")
	}
	var message []byte
	var err error
	if flags&1 != 0 {
		message, err = b.first(total, data[8:])
	} else {
		if b.total == 0 || total != b.total {
			return nil, errors.New("rdp: unexpected static channel continuation")
		}
		message, err = b.next(data[8:])
	}
	if err != nil {
		return nil, err
	}
	if (message != nil) != (flags&2 != 0) {
		return nil, errors.New("rdp: static channel last-fragment mismatch")
	}

	return message, nil
}

func readDynamicInteger(data []byte, code byte) (uint32, []byte, error) {
	if code > 2 || len(data) < 1<<code {
		return 0, nil, errors.New("rdp: invalid dynamic channel integer")
	}
	switch code {
	case 0:
		return uint32(data[0]), data[1:], nil
	case 1:
		return uint32(binary.LittleEndian.Uint16(data)), data[2:], nil
	default:
		return binary.LittleEndian.Uint32(data), data[4:], nil
	}
}

func (s *Session) writeStatic(channel uint16, data []byte) error {
	// Each dynamic fragment is at most 1600 bytes, so the default static
	// channel chunk size suffices and each has FIRST|LAST set.
	if len(data) > 1600 {
		return errors.New("rdp: static channel write exceeds chunk size")
	}
	payload := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint32(payload, uint32(len(data)))
	binary.LittleEndian.PutUint32(payload[4:], 3)
	copy(payload[8:], data)

	return s.writeChannel(channel, payload)
}

func (s *Session) writeGraphics(data []byte) error {
	data = segmentGraphics(data)
	total := len(data)
	first := true
	for len(data) > 0 {
		header := []byte{0x30, graphicsChannelID}
		if first && len(data) > 1598 {
			header = []byte{0x28, graphicsChannelID, 0, 0, 0, 0}
			binary.LittleEndian.PutUint32(header[2:], uint32(total))
		}
		size := min(1600-len(header), len(data))
		if err := s.writeStatic(s.graphics.channel, append(header, data[:size]...)); err != nil {
			return err
		}
		data = data[size:]
		first = false
	}

	return nil
}

// segmentGraphics emits uncompressed RDP 8.0 bulk segments. H.264 already
// compresses the payload; the mandatory ZGFX envelope still applies.
func segmentGraphics(data []byte) []byte {
	const maxSegment = 65535
	if len(data) <= maxSegment {
		return append([]byte{0xe0, 0x04}, data...)
	}
	count := (len(data) + maxSegment - 1) / maxSegment
	output := make([]byte, 7, 7+len(data)+count*5)
	output[0] = 0xe1
	binary.LittleEndian.PutUint16(output[1:], uint16(count))
	binary.LittleEndian.PutUint32(output[3:], uint32(len(data)))
	for len(data) > 0 {
		size := min(maxSegment, len(data))
		header := make([]byte, 5)
		binary.LittleEndian.PutUint32(header, uint32(size+1))
		header[4] = 4
		output = append(output, header...)
		output = append(output, data[:size]...)
		data = data[size:]
	}

	return output
}
