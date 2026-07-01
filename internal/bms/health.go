package bms

import "strings"

// Health levels (also used as a sort key: good < warn < critical < offline).
const (
	HealthGood     = "good"
	HealthWarn     = "warn"
	HealthCritical = "critical"
	HealthOffline  = "offline"
)

// Thresholds (seeded from known protection setpoints + sensible balance limits).
const (
	spreadWarnMV = 50.0  // cell imbalance to watch
	spreadCritMV = 100.0 // cells out of balance
	tempWarnC    = 45.0
	tempCritC    = 55.0
	sohWarnPct   = 80
)

// HealthStatus is the plain-language traffic-light shown to technicians.
type HealthStatus struct {
	Level    string   `json:"level"`
	Headline string   `json:"headline"`
	Reasons  []string `json:"reasons"`
}

var healthRank = map[string]int{HealthGood: 0, HealthWarn: 1, HealthCritical: 2, HealthOffline: 3}

func worse(a, b string) string {
	if healthRank[b] > healthRank[a] {
		return b
	}
	return a
}

// assessHealth derives a plain-language status from a snapshot.
func assessHealth(snap *SystemSnapshot) HealthStatus {
	level := HealthGood
	var reasons []string
	bump := func(l string) { level = worse(level, l) }

	online := snap.Chain.Online
	switch {
	case online == 0:
		bump(HealthCritical)
		reasons = append(reasons, "No strings online — system in standby or not communicating.")
	case online < 6:
		bump(HealthWarn)
		reasons = append(reasons, plural(6-online, "string")+" not online (chain stops after string "+itoa(snap.Chain.StopsAfter)+").")
	}

	for _, s := range snap.Strings {
		if s.Status == StatusNoResponse {
			bump(HealthWarn)
			reasons = append(reasons, "String "+itoa(s.Index)+" not responding.")
			continue
		}
		if s.Status != StatusEnumerated {
			continue
		}
		if len(s.Alarms) > 0 {
			bump(HealthCritical)
			reasons = append(reasons, "String "+itoa(s.Index)+": "+strings.Join(s.Alarms, ", ")+".")
		}
		switch {
		case s.CellSpreadMV >= spreadCritMV:
			bump(HealthCritical)
			reasons = append(reasons, "String "+itoa(s.Index)+" cells out of balance ("+itoa(int(s.CellSpreadMV))+" mV).")
		case s.CellSpreadMV >= spreadWarnMV:
			bump(HealthWarn)
			reasons = append(reasons, "String "+itoa(s.Index)+" cell imbalance ("+itoa(int(s.CellSpreadMV))+" mV).")
		}
		switch {
		case s.Temp >= tempCritC || s.CellMaxT >= tempCritC:
			bump(HealthCritical)
			reasons = append(reasons, "String "+itoa(s.Index)+" high temperature.")
		case s.Temp >= tempWarnC:
			bump(HealthWarn)
			reasons = append(reasons, "String "+itoa(s.Index)+" elevated temperature.")
		}
		if s.SOH > 0 && s.SOH < sohWarnPct {
			bump(HealthWarn)
			reasons = append(reasons, "String "+itoa(s.Index)+" SOH "+itoa(s.SOH)+"%.")
		}
	}

	// Field-friendly headlines (ASCII only — also used in the PDF report).
	headline := "Attention needed"
	switch level {
	case HealthGood:
		headline = "System healthy - all 6 strings online"
	case HealthWarn:
		if online < 6 {
			headline = "System online - " + plural(6-online, "string") + " offline"
		} else {
			headline = "System online - check warnings"
		}
	case HealthCritical:
		if online == 0 {
			headline = "System offline - no strings online"
		} else {
			headline = "Attention needed - active protection"
		}
	}

	return HealthStatus{Level: level, Headline: headline, Reasons: reasons}
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return itoa(n) + " " + noun + "s"
}

// AssessHealth builds the plain-language health status from a snapshot. Exported
// so alternate data sources (e.g. the Solarman remote bridge) reuse the same logic.
func AssessHealth(snap *SystemSnapshot) HealthStatus { return assessHealth(snap) }

// BuildChain computes chain status from a set of strings (exported for reuse).
func BuildChain(strings []StringInfo) ChainStatus { return chainStatus(strings) }

// Round exposes the internal rounding helper for reuse.
func Round(v float64, dec int) float64 { return round(v, dec) }
