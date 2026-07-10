import { Area, AreaChart, ResponsiveContainer, Tooltip, YAxis } from 'recharts'
import type { PlantSnapshot, PlantReading } from '../../../lib/api'
import { clsx } from '../../../lib/ui'

export interface Pt {
  t: number
  label: string
  v: number | null
}

function online(r?: PlantReading): boolean {
  return !!r && (r.online || r.stale)
}

// series pulls one metric for one device across the session history. Points
// where the device was offline become null (a gap in the line).
export function series(
  history: PlantSnapshot[],
  deviceId: string,
  pick: (r: PlantReading) => number | null | undefined,
): Pt[] {
  return history.map((s) => {
    const r = s.devices.find((d) => d.id === deviceId)
    const d = new Date(s.timestamp)
    let v: number | null = null
    if (online(r)) {
      const val = pick(r as PlantReading)
      v = val == null ? null : val
    }
    return { t: d.getTime(), label: d.toLocaleTimeString(), v }
  })
}

// seriesSum sums a metric across several devices (e.g. total PV output).
export function seriesSum(
  history: PlantSnapshot[],
  deviceIds: string[],
  pick: (r: PlantReading) => number | null | undefined,
): Pt[] {
  return history.map((s) => {
    let sum = 0
    let any = false
    for (const id of deviceIds) {
      const r = s.devices.find((d) => d.id === id)
      if (online(r)) {
        const val = pick(r as PlantReading)
        if (val != null) {
          sum += val
          any = true
        }
      }
    }
    const d = new Date(s.timestamp)
    return { t: d.getTime(), label: d.toLocaleTimeString(), v: any ? sum : null }
  })
}

function TrendTip({ active, payload, unit, dec }: any) {
  if (!active || !payload?.length || payload[0].value == null) return null
  return (
    <div className="rounded-lg border border-ink-700 bg-ink-900/95 px-2.5 py-1.5 text-xs shadow-panel">
      <div className="text-slate-400">{payload[0].payload.label}</div>
      <div className="mono font-semibold text-slate-100">
        {Number(payload[0].value).toFixed(dec)} {unit}
      </div>
    </div>
  )
}

// Trend is a compact session sparkline with a current/min/max readout.
export function Trend({
  title,
  data,
  unit = '',
  color = '#38BDF8',
  dec = 1,
  height = 96,
}: {
  title: string
  data: Pt[]
  unit?: string
  color?: string
  dec?: number
  height?: number
}) {
  const vals = data.filter((d) => d.v != null).map((d) => d.v as number)
  const last = vals.length ? vals[vals.length - 1] : null
  const min = vals.length ? Math.min(...vals) : null
  const max = vals.length ? Math.max(...vals) : null
  const id = 'grad-' + title.replace(/[^a-z0-9]/gi, '')

  return (
    <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 p-3">
      <div className="mb-1 flex items-baseline justify-between">
        <span className="stat-label">{title}</span>
        <span className="mono text-sm font-semibold text-slate-100">
          {last == null ? '—' : last.toFixed(dec)} <span className="text-[11px] text-slate-500">{unit}</span>
        </span>
      </div>
      {vals.length < 2 ? (
        <div className={clsx('flex items-center justify-center text-[11px] text-slate-600')} style={{ height }}>
          Collecting trend… (refresh a few times)
        </div>
      ) : (
        <>
          <ResponsiveContainer width="100%" height={height}>
            <AreaChart data={data} margin={{ top: 4, right: 2, bottom: 0, left: 2 }}>
              <defs>
                <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color} stopOpacity={0.35} />
                  <stop offset="100%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              </defs>
              <YAxis hide domain={['auto', 'auto']} />
              <Tooltip content={<TrendTip unit={unit} dec={dec} />} />
              <Area
                type="monotone"
                dataKey="v"
                stroke={color}
                strokeWidth={2}
                fill={`url(#${id})`}
                connectNulls
                isAnimationActive={false}
                dot={false}
              />
            </AreaChart>
          </ResponsiveContainer>
          <div className="mono mt-1 flex justify-between text-[10px] text-slate-600">
            <span>min {min?.toFixed(dec)}</span>
            <span>max {max?.toFixed(dec)}</span>
          </div>
        </>
      )}
    </div>
  )
}
