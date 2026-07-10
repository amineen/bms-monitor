package plant

import (
	"fmt"
	"time"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/dse"
)

// maxEvents caps the in-memory timeline ring.
const maxEvents = 300

// Event severities.
const (
	SevInfo = "info"
	SevWarn = "warn"
	SevCrit = "crit"
)

// Event is one timeline entry: a state change observed between two
// consecutive plant polls, attributed to a device.
type Event struct {
	At       string `json:"at"` // RFC3339
	DeviceID string `json:"deviceId"`
	Device   string `json:"device"`
	Severity string `json:"severity"`
	Text     string `json:"text"`
}

// diffEvents compares the previous snapshot with the current one and emits
// the troubleshooting timeline entries ("PV2 grid relay opened", "OzTek #1 ->
// Online (grid-tie)", "Genset started"). First poll emits nothing.
func diffEvents(prev, cur *Snapshot) []Event {
	if prev == nil {
		return nil
	}
	prevByID := map[string]*Reading{}
	for i := range prev.Devices {
		prevByID[prev.Devices[i].ID] = &prev.Devices[i]
	}

	now := time.Now().Format(time.RFC3339)
	var out []Event
	add := func(r *Reading, sev, text string) {
		out = append(out, Event{At: now, DeviceID: r.ID, Device: r.Name, Severity: sev, Text: text})
	}

	for i := range cur.Devices {
		c := &cur.Devices[i]
		p, ok := prevByID[c.ID]
		if !ok || !c.Enabled {
			continue
		}
		// A device on a shared bus with polling off is not "offline" — it is
		// simply not read. Never emit comms/state events for it.
		if c.PollSkipped || p.PollSkipped {
			continue
		}

		// Comms transitions. Stale readings carry previous data, so only a
		// hard offline (past the stale window) is an event.
		cOn, pOn := c.Online || c.Stale, p.Online || p.Stale
		if cOn != pOn {
			if cOn {
				add(c, SevInfo, "back online")
			} else {
				add(c, SevWarn, "lost communication")
			}
			continue
		}
		if !cOn {
			continue
		}

		switch {
		case c.Bms != nil && p.Bms != nil:
			if c.Bms.Combiner.State != p.Bms.Combiner.State {
				sev := SevInfo
				if c.Bms.Combiner.State == "Run" {
					sev = SevWarn // energization is always worth a highlighted line
				}
				add(c, sev, fmt.Sprintf("combiner %s -> %s", p.Bms.Combiner.State, c.Bms.Combiner.State))
			}
			if c.Bms.Piles != p.Bms.Piles {
				sev := SevWarn
				if c.Bms.Piles > p.Bms.Piles {
					sev = SevInfo
				}
				add(c, sev, fmt.Sprintf("strings on chain %d -> %d", p.Bms.Piles, c.Bms.Piles))
			}
		case c.Oztek != nil && p.Oztek != nil:
			if c.Oztek.State != p.Oztek.State {
				sev := SevInfo
				if c.Oztek.State == 1 {
					sev = SevCrit
				}
				add(c, sev, fmt.Sprintf("state %s -> %s", p.Oztek.StateText, c.Oztek.StateText))
			}
			for _, f := range newStrings(p.Oztek.Faults, c.Oztek.Faults) {
				add(c, SevCrit, "fault raised: "+f)
			}
			for _, f := range newStrings(c.Oztek.Faults, p.Oztek.Faults) {
				add(c, SevInfo, "fault cleared: "+f)
			}
			for _, w := range newStrings(p.Oztek.Warnings, c.Oztek.Warnings) {
				add(c, SevWarn, "warning: "+w)
			}
			for _, w := range newStrings(c.Oztek.Warnings, p.Oztek.Warnings) {
				add(c, SevInfo, "warning cleared: "+w)
			}
			for _, al := range newStrings(p.Oztek.Alarms, c.Oztek.Alarms) {
				add(c, SevWarn, "DER alarm: "+al)
			}
			for _, al := range newStrings(c.Oztek.Alarms, p.Oztek.Alarms) {
				add(c, SevInfo, "DER alarm cleared: "+al)
			}
		case c.Sma != nil && p.Sma != nil:
			if c.Sma.Condition != p.Sma.Condition {
				sev := SevInfo
				if c.Sma.Condition == "Fault" {
					sev = SevCrit
				} else if c.Sma.Condition == "Warning" {
					sev = SevWarn
				}
				add(c, sev, fmt.Sprintf("condition %s -> %s", p.Sma.Condition, c.Sma.Condition))
			}
			if c.Sma.GridRelay != p.Sma.GridRelay {
				sev := SevWarn
				if c.Sma.GridRelay == "Closed" {
					sev = SevInfo
				}
				add(c, sev, fmt.Sprintf("grid relay %s -> %s", p.Sma.GridRelay, c.Sma.GridRelay))
			}
			if c.Sma.Derating != p.Sma.Derating {
				if c.Sma.Derating != "" {
					add(c, SevWarn, "derating: "+c.Sma.Derating)
				} else {
					add(c, SevInfo, "derating cleared ("+p.Sma.Derating+")")
				}
			}
		case c.Dse != nil && p.Dse != nil:
			if c.Dse.Running != p.Dse.Running {
				if c.Dse.Running {
					add(c, SevWarn, "genset STARTED")
				} else {
					add(c, SevInfo, "genset stopped")
				}
			}
			// Named alarm conditions (GenComm page 8): raised / cleared.
			prevA := alarmSet(p.Dse.Alarms)
			curA := alarmSet(c.Dse.Alarms)
			for _, a := range c.Dse.Alarms {
				if !prevA[key(a)] {
					sev := SevWarn
					if a.Severe() {
						sev = SevCrit
					}
					add(c, sev, "alarm raised: "+a.Name+" ("+a.State+")")
				}
			}
			for _, a := range p.Dse.Alarms {
				if !curA[key(a)] {
					add(c, SevInfo, "alarm cleared: "+a.Name+" ("+a.State+")")
				}
			}
		}
	}
	return out
}

// AddEvent records an app-level event (e.g. an operator toggling shared-bus
// polling) into the ring and the persistence sink — the audit trail should
// show exactly when the app was/wasn't a second master on ARC's bus.
func (m *Monitor) AddEvent(deviceID, device, severity, text string) {
	e := Event{At: time.Now().Format(time.RFC3339), DeviceID: deviceID, Device: device, Severity: severity, Text: text}
	m.mu.Lock()
	m.events = append([]Event{e}, m.events...)
	if len(m.events) > maxEvents {
		m.events = m.events[:maxEvents]
	}
	sink := m.sink
	m.mu.Unlock()
	if sink != nil {
		sink([]Event{e})
	}
}

// IngestBatteryEvents diffs a DETAILED battery snapshot (bms.ReadSystem)
// against the previous one and emits alarm events for per-string protection
// changes — the granular battery alarms the plant round-robin (aggregate-only)
// cannot see. Events land in the same ring + sink as everything else, so they
// show on the timeline and persist to the datastore exactly once.
func (m *Monitor) IngestBatteryEvents(snap *bms.SystemSnapshot) {
	if snap == nil {
		return
	}
	now := time.Now().Format(time.RFC3339)
	var out []Event
	add := func(sev, text string) {
		out = append(out, Event{At: now, DeviceID: "bms", Device: "Pylontech BMS", Severity: sev, Text: text})
	}

	m.mu.Lock()
	prev := m.prevBatt
	m.prevBatt = snap
	if prev != nil {
		prevBy := map[int]*bms.StringInfo{}
		for i := range prev.Strings {
			prevBy[prev.Strings[i].Index] = &prev.Strings[i]
		}
		for i := range snap.Strings {
			c := &snap.Strings[i]
			p, ok := prevBy[c.Index]
			if !ok {
				continue
			}
			sn := "string " + itoa(c.Index)
			for _, a := range newStrings(p.Alarms, c.Alarms) {
				add(SevCrit, "protection raised on "+sn+": "+a)
			}
			for _, a := range newStrings(c.Alarms, p.Alarms) {
				add(SevInfo, "protection cleared on "+sn+": "+a)
			}
			// A string dropping off the chain between detailed reads.
			if p.Status == bms.StatusEnumerated && c.Status != bms.StatusEnumerated {
				add(SevWarn, sn+" dropped off the chain ("+c.Status+")")
			}
			if p.Status != bms.StatusEnumerated && c.Status == bms.StatusEnumerated {
				add(SevInfo, sn+" back on the chain")
			}
		}
	}
	if len(out) > 0 {
		m.events = append(out, m.events...)
		if len(m.events) > maxEvents {
			m.events = m.events[:maxEvents]
		}
	}
	sink := m.sink
	m.mu.Unlock()

	if sink != nil && len(out) > 0 {
		sink(out)
	}
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func key(a dse.Alarm) string { return a.Name + "|" + a.State }

func alarmSet(alarms []dse.Alarm) map[string]bool {
	m := map[string]bool{}
	for _, a := range alarms {
		m[key(a)] = true
	}
	return m
}

// newStrings returns the entries in cur that are not in prevList.
func newStrings(prevList, cur []string) []string {
	seen := map[string]bool{}
	for _, s := range prevList {
		seen[s] = true
	}
	var out []string
	for _, s := range cur {
		if !seen[s] {
			out = append(out, s)
		}
	}
	return out
}
