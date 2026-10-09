package machine

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"
	"unsafe"

	"github.com/bobuhiro11/gokvm/kvm"
)

func TestGuestMemoryAlignment(t *testing.T) {
	t.Parallel()
	for _, size := range []int{1, 4096, MinMemSize, guestMemoryAlignment, guestMemoryAlignment + 4096, 3*guestMemoryAlignment + 17} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			mapping, mem, err := mapGuestMemory(size)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := syscall.Munmap(mapping); err != nil {
					t.Errorf("unmapping original allocation: %v", err)
				}
			})
			addr := uintptr(unsafe.Pointer(&mem[0]))
			base := uintptr(unsafe.Pointer(&mapping[0]))
			if addr%guestMemoryAlignment != 0 {
				t.Fatalf("guest RAM %#x is not 2 MiB aligned", addr)
			}
			if len(mem) != size || cap(mem) != size || addr < base || addr+uintptr(size) > base+uintptr(len(mapping)) {
				t.Fatal("aligned RAM does not preserve requested size within the allocation")
			}
			if mem[0] != 0 || mem[len(mem)-1] != 0 {
				t.Fatal("anonymous RAM was not zero initialized")
			}
			mem[0], mem[len(mem)-1] = 0x35, 0x7a
			if mapping[int(addr-base)+len(mem)-1] != 0x7a {
				t.Fatal("aligned RAM does not alias the original allocation")
			}
		})
	}
}

func TestGuestMemoryRejectsInvalidSize(t *testing.T) {
	t.Parallel()
	for _, size := range []int{-1, 0, int(^uint(0) >> 1)} {
		mapping, mem, err := mapGuestMemory(size)
		if !errors.Is(err, syscall.EINVAL) || mapping != nil || mem != nil {
			t.Fatalf("size %d: got allocation %d/%d and error %v", size, len(mapping), len(mem), err)
		}
	}
}

func TestGuestMemoryKVMCoherence(t *testing.T) {
	dev, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
		t.Skipf("KVM unavailable: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dev.Close() })
	mapping, mem, err := mapGuestMemory(guestMemoryAlignment + 4096)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Munmap(mapping) })
	vm, err := kvm.CreateVM(dev.Fd())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Close(int(vm)) })
	if err := kvm.SetUserMemoryRegion(vm, &kvm.UserspaceMemoryRegion{
		MemorySize: uint64(len(mem)), UserspaceAddr: uint64(uintptr(unsafe.Pointer(&mem[0]))),
	}); err != nil {
		t.Fatal(err)
	}
	vcpu, err := kvm.CreateVCPU(vm, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Close(int(vcpu)) })
	runSize, err := kvm.GetVCPUMMmapSize(dev.Fd())
	if err != nil {
		t.Fatal(err)
	}
	runMapping, err := syscall.Mmap(int(vcpu), 0, int(runSize), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Munmap(runMapping) })
	sregs, err := kvm.GetSregs(vcpu)
	if err != nil {
		t.Fatal(err)
	}
	sregs.CS.Base, sregs.CS.Selector, sregs.DS.Base, sregs.DS.Selector = 0, 0, 0, 0
	if err := kvm.SetSregs(vcpu, sregs); err != nil {
		t.Fatal(err)
	}

	// Read a host-written byte, add seven, write it back through the guest
	// mapping, then halt. Private mmap must still be shared with the vCPU.
	copy(mem[0x1000:], []byte{0xa0, 0x00, 0x20, 0x04, 0x07, 0xa2, 0x01, 0x20, 0xf4})
	mem[0x2000] = 35
	if err := kvm.SetRegs(vcpu, &kvm.Regs{RIP: 0x1000, RFLAGS: 2}); err != nil {
		t.Fatal(err)
	}
	if err := kvm.Run(vcpu); err != nil {
		t.Fatal(err)
	}
	run := (*kvm.RunData)(unsafe.Pointer(&runMapping[0]))
	if kvm.ExitType(run.ExitReason) != kvm.EXITHLT || mem[0x2001] != 42 {
		t.Fatalf("guest/host coherence failed: exit %v, result %d", run.ExitReason, mem[0x2001])
	}
}
