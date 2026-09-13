export namespace config {
	
	export class Config {
	    doc_urls: string;
	    cups_rooms: string;
	    transport: string;
	    max_token: string;
	    max_uid: string;
	    secret_key: string;
	    socks_port: number;
	    mode: string;
	    theme: string;
	    bypass: string;
	    auto_start: boolean;
	    start_minimized: boolean;
	    close_to_tray: boolean;
	    debug: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.doc_urls = source["doc_urls"];
	        this.cups_rooms = source["cups_rooms"];
	        this.transport = source["transport"];
	        this.max_token = source["max_token"];
	        this.max_uid = source["max_uid"];
	        this.secret_key = source["secret_key"];
	        this.socks_port = source["socks_port"];
	        this.mode = source["mode"];
	        this.theme = source["theme"];
	        this.bypass = source["bypass"];
	        this.auto_start = source["auto_start"];
	        this.start_minimized = source["start_minimized"];
	        this.close_to_tray = source["close_to_tray"];
	        this.debug = source["debug"];
	    }
	}

}

export namespace core {
	
	export class ConnectionStatus {
	    connected: boolean;
	    mode: string;
	    transport: string;
	    uptime: string;
	    upload_speed: string;
	    download_speed: string;
	    total_upload: string;
	    total_download: string;
	    ping_ms: number;
	    recent_logs: string;
	    current_doc_url: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.mode = source["mode"];
	        this.transport = source["transport"];
	        this.uptime = source["uptime"];
	        this.upload_speed = source["upload_speed"];
	        this.download_speed = source["download_speed"];
	        this.total_upload = source["total_upload"];
	        this.total_download = source["total_download"];
	        this.ping_ms = source["ping_ms"];
	        this.recent_logs = source["recent_logs"];
	        this.current_doc_url = source["current_doc_url"];
	    }
	}

}

