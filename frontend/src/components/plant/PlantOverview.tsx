import { Sun, Fuel, BatteryCharging, Factory, CheckCircle2, AlertTriangle, XCircle } from 'lucide-react'
import type { PlantSnapshot, PlantReading } from '../../lib/api'
import type { DetailTarget } from './detail/DetailPage'
import { clsx, healthTheme, kw } from '../../lib/ui'

// One node on the single-line mimic.
function SourceNode({
  icon,
  label,
  sub,
  kW,
  health,
  onClick,
}: {
  icon: React.ReactNode
  label: string
  sub: string
  kW: number
  health: string
  onClick?: () => void
}) {
  const t = healthTheme[health] ?? healthTheme.offline
  return (
    <button
      onClick={onClick}
      className={clsx(
        'flex w-[200px] items-center gap-3 rounded-xl border border-ink-700/70 bg-ink-900/60 px-3.5 py-3 text-left transition',
        onClick && 'hover:border-accent/50 hover:shadow-glow',
      )}
    >
      <div className={clsx('relative grid h-10 w-10 shrink-0 place-items-center rounded-lg ring-1', t.bg, t.ring)}>
        {icon}
        <span className={clsx('absolute -right-1 -top-1 h-2.5 w-2.5 rounded-full ring-2 ring-ink-900', t.dot)} />
      </div>
      <div className="min-w-0">
        <div className="truncate text-[13px] font-semibold text-slate-100">{label}</div>
        <div className="mono text-lg font-semibold leading-tight text-slate-100">{kw(kW)}</div>
        <div className="truncate text-[10px] text-slate-500">{sub}</div>
      </div>
    </button>
  )
}

// An animated horizontal power bus. Flow runs left->right when kW > 0 (toward
// the MDP) and reverses when negative (e.g. BESS charging).
function Bus({ kW, color }: { kW: number; color: string }) {
  const active = Math.abs(kW) >= 0.2
  return (
    <div className="relative h-[3px] min-w-[30px] flex-1 overflow-hidden rounded-full bg-ink-700/80">
      {active && (
        <div
          className="absolute inset-0 animate-busFlow"
          style={{
            backgroundImage: `repeating-linear-gradient(90deg, ${color} 0 5px, transparent 5px 22px)`,
            backgroundSize: '22px 100%',
            animationDirection: kW >= 0 ? 'normal' : 'reverse',
          }}
        />
      )}
    </div>
  )
}

const plantIcon: Record<string, React.ReactNode> = {
  good: <CheckCircle2 className="h-6 w-6" />,
  warn: <AlertTriangle className="h-6 w-6" />,
  critical: <XCircle className="h-6 w-6" />,
  offline: <XCircle className="h-6 w-6" />,
}

export function PlantOverview({
  snap,
  onOpenGroup,
}: {
  snap: PlantSnapshot
  onOpenGroup: (t: DetailTarget) => void
}) {
  const p = snap.power
  const h = snap.health
  const theme = healthTheme[h.level] ?? healthTheme.offline

  const devs = snap.devices ?? []
  const pvs = devs.filter((d) => d.kind === 'sma')
  const oz = devs.filter((d) => d.kind === 'oztek' && d.enabled)
  const bms = devs.find((d) => d.kind === 'bms')
  const gen = devs.find((d) => d.kind === 'dse')

  const worst = (rs: (PlantReading | undefined)[]): string => {
    const rank: Record<string, number> = { good: 0, warn: 1, critical: 2, offline: 3 }
    let w = 'good'
    let seen = false
    for (const r of rs) {
      if (!r || r.pollSkipped) continue // ignore intentionally-unpolled devices
      seen = true
      const lvl = r.online || r.stale ? r.health : 'offline'
      if ((rank[lvl] ?? 0) > (rank[w] ?? 0)) w = lvl
    }
    return seen ? w : 'offline'
  }

  const pvOnline = pvs.filter((d) => d.online || d.stale).length
  const ozActive = oz.filter((d) => !d.pollSkipped)
  const ozOnline = ozActive.filter((d) => d.online || d.stale).length
  const ozPollOff = oz.length > 0 && ozActive.length === 0
  const socText = bms?.bms ? `SOC ${bms.bms.soc}%` : 'BMS offline'

  return (
    <div className="panel relative overflow-hidden p-5">
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-accent/40 to-transparent" />

      {/* health banner */}
      <div className={clsx('mb-5 flex items-center gap-3 rounded-xl px-4 py-3 ring-1', theme.bg, theme.ring)}>
        <span className={theme.text}>{plantIcon[h.level]}</span>
        <div>
          <div className={clsx('text-sm font-semibold', theme.text)}>{h.headline}</div>
          {h.reasons?.length > 0 && (
            <div className="text-[11px] text-slate-400">{h.reasons.slice(0, 3).join(' · ')}</div>
          )}
        </div>
        <div className="ml-auto text-right">
          <div className="stat-label">Site load (derived)</div>
          <div className="mono text-xl font-semibold text-slate-100">{kw(p.loadKW)}</div>
        </div>
      </div>

      {/* single-line mimic */}
      <div className="flex items-center gap-0">
        <div className="flex flex-col gap-3">
          <SourceNode
            icon={<Sun className="h-5 w-5 text-warn" />}
            label={`Solar PV · ${pvOnline}/${pvs.length}`}
            sub="3× SMA Tripower 25 kVA"
            kW={p.pvKW}
            health={worst(pvs)}
            onClick={() => onOpenGroup('pv')}
          />
          <SourceNode
            icon={<BatteryCharging className="h-5 w-5 text-accent" />}
            label={ozPollOff ? 'BESS · battery' : `BESS · ${ozOnline}/${ozActive.length} PCS`}
            sub={ozPollOff ? `${socText} · PCS polling off` : `Pylontech + OzTek · ${socText}`}
            kW={p.bessKW}
            health={worst([...oz, bms])}
            onClick={() => onOpenGroup('bess')}
          />
          <SourceNode
            icon={<Fuel className="h-5 w-5 text-crit" />}
            label="Genset"
            sub="Perkins 135 kVA · DSE8610"
            kW={p.gensetKW}
            health={worst([gen])}
            onClick={() => onOpenGroup('genset')}
          />
        </div>

        {/* buses */}
        <div className="flex flex-1 flex-col gap-[46px] px-1">
          <Bus kW={p.pvKW} color="#FBBF24" />
          <Bus kW={p.bessKW} color="#38BDF8" />
          <Bus kW={p.gensetKW} color="#F87171" />
        </div>

        {/* MDP / load */}
        <div className="flex w-[190px] flex-col items-center gap-2 rounded-xl border border-ink-700/70 bg-ink-900/60 px-4 py-6">
          <div className="grid h-12 w-12 place-items-center rounded-xl bg-accent/10 ring-1 ring-accent/25">
            <Factory className="h-6 w-6 text-accent" />
          </div>
          <div className="text-[13px] font-semibold text-slate-100">MDP · Totota</div>
          <div className="mono text-2xl font-semibold text-slate-100">{kw(p.loadKW)}</div>
          <div className="text-center text-[10px] leading-snug text-slate-500">
            load derived from sources
            <br />
            (no PCC meter in use)
          </div>
        </div>
      </div>

      {/* insights */}
      {snap.insights?.length > 0 && (
        <div className="mt-4 flex flex-col gap-1.5">
          {snap.insights.map((s, i) => (
            <div
              key={i}
              className="flex items-start gap-2 rounded-lg border border-warn/25 bg-warn/5 px-3 py-2 text-[12px] text-warn"
            >
              <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              {s}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
