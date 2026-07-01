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
