export namespace bms {
	
	export class CombinerStatus {
	    ok: boolean;
	    live: boolean;
	    state: string;
	    sysOpRaw: number;
	    switching: number;
	    relays: string[];
	    relayClosed: boolean;
	    busV: number;
	
	    static createFrom(source: any = {}) {
	        return new CombinerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.live = source["live"];
	        this.state = source["state"];
	        this.sysOpRaw = source["sysOpRaw"];
	        this.switching = source["switching"];
	        this.relays = source["relays"];
	        this.relayClosed = source["relayClosed"];
	        this.busV = source["busV"];
	    }
	}
	export class AggSummary {
	    ok: boolean;
	    timestamp: string;
	    piles: number;
	    totalV: number;
	    current: number;
	    powerKW: number;
	    soc: number;
	    soh: number;
	    temp: number;
	    cellMaxV: number;
	    cellMinV: number;
	    combiner: CombinerStatus;
	
	    static createFrom(source: any = {}) {
	        return new AggSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.timestamp = source["timestamp"];
	        this.piles = source["piles"];
	        this.totalV = source["totalV"];
	        this.current = source["current"];
	        this.powerKW = source["powerKW"];
	        this.soc = source["soc"];
	        this.soh = source["soh"];
	        this.temp = source["temp"];
	        this.cellMaxV = source["cellMaxV"];
	        this.cellMinV = source["cellMinV"];
	        this.combiner = this.convertValues(source["combiner"], CombinerStatus);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Aggregate {
	    ok: boolean;
	    piles: number;
	    totalV: number;
	    current: number;
	    powerKW: number;
	    soc: number;
	    soh: number;
	    temp: number;
	    cellMaxV: number;
	    cellMinV: number;
	
	    static createFrom(source: any = {}) {
	        return new Aggregate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.piles = source["piles"];
	        this.totalV = source["totalV"];
	        this.current = source["current"];
	        this.powerKW = source["powerKW"];
	        this.soc = source["soc"];
	        this.soh = source["soh"];
	        this.temp = source["temp"];
	        this.cellMaxV = source["cellMaxV"];
	        this.cellMinV = source["cellMinV"];
	    }
	}
	export class ChainStatus {
	    online: number;
	    total: number;
	    stopsAfter: number;
	    verdict: string;
	    states: string[];
	
	    static createFrom(source: any = {}) {
	        return new ChainStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.online = source["online"];
	        this.total = source["total"];
	        this.stopsAfter = source["stopsAfter"];
	        this.verdict = source["verdict"];
	        this.states = source["states"];
	    }
	}
	
	export class Config {
	    ip: string;
	    port: number;
	    unit: number;
	    timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.unit = source["unit"];
	        this.timeout = source["timeout"];
	    }
	}
	export class GateCheck {
	    label: string;
	    pass: boolean;
	    detail: string;
	    hard: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GateCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.pass = source["pass"];
	        this.detail = source["detail"];
	        this.hard = source["hard"];
	    }
	}
	export class HealthStatus {
	    level: string;
	    headline: string;
	    reasons: string[];
	
	    static createFrom(source: any = {}) {
	        return new HealthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.headline = source["headline"];
	        this.reasons = source["reasons"];
	    }
	}
	export class Identity {
	    ok: boolean;
	    name: string;
	    firmware: string;
	    build: number;
	
	    static createFrom(source: any = {}) {
	        return new Identity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.name = source["name"];
	        this.firmware = source["firmware"];
	        this.build = source["build"];
	    }
	}
	export class RunGate {
	    ok: boolean;
	    hardOk: boolean;
	    checks: GateCheck[];
	    reasons: string[];
	
	    static createFrom(source: any = {}) {
	        return new RunGate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.hardOk = source["hardOk"];
	        this.checks = this.convertValues(source["checks"], GateCheck);
	        this.reasons = source["reasons"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunResult {
	    written: boolean;
	    woke: boolean;
	    stateBefore: string;
	    stateAfter: string;
	    live: boolean;
	    busV: number;
	    gate: RunGate;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new RunResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.written = source["written"];
	        this.woke = source["woke"];
	        this.stateBefore = source["stateBefore"];
	        this.stateAfter = source["stateAfter"];
	        this.live = source["live"];
	        this.busV = source["busV"];
	        this.gate = this.convertValues(source["gate"], RunGate);
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StringInfo {
	    index: number;
	    base: number;
	    baseHex: string;
	    status: string;
	    serial: string;
	    isMaster: boolean;
	    hasDetail: boolean;
	    totalV: number;
	    current: number;
	    powerKW: number;
	    soc: number;
	    soh: number;
	    soe: number;
	    temp: number;
	    cycles: number;
	    cellMaxV: number;
	    cellMinV: number;
	    cellSpreadMV: number;
	    cellMaxT: number;
	    cellMinT: number;
	    modMaxV: number;
	    modMinV: number;
	    modules: number;
	    cells: number;
	    nominalAh: number;
	    remainWh: number;
	    basicStatus: string;
	    protectionText: string;
	    alarms: string[];
	    moduleV: number[];
	    moduleT: number[];
	    cellV: number[];
	
	    static createFrom(source: any = {}) {
	        return new StringInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.base = source["base"];
	        this.baseHex = source["baseHex"];
	        this.status = source["status"];
	        this.serial = source["serial"];
	        this.isMaster = source["isMaster"];
	        this.hasDetail = source["hasDetail"];
	        this.totalV = source["totalV"];
	        this.current = source["current"];
	        this.powerKW = source["powerKW"];
	        this.soc = source["soc"];
	        this.soh = source["soh"];
	        this.soe = source["soe"];
	        this.temp = source["temp"];
	        this.cycles = source["cycles"];
	        this.cellMaxV = source["cellMaxV"];
	        this.cellMinV = source["cellMinV"];
	        this.cellSpreadMV = source["cellSpreadMV"];
	        this.cellMaxT = source["cellMaxT"];
	        this.cellMinT = source["cellMinT"];
	        this.modMaxV = source["modMaxV"];
	        this.modMinV = source["modMinV"];
	        this.modules = source["modules"];
	        this.cells = source["cells"];
	        this.nominalAh = source["nominalAh"];
	        this.remainWh = source["remainWh"];
	        this.basicStatus = source["basicStatus"];
	        this.protectionText = source["protectionText"];
	        this.alarms = source["alarms"];
	        this.moduleV = source["moduleV"];
	        this.moduleT = source["moduleT"];
	        this.cellV = source["cellV"];
	    }
	}
	export class SystemSnapshot {
	    timestamp: string;
	    ip: string;
	    port: number;
	    unit: number;
	    identity: Identity;
	    aggregate: Aggregate;
	    combiner: CombinerStatus;
	    heartbeatA: number;
	    heartbeatB: number;
	    linkLive: boolean;
	    strings: StringInfo[];
	    chain: ChainStatus;
	    health: HealthStatus;
	    readMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.unit = source["unit"];
	        this.identity = this.convertValues(source["identity"], Identity);
	        this.aggregate = this.convertValues(source["aggregate"], Aggregate);
	        this.combiner = this.convertValues(source["combiner"], CombinerStatus);
	        this.heartbeatA = source["heartbeatA"];
	        this.heartbeatB = source["heartbeatB"];
	        this.linkLive = source["linkLive"];
	        this.strings = this.convertValues(source["strings"], StringInfo);
	        this.chain = this.convertValues(source["chain"], ChainStatus);
	        this.health = this.convertValues(source["health"], HealthStatus);
	        this.readMillis = source["readMillis"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace config {
	
	export class Logging {
	    enabled: boolean;
	    intervalSec: number;
	    retentionDays: number;
	
	    static createFrom(source: any = {}) {
	        return new Logging(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.intervalSec = source["intervalSec"];
	        this.retentionDays = source["retentionDays"];
	    }
	}
	export class Polling {
	    pollSharedBus: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Polling(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pollSharedBus = source["pollSharedBus"];
	    }
	}
	export class Remote {
	    token: string;
	    stationId: number;
	
	    static createFrom(source: any = {}) {
	        return new Remote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	        this.stationId = source["stationId"];
	    }
	}

}

export namespace datastore {
	
	export class Stats {
	    path: string;
	    sizeBytes: number;
	    rows: Record<string, number>;
	    oldestTs: number;
	    newestTs: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.sizeBytes = source["sizeBytes"];
	        this.rows = source["rows"];
	        this.oldestTs = source["oldestTs"];
	        this.newestTs = source["newestTs"];
	    }
	}
	export class Status {
	    config: config.Logging;
	    running: boolean;
	    lastPlant: string;
	    lastBatt: string;
	    stats: Stats;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], config.Logging);
	        this.running = source["running"];
	        this.lastPlant = source["lastPlant"];
	        this.lastBatt = source["lastBatt"];
	        this.stats = this.convertValues(source["stats"], Stats);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace dse {
	
	export class Alarm {
	    index: number;
	    name: string;
	    state: string;
	
	    static createFrom(source: any = {}) {
	        return new Alarm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.state = source["state"];
	    }
	}
	export class Metric {
	    ok: boolean;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new Metric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.value = source["value"];
	    }
	}
	export class Snapshot {
	    ok: boolean;
	    timestamp: string;
	    oilPressureKPa: Metric;
	    coolantTempC: Metric;
	    fuelLevelPct: Metric;
	    chargeAltV: Metric;
	    batteryV: Metric;
	    engineRPM: Metric;
	    freqHz: Metric;
	    vLn: Metric[];
	    vLl: Metric[];
	    ampsL: Metric[];
	    wattsL: Metric[];
	    totalW: Metric;
	    totalVA: Metric;
	    totalVAr: Metric;
	    avgPF: Metric;
	    posKWh: Metric;
	    negKWh: Metric;
	    runHours: Metric;
	    starts: Metric;
	    running: boolean;
	    alarmsOk: boolean;
	    alarms: Alarm[];
	    readMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.timestamp = source["timestamp"];
	        this.oilPressureKPa = this.convertValues(source["oilPressureKPa"], Metric);
	        this.coolantTempC = this.convertValues(source["coolantTempC"], Metric);
	        this.fuelLevelPct = this.convertValues(source["fuelLevelPct"], Metric);
	        this.chargeAltV = this.convertValues(source["chargeAltV"], Metric);
	        this.batteryV = this.convertValues(source["batteryV"], Metric);
	        this.engineRPM = this.convertValues(source["engineRPM"], Metric);
	        this.freqHz = this.convertValues(source["freqHz"], Metric);
	        this.vLn = this.convertValues(source["vLn"], Metric);
	        this.vLl = this.convertValues(source["vLl"], Metric);
	        this.ampsL = this.convertValues(source["ampsL"], Metric);
	        this.wattsL = this.convertValues(source["wattsL"], Metric);
	        this.totalW = this.convertValues(source["totalW"], Metric);
	        this.totalVA = this.convertValues(source["totalVA"], Metric);
	        this.totalVAr = this.convertValues(source["totalVAr"], Metric);
	        this.avgPF = this.convertValues(source["avgPF"], Metric);
	        this.posKWh = this.convertValues(source["posKWh"], Metric);
	        this.negKWh = this.convertValues(source["negKWh"], Metric);
	        this.runHours = this.convertValues(source["runHours"], Metric);
	        this.starts = this.convertValues(source["starts"], Metric);
	        this.running = source["running"];
	        this.alarmsOk = source["alarmsOk"];
	        this.alarms = this.convertValues(source["alarms"], Alarm);
	        this.readMillis = source["readMillis"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace oztek {
	
	export class Metric {
	    ok: boolean;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new Metric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.value = source["value"];
	    }
	}
	export class Snapshot {
	    ok: boolean;
	    timestamp: string;
	    unit: number;
	    state: number;
	    stateText: string;
	    online: boolean;
	    gridConnected: boolean;
	    gridForming: boolean;
	    acPowerW: Metric;
	    reactiveVAR: Metric;
	    apparentVA: Metric;
	    powerFactor: Metric;
	    acCurrentA: Metric;
	    vLl: Metric;
	    vLn: Metric;
	    freqHz: Metric;
	    dcVoltage: Metric;
	    dcCurrent: Metric;
	    dcPowerW: Metric;
	    cabinetTempC: Metric;
	    heatsinkTempC: Metric;
	    heartbeat: number;
	    alarmRaw: number;
	    alarms: string[];
	    warningRaw: number;
	    warnings: string[];
	    faultRaw: number;
	    faults: string[];
	    factoryRaw: number;
	    factory: string[];
	    readMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.timestamp = source["timestamp"];
	        this.unit = source["unit"];
	        this.state = source["state"];
	        this.stateText = source["stateText"];
	        this.online = source["online"];
	        this.gridConnected = source["gridConnected"];
	        this.gridForming = source["gridForming"];
	        this.acPowerW = this.convertValues(source["acPowerW"], Metric);
	        this.reactiveVAR = this.convertValues(source["reactiveVAR"], Metric);
	        this.apparentVA = this.convertValues(source["apparentVA"], Metric);
	        this.powerFactor = this.convertValues(source["powerFactor"], Metric);
	        this.acCurrentA = this.convertValues(source["acCurrentA"], Metric);
	        this.vLl = this.convertValues(source["vLl"], Metric);
	        this.vLn = this.convertValues(source["vLn"], Metric);
	        this.freqHz = this.convertValues(source["freqHz"], Metric);
	        this.dcVoltage = this.convertValues(source["dcVoltage"], Metric);
	        this.dcCurrent = this.convertValues(source["dcCurrent"], Metric);
	        this.dcPowerW = this.convertValues(source["dcPowerW"], Metric);
	        this.cabinetTempC = this.convertValues(source["cabinetTempC"], Metric);
	        this.heatsinkTempC = this.convertValues(source["heatsinkTempC"], Metric);
	        this.heartbeat = source["heartbeat"];
	        this.alarmRaw = source["alarmRaw"];
	        this.alarms = source["alarms"];
	        this.warningRaw = source["warningRaw"];
	        this.warnings = source["warnings"];
	        this.faultRaw = source["faultRaw"];
	        this.faults = source["faults"];
	        this.factoryRaw = source["factoryRaw"];
	        this.factory = source["factory"];
	        this.readMillis = source["readMillis"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace plant {
	
	export class Device {
	    id: string;
	    name: string;
	    kind: string;
	    host: string;
	    port: number;
	    unit: number;
	    enabled: boolean;
	    notes: string;
	    sharedBus: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.unit = source["unit"];
	        this.enabled = source["enabled"];
	        this.notes = source["notes"];
	        this.sharedBus = source["sharedBus"];
	    }
	}
	export class Event {
	    at: string;
	    deviceId: string;
	    device: string;
	    severity: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.at = source["at"];
	        this.deviceId = source["deviceId"];
	        this.device = source["device"];
	        this.severity = source["severity"];
	        this.text = source["text"];
	    }
	}
	export class PlantHealth {
	    level: string;
	    headline: string;
	    reasons: string[];
	
	    static createFrom(source: any = {}) {
	        return new PlantHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.headline = source["headline"];
	        this.reasons = source["reasons"];
	    }
	}
	export class PowerBalance {
	    pvKW: number;
	    bessKW: number;
	    gensetKW: number;
	    loadKW: number;
	    derived: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PowerBalance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pvKW = source["pvKW"];
	        this.bessKW = source["bessKW"];
	        this.gensetKW = source["gensetKW"];
	        this.loadKW = source["loadKW"];
	        this.derived = source["derived"];
	    }
	}
	export class Reading {
	    id: string;
	    name: string;
	    kind: string;
	    host: string;
	    port: number;
	    unit: number;
	    enabled: boolean;
	    notes: string;
	    online: boolean;
	    stale: boolean;
	    err: string;
	    pollSkipped: boolean;
	    health: string;
	    headline: string;
	    bms?: bms.AggSummary;
	    oztek?: oztek.Snapshot;
	    sma?: sma.Snapshot;
	    dse?: dse.Snapshot;
	    readMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Reading(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.unit = source["unit"];
	        this.enabled = source["enabled"];
	        this.notes = source["notes"];
	        this.online = source["online"];
	        this.stale = source["stale"];
	        this.err = source["err"];
	        this.pollSkipped = source["pollSkipped"];
	        this.health = source["health"];
	        this.headline = source["headline"];
	        this.bms = this.convertValues(source["bms"], bms.AggSummary);
	        this.oztek = this.convertValues(source["oztek"], oztek.Snapshot);
	        this.sma = this.convertValues(source["sma"], sma.Snapshot);
	        this.dse = this.convertValues(source["dse"], dse.Snapshot);
	        this.readMillis = source["readMillis"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Snapshot {
	    timestamp: string;
	    devices: Reading[];
	    power: PowerBalance;
	    health: PlantHealth;
	    insights: string[];
	    events: Event[];
	    readMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.devices = this.convertValues(source["devices"], Reading);
	        this.power = this.convertValues(source["power"], PowerBalance);
	        this.health = this.convertValues(source["health"], PlantHealth);
	        this.insights = source["insights"];
	        this.events = this.convertValues(source["events"], Event);
	        this.readMillis = source["readMillis"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace sma {
	
	export class Metric {
	    ok: boolean;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new Metric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.value = source["value"];
	    }
	}
	export class MPPT {
	    currentA: Metric;
	    voltageV: Metric;
	    powerW: Metric;
	
	    static createFrom(source: any = {}) {
	        return new MPPT(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentA = this.convertValues(source["currentA"], Metric);
	        this.voltageV = this.convertValues(source["voltageV"], Metric);
	        this.powerW = this.convertValues(source["powerW"], Metric);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Snapshot {
	    ok: boolean;
	    timestamp: string;
	    deviceTypeRaw: number;
	    deviceType: string;
	    serial: number;
	    conditionRaw: number;
	    condition: string;
	    gridRelayRaw: number;
	    gridRelay: string;
	    derating: string;
	    acPowerW: Metric;
	    apparentVA: Metric;
	    gridV: Metric[];
	    gridA: Metric[];
	    freqHz: Metric;
	    mpptA: MPPT;
	    mpptB: MPPT;
	    dcPowerW: Metric;
	    dcLinkV: Metric;
	    totalYieldKWh: Metric;
	    dailyYieldKWh: Metric;
	    internalTempC: Metric;
	    readMillis: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.timestamp = source["timestamp"];
	        this.deviceTypeRaw = source["deviceTypeRaw"];
	        this.deviceType = source["deviceType"];
	        this.serial = source["serial"];
	        this.conditionRaw = source["conditionRaw"];
	        this.condition = source["condition"];
	        this.gridRelayRaw = source["gridRelayRaw"];
	        this.gridRelay = source["gridRelay"];
	        this.derating = source["derating"];
	        this.acPowerW = this.convertValues(source["acPowerW"], Metric);
	        this.apparentVA = this.convertValues(source["apparentVA"], Metric);
	        this.gridV = this.convertValues(source["gridV"], Metric);
	        this.gridA = this.convertValues(source["gridA"], Metric);
	        this.freqHz = this.convertValues(source["freqHz"], Metric);
	        this.mpptA = this.convertValues(source["mpptA"], MPPT);
	        this.mpptB = this.convertValues(source["mpptB"], MPPT);
	        this.dcPowerW = this.convertValues(source["dcPowerW"], Metric);
	        this.dcLinkV = this.convertValues(source["dcLinkV"], Metric);
	        this.totalYieldKWh = this.convertValues(source["totalYieldKWh"], Metric);
	        this.dailyYieldKWh = this.convertValues(source["dailyYieldKWh"], Metric);
	        this.internalTempC = this.convertValues(source["internalTempC"], Metric);
	        this.readMillis = source["readMillis"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

