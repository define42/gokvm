// constants for creating ioctl commands.
package kvm

import (
	"errors"
	"syscall"
)

// Ioctl is a convenience function to call ioctl.
// Its main purpose is to format arguments
// and return values to make things easier for
// programmers. Pointer-valued arguments must be converted in the call expression;
// uintptrescapes keeps their storage alive and off movable goroutine stacks
// while this wrapper forwards the address to the syscall helper.
//
//go:uintptrescapes
func Ioctl(fd, op, arg uintptr) (uintptr, error) {
	for {
		res, err := ioctlOnce(fd, op, arg)
		if errors.Is(err, syscall.EINTR) {
			continue
		}

		return res, err
	}
}

// ioctlOnce leaves interrupted calls to the caller. KVM_RUN must observe EINTR
// so its owner can stop the vCPU instead of retrying immediate_exit forever.
// It has the same pointer-lifetime contract as Ioctl.
//
//go:uintptrescapes
func ioctlOnce(fd, op, arg uintptr) (uintptr, error) {
	res, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, op, arg)
	if errno != 0 {
		return res, errno
	}

	return res, nil
}

const (
	nrbits   = 8
	typebits = 8
	sizebits = 14
	dirbits  = 2

	nrmask   = (1 << nrbits) - 1
	sizemask = (1 << sizebits) - 1
	dirmask  = (1 << dirbits) - 1

	none      = 0
	write     = 1
	read      = 2
	readwrite = 3

	nrshift   = 0
	typeshift = nrshift + nrbits
	sizeshift = typeshift + typebits
	dirshift  = sizeshift + sizebits
)

// KVMIO is for the KVMIO ioctl.
const KVMIO = 0xAE

// IIOWR creates an IIOWR ioctl.
func IIOWR(nr, size uintptr) uintptr {
	return IIOC(readwrite, nr, size)
}

// IIOR creates an IIOR ioctl.
func IIOR(nr, size uintptr) uintptr {
	return IIOC(read, nr, size)
}

// IIOW creates an IIOW ioctl.
func IIOW(nr, size uintptr) uintptr {
	return IIOC(write, nr, size)
}

// IIO creates an IIOC ioctl from a number.
func IIO(nr uintptr) uintptr {
	return IIOC(none, nr, 0)
}

// IIOC creates an IIOC ioctl from a direction, nr, and size.
func IIOC(dir, nr, size uintptr) uintptr {
	// This is another case of forced wrapping which is considered an anti-pattern in Google.
	return ((dir & dirmask) << dirshift) | (KVMIO << typeshift) |
		((nr & nrmask) << nrshift) | ((size & sizemask) << sizeshift)
}
