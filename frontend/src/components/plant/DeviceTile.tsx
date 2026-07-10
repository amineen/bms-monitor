import { Sun, Fuel, BatteryCharging, Zap, ChevronRight, Clock, ShieldCheck } from 'lucide-react'
import type { PlantReading } from '../../lib/api'
import { clsx, healthTheme, metric, metricKW, fmt } from '../../lib/ui'

const kindIcon: Record<string, React.ReactNode> = {
  bms: <BatteryCharging className="h-4 w-4" />,
  oztek: <Zap className="h-4 w-4" />,
  sma: <Sun className="h-4 w-4" />,
  dse: <Fuel className="h-4 w-4" />,
}

function Cell({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="stat-label">{label}</div>
      <div className="mono text-sm font-semibold text-slate-100">{value}</div>
    </div>
  )
}

// The 3-4 numbers that matter, per device kind.
function tileMetrics(r: PlantReading): { label: string; value: string }[] {
  if (r.bms) {
    const b = r.bms
    return [
      { label: 'Strings', value: `${b.piles}/6` },
      { label: 'Bus', value: `${fmt.v(b.totalV)} V` },
      { label: 'SOC', value: `${b.soc}%` },
      { label: 'Power', value: `${b.powerKW?.toFixed(1)} kW` },
    ]
  }
  if (r.oztek) {
    const o = r.oztek
    return [
      { label: 'AC power', value: metricKW(o.acPowerW) },
      { label: 'DC bus', value: metric(o.dcVoltage, 1, 'V') },
      { label: 'DC amps', value: metric(o.dcCurrent, 1, 'A') },
      { label: 'Freq', value: metric(o.freqHz, 2, 'Hz') },
    ]
  }
  if (r.sma) {
    const s = r.sma
    return [
      { label: 'AC power', value: metricKW(s.acPowerW) },
      { label: 'DC power', value: metricKW(s.dcPowerW) },
      { label: 'Today', value: metric(s.dailyYieldKWh, 1, 'kWh') },
      { label: 'Relay', value: s.gridRelay },
    ]
  }
  if (r.dse) {
    const g = r.dse
    return [
      { label: 'Power', value: metricKW(g.totalW) },
      { label: 'Speed', value: metric(g.engineRPM, 0, 'RPM') },
      { label: 'Freq', value: metric(g.freqHz, 1, 'Hz') },
      { label: 'Fuel', value: metric(g.fuelLevelPct, 0, '%') },
    ]
  }
  return []
}

export function DeviceTile({ r, onClick }: { r: PlantReading; onClick: () => void }) {
  const off = !r.online && !r.stale
  const level = r.enabled ? (off ? 'offline' : r.health) : 'offline'
  const t = healthTheme[level] ?? healthTheme.offline
  const metrics = tileMetrics(r)

  // Poll-skipped: on ARC's shared bus with polling off. Show a calm, neutral
  // "not polled" state — this is intentional and safe, not a fault.
  if (r.pollSkipped) {
    return (
      <button
        onClick={onClick}
        className="group panel relative overflow-hidden p-4 text-left transition hover:border-accent/40"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="grid h-7 w-7 place-items-center rounded-lg bg-slate-500/10 text-slate-400 ring-1 ring-slate-500/30">
              {kindIcon[r.kind]}
            </span>
            <span className="text-sm font-semibold text-slate-300">{r.name}</span>
          </div>
          <span className="inline-flex items-center gap-1 rounded-md bg-accent/10 px-1.5 py-0.5 text-[10px] font-medium text-accent ring-1 ring-accent/25">
            <ShieldCheck className="h-2.5 w-2.5" /> ARC-safe
          </span>
        </div>
        <div className="mt-2 text-[12px] font-medium text-slate-400">Polling off — shared ARC bus</div>
        <div className="mt-1 text-[11px] leading-snug text-slate-500">
          Not read, so the app never masters ARC's inverter bus. Enable polling from the BESS page if ARC allows it.
        </div>
        <div className="mt-3 flex items-center justify-between border-t border-ink-700/60 pt-2">
          <span className="mono text-[11px] text-slate-500">{r.host}:{r.port} · id {r.unit}</span>
          <span className="inline-flex items-center gap-0.5 text-[11px] text-slate-500 transition group-hover:text-accent">
            Detail <ChevronRight className="h-3.5 w-3.5" />
          </span>
        </div>
      </button>
    )
  }

  return (
    <button
      onClick={onClick}
      className={clsx(
        'group panel relative overflow-hidden p-4 text-left transition',
        r.enabled && !off ? 'hover:border-accent/50 hover:shadow-glow' : 'opacity-70 hover:opacity-100',
      )}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className={clsx('grid h-7 w-7 place-items-center rounded-lg ring-1', t.bg, t.ring, t.text)}>
            {kindIcon[r.kind]}
          </span>
          <span className="text-sm font-semibold text-slate-200">{r.name}</span>
        </div>
        <span className="flex items-center gap-1.5">
          {r.stale && (
            <span className="inline-flex items-center gap-1 rounded-md bg-warn/10 px-1.5 py-0.5 text-[10px] font-medium text-warn ring-1 ring-warn/30">
              <Clock className="h-2.5 w-2.5" /> stale
            </span>
          )}
          <span className={clsx('h-2 w-2 rounded-full', t.dot, !off && r.enabled && 'animate-pulseSoft')} />
        </span>
      </div>

      <div className={clsx('mt-1.5 truncate text-[12px]', t.text)}>{r.headline || (off ? 'Unreachable' : '')}</div>

      {!off && r.enabled && metrics.length > 0 ? (
        <div className="mt-3 grid grid-cols-4 gap-2">
          {metrics.map((m) => (
            <Cell key={m.label} label={m.label} value={m.value} />
          ))}
        </div>
      ) : (
        <div className="mt-3 text-[12px] text-slate-500">
          {!r.enabled ? r.notes || 'Disabled in the device registry' : r.err || 'No response'}
        </div>
      )}

      <div className="mt-3 flex items-center justify-between border-t border-ink-700/60 pt-2">
        <span className="mono text-[11px] text-slate-500">
          {r.host}:{r.port} · id {r.unit}
        </span>
        <span className="inline-flex items-center gap-0.5 text-[11px] text-slate-500 transition group-hover:text-accent">
          Detail <ChevronRight className="h-3.5 w-3.5" />
        </span>
      </div>
    </button>
  )
}
