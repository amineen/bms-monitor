// Package solarman is the REMOTE data source: it reads the same battery data
// from the Solarman cloud portal's internal API using a Bearer token captured
// from a one-time browser login. It maps the result into bms.SystemSnapshot so
// the exact same dashboard renders on-site (Modbus) or remote (Solarman).
package solarman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bms-monitor/internal/bms"
)

const Host = "https://home.solarmanpv.com"

// Config holds the remote-session settings (token is a long-lived Bearer JWT).
type Config struct {
	Token     string `json:"token"`
	StationID int64  `json:"stationId"`
}

type Client struct {
	http  *http.Client
	token string
}

func New(token string) *Client {
	return &Client{http: &http.Client{Timeout: 25 * time.Second}, token: strings.TrimSpace(token)}
}

func (c *Client) do(method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, Host+path, rdr)
	if err != nil {
		return err
	}
	tok := c.token
	if !strings.HasPrefix(tok, "Bearer ") {
		tok = "Bearer " + tok
	}
	req.Header.Set("Authorization", tok)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("solarman session expired (HTTP %d) — reconnect", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("solarman API %s -> HTTP %d", path, resp.StatusCode)
	}
	return json.Unmarshal(data, out)
}

// ---- API response types ----

type treeResp struct {
	Total int `json:"total"`
	Data  []struct {
		ID             int64  `json:"id"`
		DeviceSn       string `json:"deviceSn"`
		SystemName     string `json:"systemName"`
		NetState       int    `json:"netState"`
		AlertState     int    `json:"alertState"`
		DeviceState    int    `json:"deviceState"`
		ParentDeviceSn string `json:"parentDeviceSn"`
	} `json:"data"`
}

type detailReq struct {
	DeviceID             int64  `json:"deviceId"`
	Language             string `json:"language"`
	NeedRealTimeDataFlag bool   `json:"needRealTimeDataFlag"`
	SiteID               int64  `json:"siteId"`
}

type field struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Storage string `json:"storageName"`
	Unit    string `json:"unit"`
}

type detailResp struct {
	DeviceSn          string `json:"deviceSn"`
	ConnectStatus     int    `json:"connectStatus"`
	ParamCategoryList []struct {
		FieldList []field `json:"fieldList"`
	} `json:"paramCategoryList"`
}

func (d detailResp) byStorage() map[string]string {
	m := make(map[string]string)
	for _, cat := range d.ParamCategoryList {
		for _, f := range cat.FieldList {
			if f.Storage != "" {
				m[f.Storage] = f.Value
			}
		}
	}
	return m
}

func pf(s string) float64 { v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64); return v }
func pi(s string) int     { return int(pf(s)) }

// indexedSeries collects m["<prefix>0"], m["<prefix>1"], … up to count (or the
// first missing index), rounding each to dec decimals. Solarman publishes the
// per-module and per-cell arrays as flat, zero-indexed keys (V_MDU0…, V_SC0…).
func indexedSeries(m map[string]string, prefix string, count, dec int) []float64 {
	if count <= 0 || count > 1024 {
		count = 1024
	}
	out := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		v, ok := m[prefix+strconv.Itoa(i)]
		if !ok {
			break
		}
		out = append(out, bms.Round(pf(v), dec))
	}
	return out
}

// ReadSystem builds a bms.SystemSnapshot from Solarman for the given station.
// One battery/tree call + one v3/detail call per string (sequential).
func (c *Client) ReadSystem(stationID int64) (*bms.SystemSnapshot, error) {
	if stationID == 0 {
		return nil, fmt.Errorf("no station selected")
	}
	start := time.Now()
	var tree treeResp
	if err := c.do("GET", fmt.Sprintf(
		"/maintain-s/operating/station/%d/battery/tree?order.direction=DESC&order.property=name&page=1&size=1000&total=0",
		stationID), nil, &tree); err != nil {
		return nil, err
	}

	snap := &bms.SystemSnapshot{
		Timestamp: time.Now().Format(time.RFC3339),
		IP:        "solarman:" + strconv.FormatInt(stationID, 10),
		Identity:  bms.Identity{OK: true, Name: "Solarman (remote)", Firmware: "V1.6"},
		LinkLive:  true,
	}

	var sumV, sumSOC, sumSOH, sumTemp, sumCur float64
	var online int
	cellMax, cellMin := 0.0, 99.0
	for i, b := range tree.Data {
		if i >= 6 {
			break
		}
		si := bms.StringInfo{Index: i + 1, Serial: b.DeviceSn, IsMaster: i == 0, Alarms: []string{}}
		var det detailResp
		err := c.do("POST", "/device-s/device/v3/detail",
			detailReq{DeviceID: b.ID, Language: "en", NeedRealTimeDataFlag: true, SiteID: stationID}, &det)
		enumerated := err == nil && b.NetState == 1 && pf(det.byStorage()["V_BAP1"]) > 0
		if !enumerated {
			if err != nil {
				si.Status = bms.StatusNoResponse
			} else {
				si.Status = bms.StatusEmpty
			}
			snap.Strings = append(snap.Strings, si)
			continue
		}
		m := det.byStorage()
		si.Status = bms.StatusEnumerated
		si.TotalV = bms.Round(pf(m["V_BAP1"]), 1)
		si.Current = bms.Round(pf(m["I_BAP1"]), 2)
		si.PowerKW = bms.Round(si.TotalV*si.Current/1000, 3)
		si.SOC = pi(m["SOC_BAP1"])
		si.SOH = pi(m["SOH_BAP1"])
		si.Temp = bms.Round(pf(m["T_mai_ctrl1"]), 1)
		si.Cycles = pi(m["NUMcyc1"])
		si.CellMaxV = bms.Round(pf(m["V_SCE_Bma1"]), 3)
		si.CellMinV = bms.Round(pf(m["V_SCE_Bmi1"]), 3)
		si.CellSpreadMV = bms.Round((si.CellMaxV-si.CellMinV)*1000, 0)
		si.CellMaxT = bms.Round(pf(m["T_SCE_Bma1"]), 1)
		si.CellMinT = bms.Round(pf(m["T_SCE_Bmi1"]), 1)
		si.ModMaxV = bms.Round(pf(m["V_MDU_Bma1"]), 2)
		si.ModMinV = bms.Round(pf(m["V_MDU_Bmi1"]), 2)
		si.Modules = pi(m["NUM_BAP_MDUser1"])
		si.Cells = pi(m["NUM_BAP_SCEser1"])
		si.NominalAh = pi(m["cap_BAP1"])
		si.BasicStatus = "Online"
		si.ProtectionText = "OK (none)"

		// Solarman returns the full per-module and per-cell arrays in this same
		// detail response — one device (string) at a time — so unlike the
		// on-site Modbus read we get this detail for every string, not just the
		// master.
		si.ModuleV = indexedSeries(m, "V_MDU", si.Modules, 2)
		si.ModuleT = indexedSeries(m, "T_MDU", si.Modules, 1)
		si.CellV = indexedSeries(m, "V_SC", si.Cells, 3)
		si.HasDetail = len(si.CellV) > 0 && si.CellMaxV > 0

		online++
		sumV += si.TotalV
		sumSOC += float64(si.SOC)
		sumSOH += float64(si.SOH)
		sumTemp += si.Temp
		sumCur += si.Current
		if si.CellMaxV > cellMax {
			cellMax = si.CellMaxV
		}
		if si.CellMinV < cellMin {
			cellMin = si.CellMinV
		}
		snap.Strings = append(snap.Strings, si)
	}

	// pad to 6 strings so the UI always shows 6 slots
	for n := len(snap.Strings) + 1; n <= 6; n++ {
		snap.Strings = append(snap.Strings, bms.StringInfo{Index: n, Status: bms.StatusEmpty, Alarms: []string{}})
	}

	if online > 0 {
		snap.Aggregate = bms.Aggregate{
			OK: true, Piles: online,
			TotalV:   bms.Round(sumV/float64(online), 1),
			Current:  bms.Round(sumCur, 2),
			PowerKW:  bms.Round(sumV/float64(online)*sumCur/1000, 3),
			SOC:      int(bms.Round(sumSOC/float64(online), 0)),
			SOH:      int(bms.Round(sumSOH/float64(online), 0)),
			Temp:     bms.Round(sumTemp/float64(online), 1),
			CellMaxV: cellMax, CellMinV: cellMin,
		}
	}

	snap.Chain = bms.BuildChain(snap.Strings)
	snap.Health = bms.AssessHealth(snap)
	snap.ReadMillis = time.Since(start).Milliseconds()
	return snap, nil
}

// TestToken makes a lightweight authenticated call to verify the token works.
func (c *Client) TestToken() error {
	var v any
	return c.do("GET", "/user-s/acc/org/login-user", nil, &v)
}
