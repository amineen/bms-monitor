import { Crown, ChevronRight, Thermometer, AlertTriangle } from 'lucide-react'
import type { StringInfo } from '../lib/api'
import { fmt, statusMeta, socColor, balanceMeta, clsx } from '../lib/ui'

export function StringCard({ s, onClick }: { s: StringInfo; onClick: () => void }) {
  const m = statusMeta(s.status)
  const enumerated = s.status === 'enumerated'
  const bal = balanceMeta(s.cellSpreadMV)

  return (
    <button
      onClick={onClick}
      className={clsx(
        'group panel relative overflow-hidden p-4 text-left transition',
        enumerated ? 'hover:border-accent/50 hover:shadow-glow' : 'opacity-80 hover:opacity-100',
      )}
    >
      {/* header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-semibold text-slate-200">String {s.index}</span>
          {s.isMaster && (
            <span className="inline-flex items-center gap-1 rounded-md bg-accent/15 px-1.5 py-0.5 text-[10px] font-medium text-accent ring-1 ring-accent/30">
              <Crown className="h-3 w-3" /> Master
            </span>
          )}
        </div>
        <span className={clsx('inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium', m.text)}>
          <span className={clsx('h-1.5 w-1.5 rounded-full', m.dot, enumerated && 'animate-pulseSoft')} />
          {m.label}
        </span>
      </div>

      {!enumerated ? (
        <div className="mt-6 mb-3 flex items-center gap-2 text-slate-500">
          {s.status === 'no-response' ? <AlertTriangle className="h-4 w-4 text-crit" /> : null}
          <span className="text-sm">
            {s.status === 'no-response' ? 'Not responding' : 'Not enumerated'}
          </span>
        </div>
      ) : (
        <>
          {/* SOC + voltage */}
          <div className="mt-3 flex items-end justify-between">
            <div className="mono text-3xl font-semibold" style={{ color: socColor(s.soc) }}>
              {s.soc}
              <span className="ml-0.5 text-base text-slate-400">%</span>
            </div>
            <div className="text-right">
              <div className="mono text-xl font-semibold text-slate-100">{fmt.v(s.totalV)} V</div>
              <div className="text-[11px] text-slate-500">SOH {fmt.pct(s.soh)}</div>
            </div>
          </div>

          {/* soc bar */}
          <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-ink-900">
            <div className="h-full rounded-full transition-all" style={{ width: `${s.soc}%`, background: socColor(s.soc) }} />
          </div>

          {/* footer metrics */}
          <div className="mt-3 flex items-center justify-between text-[11px]">
            <span className="inline-flex items-center gap-1 text-slate-400">
              <Thermometer className="h-3 w-3" /> {fmt.temp(s.temp)}
            </span>
            <span className={clsx('font-medium', bal.color)}>
              {fmt.int(s.cellSpreadMV)} mV · {bal.label}
            </span>
            <span className="text-slate-500">{s.modules}/{s.cells}</span>
          </div>
        </>
      )}

      <div className="mt-3 flex items-center justify-between border-t border-ink-700/60 pt-2">
        <span className="mono truncate text-[11px] text-slate-500">{s.serial || s.baseHex}</span>
        {enumerated && (
          <span className="inline-flex items-center gap-0.5 text-[11px] text-slate-500 transition group-hover:text-accent">
            Detail <ChevronRight className="h-3.5 w-3.5" />
          </span>
        )}
      </div>
    </button>
  )
}
