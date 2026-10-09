package openh264

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/debug"
	"testing"
	"time"
)

// TestCorruptStreams decodes randomly corrupted conformance streams and
// checks that the decoder neither panics nor hangs. Set MUTDIR to keep the
// failing inputs.
func TestCorruptStreams(t *testing.T) {
	outDir := os.Getenv("MUTDIR")
	iters := 300
	if testing.Short() {
		iters = 30
	}
	files := []string{"BA_MW_D.264", "test_qcif_cabac.264", "CVFC1_Sony_C.jsv", "MR1_BT_A.h264", "SVA_FM1_E.264", "Cisco_Men_whisper_640x320_CABAC_Bframe_9.264"}
	fixtures := make(map[string][]byte, len(files))
	for _, name := range files {
		data, err := os.ReadFile("../../res/" + name)
		if err != nil {
			t.Skipf("OpenH264 conformance data is unavailable: %v", err)
		}
		if len(data) <= 4 {
			t.Fatalf("OpenH264 conformance stream %s is too short", name)
		}
		if len(data) > 200000 {
			data = data[:200000]
		}
		fixtures[name] = data
	}
	r := rand.New(rand.NewSource(1))
	panics, hangs := 0, 0
	for iter := 0; iter < iters; iter++ {
		name := files[iter%len(files)]
		m := append([]byte(nil), fixtures[name]...)
		n := 1 + r.Intn(20)
		for k := 0; k < n; k++ {
			switch r.Intn(4) {
			case 0, 1:
				m[4+r.Intn(len(m)-4)] ^= byte(1 << r.Intn(8))
			case 2:
				m[4+r.Intn(len(m)-4)] = byte(r.Intn(256))
			case 3: // splice: copy a random chunk over another place
				a, b := 4+r.Intn(len(m)-4), 4+r.Intn(len(m)-4)
				l := r.Intn(64)
				if a+l < len(m) && b+l < len(m) {
					copy(m[a:a+l], m[b:b+l])
				}
			}
		}
		if r.Intn(5) == 0 {
			m = m[:4+r.Intn(len(m)-4)]
		}
		done := make(chan string, 1)
		go func() {
			defer func() {
				if p := recover(); p != nil {
					done <- fmt.Sprintf("panic: %v\n%s", p, debug.Stack())
				}
			}()
			dec, err := newQuietDecoder()
			if err != nil {
				done <- err.Error()
				return
			}
			start := 0
			for j := 4; j+4 <= len(m); j++ {
				if m[j] == 0 && m[j+1] == 0 && m[j+2] == 0 && m[j+3] == 1 {
					dec.Decode(m[start:j])
					start = j
				}
			}
			dec.Decode(m[start:])
			dec.Flush()
			dec.Close()
			done <- ""
		}()
		select {
		case s := <-done:
			if s != "" {
				panics++
				fn := fmt.Sprintf("%s/panic_%03d_%s", outDir, iter, name)
				if outDir != "" {
					os.WriteFile(fn, m, 0o644)
				}
				if panics <= 5 {
					t.Logf("%s: %s", fn, firstLines(s, 14))
				}
			}
		case <-time.After(20 * time.Second):
			hangs++
			fn := fmt.Sprintf("%s/hang_%03d_%s", outDir, iter, name)
			if outDir != "" {
				os.WriteFile(fn, m, 0o644)
			}
			t.Logf("HANG %s", fn)
			if hangs >= 3 {
				t.Fatalf("too many hangs")
			}
		}
	}
	if panics > 0 || hangs > 0 {
		t.Errorf("panics=%d hangs=%d", panics, hangs)
	}
}

func firstLines(s string, n int) string {
	c := 0
	for i := range s {
		if s[i] == '\n' {
			c++
			if c == n {
				return s[:i]
			}
		}
	}
	return s
}
