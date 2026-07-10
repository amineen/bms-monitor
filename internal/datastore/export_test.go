package datastore

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/dse"
	"bms-monitor/internal/plant"
)

func seedStore(t *testing.T) *Store {
	t.Helper()
	s := tempStore(t)
	// One plant tick with an online genset, one detailed battery tick, one event.
	plantSnap := &plant.Snapshot{
		Power:  plant.PowerBalance{GensetKW: 25.6, LoadKW: 25.6},
		Health: plant.PlantHealth{Level: "warn"},
		Devices: []plant.Reading{
			{ID: "gen1", Kind: "dse", Enabled: true, Online: true, Dse: &dse.Snapshot{
				Running: true, EngineRPM: dm(1497), FreqHz: dm(49.9), TotalW: dm(25600)}},
		},
	}
	if err := s.InsertPlant(plantSnap); err != nil {
		t.Fatalf("seed plant: %v", err)
	}
	battSnap := &bms.SystemSnapshot{
		Aggregate: bms.Aggregate{OK: true, Piles: 6, TotalV: 744.2, SOC: 40},
		Combiner:  bms.CombinerStatus{OK: true, State: "Run", RelayClosed: true, Live: true},
		Strings: []bms.StringInfo{
			{Index: 1, Status: bms.StatusEnumerated, IsMaster: true, TotalV: 744.0, SOC: 40,
				CellV:   []float64{3.325, 3.326, 3.301},
				ModuleV: []float64{106.3, 106.4},
				ModuleT: []float64{29.8, 30.1}},
			{Index: 2, Status: bms.StatusEnumerated, TotalV: 744.4, SOC: 40},
		},
	}
	if err := s.InsertBattery(battSnap); err != nil {
		t.Fatalf("seed battery: %v", err)
	}
	if err := s.InsertEvents([]plant.Event{
		{At: "2026-07-07T10:00:00Z", DeviceID: "gen1", Device: "Genset", Severity: "warn", Text: "genset STARTED"},
		{At: "2026-07-07T10:01:00Z", DeviceID: "bms", Device: "Pylontech BMS", Severity: "warn", Text: "combiner Standby -> Run"},
	}); err != nil {
		t.Fatalf("seed events: %v", err)
	}
	return s
}

func TestExportSystemXLSX(t *testing.T) {
	s := seedStore(t)
	out := filepath.Join(t.TempDir(), "system.xlsx")
	if err := s.ExportXLSX(out, ScopeSystem, 0); err != nil {
		t.Fatalf("export: %v", err)
	}

	f, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatalf("open export: %v", err)
	}
	defer f.Close()

	want := []string{"Plant", "Battery", "Battery Strings", "Battery Cells", "Battery Modules", "OzTek", "Solar PV", "Genset", "Events"}
	got := f.GetSheetList()
	if len(got) != len(want) {
		t.Fatalf("sheets = %v want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("sheet[%d] = %q want %q", i, got[i], w)
		}
	}

	// Genset sheet has a data row with total_w = 25600.
	rows, err := f.GetRows("Genset")
	if err != nil || len(rows) < 2 {
		t.Fatalf("genset rows = %d err=%v", len(rows), err)
	}
	found := false
	for _, cell := range rows[1] {
		if cell == "25600" {
			found = true
		}
	}
	if !found {
		t.Errorf("genset data row missing total_w: %v", rows[1])
	}

	// Battery Cells sheet: header Time/String/Cell 1..3, master row expanded.
	cells, err := f.GetRows("Battery Cells")
	if err != nil || len(cells) < 2 {
		t.Fatalf("cells rows = %d err=%v", len(cells), err)
	}
	if cells[0][2] != "Cell 1 (V)" || len(cells[0]) != 5 {
		t.Errorf("cells header = %v", cells[0])
	}
	if cells[1][4] != "3.301" {
		t.Errorf("cell 3 value = %q want 3.301", cells[1][4])
	}

	// Modules sheet: V then T columns.
	mods, err := f.GetRows("Battery Modules")
	if err != nil || len(mods) < 2 {
		t.Fatalf("modules rows err=%v", err)
	}
	if len(mods[0]) != 6 { // Time, String, M1V, M2V, M1T, M2T
		t.Errorf("modules header = %v", mods[0])
	}

	// Events sheet has both events.
	evts, _ := f.GetRows("Events")
	if len(evts) != 3 {
		t.Errorf("events rows = %d want 3 (header + 2)", len(evts))
	}
}

func TestExportScopedAndFiltered(t *testing.T) {
	s := seedStore(t)

	// Genset scope: only Genset + Events sheets, events filtered to gen*.
	out := filepath.Join(t.TempDir(), "genset.xlsx")
	if err := s.ExportXLSX(out, ScopeGenset, 24); err != nil {
		t.Fatalf("export genset: %v", err)
	}
	f, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) != 2 || got[0] != "Genset" || got[1] != "Events" {
		t.Errorf("genset sheets = %v", got)
	}
	evts, _ := f.GetRows("Events")
	if len(evts) != 2 { // header + only the gen1 event (bms event filtered out)
		t.Errorf("filtered events rows = %d want 2", len(evts))
	}

	// Battery scope with a time window that excludes the (2026-07-07-stamped)
	// events but includes the just-inserted telemetry.
	out2 := filepath.Join(t.TempDir(), "battery.xlsx")
	if err := s.ExportXLSX(out2, ScopeBattery, 1); err != nil {
		t.Fatalf("export battery: %v", err)
	}
	f2, err := excelize.OpenFile(out2)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f2.Close()
	if got := f2.GetSheetList(); len(got) != 5 {
		t.Errorf("battery sheets = %v", got)
	}
	strs, _ := f2.GetRows("Battery Strings")
	if len(strs) != 3 { // header + 2 strings
		t.Errorf("battery strings rows = %d want 3", len(strs))
	}
}
