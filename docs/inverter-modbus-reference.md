# TEC Totota — Inverter Modbus Reference

Reference for extending the Go/Wails BMS Monitor app to also read the **PV inverters** and the
**battery inverters**. Sources are primary manufacturer docs; **INFERRED** values are flagged and
listed in "Open questions".

**Rev 2 (2026-07-05):** reconciled with an independent Codex research pass. **Major correction:**
the OzTek section was originally mapped from **UM-0061** (2016, legacy SunSpec model 103 + MESA
64800). The deployed unit is the **OZPCS-RS40 (UM-0075 Rev N)**, which exposes the **modern DER
SunSpec models 701–715 + OzTek 64340** instead. The OzTek register numbers below are now the RS40
DER map; the old model-103 addresses did **not** apply to the deployed firmware. Codex's raw output +
a device-specific SMA STP15-25TL-30 register CSV live at
`~/Developer/Totota Expansion/TEC-Battery-Installation/modbus-integration-reference/`.

## System (from TEC-Modified-SLD 2026 + interconnection set)

| Block | Equipment | Count | DC | AC tie |
|---|---|---|---|---|
| PV inverters | **SMA Sunny Tripower 25 kVA** (INV#1–#3) | 3 | 0–1000 V, ≥2 MPPT each; rows of 40×340 W | 50 A 3P → MDP "PV" |
| Battery inverters | **OzTek 40 kVA PCS (OZPCS-RS40)** (BATT#1, #2 active; #3 spare) | 2 (+1) | Pylontech via P-Combiner-HV-6-V2, ~716–760 V | 70 A 3P → MDP |
| Battery | 6× Pylontech Force-H3 | 6 strings | HV DC bus | — |
| Control | **Ageto ARC** microgrid controller | 1 | — | polls both inverters + BMS via RS485/TCP converters |
| Genset | Perkins 135 kVA prime (108 kW), 415 V 3Ø 50 Hz | 1 | — | via transfer switch |

Site: Totota Electric Cooperative (TEC), Totota, Liberia (6.812313°, -9.941198°).
ARC controls the OzTeks + SMAs; the Pylontech is read-only to ARC (see the app's commissioning note).

Read **in parallel with ARC** only with one-shot reads, not spaced polling loops (same contention
lesson as the BMS). PV = Modbus **TCP**; battery = Modbus **RTU/RS-485**. **Do not run two Modbus
masters on the OzTek RS-485** — coordinate with ARC or use an arbitrating gateway.

---

## 1. PV INVERTER — SMA Sunny Tripower 25000TL-30

SMA exposes **two Modbus profiles on the same device, both over Modbus TCP port 502**. Modbus is
**disabled by default** — must be enabled per inverter (Webconnect/UI). SMA-native is easier for us
(fixed scaling, no SF-register chasing); SunSpec if we want the standard model chain.

### Comms
| Item | SMA native profile | SunSpec profile |
|---|---|---|
| Transport | Modbus **TCP**, port **502** | Modbus **TCP**, port **502** |
| Default Unit/Slave ID | **3** (configurable 3–123) | **126** (= native ID + 123) |
| Function codes | FC03/FC04 read, FC06/FC16/FC23 write | same |
| Word/byte order | big-endian (Motorola, hi word first) | big-endian |
| Register base | 30xxx/31xxx measured, 40xxx/41xxx settings; PDU = reg − 1 | 40001 anchor = `0x53756E53` "SunS" |
| Scaling | fixed: FIX0/1/2/3 = ÷1/10/100/1000; TEMP=÷10 | SunSpec SF companion registers |
| NaN sentinel | S32 `0x80000000`, U32 `0xFFFFFFFF`; **enum/TAGLIST NaN = 16777213 (0x00FFFFFD)** | SunSpec NaN per type |
| Poll timing | ≥1 s between transfers, ≤5 params at once, dashboard 2–5 s | same |
| Unit-ID discovery | query **reg 42109** on Unit ID 1 | — |

3 physical inverters → **3 Modbus TCP endpoints** (distinct IP each, unit ID 3 native / 126 SunSpec).
Confirm whether Modbus is direct over Speedwire/Webconnect or via an SMA Data Manager.

### SMA-native register map (STP15-25TL-30 device-specific list; big-endian, 2-reg values)

| Reg | Type | Fmt | Unit | Meaning |
|---|---|---|---|---|
| 30001 | U32 | RAW | — | Modbus profile version |
| 30005 | U32 | RAW | — | **Serial number** (also 30057) |
| 30051 | U32 | ENUM | — | Device class (**8001 = Solar Inverters**) |
| **30053** | U32 | ENUM | — | **Device type — 9285 = STP 25000TL-30** (9284 STP20000, 9336 STP15000, 9337 STP17000) |
| 30059 | U32 | FW | — | Software/firmware package |
| **30201** | U32 | ENUM | — | **Condition**: 35=Fault, 303=Off, 307=Ok, 455=Warning |
| 30211 | U32 | ENUM | — | Recommended action (336 contact mfr, 337 contact installer, 887 none) |
| 30213 | U32 | ENUM | — | Message |
| 30217 | U32 | ENUM | — | **Grid relay/contactor**: 51=Closed, 311=Open, 16777213=NaN |
| 30219 | U32 | ENUM | — | Derating reason (557 Overtemp, 884 not active, 3556 High DC V, …) |
| 30225 | U32 | FIX0 | Ω | Insulation resistance |
| 30231 | U32 | FIX0 | W | Rated active power (25000 W) |
| 30233 | U32 | FIX0 | W | Set active power limit |
| 30513 | U64 | FIX0 | Wh | **Total yield** (aliases 30529 Wh / 30531 kWh / 30533 MWh) |
| 30517 | U64 | FIX0 | Wh | **Daily yield** (aliases 30535 Wh / 30537 kWh / 30539 MWh) |
| 30521 | U64 | Dur | s | Operating time (30525 feed-in time) |
| 30583 | U32 | FIX0 | Wh | Grid feed-in counter (lifetime export) |
| **30769** | S32 | FIX3 | A | **DC current input A** (MPPT-1) |
| **30771** | S32 | FIX2 | V | **DC voltage input A** |
| **30773** | S32 | FIX0 | W | **DC power input A** |
| **30775** | S32 | FIX0 | W | **AC active power (total)** |
| 30777/30779/30781 | S32 | FIX0 | W | AC power L1/L2/L3 (TL-20 profile; verify on TL-30) |
| 30783/30785/30787 | U32 | FIX2 | V | Grid voltage L1/L2/L3 |
| 30795 | U32 | FIX3 | A | Grid current (total; TL-20 profile) |
| **30803** | U32 | FIX2 | Hz | **Grid frequency** |
| 30805 | S32 | FIX0 | VAr | Reactive power (TL-20 profile) — **TL-30 list shows 31497**; confirm |
| 30813 | S32 | FIX0 | VA | Apparent power |
| 30953 | S32 | TEMP | °C | **Internal temperature** (÷10) |
| **30957** | S32 | FIX3 | A | **DC current input B** (MPPT-2) |
| **30959** | S32 | FIX2 | V | **DC voltage input B** |
| **30961** | S32 | FIX0 | W | **DC power input B** |
| 30975 | S32 | FIX2 | V | DC-link (intermediate circuit) voltage |
| 30977/30979/30981 | S32 | FIX3 | A | Grid current L1/L2/L3 |
| 30929 | U32 | ENUM | — | Speedwire status (307 Ok, 455 Warn, 1725 No connection) |
| 31017 | STR32 | UTF8 | — | Speedwire IP address (subnet 31025, gw 31033, DNS 31041) |
| 33003 | U32 | ENUM | — | RunStt (295 MPP, 443 Const V, 1469 Shut down, 2119 Derating) |

DC total power = input A (30773) + input B (30961). Reactive power register differs TL-20 (30805) vs
TL-30 device list (31497) — read once and confirm.

### SMA SunSpec equivalents (Unit ID 126)
Models: **1** Common (40001 anchor, Mn @40005, Md @40021, SN @40053); **101/103** inverter block
(header @40187, L=50): W @40200 (SF 40201), Hz @40202 (SF 40203), VA @40204, phase A/B/C current
@40189-91 (SF 40192), phase V @40196-98 (SF 40199), TmpCab @40219, **St @40224** (303 Off / 569 Run
/ 1295 Standby / 1795 Bolted / 16777213 NaN), Evt1 @40226, WH @40210 (SF 40212); **120** nameplate
(@40238, DERTyp @40240 = device-type enum, WRtg @40241); **121–124, 126–132**; **160 multiple MPPT**
(header @40622, L=128, **N=6 modules**, SF DCA=−1@40624/DCV=0@40625/DCW=1@40626, module blocks at
40632/40652/40672/40692/40712/40732, 20-reg stride — input A @40641-43, input B @40661-63).

---

## 2. BATTERY INVERTER — OzTek OZPCS-RS40 40 kVA PCS (UM-0075 Rev N)

**SunSpec DER-compliant** PCS (models 701–715), controller-commanded. Runs on DC present + external
command; internal DC pre-charge / soft-start (<1 A inrush) — needs **no battery-BMS comms**.

> ⚠️ Corrected in Rev 2: the deployed RS40 does **NOT** use the legacy `103 / 120-145 / MESA 64800`
> map (that was UM-0061, 2016). It uses **DER models 1, 17, 701-715 + OzTek 64340/64341/64302/64304/
> 64305/64308**. Confirm nameplate is RS40 (model string `OZpcs-RS40-FB1`) vs EP40 (UM-0073).

### Comms (UM-0075 §10-11)
| Item | Value |
|---|---|
| Transport | Modbus **RTU over RS-485**, half-duplex, D-Sub 15 (A+ pin5, GND pin10, B− pin15, term 14-15) |
| Default slave addr | **1** (reg 40069; 1–247); **0 = broadcast** (parallel units respond together) |
| Baud | default **57600** (reg 40077; 4800–115200); 8 data bits, parity None (40080), 1 stop |
| Function codes | FC03 read; FC06/FC16 write |
| Word/byte order | big-endian; SunSpec `Sunssf` SF registers; base 1-based (PDU = reg − 1) |
| SunSpec anchor | 40001 = `0x53756E53`; walk model-ID/length pairs from 40003 |
| Response time | ≤2.5 ms instrumentation, ≤8 ms config |

**Termination:** terminate the RS-485 bus at both ends. With OzTek #3 removed, re-check the two end
terminations of the multidrop.

### Model chain (RS40 Rev N)
1 Common @40003 · 17 Serial @40071 · **701 DER AC Measurement @40085** · 702 DER Capacity @40240 ·
703 Enter Service @40292 · 704 AC Controls @40311 · 705 Volt-VAR @40378 · 706 Volt-Watt @40453 ·
707 Freq-Watt @40518 · 708 LFRT @40751 · 709 HFRT @40984 · 710 LVRT @41289 · 711 HVRT @41594 ·
712 Trip @41628 · **714 DER DC Measurement @41690** · 715 DER Control @41735 · **64340 OzTek Ctrl &
Status @41744** · 64341 Config @41811 · 64302 @41869 · 64304 @41933 · 64305 @42083 · **64308
Grid-Forming Ctrl @42211** · End 0xFFFF @42271.

### Key measurement registers (FC03)

| Reg / hex | Model | Name | Type | SF | Unit | Notes |
|---|---|---|---|---|---|---|
| 40021 / 0x9C55 | 1 | Model string | str16 | — | — | `OZpcs-RS40-FB1` |
| 40053 / 0x9C75 | 1 | Serial number | str16 | — | — | |
| 40088 / 0x9C98 | 701 | Generic operating state | u16 | — | — | 0 off, 1 on (prefer 41746) |
| 40090 / 0x9C9A | 701 | **Grid connection state** | u16 | — | — | 0 disconnected, 1 connected |
| 40091 / 0x9C9B | 701 | **DER alarm bitfield** | u32 | — | bits | b1 DC OV, **b2 AC disc, b3 DC disc, b4 grid disc**, b7 over-temp, b8/9 over/under-freq, b10 AC OV, b11 AC UV, b13 under-temp, b16 mfr alarm |
| 40093 / 0x9C9D | 701 | DER mode bitfield | u32 | — | bits | b0 grid-following, b1 grid-forming |
| **40095** / 0x9C9F | 701 | **Active power** | s16 | 40201 | W | SF 1 |
| 40096 / 0x9CA0 | 701 | Apparent power | s16 | 40203 | VA | |
| 40097 / 0x9CA1 | 701 | Reactive power | s16 | 40204 | VAR | |
| 40098 / 0x9CA2 | 701 | Power factor | s16 | 40202 | — | SF −3 |
| 40099 / 0x9CA3 | 701 | AC current total | s16 | 40198 | A | SF −1 |
| 40100/40101 | 701 | Avg AC L-L / L-N voltage | u16 | 40199 | V | SF −1 |
| **40102** / 0x9CA6 | 701 | **Line frequency** | u32 | 40200 | Hz | SF −3 (note: U32) |
| 40121/40122/40124 | 701 | Cabinet / heatsink / IGBT temp | s16 | 40207 | °C | SF 0 |
| 40130-40178 | 701 | Per-phase A/B/C current + AB/BC/CA/AN/BN/CN voltage | s16/u16 | 40198/40199 | A/V | |
| **41695** / 0xA2DF | 714 | **DC current total** | s16 | none | A | 0.1 A fixed, **no SF reg** |
| **41696** / 0xA2E0 | 714 | **DC power total** | s16 | none | W | 10 W fixed, **no SF reg** |
| 41710 / 0xA2EE | 714 | DC port 1 type | u16 | — | — | 5 = generic bidirectional |
| **41720** / 0xA2F8 | 714 | **DC port 1 current** | s16 | 41705 | A | SF −1 |
| **41721** / 0xA2F9 | 714 | **DC port 1 voltage** | u16 | 41706 | V | SF −1 |
| **41722** / 0xA2FA | 714 | **DC port 1 power** | s16 | 41707 | W | SF 1 |
| 41731 / 0xA303 | 714 | DC port 1 heatsink temp | s16 | 41709 | °C | |
| **41738** / 0xA30A | 715 | **PCS heartbeat** | u32 | — | count | +1/sec — **read-safe** |
| **41746** / 0xA312 | 64340 | **PCS operating state** | u16 | — | enum | **best dashboard state** — see enum |
| 41754 / 0xA31A | 64340 | Active ride-through status | u16 | — | bits | b0 lowV b1 highV b2 lowF b3 highF |
| **41756** / 0xA31C | 64340 | **PCS warning status** | u32 | — | bits | see decode |
| **41758** / 0xA31E | 64340 | **PCS fault status** | u32 | — | bits | see decode |
| **41760** / 0xA320 | 64340 | **Factory fault status** | u32 | — | bits | read when fault bit 25 set |

**Operating-state enum (41746):** 0 initialize · 1 fault · 2 calibrate · 3 disabled · 4 charge-wait ·
5 charging · 6 standby · 7 turn-on delay · **8 online-grid-tie** · 9 offline · 10 active ride-through ·
11 passive ride-through · **12 online-grid-form** · 13 power-down · 16 turn-off · 17 island-transfer
wait · 18 service-disabled.

### 41758 Fault bitfield
0-2 HW OverCur A/B/C · 3-5 RMS OverCur A/B/C · 6 DC OverCur · 7-9 Grid OverV AB/BC/CA · 10 HW DC OverV
· 11 DC OverV · 12 DC UnderV · 13 RT LowV · 14 RT HighV · 15 RT LowFreq · 16 RT HighFreq · 17 Island ·
19 Temp Fault · 20 ESTOP · 21 **Communication Error** · 22 Power-Down Error · 23 Invalid User Config ·
24 Invalid Model · 25 Factory Fault (→ read 41760) · 26-28 Saturation A/B/C · 31 AC Current Overload Trip.

### 41756 Warning bitfield
0-2 High AC Cur A/B/C · 3 High DC Cur · 4-6 High Grid V AB/BC/CA · 7 High DC V · 8 Low DC V · 9 AC Cur
Limit · 10 DC Power Limit · 11 AC Power Limit · 12 Grid OOT · 13 Resume Delay · 14 Island Detected ·
15 PLL Not Locked · 16 Temp Warning · 19 Fan Warning · 21 Limit Active Power · 22 HVRT Override ·
23 TVS Error · 24 Volt-VAR · 25 Volt-Watt · 26 Freq-Watt · 27 Loss of Phase · 28 Neg-Seq Cur Limit ·
29 Watt-VAR · 31 AC Current Overload.

### 41760 Factory-fault bitfield (partial)
0-8 various HW/DC over-current · 14 Link OverV · 15 Link V Imbalance · **16 Pre-charge Timeout** ·
17 Bias UnderV · **18 Contactor Interlock** · 19 DC/DC Comm Error · 20 Datalog Error · 21 Invalid
Factory Config · 22 Config EEPROM Error · 23 Calibration Error.

### "Is the OzTek connected to the bus?" — no single contactor register
RS40 has **no dedicated AC/DC contactor-status register**. Derive connection from:
`40090` grid-connection-state (0/1) · `40091` DER-alarm bits b2 AC-disc / b3 DC-disc / b4 grid-disc ·
`41760` factory-fault b18 Contactor-Interlock / b16 Pre-charge-Timeout · plus operating-state 41746
(8 grid-tie / 12 grid-form = connected). (My Rev-1 claim of "warning bit 30 = DC contactor open" was
from the wrong UM-0061 map — dropped.)

### Control / write registers — DO NOT WRITE (read-only app)
41740 **controller heartbeat** (writing it *enables heartbeat checking*; loss then faults the PCS —
never touch) · 41742 fault reset · 41743 set-operation (0 stop/1 start/2 standby/3 exit-standby) ·
41747 control mode (0 grid-tie/1 grid-form) · 41748/41749 max DC charge/discharge current · 42213-16
grid-form V/Hz/P/Q commands · 42231 island-pin config · 40069/40077/40080 serial settings.

---

## Recommended minimal poll set (dashboard)

**SMA PV (native, per inverter, Unit ID 3, FC03):** 30053 device-type · 30201 condition · 30217 grid
relay · 30775 AC power · 30783/85/87 grid V · 30803 freq · 30769/71/73 MPPT-A · 30957/59/61 MPPT-B ·
30513 total yield · 30517 daily · 30953 temp.

**OzTek (per inverter, slave 1, FC03):** 41746 operating state · 40095 AC power · 40102 freq · 40099
AC current · 41721/41720/41722 DC V/I/P · 40121/40122 temps · 41756 warning · 41758 fault · 40090/
40091 connection/alarm · 41738 heartbeat (liveness).

---

## Go implementation notes
- Register numbers are 1-based → `pduAddr = reg − 1`. 16-bit big-endian; 32/64-bit + strings
  high-word first. Trim trailing NUL/space on OzTek ASCII strings.
- OzTek SunSpec scaling: `value = raw × 10^sf` where `sf` is the signed-16 companion register
  (`Sunssf`). SMA-native uses fixed FIX0/1/2/3 (no companion) — simpler.
- Map SMA NaN sentinels (S32 0x80000000, U32 0xFFFFFFFF, enum 16777213) and OzTek out-of-range → null.
- Two clients: `internal/sma` (Modbus TCP:502, one endpoint per inverter) and `internal/oztek`
  (Modbus RTU/RS-485 via the converter, SunSpec DER walk). Mirror the `bms` fresh-connection-per-read
  discipline and keep everything read-only (never FC06/FC16 to the control registers above).

## Open questions / needs confirmation
1. **OzTek nameplate**: confirm **RS40 (UM-0075)** vs EP40 (UM-0073); read 40021 model string live.
   (Codex used RS40 Rev N as primary; DER 701-715 map applies to both modern variants.)
2. **Addressing**: how ARC reaches the 3 SMAs (IPs, Modbus enabled?) and the 2 OzTeks (slave IDs on
   the RS-485). SLD shows 3 OzTek positions — each needs a distinct 40069. Ask Anthony (ARC).
3. **SMA model/firmware**: 30053 should read **9285** (STP 25000TL-30) — verify live; confirm whether
   Modbus is direct (Speedwire) or via an SMA Data Manager.
4. **SMA reactive-power register** (30805 TL-20 vs 31497 TL-30) and per-phase power regs — confirm
   against the live TL-30.
5. **OzTek sign conventions** (active power, DC current: charge vs discharge) — validate on site.
6. **Contention with ARC** — one-shot reads only; never a second Modbus master on the OzTek RS-485.
7. **SMA official device-specific register ZIP** — mirrored HTML `MODBUS-HTML_STPTL-30_31005R_V10`;
   the direct SMA ZIP 404s. Verify before trusting any write/control register.

## Sources
- **OzTek**: UM-0075 OZPCS-RS40 SB PCS Manual Rev N (§10, §11.3) — trystar.com; UM-0073 EP40 (2024) &
  UM-0061 Rev AC for cross-check.
- **SMA**: STP15-25TL-30 device-specific Modbus register list (HTML/CSV); SMA-Modbus-general-TI-en-10;
  SunSpecModbus-TI-en-11; STP15-25TL-30-BE-en-18 operating manual — files.sma.de.
- TEC-Modified-SLD (2026) + TEC-BESS interconnection drawing set.
- Independent cross-check: Codex research pass (`~/Developer/Totota Expansion/TEC-Battery-Installation/
  modbus-integration-reference/`), incl. `sma_stp15_25tl_registers.csv` (554-row device-specific map).
