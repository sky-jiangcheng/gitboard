package db

import (
	"database/sql"
	"fmt"
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

// RemoveScanRoot removes a scan root directory.
func RemoveScanRoot(db *sql.DB, path string) error {
	_, err := db.Exec("DELETE FROM scan_roots WHERE path = ?", path)
	return err
}

// -- Projects --

// Project represents a grouped project in the dashboard.
type Project struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	RootPath      string `json:"root_path"`
	LevelOverride int    `json:"level_override"`
	IsAutoGrouped bool   `json:"is_auto_grouped"`
	CreatedAt     string `json:"created_at"`
}

// ProjectWithStats includes statistics summary for a project.
type ProjectWithStats struct {
	Project
	RepoCount     int `json:"repo_count"`
	TotalAdded    int `json:"total_added"`
	TotalDeleted  int `json:"total_deleted"`
	MyAdded       int `json:"my_added"`
	MyDeleted     int `json:"my_deleted"`
	MyFiles       int `json:"my_files"`
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

// GetAllProjects returns all projects.
func GetAllProjects(db *sql.DB) ([]Project, error) {
	rows, err := db.Query("SELECT id, name, root_path, level_override, is_auto_grouped, created_at FROM projects ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

// GetProjectByID returns a single project by ID.
func GetProjectByID(db *sql.DB, id int64) (*Project, error) {
	row := db.QueryRow(
		"SELECT id, name, root_path, level_override, is_auto_grouped, created_at FROM projects WHERE id = ?",
		id,
	)
	p := &Project{}
	err := row.Scan(&p.ID, &p.Name, &p.RootPath, &p.LevelOverride, &p.IsAutoGrouped, &p.CreatedAt)
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

func scanProjects(rows *sql.Rows) ([]Project, error) {
	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.RootPath, &p.LevelOverride, &p.IsAutoGrouped, &p.CreatedAt); err != nil {
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

// Note represents a project note.
type Note struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	Content   string `json:"content"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListNotes returns all notes for a project, ordered by sort_order.
func ListNotes(db *sql.DB, projectID int64) ([]Note, error) {
	rows, err := db.Query(
		"SELECT id, project_id, content, sort_order, created_at, updated_at FROM project_notes WHERE project_id = ? ORDER BY sort_order",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

// CreateNote inserts a new note and returns the created record.
func CreateNote(db *sql.DB, projectID int64, content string) (*Note, error) {
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
		"INSERT INTO project_notes (project_id, content, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		projectID, content, maxSort+1, now, now,
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
		Content:   content,
		SortOrder: maxSort + 1,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateNote updates the content and updated_at of a note.
func UpdateNote(db *sql.DB, noteID int64, content string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := db.Exec(
		"UPDATE project_notes SET content = ?, updated_at = ? WHERE id = ?",
		content, now, noteID,
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

// DeleteNote removes a note by ID.
func DeleteNote(db *sql.DB, noteID int64) error {
	_, err := db.Exec("DELETE FROM project_notes WHERE id = ?", noteID)
	return err
}

func scanNotes(rows *sql.Rows) ([]Note, error) {
	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Content, &n.SortOrder, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// -- Heatmap --

// HeatmapDay represents a single day's contribution data.
type HeatmapDay struct {
	Date        string `json:"date"`
	LinesAdded  int    `json:"lines_added"`
	LinesDeleted int   `json:"lines_deleted"`
	Commits     int    `json:"commits"`
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
