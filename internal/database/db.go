package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection.
type DB struct {
	*sql.DB
}

// Init opens (or creates) the SQLite database at dbPath and runs migrations.
func Init(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return &DB{db}, nil
}

// Close closes the database connection.
func (db *DB) Close() {
	if db.DB != nil {
		db.DB.Close()
	}
}
