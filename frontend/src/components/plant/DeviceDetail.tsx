import { motion } from 'framer-motion'
import { X, AlertTriangle, ShieldAlert, Info, BatteryCharging, ArrowRight } from 'lucide-react'
import type { PlantReading, OztekSnapshot, SmaSnapshot, DseSnapshot, BmsAggSummary } from '../../lib/api'
import { clsx, healthTheme, metric, metricKW, fmt } from '../../lib/ui'

function Metric({ label, value, accent }: { label: string; value: string; accent?: string }) {
  return (
    <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 px-3 py-2.5">
      <div className="stat-label">{label}</div>
      <div className={clsx('mono mt-0.5 text-lg font-semibold', accent ?? 'text-slate-100')}>{value}</div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="stat-label mb-2">{title}</div>
      {children}
    </div>
  )
}

function FlagList({
  title,
  items,
  tone,
}: {
  title: string
  items: string[]
  tone: 'crit' | 'warn' | 'info'
}) {
  if (!items?.length) return null
  const cls =
    tone === 'crit'
      ? 'border-crit/30 bg-crit/10 text-crit'
      : tone === 'warn'
        ? 'border-warn/30 bg-warn/10 text-warn'
        : 'border-ink-700 bg-ink-900/40 text-slate-300'
  const Icon = tone === 'crit' ? ShieldAlert : tone === 'warn' ? AlertTriangle : Info
  return (
    <div className={clsx('rounded-xl border px-3.5 py-2.5', cls)}>
      <div className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider">
        <Icon className="h-3.5 w-3.5" /> {title}
      </div>
      <ul className="mt-1 list-inside list-disc text-[13px]">
        {items.map((f) => (
          <li key={f}>{f}</li>
        ))}
      </ul>
    </div>
  )
}

function OztekBody({ o }: { o: OztekSnapshot }) {
  return (
    <div className="flex flex-col gap-5">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Metric label="State" value={o.stateText} accent={o.online ? 'text-good' : o.state === 1 ? 'text-crit' : 'text-slate-100'} />
        <Metric label="Grid" value={o.gridConnected ? (o.gridForming ? 'Forming' : 'Connected') : 'Disconnected'} />
        <Metric label="Heartbeat" value={String(o.heartbeat)} />
        <Metric label="Read time" value={`${o.readMillis} ms`} />
      </div>
      <Section title="AC side">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Metric label="Active power" value={metricKW(o.acPowerW, 2)} />
          <Metric label="Reactive" value={metric(o.reactiveVAR, 0, 'VAr')} />
          <Metric label="Apparent" value={metric(o.apparentVA, 0, 'VA')} />
          <Metric label="Power factor" value={metric(o.powerFactor, 3)} />
          <Metric label="Current" value={metric(o.acCurrentA, 1, 'A')} />
          <Metric label="V line-line" value={metric(o.vLl, 1, 'V')} />
          <Metric label="V line-neutral" value={metric(o.vLn, 1, 'V')} />
          <Metric label="Frequency" value={metric(o.freqHz, 2, 'Hz')} />
        </div>
      </Section>
      <Section title="DC side (battery bus)">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Metric label="Voltage" value={metric(o.dcVoltage, 1, 'V')} />
          <Metric label="Current" value={metric(o.dcCurrent, 1, 'A')} />
          <Metric label="Power" value={metricKW(o.dcPowerW, 2)} />
          <Metric label="Heatsink" value={metric(o.heatsinkTempC, 1, '°C')} />
        </div>
      </Section>
      <FlagList title="Faults" items={o.faults} tone="crit" />
      <FlagList title="Factory faults" items={o.factory} tone="crit" />
      <FlagList title="Warnings" items={o.warnings} tone="warn" />
      <FlagList title="DER alarms" items={o.alarms} tone="warn" />
      {!o.faults?.length && !o.warnings?.length && !o.alarms?.length && (
        <div className="text-[12px] text-good">No active faults, warnings, or alarms.</div>
      )}
    </div>
  )
}

function SmaBody({ s }: { s: SmaSnapshot }) {
  return (
    <div className="flex flex-col gap-5">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Metric
          label="Condition"
          value={s.condition}
          accent={s.condition === 'OK' ? 'text-good' : s.condition === 'Fault' ? 'text-crit' : s.condition === 'Warning' ? 'text-warn' : undefined}
        />
        <Metric label="Grid relay" value={s.gridRelay} accent={s.gridRelay === 'Closed' ? 'text-good' : undefined} />
        <Metric label="Derating" value={s.derating || 'None'} accent={s.derating ? 'text-warn' : undefined} />
        <Metric label="Internal temp" value={metric(s.internalTempC, 1, '°C')} />
      </div>
      <Section title="AC output">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Metric label="Active power" value={metricKW(s.acPowerW, 2)} />
          <Metric label="Apparent" value={metric(s.apparentVA, 0, 'VA')} />
          <Metric label="Frequency" value={metric(s.freqHz, 2, 'Hz')} />
          <Metric label="L1 / L2 / L3" value={`${metric(s.gridV[0], 0)} / ${metric(s.gridV[1], 0)} / ${metric(s.gridV[2], 0)} V`} />
        </div>
      </Section>
      <Section title="DC inputs (MPPT)">
        <div className="grid grid-cols-2 gap-3">
          {[
            { name: 'Input A', m: s.mpptA },
            { name: 'Input B', m: s.mpptB },
          ].map(({ name, m }) => (
            <div key={name} className="rounded-xl border border-ink-700/60 bg-ink-900/40 px-3.5 py-3">
              <div className="stat-label">{name}</div>
              <div className="mono mt-1 text-lg font-semibold text-slate-100">{metricKW(m.powerW, 2)}</div>
              <div className="mono text-[12px] text-slate-400">
                {metric(m.voltageV, 1, 'V')} · {metric(m.currentA, 2, 'A')}
              </div>
            </div>
          ))}
        </div>
      </Section>
      <Section title="Yield">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Metric label="Today" value={metric(s.dailyYieldKWh, 1, 'kWh')} />
          <Metric label="Lifetime" value={metric(s.totalYieldKWh, 0, 'kWh')} />
          <Metric label="Model" value={s.deviceType} />
          <Metric label="Serial" value={String(s.serial || '—')} />
        </div>
      </Section>
    </div>
  )
}

function DseBody({ g }: { g: DseSnapshot }) {
  return (
    <div className="flex flex-col gap-5">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Metric label="Engine" value={g.running ? 'RUNNING' : 'Stopped'} accent={g.running ? 'text-good' : undefined} />
        <Metric label="Speed" value={metric(g.engineRPM, 0, 'RPM')} />
        <Metric label="Total power" value={metricKW(g.totalW, 1)} />
        <Metric label="Frequency" value={metric(g.freqHz, 1, 'Hz')} />
      </div>
      <Section title="Engine">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Metric
            label="Oil pressure"
            value={metric(g.oilPressureKPa, 0, 'kPa')}
            accent={g.running && g.oilPressureKPa.ok && g.oilPressureKPa.value < 100 ? 'text-crit' : undefined}
          />
          <Metric
            label="Coolant"
            value={metric(g.coolantTempC, 0, '°C')}
            accent={g.coolantTempC.ok && g.coolantTempC.value > 98 ? 'text-warn' : undefined}
          />
          <Metric label="Fuel level" value={metric(g.fuelLevelPct, 0, '%')} />
          <Metric label="Start battery" value={metric(g.batteryV, 1, 'V')} />
          <Metric label="Charge alt" value={metric(g.chargeAltV, 1, 'V')} />
          <Metric label="Run hours" value={metric(g.runHours, 1, 'h')} />
          <Metric label="Starts" value={metric(g.starts, 0)} />
          <Metric label="Energy" value={metric(g.posKWh, 1, 'kWh')} />
        </div>
      </Section>
      <Section title="Generator electrical">
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
        <div className="mt-2 grid grid-cols-3 gap-3">
          <Metric label="Total VA" value={metric(g.totalVA, 0)} />
          <Metric label="Total VAr" value={metric(g.totalVAr, 0)} />
          <Metric label="Avg PF" value={metric(g.avgPF, 2)} />
        </div>
      </Section>
    </div>
  )
}

function BmsBody({ b, onOpenBattery }: { b: BmsAggSummary; onOpenBattery?: () => void }) {
  return (
    <div className="flex flex-col gap-5">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Metric label="Strings online" value={`${b.piles} / 6`} accent={b.piles === 6 ? 'text-good' : 'text-warn'} />
        <Metric label="Bus voltage" value={`${fmt.v(b.totalV)} V`} />
        <Metric label="SOC" value={`${b.soc}%`} />
        <Metric label="SOH" value={`${b.soh}%`} />
        <Metric label="Current" value={fmt.amp(b.current)} />
        <Metric label="Power" value={fmt.kw(b.powerKW)} />
        <Metric label="Temperature" value={fmt.temp(b.temp)} />
        <Metric label="Cell max/min" value={`${fmt.v3(b.cellMaxV)} / ${fmt.v3(b.cellMinV)}`} />
      </div>
      <div
        className={clsx(
          'rounded-xl border px-3.5 py-2.5 text-[13px]',
          b.combiner.live ? 'border-good/30 bg-good/10 text-good' : 'border-warn/30 bg-warn/10 text-warn',
        )}
      >
        {b.combiner.live
          ? `Combiner LIVE — ${fmt.v(b.combiner.busV)} V on the DC bus`
          : `Combiner de-energized — ${b.combiner.state} (relays open)`}
      </div>
      {onOpenBattery && (
        <button
          onClick={onOpenBattery}
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-ink-950 transition hover:bg-accent-glow"
        >
          <BatteryCharging className="h-4 w-4" /> Open full battery dashboard (strings, cells, modules)
          <ArrowRight className="h-4 w-4" />
        </button>
      )}
    </div>
  )
}

export function DeviceDetail({
  r,
  onClose,
  onOpenBattery,
}: {
  r: PlantReading
  onClose: () => void
  onOpenBattery?: () => void
}) {
  const off = !r.online && !r.stale
  const t = healthTheme[off ? 'offline' : r.health] ?? healthTheme.offline

  return (
    <motion.div
      className="fixed inset-0 z-40 flex items-center justify-center p-4"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <div className="absolute inset-0 bg-ink-950/70 backdrop-blur-sm" onClick={onClose} />
      <motion.div
        className="panel relative z-10 flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden"
        initial={{ scale: 0.96, y: 12, opacity: 0 }}
        animate={{ scale: 1, y: 0, opacity: 1 }}
        exit={{ scale: 0.97, opacity: 0 }}
        transition={{ type: 'spring', stiffness: 320, damping: 28 }}
      >
        <div className="flex items-center justify-between border-b border-ink-700/70 px-6 py-4">
          <div className="flex items-center gap-3">
            <span className="text-lg font-semibold text-slate-100">{r.name}</span>
            <span className={clsx('inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium ring-1', t.bg, t.ring, t.text)}>
              <span className={clsx('h-1.5 w-1.5 rounded-full', t.dot)} />
              {off ? 'Offline' : r.stale ? 'Stale' : r.headline}
            </span>
            <span className="mono text-xs text-slate-500">
              {r.host}:{r.port} · unit {r.unit}
            </span>
          </div>
          <button onClick={onClose} className="rounded-lg p-1.5 text-slate-400 transition hover:bg-ink-700 hover:text-slate-100">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-5">
          {off ? (
            <div className="flex flex-col items-center gap-2 py-10 text-center">
              <AlertTriangle className="h-8 w-8 text-crit" />
              <div className="text-sm font-medium text-slate-200">No response from this device</div>
              <div className="max-w-sm text-[12px] text-slate-500">
                {r.err || 'Check the device is powered and this computer is on the ARC network (192.168.0.0/24).'}
              </div>
              {r.notes && <div className="mono text-[11px] text-slate-600">{r.notes}</div>}
            </div>
          ) : r.oztek ? (
            <OztekBody o={r.oztek} />
          ) : r.sma ? (
            <SmaBody s={r.sma} />
          ) : r.dse ? (
            <DseBody g={r.dse} />
          ) : r.bms ? (
            <BmsBody b={r.bms} onOpenBattery={onOpenBattery} />
          ) : (
            <div className="py-10 text-center text-sm text-slate-500">{r.notes || 'Device disabled.'}</div>
          )}
        </div>
      </motion.div>
    </motion.div>
  )
}
