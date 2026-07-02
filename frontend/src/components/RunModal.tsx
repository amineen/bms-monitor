import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { X, AlertTriangle, CheckCircle2, XCircle, Zap, Loader2, ShieldAlert } from 'lucide-react'
import { api, type Config, type SystemSnapshot, type RunGate, type RunResult } from '../lib/api'
import { fmt, clsx } from '../lib/ui'

const MAX_SPREAD_V = 5

const INTERLOCKS = [
  'Downstream DC disconnect is OPEN, or the inverters are powered and pre-charging.',
  'No one is working on the combiner, DC bus, or strings.',
  'String voltages verified — safe to parallel.',
]

export function RunModal({
  config,
  snapshot,
  onClose,
  onDone,
}: {
  config: Config
  snapshot: SystemSnapshot
  onClose: () => void
  onDone: () => void
}) {
  const [gate, setGate] = useState<RunGate | null>(null)
  const [checked, setChecked] = useState<boolean[]>([false, false, false])
  const [force, setForce] = useState(false)
  const [firing, setFiring] = useState(false)
  const [result, setResult] = useState<RunResult | null>(null)

  // Evaluate the gate from the current snapshot (pure, server-side).
  useEffect(() => {
    let alive = true
    api
      .EvaluateRunGate(snapshot, MAX_SPREAD_V)
      .then((g) => alive && setGate(g))
      .catch(() => {})
    return () => {
      alive = false
    }
  }, [snapshot])

  const allChecked = checked.every(Boolean)
  const softFail = !!gate && gate.hardOk && !gate.ok
  const canFire =
    !!gate && gate.hardOk && allChecked && (gate.ok || force) && !firing && !result

  const fire = async () => {
    setFiring(true)
    try {
      const r = await api.IssueRun(config, MAX_SPREAD_V, true, force)
      setResult(r)
      onDone()
    } catch (e: any) {
      setResult({
        written: false,
        live: false,
        message: typeof e === 'string' ? e : e?.message ?? 'Run failed.',
      } as RunResult)
    } finally {
      setFiring(false)
    }
  }

  return (
    <motion.div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <div className="absolute inset-0 bg-ink-950/75 backdrop-blur-sm" onClick={firing ? undefined : onClose} />
      <motion.div
        className="panel relative z-10 flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden"
        initial={{ scale: 0.96, y: 12, opacity: 0 }}
        animate={{ scale: 1, y: 0, opacity: 1 }}
        exit={{ scale: 0.97, opacity: 0 }}
        transition={{ type: 'spring', stiffness: 320, damping: 28 }}
      >
        {/* header */}
        <div className="flex items-center justify-between border-b border-ink-700/70 px-5 py-3.5">
          <div className="flex items-center gap-2.5">
            <span className="grid h-8 w-8 place-items-center rounded-lg bg-warn/15 ring-1 ring-warn/40">
              <ShieldAlert className="h-4 w-4 text-warn" />
            </span>
            <div>
              <div className="text-[15px] font-semibold text-slate-100">Energize combiner — commissioning</div>
              <div className="text-[11px] text-slate-500">Writes Run (0x1094) · closes string relays</div>
            </div>
          </div>
          <button
            onClick={onClose}
            disabled={firing}
            className="rounded-lg p-1.5 text-slate-400 transition hover:bg-ink-700 hover:text-slate-100 disabled:opacity-40"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-5 py-4">
          {!result ? (
            <>
              {/* hazard callout */}
              <div className="flex gap-2.5 rounded-xl border border-warn/30 bg-warn/10 px-3.5 py-3 text-[13px] text-warn">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                <span>
                  This closes the string relays and puts <b>~{fmt.v(snapshot.aggregate?.totalV)} V</b> on the combiner
                  bus. Only proceed if the downstream inverters are isolated or ready — energizing a dead,
                  capacitor-loaded input causes inrush and arcing.
                </span>
              </div>

              {/* gate checks */}
              <div className="mt-4">
                <div className="stat-label mb-2">Preconditions (read from the BMS)</div>
                <div className="flex flex-col gap-1.5">
                  {gate?.checks?.map((c, i) => (
                    <div
                      key={i}
                      className="flex items-start gap-2 rounded-lg border border-ink-700/60 bg-ink-900/40 px-3 py-2"
                    >
                      {c.pass ? (
                        <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-good" />
                      ) : (
                        <XCircle className={clsx('mt-0.5 h-4 w-4 shrink-0', c.hard ? 'text-crit' : 'text-warn')} />
                      )}
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2 text-[13px] font-medium text-slate-200">
                          {c.label}
                          {!c.pass && (
                            <span
                              className={clsx(
                                'rounded px-1.5 py-0.5 text-[9px] font-semibold uppercase',
                                c.hard ? 'bg-crit/15 text-crit' : 'bg-warn/15 text-warn',
                              )}
                            >
                              {c.hard ? 'Blocking' : 'Override'}
                            </span>
                          )}
                        </div>
                        <div className="mono text-[11px] text-slate-500">{c.detail}</div>
                      </div>
                    </div>
                  )) ?? <div className="text-xs text-slate-500">Evaluating…</div>}
                </div>
              </div>

              {gate && !gate.hardOk && (
                <div className="mt-3 rounded-lg border border-crit/30 bg-crit/10 px-3 py-2 text-[12px] text-crit">
                  Blocked by a safety-critical precondition. Resolve it on the BMS — this cannot be overridden.
                </div>
              )}

              {/* interlocks */}
              <div className="mt-4">
                <div className="stat-label mb-2">Operator interlocks — confirm all</div>
                <div className="flex flex-col gap-1.5">
                  {INTERLOCKS.map((label, i) => (
                    <label
                      key={i}
                      className="flex cursor-pointer items-start gap-2.5 rounded-lg border border-ink-700/60 bg-ink-900/40 px-3 py-2 text-[13px] text-slate-300 transition hover:border-ink-600"
                    >
                      <input
                        type="checkbox"
                        checked={checked[i]}
                        onChange={(e) => setChecked((c) => c.map((v, j) => (j === i ? e.target.checked : v)))}
                        className="mt-0.5 h-4 w-4 shrink-0 accent-warn"
                      />
                      <span>{label}</span>
                    </label>
                  ))}
                </div>
              </div>

              {softFail && (
                <label className="mt-3 flex cursor-pointer items-start gap-2.5 rounded-lg border border-warn/30 bg-warn/5 px-3 py-2 text-[13px] text-warn">
                  <input
                    type="checkbox"
                    checked={force}
                    onChange={(e) => setForce(e.target.checked)}
                    className="mt-0.5 h-4 w-4 shrink-0 accent-warn"
                  />
                  <span>Override the non-blocking checks above (I have verified them manually).</span>
                </label>
              )}
            </>
          ) : (
            /* result */
            <div className="flex flex-col items-center gap-3 py-4 text-center">
              <div
                className={clsx(
                  'grid h-14 w-14 place-items-center rounded-2xl ring-1',
                  result.live ? 'bg-good/10 ring-good/30' : 'bg-warn/10 ring-warn/30',
                )}
              >
                {result.live ? (
                  <Zap className="h-7 w-7 text-good" />
                ) : (
                  <AlertTriangle className="h-7 w-7 text-warn" />
                )}
              </div>
              <div className="text-[15px] font-semibold text-slate-100">
                {result.live ? 'Combiner energized' : result.written ? 'Run issued' : 'Run not sent'}
              </div>
              <div className="max-w-sm text-[13px] text-slate-400">{result.message}</div>
              {result.written && (
                <div className="mono mt-1 text-[12px] text-slate-500">
                  {result.stateBefore} → {result.stateAfter || '…'}
                  {result.live && result.busV ? ` · ${fmt.v(result.busV)} V on bus` : ''}
                  {result.woke ? ' · woke from sleep' : ''}
                </div>
              )}
            </div>
          )}
        </div>

        {/* footer */}
        <div className="flex items-center justify-end gap-2 border-t border-ink-700/70 px-5 py-3">
          {!result ? (
            <>
              <button
                onClick={onClose}
                disabled={firing}
                className="rounded-lg border border-ink-700 bg-ink-850 px-4 py-2 text-sm text-slate-300 transition hover:text-slate-100 disabled:opacity-40"
              >
                Cancel
              </button>
              <button
                onClick={fire}
                disabled={!canFire}
                className={clsx(
                  'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition',
                  canFire ? 'bg-crit text-white hover:bg-crit/90' : 'cursor-not-allowed bg-ink-750 text-slate-500',
                )}
              >
                {firing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
                {firing ? 'Energizing…' : 'Energize combiner (Run)'}
              </button>
            </>
          ) : (
            <button
              onClick={onClose}
              className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-ink-950 transition hover:bg-accent-glow"
            >
              Done
            </button>
          )}
        </div>
      </motion.div>
    </motion.div>
  )
}
