import { RefreshCw, Radio, CircleSlash, FileSpreadsheet, FileText, HardDrive, Cloud, LogIn, Loader2, LogOut, Maximize2, Wrench, Network, BatteryCharging, Database, Download } from 'lucide-react'
import type { Config, SystemSnapshot, SourceMode, ViewMode, Remote, LoggingStatus, LoggingConfig } from '../lib/api'
import { clsx } from '../lib/ui'

interface Props {
  config: Config
  onChange: (c: Config) => void
  view: ViewMode
  setView: (v: ViewMode) => void
  mode: SourceMode
  setMode: (m: SourceMode) => void
  remote: Remote
  setRemote: (r: Remote) => void
  onConnect: () => void
  onClearSession: () => void
  connecting: boolean
  commissioning: boolean
  setCommissioning: (b: boolean) => void
  logging: LoggingStatus | null
  onSetLogging: (cfg: LoggingConfig) => void
  onExportLogs: () => void
  onMaximise: () => void
  onRefresh: () => void
  loading: boolean
  lastUpdated: Date | null
  auto: boolean
  setAuto: (b: boolean) => void
  intervalSec: number
  setIntervalSec: (n: number) => void
  snapshot: SystemSnapshot | null
  connected: boolean
  onExportExcel: () => void
  onExportPDF: () => void
  exporting: boolean
}

function Field({
  label,
  value,
  onChange,
  width = 'w-28',
  mono = true,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  width?: string
  mono?: boolean
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="stat-label">{label}</span>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && (e.target as HTMLInputElement).blur()}
        className={clsx(
          width,
          mono && 'font-mono',
          'rounded-lg border border-ink-700 bg-ink-900/80 px-3 py-1.5 text-sm text-slate-100',
          'outline-none transition focus:border-accent/60 focus:ring-2 focus:ring-accent/20',
        )}
      />
    </label>
  )
}

export function TopBar(props: Props) {
  const { config, onChange, connected, snapshot } = props
  const batteryView = props.view === 'battery'

  return (
    <header className="z-20 flex flex-wrap items-end gap-x-5 gap-y-3 border-b border-ink-700/70 bg-ink-900/70 px-6 py-3.5 backdrop-blur">
      <div className="flex items-center gap-3 pr-2">
        <div className="grid h-10 w-10 place-items-center rounded-xl bg-accent/15 ring-1 ring-accent/30">
          <Network className="h-5 w-5 text-accent" />
        </div>
        <div>
          <div className="text-[15px] font-semibold leading-tight text-slate-100">TEC Plant Monitor</div>
          <div className="text-[11px] text-slate-500">Totota microgrid · ARC network</div>
        </div>
      </div>

      {/* view */}
      <div className="flex flex-col gap-1">
        <span className="stat-label">View</span>
        <div className="flex rounded-lg border border-ink-700 bg-ink-900/80 p-0.5">
          <button
            onClick={() => props.setView('plant')}
            disabled={props.mode === 'remote'}
            title={props.mode === 'remote' ? 'Plant view needs the on-site ARC network' : 'Whole-plant overview'}
            className={clsx(
              'inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition disabled:opacity-40',
              props.view === 'plant' ? 'bg-accent text-ink-950' : 'text-slate-400 hover:text-slate-200',
            )}
          >
            <Network className="h-3.5 w-3.5" /> Plant
          </button>
          <button
            onClick={() => props.setView('battery')}
            className={clsx(
              'inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition',
              props.view === 'battery' ? 'bg-accent text-ink-950' : 'text-slate-400 hover:text-slate-200',
            )}
          >
            <BatteryCharging className="h-3.5 w-3.5" /> Battery
          </button>
        </div>
      </div>

      {/* source mode */}
      <div className="flex flex-col gap-1">
        <span className="stat-label">Source</span>
        <div className="flex rounded-lg border border-ink-700 bg-ink-900/80 p-0.5">
          <button
            onClick={() => props.setMode('onsite')}
            className={clsx(
              'inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition',
              props.mode === 'onsite' ? 'bg-accent text-ink-950' : 'text-slate-400 hover:text-slate-200',
            )}
          >
            <HardDrive className="h-3.5 w-3.5" /> On-site
          </button>
          <button
            onClick={() => props.setMode('remote')}
            className={clsx(
              'inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition',
              props.mode === 'remote' ? 'bg-accent text-ink-950' : 'text-slate-400 hover:text-slate-200',
            )}
          >
            <Cloud className="h-3.5 w-3.5" /> Remote
          </button>
        </div>
      </div>

      {props.mode === 'onsite' ? (
        batteryView && (
          <>
            <Field label="Gateway IP" value={config.ip} onChange={(v) => onChange({ ...config, ip: v })} width="w-36" />
            <Field label="Port" value={String(config.port)} onChange={(v) => onChange({ ...config, port: parseInt(v) || 0 })} width="w-20" />
            <Field label="Unit" value={String(config.unit)} onChange={(v) => onChange({ ...config, unit: parseInt(v) || 0 })} width="w-16" />
          </>
        )
      ) : (
        <>
          <Field
            label="Station ID"
            value={String(props.remote.stationId)}
            onChange={(v) => props.setRemote({ ...props.remote, stationId: parseInt(v) || 0 })}
            width="w-28"
          />
          <div className="flex flex-col gap-1">
            <span className="stat-label">Solarman session</span>
            <div className="flex items-center gap-1.5">
              <button
                onClick={props.onConnect}
                disabled={props.connecting}
                title="Open the Solarman login in a browser; the app captures your session automatically"
                className={clsx(
                  'inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition',
                  props.remote.token && !props.connecting
                    ? 'border border-good/40 bg-good/10 text-good hover:bg-good/15'
                    : 'bg-accent text-ink-950 hover:bg-accent-glow',
                  props.connecting && 'opacity-70',
                )}
              >
                {props.connecting ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" /> Waiting for login…
                  </>
                ) : props.remote.token ? (
                  <>
                    <Radio className="h-4 w-4" /> Reconnect
                  </>
                ) : (
                  <>
                    <LogIn className="h-4 w-4" /> Connect to Solarman
                  </>
                )}
              </button>
              {props.remote.token && !props.connecting && (
                <button
                  onClick={props.onClearSession}
                  title="Sign out — forget the saved session and start a fresh login next time"
                  className="inline-flex items-center gap-1.5 rounded-lg border border-ink-700 bg-ink-850 px-2.5 py-1.5 text-xs text-slate-400 transition hover:border-crit/50 hover:text-crit"
                >
                  <LogOut className="h-3.5 w-3.5" /> Sign out
                </button>
              )}
            </div>
          </div>
        </>
      )}

      <button
        onClick={props.onRefresh}
        disabled={props.loading}
        className={clsx(
          'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition',
          'bg-accent text-ink-950 hover:bg-accent-glow disabled:opacity-60',
        )}
      >
        <RefreshCw className={clsx('h-4 w-4', props.loading && 'animate-spin')} />
        {props.loading ? 'Reading…' : 'Refresh'}
      </button>

      <div className="ml-auto flex items-end gap-5">
        {props.mode === 'onsite' && batteryView && (
          <label className="flex flex-col gap-1">
            <span className="stat-label">Commissioning</span>
            <button
              onClick={() => props.setCommissioning(!props.commissioning)}
              title="Enable commissioning controls (combiner energization). Writes to the BMS."
              className={clsx(
                'inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-medium transition',
                props.commissioning
                  ? 'border-warn/50 bg-warn/10 text-warn'
                  : 'border-ink-700 bg-ink-900/80 text-slate-400 hover:text-slate-200',
              )}
            >
              <Wrench className="h-3.5 w-3.5" />
              {props.commissioning ? 'On' : 'Off'}
            </button>
          </label>
        )}
        {props.logging && (
          <label className="flex flex-col gap-1">
            <span className="stat-label">Data log</span>
            <div className="flex items-center gap-1.5">
              <button
                onClick={() => props.onSetLogging({ ...props.logging!.config, enabled: !props.logging!.config.enabled })}
                title={
                  props.logging.config.enabled
                    ? `Logging every ${props.logging.config.intervalSec}s to SQLite\n${props.logging.stats?.path ?? ''}\n` +
                      `${((props.logging.stats?.sizeBytes ?? 0) / 1024 / 1024).toFixed(1)} MB · ` +
                      `${Object.values(props.logging.stats?.rows ?? {}).reduce((a, b) => a + b, 0).toLocaleString()} rows · ` +
                      `keeps ${props.logging.config.retentionDays} days`
                    : 'Telemetry logging is off — click to start recording to SQLite'
                }
                className={clsx(
                  'inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-medium transition',
                  props.logging.config.enabled
                    ? 'border-good/40 bg-good/10 text-good'
                    : 'border-ink-700 bg-ink-900/80 text-slate-400 hover:text-slate-200',
                )}
              >
                <Database className="h-3.5 w-3.5" />
                {props.logging.config.enabled ? 'Rec' : 'Off'}
                {props.logging.config.enabled && <span className="h-1.5 w-1.5 rounded-full bg-good animate-pulseSoft" />}
              </button>
              <select
                value={props.logging.config.intervalSec}
                onChange={(e) => props.onSetLogging({ ...props.logging!.config, intervalSec: parseInt(e.target.value) })}
                disabled={!props.logging.config.enabled}
                title="Logging interval"
                className="rounded-lg border border-ink-700 bg-ink-900/80 px-2 py-1 text-xs text-slate-300 outline-none disabled:opacity-40"
              >
                <option value={15}>15s</option>
                <option value={30}>30s</option>
                <option value={60}>1m</option>
                <option value={300}>5m</option>
              </select>
            </div>
          </label>
        )}

        <label className="flex flex-col gap-1">
          <span className="stat-label">Auto-refresh</span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => props.setAuto(!props.auto)}
              className={clsx(
                'relative h-6 w-11 rounded-full transition',
                props.auto ? 'bg-accent' : 'bg-ink-700',
              )}
            >
              <span
                className={clsx(
                  'absolute top-0.5 h-5 w-5 rounded-full bg-white transition-all',
                  props.auto ? 'left-[22px]' : 'left-0.5',
                )}
              />
            </button>
            <select
              value={props.intervalSec}
              onChange={(e) => props.setIntervalSec(parseInt(e.target.value))}
              className="rounded-lg border border-ink-700 bg-ink-900/80 px-2 py-1 text-xs text-slate-300 outline-none"
            >
              <option value={15}>15s</option>
              <option value={30}>30s</option>
              <option value={60}>60s</option>
            </select>
          </div>
        </label>

        <div className="flex items-center gap-2">
          <button
            onClick={props.onExportExcel}
            disabled={!batteryView || !snapshot || props.exporting}
            title="Export Excel log"
            className="inline-flex items-center gap-1.5 rounded-lg border border-ink-700 bg-ink-850 px-3 py-2 text-xs text-slate-300 transition hover:border-accent/50 hover:text-slate-100 disabled:opacity-40"
          >
            <FileSpreadsheet className="h-4 w-4 text-good" /> Excel
          </button>
          <button
            onClick={props.onExportPDF}
            disabled={!batteryView || !snapshot || props.exporting}
            title="Export PDF report"
            className="inline-flex items-center gap-1.5 rounded-lg border border-ink-700 bg-ink-850 px-3 py-2 text-xs text-slate-300 transition hover:border-accent/50 hover:text-slate-100 disabled:opacity-40"
          >
            <FileText className="h-4 w-4 text-crit" /> PDF
          </button>
          <button
            onClick={props.onExportLogs}
            title="Export logged history (SQLite) to Excel — whole system or per equipment"
            className="inline-flex items-center gap-1.5 rounded-lg border border-ink-700 bg-ink-850 px-3 py-2 text-xs text-slate-300 transition hover:border-accent/50 hover:text-slate-100"
          >
            <Download className="h-4 w-4 text-accent" /> Logs
          </button>
          <button
            onClick={props.onMaximise}
            title="Maximise / restore window"
            className="inline-flex items-center rounded-lg border border-ink-700 bg-ink-850 p-2 text-slate-300 transition hover:border-accent/50 hover:text-slate-100"
          >
            <Maximize2 className="h-4 w-4" />
          </button>
        </div>

        <div className="flex flex-col items-end gap-1">
          <div
            className={clsx(
              'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ring-1',
              connected ? 'bg-good/10 text-good ring-good/30' : 'bg-crit/10 text-crit ring-crit/30',
            )}
          >
            {connected ? <Radio className="h-3.5 w-3.5" /> : <CircleSlash className="h-3.5 w-3.5" />}
            {connected ? 'Connected' : 'Offline'}
          </div>
          <span className="text-[11px] text-slate-500">
            {props.lastUpdated ? `Updated ${props.lastUpdated.toLocaleTimeString()}` : 'No data yet'}
          </span>
        </div>
      </div>
    </header>
  )
}
