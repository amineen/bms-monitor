package bms

import "time"

// Summary is the light, single-read view of the BMS used by the plant
// round-robin poll. The full ReadSystem walk (identity + 6 strings + master
// arrays, ~8+ connections) stays the battery-view read; the plant view only
// needs the aggregate block: one fresh connection, one block.
type AggSummary struct {
	OK        bool           `json:"ok"`
	Timestamp string         `json:"timestamp"`
	Piles     int            `json:"piles"` // strings enumerated on the chain
	TotalV    float64        `json:"totalV"`
	Current   float64        `json:"current"`
	PowerKW   float64        `json:"powerKW"`
	SOC       int            `json:"soc"`
	SOH       int            `json:"soh"`
	Temp      float64        `json:"temp"`
	CellMaxV  float64        `json:"cellMaxV"`
	CellMinV  float64        `json:"cellMinV"`
	Combiner  CombinerStatus `json:"combiner"`
}

// ReadSummary reads only the aggregate block (0x1100) — one fresh connection.
func ReadSummary(cfg Config) (*AggSummary, error) {
	agg, err := readBaseFresh(cfg, AggBase, 0x52)
	if err != nil {
		return nil, err
	}
	tv := round(float64(at(agg, 0x03))*0.1, 1)
	cur := round(float64(S32(at(agg, 0x04), at(agg, 0x05)))*0.01, 2)
	return &AggSummary{
		OK:        true,
		Timestamp: time.Now().Format(time.RFC3339),
		Piles:     int(at(agg, 0x31)),
		TotalV:    tv,
		Current:   cur,
		PowerKW:   round(tv*cur/1000, 3),
		SOC:       int(at(agg, 0x07)),
		SOH:       int(at(agg, 0x20)),
		Temp:      round(float64(S16(at(agg, 0x06)))*0.1, 1),
		CellMaxV:  round(float64(at(agg, 0x10))*0.001, 3),
		CellMinV:  round(float64(at(agg, 0x11))*0.001, 3),
		Combiner:  buildCombiner(at(agg, AggSysOpOff), at(agg, AggSwitchOff), tv),
	}, nil
}
