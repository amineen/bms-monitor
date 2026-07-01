# Pylontech BMS Monitor

A native desktop app (Go + React via [Wails](https://wails.io)) that lets on-site
technicians read the 6-string Pylontech Force-H3 BESS in one click — no command
line, no Python. It wraps the same proven Modbus logic as `read_pylontech_full.py`.

**Read-only.** The app never writes a register.

## What it shows
- **System hero** — SOC gauge, total voltage, current/power, SOH, temperature,
  and a plain-language health banner (green / amber / red).
- **Daisy-chain view** — which of the 6 strings are enumerated and where the
  chain stops (the commissioning view).
- **Per-string cards** — voltage, SOC, SOH, temperature, cell balance, serial.
- **String detail** — module-voltage bars and the 224-cell distribution with
  balance colouring, plus decoded alarms (master string has full cell detail;
  slaves report summary stats).
- **Export** — one-click Excel (same 15-column `bms_measurements` format) and a
  one-page PDF summary.

## Architecture
- `internal/bms` — Go port of the register map, scales, decoders, and the
  **fresh-connection-per-read** logic (the converter desync workaround). All
  Modbus reads are **sequential** — never concurrent.
- `app.go` — Wails-bound methods (`ReadSystem`, `TestConnection`, `ExportExcel`,
  `ExportPDF`, `GetConfig`/`SaveConfig`); Go structs become typed TS automatically.
- `frontend/` — React + TypeScript + Tailwind (dark "control-room" theme),
  Recharts, lucide-react, framer-motion.

## Prerequisites (dev only)
- Go 1.21+, Node 18+, and the Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- `wails doctor` checks platform deps (Windows: WebView2 — built into Win 10/11).

## Develop
On a machine that can reach the converter (the ARC subnet):
```
wails dev
```
Hot-reloads the frontend; Go methods are live. Defaults to gateway `192.168.0.31:502`, unit `1`.

## Build
```
# host platform (macOS here)
wails build

# Windows .exe — skip bindings when cross-compiling from macOS/Linux
wails build -platform windows/amd64 -skipbindings
```
Output lands in `build/bin/` (`bms-monitor.app`, `bms-monitor.exe`). The Windows
build is a single self-contained executable.

## Deploy to a technician
Copy `bms-monitor.exe` (Windows) to the laptop on the converter's network and
double-click. No install, no dependencies. It remembers the last gateway IP.

> macOS note: an unsigned `.app` triggers Gatekeeper — right-click → Open the
> first time (code-signing/notarization is a future step).

## Tests
```
go test ./internal/bms/
```
