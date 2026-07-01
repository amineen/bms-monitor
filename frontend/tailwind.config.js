/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: {
          950: '#06090F',
          900: '#0A0E17',
          850: '#0E1320',
          800: '#131A2A',
          750: '#192234',
          700: '#222C42',
          600: '#2E3A55',
        },
        accent: { DEFAULT: '#38BDF8', soft: '#0EA5E9', glow: '#22D3EE' },
        good: '#34D399',
        warn: '#FBBF24',
        crit: '#F87171',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'Avenir', 'Helvetica', 'Arial', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      boxShadow: {
        panel: '0 1px 0 rgba(255,255,255,0.03) inset, 0 10px 30px rgba(2,6,16,0.5)',
        glow: '0 0 24px rgba(56,189,248,0.25)',
      },
      keyframes: {
        pulseSoft: { '0%,100%': { opacity: '1' }, '50%': { opacity: '0.45' } },
        fadeUp: { '0%': { opacity: '0', transform: 'translateY(6px)' }, '100%': { opacity: '1', transform: 'none' } },
        // Moving energy packets along the DC bus interconnect.
        busFlow: { '0%': { backgroundPositionX: '0px' }, '100%': { backgroundPositionX: '22px' } },
        // A shine sweeping up a charging battery module.
        cellShine: { '0%': { transform: 'translateY(120%)', opacity: '0' }, '35%': { opacity: '0.9' }, '100%': { transform: 'translateY(-120%)', opacity: '0' } },
        // Soft breathing glow for live nodes.
        haloPulse: { '0%,100%': { opacity: '0.55', transform: 'scale(1)' }, '50%': { opacity: '0.15', transform: 'scale(1.12)' } },
      },
      animation: {
        pulseSoft: 'pulseSoft 1.6s ease-in-out infinite',
        fadeUp: 'fadeUp 0.35s ease-out both',
        busFlow: 'busFlow 0.85s linear infinite',
        cellShine: 'cellShine 2.6s ease-in-out infinite',
        haloPulse: 'haloPulse 2.4s ease-in-out infinite',
      },
    },
  },
  plugins: [],
}
