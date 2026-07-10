// Package dse is a READ-ONLY driver for the DeepSea DSE8610 MKII genset
// controller over Modbus TCP, using the GenComm standard FIXED public pages
// (SP-228): register = page*256 + offset.
//
//	Page 4 (base 1024) — basic instrumentation (engine + generator electrical)
//	Page 6 (base 1536) — derived instrumentation (total W / VA / VAr / PF)
//	Page 7 (base 1792) — accumulated instrumentation (kWh, starts, run hours)
//
// Reading these pages is non-invasive: no controller reconfiguration (GenConfig
// pages 166-169) is needed. The page-16 control keys (regs 4104/4105) are
// deliberately NOT implemented — this driver never writes.
package dse

import (
	"time"

	"bms-monitor/internal/modbustcp"
)

// GenComm fixed page bases and read windows.
const (
	Page4Base  = 4 * 256 // 1024 — basic instrumentation
	Page4Count = 48      // offsets 0x00..0x2F (through mains voltages)
	Page6Base  = 6 * 256 // 1536 — derived instrumentation
	Page6Count = 24      // offsets 0x00..0x17
	Page7Base  = 7 * 256 // 1792 — accumulated instrumentation
	Page7Count = 20      // offsets 0x00..0x13 (kWh, run time, starts)
)

// GenComm "unimplemented" sentinels (SP-228): a register the controller does
// not populate reads back as all-ones (unsigned) / max-positive (signed).
const (
	unimpU16 = 0xFFFF
	unimpS16 = 0x7FFF
	unimpU32 = 0xFFFFFFFF
	unimpS32 = 0x7FFFFFFF
)

// Metric is one decoded value; OK=false means the controller does not
// populate that GenComm offset (sentinel read back).
type Metric struct {
	OK    bool    `json:"ok"`
	Value float64 `json:"value"`
}

// Snapshot is the decoded genset state.
type Snapshot struct {
	OK        bool   `json:"ok"`
	Timestamp string `json:"timestamp"`

	// Engine (page 4)
	OilPressureKPa Metric `json:"oilPressureKPa"`
	CoolantTempC   Metric `json:"coolantTempC"`
	FuelLevelPct   Metric `json:"fuelLevelPct"`
	ChargeAltV     Metric `json:"chargeAltV"`
	BatteryV       Metric `json:"batteryV"`
	EngineRPM      Metric `json:"engineRPM"`

	// Generator electrical (page 4)
	FreqHz Metric    `json:"freqHz"`
	VLn    [3]Metric `json:"vLn"` // L1-N, L2-N, L3-N
	VLl    [3]Metric `json:"vLl"` // L1-L2, L2-L3, L3-L1
	AmpsL  [3]Metric `json:"ampsL"`
	WattsL [3]Metric `json:"wattsL"`

	// Derived (page 6)
	TotalW   Metric `json:"totalW"`
	TotalVA  Metric `json:"totalVA"`
	TotalVAr Metric `json:"totalVAr"`
	AvgPF    Metric `json:"avgPF"`

	// Accumulated (page 7)
	PosKWh   Metric `json:"posKWh"`
	NegKWh   Metric `json:"negKWh"`
	RunHours Metric `json:"runHours"`
	Starts   Metric `json:"starts"`

	Running    bool  `json:"running"` // engine turning (RPM above crank threshold)

	// Named alarm conditions (GenComm page 8). AlarmsOK=false means the
	// controller did not answer the alarm page (telemetry is still valid).
	AlarmsOK bool    `json:"alarmsOk"`
	Alarms   []Alarm `json:"alarms"` // ACTIVE alarms only

	ReadMillis int64 `json:"readMillis"`
}

// HasSevereAlarm reports an active shutdown / electrical-trip alarm.
func (s *Snapshot) HasSevereAlarm() bool {
	for _, a := range s.Alarms {
		if a.Severe() {
			return true
		}
	}
	return false
}

func mU16(block []uint16, off int, scale float64) Metric {
	v := modbustcp.At(block, off)
	if v == unimpU16 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(v)*scale, 2)}
}

func mS16(block []uint16, off int, scale float64) Metric {
	v := modbustcp.At(block, off)
	if v == unimpU16 || modbustcp.S16(v) == unimpS16 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(modbustcp.S16(v))*scale, 2)}
}

func mU32(block []uint16, off int, scale float64) Metric {
	v := modbustcp.U32(modbustcp.At(block, off), modbustcp.At(block, off+1))
	if v == unimpU32 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(v)*scale, 2)}
}

func mS32(block []uint16, off int, scale float64) Metric {
	v := modbustcp.S32(modbustcp.At(block, off), modbustcp.At(block, off+1))
	if v == int64(unimpS32) || uint32(v) == unimpU32 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(v)*scale, 2)}
}

// Decode builds a Snapshot from the three raw page blocks (each starting at
// its page base). Offsets per the plan's Appendix A (GenComm SP-228).
func Decode(page4, page6, page7 []uint16) Snapshot {
	s := Snapshot{OK: true, Alarms: []Alarm{}}

	// Page 4 — basic instrumentation (offsets are page-relative).
	s.OilPressureKPa = mU16(page4, 0, 1)
	s.CoolantTempC = mS16(page4, 1, 1)
	s.FuelLevelPct = mU16(page4, 3, 1)
	s.ChargeAltV = mU16(page4, 4, 0.1)
	s.BatteryV = mU16(page4, 5, 0.1)
	s.EngineRPM = mU16(page4, 6, 1)
	s.FreqHz = mU16(page4, 7, 0.1)
	for i := 0; i < 3; i++ {
		s.VLn[i] = mU32(page4, 8+2*i, 0.1)   // 1032..1037
		s.VLl[i] = mU32(page4, 14+2*i, 0.1)  // 1038..1043
		s.AmpsL[i] = mU32(page4, 20+2*i, 0.1) // 1044..1049
		s.WattsL[i] = mS32(page4, 28+2*i, 1) // 1052..1057
	}

	// Page 6 — derived instrumentation.
	s.TotalW = mS32(page6, 0, 1)   // 1536
	s.TotalVA = mS32(page6, 8, 1)  // 1544
	s.TotalVAr = mS32(page6, 16, 1) // 1552
	s.AvgPF = mS16(page6, 21, 0.01) // 1557

	// Page 7 — accumulated instrumentation.
	s.PosKWh = mU32(page7, 8, 0.1)  // 1800
	s.NegKWh = mU32(page7, 10, 0.1) // 1802
	s.RunHours = mU32(page7, 6, 1.0/3600.0) // 1798 engine run time (seconds) — confirm on site
	s.Starts = mU32(page7, 16, 1)   // 1808

	s.Running = s.EngineRPM.OK && s.EngineRPM.Value > 300
	return s
}

// Read polls the three fixed pages (three sequential fresh-connection reads)
// and decodes them.
func Read(ep modbustcp.Endpoint) (*Snapshot, error) {
	start := time.Now()
	p4, err := modbustcp.ReadHoldingFresh(ep, Page4Base, Page4Count)
	if err != nil {
		return nil, err
	}
	p6, err := modbustcp.ReadHoldingFresh(ep, Page6Base, Page6Count)
	if err != nil {
		return nil, err
	}
	p7, err := modbustcp.ReadHoldingFresh(ep, Page7Base, Page7Count)
	if err != nil {
		return nil, err
	}
	s := Decode(p4, p6, p7)

	// Page 8 named alarms: count register first, then the packed conditions.
	// A failure here degrades gracefully — telemetry stays valid without it.
	s.Alarms = []Alarm{}
	if cnt, err := modbustcp.ReadHoldingFresh(ep, Page8Base, 1); err == nil && len(cnt) == 1 {
		n := cnt[0]
		if n > 0 && n != 0xFFFF && n <= alarmCountMax {
			words := (int(n) + 3) / 4
			if regs, err := modbustcp.ReadHoldingFresh(ep, Page8Base+1, words); err == nil {
				s.Alarms = DecodeAlarms(n, regs)
				s.AlarmsOK = true
			}
		} else if n == 0 {
			s.AlarmsOK = true // controller answered: zero named alarms configured
		}
	}

	s.Timestamp = time.Now().Format(time.RFC3339)
	s.ReadMillis = time.Since(start).Milliseconds()
	return &s, nil
}
