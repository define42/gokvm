package vmm

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bobuhiro11/gokvm/machine"
	"github.com/bobuhiro11/gokvm/pvh"
	"github.com/bobuhiro11/gokvm/term"
	"github.com/bobuhiro11/gokvm/virtio"
	"golang.org/x/sync/errgroup"
)

var errDownloadISO = errors.New("download ISO failed")

// These parameters describe gaps in gokvm's direct Linux boot environment, not
// ISO-specific policy. The ISO's own boot config still supplies the distro
// command line; this list only adds the host/VMM plumbing that a real firmware
// boot path would normally provide or hide from the guest.
const gokvmDirectLinuxBootParams = "console=tty0 console=ttyS0 earlyprintk=serial " +
	"noapic noacpi nosmp nortc nowatchdog nmi_watchdog=0 mitigations=off " +
	"lapic pci=realloc=off virtio_pci.force_legacy=1"

// Config defines the configuration of the
// virtual machine, as determined by flags.
type Config struct {
	Debug      bool
	Dev        string
	Kernel     string
	Initrd     string
	ISO        string
	Params     string
	ParamsSet  bool
	TapIfName  string
	Disk       string
	GPU        string
	VNC        string
	NCPUs      int
	MemSize    int
	TraceCount int
}

type VMM struct {
	*machine.Machine
	Config

	serialOutput io.Writer
	vncDisplay   *virtio.VNCDisplay
	vncInput     virtio.VNCInput
	isoCleanup   func()
}

func New(c Config) *VMM {
	return &VMM{
		Machine: nil,
		Config:  c,
	}
}

// Init instantiates a machine.
func (v *VMM) Init() error {
	m, err := machine.New(v.Dev, v.NCPUs, v.MemSize)
	if err != nil {
		return err
	}

	if len(v.TapIfName) > 0 {
		if err := m.AddTapIf(v.TapIfName); err != nil {
			return err
		}
	}

	if len(v.Disk) > 0 {
		if err := m.AddDisk(v.Disk); err != nil {
			return err
		}
	}

	if len(v.GPU) > 0 || len(v.VNC) > 0 {
		var input virtio.VNCInput
		if len(v.VNC) > 0 {
			input = m.AddVirtioInput()
			v.vncInput = input
		}

		display, err := v.display(input)
		if err != nil {
			return err
		}

		if err := m.AddGPUDisplay(display); err != nil {
			return err
		}

		if len(v.ISO) > 0 && v.vncDisplay != nil {
			m.EnableVESA(v.vncDisplay)
			m.StartVGATextFallback(v.vncDisplay)
		}
	}

	v.Machine = m

	return nil
}

func (v *VMM) display(input virtio.VNCInput) (virtio.Display, error) {
	var displays []virtio.Display

	if len(v.GPU) > 0 {
		displays = append(displays, virtio.NewPNGDisplay(v.GPU))
	}

	if len(v.VNC) > 0 {
		display, err := virtio.NewVNCDisplay(v.VNC)
		if err != nil {
			for _, d := range displays {
				_ = d.Close()
			}

			return nil, err
		}

		display.SetInput(input)
		v.serialOutput = display
		v.vncDisplay = display
		log.Printf("VNC listening on %s", display.Addr())
		displays = append(displays, display)
	}

	if len(displays) == 1 {
		return displays[0], nil
	}

	return virtio.NewMultiDisplay(displays...), nil
}

func (v *VMM) Setup() error {
	if v.ISO != "" {
		return v.setupISO()
	}

	var initrd io.ReaderAt
	// Kernel arg required to load kernel or firmware image
	kern, err := os.Open(v.Kernel)
	if err != nil {
		return err
	}

	isPVH, err := pvh.CheckPVH(kern)
	if err != nil {
		return err
	}

	if v.Initrd != "" {
		initrdFile, err := os.Open(v.Initrd)
		if err != nil {
			return err
		}

		initrd = initrdFile
	}

	if isPVH {
		if err := v.LoadPVH(kern, initrd, v.Params); err != nil {
			return err
		}
	} else {
		if err := v.LoadLinux(kern, initrd, v.Params); err != nil {
			return err
		}
	}

	v.attachSerialOutput()
	v.attachSerialInput()

	return nil
}

func (v *VMM) setupISO() error {
	isoFile, cleanup, err := openISOSource(v.ISO)
	if err != nil {
		return err
	}

	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			cleanup()
		}
	}()

	bios, biosPath, err := loadBIOSFirmware()
	if err != nil {
		return err
	}

	info, err := isoFile.Stat()
	if err != nil {
		return err
	}
	v.AddReadOnlyCDROM(isoFile, info.Size())
	log.Printf("ISO media attached as read-only ATAPI CD-ROM: %s", isoFile.Name())
	log.Printf("ISO firmware boot: BIOS=%s", biosPath)

	if err := v.LoadBIOS(bios); err != nil {
		return err
	}

	v.attachSerialOutput()
	v.attachSerialInput()
	v.isoCleanup = cleanup
	cleanupOnError = false

	return nil
}

func loadBIOSFirmware() ([]byte, string, error) {
	if name := os.Getenv("GOKVM_BIOS"); name != "" {
		bios, err := os.ReadFile(name) //nolint:gosec // User-selected local firmware path.
		if err != nil {
			return nil, "", err
		}

		return bios, name, nil
	}

	var errs []error
	for _, name := range []string{
		"./bios.bin",
		"/usr/share/seabios/bios.bin",
		"/usr/share/seabios/bios-256k.bin",
	} {
		bios, err := os.ReadFile(name) //nolint:gosec // Fixed firmware search path.
		if err == nil {
			return bios, name, nil
		}

		errs = append(errs, fmt.Errorf("%s: %w", name, err))
	}

	return nil, "", fmt.Errorf("BIOS firmware not found; set GOKVM_BIOS or install seabios: %w", errors.Join(errs...))
}

func openISOSource(source string) (*os.File, func(), error) {
	u, err := url.Parse(source)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return downloadISO(source)
	}

	file, err := os.Open(source)
	if err != nil {
		return nil, nil, err
	}

	return file, func() { _ = file.Close() }, nil
}

func downloadISO(source string) (*os.File, func(), error) {
	resp, err := http.Get(source) //nolint:gosec // User-supplied boot media URL.
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, nil, fmt.Errorf("%w %s: %s", errDownloadISO, source, resp.Status)
	}

	file, err := os.CreateTemp("", "gokvm-*.iso")
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		name := file.Name()
		_ = file.Close()
		_ = os.Remove(name)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		cleanup()

		return nil, nil, err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanup()

		return nil, nil, err
	}

	log.Printf("downloaded ISO %s to %s", source, file.Name())

	return file, cleanup, nil
}

func isoBootParams(cmdline string) string {
	return mergeKernelParams(cmdline, strings.Fields(gokvmDirectLinuxBootParams))
}

func mergeKernelParams(cmdline string, defaults []string) string {
	fields := strings.Fields(cmdline)
	merged := make([]string, 0, len(defaults)+len(fields))

	for _, param := range defaults {
		if !hasKernelParam(fields, param) {
			merged = append(merged, param)
		}
	}

	merged = append(merged, fields...)

	return strings.Join(merged, " ")
}

func hasKernelParam(fields []string, param string) bool {
	paramName := kernelParamName(param)
	for _, field := range fields {
		if paramName == "console" {
			if field == param {
				return true
			}

			continue
		}

		if kernelParamName(field) == paramName {
			return true
		}
	}

	return false
}

func kernelParamName(field string) string {
	name, _, _ := strings.Cut(field, "=")

	return name
}

func (v *VMM) attachSerialOutput() {
	if v.serialOutput == nil || v.GetSerial() == nil {
		return
	}

	v.GetSerial().SetOutput(io.MultiWriter(os.Stdout, v.serialOutput))
}

func (v *VMM) attachSerialInput() {
	if v.ISO == "" || v.vncDisplay == nil || v.GetSerial() == nil {
		return
	}

	v.vncDisplay.SetInput(&serialMirrorInput{
		primary: v.vncInput,
		serial:  v.GetSerial().GetInputChan(),
		inject:  v.InjectSerialIRQ,
	})
}

type serialMirrorInput struct {
	primary virtio.VNCInput
	serial  chan<- byte
	inject  func() error
}

func (s *serialMirrorInput) KeyEvent(down bool, keysym uint32) {
	if s.primary != nil {
		s.primary.KeyEvent(down, keysym)
	}

	if !down || s.serial == nil {
		return
	}

	for _, b := range serialBytesForKeysym(keysym) {
		s.serial <- b
		if s.inject != nil {
			_ = s.inject()
		}
	}
}

func (s *serialMirrorInput) PointerEvent(buttonMask uint8, x, y uint16) {
	if s.primary != nil {
		s.primary.PointerEvent(buttonMask, x, y)
	}
}

func serialBytesForKeysym(keysym uint32) []byte {
	if keysym >= 0x20 && keysym <= 0x7e {
		return []byte{byte(keysym)}
	}

	switch keysym {
	case 0xff08: // BackSpace
		return []byte{0x7f}
	case 0xff09: // Tab
		return []byte{'\t'}
	case 0xff0d: // Return
		return []byte{'\r'}
	case 0xff1b: // Escape
		return []byte{0x1b}
	case 0xff51: // Left
		return []byte{0x1b, '[', 'D'}
	case 0xff52: // Up
		return []byte{0x1b, '[', 'A'}
	case 0xff53: // Right
		return []byte{0x1b, '[', 'C'}
	case 0xff54: // Down
		return []byte{0x1b, '[', 'B'}
	default:
		return nil
	}
}

func addTinyCoreVNCAutostart(initrd []byte) ([]byte, error) {
	const autologin = `#!/bin/sh
if [ -f /var/log/autologin ]; then
	exec /sbin/getty 38400 tty1
fi
touch /var/log/autologin
TCUSER="$(cat /etc/sysconfig/tcuser 2>/dev/null || echo tc)"
if command -v Xvesa >/dev/null 2>&1 && \
	command -v flwm >/dev/null 2>&1 && \
	[ ! -f /etc/sysconfig/text ]; then
	[ -s /etc/sysconfig/Xserver ] || echo Xvesa > /etc/sysconfig/Xserver
	[ -s /etc/sysconfig/desktop ] || echo flwm > /etc/sysconfig/desktop
	[ -s /etc/sysconfig/icons ] || echo wbar > /etc/sysconfig/icons
	for file in .xsession .setbackground .Xdefaults; do
		if [ ! -e /home/"$TCUSER"/"$file" ] && [ -e /etc/skel/"$file" ]; then
			cp /etc/skel/"$file" /home/"$TCUSER"/
			chown "$TCUSER":staff /home/"$TCUSER"/"$file" 2>/dev/null || chown "$TCUSER" /home/"$TCUSER"/"$file"
		fi
	done
	cat > /tmp/gokvm-vnc-session <<'EOF'
#!/bin/sh
export DISPLAY=:0.0
export DESKTOP=flwm
export ICONS=wbar
Xvesa -br -screen 1024x768x32 -mouse /dev/input/mice,5 -a 1 -t 0 -nolisten tcp -I >/tmp/Xvesa.log 2>&1 &
XPID=$!
for _ in 1 2 3 4 5 6 7 8 9 10; do
	waitforX >/dev/null 2>&1 && break
	sleep 0.2
done
flwm >/tmp/flwm.log 2>&1 &
[ -x "$HOME/.setbackground" ] && "$HOME/.setbackground" >/tmp/background.log 2>&1
wbar.sh >/tmp/wbar.log 2>&1 &
wait "$XPID"
EOF
	chmod 755 /tmp/gokvm-vnc-session
	chown "$TCUSER":staff /tmp/gokvm-vnc-session 2>/dev/null || chown "$TCUSER" /tmp/gokvm-vnc-session
	exec su "$TCUSER" -c /tmp/gokvm-vnc-session
fi
exec login -f "$TCUSER"
`

	return appendInitramfsFile(initrd, "sbin/autologin", []byte(autologin), 0o100755)
}

func appendInitramfsFile(initrd []byte, name string, data []byte, mode uint32) ([]byte, error) {
	raw := initrd
	compressed := len(initrd) >= 2 && initrd[0] == 0x1f && initrd[1] == 0x8b
	if compressed {
		zr, err := gzip.NewReader(bytes.NewReader(initrd))
		if err != nil {
			return nil, fmt.Errorf("initramfs gzip: %w", err)
		}

		raw, err = io.ReadAll(zr)
		closeErr := zr.Close()
		if err != nil {
			return nil, fmt.Errorf("initramfs decompress: %w", err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("initramfs gzip close: %w", closeErr)
		}
	}

	if trimmed, ok := trimNewcTrailer(raw); ok {
		raw = trimmed
	}

	buf := bytes.NewBuffer(make([]byte, 0, len(raw)+len(data)+512))
	buf.Write(raw)
	writeNewcEntry(buf, name, mode, data)
	writeNewcEntry(buf, "TRAILER!!!", 0, nil)

	if !compressed {
		return buf.Bytes(), nil
	}

	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	if _, err := zw.Write(buf.Bytes()); err != nil {
		return nil, fmt.Errorf("initramfs recompress: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("initramfs gzip finish: %w", err)
	}

	return out.Bytes(), nil
}

func trimNewcTrailer(data []byte) ([]byte, bool) {
	for off := 0; off+110 <= len(data); {
		if string(data[off:off+6]) != "070701" {
			return nil, false
		}

		filesize, ok := parseNewcHex(data[off+54 : off+62])
		if !ok {
			return nil, false
		}

		namesize, ok := parseNewcHex(data[off+94 : off+102])
		if !ok || namesize == 0 {
			return nil, false
		}

		nameStart := off + 110
		nameEnd := nameStart + int(namesize)
		if nameEnd > len(data) {
			return nil, false
		}

		name := string(bytes.TrimRight(data[nameStart:nameEnd], "\x00"))
		next := align4(nameEnd) + align4(int(filesize))
		if next > len(data) {
			return nil, false
		}

		if name == "TRAILER!!!" {
			return data[:off], true
		}

		off = next
	}

	return nil, false
}

func writeNewcEntry(buf *bytes.Buffer, name string, mode uint32, data []byte) {
	namesize := len(name) + 1
	fmt.Fprintf(buf,
		"070701%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x",
		1, mode, 0, 0, 1, 0, len(data), 0, 0, 0, 0, namesize, 0,
	)
	buf.WriteString(name)
	buf.WriteByte(0)
	padBuffer(buf)
	buf.Write(data)
	padBuffer(buf)
}

func parseNewcHex(data []byte) (uint64, bool) {
	v, err := strconv.ParseUint(string(data), 16, 64)

	return v, err == nil
}

func align4(n int) int {
	return (n + 3) &^ 3
}

func padBuffer(buf *bytes.Buffer) {
	for buf.Len()%4 != 0 {
		buf.WriteByte(0)
	}
}

func (v *VMM) Boot() error {
	var err error
	defer v.cleanupISO()

	trace := v.TraceCount > 0
	if err := v.SingleStep(trace); err != nil {
		return fmt.Errorf("setting trace to %v:%w", trace, err)
	}

	g := new(errgroup.Group)

	for cpu := 0; cpu < v.NCPUs; cpu++ {
		fmt.Printf("Start CPU %d of %d\r\n", cpu, v.NCPUs)

		i := cpu

		f := func() error {
			return v.VCPU(os.Stderr, i, v.TraceCount)
		}

		g.Go(f)
	}

	if !term.IsTerminal() {
		fmt.Fprintln(os.Stderr, "this is not terminal and does not accept input")
		select {}
	}

	restoreMode, err := term.SetRawMode()
	if err != nil {
		return err
	}

	defer restoreMode()

	if err := v.SingleStep(trace); err != nil {
		log.Printf("SingleStep(%v): %v", trace, err)

		return err
	}

	in := bufio.NewReader(os.Stdin)

	g.Go(func() error {
		err := v.GetSerial().Start(*in, restoreMode, v.InjectSerialIRQ)
		log.Printf("Serial exits: %v", err)

		return err
	})

	fmt.Printf("Waiting for CPUs to exit\r\n")

	if err := g.Wait(); err != nil {
		log.Print(err)
	}

	fmt.Printf("All cpus done\n\r")

	return nil
}

func (v *VMM) cleanupISO() {
	if v.isoCleanup == nil {
		return
	}

	v.isoCleanup()
	v.isoCleanup = nil
}
