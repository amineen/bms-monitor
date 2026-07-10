// Package datastore is the local time-series log for the plant monitor: a
// single SQLite file that records the throughput and key parameters of every
// device on each logging tick, with the battery stored at full granularity
// (per-string rows plus the master's complete per-cell and per-module arrays).
//
// Design notes:
//   - modernc.org/sqlite (pure Go) so the Windows cross-build keeps working.
//   - WAL mode + one transaction per tick: a full plant tick is a handful of
//     inserts, so 30 s cadence is trivial load.
//   - Offline devices are NOT logged as rows (the event_log records the
//     online/offline transitions instead) — keeps the tables dense and small.
//   - Cell/module arrays are stored as JSON text so the DB stays inspectable
//     with any SQLite browser (and SQLite's json_each() can unpack them).
//   - Metrics that a device does not populate are stored as NULL, never 0.
package datastore

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/dse"
	"bms-monitor/internal/oztek"
	"bms-monitor/internal/plant"
	"bms-monitor/internal/sma"
)

// DefaultPath returns the standard DB location next to the app config.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	d := filepath.Join(dir, "bms-monitor")
	_ = os.MkdirAll(d, 0o755)
	return filepath.Join(d, "plantlog.db")
}

// Store wraps the SQLite handle. All writes are serialized by mu (SQLite has
// a single writer anyway; this keeps "one transaction per tick" simple).
type Store struct {
	mu   sync.Mutex
	db   *sql.DB
	path string
}

const schema = `
CREATE TABLE IF NOT EXISTS plant_log (
  ts        INTEGER NOT NULL,          -- unix milliseconds
  pv_kw     REAL, bess_kw REAL, genset_kw REAL, load_kw REAL,
  health    TEXT, online INTEGER, total INTEGER
);
CREATE INDEX IF NOT EXISTS idx_plant_ts ON plant_log(ts);

CREATE TABLE IF NOT EXISTS oztek_log (
  ts INTEGER NOT NULL, device TEXT NOT NULL, stale INTEGER NOT NULL DEFAULT 0,
  state INTEGER, state_text TEXT, grid_connected INTEGER, grid_forming INTEGER,
  ac_w REAL, ac_a REAL, v_ll REAL, hz REAL, pf REAL, va REAL, var REAL,
  dc_v REAL, dc_a REAL, dc_w REAL, cab_c REAL, hs_c REAL,
  heartbeat INTEGER, alarm INTEGER, warning INTEGER, fault INTEGER, factory INTEGER
);
CREATE INDEX IF NOT EXISTS idx_oztek_ts ON oztek_log(device, ts);

CREATE TABLE IF NOT EXISTS sma_log (
  ts INTEGER NOT NULL, device TEXT NOT NULL, stale INTEGER NOT NULL DEFAULT 0,
  condition TEXT, relay TEXT, derating TEXT,
  ac_w REAL, va REAL, hz REAL, v1 REAL, v2 REAL, v3 REAL,
  mppt_a_v REAL, mppt_a_a REAL, mppt_a_w REAL,
  mppt_b_v REAL, mppt_b_a REAL, mppt_b_w REAL, dc_w REAL,
  temp_c REAL, daily_kwh REAL, total_kwh REAL
);
CREATE INDEX IF NOT EXISTS idx_sma_ts ON sma_log(device, ts);

CREATE TABLE IF NOT EXISTS dse_log (
  ts INTEGER NOT NULL, device TEXT NOT NULL, stale INTEGER NOT NULL DEFAULT 0,
  running INTEGER, rpm REAL, hz REAL,
  oil_kpa REAL, coolant_c REAL, fuel_pct REAL, batt_v REAL, alt_v REAL,
  total_w REAL, total_va REAL, total_var REAL, pf REAL,
  l1_w REAL, l2_w REAL, l3_w REAL,
  l1_v REAL, l2_v REAL, l3_v REAL,
  l1_a REAL, l2_a REAL, l3_a REAL,
  kwh REAL, starts REAL, run_h REAL,
  alarms TEXT
);
CREATE INDEX IF NOT EXISTS idx_dse_ts ON dse_log(device, ts);

CREATE TABLE IF NOT EXISTS bms_log (
  ts INTEGER NOT NULL,
  piles INTEGER, total_v REAL, current_a REAL, power_kw REAL,
  soc INTEGER, soh INTEGER, temp_c REAL,
  cell_max_v REAL, cell_min_v REAL,
  combiner TEXT, relay_closed INTEGER, link_live INTEGER
);
CREATE INDEX IF NOT EXISTS idx_bms_ts ON bms_log(ts);

CREATE TABLE IF NOT EXISTS bms_string_log (
  ts INTEGER NOT NULL, string INTEGER NOT NULL,
  status TEXT, serial TEXT,
  total_v REAL, current_a REAL, power_kw REAL,
  soc INTEGER, soh INTEGER, soe INTEGER,
  temp_c REAL, cycles INTEGER,
  cell_max_v REAL, cell_min_v REAL, spread_mv REAL,
  cell_max_t REAL, cell_min_t REAL,
  mod_max_v REAL, mod_min_v REAL,
  basic TEXT, protection TEXT, alarms TEXT,
  cell_v TEXT, module_v TEXT, module_t TEXT      -- JSON arrays (master only)
);
CREATE INDEX IF NOT EXISTS idx_bms_string_ts ON bms_string_log(string, ts);

CREATE TABLE IF NOT EXISTS event_log (
  ts INTEGER NOT NULL, device_id TEXT, device TEXT, severity TEXT, text TEXT
);
CREATE INDEX IF NOT EXISTS idx_event_ts ON event_log(ts);
`

// Open creates/opens the database and applies the schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // single writer; readers share the same conn fine at our volume
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	// Migrations for databases created by earlier builds (CREATE TABLE IF NOT
	// EXISTS won't add new columns). "duplicate column" errors are expected.
	for _, mig := range []string{
		`ALTER TABLE dse_log ADD COLUMN alarms TEXT`,
	} {
		_, _ = db.Exec(mig)
	}
	return &Store{db: db, path: path}, nil
}

// Close flushes and closes the database.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

// nf converts a driver Metric to a nullable float (NULL when not populated).
func nf(ok bool, v float64) any {
	if !ok {
		return nil
	}
	return v
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func jarr(v []float64) any {
	if len(v) == 0 {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(b)
}

// InsertPlant writes one plant tick: the overall power-balance row plus one
// typed row per ONLINE device (stale readings are logged and flagged, so a
// brief .41 bus miss doesn't punch holes in the series).
func (s *Store) InsertPlant(snap *plant.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ts := time.Now().UnixMilli()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	online, total := 0, 0
	for _, d := range snap.Devices {
		if d.Enabled && !d.PollSkipped {
			total++
			if d.Online || d.Stale {
				online++
			}
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO plant_log (ts, pv_kw, bess_kw, genset_kw, load_kw, health, online, total)
		 VALUES (?,?,?,?,?,?,?,?)`,
		ts, snap.Power.PVKW, snap.Power.BessKW, snap.Power.GensetKW, snap.Power.LoadKW,
		snap.Health.Level, online, total,
	); err != nil {
		return err
	}

	for _, r := range snap.Devices {
		if !r.Enabled || (!r.Online && !r.Stale) {
			continue
		}
		switch {
		case r.Oztek != nil:
			if err := insertOztek(tx, ts, r.ID, r.Stale, r.Oztek); err != nil {
				return err
			}
		case r.Sma != nil:
			if err := insertSma(tx, ts, r.ID, r.Stale, r.Sma); err != nil {
				return err
			}
		case r.Dse != nil:
			if err := insertDse(tx, ts, r.ID, r.Stale, r.Dse); err != nil {
				return err
			}
			// The BMS aggregate inside the plant snapshot is intentionally NOT
			// logged here — the battery gets its own detailed tick (InsertBattery)
			// with per-string and per-cell data, and double-logging the aggregate
			// would just duplicate rows.
		}
	}
	return tx.Commit()
}

func insertOztek(tx *sql.Tx, ts int64, id string, stale bool, o *oztek.Snapshot) error {
	_, err := tx.Exec(
		`INSERT INTO oztek_log (ts, device, stale, state, state_text, grid_connected, grid_forming,
		   ac_w, ac_a, v_ll, hz, pf, va, var, dc_v, dc_a, dc_w, cab_c, hs_c,
		   heartbeat, alarm, warning, fault, factory)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ts, id, b2i(stale), o.State, o.StateText, b2i(o.GridConnected), b2i(o.GridForming),
		nf(o.ACPowerW.OK, o.ACPowerW.Value), nf(o.ACCurrentA.OK, o.ACCurrentA.Value),
		nf(o.VLl.OK, o.VLl.Value), nf(o.FreqHz.OK, o.FreqHz.Value),
		nf(o.PowerFactor.OK, o.PowerFactor.Value), nf(o.ApparentVA.OK, o.ApparentVA.Value),
		nf(o.ReactiveVAR.OK, o.ReactiveVAR.Value),
		nf(o.DCVoltage.OK, o.DCVoltage.Value), nf(o.DCCurrent.OK, o.DCCurrent.Value),
		nf(o.DCPowerW.OK, o.DCPowerW.Value),
		nf(o.CabinetTempC.OK, o.CabinetTempC.Value), nf(o.HeatsinkTempC.OK, o.HeatsinkTempC.Value),
		o.Heartbeat, o.AlarmRaw, o.WarningRaw, o.FaultRaw, o.FactoryRaw,
	)
	return err
}

func insertSma(tx *sql.Tx, ts int64, id string, stale bool, m *sma.Snapshot) error {
	_, err := tx.Exec(
		`INSERT INTO sma_log (ts, device, stale, condition, relay, derating,
		   ac_w, va, hz, v1, v2, v3,
		   mppt_a_v, mppt_a_a, mppt_a_w, mppt_b_v, mppt_b_a, mppt_b_w, dc_w,
		   temp_c, daily_kwh, total_kwh)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ts, id, b2i(stale), m.Condition, m.GridRelay, m.Derating,
		nf(m.ACPowerW.OK, m.ACPowerW.Value), nf(m.ApparentVA.OK, m.ApparentVA.Value),
		nf(m.FreqHz.OK, m.FreqHz.Value),
		nf(m.GridV[0].OK, m.GridV[0].Value), nf(m.GridV[1].OK, m.GridV[1].Value), nf(m.GridV[2].OK, m.GridV[2].Value),
		nf(m.MPPTA.VoltageV.OK, m.MPPTA.VoltageV.Value), nf(m.MPPTA.CurrentA.OK, m.MPPTA.CurrentA.Value), nf(m.MPPTA.PowerW.OK, m.MPPTA.PowerW.Value),
		nf(m.MPPTB.VoltageV.OK, m.MPPTB.VoltageV.Value), nf(m.MPPTB.CurrentA.OK, m.MPPTB.CurrentA.Value), nf(m.MPPTB.PowerW.OK, m.MPPTB.PowerW.Value),
		nf(m.DCPowerW.OK, m.DCPowerW.Value),
		nf(m.InternalTempC.OK, m.InternalTempC.Value),
		nf(m.DailyYieldKWh.OK, m.DailyYieldKWh.Value), nf(m.TotalYieldKWh.OK, m.TotalYieldKWh.Value),
	)
	return err
}

func insertDse(tx *sql.Tx, ts int64, id string, stale bool, g *dse.Snapshot) error {
	var alarms any
	if len(g.Alarms) > 0 {
		if b, err := json.Marshal(dse.AlarmStrings(g.Alarms)); err == nil {
			alarms = string(b)
		}
	}
	_, err := tx.Exec(
		`INSERT INTO dse_log (ts, device, stale, running, rpm, hz,
		   oil_kpa, coolant_c, fuel_pct, batt_v, alt_v,
		   total_w, total_va, total_var, pf,
		   l1_w, l2_w, l3_w, l1_v, l2_v, l3_v, l1_a, l2_a, l3_a,
		   kwh, starts, run_h, alarms)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ts, id, b2i(stale), b2i(g.Running),
		nf(g.EngineRPM.OK, g.EngineRPM.Value), nf(g.FreqHz.OK, g.FreqHz.Value),
		nf(g.OilPressureKPa.OK, g.OilPressureKPa.Value), nf(g.CoolantTempC.OK, g.CoolantTempC.Value),
		nf(g.FuelLevelPct.OK, g.FuelLevelPct.Value), nf(g.BatteryV.OK, g.BatteryV.Value), nf(g.ChargeAltV.OK, g.ChargeAltV.Value),
		nf(g.TotalW.OK, g.TotalW.Value), nf(g.TotalVA.OK, g.TotalVA.Value), nf(g.TotalVAr.OK, g.TotalVAr.Value), nf(g.AvgPF.OK, g.AvgPF.Value),
		nf(g.WattsL[0].OK, g.WattsL[0].Value), nf(g.WattsL[1].OK, g.WattsL[1].Value), nf(g.WattsL[2].OK, g.WattsL[2].Value),
		nf(g.VLn[0].OK, g.VLn[0].Value), nf(g.VLn[1].OK, g.VLn[1].Value), nf(g.VLn[2].OK, g.VLn[2].Value),
		nf(g.AmpsL[0].OK, g.AmpsL[0].Value), nf(g.AmpsL[1].OK, g.AmpsL[1].Value), nf(g.AmpsL[2].OK, g.AmpsL[2].Value),
		nf(g.PosKWh.OK, g.PosKWh.Value), nf(g.Starts.OK, g.Starts.Value), nf(g.RunHours.OK, g.RunHours.Value),
		alarms,
	)
	return err
}

// InsertBattery writes one detailed battery tick: the aggregate row plus one
// row per enumerated string. The master string's row carries the FULL cell
// voltage array (all ~224 cells) and per-module voltage/temperature arrays as
// JSON — the granular record requested for cell-level analysis.
func (s *Store) InsertBattery(snap *bms.SystemSnapshot) error {
	if snap == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ts := time.Now().UnixMilli()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	a := snap.Aggregate
	if _, err := tx.Exec(
		`INSERT INTO bms_log (ts, piles, total_v, current_a, power_kw, soc, soh, temp_c,
		   cell_max_v, cell_min_v, combiner, relay_closed, link_live)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ts, a.Piles, a.TotalV, a.Current, a.PowerKW, a.SOC, a.SOH, a.Temp,
		a.CellMaxV, a.CellMinV, snap.Combiner.State, b2i(snap.Combiner.RelayClosed), b2i(snap.LinkLive),
	); err != nil {
		return err
	}

	for _, st := range snap.Strings {
		if st.Status != bms.StatusEnumerated {
			continue
		}
		alarms := ""
		if len(st.Alarms) > 0 {
			if b, err := json.Marshal(st.Alarms); err == nil {
				alarms = string(b)
			}
		}
		if _, err := tx.Exec(
			`INSERT INTO bms_string_log (ts, string, status, serial,
			   total_v, current_a, power_kw, soc, soh, soe, temp_c, cycles,
			   cell_max_v, cell_min_v, spread_mv, cell_max_t, cell_min_t,
			   mod_max_v, mod_min_v, basic, protection, alarms,
			   cell_v, module_v, module_t)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			ts, st.Index, st.Status, st.Serial,
			st.TotalV, st.Current, st.PowerKW, st.SOC, st.SOH, st.SOE, st.Temp, st.Cycles,
			st.CellMaxV, st.CellMinV, st.CellSpreadMV, st.CellMaxT, st.CellMinT,
			st.ModMaxV, st.ModMinV, st.BasicStatus, st.ProtectionText, alarms,
			jarr(st.CellV), jarr(st.ModuleV), jarr(st.ModuleT),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// InsertEvents persists timeline events (called from the Monitor's event sink,
// so every state transition lands in the DB exactly once).
func (s *Store) InsertEvents(events []plant.Event) error {
	if len(events) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, e := range events {
		ts := time.Now().UnixMilli()
		if t, err := time.Parse(time.RFC3339, e.At); err == nil {
			ts = t.UnixMilli()
		}
		if _, err := tx.Exec(
			`INSERT INTO event_log (ts, device_id, device, severity, text) VALUES (?,?,?,?,?)`,
			ts, e.DeviceID, e.Device, e.Severity, e.Text,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Prune deletes rows older than the retention window from every table.
func (s *Store) Prune(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().AddDate(0, 0, -retentionDays).UnixMilli()
	for _, table := range []string{"plant_log", "oztek_log", "sma_log", "dse_log", "bms_log", "bms_string_log", "event_log"} {
		if _, err := s.db.Exec(`DELETE FROM `+table+` WHERE ts < ?`, cutoff); err != nil {
			return err
		}
	}
	return nil
}

// Stats summarizes the datastore for the UI.
type Stats struct {
	Path      string           `json:"path"`
	SizeBytes int64            `json:"sizeBytes"`
	Rows      map[string]int64 `json:"rows"`
	OldestTS  int64            `json:"oldestTs"` // unix ms; 0 = empty
	NewestTS  int64            `json:"newestTs"`
}

// GetStats returns row counts, file size, and the covered time range.
func (s *Store) GetStats() (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Stats{Path: s.path, Rows: map[string]int64{}}
	if fi, err := os.Stat(s.path); err == nil {
		st.SizeBytes = fi.Size()
	}
	for _, table := range []string{"plant_log", "oztek_log", "sma_log", "dse_log", "bms_log", "bms_string_log", "event_log"} {
		var n int64
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
			return st, err
		}
		st.Rows[table] = n
	}
	var oldest, newest sql.NullInt64
	_ = s.db.QueryRow(`SELECT MIN(ts), MAX(ts) FROM plant_log`).Scan(&oldest, &newest)
	var bo, bn sql.NullInt64
	_ = s.db.QueryRow(`SELECT MIN(ts), MAX(ts) FROM bms_log`).Scan(&bo, &bn)
	st.OldestTS = minNonZero(oldest.Int64, bo.Int64)
	if newest.Int64 > bn.Int64 {
		st.NewestTS = newest.Int64
	} else {
		st.NewestTS = bn.Int64
	}
	return st, nil
}

func minNonZero(a, b int64) int64 {
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}
