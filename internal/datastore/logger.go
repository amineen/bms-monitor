package datastore

import (
	"sync"
	"time"

	"bms-monitor/internal/bms"
	"bms-monitor/internal/config"
	"bms-monitor/internal/modbustcp"
	"bms-monitor/internal/plant"
)

// Logger is the background logging service. It ticks at the configured
// interval (default 30 s) and records a plant round-robin plus a detailed
// battery walk (per-string + per-cell arrays).
//
// Efficiency: reads the UI already performed are INGESTED instead of
// re-polled — App.ReadPlant / App.ReadSystem push their fresh snapshots here,
// which stamps the stream's clock. The ticker only polls a stream itself when
// nothing has fed it within the interval. With the UI auto-refreshing at the
// same cadence, the logger adds zero extra Modbus traffic.
type Logger struct {
	store   *Store
	monitor *plant.Monitor

	mu        sync.Mutex
	cfg       config.Logging
	lastPlant time.Time
	lastBatt  time.Time
	lastPrune time.Time
	stopCh    chan struct{}
	started   bool
}

// NewLogger builds the service around an open store and the shared plant
// monitor (sharing the monitor keeps ONE event timeline and stale-cache).
func NewLogger(store *Store, monitor *plant.Monitor) *Logger {
	return &Logger{store: store, monitor: monitor}
}

// Start launches the background ticker (idempotent).
func (l *Logger) Start(cfg config.Logging) {
	l.mu.Lock()
	l.cfg = config.WithLoggingDefaults(cfg)
	if l.started {
		l.mu.Unlock()
		return
	}
	l.started = true
	l.stopCh = make(chan struct{})
	l.mu.Unlock()

	go l.run()
}

// Stop halts the ticker (safe to call multiple times).
func (l *Logger) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		close(l.stopCh)
		l.started = false
	}
}

// SetConfig applies a new logging configuration live.
func (l *Logger) SetConfig(cfg config.Logging) {
	l.mu.Lock()
	l.cfg = config.WithLoggingDefaults(cfg)
	l.mu.Unlock()
}

func (l *Logger) config() config.Logging {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cfg
}

// grace: an ingest/poll counts as "fresh" for the stream if it happened
// within interval minus this fraction — so a 30 s UI auto-refresh satisfies a
// 30 s log interval even when the ticks don't align exactly.
func due(last time.Time, interval time.Duration) bool {
	return time.Since(last) >= interval-interval/5
}

// IngestPlant records a plant snapshot the UI just read (no extra polling).
func (l *Logger) IngestPlant(snap *plant.Snapshot) {
	if snap == nil {
		return
	}
	cfg := l.config()
	if !cfg.Enabled {
		return
	}
	interval := time.Duration(cfg.IntervalSec) * time.Second
	l.mu.Lock()
	if !due(l.lastPlant, interval) {
		l.mu.Unlock()
		return
	}
	l.lastPlant = time.Now()
	l.mu.Unlock()
	_ = l.store.InsertPlant(snap)
}

// IngestBattery records a full battery snapshot the UI just read.
func (l *Logger) IngestBattery(snap *bms.SystemSnapshot) {
	if snap == nil {
		return
	}
	cfg := l.config()
	if !cfg.Enabled {
		return
	}
	interval := time.Duration(cfg.IntervalSec) * time.Second
	l.mu.Lock()
	if !due(l.lastBatt, interval) {
		l.mu.Unlock()
		return
	}
	l.lastBatt = time.Now()
	l.mu.Unlock()
	_ = l.store.InsertBattery(snap)
}

func (l *Logger) run() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case <-t.C:
			l.tick()
		}
	}
}

func (l *Logger) tick() {
	cfg := l.config()
	if !cfg.Enabled {
		return
	}
	interval := time.Duration(cfg.IntervalSec) * time.Second

	// Plant round-robin (the same serialized, read-only walk the UI does).
	l.mu.Lock()
	plantDue := due(l.lastPlant, interval)
	if plantDue {
		l.lastPlant = time.Now() // stamp first so a slow walk can't double-fire
	}
	l.mu.Unlock()
	var devices []plant.Device
	if plantDue {
		devices = plant.LoadDevices()
		snap := l.monitor.Read(devices, config.LoadPolling().PollSharedBus)
		_ = l.store.InsertPlant(snap)
	}

	// Detailed battery walk (aggregate + 6 strings + master cell/module arrays).
	l.mu.Lock()
	battDue := due(l.lastBatt, interval)
	if battDue {
		l.lastBatt = time.Now()
	}
	l.mu.Unlock()
	if battDue {
		if devices == nil {
			devices = plant.LoadDevices()
		}
		if bmsEnabled(devices) {
			c := config.Load()
			ep := modbustcp.Endpoint{Host: c.IP, Port: c.Port, Unit: c.Unit}
			if modbustcp.Reachable(ep, 800*time.Millisecond) {
				if snap, err := bms.ReadSystem(c); err == nil {
					l.monitor.IngestBatteryEvents(snap) // protection alarm events
					_ = l.store.InsertBattery(snap)
				}
			}
		}
	}

	// Housekeeping: prune old rows once a day.
	l.mu.Lock()
	pruneDue := time.Since(l.lastPrune) > 24*time.Hour
	if pruneDue {
		l.lastPrune = time.Now()
	}
	l.mu.Unlock()
	if pruneDue {
		_ = l.store.Prune(cfg.RetentionDays)
	}
}

func bmsEnabled(devices []plant.Device) bool {
	for _, d := range devices {
		if d.Kind == plant.KindBMS && d.Enabled {
			return true
		}
	}
	return false
}

// Status is the logging state surfaced to the UI.
type Status struct {
	Config     config.Logging `json:"config"`
	Running    bool           `json:"running"`
	LastPlant  string         `json:"lastPlant"` // RFC3339, "" = never
	LastBatt   string         `json:"lastBatt"`
	Stats      Stats          `json:"stats"`
}

// Status reports the current logging state plus datastore stats.
func (l *Logger) Status() Status {
	l.mu.Lock()
	st := Status{Config: l.cfg, Running: l.started && l.cfg.Enabled}
	if !l.lastPlant.IsZero() {
		st.LastPlant = l.lastPlant.Format(time.RFC3339)
	}
	if !l.lastBatt.IsZero() {
		st.LastBatt = l.lastBatt.Format(time.RFC3339)
	}
	l.mu.Unlock()
	if s, err := l.store.GetStats(); err == nil {
		st.Stats = s
	}
	return st
}
