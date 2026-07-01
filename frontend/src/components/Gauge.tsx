import { socColor } from '../lib/ui'

// A 270° arc gauge with the value in the centre.
export function Gauge({
  value,
  size = 210,
  unit = '%',
  caption = 'State of Charge',
}: {
  value?: number
  size?: number
  unit?: string
  caption?: string
}) {
  const stroke = 14
  const r = (size - stroke) / 2
  const cx = size / 2
  const cy = size / 2
  const circ = 2 * Math.PI * r
  const arcFrac = 0.75 // 270°
  const pct = value == null ? 0 : Math.max(0, Math.min(100, value))
  const color = socColor(value)

  const trackDash = `${arcFrac * circ} ${circ}`
  const valueDash = `${(pct / 100) * arcFrac * circ} ${circ}`
  // rotate so the gap sits at the bottom
  const rot = `rotate(135 ${cx} ${cy})`

  return (
    <div className="relative flex items-center justify-center" style={{ width: size, height: size }}>
      <svg width={size} height={size}>
        <defs>
          <linearGradient id="gaugeGrad" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.85" />
            <stop offset="100%" stopColor={color} />
          </linearGradient>
        </defs>
        <circle
          cx={cx}
          cy={cy}
          r={r}
          fill="none"
          stroke="#1A2336"
          strokeWidth={stroke}
          strokeLinecap="round"
          strokeDasharray={trackDash}
          transform={rot}
        />
        <circle
          cx={cx}
          cy={cy}
          r={r}
          fill="none"
          stroke="url(#gaugeGrad)"
          strokeWidth={stroke}
          strokeLinecap="round"
          strokeDasharray={valueDash}
          transform={rot}
          style={{ transition: 'stroke-dasharray 0.6s ease, stroke 0.4s ease', filter: `drop-shadow(0 0 8px ${color}66)` }}
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <div className="mono text-5xl font-semibold leading-none" style={{ color }}>
          {value == null ? '—' : Math.round(value)}
          <span className="ml-0.5 text-2xl text-slate-400">{unit}</span>
        </div>
        <div className="mt-2 text-xs font-medium uppercase tracking-wider text-slate-400">{caption}</div>
      </div>
    </div>
  )
}
