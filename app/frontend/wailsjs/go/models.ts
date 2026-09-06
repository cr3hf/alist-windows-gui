export namespace alist {
	
	export class LogLine {
	    time: string;
	    level: string;
	    msg: string;
	
	    static createFrom(source: any = {}) {
	        return new LogLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.msg = source["msg"];
	    }
	}
	export class State {
	    status: string;
	    isPortable: boolean;
	    autoStart: boolean;
	    autoStartService: boolean;
	    address: string;
	    httpPort: number;
	    httpsPort: number;
	    url: string;
	    account: string;
	    dataDir: string;
	    exePath: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.isPortable = source["isPortable"];
	        this.autoStart = source["autoStart"];
	        this.autoStartService = source["autoStartService"];
	        this.address = source["address"];
	        this.httpPort = source["httpPort"];
	        this.httpsPort = source["httpsPort"];
	        this.url = source["url"];
	        this.account = source["account"];
	        this.dataDir = source["dataDir"];
	        this.exePath = source["exePath"];
	    }
	}

}

