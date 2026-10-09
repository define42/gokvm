package virtio

import (
	"fmt"
	"log"
	"time"
)

type rdpStageTime struct {
	total time.Duration
	max   time.Duration
	count uint64
}

func (s *rdpStageTime) add(elapsed time.Duration) {
	if elapsed <= 0 {
		return
	}
	s.total += elapsed
	s.max = max(s.max, elapsed)
	s.count++
}

func (s rdpStageTime) String() string {
	var average time.Duration
	if s.count != 0 {
		average = s.total / time.Duration(s.count)
	}

	return fmt.Sprintf("%.2f/%.2fms", float64(average)/float64(time.Millisecond), float64(s.max)/float64(time.Millisecond))
}

type rdpWaitTime struct {
	start time.Time
	total time.Duration
}

func (w *rdpWaitTime) update(waiting bool, now time.Time) {
	if waiting && w.start.After(now) {
		return // Pacing still prevents sending; backpressure has not delayed a frame yet.
	}
	if !w.start.IsZero() {
		w.total += max(0, now.Sub(w.start))
		w.start = time.Time{}
	}
	if waiting {
		w.start = now
	}
}

// Stats belong to one frame writer. Disabled stats do not read the clock or
// allocate. Report only during activity and at disconnect; no idle timer.
type rdpFrameStats struct {
	enabled bool
	peer    string
	start   time.Time
	frames  uint64
	bytes   uint64
	copy    rdpStageTime
	convert rdpStageTime
	encode  rdpStageTime
	write   rdpStageTime
	ack     rdpWaitTime
	workers rdpWaitTime
}

func (s *rdpFrameStats) begin() time.Time {
	if !s.enabled {
		return time.Time{}
	}

	return time.Now()
}

func (s *rdpFrameStats) elapsed(start time.Time) time.Duration {
	if !s.enabled {
		return 0
	}

	return time.Since(start)
}

func (s *rdpFrameStats) waiting(ack, workers bool, nextFrame time.Time) {
	if s.enabled {
		now := time.Now()
		s.ack.update(ack, now)
		if ack && s.ack.start.Before(nextFrame) {
			s.ack.start = nextFrame
		}
		s.workers.update(workers, now)
	}
}

func (s *rdpFrameStats) sent(bytes int) {
	if s.enabled && bytes != 0 {
		s.frames++
		s.bytes += uint64(bytes)
	}
}

func (s *rdpFrameStats) report(threads int, final bool) {
	if !s.enabled {
		return
	}
	now := time.Now()
	if s.start.IsZero() {
		s.start = now
	}
	if !final && now.Sub(s.start) < 5*time.Second {
		return
	}
	s.ack.update(!s.ack.start.IsZero(), now)
	s.workers.update(!s.workers.start.IsZero(), now)
	if s.frames != 0 || s.ack.total != 0 || s.workers.total != 0 {
		log.Printf("rdp: stats client=%s threads=%d frames=%d payload_bytes=%d elapsed=%.2fs "+
			"avg/max copy=%s convert=%s encode=%s write=%s ack_wait=%s worker_wait=%s",
			s.peer, threads, s.frames, s.bytes, now.Sub(s.start).Seconds(),
			s.copy, s.convert, s.encode, s.write, s.ack.total.Round(time.Millisecond), s.workers.total.Round(time.Millisecond))
	}
	*s = rdpFrameStats{
		enabled: true, peer: s.peer, start: now,
		ack: rdpWaitTime{start: s.ack.start}, workers: rdpWaitTime{start: s.workers.start},
	}
}
