export namespace bms {
	
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

