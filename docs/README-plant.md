# TEC Plant Monitor — plant-wide monitoring

The app (formerly "Pylontech BMS Monitor") is now a **plant-wide, read-only monitoring &
troubleshooting tool** for the whole Ageto ARC network at the TEC Totota minigrid. Plug the
laptop into the 192.168.0.0/24 ARC LAN, open the app, and the **Plant** view polls every
energy device; the **Battery** view is the original full Pylontech dashboard (strings, cells,
modules, Solarman remote, gated commissioning Run, Excel/PDF export) — unchanged.

## Device registry

The registry ships bundled (`internal/plant/devices.json`, from the `10012 Totota Network
List`, 2026-07-06) and is copied to a per-user file on first edit
(`~/Library/Application Support/bms-monitor/devices.json` on Mac, `%AppData%\bms-monitor\` on
Windows). Edit that file (or call `SaveDevices`) to add/remove/enable devices; delete it to
restore the bundled defaults.

| ID | Device | Address | Unit | Default |
|---|---|---|---|---|
| bms | Pylontech Force-H3 BMS | 192.168.0.31:502 | 1 | enabled |
| oztek1 / oztek2 | OzTek OZPCS-RS40 PCS | 192.168.0.41:502 | 1 / 2 | enabled |
| oztek3 | OzTek #3 (BATT#3, not wired) | 192.168.0.41:502 | 3 | **disabled** — flip `enabled` when installed |
| pv1 / pv2 / pv3 | SMA STP 25000TL-30 | 192.168.0.51/.52/.53:502 | 3 | enabled |
| gen1 | DSE8610 MKII genset | 192.168.0.71:502 | 1 | enabled |

Excluded: Acuvim II meter (.61, **not in use**), Energport BMS (.104, deprecated), eWon (.21)
and Moxa (.20) infrastructure.

## Read-only guarantee

Every plant driver (`internal/oztek`, `internal/sma`, `internal/dse`) and the shared transport
(`internal/modbustcp`) implement **reads only** — there is no write helper in the transport at
all. The single register write in the entire app remains the gated commissioning Pylontech Run
(`bms.IssueRun`, 0x1094), available only from the Battery view with commissioning mode on.
Specifically never implemented: DSE page-16 control keys, OzTek 41740 controller-heartbeat /
41742-43 / 64308 grid-form commands, SMA setpoints.

## Shared-bus polling policy (OzTek) — OFF by default

The three OzTeks share one RS485 converter (.41) that **ARC actively masters to control the
inverters**. Becoming a second master there is the only place the app could contend with ARC, so
the OzTek devices are flagged `sharedBus` in the registry and **are not polled by default** — the
app issues *zero* Modbus to that bus and cannot conflict with ARC. The battery (its own converter),
genset, and SMAs are read normally, so no battery data is lost.

In the UI the OzTeks show as "Polling off · shared ARC bus" (an accent "ARC-safe" chip, not a red
fault). To collect OzTek inverter data anyway, open the **BESS** page → *Enable polling…* (a
two-step confirm that warns it makes the app a second master on ARC's control bus). Only do this
with ARC coordination, and watch ARC for OzTek comms warnings while it's on. The setting persists in
`polling.json` (`pollSharedBus`, default `false`) and is honored by both the UI and the background
logger. Getting the OzTek data risk-free is still better done by (a) having ARC expose those
registers northbound, or (b) passive listen on the .41 converter — see the plan's open items.

## Polling discipline

- ARC is the primary Modbus master; we are a polite second reader: **one-shot reads, fresh TCP
  connection per block, strictly sequential**, refresh cadence set by the auto-refresh control
  (15 s default; not a tight loop).
- **192.168.0.41 is one RS485 bus** carrying all three OzTeks behind a single USR converter — a
  per-host mutex serializes every read to it. A missed read there marks the unit **stale**
  (previous data shown for up to 90 s) rather than offline; this is expected occasionally while
  ARC is polling. If OzTek data is too lossy during heavy commissioning, disable the OzTek
  entries in the registry while ARC works (the manual "read window") or coordinate a pause with
  the ARC programmer.
- Unreachable devices fail fast (800 ms TCP pre-check) and show as offline tiles; the UI never
  blocks on a dead device.

## Field validation (first time on the ARC LAN)

The decoders are unit-tested against the documented register maps, but the maps carry a few
"confirm on site" items. Compare each device's tile against its native UI:

1. **SMA** (Sunny Explorer / web UI): AC power (30775), MPPT A/B volts, daily yield.
   *Flagged:* the reference doc's generic "PDU = register − 1" note was **not** applied to the
   SMA-native profile (SMA publishes protocol addresses — the common working convention). If
   live values look shifted/garbage, that offset is the first thing to toggle
   (`internal/sma/sma.go`, read start addresses).
2. **OzTek** (web UI at http://192.168.0.41/): operating state 41746, DC volts ≈ combiner
   voltage (~745 V), AC power sign convention (charge vs discharge) — the derived "Site load"
   assumes discharge-positive; if inverted, flip the BESS term in `plant.derivePower`.
3. **DSE8610** (front panel / GenConfig): oil pressure, coolant, RPM, frequency, total kW.
   *Flagged:* engine run-hours offset (page 7 offset 6, seconds) — confirm against the panel;
   also confirm the controller populates fixed pages 4/6/7 (SP-228 standard says yes; a value
   the controller skips shows as "—"). **Alarm names** (GenComm page 8) use the standard DSE 8xxx
   named-alarm order — on the first live alarm, check the app's wording matches the front panel;
   an unmapped index shows as "Alarm N".
4. **Pylontech**: already validated during commissioning (Battery view).

## Datastore (SQLite telemetry log)

The app records plant + battery telemetry to a local SQLite file:
`~/Library/Application Support/bms-monitor/plantlog.db` (Mac) /
`%AppData%\bms-monitor\plantlog.db` (Windows). Pure-Go driver (`modernc.org/sqlite`), WAL mode,
one transaction per tick. Control it from the **Data log** toggle in the top bar (interval
15 s/30 s/1 m/5 m, default **30 s**; retention default **90 days**, pruned daily; settings persist
in `logging.json`).

**Efficiency:** reads the UI already performs are *ingested* rather than re-polled — with
auto-refresh at the logging cadence the logger adds **zero** extra Modbus traffic; the background
ticker only polls a stream itself when nothing has fed it within the interval. All BMS gateway
connections are serialized through one mutex so logger and UI can never collide on the converter.

| Table | Contents | Cadence |
|---|---|---|
| `plant_log` | PV/BESS/genset/load kW, plant health, online count | every tick |
| `oztek_log` / `sma_log` / `dse_log` | one row per **online** device: full key-parameter set incl. raw fault/warning bitfields (offline devices are skipped; `stale=1` flags a held reading) | every tick |
| `bms_log` | battery aggregate: bus V/A/kW, SOC/SOH, combiner + relay state | every tick |
| `bms_string_log` | **one row per enumerated string**: V/A/kW, SOC/SOH/SOE, temps, spread, status/protection — and on the master row the **complete per-cell voltage array (~224 cells)** plus per-module V/T arrays as JSON | every tick |
| `event_log` | every timeline event, exactly once: state changes, comms transitions, and **alarms raised/cleared** — genset named alarms (GenComm page 8: shutdowns, electrical trips, warnings), battery per-string protections + chain drops, OzTek faults/warnings/DER alarms, SMA condition/relay/derating | as detected |

Unpopulated metrics are stored as `NULL` (never 0). Cell arrays unpack in plain SQL, e.g. the
weakest cell over time:
`SELECT ts, MIN(value) FROM bms_string_log, json_each(cell_v) WHERE string=1 GROUP BY ts;`

**Exporting:** the **Logs** button (top bar) writes the history to an Excel workbook — scope
selectable (**Whole system** / Battery detailed / Battery inverters / Solar PV / Genset / Events
only) with a time range (2 h / 12 h / 24 h / 7 d / everything). Each data set is its own sheet;
the Battery Cells sheet expands the per-cell arrays to one column per cell (Cell 1…Cell 224) and
Battery Modules to per-module V/°C columns, ready for charting. Streamed writes keep even a full
90-day export memory-flat.

## Layout

- `internal/modbustcp` — shared read-only transport: fresh-connection-per-read, per-host
  serialization, FC03 (+FC04 fallback), decode helpers.
- `internal/{oztek,sma,dse}` — device drivers, each `Read(endpoint) -> Snapshot` + pure
  `Decode(raw...)` (unit-tested).
- `internal/plant` — registry (bundled `devices.json`), sequential round-robin `Monitor.Read`,
  per-device + plant health, cross-device insights, timeline event diffing.
- `internal/bms` — the original Pylontech driver, plus `ReadSummary` (single-block aggregate
  read used by the plant poll).
- `frontend/src/components/plant/` — Plant overview single-line, device tiles, detail drawers,
  troubleshooting timeline.

Builds: `wails build -platform darwin/universal` and
`wails build -platform windows/amd64 -skipbindings`.
