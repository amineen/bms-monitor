import { Fragment } from 'react'
import { Crown, Thermometer, Waypoints, BatteryCharging, Zap, Minus } from 'lucide-react'
import type { SystemSnapshot, StringInfo } from '../lib/api'
import { statusMeta, fmt, clsx } from '../lib/ui'

function tempColor(t?: number): string {
  if (t == null) return 'text-slate-500'
  if (t >= 55) return 'text-crit'
  if (t >= 45) return 'text-warn'
  return 'text-slate-400'
}

// Tower SOC fill — blue to match the control-room theme, red only when the
// string is critically low so a genuine low-charge warning still stands out.
function towerFill(soc?: number): string {
  if (soc != null && soc < 20) return '#F87171'
  return '#38BDF8'
}

// A single string rendered as a Pylontech Force-H3 stacked tower: a blue-grey
// BMS control head on top, a stack of white battery modules with seams, and a
// dark base. SOC is shown both as a rising energy fill and a percentage.
function Node({ s, charging }: { s: StringInfo; charging: boolean }) {
  const m = statusMeta(s.status)
  const online = s.status === 'enumerated'
  const noResp = s.status === 'no-response'
  const col = towerFill(s.soc)
  const soc = Math.max(0, Math.min(100, s.soc ?? 0))

  const modules = Math.min(Math.max(s.modules || 7, 3), 8)
  const bodyH = 116
  const seg = bodyH / modules

  const headBg = online
    ? 'linear-gradient(180deg,#6B7A90,#475569)'
    : noResp
      ? 'linear-gradient(180deg,#57606f,#3b4251)'
      : 'linear-gradient(180deg,#293244,#1c2434)'
  const baseBg = online
    ? 'linear-gradient(180deg,#475569,#2f3b4f)'
    : 'linear-gradient(180deg,#3b4251,#262c39)'
  const bodyBg = online
    ? 'linear-gradient(100deg,#F3F6FA 0%,#E2E8F1 52%,#CAD3E0 100%)'
    : noResp
      ? 'linear-gradient(100deg,#7C8698 0%,#5B6576 100%)'
      : 'linear-gradient(100deg,#28313f 0%,#161d29 100%)'
  const led = online ? '#34D399' : noResp ? '#F87171' : '#64748B'
  const seams = `repeating-linear-gradient(180deg, rgba(15,23,42,0.22) 0 1.4px, transparent 1.4px ${seg}px)`

  return (
    <div className="flex w-[92px] shrink-0 flex-col items-center">
      {/* tower */}
      <div className="relative" style={{ width: 64 }}>
        {online && <div className="absolute -inset-2 -z-10 rounded-2xl bg-good/15 blur-md animate-haloPulse" />}

        {/* BMS control head */}
        <div
          className="relative z-10 mx-auto flex h-[19px] w-[62px] items-center justify-between rounded-t-md px-1.5 shadow-[0_1px_0_rgba(255,255,255,0.12)_inset]"
          style={{ background: headBg }}
        >
          <span className="flex items-center gap-[3px]">
            <span className="h-1.5 w-1.5 rounded-full" style={{ background: led, boxShadow: `0 0 5px ${led}` }} />
            <span className="h-[3px] w-2 rounded-full bg-white/25" />
          </span>
          <span className="flex items-center gap-1">
            <span className="text-[9px] font-bold tracking-wide text-white/90">S{s.index}</span>
            {s.isMaster && <Crown className="h-2.5 w-2.5 text-accent" />}
          </span>
        </div>

        {/* module stack */}
        <div
          className={clsx(
            'relative mx-auto w-[62px] overflow-hidden ring-1 ring-black/25',
            online && 'shadow-[0_0_18px_rgba(52,211,153,0.14)]',
          )}
          style={{ height: bodyH, background: bodyBg }}
        >
          {/* SOC energy fill */}
          {online && (
            <>
              <div
                className="absolute inset-x-0 bottom-0 transition-[height] duration-700 ease-out"
                style={{ height: `${Math.max(5, soc)}%`, background: col, opacity: 0.5 }}
              />
              <div
                className="absolute inset-x-0 transition-all duration-700 ease-out"
                style={{ bottom: `${Math.max(5, soc)}%`, height: 2, background: col, boxShadow: `0 0 8px ${col}` }}
              />
              {charging && (
                <div
                  className="absolute inset-x-0 bottom-0 h-10 animate-cellShine"
                  style={{ background: 'linear-gradient(180deg, transparent, rgba(255,255,255,0.6), transparent)' }}
                />
              )}
            </>
          )}
          {/* module seams */}
          <div className="pointer-events-none absolute inset-0" style={{ backgroundImage: seams }} />
          {/* left specular highlight */}
          <div
            className="pointer-events-none absolute inset-0"
            style={{ background: 'linear-gradient(100deg, rgba(255,255,255,0.35), transparent 26%)' }}
          />
          {/* SOC percentage */}
          <div className="absolute inset-0 grid place-items-center">
            <span
              className="mono text-[19px] font-extrabold leading-none"
              style={
                online
                  ? { color: '#0F172A', textShadow: '0 1px 1px rgba(255,255,255,0.5)' }
                  : { color: noResp ? '#FCA5A5' : '#64748B' }
              }
            >
              {online ? `${soc}%` : noResp ? '×' : '—'}
            </span>
          </div>
        </div>

        {/* base plinth */}
        <div className="mx-auto h-2 w-[66px] rounded-b-md" style={{ background: baseBg }} />
      </div>

      {/* readouts */}
      <div className="mt-2.5 flex flex-col items-center gap-0.5 text-center">
        <span className={clsx('inline-flex items-center gap-1 text-[11px] font-medium', m.text)}>
          <span className={clsx('h-1.5 w-1.5 rounded-full', m.dot, online && 'animate-pulseSoft')} />
          {m.label}
        </span>
        <span className="mono text-[12px] font-semibold text-slate-200">{online ? `${fmt.v(s.totalV)} V` : '—'}</span>
        {online && (
          <span className={clsx('mono inline-flex items-center gap-0.5 text-[10px]', tempColor(s.temp))}>
            <Thermometer className="h-2.5 w-2.5" />
            {fmt.temp(s.temp)}
          </span>
        )}
      </div>
    </div>
  )
}

// The DC-bus interconnect between two adjacent modules.
function Bus({ active, live, reverse }: { active: boolean; live: boolean; reverse: boolean }) {
  return (
    <div className="mt-[65px] flex h-6 min-w-[16px] flex-1 items-center">
      <div className="relative h-[3px] w-full overflow-hidden rounded-full bg-ink-700/80">
        {active && !live && <div className="absolute inset-0 rounded-full bg-good/45" />}
        {active && live && (
          <div
            className="absolute inset-0 animate-busFlow"
            style={{
              backgroundImage: `repeating-linear-gradient(90deg, ${'#34D399'} 0 5px, transparent 5px 22px)`,
              backgroundSize: '22px 100%',
              animationDirection: reverse ? 'reverse' : 'normal',
            }}
          />
        )}
      </div>
    </div>
  )
}

function FlowBadge({ current }: { current: number }) {
  const charging = current > 0.1
  const discharging = current < -0.1
  const [icon, label, cls] = charging
    ? [<BatteryCharging className="h-3.5 w-3.5" key="c" />, 'Charging', 'text-good bg-good/10 ring-good/30']
    : discharging
      ? [<Zap className="h-3.5 w-3.5" key="d" />, 'Discharging', 'text-warn bg-warn/10 ring-warn/30']
      : [<Minus className="h-3.5 w-3.5" key="i" />, 'Idle', 'text-slate-400 bg-slate-500/10 ring-slate-500/30']
  return (
    <div className={clsx('inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium ring-1', cls)}>
      {icon}
      {label}
      <span className="mono text-slate-500">· {fmt.amp(current)}</span>
    </div>
  )
}

export function ChainStrip({ snap }: { snap: SystemSnapshot }) {
  const states = snap.strings
  const ok = snap.chain.online === 6
  const current = snap.aggregate.current ?? 0
  const live = Math.abs(current) >= 0.1
  const reverse = current < 0 // discharging → energy flows outward

  return (
    <div className="panel relative overflow-hidden p-5">
      {/* accent top edge */}
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-accent/40 to-transparent" />

      <div className="mb-5 flex items-start justify-between gap-3">
        <div className="flex items-start gap-2.5">
          <div className="grid h-8 w-8 place-items-center rounded-lg bg-accent/10 ring-1 ring-accent/25">
            <Waypoints className="h-4 w-4 text-accent" />
          </div>
          <div>
            <div className="stat-label">DC String Bus</div>
            <div className={clsx('text-sm font-medium', ok ? 'text-good' : 'text-slate-200')}>{snap.chain.verdict}</div>
          </div>
        </div>
        <FlowBadge current={current} />
      </div>

      <div className="flex items-start">
        {states.map((s, i) => (
          <Fragment key={s.index}>
            <Node s={s} charging={live && !reverse && s.status === 'enumerated'} />
            {i < states.length - 1 && (
              <Bus
                active={s.status === 'enumerated' && states[i + 1].status === 'enumerated'}
                live={live && s.status === 'enumerated' && states[i + 1].status === 'enumerated'}
                reverse={reverse}
              />
            )}
          </Fragment>
        ))}
      </div>

      {/* legend */}
      <div className="mt-4 flex items-center justify-end gap-4 text-[10px] text-slate-500">
        <span className="inline-flex items-center gap-1.5">
          <span className="h-1.5 w-1.5 rounded-full bg-good" /> Online
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span className="h-1.5 w-1.5 rounded-full bg-crit" /> No response
        </span>
        <span className="inline-flex items-center gap-1.5">
          <Crown className="h-2.5 w-2.5 text-accent" /> Master · comms head
        </span>
      </div>
    </div>
  )
}
