package virtio

import (
	"testing"
	"time"
)

func TestRDPStatsWaitWindows(t *testing.T) {
	t.Parallel()
	var wait rdpWaitTime
	start := time.Unix(100, 0)
	wait.update(true, start)
	wait.update(true, start.Add(2*time.Second))
	if wait.total != 2*time.Second {
		t.Fatalf("continuing wait=%v", wait.total)
	}
	// Reporting consumes the interval but carries the outstanding wait into
	// the next window, with no double counting when the client acknowledges.
	wait.total = 0
	wait.update(false, start.Add(3*time.Second))
	if wait.total != time.Second || !wait.start.IsZero() {
		t.Fatalf("next window wait=%v start=%v", wait.total, wait.start)
	}
	wait.update(false, start.Add(5*time.Second))
	if wait.total != time.Second {
		t.Fatal("idle time was counted as client backpressure")
	}
}

func TestRDPStatsDisabled(t *testing.T) {
	t.Parallel()
	var stats rdpFrameStats
	if !stats.begin().IsZero() || stats.elapsed(time.Time{}) != 0 {
		t.Fatal("disabled stats sampled the clock")
	}
	stats.waiting(true, time.Time{})
	stats.sent(1024)
	stats.report(2, true)
	if stats != (rdpFrameStats{}) {
		t.Fatal("disabled statistics retained samples")
	}
}

func TestRDPStatsACKWaitExcludesPacing(t *testing.T) {
	t.Parallel()
	start := time.Unix(100, 0)
	nextFrame := start.Add(17 * time.Millisecond)
	wait := rdpWaitTime{start: nextFrame}
	// ACKs received before the next permitted frame do not delay rendering.
	wait.update(true, start.Add(time.Millisecond))
	wait.update(false, start.Add(5*time.Millisecond))
	if wait.total != 0 || !wait.start.IsZero() {
		t.Fatalf("pacing was attributed to client delay: %+v", wait)
	}
	wait.start = nextFrame
	wait.update(false, start.Add(20*time.Millisecond))
	if wait.total != 3*time.Millisecond {
		t.Fatalf("only time past the frame deadline should count: %v", wait.total)
	}
}
