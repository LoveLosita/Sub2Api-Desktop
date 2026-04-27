export namespace config {
	
	export class LogConfig {
	    Level: string;
	
	    static createFrom(source: any = {}) {
	        return new LogConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Level = source["Level"];
	    }
	}
	export class DatabaseConfig {
	    Path: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	    }
	}
	export class GatewayConfig {
	    MaxBodySize: number;
	    MaxAccountRetries: number;
	    PricingURL: string;
	
	    static createFrom(source: any = {}) {
	        return new GatewayConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MaxBodySize = source["MaxBodySize"];
	        this.MaxAccountRetries = source["MaxAccountRetries"];
	        this.PricingURL = source["PricingURL"];
	    }
	}
	export class ServerConfig {
	    Port: number;
	    Host: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Port = source["Port"];
	        this.Host = source["Host"];
	    }
	}
	export class Config {
	    Server: ServerConfig;
	    Gateway: GatewayConfig;
	    Database: DatabaseConfig;
	    Log: LogConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Server = this.convertValues(source["Server"], ServerConfig);
	        this.Gateway = this.convertValues(source["Gateway"], GatewayConfig);
	        this.Database = this.convertValues(source["Database"], DatabaseConfig);
	        this.Log = this.convertValues(source["Log"], LogConfig);
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

export namespace model {
	
	export class Group {
	    id: number;
	    name: string;
	    description?: string;
	    platform: string;
	    rate_multiplier: number;
	    is_exclusive: boolean;
	    status: string;
	    model_routing: Record<string, Array<number>>;
	    model_routing_enabled: boolean;
	    account_ids: number[];
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Group(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.platform = source["platform"];
	        this.rate_multiplier = source["rate_multiplier"];
	        this.is_exclusive = source["is_exclusive"];
	        this.status = source["status"];
	        this.model_routing = source["model_routing"];
	        this.model_routing_enabled = source["model_routing_enabled"];
	        this.account_ids = source["account_ids"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class APIKey {
	    id: number;
	    name: string;
	    key: string;
	    group_id?: number;
	    status: string;
	    ip_whitelist: string[];
	    ip_blacklist: string[];
	    // Go type: time
	    last_used_at?: any;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    group?: Group;
	
	    static createFrom(source: any = {}) {
	        return new APIKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.key = source["key"];
	        this.group_id = source["group_id"];
	        this.status = source["status"];
	        this.ip_whitelist = source["ip_whitelist"];
	        this.ip_blacklist = source["ip_blacklist"];
	        this.last_used_at = this.convertValues(source["last_used_at"], null);
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.group = this.convertValues(source["group"], Group);
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
	export class Account {
	    id: number;
	    name: string;
	    platform: string;
	    type: string;
	    credentials: Record<string, any>;
	    extra: Record<string, any>;
	    proxy_id?: number;
	    base_url?: string;
	    concurrency: number;
	    priority: number;
	    status: string;
	    error_message?: string;
	    schedulable: boolean;
	    // Go type: time
	    rate_limited_at?: any;
	    // Go type: time
	    rate_limit_reset_at?: any;
	    // Go type: time
	    overload_until?: any;
	    // Go type: time
	    last_used_at?: any;
	    group_ids: number[];
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.platform = source["platform"];
	        this.type = source["type"];
	        this.credentials = source["credentials"];
	        this.extra = source["extra"];
	        this.proxy_id = source["proxy_id"];
	        this.base_url = source["base_url"];
	        this.concurrency = source["concurrency"];
	        this.priority = source["priority"];
	        this.status = source["status"];
	        this.error_message = source["error_message"];
	        this.schedulable = source["schedulable"];
	        this.rate_limited_at = this.convertValues(source["rate_limited_at"], null);
	        this.rate_limit_reset_at = this.convertValues(source["rate_limit_reset_at"], null);
	        this.overload_until = this.convertValues(source["overload_until"], null);
	        this.last_used_at = this.convertValues(source["last_used_at"], null);
	        this.group_ids = source["group_ids"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class ModelCostEntry {
	    model: string;
	    requests: number;
	    tokens: number;
	    cost: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelCostEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.requests = source["requests"];
	        this.tokens = source["tokens"];
	        this.cost = source["cost"];
	    }
	}
	export class DashboardStats {
	    total_accounts: number;
	    active_accounts: number;
	    error_accounts: number;
	    rate_limit_accounts: number;
	    total_requests: number;
	    today_requests: number;
	    total_tokens: number;
	    today_tokens: number;
	    total_cost: number;
	    today_cost: number;
	    by_model: ModelCostEntry[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_accounts = source["total_accounts"];
	        this.active_accounts = source["active_accounts"];
	        this.error_accounts = source["error_accounts"];
	        this.rate_limit_accounts = source["rate_limit_accounts"];
	        this.total_requests = source["total_requests"];
	        this.today_requests = source["today_requests"];
	        this.total_tokens = source["total_tokens"];
	        this.today_tokens = source["today_tokens"];
	        this.total_cost = source["total_cost"];
	        this.today_cost = source["today_cost"];
	        this.by_model = this.convertValues(source["by_model"], ModelCostEntry);
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
	
	
	export class ModelPricing {
	    id: number;
	    model: string;
	    input_price: number;
	    output_price: number;
	    cache_creation_price: number;
	    cache_read_price: number;
	    image_price: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelPricing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.model = source["model"];
	        this.input_price = source["input_price"];
	        this.output_price = source["output_price"];
	        this.cache_creation_price = source["cache_creation_price"];
	        this.cache_read_price = source["cache_read_price"];
	        this.image_price = source["image_price"];
	    }
	}
	export class Proxy {
	    id: number;
	    name: string;
	    protocol: string;
	    host: string;
	    port: number;
	    username?: string;
	    password?: string;
	    status: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Proxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.status = source["status"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class UsageLog {
	    id: number;
	    request_id: string;
	    api_key_id?: number;
	    account_id: number;
	    account_name: string;
	    group_id?: number;
	    model: string;
	    requested_model?: string;
	    input_tokens: number;
	    output_tokens: number;
	    cache_creation_tokens: number;
	    cache_read_tokens: number;
	    input_cost: number;
	    output_cost: number;
	    cache_creation_cost: number;
	    cache_read_cost: number;
	    total_cost: number;
	    stream: boolean;
	    duration_ms?: number;
	    first_token_ms?: number;
	    status_code?: number;
	    error_type?: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new UsageLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.request_id = source["request_id"];
	        this.api_key_id = source["api_key_id"];
	        this.account_id = source["account_id"];
	        this.account_name = source["account_name"];
	        this.group_id = source["group_id"];
	        this.model = source["model"];
	        this.requested_model = source["requested_model"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.cache_creation_tokens = source["cache_creation_tokens"];
	        this.cache_read_tokens = source["cache_read_tokens"];
	        this.input_cost = source["input_cost"];
	        this.output_cost = source["output_cost"];
	        this.cache_creation_cost = source["cache_creation_cost"];
	        this.cache_read_cost = source["cache_read_cost"];
	        this.total_cost = source["total_cost"];
	        this.stream = source["stream"];
	        this.duration_ms = source["duration_ms"];
	        this.first_token_ms = source["first_token_ms"];
	        this.status_code = source["status_code"];
	        this.error_type = source["error_type"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class UsageListResult {
	    logs: UsageLog[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new UsageListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logs = this.convertValues(source["logs"], UsageLog);
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

}

export namespace service {
	
	export class HealthCheckResult {
	    account_id: number;
	    account_name: string;
	    platform: string;
	    healthy: boolean;
	    latency_ms: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthCheckResult(source);
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

