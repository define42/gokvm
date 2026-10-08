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

	"github.com/bobuhiro11/gokvm/internal/rdp"
)

const rdpMaxClients = 8

// RDPDisplay exports the guest console using TLS-secured RDP bitmap updates.
// It is a console server, without account authentication or NLA. Use a trusted
// network or an authenticated tunnel when allowing access beyond localhost.
type RDPDisplay struct {
	*framebuffer
	listener net.Listener
	tls      *tls.Config
	connMu   sync.Mutex
	conns    map[net.Conn]struct{}
	wg       sync.WaitGroup
	once     sync.Once
}

// NewRDPDisplay starts an RDP listener with an ephemeral self-signed certificate.
func NewRDPDisplay(addr string) (*RDPDisplay, error) {
	return NewRDPDisplayWithTLS(addr, nil)
}

// NewRDPDisplayWithTLS uses the supplied server certificate, or generates one
// for this process when config is nil. TLS 1.2 or newer is always required.
func NewRDPDisplayWithTLS(addr string, config *tls.Config) (*RDPDisplay, error) {
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
		conns: make(map[net.Conn]struct{}),
	}
	display.wg.Add(1)
	go display.acceptLoop()

	return display, nil
}

func (d *RDPDisplay) Addr() string { return d.listener.Addr().String() }

func (d *RDPDisplay) Close() error {
	var err error
	d.once.Do(func() {
		d.shutdown()
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

func (d *RDPDisplay) serveConn(conn net.Conn) error {
	session, err := rdp.Accept(conn, d.tls, 1024, 768)
	if err != nil {
		return err
	}
	defer session.Close()

	encoder, err := rdp.NewBitmapEncoder(session.Width, session.Height, session.BitsPerPixel)
	if err != nil {
		return err
	}

	// The guest draws its own cursor, as it does on the VNC console.
	if err := session.WriteDataPDU(27, []byte{1, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		return err
	}

	var decoder rdp.InputDecoder
	defer func() { d.dispatchInput(session, decoder.ReleaseAll()) }()
	refresh := make(chan struct{}, 1)
	stop := make(chan struct{})
	writerDone := make(chan struct{})
	var suppressed atomic.Bool
	go func() {
		defer close(writerDone)
		defer session.Close()
		_ = d.writeFrames(conn, session, encoder, stop, refresh, &suppressed)
	}()
	defer func() {
		close(stop)
		_ = session.Close()
		<-writerDone
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

func requestRDPRefresh(refresh chan<- struct{}) {
	select {
	case refresh <- struct{}{}:
	default:
	}
}

func (d *RDPDisplay) writeFrames(
	conn net.Conn, session *rdp.Session, encoder *rdp.BitmapEncoder,
	stop, refresh <-chan struct{}, suppressed *atomic.Bool,
) error {
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()

	var sequence uint64
	force := true
	for {
		if !suppressed.Load() {
			frame, changed := d.changedFrame(sequence, force)
			if changed {
				if err := writeRDPFrame(conn, session, encoder, frame, force); err != nil {
					return err
				}
				sequence = frame.seq
				force = false
			}
		}

		select {
		case <-stop:
			return nil
		case <-d.done:
			return nil
		case <-refresh:
			force = true
		case <-ticker.C:
		}
	}
}

func writeRDPFrame(conn net.Conn, session *rdp.Session, encoder *rdp.BitmapEncoder, frame vncFrame, force bool) error {
	img := &image.RGBA{Pix: frame.pix, Stride: frame.width * 4, Rect: image.Rect(0, 0, frame.width, frame.height)}
	if len(frame.pix) == 0 {
		img = image.NewRGBA(img.Rect)
	}
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	defer func() { _ = conn.SetWriteDeadline(time.Time{}) }()

	return encoder.WriteFrame(img, force, func(data []byte) error {
		return session.WriteDataPDU(2, data)
	})
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
			x := min(int(event.X), session.Width-1) * width / session.Width
			y := min(int(event.Y), session.Height-1) * height / session.Height
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
