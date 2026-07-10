// Package config persists the last-used connection settings (no database).
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"bms-monitor/internal/bms"
)

func file() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	d := filepath.Join(dir, "bms-monitor")
	_ = os.MkdirAll(d, 0o755)
	return filepath.Join(d, "config.json")
}

// WithDefaults fills sensible defaults for the TEC site.
func WithDefaults(c bms.Config) bms.Config {
	if c.IP == "" {
		c.IP = "192.168.0.31"
	}
	if c.Port == 0 {
		c.Port = 502
	}
	if c.Unit == 0 {
		c.Unit = 1
	}
	if c.Timeout == 0 {
		c.Timeout = 3
	}
	return c
}

// Load returns the saved config (with defaults applied).
func Load() bms.Config {
	var c bms.Config
	if b, err := os.ReadFile(file()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	return WithDefaults(c)
}

// Save persists the config.
func Save(c bms.Config) error {
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(file(), b, 0o644)
}

// Logging configures the local SQLite datastore that records plant + battery
// telemetry over time.
type Logging struct {
	Enabled       bool `json:"enabled"`
	IntervalSec   int  `json:"intervalSec"`   // logging cadence; default 30
	RetentionDays int  `json:"retentionDays"` // prune window; default 90
}

// WithLoggingDefaults fills defaults (enabled, 30 s, 90 days).
func WithLoggingDefaults(l Logging) Logging {
	if l.IntervalSec < 5 {
		l.IntervalSec = 30
	}
	if l.RetentionDays <= 0 {
		l.RetentionDays = 90
	}
	return l
}

func loggingFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	d := filepath.Join(dir, "bms-monitor")
	_ = os.MkdirAll(d, 0o755)
	return filepath.Join(d, "logging.json")
}

// LoadLogging returns saved logging settings (default: enabled at 30 s).
func LoadLogging() Logging {
	l := Logging{Enabled: true}
	if b, err := os.ReadFile(loggingFile()); err == nil {
		_ = json.Unmarshal(b, &l)
	}
	return WithLoggingDefaults(l)
}

// SaveLogging persists logging settings.
func SaveLogging(l Logging) error {
	b, _ := json.MarshalIndent(WithLoggingDefaults(l), "", "  ")
	return os.WriteFile(loggingFile(), b, 0o644)
}

// Polling controls whether the app will act as a Modbus master on a bus that
// ARC also masters (the shared RS485 OzTek converter). Default is OFF — the
// app never touches an ARC-controlled shared bus unless the operator opts in,
// so it can never contend with ARC's live control of the inverters.
type Polling struct {
	PollSharedBus bool `json:"pollSharedBus"` // default false
}

func pollingFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	d := filepath.Join(dir, "bms-monitor")
	_ = os.MkdirAll(d, 0o755)
	return filepath.Join(d, "polling.json")
}

// LoadPolling returns saved polling policy (default: shared bus OFF).
func LoadPolling() Polling {
	var p Polling // zero value: PollSharedBus=false
	if b, err := os.ReadFile(pollingFile()); err == nil {
		_ = json.Unmarshal(b, &p)
	}
	return p
}

// SavePolling persists the polling policy.
func SavePolling(p Polling) error {
	b, _ := json.MarshalIndent(p, "", "  ")
	return os.WriteFile(pollingFile(), b, 0o644)
}

// Remote holds the Solarman remote-session settings (stored locally, like a
// saved password). The token is a long-lived Bearer JWT.
type Remote struct {
	Token     string `json:"token"`
	StationID int64  `json:"stationId"`
}

func remoteFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	d := filepath.Join(dir, "bms-monitor")
	_ = os.MkdirAll(d, 0o755)
	return filepath.Join(d, "remote.json")
}

// LoadRemote returns saved remote settings (defaults to the Totota Minigrid station).
func LoadRemote() Remote {
	r := Remote{}
	if b, err := os.ReadFile(remoteFile()); err == nil {
		_ = json.Unmarshal(b, &r)
	}
	if r.StationID == 0 {
		r.StationID = 66280946
	}
	return r
}

// SaveRemote persists remote settings (0600 — contains a session token).
func SaveRemote(r Remote) error {
	b, _ := json.MarshalIndent(r, "", "  ")
	return os.WriteFile(remoteFile(), b, 0o600)
}
