package solarman

import (
	"encoding/json"
	"os"
	"testing"
)

// sample mirrors the real POST /device-s/device/v3/detail response shape
// (fields keyed by storageName), trimmed to the load-bearing measure points.
const sample = `{
  "deviceSn": "Y250704D30151204",
  "connectStatus": 1,
  "paramCategoryList": [
    {"fieldList": [
      {"key":"Connected Device SN","value":"Y250704D30151204","storageName":"SN1","unit":null},
      {"key":"Series Number of Battery Module","value":"7","storageName":"NUM_BAP_MDUser1","unit":null},
      {"key":"Series Number of Battery Cell","value":"224","storageName":"NUM_BAP_SCEser1","unit":null}
    ]},
    {"fieldList": [
      {"key":"Battery Pack Voltage","value":"744.50","storageName":"V_BAP1","unit":"V"},
      {"key":"Battery Pack Current","value":"0.00","storageName":"I_BAP1","unit":"A"},
      {"key":"Main Control Temperature","value":"38.00","storageName":"T_mai_ctrl1","unit":"C"},
      {"key":"Battery Pack SOC","value":"45","storageName":"SOC_BAP1","unit":"%"},
      {"key":"Battery Pack SOH","value":"100","storageName":"SOH_BAP1","unit":"%"},
      {"key":"Max. Voltage of Battery Cell","value":"3.32","storageName":"V_SCE_Bma1","unit":"V"},
      {"key":"Min. Voltage of Battery Cell","value":"3.31","storageName":"V_SCE_Bmi1","unit":"V"},
      {"key":"Max. Voltage of Module Battery","value":"106.36","storageName":"V_MDU_Bma1","unit":"V"},
      {"key":"Min. Voltage of Module Battery","value":"106.34","storageName":"V_MDU_Bmi1","unit":"V"},
      {"key":"Battery Capacity","value":"50","storageName":"cap_BAP1","unit":"Ah"}
    ]}
  ]
}`

func TestDetailByStorage(t *testing.T) {
	var d detailResp
	if err := json.Unmarshal([]byte(sample), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	m := d.byStorage()
	cases := map[string]string{
		"V_BAP1": "744.50", "SOC_BAP1": "45", "SOH_BAP1": "100",
		"NUM_BAP_MDUser1": "7", "NUM_BAP_SCEser1": "224", "SN1": "Y250704D30151204",
	}
	for k, want := range cases {
		if m[k] != want {
			t.Errorf("byStorage[%q]=%q want %q", k, m[k], want)
		}
	}
	// numeric parse + derived values
	if pf(m["V_BAP1"]) != 744.5 {
		t.Errorf("V_BAP1 parse = %v", pf(m["V_BAP1"]))
	}
	spread := (pf(m["V_SCE_Bma1"]) - pf(m["V_SCE_Bmi1"])) * 1000
	if int(spread+0.5) != 10 {
		t.Errorf("cell spread = %v mV want 10", spread)
	}
}

// TestRealDetailArrays proves that the per-module and per-cell arrays parse out
// of an actual captured Solarman v3/detail response — this is the data the
// remote String detail view renders (module bar chart + 224-cell distribution).
func TestRealDetailArrays(t *testing.T) {
	raw, err := os.ReadFile("testdata/v3_detail_master.json")
	if err != nil {
		t.Skipf("no captured sample: %v", err)
	}
	var d detailResp
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("unmarshal real detail: %v", err)
	}
	m := d.byStorage()

	modules := pi(m["NUM_BAP_MDUser1"])
	cells := pi(m["NUM_BAP_SCEser1"])
	if modules != 7 || cells != 224 {
		t.Fatalf("counts: modules=%d cells=%d want 7/224", modules, cells)
	}

	mv := indexedSeries(m, "V_MDU", modules, 2)
	mt := indexedSeries(m, "T_MDU", modules, 1)
	cv := indexedSeries(m, "V_SC", cells, 3)
	if len(mv) != 7 {
		t.Errorf("module voltages = %d want 7", len(mv))
	}
	if len(mt) != 7 {
		t.Errorf("module temps = %d want 7", len(mt))
	}
	if len(cv) != 224 {
		t.Errorf("cell voltages = %d want 224", len(cv))
	}
	// every cell voltage should be a plausible LFP reading (2.5–3.7 V)
	for i, v := range cv {
		if v < 2.5 || v > 3.7 {
			t.Errorf("cell %d voltage %.3f out of range", i, v)
			break
		}
	}
	// V_MDU0 in the captured sample is 106.36 V
	if mv[0] != 106.36 {
		t.Errorf("V_MDU0 = %.2f want 106.36", mv[0])
	}
}
