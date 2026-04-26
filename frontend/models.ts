	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_id = source["account_id"];
	        this.account_name = source["account_name"];
	        this.platform = source["platform"];
	        this.healthy = source["healthy"];
	        this.latency_ms = source["latency_ms"];
	        this.error = source["error"];
	    }
	}

	}

