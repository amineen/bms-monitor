import { ArrowLeft, Sun, BatteryCharging, Fuel, History } from 'lucide-react'
import type { PlantSnapshot } from '../../../lib/api'
import { clsx, severityTheme } from '../../../lib/ui'
import { GensetDetail } from './GensetDetail'
import { BessDetail } from './BessDetail'
import { PvDetail } from './PvDetail'

export type DetailTarget = 'pv' | 'bess' | 'genset'

export function targetForKind(kind: string): DetailTarget {
  if (kind === 'sma') return 'pv'
  if (kind === 'dse') return 'genset'
  return 'bess' // bms + oztek share the DC bus
}

const meta: Record<DetailTarget, { title: string; sub: string; icon: React.ReactNode; ids: (s: PlantSnapshot) => string[] }> = {
  pv: {
    title: 'Solar PV',
    sub: '3× SMA Sunny Tripower 25 kVA',
    icon: <Sun className="h-5 w-5 text-warn" />,
    ids: (s) => s.devices.filter((d) => d.kind === 'sma').map((d) => d.id),
  },
  bess: {
    title: 'Battery Energy Storage',
    sub: 'Pylontech Force-H3 + OzTek OZPCS-RS40',
    icon: <BatteryCharging className="h-5 w-5 text-accent" />,
    ids: (s) => s.devices.filter((d) => d.kind === 'bms' || d.kind === 'oztek').map((d) => d.id),
  },
  genset: {
    title: 'Genset',
    sub: 'Perkins 135 kVA · DSE8610 MKII',
    icon: <Fuel className="h-5 w-5 text-crit" />,
    ids: (s) => s.devices.filter((d) => d.kind === 'dse').map((d) => d.id),
  },
}

function RelatedEvents({ snap, ids }: { snap: PlantSnapshot; ids: string[] }) {
  const set = new Set(ids)
  const events = (snap.events ?? []).filter((e) => set.has(e.deviceId)).slice(0, 8)
  if (events.length === 0) return null
  return (
    <div className="panel p-4">
      <div className="mb-2 flex items-center gap-2 text-slate-300">
        <History className="h-4 w-4 text-accent" />
        <span className="stat-label">Recent events</span>
      </div>
      <ul className="flex flex-col">
        {events.map((e, i) => {
          const t = severityTheme[e.severity] ?? severityTheme.info
          const time = new Date(e.at)
          return (
            <li key={i} className="flex items-baseline gap-3 border-b border-ink-700/40 py-1.5 last:border-0">
              <span className="mono w-20 shrink-0 text-[11px] text-slate-500">{isNaN(time.getTime()) ? '' : time.toLocaleTimeString()}</span>
              <span className={clsx('h-1.5 w-1.5 shrink-0 self-center rounded-full', t.dot)} />
              <span className="w-32 shrink-0 truncate text-[12px] font-medium text-slate-300">{e.device}</span>
              <span className={clsx('text-[12px]', t.text)}>{e.text}</span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}

export function DetailPage({
  target,
  snap,
  history,
  onBack,
  onOpenBattery,
  pollSharedBus,
  onSetPollSharedBus,
}: {
  target: DetailTarget
  snap: PlantSnapshot
  history: PlantSnapshot[]
  onBack: () => void
  onOpenBattery: () => void
  pollSharedBus: boolean
  onSetPollSharedBus: (b: boolean) => void
}) {
  const m = meta[target]
  const devices = snap.devices

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-4 animate-fadeUp">
      <div className="flex items-center gap-3">
        <button
          onClick={onBack}
          className="inline-flex items-center gap-1.5 rounded-lg border border-ink-700 bg-ink-850 px-3 py-2 text-sm text-slate-300 transition hover:border-accent/50 hover:text-slate-100"
        >
          <ArrowLeft className="h-4 w-4" /> Plant
        </button>
        <div className="flex items-center gap-2.5">
          <div className="grid h-10 w-10 place-items-center rounded-xl bg-ink-800 ring-1 ring-ink-700">{m.icon}</div>
          <div>
            <div className="text-lg font-semibold text-slate-100">{m.title}</div>
            <div className="text-[12px] text-slate-500">{m.sub}</div>
          </div>
        </div>
      </div>

      <div className="panel p-5">
        {target === 'genset' ? (
          <GensetDetail r={devices.find((d) => d.kind === 'dse')!} history={history} />
        ) : target === 'pv' ? (
          <PvDetail pvs={devices.filter((d) => d.kind === 'sma')} history={history} />
        ) : (
          <BessDetail
            bms={devices.find((d) => d.kind === 'bms')}
            ozteks={devices.filter((d) => d.kind === 'oztek')}
            history={history}
            onOpenBattery={onOpenBattery}
            pollSharedBus={pollSharedBus}
            onSetPollSharedBus={onSetPollSharedBus}
          />
        )}
      </div>

      <RelatedEvents snap={snap} ids={m.ids(snap)} />
    </div>
  )
}
