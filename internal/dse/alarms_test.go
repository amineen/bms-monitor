package dse

import "testing"

func TestDecodeAlarmsNibbles(t *testing.T) {
	// 8 alarms in 2 registers. Alarm 1 = top nibble of reg 0.
	// Alarm 2 (Low oil pressure) = Shutdown(3); alarm 6 (Over speed) = Warning(2);
	// alarm 8 = Electrical trip(4); everything else not active (1).
	regs := []uint16{
		0x1311, // alarms 1..4: 1,3,1,1
		0x1214, // alarms 5..8: 1,2,1,4
	}
	alarms := DecodeAlarms(8, regs)
	if len(alarms) != 3 {
		t.Fatalf("active alarms = %d want 3: %+v", len(alarms), alarms)
	}
	if alarms[0].Index != 2 || alarms[0].Name != "Low oil pressure" || alarms[0].State != AlarmShutdown {
		t.Errorf("alarm[0] = %+v", alarms[0])
	}
	if alarms[1].Index != 6 || alarms[1].Name != "Over speed" || alarms[1].State != AlarmWarning {
		t.Errorf("alarm[1] = %+v", alarms[1])
	}
	if alarms[2].Index != 8 || alarms[2].State != AlarmTrip {
		t.Errorf("alarm[2] = %+v", alarms[2])
	}
	if !alarms[0].Severe() || alarms[1].Severe() || !alarms[2].Severe() {
		t.Error("severity flags wrong")
	}
}

func TestDecodeAlarmsQuietAndSentinels(t *testing.T) {
	// All not-active / unimplemented -> nothing.
	if got := DecodeAlarms(8, []uint16{0x1111, 0x11FF}); len(got) != 0 {
		t.Errorf("quiet decode = %+v want none", got)
	}
	// Count sentinel / zero / absurd count -> nothing, no panic.
	if got := DecodeAlarms(0xFFFF, []uint16{0x3333}); len(got) != 0 {
		t.Errorf("sentinel count decoded %+v", got)
	}
	if got := DecodeAlarms(0, nil); len(got) != 0 {
		t.Errorf("zero count decoded %+v", got)
	}
	// Count larger than supplied regs -> stops at data end.
	if got := DecodeAlarms(12, []uint16{0x1113}); len(got) != 1 || got[0].Index != 4 {
		t.Errorf("truncated decode = %+v", got)
	}
}

func TestAlarmNameFallback(t *testing.T) {
	if alarmName(2) != "Low oil pressure" {
		t.Errorf("name(2) = %q", alarmName(2))
	}
	if alarmName(150) != "Alarm 150" {
		t.Errorf("fallback = %q", alarmName(150))
	}
	// Active indication (9) surfaces as an indication, not a fault.
	got := DecodeAlarms(1, []uint16{0x9000})
	if len(got) != 1 || got[0].State != AlarmIndicate || got[0].Severe() {
		t.Errorf("indication decode = %+v", got)
	}
	if s := AlarmStrings(got); len(s) != 1 || s[0] != "Emergency stop (Indication)" {
		t.Errorf("alarm strings = %v", s)
	}
}
