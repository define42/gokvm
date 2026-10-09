package common

import (
	"math/rand"
	"testing"
)

type testBitReader struct {
	buf []uint8
	pos int // bit position
}

func (r *testBitReader) bit() uint32 {
	b := (r.buf[r.pos>>3] >> (7 - uint(r.pos&7))) & 1
	r.pos++
	return uint32(b)
}

func (r *testBitReader) bits(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = v<<1 | r.bit()
	}
	return v
}

func (r *testBitReader) ue() uint32 {
	lz := 0
	for r.bit() == 0 {
		lz++
	}
	return (1<<uint(lz) - 1) + r.bits(lz)
}

func (r *testBitReader) se() int32 {
	k := r.ue()
	if k&1 != 0 {
		return int32((k + 1) / 2)
	}
	return -int32(k / 2)
}

func TestGolombWriteRoundTrip(t *testing.T) {
	rnd := rand.New(rand.NewSource(50))
	buf := make([]uint8, 1<<16)
	var bs SBitStringAux
	const off = 7
	InitBits(&bs, buf, off, int32(len(buf)-off))
	type item struct {
		kind int // 0 bits, 1 ue, 2 se
		n    int32
		v    uint32
		sv   int32
	}
	var items []item
	for i := 0; i < 3000; i++ {
		switch rnd.Intn(3) {
		case 0:
			n := int32(1 + rnd.Intn(31))
			v := rnd.Uint32() & (uint32(1)<<uint(n) - 1)
			BsWriteBits(&bs, n, v)
			items = append(items, item{kind: 0, n: n, v: v})
		case 1:
			v := uint32(rnd.Intn(1 << uint(rnd.Intn(16)+1)))
			BsWriteUE(&bs, v)
			items = append(items, item{kind: 1, v: v})
		default:
			sv := int32(rnd.Intn(1<<15)) - (1 << 14)
			BsWriteSE(&bs, sv)
			items = append(items, item{kind: 2, sv: sv})
		}
	}
	BsRbspTrailingBits(&bs)
	r := &testBitReader{buf: buf[off:]}
	for i, it := range items {
		switch it.kind {
		case 0:
			if got := r.bits(int(it.n)); got != it.v {
				t.Fatalf("item %d bits(%d): got %#x want %#x", i, it.n, got, it.v)
			}
		case 1:
			if got := r.ue(); got != it.v {
				t.Fatalf("item %d ue: got %d want %d", i, got, it.v)
			}
		default:
			if got := r.se(); got != it.sv {
				t.Fatalf("item %d se: got %d want %d", i, got, it.sv)
			}
		}
	}
	if r.bit() != 1 {
		t.Fatalf("missing rbsp stop bit")
	}
	for r.pos&7 != 0 {
		if r.bit() != 0 {
			t.Fatalf("non-zero alignment bit")
		}
	}
	if (bs.PCurBuf-bs.PStartBuf)*8 != r.pos {
		t.Fatalf("PCurBuf advanced %d bytes, reader consumed %d bits", bs.PCurBuf-bs.PStartBuf, r.pos)
	}
}

func TestGolombUELengthTable(t *testing.T) {
	for i := 0; i < 256; i++ {
		want := uint32(2*WELS_LOG2(uint32(i+1)) + 1)
		if G_kuiGolombUELength[i] != want {
			t.Fatalf("G_kuiGolombUELength[%d] = %d want %d", i, G_kuiGolombUELength[i], want)
		}
	}
}
