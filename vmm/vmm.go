package vmm

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/bobuhiro11/gokvm/iso9660"
	"github.com/bobuhiro11/gokvm/machine"
	"github.com/bobuhiro11/gokvm/pvh"
	"github.com/bobuhiro11/gokvm/term"
	"github.com/bobuhiro11/gokvm/virtio"
	"golang.org/x/sync/errgroup"
)

var errDownloadISO = errors.New("download ISO failed")

const isoCompatibilityParams = "console=tty0 console=ttyS0 earlyprintk=serial " +
	"noapic noacpi notsc nowatchdog nmi_watchdog=0 mitigations=off " +
	"lapic tsc_early_khz=2000 pci=realloc=off virtio_pci.force_legacy=1"

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
			input = m.AddPS2Input()
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

	var initrd *os.File
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
		initrd, err = os.Open(v.Initrd)
		if err != nil {
			return err
		}
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
	defer cleanup()

	info, err := isoFile.Stat()
	if err != nil {
		return err
	}

	isoReader, err := iso9660.NewReader(isoFile, info.Size())
	if err != nil {
		return err
	}

	files, err := iso9660.LoadBootFiles(isoReader)
	if err != nil {
		return err
	}

	params := v.Params
	if !v.ParamsSet {
		params = isoCmdline(files.Cmdline)
	}

	log.Printf("ISO boot: kernel=%s initrd=%s", files.KernelPath, files.InitrdPath)

	kern := bytes.NewReader(files.Kernel)
	initrd := readerAtOrNil(files.Initrd)

	isPVH, err := pvh.CheckPVH(kern)
	if err != nil {
		return err
	}

	if isPVH {
		if err := v.LoadPVH(kern, initrd, params); err != nil {
			return err
		}
	} else if err := v.LoadLinux(kern, initrd, params); err != nil {
		return err
	}

	v.attachSerialOutput()
	v.attachSerialInput()

	return nil
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

func readerAtOrNil(data []byte) io.ReaderAt {
	if len(data) == 0 {
		return nil
	}

	return bytes.NewReader(data)
}

func isoCmdline(cmdline string) string {
	fields := strings.Fields(cmdline)

	compatFields := strings.Fields(isoCompatibilityParams)
	merged := make([]string, 0, len(compatFields)+len(fields))
	for _, param := range compatFields {
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

func (v *VMM) Boot() error {
	var err error

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
