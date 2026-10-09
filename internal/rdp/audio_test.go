package rdp

import (
	"bytes"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"
)

func TestAudioStaticChannelNegotiation(t *testing.T) {
	t.Parallel()
	settings, err := parseConnectInitialDetails(graphicsConnectFixture(false, []string{"cliprdr", "rdpsnd"}))
	if err != nil || settings.audioChannel != 1005 || settings.graphics {
		t.Fatalf("audio without graphics = %+v, %v", settings, err)
	}
	if _, err := parseConnectInitialDetails(graphicsConnectFixture(true, []string{"rdpsnd", "rdpsnd"})); err == nil {
		t.Fatal("accepted duplicate audio channel")
	}
	traffic := bytes.Join([][]byte{
		graphicsInputFixture(audioFormatsFixture(8, 1, pcmAudioFormat())),
		graphicsInputFixture(audioPDU(audioQuality, []byte{2, 0, 0, 0})),
		graphicsInputFixture(audioPDU(audioTraining, []byte{1, 0, 0, 0})),
		{4, 4, 0, 0x1e},
	}, nil)
	conn := &graphicsTestConn{incoming: bytes.NewReader(traffic)}
	s := &Session{conn: conn, audio: &audioState{channel: 1004}, joined: map[uint16]bool{1004: true}}
	changed := s.AudioChanged()
	if changed != s.AudioChanged() {
		t.Fatal("notification channel changed")
	}
	if started, err := s.EnableAudio(); !started || err != nil {
		t.Fatalf("enable audio = %v, %v", started, err)
	}
	if started, err := s.EnableAudio(); !started || err != nil {
		t.Fatalf("repeated enable audio = %v, %v", started, err)
	}
	if s.AudioReady() {
		t.Fatal("audio ready before client response")
	}
	data, fast, err := s.ReadPacket()
	if err != nil || !fast || !bytes.Equal(data, []byte{4, 0, 0x1e}) {
		t.Fatalf("input after audio negotiation = %x, %v, %v", data, fast, err)
	}
	if !s.AudioReady() || !s.AudioCanSend() {
		t.Fatal("audio not ready after training")
	}
	select {
	case <-changed:
	default:
		t.Fatal("negotiation did not notify writer")
	}
	output := audioOutputFixture(t, conn.outgoing.Bytes())
	if len(output) != 2 || !bytes.Equal(output[0], audioPDU(audioFormats, serverAudioFormats())) ||
		!bytes.Equal(output[1], audioPDU(audioTraining, []byte{1, 0, 0, 0})) {
		t.Fatalf("unexpected audio negotiation: %x", output)
	}
}

func TestClientInfoAudioPreferences(t *testing.T) {
	t.Parallel()
	for _, flags := range []uint32{0x10, 0x80010, 0x2010} {
		conn := &graphicsTestConn{}
		s := &Session{conn: conn, Width: 640, Height: 480, audio: &audioState{channel: 1004}}
		a := activation{session: s}
		info := make([]byte, 32)
		info[0] = 0x40
		binary.LittleEndian.PutUint32(info[8:], flags)
		if _, err := a.globalData(info); err != nil {
			t.Fatal(err)
		}
		if s.audio.disabled != (flags != 0x10) {
			t.Fatalf("audio preference flags %#x: disabled=%v", flags, s.audio.disabled)
		}
	}
}

func TestAudioQualityTimeoutFallsBackToTraining(t *testing.T) {
	t.Parallel()
	now := time.Unix(100, 0)
	conn := &graphicsTestConn{}
	a := &audioState{channel: 1004, phase: audioWaitFormats}
	s := &Session{conn: conn, audio: a}
	a.readPDUAt(audioFormatsFixture(8, 1, pcmAudioFormat()), now)
	if !a.deadline.Equal(now.Add(audioQualityTimeout)) {
		t.Fatal("quality deadline was not set after format negotiation")
	}
	if err := s.checkAudioTimeout(a.deadline.Add(-time.Nanosecond)); err != nil || conn.outgoing.Len() != 0 {
		t.Fatalf("quality fallback fired early: %v", err)
	}
	if err := s.checkAudioTimeout(now.Add(audioQualityTimeout)); err != nil {
		t.Fatal(err)
	}
	output := audioOutputFixture(t, conn.outgoing.Bytes())
	if len(output) != 1 || !bytes.Equal(output[0], audioPDU(audioTraining, []byte{1, 0, 0, 0})) ||
		a.phase != audioWaitTraining || !a.deadline.Equal(now.Add(audioQualityTimeout+audioHandshakeTimeout)) {
		t.Fatal("quality timeout did not start one bounded training exchange")
	}
	if reply := a.readPDUAt(audioPDU(audioQuality, []byte{2, 0, 0, 0}), now.Add(2*time.Second)); reply != nil {
		t.Fatal("late quality PDU repeated training")
	}
	a.readPDUAt(audioPDU(audioTraining, []byte{1, 0, 0, 0}), now.Add(3*time.Second))
	if !s.AudioReady() || !a.deadline.IsZero() {
		t.Fatal("training after quality timeout did not enable playback")
	}
	if err := s.checkAudioTimeout(now.Add(time.Hour)); err != nil || !s.AudioReady() {
		t.Fatal("old negotiation deadline disabled active playback")
	}
}

func TestAudioNegotiationTimeoutPreservesDesktop(t *testing.T) {
	t.Parallel()
	if err := new(Session).CheckAudioTimeout(); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []byte{audioWaitFormats, audioWaitTraining} {
		conn := &graphicsTestConn{}
		now := time.Unix(100, 0)
		a := &audioState{channel: 1004, phase: phase, deadline: now.Add(audioHandshakeTimeout)}
		s := &Session{conn: conn, audio: a}
		changed := s.AudioChanged()
		if err := s.checkAudioTimeout(now.Add(audioHandshakeTimeout - time.Nanosecond)); err != nil || a.disabled {
			t.Fatalf("negotiation disabled before deadline: %v", err)
		}
		if err := s.checkAudioTimeout(now.Add(audioHandshakeTimeout)); err != nil || !a.disabled ||
			s.AudioReady() || s.AudioCanSend() || !a.deadline.IsZero() {
			t.Fatalf("negotiation did not disable audio at deadline: %v", err)
		}
		select {
		case <-changed:
		default:
			t.Fatal("audio timeout did not notify writer")
		}
		if err := s.checkAudioTimeout(now.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		select {
		case <-changed:
			t.Fatal("disabled audio kept emitting timeout notifications")
		default:
		}
		a.readPDUAt(audioFormatsFixture(8, 1, pcmAudioFormat()), now.Add(time.Hour))
		a.readPDUAt(audioPDU(audioTraining, []byte{1, 0, 0, 0}), now.Add(time.Hour))
		if s.AudioReady() || !a.disabled {
			t.Fatal("late negotiation revived expired audio")
		}
		if err := s.WriteDataPDU(0x1f, []byte{1, 0, 0xe9, 3}); err != nil || conn.outgoing.Len() == 0 {
			t.Fatalf("audio timeout prevented desktop output: %v", err)
		}
	}
}

func TestAudioQualityRacesTimeoutOnlySendsTrainingOnce(t *testing.T) {
	t.Parallel()
	conn := &graphicsTestConn{}
	now := time.Unix(100, 0)
	s := &Session{conn: conn, audio: &audioState{channel: 1004, phase: audioWaitQuality, deadline: now}}
	quality := audioPDU(audioQuality, []byte{1, 0, 0, 0})
	fragment := make([]byte, 8+len(quality))
	binary.LittleEndian.PutUint32(fragment, uint32(len(quality)))
	binary.LittleEndian.PutUint32(fragment[4:], 3)
	copy(fragment[8:], quality)
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := s.readAudioChannel(1004, fragment); err != nil {
			t.Errorf("read quality PDU: %v", err)
		}
	})
	wg.Go(func() {
		if err := s.checkAudioTimeout(now); err != nil {
			t.Errorf("check timeout: %v", err)
		}
	})
	wg.Wait()
	if output := audioOutputFixture(t, conn.outgoing.Bytes()); len(output) != 1 || output[0][0] != audioTraining {
		t.Fatalf("quality/timeout race produced %d training messages", len(output))
	}
}

func TestAudioWaveEncodingAndFragmentation(t *testing.T) {
	t.Parallel()
	pcm := make([]byte, audioMaxPCM)
	for i := range pcm {
		pcm[i] = byte(i)
	}
	for _, version := range []uint16{5, 8} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			t.Parallel()
			conn := &graphicsTestConn{}
			s := &Session{conn: conn, audio: &audioState{channel: 1004, phase: audioPlaying, version: version, format: 2}}
			if sent, err := s.WriteAudio(pcm, 0x12345678); !sent || err != nil {
				t.Fatalf("write audio = %v, %v", sent, err)
			}
			output := audioOutputFixture(t, conn.outgoing.Bytes())
			var decoded []byte
			if version == 8 {
				if len(output) != 1 || len(output[0]) != len(pcm)+16 || output[0][0] != 13 ||
					binary.LittleEndian.Uint16(output[0][2:]) != uint16(len(pcm)+12) ||
					binary.LittleEndian.Uint32(output[0][12:]) != 0x12345678 {
					t.Fatalf("invalid Wave2 framing: %x", output[0][:16])
				}
				decoded = output[0][16:]
			} else {
				if len(output) != 2 || len(output[0]) != 16 || output[0][0] != 2 ||
					binary.LittleEndian.Uint16(output[0][2:]) != uint16(len(pcm)+8) ||
					!bytes.Equal(output[1][:4], make([]byte, 4)) {
					t.Fatalf("invalid legacy Wave framing: %x", output)
				}
				decoded = append(append([]byte(nil), output[0][12:]...), output[1][4:]...)
			}
			if binary.LittleEndian.Uint16(output[0][4:]) != 0x5678 ||
				binary.LittleEndian.Uint16(output[0][6:]) != 2 || output[0][8] != 0 || !bytes.Equal(decoded, pcm) {
				t.Fatal("timestamp, format, block, or PCM did not round-trip")
			}
		})
	}
}

func TestAudioAcknowledgmentsBoundPendingPlayback(t *testing.T) {
	t.Parallel()
	conn := &graphicsTestConn{}
	s := &Session{conn: conn, audio: &audioState{channel: 1004, phase: audioPlaying, version: 8}}
	changed := s.AudioChanged()
	for range audioMaxBlocks {
		if sent, err := s.WriteAudio(make([]byte, 4), 12); !sent || err != nil {
			t.Fatalf("write audio = %v, %v", sent, err)
		}
	}
	before := conn.outgoing.Len()
	sent, err := s.WriteAudio(make([]byte, 4), 13)
	if sent || err != nil || s.AudioCanSend() || conn.outgoing.Len() != before {
		t.Fatalf("ACK window did not stop output: %v, %v", sent, err)
	}
	s.audio.readPDU(audioPDU(audioConfirm, []byte{12, 0, 99, 0}))
	if s.AudioCanSend() {
		t.Fatal("unknown block acknowledgment released window")
	}
	s.audio.readPDU(audioPDU(audioConfirm, []byte{12, 0, 0, 0}))
	if !s.AudioCanSend() || len(s.audio.inFlight) != audioMaxBlocks-1 {
		t.Fatal("valid acknowledgment did not release window")
	}
	select {
	case <-changed:
	default:
		t.Fatal("ACK did not notify writer")
	}
	// FreeRDP sends both an arrival and a playback ACK for the same block.
	s.audio.readPDU(audioPDU(audioConfirm, []byte{32, 0, 0, 0}))
	if len(s.audio.inFlight) != audioMaxBlocks-1 {
		t.Fatal("duplicate acknowledgment released another block")
	}
	for _, block := range append([]byte(nil), s.audio.inFlight...) {
		s.audio.readPDU(audioPDU(audioConfirm, []byte{12, 0, block, 0}))
	}
	for range 260 {
		block := s.audio.block
		if sent, err := s.WriteAudio(make([]byte, 4), 20); !sent || err != nil {
			t.Fatalf("block counter wrap write = %v, %v", sent, err)
		}
		s.audio.readPDU(audioPDU(audioConfirm, []byte{20, 0, block, 0}))
	}
	if len(s.audio.inFlight) != 0 {
		t.Fatal("block counter wrap leaked acknowledgments")
	}
	// A fresh Session must not inherit the previous connection's ready state.
	fresh := &Session{audio: &audioState{channel: 1004}}
	if fresh.AudioReady() || fresh.AudioCanSend() || fresh.audio.block != 0 {
		t.Fatal("new connection inherited audio state")
	}
}

func TestAudioUnsupportedAndMalformedMessages(t *testing.T) {
	t.Parallel()
	for _, state := range []*audioState{nil, {channel: 1004, disabled: true}, {channel: 1005}} {
		s := &Session{audio: state, joined: map[uint16]bool{1004: true}}
		if started, err := s.EnableAudio(); started || err != nil || s.AudioReady() || s.AudioCanSend() {
			t.Fatalf("unavailable audio enabled: %v, %v", started, err)
		}
		if sent, err := s.WriteAudio(make([]byte, 4), 0); sent || err != nil {
			t.Fatalf("unavailable audio write = %v, %v", sent, err)
		}
	}
	if new(Session).AudioChanged() != nil {
		t.Fatal("missing audio has notifications")
	}
	for _, version := range []uint16{5, 8} {
		a := &audioState{phase: audioWaitFormats}
		a.readPDU(audioFormatsFixture(version, 1, pcmAudioFormat()))
		if version >= 6 {
			if a.phase != audioWaitQuality {
				t.Fatal("version 8 skipped quality negotiation")
			}
			a.readPDU(audioPDU(audioQuality, []byte{1, 0, 0, 0}))
		}
		for _, data := range [][]byte{
			nil, {6}, {6, 0, 4, 0}, audioPDU(99, nil), audioPDU(audioTraining, []byte{2, 0, 0, 0}),
		} {
			a.readPDU(data)
			if a.phase != audioWaitTraining {
				t.Fatal("malformed training advanced negotiation")
			}
		}
		a.readPDU(audioPDU(audioTraining, []byte{1, 0, 0, 0}))
		if a.phase != audioPlaying {
			t.Fatal("valid training did not recover after malformed messages")
		}
	}
	for _, formats := range [][]byte{audioFormatsFixture(8, 0, pcmAudioFormat()), audioFormatsFixture(8, 1)} {
		a := &audioState{phase: audioWaitFormats}
		a.readPDU(formats)
		if !a.disabled || a.phase == audioPlaying {
			t.Fatal("unsupported audio was not disabled")
		}
	}
	s := &Session{audio: &audioState{channel: 1004, phase: audioWaitFormats}}
	oversized := make([]byte, 8)
	binary.LittleEndian.PutUint32(oversized, audioMaxIncoming+1)
	binary.LittleEndian.PutUint32(oversized[4:], 1)
	if err := s.readAudioChannel(1004, oversized); err != nil || s.audio.fragments.data != nil {
		t.Fatal("oversized input allocated memory or terminated desktop")
	}
	for _, pcm := range [][]byte{nil, {1}, make([]byte, audioMaxPCM+4)} {
		if _, err := s.WriteAudio(pcm, 0); err == nil {
			t.Fatal("accepted invalid PCM length")
		}
	}
}

func TestAudioSelectClientFormat(t *testing.T) {
	t.Parallel()
	other := pcmAudioFormat()
	binary.LittleEndian.PutUint32(other[4:], 44100)
	data := audioFormatsFixture(8, 1, other, pcmAudioFormat())[4:]
	index, version, supported, err := selectAudioFormat(data)
	if err != nil || !supported || index != 1 || version != 8 {
		t.Fatalf("format selection = %d, %d, %v, %v", index, version, supported, err)
	}
	for _, invalid := range [][]byte{nil, data[:19], data[:len(data)-1], append(append([]byte(nil), data...), 0)} {
		if _, _, _, err := selectAudioFormat(invalid); err == nil {
			t.Fatal("accepted truncated or trailing format data")
		}
	}
}

func TestSessionWriteDeadlinesSerialized(t *testing.T) {
	t.Parallel()
	conn := &audioDeadlineConn{}
	s := &Session{conn: conn, writeTimeout: time.Second}
	var writers sync.WaitGroup
	for range 4 {
		writers.Go(func() {
			for range 10 {
				if err := s.WriteDataPDU(0x1f, []byte{1, 0, 0xe9, 3}); err != nil {
					t.Errorf("write PDU: %v", err)
				}
			}
		})
	}
	writers.Wait()
	if conn.bad || conn.active || conn.writes != 40 {
		t.Fatalf("deadline/write serialization: bad=%v active=%v writes=%d", conn.bad, conn.active, conn.writes)
	}
}

type audioDeadlineConn struct {
	net.Conn
	active bool
	bad    bool
	writes int
}

func (c *audioDeadlineConn) SetWriteDeadline(deadline time.Time) error {
	if !deadline.IsZero() == c.active {
		c.bad = true
	}
	c.active = !deadline.IsZero()

	return nil
}

func (c *audioDeadlineConn) Write(data []byte) (int, error) {
	if !c.active {
		c.bad = true
	}
	c.writes++

	return len(data), nil
}

func FuzzAudioProtocol(f *testing.F) {
	f.Add(audioFormatsFixture(8, 1, pcmAudioFormat()))
	f.Add(audioPDU(audioConfirm, []byte{0, 0, 0, 0}))
	f.Add([]byte{7, 0, 0xff, 0xff})
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _, _, _ = selectAudioFormat(data)
		a := &audioState{phase: audioWaitFormats}
		a.readPDU(data)
		a.phase = audioWaitTraining
		a.readPDU(data)
		a.phase = audioPlaying
		a.readPDU(data)
	})
}

func audioFormatsFixture(version uint16, flags uint32, formats ...[]byte) []byte {
	data := make([]byte, 20, 20+18*len(formats))
	binary.LittleEndian.PutUint32(data, flags)
	binary.LittleEndian.PutUint16(data[14:], uint16(len(formats)))
	binary.LittleEndian.PutUint16(data[17:], version)
	for _, format := range formats {
		data = append(data, format...)
	}

	return audioPDU(audioFormats, data)
}

func audioOutputFixture(t *testing.T, data []byte) [][]byte {
	t.Helper()
	reader := bytes.NewReader(data)
	var messages [][]byte
	var fragments fragmentBuffer
	for reader.Len() > 0 {
		mcs, err := readMCS(reader)
		if err != nil {
			t.Fatal(err)
		}
		if len(mcs) < 7 || !bytes.Equal(mcs[:6], []byte{0x68, 0, 1, 3, 0xec, 0x70}) {
			t.Fatalf("invalid audio MCS framing: %x", mcs)
		}
		payload, err := takePER(mcs[6:])
		if err != nil || len(payload) > 1608 {
			t.Fatalf("invalid audio static chunk: len=%d, %v", len(payload), err)
		}
		message, err := fragments.static(payload)
		if err != nil {
			t.Fatal(err)
		}
		if message != nil {
			messages = append(messages, message)
		}
	}
	if fragments.total != 0 {
		t.Fatal("unterminated audio fragments")
	}

	return messages
}
