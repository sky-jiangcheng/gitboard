package db

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

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

	if err := upgradeSchema(db); err != nil {
		db.Close() //nolint:errcheck
		return nil, fmt.Errorf("failed to upgrade schema: %w", err)
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

	CREATE TABLE IF NOT EXISTS project_todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		completed BOOLEAN DEFAULT 0,
		priority INTEGER DEFAULT 0,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS project_notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		title TEXT DEFAULT '',
		content TEXT NOT NULL,
		tags TEXT DEFAULT '',
		kind TEXT DEFAULT 'other',
		pinned INTEGER DEFAULT 0,
		source TEXT DEFAULT 'manual',
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS repo_meta (
		repository_id INTEGER PRIMARY KEY,
		tech_stack TEXT DEFAULT '[]',
		readme_excerpt TEXT DEFAULT '',
		languages TEXT DEFAULT '{}',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
	);
	`

	_, err := db.Exec(schema)
	return err
}

// migrationVersionKey is the app_config key tracking the applied schema version.
const migrationVersionKey = "schema_version"

// upgradeSchema applies incremental schema changes for existing databases.
// Migrations are tracked by a monotonically increasing version number stored in
// app_config, so each migration runs exactly once even if ALTER errors leave a
// column already present.
func upgradeSchema(db *sql.DB) error {
	from, err := readSchemaVersion(db)
	if err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}

	migrations := []migration{
		// v1: add is_starred to projects (introduced v0.11.0)
		{id: 1, sql: "ALTER TABLE projects ADD COLUMN is_starred INTEGER DEFAULT 0"},
		// v2: structured notes metadata
		{id: 2, sql: []string{
			"ALTER TABLE project_notes ADD COLUMN title TEXT DEFAULT ''",
			"ALTER TABLE project_notes ADD COLUMN tags TEXT DEFAULT ''",
			"ALTER TABLE project_notes ADD COLUMN kind TEXT DEFAULT 'other'",
			"ALTER TABLE project_notes ADD COLUMN pinned INTEGER DEFAULT 0",
			"ALTER TABLE project_notes ADD COLUMN source TEXT DEFAULT 'manual'",
		}},
		// v3: knowledge mining cache + config key for git user
		{id: 3, sql: []string{
			"CREATE TABLE IF NOT EXISTS repo_meta (" +
				"repository_id INTEGER PRIMARY KEY," +
				"tech_stack TEXT DEFAULT '[]'," +
				"readme_excerpt TEXT DEFAULT ''," +
				"languages TEXT DEFAULT '{}'," +
				"updated_at DATETIME DEFAULT CURRENT_TIMESTAMP," +
				"FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE)",
		}},
	}

	for _, m := range migrations {
		if m.id <= from {
			continue
		}
		if err := m.apply(db); err != nil {
			logMigrationError(m.id, err)
			// Do not return: a failed ALTER (e.g. column already exists) must not
			// block the version stamp, or re-runs would repeat forever.
		}
		if err := writeSchemaVersion(db, m.id); err != nil {
			return fmt.Errorf("failed to stamp schema version %d: %w", m.id, err)
		}
	}
	return nil
}

// migration represents one schema change.
type migration struct {
	id  int
	sql any // string or []string
}

func (m migration) apply(db *sql.DB) error {
	stmts := stmtList(m.sql)
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			// SQLite returns "duplicate column name" when a column already exists.
			// Treat that as already-applied rather than a hard failure.
			if isAlreadyExistsErr(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func stmtList(s any) []string {
	switch v := s.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	default:
		return nil
	}
}

func isAlreadyExistsErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate column name") || strings.Contains(msg, "already exists")
}

func readSchemaVersion(db *sql.DB) (int, error) {
	// app_config.value is TEXT, so scan into a string and parse.
	var raw string
	err := db.QueryRow("SELECT value FROM app_config WHERE key = ?", migrationVersionKey).Scan(&raw)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, nil
	}
	return v, nil
}

func writeSchemaVersion(db *sql.DB, version int) error {
	_, err := db.Exec(
		"INSERT OR REPLACE INTO app_config (key, value) VALUES (?, ?)",
		migrationVersionKey, strconv.Itoa(version),
	)
	return err
}

func logMigrationError(id int, err error) {
	log.Printf("schema migration %d warning: %v", id, err)
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
