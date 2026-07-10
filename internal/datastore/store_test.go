package datastore

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/dse"
	"bms-monitor/internal/oztek"
	"bms-monitor/internal/plant"
	"bms-monitor/internal/sma"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func om(v float64) oztek.Metric { return oztek.Metric{OK: true, Value: v} }
func dm(v float64) dse.Metric   { return dse.Metric{OK: true, Value: v} }
func sm(v float64) sma.Metric   { return sma.Metric{OK: true, Value: v} }

func TestInsertPlantRoundTrip(t *testing.T) {
	s := tempStore(t)
	snap := &plant.Snapshot{
		Power:  plant.PowerBalance{PVKW: 27, BessKW: -8, GensetKW: 20, LoadKW: 39},
		Health: plant.PlantHealth{Level: "warn"},
		Devices: []plant.Reading{
			{ID: "gen1", Kind: "dse", Enabled: true, Online: true, Dse: &dse.Snapshot{
				Running: true, EngineRPM: dm(1500), FreqHz: dm(50.0), TotalW: dm(25600),
				OilPressureKPa: dm(389), CoolantTempC: dm(82),
				VLn: [3]dse.Metric{dm(242.3), dm(240.5), dm(245.8)},
				VLl: [3]dse.Metric{dm(415.6), dm(422.0), dm(423.9)},
				AmpsL: [3]dse.Metric{dm(60), dm(38), dm(26)},
				WattsL: [3]dse.Metric{dm(13510), dm(8050), dm(5820)},
			}},
			{ID: "oztek1", Kind: "oztek", Enabled: true, Online: true, Oztek: &oztek.Snapshot{
				State: 8, StateText: "Online (grid-tie)", GridConnected: true,
				ACPowerW: om(18000), DCVoltage: om(745.5), DCCurrent: om(-25),
				FaultRaw: 0, WarningRaw: 0x100,
			}},
			{ID: "pv1", Kind: "sma", Enabled: true, Online: false, Err: "no TCP connection"}, // offline: no row
			{ID: "pv2", Kind: "sma", Enabled: true, Online: true, Stale: true, Sma: &sma.Snapshot{
				Condition: "OK", GridRelay: "Closed", ACPowerW: sm(14980), DailyYieldKWh: sm(87.65),
			}},
			{ID: "oztek3", Kind: "oztek", Enabled: false}, // disabled: no row, not counted
		},
	}
	if err := s.InsertPlant(snap); err != nil {
		t.Fatalf("insert plant: %v", err)
	}

	var pv, load float64
	var health string
	var online, total int
	if err := s.db.QueryRow(`SELECT pv_kw, load_kw, health, online, total FROM plant_log`).
		Scan(&pv, &load, &health, &online, &total); err != nil {
		t.Fatalf("query plant: %v", err)
	}
	if pv != 27 || load != 39 || health != "warn" || online != 3 || total != 4 {
		t.Errorf("plant row = %v %v %q %d/%d", pv, load, health, online, total)
	}

	var dseRows, ozRows, smaRows int
	s.db.QueryRow(`SELECT COUNT(*) FROM dse_log`).Scan(&dseRows)
	s.db.QueryRow(`SELECT COUNT(*) FROM oztek_log`).Scan(&ozRows)
	s.db.QueryRow(`SELECT COUNT(*) FROM sma_log`).Scan(&smaRows)
	if dseRows != 1 || ozRows != 1 || smaRows != 1 {
		t.Errorf("rows dse=%d oz=%d sma=%d want 1/1/1", dseRows, ozRows, smaRows)
	}

	// Stale flag persisted; NULL metrics stay NULL.
	var stale int
	var hz any
	if err := s.db.QueryRow(`SELECT stale, hz FROM sma_log WHERE device='pv2'`).Scan(&stale, &hz); err != nil {
		t.Fatalf("query sma: %v", err)
	}
	if stale != 1 {
		t.Error("pv2 stale flag not persisted")
	}
	if hz != nil {
		t.Errorf("unpopulated freq should be NULL, got %v", hz)
	}

	var dcV float64
	if err := s.db.QueryRow(`SELECT dc_v FROM oztek_log WHERE device='oztek1'`).Scan(&dcV); err != nil {
		t.Fatalf("query oztek: %v", err)
	}
	if dcV != 745.5 {
		t.Errorf("oztek dc_v = %v want 745.5", dcV)
	}
}

func TestInsertBatteryGranular(t *testing.T) {
	s := tempStore(t)
	cells := make([]float64, 224)
	for i := range cells {
		cells[i] = 3.325
	}
	cells[42] = 3.301 // the interesting one
	snap := &bms.SystemSnapshot{
		LinkLive: true,
		Aggregate: bms.Aggregate{OK: true, Piles: 6, TotalV: 744.2, Current: -5.5,
			PowerKW: -4.093, SOC: 40, SOH: 100, Temp: 30.1, CellMaxV: 3.331, CellMinV: 3.301},
		Combiner: bms.CombinerStatus{OK: true, Live: true, State: "Run", RelayClosed: true},
		Strings: []bms.StringInfo{
			{Index: 1, Status: bms.StatusEnumerated, IsMaster: true, Serial: "PPTBK...",
				TotalV: 744.0, SOC: 40, SOH: 100, Temp: 30.0, CellSpreadMV: 30,
				CellV: cells, ModuleV: []float64{106.3, 106.4, 106.3, 106.3, 106.4, 106.3, 106.2},
				ModuleT: []float64{29.8, 30.1, 30.0, 30.2, 29.9, 30.0, 30.1}},
			{Index: 2, Status: bms.StatusEnumerated, TotalV: 744.4, SOC: 40},
			{Index: 3, Status: bms.StatusNoResponse}, // not logged
		},
	}
	if err := s.InsertBattery(snap); err != nil {
		t.Fatalf("insert battery: %v", err)
	}

	var piles, relay int
	var combiner string
	if err := s.db.QueryRow(`SELECT piles, relay_closed, combiner FROM bms_log`).Scan(&piles, &relay, &combiner); err != nil {
		t.Fatalf("query bms_log: %v", err)
	}
	if piles != 6 || relay != 1 || combiner != "Run" {
		t.Errorf("bms row = %d %d %q", piles, relay, combiner)
	}

	var stringRows int
	s.db.QueryRow(`SELECT COUNT(*) FROM bms_string_log`).Scan(&stringRows)
	if stringRows != 2 {
		t.Errorf("string rows = %d want 2 (no-response strings skipped)", stringRows)
	}

	// The master row must carry the full cell array, decodable from JSON.
	var cellJSON string
	if err := s.db.QueryRow(`SELECT cell_v FROM bms_string_log WHERE string=1`).Scan(&cellJSON); err != nil {
		t.Fatalf("query master cells: %v", err)
	}
	var decoded []float64
	if err := json.Unmarshal([]byte(cellJSON), &decoded); err != nil {
		t.Fatalf("cell json: %v", err)
	}
	if len(decoded) != 224 || decoded[42] != 3.301 {
		t.Errorf("cells len=%d [42]=%v want 224 / 3.301", len(decoded), decoded[42])
	}

	// Slave row has NULL arrays.
	var slaveCells any
	s.db.QueryRow(`SELECT cell_v FROM bms_string_log WHERE string=2`).Scan(&slaveCells)
	if slaveCells != nil {
		t.Errorf("slave cell_v should be NULL, got %v", slaveCells)
	}

	// SQLite-side analysis works: min over json_each of the master array.
	var minV float64
	if err := s.db.QueryRow(
		`SELECT MIN(value) FROM bms_string_log, json_each(bms_string_log.cell_v) WHERE string=1`,
	).Scan(&minV); err != nil {
		t.Fatalf("json_each: %v", err)
	}
	if minV != 3.301 {
		t.Errorf("json min = %v want 3.301", minV)
	}
}

func TestEventsAndStats(t *testing.T) {
	s := tempStore(t)
	events := []plant.Event{
		{At: "2026-07-07T10:00:00Z", DeviceID: "gen1", Device: "Genset", Severity: "warn", Text: "genset STARTED"},
		{At: "bad-timestamp", DeviceID: "pv1", Device: "SMA PV1", Severity: "info", Text: "back online"},
	}
	if err := s.InsertEvents(events); err != nil {
		t.Fatalf("insert events: %v", err)
	}
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM event_log`).Scan(&n)
	if n != 2 {
		t.Errorf("event rows = %d want 2", n)
	}

	st, err := s.GetStats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if st.Rows["event_log"] != 2 {
		t.Errorf("stats event rows = %d", st.Rows["event_log"])
	}
	if st.SizeBytes <= 0 {
		t.Error("db size should be > 0")
	}
}

func TestPrune(t *testing.T) {
	s := tempStore(t)
	// Insert one current row and one ancient row directly.
	if _, err := s.db.Exec(`INSERT INTO plant_log (ts, pv_kw) VALUES (?, 1), (?, 2)`,
		1_000_000, // 1970 — far past any retention window
		timeNowMs()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := s.Prune(30); err != nil {
		t.Fatalf("prune: %v", err)
	}
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM plant_log`).Scan(&n)
	if n != 1 {
		t.Errorf("rows after prune = %d want 1", n)
	}
}

func timeNowMs() int64 {
	return time.Now().UnixMilli()
}
