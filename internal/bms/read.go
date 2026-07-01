package bms

import (
	"time"
)

// ---- data model (exported as TypeScript types via Wails bindings) ----

type Identity struct {
	OK       bool   `json:"ok"`
	Name     string `json:"name"`
	Firmware string `json:"firmware"`
	Build    int    `json:"build"`
}

type Aggregate struct {
	OK       bool    `json:"ok"`
	Piles    int     `json:"piles"`
	TotalV   float64 `json:"totalV"`
	Current  float64 `json:"current"`
	PowerKW  float64 `json:"powerKW"`
	SOC      int     `json:"soc"`
	SOH      int     `json:"soh"`
	Temp     float64 `json:"temp"`
	CellMaxV float64 `json:"cellMaxV"`
	CellMinV float64 `json:"cellMinV"`
}

// String status values.
const (
	StatusEnumerated = "enumerated"
	StatusEmpty      = "empty"
	StatusNoResponse = "no-response"
)

type StringInfo struct {
	Index          int       `json:"index"` // 1..6
	Base           int       `json:"base"`
	BaseHex        string    `json:"baseHex"`
	Status         string    `json:"status"`
	Serial         string    `json:"serial"`
	IsMaster       bool      `json:"isMaster"`
	HasDetail      bool      `json:"hasDetail"`
	TotalV         float64   `json:"totalV"`
	Current        float64   `json:"current"`
	PowerKW        float64   `json:"powerKW"`
	SOC            int       `json:"soc"`
	SOH            int       `json:"soh"`
	SOE            int       `json:"soe"`
	Temp           float64   `json:"temp"`
	Cycles         int       `json:"cycles"`
	CellMaxV       float64   `json:"cellMaxV"`
	CellMinV       float64   `json:"cellMinV"`
	CellSpreadMV   float64   `json:"cellSpreadMV"`
	CellMaxT       float64   `json:"cellMaxT"`
	CellMinT       float64   `json:"cellMinT"`
	ModMaxV        float64   `json:"modMaxV"`
	ModMinV        float64   `json:"modMinV"`
	Modules        int       `json:"modules"` // modules in series
	Cells          int       `json:"cells"`   // cells in series
	NominalAh      int       `json:"nominalAh"`
	RemainWh       int64     `json:"remainWh"`
	BasicStatus    string    `json:"basicStatus"`
	ProtectionText string    `json:"protectionText"`
	Alarms         []string  `json:"alarms"`
	ModuleV        []float64 `json:"moduleV"`
	ModuleT        []float64 `json:"moduleT"`
	CellV          []float64 `json:"cellV"`
}

type ChainStatus struct {
	Online     int      `json:"online"`
	Total      int      `json:"total"`
	StopsAfter int      `json:"stopsAfter"`
	Verdict    string   `json:"verdict"`
	States     []string `json:"states"` // index 0..5 -> status of strings 1..6
}

type SystemSnapshot struct {
	Timestamp  string       `json:"timestamp"`
	IP         string       `json:"ip"`
	Port       int          `json:"port"`
	Unit       int          `json:"unit"`
	Identity   Identity     `json:"identity"`
	Aggregate  Aggregate    `json:"aggregate"`
	HeartbeatA int          `json:"heartbeatA"`
	HeartbeatB int          `json:"heartbeatB"`
	LinkLive   bool         `json:"linkLive"`
	Strings    []StringInfo `json:"strings"`
	Chain      ChainStatus  `json:"chain"`
	Health     HealthStatus `json:"health"`
	ReadMillis int64        `json:"readMillis"`
}

// at safely reads block[off]; returns 0 if out of range.
func at(block []uint16, off int) uint16 {
	if off < 0 || off >= len(block) {
		return 0
	}
	return block[off]
}

func decodeASCII(words []uint16) string {
	b := make([]byte, 0, len(words)*2)
	for _, w := range words {
		hi, lo := byte(w>>8), byte(w&0xFF)
		if hi >= 32 && hi < 127 {
			b = append(b, hi)
		}
		if lo >= 32 && lo < 127 {
			b = append(b, lo)
		}
	}
	// trim spaces
	s := string(b)
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	return s
}

// readIdentity reads the equipment-info block (0x1000).
func readIdentity(cfg Config) Identity {
	ident, err := readBaseFresh(cfg, IdentBase, 0x10)
	if err != nil || len(ident) < 0x0C {
		return Identity{OK: false}
	}
	fw := ident[0x0A]
	return Identity{
		OK:       true,
		Name:     decodeASCII(ident[:10]),
		Firmware: "V" + itoa(int(fw>>8)) + "." + itoa(int(fw&0xFF)),
		Build:    int(ident[0x0B]),
	}
}

// TestConnection verifies the gateway is reachable and is a Pylontech BMS.
func TestConnection(cfg Config) (Identity, error) {
	ident, err := readBaseFresh(cfg, IdentBase, 0x10)
	if err != nil {
		return Identity{OK: false}, err
	}
	fw := uint16(0)
	if len(ident) > 0x0A {
		fw = ident[0x0A]
	}
	id := Identity{OK: true, Name: decodeASCII(ident[:min(10, len(ident))]),
		Firmware: "V" + itoa(int(fw>>8)) + "." + itoa(int(fw&0xFF))}
	if len(ident) > 0x0B {
		id.Build = int(ident[0x0B])
	}
	return id, nil
}

// decodeStringSummary fills the summary-level fields of a StringInfo from a block.
func decodeStringSummary(s *StringInfo, block []uint16) {
	s.TotalV = round(float64(at(block, 0x03))*0.1, 1)
	s.Current = round(float64(S32(at(block, 0x04), at(block, 0x05)))*0.01, 2)
	s.PowerKW = round(s.TotalV*s.Current/1000, 3)
	s.SOC = int(at(block, 0x07))
	s.SOH = int(at(block, 0x20))
	s.SOE = int(at(block, 0x48))
	s.Temp = round(float64(S16(at(block, 0x06)))*0.1, 1)
	s.Cycles = int(at(block, 0x08))
	s.CellMaxV = round(float64(at(block, 0x10))*0.001, 3)
	s.CellMinV = round(float64(at(block, 0x11))*0.001, 3)
	s.CellSpreadMV = round((s.CellMaxV-s.CellMinV)*1000, 0)
	s.CellMaxT = round(float64(S16(at(block, 0x14)))*0.1, 1)
	s.CellMinT = round(float64(S16(at(block, 0x15)))*0.1, 1)
	s.ModMaxV = round(float64(at(block, 0x18))*0.01, 2)
	s.ModMinV = round(float64(at(block, 0x19))*0.01, 2)
	s.Modules = clamp(int(at(block, 0x36)), 0, MaxModules)
	s.Cells = clamp(int(at(block, 0x37)), 0, MaxCells)
	s.NominalAh = int(at(block, 0x3B))
	s.RemainWh = S32(at(block, 0x21), at(block, 0x22))
	s.BasicStatus = DecodeBasic(at(block, 0x00))
	s.ProtectionText = DecodeProt(at(block, 0x01))
	s.Alarms = collectAlarms(block)
}

// collectAlarms returns only GENUINE active protections (from the Protection
// status bitfield, 0x01). These are the real "something is actively wrong"
// conditions — over/under-voltage, over/under-temp, over-current, short-circuit,
// etc. — and are what drive the red banner.
//
// Deliberately NOT included: the raw "Alarm status" and "Error code" bitfields
// (0x02 / 0x49 / 0x32). On a healthy, resting, fully-commissioned stack these
// can read non-zero with benign/informational bits set, which would otherwise
// force a false red on a 6/6 system. They remain readable in the Excel export.
func collectAlarms(block []uint16) []string {
	flags := ProtFlags(at(block, 0x01))
	if flags == nil {
		return []string{}
	}
	return flags
}

// readString reads one string block and decodes it. Master (index 1), when
// enumerated, also gets its per-module and per-cell arrays.
func readString(cfg Config, index int) StringInfo {
	base := PileBase + (index-1)*PileStride
	s := StringInfo{Index: index, Base: base, BaseHex: hex4(base), IsMaster: index == 1, Alarms: []string{}}

	block, err := readBaseFresh(cfg, base, 0x52)
	if err != nil {
		s.Status = StatusNoResponse
		return s
	}
	enumerated := at(block, 0x03) > 0 || at(block, 0x36) > 0
	if !enumerated {
		s.Status = StatusEmpty
		return s
	}
	s.Status = StatusEnumerated
	decodeStringSummary(&s, block)

	// Serial number (master/slave both carry SN in their block).
	if len(block) > SNOff+SNLen {
		s.Serial = decodeASCII(block[SNOff : SNOff+SNLen])
	}

	// Only the master string exposes full module/cell arrays.
	if s.IsMaster && s.Modules > 0 {
		if mv, err := readRegionFresh(cfg, base+ModVOff, s.Modules); err == nil {
			s.ModuleV = scaleSlice(mv, 0.01, 2, false)
		}
		if mt, err := readRegionFresh(cfg, base+ModTOff, s.Modules); err == nil {
			s.ModuleT = scaleSlice(mt, 0.1, 1, true)
		}
		if s.Cells > 0 {
			if cv, err := readRegionFresh(cfg, base+CellVOff, s.Cells); err == nil {
				s.CellV = scaleSlice(cv, 0.001, 3, false)
			}
		}
		s.HasDetail = len(s.ModuleV) > 0
	}
	return s
}

// ReadSystem reads identity + aggregate + all six strings (+ master detail).
// Reads are strictly sequential (fresh connection per block).
func ReadSystem(cfg Config) (*SystemSnapshot, error) {
	start := time.Now()
	snap := &SystemSnapshot{
		Timestamp: time.Now().Format(time.RFC3339),
		IP:        cfg.IP, Port: portOr502(cfg.Port), Unit: unitOr1(cfg.Unit),
	}

	// Identity is also the reachability probe.
	ident, err := readBaseFresh(cfg, IdentBase, 0x10)
	if err != nil {
		return nil, err
	}
	fw := at(ident, 0x0A)
	snap.Identity = Identity{OK: true, Name: decodeASCII(ident[:min(10, len(ident))]),
		Firmware: "V" + itoa(int(fw>>8)) + "." + itoa(int(fw&0xFF)), Build: int(at(ident, 0x0B))}

	// Aggregate block (0x1100). Offsets mirror the per-string §3.6 layout.
	if agg, err := readBaseFresh(cfg, AggBase, 0x52); err == nil {
		snap.HeartbeatA = int(at(agg, 0x3C))
		tv := round(float64(at(agg, 0x03))*0.1, 1)
		cur := round(float64(S32(at(agg, 0x04), at(agg, 0x05)))*0.01, 2)
		snap.Aggregate = Aggregate{
			OK: true, Piles: int(at(agg, 0x31)), TotalV: tv, Current: cur,
			PowerKW:  round(tv*cur/1000, 3),
			SOC:      int(at(agg, 0x07)),
			SOH:      int(at(agg, 0x20)),
			Temp:     round(float64(S16(at(agg, 0x06)))*0.1, 1),
			CellMaxV: round(float64(at(agg, 0x10))*0.001, 3),
			CellMinV: round(float64(at(agg, 0x11))*0.001, 3),
		}
	}

	// Six strings.
	snap.Strings = make([]StringInfo, 0, 6)
	for n := 1; n <= 6; n++ {
		snap.Strings = append(snap.Strings, readString(cfg, n))
	}

	// Heartbeat sample 2 -> link liveness.
	if hb, err := readBaseFresh(cfg, HeartReg, 1); err == nil {
		snap.HeartbeatB = int(at(hb, 0))
	}
	snap.LinkLive = snap.HeartbeatA != snap.HeartbeatB

	snap.Chain = chainStatus(snap.Strings)
	snap.Health = assessHealth(snap)
	snap.ReadMillis = time.Since(start).Milliseconds()
	return snap, nil
}

// chainStatus computes online count + where the chain stops (contiguous from S1).
func chainStatus(strings []StringInfo) ChainStatus {
	cs := ChainStatus{Total: 6, States: make([]string, len(strings))}
	contiguous := 0
	contiguousBroken := false
	for i, s := range strings {
		cs.States[i] = s.Status
		if s.Status == StatusEnumerated {
			cs.Online++
			if !contiguousBroken {
				contiguous++
			}
		} else {
			contiguousBroken = true
		}
	}
	cs.StopsAfter = contiguous
	switch {
	case cs.Online == 6:
		cs.Verdict = "All 6 strings online (chain complete)."
	case cs.Online == 0:
		cs.Verdict = "No strings on the chain — master in standby or not running."
	default:
		cs.Verdict = "Chain enumerates " + itoa(cs.Online) + " of 6; stops after string " + itoa(contiguous) + "."
	}
	return cs
}

func portOr502(p int) int {
	if p == 0 {
		return 502
	}
	return p
}

func hex4(v int) string {
	const digits = "0123456789ABCDEF"
	return "0x" + string([]byte{
		digits[(v>>12)&0xF], digits[(v>>8)&0xF], digits[(v>>4)&0xF], digits[v&0xF],
	})
}

func scaleSlice(words []uint16, sc float64, dec int, signed bool) []float64 {
	out := make([]float64, len(words))
	for i, w := range words {
		v := float64(w)
		if signed {
			v = float64(S16(w))
		}
		out[i] = round(v*sc, dec)
	}
	return out
}

func round(v float64, dec int) float64 {
	p := 1.0
	for i := 0; i < dec; i++ {
		p *= 10
	}
	if v >= 0 {
		return float64(int64(v*p+0.5)) / p
	}
	return float64(int64(v*p-0.5)) / p
}
