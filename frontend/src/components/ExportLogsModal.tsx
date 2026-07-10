import { useState } from 'react'
import { motion } from 'framer-motion'
import { X, Download, Loader2, Network, BatteryCharging, Zap, Sun, Fuel, History } from 'lucide-react'
import { api } from '../lib/api'
import { clsx } from '../lib/ui'

const SCOPES = [
  { id: 'system', label: 'Whole system', sub: 'Plant balance + every equipment + events', icon: <Network className="h-4 w-4" /> },
  { id: 'battery', label: 'Battery (detailed)', sub: 'Aggregate, per-string, per-cell & module arrays', icon: <BatteryCharging className="h-4 w-4" /> },
  { id: 'oztek', label: 'Battery inverters', sub: 'OzTek PCS AC/DC, states, fault bitfields', icon: <Zap className="h-4 w-4" /> },
  { id: 'pv', label: 'Solar PV', sub: 'SMA output, MPPT, yields, conditions', icon: <Sun className="h-4 w-4" /> },
  { id: 'genset', label: 'Genset', sub: 'Engine + electrical + accumulated', icon: <Fuel className="h-4 w-4" /> },
  { id: 'events', label: 'Events only', sub: 'The troubleshooting timeline', icon: <History className="h-4 w-4" /> },
]

const RANGES = [
  { hours: 2, label: 'Last 2 hours' },
  { hours: 12, label: 'Last 12 hours' },
  { hours: 24, label: 'Last 24 hours' },
  { hours: 24 * 7, label: 'Last 7 days' },
  { hours: 0, label: 'Everything' },
]

export function ExportLogsModal({
  onClose,
  onDone,
}: {
  onClose: () => void
  onDone: (msg: string, ok: boolean) => void
}) {
  const [scope, setScope] = useState('system')
  const [hours, setHours] = useState(24)
  const [busy, setBusy] = useState(false)

  const onExport = async () => {
    setBusy(true)
    try {
      const path = await api.ExportLogs(scope, hours)
      if (path) {
        onDone(`Exported ${path}`, true)
        onClose()
      }
    } catch (e: any) {
      onDone('Export failed: ' + (e?.message ?? e), false)
    } finally {
      setBusy(false)
    }
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
        className="panel relative z-10 flex w-full max-w-md flex-col overflow-hidden"
        initial={{ scale: 0.96, y: 12, opacity: 0 }}
        animate={{ scale: 1, y: 0, opacity: 1 }}
        exit={{ scale: 0.97, opacity: 0 }}
        transition={{ type: 'spring', stiffness: 320, damping: 28 }}
      >
        <div className="flex items-center justify-between border-b border-ink-700/70 px-5 py-4">
          <div className="flex items-center gap-2.5">
            <Download className="h-5 w-5 text-accent" />
            <span className="text-lg font-semibold text-slate-100">Export logged history</span>
          </div>
          <button onClick={onClose} className="rounded-lg p-1.5 text-slate-400 transition hover:bg-ink-700 hover:text-slate-100">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex flex-col gap-4 px-5 py-4">
          <div>
            <div className="stat-label mb-2">Equipment</div>
            <div className="flex flex-col gap-1.5">
              {SCOPES.map((s) => (
                <button
                  key={s.id}
                  onClick={() => setScope(s.id)}
                  className={clsx(
                    'flex items-center gap-3 rounded-xl border px-3.5 py-2.5 text-left transition',
                    scope === s.id
                      ? 'border-accent/60 bg-accent/10 ring-1 ring-accent/30'
                      : 'border-ink-700/60 bg-ink-900/40 hover:border-ink-600',
                  )}
                >
                  <span className={scope === s.id ? 'text-accent' : 'text-slate-400'}>{s.icon}</span>
                  <span className="min-w-0">
                    <span className="block text-[13px] font-medium text-slate-100">{s.label}</span>
                    <span className="block truncate text-[11px] text-slate-500">{s.sub}</span>
                  </span>
                </button>
              ))}
            </div>
          </div>

          <div>
            <div className="stat-label mb-2">Time range</div>
            <div className="flex flex-wrap gap-1.5">
              {RANGES.map((r) => (
                <button
                  key={r.hours}
                  onClick={() => setHours(r.hours)}
                  className={clsx(
                    'rounded-lg border px-3 py-1.5 text-xs font-medium transition',
                    hours === r.hours
                      ? 'border-accent/60 bg-accent/10 text-accent'
                      : 'border-ink-700 bg-ink-900/60 text-slate-400 hover:text-slate-200',
                  )}
                >
                  {r.label}
                </button>
              ))}
            </div>
          </div>

          <button
            onClick={onExport}
            disabled={busy}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-ink-950 transition hover:bg-accent-glow disabled:opacity-60"
          >
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
            {busy ? 'Writing workbook…' : 'Export to Excel'}
          </button>
          <div className="text-center text-[11px] text-slate-500">
            One .xlsx workbook — a sheet per data set; battery cells expanded one column per cell.
          </div>
        </div>
      </motion.div>
    </motion.div>
  )
}
