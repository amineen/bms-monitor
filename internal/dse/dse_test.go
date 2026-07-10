package dse

import "testing"

// buildPage returns a block pre-filled with the U16 unimplemented sentinel.
func buildPage(n int) []uint16 {
	b := make([]uint16, n)
	for i := range b {
		b[i] = 0xFFFF
	}
	return b
}

func TestDecodePage4(t *testing.T) {
	p4 := buildPage(Page4Count)
	p4[0] = 462          // oil pressure 462 kPa
	p4[1] = 83           // coolant 83 C
	p4[3] = 76           // fuel 76 %
	p4[4] = 279          // charge alt 27.9 V
	p4[5] = 247          // engine battery 24.7 V
	p4[6] = 1500         // 1500 RPM
	p4[7] = 500          // 50.0 Hz
	p4[8], p4[9] = 0, 2400   // L1-N 240.0 V (U32)
	p4[20], p4[21] = 0, 1253 // L1 current 125.3 A (U32 x0.1)
	p4[28], p4[29] = 0, 28500 // L1 watts 28.5 kW (S32)

	s := Decode(p4, buildPage(Page6Count), buildPage(Page7Count))

	if !s.OilPressureKPa.OK || s.OilPressureKPa.Value != 462 {
		t.Errorf("oil pressure = %+v want 462", s.OilPressureKPa)
	}
	if !s.CoolantTempC.OK || s.CoolantTempC.Value != 83 {
		t.Errorf("coolant = %+v want 83", s.CoolantTempC)
	}
	if !s.FreqHz.OK || s.FreqHz.Value != 50.0 {
		t.Errorf("freq = %+v want 50.0", s.FreqHz)
	}
	if !s.VLn[0].OK || s.VLn[0].Value != 240.0 {
		t.Errorf("L1-N = %+v want 240.0", s.VLn[0])
	}
	if !s.AmpsL[0].OK || s.AmpsL[0].Value != 125.3 {
		t.Errorf("L1 A = %+v want 125.3", s.AmpsL[0])
	}
	if !s.WattsL[0].OK || s.WattsL[0].Value != 28500 {
		t.Errorf("L1 W = %+v want 28500", s.WattsL[0])
	}
	if !s.Running {
		t.Error("1500 RPM should report Running")
	}
	// Charge alternator ×0.1.
	if s.ChargeAltV.Value != 27.9 {
		t.Errorf("charge alt = %v want 27.9", s.ChargeAltV.Value)
	}
}

func TestDecodePage6And7(t *testing.T) {
	p6 := buildPage(Page6Count)
	p6[0], p6[1] = 1, 20000 // total W = 0x10000+20000 = 85536
	p6[21] = 92             // PF 0.92
	p7 := buildPage(Page7Count)
	p7[8], p7[9] = 0, 12345  // +kWh 1234.5
	p7[16], p7[17] = 0, 87   // 87 starts
	p7[6], p7[7] = 0, 7200   // 7200 s run = 2 h

	s := Decode(buildPage(Page4Count), p6, p7)

	if !s.TotalW.OK || s.TotalW.Value != 85536 {
		t.Errorf("total W = %+v want 85536", s.TotalW)
	}
	if !s.AvgPF.OK || s.AvgPF.Value != 0.92 {
		t.Errorf("PF = %+v want 0.92", s.AvgPF)
	}
	if !s.PosKWh.OK || s.PosKWh.Value != 1234.5 {
		t.Errorf("kWh = %+v want 1234.5", s.PosKWh)
	}
	if !s.Starts.OK || s.Starts.Value != 87 {
		t.Errorf("starts = %+v want 87", s.Starts)
	}
	if !s.RunHours.OK || s.RunHours.Value != 2 {
		t.Errorf("run hours = %+v want 2", s.RunHours)
	}
	if s.Running {
		t.Error("unimplemented RPM must not report Running")
	}
}

func TestSentinels(t *testing.T) {
	s := Decode(buildPage(Page4Count), buildPage(Page6Count), buildPage(Page7Count))
	if s.OilPressureKPa.OK || s.CoolantTempC.OK || s.TotalW.OK || s.PosKWh.OK {
		t.Errorf("unimplemented sentinels must decode to OK=false: %+v", s)
	}
	// Signed 16-bit sentinel 0x7FFF.
	p4 := buildPage(Page4Count)
	p4[1] = 0x7FFF
	if got := Decode(p4, buildPage(Page6Count), buildPage(Page7Count)); got.CoolantTempC.OK {
		t.Error("S16 0x7FFF sentinel must decode to OK=false")
	}
}

func TestNegativeCoolant(t *testing.T) {
	p4 := buildPage(Page4Count)
	p4[1] = 0xFFF6 // S16 -10 — but 0xFFF6 != 0xFFFF so it must decode
	s := Decode(p4, buildPage(Page6Count), buildPage(Page7Count))
	if !s.CoolantTempC.OK || s.CoolantTempC.Value != -10 {
		t.Errorf("coolant = %+v want -10", s.CoolantTempC)
	}
}

func TestPageBases(t *testing.T) {
	if Page4Base != 1024 || Page6Base != 1536 || Page7Base != 1792 {
		t.Errorf("GenComm page bases wrong: %d %d %d", Page4Base, Page6Base, Page7Base)
	}
}
