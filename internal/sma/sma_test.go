package sma

import "testing"

func blocks() (ident, status, yield, ac, dcb []uint16) {
	mk := func(n int) []uint16 { return make([]uint16, n) }
	return mk(blockIdentCount), mk(blockStatusCount), mk(blockYieldCount), mk(blockACCount), mk(blockDCBCount)
}

func setU32(b []uint16, base, reg int, v uint32) {
	b[reg-base] = uint16(v >> 16)
	b[reg-base+1] = uint16(v & 0xFFFF)
}

func TestDecodeProducing(t *testing.T) {
	ident, status, yield, ac, dcb := blocks()
	setU32(ident, blockIdentStart, 30053, 9285)      // STP 25000TL-30
	setU32(ident, blockIdentStart, 30057, 3010123456) // serial
	setU32(status, blockStatusStart, 30201, 307)     // OK
	setU32(status, blockStatusStart, 30217, 51)      // relay closed
	setU32(status, blockStatusStart, 30219, 884)     // derating not active

	// Yields: total 45.6789 MWh, daily 87.65 kWh (U64 Wh).
	yield[30513-blockYieldStart+3] = 0
	setU32(yield, blockYieldStart, 30515, 45678900)
	setU32(yield, blockYieldStart, 30519, 87650)

	setU32(ac, blockACStart, 30769, 12345)  // MPPT-A 12.345 A (FIX3)
	setU32(ac, blockACStart, 30771, 60520)  // 605.20 V (FIX2)
	setU32(ac, blockACStart, 30773, 7473)   // 7473 W (FIX0)
	setU32(ac, blockACStart, 30775, 14980)  // AC 14980 W
	setU32(ac, blockACStart, 30783, 23951)  // L1 239.51 V (FIX2)
	setU32(ac, blockACStart, 30803, 4999)   // 49.99 Hz (FIX2)
	setU32(ac, blockACStart, 30813, 15100)  // 15100 VA

	setU32(dcb, blockDCBStart, 30953, 412)  // 41.2 C (TEMP /10)
	setU32(dcb, blockDCBStart, 30957, 12000) // MPPT-B 12.000 A
	setU32(dcb, blockDCBStart, 30959, 60000) // 600.00 V
	setU32(dcb, blockDCBStart, 30961, 7500)  // 7500 W
	setU32(dcb, blockDCBStart, 30977, 21500) // L1 21.5 A (FIX3)

	s := Decode(ident, status, yield, ac, dcb)

	if s.DeviceType != "STP 25000TL-30" || s.DeviceTypeRaw != 9285 {
		t.Errorf("device type = %q (%d)", s.DeviceType, s.DeviceTypeRaw)
	}
	if s.Serial != 3010123456 {
		t.Errorf("serial = %d", s.Serial)
	}
	if s.Condition != "OK" || s.GridRelay != "Closed" || s.Derating != "" {
		t.Errorf("condition=%q relay=%q derating=%q", s.Condition, s.GridRelay, s.Derating)
	}
	if !s.ACPowerW.OK || s.ACPowerW.Value != 14980 {
		t.Errorf("AC power = %+v want 14980", s.ACPowerW)
	}
	if !s.MPPTA.CurrentA.OK || s.MPPTA.CurrentA.Value != 12.35 {
		t.Errorf("MPPT-A current = %+v want 12.35", s.MPPTA.CurrentA)
	}
	if !s.MPPTA.VoltageV.OK || s.MPPTA.VoltageV.Value != 605.2 {
		t.Errorf("MPPT-A voltage = %+v want 605.2", s.MPPTA.VoltageV)
	}
	if !s.DCPowerW.OK || s.DCPowerW.Value != 14973 {
		t.Errorf("DC power = %+v want 14973 (A 7473 + B 7500)", s.DCPowerW)
	}
	if !s.GridV[0].OK || s.GridV[0].Value != 239.5 {
		t.Errorf("L1 V = %+v want 239.5", s.GridV[0])
	}
	if !s.FreqHz.OK || s.FreqHz.Value != 49.99 {
		t.Errorf("freq = %+v want 49.99", s.FreqHz)
	}
	if !s.InternalTempC.OK || s.InternalTempC.Value != 41.2 {
		t.Errorf("temp = %+v want 41.2", s.InternalTempC)
	}
	if !s.TotalYieldKWh.OK || s.TotalYieldKWh.Value != 45678.9 {
		t.Errorf("total yield = %+v want 45678.9", s.TotalYieldKWh)
	}
	if !s.DailyYieldKWh.OK || s.DailyYieldKWh.Value != 87.65 {
		t.Errorf("daily yield = %+v want 87.65", s.DailyYieldKWh)
	}
	if !s.GridA[0].OK || s.GridA[0].Value != 21.5 {
		t.Errorf("L1 A = %+v want 21.5", s.GridA[0])
	}
	if !s.Producing() {
		t.Error("14.98 kW should report Producing")
	}
}

func TestNaNSentinels(t *testing.T) {
	ident, status, yield, ac, dcb := blocks()
	setU32(status, blockStatusStart, 30201, nanEnum) // condition NaN
	setU32(status, blockStatusStart, 30217, nanEnum) // relay NaN
	setU32(ac, blockACStart, 30775, 0x80000000)      // S32 NaN
	setU32(ac, blockACStart, 30803, 0xFFFFFFFF)      // U32 NaN
	for i := range yield {
		yield[i] = 0xFFFF // U64 NaN
	}
	setU32(dcb, blockDCBStart, 30957, 0x80000000)
	setU32(dcb, blockDCBStart, 30961, 0x80000000)
	setU32(ac, blockACStart, 30773, 0x80000000)

	s := Decode(ident, status, yield, ac, dcb)
	if s.Condition != "Unknown" || s.GridRelay != "Unknown" {
		t.Errorf("condition=%q relay=%q want Unknown/Unknown", s.Condition, s.GridRelay)
	}
	if s.ACPowerW.OK || s.FreqHz.OK || s.TotalYieldKWh.OK || s.DailyYieldKWh.OK {
		t.Errorf("NaN values must decode OK=false")
	}
	if s.DCPowerW.OK {
		t.Errorf("DC power should be NaN when both MPPT powers are NaN: %+v", s.DCPowerW)
	}
	if s.Producing() {
		t.Error("NaN AC power must not report Producing")
	}
}

func TestConditionEnums(t *testing.T) {
	ident, status, yield, ac, dcb := blocks()
	setU32(status, blockStatusStart, 30201, 35) // Fault
	setU32(status, blockStatusStart, 30217, 311)
	s := Decode(ident, status, yield, ac, dcb)
	if s.Condition != "Fault" || s.GridRelay != "Open" {
		t.Errorf("condition=%q relay=%q want Fault/Open", s.Condition, s.GridRelay)
	}

	setU32(status, blockStatusStart, 30201, 455) // Warning
	setU32(status, blockStatusStart, 30219, 557) // over-temp derating
	s = Decode(ident, status, yield, ac, dcb)
	if s.Condition != "Warning" || s.Derating != "Over-temperature" {
		t.Errorf("condition=%q derating=%q", s.Condition, s.Derating)
	}
}

func TestNegativePower(t *testing.T) {
	// S32 FIX0 can be negative (standby self-consumption).
	ident, status, yield, ac, dcb := blocks()
	setU32(ac, blockACStart, 30775, 0xFFFFFFF6) // -10 W
	s := Decode(ident, status, yield, ac, dcb)
	if !s.ACPowerW.OK || s.ACPowerW.Value != -10 {
		t.Errorf("AC power = %+v want -10", s.ACPowerW)
	}
}
