import type { ReactNode } from 'react'
import { CheckCircle2, AlertTriangle, XCircle, Zap, Gauge as GaugeIcon, Thermometer, Activity, Layers } from 'lucide-react'
import type { SystemSnapshot } from '../lib/api'
import { Gauge } from './Gauge'
import { fmt, healthTheme, clsx } from '../lib/ui'

function Stat({
  icon,
  label,
  value,
  sub,
  accent,
}: {
  icon: ReactNode
  label: string
  value: string
  sub?: string
  accent?: string
}) {
  return (
    <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 px-4 py-3">
      <div className="flex items-center gap-1.5 text-slate-400">
        {icon}
        <span className="stat-label">{label}</span>
      </div>
      <div className={clsx('mono mt-1 text-2xl font-semibold', accent ?? 'text-slate-100')}>{value}</div>
      {sub && <div className="text-[11px] text-slate-500">{sub}</div>}
    </div>
  )
}

const healthIcon: Record<string, ReactNode> = {
  good: <CheckCircle2 className="h-7 w-7" />,
  warn: <AlertTriangle className="h-7 w-7" />,
  critical: <XCircle className="h-7 w-7" />,
  offline: <XCircle className="h-7 w-7" />,
}

export function SystemHero({ snap }: { snap: SystemSnapshot }) {
  const a = snap.aggregate
  const h = snap.health
  const theme = healthTheme[h.level] ?? healthTheme.offline
  const online = snap.chain.online
  const readAt = snap.timestamp ? new Date(snap.timestamp) : null
  const readAtStr = readAt && !isNaN(readAt.getTime()) ? readAt.toLocaleTimeString() : ''

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-[auto,1fr]">
      {/* Gauge + health */}
      <div className="panel flex flex-col items-center justify-between gap-4 p-5">
        <Gauge value={online > 0 ? a.soc : undefined} />
        <div className={clsx('w-full rounded-xl px-4 py-3 ring-1', theme.bg, theme.ring)}>
          <div className={clsx('flex items-center gap-2.5', theme.text)}>
            {healthIcon[h.level]}
            <div>
              <div className="text-sm font-semibold leading-tight">{h.headline}</div>
              <div className="text-[11px] text-slate-400">
                {online} of 6 strings online · link {snap.linkLive ? 'live' : 'static'}
                {readAtStr && <> · as of {readAtStr}</>}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="panel flex flex-col gap-4 p-5">
        <div className="flex items-baseline justify-between">
          <div>
            <div className="stat-label">System</div>
            <div className="font-mono text-3xl font-semibold text-slate-100">
              {online > 0 ? fmt.v(a.totalV) : fmt.v(undefined)} <span className="text-lg text-slate-400">V</span>
            </div>
          </div>
          <div className="text-right text-[11px] text-slate-500">
            <div>{snap.identity.name} · {snap.identity.firmware}</div>
            <div className="mono">{snap.ip}:{snap.port}</div>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <Stat icon={<Layers className="h-3.5 w-3.5" />} label="Strings online" value={`${online} / 6`}
            accent={online === 6 ? 'text-good' : online === 0 ? 'text-crit' : 'text-warn'} />
          <Stat icon={<Activity className="h-3.5 w-3.5" />} label="Current" value={fmt.amp(a.current)}
            sub={a.current > 0 ? 'charging' : a.current < 0 ? 'discharging' : 'idle / no load'} />
          <Stat icon={<Zap className="h-3.5 w-3.5" />} label="Power" value={fmt.kw(a.powerKW)} />
          <Stat icon={<GaugeIcon className="h-3.5 w-3.5" />} label="SOH" value={fmt.pct(online > 0 ? a.soh : undefined)} />
          <Stat icon={<Thermometer className="h-3.5 w-3.5" />} label="Temperature" value={fmt.temp(online > 0 ? a.temp : undefined)} />
          <Stat icon={<GaugeIcon className="h-3.5 w-3.5" />} label="Cell max / min"
            value={online > 0 ? `${fmt.v3(a.cellMaxV)} / ${fmt.v3(a.cellMinV)}` : '—'} sub="volts" />
        </div>

        {h.reasons && h.reasons.length > 0 && (
          <div className="mt-1 flex flex-wrap gap-2">
            {h.reasons.slice(0, 4).map((r, i) => (
              <span key={i} className={clsx('rounded-md px-2 py-1 text-[11px] ring-1', theme.bg, theme.ring, theme.text)}>
                {r}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
