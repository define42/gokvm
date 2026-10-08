package flag_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/bobuhiro11/gokvm/flag"
)

func TestParsesize(t *testing.T) { // nolint:paralleltest
	for _, tt := range []struct {
		name string
		unit string
		m    string
		amt  int
		err  error
	}{
		{name: "badsuffix", m: "1T", amt: -1, err: strconv.ErrSyntax},
		{name: "1G", m: "1G", amt: 1 << 30, err: nil},
		{name: "1g", m: "1g", amt: 1 << 30, err: nil},
		{name: "1M", m: "1M", amt: 1 << 20, err: nil},
		{name: "1m", m: "1m", amt: 1 << 20, err: nil},
		{name: "1K", m: "1K", amt: 1 << 10, err: nil},
		{name: "1k", m: "1k", amt: 1 << 10, err: nil},
		{name: "1 with unit k", m: "1", unit: "k", amt: 1 << 10, err: nil},
		{name: "1 with unit \"\"", m: "1", unit: "", amt: 1, err: nil},
		{name: "8192m", m: "8192m", amt: 8192 << 20, err: nil},
		{name: "bogusgarbage", m: "123411;3413234134", amt: -1, err: strconv.ErrSyntax},
		{name: "bogusgarbagemsuffix", m: "123411;3413234134m", amt: -1, err: strconv.ErrSyntax},
		{name: "bogustoobig", m: "0xfffffffffffffffffffffff", amt: -1, err: strconv.ErrRange},
	} {
		amt, err := flag.ParseSize(tt.m, tt.unit)
		if !errors.Is(err, tt.err) || amt != tt.amt {
			t.Errorf("%s:parseMemSize(%s): got (%d, %v), want (%d, %v)", tt.name, tt.m, amt, err, tt.amt, tt.err)
		}
	}
}

func TestParseBootArgs(t *testing.T) {
	t.Parallel()

	args := []string{
		"gokvm",
		"boot",
		"-i",
		"initrd_path",
		"-k",
		"kernel_path",
		"-iso",
		"iso_path",
		"-p",
		"params",
		"-t",
		"tap_if_name",
		"-c",
		"2",
		"-d",
		"disk_path",
		"-vnc",
		"127.0.0.1:5900",
		"-rdp", "127.0.0.1:3389",
		"-rdp-cert", "console.crt",
		"-rdp-key", "console.key",
		"-rdp-h264",
		"-m",
		"1G",
		"-T",
		"1M",
	}

	c, _, err := flag.ParseArgs(args)
	if err != nil {
		t.Fatal(err)
	}

	if c.Dev != "/dev/kvm" {
		t.Error("invalid kvm  path")
	}

	if c.Kernel != "kernel_path" {
		t.Error("invalid kernel image path")
	}

	if c.Initrd != "initrd_path" {
		t.Error("invalid initrd path")
	}

	if c.ISO != "iso_path" {
		t.Errorf("invalid ISO path: got %v, want iso_path", c.ISO)
	}

	if c.Params != "params" {
		t.Error("invalid kernel command-line parameters")
	}

	if !c.ParamsSet {
		t.Error("kernel command-line should be marked explicitly set")
	}

	if c.TapIfName != "tap_if_name" {
		t.Error("invalid name of tap interface")
	}

	if c.Disk != "disk_path" {
		t.Errorf("invalid path of disk file: got %v, want %v", c.Disk, "disk_path")
	}

	if c.VNC != "127.0.0.1:5900" {
		t.Errorf("invalid VNC listen address: got %v, want %v", c.VNC, "127.0.0.1:5900")
	}
	if c.RDP != "127.0.0.1:3389" || c.RDPCert != "console.crt" || c.RDPKey != "console.key" || !c.RDPH264 {
		t.Errorf("invalid RDP options: %+v", c)
	}

	if c.NCPUs != 2 {
		t.Error("invalid number of vcpus")
	}

	if c.MemSize != 1<<30 {
		t.Errorf("msize: got %#x, want %#x", c.MemSize, 1<<30)
	}

	if c.TraceCount != 1<<20 {
		t.Errorf("trace: got %#x, want %#x", c.TraceCount, 1<<20)
	}
}

func TestParseBootArgsWithDefaults(t *testing.T) {
	t.Parallel()

	args := []string{
		"gokvm",
		"boot",
	}

	c, _, err := flag.ParseArgs(args)
	if err != nil {
		t.Fatal(err)
	}

	if c.Dev != "/dev/kvm" {
		t.Error("invalid kvm path")
	}

	if c.Kernel != "./bzImage" {
		t.Error("invalid kernel image path")
	}

	if c.Initrd != "" {
		t.Error("invalid initrd path")
	}

	if c.ISO != "" {
		t.Error("invalid ISO path")
	}

	if c.Params != `console=tty0 console=ttyS0 earlyprintk=serial `+
		`noapic noacpi notsc nowatchdog `+
		`nmi_watchdog=0 debug apic=debug show_lapic=all mitigations=off `+
		`lapic tsc_early_khz=2000 `+
		`dyndbg="file arch/x86/kernel/smpboot.c +plf ; file drivers/net/virtio_net.c +plf" `+
		`pci=realloc=off `+
		`virtio_pci.force_legacy=1 rdinit=/init init=/init `+
		`gokvm.ipv4_addr=192.168.20.1/24` {
		t.Error("invalid kernel command-line parameters")
	}

	if c.TapIfName != "" {
		t.Error("invalid name of tap interface")
	}

	if c.ParamsSet {
		t.Error("kernel command-line should not be marked explicitly set")
	}

	if c.Disk != "" {
		t.Errorf("invalid path of disk file: got %v, want %v", c.Disk, "disk_path")
	}

	if c.VNC != "" {
		t.Errorf("invalid VNC listen address: got %v, want empty", c.VNC)
	}
	if c.RDP != "" || c.RDPCert != "" || c.RDPKey != "" || c.RDPH264 {
		t.Errorf("RDP must be disabled by default: %+v", c)
	}

	if c.NCPUs != 1 {
		t.Error("invalid number of vcpus")
	}

	if c.MemSize != 1<<30 {
		t.Errorf("msize: got %#x, want %#x", c.MemSize, 1<<30)
	}

	if c.TraceCount != 0 {
		t.Errorf("trace: got %#x, want %#x", c.TraceCount, 1<<20)
	}
}

func TestRDPRequiresCertificatePair(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"-rdp", "127.0.0.1:3389", "-rdp-cert", "console.crt"},
		{"-rdp", "127.0.0.1:3389", "-rdp-key", "console.key"},
		{"-rdp-cert", "console.crt", "-rdp-key", "console.key"},
	} {
		if _, _, err := flag.ParseArgs(append([]string{"gokvm", "boot"}, args...)); !errors.Is(err, flag.ErrRDPOptions) {
			t.Errorf("args %v: got %v, want %v", args, err, flag.ErrRDPOptions)
		}
	}
}

func TestRDPH264RequiresListener(t *testing.T) {
	t.Parallel()
	if _, _, err := flag.ParseArgs([]string{"gokvm", "boot", "-rdp-h264"}); !errors.Is(err, flag.ErrRDPH264) {
		t.Fatalf("H.264 without RDP listener: got %v, want %v", err, flag.ErrRDPH264)
	}
}

func TestParseProbeArgs(t *testing.T) {
	t.Parallel()

	args := []string{
		"gokvm",
		"probe",
	}

	_, probeConfig, err := flag.ParseArgs(args)
	if err != nil {
		t.Fatal(err)
	}

	if probeConfig == nil {
		t.Fatal("probeConfig is nil")
	}
}
