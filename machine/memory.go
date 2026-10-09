package machine

import (
	"syscall"
	"unsafe"
)

const guestMemoryAlignment = 2 << 20

// mapGuestMemory returns both the complete mmap allocation and the aligned RAM
// slice. Only the complete allocation may be passed to syscall.Munmap.
func mapGuestMemory(size int) (mapping, mem []byte, err error) {
	const padding = guestMemoryAlignment - 1
	if size <= 0 || size > int(^uint(0)>>1)-padding {
		return nil, nil, syscall.EINVAL
	}

	mapping, err = syscall.Mmap(-1, 0, size+padding,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS)
	if err != nil {
		return nil, nil, err
	}

	// KVM can use large pages when host and guest addresses have matching low
	// 21 bits. Guest RAM starts at physical address zero.
	offset := int(-uintptr(unsafe.Pointer(&mapping[0])) & padding)
	mem = mapping[offset : offset+size : offset+size]
	// Private anonymous RAM supports THP independently of the host's shmem
	// policy. Unsupported or unavailable huge pages safely fall back to 4 KiB.
	_ = syscall.Madvise(mem, syscall.MADV_HUGEPAGE)

	return mapping, mem, nil
}
