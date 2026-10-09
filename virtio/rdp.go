package virtio

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"errors"
	"image"
	"io"
	"log"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bobuhiro11/gokvm/internal/audio"
	"github.com/bobuhiro11/gokvm/internal/rdp"
	"github.com/bobuhiro11/gokvm/internal/rdp/avc"
)

const (
	rdpMaxClients  = 8
	rdpResizeDelay = 150 * time.Millisecond
)

// RDPConfig selects the server certificate, OpenH264 graphics, and audio playback.
type RDPConfig struct {
	TLS   *tls.Config
	H264  bool
	Audio bool
}

// RDPDisplay exports the guest console using TLS-secured RDP bitmap updates.
// It is a console server, without account authentication or NLA. Use a trusted
// network or an authenticated tunnel when allowing access beyond localhost.
type RDPDisplay struct {
	*framebuffer
	listener net.Listener
	tls      *tls.Config
	h264     bool
	audio    *audio.Hub
	connMu   sync.Mutex
	conns    map[net.Conn]struct{}
	wg       sync.WaitGroup
	once     sync.Once
	resizeMu sync.Mutex
	resize   func(int, int) error
	viewers  []*rdp.Session
}

// NewRDPDisplay starts an RDP listener with an ephemeral self-signed certificate.
func NewRDPDisplay(addr string) (*RDPDisplay, error) {
	return NewRDPDisplayWithTLS(addr, nil)
}

// NewRDPDisplayWithTLS uses the supplied server certificate, or generates one
// for this process when config is nil. TLS 1.2 or newer is always required.
func NewRDPDisplayWithTLS(addr string, config *tls.Config) (*RDPDisplay, error) {
	return NewRDPDisplayWithConfig(addr, RDPConfig{TLS: config})
}

// NewRDPDisplayWithConfig enables AVC420 for capable clients when H264 is set.
// Other clients retain bitmap updates. H264 requires the openh264 build tag.
func NewRDPDisplayWithConfig(addr string, options RDPConfig) (*RDPDisplay, error) {
	if options.H264 {
		// Fail before opening the listener if the native encoder is unavailable.
		encoder, err := avc.NewEncoder(1024, 768)
		if err != nil {
			return nil, err
		}
		encoder.Close()
	}
	config := options.TLS
	if config == nil {
		certificate, err := rdpCertificate()
		if err != nil {
			return nil, err
		}

		config = &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}
	} else {
		config = config.Clone()
		config.MinVersion = max(config.MinVersion, tls.VersionTLS12)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	display := &RDPDisplay{
		framebuffer: newFramebuffer(), listener: listener, tls: config,
		conns: make(map[net.Conn]struct{}), h264: options.H264,
	}
	if options.Audio {
		display.audio = audio.NewHub()
	}
	if options.H264 {
		display.linearInterval = time.Second / avc.FrameRate
	}
	display.wg.Add(1)
	go display.acceptLoop()

	return display, nil
}

func (d *RDPDisplay) Addr() string { return d.listener.Addr().String() }

// SetResizeHandler attaches the guest GPU. The first connected viewer controls
// its resolution; other viewers display that shared desktop at their own size.
func (d *RDPDisplay) SetResizeHandler(resize func(int, int) error) {
	d.resizeMu.Lock()
	defer d.resizeMu.Unlock()
	d.resize = resize
	if len(d.viewers) != 0 {
		d.resizeGuestLocked(d.viewers[0])
	}
}

func (d *RDPDisplay) addResizeViewer(session *rdp.Session) {
	d.resizeMu.Lock()
	defer d.resizeMu.Unlock()
	d.viewers = append(d.viewers, session)
	if len(d.viewers) == 1 {
		d.resizeGuestLocked(session)
	}
}

func (d *RDPDisplay) removeResizeViewer(session *rdp.Session) {
	d.resizeMu.Lock()
	defer d.resizeMu.Unlock()
	for i, viewer := range d.viewers {
		if viewer != session {
			continue
		}
		copy(d.viewers[i:], d.viewers[i+1:])
		d.viewers[len(d.viewers)-1] = nil
		d.viewers = d.viewers[:len(d.viewers)-1]
		if i == 0 && len(d.viewers) != 0 {
			d.resizeGuestLocked(d.viewers[0])
		}

		return
	}
}

func (d *RDPDisplay) resizeGuest(session *rdp.Session) {
	d.resizeMu.Lock()
	defer d.resizeMu.Unlock()
	if len(d.viewers) != 0 && d.viewers[0] == session {
		d.resizeGuestLocked(session)
	}
}

// resizeMu orders GPU mode requests against controller disconnect/handoff.
// The callback must not call back into viewer management.
func (d *RDPDisplay) resizeGuestLocked(session *rdp.Session) {
	if d.resize != nil {
		width, height := session.Size()
		if err := d.resize(width, height); err != nil {
			log.Printf("rdp: guest display resize to %dx%d: %v", width, height, err)
		}
	}
}

// WritePCM receives borrowed, clocked PCM from the virtual sound card.
func (d *RDPDisplay) WritePCM(pcm []byte) {
	if d.audio != nil {
		d.audio.WritePCM(pcm)
	}
}

func (d *RDPDisplay) Close() error {
	var err error
	d.once.Do(func() {
		d.shutdown()
		if d.audio != nil {
			d.audio.Close()
		}
		err = d.listener.Close()
		d.connMu.Lock()
		for conn := range d.conns {
			_ = conn.Close()
		}
		d.connMu.Unlock()
		d.wg.Wait()
	})

	return err
}

func (d *RDPDisplay) acceptLoop() {
	defer d.wg.Done()
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			return
		}

		d.connMu.Lock()
		select {
		case <-d.done:
			d.connMu.Unlock()
			_ = conn.Close()

			return
		default:
		}
		if len(d.conns) >= rdpMaxClients {
			d.connMu.Unlock()
			_ = conn.Close()

			continue
		}
		d.conns[conn] = struct{}{}
		d.wg.Add(1)
		d.connMu.Unlock()
		go d.handleConn(conn)
	}
}

func (d *RDPDisplay) handleConn(conn net.Conn) {
	defer d.wg.Done()
	defer func() {
		_ = conn.Close()
		d.connMu.Lock()
		delete(d.conns, conn)
		d.connMu.Unlock()
	}()

	if err := d.serveConn(conn); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
		log.Printf("rdp: client %s: %v", conn.RemoteAddr(), err)
	}
}

func (d *RDPDisplay) serveConn(conn net.Conn) (serveErr error) {
	session, err := rdp.Accept(conn, d.tls, 1024, 768)
	if err != nil {
		return err
	}
	defer session.Close()
	withAudio := false
	if d.audio != nil {
		withAudio, err = session.EnableAudio()
		if err != nil {
			return err
		}
	}
	if _, err := session.BeginDisplayControl(); err != nil {
		return err
	}

	if d.h264 {
		if _, err := session.BeginGraphics(); err != nil {
			return err
		}
	}
	d.addResizeViewer(session)
	defer d.removeResizeViewer(session)

	// The guest draws its own cursor, as it does on the VNC console.
	if err := session.WriteDataPDU(27, []byte{1, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		return err
	}

	var decoder rdp.InputDecoder
	defer func() { d.dispatchInput(session, decoder.ReleaseAll()) }()
	refresh := make(chan struct{}, 1)
	stop := make(chan struct{})
	writerDone := make(chan error, 2)
	writers := 1
	var suppressed atomic.Bool
	go func() {
		writerDone <- d.writeFrames(conn, session, stop, refresh, &suppressed)
		_ = session.Close()
	}()
	if withAudio {
		writers++
		go func() {
			writerDone <- d.writeAudio(session, stop)
			_ = session.Close()
		}()
	}
	defer func() {
		close(stop)
		_ = session.Close()
		for range writers {
			writerErr := <-writerDone
			if writerErr != nil && (serveErr == nil || errors.Is(serveErr, io.EOF) || errors.Is(serveErr, net.ErrClosed)) {
				serveErr = writerErr
			}
		}
	}()

	for {
		data, fastPath, err := session.ReadPacket()
		if err != nil {
			return err
		}
		if !fastPath && len(data) >= 18 && binary.LittleEndian.Uint16(data[2:])&15 == 7 {
			switch data[14] {
			case 33: // Refresh Rect: a complete repaint is always a valid response.
				requestRDPRefresh(refresh)
			case 35: // Suppress Output (minimized client).
				if len(data) >= 22 {
					suppressed.Store(data[18] == 0)
					if data[18] == 1 {
						requestRDPRefresh(refresh)
					}
				}
			}
		}
		if !fastPath && (len(data) < 18 || data[14] != 28) {
			continue
		}

		events, err := decoder.Decode(data, fastPath)
		if err != nil {
			return err
		}
		d.dispatchInput(session, events)
	}
}

func (d *RDPDisplay) writeAudio(session *rdp.Session, stop <-chan struct{}) error {
	sub := d.audio.Subscribe()
	defer sub.Close()
	changed := session.AudioChanged()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	announced := false
	for {
		if !announced && session.AudioReady() {
			log.Print("rdp: client using PCM audio (48 kHz, stereo, 16-bit)")
			announced = true
		}
		select {
		case <-stop:
			return nil
		case <-d.done:
			return nil
		case <-changed:
		case <-ticker.C:
			if err := session.CheckAudioTimeout(); err != nil {
				return err
			}
		case packet, ok := <-sub.Packets:
			if !ok {
				return nil
			}
			if !session.AudioCanSend() {
				continue
			}
			if _, err := session.WriteAudio(packet.PCM, packet.Timestamp); err != nil {
				return err
			}
		}
	}
}

func requestRDPRefresh(refresh chan<- struct{}) {
	select {
	case refresh <- struct{}{}:
	default:
	}
}

func (d *RDPDisplay) writeFrames(
	conn net.Conn, session *rdp.Session,
	stop, refresh <-chan struct{}, suppressed *atomic.Bool,
) error {
	width, height := session.Size()
	bitmap, err := rdp.NewBitmapEncoder(width, height, session.BitsPerPixel)
	if err != nil {
		return err
	}
	writer := &rdpFrameWriter{session: session, bitmap: bitmap, width: width, height: height, force: true}
	defer writer.close()
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	graphicsChanged := session.GraphicsChanged()
	displayChanged := session.DisplayChanged()
	resizeRequests := session.ResizeRequests()
	resizeTimer := time.NewTimer(time.Hour)
	resizeTimer.Stop()
	defer resizeTimer.Stop()
	var resizeDue <-chan time.Time
	var pending rdp.DesktopSize
	resizeReady := false

	// Coalesce publications while pacing or awaiting acknowledgments. Only arm
	// the timer for a pending frame; idle clients need no periodic polling.
	for {
		if resizeReady && session.DisplayReady() {
			resized, resizeErr := session.Resize(pending.Width, pending.Height)
			if resizeErr != nil {
				return resizeErr
			}
			resizeReady = false
			if resized {
				d.resizeGuest(session)
			}
		}
		if err := writer.updateSize(); err != nil {
			return err
		}
		if writer.updateGraphics(conn) {
			writer.force = true
		}
		frame, changed, published := d.changedFrame(writer.sequence, writer.force)
		var paced <-chan time.Time
		if changed && session.DisplayReady() && !suppressed.Load() && (!writer.graphics || session.GraphicsCanSend()) {
			paced, err = writer.writeWhenReady(frame, timer)
			if err != nil {
				return err
			}
		}

		select {
		case <-stop:
			return nil
		case <-d.done:
			return nil
		case <-refresh:
			writer.force = true
		case <-published:
		case <-graphicsChanged:
		case <-displayChanged:
		case pending = <-resizeRequests:
			resizeReady = false
			resizeTimer.Reset(rdpResizeDelay)
			resizeDue = resizeTimer.C
		case <-resizeDue:
			resizeDue = nil
			resizeReady = true
		case <-paced:
		}
		timer.Stop()
	}
}

type rdpFrameWriter struct {
	session   *rdp.Session
	bitmap    *rdp.BitmapEncoder
	avc       *avc.Encoder
	width     int
	height    int
	graphics  bool
	sequence  uint64
	force     bool
	nextFrame time.Time
}

// Encoders retain geometry and the previous frame. Discard both after a resize
// so the new surface starts with a complete bitmap or an AVC keyframe.
func (w *rdpFrameWriter) updateSize() error {
	width, height := w.session.Size()
	if width == w.width && height == w.height {
		return nil
	}
	bitmap, err := rdp.NewBitmapEncoder(width, height, w.session.BitsPerPixel)
	if err != nil {
		return err
	}
	w.close()
	w.avc = nil
	w.bitmap = bitmap
	w.width, w.height = width, height
	w.force = true
	w.nextFrame = time.Time{}

	return nil
}

func (w *rdpFrameWriter) writeWhenReady(frame vncFrame, timer *time.Timer) (<-chan time.Time, error) {
	if delay := time.Until(w.nextFrame); delay > 0 {
		timer.Reset(delay)

		return timer.C, nil
	}
	w.nextFrame = time.Now().Add(w.frameInterval())
	sent, err := w.write(frame, w.force)
	if err != nil {
		return nil, err
	}
	w.force = !sent
	if sent {
		w.sequence = frame.seq
	}

	return nil, nil //nolint:nilnil // A nil timer channel keeps an idle writer asleep until an event.
}

func (w *rdpFrameWriter) updateGraphics(conn net.Conn) bool {
	ready := w.session.GraphicsReady()
	if ready == w.graphics {
		return false
	}
	w.graphics = ready
	if ready {
		log.Printf("rdp: client %s using OpenH264 AVC420 graphics", conn.RemoteAddr())
	}

	return true
}

func (w *rdpFrameWriter) frameInterval() time.Duration {
	if w.graphics {
		return time.Second / avc.FrameRate
	}

	return time.Second / 30
}

func (w *rdpFrameWriter) close() {
	if w.avc != nil {
		w.avc.Close()
	}
}

func (w *rdpFrameWriter) write(frame vncFrame, force bool) (bool, error) {
	img := &image.RGBA{Pix: frame.pix, Stride: frame.width * 4, Rect: image.Rect(0, 0, frame.width, frame.height)}
	if len(frame.pix) == 0 {
		img = image.NewRGBA(img.Rect)
	}
	if !w.graphics {
		err := w.bitmap.WriteFrame(img, force, func(data []byte) error {
			return w.session.WriteDataPDU(2, data)
		})

		return err == nil, err
	}
	if w.avc == nil {
		width, height := w.session.Size()
		encoder, err := avc.NewEncoder(width, height)
		if err != nil {
			return false, err
		}
		w.avc = encoder
	}
	data, err := w.avc.Encode(img, force)
	if err != nil || len(data) == 0 {
		return err == nil, err
	}

	return w.session.WriteAVC420(data)
}

func (d *RDPDisplay) dispatchInput(session *rdp.Session, events []rdp.InputEvent) {
	for _, event := range events {
		switch event.Kind {
		case rdp.InputKey:
			d.sendKeyEvent(event.Down, event.Key)
		case rdp.InputPointer:
			d.mu.Lock()
			width, height := d.width, d.height
			d.mu.Unlock()
			clientWidth, clientHeight := session.Size()
			x := min(int(event.X), clientWidth-1) * width / clientWidth
			y := min(int(event.Y), clientHeight-1) * height / clientHeight
			d.sendPointerEvent(event.Buttons, uint16(x), uint16(y))
		}
	}
}

func rdpCertificate() (tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: "gokvm console"},
		NotBefore: now.Add(-time.Minute), NotAfter: now.Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"}, IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}
