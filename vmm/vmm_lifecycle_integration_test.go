//go:build integration

package vmm

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/define42/gokvm/kvm"
	"github.com/define42/gokvm/machine"
	"github.com/define42/gokvm/term"
)

func TestHeadlessBootReturnsCPUFailure(t *testing.T) {
	t.Parallel()
	if term.IsTerminal() {
		t.Skip("requires headless stdin")
	}
	m, err := machine.New("/dev/kvm", 2, machine.MinMemSize)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
		t.Skipf("KVM unavailable: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	// The poisoned RAM deliberately executes UD2 without an IDT, producing a
	// fatal exit on the BSP while the second CPU is waiting for startup.
	if err := m.SetupRegs(0x100000, 0x10000, false); err != nil {
		t.Fatal(err)
	}
	v := New(Config{NCPUs: 2})
	v.Machine = m
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	err = v.BootContext(ctx)
	if !errors.Is(err, kvm.ErrUnexpectedExitReason) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("headless Boot returned %v", err)
	}
	if _, err := m.RunOnce(1); !errors.Is(err, machine.ErrMachineStopped) {
		t.Fatalf("sibling CPU not stopped: %v", err)
	}
}
