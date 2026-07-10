import type { PlantReading, PlantSnapshot } from '../../../lib/api'
import { metric } from '../../../lib/ui'
import { Big, Metric, Section, Bar, FlagList, tempAccent } from './parts'
import { Trend, series } from './Trend'
import { PhaseBars } from './PhaseBars'

// Perkins 135 kVA prime rating (from the SLD): ~108 kW / 135 kVA.
const PRIME_KW = 108
const PRIME_KVA = 135

export function GensetDetail({ r, history }: { r: PlantReading; history: PlantSnapshot[] }) {
  const g = r.dse
  const off = !r.online && !r.stale
  if (off || !g) {
    return (
      <div className="py-16 text-center text-sm text-slate-500">
        Genset unreachable — {r.err || 'no response from 192.168.0.71'}.
      </div>
    )
  }

  const kW = g.totalW.ok ? g.totalW.value / 1000 : 0
  const kVA = g.totalVA.ok ? g.totalVA.value / 1000 : 0
  const loadFrac = kW / PRIME_KW
  const kvaFrac = kVA / PRIME_KVA
  const loadColor = loadFrac >= 0.9 ? '#F87171' : loadFrac >= 0.75 ? '#FBBF24' : '#34D399'

  return (
    <div className="flex flex-col gap-6">
      {/* hero */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Big label="Engine" value={g.running ? 'RUNNING' : 'STOPPED'} accent={g.running ? 'text-good' : 'text-slate-300'} />
        <Big label="Total power" value={kW.toFixed(1)} unit="kW" />
        <Big label="Speed" value={metric(g.engineRPM, 0)} unit="RPM" />
        <Big label="Frequency" value={metric(g.freqHz, 1)} unit="Hz" />
      </div>

      {/* active alarm conditions (GenComm page 8) */}
      <FlagList
        title="Shutdown / trip alarms"
        items={(g.alarms ?? []).filter((a) => a.state === 'Shutdown' || a.state === 'Electrical trip').map((a) => `${a.name} (${a.state})`)}
        tone="crit"
      />
      <FlagList
        title="Warning alarms"
        items={(g.alarms ?? []).filter((a) => a.state === 'Warning').map((a) => a.name)}
        tone="warn"
      />
      <FlagList
        title="Indications"
        items={(g.alarms ?? []).filter((a) => a.state === 'Indication').map((a) => a.name)}
        tone="info"
      />
      {g.alarmsOk && (g.alarms ?? []).length === 0 && (
        <div className="text-[12px] text-good">No active alarm conditions on the controller.</div>
      )}
      {!g.alarmsOk && (
        <div className="text-[12px] text-slate-500">
          Alarm page (GenComm page 8) not answered by the controller — engine telemetry unaffected.
        </div>
      )}

      {/* loading */}
      <div className="rounded-2xl border border-ink-700/60 bg-ink-900/40 px-4 py-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Bar frac={loadFrac} color={loadColor} label={`Real load · ${PRIME_KW} kW prime`} right={`${(loadFrac * 100).toFixed(0)}%  ·  ${kW.toFixed(1)} kW`} />
          <Bar frac={kvaFrac} color="#38BDF8" label={`Apparent load · ${PRIME_KVA} kVA`} right={`${(kvaFrac * 100).toFixed(0)}%  ·  ${kVA.toFixed(1)} kVA`} />
        </div>
      </div>

      {/* trends */}
      <Section title="Session trends">
        <div className="grid gap-3 sm:grid-cols-3">
          <Trend title="Power" unit="kW" color="#34D399" data={series(history, r.id, (x) => (x.dse?.totalW.ok ? x.dse.totalW.value / 1000 : null))} />
          <Trend title="Frequency" unit="Hz" color="#38BDF8" dec={2} data={series(history, r.id, (x) => (x.dse?.freqHz.ok ? x.dse.freqHz.value : null))} />
          <Trend title="Coolant" unit="°C" color="#FBBF24" dec={0} data={series(history, r.id, (x) => (x.dse?.coolantTempC.ok ? x.dse.coolantTempC.value : null))} />
        </div>
      </Section>

      {/* engine */}
      <Section title="Engine">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Metric label="Oil pressure" value={metric(g.oilPressureKPa, 0, 'kPa')} accent={g.running && g.oilPressureKPa.ok && g.oilPressureKPa.value < 100 ? 'text-crit' : undefined} />
          <Metric label="Coolant" value={metric(g.coolantTempC, 0, '°C')} accent={tempAccent(g.coolantTempC.ok ? g.coolantTempC.value : undefined, 90, 98)} />
          <Metric label="Fuel level" value={metric(g.fuelLevelPct, 0, '%')} />
          <Metric label="Start battery" value={metric(g.batteryV, 1, 'V')} accent={g.batteryV.ok && g.batteryV.value < 11.5 ? 'text-warn' : undefined} />
          <Metric label="Charge alternator" value={metric(g.chargeAltV, 1, 'V')} />
          <Metric label="Run hours" value={metric(g.runHours, 1, 'h')} />
          <Metric label="Starts" value={metric(g.starts, 0)} />
          <Metric label="Lifetime energy" value={metric(g.posKWh, 1, 'kWh')} />
        </div>
      </Section>

      {/* electrical */}
      <Section title="Generator electrical">
        <div className="grid gap-3 lg:grid-cols-[1.4fr,1fr]">
          <div className="overflow-hidden rounded-xl border border-ink-700/60">
            <table className="w-full text-[13px]">
              <thead>
                <tr className="bg-ink-900/60 text-left">
                  <th className="px-3 py-2 font-medium text-slate-400">Phase</th>
                  <th className="px-3 py-2 font-medium text-slate-400">V L-N</th>
                  <th className="px-3 py-2 font-medium text-slate-400">V L-L</th>
                  <th className="px-3 py-2 font-medium text-slate-400">Amps</th>
                  <th className="px-3 py-2 font-medium text-slate-400">Watts</th>
                </tr>
              </thead>
              <tbody className="mono">
                {[0, 1, 2].map((i) => (
                  <tr key={i} className="border-t border-ink-700/50">
                    <td className="px-3 py-1.5 text-slate-400">L{i + 1}</td>
                    <td className="px-3 py-1.5 text-slate-100">{metric(g.vLn[i], 1)}</td>
                    <td className="px-3 py-1.5 text-slate-100">{metric(g.vLl[i], 1)}</td>
                    <td className="px-3 py-1.5 text-slate-100">{metric(g.ampsL[i], 1)}</td>
                    <td className="px-3 py-1.5 text-slate-100">{metric(g.wattsL[i], 0)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <PhaseBars watts={g.wattsL} />
        </div>
        <div className="mt-3 grid grid-cols-3 gap-3">
          <Metric label="Total VA" value={metric(g.totalVA, 0)} />
          <Metric label="Total VAr" value={metric(g.totalVAr, 0)} />
          <Metric label="Avg power factor" value={metric(g.avgPF, 2)} />
        </div>
      </Section>
    </div>
  )
}
