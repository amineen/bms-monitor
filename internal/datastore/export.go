package datastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Export scopes.
const (
	ScopeSystem  = "system"  // everything: plant + all equipment + events
	ScopeBattery = "battery" // aggregate + strings + expanded cells/modules
	ScopeOzTek   = "oztek"
	ScopePV      = "pv"
	ScopeGenset  = "genset"
	ScopeEvents  = "events"
)

// ExportXLSX writes the logged history for a scope to an Excel workbook.
// sinceHours limits the range (0 = everything in the store). Every sheet is
// written with excelize's StreamWriter, so even a full 90-day export stays
// memory-flat.
func (s *Store) ExportXLSX(path, scope string, sinceHours int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	from := int64(0)
	if sinceHours > 0 {
		from = time.Now().Add(-time.Duration(sinceHours) * time.Hour).UnixMilli()
	}

	f := excelize.NewFile()
	defer f.Close()

	type sheet struct {
		name  string
		write func(sw *excelize.StreamWriter) error
	}
	var sheets []sheet
	table := func(name, tbl, deviceFilter string) sheet {
		return sheet{name, func(sw *excelize.StreamWriter) error { return s.dumpTable(sw, tbl, deviceFilter, from) }}
	}
	events := func(idsLike ...string) sheet {
		return sheet{"Events", func(sw *excelize.StreamWriter) error { return s.dumpEvents(sw, from, idsLike) }}
	}

	switch scope {
	case ScopeBattery:
		sheets = []sheet{
			table("Battery", "bms_log", ""),
			table("Battery Strings", "bms_string_log", ""),
			{"Battery Cells", func(sw *excelize.StreamWriter) error { return s.dumpCells(sw, from) }},
			{"Battery Modules", func(sw *excelize.StreamWriter) error { return s.dumpModules(sw, from) }},
			events("bms%"),
		}
	case ScopeOzTek:
		sheets = []sheet{table("OzTek", "oztek_log", ""), events("oztek%")}
	case ScopePV:
		sheets = []sheet{table("Solar PV", "sma_log", ""), events("pv%")}
	case ScopeGenset:
		sheets = []sheet{table("Genset", "dse_log", ""), events("gen%")}
	case ScopeEvents:
		sheets = []sheet{events()}
	default: // ScopeSystem
		sheets = []sheet{
			table("Plant", "plant_log", ""),
			table("Battery", "bms_log", ""),
			table("Battery Strings", "bms_string_log", ""),
			{"Battery Cells", func(sw *excelize.StreamWriter) error { return s.dumpCells(sw, from) }},
			{"Battery Modules", func(sw *excelize.StreamWriter) error { return s.dumpModules(sw, from) }},
			table("OzTek", "oztek_log", ""),
			table("Solar PV", "sma_log", ""),
			table("Genset", "dse_log", ""),
			events(),
		}
	}

	for i, sh := range sheets {
		if i == 0 {
			// Rename the default sheet instead of leaving an empty "Sheet1".
			if err := f.SetSheetName("Sheet1", sh.name); err != nil {
				return err
			}
		} else if _, err := f.NewSheet(sh.name); err != nil {
			return err
		}
		sw, err := f.NewStreamWriter(sh.name)
		if err != nil {
			return err
		}
		if err := sh.write(sw); err != nil {
			return err
		}
		if err := sw.Flush(); err != nil {
			return err
		}
	}
	return f.SaveAs(path)
}

func cellRef(col, row int) string {
	ref, _ := excelize.CoordinatesToCellName(col, row)
	return ref
}

func fmtTS(ms int64) string { return time.UnixMilli(ms).Format("2006-01-02 15:04:05") }

// dumpTable streams a whole table generically: a human-readable Time column,
// then every DB column (ts kept as raw epoch ms for tooling).
func (s *Store) dumpTable(sw *excelize.StreamWriter, tbl, deviceFilter string, from int64) error {
	q := `SELECT * FROM ` + tbl + ` WHERE ts >= ?`
	args := []any{from}
	if deviceFilter != "" {
		q += ` AND device LIKE ?`
		args = append(args, deviceFilter)
	}
	q += ` ORDER BY ts`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	header := make([]any, 0, len(cols)+1)
	header = append(header, "Time")
	for _, c := range cols {
		header = append(header, c)
	}
	if err := sw.SetRow("A1", header); err != nil {
		return err
	}

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	rowN := 2
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		out := make([]any, 0, len(cols)+1)
		var ts int64
		if v, ok := vals[0].(int64); ok {
			ts = v
		}
		out = append(out, fmtTS(ts))
		for _, v := range vals {
			switch x := v.(type) {
			case []byte:
				out = append(out, string(x))
			default:
				out = append(out, x)
			}
		}
		if err := sw.SetRow(cellRef(1, rowN), out); err != nil {
			return err
		}
		rowN++
	}
	return rows.Err()
}

func (s *Store) dumpEvents(sw *excelize.StreamWriter, from int64, idsLike []string) error {
	q := `SELECT ts, device, severity, text FROM event_log WHERE ts >= ?`
	args := []any{from}
	if len(idsLike) > 0 {
		parts := make([]string, len(idsLike))
		for i, p := range idsLike {
			parts[i] = `device_id LIKE ?`
			args = append(args, p)
		}
		q += ` AND (` + strings.Join(parts, " OR ") + `)`
	}
	q += ` ORDER BY ts`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if err := sw.SetRow("A1", []any{"Time", "Device", "Severity", "Event"}); err != nil {
		return err
	}
	rowN := 2
	for rows.Next() {
		var ts int64
		var device, sev, text string
		if err := rows.Scan(&ts, &device, &sev, &text); err != nil {
			return err
		}
		if err := sw.SetRow(cellRef(1, rowN), []any{fmtTS(ts), device, sev, text}); err != nil {
			return err
		}
		rowN++
	}
	return rows.Err()
}

// dumpCells expands the master string's JSON cell arrays into one column per
// cell — the granular battery history, ready for conditional formatting or a
// per-cell chart in Excel.
func (s *Store) dumpCells(sw *excelize.StreamWriter, from int64) error {
	return s.dumpArray(sw, from, "cell_v", func(n int) []any {
		h := []any{"Time", "String"}
		for i := 1; i <= n; i++ {
			h = append(h, fmt.Sprintf("Cell %d (V)", i))
		}
		return h
	})
}

// dumpModules expands module voltage + temperature arrays side by side.
func (s *Store) dumpModules(sw *excelize.StreamWriter, from int64) error {
	var maxN sql.NullInt64
	if err := s.db.QueryRow(
		`SELECT MAX(json_array_length(module_v)) FROM bms_string_log WHERE ts >= ? AND module_v IS NOT NULL`, from,
	).Scan(&maxN); err != nil {
		return err
	}
	n := int(maxN.Int64)
	header := []any{"Time", "String"}
	for i := 1; i <= n; i++ {
		header = append(header, fmt.Sprintf("M%d V", i))
	}
	for i := 1; i <= n; i++ {
		header = append(header, fmt.Sprintf("M%d °C", i))
	}
	if err := sw.SetRow("A1", header); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}

	rows, err := s.db.Query(
		`SELECT ts, string, module_v, module_t FROM bms_string_log
		 WHERE ts >= ? AND module_v IS NOT NULL ORDER BY ts, string`, from)
	if err != nil {
		return err
	}
	defer rows.Close()

	rowN := 2
	for rows.Next() {
		var ts int64
		var idx int
		var mv, mt sql.NullString
		if err := rows.Scan(&ts, &idx, &mv, &mt); err != nil {
			return err
		}
		out := []any{fmtTS(ts), idx}
		out = appendJSONFloats(out, mv.String, n)
		out = appendJSONFloats(out, mt.String, n)
		if err := sw.SetRow(cellRef(1, rowN), out); err != nil {
			return err
		}
		rowN++
	}
	return rows.Err()
}

func (s *Store) dumpArray(sw *excelize.StreamWriter, from int64, col string, header func(n int) []any) error {
	var maxN sql.NullInt64
	if err := s.db.QueryRow(
		`SELECT MAX(json_array_length(`+col+`)) FROM bms_string_log WHERE ts >= ? AND `+col+` IS NOT NULL`, from,
	).Scan(&maxN); err != nil {
		return err
	}
	n := int(maxN.Int64)
	if err := sw.SetRow("A1", header(n)); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}

	rows, err := s.db.Query(
		`SELECT ts, string, `+col+` FROM bms_string_log
		 WHERE ts >= ? AND `+col+` IS NOT NULL ORDER BY ts, string`, from)
	if err != nil {
		return err
	}
	defer rows.Close()

	rowN := 2
	for rows.Next() {
		var ts int64
		var idx int
		var arr sql.NullString
		if err := rows.Scan(&ts, &idx, &arr); err != nil {
			return err
		}
		out := []any{fmtTS(ts), idx}
		out = appendJSONFloats(out, arr.String, n)
		if err := sw.SetRow(cellRef(1, rowN), out); err != nil {
			return err
		}
		rowN++
	}
	return rows.Err()
}

// appendJSONFloats decodes a JSON float array and appends exactly n values
// (padding with nil so ragged rows stay column-aligned).
func appendJSONFloats(out []any, jsonStr string, n int) []any {
	var vals []float64
	_ = json.Unmarshal([]byte(jsonStr), &vals)
	for i := 0; i < n; i++ {
		if i < len(vals) {
			out = append(out, vals[i])
		} else {
			out = append(out, nil)
		}
	}
	return out
}
