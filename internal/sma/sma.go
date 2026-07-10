// Package sma is a READ-ONLY driver for the SMA Sunny Tripower 25000TL-30 PV
// inverters over Modbus TCP (SMA-native profile, unit ID 3, port 502). Each of
// the three inverters is its own TCP endpoint (.51/.52/.53), so the shared
// unit ID is fine.
//
// SMA-native uses FIXED scaling (FIX0/1/2/3 = /1 /10 /100 /1000, TEMP = /10)
// — no SunSpec scale-factor chasing. Register addresses from the STP15-25TL-30
// device-specific list are used DIRECTLY as Modbus start addresses (SMA's
// published addresses are protocol addresses; the generic "PDU = reg-1" note
// in the reference does not apply to the SMA-native profile — flagged for
// field validation in README-plant.md).
//
// NaN sentinels: S32 0x80000000, U32 0xFFFFFFFF, enum/TAGLIST 16777213.
// This driver never writes (no setpoints, no power limits).
package sma

import (
	"time"

	"bms-monitor/internal/modbustcp"
)

// NaN sentinels (SMA-Modbus-general-TI).
const (
	nanS32  = -0x80000000
	nanU32  = 0xFFFFFFFF
	nanEnum = 16777213 // 0x00FFFFFD — TAGLIST "information not available"
)

// Read windows (device addresses, read FC03).
const (
	blockIdentStart = 30051 // device class, type, serial, firmware
	blockIdentCount = 10    // 30051..30060
	blockStatusStart = 30201 // condition, action, message, grid relay, derating
	blockStatusCount = 34    // 30201..30234
	blockYieldStart = 30513 // total yield U64, daily yield U64
	blockYieldCount = 8     // 30513..30520
	blockACStart    = 30769 // MPPT-A, AC power, phase power/voltage, current, freq
	blockACCount    = 46    // 30769..30814
	blockDCBStart   = 30953 // temperature, MPPT-B, DC-link, phase currents
	blockDCBCount   = 30    // 30953..30982
)

// Condition enum (30201).
var conditionText = map[uint32]string{
	35: "Fault", 303: "Off", 307: "OK", 455: "Warning",
}

// Grid relay enum (30217).
const (
	relayClosed = 51
	relayOpen   = 311
)

// Derating reason enum (30219) — common values.
var deratingText = map[uint32]string{
	557: "Over-temperature", 884: "Not active", 3556: "High DC voltage",
}

// Device type enum (30053).
var deviceTypeText = map[uint32]string{
	9284: "STP 20000TL-30", 9285: "STP 25000TL-30",
	9336: "STP 15000TL-30", 9337: "STP 17000TL-30",
}

// Metric is one decoded value; OK=false when the register reads NaN.
type Metric struct {
	OK    bool    `json:"ok"`
	Value float64 `json:"value"`
}

func mS32(hi, lo uint16, div float64, dec int) Metric {
	v := modbustcp.S32(hi, lo)
	if v == nanS32 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(v)/div, dec)}
}

func mU32(hi, lo uint16, div float64, dec int) Metric {
	v := modbustcp.U32(hi, lo)
	if v == nanU32 {
		return Metric{}
	}
	return Metric{OK: true, Value: modbustcp.Round(float64(v)/div, dec)}
}

// MPPT is one DC input (string) measurement set.
type MPPT struct {
	CurrentA Metric `json:"currentA"`
	VoltageV Metric `json:"voltageV"`
	PowerW   Metric `json:"powerW"`
}

// Snapshot is the decoded PV inverter state.
type Snapshot struct {
	OK        bool   `json:"ok"`
	Timestamp string `json:"timestamp"`

	DeviceTypeRaw uint32 `json:"deviceTypeRaw"` // 30053 (9285 = STP 25000TL-30)
	DeviceType    string `json:"deviceType"`
	Serial        uint32 `json:"serial"` // 30057

	ConditionRaw uint32 `json:"conditionRaw"` // 30201
	Condition    string `json:"condition"`    // Fault / Off / OK / Warning
	GridRelayRaw uint32 `json:"gridRelayRaw"` // 30217
	GridRelay    string `json:"gridRelay"`    // Closed / Open / Unknown
	Derating     string `json:"derating"`     // 30219 when active

	ACPowerW   Metric    `json:"acPowerW"`   // 30775
	ApparentVA Metric    `json:"apparentVA"` // 30813
	GridV      [3]Metric `json:"gridV"`      // 30783/85/87
	GridA      [3]Metric `json:"gridA"`      // 30977/79/81
	FreqHz     Metric    `json:"freqHz"`     // 30803

	MPPTA     MPPT   `json:"mpptA"` // 30769/71/73
	MPPTB     MPPT   `json:"mpptB"` // 30957/59/61
	DCPowerW  Metric `json:"dcPowerW"` // A + B
	DCLinkV   Metric `json:"dcLinkV"`  // 30975

	TotalYieldKWh Metric `json:"totalYieldKWh"` // 30513 (U64 Wh)
	DailyYieldKWh Metric `json:"dailyYieldKWh"` // 30517 (U64 Wh)

	InternalTempC Metric `json:"internalTempC"` // 30953 (TEMP /10)

	ReadMillis int64 `json:"readMillis"`
}

// Producing reports whether the inverter is exporting meaningful AC power.
func (s *Snapshot) Producing() bool { return s.ACPowerW.OK && s.ACPowerW.Value > 50 }

// Decode builds a Snapshot from the raw block reads.
func Decode(ident, status, yield, ac, dcb []uint16) Snapshot {
	s := Snapshot{OK: true}
	iw := func(reg int) uint16 { return modbustcp.At(ident, reg-blockIdentStart) }
	sw := func(reg int) uint16 { return modbustcp.At(status, reg-blockStatusStart) }
	yw := func(reg int) uint16 { return modbustcp.At(yield, reg-blockYieldStart) }
	aw := func(reg int) uint16 { return modbustcp.At(ac, reg-blockACStart) }
	dw := func(reg int) uint16 { return modbustcp.At(dcb, reg-blockDCBStart) }

	s.DeviceTypeRaw = modbustcp.U32(iw(30053), iw(30054))
	if t, ok := deviceTypeText[s.DeviceTypeRaw]; ok {
		s.DeviceType = t
	} else {
		s.DeviceType = "Sunny Tripower"
	}
	s.Serial = modbustcp.U32(iw(30057), iw(30058))

	s.ConditionRaw = modbustcp.U32(sw(30201), sw(30202))
	if t, ok := conditionText[s.ConditionRaw]; ok {
		s.Condition = t
	} else if s.ConditionRaw == nanEnum {
		s.Condition = "Unknown"
	} else {
		s.Condition = "Code " + utoa(s.ConditionRaw)
	}
	relay := modbustcp.U32(sw(30217), sw(30218))
	switch relay {
	case relayClosed:
		s.GridRelay = "Closed"
	case relayOpen:
		s.GridRelay = "Open"
	default:
		s.GridRelay = "Unknown"
	}
	s.GridRelayRaw = relay
	if der := modbustcp.U32(sw(30219), sw(30220)); der != nanEnum {
		if t, ok := deratingText[der]; ok {
			s.Derating = t
		} else if der != 0 {
			s.Derating = "Code " + utoa(der)
		}
	}
	if s.Derating == "Not active" {
		s.Derating = ""
	}

	// Yields are U64 Wh; NaN when all-ones.
	if tot := modbustcp.U64(yw(30513), yw(30514), yw(30515), yw(30516)); tot != ^uint64(0) {
		s.TotalYieldKWh = Metric{OK: true, Value: modbustcp.Round(float64(tot)/1000, 1)}
	}
	if day := modbustcp.U64(yw(30517), yw(30518), yw(30519), yw(30520)); day != ^uint64(0) {
		s.DailyYieldKWh = Metric{OK: true, Value: modbustcp.Round(float64(day)/1000, 2)}
	}

	// MPPT-A + AC block.
	s.MPPTA = MPPT{
		CurrentA: mS32(aw(30769), aw(30770), 1000, 2), // FIX3
		VoltageV: mS32(aw(30771), aw(30772), 100, 1),  // FIX2
		PowerW:   mS32(aw(30773), aw(30774), 1, 0),    // FIX0
	}
	s.ACPowerW = mS32(aw(30775), aw(30776), 1, 0)
	for i := 0; i < 3; i++ {
		s.GridV[i] = mU32(aw(30783+2*i), aw(30784+2*i), 100, 1)
	}
	s.FreqHz = mU32(aw(30803), aw(30804), 100, 2)
	s.ApparentVA = mS32(aw(30813), aw(30814), 1, 0)

	// Temperature, MPPT-B, DC link, phase currents.
	s.InternalTempC = mS32(dw(30953), dw(30954), 10, 1) // TEMP
	s.MPPTB = MPPT{
		CurrentA: mS32(dw(30957), dw(30958), 1000, 2),
		VoltageV: mS32(dw(30959), dw(30960), 100, 1),
		PowerW:   mS32(dw(30961), dw(30962), 1, 0),
	}
	s.DCLinkV = mS32(dw(30975), dw(30976), 100, 1)
	for i := 0; i < 3; i++ {
		s.GridA[i] = mS32(dw(30977+2*i), dw(30978+2*i), 1000, 2)
	}

	if s.MPPTA.PowerW.OK || s.MPPTB.PowerW.OK {
		s.DCPowerW = Metric{OK: true, Value: s.MPPTA.PowerW.Value + s.MPPTB.PowerW.Value}
	}
	return s
}

// Read polls one inverter: five sequential FC03 block reads on fresh
// connections. SMA asks for >=1 s between transfers under heavy polling; our
// per-endpoint cadence is one snapshot per plant refresh, which is far below
// the device's ceiling.
func Read(ep modbustcp.Endpoint) (*Snapshot, error) {
	start := time.Now()
	ident, err := modbustcp.ReadHoldingFresh(ep, blockIdentStart, blockIdentCount)
	if err != nil {
		return nil, err
	}
	status, err := modbustcp.ReadHoldingFresh(ep, blockStatusStart, blockStatusCount)
	if err != nil {
		return nil, err
	}
	yield, err := modbustcp.ReadHoldingFresh(ep, blockYieldStart, blockYieldCount)
	if err != nil {
		return nil, err
	}
	ac, err := modbustcp.ReadHoldingFresh(ep, blockACStart, blockACCount)
	if err != nil {
		return nil, err
	}
	dcb, err := modbustcp.ReadHoldingFresh(ep, blockDCBStart, blockDCBCount)
	if err != nil {
		return nil, err
	}
	s := Decode(ident, status, yield, ac, dcb)
	s.Timestamp = time.Now().Format(time.RFC3339)
	s.ReadMillis = time.Since(start).Milliseconds()
	return &s, nil
}

func utoa(v uint32) string {
	if v == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
