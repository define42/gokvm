package rdp

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

type transportBatchConn struct {
	net.Conn
	output   bytes.Buffer
	writes   int
	maxWrite int
}

func (c *transportBatchConn) Write(data []byte) (int, error) {
	c.writes++
	if c.maxWrite > 0 && len(data) > c.maxWrite {
		data = data[:c.maxWrite]
	}

	return c.output.Write(data)
}

func TestGraphicsTransportBatchesFragments(t *testing.T) {
	t.Parallel()
	data := make([]byte, 140000)
	for index := range data {
		data[index] = byte(index)
	}
	conn := &transportBatchConn{}
	session := &Session{conn: conn, dynamic: &dynamicState{channel: 1004}}
	if err := session.writeGraphics(data); err != nil {
		t.Fatal(err)
	}
	fragments := graphicsOutputFixture(t, conn.output.Bytes())
	if len(fragments) < 80 {
		t.Fatalf("large graphics message used only %d protocol fragments", len(fragments))
	}
	if conn.writes >= len(fragments)/2 {
		t.Fatalf("transport writes = %d for %d fragments; batching ineffective", conn.writes, len(fragments))
	}
	if got := reassembleGraphicsTransport(t, fragments); !bytes.Equal(got, data) {
		t.Fatal("batched graphics payload changed")
	}
}

func TestGraphicsTransportBatchHandlesShortWrites(t *testing.T) {
	t.Parallel()
	data := make([]byte, 70000)
	for index := range data {
		data[index] = byte(index * 31)
	}
	conn := &transportBatchConn{maxWrite: 17}
	session := &Session{conn: conn, dynamic: &dynamicState{channel: 1004}}
	if err := session.writeGraphics(data); err != nil {
		t.Fatal(err)
	}
	if got := reassembleGraphicsTransport(t, graphicsOutputFixture(t, conn.output.Bytes())); !bytes.Equal(got, data) {
		t.Fatal("short transport writes changed graphics payload")
	}
}

func reassembleGraphicsTransport(t *testing.T, fragments [][]byte) []byte {
	t.Helper()
	var buffer fragmentBuffer
	var assembled []byte
	for index, fragment := range fragments {
		var err error
		if index == 0 && len(fragments) > 1 {
			if len(fragment) < 6 || fragment[0] != 0x28 {
				t.Fatalf("invalid first DVC fragment: %x", fragment)
			}
			assembled, err = buffer.first(binary.LittleEndian.Uint32(fragment[2:]), fragment[6:])
		} else {
			if len(fragment) < 2 || fragment[0] != 0x30 {
				t.Fatalf("invalid DVC fragment %d: %x", index, fragment)
			}
			assembled, err = buffer.next(fragment[2:])
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	return decodeZGFXFixture(t, assembled)
}

type discardTransportConn struct {
	net.Conn
	writes int
}

func (c *discardTransportConn) Write(data []byte) (int, error) {
	c.writes++

	return len(data), nil
}

func BenchmarkGraphicsTransport(b *testing.B) {
	for _, test := range []struct {
		name string
		size int
	}{
		{"64KiB", 64 << 10},
		{"256KiB", 256 << 10},
		{"1MiB", 1 << 20},
	} {
		b.Run(test.name, func(b *testing.B) {
			data := make([]byte, test.size)
			conn := &discardTransportConn{}
			session := &Session{conn: conn, dynamic: &dynamicState{channel: 1004}}
			b.ReportAllocs()
			b.SetBytes(int64(test.size))
			writes := 0
			for b.Loop() {
				before := conn.writes
				if err := session.writeGraphics(data); err != nil {
					b.Fatal(err)
				}
				writes += conn.writes - before
			}
			b.ReportMetric(float64(writes)/float64(b.N), "writes/op")
		})
	}
}
