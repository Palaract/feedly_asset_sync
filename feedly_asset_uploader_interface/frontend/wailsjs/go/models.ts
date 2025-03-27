export namespace main {
	
	export class Config {
	    upload_url: string;
	    api_key: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.upload_url = source["upload_url"];
	        this.api_key = source["api_key"];
	    }
	}

}

