package virtio

import (
	"errors"
	"sync"
)

var errRDPThreads = errors.New("rdp: H.264 worker limit must be 0..16; nonzero requires H.264")

// Auto uses a modest per-client limit; the shared budget also leaves CPU time
// for the guest and host/audio work. Explicit limits still respect that budget.
const rdpAutoThreads = 2

type rdpWorkerRequest struct{ queued bool }

type rdpEncoderBudget struct {
	mu       sync.Mutex
	capacity int
	used     int
	clients  int
	changed  chan struct{}
	waiters  []*rdpWorkerRequest
}

func newRDPEncoderBudget(cpus, guestCPUs int) *rdpEncoderBudget {
	return &rdpEncoderBudget{
		capacity: max(1, cpus-max(1, guestCPUs)-1),
		changed:  make(chan struct{}),
	}
}

func (b *rdpEncoderBudget) viewer(delta int) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients += delta
	b.notifyLocked()
}

// tryAcquire never blocks a frame writer. While waiting, it can process cursor,
// resize and disconnect events, and capture only the latest frame when ready.
func (b *rdpEncoderBudget) tryAcquire(limit int, request *rdpWorkerRequest) (int, <-chan struct{}) {
	if b == nil {
		return 1, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit == 0 {
		limit = rdpAutoThreads
	}
	workers := min(limit, max(1, b.capacity/max(1, b.clients)))
	// FIFO admission prevents a continuously updating client from racing past
	// a viewer already waiting for capacity. No image is queued with a request.
	if !request.queued {
		b.waiters = append(b.waiters, request)
		request.queued = true
	}
	if b.waiters[0] != request || b.used+workers > b.capacity {
		return 0, b.changed
	}
	b.removeLocked(request)
	b.used += workers

	return workers, nil
}

// A suppressed, resized or disconnected viewer must not hold up the queue.
func (b *rdpEncoderBudget) cancel(request *rdpWorkerRequest) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if request.queued {
		b.removeLocked(request)
	}
}

func (b *rdpEncoderBudget) removeLocked(request *rdpWorkerRequest) {
	for i, waiter := range b.waiters {
		if waiter == request {
			copy(b.waiters[i:], b.waiters[i+1:])
			b.waiters[len(b.waiters)-1] = nil
			b.waiters = b.waiters[:len(b.waiters)-1]
			request.queued = false
			b.notifyLocked()

			return
		}
	}
}

func (b *rdpEncoderBudget) release(workers int) {
	if b == nil || workers == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.used -= workers
	b.notifyLocked()
}

func (b *rdpEncoderBudget) notifyLocked() {
	close(b.changed)
	b.changed = make(chan struct{})
}
