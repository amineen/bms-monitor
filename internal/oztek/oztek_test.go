package oztek

import "testing"

// blank returns a block filled with the SunSpec s16/sunssf NaN pattern.
func blank(n int) []uint16 {
	b := make([]uint16, n)
	for i := range b {
		b[i] = 0x8000
	}
	return b
}

func set(block []uint16, start, reg int, v uint16) { block[reg-start] = v }

func TestSFMul(t *testing.T) {
	cases := map[uint16]float64{
		0:      1,
		1:      10,
		2:      100,
		0xFFFF: 0.1,   // sf -1
		0xFFFD: 0.001, // sf -3
		0x8000: 1,     // NaN SF -> multiplier 1
	}
	for in, want := range cases {
		if got := sfMul(in); got != want {
			t.Errorf("sfMul(0x%04X)=%v want %v", in, got, want)
		}
	}
}

func TestDecodeACAndDC(t *testing.T) {
	b701 := blank(block701Count)
	// SFs: A -1, V -1, Hz -3, W 1, PF -3, VA 1, VAR 1, Tmp 0
	set(b701, block701Start, 40198, 0xFFFF)
	set(b701, block701Start, 40199, 0xFFFF)
	set(b701, block701Start, 40200, 0xFFFD)
	set(b701, block701Start, 40201, 1)
	set(b701, block701Start, 40202, 0xFFFD)
	set(b701, block701Start, 40203, 1)
	set(b701, block701Start, 40204, 1)
	set(b701, block701Start, 40207, 0)

	set(b701, block701Start, 40090, 1)      // grid connected
	set(b701, block701Start, 40091, 0)      // alarm hi
	set(b701, block701Start, 40092, 0x0004) // alarm lo: bit2 AC disconnect open
	set(b701, block701Start, 40093, 0)
	set(b701, block701Start, 40094, 0x0002)          // mode: grid-forming
	set(b701, block701Start, 40095, 1800)            // 1800 x10 = 18000 W
	set(b701, block701Start, 40099, 0xFFE7)          // -25 x0.1 = -2.5 A
	set(b701, block701Start, 40100, 4155)            // 415.5 V L-L
	set(b701, block701Start, 40102, 0)               // freq hi
	set(b701, block701Start, 40103, 50020)           // 50020 x0.001 = 50.02 Hz
	set(b701, block701Start, 40121, 41)              // cabinet 41 C

	b714 := blank(block714Count)
	set(b714, block714Start, 41705, 0xFFFF) // DCA sf -1
	set(b714, block714Start, 41706, 0xFFFF) // DCV sf -1
	set(b714, block714Start, 41707, 1)      // DCW sf 1
	set(b714, block714Start, 41720, 0xFF06) // -250 x0.1 = -25.0 A (charging)
	set(b714, block714Start, 41721, 7455)   // 745.5 V
	set(b714, block714Start, 41722, 0xF8BE) // -1858 x10 = -18580 W

	bctl := blank(blockCtlCount)
	set(bctl, blockCtlStart, 41738, 0)
	set(bctl, blockCtlStart, 41739, 12345) // heartbeat
	set(bctl, blockCtlStart, 41746, 8)     // online grid-tie
	set(bctl, blockCtlStart, 41756, 0)
	set(bctl, blockCtlStart, 41757, 0x0100) // warning bit 8: Low DC voltage
	set(bctl, blockCtlStart, 41758, 0)
	set(bctl, blockCtlStart, 41759, 0)

	s := Decode(b701, b714, bctl)

	if !s.ACPowerW.OK || s.ACPowerW.Value != 18000 {
		t.Errorf("AC power = %+v want 18000", s.ACPowerW)
	}
	if !s.ACCurrentA.OK || s.ACCurrentA.Value != -2.5 {
		t.Errorf("AC current = %+v want -2.5", s.ACCurrentA)
	}
	if !s.VLl.OK || s.VLl.Value != 415.5 {
		t.Errorf("VLL = %+v want 415.5", s.VLl)
	}
	if !s.FreqHz.OK || s.FreqHz.Value != 50.02 {
		t.Errorf("freq = %+v want 50.02", s.FreqHz)
	}
	if !s.DCVoltage.OK || s.DCVoltage.Value != 745.5 {
		t.Errorf("DC voltage = %+v want 745.5", s.DCVoltage)
	}
	if !s.DCCurrent.OK || s.DCCurrent.Value != -25.0 {
		t.Errorf("DC current = %+v want -25.0", s.DCCurrent)
	}
	if !s.DCPowerW.OK || s.DCPowerW.Value != -18580 {
		t.Errorf("DC power = %+v want -18580", s.DCPowerW)
	}
	if !s.CabinetTempC.OK || s.CabinetTempC.Value != 41 {
		t.Errorf("cabinet temp = %+v want 41", s.CabinetTempC)
	}
	if s.Heartbeat != 12345 {
		t.Errorf("heartbeat = %d want 12345", s.Heartbeat)
	}
	if s.State != 8 || s.StateText != "Online (grid-tie)" || !s.Online {
		t.Errorf("state = %d %q online=%v", s.State, s.StateText, s.Online)
	}
	if !s.GridConnected || !s.GridForming {
		t.Errorf("grid connected=%v forming=%v want true/true", s.GridConnected, s.GridForming)
	}
	if len(s.Warnings) != 1 || s.Warnings[0] != "Low DC voltage" {
		t.Errorf("warnings = %v want [Low DC voltage]", s.Warnings)
	}
	if len(s.Faults) != 0 {
		t.Errorf("faults = %v want none", s.Faults)
	}
	if len(s.Alarms) != 1 || s.Alarms[0] != "AC disconnect open" {
		t.Errorf("alarms = %v want [AC disconnect open]", s.Alarms)
	}
}

func TestFaultAndFactoryDecode(t *testing.T) {
	// Fault bit 12 (DC under-voltage) + bit 25 (factory fault).
	faults := DecodeFaults(1<<12 | 1<<25)
	if len(faults) != 2 || faults[0] != "DC under-voltage" {
		t.Errorf("faults = %v", faults)
	}
	factory := DecodeFactory(1<<16 | 1<<18)
	if len(factory) != 2 || factory[0] != "Pre-charge timeout" || factory[1] != "Contactor interlock" {
		t.Errorf("factory = %v", factory)
	}

	// Factory decode only surfaces when fault bit 25 is set.
	bctl := blank(blockCtlCount)
	set(bctl, blockCtlStart, 41758, 0)
	set(bctl, blockCtlStart, 41759, 1<<12) // DC under-voltage, no bit 25
	set(bctl, blockCtlStart, 41760, 1) // factory bit 16 lives in the hi word
	set(bctl, blockCtlStart, 41761, 0)
	s := Decode(blank(block701Count), blank(block714Count), bctl)
	if len(s.Factory) != 0 {
		t.Errorf("factory should be empty without fault bit 25, got %v", s.Factory)
	}
	if s.State == 8 {
		t.Error("blank state must not decode as online")
	}
}

func TestNaNHandling(t *testing.T) {
	b701 := blank(block701Count)
	set(b701, block701Start, 40102, 0xFFFF) // u32 NaN = 0xFFFFFFFF
	set(b701, block701Start, 40103, 0xFFFF)
	set(b701, block701Start, 40101, 0xFFFF) // u16 NaN
	b714 := blank(block714Count)
	set(b714, block714Start, 41721, 0xFFFF) // DC voltage u16 NaN
	s := Decode(b701, b714, blank(blockCtlCount))
	if s.ACPowerW.OK || s.DCVoltage.OK || s.FreqHz.OK || s.VLn.OK {
		t.Errorf("NaN registers must decode OK=false: %+v %+v %+v %+v", s.ACPowerW, s.DCVoltage, s.FreqHz, s.VLn)
	}
	// DC current NaN in port regs AND NaN in the fixed-scale total -> not OK.
	if s.DCCurrent.OK {
		t.Errorf("DC current should be NaN, got %+v", s.DCCurrent)
	}
}

func TestStateText(t *testing.T) {
	if StateText(12) != "Online (grid-form)" || !StateOnline(12) {
		t.Error("state 12 wrong")
	}
	if StateText(1) != "Fault" || StateOnline(1) {
		t.Error("state 1 wrong")
	}
	if StateText(99) != "State 99" {
		t.Errorf("unknown state renders %q", StateText(99))
	}
}
