package plant

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/dse"
	"bms-monitor/internal/modbustcp"
	"bms-monitor/internal/oztek"
	"bms-monitor/internal/sma"
)

// Health levels (shared vocabulary with internal/bms).
const (
	HealthGood     = "good"
	HealthWarn     = "warn"
	HealthCritical = "critical"
	HealthOffline  = "offline"
)

// staleWindow: after a failed read, the previous good reading is shown as
// "stale" for this long before the device is reported offline. On the shared
// .41 OzTek bus an occasional collision with ARC's poll is expected — a single
// miss must not flip a healthy inverter to offline.
const staleWindow = 90 * time.Second

// Reading is the per-device result inside a Snapshot. Exactly one of the
// kind-specific pointers is set when data is available.
type Reading struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Unit    int    `json:"unit"`
	Enabled bool   `json:"enabled"`
	Notes   string `json:"notes"`

	Online bool   `json:"online"`
	Stale  bool   `json:"stale"` // last read missed; showing previous data
	Err    string `json:"err"`

	// PollSkipped: this device is on an ARC-shared bus and polling is off, so
	// the app deliberately did NOT read it (no Modbus issued). It is neither
	// online nor a fault — just not being read, to stay off ARC's bus.
	PollSkipped bool `json:"pollSkipped"`

	Health   string `json:"health"`
	Headline string `json:"headline"`

	Bms   *bms.AggSummary `json:"bms,omitempty"`
	Oztek *oztek.Snapshot `json:"oztek,omitempty"`
	Sma   *sma.Snapshot   `json:"sma,omitempty"`
	Dse   *dse.Snapshot   `json:"dse,omitempty"`

	ReadMillis int64 `json:"readMillis"`
}

// PowerBalance is the derived MDP view (no PCC meter on site — the inverter
// and genset AC readings are the reference points; signs validate on site).
type PowerBalance struct {
	PVKW     float64 `json:"pvKW"`     // sum of SMA AC output
	BessKW   float64 `json:"bessKW"`   // sum of OzTek AC (sign as read)
	GensetKW float64 `json:"gensetKW"` // DSE total watts
	LoadKW   float64 `json:"loadKW"`   // derived estimate: PV + genset + BESS
	Derived  bool    `json:"derived"`  // always true — no revenue meter in use
}

// PlantHealth is the roll-up banner for the whole plant.
type PlantHealth struct {
	Level    string   `json:"level"`
	Headline string   `json:"headline"`
	Reasons  []string `json:"reasons"`
}

// Snapshot is one full plant poll.
type Snapshot struct {
	Timestamp  string       `json:"timestamp"`
	Devices    []Reading    `json:"devices"`
	Power      PowerBalance `json:"power"`
	Health     PlantHealth  `json:"health"`
	Insights   []string     `json:"insights"`
	Events     []Event      `json:"events"` // recent timeline, newest first
	ReadMillis int64        `json:"readMillis"`
}

func endpoint(d Device) modbustcp.Endpoint {
	return modbustcp.Endpoint{Host: d.Host, Port: d.Port, Unit: d.Unit, TimeoutSec: 3}
}

// readDevice performs one device poll. A fast TCP pre-check keeps an
// unplugged device from costing a full Modbus timeout per block.
func readDevice(d Device) Reading {
	r := Reading{ID: d.ID, Name: d.Name, Kind: d.Kind, Host: d.Host, Port: d.Port,
		Unit: d.Unit, Enabled: d.Enabled, Notes: d.Notes}
	start := time.Now()
	ep := endpoint(d)

	if !modbustcp.Reachable(ep, 800*time.Millisecond) {
		r.Err = "no TCP connection"
		r.Health = HealthOffline
		r.Headline = "Unreachable"
		return r
	}

	var err error
	switch d.Kind {
	case KindBMS:
		var s *bms.AggSummary
		s, err = bms.ReadSummary(bms.Config{IP: d.Host, Port: d.Port, Unit: d.Unit, Timeout: 3})
		r.Bms = s
	case KindOzTek:
		r.Oztek, err = oztek.Read(ep)
	case KindSMA:
		r.Sma, err = sma.Read(ep)
	case KindDSE:
		r.Dse, err = dse.Read(ep)
	default:
		r.Err = "unknown device kind " + d.Kind
		r.Health = HealthOffline
		r.Headline = "Misconfigured"
		return r
	}
	r.ReadMillis = time.Since(start).Milliseconds()
	if err != nil {
		r.Err = err.Error()
		r.Health = HealthOffline
		r.Headline = "Read failed"
		return r
	}
	r.Online = true
	assessReading(&r)
	return r
}

// assessReading derives the per-device health dot + one-line headline.
func assessReading(r *Reading) {
	r.Health = HealthGood
	switch {
	case r.Bms != nil:
		b := r.Bms
		r.Headline = fmt.Sprintf("%d/6 strings · %.1f V · SOC %d%%", b.Piles, b.TotalV, b.SOC)
		if b.Combiner.Live {
			r.Headline += " · bus LIVE"
		}
		if b.Piles < 6 {
			r.Health = HealthWarn
		}
		if b.Piles == 0 {
			r.Health = HealthCritical
			r.Headline = "No strings online"
		}
	case r.Oztek != nil:
		o := r.Oztek
		r.Headline = o.StateText
		if o.ACPowerW.OK && o.Online {
			r.Headline += fmt.Sprintf(" · %.1f kW", o.ACPowerW.Value/1000)
		}
		if len(o.Warnings) > 0 {
			r.Health = HealthWarn
			r.Headline = o.StateText + " · " + o.Warnings[0]
		}
		if len(o.Faults) > 0 || o.State == 1 {
			r.Health = HealthCritical
			if len(o.Faults) > 0 {
				r.Headline = "FAULT: " + o.Faults[0]
			} else {
				r.Headline = "FAULT"
			}
		}
	case r.Sma != nil:
		s := r.Sma
		switch s.Condition {
		case "Fault":
			r.Health = HealthCritical
			r.Headline = "Fault · grid relay " + strings.ToLower(s.GridRelay)
		case "Warning":
			r.Health = HealthWarn
			r.Headline = "Warning · " + fmt.Sprintf("%.1f kW", s.ACPowerW.Value/1000)
		case "Off":
			r.Headline = "Off (no sun / shutdown)"
		default:
			if s.ACPowerW.OK {
				r.Headline = fmt.Sprintf("%.1f kW", s.ACPowerW.Value/1000)
			} else {
				r.Headline = s.Condition
			}
			if s.Derating != "" {
				r.Health = HealthWarn
				r.Headline += " · derating: " + s.Derating
			}
		}
	case r.Dse != nil:
		g := r.Dse
		if len(g.Alarms) > 0 {
			a := g.Alarms[0]
			if g.HasSevereAlarm() {
				r.Health = HealthCritical
				for _, x := range g.Alarms {
					if x.Severe() {
						a = x
						break
					}
				}
			} else {
				r.Health = HealthWarn
			}
			r.Headline = a.State + ": " + a.Name
			if n := len(g.Alarms); n > 1 {
				r.Headline += fmt.Sprintf(" (+%d more)", n-1)
			}
			return
		}
		if g.Running {
			r.Headline = fmt.Sprintf("Running · %.0f RPM · %.1f kW", g.EngineRPM.Value, g.TotalW.Value/1000)
			if g.OilPressureKPa.OK && g.OilPressureKPa.Value < 100 {
				r.Health = HealthCritical
				r.Headline = "Running · LOW OIL PRESSURE"
			} else if g.CoolantTempC.OK && g.CoolantTempC.Value > 98 {
				r.Health = HealthWarn
				r.Headline = fmt.Sprintf("Running · coolant %.0f °C", g.CoolantTempC.Value)
			}
		} else {
			r.Headline = "Stopped / standby"
			if g.BatteryV.OK && g.BatteryV.Value > 0 && g.BatteryV.Value < 11.5 {
				r.Health = HealthWarn
				r.Headline = fmt.Sprintf("Stopped · start battery %.1f V", g.BatteryV.Value)
			}
		}
	}
}

// Monitor owns poll state across refreshes: last-good readings for stale
// handling, the previous snapshot for event diffing, and the event ring.
type Monitor struct {
	mu       sync.Mutex
	lastGood map[string]Reading
	lastAt   map[string]time.Time
	prev     *Snapshot
	prevBatt *bms.SystemSnapshot // last DETAILED battery read (alarm diffing)
	events   []Event
	sink     func([]Event) // optional: receives each fresh diff exactly once
}

// NewMonitor returns an empty poll-state holder.
func NewMonitor() *Monitor {
	return &Monitor{lastGood: map[string]Reading{}, lastAt: map[string]time.Time{}}
}

// SetEventSink registers a callback invoked with each batch of NEW timeline
// events as they are detected (used to persist the timeline to the datastore).
func (m *Monitor) SetEventSink(fn func([]Event)) {
	m.mu.Lock()
	m.sink = fn
	m.mu.Unlock()
}

// Read performs one sequential round-robin over the enabled devices and
// returns the assembled Snapshot. Reads are strictly one at a time; the
// modbustcp per-host mutex additionally guarantees one-in-flight per gateway.
//
// pollSharedBus gates the OzTek RS485 bus (the one ARC also masters): when
// false (the default), devices flagged SharedBus are NOT read at all — the app
// issues zero Modbus to that bus and so can never contend with ARC's control
// of the inverters. Disabled devices appear greyed out; shared-bus devices
// appear as "polling off" without any bus access.
func (m *Monitor) Read(devices []Device, pollSharedBus bool) *Snapshot {
	start := time.Now()
	snap := &Snapshot{Timestamp: time.Now().Format(time.RFC3339)}

	for _, d := range devices {
		if !d.Enabled {
			snap.Devices = append(snap.Devices, Reading{
				ID: d.ID, Name: d.Name, Kind: d.Kind, Host: d.Host, Port: d.Port,
				Unit: d.Unit, Enabled: false, Notes: d.Notes,
				Health: HealthOffline, Headline: "Disabled",
			})
			continue
		}
		if d.SharedBus && !pollSharedBus {
			// Do NOT touch ARC's shared bus. No reachability probe, no Modbus.
			snap.Devices = append(snap.Devices, Reading{
				ID: d.ID, Name: d.Name, Kind: d.Kind, Host: d.Host, Port: d.Port,
				Unit: d.Unit, Enabled: true, Notes: d.Notes,
				PollSkipped: true, Health: HealthOffline,
				Headline: "Polling off · shared ARC bus",
			})
			continue
		}
		r := readDevice(d)

		m.mu.Lock()
		if r.Online {
			m.lastGood[d.ID] = r
			m.lastAt[d.ID] = time.Now()
		} else if prev, ok := m.lastGood[d.ID]; ok && time.Since(m.lastAt[d.ID]) < staleWindow {
			// A missed read inside the stale window keeps the previous data on
			// screen, flagged stale (expected occasionally on the shared bus).
			errText := r.Err
			r = prev
			r.Stale = true
			r.Err = errText
		}
		m.mu.Unlock()

		snap.Devices = append(snap.Devices, r)
	}

	snap.Power = derivePower(snap.Devices)
	snap.Health = assessPlant(snap.Devices)
	snap.Insights = deriveInsights(snap.Devices)

	m.mu.Lock()
	evts := diffEvents(m.prev, snap)
	if len(evts) > 0 {
		m.events = append(evts, m.events...)
		if len(m.events) > maxEvents {
			m.events = m.events[:maxEvents]
		}
	}
	snap.Events = append([]Event{}, m.events...)
	m.prev = snap
	sink := m.sink
	m.mu.Unlock()

	// Deliver new events outside the lock (the sink writes to the datastore).
	if sink != nil && len(evts) > 0 {
		sink(evts)
	}

	snap.ReadMillis = time.Since(start).Milliseconds()
	return snap
}

func derivePower(devs []Reading) PowerBalance {
	p := PowerBalance{Derived: true}
	for _, r := range devs {
		if !r.Online && !r.Stale {
			continue
		}
		switch {
		case r.Sma != nil && r.Sma.ACPowerW.OK:
			p.PVKW += r.Sma.ACPowerW.Value / 1000
		case r.Oztek != nil && r.Oztek.ACPowerW.OK:
			p.BessKW += r.Oztek.ACPowerW.Value / 1000
		case r.Dse != nil && r.Dse.TotalW.OK:
			p.GensetKW += r.Dse.TotalW.Value / 1000
		}
	}
	p.PVKW = modbustcp.Round(p.PVKW, 2)
	p.BessKW = modbustcp.Round(p.BessKW, 2)
	p.GensetKW = modbustcp.Round(p.GensetKW, 2)
	// Load estimate at the MDP. BESS sign convention validates on site: if
	// OzTek reads discharge-positive this is correct; if charge-positive the
	// BESS term flips (flagged in README-plant.md).
	p.LoadKW = modbustcp.Round(p.PVKW+p.GensetKW+p.BessKW, 2)
	return p
}

var healthRank = map[string]int{HealthGood: 0, HealthWarn: 1, HealthCritical: 2, HealthOffline: 1}

func assessPlant(devs []Reading) PlantHealth {
	level := HealthGood
	reasons := []string{}
	online := 0
	polled := 0
	for _, r := range devs {
		if !r.Enabled || r.PollSkipped {
			continue // disabled or intentionally not polled (shared ARC bus)
		}
		polled++
		if r.Online || r.Stale {
			online++
		} else {
			reasons = append(reasons, r.Name+" unreachable.")
			if healthRank[level] < 1 {
				level = HealthWarn
			}
			continue
		}
		switch r.Health {
		case HealthCritical:
			level = HealthCritical
			reasons = append(reasons, r.Name+": "+r.Headline+".")
		case HealthWarn:
			if healthRank[level] < 1 {
				level = HealthWarn
			}
			reasons = append(reasons, r.Name+": "+r.Headline+".")
		}
	}
	h := PlantHealth{Level: level, Reasons: reasons}
	switch {
	case online == 0:
		h.Level = HealthOffline
		h.Headline = "No plant devices reachable - check the ARC network connection"
	case level == HealthGood:
		h.Headline = fmt.Sprintf("Plant healthy - %d/%d devices online", online, polled)
	case level == HealthWarn:
		h.Headline = fmt.Sprintf("Plant online (%d/%d) - check warnings", online, polled)
	default:
		h.Headline = "Attention needed - active fault on the plant"
	}
	return h
}

// deriveInsights runs the cross-device correlations (§6 of the plan).
func deriveInsights(devs []Reading) []string {
	out := []string{}
	var bmsR *Reading
	var ozs []*Reading
	var gen *Reading
	pvKW := 0.0
	pvOnline := 0
	for i := range devs {
		r := &devs[i]
		if !r.Online && !r.Stale {
			continue
		}
		switch {
		case r.Bms != nil:
			bmsR = r
		case r.Oztek != nil:
			ozs = append(ozs, r)
		case r.Dse != nil:
			gen = r
		case r.Sma != nil:
			pvOnline++
			if r.Sma.ACPowerW.OK {
				pvKW += r.Sma.ACPowerW.Value / 1000
			}
		}
	}

	// Battery <-> OzTek DC bus agreement: same physical bus, so the voltages
	// must match. A big gap = wiring, contactor, or comms problem.
	if bmsR != nil && bmsR.Bms.TotalV > 100 {
		for _, oz := range ozs {
			if oz.Oztek.DCVoltage.OK && oz.Oztek.DCVoltage.Value > 50 {
				diff := bmsR.Bms.TotalV - oz.Oztek.DCVoltage.Value
				if diff < 0 {
					diff = -diff
				}
				if diff > 20 {
					out = append(out, fmt.Sprintf(
						"%s DC bus reads %.0f V but the BMS reports %.0f V — check DC wiring/contactor between the combiner and the PCS.",
						oz.Name, oz.Oztek.DCVoltage.Value, bmsR.Bms.TotalV))
				}
			}
		}
	}

	// BMS relays open while an OzTek expects DC.
	if bmsR != nil && !bmsR.Bms.Combiner.Live {
		for _, oz := range ozs {
			if oz.Oztek.State == 4 || oz.Oztek.State == 5 || oz.Oztek.Online {
				out = append(out, fmt.Sprintf(
					"%s is in '%s' but the BMS combiner is de-energized (relays open) — the PCS has no battery bus.",
					oz.Name, oz.Oztek.StateText))
			}
		}
	}

	// Dispatch sanity: genset burning fuel while the battery is nearly full.
	if gen != nil && gen.Dse.Running && bmsR != nil && bmsR.Bms.SOC >= 90 {
		out = append(out, fmt.Sprintf(
			"Genset is running while the battery is at %d%% SOC — check ARC dispatch settings.", bmsR.Bms.SOC))
	}

	// Grid-forming source check: PV inverters need a formed grid to follow.
	if pvKW > 1 || pvOnline > 0 {
		forming := gen != nil && gen.Dse.Running
		for _, oz := range ozs {
			if oz.Oztek.State == 12 || oz.Oztek.GridForming {
				forming = true
			}
		}
		if !forming && len(ozs) > 0 && pvKW > 1 {
			out = append(out, "PV is producing but no grid-forming source is online (no OzTek in grid-form, genset stopped) — verify who is holding the grid.")
		}
	}

	return out
}
