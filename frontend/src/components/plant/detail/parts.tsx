import type { ReactNode } from 'react'
import { AlertTriangle, ShieldAlert, Info } from 'lucide-react'
import { clsx } from '../../../lib/ui'

// ---- shared building blocks for the full detail pages ----

export function Section({ title, right, children }: { title: string; right?: ReactNode; children: ReactNode }) {
  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <div className="stat-label">{title}</div>
        {right}
      </div>
      {children}
    </div>
  )
}

export function Metric({
  label,
  value,
  sub,
  accent,
}: {
  label: string
  value: string
  sub?: string
  accent?: string
}) {
  return (
    <div className="rounded-xl border border-ink-700/60 bg-ink-900/40 px-3 py-2.5">
      <div className="stat-label">{label}</div>
      <div className={clsx('mono mt-0.5 text-lg font-semibold', accent ?? 'text-slate-100')}>{value}</div>
      {sub && <div className="text-[11px] text-slate-500">{sub}</div>}
    </div>
  )
}

// Big hero number.
export function Big({
  label,
  value,
  unit,
  accent,
  sub,
}: {
  label: string
  value: string
  unit?: string
  accent?: string
  sub?: string
}) {
  return (
    <div className="rounded-2xl border border-ink-700/60 bg-ink-900/40 px-4 py-3.5">
      <div className="stat-label">{label}</div>
      <div className={clsx('mono mt-1 text-3xl font-semibold leading-none', accent ?? 'text-slate-100')}>
        {value}
        {unit && <span className="ml-1 text-lg text-slate-400">{unit}</span>}
      </div>
      {sub && <div className="mt-1 text-[11px] text-slate-500">{sub}</div>}
    </div>
  )
}

// Horizontal capacity/load bar (0..1 fraction).
export function Bar({
  frac,
  color = '#38BDF8',
  label,
  right,
}: {
  frac: number
  color?: string
  label?: string
  right?: string
}) {
  const pct = Math.max(0, Math.min(1, frac)) * 100
  return (
    <div>
      {(label || right) && (
        <div className="mb-1 flex items-baseline justify-between text-[11px]">
          <span className="text-slate-400">{label}</span>
          <span className="mono text-slate-300">{right}</span>
        </div>
      )}
      <div className="h-2 w-full overflow-hidden rounded-full bg-ink-900">
        <div className="h-full rounded-full transition-all duration-700" style={{ width: `${pct}%`, background: color }} />
      </div>
    </div>
  )
}

export function FlagList({ title, items, tone }: { title: string; items?: string[]; tone: 'crit' | 'warn' | 'info' }) {
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

// A small status pill (online / stale / offline / disabled / polling-off).
export function StatePill({ r }: { r: { online: boolean; stale: boolean; enabled: boolean; health: string; pollSkipped?: boolean } }) {
  const off = !r.online && !r.stale
  const level = r.pollSkipped ? 'muted' : !r.enabled ? 'offline' : off ? 'offline' : r.health
  const label = r.pollSkipped ? 'Polling off' : !r.enabled ? 'Disabled' : off ? 'Offline' : r.stale ? 'Stale' : 'Online'
  const map: Record<string, string> = {
    good: 'bg-good/10 text-good ring-good/30',
    warn: 'bg-warn/10 text-warn ring-warn/30',
    critical: 'bg-crit/10 text-crit ring-crit/30',
    offline: 'bg-slate-500/10 text-slate-400 ring-slate-500/30',
    muted: 'bg-accent/10 text-accent ring-accent/25',
  }
  return (
    <span className={clsx('inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium ring-1', map[level] ?? map.offline)}>
      <span className={clsx('h-1.5 w-1.5 rounded-full', level === 'good' ? 'bg-good' : level === 'warn' ? 'bg-warn' : level === 'critical' ? 'bg-crit' : level === 'muted' ? 'bg-accent' : 'bg-slate-500')} />
      {label}
    </span>
  )
}

// Threshold coloring helpers.
export function tempAccent(t?: number, warn = 90, crit = 100): string | undefined {
  if (t == null) return undefined
  if (t >= crit) return 'text-crit'
  if (t >= warn) return 'text-warn'
  return undefined
}
