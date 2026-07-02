package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/config"
	"bms-monitor/internal/solarman"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application struct. Its exported methods are callable
// from the React frontend via generated, typed bindings. All BMS access is
// read-only.
type App struct {
	ctx context.Context
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

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
