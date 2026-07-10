// Package oztek is a READ-ONLY driver for the OzTek OZPCS-RS40 40 kVA battery
// PCS (UM-0075 Rev N), reached as Modbus RTU behind the USR-TCP232-410s
// RS485->TCP gateway at 192.168.0.41. All three units (slave IDs 1/2/3) share
// ONE physical RS485 line: modbustcp's per-host mutex serializes every read.
//
// Register map: SunSpec DER models 701 (AC measurement), 714 (DC measurement),
// 715 (DER control — heartbeat read only) and OzTek vendor model 64340
// (control & status — status registers read only). SunSpec registers are
// 1-based: PDU address = register - 1.
//
// NO WRITE PATH. In particular, never touch 41740 (controller heartbeat —
// writing it arms heartbeat supervision and a later loss faults the PCS),
// 41742/41743 (reset / set-operation) or the 64308 grid-forming commands.
package oztek

import (
	"time"

	"bms-monitor/internal/modbustcp"
)

// Read windows (1-based SunSpec register numbers; -1 applied at read time).
const (
	block701Start = 40088 // model 701: states, alarms, AC measurements + SFs
	block701Count = 120   // 40088..40207 (covers SFs 40198..40207)
	block714Start = 41690 // model 714: DC measurements + SFs
	block714Count = 44    // 41690..41733
	blockCtlStart = 41735 // model 715 (heartbeat @41738) + 64340 status @41744..
	blockCtlCount = 28    // 41735..41762 (state/warning/fault/factory-fault)
)

// off computes the index of a 1-based register inside a block read that
// started at 1-based register `start`.
func off(start, reg int) int { return reg - start }

// SunSpec "not implemented" sentinels.
const (
	nanS16   = 0x8000
	nanU16   = 0xFFFF
	nanSF    = 0x8000
	nanU32Hi = 0xFFFF // u32 NaN = 0xFFFFFFFF
)

// sf returns the scale factor multiplier 10^sf from a Sunssf register.
func sfMul(raw uint16) float64 {
	if raw == nanSF {
		return 1
	}
	n := modbustcp.S16(raw)
	m := 1.0
	for i := 0; i < n; i++ {
		m *= 10
	}
	for i := 0; i > n; i-- {
		m /= 10
	}
	return m
}

// Metric is one decoded value; OK=false when the register reads NaN.
type Metric struct {
	OK    bool    `json:"ok"`
	Value float64 `json:"value"`
}

func mS16SF(raw uint16, sfRaw uint16, dec int) Metric {
	if raw == nanS16 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(modbustcp.S16(raw))*sfMul(sfRaw), dec)}
}

func mU16SF(raw uint16, sfRaw uint16, dec int) Metric {
	if raw == nanU16 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(raw)*sfMul(sfRaw), dec)}
}

func mU32SF(hi, lo uint16, sfRaw uint16, dec int) Metric {
	if hi == nanU32Hi && lo == nanU16 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(modbustcp.U32(hi, lo))*sfMul(sfRaw), dec)}
}

// mFixed decodes a model-714 register with a FIXED scale (no SF companion —
// DC totals are 0.1 A / 10 W fixed per UM-0075).
func mFixedS16(raw uint16, scale float64, dec int) Metric {
	if raw == nanS16 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(modbustcp.S16(raw))*scale, dec)}
}

// Operating-state enum (64340 reg 41746) — the best dashboard state.
var stateText = map[int]string{
	0: "Initialize", 1: "Fault", 2: "Calibrate", 3: "Disabled",
	4: "Charge wait", 5: "Charging", 6: "Standby", 7: "Turn-on delay",
	8: "Online (grid-tie)", 9: "Offline", 10: "Active ride-through",
	11: "Passive ride-through", 12: "Online (grid-form)", 13: "Power-down",
	16: "Turn-off", 17: "Island-transfer wait", 18: "Service-disabled",
}

// StateText renders the 41746 operating-state enum.
func StateText(v int) string {
	if t, ok := stateText[v]; ok {
		return t
	}
	return "State " + itoa(v)
}

// StateOnline reports whether the PCS is exporting/absorbing on the grid.
func StateOnline(v int) bool { return v == 8 || v == 12 }

type bitLabel struct {
	Bit   int
	Label string
}

// 41758 PCS fault status (UM-0075 §64340).
var faultBits = []bitLabel{
	{0, "HW over-current A"}, {1, "HW over-current B"}, {2, "HW over-current C"},
	{3, "RMS over-current A"}, {4, "RMS over-current B"}, {5, "RMS over-current C"},
	{6, "DC over-current"}, {7, "Grid over-voltage AB"}, {8, "Grid over-voltage BC"},
	{9, "Grid over-voltage CA"}, {10, "HW DC over-voltage"}, {11, "DC over-voltage"},
	{12, "DC under-voltage"}, {13, "Ride-through low voltage"}, {14, "Ride-through high voltage"},
	{15, "Ride-through low frequency"}, {16, "Ride-through high frequency"}, {17, "Island detected"},
	{19, "Temperature fault"}, {20, "E-STOP"}, {21, "Communication error"},
	{22, "Power-down error"}, {23, "Invalid user config"}, {24, "Invalid model"},
	{25, "Factory fault (see factory status)"}, {26, "Saturation A"}, {27, "Saturation B"},
	{28, "Saturation C"}, {31, "AC current overload trip"},
}

// 41756 PCS warning status.
var warningBits = []bitLabel{
	{0, "High AC current A"}, {1, "High AC current B"}, {2, "High AC current C"},
	{3, "High DC current"}, {4, "High grid voltage AB"}, {5, "High grid voltage BC"},
	{6, "High grid voltage CA"}, {7, "High DC voltage"}, {8, "Low DC voltage"},
	{9, "AC current limit"}, {10, "DC power limit"}, {11, "AC power limit"},
	{12, "Grid out of tolerance"}, {13, "Resume delay"}, {14, "Island detected"},
	{15, "PLL not locked"}, {16, "Temperature warning"}, {19, "Fan warning"},
	{21, "Active power limited"}, {22, "HVRT override"}, {23, "TVS error"},
	{24, "Volt-VAR active"}, {25, "Volt-Watt active"}, {26, "Freq-Watt active"},
	{27, "Loss of phase"}, {28, "Negative-sequence current limit"},
	{29, "Watt-VAR active"}, {31, "AC current overload"},
}

// 41760 factory fault status (read when fault bit 25 is set).
var factoryBits = []bitLabel{
	{14, "Link over-voltage"}, {15, "Link voltage imbalance"},
	{16, "Pre-charge timeout"}, {17, "Bias under-voltage"},
	{18, "Contactor interlock"}, {19, "DC/DC comm error"},
	{20, "Datalog error"}, {21, "Invalid factory config"},
	{22, "Config EEPROM error"}, {23, "Calibration error"},
}

// 40091 DER alarm bitfield (model 701).
var derAlarmBits = []bitLabel{
	{1, "DC over-voltage"}, {2, "AC disconnect open"}, {3, "DC disconnect open"},
	{4, "Grid disconnect"}, {7, "Over-temperature"}, {8, "Over-frequency"},
	{9, "Under-frequency"}, {10, "AC over-voltage"}, {11, "AC under-voltage"},
	{13, "Under-temperature"}, {16, "Manufacturer alarm"},
}

func decode32(v uint32, table []bitLabel) []string {
	out := []string{}
	for _, b := range table {
		if v&(1<<uint(b.Bit)) != 0 {
			out = append(out, b.Label)
		}
	}
	return out
}

// DecodeFaults / DecodeWarnings / DecodeFactory / DecodeDERAlarms expose the
// bitfield tables for tests and the health engine.
func DecodeFaults(v uint32) []string    { return decode32(v, faultBits) }
func DecodeWarnings(v uint32) []string  { return decode32(v, warningBits) }
func DecodeFactory(v uint32) []string   { return decode32(v, factoryBits) }
func DecodeDERAlarms(v uint32) []string { return decode32(v, derAlarmBits) }

// Snapshot is the decoded PCS state for one unit.
type Snapshot struct {
	OK        bool   `json:"ok"`
	Timestamp string `json:"timestamp"`
	Unit      int    `json:"unit"`

	State         int    `json:"state"` // 41746 enum
	StateText     string `json:"stateText"`
	Online        bool   `json:"online"`        // grid-tie or grid-form
	GridConnected bool   `json:"gridConnected"` // 40090
	GridForming   bool   `json:"gridForming"`   // DER mode bit 1

	ACPowerW   Metric `json:"acPowerW"`
	ReactiveVAR Metric `json:"reactiveVAR"`
	ApparentVA Metric `json:"apparentVA"`
	PowerFactor Metric `json:"powerFactor"`
	ACCurrentA Metric `json:"acCurrentA"`
	VLl        Metric `json:"vLl"` // average line-line voltage
	VLn        Metric `json:"vLn"` // average line-neutral voltage
	FreqHz     Metric `json:"freqHz"`

	DCVoltage Metric `json:"dcVoltage"` // port 1 (battery bus)
	DCCurrent Metric `json:"dcCurrent"`
	DCPowerW  Metric `json:"dcPowerW"`

	CabinetTempC  Metric `json:"cabinetTempC"`
	HeatsinkTempC Metric `json:"heatsinkTempC"`

	Heartbeat uint32 `json:"heartbeat"` // 41738, +1/s — liveness

	AlarmRaw   uint32   `json:"alarmRaw"` // 40091
	Alarms     []string `json:"alarms"`
	WarningRaw uint32   `json:"warningRaw"` // 41756
	Warnings   []string `json:"warnings"`
	FaultRaw   uint32   `json:"faultRaw"` // 41758
	Faults     []string `json:"faults"`
	FactoryRaw uint32   `json:"factoryRaw"` // 41760
	Factory    []string `json:"factory"`

	ReadMillis int64 `json:"readMillis"`
}

// Decode builds a Snapshot from the three raw block reads.
func Decode(b701, b714, bctl []uint16) Snapshot {
	s := Snapshot{OK: true}
	a := func(reg int) uint16 { return modbustcp.At(b701, off(block701Start, reg)) }
	d := func(reg int) uint16 { return modbustcp.At(b714, off(block714Start, reg)) }
	c := func(reg int) uint16 { return modbustcp.At(bctl, off(blockCtlStart, reg)) }

	// Model 701 scale factors.
	sfA, sfV, sfHz := a(40198), a(40199), a(40200)
	sfW, sfPF, sfVA, sfVAR := a(40201), a(40202), a(40203), a(40204)
	sfTmp := a(40207)

	s.GridConnected = a(40090) == 1
	s.AlarmRaw = modbustcp.U32(a(40091), a(40092))
	s.Alarms = DecodeDERAlarms(s.AlarmRaw)
	mode := modbustcp.U32(a(40093), a(40094))
	s.GridForming = mode&0x2 != 0

	s.ACPowerW = mS16SF(a(40095), sfW, 0)
	s.ApparentVA = mS16SF(a(40096), sfVA, 0)
	s.ReactiveVAR = mS16SF(a(40097), sfVAR, 0)
	s.PowerFactor = mS16SF(a(40098), sfPF, 3)
	s.ACCurrentA = mS16SF(a(40099), sfA, 1)
	s.VLl = mU16SF(a(40100), sfV, 1)
	s.VLn = mU16SF(a(40101), sfV, 1)
	s.FreqHz = mU32SF(a(40102), a(40103), sfHz, 2)
	s.CabinetTempC = mS16SF(a(40121), sfTmp, 1)
	s.HeatsinkTempC = mS16SF(a(40122), sfTmp, 1)

	// Model 714 — DC port 1 (battery bus) with SF companions.
	sfDCA, sfDCV, sfDCW := d(41705), d(41706), d(41707)
	s.DCCurrent = mS16SF(d(41720), sfDCA, 1)
	s.DCVoltage = mU16SF(d(41721), sfDCV, 1)
	s.DCPowerW = mS16SF(d(41722), sfDCW, 0)
	// Fall back to the fixed-scale DC totals if port-1 regs read NaN.
	if !s.DCCurrent.OK {
		s.DCCurrent = mFixedS16(d(41695), 0.1, 1)
	}
	if !s.DCPowerW.OK {
		s.DCPowerW = mFixedS16(d(41696), 10, 0)
	}

	// Model 715 heartbeat + 64340 status.
	s.Heartbeat = modbustcp.U32(c(41738), c(41739))
	s.State = int(c(41746))
	s.StateText = StateText(s.State)
	s.Online = StateOnline(s.State)
	s.WarningRaw = modbustcp.U32(c(41756), c(41757))
	s.Warnings = DecodeWarnings(s.WarningRaw)
	s.FaultRaw = modbustcp.U32(c(41758), c(41759))
	s.Faults = DecodeFaults(s.FaultRaw)
	s.FactoryRaw = modbustcp.U32(c(41760), c(41761))
	if s.FaultRaw&(1<<25) != 0 {
		s.Factory = DecodeFactory(s.FactoryRaw)
	} else {
		s.Factory = []string{}
	}
	return s
}

// Read polls one PCS unit: three sequential FC03 block reads, each on a fresh
// connection, serialized with every other read to the same gateway host.
func Read(ep modbustcp.Endpoint) (*Snapshot, error) {
	start := time.Now()
	b701, err := modbustcp.ReadHoldingFresh(ep, block701Start-1, block701Count)
	if err != nil {
		return nil, err
	}
	b714, err := modbustcp.ReadHoldingFresh(ep, block714Start-1, block714Count)
	if err != nil {
		return nil, err
	}
	bctl, err := modbustcp.ReadHoldingFresh(ep, blockCtlStart-1, blockCtlCount)
	if err != nil {
		return nil, err
	}
	s := Decode(b701, b714, bctl)
	s.Unit = ep.Unit
	s.Timestamp = time.Now().Format(time.RFC3339)
	s.ReadMillis = time.Since(start).Milliseconds()
	return &s, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
