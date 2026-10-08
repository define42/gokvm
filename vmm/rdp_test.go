package vmm

import (
	"net"
	"testing"
	"time"
)

func TestRDPDisplayConfiguration(t *testing.T) {
	t.Parallel()
	for _, both := range []bool{false, true} {
		name := "rdp"
		if both {
			name = "rdp-and-vnc"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			v := New(Config{RDP: "127.0.0.1:0"})
			want := 1
			if both {
				v.VNC, want = "127.0.0.1:0", 2
			}
			display, err := v.display(nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = display.Close() })
			if !v.hasRemoteDisplay() || len(v.consoles) != want || v.rdpDisplay == nil || v.serialOutput == nil {
				t.Fatalf("incomplete remote console configuration: %+v", v)
			}
			if _, err := v.serialOutput.Write([]byte("booting\n")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRDPFailureClosesVNCListener(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	v := New(Config{VNC: addr, RDP: "127.0.0.1:0", RDPCert: "missing.crt", RDPKey: "missing.key"})
	if _, err := v.display(nil); err == nil {
		t.Fatal("invalid certificate unexpectedly accepted")
	}
	if v.vncDisplay != nil || v.rdpDisplay != nil || len(v.consoles) != 0 {
		t.Fatal("partially constructed display retained")
	}
	rebound, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("VNC listener leaked after RDP failure: %v", err)
	}
	_ = rebound.Close()
}

func TestSerialMirrorDoesNotBlockRemoteInput(t *testing.T) {
	t.Parallel()
	serial := make(chan byte, 1)
	serial <- 'x'
	primary := &recordedKeyInput{keys: make(chan uint32, 1)}
	mirror := &serialMirrorInput{primary: primary, serial: serial}
	done := make(chan struct{})
	go func() {
		mirror.KeyEvent(true, 'a')
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		<-serial // Release a blocked writer before reporting the regression.
		t.Fatal("full serial buffer blocked remote input")
	}
	if got := <-primary.keys; got != 'a' {
		t.Fatalf("primary keyboard event: %d", got)
	}
	if got := <-serial; got != 'x' {
		t.Fatalf("buffered serial byte changed: %q", got)
	}
}

type recordedKeyInput struct{ keys chan uint32 }

func (r *recordedKeyInput) KeyEvent(_ bool, keysym uint32)    { r.keys <- keysym }
func (r *recordedKeyInput) PointerEvent(_ uint8, _, _ uint16) {}
