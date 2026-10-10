package guestmem

import (
	"syscall"
	"testing"
)

func TestValidRange(t *testing.T) {
	t.Parallel()

	// Reserve address space without committing RAM; validation never reads it.
	mem, err := syscall.Mmap(-1, 0, int(HighRAMStart+4096), syscall.PROT_NONE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS|syscall.MAP_NORESERVE)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := syscall.Munmap(mem); err != nil {
			t.Error(err)
		}
	})

	for _, tc := range []struct {
		name       string
		addr, size uint64
		want       bool
	}{
		{"low RAM", 0, 4096, true},
		{"last low page", MMIOStart - 4096, 4096, true},
		{"cross low boundary", MMIOStart - 1, 2, false},
		{"MMIO", MMIOStart, 1, false},
		{"hole", MMIOStart + 4096, 4096, false},
		{"cross high boundary", HighRAMStart - 1, 2, false},
		{"span hole", MMIOStart - 4096, HighRAMStart - MMIOStart + 8192, false},
		{"high RAM", HighRAMStart, 4096, true},
		{"beyond high RAM", HighRAMStart + 4096, 1, false},
		{"empty end of low RAM", MMIOStart, 0, true},
		{"empty hole", MMIOStart + 1, 0, false},
		{"empty end of high RAM", HighRAMStart + 4096, 0, true},
		{"overflow address", ^uint64(0), 2, false},
		{"overflow size", 1, ^uint64(0), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidRange(mem, tc.addr, tc.size); got != tc.want {
				t.Errorf("ValidRange(%#x, %#x) = %v, want %v", tc.addr, tc.size, got, tc.want)
			}
		})
	}

	if ValidRange(mem[:4096], 4096, 1) || !ValidRange(mem[:4096], 4096, 0) {
		t.Error("small RAM extent was not respected")
	}
}
