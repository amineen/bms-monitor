import { Bar, BarChart, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { DseMetric } from '../../../lib/api'

// PhaseBars visualizes per-phase load (watts) with an imbalance readout — a
// commissioning tech's fastest way to spot a lopsided genset/load.
export function PhaseBars({ watts, unit = 'W' }: { watts: DseMetric[]; unit?: string }) {
  const vals = watts.map((m, i) => ({ name: `L${i + 1}`, v: m?.ok ? m.value : 0, ok: !!m?.ok }))
  const nums = vals.filter((d) => d.ok).map((d) => d.v)
  const avg = nums.length ? nums.reduce((a, b) => a + b, 0) / nums.length : 0
  const max = nums.length ? Math.max(...nums) : 0
  const min = nums.length ? Math.min(...nums) : 0
  const imbalance = avg > 0 ? ((max - min) / avg) * 100 : 0
  const colors = ['#38BDF8', '#22D3EE', '#818CF8']

  return (
    <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 p-3">
      <div className="mb-1 flex items-baseline justify-between">
        <span className="stat-label">Phase balance</span>
        <span
          className="mono text-sm font-semibold"
          style={{ color: imbalance >= 30 ? '#F87171' : imbalance >= 15 ? '#FBBF24' : '#34D399' }}
        >
          {imbalance.toFixed(0)}% spread
        </span>
      </div>
      <ResponsiveContainer width="100%" height={130}>
        <BarChart data={vals} margin={{ top: 6, right: 6, bottom: 0, left: 6 }}>
          <XAxis dataKey="name" tick={{ fill: '#94A3B8', fontSize: 11 }} axisLine={false} tickLine={false} />
          <YAxis hide />
          <Tooltip
            cursor={{ fill: 'rgba(148,163,184,0.08)' }}
            content={({ active, payload }: any) =>
              active && payload?.length ? (
                <div className="rounded-lg border border-ink-700 bg-ink-900/95 px-2.5 py-1.5 text-xs shadow-panel">
                  <div className="text-slate-400">{payload[0].payload.name}</div>
                  <div className="mono font-semibold text-slate-100">
                    {Math.round(payload[0].value).toLocaleString()} {unit}
                  </div>
                </div>
              ) : null
            }
          />
          <Bar dataKey="v" radius={[4, 4, 0, 0]} isAnimationActive={false}>
            {vals.map((_, i) => (
              <Cell key={i} fill={colors[i % colors.length]} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
