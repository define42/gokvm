//nolint:dupl
package kvm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"syscall"
	"unsafe"
)

// CPUID is the set of CPUID entries returned by GetCPUID.
type CPUID struct {
	Nent    uint32
	Padding uint32
	Entries []CPUIDEntry2
}

func (c *CPUID) Bytes() ([]byte, error) {
	var buf bytes.Buffer

	if err := binary.Write(&buf, binary.LittleEndian, c.Nent); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, c.Padding); err != nil {
		return nil, err
	}

	for _, entry := range c.Entries {
		if err := binary.Write(&buf, binary.LittleEndian, entry); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func NewCPUID(data []byte) (*CPUID, error) {
	c := CPUID{}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, data); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if err := binary.Read(&buf, binary.LittleEndian, &c.Nent); err != nil {
		return nil, err
	}

	if err := binary.Read(&buf, binary.LittleEndian, &c.Padding); err != nil {
		return nil, err
	}

	c.Entries = make([]CPUIDEntry2, c.Nent)

	if err := binary.Read(&buf, binary.LittleEndian, &c.Entries); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	return &c, nil
}

// CPUIDEntry2 is one entry for CPUID. It took 2 tries to get it right :-)
// Thanks x86 :-).
type CPUIDEntry2 struct {
	Function uint32
	Index    uint32
	Flags    uint32
	Eax      uint32
	Ebx      uint32
	Ecx      uint32
	Edx      uint32
	Padding  [3]uint32
}

// GetSupportedCPUID gets all supported CPUID entries for a VM. Nent is an
// optional initial capacity; the buffer grows if KVM reports E2BIG.
func GetSupportedCPUID(kvmFd uintptr, kvmCPUID *CPUID) error {
	cpuid, err := getSupportedCPUID(kvmCPUID.Nent, func(data []byte) error {
		_, err := Ioctl(kvmFd,
			IIOWR(kvmGetSupportedCPUID, 8),
			uintptr(unsafe.Pointer(&data[0])))

		return err
	})
	if err != nil {
		return err
	}
	*kvmCPUID = *cpuid

	return nil
}

// Leave room for future CPU leaves, but bound retries and memory use if a
// kernel keeps rejecting the buffer. Current KVM limits are far below this.
const maxSupportedCPUIDEntries = 4096

func getSupportedCPUID(nent uint32, ioctl func([]byte) error) (*CPUID, error) {
	if nent == 0 {
		nent = 128
	}
	if nent > maxSupportedCPUIDEntries {
		return nil, fmt.Errorf("supported CPUID capacity %d exceeds %d: %w",
			nent, maxSupportedCPUIDEntries, syscall.E2BIG)
	}
	for {
		// struct kvm_cpuid2 has an 8-byte header and 40-byte entries.
		data := make([]byte, 8+int(nent)*40)
		binary.LittleEndian.PutUint32(data, nent)
		err := ioctl(data)
		if errors.Is(err, syscall.E2BIG) && nent < maxSupportedCPUIDEntries {
			nent = min(nent*2, maxSupportedCPUIDEntries)

			continue
		}
		if err != nil {
			return nil, fmt.Errorf("get supported CPUID with %d entries: %w", nent, err)
		}
		if returned := binary.LittleEndian.Uint32(data); returned > nent {
			return nil, fmt.Errorf("supported CPUID returned %d entries for capacity %d: %w",
				returned, nent, io.ErrUnexpectedEOF)
		}

		return NewCPUID(data)
	}
}

// SetCPUID2 sets entries for a vCPU.
// The progression is, hence, get the CPUID entries for a vm, then set them into
// individual vCPUs. This seems odd, but in fact lets code tailor CPUID entries
// as needed.
func SetCPUID2(vcpuFd uintptr, kvmCPUID *CPUID) error {
	data, err := kvmCPUID.Bytes()
	if err != nil {
		return err
	}

	if _, err := Ioctl(vcpuFd,
		IIOW(kvmSetCPUID2, unsafe.Sizeof(kvmCPUID)),
		uintptr(unsafe.Pointer(&data[0]))); err != nil {
		return err
	}

	return err
}

func GetCPUID2(vcpuFd uintptr, kvmCPUID *CPUID) error {
	var c *CPUID

	data, err := kvmCPUID.Bytes()
	if err != nil {
		return err
	}

	if _, err = Ioctl(vcpuFd,
		IIOWR(kvmGetCPUID2, 8),
		uintptr(unsafe.Pointer(&data[0]))); err != nil {
		return err
	}

	if c, err = NewCPUID(data); err != nil {
		return err
	}

	*kvmCPUID = *c

	return err
}

// GetEmulatedCPUID returns x86 cpuid features which are emulated by kvm.
func GetEmulatedCPUID(kvmFd uintptr, kvmCPUID *CPUID) error {
	var c *CPUID

	data, err := kvmCPUID.Bytes()
	if err != nil {
		return err
	}

	if _, err = Ioctl(kvmFd,
		IIOWR(kvmGetEmulatedCPUID, unsafe.Sizeof(kvmCPUID)),
		uintptr(unsafe.Pointer(&data[0]))); err != nil {
		return err
	}

	if c, err = NewCPUID(data); err != nil {
		return err
	}

	*kvmCPUID = *c

	return nil
}
