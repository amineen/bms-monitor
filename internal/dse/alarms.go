package dse

import "fmt"

// GenComm Page 8 — "Alarm conditions" (SP-228 §10.6). Register 2048 holds the
// number of named alarms; the conditions follow from 2049 packed FOUR per
// register as 4-bit nibbles, first alarm in the MOST significant nibble.
//
// Nibble values: 0 disabled digital input · 1 not active · 2 WARNING ·
// 3 SHUTDOWN · 4 ELECTRICAL TRIP · 8 inactive indication · 9 active
// indication · 15 unimplemented.
const (
	Page8Base      = 8 * 256 // 2048 — number of named alarms
	alarmCountMax  = 200     // sanity bound on the count register
)

// Alarm severity states (only active states are surfaced).
const (
	AlarmWarning  = "Warning"
	AlarmShutdown = "Shutdown"
	AlarmTrip     = "Electrical trip"
	AlarmIndicate = "Indication"
)

// Alarm is one ACTIVE named alarm condition.
type Alarm struct {
	Index int    `json:"index"` // 1-based GenComm alarm number
	Name  string `json:"name"`
	State string `json:"state"`
}

// Severe reports whether the alarm stops the set (shutdown / electrical trip).
func (a Alarm) Severe() bool { return a.State == AlarmShutdown || a.State == AlarmTrip }

// alarmNames is the standard GenComm named-alarm order for the DSE 8xxx
// family (SP-228). The DSE8610 MKII follows this ordering; verify any alarm
// wording against the controller front panel on first live trip
// (README-plant.md field-validation list).
var alarmNames = []string{
	1:  "Emergency stop",
	2:  "Low oil pressure",
	3:  "High coolant temperature",
	4:  "High oil temperature",
	5:  "Under speed",
	6:  "Over speed",
	7:  "Fail to start",
	8:  "Fail to come to rest",
	9:  "Loss of speed sensing",
	10: "Generator low voltage",
	11: "Generator high voltage",
	12: "Generator low frequency",
	13: "Generator high frequency",
	14: "Generator low current",
	15: "Generator high current",
	16: "Generator earth fault",
	17: "Generator reverse power",
	18: "Air flap",
	19: "Oil pressure sender fault",
	20: "Coolant temperature sender fault",
	21: "Oil temperature sender fault",
	22: "Fuel level sender fault",
	23: "Magnetic pickup fault",
	24: "Loss of AC speed signal",
	25: "Charge alternator failure",
	26: "Low battery voltage",
	27: "High battery voltage",
	28: "Low fuel level",
	29: "High fuel level",
	30: "Generator failed to close",
	31: "Mains failed to close",
	32: "Generator instantaneous over current",
	33: "Mains instantaneous over current",
	34: "Generator over current",
	35: "Mains over current",
	36: "Failure to synchronise",
	37: "Bus live",
	38: "Scheduled run",
	39: "Bus not live",
	40: "Bus wrong phase rotation",
	41: "Priority selection error",
	42: "MSC data error",
	43: "MSC ID error",
	44: "MSC failure to sync",
	45: "Bus low voltage",
	46: "Bus high voltage",
	47: "Bus low frequency",
	48: "Bus high frequency",
	49: "MSC too few sets",
	50: "MSC alarms inhibited",
	51: "MSC old version units",
	52: "Mains reverse power",
	53: "Minimum sets not reached",
	54: "Insufficient capacity available",
	55: "Out of sync",
	56: "Alternative aux mains fail",
	57: "Loading frequency alarm",
	58: "Loading voltage alarm",
	59: "Fuel usage running",
	60: "Fuel usage stopped",
	61: "Protections disabled",
	62: "Protections blocked",
}

func alarmName(idx int) string {
	if idx > 0 && idx < len(alarmNames) && alarmNames[idx] != "" {
		return alarmNames[idx]
	}
	return fmt.Sprintf("Alarm %d", idx)
}

func alarmState(nibble uint16) string {
	switch nibble {
	case 2:
		return AlarmWarning
	case 3:
		return AlarmShutdown
	case 4:
		return AlarmTrip
	case 9:
		return AlarmIndicate
	default:
		return "" // disabled / not active / inactive indication / unimplemented
	}
}

// DecodeAlarms unpacks the page-8 block. count is the value of register 2048;
// regs are the packed condition registers from 2049 onward (4 alarms per
// register, first alarm in the top nibble). Only ACTIVE alarms are returned.
func DecodeAlarms(count uint16, regs []uint16) []Alarm {
	out := []Alarm{}
	if count == 0 || count == 0xFFFF || count > alarmCountMax {
		return out
	}
	for i := 0; i < int(count); i++ {
		reg := i / 4
		if reg >= len(regs) {
			break
		}
		shift := uint(12 - 4*(i%4)) // alarm 1 = bits 15..12
		nibble := (regs[reg] >> shift) & 0xF
		if state := alarmState(nibble); state != "" {
			out = append(out, Alarm{Index: i + 1, Name: alarmName(i + 1), State: state})
		}
	}
	return out
}

// AlarmStrings renders alarms as "Name (State)" lines for events/health.
func AlarmStrings(alarms []Alarm) []string {
	out := make([]string, len(alarms))
	for i, a := range alarms {
		out[i] = a.Name + " (" + a.State + ")"
	}
	return out
}
