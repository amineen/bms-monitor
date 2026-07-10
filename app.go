package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/config"
	"bms-monitor/internal/datastore"
	"bms-monitor/internal/plant"
	"bms-monitor/internal/solarman"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application struct. Its exported methods are callable
// from the React frontend via generated, typed bindings. All device access is
// read-only (the single exception is the gated Pylontech Run below).
type App struct {
	ctx     context.Context
	monitor *plant.Monitor    // plant poll state (stale cache + event timeline)
	store   *datastore.Store  // SQLite time-series log
	logger  *datastore.Logger // background logging service
}

func NewApp() *App { return &App{monitor: plant.NewMonitor()} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Open the local datastore and start the background logger. A datastore
	// failure must never block the monitor UI — log-less operation is fine.
	if store, err := datastore.Open(datastore.DefaultPath()); err == nil {
		a.store = store
		a.logger = datastore.NewLogger(store, a.monitor)
		// Persist every timeline event exactly once, as it is detected.
		a.monitor.SetEventSink(func(evts []plant.Event) { _ = store.InsertEvents(evts) })
		a.logger.Start(config.LoadLogging())
	}
}

func (a *App) shutdown(_ context.Context) {
	if a.logger != nil {
		a.logger.Stop()
	}
	if a.store != nil {
		_ = a.store.Close()
	}
}

// GetConfig returns the saved connection settings (with TEC defaults).
func (a *App) GetConfig() bms.Config { return config.Load() }

// SaveConfig persists connection settings.
func (a *App) SaveConfig(cfg bms.Config) error { return config.Save(config.WithDefaults(cfg)) }

// TestConnection verifies the gateway is reachable and identifies the BMS.
func (a *App) TestConnection(cfg bms.Config) (bms.Identity, error) {
	return bms.TestConnection(config.WithDefaults(cfg))
}

// ReadSystem reads the full system snapshot (identity + aggregate + 6 strings
// + master module/cell detail). Reads are sequential (fresh connection each).
func (a *App) ReadSystem(cfg bms.Config) (*bms.SystemSnapshot, error) {
	c := config.WithDefaults(cfg)
	snap, err := bms.ReadSystem(c)
	if err == nil {
		_ = config.Save(c)
		a.monitor.IngestBatteryEvents(snap) // per-string protection alarm events
		if a.logger != nil {
			a.logger.IngestBattery(snap) // reuse this read for the datastore tick
		}
	}
	return snap, err
}

// ---- Commissioning: combiner energization (the only register writes) ----

// EvaluateRunGate returns the Run preconditions for a snapshot (pure; no device
// I/O). The frontend uses it to show which gates pass before offering Run.
func (a *App) EvaluateRunGate(snap bms.SystemSnapshot, maxSpreadV float64) bms.RunGate {
	return bms.EvaluateRunGate(&snap, maxSpreadV)
}

// IssueRun performs the gated commissioning Run write (0x1094 = 0xAA), which
// closes the string relays and energizes the combiner bus. COMMISSIONING ONLY —
// this is the single register write in the app. Hard gates (all strings online,
// no active protections) always apply; force bypasses only the soft gates.
func (a *App) IssueRun(cfg bms.Config, maxSpreadV float64, autoWake, force bool) (*bms.RunResult, error) {
	return bms.IssueRun(config.WithDefaults(cfg), bms.RunOptions{
		MaxSpreadV: maxSpreadV, AutoWake: autoWake, Force: force,
	})
}

// ---- Plant-wide monitoring (read-only) ----

// GetDevices returns the plant device registry (user-edited if saved,
// otherwise the bundled Totota network list).
func (a *App) GetDevices() []plant.Device { return plant.LoadDevices() }

// SaveDevices persists registry edits (enable/disable, IPs, added devices).
func (a *App) SaveDevices(devs []plant.Device) error { return plant.SaveDevices(devs) }

// ResetDevices restores the bundled Totota registry.
func (a *App) ResetDevices() ([]plant.Device, error) {
	if err := plant.ResetDevices(); err != nil {
		return nil, err
	}
	return plant.LoadDevices(), nil
}

// ReadPlant performs one sequential, read-only round-robin over every enabled
// plant device (BMS, OzTeks, SMAs, genset) and returns the unified snapshot
// with health, insights, and the troubleshooting timeline. Reads to the
// shared .41 OzTek gateway are strictly serialized.
func (a *App) ReadPlant() (*plant.Snapshot, error) {
	snap := a.monitor.Read(plant.LoadDevices(), config.LoadPolling().PollSharedBus)
	if a.logger != nil {
		a.logger.IngestPlant(snap) // reuse this read for the datastore tick
	}
	return snap, nil
}

// GetPolling returns the polling policy (shared-bus master on/off).
func (a *App) GetPolling() config.Polling { return config.LoadPolling() }

// SetPolling persists the polling policy. Enabling PollSharedBus makes the app
// a second Modbus master on ARC's shared OzTek RS485 bus — an explicit,
// operator-authorized action (off by default). Every change is written to the
// event timeline/datastore as an audit entry.
func (a *App) SetPolling(p config.Polling) config.Polling {
	old := config.LoadPolling()
	_ = config.SavePolling(p)
	if old.PollSharedBus != p.PollSharedBus {
		if p.PollSharedBus {
			a.monitor.AddEvent("app", "Plant Monitor", plant.SevWarn,
				"OzTek shared-bus polling ENABLED by operator — app is now a second master on ARC's bus")
		} else {
			a.monitor.AddEvent("app", "Plant Monitor", plant.SevInfo,
				"OzTek shared-bus polling disabled — app no longer touches ARC's bus")
		}
	}
	return p
}

// ---- Datastore (SQLite telemetry log) ----

// GetLogging returns the logging configuration, service state, and DB stats.
func (a *App) GetLogging() datastore.Status {
	if a.logger == nil {
		return datastore.Status{Config: config.LoadLogging()}
	}
	return a.logger.Status()
}

// SetLogging applies + persists a new logging configuration (interval,
// retention, on/off) and returns the updated status.
func (a *App) SetLogging(cfg config.Logging) datastore.Status {
	cfg = config.WithLoggingDefaults(cfg)
	_ = config.SaveLogging(cfg)
	if a.logger != nil {
		a.logger.SetConfig(cfg)
	}
	return a.GetLogging()
}

// ExportLogs writes the logged history to an Excel workbook via a save
// dialog. scope: system | battery | oztek | pv | genset | events.
// sinceHours limits the range (0 = everything). Returns the saved path
// ("" if cancelled).
func (a *App) ExportLogs(scope string, sinceHours int) (string, error) {
	if a.store == nil {
		return "", fmt.Errorf("datastore is not available")
	}
	name := fmt.Sprintf("tec-%s-logs-%s.xlsx", scope, time.Now().Format("2006-01-02"))
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export logged history",
		DefaultFilename: name,
		Filters:         []runtime.FileFilter{{DisplayName: "Excel Workbook (*.xlsx)", Pattern: "*.xlsx"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := a.store.ExportXLSX(path, scope, sinceHours); err != nil {
		return "", err
	}
	return path, nil
}

// ---- Remote (Solarman) source ----

// GetRemoteConfig returns saved Solarman settings (token + station).
func (a *App) GetRemoteConfig() config.Remote { return config.LoadRemote() }

// OpenSolarmanLogin opens the Solarman portal login in the default browser so
// the user can sign in (past the captcha) and copy their session token.
func (a *App) OpenSolarmanLogin() {
	runtime.BrowserOpenURL(a.ctx, "https://home.solarmanpv.com/login")
}

// TestRemote verifies a Solarman bearer token works.
func (a *App) TestRemote(token string) error {
	return solarman.New(token).TestToken()
}

// ConnectSolarman launches a browser at the Solarman login, waits for the user
// to sign in (solving the captcha), captures the session token, saves it, and
// returns it. Blocks until login completes or times out.
func (a *App) ConnectSolarman() (string, error) {
	tok, err := solarman.CaptureToken(5 * time.Minute)
	if err != nil {
		return "", err
	}
	r := config.LoadRemote()
	r.Token = tok
	_ = config.SaveRemote(r)
	return tok, nil
}

// ClearSolarmanSession forgets the saved Solarman token so the next Connect
// starts a fresh login (the browser it launches already uses a clean profile).
func (a *App) ClearSolarmanSession() error {
	r := config.LoadRemote()
	r.Token = ""
	return config.SaveRemote(r)
}

// ToggleMaximise maximises the window to fill the screen (or restores it). On
// macOS this is a reliable alternative to the green traffic-light zoom button.
func (a *App) ToggleMaximise() { runtime.WindowToggleMaximise(a.ctx) }

// GatewayReachable reports whether the on-site BMS gateway is reachable (TCP).
// Used to auto-select On-site vs Remote at startup.
func (a *App) GatewayReachable(cfg bms.Config) bool {
	c := config.WithDefaults(cfg)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.IP, c.Port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ReadRemote reads the full system snapshot from Solarman for a station.
// Returns the same SystemSnapshot type as the on-site read.
func (a *App) ReadRemote(token string, stationID int64) (*bms.SystemSnapshot, error) {
	snap, err := solarman.New(token).ReadSystem(stationID)
	if err == nil {
		_ = config.SaveRemote(config.Remote{Token: token, StationID: stationID})
	}
	return snap, err
}

// ExportExcel re-reads live registers and writes the bms_measurements-format
// workbook to a user-chosen path. Returns the saved path ("" if cancelled).
func (a *App) ExportExcel(cfg bms.Config) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Excel report",
		DefaultFilename: "bms_readings.xlsx",
		Filters:         []runtime.FileFilter{{DisplayName: "Excel Workbook (*.xlsx)", Pattern: "*.xlsx"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := bms.ExportMeasurementsXLSX(config.WithDefaults(cfg), path); err != nil {
		return "", err
	}
	return path, nil
}

// ExportPDF writes a one-page summary PDF of the given snapshot.
func (a *App) ExportPDF(snap bms.SystemSnapshot) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save PDF report",
		DefaultFilename: "bms_report.pdf",
		Filters:         []runtime.FileFilter{{DisplayName: "PDF Document (*.pdf)", Pattern: "*.pdf"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := bms.ExportPDFReport(snap, path); err != nil {
		return "", err
	}
	return path, nil
}
