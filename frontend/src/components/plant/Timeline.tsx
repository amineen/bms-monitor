import { History } from 'lucide-react'
import type { PlantEvent } from '../../lib/api'
import { clsx, severityTheme } from '../../lib/ui'

function timeOf(at: string): string {
  const d = new Date(at)
  return isNaN(d.getTime()) ? '' : d.toLocaleTimeString()
}

// The troubleshooting timeline: state changes correlated across devices,
// newest first ("12:04:01 PV2 grid relay closed -> open · 12:04:05 GEN1
// started"). This is the cascade-fault view.
export function Timeline({ events }: { events: PlantEvent[] }) {
  return (
    <div className="panel p-5">
      <div className="mb-3 flex items-center gap-2.5">
        <div className="grid h-8 w-8 place-items-center rounded-lg bg-accent/10 ring-1 ring-accent/25">
          <History className="h-4 w-4 text-accent" />
        </div>
        <div>
          <div className="stat-label">Troubleshooting timeline</div>
          <div className="text-sm font-medium text-slate-200">
            {events.length ? `${events.length} state change${events.length === 1 ? '' : 's'} this session` : 'Watching for state changes…'}
          </div>
        </div>
      </div>

      {events.length === 0 ? (
        <div className="py-6 text-center text-[12px] text-slate-500">
          Nothing yet — events appear here when a device changes state between refreshes
          (inverter online/offline, grid relay, genset start/stop, faults raised or cleared).
        </div>
      ) : (
        <div className="max-h-72 overflow-y-auto pr-1">
          <ul className="flex flex-col">
            {events.map((e, i) => {
              const t = severityTheme[e.severity] ?? severityTheme.info
              return (
                <li key={`${e.at}-${e.deviceId}-${i}`} className="flex items-baseline gap-3 border-b border-ink-700/40 py-1.5 last:border-0">
                  <span className="mono w-20 shrink-0 text-[11px] text-slate-500">{timeOf(e.at)}</span>
                  <span className={clsx('mt-1 h-1.5 w-1.5 shrink-0 self-center rounded-full', t.dot)} />
                  <span className="w-32 shrink-0 truncate text-[12px] font-medium text-slate-300">{e.device}</span>
                  <span className={clsx('text-[12px]', t.text)}>{e.text}</span>
                </li>
              )
            })}
          </ul>
        </div>
      )}
    </div>
  )
}
