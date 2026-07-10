import { Bar, BarChart, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { PlantReading, PlantSnapshot } from '../../../lib/api'
import { clsx, metric, metricKW } from '../../../lib/ui'
import { Big, Metric, Section, StatePill, tempAccent } from './parts'
import { Trend, seriesSum } from './Trend'

// 3× SMA Sunny Tripower 25 kVA = 75 kVA installed PV.
const INSTALLED_KVA = 75

function InverterColumn({ r }: { r: PlantReading }) {
  const off = !r.online && !r.stale
  const s = r.sma
  return (
    <div className="flex flex-col gap-3 rounded-2xl border border-ink-700/60 bg-ink-900/40 p-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-slate-100">{r.name}</span>
        <StatePill r={r} />
      </div>
      {off || !s ? (
        <div className="py-6 text-center text-[12px] text-slate-500">{r.err || 'No response'}</div>
      ) : (
        <>
          <div className="flex items-center justify-between text-[13px]">
            <span className={clsx('font-medium', s.condition === 'OK' ? 'text-good' : s.condition === 'Fault' ? 'text-crit' : s.condition === 'Warning' ? 'text-warn' : 'text-slate-300')}>
              {s.condition}
            </span>
            <span className="mono text-slate-400">relay {s.gridRelay}</span>
          </div>
          <div className="grid grid-cols-2 gap-2.5">
            <Metric label="AC power" value={metricKW(s.acPowerW, 2)} />
            <Metric label="DC power" value={metricKW(s.dcPowerW, 2)} />
            <Metric label="Frequency" value={metric(s.freqHz, 2, 'Hz')} />
            <Metric label="Internal temp" value={metric(s.internalTempC, 0, '°C')} accent={tempAccent(s.internalTempC.ok ? s.internalTempC.value : undefined, 60, 75)} />
          </div>
          <div className="grid grid-cols-2 gap-2.5">
            <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 px-3 py-2">
              <div className="stat-label">MPPT A</div>
              <div className="mono text-sm font-semibold text-slate-100">{metricKW(s.mpptA.powerW, 2)}</div>
              <div className="mono text-[11px] text-slate-500">{metric(s.mpptA.voltageV, 0, 'V')} · {metric(s.mpptA.currentA, 1, 'A')}</div>
            </div>
            <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 px-3 py-2">
              <div className="stat-label">MPPT B</div>
              <div className="mono text-sm font-semibold text-slate-100">{metricKW(s.mpptB.powerW, 2)}</div>
              <div className="mono text-[11px] text-slate-500">{metric(s.mpptB.voltageV, 0, 'V')} · {metric(s.mpptB.currentA, 1, 'A')}</div>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-2.5">
            <Metric label="Today" value={metric(s.dailyYieldKWh, 1, 'kWh')} />
            <Metric label="Lifetime" value={metric(s.totalYieldKWh, 0, 'kWh')} />
          </div>
          {s.derating && <div className="rounded-lg border border-warn/25 bg-warn/5 px-2.5 py-1.5 text-[12px] text-warn">Derating: {s.derating}</div>}
        </>
      )}
    </div>
  )
}

export function PvDetail({ pvs, history }: { pvs: PlantReading[]; history: PlantSnapshot[] }) {
  const online = pvs.filter((r) => r.online || r.stale)
  const totalKW = pvs.reduce((sum, r) => (r.sma?.acPowerW.ok ? sum + r.sma.acPowerW.value / 1000 : sum), 0)
  const todayKWh = pvs.reduce((sum, r) => (r.sma?.dailyYieldKWh.ok ? sum + r.sma.dailyYieldKWh.value : sum), 0)
  const lifeKWh = pvs.reduce((sum, r) => (r.sma?.totalYieldKWh.ok ? sum + r.sma.totalYieldKWh.value : sum), 0)
  const capFrac = totalKW / INSTALLED_KVA

  const compare = pvs.map((r) => ({ name: r.name.replace('SMA ', ''), v: r.sma?.acPowerW.ok ? r.sma.acPowerW.value / 1000 : 0, on: r.online || r.stale }))

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Big label="Total PV output" value={totalKW.toFixed(1)} unit="kW" sub={`${online.length}/${pvs.length} inverters online`} />
        <Big label="Capacity factor" value={(capFrac * 100).toFixed(0)} unit="%" sub={`of ${INSTALLED_KVA} kVA installed`} />
        <Big label="Today" value={todayKWh.toFixed(1)} unit="kWh" />
        <Big label="Lifetime" value={lifeKWh.toFixed(0)} unit="kWh" />
      </div>

      {online.length > 0 && (
        <Section title="Output comparison">
          <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 p-3">
            <ResponsiveContainer width="100%" height={150}>
              <BarChart data={compare} margin={{ top: 6, right: 8, bottom: 0, left: 6 }}>
                <XAxis dataKey="name" tick={{ fill: '#94A3B8', fontSize: 11 }} axisLine={false} tickLine={false} />
                <YAxis hide />
                <Tooltip
                  cursor={{ fill: 'rgba(148,163,184,0.08)' }}
                  content={({ active, payload }: any) =>
                    active && payload?.length ? (
                      <div className="rounded-lg border border-ink-700 bg-ink-900/95 px-2.5 py-1.5 text-xs shadow-panel">
                        <div className="text-slate-400">{payload[0].payload.name}</div>
                        <div className="mono font-semibold text-slate-100">{Number(payload[0].value).toFixed(2)} kW</div>
                      </div>
                    ) : null
                  }
                />
                <Bar dataKey="v" radius={[4, 4, 0, 0]} isAnimationActive={false}>
                  {compare.map((d, i) => (
                    <Cell key={i} fill={d.on ? '#FBBF24' : '#334155'} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
            <div className="mt-1 text-[11px] text-slate-500">
              Inverters see the same sun — a lagging bar points to shading, a string fault, or a tripped input on that unit.
            </div>
          </div>
        </Section>
      )}

      <Section title="Inverters (SMA Sunny Tripower 25000TL-30)">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {pvs.map((r) => (
            <InverterColumn key={r.id} r={r} />
          ))}
        </div>
      </Section>

      <Section title="Session trend">
        <div className="grid gap-3 sm:grid-cols-3">
          <Trend title="Total PV output" unit="kW" color="#FBBF24" data={seriesSum(history, pvs.map((r) => r.id), (x) => (x.sma?.acPowerW.ok ? x.sma.acPowerW.value / 1000 : null))} />
        </div>
      </Section>
    </div>
  )
}
