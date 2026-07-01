package bms

import (
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

var logHeaders = []interface{}{
	"Timestamp", "IP", "Port", "Unit ID", "Read Method", "Register", "Register Count",
	"Measurement", "Raw Registers Hex", "Raw Decimal", "Scale", "Value", "Unit", "Data Type", "Source",
}

func dtypeOf(fld Field) string {
	t := "U"
	if fld.Signed {
		t = "I"
	}
	if fld.Words == 2 {
		return t + "32"
	}
	return t + "16"
}

func summaryLogRows(ts, ip string, port, unit, base int, block []uint16, src string) [][]interface{} {
	var rows [][]interface{}
	for _, fld := range Summary {
		o := fld.Off
		var raw uint32
		var dec int64
		var rawHex string
		if fld.Words == 2 {
			raw = U32(at(block, o), at(block, o+1))
			if fld.Signed {
				dec = S32(at(block, o), at(block, o+1))
			} else {
				dec = int64(raw)
			}
			rawHex = fmt.Sprintf("0x%08X", raw)
		} else {
			w := at(block, o)
			raw = uint32(w)
			if fld.Signed {
				dec = int64(S16(w))
			} else {
				dec = int64(w)
			}
			rawHex = fmt.Sprintf("0x%04X", w)
		}
		var value, unit interface{}
		if fld.Display == DispNum {
			value = round(float64(dec)*fld.Scale, 4)
			if fld.Unit != "" {
				unit = fld.Unit
			}
		} else {
			value = rawHex
		}
		rows = append(rows, []interface{}{
			ts, ip, port, unit, "input_registers", fmt.Sprintf("0x%04X", base+o), fld.Words,
			fld.Name, rawHex, raw, fld.Scale, value, unit, dtypeOf(fld), src,
		})
	}
	return rows
}

func arrayLogRows(ts, ip string, port, unit, base int, modV, modT, cellV []uint16) [][]interface{} {
	var rows [][]interface{}
	for i, w := range modV {
		rows = append(rows, []interface{}{ts, ip, port, unit, "input_registers",
			fmt.Sprintf("0x%04X", base+ModVOff+i), 1, fmt.Sprintf("Module %d voltage", i),
			fmt.Sprintf("0x%04X", w), uint32(w), 0.01, round(float64(w)*0.01, 2), "V", "U16", SrcStr})
	}
	for i, w := range modT {
		rows = append(rows, []interface{}{ts, ip, port, unit, "input_registers",
			fmt.Sprintf("0x%04X", base+ModTOff+i), 1, fmt.Sprintf("Module %d temperature", i),
			fmt.Sprintf("0x%04X", w), uint32(w), 0.1, round(float64(S16(w))*0.1, 1), "degC", "I16", SrcStr})
	}
	for i, w := range cellV {
		rows = append(rows, []interface{}{ts, ip, port, unit, "input_registers",
			fmt.Sprintf("0x%04X", base+CellVOff+i), 1, fmt.Sprintf("Cell %d voltage", i),
			fmt.Sprintf("0x%04X", w), uint32(w), 0.001, round(float64(w)*0.001, 3), "V", "U16", SrcStr})
	}
	return rows
}

func joinOr(s []string, fallback string) string {
	if len(s) == 0 {
		return fallback
	}
	return strings.Join(s, ", ")
}

// ExportMeasurementsXLSX re-reads the live registers (fresh connection per block,
// sequential) and writes the 15-column Readings/Latest/Runs workbook — the same
// format as bms_measurements.xlsx.
func ExportMeasurementsXLSX(cfg Config, path string) error {
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip, port, unit := cfg.IP, portOr502(cfg.Port), unitOr1(cfg.Unit)

	var log [][]interface{}
	var online, empty, noResp []string

	type blk struct {
		tag  string
		base int
		src  string
	}
	blocks := []blk{{"agg", AggBase, SrcAgg}}
	for n := 1; n <= 6; n++ {
		blocks = append(blocks, blk{fmt.Sprintf("S%d", n), PileBase + (n-1)*PileStride, SrcStr})
	}

	for _, b := range blocks {
		block, err := readBaseFresh(cfg, b.base, 0x52)
		if err != nil {
			if b.tag != "agg" {
				log = append(log, []interface{}{ts, ip, port, unit, "input_registers",
					fmt.Sprintf("0x%04X", b.base), 1, "(string not responding)", "-", nil, 1, nil, nil, "-", b.src})
				noResp = append(noResp, b.tag)
			}
			continue
		}
		enum := at(block, 0x03) > 0 || at(block, 0x36) > 0
		if b.tag != "agg" {
			if enum {
				online = append(online, b.tag)
			} else {
				empty = append(empty, b.tag)
			}
		}
		log = append(log, summaryLogRows(ts, ip, port, unit, b.base, block, b.src)...)
		if b.tag == "S1" && enum {
			nMod := clamp(int(at(block, 0x36)), 0, MaxModules)
			nCell := clamp(int(at(block, 0x37)), 0, MaxCells)
			modV, _ := readRegionFresh(cfg, b.base+ModVOff, nMod)
			modT, _ := readRegionFresh(cfg, b.base+ModTOff, nMod)
			cellV, _ := readRegionFresh(cfg, b.base+CellVOff, nCell)
			log = append(log, arrayLogRows(ts, ip, port, unit, b.base, modV, modT, cellV)...)
		}
	}
	if len(log) == 0 {
		return fmt.Errorf("no data read from %s", cfg.addr())
	}

	fx := excelize.NewFile()
	defer fx.Close()
	headerStyle, _ := fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E78"}, Pattern: 1},
		Font: &excelize.Font{Color: "FFFFFF", Bold: true},
	})
	widths := map[string]float64{"A": 20, "B": 15, "C": 9, "E": 18, "F": 11, "G": 14,
		"H": 38, "I": 18, "J": 14, "K": 10, "L": 14, "M": 10, "O": 48}

	writeLog := func(name string) {
		fx.NewSheet(name)
		hdr := logHeaders
		fx.SetSheetRow(name, "A1", &hdr)
		fx.SetCellStyle(name, "A1", "O1", headerStyle)
		fx.SetPanes(name, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
		for col, w := range widths {
			_ = fx.SetColWidth(name, col, col, w)
		}
		for i := range log {
			row := log[i]
			_ = fx.SetSheetRow(name, fmt.Sprintf("A%d", i+2), &row)
		}
	}
	writeLog("Readings")
	writeLog("Latest")

	fx.NewSheet("Runs")
	runsHdr := []interface{}{"Timestamp", "IP", "Port", "Unit ID", "Read Method", "Measurements", "Status"}
	fx.SetSheetRow("Runs", "A1", &runsHdr)
	fx.SetCellStyle("Runs", "A1", "G1", headerStyle)
	status := fmt.Sprintf("OK - online: %s; empty: %s", joinOr(online, "none"), joinOr(empty, "none"))
	if len(noResp) > 0 {
		status += "; no-response: " + strings.Join(noResp, ", ")
	}
	runsRow := []interface{}{ts, ip, port, unit, "input_registers", len(log), status}
	fx.SetSheetRow("Runs", "A2", &runsRow)

	fx.DeleteSheet("Sheet1")
	if idx, err := fx.GetSheetIndex("Readings"); err == nil {
		fx.SetActiveSheet(idx)
	}
	return fx.SaveAs(path)
}

// ExportPDFReport writes a one-page summary PDF from a snapshot.
func ExportPDFReport(snap SystemSnapshot, path string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(31, 78, 120)
	pdf.Cell(0, 10, "Pylontech BMS - System Report")
	pdf.Ln(11)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(80, 80, 80)
	pdf.Cell(0, 6, fmt.Sprintf("Captured %s   Gateway %s:%d   %s %s",
		snap.Timestamp, snap.IP, snap.Port, snap.Identity.Name, snap.Identity.Firmware))
	pdf.Ln(9)

	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(31, 78, 120)
	pdf.Cell(0, 7, "System")
	pdf.Ln(7)
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(40, 40, 40)
	a := snap.Aggregate
	pdf.Cell(0, 6, fmt.Sprintf("Strings online: %d of 6    Total: %.1f V    SOC: %d%%    SOH: %d%%    Current: %.2f A",
		snap.Chain.Online, a.TotalV, a.SOC, a.SOH, a.Current))
	pdf.Ln(6)
	pdf.Cell(0, 6, "Status: "+snap.Health.Headline)
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(31, 78, 120)
	pdf.Cell(0, 7, "Strings")
	pdf.Ln(7)

	headers := []string{"#", "Status", "Voltage", "SOC", "SOH", "Temp", "Mods/Cells", "Serial"}
	widths := []float64{10, 30, 22, 14, 14, 18, 24, 48}
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(31, 78, 120)
	pdf.SetTextColor(255, 255, 255)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 7, h, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(7)
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(40, 40, 40)
	for _, s := range snap.Strings {
		cells := []string{
			itoa(s.Index), s.Status, fmt.Sprintf("%.1f V", s.TotalV),
			fmt.Sprintf("%d%%", s.SOC), fmt.Sprintf("%d%%", s.SOH),
			fmt.Sprintf("%.1f C", s.Temp), fmt.Sprintf("%d/%d", s.Modules, s.Cells), s.Serial,
		}
		for i, c := range cells {
			pdf.CellFormat(widths[i], 6, c, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(6)
	}
	return pdf.OutputFileAndClose(path)
}
