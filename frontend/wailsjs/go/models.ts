export namespace cleaner {
	
	export class CleanCategory {
	    id: string;
	    name: string;
	    description: string;
	    icon: string;
	    risk: string;
	    size: number;
	    fileCount: number;
	
	    static createFrom(source: any = {}) {
	        return new CleanCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.icon = source["icon"];
	        this.risk = source["risk"];
	        this.size = source["size"];
	        this.fileCount = source["fileCount"];
	    }
	}
	export class CleanResult {
	    freedSpace: number;
	    deletedFiles: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new CleanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.freedSpace = source["freedSpace"];
	        this.deletedFiles = source["deletedFiles"];
	        this.errors = source["errors"];
	    }
	}
	export class ScanResult {
	    categories: CleanCategory[];
	    totalSize: number;
	    totalFiles: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categories = this.convertValues(source["categories"], CleanCategory);
	        this.totalSize = source["totalSize"];
	        this.totalFiles = source["totalFiles"];
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

export namespace gaming {
	
	export class BoostStatus {
	    active: boolean;
	    profile: string;
	    tweaksApplied: string[];
	    startedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new BoostStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.profile = source["profile"];
	        this.tweaksApplied = source["tweaksApplied"];
	        this.startedAt = source["startedAt"];
	    }
	}
	export class GPUInfo {
	    name: string;
	    vendor: string;
	    driver: string;
	    profileName: string;
	
	    static createFrom(source: any = {}) {
	        return new GPUInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.vendor = source["vendor"];
	        this.driver = source["driver"];
	        this.profileName = source["profileName"];
	    }
	}
	export class GameProfile {
	    id: string;
	    name: string;
	    icon: string;
	    description: string;
	    tweaks: Record<string, boolean>;
	
	    static createFrom(source: any = {}) {
	        return new GameProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.description = source["description"];
	        this.tweaks = source["tweaks"];
	    }
	}
	export class TweakInfo {
	    id: string;
	    name: string;
	    description: string;
	    category: string;
	    enabled: boolean;
	    applied: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TweakInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.enabled = source["enabled"];
	        this.applied = source["applied"];
	    }
	}

}

export namespace memory {
	
	export class ProcessMemory {
	    name: string;
	    pid: number;
	    memory: number;
	    percent: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessMemory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pid = source["pid"];
	        this.memory = source["memory"];
	        this.percent = source["percent"];
	    }
	}
	export class MemoryStatus {
	    total: number;
	    used: number;
	    available: number;
	    usagePercent: number;
	    cached: number;
	    topProcesses: ProcessMemory[];
	
	    static createFrom(source: any = {}) {
	        return new MemoryStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.used = source["used"];
	        this.available = source["available"];
	        this.usagePercent = source["usagePercent"];
	        this.cached = source["cached"];
	        this.topProcesses = this.convertValues(source["topProcesses"], ProcessMemory);
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

export namespace monitor {
	
	export class BenchmarkResult {
	    cpuScore: number;
	    ramScore: number;
	    diskScore: number;
	    overallScore: number;
	    duration: string;
	
	    static createFrom(source: any = {}) {
	        return new BenchmarkResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpuScore = source["cpuScore"];
	        this.ramScore = source["ramScore"];
	        this.diskScore = source["diskScore"];
	        this.overallScore = source["overallScore"];
	        this.duration = source["duration"];
	    }
	}
	export class MonitorSnapshot {
	    timestamp: number;
	    cpuUsage: number;
	    ramUsage: number;
	    gpuTemp: number;
	    cpuTemp: number;
	    diskUsage: number;
	    fanSpeed: number;
	
	    static createFrom(source: any = {}) {
	        return new MonitorSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.cpuUsage = source["cpuUsage"];
	        this.ramUsage = source["ramUsage"];
	        this.gpuTemp = source["gpuTemp"];
	        this.cpuTemp = source["cpuTemp"];
	        this.diskUsage = source["diskUsage"];
	        this.fanSpeed = source["fanSpeed"];
	    }
	}

}

export namespace network {
	
	export class DNSPreset {
	    id: string;
	    name: string;
	    primary: string;
	    secondary: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new DNSPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.primary = source["primary"];
	        this.secondary = source["secondary"];
	        this.description = source["description"];
	    }
	}
	export class NetworkStatus {
	    currentDns: string;
	    nagleDisabled: boolean;
	    adapter: string;
	    ipAddress: string;
	    gateway: string;
	
	    static createFrom(source: any = {}) {
	        return new NetworkStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentDns = source["currentDns"];
	        this.nagleDisabled = source["nagleDisabled"];
	        this.adapter = source["adapter"];
	        this.ipAddress = source["ipAddress"];
	        this.gateway = source["gateway"];
	    }
	}

}

export namespace privacy {
	
	export class PrivacyTweak {
	    id: string;
	    name: string;
	    description: string;
	    category: string;
	    enabled: boolean;
	    applied: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PrivacyTweak(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.enabled = source["enabled"];
	        this.applied = source["applied"];
	    }
	}

}

export namespace startup {
	
	export class StartupItem {
	    name: string;
	    path: string;
	    publisher: string;
	    impact: string;
	    enabled: boolean;
	    location: string;
	
	    static createFrom(source: any = {}) {
	        return new StartupItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.publisher = source["publisher"];
	        this.impact = source["impact"];
	        this.enabled = source["enabled"];
	        this.location = source["location"];
	    }
	}

}

export namespace system {
	
	export class DiskInfo {
	    drive: string;
	    total: number;
	    used: number;
	    free: number;
	    usagePercent: number;
	    fsType: string;
	
	    static createFrom(source: any = {}) {
	        return new DiskInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.drive = source["drive"];
	        this.total = source["total"];
	        this.used = source["used"];
	        this.free = source["free"];
	        this.usagePercent = source["usagePercent"];
	        this.fsType = source["fsType"];
	    }
	}
	export class GPUDetail {
	    name: string;
	    driver: string;
	    vram: number;
	
	    static createFrom(source: any = {}) {
	        return new GPUDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.driver = source["driver"];
	        this.vram = source["vram"];
	    }
	}
	export class PhysDisk {
	    model: string;
	    size: number;
	    mediaType: string;
	    interface: string;
	
	    static createFrom(source: any = {}) {
	        return new PhysDisk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.size = source["size"];
	        this.mediaType = source["mediaType"];
	        this.interface = source["interface"];
	    }
	}
	export class RAMModule {
	    manufacturer: string;
	    capacity: number;
	    speed: number;
	    partNumber: string;
	    formFactor: string;
	    slot: string;
	
	    static createFrom(source: any = {}) {
	        return new RAMModule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manufacturer = source["manufacturer"];
	        this.capacity = source["capacity"];
	        this.speed = source["speed"];
	        this.partNumber = source["partNumber"];
	        this.formFactor = source["formFactor"];
	        this.slot = source["slot"];
	    }
	}
	export class SystemInfo {
	    os: string;
	    hostname: string;
	    platform: string;
	    cpuModel: string;
	    cpuCores: number;
	    cpuThreads: number;
	    cpuUsage: number;
	    ramTotal: number;
	    ramUsed: number;
	    ramUsage: number;
	    ramModules: RAMModule[];
	    gpuName: string;
	    gpuDriver: string;
	    gpus: GPUDetail[];
	    disks: DiskInfo[];
	    physDisks: PhysDisk[];
	    uptime: string;
	    healthScore: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.hostname = source["hostname"];
	        this.platform = source["platform"];
	        this.cpuModel = source["cpuModel"];
	        this.cpuCores = source["cpuCores"];
	        this.cpuThreads = source["cpuThreads"];
	        this.cpuUsage = source["cpuUsage"];
	        this.ramTotal = source["ramTotal"];
	        this.ramUsed = source["ramUsed"];
	        this.ramUsage = source["ramUsage"];
	        this.ramModules = this.convertValues(source["ramModules"], RAMModule);
	        this.gpuName = source["gpuName"];
	        this.gpuDriver = source["gpuDriver"];
	        this.gpus = this.convertValues(source["gpus"], GPUDetail);
	        this.disks = this.convertValues(source["disks"], DiskInfo);
	        this.physDisks = this.convertValues(source["physDisks"], PhysDisk);
	        this.uptime = source["uptime"];
	        this.healthScore = source["healthScore"];
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

export namespace toolkit {
	
	export class BloatwareApp {
	    name: string;
	    packageName: string;
	    publisher: string;
	    installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BloatwareApp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.packageName = source["packageName"];
	        this.publisher = source["publisher"];
	        this.installed = source["installed"];
	    }
	}
	export class ToolResult {
	    name: string;
	    success: boolean;
	    output: string;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ToolResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.success = source["success"];
	        this.output = source["output"];
	        this.errors = source["errors"];
	    }
	}

}

