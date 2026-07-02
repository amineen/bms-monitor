// Package bms is a read-only Modbus port of read_pylontech_full.py for the
// Pylontech Force-H3 (FC1000 / MBMS) high-voltage BMS, reached over an
// RS485->TCP converter. It NEVER writes a register.
//
// Register map: ModBus-Protocol-Pylon-high-voltage-V1.38
//
//	0x1000 = equipment info (identity / firmware)
//	0x1100 = aggregate "overall system" (section 3.4)
//	0x1400 = String 1 (section 3.6); each further string is +0x700.
package bms

// Register-map constants (offsets are relative to a string's base unless noted).
const (
	PileBase   = 0x1400 // String 1 base register
	PileStride = 0x700  // +0x700 per further string: base(n) = 0x1400 + (n-1)*0x700
	MaxRegs    = 120    // chunk size; Modbus FC03/04 hard limit is 125

	ModVOff  = 0x60  // module voltages (up to 75 modules)
	ModTOff  = 0xB0  // module temperatures (up to 75 modules)
	CellVOff = 0x100 // cell voltages (up to 450 cells)
	CellTOff = 0x400 // cell temperatures (up to 450 cells)
	SNOff    = 0x50  // serial number (16 ASCII words)
	SNLen    = 16

	IdentBase = 0x1000 // equipment identity / firmware (read 0x10 words)
	AggBase   = 0x1100 // aggregate system block, section 3.4 (read 0x52 words)
	HeartReg  = 0x113C // heartbeat (== aggregate offset 0x3C)

	MaxModules = 75
	MaxCells   = 450
)

// Control & status registers (§3.3 Remote Control, §3.4 system info).
//
// WRITING is used by exactly ONE feature — the gated commissioning "Run" flow
// in control.go. Writing 0x1094 closes the string relays and energizes the
// combiner bus, so it is guarded by hard preconditions + operator interlocks.
const (
	RegSleep = 0x1090 // write 0xAA = sleep, 0x55 = wake
	RegRun   = 0x1094 // write 0xAA = 'Run' — start paralleling into the combiner

	CmdEffective uint16 = 0xAA // "effective" for command registers
	CmdWake      uint16 = 0x55 // wake from sleep (0x1090)

	AggSwitchOff = 0x0F // aggregate offset of Switching value (reg 0x110F)
	AggSysOpOff  = 0x41 // aggregate offset of System operation status (reg 0x1141)

	SysOpStandby = 0x11 // self-inspection done, relay OPEN (special MBMS firmware)
	SysOpRun     = 0x22 // relay CLOSED -> combiner energized (special MBMS firmware)
)

// switchBits decodes the Switching value (0x110F / Appendix II). Bit0/Bit1 are
// the main power relays: either closed means the string is tied to the bus.
var switchBits = []bitLabel{
	{0, "Discharge circuit"}, {1, "Charge circuit"}, {2, "Pre-charge circuit"},
	{3, "Buzzer"}, {4, "Heating film"}, {5, "Current-limiting module"},
	{6, "Fan"}, {7, "Dry contact in 1"},
}

// SwitchFlags returns the labels of the switch bits that are ON.
func SwitchFlags(v uint16) []string { return DecodeFlags(v, switchBits) }

// RelayClosed reports whether a main power relay (charge or discharge circuit)
// is closed — i.e. the string is connected to the combiner bus.
func RelayClosed(sw uint16) bool { return sw&0x03 != 0 }

// SysOpText renders the System operation status register (special firmware).
func SysOpText(v uint16) string {
	switch int(v) {
	case SysOpStandby:
		return "Standby"
	case SysOpRun:
		return "Run"
	default:
		return "Unknown"
	}
}

const (
	SrcAgg = "ModBus-Protocol-Pylon-high-voltage-V1.38 section 3.4"
	SrcStr = "ModBus-Protocol-Pylon-high-voltage-V1.38 section 3.6"
)

// Display kinds for a SUMMARY field.
const (
	DispNum    = "num"
	DispHex    = "hex"
	DispBasic  = "status_basic"
	DispProt   = "status_prot"
)

// Field describes one SUMMARY register (per-string §3.6 layout; the aggregate
// §3.4 block shares the same offsets).
type Field struct {
	Off     int
	Name    string
	Scale   float64
	Unit    string
	Signed  bool
	Words   int // 1 or 2 (32-bit values are hi-word-first)
	Display string
}

func f(off int, name string, opts ...func(*Field)) Field {
	fld := Field{Off: off, Name: name, Scale: 1, Words: 1, Display: DispNum}
	for _, o := range opts {
		o(&fld)
	}
	return fld
}

func scale(s float64, unit string) func(*Field) {
	return func(x *Field) { x.Scale = s; x.Unit = unit }
}
func signed() func(*Field)     { return func(x *Field) { x.Signed = true } }
func words2() func(*Field)     { return func(x *Field) { x.Words = 2 } }
func disp(d string) func(*Field) { return func(x *Field) { x.Display = d } }

// Summary is the per-string register table (offsets relative to the string base).
// Gaps are intentional; 2-word entries consume Off and Off+1.
var Summary = []Field{
	f(0x00, "Basic status", disp(DispBasic)),
	f(0x01, "Protection status", disp(DispProt)),
	f(0x02, "Alarm status 1", disp(DispHex)),
	f(0x03, "Total voltage", scale(0.1, "V")),
	f(0x04, "Current", scale(0.01, "A"), signed(), words2()),
	f(0x06, "Temperature", scale(0.1, "degC"), signed()),
	f(0x07, "SOC", scale(1, "%")),
	f(0x08, "Cycle times"),
	f(0x09, "Pile max charge voltage", scale(0.1, "V")),
	f(0x0A, "Pile max charge current", scale(0.01, "A"), words2()),
	f(0x0C, "Pile min discharge voltage", scale(0.1, "V")),
	f(0x0D, "Pile max discharge current", scale(0.01, "A"), signed(), words2()),
	f(0x0F, "Switching value", disp(DispHex)),
	f(0x10, "Cell max voltage", scale(0.001, "V")),
	f(0x11, "Cell min voltage", scale(0.001, "V")),
	f(0x12, "Cell max voltage channel"),
	f(0x13, "Cell min voltage channel"),
	f(0x14, "Cell max temperature", scale(0.1, "degC"), signed()),
	f(0x15, "Cell min temperature", scale(0.1, "degC"), signed()),
	f(0x16, "Cell max temperature channel"),
	f(0x17, "Cell min temperature channel"),
	f(0x18, "Module max voltage", scale(0.01, "V")),
	f(0x19, "Module min voltage", scale(0.01, "V")),
	f(0x1A, "Module max voltage channel"),
	f(0x1B, "Module min voltage channel"),
	f(0x1C, "Module max temperature", scale(0.1, "degC"), signed()),
	f(0x1D, "Module min temperature", scale(0.1, "degC"), signed()),
	f(0x1E, "Module max temperature channel"),
	f(0x1F, "Module min temperature channel"),
	f(0x20, "SOH", scale(1, "%")),
	f(0x21, "Remain capacity", scale(1, "Wh"), words2()),
	f(0x23, "Charge capacity", scale(1, "Wh"), words2()),
	f(0x25, "Discharge capacity", scale(1, "Wh"), words2()),
	f(0x27, "Daily charge capacity", scale(1, "Wh"), words2()),
	f(0x29, "Daily discharge capacity", scale(1, "Wh"), words2()),
	f(0x2B, "History charge capacity", scale(1, "kWh"), words2()),
	f(0x2D, "History discharge capacity", scale(1, "kWh"), words2()),
	f(0x2F, "Force charge request"),
	f(0x30, "Balance charge request"),
	f(0x32, "Error code 1", words2(), disp(DispHex)),
	f(0x34, "Error code 2", words2(), disp(DispHex)),
	f(0x36, "Modules in series"),
	f(0x37, "Cells in series"),
	f(0x38, "Charge forbidden"),
	f(0x39, "Discharge forbidden"),
	f(0x3A, "Pile voltage spec", scale(0.1, "V")),
	f(0x3B, "Nominal capacity", scale(1, "Ah")),
	f(0x3C, "BMS B+ terminal temp", scale(0.1, "degC"), signed()),
	f(0x3D, "BMS B- terminal temp", scale(0.1, "degC"), signed()),
	f(0x3E, "BMS D+ terminal temp", scale(0.1, "degC"), signed()),
	f(0x3F, "BMS D- terminal temp", scale(0.1, "degC"), signed()),
	f(0x40, "Terminal max temp", scale(0.1, "degC"), signed()),
	f(0x41, "Terminal min temp", scale(0.1, "degC"), signed()),
	f(0x44, "Module PCB max temp", scale(0.1, "degC"), signed()),
	f(0x45, "Module PCB min temp", scale(0.1, "degC"), signed()),
	f(0x48, "SOE", scale(1, "%")),
	f(0x49, "Alarm status 2", disp(DispHex)),
	f(0x4A, "Max continuous charge power", scale(0.1, "kW")),
	f(0x4B, "Max continuous discharge power", scale(0.1, "kW")),
}

// ---- numeric helpers ----

// S16 interprets an unsigned 16-bit register as signed.
func S16(v uint16) int {
	if v >= 0x8000 {
		return int(v) - 0x10000
	}
	return int(v)
}

// U32 combines two registers (hi-word-first) into an unsigned 32-bit value.
func U32(hi, lo uint16) uint32 { return uint32(hi)<<16 | uint32(lo) }

// S32 combines two registers (hi-word-first) into a signed 32-bit value.
func S32(hi, lo uint16) int64 {
	v := int64(hi)<<16 | int64(lo)
	if v >= 0x80000000 {
		return v - 0x100000000
	}
	return v
}

// ---- bitfield decode tables (insertion order preserved) ----

var basicState = map[int]string{0: "Sleep", 1: "Charge", 2: "Discharge", 3: "Idle"}

type bitLabel struct {
	Bit   int
	Label string
}

var basicBits = []bitLabel{
	{3, "System error protection"}, {4, "Current protection"},
	{5, "Voltage protection"}, {6, "Temperature protection"},
	{7, "Voltage alarm"}, {8, "Current alarm"}, {9, "Temperature alarm"},
	{10, "Idle"}, {11, "Charging"}, {12, "Discharging"},
	{13, "Sleep"}, {14, "Fan warning"},
}

var protBits = []bitLabel{
	{2, "Pile under-voltage"}, {3, "Pile over-voltage"},
	{4, "Charge under-temp"}, {5, "Charge over-temp"},
	{6, "Discharge under-temp"}, {7, "Discharge over-temp"},
	{8, "Charge over-current"}, {9, "Discharge over-current"},
	{10, "Short circuit"}, {11, "Power-terminal over-temp"},
	{12, "Module over-temp"}, {13, "Module under-voltage"},
	{14, "Module over-voltage"}, {15, "Cell 2nd under-voltage"},
}

// DecodeBasic renders the Basic status register (0x00) to plain text.
func DecodeBasic(v uint16) string {
	state, ok := basicState[int(v&0x07)]
	if !ok {
		state = "state" + itoa(int(v&0x07))
	}
	out := state
	for _, b := range basicBits {
		if v&(1<<uint(b.Bit)) != 0 {
			out += ", " + b.Label
		}
	}
	return out
}

// DecodeProt renders the Protection status register (0x01) to plain text.
func DecodeProt(v uint16) string { return decodeBits(v, protBits) }

// DecodeFlags returns the individual set flag labels (for alarm/insight lists).
func DecodeFlags(v uint16, table []bitLabel) []string {
	var out []string
	for _, b := range table {
		if v&(1<<uint(b.Bit)) != 0 {
			out = append(out, b.Label)
		}
	}
	return out
}

// ProtFlags / BasicFlags expose the set flags for the health engine.
func ProtFlags(v uint16) []string  { return DecodeFlags(v, protBits) }
func BasicFlags(v uint16) []string { return DecodeFlags(v, basicBits) }

func decodeBits(v uint16, table []bitLabel) string {
	flags := DecodeFlags(v, table)
	if len(flags) == 0 {
		return "OK (none)"
	}
	out := flags[0]
	for _, fl := range flags[1:] {
		out += ", " + fl
	}
	return out
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
