package database

import "database/sql"

func migrate(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS proxies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			protocol TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			username TEXT,
			password TEXT,
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			platform TEXT DEFAULT 'anthropic',
			rate_multiplier REAL DEFAULT 1.0,
			is_exclusive INTEGER DEFAULT 0,
			status TEXT DEFAULT 'active',
			model_routing TEXT DEFAULT '{}',
			model_routing_enabled INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			platform TEXT NOT NULL,
			type TEXT NOT NULL,
			credentials TEXT NOT NULL DEFAULT '{}',
			extra TEXT DEFAULT '{}',
			proxy_id INTEGER,
			concurrency INTEGER DEFAULT 3,
			priority INTEGER DEFAULT 50,
			status TEXT DEFAULT 'active',
			error_message TEXT,
			schedulable INTEGER DEFAULT 1,
			rate_limited_at DATETIME,
			rate_limit_reset_at DATETIME,
			overload_until DATETIME,
			last_used_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (proxy_id) REFERENCES proxies(id)
		)`,
		`CREATE TABLE IF NOT EXISTS account_groups (
			account_id INTEGER NOT NULL,
			group_id INTEGER NOT NULL,
			PRIMARY KEY (account_id, group_id),
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
			FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			key TEXT NOT NULL UNIQUE,
			group_id INTEGER,
			status TEXT DEFAULT 'active',
			ip_whitelist TEXT DEFAULT '[]',
			ip_blacklist TEXT DEFAULT '[]',
			last_used_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES groups(id)
		)`,
		`CREATE TABLE IF NOT EXISTS usage_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL,
			api_key_id INTEGER,
			account_id INTEGER NOT NULL,
			group_id INTEGER,
			model TEXT NOT NULL,
			requested_model TEXT,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			cache_creation_tokens INTEGER DEFAULT 0,
			cache_read_tokens INTEGER DEFAULT 0,
			input_cost REAL DEFAULT 0,
			output_cost REAL DEFAULT 0,
			cache_creation_cost REAL DEFAULT 0,
			cache_read_cost REAL DEFAULT 0,
			total_cost REAL DEFAULT 0,
			stream INTEGER DEFAULT 0,
			duration_ms INTEGER,
			first_token_ms INTEGER,
			status_code INTEGER,
			error_type TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (api_key_id) REFERENCES api_keys(id),
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
			FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_model ON usage_logs(model)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_account_id ON usage_logs(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_logs_group_id ON usage_logs(group_id)`,
		`CREATE TABLE IF NOT EXISTS model_pricing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model TEXT NOT NULL UNIQUE,
			input_price REAL DEFAULT 0,
			output_price REAL DEFAULT 0,
			cache_creation_price REAL DEFAULT 0,
			cache_read_price REAL DEFAULT 0,
			image_price REAL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}
	for _, stmt := range tables {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	// Migrations for existing databases
	alterStatements := []string{
		`ALTER TABLE accounts ADD COLUMN base_url TEXT`,
		`ALTER TABLE accounts ADD COLUMN multiplier REAL DEFAULT 1.0`,
	}
	for _, stmt := range alterStatements {
		db.Exec(stmt) // Ignore errors if column already exists
	}
	return nil
}
