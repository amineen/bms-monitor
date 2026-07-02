package bms

import "testing"

func TestSwitchDecodeAndRelay(t *testing.T) {
	// Bit0 discharge + Bit1 charge closed, plus fan (bit6) -> relay closed.
	sw := uint16(0b0100_0011)
	flags := SwitchFlags(sw)
	if len(flags) != 3 {
		t.Fatalf("flags = %v want 3", flags)
	}
	if !RelayClosed(sw) {
		t.Errorf("RelayClosed should be true for %b", sw)
	}
	// Only pre-charge (bit2) and buzzer (bit3) -> NOT closed (no main relay).
	if RelayClosed(0b0000_1100) {
		t.Errorf("pre-charge/buzzer alone should not count as relay closed")
	}
}

func TestSysOpText(t *testing.T) {
	cases := map[uint16]string{0x11: "Standby", 0x22: "Run", 0x00: "Unknown", 250: "Unknown"}
	for v, want := range cases {
		if got := SysOpText(v); got != want {
			t.Errorf("SysOpText(0x%02X) = %q want %q", v, got, want)
		}
	}
}

func TestBuildCombiner(t *testing.T) {
	// Standby, no relays -> not live.
	c := buildCombiner(SysOpStandby, 0, 744.5)
	if c.Live || c.State != "Standby" || c.BusV != 0 {
		t.Errorf("standby: %+v", c)
	}
	// Run + relays closed -> live, bus voltage carried.
	c = buildCombiner(SysOpRun, 0b11, 744.5)
	if !c.Live || c.State != "Run" || c.BusV != 744.5 {
		t.Errorf("run: %+v", c)
	}
	// Unknown status but relay physically closed -> still live.
	c = buildCombiner(250, 0b01, 744.4)
	if !c.Live || !c.RelayClosed {
		t.Errorf("relay-closed fallback: %+v", c)
	}
}

// helper to build a snapshot of N online strings at the given voltages.
func snapWith(volts []float64, alarms map[int][]string, state string) *SystemSnapshot {
	s := &SystemSnapshot{Combiner: CombinerStatus{OK: true, State: state}}
	for i, v := range volts {
		si := StringInfo{Index: i + 1, Status: StatusEnumerated, TotalV: v, Alarms: []string{}}
		if a, ok := alarms[i+1]; ok {
			si.Alarms = a
		}
		s.Strings = append(s.Strings, si)
	}
	return s
}

func TestRunGate(t *testing.T) {
	// Happy path: 6 online, matched, standby, no alarms -> OK.
	g := EvaluateRunGate(snapWith([]float64{744.5, 744.5, 744.4, 744.5, 744.4, 744.5}, nil, "Standby"), 5)
	if !g.OK || !g.HardOK {
		t.Fatalf("expected pass gate, got %+v", g)
	}

	// Only 5 online -> hard fail (cannot force).
	g = EvaluateRunGate(snapWith([]float64{744, 744, 744, 744, 744}, nil, "Standby"), 5)
	if g.HardOK {
		t.Errorf("5 online should fail HARD gate")
	}

	// Active protection -> hard fail.
	g = EvaluateRunGate(snapWith([]float64{744, 744, 744, 744, 744, 744}, map[int][]string{3: {"Pile over-voltage"}}, "Standby"), 5)
	if g.HardOK {
		t.Errorf("active protection should fail HARD gate")
	}

	// Voltage spread too wide -> soft fail only (HardOK true, OK false).
	g = EvaluateRunGate(snapWith([]float64{744, 744, 744, 744, 744, 760}, nil, "Standby"), 5)
	if !g.HardOK || g.OK {
		t.Errorf("wide spread should be soft-fail only, got %+v", g)
	}
}
