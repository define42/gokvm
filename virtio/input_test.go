package virtio

import (
	"encoding/binary"
	"testing"
)

func TestInputGetDeviceHeader(t *testing.T) {
	t.Parallel()

	v := NewInputKeyboard(5, func() error { return nil }, nil)
	hdr := v.GetDeviceHeader()

	if hdr.DeviceID != 0x1052 {
		t.Fatalf("DeviceID: got 0x%x, want 0x1052", hdr.DeviceID)
	}

	if hdr.SubsystemID != virtioInputDeviceID {
		t.Fatalf("SubsystemID: got %d, want %d", hdr.SubsystemID, virtioInputDeviceID)
	}

	if hdr.ClassCode != 0x09 || hdr.Subclass != 0x80 {
		t.Fatalf("class: got %#x/%#x, want 0x09/0x80", hdr.ClassCode, hdr.Subclass)
	}

	if hdr.Status&0x10 == 0 {
		t.Fatal("capabilities-list status bit not set")
	}

	if hdr.BAR[0] != InputKeyboardMMIOBase {
		t.Fatalf("BAR0: got 0x%x, want 0x%x", hdr.BAR[0], InputKeyboardMMIOBase)
	}
}

func TestInputKeyboardConfig(t *testing.T) {
	t.Parallel()

	v := NewInputKeyboard(5, func() error { return nil }, nil)

	cfg := readInputConfig(v, inputCfgIDName, 0)
	if string(cfg[inputUnionOff:inputUnionOff+cfg[2]]) != "gokvm keyboard" {
		t.Fatalf("name: got %q", cfg[inputUnionOff:inputUnionOff+cfg[2]])
	}

	cfg = readInputConfig(v, inputCfgIDDevids, 0)
	if cfg[2] != 8 {
		t.Fatalf("devids size: got %d, want 8", cfg[2])
	}

	if bus := binary.LittleEndian.Uint16(cfg[inputUnionOff:]); bus != busVirtual {
		t.Fatalf("bus: got %d, want %d", bus, busVirtual)
	}

	cfg = readInputConfig(v, inputCfgEvBits, evKey)
	if !inputBitSet(cfg[inputUnionOff:], keyA) {
		t.Fatal("keyboard EV_KEY bitmap does not include KEY_A")
	}

	if !inputBitSet(cfg[inputUnionOff:], keyEnter) {
		t.Fatal("keyboard EV_KEY bitmap does not include KEY_ENTER")
	}

	if inputBitSet(cfg[inputUnionOff:], btnLeft) {
		t.Fatal("keyboard EV_KEY bitmap unexpectedly includes BTN_LEFT")
	}
}

func TestInputPointerConfig(t *testing.T) {
	t.Parallel()

	v := NewInputPointer(6, func() error { return nil }, nil)

	cfg := readInputConfig(v, inputCfgPropBits, 0)
	if cfg[2] != 0 {
		t.Fatalf("pointer prop bitmap size: got %d, want 0", cfg[2])
	}

	cfg = readInputConfig(v, inputCfgEvBits, evKey)
	if !inputBitSet(cfg[inputUnionOff:], btnLeft) ||
		!inputBitSet(cfg[inputUnionOff:], btnMiddle) ||
		!inputBitSet(cfg[inputUnionOff:], btnRight) {
		t.Fatal("pointer EV_KEY bitmap does not include all primary buttons")
	}

	cfg = readInputConfig(v, inputCfgEvBits, evRel)
	if !inputBitSet(cfg[inputUnionOff:], relX) ||
		!inputBitSet(cfg[inputUnionOff:], relY) ||
		!inputBitSet(cfg[inputUnionOff:], relWheel) {
		t.Fatal("pointer EV_REL bitmap does not include REL_X/REL_Y/REL_WHEEL")
	}

	cfg = readInputConfig(v, inputCfgAbsInfo, 0)
	if cfg[2] != 0 {
		t.Fatalf("ABS info size: got %d, want 0", cfg[2])
	}
}

func TestInputKeyboardDeliversEvents(t *testing.T) {
	t.Parallel()

	var interrupts int
	mem := make([]byte, 0x1000)
	v := NewInputKeyboard(5, func() error {
		interrupts++

		return nil
	}, mem)

	q := newInputSplitQueue()
	queueInputBuffer(q, 0, 0x100)
	queueInputBuffer(q, 1, 0x108)
	v.QueueReady(inputEventQueue, q)

	v.KeyEvent(true, 'a')

	if err := v.flushEvents(); err != nil {
		t.Fatalf("flush key event: %v", err)
	}

	if err := v.flushEvents(); err != nil {
		t.Fatalf("flush sync event: %v", err)
	}

	assertInputEvent(t, mem[0x100:0x108], evKey, keyA, 1)
	assertInputEvent(t, mem[0x108:0x110], evSyn, synReport, 0)

	if got := LoadU16(&q.Used.Idx); got != 2 {
		t.Fatalf("used idx: got %d, want 2", got)
	}

	if interrupts != 2 {
		t.Fatalf("interrupts: got %d, want 2", interrupts)
	}
}

func TestInputPointerDeliversRelativeButtonAndWheelEvents(t *testing.T) {
	t.Parallel()

	mem := make([]byte, 0x1000)
	v := NewInputPointer(6, func() error { return nil }, mem)
	q := newInputSplitQueue()

	for i := 0; i < 5; i++ {
		queueInputBuffer(q, uint16(i), uint64(0x100+i*inputEventLen))
	}

	v.QueueReady(inputEventQueue, q)
	v.PointerEvent(0, 100, 100)
	v.PointerEvent(0x09, 200, 120)

	for i := 0; i < 5; i++ {
		if err := v.flushEvents(); err != nil {
			t.Fatalf("flush event %d: %v", i, err)
		}
	}

	assertInputEvent(t, mem[0x100:0x108], evRel, relX, 100)
	assertInputEvent(t, mem[0x108:0x110], evRel, relY, 20)
	assertInputEvent(t, mem[0x110:0x118], evKey, btnLeft, 1)
	assertInputEvent(t, mem[0x118:0x120], evRel, relWheel, 1)
	assertInputEvent(t, mem[0x120:0x128], evSyn, synReport, 0)
}

func TestInputKeyCodeFunctionKeys(t *testing.T) {
	t.Parallel()

	code, ok := inputKeyCode(0xffc8)
	if !ok || code != keyF11 {
		t.Fatalf("F11: got (%d, %v), want (%d, true)", code, ok, keyF11)
	}

	code, ok = inputKeyCode(0xffc9)
	if !ok || code != keyF12 {
		t.Fatalf("F12: got (%d, %v), want (%d, true)", code, ok, keyF12)
	}
}

func readInputConfig(v *InputDevice, sel, subsel uint8) [inputConfigLen]byte {
	v.WriteDeviceConfig(0, []byte{sel})
	v.WriteDeviceConfig(1, []byte{subsel})

	var cfg [inputConfigLen]byte
	v.ReadDeviceConfig(0, cfg[:])

	return cfg
}

func newInputSplitQueue() *SplitQueue {
	return &SplitQueue{
		Desc:  &[QueueSize]SplitDesc{},
		Avail: &SplitAvail{},
		Used:  &SplitUsed{},
	}
}

func queueInputBuffer(q *SplitQueue, descID uint16, addr uint64) {
	q.Desc[descID] = SplitDesc{Addr: addr, Len: inputEventLen, Flags: descFWrite}
	q.Avail.Ring[q.Avail.Idx%QueueSize] = descID
	q.Avail.Idx++
}

func assertInputEvent(t *testing.T, raw []byte, typ, code uint16, value int32) {
	t.Helper()

	if got := binary.LittleEndian.Uint16(raw[0:]); got != typ {
		t.Fatalf("event type: got %d, want %d", got, typ)
	}

	if got := binary.LittleEndian.Uint16(raw[2:]); got != code {
		t.Fatalf("event code: got %d, want %d", got, code)
	}

	if got := int32(binary.LittleEndian.Uint32(raw[4:])); got != value {
		t.Fatalf("event value: got %d, want %d", got, value)
	}
}
