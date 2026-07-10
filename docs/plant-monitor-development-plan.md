# TEC Totota — Plant Monitor Development Plan

**From "Pylontech BMS Monitor" → "TEC Plant Monitor": a read-only, plug-in-anywhere
troubleshooting tool for the whole ARC energy network.**

Draft 2026-07-06. Owner: TEC/NRECA Int'l commissioning. Companion doc:
`inverter-modbus-reference.md` (SMA + OzTek maps). Codex cross-checks in
`~/Developer/Totota Expansion/TEC-Battery-Installation/modbus-integration-reference/`.

---

## 1. Goal & scope

The existing Wails/Go app reads only the Pylontech BMS. **Reframe it as a plant-wide
monitoring & troubleshooting tool** that a technician plugs into the ARC network (or reaches
remotely) and immediately sees every energy device: batteries, battery inverters, PV inverters,
genset, and the revenue meter — with plain-language health, live telemetry, and fault decodes.

- **Not** an ARC replacement / controller. **Read-only** by default (the one gated Pylontech Run
  write stays as-is; nothing else writes).
- **Purpose:** commissioning + troubleshooting — pinpoint *which* device/subsystem is faulting,
  cross-correlate (e.g. "PV tripped → battery picked up load → genset started"), and give ARC-like
  visibility without ARC access.
- **Deploy:** single portable binary (Win field laptop + Mac), plugged into the 192.168.0.0/24 ARC
  LAN; optional remote path via the eWon/Solarman.

## 2. Device inventory (from `10012 Totota Network List`, 2026-07-06)

| ARC name | Device | IP | Modbus | RS485 | Access from our app | Map status |
|---|---|---|---|---|---|---|
| **BMS2** | Pylontech Force-H3 BMS | 192.168.0.31 | ID 1 | NO | Modbus TCP :502 | ✅ done (`internal/bms`) |
| **INV1** | OzTek RS40 #1 (BATT#1) | 192.168.0.41 | ID **1** | YES | Modbus TCP→RTU via **USR-TCP232-410s** :502 | ✅ mapped (needs client) |
| **INV2** | OzTek RS40 #2 (BATT#2) | 192.168.0.41 | ID **2** | YES | **same IP/converter as INV1**, unit 2 | ✅ mapped |
| **INV3** | OzTek RS40 #3 (BATT#3) | 192.168.0.41 | ID **3** | YES | **same IP/converter**, unit 3 — confirmed on `.41` | ✅ mapped (reserve) |
| **PV1** | SMA Sunny Tripower 25 kVA | 192.168.0.51 | ID 3 | NO | Modbus TCP :502 | ✅ mapped · Modbus enabled |
| **PV2** | SMA Sunny Tripower 25 kVA | 192.168.0.52 | ID 3 | NO | Modbus TCP :502 | ✅ mapped · Modbus enabled |
| **PV3** | SMA Sunny Tripower 25 kVA | 192.168.0.53 | ID 3 | NO | Modbus TCP :502 | ✅ mapped · Modbus enabled |
| **GEN1** | DeepSea DSE8610 MKII | 192.168.0.71 | ID 1 | NO | Modbus TCP :502 | ✅ **verified below** |
| METER1 | AccuEnergy Acuvim II | 192.168.0.61 | ID 1 | NO | Modbus TCP :502 | ⛔ **NOT IN USE — out of scope** (map kept in Appendix B for reference) |
| BMS1 | Energport L00120 | 192.168.0.104 | ID 1 | NO | — | ⛔ DEPRECATED — hide |
| EWON1 | eWon Flexy 205 | 192.168.0.21 | ID 100 | NO | remote-access VPN gateway | infra (not polled) |
| — | Moxa MC-1112 IPC | 192.168.0.20 | — | — | likely the ARC host | infra (not polled) |

**Everything is reachable over Modbus TCP** on one Ethernet drop — even the RS485 OzTeks are bridged
by a USR-TCP232-410s serial gateway at .41. So the app needs a **single network connection** and one
transport (Modbus TCP), with per-device unit IDs.

Two addressing notes (both fine, confirmed 2026-07-06):
- **All 3 SMAs use unit ID 3** — no conflict: each is a separate TCP endpoint (`.51/.52/.53`), so the
  IP disambiguates them. Modbus TCP is **already enabled** on all three (ARC talks to them).
- **All 3 OzTeks share the `.41` converter** on one RS485 bus with unit IDs **1/2/3** (INV3/BATT#3
  now confirmed on `.41`). This is the system's **only shared serial bus and its top contention
  point** — see §3.2 and the read-window note in §3.3.

## 3. Load-bearing constraints (carry into every driver)

1. **Read-only.** No FC06/FC16 to any device except the existing gated Pylontech Run. Genset, OzTek,
   SMA, meter = strictly FC03/FC04.
2. **OzTek shared RS485 bus (the critical one).** All **three** OzTeks (INV1/2/3, units 1/2/3) sit on
   ONE USR converter at `.41`. Every read to any of them is the *same physical serial line* →
   **strictly sequential, one-in-flight, fresh connection between reads** (same discipline as the BMS
   converter). Never issue overlapping reads to `.41`. Budget ~1–2 s to walk all three units once.
3. **ARC is the primary Modbus master and is polling continuously.** Our reads are a *second* master.
   Use **one-shot reads on a relaxed cadence** (2–5 s), never tight loops — the contention lesson from
   the BMS (`pylontech-bms-arc-modbus-contention`). SMAs and the genset are native Ethernet with ≤5
   TCP masters, so they tolerate a second reader fine. **The `.41` OzTek serial bus is the only
   fragile point.** Two options there, decide with Anthony:
   - **(a) Relaxed one-shot** (default): poll `.41` every 3–5 s, tolerate occasional timeouts, mark a
     unit "stale" rather than "offline" on a miss. Zero coordination with ARC. Start here.
   - **(b) Read window**: ARC briefly pauses its OzTek poll while we take one snapshot (a few hundred
     ms), or exposes a lock. Cleaner data during heavy commissioning polling, but needs ARC support.
   Design the OzTek driver so a per-bus mutex + configurable cadence makes (a)→(b) a config change.
4. **SMA Modbus is off by default** — must be enabled per inverter before PV1/2/3 respond.
5. **Fresh-connection-per-read** discipline from `internal/bms` generalizes to all TCP drivers.

## 4. Architecture

```
internal/
  bms/      Pylontech Force-H3      (exists)
  oztek/    OZPCS-RS40 PCS          (SunSpec DER 701-715 + 64340, RTU-over-TCP, units 1 & 2)
  sma/      Sunny Tripower 25000TL  (SMA-native 30xxx over TCP, 3 endpoints)
  dse/      DSE8610 MKII genset     (GenComm fixed pages 4/6/7 over TCP)
  plant/    orchestrator: device registry, poll scheduler, unified PlantSnapshot, health engine
  # meter/  AccuEnergy Acuvim II — OUT OF SCOPE (not in use); driver deferred, map in Appendix B
  modbustcp/ shared TCP transport helper (fresh conn, timeout, unit-id, FC04→FC03 fallback)
```

- **Device driver interface:** each package exposes `Read(cfg) (DeviceSnapshot, error)` returning a
  common shape `{ Kind, Name, Online, Metrics map, Faults []string, Health }` plus a device-specific
  detail struct. Drivers are pure Modbus + decode; no UI.
- **`plant` orchestrator:** holds the device registry (seeded from the network list, editable in UI),
  runs a **sequential round-robin poll** (respecting the OzTek shared-bus rule), assembles a
  `PlantSnapshot`, and runs a **cross-device health/insight engine** (see §6).
- **Config:** ship a `devices.json` preloaded with the Totota network list (IPs/unit IDs above) so the
  tech plugs in and it just works; allow add/remove/enable in the UI.
- **Wails bindings:** `GetPlant()`, `ReadDevice(id)`, `GetDevices()/SaveDevices()`, keep existing BMS
  methods. Reuse the semi-auto onsite/remote and the commissioning gate.

## 5. UI/UX redesign — from single battery to plant view

- **Plant Overview (new home):** a one-screen microgrid mimic / single-line — PV → MDP ← Genset, with
  the BESS (batteries+OzTek) at the point of common coupling. Each node shows live power + status
  color; energy-flow arrows (who's sourcing, who's sinking). This is the "ARC-like situational
  awareness" view.
- **Device tiles:** one card per in-scope device (up to 8: BMS + 3 OzTek + 3 SMA + genset; 2 OzTek
  active until BATT#3 lands) with the 3–4 numbers that matter + health dot; click → device detail.
- **Device detail drawers:** reuse the battery detail pattern for each kind — OzTek (AC/DC, state,
  fault/warning decode, contactor-via-alarm), SMA (per-MPPT strings, yield, condition/grid-relay),
  DSE (engine + genset electrical + alarms).
- **Troubleshooting timeline (high value):** a rolling event log correlating state changes across
  devices ("12:04:01 PV2 grid-relay Open · 12:04:02 OzTek#1 → discharge 18 kW · 12:04:05 GEN1 start").
  This is what pinpoints cascade faults during commissioning.
- Keep the field-friendly plain-language banners; extend the health engine plant-wide.

## 6. Cross-device health / insight engine

Beyond per-device faults, correlate (no external PCC meter — the Acuvim is not in use, so power
balance is derived from the devices themselves):
- **Power balance** from sum(PV AC + OzTek AC ± genset) at the MDP → detect gross mismatches; the
  genset (DSE) and inverter AC power become the reference points instead of a dedicated meter.
- **Battery ↔ OzTek agreement**: Pylontech current vs OzTek DC current/power (same bus) — mismatch =
  comms or CT problem.
- **Source hand-off**: PV drop → BESS/genset pickup latency; genset run while BESS full = dispatch bug.
- **Fault propagation**: map each device's fault bits to plain English and flag the *root* (e.g. "DC
  under-voltage on OzTek because 2 strings offline").

## 7. Phased implementation

| Phase | Deliverable | Notes |
|---|---|---|
| **0** | `modbustcp` shared helper + `plant` registry + `devices.json` (Totota preload) | refactor BMS onto it |
| **1** | `internal/dse` genset driver (GenComm page 4/6/7) + device tile + detail | genset is live during commissioning; high troubleshooting value |
| **2** | `internal/oztek` (units 1&2, shared-bus serialized) + `internal/sma` (×3) | the inverters — reuse `inverter-modbus-reference.md` |
| **3** | Plant Overview single-line + device tiles + unified health | the new home screen |
| **4** | Troubleshooting timeline + cross-device insights | correlation engine |
| **5** | Remote path (eWon/Solarman) + polish + Win/Mac builds | |
| — | ~~`internal/meter` Acuvim II~~ | **dropped — meter not in use** (map retained in Appendix B if it returns) |

Each phase ends with: `go test` on the decode logic (known raw→value pairs), a live read against the
device on the ARC LAN, and a cross-check of values against the device's own UI (SMA Sunny Explorer,
DSE GenConfig/front panel, Acuvim web, OzTek web).

## 8. Verification & safety
- Read-only enforced in code review; only `bms.IssueRun` may write. New drivers have **no write path**.
- Validate each driver's scaling against the device's native UI before trusting the dashboard.
- Respect ARC: relaxed cadence, one-shot reads, serialized OzTek bus; abort cleanly on timeout.
- Unsigned-binary caveats unchanged (SmartScreen/Gatekeeper).

## 9. Open questions
1. ✅ RESOLVED — SMA Modbus **enabled**, all three on unit ID 3 (fine; separate IPs disambiguate).
2. ✅ RESOLVED — BATT#3 (OzTek #3) is on `.41`, unit ID 3. `.41` now hosts 3 OzTeks; driver reserves it.
3. **Decide the `.41` read strategy** (§3.3): relaxed one-shot (default) vs a read window coordinated
   with ARC. Start with one-shot; ask Anthony whether ARC can offer a pause/lock if data is too lossy.
4. DSE8610: confirm which GenComm pages the controller populates live (see Appendix A) + TCP port 502.
5. Remote access: is the eWon the intended remote path for the whole LAN (vs Solarman for battery)?
6. ~~Acuvim II~~ — meter **not in use**; out of scope.

---

# Appendix A — DSE8610 MKII: independent verification of Codex's research

**Verdict: Codex's transport + control facts are CORRECT, but it missed the public fixed register
map and over-recommended reconfiguring the controller.**

| Codex claim | My verification |
|---|---|
| Modbus TCP, port 502, ≤5 masters, unit 1 @192.168.0.71, RS485=NO | ✅ Confirmed (operator manual + network list) |
| GenComm `register = page×256 + offset` | ✅ Confirmed (SP-228 §standard) |
| Page 16 control: reg 8/9 (4104/4105) = key + ones-complement, FC16 | ✅ Confirmed (SP-228 §8.17) |
| Model-specific full register table is DSE-support-only | ✅ Confirmed |
| "No public fixed map → configure Gencomm pages 166-169" | ⚠️ **Incomplete.** GenComm **SP-228 defines FIXED, PUBLIC, read-only pages 4/5/6/7.** We can read standard instrumentation **without reconfiguring** the controller. |
| "All Gencomm values are 32-bit unsigned" | ⚠️ True only for the *configurable* pages 166-169; the fixed pages 4/6/7 mix 16/32-bit, some signed, with defined scaling. |

**Why this matters:** touching a live, commissioned genset controller's config in GenConfig (to fill
pages 166-169) is risky and needs DSE involvement. **Reading the standard read-only pages 4/6/7 is
non-invasive** and needs no reconfiguration. Recommend that as the primary path; keep Codex's
configurable-page approach as a fallback if the 8610 MKII doesn't populate some fixed offset.

### DSE8610 fixed GenComm map (SP-228, read-only, FC03) — primary path

**Page 4 Basic Instrumentation (register = 1024 + offset):**
| Reg | Item | Scale | Unit | Type |
|---|---|---|---|---|
| 1024 | Oil pressure | ×1 | kPa | U16 |
| 1025 | Coolant temperature | ×1 | °C | S16 |
| 1027 | Fuel level | ×1 | % | U16 |
| 1028 | Charge alternator voltage | ×0.1 | V | U16 |
| 1029 | Engine battery voltage | ×0.1 | V | U16 |
| 1030 | Engine speed | ×1 | RPM | U16 |
| 1031 | Generator frequency | ×0.1 | Hz | U16 |
| 1032–1033 | Generator L1-N voltage | ×0.1 | V | U32 |
| 1034–1035 | Generator L2-N voltage | ×0.1 | V | U32 |
| 1036–1037 | Generator L3-N voltage | ×0.1 | V | U32 |
| 1038–1043 | Gen L1-L2 / L2-L3 / L3-L1 voltage | ×0.1 | V | U32 |
| 1044–1045 | Generator L1 current | ×0.1 | A | U32 |
| 1046–1047 | Generator L2 current | ×0.1 | A | U32 |
| 1048–1049 | Generator L3 current | ×0.1 | A | U32 |
| 1052–1057 | Generator L1/L2/L3 watts | ×1 | W | S32 |
| 1060–1071 | Mains L-N / L-L voltages | ×0.1 | V | U32 |

**Page 6 Derived Instrumentation (register = 1536 + offset):**
| Reg | Item | Scale | Unit | Type |
|---|---|---|---|---|
| 1536–1537 | Generator total watts | ×1 | W | S32 |
| 1544–1545 | Generator total VA | ×1 | VA | S32 |
| 1552–1553 | Generator total Var | ×1 | VAr | S32 |
| 1557 | Generator average power factor | ×0.01 | — | S16 |

**Page 7 Accumulated (register = 1792 + offset):**
| Reg | Item | Scale | Unit | Type |
|---|---|---|---|---|
| 1800–1801 | Generator positive kWh | ×0.1 | kWh | U32 |
| 1802–1803 | Generator negative kWh | ×0.1 | kWh | U32 |
| 1808–1809 | Number of starts | ×1 | count | U32 |
| (page 7) | Engine run hours | confirm offset live | h | U32 |
| 1886–1889 | Battery charging / discharging kWh (microgrid) | ×0.1 | kWh | U32 |

**Alarm/status:** GenComm Page 8 "Alarm conditions" encodes each alarm as a multi-bit nibble
(0 disabled / 1 warning / 2 shutdown / 3 electrical trip …). Decode the alarm strings the DSE front
panel shows (Low oil pressure, High coolant temp, Electrical trip, Bus not live). Confirm exact page-8
offsets against the live controller; expose "common alarm / shutdown / electrical-trip / bus-live".

**Control (page 16, DO NOT WRITE):** reg 4104 = system-control key, reg 4105 = ones-complement, FC16.
Documented for reference only; the monitor never writes it.

---

# Appendix B — AccuEnergy Acuvim II (METER1) Modbus map — NOT IN USE

> ⛔ The Acuvim II meter is **not in use** at the site and is **out of scope** for the build. This map
> is retained only so the driver can be added quickly if the meter is ever commissioned. Skip on
> first delivery.

- **Transport:** Modbus TCP :502 (via AXM-WEB2), unit ID **1** @192.168.0.61. FC03/04 read; 100 ms
  refresh. Real-time metering block is **IEEE-754 float32** (2 regs, big-endian) in the `0x3000` page.

| Reg (hex/dec) | Item | Type | Unit |
|---|---|---|---|
| 0x3000 / 12288 | Frequency | float32 | Hz |
| 0x3002 / 12290 | Voltage V1 (L1-N) | float32 | V |
| 0x3004 / 12292 | Voltage V2 | float32 | V |
| 0x3006 / 12294 | Voltage V3 | float32 | V |
| 0x300A / 12298 | Line voltage V12 | float32 | V |
| 0x300C / 12300 | Line voltage V23 | float32 | V |
| 0x300E / 12302 | Line voltage V31 | float32 | V |
| 0x3012 / 12306 | Current I1 | float32 | A |
| 0x3014 / 12308 | Current I2 | float32 | A |
| 0x3016 / 12310 | Current I3 | float32 | A |
| 0x3022 / 12322 | Total active power | float32 | kW |
| 0x302A / 12330 | Total reactive power | float32 | kVAr |
| 0x3032 / 12338 | Total apparent power | float32 | kVA |
| 0x303A / 12346 | System power factor | float32 | — |

Also available (400+ params): per-phase power, THD (V1 THD @0x... INT16 ×0.01), demand, and the
energy block (import/export kWh — **confirm exact registers** from AccuEnergy Modbus map; commonly in
the `0x4000` accumulated-energy page). PT/CT ratios must be read/confirmed to scale primary values.

## Sources (this plan)
- `10012 Totota Network List` (2026-07-06) — device IPs, unit IDs, converters.
- DSE **GenComm SP-228 standard** (fixed pages 4/6/7/16) — verified extraction; DSE8610 MKII operator
  manual (Modbus TCP :502, ≤5 masters, register table support-only).
- AccuEnergy Acuvim II Modbus map (accuenergy.com; aggsoft/quantumbit register lists).
- `inverter-modbus-reference.md` (SMA + OzTek), reconciled with Codex.
- Codex DSE draft: `.../modbus-integration-reference/generator/dse8610_mkii_generator_modbus_reference.md`.
