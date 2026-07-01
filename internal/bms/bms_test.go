package bms

import "testing"

func TestS16(t *testing.T) {
	cases := map[uint16]int{0x0000: 0, 0x7FFF: 32767, 0x8000: -32768, 0xFFFF: -1, 0xFFA6: -90, 0xFF92: -110}
	for in, want := range cases {
		if got := S16(in); got != want {
			t.Errorf("S16(0x%04X)=%d want %d", in, got, want)
		}
	}
}

func TestS32(t *testing.T) {
	if got := S32(0x0000, 0x0028); got != 40 {
		t.Errorf("S32 charge alarm = %d want 40", got)
	}
	if got := S32(0xFFFF, 0xFFD8); got != -40 { // discharge alarm (two's complement)
		t.Errorf("S32 discharge alarm = %d want -40", got)
	}
	if got := S32(0xFC18, 0x0000); got >= 0 {
		t.Errorf("S32 large negative hi-word should be negative, got %d", got)
	}
}

func TestU32(t *testing.T) {
	if got := U32(0x0001, 0x0000); got != 0x10000 {
		t.Errorf("U32 = 0x%X want 0x10000", got)
	}
}

func TestVoltageScale(t *testing.T) {
	// 0x1D4B = 7499 ; ×0.1 = 749.9 V (the value seen on the live stack)
	if got := round(float64(0x1D4B)*0.1, 1); got != 749.9 {
		t.Errorf("total voltage = %v want 749.9", got)
	}
	// cell temp 0xFFA6 = -90 ; ×0.1 = -9.0 C
	if got := round(float64(S16(0xFFA6))*0.1, 1); got != -9.0 {
		t.Errorf("cell low-temp = %v want -9.0", got)
	}
	// module voltage 0x29CA = 10698 ; ×0.01 = 106.98 V
	if got := round(float64(0x29CA)*0.01, 2); got != 106.98 {
		t.Errorf("module voltage = %v want 106.98", got)
	}
}

func TestDecodeASCII(t *testing.T) {
	// "Pylon" packed two chars per word (hi byte first)
	words := []uint16{'P'<<8 | 'y', 'l'<<8 | 'o', 'n'<<8 | 0}
	if got := decodeASCII(words); got != "Pylon" {
		t.Errorf("decodeASCII=%q want Pylon", got)
	}
}

func TestDecodeBasic(t *testing.T) {
	// 0x0403 = state 3 (Idle) + bit 10 (Idle) -> "Idle, Idle" (matches Python)
	if got := DecodeBasic(0x0403); got != "Idle, Idle" {
		t.Errorf("DecodeBasic(0x0403)=%q want %q", got, "Idle, Idle")
	}
	if got := DecodeProt(0x0000); got != "OK (none)" {
		t.Errorf("DecodeProt(0)=%q want OK (none)", got)
	}
}

func TestChainStatus(t *testing.T) {
	ss := []StringInfo{
		{Index: 1, Status: StatusEnumerated}, {Index: 2, Status: StatusEnumerated},
		{Index: 3, Status: StatusEnumerated}, {Index: 4, Status: StatusEnumerated},
		{Index: 5, Status: StatusEnumerated}, {Index: 6, Status: StatusEmpty},
	}
	cs := chainStatus(ss)
	if cs.Online != 5 || cs.StopsAfter != 5 {
		t.Errorf("chain online=%d stopsAfter=%d want 5/5", cs.Online, cs.StopsAfter)
	}
}

func TestSummaryTable(t *testing.T) {
	if len(Summary) < 50 {
		t.Fatalf("summary table too small: %d", len(Summary))
	}
	// spot-check a few load-bearing entries
	byOff := map[int]Field{}
	for _, f := range Summary {
		byOff[f.Off] = f
	}
	if f := byOff[0x04]; !f.Signed || f.Words != 2 || f.Scale != 0.01 {
		t.Errorf("Current field wrong: %+v", f)
	}
	if f := byOff[0x03]; f.Scale != 0.1 || f.Unit != "V" {
		t.Errorf("Total voltage field wrong: %+v", f)
	}
}
