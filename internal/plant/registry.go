// Package plant orchestrates the plant-wide, read-only round-robin poll over
// every device on the ARC network: the Pylontech BMS, the OzTek battery PCSs,
// the SMA PV inverters, and the DSE genset controller. It assembles a unified
// Snapshot, derives per-device and cross-device health, and diffs consecutive
// snapshots into a troubleshooting timeline.
package plant

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
)

// Device kinds.
const (
	KindBMS   = "bms"
	KindOzTek = "oztek"
	KindSMA   = "sma"
	KindDSE   = "dse"
)

// Device is one registry entry (seeded from the bundled Totota network list,
// editable and persisted per-user).
type Device struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Unit    int    `json:"unit"`
	Enabled bool   `json:"enabled"`
	Notes   string `json:"notes"`

	// SharedBus marks a device that sits on an RS485 bus ARC also masters (the
	// OzTek converter). The app will NOT poll it — i.e. never become a second
	// master on ARC's control bus — unless polling is explicitly enabled
	// (config.Polling.PollSharedBus). Off by default.
	SharedBus bool `json:"sharedBus"`
}

type registryFile struct {
	Site    string   `json:"site"`
	Network string   `json:"network"`
	Updated string   `json:"updated"`
	Devices []Device `json:"devices"`
}

//go:embed devices.json
var embeddedDevices []byte

// DefaultDevices returns the bundled Totota registry.
func DefaultDevices() []Device {
	var rf registryFile
	if err := json.Unmarshal(embeddedDevices, &rf); err != nil {
		return nil
	}
	return withSafetyOverlay(rf.Devices)
}

// withSafetyOverlay enforces invariants that must hold NO MATTER what a
// registry file says — the ARC-safety guard is structural, not data-driven.
// Every OzTek sits on the RS485 bus that ARC masters to control the
// inverters, so it is always SharedBus: a saved registry from an older build
// (before the flag existed) or a hand-edited file can never silently re-arm
// polling on ARC's bus.
func withSafetyOverlay(devs []Device) []Device {
	for i := range devs {
		if devs[i].Kind == KindOzTek {
			devs[i].SharedBus = true
		}
	}
	return devs
}

func devicesFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	d := filepath.Join(dir, "bms-monitor")
	_ = os.MkdirAll(d, 0o755)
	return filepath.Join(d, "devices.json")
}

// LoadDevices returns the user-edited registry when one has been saved,
// otherwise the bundled default — so the tech plugs in and it just works.
// The safety overlay applies to BOTH sources.
func LoadDevices() []Device {
	if b, err := os.ReadFile(devicesFile()); err == nil {
		var devs []Device
		if json.Unmarshal(b, &devs) == nil && len(devs) > 0 {
			return withSafetyOverlay(devs)
		}
	}
	return DefaultDevices()
}

// SaveDevices persists the registry.
func SaveDevices(devs []Device) error {
	b, _ := json.MarshalIndent(devs, "", "  ")
	return os.WriteFile(devicesFile(), b, 0o644)
}

// ResetDevices deletes the user registry so LoadDevices falls back to the
// bundled default.
func ResetDevices() error {
	err := os.Remove(devicesFile())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
