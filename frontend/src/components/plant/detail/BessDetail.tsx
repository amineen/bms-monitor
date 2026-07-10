import { useState } from 'react'
import { BatteryCharging, ArrowRight, PlugZap, PowerOff, Link2, Link2Off, ShieldCheck, AlertTriangle } from 'lucide-react'
import type { PlantReading, PlantSnapshot } from '../../../lib/api'
import { clsx, fmt, metric, metricKW } from '../../../lib/ui'
import { Gauge } from '../../Gauge'
import { Big, Metric, Section, FlagList, StatePill, tempAccent } from './parts'
import { Trend, series, seriesSum } from './Trend'

function OztekColumn({ r }: { r: PlantReading }) {
  const off = !r.online && !r.stale
  const o = r.oztek
  return (
    <div className="flex flex-col gap-3 rounded-2xl border border-ink-700/60 bg-ink-900/40 p-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-slate-100">{r.name}</span>
        <StatePill r={r} />
      </div>
      {r.pollSkipped ? (
        <div className="py-6 text-center text-[12px] text-slate-500">
          Not polled — on ARC's shared RS485 bus.
          <br />
          Enable polling below to read it.
        </div>
      ) : off || !o ? (
        <div className="py-6 text-center text-[12px] text-slate-500">
          {!r.enabled ? r.notes || 'Disabled — enable when installed' : r.err || 'No response'}
        </div>
      ) : (
        <>
          <div className={clsx('text-[13px] font-medium', o.online ? 'text-good' : o.state === 1 ? 'text-crit' : 'text-slate-300')}>
            {o.stateText}
          </div>
          <div className="grid grid-cols-2 gap-2.5">
            <Metric label="AC power" value={metricKW(o.acPowerW, 2)} />
            <Metric label="DC power" value={metricKW(o.dcPowerW, 2)} />
            <Metric label="DC voltage" value={metric(o.dcVoltage, 1, 'V')} />
            <Metric label="DC current" value={metric(o.dcCurrent, 1, 'A')} />
            <Metric label="Frequency" value={metric(o.freqHz, 2, 'Hz')} />
            <Metric label="Cabinet" value={metric(o.cabinetTempC, 0, '°C')} accent={tempAccent(o.cabinetTempC.ok ? o.cabinetTempC.value : undefined, 55, 65)} />
          </div>
          <FlagList title="Faults" items={o.faults} tone="crit" />
          <FlagList title="Factory faults" items={o.factory} tone="crit" />
          <FlagList title="Warnings" items={o.warnings} tone="warn" />
          <FlagList title="DER alarms" items={o.alarms} tone="warn" />
          {!o.faults?.length && !o.warnings?.length && !o.alarms?.length && (
            <div className="text-[11px] text-good">No active faults or warnings.</div>
          )}
        </>
      )}
    </div>
  )
}

// SharedBusPanel explains the OzTek bus policy and holds the gated opt-in to
// start polling it. Default (off) means the app issues zero Modbus to the bus
// ARC controls — no possibility of contending with ARC.
function SharedBusPanel({
  pollSharedBus,
  onSetPollSharedBus,
}: {
  pollSharedBus: boolean
  onSetPollSharedBus: (b: boolean) => void
}) {
  const [confirming, setConfirming] = useState(false)
  if (pollSharedBus) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-warn/30 bg-warn/10 px-3.5 py-3">
        <div className="flex items-start gap-2.5 text-warn">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          <div className="text-[12px]">
            <div className="font-semibold">OzTek bus polling is ON — the app is a second master on ARC's control bus.</div>
            <div className="text-warn/80">Only run this with ARC coordination. Watch ARC for OzTek comms warnings; turn off if any appear.</div>
          </div>
        </div>
        <button
          onClick={() => onSetPollSharedBus(false)}
          className="rounded-lg border border-warn/40 bg-warn/10 px-3 py-1.5 text-xs font-semibold text-warn transition hover:bg-warn/20"
        >
          Stop polling (ARC-safe)
        </button>
      </div>
    )
  }
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-accent/25 bg-accent/5 px-3.5 py-3">
      <div className="flex items-start gap-2.5">
        <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-accent" />
        <div className="text-[12px] text-slate-300">
          <div className="font-semibold text-slate-100">Battery inverters are on ARC's shared RS485 bus — polling is off.</div>
          <div className="text-slate-400">
            The app reads the battery (its own converter), genset, and PV, but issues{' '}
            <b className="text-slate-200">zero Modbus</b> to the OzTek bus, so it can never contend with ARC's control of
            the inverters. Battery data is unaffected.
          </div>
        </div>
      </div>
      {!confirming ? (
        <button
          onClick={() => setConfirming(true)}
          className="shrink-0 rounded-lg border border-ink-600 bg-ink-850 px-3 py-1.5 text-xs font-medium text-slate-300 transition hover:border-accent/50 hover:text-slate-100"
        >
          Enable polling…
        </button>
      ) : (
        <div className="flex shrink-0 items-center gap-2">
          <span className="text-[11px] text-warn">Makes the app a 2nd master on ARC's bus. Sure?</span>
          <button
            onClick={() => {
              onSetPollSharedBus(true)
              setConfirming(false)
            }}
            className="rounded-lg bg-warn px-3 py-1.5 text-xs font-semibold text-ink-950 transition hover:bg-warn/90"
          >
            Enable
          </button>
          <button
            onClick={() => setConfirming(false)}
            className="rounded-lg border border-ink-700 bg-ink-850 px-2.5 py-1.5 text-xs text-slate-400 transition hover:text-slate-200"
          >
            Cancel
          </button>
        </div>
      )}
    </div>
  )
}

export function BessDetail({
  bms,
  ozteks,
  history,
  onOpenBattery,
  pollSharedBus,
  onSetPollSharedBus,
}: {
  bms?: PlantReading
  ozteks: PlantReading[]
  history: PlantSnapshot[]
  onOpenBattery: () => void
  pollSharedBus: boolean
  onSetPollSharedBus: (b: boolean) => void
}) {
  const b = bms?.bms
  const bmsOnline = !!bms && (bms.online || bms.stale) && !!b
  const activeOz = ozteks.filter((r) => r.enabled && !r.pollSkipped)
  const ozIds = activeOz.map((r) => r.id)
  const ozPollOff = ozteks.some((r) => r.pollSkipped)

  const bessKW = ozteks.reduce((sum, r) => (r.oztek?.acPowerW.ok ? sum + r.oztek.acPowerW.value / 1000 : sum), 0)
  const busV = bmsOnline ? b!.totalV : undefined
  const combinerLive = bmsOnline && b!.combiner.live

  return (
    <div className="flex flex-col gap-6">
      {/* hero */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Big
          label="BESS AC power"
          value={ozPollOff ? '—' : bessKW.toFixed(1)}
          unit={ozPollOff ? '' : 'kW'}
          accent={ozPollOff ? 'text-slate-500' : undefined}
          sub={ozPollOff ? 'PCS polling off (ARC bus)' : `${activeOz.filter((r) => r.online || r.stale).length}/${activeOz.length} PCS online`}
        />
        <Big label="Battery SOC" value={bmsOnline ? String(b!.soc) : '—'} unit="%" accent={bmsOnline ? undefined : 'text-slate-500'} />
        <Big label="DC bus" value={busV != null ? fmt.v(busV) : '—'} unit="V" />
        <Big label="Combiner" value={bmsOnline ? (combinerLive ? 'LIVE' : b!.combiner.state) : '—'} accent={combinerLive ? 'text-good' : 'text-warn'} />
      </div>

      {/* battery + gauge */}
      <Section title="Battery (Pylontech Force-H3)">
        {bmsOnline ? (
          <div className="grid gap-4 lg:grid-cols-[auto,1fr]">
            <div className="flex items-center justify-center rounded-2xl border border-ink-700/60 bg-ink-900/40 px-6 py-4">
              <Gauge value={b!.soc} size={180} />
            </div>
            <div className="flex flex-col gap-3">
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                <Metric label="Strings online" value={`${b!.piles} / 6`} accent={b!.piles === 6 ? 'text-good' : 'text-warn'} />
                <Metric label="Bus voltage" value={`${fmt.v(b!.totalV)} V`} />
                <Metric label="SOH" value={`${b!.soh}%`} />
                <Metric label="Current" value={fmt.amp(b!.current)} sub={b!.current > 0 ? 'charging' : b!.current < 0 ? 'discharging' : 'idle'} />
                <Metric label="Power" value={fmt.kw(b!.powerKW)} />
                <Metric label="Temperature" value={fmt.temp(b!.temp)} />
                <Metric label="Cell max" value={`${fmt.v3(b!.cellMaxV)} V`} />
                <Metric label="Cell min" value={`${fmt.v3(b!.cellMinV)} V`} />
              </div>
              <div className={clsx('flex items-center gap-2 rounded-xl border px-3.5 py-2.5 text-[13px]', combinerLive ? 'border-good/30 bg-good/10 text-good' : 'border-warn/30 bg-warn/10 text-warn')}>
                {combinerLive ? <PlugZap className="h-4 w-4" /> : <PowerOff className="h-4 w-4" />}
                {combinerLive ? `Combiner LIVE — ${fmt.v(b!.combiner.busV)} V on the DC bus` : `Combiner de-energized — ${b!.combiner.state} (relays open)`}
              </div>
              <button
                onClick={onOpenBattery}
                className="inline-flex items-center justify-center gap-2 rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-ink-950 transition hover:bg-accent-glow"
              >
                <BatteryCharging className="h-4 w-4" /> Open full battery dashboard (strings, cells, modules)
                <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        ) : (
          <div className="rounded-2xl border border-ink-700/60 bg-ink-900/40 py-8 text-center text-sm text-slate-500">
            BMS unreachable — {bms?.err || 'no response from 192.168.0.31'}.
          </div>
        )}
      </Section>

      {/* DC-bus cross-check: battery vs each PCS on the same physical bus */}
      {bmsOnline && activeOz.some((r) => r.oztek?.dcVoltage.ok) && (
        <Section title="DC bus agreement (battery ↔ PCS)">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {activeOz.map((r) => {
              const dv = r.oztek?.dcVoltage
              if (!dv?.ok) return null
              const diff = Math.abs(b!.totalV - dv.value)
              const ok = diff <= 20
              return (
                <div key={r.id} className={clsx('flex items-center gap-2.5 rounded-xl border px-3.5 py-3', ok ? 'border-ink-700/60 bg-ink-900/40' : 'border-crit/30 bg-crit/10')}>
                  {ok ? <Link2 className="h-4 w-4 text-good" /> : <Link2Off className="h-4 w-4 text-crit" />}
                  <div className="min-w-0">
                    <div className="text-[12px] font-medium text-slate-200">{r.name}</div>
                    <div className="mono text-[12px] text-slate-400">
                      PCS {dv.value.toFixed(0)} V vs BMS {b!.totalV.toFixed(0)} V ·{' '}
                      <span className={ok ? 'text-good' : 'text-crit'}>Δ {diff.toFixed(0)} V</span>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
          <div className="mt-1.5 text-[11px] text-slate-500">
            Battery and PCS share the combiner DC bus, so these should match within a few volts. A large gap points to DC wiring, a contactor, or a comms fault.
          </div>
        </Section>
      )}

      {/* per-PCS columns */}
      <Section title="Battery inverters (OzTek OZPCS-RS40)">
        <div className="mb-3">
          <SharedBusPanel pollSharedBus={pollSharedBus} onSetPollSharedBus={onSetPollSharedBus} />
        </div>
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {ozteks.map((r) => (
            <OztekColumn key={r.id} r={r} />
          ))}
        </div>
      </Section>

      {/* trends */}
      <Section title="Session trends">
        <div className="grid gap-3 sm:grid-cols-3">
          <Trend title="BESS AC power" unit="kW" color="#38BDF8" data={seriesSum(history, ozIds, (x) => (x.oztek?.acPowerW.ok ? x.oztek.acPowerW.value / 1000 : null))} />
          {bms && <Trend title="Battery SOC" unit="%" dec={0} color="#34D399" data={series(history, bms.id, (x) => (x.bms ? x.bms.soc : null))} />}
          {bms && <Trend title="DC bus" unit="V" color="#818CF8" data={series(history, bms.id, (x) => (x.bms ? x.bms.totalV : null))} />}
        </div>
      </Section>
    </div>
  )
}
