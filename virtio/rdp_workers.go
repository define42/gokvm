package virtio

import "errors"

var errRDPThreads = errors.New("rdp: H.264 worker limit must be 0..16; nonzero requires H.264")

const rdpAutoThreads = 2

// rdpEncoderThreads bounds the single encoder, leaving CPU capacity for guest
// vCPUs and one host/audio worker. Small hosts still get one encoding worker.
func rdpEncoderThreads(cpus, guestCPUs, limit int) int {
	if limit == 0 {
		limit = rdpAutoThreads
	}

	return min(limit, max(1, cpus-max(1, guestCPUs)-1))
}
