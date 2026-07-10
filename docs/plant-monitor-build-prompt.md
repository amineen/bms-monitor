# Build prompt — TEC Plant Monitor (one-shot implementation)

> Paste the block below into a fresh session to build the whole thing. It is intentionally
> end-to-end and does not restrict you to phases — read the design docs, then build it all.

---

You are extending an existing **Wails (Go + React/TypeScript) desktop app** at
`bms-monitor/` from a single-device Pylontech BMS monitor into a **plant-wide, read-only
monitoring & troubleshooting tool** for the whole Ageto ARC energy network at the TEC Totota
Liberia microgrid. Build the complete tool in one pass — all device drivers, the orchestrator,
the redesigned plant UI, tests, and Windows + macOS binaries. Use your judgment on ordering; do
not stop after one "phase."

## First, read these (they contain the full design + verified Modbus register maps)
1. `bms-monitor/docs/plant-monitor-development-plan.md` — the architecture, device inventory,
   constraints, UI plan, and appendices with the DSE genset GenComm map. **This is the spec.**
2. `bms-monitor/docs/inverter-modbus-reference.md` — the verified SMA Sunny Tripower and OzTek
   RS40 register maps (use the register numbers/scaling verbatim).
3. The existing code you are extending and must not regress:
   - `bms-monitor/internal/bms/` — the Pylontech driver. **Copy its patterns**: fresh-TCP-
     connection-per-read (`modbus.go`), decode tables (`registers.go`), snapshot model + health
     (`read.go`, `health.go`), exports (`export.go`). Reads are strictly sequential.
   - `bms-monitor/app.go` — Wails-bound methods; `bms-monitor/frontend/src/` — React UI
     (TopBar, SystemHero, ChainStrip with the Pylontech tower visuals, StringDetail, RunModal,
     lib/api.ts, lib/ui.ts, Tailwind theme).

## What to build

**Backend — one driver package per device kind, all Modbus TCP, all read-only:**
- `internal/oztek/` — OzTek OZPCS-RS40 PCS (SunSpec DER models 701–715 + OzTek 64340). Three
  units (IDs 1/2/3) **all behind one RS485→TCP gateway at 192.168.0.41:502**.
- `internal/sma/` — SMA Sunny Tripower 25000TL-30 (SMA-native 30xxx registers, fixed FIX0/1/2/3
  scaling). Three endpoints `192.168.0.51/.52/.53:502`, all unit ID 3.
- `internal/dse/` — DSE8610 MKII genset via **GenComm standard fixed read-only pages 4/6/7**
  (register = page×256+offset; exact map in the plan's Appendix A). `192.168.0.71:502` unit 1.
- `internal/modbustcp/` — shared TCP transport helper (fresh connection, timeout, unit ID,
  FC04→FC03 fallback), generalizing `internal/bms/modbus.go`.
- `internal/plant/` — orchestrator: a device registry seeded from a bundled `devices.json`
  (the network list below), a **sequential poll scheduler**, a unified `PlantSnapshot`, and a
  cross-device health/insight engine (per-device faults decoded to plain English + simple
  cross-device correlation from the plan §6).
- Keep `internal/bms/` as the Pylontech driver; refactor it onto `modbustcp` if clean.

**Bundled device registry (`devices.json`), confirmed 2026-07-06:**
| Name | Kind | Host:port | Unit | Notes |
|---|---|---|---|---|
| Pylontech BMS | bms | 192.168.0.31:502 | 1 | existing |
| OzTek #1 | oztek | 192.168.0.41:502 | 1 | shared bus |
| OzTek #2 | oztek | 192.168.0.41:502 | 2 | shared bus |
| OzTek #3 | oztek | 192.168.0.41:502 | 3 | shared bus (BATT#3; may be offline until installed) |
| SMA PV1 | sma | 192.168.0.51:502 | 3 | |
| SMA PV2 | sma | 192.168.0.52:502 | 3 | |
| SMA PV3 | sma | 192.168.0.53:502 | 3 | |
| Genset | dse | 192.168.0.71:502 | 1 | |
(Energport BMS .104 and the Acuvim meter .61 are NOT in use — exclude. eWon .21 / Moxa .20 are
infra, not polled.)

**Frontend — redesign from single-battery to plant view (keep the dark control-room theme):**
- A **Plant Overview home**: microgrid single-line/mimic (PV → MDP ← Genset, BESS at the PCC)
  with live power + status color per node and energy-flow direction.
- A **device tile** per in-scope device (up to 8) with its 3–4 key numbers + health dot.
- **Device detail drawers** per kind (OzTek AC/DC + state + fault/warning decode; SMA per-MPPT
  strings + yield + condition/grid-relay; DSE engine + genset electrical + alarms). Reuse the
  existing Pylontech detail/tower components for the battery.
- A **troubleshooting timeline** correlating state changes across devices (cascade-fault view).
- Preserve all existing behavior: Pylontech dashboard + tower visuals, Solarman remote mode,
  onsite/remote auto-switch, the gated commissioning Run modal, Excel/PDF export, the macOS
  maximize fix, Sign-out/Connect.

## Hard constraints (do not violate)
- **Read-only everywhere.** The ONLY register write permitted in the whole app is the existing
  gated Pylontech Run (`bms.IssueRun`, 0x1094). New drivers have **no write path** — no FC06/16.
  Do not implement genset control keys, OzTek control registers, or SMA setpoints.
- **ARC is the primary Modbus master and polls continuously**; we are a second, polite reader.
  Use one-shot reads at a relaxed cadence (2–5 s), never tight loops.
- **The 192.168.0.41 OzTek bus is a single shared RS485 line** behind one gateway carrying all
  three OzTek units. Serialize every access to `.41` (per-host mutex, one-in-flight); a miss
  marks a unit "stale," not a crash. Make the cadence + an optional ARC "read window" a config
  toggle. SMAs and the genset are native Ethernet (≤5 masters) and need no serialization.
- Devices will be **unreachable in the dev environment** (not on the ARC LAN). Everything must
  degrade gracefully: show per-device offline/stale states, never block the UI, never panic.

## Testing & verification
- Unit-test every decoder against known raw→engineering pairs (SunSpec scale factors, SMA
  FIX0/1/2/3, DSE ×0.1/×0.01 and 32-bit hi-word-first, signed handling). Follow the style of the
  existing `internal/bms` and `internal/solarman` tests.
- `go vet ./...` and `go test ./...` must pass.
- Build both binaries: `wails build -platform darwin/universal` and
  `wails build -platform windows/amd64 -skipbindings` (Windows needs `-skipbindings`). Frontend
  is tsc-checked by the build; fix all type errors.
- You cannot read live devices here — validate logic by tests + graceful-offline behavior. Note
  in a short README that field validation is: compare each value against the device's own UI
  (SMA Sunny Explorer, DSE GenConfig/front panel, OzTek web at http://192.168.0.41/).

## Deliverables
1. The new driver packages + orchestrator + `devices.json`, wired through Wails bindings.
2. The redesigned plant UI with the overview, tiles, per-device detail, and timeline — existing
   Pylontech features preserved.
3. Passing `go vet` + `go test`; built `bms-monitor.app` and `bms-monitor.exe` in `build/bin/`.
4. A short `docs/README-plant.md` describing the device registry, how to add/edit devices, the
   read-only guarantee, and the field-validation steps.

Rename the app's user-facing title to **"TEC Plant Monitor"** (keep the binary name if simpler).
Ask no clarifying questions unless genuinely blocked — the two design docs are the source of
truth; where they leave a value "confirm on site," implement it with the documented default and
flag it in `README-plant.md`.
