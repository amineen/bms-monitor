import {
  GetConfig,
  SaveConfig,
  TestConnection,
  ReadSystem,
  ExportExcel,
  ExportPDF,
  ReadRemote,
  TestRemote,
  GetRemoteConfig,
  OpenSolarmanLogin,
  ConnectSolarman,
  ClearSolarmanSession,
  GatewayReachable,
  ToggleMaximise,
  EvaluateRunGate,
  IssueRun,
  ReadPlant,
  GetDevices,
  SaveDevices,
  ResetDevices,
  GetLogging,
  SetLogging,
  ExportLogs,
  GetPolling,
  SetPolling,
} from '../../wailsjs/go/main/App'
import { bms, config, plant, oztek, sma, dse, datastore } from '../../wailsjs/go/models'

export type Config = bms.Config
export type SystemSnapshot = bms.SystemSnapshot
export type StringInfo = bms.StringInfo
export type Identity = bms.Identity
export type Aggregate = bms.Aggregate
export type ChainStatus = bms.ChainStatus
export type HealthStatus = bms.HealthStatus
export type CombinerStatus = bms.CombinerStatus
export type RunGate = bms.RunGate
export type GateCheck = bms.GateCheck
export type RunResult = bms.RunResult
export type Remote = config.Remote

// Plant-wide monitoring types.
export type PlantDevice = plant.Device
export type PlantSnapshot = plant.Snapshot
export type PlantReading = plant.Reading
export type PlantEvent = plant.Event
export type PowerBalance = plant.PowerBalance
export type PlantHealth = plant.PlantHealth
export type OztekSnapshot = oztek.Snapshot
export type OztekMetric = oztek.Metric
export type SmaSnapshot = sma.Snapshot
export type SmaMetric = sma.Metric
export type DseSnapshot = dse.Snapshot
export type DseMetric = dse.Metric
export type BmsAggSummary = bms.AggSummary

// Datastore (SQLite telemetry log).
export type LoggingConfig = config.Logging
export type LoggingStatus = datastore.Status
export type LoggingStats = datastore.Stats

// Polling policy: whether the app masters ARC's shared OzTek bus (default off).
export type PollingConfig = config.Polling

export type SourceMode = 'onsite' | 'remote'
export type ViewMode = 'plant' | 'battery'

export const api = {
  GetConfig,
  SaveConfig,
  TestConnection,
  ReadSystem,
  ExportExcel,
  ExportPDF,
  ReadRemote,
  TestRemote,
  GetRemoteConfig,
  OpenSolarmanLogin,
  ConnectSolarman,
  ClearSolarmanSession,
  GatewayReachable,
  ToggleMaximise,
  EvaluateRunGate,
  IssueRun,
  ReadPlant,
  GetDevices,
  SaveDevices,
  ResetDevices,
  GetLogging,
  SetLogging,
  ExportLogs,
  GetPolling,
  SetPolling,
}
