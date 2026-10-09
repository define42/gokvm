package virtio

import "errors"

var errRDPThreads = errors.New("rdp: H.264 slice count must be 0..16; nonzero requires H.264")

const rdpAutoSlices = 2

// rdpEncoderSlices bounds the requested slice count using the host CPU budget.
// Small hosts still get one slice. The Go scheduler controls actual CPU use.
func rdpEncoderSlices(cpus, guestCPUs, limit int) int {
	if limit == 0 {
		limit = rdpAutoSlices
	}

	return min(limit, max(1, cpus-max(1, guestCPUs)-1))
}
