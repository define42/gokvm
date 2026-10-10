// Package guestmem validates guest-physical RAM addresses shared by devices.
package guestmem

const (
	// MMIOStart is the first address reserved for device registers below 4 GiB.
	MMIOStart uint64 = 0xd0000000
	// HighRAMStart is where RAM continues after the device address hole.
	HighRAMStart uint64 = 1 << 32
)

// ValidRange reports whether the complete range addresses guest RAM. The slice
// is indexed by guest-physical address and includes the unmapped MMIO hole when
// high RAM is present. Empty ranges may address the end of either RAM region,
// but never an address strictly inside the hole.
func ValidRange(mem []byte, addr, size uint64) bool {
	if addr > uint64(len(mem)) || size > uint64(len(mem))-addr {
		return false
	}

	return addr >= HighRAMStart || addr <= MMIOStart && size <= MMIOStart-addr
}
