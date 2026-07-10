package plant

import (
	"testing"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/dse"
	"bms-monitor/internal/oztek"
	"bms-monitor/internal/sma"
)

func TestDefaultDevices(t *testing.T) {
	devs := DefaultDevices()
	if len(devs) != 8 {
		t.Fatalf("bundled registry has %d devices, want 8", len(devs))
	}
	byID := map[string]Device{}
	oztekCount := 0
	for _, d := range devs {
		byID[d.ID] = d
		if d.Kind == KindOzTek {
			oztekCount++
			if d.Host != "192.168.0.41" {
				t.Errorf("OzTek %s host = %s, all must share the .41 gateway", d.ID, d.Host)
			}
			if !d.SharedBus {
				t.Errorf("OzTek %s must be SharedBus (ARC masters that bus)", d.ID)
			}
		} else if d.SharedBus {
			t.Errorf("%s must NOT be SharedBus (own converter / native Ethernet)", d.ID)
		}
	}
	if oztekCount != 3 {
		t.Errorf("oztek count = %d want 3", oztekCount)
	}
	if d := byID["bms"]; d.Host != "192.168.0.31" || d.Unit != 1 || !d.Enabled {
		t.Errorf("bms entry wrong: %+v", d)
	}
	if d := byID["oztek3"]; d.Enabled {
		t.Error("oztek3 (BATT#3, not wired) must ship disabled")
	}
	if d := byID["gen1"]; d.Kind != KindDSE || d.Host != "192.168.0.71" {
		t.Errorf("genset entry wrong: %+v", d)
	}
	if d := byID["pv2"]; d.Kind != KindSMA || d.Host != "192.168.0.52" || d.Unit != 3 {
		t.Errorf("pv2 entry wrong: %+v", d)
	}
}

func m(v float64) oztek.Metric { return oztek.Metric{OK: true, Value: v} }

func TestDerivePower(t *testing.T) {
	devs := []Reading{
		{Online: true, Sma: &sma.Snapshot{ACPowerW: sma.Metric{OK: true, Value: 15000}}},
		{Online: true, Sma: &sma.Snapshot{ACPowerW: sma.Metric{OK: true, Value: 12000}}},
		{Online: true, Oztek: &oztek.Snapshot{ACPowerW: m(-8000)}}, // charging
		{Online: true, Dse: &dse.Snapshot{TotalW: dse.Metric{OK: true, Value: 20000}}},
		{Online: false, Sma: &sma.Snapshot{ACPowerW: sma.Metric{OK: true, Value: 99999}}}, // offline: excluded
	}
	p := derivePower(devs)
	if p.PVKW != 27 || p.BessKW != -8 || p.GensetKW != 20 {
		t.Errorf("power = %+v", p)
	}
	if p.LoadKW != 39 {
		t.Errorf("load = %v want 39 (27+20-8)", p.LoadKW)
	}
}

func TestAssessReadingLevels(t *testing.T) {
	r := Reading{Online: true, Bms: &bms.AggSummary{Piles: 6, TotalV: 749.9, SOC: 80}}
	assessReading(&r)
	if r.Health != HealthGood {
		t.Errorf("6/6 bms health = %s", r.Health)
	}
	r = Reading{Online: true, Bms: &bms.AggSummary{Piles: 4}}
	assessReading(&r)
	if r.Health != HealthWarn {
		t.Errorf("4/6 bms health = %s", r.Health)
	}
	r = Reading{Online: true, Oztek: &oztek.Snapshot{State: 1, StateText: "Fault", Faults: []string{"DC under-voltage"}}}
	assessReading(&r)
	if r.Health != HealthCritical {
		t.Errorf("faulted oztek health = %s", r.Health)
	}
	r = Reading{Online: true, Sma: &sma.Snapshot{Condition: "OK", ACPowerW: sma.Metric{OK: true, Value: 15000}, Derating: "Over-temperature"}}
	assessReading(&r)
	if r.Health != HealthWarn {
		t.Errorf("derating sma health = %s", r.Health)
	}
	r = Reading{Online: true, Dse: &dse.Snapshot{Running: true,
		EngineRPM:      dse.Metric{OK: true, Value: 1500},
		TotalW:         dse.Metric{OK: true, Value: 50000},
		OilPressureKPa: dse.Metric{OK: true, Value: 60}}}
	assessReading(&r)
	if r.Health != HealthCritical {
		t.Errorf("low-oil genset health = %s", r.Health)
	}
}

func TestDiffEvents(t *testing.T) {
	prev := &Snapshot{Devices: []Reading{
		{ID: "gen1", Name: "Genset", Enabled: true, Online: true, Dse: &dse.Snapshot{Running: false}},
		{ID: "oztek1", Name: "OzTek #1", Enabled: true, Online: true,
			Oztek: &oztek.Snapshot{State: 6, StateText: "Standby", Faults: []string{}}},
		{ID: "pv1", Name: "SMA PV1", Enabled: true, Online: true,
			Sma: &sma.Snapshot{Condition: "OK", GridRelay: "Closed"}},
		{ID: "bms", Name: "Pylontech BMS", Enabled: true, Online: true,
			Bms: &bms.AggSummary{Piles: 6, Combiner: bms.CombinerStatus{State: "Standby"}}},
	}}
	cur := &Snapshot{Devices: []Reading{
		{ID: "gen1", Name: "Genset", Enabled: true, Online: true, Dse: &dse.Snapshot{Running: true}},
		{ID: "oztek1", Name: "OzTek #1", Enabled: true, Online: true,
			Oztek: &oztek.Snapshot{State: 8, StateText: "Online (grid-tie)", Faults: []string{"Communication error"}}},
		{ID: "pv1", Name: "SMA PV1", Enabled: true, Online: false},
		{ID: "bms", Name: "Pylontech BMS", Enabled: true, Online: true,
			Bms: &bms.AggSummary{Piles: 6, Combiner: bms.CombinerStatus{State: "Run", Live: true}}},
	}}

	evts := diffEvents(prev, cur)
	texts := map[string]string{}
	for _, e := range evts {
		texts[e.Device+": "+e.Text] = e.Severity
	}
	if sev, ok := texts["Genset: genset STARTED"]; !ok || sev != SevWarn {
		t.Errorf("missing genset start event: %v", texts)
	}
	if _, ok := texts["OzTek #1: state Standby -> Online (grid-tie)"]; !ok {
		t.Errorf("missing oztek state event: %v", texts)
	}
	if sev, ok := texts["OzTek #1: fault raised: Communication error"]; !ok || sev != SevCrit {
		t.Errorf("missing oztek fault event: %v", texts)
	}
	if sev, ok := texts["SMA PV1: lost communication"]; !ok || sev != SevWarn {
		t.Errorf("missing pv offline event: %v", texts)
	}
	if sev, ok := texts["Pylontech BMS: combiner Standby -> Run"]; !ok || sev != SevWarn {
		t.Errorf("missing combiner event: %v", texts)
	}
	// First poll produces no events.
	if got := diffEvents(nil, cur); got != nil {
		t.Errorf("first poll should emit nothing, got %v", got)
	}
}

func TestInsights(t *testing.T) {
	devs := []Reading{
		{Name: "Pylontech BMS", Online: true, Bms: &bms.AggSummary{TotalV: 749.9, SOC: 95,
			Combiner: bms.CombinerStatus{Live: true, State: "Run"}}},
		{Name: "OzTek #1", Online: true, Oztek: &oztek.Snapshot{State: 8, StateText: "Online (grid-tie)",
			Online: true, DCVoltage: m(700)}}, // 50 V below the BMS bus
		{Name: "Genset", Online: true, Dse: &dse.Snapshot{Running: true,
			EngineRPM: dse.Metric{OK: true, Value: 1500}}},
	}
	insights := deriveInsights(devs)
	foundDC, foundDispatch := false, false
	for _, s := range insights {
		if len(s) > 0 && contains(s, "DC bus reads") {
			foundDC = true
		}
		if contains(s, "dispatch") {
			foundDispatch = true
		}
	}
	if !foundDC {
		t.Errorf("missing DC mismatch insight: %v", insights)
	}
	if !foundDispatch {
		t.Errorf("missing dispatch insight: %v", insights)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSafetyOverlayForcesSharedBusOnOzTeks(t *testing.T) {
	// A saved registry from an older build (no sharedBus field) or a
	// hand-edited file must NEVER re-arm polling on ARC's bus: the overlay
	// forces SharedBus on every OzTek regardless of the stored value.
	stale := []Device{
		{ID: "oztek1", Kind: KindOzTek, Host: "192.168.0.41", Unit: 1, Enabled: true, SharedBus: false},
		{ID: "oztek9", Kind: KindOzTek, Host: "192.168.0.99", Unit: 1, Enabled: true, SharedBus: false},
		{ID: "bms", Kind: KindBMS, Host: "192.168.0.31", Unit: 1, Enabled: true},
		{ID: "pv1", Kind: KindSMA, Host: "192.168.0.51", Unit: 3, Enabled: true},
	}
	out := withSafetyOverlay(stale)
	for _, d := range out {
		if d.Kind == KindOzTek && !d.SharedBus {
			t.Errorf("overlay must force SharedBus on %s", d.ID)
		}
		if d.Kind != KindOzTek && d.SharedBus {
			t.Errorf("overlay must not touch non-OzTek %s", d.ID)
		}
	}
}

func TestAddEventReachesRingAndSink(t *testing.T) {
	m := NewMonitor()
	var sunk []Event
	m.SetEventSink(func(e []Event) { sunk = append(sunk, e...) })
	m.AddEvent("app", "Plant Monitor", SevWarn, "OzTek shared-bus polling ENABLED by operator")
	if len(sunk) != 1 || sunk[0].Severity != SevWarn {
		t.Fatalf("sink events = %+v", sunk)
	}
	// The event must also appear in the next snapshot's ring.
	snap := m.Read(nil, false)
	if len(snap.Events) != 1 || snap.Events[0].Text != "OzTek shared-bus polling ENABLED by operator" {
		t.Errorf("ring events = %+v", snap.Events)
	}
}

func TestReadSkipsSharedBusWhenOff(t *testing.T) {
	m := NewMonitor()
	devs := []Device{
		{ID: "oztek1", Name: "OzTek #1", Kind: KindOzTek, Host: "192.168.0.41", Port: 502, Unit: 1, Enabled: true, SharedBus: true, Notes: "shared bus"},
		{ID: "oztek3", Name: "OzTek #3", Kind: KindOzTek, Host: "192.168.0.41", Port: 502, Unit: 3, Enabled: false, SharedBus: true},
	}

	// Polling OFF: shared-bus device is reported as poll-skipped and NO Modbus
	// is attempted (no reachability probe, so Err stays empty).
	snap := m.Read(devs, false)
	oz := snap.Devices[0]
	if !oz.PollSkipped {
		t.Errorf("shared-bus device should be PollSkipped when polling off: %+v", oz)
	}
	if oz.Err != "" {
		t.Errorf("no read should be attempted; Err = %q", oz.Err)
	}
	if oz.Headline != "Polling off · shared ARC bus" {
		t.Errorf("headline = %q", oz.Headline)
	}
	// A disabled shared-bus device is still just "Disabled", not poll-skipped.
	if snap.Devices[1].PollSkipped || snap.Devices[1].Headline != "Disabled" {
		t.Errorf("disabled device = %+v", snap.Devices[1])
	}

	// Polling ON: it actually tries to read (offline in test env → an Err, and
	// NOT poll-skipped).
	snap2 := m.Read(devs, true)
	oz2 := snap2.Devices[0]
	if oz2.PollSkipped {
		t.Error("with polling on, device must not be PollSkipped")
	}
	if oz2.Err == "" {
		t.Error("with polling on, an unreachable device should carry an Err")
	}
}

func TestAssessPlantExcludesPollSkipped(t *testing.T) {
	devs := []Reading{
		{ID: "gen1", Name: "Genset", Enabled: true, Online: true, Health: HealthGood, Headline: "Running"},
		{ID: "oztek1", Name: "OzTek #1", Enabled: true, PollSkipped: true, Health: HealthOffline},
	}
	h := assessPlant(devs)
	// Poll-skipped OzTek must not drag the plant to warn or count as offline.
	if h.Level != HealthGood {
		t.Errorf("plant level = %s want good (poll-skipped excluded)", h.Level)
	}
	if h.Headline != "Plant healthy - 1/1 devices online" {
		t.Errorf("headline = %q want 1/1", h.Headline)
	}
}

func TestGensetAlarmEventsAndHealth(t *testing.T) {
	quiet := &dse.Snapshot{Running: true, Alarms: []dse.Alarm{}}
	tripped := &dse.Snapshot{Running: false, Alarms: []dse.Alarm{
		{Index: 2, Name: "Low oil pressure", State: dse.AlarmShutdown},
		{Index: 28, Name: "Low fuel level", State: dse.AlarmWarning},
	}}

	prev := &Snapshot{Devices: []Reading{{ID: "gen1", Name: "Genset", Enabled: true, Online: true, Dse: quiet}}}
	cur := &Snapshot{Devices: []Reading{{ID: "gen1", Name: "Genset", Enabled: true, Online: true, Dse: tripped}}}
	evts := diffEvents(prev, cur)
	texts := map[string]string{}
	for _, e := range evts {
		texts[e.Text] = e.Severity
	}
	if sev, ok := texts["alarm raised: Low oil pressure (Shutdown)"]; !ok || sev != SevCrit {
		t.Errorf("missing shutdown alarm event: %v", texts)
	}
	if sev, ok := texts["alarm raised: Low fuel level (Warning)"]; !ok || sev != SevWarn {
		t.Errorf("missing warning alarm event: %v", texts)
	}
	// Clearing emits info events.
	back := diffEvents(cur, prev)
	found := 0
	for _, e := range back {
		if e.Severity == SevInfo && contains(e.Text, "alarm cleared") {
			found++
		}
	}
	if found != 2 {
		t.Errorf("cleared events = %d want 2: %v", found, back)
	}

	// Health: severe alarm dominates the headline.
	r := Reading{Online: true, Dse: tripped}
	assessReading(&r)
	if r.Health != HealthCritical || !contains(r.Headline, "Low oil pressure") {
		t.Errorf("alarm health = %s %q", r.Health, r.Headline)
	}
}

func TestIngestBatteryEvents(t *testing.T) {
	m := NewMonitor()
	var sunk []Event
	m.SetEventSink(func(e []Event) { sunk = append(sunk, e...) })

	first := &bms.SystemSnapshot{Strings: []bms.StringInfo{
		{Index: 1, Status: bms.StatusEnumerated, Alarms: []string{}},
		{Index: 2, Status: bms.StatusEnumerated, Alarms: []string{}},
	}}
	m.IngestBatteryEvents(first) // baseline: no events
	if len(sunk) != 0 {
		t.Fatalf("baseline produced events: %v", sunk)
	}

	second := &bms.SystemSnapshot{Strings: []bms.StringInfo{
		{Index: 1, Status: bms.StatusEnumerated, Alarms: []string{"Cell 2nd under-voltage"}},
		{Index: 2, Status: bms.StatusNoResponse, Alarms: []string{}},
	}}
	m.IngestBatteryEvents(second)
	if len(sunk) != 2 {
		t.Fatalf("events = %d want 2: %v", len(sunk), sunk)
	}
	byText := map[string]string{}
	for _, e := range sunk {
		byText[e.Text] = e.Severity
		if e.DeviceID != "bms" {
			t.Errorf("device id = %q", e.DeviceID)
		}
	}
	if sev := byText["protection raised on string 1: Cell 2nd under-voltage"]; sev != SevCrit {
		t.Errorf("protection event missing/wrong sev: %v", byText)
	}
	if sev := byText["string 2 dropped off the chain (no-response)"]; sev != SevWarn {
		t.Errorf("chain-drop event missing: %v", byText)
	}

	// Clearing the protection emits an info event; ring holds everything.
	m.IngestBatteryEvents(first)
	if len(sunk) != 4 { // cleared + string 2 back
		t.Fatalf("after clear events = %d want 4: %v", len(sunk), sunk)
	}
}
