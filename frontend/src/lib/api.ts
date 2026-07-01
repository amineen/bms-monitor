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
} from '../../wailsjs/go/main/App'
import { bms, config } from '../../wailsjs/go/models'

export type Config = bms.Config
export type SystemSnapshot = bms.SystemSnapshot
export type StringInfo = bms.StringInfo
export type Identity = bms.Identity
export type Aggregate = bms.Aggregate
export type ChainStatus = bms.ChainStatus
export type HealthStatus = bms.HealthStatus
export type Remote = config.Remote

export type SourceMode = 'onsite' | 'remote'

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
}
