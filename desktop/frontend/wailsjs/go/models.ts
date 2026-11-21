export namespace api {
	
	export class ClientInfo {
	    running: boolean;
	    host: string;
	    listen: string;
	    user: string;
	    // Go type: time
	    started: any;
	
	    static createFrom(source: any = {}) {
	        return new ClientInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.host = source["host"];
	        this.listen = source["listen"];
	        this.user = source["user"];
	        this.started = this.convertValues(source["started"], null);
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
	export class ClientOptions {
	    host: string;
	    user: string;
	    password: string;
	    listen: string;
	
	    static createFrom(source: any = {}) {
	        return new ClientOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.listen = source["listen"];
	    }
	}
	export class Defaults {
	    name: string;
	    lanAddress: string;
	    hostListen: string;
	    clientListen: string;
	    hostPort: string;
	
	    static createFrom(source: any = {}) {
	        return new Defaults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.lanAddress = source["lanAddress"];
	        this.hostListen = source["hostListen"];
	        this.clientListen = source["clientListen"];
	        this.hostPort = source["hostPort"];
	    }
	}
	export class HostInfo {
	    running: boolean;
	    name: string;
	    listen: string;
	    advertise: string;
	    user: string;
	    password: string;
	    noAuth: boolean;
	    mdns: boolean;
	    invite: string;
	    qr: string;
	    // Go type: time
	    started: any;
	
	    static createFrom(source: any = {}) {
	        return new HostInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.name = source["name"];
	        this.listen = source["listen"];
	        this.advertise = source["advertise"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.noAuth = source["noAuth"];
	        this.mdns = source["mdns"];
	        this.invite = source["invite"];
	        this.qr = source["qr"];
	        this.started = this.convertValues(source["started"], null);
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
	export class HostOptions {
	    listen: string;
	    user: string;
	    password: string;
	    noAuth: boolean;
	    name: string;
	    advertise: string;
	    disableMDNS: boolean;
	
	    static createFrom(source: any = {}) {
	        return new HostOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listen = source["listen"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.noAuth = source["noAuth"];
	        this.name = source["name"];
	        this.advertise = source["advertise"];
	        this.disableMDNS = source["disableMDNS"];
	    }
	}
	export class LogEntry {
	    seq: number;
	    // Go type: time
	    time: any;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.time = this.convertValues(source["time"], null);
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

}

export namespace discovery {
	
	export class Peer {
	    name: string;
	    hostID: string;
	    address: string;
	
	    static createFrom(source: any = {}) {
	        return new Peer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.hostID = source["hostID"];
	        this.address = source["address"];
	    }
	}

}

