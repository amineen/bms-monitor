import { useState } from 'react'
import { Loader2, Network } from 'lucide-react'
import type { PlantSnapshot } from '../../lib/api'
import { PlantOverview } from './PlantOverview'
import { DeviceTile } from './DeviceTile'
import { Timeline } from './Timeline'
import { DetailPage, targetForKind, type DetailTarget } from './detail/DetailPage'

export function PlantView({
  snap,
  history,
  loading,
  onOpenBattery,
  pollSharedBus,
  onSetPollSharedBus,
}: {
  snap: PlantSnapshot | null
  history: PlantSnapshot[]
  loading: boolean
  onOpenBattery: () => void
  pollSharedBus: boolean
  onSetPollSharedBus: (b: boolean) => void
}) {
  const [target, setTarget] = useState<DetailTarget | null>(null)

  if (!snap) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4 text-center">
        <div className="grid h-16 w-16 place-items-center rounded-2xl bg-accent/10 ring-1 ring-accent/30">
          {loading ? <Loader2 className="h-7 w-7 animate-spin text-accent" /> : <Network className="h-7 w-7 text-accent" />}
        </div>
        <div>
          <div className="text-lg font-semibold text-slate-100">{loading ? 'Polling the plant…' : 'Plant view'}</div>
          <div className="mt-1 max-w-md text-sm text-slate-400">
            {loading
              ? 'Walking the ARC network devices one at a time (BMS, OzTek PCS, SMA PV, genset)…'
              : 'Press Refresh to poll every device on the ARC network (192.168.0.0/24).'}
          </div>
        </div>
      </div>
    )
  }

  // Full detail page for a device group (Solar PV / BESS / Genset).
  if (target) {
    return (
      <DetailPage
        target={target}
        snap={snap}
        history={history}
        onBack={() => setTarget(null)}
        onOpenBattery={onOpenBattery}
        pollSharedBus={pollSharedBus}
        onSetPollSharedBus={onSetPollSharedBus}
      />
    )
  }

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-4 animate-fadeUp">
      <PlantOverview snap={snap} onOpenGroup={setTarget} />

      <div>
        <div className="stat-label mb-2">Devices</div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {snap.devices.map((r) => (
            <DeviceTile key={r.id} r={r} onClick={() => setTarget(targetForKind(r.kind))} />
          ))}
        </div>
      </div>

      <Timeline events={snap.events ?? []} />
    </div>
  )
}
