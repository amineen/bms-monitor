// Formatting + status-color helpers shared across components.

export const dash = '—'

export const fmt = {
  v: (n?: number) => (n == null ? dash : n.toFixed(1)),
  v2: (n?: number) => (n == null ? dash : n.toFixed(2)),
  v3: (n?: number) => (n == null ? dash : n.toFixed(3)),
  int: (n?: number) => (n == null ? dash : String(n)),
  pct: (n?: number) => (n == null ? dash : `${n}%`),
  amp: (n?: number) => (n == null ? dash : `${n > 0 ? '+' : ''}${n.toFixed(2)} A`),
  kw: (n?: number) => (n == null ? dash : `${n > 0 ? '+' : ''}${n.toFixed(2)} kW`),
  temp: (n?: number) => (n == null ? dash : `${n.toFixed(1)}°C`),
}

export type HealthLevel = 'good' | 'warn' | 'critical' | 'offline'

export const healthTheme: Record<string, { text: string; bg: string; ring: string; dot: string; bar: string }> = {
  good: { text: 'text-good', bg: 'bg-good/10', ring: 'ring-good/30', dot: 'bg-good', bar: 'bg-good' },
  warn: { text: 'text-warn', bg: 'bg-warn/10', ring: 'ring-warn/30', dot: 'bg-warn', bar: 'bg-warn' },
  critical: { text: 'text-crit', bg: 'bg-crit/10', ring: 'ring-crit/30', dot: 'bg-crit', bar: 'bg-crit' },
  offline: { text: 'text-slate-400', bg: 'bg-slate-500/10', ring: 'ring-slate-500/30', dot: 'bg-slate-500', bar: 'bg-slate-500' },
}

export function statusMeta(status: string): { label: string; color: string; dot: string; text: string } {
  switch (status) {
    case 'enumerated':
      return { label: 'Online', color: 'good', dot: 'bg-good', text: 'text-good' }
    case 'no-response':
      return { label: 'No response', color: 'critical', dot: 'bg-crit', text: 'text-crit' }
    default:
      return { label: 'Not enumerated', color: 'offline', dot: 'bg-slate-500', text: 'text-slate-400' }
  }
}

// SOC -> gauge color
export function socColor(soc?: number): string {
  if (soc == null) return '#64748B'
  if (soc >= 60) return '#34D399'
  if (soc >= 25) return '#FBBF24'
  return '#F87171'
}

// cell-balance spread (mV) -> color + label
export function balanceMeta(mv?: number): { color: string; label: string } {
  if (mv == null) return { color: 'text-slate-400', label: dash }
  if (mv >= 100) return { color: 'text-crit', label: 'Out of balance' }
  if (mv >= 50) return { color: 'text-warn', label: 'Watch' }
  return { color: 'text-good', label: 'Balanced' }
}

export function clsx(...parts: (string | false | null | undefined)[]): string {
  return parts.filter(Boolean).join(' ')
}
