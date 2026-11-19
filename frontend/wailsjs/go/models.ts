export namespace main {
	
	export class Bootstrap {
	    settings: store.Settings;
	    formats: media.FormatSpec[];
	    encryptedExts: string[];
	    plainExts: string[];
	    version: string;
	    logPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Bootstrap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], store.Settings);
	        this.formats = this.convertValues(source["formats"], media.FormatSpec);
	        this.encryptedExts = source["encryptedExts"];
	        this.plainExts = source["plainExts"];
	        this.version = source["version"];
	        this.logPath = source["logPath"];
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
	export class StartOptions {
	    outputDir: string;
	    format: string;
	    bitrate: string;
	    useVbr: boolean;
	    quality: number;
	    sampleRate: number;
	    channels: number;
	    compression: number;
	    bitDepth: number;
	    workers: number;
	    kggDbPath: string;
	
	    static createFrom(source: any = {}) {
	        return new StartOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputDir = source["outputDir"];
	        this.format = source["format"];
	        this.bitrate = source["bitrate"];
	        this.useVbr = source["useVbr"];
	        this.quality = source["quality"];
	        this.sampleRate = source["sampleRate"];
	        this.channels = source["channels"];
	        this.compression = source["compression"];
	        this.bitDepth = source["bitDepth"];
	        this.workers = source["workers"];
	        this.kggDbPath = source["kggDbPath"];
	    }
	}

}

export namespace media {
	
	export class FormatSpec {
	    id: string;
	    name: string;
	    ext: string;
	    lossless: boolean;
	    cover: boolean;
	    hint: string;
	
	    static createFrom(source: any = {}) {
	        return new FormatSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.ext = source["ext"];
	        this.lossless = source["lossless"];
	        this.cover = source["cover"];
	        this.hint = source["hint"];
	    }
	}
	export class Info {
	    container: string;
	    codec: string;
	    duration: number;
	    sampleRate: number;
	    channels: number;
	    bitRate: number;
	    hasCover: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.container = source["container"];
	        this.codec = source["codec"];
	        this.duration = source["duration"];
	        this.sampleRate = source["sampleRate"];
	        this.channels = source["channels"];
	        this.bitRate = source["bitRate"];
	        this.hasCover = source["hasCover"];
	    }
	}

}

export namespace store {
	
	export class Settings {
	    outputDir: string;
	    format: string;
	    bitrate: string;
	    useVbr: boolean;
	    quality: number;
	    sampleRate: number;
	    channels: number;
	    compression: number;
	    bitDepth: number;
	    workers: number;
	    acceptedDisclaimer: boolean;
	    kggDbPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputDir = source["outputDir"];
	        this.format = source["format"];
	        this.bitrate = source["bitrate"];
	        this.useVbr = source["useVbr"];
	        this.quality = source["quality"];
	        this.sampleRate = source["sampleRate"];
	        this.channels = source["channels"];
	        this.compression = source["compression"];
	        this.bitDepth = source["bitDepth"];
	        this.workers = source["workers"];
	        this.acceptedDisclaimer = source["acceptedDisclaimer"];
	        this.kggDbPath = source["kggDbPath"];
	    }
	}

}

export namespace task {
	
	export class Task {
	    id: string;
	    src: string;
	    name: string;
	    ext: string;
	    encrypted: boolean;
	    state: string;
	    progress: number;
	    message: string;
	    output: string;
	    info?: media.Info;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.src = source["src"];
	        this.name = source["name"];
	        this.ext = source["ext"];
	        this.encrypted = source["encrypted"];
	        this.state = source["state"];
	        this.progress = source["progress"];
	        this.message = source["message"];
	        this.output = source["output"];
	        this.info = this.convertValues(source["info"], media.Info);
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

