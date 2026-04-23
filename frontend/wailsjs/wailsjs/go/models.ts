export namespace programs {
	
	export class EnvVar {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvVar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class Input {
	    name: string;
	    description: string;
	    notes: string;
	    tags: string[];
	    path: string;
	    workingDirectory: string;
	    args: string[];
	    env: EnvVar[];
	    runAsAdmin: boolean;
	    restartPolicy: string;
	    restartLimit: number;
	    restartDelaySeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new Input(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.notes = source["notes"];
	        this.tags = source["tags"];
	        this.path = source["path"];
	        this.workingDirectory = source["workingDirectory"];
	        this.args = source["args"];
	        this.env = this.convertValues(source["env"], EnvVar);
	        this.runAsAdmin = source["runAsAdmin"];
	        this.restartPolicy = source["restartPolicy"];
	        this.restartLimit = source["restartLimit"];
	        this.restartDelaySeconds = source["restartDelaySeconds"];
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
	export class ListQuery {
	    search: string;
	    status: string;
	    tag: string;
	    sortBy: string;
	    sortDirection: string;
	
	    static createFrom(source: any = {}) {
	        return new ListQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.search = source["search"];
	        this.status = source["status"];
	        this.tag = source["tag"];
	        this.sortBy = source["sortBy"];
	        this.sortDirection = source["sortDirection"];
	    }
	}
	export class LogEntry {
	    stream: string;
	    line: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stream = source["stream"];
	        this.line = source["line"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class LogQuery {
	    limit: number;
	    stream: string;
	
	    static createFrom(source: any = {}) {
	        return new LogQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.limit = source["limit"];
	        this.stream = source["stream"];
	    }
	}
	export class LogView {
	    programId: string;
	    entries: LogEntry[];
	    truncated: boolean;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new LogView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.programId = source["programId"];
	        this.entries = this.convertValues(source["entries"], LogEntry);
	        this.truncated = source["truncated"];
	        this.total = source["total"];
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
	export class View {
	    id: string;
	    name: string;
	    description: string;
	    notes: string;
	    tags: string[];
	    path: string;
	    kind: string;
	    workingDirectory: string;
	    args: string[];
	    env: EnvVar[];
	    runAsAdmin: boolean;
	    restartPolicy: string;
	    restartLimit: number;
	    restartDelaySeconds: number;
	    status: string;
	    lastError: string;
	    pid: number;
	    startedAt: string;
	    lastExitAt: string;
	    memoryWorkingSetBytes: number;
	    memoryPrivateBytes: number;
	    restartCount: number;
	    elevated: boolean;
	    canReconnect: boolean;
	    lastLogAt: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new View(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.notes = source["notes"];
	        this.tags = source["tags"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.workingDirectory = source["workingDirectory"];
	        this.args = source["args"];
	        this.env = this.convertValues(source["env"], EnvVar);
	        this.runAsAdmin = source["runAsAdmin"];
	        this.restartPolicy = source["restartPolicy"];
	        this.restartLimit = source["restartLimit"];
	        this.restartDelaySeconds = source["restartDelaySeconds"];
	        this.status = source["status"];
	        this.lastError = source["lastError"];
	        this.pid = source["pid"];
	        this.startedAt = source["startedAt"];
	        this.lastExitAt = source["lastExitAt"];
	        this.memoryWorkingSetBytes = source["memoryWorkingSetBytes"];
	        this.memoryPrivateBytes = source["memoryPrivateBytes"];
	        this.restartCount = source["restartCount"];
	        this.elevated = source["elevated"];
	        this.canReconnect = source["canReconnect"];
	        this.lastLogAt = source["lastLogAt"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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

