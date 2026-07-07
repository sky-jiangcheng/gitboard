package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// InitDB initializes the SQLite database, creating tables if they don't exist.
// Returns the database handle and any error encountered.
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close() //nolint:errcheck // close on init failure, error irrelevant
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close() //nolint:errcheck
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := createTables(db); err != nil {
		db.Close() //nolint:errcheck
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	if err := insertDefaults(db); err != nil {
		db.Close() //nolint:errcheck
		return nil, fmt.Errorf("failed to insert defaults: %w", err)
	}

	return db, nil
}

// createTables creates all required tables if they do not exist.
func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS scan_roots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		root_path TEXT NOT NULL,
		level_override INTEGER DEFAULT 0,
		is_auto_grouped BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS repositories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		project_id INTEGER,
		last_scanned_at DATETIME,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE TABLE IF NOT EXISTS daily_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repository_id INTEGER NOT NULL,
		stat_date DATE NOT NULL,
		author TEXT NOT NULL,
		files_changed INTEGER DEFAULT 0,
		lines_added INTEGER DEFAULT 0,
		lines_deleted INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (repository_id) REFERENCES repositories(id),
		UNIQUE(repository_id, stat_date, author)
	);

	CREATE TABLE IF NOT EXISTS app_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`

	_, err := db.Exec(schema)
	return err
}

// insertDefaults inserts default configuration values if they don't exist.
func insertDefaults(db *sql.DB) error {
	defaults := map[string]string{
		"daily_code_standard": "500",
		"scan_depth":          "5",
	}

	for key, value := range defaults {
		_, err := db.Exec(
			"INSERT OR IGNORE INTO app_config (key, value) VALUES (?, ?)",
			key, value,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
