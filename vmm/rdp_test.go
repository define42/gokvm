package vmm

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestRDPPerformanceConfigBeforeMachineInit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		config Config
		want   error
	}{
		{name: "negative-threads", config: Config{RDPH264: true, RDPH264Threads: -1}, want: errRDPH264ThreadsConfig},
		{name: "excessive-threads", config: Config{RDPH264: true, RDPH264Threads: 17}, want: errRDPH264ThreadsConfig},
		{
			name:   "threads-without-h264",
			config: Config{RDP: "127.0.0.1:0", RDPH264Threads: 2},
			want:   errRDPH264ThreadsConfig,
		},
		{name: "stats-without-rdp", config: Config{RDPStats: true}, want: errRDPStatsConfig},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.config.Dev = "/missing-kvm-device"
			v := New(tc.config)
			if err := v.Init(); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want configuration error %v before opening KVM", err, tc.want)
			}
			if v.Machine != nil {
				t.Fatal("invalid RDP configuration created a machine")
			}
		})
	}
}

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
