// Package modbustcp is the shared read-only Modbus TCP transport for all plant
// drivers (OzTek, SMA, DSE). It generalizes the two load-bearing disciplines
// proven on the Pylontech converter:
//
//  1. FRESH CONNECTION PER READ — RS485->TCP converters desync when several
//     block reads share one persistent connection. Open, read once, close.
//  2. ONE-IN-FLIGHT PER HOST — a per-host mutex strictly serializes reads to
//     the same endpoint. This is critical for 192.168.0.41, where ONE USR
//     converter carries all three OzTek units on a single RS485 line; it is
//     harmless (and still polite) for native-Ethernet devices.
//
// The package exposes READS ONLY. There is deliberately no write helper: the
// only register write in the whole app is the gated Pylontech Run in
// internal/bms/control.go.
package modbustcp

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/goburrow/modbus"
)

// Endpoint identifies one Modbus TCP target (host + port + unit ID).
type Endpoint struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Unit       int    `json:"unit"`
	TimeoutSec int    `json:"timeoutSec"` // 0 -> 3
}

func (e Endpoint) addr() string {
	port := e.Port
	if port == 0 {
		port = 502
	}
	return fmt.Sprintf("%s:%d", e.Host, port)
}

func (e Endpoint) timeout() time.Duration {
	if e.TimeoutSec <= 0 {
		return 3 * time.Second
	}
	return time.Duration(e.TimeoutSec) * time.Second
}

func (e Endpoint) unit() byte {
	if e.Unit <= 0 {
		return 1
	}
	return byte(e.Unit)
}

// hostLocks serializes access per host:port. All three OzTeks share .41, so
// their per-unit reads contend on the same mutex — exactly what the shared
// RS485 bus requires.
var hostLocks sync.Map // string -> *sync.Mutex

func lockHost(addr string) func() {
	m, _ := hostLocks.LoadOrStore(addr, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// Reachable reports whether the endpoint accepts a TCP connection. Used as a
// fast pre-check so an unplugged device fails in <1 s instead of a full
// Modbus timeout per block.
func Reachable(e Endpoint, d time.Duration) bool {
	if d <= 0 {
		d = 800 * time.Millisecond
	}
	conn, err := net.DialTimeout("tcp", e.addr(), d)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func wordsFromBytes(b []byte) []uint16 {
	out := make([]uint16, len(b)/2)
	for i := range out {
		out[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	return out
}

// MaxRegs is the chunk size for region reads (Modbus hard limit is 125).
const MaxRegs = 120

// ReadHoldingFresh performs exactly one FC03 read on a fresh connection,
// serialized per host.
func ReadHoldingFresh(e Endpoint, start, count int) ([]uint16, error) {
	unlock := lockHost(e.addr())
	defer unlock()
	return readOnce(e, start, count, false)
}

// ReadFresh performs one read on a fresh connection, trying FC04 (input
// registers) first and falling back to FC03 — the Pylontech-style probe order
// for devices whose map spans both tables.
func ReadFresh(e Endpoint, start, count int) ([]uint16, error) {
	unlock := lockHost(e.addr())
	defer unlock()
	return readOnce(e, start, count, true)
}

func readOnce(e Endpoint, start, count int, inputFirst bool) ([]uint16, error) {
	handler := modbus.NewTCPClientHandler(e.addr())
	handler.Timeout = e.timeout()
	handler.SlaveId = e.unit()
	if err := handler.Connect(); err != nil {
		return nil, err
	}
	defer handler.Close()
	client := modbus.NewClient(handler)

	if inputFirst {
		if b, err := client.ReadInputRegisters(uint16(start), uint16(count)); err == nil {
			return wordsFromBytes(b), nil
		}
	}
	b, err := client.ReadHoldingRegisters(uint16(start), uint16(count))
	if err != nil {
		return nil, err
	}
	return wordsFromBytes(b), nil
}

// ReadHoldingRegion reads an arbitrary FC03 range chunked under the Modbus
// limit, each chunk on its own fresh connection (sequential).
func ReadHoldingRegion(e Endpoint, start, count int) ([]uint16, error) {
	out := make([]uint16, 0, count)
	addr, remaining := start, count
	for remaining > 0 {
		n := remaining
		if n > MaxRegs {
			n = MaxRegs
		}
		regs, err := ReadHoldingFresh(e, addr, n)
		if err != nil {
			return nil, err
		}
		out = append(out, regs...)
		addr += n
		remaining -= n
	}
	return out, nil
}

// ---- shared numeric decode helpers (hi-word-first, big-endian) ----

// At safely reads block[off]; returns 0 if out of range.
func At(block []uint16, off int) uint16 {
	if off < 0 || off >= len(block) {
		return 0
	}
	return block[off]
}

// S16 interprets an unsigned 16-bit register as signed.
func S16(v uint16) int {
	if v >= 0x8000 {
		return int(v) - 0x10000
	}
	return int(v)
}

// U32 combines two registers (hi-word-first) into an unsigned 32-bit value.
func U32(hi, lo uint16) uint32 { return uint32(hi)<<16 | uint32(lo) }

// S32 combines two registers (hi-word-first) into a signed 32-bit value.
func S32(hi, lo uint16) int64 {
	v := int64(hi)<<16 | int64(lo)
	if v >= 0x80000000 {
		return v - 0x100000000
	}
	return v
}

// U64 combines four registers (hi-word-first) into an unsigned 64-bit value.
func U64(w0, w1, w2, w3 uint16) uint64 {
	return uint64(w0)<<48 | uint64(w1)<<32 | uint64(w2)<<16 | uint64(w3)
}

// Round rounds v to dec decimal places (same behavior as internal/bms).
func Round(v float64, dec int) float64 {
	p := 1.0
	for i := 0; i < dec; i++ {
		p *= 10
	}
	if v >= 0 {
		return float64(int64(v*p+0.5)) / p
	}
	return float64(int64(v*p-0.5)) / p
}
