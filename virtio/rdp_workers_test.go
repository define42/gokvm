package virtio

import (
	"errors"
	"image"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bobuhiro11/gokvm/internal/rdp"
)

func TestRDPWorkerBudgetSharesAndLimits(t *testing.T) {
	t.Parallel()
	b := newRDPEncoderBudget(8, 2) // Five encoder CPUs after guest and host reservation.
	b.viewer(1)
	first, _ := b.tryAcquire(4, new(rdpWorkerRequest))
	if first != 4 {
		t.Fatalf("single client got %d workers", first)
	}
	b.viewer(1)
	joining := new(rdpWorkerRequest)
	if workers, ready := b.tryAcquire(4, joining); workers != 0 || ready == nil {
		t.Fatalf("joining client exceeded budget: workers=%d ready=%v", workers, ready)
	} else {
		b.release(first)
		select {
		case <-ready:
		default:
			t.Fatal("released workers did not wake the waiting viewer")
		}
	}
	first, _ = b.tryAcquire(4, joining)
	second, _ := b.tryAcquire(4, new(rdpWorkerRequest))
	if first != 2 || second != 2 || b.used != 4 {
		t.Fatalf("two clients must share: first=%d second=%d used=%d", first, second, b.used)
	}
	b.release(first)
	b.release(second)
	b.viewer(-1)
	workers, _ := b.tryAcquire(0, new(rdpWorkerRequest))
	if workers != rdpAutoThreads {
		t.Fatalf("automatic limit: got %d", workers)
	}
	b.release(workers)
	workers, _ = b.tryAcquire(1, new(rdpWorkerRequest))
	if workers != 1 {
		t.Fatalf("serial override got %d workers", workers)
	}
	b.release(workers)
	b.viewer(-1)
}

func TestRDPWorkerBudgetConcurrentViewers(t *testing.T) {
	t.Parallel()
	b := newRDPEncoderBudget(5, 1)
	const clients = 16
	b.viewer(clients)
	var active atomic.Int32
	var wg sync.WaitGroup
	for range clients {
		wg.Go(func() {
			request := new(rdpWorkerRequest)
			for range 20 {
				for {
					workers, ready := b.tryAcquire(4, request)
					if workers == 0 {
						<-ready

						continue
					}
					if running := active.Add(int32(workers)); running > int32(b.capacity) {
						t.Errorf("concurrent workers %d exceeded budget %d", running, b.capacity)
					}
					runtime.Gosched()
					active.Add(-int32(workers))
					b.release(workers)

					break
				}
			}
		})
	}
	wg.Wait()
	if b.used != 0 || active.Load() != 0 {
		t.Fatal("encoder workers leaked")
	}
}

func TestRDPWorkerBudgetFairAdmissionAndCancellation(t *testing.T) {
	t.Parallel()
	b := newRDPEncoderBudget(3, 1)
	b.viewer(3)
	busy, older, newer := new(rdpWorkerRequest), new(rdpWorkerRequest), new(rdpWorkerRequest)
	workers, _ := b.tryAcquire(1, busy)
	if got, _ := b.tryAcquire(1, older); got != 0 {
		t.Fatal("waiting viewer bypassed the capacity limit")
	}
	b.release(workers)
	if got, _ := b.tryAcquire(1, busy); got != 0 {
		t.Fatal("continuously updating viewer overtook an older waiter")
	}
	workers, _ = b.tryAcquire(1, older)
	if workers != 1 {
		t.Fatal("older waiter did not receive the next encoding slot")
	}
	if got, _ := b.tryAcquire(1, newer); got != 0 {
		t.Fatal("third viewer bypassed the capacity limit")
	}
	// The busy viewer is now at the queue head. A disconnect or suppression
	// must remove it, allowing the next viewer to proceed on release.
	b.cancel(busy)
	b.release(workers)
	workers, _ = b.tryAcquire(1, newer)
	if workers != 1 || len(b.waiters) != 0 {
		t.Fatal("cancelled viewer left the admission queue blocked")
	}
	b.release(workers)
}

func TestRDPWorkerWaitKeepsLatestFrame(t *testing.T) {
	t.Parallel()
	b := newRDPEncoderBudget(1, 8)
	b.viewer(2)
	occupied, _ := b.tryAcquire(4, new(rdpWorkerRequest))
	if occupied != 1 {
		t.Fatalf("small hosts must keep one usable worker: %d", occupied)
	}
	display := newFramebuffer()
	writer := &rdpFrameWriter{graphics: true, budget: b, threadLimit: 4, force: true}
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	for i := range 3 {
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))
		img.Pix[0] = byte(i + 1)
		if err := display.flush(16, 16, img); err != nil {
			t.Fatal(err)
		}
		if paced, err := writer.writeWhenReady(display, timer); err != nil || paced != nil {
			t.Fatalf("waiting should select the worker notification: paced=%v err=%v", paced, err)
		}
		if len(writer.frame.pix) != 0 || writer.workerReady == nil {
			t.Fatal("waiting writer captured an obsolete image or lost its wakeup")
		}
	}
	b.release(occupied)
	select {
	case <-writer.workerReady:
	default:
		t.Fatal("writer did not wake up after CPU capacity became available")
	}
	var latest vncFrame
	display.copyFrameChanges(&latest, false)
	if latest.pix[0] != 3 {
		t.Fatal("waiting writer lost the latest guest publication")
	}
}

func TestRDPWorkerConfigValidation(t *testing.T) {
	t.Parallel()
	for _, options := range []RDPConfig{
		{H264Threads: -1}, {H264Threads: 17, H264: true}, {H264Threads: 1},
	} {
		if _, err := NewRDPDisplayWithConfig("invalid-address", options); !errors.Is(err, errRDPThreads) {
			t.Fatalf("invalid worker options reached listener creation: %v", err)
		}
	}
}

func TestRDPWorkerQueueReleasesBeforeCursorWrite(t *testing.T) {
	t.Parallel()
	b := newRDPEncoderBudget(3, 1)
	b.viewer(3)
	occupied, _ := b.tryAcquire(1, new(rdpWorkerRequest))
	writer := &rdpFrameWriter{budget: b}
	b.tryAcquire(1, &writer.workerRequest)
	other := new(rdpWorkerRequest)
	b.tryAcquire(1, other)
	b.release(occupied)
	// The sink checks admission inside the control write: a blocked socket
	// must not leave its viewer queued ahead of another ready encoder.
	sink := &rdpAdmissionProbe{t: t, budget: b, request: other}
	pointer := rdpScheduledPointer{writer, sink}
	if err := pointer.WritePointer(nil, 0, 0); err != nil {
		t.Fatal(err)
	}
	if !sink.called || writer.workerRequest.queued {
		t.Fatal("cursor write did not cancel the encoder wait")
	}
}

type rdpAdmissionProbe struct {
	t       *testing.T
	budget  *rdpEncoderBudget
	request *rdpWorkerRequest
	called  bool
}

func (p *rdpAdmissionProbe) WritePointer(_ *image.RGBA, _, _ int) error {
	p.called = true
	workers, _ := p.budget.tryAcquire(1, p.request)
	if workers != 1 {
		p.t.Fatal("slow cursor client would block another client's encoder")
	}
	p.budget.release(workers)

	return nil
}

func (p *rdpAdmissionProbe) WritePointerPosition(x, y int) error {
	return p.WritePointer(nil, x, y)
}

func TestRDPWorkerQueueReleasesOnBitmapFallback(t *testing.T) {
	t.Parallel()
	b := newRDPEncoderBudget(3, 1)
	b.viewer(2)
	occupied, _ := b.tryAcquire(1, new(rdpWorkerRequest))
	writer := &rdpFrameWriter{session: &rdp.Session{}, graphics: true, budget: b}
	b.tryAcquire(1, &writer.workerRequest)
	if !writer.updateGraphics(nil) || writer.workerRequest.queued || b.clients != 1 {
		t.Fatal("bitmap fallback retained an AVC admission request or viewer")
	}
	b.release(occupied)
}
