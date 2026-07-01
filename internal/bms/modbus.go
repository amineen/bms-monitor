package bms

import (
	"fmt"
	"time"

	"github.com/goburrow/modbus"
)

// Config holds connection parameters for the BMS gateway.
type Config struct {
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	Unit    int    `json:"unit"`
	Timeout int    `json:"timeout"` // seconds; 0 -> 3
}

func (c Config) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 3 * time.Second
	}
	return time.Duration(c.Timeout) * time.Second
}

func (c Config) addr() string {
	port := c.Port
	if port == 0 {
		port = 502
	}
	return fmt.Sprintf("%s:%d", c.IP, port)
}

func wordsFromBytes(b []byte) []uint16 {
	out := make([]uint16, len(b)/2)
	for i := range out {
		out[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	return out
}

// readBaseFresh opens a FRESH TCP connection, performs exactly one read (FC04
// input registers, falling back to FC03 holding registers), then closes it.
//
// This is the load-bearing workaround for the RS485->TCP converter, which
// desyncs when several block reads share one persistent connection (responses
// shift one slot and the first block reads as "no response"). Reconnecting per
// read is the only reliable fix observed. CALLERS MUST KEEP READS SEQUENTIAL —
// never read concurrently on the same bus.
func readBaseFresh(cfg Config, start, count int) ([]uint16, error) {
	handler := modbus.NewTCPClientHandler(cfg.addr())
	handler.Timeout = cfg.timeout()
	handler.SlaveId = byte(unitOr1(cfg.Unit))
	if err := handler.Connect(); err != nil {
		return nil, err
	}
	defer handler.Close()
	client := modbus.NewClient(handler)

	if b, err := client.ReadInputRegisters(uint16(start), uint16(count)); err == nil {
		return wordsFromBytes(b), nil
	}
	b, err := client.ReadHoldingRegisters(uint16(start), uint16(count))
	if err != nil {
		return nil, err
	}
	return wordsFromBytes(b), nil
}

// readRegionFresh reads an arbitrary range chunked under the Modbus limit, each
// chunk on its own fresh connection (sequential).
func readRegionFresh(cfg Config, start, count int) ([]uint16, error) {
	out := make([]uint16, 0, count)
	addr, remaining := start, count
	for remaining > 0 {
		n := remaining
		if n > MaxRegs {
			n = MaxRegs
		}
		regs, err := readBaseFresh(cfg, addr, n)
		if err != nil {
			return nil, err
		}
		out = append(out, regs...)
		addr += n
		remaining -= n
	}
	return out, nil
}

func unitOr1(u int) int {
	if u <= 0 {
		return 1
	}
	return u
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
