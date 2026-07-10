import { useCallback, useEffect, useRef, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { Loader2, WifiOff, AlertTriangle, CheckCircle2, Cloud, LogIn } from 'lucide-react'
import {
  api,
  type Config,
  type SystemSnapshot,
  type StringInfo,
  type SourceMode,
  type ViewMode,
  type Remote,
  type PlantSnapshot,
  type LoggingStatus,
  type LoggingConfig,
  type PollingConfig,
} from './lib/api'
import { TopBar } from './components/TopBar'
import { SystemHero } from './components/SystemHero'
import { ChainStrip } from './components/ChainStrip'
import { StringCard } from './components/StringCard'
import { StringDetail } from './components/StringDetail'
import { RunModal } from './components/RunModal'
import { ExportLogsModal } from './components/ExportLogsModal'
import { PlantView } from './components/plant/PlantView'

const DEFAULT_CONFIG = { ip: '192.168.0.31', port: 502, unit: 1, timeout: 3 } as Config

function OfflineState({
  ip,
  error,
  loading,
  mode,
  hasToken,
  connecting,
  onUseSolarman,
}: {
  ip: string
  error: string
  loading: boolean
  mode: SourceMode
  hasToken: boolean
  connecting: boolean
  onUseSolarman: () => void
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-4 text-center">
      <div className="grid h-16 w-16 place-items-center rounded-2xl bg-crit/10 ring-1 ring-crit/30">
        {loading ? <Loader2 className="h-7 w-7 animate-spin text-accent" /> : <WifiOff className="h-7 w-7 text-crit" />}
      </div>
      <div>
        <div className="text-lg font-semibold text-slate-100">
          {loading ? 'Connecting…' : 'Can’t reach the BMS gateway'}
        </div>
        <div className="mt-1 max-w-md text-sm text-slate-400">
          {loading ? (
            <>
              Reading <span className="mono">{ip}</span>…
            </>
          ) : (
            <>
              No response from <span className="mono text-slate-300">{ip}</span>. Check this computer is on the
              converter’s network and the gateway is powered.
            </>
          )}
        </div>
        {!loading && error && <div className="mono mt-2 text-xs text-slate-600">{error}</div>}
      </div>

      {/* Not on-site? Read the strings remotely through the Solarman portal. */}
      {!loading && mode === 'onsite' && (
        <div className="mt-2 flex flex-col items-center gap-2">
          <div className="text-xs text-slate-500">Not on-site? Read the batteries remotely instead.</div>
          <button
            onClick={onUseSolarman}
            disabled={connecting}
            className="inline-flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-ink-950 transition hover:bg-accent-glow disabled:opacity-60"
          >
            {connecting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" /> Waiting for login…
              </>
            ) : hasToken ? (
              <>
                <Cloud className="h-4 w-4" /> Read from Solarman
              </>
            ) : (
              <>
                <LogIn className="h-4 w-4" /> Sign in to Solarman
              </>
            )}
          </button>
        </div>
      )}
    </div>
  )
}

function Toast({ msg, ok }: { msg: string; ok: boolean }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 20 }}
      className="fixed bottom-5 left-1/2 z-50 flex -translate-x-1/2 items-center gap-2 rounded-xl border border-ink-700 bg-ink-850/95 px-4 py-2.5 text-sm text-slate-200 shadow-panel backdrop-blur"
    >
      {ok ? <CheckCircle2 className="h-4 w-4 text-good" /> : <AlertTriangle className="h-4 w-4 text-crit" />}
      <span className="max-w-md truncate">{msg}</span>
    </motion.div>
  )
}

export default function App() {
  const [config, setConfig] = useState<Config>(DEFAULT_CONFIG)
  const [snapshot, setSnapshot] = useState<SystemSnapshot | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [auto, setAuto] = useState(false)
  const [intervalSec, setIntervalSec] = useState(30)
  const [selected, setSelected] = useState<StringInfo | null>(null)
  const [exporting, setExporting] = useState(false)
  const [toast, setToast] = useState<{ msg: string; ok: boolean } | null>(null)

  const [mode, setMode] = useState<SourceMode>('onsite')
  const [remote, setRemote] = useState<Remote>({ token: '', stationId: 66280946 } as Remote)
  const [connecting, setConnecting] = useState(false)

  // Plant view (on-site only): unified poll across all ARC-network devices.
  const [view, setView] = useState<ViewMode>(() => (localStorage.getItem('view') as ViewMode) || 'plant')
  const [plantSnap, setPlantSnap] = useState<PlantSnapshot | null>(null)
  const [plantHistory, setPlantHistory] = useState<PlantSnapshot[]>([])
  const [plantLoading, setPlantLoading] = useState(false)
  useEffect(() => {
    localStorage.setItem('view', view)
  }, [view])

  // Datastore logging status (SQLite telemetry log in the Go backend).
  const [logging, setLogging] = useState<LoggingStatus | null>(null)
  const [exportLogsOpen, setExportLogsOpen] = useState(false)

  // Polling policy: whether we master ARC's shared OzTek bus (default off).
  const [polling, setPolling] = useState<PollingConfig>({ pollSharedBus: false } as PollingConfig)
  useEffect(() => {
    api.GetPolling().then(setPolling).catch(() => {})
  }, [])
  useEffect(() => {
    api.GetLogging().then(setLogging).catch(() => {})
  }, [])
  const onSetLogging = async (cfg: LoggingConfig) => {
    try {
      setLogging(await api.SetLogging(cfg))
    } catch {
      /* backend without datastore keeps running log-less */
    }
  }

  const [commissioning, setCommissioning] = useState<boolean>(() => localStorage.getItem('commissioning') === '1')
  const [runOpen, setRunOpen] = useState(false)
  useEffect(() => {
    localStorage.setItem('commissioning', commissioning ? '1' : '0')
  }, [commissioning])

  const inFlight = useRef(false)
  const plantInFlight = useRef(false)
  const configRef = useRef(config)
  configRef.current = config
  const modeRef = useRef(mode)
  modeRef.current = mode
  const remoteRef = useRef(remote)
  remoteRef.current = remote
  const viewRef = useRef(view)
  viewRef.current = view

  const showToast = (msg: string, ok = true) => {
    setToast({ msg, ok })
    setTimeout(() => setToast(null), 4500)
  }

  const read = useCallback(async () => {
    if (inFlight.current) return
    inFlight.current = true
    setLoading(true)
    try {
      if (modeRef.current === 'remote' && !remoteRef.current.token) {
        throw 'Enter your Solarman session token to connect remotely.'
      }
      const snap =
        modeRef.current === 'remote'
          ? await api.ReadRemote(remoteRef.current.token, remoteRef.current.stationId)
          : await api.ReadSystem(configRef.current)
      setSnapshot(snap)
      setError(null)
      setLastUpdated(new Date())
    } catch (e: any) {
      setError(typeof e === 'string' ? e : e?.message ?? 'Could not reach the BMS gateway.')
    } finally {
      setLoading(false)
      inFlight.current = false
    }
  }, [])

  // One sequential round-robin over every enabled plant device. The backend
  // serializes the shared .41 OzTek bus; unreachable devices fail fast.
  const readPlant = useCallback(async () => {
    if (plantInFlight.current) return
    plantInFlight.current = true
    setPlantLoading(true)
    try {
      const snap = await api.ReadPlant()
      setPlantSnap(snap)
      // Keep a session ring of snapshots so detail pages can draw trends.
      setPlantHistory((h) => {
        const next = [...h, snap]
        return next.length > 180 ? next.slice(next.length - 180) : next
      })
      setLastUpdated(new Date())
    } catch {
      /* per-device errors are inside the snapshot; a throw here is unexpected */
    } finally {
      setPlantLoading(false)
      plantInFlight.current = false
    }
  }, [])

  // Refresh whatever the active view shows.
  const refresh = useCallback(() => {
    if (viewRef.current === 'plant' && modeRef.current === 'onsite') readPlant()
    else read()
  }, [read, readPlant])

  // Toggle whether we master ARC's shared OzTek bus, then re-read the plant so
  // the change (poll off ↔ on) reflects immediately.
  const onSetPollShared = useCallback(
    async (b: boolean) => {
      try {
        const p = await api.SetPolling({ pollSharedBus: b } as PollingConfig)
        setPolling(p)
        if (viewRef.current === 'plant') readPlant()
      } catch {
        /* ignore */
      }
    },
    [readPlant],
  )

  // load saved config, then read once
  // On startup: load saved config, then auto-pick the source — if the on-site
  // gateway is reachable use On-site, otherwise fall back to Remote when a saved
  // Solarman token exists. The manual toggle still overrides.
  useEffect(() => {
    ;(async () => {
      const c = await api.GetConfig().catch(() => DEFAULT_CONFIG)
      setConfig(c)
      configRef.current = c
      const r = await api.GetRemoteConfig().catch(() => null)
      if (r) {
        setRemote(r)
        remoteRef.current = r
      }
      let m: SourceMode = 'onsite'
      try {
        const reachable = await api.GatewayReachable(c)
        if (!reachable && r?.token) m = 'remote'
      } catch {
        /* ignore */
      }
      setMode(m)
      modeRef.current = m
      // Remote (Solarman) covers only the battery — force the battery view.
      if (m === 'remote' && viewRef.current === 'plant') {
        setView('battery')
        viewRef.current = 'battery'
      }
      refresh()
    })()
  }, [read, refresh])

  // auto-refresh (whichever view is active)
  useEffect(() => {
    if (!auto) return
    const id = setInterval(() => refresh(), intervalSec * 1000)
    return () => clearInterval(id)
  }, [auto, intervalSec, refresh])

  // keep the open detail in sync with new data
  useEffect(() => {
    if (selected && snapshot) {
      const updated = snapshot.strings.find((s) => s.index === selected.index)
      if (updated) setSelected(updated)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [snapshot])

  // Switching source should immediately re-read from the new source (otherwise
  // the old snapshot lingers — e.g. remote data still showing under On-site).
  const onSetMode = (m: SourceMode) => {
    setMode(m)
    modeRef.current = m
    if (m === 'remote' && viewRef.current === 'plant') {
      // Solarman reaches only the battery — the plant needs the ARC LAN.
      setView('battery')
      viewRef.current = 'battery'
    }
    refresh()
  }

  // Switching view triggers a read for that view if it has no data yet.
  const onSetView = (v: ViewMode) => {
    setView(v)
    viewRef.current = v
    if (v === 'plant' && !plantSnap) readPlant()
    if (v === 'battery' && !snapshot) read()
  }

  const onExportExcel = async () => {
    setExporting(true)
    try {
      const path = await api.ExportExcel(configRef.current)
      if (path) showToast(`Saved ${path}`)
    } catch (e: any) {
      showToast('Export failed: ' + (e?.message ?? e), false)
    } finally {
      setExporting(false)
    }
  }

  const onExportPDF = async () => {
    if (!snapshot) return
    setExporting(true)
    try {
      const path = await api.ExportPDF(snapshot)
      if (path) showToast(`Saved ${path}`)
    } catch (e: any) {
      showToast('Export failed: ' + (e?.message ?? e), false)
    } finally {
      setExporting(false)
    }
  }

  const onConnect = async () => {
    setConnecting(true)
    showToast('Opening the Solarman login — sign in in the browser window…')
    try {
      const tok = await api.ConnectSolarman()
      if (tok) {
        const r = { token: tok, stationId: remoteRef.current.stationId } as Remote
        setRemote(r)
        remoteRef.current = r
        setMode('remote')
        modeRef.current = 'remote'
        showToast('Connected to Solarman')
        read()
      }
    } catch (e: any) {
      showToast('Solarman login failed: ' + (e?.message ?? e), false)
    } finally {
      setConnecting(false)
    }
  }

  // From the offline screen: use Solarman without leaving On-site manually.
  // If we already hold a token, just switch and read; otherwise start a login.
  const onUseSolarman = () => {
    if (remoteRef.current.token) {
      setMode('remote')
      modeRef.current = 'remote'
      read()
    } else {
      onConnect()
    }
  }

  // Forget the saved token so the next Connect is a clean login.
  const onClearSession = async () => {
    try {
      await api.ClearSolarmanSession()
    } catch {
      /* ignore */
    }
    const r = { token: '', stationId: remoteRef.current.stationId } as Remote
    setRemote(r)
    remoteRef.current = r
    if (modeRef.current === 'remote') {
      setSnapshot(null)
      setError(null)
    }
    showToast('Signed out of Solarman — Connect to log in again')
  }

  const plantActive = view === 'plant' && mode === 'onsite'
  const connected = plantActive
    ? !!plantSnap && plantSnap.health.level !== 'offline'
    : !!snapshot && !error

  return (
    <div className="flex h-screen flex-col">
      <TopBar
        config={config}
        onChange={setConfig}
        view={view}
        setView={onSetView}
        mode={mode}
        setMode={onSetMode}
        remote={remote}
        setRemote={setRemote}
        onConnect={onConnect}
        onClearSession={onClearSession}
        connecting={connecting}
        commissioning={commissioning}
        setCommissioning={setCommissioning}
        logging={logging}
        onSetLogging={onSetLogging}
        onExportLogs={() => setExportLogsOpen(true)}
        onMaximise={() => api.ToggleMaximise()}
        onRefresh={refresh}
        loading={plantActive ? plantLoading : loading}
        lastUpdated={lastUpdated}
        auto={auto}
        setAuto={setAuto}
        intervalSec={intervalSec}
        setIntervalSec={setIntervalSec}
        snapshot={snapshot}
        connected={connected}
        onExportExcel={onExportExcel}
        onExportPDF={onExportPDF}
        exporting={exporting}
      />

      <main className="flex-1 overflow-y-auto px-6 py-5">
        {plantActive ? (
          <PlantView
            snap={plantSnap}
            history={plantHistory}
            loading={plantLoading}
            onOpenBattery={() => onSetView('battery')}
            pollSharedBus={polling.pollSharedBus}
            onSetPollSharedBus={onSetPollShared}
          />
        ) : !snapshot ? (
          <OfflineState
            ip={config.ip}
            error={error ?? ''}
            loading={loading}
            mode={mode}
            hasToken={!!remote.token}
            connecting={connecting}
            onUseSolarman={onUseSolarman}
          />
        ) : (
          <div className="mx-auto flex max-w-7xl flex-col gap-4 animate-fadeUp">
            {error && (
              <div className="flex items-center gap-2 rounded-xl border border-warn/30 bg-warn/10 px-4 py-2 text-sm text-warn">
                <AlertTriangle className="h-4 w-4" />
                Showing last reading — latest refresh failed: <span className="mono text-xs">{error}</span>
              </div>
            )}
            <SystemHero snap={snapshot} />
            <ChainStrip
              snap={snapshot}
              commissioning={commissioning && mode === 'onsite'}
              onEnergize={() => setRunOpen(true)}
            />
            <div>
              <div className="stat-label mb-2">Strings</div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {snapshot.strings.map((s) => (
                  <StringCard key={s.index} s={s} onClick={() => s.status === 'enumerated' && setSelected(s)} />
                ))}
              </div>
            </div>
          </div>
        )}
      </main>

      <AnimatePresence>{selected && <StringDetail s={selected} onClose={() => setSelected(null)} />}</AnimatePresence>
      <AnimatePresence>
        {runOpen && snapshot && mode === 'onsite' && (
          <RunModal
            config={config}
            snapshot={snapshot}
            onClose={() => setRunOpen(false)}
            onDone={() => read()}
          />
        )}
      </AnimatePresence>
      <AnimatePresence>
        {exportLogsOpen && (
          <ExportLogsModal onClose={() => setExportLogsOpen(false)} onDone={(msg, ok) => showToast(msg, ok)} />
        )}
      </AnimatePresence>
      <AnimatePresence>{toast && <Toast msg={toast.msg} ok={toast.ok} />}</AnimatePresence>
    </div>
  )
}
