package kvm

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"syscall"
	"testing"
)

func TestGetSupportedCPUIDGrowsBuffer(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		initial uint32
		needed  uint32
	}{
		{name: "default-capacity", needed: 197},
		{name: "small-hint", initial: 1, needed: 197},
		{name: "existing-capacity", initial: 256, needed: 197},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			want := CPUID{Nent: tc.needed, Entries: make([]CPUIDEntry2, tc.needed)}
			for i := range want.Entries {
				want.Entries[i] = CPUIDEntry2{Function: uint32(i), Eax: uint32(i + 1), Edx: 0x12345678}
			}
			wire, err := want.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			var previous uint32
			cpuid, err := getSupportedCPUID(tc.initial, func(data []byte) error {
				nent := binary.LittleEndian.Uint32(data)
				if nent <= previous || len(data) != 8+int(nent)*40 {
					t.Fatalf("invalid buffer growth: capacity=%d previous=%d bytes=%d",
						nent, previous, len(data))
				}
				previous = nent
				if nent < tc.needed {
					return syscall.E2BIG
				}
				copy(data, wire)

				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*cpuid, want) {
				t.Fatalf("CPUID entries lost after growth: got %+v, want %+v", cpuid, want)
			}
		})
	}
}

func TestGetSupportedCPUIDError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "bad-descriptor", err: syscall.EBADF},
		{name: "allocation-failure", err: syscall.ENOMEM},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			_, err := getSupportedCPUID(0, func([]byte) error {
				calls++

				return tc.err
			})
			if !errors.Is(err, tc.err) || calls != 1 {
				t.Fatalf("got error %v after %d calls, want %v after one call", err, calls, tc.err)
			}
		})
	}
}

func TestGetSupportedCPUIDBoundsRetries(t *testing.T) {
	t.Parallel()
	calls := 0
	_, err := getSupportedCPUID(0, func(data []byte) error {
		calls++
		if len(data) > 8+40*maxSupportedCPUIDEntries {
			t.Fatalf("unbounded CPUID allocation: %d bytes", len(data))
		}

		return syscall.E2BIG
	})
	if !errors.Is(err, syscall.E2BIG) || calls < 2 {
		t.Fatalf("got error %v after %d calls, want bounded E2BIG retries", err, calls)
	}
}

func TestGetSupportedCPUIDRejectsTruncatedResponse(t *testing.T) {
	t.Parallel()
	_, err := getSupportedCPUID(1, func(data []byte) error {
		binary.LittleEndian.PutUint32(data, 2)

		return nil
	})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want truncated response error", err)
	}
}
