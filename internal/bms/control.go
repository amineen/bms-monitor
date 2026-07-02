package bms

// control.go holds the ONLY register writes in this application: the gated
// commissioning "Run" flow that closes the string relays and energizes the
// combiner bus (write 0x1094 = 0xAA). Everything else in the app is read-only.
//
// Writing Run puts ~744 V on the combiner. In a normal system the upper
// controller (ARC / inverter-PCS) issues this in coordination with the inverter
// DC-link pre-charge. This flow exists for on-site commissioning when NO such
// controller is present yet. It is guarded by:
//   - HARD gates that can never be overridden (all strings online, no active
//     protections),
//   - SOFT gates that require an explicit Force (voltage match, Standby state),
//   - and, in the UI, operator interlock check-boxes affirming the downstream
//     inverters are isolated or ready and the bus is clear.

import (
	"fmt"
	"strings"
	"time"

	"github.com/goburrow/modbus"
)

// writeSingleRegisterFresh opens a fresh TCP connection and writes one holding
// register via FC06, then closes it (same connection discipline as reads).
func writeSingleRegisterFresh(cfg Config, addr, value uint16) error {
	handler := modbus.NewTCPClientHandler(cfg.addr())
	handler.Timeout = cfg.timeout()
	handler.SlaveId = byte(unitOr1(cfg.Unit))
	if err := handler.Connect(); err != nil {
		return err
	}
	defer handler.Close()
	client := modbus.NewClient(handler)
	_, err := client.WriteSingleRegister(addr, value)
	return err
}

// GateCheck is one precondition for issuing Run.
type GateCheck struct {
	Label  string `json:"label"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
	Hard   bool   `json:"hard"` // hard gates can never be Force-overridden
}

// RunGate is the full precondition evaluation.
type RunGate struct {
	OK      bool        `json:"ok"`     // every gate passes
	HardOK  bool        `json:"hardOk"` // every HARD gate passes (required even with Force)
	Checks  []GateCheck `json:"checks"`
	Reasons []string    `json:"reasons"` // labels of failing gates
}

const defaultMaxSpreadV = 5.0

// EvaluateRunGate checks the Run preconditions against a snapshot (pure; no I/O).
func EvaluateRunGate(snap *SystemSnapshot, maxSpreadV float64) RunGate {
	if maxSpreadV <= 0 {
		maxSpreadV = defaultMaxSpreadV
	}

	online := 0
	var vmin, vmax float64
	first := true
	var protStrings []string
	for _, s := range snap.Strings {
		if s.Status != StatusEnumerated {
			continue
		}
		online++
		if first || s.TotalV < vmin {
			vmin = s.TotalV
		}
		if first || s.TotalV > vmax {
			vmax = s.TotalV
		}
		first = false
		if len(s.Alarms) > 0 {
			protStrings = append(protStrings, fmt.Sprintf("S%d(%s)", s.Index, strings.Join(s.Alarms, ",")))
		}
	}
	spread := 0.0
	if !first {
		spread = round(vmax-vmin, 1)
	}

	protDetail := "none"
	if len(protStrings) > 0 {
		protDetail = strings.Join(protStrings, "; ")
	}

	checks := []GateCheck{
		{
			Label: "All 6 strings online", Hard: true, Pass: online == 6,
			Detail: fmt.Sprintf("%d of 6 enumerated", online),
		},
		{
			Label: "No active protections", Hard: true, Pass: len(protStrings) == 0,
			Detail: protDetail,
		},
		{
			Label: "String voltages matched", Hard: false, Pass: !first && spread <= maxSpreadV,
			Detail: fmt.Sprintf("spread %.1f V (limit %.1f V)", spread, maxSpreadV),
		},
		{
			Label: "System in Standby (relays open)", Hard: false, Pass: snap.Combiner.State == "Standby",
			Detail: fmt.Sprintf("status %s%s", snap.Combiner.State, relaySuffix(snap.Combiner)),
		},
	}

	g := RunGate{Checks: checks, OK: true, HardOK: true}
	for _, c := range checks {
		if !c.Pass {
			g.OK = false
			g.Reasons = append(g.Reasons, c.Label)
			if c.Hard {
				g.HardOK = false
			}
		}
	}
	if g.Reasons == nil {
		g.Reasons = []string{}
	}
	return g
}

func relaySuffix(c CombinerStatus) string {
	if len(c.Relays) == 0 {
		return ""
	}
	return " (" + strings.Join(c.Relays, ", ") + ")"
}

// RunOptions parameterizes IssueRun.
type RunOptions struct {
	MaxSpreadV float64 `json:"maxSpreadV"`
	AutoWake   bool    `json:"autoWake"`
	Force      bool    `json:"force"` // bypass SOFT gates only; HARD gates always apply
}

// RunResult reports the outcome of an IssueRun attempt.
type RunResult struct {
	Written     bool    `json:"written"`
	Woke        bool    `json:"woke"`
	StateBefore string  `json:"stateBefore"`
	StateAfter  string  `json:"stateAfter"`
	Live        bool    `json:"live"`
	BusV        float64 `json:"busV"`
	Gate        RunGate `json:"gate"`
	Message     string  `json:"message"`
}

// IssueRun runs the gated commissioning Run: fresh read -> verify gates ->
// optional wake -> write 0x1094=0xAA -> poll status. This is the ONLY register
// write in the app. It energizes the combiner bus; callers must have confirmed
// the downstream inverters are isolated or ready.
func IssueRun(cfg Config, opts RunOptions) (*RunResult, error) {
	snap, err := ReadSystem(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not read the BMS before Run: %w", err)
	}
	gate := EvaluateRunGate(snap, opts.MaxSpreadV)
	res := &RunResult{Gate: gate, StateBefore: snap.Combiner.State}

	// Hard gates are absolute.
	if !gate.HardOK {
		return res, fmt.Errorf("Run blocked: %s", strings.Join(gate.Reasons, "; "))
	}
	// Soft gates require an explicit override.
	if !gate.OK && !opts.Force {
		return res, fmt.Errorf("Run blocked (override required): %s", strings.Join(gate.Reasons, "; "))
	}

	// Already live? Don't re-issue.
	if snap.Combiner.Live {
		res.Live = true
		res.StateAfter = snap.Combiner.State
		res.BusV = snap.Aggregate.TotalV
		res.Message = "System already running — combiner is live."
		return res, nil
	}

	// Wake first if any string is asleep (protocol note on 0x1091/0x1094).
	if opts.AutoWake && systemSleeping(snap) {
		if err := writeSingleRegisterFresh(cfg, RegSleep, CmdWake); err != nil {
			return res, fmt.Errorf("wake command (0x1090) failed: %w", err)
		}
		res.Woke = true
		time.Sleep(1500 * time.Millisecond)
	}

	// Fire the Run command.
	if err := writeSingleRegisterFresh(cfg, RegRun, CmdEffective); err != nil {
		return res, fmt.Errorf("Run command (0x1094) failed: %w", err)
	}
	res.Written = true

	// Poll for the relays to close / status to reach Run (~6 s).
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(1 * time.Second)
		agg, err := readBaseFresh(cfg, AggBase, 0x52)
		if err != nil {
			continue
		}
		c := buildCombiner(at(agg, AggSysOpOff), at(agg, AggSwitchOff), round(float64(at(agg, 0x03))*0.1, 1))
		res.StateAfter = c.State
		if c.Live {
			res.Live = true
			res.BusV = c.BusV
			break
		}
	}
	if res.Live {
		res.Message = "Run issued — combiner energized."
	} else {
		res.Message = "Run issued, but the system has not reported Run/relay-closed yet. Verify the BMS and downstream before assuming the bus is live."
	}
	return res, nil
}

// systemSleeping reports whether any online string is in the Sleep basic state.
func systemSleeping(snap *SystemSnapshot) bool {
	for _, s := range snap.Strings {
		if s.Status == StatusEnumerated && strings.HasPrefix(s.BasicStatus, "Sleep") {
			return true
		}
	}
	return false
}
