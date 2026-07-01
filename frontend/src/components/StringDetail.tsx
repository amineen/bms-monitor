import { motion } from 'framer-motion'
import { X, Crown, AlertTriangle, Info } from 'lucide-react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Cell,
  ReferenceLine,
} from 'recharts'
import type { StringInfo } from '../lib/api'
import { fmt, statusMeta, socColor, balanceMeta, clsx } from '../lib/ui'

function Metric({ label, value, accent }: { label: string; value: string; accent?: string }) {
  return (
    <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 px-3 py-2.5">
      <div className="stat-label">{label}</div>
      <div className={clsx('mono mt-0.5 text-lg font-semibold', accent ?? 'text-slate-100')}>{value}</div>
    </div>
  )
}

function ChartTooltip({ active, payload, label, unit }: any) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg border border-ink-700 bg-ink-900/95 px-2.5 py-1.5 text-xs shadow-panel">
      <div className="text-slate-400">{label}</div>
      <div className="mono font-semibold text-slate-100">
        {payload[0].value} {unit}
      </div>
    </div>
  )
}

export function StringDetail({ s, onClose }: { s: StringInfo; onClose: () => void }) {
  const bal = balanceMeta(s.cellSpreadMV)
  const moduleData = (s.moduleV ?? []).map((v, i) => ({ name: `M${i}`, v, t: s.moduleT?.[i] ?? 0 }))
  const cellData = (s.cellV ?? []).map((v, i) => ({ i, v }))
  const cellMin = cellData.length ? Math.min(...cellData.map((d) => d.v)) : 0
  const cellMax = cellData.length ? Math.max(...cellData.map((d) => d.v)) : 0
  const cellAvg = cellData.length ? cellData.reduce((a, d) => a + d.v, 0) / cellData.length : 0
  const modMin = moduleData.length ? Math.min(...moduleData.map((d) => d.v)) : 0

  const cellColor = (v: number) => {
    const dev = Math.abs(v - cellAvg) * 1000
    if (dev >= 30) return '#F87171'
    if (dev >= 15) return '#FBBF24'
    return '#34D399'
  }

  return (
    <motion.div
      className="fixed inset-0 z-40 flex items-center justify-center p-4"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <div className="absolute inset-0 bg-ink-950/70 backdrop-blur-sm" onClick={onClose} />
      <motion.div
        className="panel relative z-10 flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden"
        initial={{ scale: 0.96, y: 12, opacity: 0 }}
        animate={{ scale: 1, y: 0, opacity: 1 }}
        exit={{ scale: 0.97, opacity: 0 }}
        transition={{ type: 'spring', stiffness: 320, damping: 28 }}
      >
        {/* header */}
        <div className="flex items-center justify-between border-b border-ink-700/70 px-6 py-4">
          <div className="flex items-center gap-3">
            <span className="text-lg font-semibold text-slate-100">String {s.index}</span>
            {s.isMaster && (
              <span className="inline-flex items-center gap-1 rounded-md bg-accent/15 px-2 py-0.5 text-[11px] font-medium text-accent ring-1 ring-accent/30">
                <Crown className="h-3 w-3" /> Master
              </span>
            )}
            <span className="mono text-xs text-slate-500">{s.serial || s.baseHex}</span>
          </div>
          <button onClick={onClose} className="rounded-lg p-1.5 text-slate-400 transition hover:bg-ink-700 hover:text-slate-100">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-5">
          {/* metrics */}
          <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
            <Metric label="Total voltage" value={`${fmt.v(s.totalV)} V`} />
            <Metric label="SOC" value={fmt.pct(s.soc)} accent="text-slate-100" />
            <Metric label="SOH" value={fmt.pct(s.soh)} />
            <Metric label="Current" value={fmt.amp(s.current)} />
            <Metric label="Temperature" value={fmt.temp(s.temp)} />
            <Metric label="Cell max / min" value={`${fmt.v3(s.cellMaxV)}/${fmt.v3(s.cellMinV)}`} />
            <Metric label="Cell spread" value={`${fmt.int(s.cellSpreadMV)} mV`} accent={bal.color} />
            <Metric label="Modules / cells" value={`${s.modules} / ${s.cells}`} />
          </div>

          {/* status / alarms */}
          <div className="mt-4 flex flex-wrap items-center gap-2 text-xs">
            <span className="rounded-md bg-ink-900/60 px-2 py-1 text-slate-300 ring-1 ring-ink-700">{s.basicStatus}</span>
            {s.alarms && s.alarms.length > 0 ? (
              s.alarms.map((al, i) => (
                <span key={i} className="inline-flex items-center gap-1 rounded-md bg-crit/10 px-2 py-1 text-crit ring-1 ring-crit/30">
                  <AlertTriangle className="h-3 w-3" /> {al}
                </span>
              ))
            ) : (
              <span className="rounded-md bg-good/10 px-2 py-1 text-good ring-1 ring-good/30">No active alarms</span>
            )}
          </div>

          {s.hasDetail ? (
            <>
              {/* module voltages */}
              <div className="mt-6">
                <div className="mb-2 flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-slate-200">Module voltages</h3>
                  <span className="mono text-[11px] text-slate-500">{moduleData.length} modules</span>
                </div>
                <div className="h-44 w-full rounded-xl border border-ink-700/50 bg-ink-900/40 p-2">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={moduleData} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
                      <XAxis dataKey="name" tick={{ fill: '#64748B', fontSize: 11 }} axisLine={false} tickLine={false} />
                      <YAxis domain={[Math.floor(modMin - 0.5), 'auto']} tick={{ fill: '#64748B', fontSize: 11 }} axisLine={false} tickLine={false} width={48} />
                      <Tooltip content={(p) => <ChartTooltip {...p} unit="V" />} cursor={{ fill: 'rgba(56,189,248,0.06)' }} />
                      <Bar dataKey="v" radius={[4, 4, 0, 0]} fill="#38BDF8" />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>

              {/* cell distribution */}
              <div className="mt-6">
                <div className="mb-2 flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-slate-200">Cell voltages</h3>
                  <span className="mono text-[11px] text-slate-500">
                    {cellData.length} cells · min {fmt.v3(cellMin)} · max {fmt.v3(cellMax)} · avg {fmt.v3(cellAvg)}
                  </span>
                </div>
                <div className="h-48 w-full rounded-xl border border-ink-700/50 bg-ink-900/40 p-2">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={cellData} margin={{ top: 8, right: 8, left: -16, bottom: 0 }} barCategoryGap={0}>
                      <XAxis dataKey="i" tick={{ fill: '#64748B', fontSize: 10 }} axisLine={false} tickLine={false} interval={31} />
                      <YAxis domain={[cellMin - 0.003, cellMax + 0.003]} tickFormatter={(v) => v.toFixed(2)} tick={{ fill: '#64748B', fontSize: 10 }} axisLine={false} tickLine={false} width={48} />
                      <Tooltip content={(p) => <ChartTooltip {...p} unit="V" />} cursor={{ fill: 'rgba(56,189,248,0.06)' }} />
                      <ReferenceLine y={cellAvg} stroke="#475569" strokeDasharray="3 3" />
                      <Bar dataKey="v" isAnimationActive={false}>
                        {cellData.map((d, i) => (
                          <Cell key={i} fill={cellColor(d.v)} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>
            </>
          ) : (
            <div className="mt-6 flex items-center gap-2 rounded-xl border border-ink-700/60 bg-ink-900/40 px-4 py-4 text-sm text-slate-400">
              <Info className="h-4 w-4 text-accent" />
              {s.isMaster
                ? 'Per-module and per-cell detail wasn’t returned for this string on this read. The summary statistics above are current.'
                : 'On the on-site link, per-module and per-cell detail is published only by the master string. This slave reports the summary statistics shown above.'}
            </div>
          )}
        </div>
      </motion.div>
    </motion.div>
  )
}
