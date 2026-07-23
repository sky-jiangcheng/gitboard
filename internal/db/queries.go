package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// -- Config --

// GetConfig retrieves a configuration value by key.
func GetConfig(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM app_config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("config key not found: %s", key)
	}
	return value, err
}

// SetConfig sets a configuration value.
func SetConfig(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		"INSERT OR REPLACE INTO app_config (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}

// GetAllConfigs returns all configuration key-value pairs.
func GetAllConfigs(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT key, value FROM app_config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

// -- ScanRoots --

// GetScanRoots returns all configured scan root directories.
func GetScanRoots(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT path FROM scan_roots ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roots []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		roots = append(roots, path)
	}
	return roots, rows.Err()
}

// AddScanRoot adds a scan root directory.
func AddScanRoot(db *sql.DB, path string) error {
	_, err := db.Exec("INSERT OR IGNORE INTO scan_roots (path) VALUES (?)", path)
	return err
}

// ReplaceScanRoots atomically replaces all scan roots using a transaction.
func ReplaceScanRoots(db *sql.DB, roots []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec("DELETE FROM scan_roots"); err != nil {
		return fmt.Errorf("failed to clear scan roots: %w", err)
	}

	for _, root := range roots {
		if root == "" || strings.Contains(root, "\x00") {
			continue
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO scan_roots (path) VALUES (?)", root); err != nil {
			return fmt.Errorf("failed to insert scan root: %w", err)
		}
	}

	return tx.Commit()
}

// -- Projects --

// Project represents a grouped project in the dashboard.
type Project struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	RootPath      string `json:"root_path"`
	LevelOverride int    `json:"level_override"`
	IsAutoGrouped bool   `json:"is_auto_grouped"`
	IsStarred     bool   `json:"is_starred"`
	CreatedAt     string `json:"created_at"`
}

// ProjectWithStats includes statistics summary for a project.
type ProjectWithStats struct {
	Project
	RepoCount    int `json:"repo_count"`
	TotalAdded   int `json:"total_added"`
	TotalDeleted int `json:"total_deleted"`
	MyAdded      int `json:"my_added"`
	MyDeleted    int `json:"my_deleted"`
	MyFiles      int `json:"my_files"`
}

// UpsertProject inserts or updates a project record.
// Returns the project ID.
func UpsertProject(db *sql.DB, name, rootPath string, levelOverride int, isAutoGrouped bool) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO projects (name, root_path, level_override, is_auto_grouped)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 name=excluded.name, root_path=excluded.root_path,
		 level_override=excluded.level_override, is_auto_grouped=excluded.is_auto_grouped`,
		name, rootPath, levelOverride, isAutoGrouped,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpsertProjectTx performs UpsertProject within a given transaction.
func UpsertProjectTx(tx *sql.Tx, name, rootPath string, levelOverride int, isAutoGrouped bool) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO projects (name, root_path, level_override, is_auto_grouped)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 name=excluded.name, root_path=excluded.root_path,
		 level_override=excluded.level_override, is_auto_grouped=excluded.is_auto_grouped`,
		name, rootPath, levelOverride, isAutoGrouped,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetAllProjects returns all projects.
func GetAllProjects(db *sql.DB) ([]Project, error) {
	rows, err := db.Query("SELECT id, name, root_path, level_override, is_auto_grouped, is_starred, created_at FROM projects ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

// GetProjectByID returns a single project by ID.
func GetProjectByID(db *sql.DB, id int64) (*Project, error) {
	row := db.QueryRow(
		"SELECT id, name, root_path, level_override, is_auto_grouped, is_starred, created_at FROM projects WHERE id = ?",
		id,
	)
	p := &Project{}
	err := row.Scan(&p.ID, &p.Name, &p.RootPath, &p.LevelOverride, &p.IsAutoGrouped, &p.IsStarred, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateProjectLevel updates the level_override and is_auto_grouped for a project.
func UpdateProjectLevel(db *sql.DB, id int64, levelOverride int) error {
	_, err := db.Exec(
		"UPDATE projects SET level_override = ?, is_auto_grouped = 0 WHERE id = ?",
		levelOverride, id,
	)
	return err
}

// DeleteAllProjects removes all projects (for re-scan).
func DeleteAllProjects(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM projects")
	return err
}

// SyncProjectTx finds an existing project by root_path or creates a new one.
// Preserves is_starred for existing projects. Only updates name/is_auto_grouped
// for projects that were auto-grouped, respecting manual user adjustments.
// Returns the project ID.
func SyncProjectTx(tx *sql.Tx, name, rootPath string, levelOverride int, isAutoGrouped bool) (int64, error) {
	var id int64
	var existingAutoGrouped bool
	err := tx.QueryRow("SELECT id, is_auto_grouped FROM projects WHERE root_path = ?", rootPath).Scan(&id, &existingAutoGrouped)
	if err == nil {
		// Only update auto-grouped projects; preserve manually adjusted ones
		if existingAutoGrouped {
			_, err = tx.Exec(
				"UPDATE projects SET name = ?, is_auto_grouped = ? WHERE id = ?",
				name, isAutoGrouped, id,
			)
		}
		return id, err
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	// Insert new project
	res, err := tx.Exec(
		"INSERT INTO projects (name, root_path, level_override, is_auto_grouped) VALUES (?, ?, ?, ?)",
		name, rootPath, levelOverride, isAutoGrouped,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// CleanupStaleDataTx removes repositories not in scannedPaths (along with their
// stats and meta), then deletes projects that have no repos, notes, or todos.
// User-created notes and todos are always preserved.
func CleanupStaleDataTx(tx *sql.Tx, scannedPaths []string) error {
	if len(scannedPaths) == 0 {
		return nil
	}

	// Create a temporary table to hold scanned paths (avoids SQL parameter limits)
	if _, err := tx.Exec("CREATE TEMP TABLE IF NOT EXISTS _scanned_paths (path TEXT NOT NULL UNIQUE)"); err != nil {
		return fmt.Errorf("failed to create temp table: %w", err)
	}
	defer func() { _, _ = tx.Exec("DROP TABLE IF EXISTS _scanned_paths") }() //nolint:errcheck

	// Batch insert scanned paths
	stmt, err := tx.Prepare("INSERT OR IGNORE INTO _scanned_paths (path) VALUES (?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert: %w", err)
	}
	defer stmt.Close() //nolint:errcheck
	for _, p := range scannedPaths {
		if _, err := stmt.Exec(p); err != nil {
			return fmt.Errorf("failed to insert scanned path: %w", err)
		}
	}

	// Delete daily_stats for repos not in scanned set
	if _, err := tx.Exec(`
		DELETE FROM daily_stats
		WHERE repository_id IN (
			SELECT id FROM repositories WHERE path NOT IN (SELECT path FROM _scanned_paths)
		)
	`); err != nil {
		return fmt.Errorf("failed to cleanup stale stats: %w", err)
	}

	// Delete stale repos (repo_meta cascades via FK)
	if _, err := tx.Exec("DELETE FROM repositories WHERE path NOT IN (SELECT path FROM _scanned_paths)"); err != nil {
		return fmt.Errorf("failed to cleanup stale repos: %w", err)
	}

	// Delete orphaned projects: no repos, no notes, no todos
	if _, err := tx.Exec(`
		DELETE FROM projects WHERE
			id NOT IN (SELECT DISTINCT project_id FROM repositories WHERE project_id IS NOT NULL)
			AND id NOT IN (SELECT DISTINCT project_id FROM project_notes)
			AND id NOT IN (SELECT DISTINCT project_id FROM project_todos)
	`); err != nil {
		return fmt.Errorf("failed to cleanup orphaned projects: %w", err)
	}

	return nil
}

// ToggleProjectStar flips the starred status of a project.
func ToggleProjectStar(db *sql.DB, id int64) (bool, error) {
	res, err := db.Exec(
		"UPDATE projects SET is_starred = NOT is_starred WHERE id = ?",
		id,
	)
	if err != nil {
		return false, err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return false, fmt.Errorf("project not found: %d", id)
	}
	var newVal bool
	err = db.QueryRow("SELECT is_starred FROM projects WHERE id = ?", id).Scan(&newVal)
	return newVal, err
}

// GetStarredProjects returns only starred projects.
func GetStarredProjects(db *sql.DB) ([]Project, error) {
	rows, err := db.Query("SELECT id, name, root_path, level_override, is_auto_grouped, is_starred, created_at FROM projects WHERE is_starred = 1 ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

func scanProjects(rows *sql.Rows) ([]Project, error) {
	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.RootPath, &p.LevelOverride, &p.IsAutoGrouped, &p.IsStarred, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// -- Repositories --

// Repository represents a discovered git repository.
type Repository struct {
	ID            int64  `json:"id"`
	Path          string `json:"path"`
	ProjectID     *int64 `json:"project_id"`
	LastScannedAt string `json:"last_scanned_at"`
}

// UpsertRepository inserts or updates a repository record.
func UpsertRepository(db *sql.DB, path string, projectID int64) error {
	_, err := db.Exec(
		`INSERT INTO repositories (path, project_id)
		 VALUES (?, ?)
		 ON CONFLICT(path) DO UPDATE SET project_id=excluded.project_id`,
		path, projectID,
	)
	return err
}

// UpsertRepositoryTx performs UpsertRepository within a given transaction.
func UpsertRepositoryTx(tx *sql.Tx, path string, projectID int64) error {
	_, err := tx.Exec(
		`INSERT INTO repositories (path, project_id)
		 VALUES (?, ?)
		 ON CONFLICT(path) DO UPDATE SET project_id=excluded.project_id`,
		path, projectID,
	)
	return err
}

// GetRepositoriesByProjectID returns all repositories for a project.
func GetRepositoriesByProjectID(db *sql.DB, projectID int64) ([]Repository, error) {
	rows, err := db.Query(
		"SELECT id, path, project_id, last_scanned_at FROM repositories WHERE project_id = ?",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRepositories(rows)
}

// GetAllRepositories returns all known repositories.
func GetAllRepositories(db *sql.DB) ([]Repository, error) {
	rows, err := db.Query("SELECT id, path, project_id, last_scanned_at FROM repositories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRepositories(rows)
}

// UpdateRepositoryLastScanned updates the last_scanned_at timestamp.
func UpdateRepositoryLastScanned(db *sql.DB, id int64) error {
	_, err := db.Exec(
		"UPDATE repositories SET last_scanned_at = ? WHERE id = ?",
		time.Now().Format("2006-01-02 15:04:05"), id,
	)
	return err
}

// DeleteAllRepositories removes all repository records.
func DeleteAllRepositories(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM repositories")
	return err
}

func scanRepositories(rows *sql.Rows) ([]Repository, error) {
	var repos []Repository
	for rows.Next() {
		var r Repository
		var lastScanned sql.NullString
		if err := rows.Scan(&r.ID, &r.Path, &r.ProjectID, &lastScanned); err != nil {
			return nil, err
		}
		if lastScanned.Valid {
			r.LastScannedAt = lastScanned.String
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

// -- DailyStats --

// DailyStat represents a single day's commit statistics for a repository and author.
type DailyStat struct {
	ID           int64  `json:"id"`
	RepositoryID int64  `json:"repository_id"`
	StatDate     string `json:"stat_date"`
	Author       string `json:"author"`
	FilesChanged int    `json:"files_changed"`
	LinesAdded   int    `json:"lines_added"`
	LinesDeleted int    `json:"lines_deleted"`
}

// UpsertDailyStat inserts or updates a daily stats record.
func UpsertDailyStat(db *sql.DB, repoID int64, date, author string, files, added, deleted int) error {
	_, err := db.Exec(
		`INSERT INTO daily_stats (repository_id, stat_date, author, files_changed, lines_added, lines_deleted)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(repository_id, stat_date, author) DO UPDATE SET
		 files_changed=excluded.files_changed, lines_added=excluded.lines_added,
		 lines_deleted=excluded.lines_deleted`,
		repoID, date, author, files, added, deleted,
	)
	return err
}

// GetStatsByDate returns all stats for a given date.
func GetStatsByDate(db *sql.DB, date string) ([]DailyStat, error) {
	rows, err := db.Query(
		"SELECT id, repository_id, stat_date, author, files_changed, lines_added, lines_deleted FROM daily_stats WHERE stat_date = ?",
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDailyStats(rows)
}

// GetStatsByRepositoryAndDate returns stats for a specific repository and date.
func GetStatsByRepositoryAndDate(db *sql.DB, repoID int64, date string) ([]DailyStat, error) {
	rows, err := db.Query(
		`SELECT id, repository_id, stat_date, author, files_changed, lines_added, lines_deleted
		 FROM daily_stats WHERE repository_id = ? AND stat_date = ?`,
		repoID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDailyStats(rows)
}

// GetStatsByProject returns all stats for repositories belonging to a project.
func GetStatsByProject(db *sql.DB, projectID int64, date string) ([]DailyStat, error) {
	query := `
		SELECT ds.id, ds.repository_id, ds.stat_date, ds.author, ds.files_changed, ds.lines_added, ds.lines_deleted
		FROM daily_stats ds
		JOIN repositories r ON ds.repository_id = r.id
		WHERE r.project_id = ?
	`
	args := []interface{}{projectID}

	if date != "" {
		query += " AND ds.stat_date = ?"
		args = append(args, date)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDailyStats(rows)
}

func scanDailyStats(rows *sql.Rows) ([]DailyStat, error) {
	var stats []DailyStat
	for rows.Next() {
		var s DailyStat
		if err := rows.Scan(&s.ID, &s.RepositoryID, &s.StatDate, &s.Author, &s.FilesChanged, &s.LinesAdded, &s.LinesDeleted); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// -- Todos --

// Todo represents a project todo item.
type Todo struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Priority  int    `json:"priority"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListTodos returns all todos for a project, ordered by sort_order.
func ListTodos(db *sql.DB, projectID int64) ([]Todo, error) {
	rows, err := db.Query(
		"SELECT id, project_id, title, completed, priority, sort_order, created_at, updated_at FROM project_todos WHERE project_id = ? ORDER BY sort_order",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTodos(rows)
}

// CreateTodo inserts a new todo and returns the created record.
func CreateTodo(db *sql.DB, projectID int64, title string) (*Todo, error) {
	// Get the next sort_order value
	var maxSort int
	err := db.QueryRow(
		"SELECT COALESCE(MAX(sort_order), -1) FROM project_todos WHERE project_id = ?",
		projectID,
	).Scan(&maxSort)
	if err != nil {
		return nil, fmt.Errorf("failed to get max sort_order: %w", err)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := db.Exec(
		"INSERT INTO project_todos (project_id, title, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		projectID, title, maxSort+1, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create todo: %w", err)
	}

	todoID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &Todo{
		ID:        todoID,
		ProjectID: projectID,
		Title:     title,
		Completed: false,
		Priority:  0,
		SortOrder: maxSort + 1,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ToggleTodo flips the completed status of a todo and updates updated_at.
func ToggleTodo(db *sql.DB, todoID int64) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := db.Exec(
		"UPDATE project_todos SET completed = NOT completed, updated_at = ? WHERE id = ?",
		now, todoID,
	)
	if err != nil {
		return fmt.Errorf("failed to toggle todo: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("todo not found: %d", todoID)
	}
	return nil
}

// DeleteTodo removes a todo by ID.
func DeleteTodo(db *sql.DB, todoID int64) error {
	_, err := db.Exec("DELETE FROM project_todos WHERE id = ?", todoID)
	return err
}

// ReorderTodos updates sort_order for a list of todo IDs in a transaction.
func ReorderTodos(db *sql.DB, todoIDs []int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for i, id := range todoIDs {
		_, err := tx.Exec(
			"UPDATE project_todos SET sort_order = ?, updated_at = ? WHERE id = ?",
			i, time.Now().Format("2006-01-02 15:04:05"), id,
		)
		if err != nil {
			return fmt.Errorf("failed to update sort_order for todo %d: %w", id, err)
		}
	}

	return tx.Commit()
}

func scanTodos(rows *sql.Rows) ([]Todo, error) {
	var todos []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Completed, &t.Priority, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

// -- Notes --

// NoteKindKnowledge marks notes that read like structured knowledge (headings or frontmatter).
const NoteKindKnowledge = "knowledge"

// Note represents a project note.
type Note struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Tags      string `json:"tags"`
	Kind      string `json:"kind"`
	Pinned    bool   `json:"pinned"`
	Source    string `json:"source"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// NoteWithProject is a note joined with its parent project, for the global knowledge hub.
type NoteWithProject struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Tags        string `json:"tags"`
	Kind        string `json:"kind"`
	Pinned      bool   `json:"pinned"`
	Source      string `json:"source"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	ProjectName string `json:"project_name"`
	RootPath    string `json:"root_path"`
}

// noteColumns is the canonical SELECT list for project_notes, kept in sync with scanNote.
const noteColumns = `id, project_id, title, content, tags, kind, pinned, source, sort_order, created_at, updated_at`

// InferNoteMeta derives a default title and kind from note content.
// - kind is "knowledge" when the content starts with a Markdown heading or YAML frontmatter.
// - title is the first non-empty line, stripped of leading # markers and bold markers.
func InferNoteMeta(content string) (title, kind string) {
	kind = "other"
	if strings.HasPrefix(content, "---") || strings.HasPrefix(content, "# ") || strings.HasPrefix(content, "## ") {
		kind = NoteKindKnowledge
	}
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || t == "---" {
			continue
		}
		t = strings.TrimLeft(t, "#")
		t = strings.TrimSpace(t)
		t = strings.TrimPrefix(t, "**")
		t = strings.TrimSuffix(t, "**")
		if t != "" {
			return t, kind
		}
	}
	return "笔记", kind
}

// ListNotes returns all notes for a project, pinned first then by updated_at desc.
func ListNotes(db *sql.DB, projectID int64) ([]Note, error) {
	rows, err := db.Query(
		`SELECT `+noteColumns+` FROM project_notes WHERE project_id = ? ORDER BY pinned DESC, updated_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

// ListAllNotes returns every note across all projects, joined with project info,
// ordered pinned first then by most recently updated. Used by the global knowledge hub.
func ListAllNotes(db *sql.DB) ([]NoteWithProject, error) {
	rows, err := db.Query(
		`SELECT pn.` + noteColumns + `, p.name, p.root_path
		 FROM project_notes pn
		 JOIN projects p ON pn.project_id = p.id
		 ORDER BY pn.pinned DESC, pn.updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []NoteWithProject
	for rows.Next() {
		var n NoteWithProject
		if err := rows.Scan(
			&n.ID, &n.ProjectID, &n.Title, &n.Content, &n.Tags, &n.Kind, &n.Pinned, &n.Source,
			&n.SortOrder, &n.CreatedAt, &n.UpdatedAt, &n.ProjectName, &n.RootPath,
		); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// GetNote returns a single note by ID.
func GetNote(db *sql.DB, noteID int64) (*Note, error) {
	row := db.QueryRow(`SELECT `+noteColumns+` FROM project_notes WHERE id = ?`, noteID)
	n := &Note{}
	err := row.Scan(&n.ID, &n.ProjectID, &n.Title, &n.Content, &n.Tags, &n.Kind, &n.Pinned, &n.Source, &n.SortOrder, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// GetNoteBySourceTitle finds a note for a project with the given source+title,
// used to make knowledge imports idempotent. Returns nil when no match exists.
func GetNoteBySourceTitle(db *sql.DB, projectID int64, source, title string) (*Note, error) {
	row := db.QueryRow(
		`SELECT `+noteColumns+` FROM project_notes WHERE project_id = ? AND source = ? AND title = ? LIMIT 1`,
		projectID, source, title,
	)
	n := &Note{}
	err := row.Scan(&n.ID, &n.ProjectID, &n.Title, &n.Content, &n.Tags, &n.Kind, &n.Pinned, &n.Source, &n.SortOrder, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}

// CreateNote inserts a new note (legacy signature: title/kind inferred from content).
func CreateNote(db *sql.DB, projectID int64, content string) (*Note, error) {
	title, kind := InferNoteMeta(content)
	return createNote(db, projectID, title, content, "", kind, "manual")
}

// CreateNoteEx inserts a new note with explicit metadata.
func CreateNoteEx(db *sql.DB, projectID int64, title, content, tags, kind, source string) (*Note, error) {
	if title == "" {
		title, kind = InferNoteMeta(content)
	}
	if kind == "" {
		kind = "other"
	}
	if source == "" {
		source = "manual"
	}
	return createNote(db, projectID, title, content, tags, kind, source)
}

func createNote(db *sql.DB, projectID int64, title, content, tags, kind, source string) (*Note, error) {
	var maxSort int
	err := db.QueryRow(
		"SELECT COALESCE(MAX(sort_order), -1) FROM project_notes WHERE project_id = ?",
		projectID,
	).Scan(&maxSort)
	if err != nil {
		return nil, fmt.Errorf("failed to get max sort_order: %w", err)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := db.Exec(
		"INSERT INTO project_notes (project_id, title, content, tags, kind, source, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		projectID, title, content, tags, kind, source, maxSort+1, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	noteID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &Note{
		ID:        noteID,
		ProjectID: projectID,
		Title:     title,
		Content:   content,
		Tags:      tags,
		Kind:      kind,
		Source:    source,
		SortOrder: maxSort + 1,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateNote updates the content of a note and refreshes the inferred title/kind.
func UpdateNote(db *sql.DB, noteID int64, content string) error {
	title, kind := InferNoteMeta(content)
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := db.Exec(
		"UPDATE project_notes SET content = ?, title = ?, kind = ?, updated_at = ? WHERE id = ?",
		content, title, kind, now, noteID,
	)
	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("note not found: %d", noteID)
	}
	return nil
}

// UpdateNoteMeta updates editable metadata for a note (title/tags/kind/pinned).
func UpdateNoteMeta(db *sql.DB, noteID int64, title, tags, kind string, pinned bool) error {
	if kind == "" {
		kind = "other"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := db.Exec(
		"UPDATE project_notes SET title = ?, tags = ?, kind = ?, pinned = ?, updated_at = ? WHERE id = ?",
		title, tags, kind, pinned, now, noteID,
	)
	if err != nil {
		return fmt.Errorf("failed to update note meta: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("note not found: %d", noteID)
	}
	return nil
}

// PinNote sets the pinned flag on a note.
func PinNote(db *sql.DB, noteID int64, pinned bool) error {
	res, err := db.Exec("UPDATE project_notes SET pinned = ? WHERE id = ?", pinned, noteID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("note not found: %d", noteID)
	}
	return nil
}

// DeleteNote removes a note by ID.
func DeleteNote(db *sql.DB, noteID int64) error {
	_, err := db.Exec("DELETE FROM project_notes WHERE id = ?", noteID)
	return err
}

// ListAllTags returns the distinct set of tags used across all notes.
// Tags are stored as a comma-separated string; this splits and dedupes them.
func ListAllTags(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT tags FROM project_notes WHERE tags != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var tags []string
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		for _, t := range strings.Split(raw, ",") {
			t = strings.TrimSpace(t)
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			tags = append(tags, t)
		}
	}
	return tags, rows.Err()
}

func scanNotes(rows *sql.Rows) ([]Note, error) {
	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Title, &n.Content, &n.Tags, &n.Kind, &n.Pinned, &n.Source, &n.SortOrder, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// -- Heatmap --

// HeatmapDay represents a single day's contribution data.
type HeatmapDay struct {
	Date         string `json:"date"`
	LinesAdded   int    `json:"lines_added"`
	LinesDeleted int    `json:"lines_deleted"`
	Commits      int    `json:"commits"`
}

// GetHeatmapData returns daily aggregated stats for the given date range and author.
func GetHeatmapData(dbConn *sql.DB, startDate, endDate, author string) ([]HeatmapDay, error) {
	query := `
		SELECT stat_date, SUM(lines_added), SUM(lines_deleted), COUNT(*)
		FROM daily_stats
		WHERE stat_date >= ? AND stat_date <= ? AND author = ?
		GROUP BY stat_date
		ORDER BY stat_date
	`
	rows, err := dbConn.Query(query, startDate, endDate, author)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var days []HeatmapDay
	for rows.Next() {
		var d HeatmapDay
		var commits int
		if err := rows.Scan(&d.Date, &d.LinesAdded, &d.LinesDeleted, &commits); err != nil {
			return nil, err
		}
		d.Commits = commits
		days = append(days, d)
	}
	return days, rows.Err()
}

// HasStatsSince checks if we have daily stats for the given author going back
// to at least the given date. Returns true if stats exist on or before startDate.
func HasStatsSince(dbConn *sql.DB, startDate, author string) (bool, error) {
	var earliest string
	query := `
		SELECT MIN(stat_date) FROM daily_stats WHERE author = ?
	`
	err := dbConn.QueryRow(query, author).Scan(&earliest)
	if err != nil {
		return false, err
	}
	if earliest == "" {
		return false, nil
	}
	return earliest <= startDate, nil
}

// -- NoteCounts --

// NoteCount holds the note summary for a project.
type NoteCount struct {
	ProjectID int64 `json:"project_id"`
	Count     int   `json:"count"`
}

// GetNoteCounts returns the count of notes per project.
func GetNoteCounts(db *sql.DB) ([]NoteCount, error) {
	rows, err := db.Query(`
		SELECT
			project_id,
			COUNT(*) AS count
		FROM project_notes
		GROUP BY project_id
		ORDER BY project_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []NoteCount
	for rows.Next() {
		var c NoteCount
		if err := rows.Scan(&c.ProjectID, &c.Count); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}

// SearchHit is a unified search result across notes and todos.
type SearchHit struct {
	Type        string `json:"type"` // "note" | "todo"
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	ProjectName string `json:"project_name"`
	Title       string `json:"title"`
	Snippet     string `json:"snippet"`
	Tags        string `json:"tags,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

// searchMaxQuery caps the query length to bound work.
const searchMaxQuery = 200

// searchResultLimit caps the number of hits returned.
const searchResultLimit = 30

// searchSnippetWindow is the number of characters of context kept on each side of a match.
const searchSnippetWindow = 60

// SearchNotes searches note content/title/tags across all projects, returning
// ranked hits with context snippets. Ranking rewards term frequency, title and
// tag matches, and recency.
func SearchNotes(db *sql.DB, query string) ([]SearchHit, error) {
	return searchNotesLike(db, query, false)
}

// searchNotesLike runs the note search. When includeTodos is true the result is
// merged with todo matches and de-duplicated by (type,id).
func searchNotesLike(db *sql.DB, query string, includeTodos bool) ([]SearchHit, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if len(q) > searchMaxQuery {
		q = q[:searchMaxQuery]
	}
	like := "%" + q + "%"

	// Build candidate notes (LIKE on content, title, or tags).
	rows, err := db.Query(`
		SELECT pn.id, pn.title, pn.content, pn.tags, pn.updated_at, pn.project_id, p.name
		FROM project_notes pn
		JOIN projects p ON pn.project_id = p.id
		WHERE pn.content LIKE ? OR pn.title LIKE ? OR pn.tags LIKE ?
	`, like, like, like)
	if err != nil {
		return nil, err
	}

	type noteCand struct {
		SearchHit
		score int
	}
	var cands []noteCand
	for rows.Next() {
		var c noteCand
		if err := rows.Scan(&c.ID, &c.Title, &c.Snippet, &c.Tags, &c.UpdatedAt, &c.ProjectID, &c.ProjectName); err != nil {
			rows.Close()
			return nil, err
		}
		// Snippet field temporarily holds full content; build the real snippet below.
		content := c.Snippet
		c.Snippet = makeSnippet(content, q)
		c.Type = "note"
		c.score = scoreNote(content, c.Title, c.Tags, c.UpdatedAt, q)
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if includeTodos {
		todoHits, err := searchTodos(db, q)
		if err != nil {
			return nil, err
		}
		for _, th := range todoHits {
			cands = append(cands, noteCand{SearchHit: th, score: 5 + countOccurrences(strings.ToLower(th.Title), strings.ToLower(q))})
		}
	}

	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].score > cands[j].score
	})

	hits := make([]SearchHit, 0, searchResultLimit)
	for _, c := range cands {
		if len(hits) >= searchResultLimit {
			break
		}
		hits = append(hits, c.SearchHit)
	}
	return hits, nil
}

// SearchAll searches notes and todos together, returning ranked, unified hits.
func SearchAll(db *sql.DB, query string) ([]SearchHit, error) {
	return searchNotesLike(db, query, true)
}

// searchTodos matches todos by title.
func searchTodos(db *sql.DB, q string) ([]SearchHit, error) {
	like := "%" + q + "%"
	rows, err := db.Query(`
		SELECT t.id, t.title, t.updated_at, t.project_id, p.name
		FROM project_todos t
		JOIN projects p ON t.project_id = p.id
		WHERE t.title LIKE ?
	`, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.ID, &h.Title, &h.UpdatedAt, &h.ProjectID, &h.ProjectName); err != nil {
			return nil, err
		}
		h.Type = "todo"
		h.Snippet = h.Title
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// scoreNote computes a relevance score for a note match.
func scoreNote(content, title, tags, updatedAt, query string) int {
	lowQ := strings.ToLower(query)
	score := countOccurrences(strings.ToLower(content), lowQ) * 2
	if strings.Contains(strings.ToLower(title), lowQ) {
		score += 6
	}
	if strings.Contains(strings.ToLower(tags), lowQ) {
		score += 8
	}
	// Recency: up to +5 for notes updated in the last 30 days.
	if t, err := time.Parse("2006-01-02 15:04:05", updatedAt); err == nil {
		days := int(time.Since(t).Hours() / 24)
		switch {
		case days <= 1:
			score += 5
		case days <= 7:
			score += 4
		case days <= 30:
			score += 2
		}
	}
	return score
}

// countOccurrences counts non-overlapping case-folded occurrences of sub in s.
func countOccurrences(s, sub string) int {
	if sub == "" {
		return 0
	}
	c := 0
	for i := 0; i+len(sub) <= len(s); {
		if s[i:i+len(sub)] == sub {
			c++
			if c >= 12 {
				return c
			}
			i += len(sub)
		} else {
			i++
		}
	}
	return c
}

// makeSnippet returns a window of content around the first match of query.
func makeSnippet(content, query string) string {
	if content == "" {
		return ""
	}
	low := strings.ToLower(content)
	idx := strings.Index(low, strings.ToLower(query))
	const w = searchSnippetWindow
	if idx < 0 {
		// No exact substring (e.g. multi-word query); return the head.
		if len(content) > w*2 {
			return content[:w*2] + "…"
		}
		return content
	}
	start := idx - w
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + w
	if end > len(content) {
		end = len(content)
	}
	snip := strings.ReplaceAll(content[start:end], "\n", " ")
	prefix := ""
	if start > 0 {
		prefix = "…"
	}
	suffix := ""
	if end < len(content) {
		suffix = "…"
	}
	return prefix + snip + suffix
}

// -- TodoCounts --

// TodoCount holds the todo summary for a project.
type TodoCount struct {
	ProjectID int64 `json:"project_id"`
	Count     int   `json:"count"`
	Total     int   `json:"total"`
}

// GetTodoCounts returns the incomplete and total todo counts per project.
func GetTodoCounts(db *sql.DB) ([]TodoCount, error) {
	rows, err := db.Query(`
		SELECT
			project_id,
			SUM(CASE WHEN completed = 0 THEN 1 ELSE 0 END) AS count,
			COUNT(*) AS total
		FROM project_todos
		GROUP BY project_id
		ORDER BY project_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []TodoCount
	for rows.Next() {
		var c TodoCount
		if err := rows.Scan(&c.ProjectID, &c.Count, &c.Total); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}

// -- RepoMeta (knowledge mining cache) --

// RepoMeta caches mined knowledge (tech stack, README excerpt, language breakdown)
// for a repository so repeated dashboard loads do not re-scan the filesystem.
type RepoMeta struct {
	RepositoryID  int64  `json:"repository_id"`
	TechStack     string `json:"tech_stack"` // JSON array of tech names
	ReadmeExcerpt string `json:"readme_excerpt"`
	Languages     string `json:"languages"` // JSON map of language->file count
	UpdatedAt     string `json:"updated_at"`
}

// GetRepoMeta returns cached meta for a repository, or nil if absent.
func GetRepoMeta(db *sql.DB, repoID int64) (*RepoMeta, error) {
	row := db.QueryRow(
		"SELECT repository_id, tech_stack, readme_excerpt, languages, updated_at FROM repo_meta WHERE repository_id = ?",
		repoID,
	)
	var m RepoMeta
	err := row.Scan(&m.RepositoryID, &m.TechStack, &m.ReadmeExcerpt, &m.Languages, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpsertRepoMeta stores mined knowledge for a repository, replacing any prior cache.
func UpsertRepoMeta(db *sql.DB, repoID int64, techStack, readmeExcerpt, languages string) error {
	_, err := db.Exec(
		`INSERT INTO repo_meta (repository_id, tech_stack, readme_excerpt, languages, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(repository_id) DO UPDATE SET
		   tech_stack=excluded.tech_stack,
		   readme_excerpt=excluded.readme_excerpt,
		   languages=excluded.languages,
		   updated_at=excluded.updated_at`,
		repoID, techStack, readmeExcerpt, languages, time.Now().Format("2006-01-02 15:04:05"),
	)
	return err
}
